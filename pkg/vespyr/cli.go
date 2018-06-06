package vespyr

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	coinbase "github.com/DavidHuie/go-coinbase-exchange"
	"github.com/DavidHuie/kraken-go-api-client"
	"github.com/MaxHalford/gago"
	"github.com/go-pg/pg"
	"github.com/heroku/rollrus"
	_ "github.com/mattes/migrate/database/postgres"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"

	"io"

	"os/exec"

	"math/rand"

	"github.com/jonboulle/clockwork"
	"github.com/mattes/migrate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// RootCmd is the CLI interface to vespyr.
var RootCmd = &cobra.Command{
	Use:   "vespyr",
	Short: "vespyr is a currency trading engine",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Run `vespyr help` for more information.")
	},
}

func defineCommands() {
	func() {
		realtimeImport := &cobra.Command{
			Use:   "realtime-import",
			Short: "import data in realtime",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				wg := &sync.WaitGroup{}

				for product := range ProductToMetadata {
					var exchange Exchange
					if ProductToMetadata[product].ExchangeType == ExchangeKraken {
						exchange = runner.KrakenExchange
					} else {
						exchange = runner.GDAXExchange
					}
					importer := NewRealtimeImporter(product, runner.Backend, exchange)

					wg.Add(1)
					go func() {
						defer wg.Done()
						importer.Start(time.Second)
					}()
				}

				wg.Wait()
			},
		}
		RootCmd.AddCommand(realtimeImport)
	}()

	func() {
		bot := &cobra.Command{
			Use:   "bot",
			Short: "run the automated trading bot",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				wg := &sync.WaitGroup{}
				wg.Add(3)

				go func() {
					defer wg.Done()
					runner.BTCUSDBot.Run(context.Background())
				}()
				go func() {
					defer wg.Done()
					runner.ETHUSDBot.Run(context.Background())
				}()
				go func() {
					defer wg.Done()
					runner.LTCUSDBot.Run(context.Background())
				}()

				wg.Wait()
			},
		}
		RootCmd.AddCommand(bot)
	}()

	func() {
		var startTime, endTime string
		var longPeriod, shortPeriod uint
		var downThreshold, upThreshold float64
		var granularity uint
		var resultsFile string
		var generatePlotlyGraph bool
		var product string
		backtest := &cobra.Command{
			Use:   "backtest-ema-crossover",
			Short: "backtest an EMA crossover strategy",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				if _, ok := ProductToMetadata[Product(product)]; !ok {
					fmt.Println("unknown product: ", product)
					os.Exit(1)
				}

				model := &TradingStrategyModel{
					Product:          Product(product),
					HistoryTicks:     1000,
					State:            StrategyStateTryingToBuy,
					InitialBudget:    100,
					Budget:           100,
					BudgetCurrency:   ProductToMetadata[Product(product)].MarketOrderBuyCurrency,
					InvestedCurrency: ProductToMetadata[Product(product)].MarketOrderSellCurrency,
					TickSizeMinutes:  granularity,
					TradingStrategy:  TradingStrategyEMACrossover,
				}

				strategy := &EMACrossoverStrategy{
					ShortPeriod:   shortPeriod,
					LongPeriod:    longPeriod,
					DownThreshold: downThreshold,
					UpThreshold:   upThreshold,
				}
				model.TradingStrategyData, err = yaml.Marshal(strategy)
				if err != nil {
					fmt.Printf("error marshaling trading strategy")
					os.Exit(1)
				}

				s, err := time.Parse(time.RFC822, startTime)
				if err != nil {
					fmt.Printf("error parsing start time: %s", err)
					os.Exit(1)
				}
				e, err := time.Parse(time.RFC822, endTime)
				if err != nil {
					fmt.Printf("error parsing end time: %s", err)
					os.Exit(1)
				}

				backtester, err := NewBacktester(
					s, e,
					model,
					runner.Backend,
					rand.NewSource(time.Now().Unix()),
				)
				if err != nil {
					fmt.Printf("error creating backtester: %s", err)
					os.Exit(1)
				}
				if err := backtester.Backtest(); err != nil {
					fmt.Printf("error running backtest: %s", err)
					os.Exit(1)
				}

				results := backtester.Results()
				fmt.Printf("Total trades: %d\n", results.LossTrades+results.ProfitTrades)
				fmt.Printf("Gross profit: %f\n", results.GrossProfit)
				fmt.Printf("Gross loss: %f\n", results.GrossLoss)
				fmt.Printf("Net profit: %f\n", results.NetProfit())
				fmt.Printf("Profit factor: %f\n", results.ProfitFactor())
				fmt.Printf("Expected payoff: %f\n", results.ExpectedPayoff())
				fmt.Printf("Budget currency: %s\n", results.BudgetCurrency)
				fmt.Printf("Starting budget: %f\n", results.InitialBudget)
				fmt.Printf("Ending budget: %f\n", results.FinalBudget)
				fmt.Printf("Trade currency: %s\n", results.TradeCurrency)
				fmt.Printf("Initial trade currency price: %f\n", results.InitialCurrencyPrice)
				fmt.Printf("Final trade currency price: %f\n", results.FinalCurrencyPrice)
				fmt.Printf("Currency gains: %f%%\n", 100*(results.FinalCurrencyPrice-results.InitialCurrencyPrice)/results.InitialCurrencyPrice)
				fmt.Printf("Trading gains: %f%%\n", 100*(results.NetProfit())/results.InitialBudget)
				fmt.Printf("Total days traded: %#v\n", len(results.PortfolioValuePerDay))
				fmt.Printf("Sharpe ratio: %#v\n", results.SharpeRatio())

				reader, err := backtester.ResultsCSV()
				if err != nil {
					fmt.Printf("error creating results CS: %s", err)
					os.Exit(1)
				}

				file, err := os.Create(resultsFile)
				if err != nil {
					fmt.Printf("error opening file: %s", err)
					os.Exit(1)
				}
				defer file.Close()

				if _, err := io.Copy(file, reader); err != nil {
					fmt.Printf("error copying to file: %s", err)
					os.Exit(1)
				}

				if generatePlotlyGraph {
					os.Setenv("GRAPH_NAME", strategy.String())
					if err := exec.Command("python", "graphs/graph.py").Run(); err != nil {
						fmt.Printf("error creating plotly graph: %s", err)
						os.Exit(1)
					}
				}
			},
		}

		backtest.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the product to import")
		backtest.Flags().StringVar(&startTime, "start-time", time.Now().Add(-30*24*time.Hour).Format(time.RFC822), "the start time for the experiment")
		backtest.Flags().StringVar(&endTime, "end-time", time.Now().Format(time.RFC822), "the end time for the experiment")
		backtest.Flags().UintVar(&shortPeriod, "short-period", 10, "the EMA short period")
		backtest.Flags().UintVar(&longPeriod, "long-period", 25, "the EMA long period")
		backtest.Flags().Float64Var(&downThreshold, "down-threshold", 0, "the EMA down threshold")
		backtest.Flags().Float64Var(&upThreshold, "up-threshold", 0, "the EMA up threshold")
		backtest.Flags().UintVar(&granularity, "tick-size-minutes", 15, "the tick size in minutes")
		backtest.Flags().StringVar(&resultsFile, "results-file", "results.csv", "where to store the results")
		backtest.Flags().BoolVar(&generatePlotlyGraph, "graph", false, "generate plotly graph")

		RootCmd.AddCommand(backtest)
	}()

	func() {
		var startTime, endTime string
		var emaLongPeriod, emaShortPeriod uint
		var emaDownThreshold, emaUpThreshold float64
		var rsiEntrance, rsiExit float64
		var granularity uint
		var resultsFile string
		var generatePlotlyGraph bool
		var product string
		backtest := &cobra.Command{
			Use:   "backtest-s1",
			Short: "backtest the s1 strategy",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				if _, ok := ProductToMetadata[Product(product)]; !ok {
					fmt.Println("unknown product: ", product)
					os.Exit(1)
				}

				model := &TradingStrategyModel{
					Product:          Product(product),
					HistoryTicks:     1000,
					State:            StrategyStateTryingToBuy,
					InitialBudget:    100,
					Budget:           100,
					BudgetCurrency:   ProductToMetadata[Product(product)].MarketOrderBuyCurrency,
					InvestedCurrency: ProductToMetadata[Product(product)].MarketOrderSellCurrency,
					TickSizeMinutes:  granularity,
				}

				strategy := &S1Strategy{
					EMAShortPeriod:       emaShortPeriod,
					EMALongPeriod:        emaLongPeriod,
					EMADownThreshold:     emaDownThreshold,
					EMAUpThreshold:       emaUpThreshold,
					RSIEntranceThreshold: rsiEntrance,
					RSIExitThreshold:     rsiExit,
				}
				if err := model.SetStrategy(strategy); err != nil {
					fmt.Printf("error setting trading strategy")
					os.Exit(1)
				}

				s, err := time.Parse(time.RFC822, startTime)
				if err != nil {
					fmt.Printf("error parsing start time: %s", err)
					os.Exit(1)
				}
				e, err := time.Parse(time.RFC822, endTime)
				if err != nil {
					fmt.Printf("error parsing end time: %s", err)
					os.Exit(1)
				}

				backtester, err := NewBacktester(
					s, e,
					model,
					runner.Backend,
					rand.NewSource(time.Now().Unix()),
				)
				if err != nil {
					fmt.Printf("error creating backtester: %s", err)
					os.Exit(1)
				}
				if err := backtester.Backtest(); err != nil {
					fmt.Printf("error running backtest: %s", err)
					os.Exit(1)
				}

				results := backtester.Results()
				fmt.Printf("Total trades: %d\n", results.LossTrades+results.ProfitTrades)
				fmt.Printf("Gross profit: %f\n", results.GrossProfit)
				fmt.Printf("Gross loss: %f\n", results.GrossLoss)
				fmt.Printf("Net profit: %f\n", results.NetProfit())
				fmt.Printf("Profit factor: %f\n", results.ProfitFactor())
				fmt.Printf("Expected payoff: %f\n", results.ExpectedPayoff())
				fmt.Printf("Budget currency: %s\n", results.BudgetCurrency)
				fmt.Printf("Starting budget: %f\n", results.InitialBudget)
				fmt.Printf("Ending budget: %f\n", results.FinalBudget)
				fmt.Printf("Trade currency: %s\n", results.TradeCurrency)
				fmt.Printf("Initial trade currency price: %f\n", results.InitialCurrencyPrice)
				fmt.Printf("Final trade currency price: %f\n", results.FinalCurrencyPrice)
				fmt.Printf("Currency gains: %f%%\n", 100*(results.FinalCurrencyPrice-results.InitialCurrencyPrice)/results.InitialCurrencyPrice)
				fmt.Printf("Trading gains: %f%%\n", 100*(results.NetProfit())/results.InitialBudget)

				reader, err := backtester.ResultsCSV()
				if err != nil {
					fmt.Printf("error creating results CS: %s", err)
					os.Exit(1)
				}

				file, err := os.Create(resultsFile)
				if err != nil {
					fmt.Printf("error opening file: %s", err)
					os.Exit(1)
				}
				defer file.Close()

				if _, err := io.Copy(file, reader); err != nil {
					fmt.Printf("error copying to file: %s", err)
					os.Exit(1)
				}

				if generatePlotlyGraph {
					os.Setenv("GRAPH_NAME", fmt.Sprintf("s1-%d", time.Now().Unix()))
					cmd := exec.Command("python", "graphs/graph.py")
					if err := cmd.Run(); err != nil {
						fmt.Printf("error creating plotly graph: %s", err)
						out, _ := cmd.CombinedOutput()
						fmt.Printf(string(out))
						os.Exit(1)
					}
				}
			},
		}

		backtest.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the product to import")
		backtest.Flags().StringVar(&startTime, "start-time", time.Now().Add(-30*24*time.Hour).Format(time.RFC822), "the start time for the experiment")
		backtest.Flags().StringVar(&endTime, "end-time", time.Now().Format(time.RFC822), "the end time for the experiment")
		backtest.Flags().UintVar(&emaShortPeriod, "ema-short-period", 10, "the EMA short period")
		backtest.Flags().UintVar(&emaLongPeriod, "ema-long-period", 25, "the EMA long period")
		backtest.Flags().Float64Var(&emaDownThreshold, "ema-down-threshold", 0, "the EMA down threshold")
		backtest.Flags().Float64Var(&emaUpThreshold, "ema-up-threshold", 0, "the EMA up threshold")
		backtest.Flags().Float64Var(&rsiEntrance, "rsi-entrance", 0, "the RSI entrance threshold")
		backtest.Flags().Float64Var(&rsiExit, "rsi-exit", 100, "the RSI exit threshold")
		backtest.Flags().UintVar(&granularity, "tick-size-minutes", 15, "the tick size in minutes")
		backtest.Flags().StringVar(&resultsFile, "results-file", "results.csv", "where to store the results")
		backtest.Flags().BoolVar(&generatePlotlyGraph, "graph", false, "generate plotly graph")

		RootCmd.AddCommand(backtest)
	}()

	func() {
		var granularity uint
		var startTime, endTime string
		var rsiPeriod int
		var rsiEntrance, rsiExit float64
		var product string
		backtest := &cobra.Command{
			Use:   "backtest-rsi",
			Short: "backtest the rsi strategy",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				if _, ok := ProductToMetadata[Product(product)]; !ok {
					fmt.Println("unknown product: ", product)
					os.Exit(1)
				}

				model := &TradingStrategyModel{
					Product:          Product(product),
					HistoryTicks:     1000,
					State:            StrategyStateTryingToBuy,
					InitialBudget:    100,
					Budget:           100,
					BudgetCurrency:   ProductToMetadata[Product(product)].MarketOrderBuyCurrency,
					InvestedCurrency: ProductToMetadata[Product(product)].MarketOrderSellCurrency,
					TickSizeMinutes:  granularity,
				}

				strategy := &RSIStrategy{
					Period:        uint(rsiPeriod),
					BuyThreshold:  rsiEntrance,
					SellThreshold: rsiExit,
				}
				if err := model.SetStrategy(strategy); err != nil {
					fmt.Printf("error setting trading strategy")
					os.Exit(1)
				}

				s, err := time.Parse(time.RFC822, startTime)
				if err != nil {
					fmt.Printf("error parsing start time: %s", err)
					os.Exit(1)
				}
				e, err := time.Parse(time.RFC822, endTime)
				if err != nil {
					fmt.Printf("error parsing end time: %s", err)
					os.Exit(1)
				}

				backtester, err := NewBacktester(
					s, e,
					model,
					runner.Backend,
					rand.NewSource(time.Now().Unix()),
				)
				if err != nil {
					fmt.Printf("error creating backtester: %s", err)
					os.Exit(1)
				}
				if err := backtester.Backtest(); err != nil {
					fmt.Printf("error running backtest: %s", err)
					os.Exit(1)
				}

				results := backtester.Results()
				fmt.Printf("Total trades: %d\n", results.LossTrades+results.ProfitTrades)
				fmt.Printf("Gross profit: %f\n", results.GrossProfit)
				fmt.Printf("Gross loss: %f\n", results.GrossLoss)
				fmt.Printf("Net profit: %f\n", results.NetProfit())
				fmt.Printf("Profit factor: %f\n", results.ProfitFactor())
				fmt.Printf("Expected payoff: %f\n", results.ExpectedPayoff())
				fmt.Printf("Budget currency: %s\n", results.BudgetCurrency)
				fmt.Printf("Starting budget: %f\n", results.InitialBudget)
				fmt.Printf("Ending budget: %f\n", results.FinalBudget)
				fmt.Printf("Trade currency: %s\n", results.TradeCurrency)
				fmt.Printf("Initial trade currency price: %f\n", results.InitialCurrencyPrice)
				fmt.Printf("Final trade currency price: %f\n", results.FinalCurrencyPrice)
				fmt.Printf("Currency gains: %f%%\n", 100*(results.FinalCurrencyPrice-results.InitialCurrencyPrice)/results.InitialCurrencyPrice)
				fmt.Printf("Trading gains: %f%%\n", 100*(results.NetProfit())/results.InitialBudget)
			},
		}

		backtest.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the product to import")
		backtest.Flags().StringVar(&startTime, "start-time", time.Now().Add(-30*24*time.Hour).Format(time.RFC822), "the start time for the experiment")
		backtest.Flags().StringVar(&endTime, "end-time", time.Now().Format(time.RFC822), "the end time for the experiment")
		backtest.Flags().Float64Var(&rsiEntrance, "rsi-entrance", 0, "the RSI entrance threshold")
		backtest.Flags().Float64Var(&rsiExit, "rsi-exit", 100, "the RSI exit threshold")
		backtest.Flags().IntVar(&rsiPeriod, "rsi-period", 14, "the RSI period")
		backtest.Flags().UintVar(&granularity, "tick-size-minutes", 15, "the tick size in minutes")

		RootCmd.AddCommand(backtest)
	}()

	func() {
		var startTime, endTime string
		var granularity uint
		var generations uint
		var populationSize uint
		var tradingStrategyType string
		var product string
		optimizer := &cobra.Command{
			Use:   "optimize-strategy",
			Short: "optimizes a genetic algorithm",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				if _, ok := ProductToMetadata[Product(product)]; !ok {
					fmt.Println("unknown product: ", product)
					os.Exit(1)
				}

				model := &TradingStrategyModel{
					Product:          Product(product),
					HistoryTicks:     1000,
					State:            StrategyStateTryingToBuy,
					InitialBudget:    100,
					Budget:           100,
					BudgetCurrency:   ProductToMetadata[Product(product)].MarketOrderBuyCurrency,
					InvestedCurrency: ProductToMetadata[Product(product)].MarketOrderSellCurrency,
					TickSizeMinutes:  granularity,
					TradingStrategy:  tradingStrategyType,
				}

				s, err := time.Parse(time.RFC822, startTime)
				if err != nil {
					fmt.Printf("error parsing start time: %s", err)
					os.Exit(1)
				}
				e, err := time.Parse(time.RFC822, endTime)
				if err != nil {
					fmt.Printf("error parsing end time: %s", err)
					os.Exit(1)
				}

				factory, err := NewBacktesterGenomeFactory(
					s, e,
					model,
					runner.Backend,
				)
				if err != nil {
					fmt.Printf("error creating genome factory: %s", err)
					os.Exit(1)
				}

				ga := gago.Generational(factory.Generate)
				ga.PopSize = int(populationSize)
				ga.Initialize()

				for i := 1; i <= int(generations); i++ {
					if err := ga.Enhance(); err != nil {
						fmt.Printf("error enhancing generation: %s\n", err)
						os.Exit(1)
					}
					genome := ga.Best.Genome.(*BacktesterGenome)
					fmt.Printf("Best value at generation %d: $%f\n", i, -ga.Best.Genome.Evaluate())
					fmt.Printf("  %s\n", genome.strategy)
				}
			},
		}

		optimizer.Flags().UintVar(&generations, "generations", 100, "the number of generations to run")
		optimizer.Flags().UintVar(&populationSize, "population-size", 100, "the number of generations to run")
		optimizer.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the exchange product to use")
		optimizer.Flags().StringVar(&startTime, "start-time", time.Now().Add(-30*24*time.Hour).Format(time.RFC822), "the start time for the experiment")
		optimizer.Flags().StringVar(&endTime, "end-time", time.Now().Format(time.RFC822), "the end time for the experiment")
		optimizer.Flags().StringVar(&tradingStrategyType, "strategy-type", TradingStrategyEMACrossover, "the type of strategy to use")
		optimizer.Flags().UintVar(&granularity, "tick-size-minutes", 15, "the tick size in minutes")

		RootCmd.AddCommand(optimizer)
	}()

	func() {
		var startTime, endTime string
		var product string
		historicalImport := &cobra.Command{
			Use:   "import",
			Short: "import historical data",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				s, err := time.Parse(time.RFC822, startTime)
				if err != nil {
					fmt.Printf("error parsing start time: %s", err)
					os.Exit(1)
				}
				e, err := time.Parse(time.RFC822, endTime)
				if err != nil {
					fmt.Printf("error parsing end time: %s", err)
					os.Exit(1)
				}

				switch Product(product) {
				case ProductBTCUSD:
					runner.BTCUSDHistoricalImporter.Import(s, e)
				case ProductETHUSD:
					runner.ETHUSDHistoricalImporter.Import(s, e)
				case ProductLTCUSD:
					runner.LTCUSDHistoricalImporter.Import(s, e)
				default:
					fmt.Printf("unsupported product: %s", product)
					os.Exit(1)
				}
			},
		}

		historicalImport.Flags().StringVar(&startTime, "start-time", time.Now().Add(-30*24*time.Hour).Format(time.RFC822), "the start time for the experiment")
		historicalImport.Flags().StringVar(&endTime, "end-time", time.Now().Format(time.RFC822), "the end time for the experiment")
		historicalImport.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the product to import")
		RootCmd.AddCommand(historicalImport)
	}()

	func() {
		migrationsCmd := &cobra.Command{
			Use:   "migrate",
			Short: "migrate the database",
			Run: func(cmd *cobra.Command, _ []string) {
				m, err := migrate.New(
					"code://",
					viper.GetString("postgres"),
				)
				if err != nil {
					fmt.Printf("error creating migrator: %s\n", err)
					os.Exit(1)
				}
				defer m.Close()

				m.Log = new(migrationLogger)

				currentVersion, dirty, err := m.Version()
				if err != nil && err != migrate.ErrNilVersion {
					fmt.Printf("error getting migration version: %s", err)
					os.Exit(1)
				}

				if dirty {
					if currentVersion-1 == 0 {
						panic("v0 migration detected")
					}

					if err := m.Force(int(currentVersion - 1)); err != nil {
						fmt.Printf("error forcing version: %s", err)
						os.Exit(1)
					}
				}

				logrus.Println("running migrations")
				if err := m.Up(); err != nil {
					if err == migrate.ErrNoChange {
						os.Exit(0)
					}

					fmt.Printf("error migrating: %s\n", err)
					os.Exit(1)
				}
			},
		}
		RootCmd.AddCommand(migrationsCmd)
	}()

	func() {
		rollbackCmd := &cobra.Command{
			Use:   "rollback",
			Short: "rollback the database",
			Run: func(cmd *cobra.Command, _ []string) {
				m, err := migrate.New(
					"code://",
					viper.GetString("postgres"),
				)
				if err != nil {
					fmt.Printf("error creating migrator: %s\n", err)
					os.Exit(1)
				}
				defer m.Close()

				m.Log = new(migrationLogger)

				currentVersion, _, err := m.Version()
				if err != nil && err != migrate.ErrNilVersion {
					fmt.Printf("error getting migration version: %s", err)
					os.Exit(1)
				}

				logrus.Println("rolling back the last migration")
				if err := m.Steps(-1); err != nil {
					if err := m.Force(int(currentVersion)); err != nil {
						fmt.Printf("error forcing version: %s", err)
					}

					fmt.Printf("error rolling back: %s\n", err)
					os.Exit(1)
				}
			},
		}
		RootCmd.AddCommand(rollbackCmd)
	}()

	func() {
		var budget float64
		var tickSizeMinutes uint
		var longPeriod, shortPeriod uint
		var downThreshold, upThreshold float64
		var product string
		ts := &cobra.Command{
			Use:   "create-ema",
			Short: "creates an ema trading strategy",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				if _, ok := ProductToMetadata[Product(product)]; !ok {
					fmt.Println("unknown product: ", product)
					os.Exit(1)
				}

				if budget == 0 {
					fmt.Println("budget must be specified")
					os.Exit(1)
				}

				cs := &EMACrossoverStrategy{
					ShortPeriod:   shortPeriod,
					LongPeriod:    longPeriod,
					UpThreshold:   upThreshold,
					DownThreshold: downThreshold,
				}
				s := &TradingStrategyModel{
					NextTickAt:       CandlestickBucket(time.Now(), int64(tickSizeMinutes)).Add(time.Minute * time.Duration(tickSizeMinutes)),
					Product:          Product(product),
					HistoryTicks:     1000,
					State:            StrategyStateTryingToBuy,
					InitialBudget:    budget,
					Budget:           budget,
					BudgetCurrency:   ProductToMetadata[Product(product)].MarketOrderBuyCurrency,
					InvestedCurrency: ProductToMetadata[Product(product)].MarketOrderSellCurrency,
					TickSizeMinutes:  tickSizeMinutes,
					TradingStrategy:  TradingStrategyEMACrossover,
				}
				if err := s.SetStrategy(cs); err != nil {
					fmt.Printf("error setting trading strategy: %s", err)
					os.Exit(1)
				}
				if err := runner.Backend.CreateTradingStrategy(s); err != nil {
					fmt.Printf("error creating trading strategy: %s", err)
					os.Exit(1)
				}

				fmt.Printf("successfully created trading strategy: %d\n", s.ID)
			},
		}
		RootCmd.AddCommand(ts)
		ts.Flags().Float64VarP(&budget, "budget", "b", 0, "the initial budget in USD")
		ts.Flags().UintVarP(&tickSizeMinutes, "tick-size-minutes", "t", 15, "the size of each tick")
		ts.Flags().UintVar(&shortPeriod, "short-period", 10, "the EMA short period")
		ts.Flags().UintVar(&longPeriod, "long-period", 25, "the EMA long period")
		ts.Flags().Float64Var(&downThreshold, "down-threshold", 0, "the EMA down threshold")
		ts.Flags().Float64Var(&upThreshold, "up-threshold", 0, "the EMA up threshold")
		ts.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the product to use")
	}()

	func() {
		var invested float64
		var budget float64
		var tickSizeMinutes uint
		var emaLongPeriod, emaShortPeriod uint
		var emaDownThreshold, emaUpThreshold float64
		var rsiEntrance, rsiExit float64
		var product string
		ts := &cobra.Command{
			Use:   "create-s1",
			Short: "creates an s1 trading strategy",
			Run: func(cmd *cobra.Command, _ []string) {
				runner, err := GetRunner()
				if err != nil {
					fmt.Printf("error getting runner: %s", err)
					os.Exit(1)
				}

				if _, ok := ProductToMetadata[Product(product)]; !ok {
					fmt.Println("unknown product: ", product)
					os.Exit(1)
				}

				if budget == 0 {
					fmt.Println("budget must be specified")
					os.Exit(1)
				}

				strategy := &S1Strategy{
					EMAShortPeriod:       emaShortPeriod,
					EMALongPeriod:        emaLongPeriod,
					EMADownThreshold:     emaDownThreshold,
					EMAUpThreshold:       emaUpThreshold,
					RSIEntranceThreshold: rsiEntrance,
					RSIExitThreshold:     rsiExit,
				}
				s := &TradingStrategyModel{
					NextTickAt:       CandlestickBucket(time.Now(), int64(tickSizeMinutes)).Add(time.Minute * time.Duration(tickSizeMinutes)),
					Product:          Product(product),
					HistoryTicks:     1000,
					State:            StrategyStateTryingToBuy,
					InitialBudget:    budget,
					Budget:           budget,
					BudgetCurrency:   ProductToMetadata[Product(product)].MarketOrderBuyCurrency,
					InvestedCurrency: ProductToMetadata[Product(product)].MarketOrderSellCurrency,
					TickSizeMinutes:  tickSizeMinutes,
				}
				if invested > 0 {
					s.Invested = invested
					s.Budget = 0
					s.State = StrategyStateTryingToSell
				}

				if err := s.SetStrategy(strategy); err != nil {
					fmt.Printf("error setting trading strategy: %s", err)
					os.Exit(1)
				}
				if err := runner.Backend.CreateTradingStrategy(s); err != nil {
					fmt.Printf("error creating trading strategy: %s", err)
					os.Exit(1)
				}

				fmt.Printf("successfully created trading strategy: %d\n", s.ID)
			},
		}
		RootCmd.AddCommand(ts)
		ts.Flags().Float64VarP(&invested, "invested", "i", 0, "the invested amount in BTC")
		ts.Flags().Float64VarP(&budget, "budget", "b", 0, "the initial budget in USD")
		ts.Flags().UintVarP(&tickSizeMinutes, "tick-size-minutes", "t", 15, "the size of each tick")
		ts.Flags().UintVar(&emaShortPeriod, "ema-short-period", 10, "the EMA short period")
		ts.Flags().UintVar(&emaLongPeriod, "ema-long-period", 25, "the EMA long period")
		ts.Flags().Float64Var(&emaDownThreshold, "ema-down-threshold", 0, "the EMA down threshold")
		ts.Flags().Float64Var(&emaUpThreshold, "ema-up-threshold", 0, "the EMA up threshold")
		ts.Flags().Float64Var(&rsiEntrance, "rsi-entrance", 0, "the RSI entrance threshold")
		ts.Flags().Float64Var(&rsiExit, "rsi-exit", 100, "the RSI exit threshold")
		ts.Flags().StringVar(&product, "product", string(ProductBTCUSD), "the product to use")
	}()
}

type config struct {
	configFile         string
	postgresURI        string
	gdaxPassphrase     string
	gdaxAPIKey         string
	gdaxAPISecret      string
	useFakeExchange    bool
	slackToken         string
	slackTradesChannel string
	slackDataChannel   string
	rollbarToken       string
}

var appConfig = new(config)

// Runner contains singletons exported by the package.
type Runner struct {
	Backend                  Backend
	BTCUSDBot                *Bot
	ETHUSDBot                *Bot
	LTCUSDBot                *Bot
	BTCUSDRealtimeImporter   *RealtimeImporter
	ETHUSDRealtimeImporter   *RealtimeImporter
	LTCUSDRealtimeImporter   *RealtimeImporter
	BTCUSDHistoricalImporter *HistoricalImporter
	ETHUSDHistoricalImporter *HistoricalImporter
	LTCUSDHistoricalImporter *HistoricalImporter
	GDAXExchange             Exchange
	KrakenExchange           Exchange
}

var (
	appRunner  *Runner
	runnerOnce sync.Once
)

// GetRunner returns the main Runner instance.
func GetRunner() (*Runner, error) {
	if appRunner != nil {
		return appRunner, nil
	}

	createRunner := func() error {
		logrus.SetOutput(os.Stdout)

		if os.Getenv("VESPYR_LOG_JSON") != "" {
			logrus.SetFormatter(&logrus.JSONFormatter{
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyLevel: "severity",
				},
			})
		}

		switch os.Getenv("VESPYR_LOG") {
		case "DEBUG":
			logrus.SetLevel(logrus.DebugLevel)
		case "INFO":
			logrus.SetLevel(logrus.InfoLevel)
		case "WARN":
			logrus.SetLevel(logrus.WarnLevel)
		case "ERROR":
			logrus.SetLevel(logrus.ErrorLevel)
		case "FATAL":
			logrus.SetLevel(logrus.FatalLevel)
		default:
			logrus.SetLevel(logrus.InfoLevel)
		}

		pgConfig, err := pg.ParseURL(viper.GetString("postgres"))
		if err != nil {
			return err
		}
		db := pg.Connect(pgConfig)

		gdaxClient := coinbase.NewClient(
			viper.GetString("gdax_api_secret"),
			viper.GetString("gdax_api_key"),
			viper.GetString("gdax_passphrase"),
		)

		krakenClient := krakenapi.New(viper.GetString("kraken_key"), viper.GetString("kraken_secret"))

		var gdax, kraken Exchange
		if viper.GetBool("use_fake_exchange") {
			gdax = NewFakeGDAXExchange(clockwork.NewRealClock())
			kraken = NewFakeKrakenExchange(krakenClient, clockwork.NewRealClock())
		} else {
			gdax = NewGDAXExchange(gdaxClient, clockwork.NewRealClock())
			kraken = NewKrakenExchange(krakenClient, clockwork.NewRealClock())
		}

		backend := NewDBConn(db)

		if slackToken := viper.GetString("slack_token"); slackToken != "" {
			slackNotifier = NewSlackNotifier(
				viper.GetString("slack_trades_channel"),
				viper.GetString("slack_data_channel"),
				slack.New(slackToken),
			)
		}

		if rollbarToken := viper.GetString("rollbar_token"); rollbarToken != "" {
			rollbarHook := rollrus.NewHook(rollbarToken, viper.GetString("env"))
			logrus.AddHook(rollbarHook)
		}

		appRunner = new(Runner)
		appRunner.BTCUSDBot = NewBot(time.Second, clockwork.NewRealClock(), backend, gdax, ProductBTCUSD)
		appRunner.ETHUSDBot = NewBot(time.Second, clockwork.NewRealClock(), backend, gdax, ProductETHUSD)
		appRunner.LTCUSDBot = NewBot(time.Second, clockwork.NewRealClock(), backend, gdax, ProductLTCUSD)
		appRunner.BTCUSDRealtimeImporter = NewRealtimeImporter(ProductBTCUSD, backend, gdax)
		appRunner.ETHUSDRealtimeImporter = NewRealtimeImporter(ProductETHUSD, backend, gdax)
		appRunner.LTCUSDRealtimeImporter = NewRealtimeImporter(ProductLTCUSD, backend, gdax)
		appRunner.BTCUSDHistoricalImporter = NewHistoricalImporter(ProductBTCUSD, gdax, backend, 6)
		appRunner.LTCUSDHistoricalImporter = NewHistoricalImporter(ProductLTCUSD, gdax, backend, 6)
		appRunner.ETHUSDHistoricalImporter = NewHistoricalImporter(ProductETHUSD, gdax, backend, 6)
		appRunner.Backend = backend
		appRunner.GDAXExchange = gdax
		appRunner.KrakenExchange = kraken

		return nil
	}
	runnerOnce.Do(func() {
		if err := createRunner(); err != nil {
			fmt.Printf("error creating runner: %s", err)
			os.Exit(1)
		}
	})

	return appRunner, nil
}

func init() {
	viper.AutomaticEnv()
	defineCommands()
	cobra.OnInitialize(initConfig)

	// External config file
	RootCmd.PersistentFlags().StringVarP(&appConfig.configFile, "config-file", "c", "", "an optional configuration file")
	viper.BindPFlag("config_file", RootCmd.PersistentFlags().Lookup("config-file"))

	// GDAX
	RootCmd.PersistentFlags().StringVar(&appConfig.gdaxPassphrase, "gdax-passphrase", "", "the GDAX API passphrase")
	viper.BindPFlag("gdax_passphrase", RootCmd.PersistentFlags().Lookup("gdax-passphrase"))
	RootCmd.PersistentFlags().StringVar(&appConfig.gdaxAPIKey, "gdax-api-key", "", "the GDAX API key")
	viper.BindPFlag("gdax_api_key", RootCmd.PersistentFlags().Lookup("gdax-api-key"))
	RootCmd.PersistentFlags().StringVar(&appConfig.gdaxAPISecret, "gdax-api-secret", "", "the GDAX API secret")
	viper.BindPFlag("gdax_api_secret", RootCmd.PersistentFlags().Lookup("gdax-api-secret"))
	RootCmd.PersistentFlags().BoolVar(&appConfig.useFakeExchange, "use-fake-exchange", false, "use a fake exchange")
	viper.BindPFlag("use_fake_exchange", RootCmd.PersistentFlags().Lookup("use-fake-exchange"))

	// Slack
	RootCmd.PersistentFlags().StringVar(&appConfig.slackToken, "slack-token", "", "the Slack API token")
	viper.BindPFlag("slack_token", RootCmd.PersistentFlags().Lookup("slack-token"))
	RootCmd.PersistentFlags().StringVar(&appConfig.slackTradesChannel, "slack-trades-channel", "#trades-dev", "the trades Slack channel")
	viper.BindPFlag("slack_trades_channel", RootCmd.PersistentFlags().Lookup("slack-trades-channel"))
	RootCmd.PersistentFlags().StringVar(&appConfig.slackDataChannel, "slack-data-channel", "#data-dev", "the data Slack channel")
	viper.BindPFlag("slack_data_channel", RootCmd.PersistentFlags().Lookup("slack-data-channel"))

	// Rollbar
	RootCmd.PersistentFlags().StringVar(&appConfig.rollbarToken, "rollbar-token", "", "the Rollbar API token")
	viper.BindPFlag("rollbar_token", RootCmd.PersistentFlags().Lookup("rollbar-token"))

	// Postgres
	RootCmd.PersistentFlags().StringVar(&appConfig.postgresURI,
		"postgres", "postgres://vespyr:password@localhost:5432/vespyr?sslmode=disable",
		"the postgres URI")
	viper.BindPFlag("postgres", RootCmd.PersistentFlags().Lookup("postgres"))

	if configFile := viper.GetString("config_file"); configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("Can't read config:", err)
			os.Exit(1)
		}
	}
}

func initConfig() {
	if appConfig.configFile != "" {
		viper.SetConfigFile(appConfig.configFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Println("Can't read config:", err)
			os.Exit(1)
		}
	}
}
