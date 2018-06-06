package vespyr

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"encoding/csv"

	"math"

	"github.com/jonboulle/clockwork"
	"github.com/montanaflynn/stats"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var (
	exchangeFee      = float64(.0025)
	exchangeSlippage = float64(0)
)

// BacktesterBackend is a custom backend used specifically for
// backtesting that registers all model updates.
type BacktesterBackend struct {
	Backend
	// backend          Backend
	resultCalculator *backtestResultCalculator
}

// UpsertCandlestick is a noop.
func (b *BacktesterBackend) UpsertCandlestick(*CandlestickModel) error {
	return nil
}

// FindCandlesticks passes the call over to the underlying backend.
func (b *BacktesterBackend) FindCandlesticks(startTime, endTime time.Time,
	p Product, tickSizeMinutes int64) ([]*CandlestickModel, error) {
	return b.Backend.FindCandlesticks(startTime, endTime, p, tickSizeMinutes)
}

// CreateMarketOrder registers the market order as the current market
// order.
func (b *BacktesterBackend) CreateMarketOrder(m *MarketOrderModel) error {
	b.resultCalculator.current.marketOrder = m
	return nil
}

// UpdateTradingStrategy updates the trading strategy metadata.
func (b *BacktesterBackend) UpdateTradingStrategy(m *TradingStrategyModel) error {
	b.resultCalculator.current.budget = m.Budget
	b.resultCalculator.current.invested = m.Invested
	return nil
}

// BacktesterExchange is a mock exchange that can perform mock trades.
type BacktesterExchange struct {
	candles             []*CandlestickModel
	marketOrderSlippage float64
	current             int
	randSource          rand.Source
}

// NewBacktesterExchange instantiates a new backtester exchange.
func NewBacktesterExchange(candles []*CandlestickModel, marketOrderSlippage float64,
	randSource rand.Source) *BacktesterExchange {
	return &BacktesterExchange{
		marketOrderSlippage: marketOrderSlippage,
		candles:             candles,
		current:             -1,
		randSource:          randSource,
	}
}

// NextTick advances the exchange to the next candle.
func (b *BacktesterExchange) NextTick() {
	b.current++
}

// GetMessageChan is a noop.
func (b *BacktesterExchange) GetMessageChan(context.Context, Product) (<-chan *ExchangeMessage, error) {
	panic("not implemented")
}

// GetCandlesticks is a noop.
func (b *BacktesterExchange) GetCandlesticks(p Product, start time.Time, end time.Time, granularity int) ([]*CandlestickModel, error) {
	panic("not implemented")
}

func (b *BacktesterExchange) StreamCandlesticks(ctx context.Context, product Product) (<-chan *CandlestickModel, error) {
	panic("not implemented")
}

func (k *BacktesterExchange) EmitsFullCandlesticks() bool {
	panic("not implemented")
}

// CalculatePriceWithSlippage calculates a price taking into account
// slippage.
func CalculatePriceWithSlippage(source rand.Source, price, orderSlippage float64) float64 {
	r := rand.New(source)
	slippage := r.Float64() * orderSlippage
	return price * (1 - slippage)
}

// CreateMarketOrder creates a mock market order taking into account
// a slippage percentage.
func (b *BacktesterExchange) CreateMarketOrder(m *MarketOrder) (*CreateMarketOrderResponse, error) {
	price := CalculatePriceWithSlippage(b.randSource, b.candles[b.current].Close,
		b.marketOrderSlippage)

	response := &CreateMarketOrderResponse{
		ExchangeID: uuid.NewV4().String(),
	}

	logrus.Infof("exchange price: %f", b.candles[b.current].Close)

	if m.Side == OrderBuy {
		response.FilledSize = m.Cost * (1 - exchangeFee) / price
		response.FilledSizeCurrency = CurrencyBTC
		response.Fees = m.Cost * exchangeFee
		response.FeesCurrency = CurrencyUSD
	} else {
		response.FilledSize = m.Cost * price
		response.FilledSizeCurrency = CurrencyUSD
		response.Fees = response.FilledSize * exchangeFee
		response.FeesCurrency = CurrencyUSD
		response.FilledSize = response.FilledSize * (1 - exchangeFee)
	}

	return response, nil
}

// Backtester backtests a trading algorithm with historical data.
type Backtester struct {
	startTime        time.Time
	endTime          time.Time
	model            *TradingStrategyModel
	strategy         StrategyInterface
	backend          *BacktesterBackend
	resultCalculator *backtestResultCalculator
	source           rand.Source
}

// NewBacktester creates a new Backtester.
func NewBacktester(startTime, endTime time.Time, model *TradingStrategyModel,
	backend Backend, source rand.Source) (*Backtester, error) {
	strategy, err := model.Strategy()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting model strategy")
	}
	calc := newBacktestResultCalculator()

	return &Backtester{
		startTime:        startTime,
		endTime:          endTime,
		model:            model,
		strategy:         strategy,
		backend:          &BacktesterBackend{backend, calc},
		resultCalculator: calc,
		source:           source,
	}, nil
}

func generateIndicatorSets(indicators []Indicator, candles []*CandlestickModel) ([]*IndicatorSet, []*CandlestickModel, error) {
	var indicatorSets []*IndicatorSet
	var validCandles []*CandlestickModel
	for _, c := range candles {
		if c.Volume == 0 {
			continue
		}

		var set IndicatorSet
		set.Time = c.StartTime
		skipSet := false
		for _, indicator := range indicators {
			if err := indicator.AddCandlestick(c); err != nil {
				return nil, nil, errors.Wrapf(err, "error adding candlestick to indicator: %s", indicator.Name())
			}
			value, err := indicator.Value()
			if err != nil {
				if errors.Cause(err) == ErrNotEnoughData {
					skipSet = true
				} else {
					return nil, nil, errors.Wrapf(err, "error calculating value for indicator: %s", indicator.Name())
				}
			}
			set.Values = append(set.Values, value)
		}
		if !skipSet {
			indicatorSets = append(indicatorSets, &set)
			validCandles = append(validCandles, c)
		}
	}
	return indicatorSets, validCandles, nil
}

// Backtest performs the backtest.
func (b *Backtester) Backtest() error {
	logrus.Debugf("starting backtest")
	indicators := b.strategy.Indicators()

	actualStartTime := b.startTime.Add(-time.Duration(b.model.TickSizeMinutes) *
		time.Duration(b.model.HistoryTicks) * time.Minute)

	logrus.Debugf("finding candles")
	candles, err := b.backend.FindCandlesticks(actualStartTime, b.endTime,
		b.model.Product, int64(b.model.TickSizeMinutes))
	if err != nil {
		return errors.Wrapf(err, "error finding candlesticks")
	}

	logrus.Debugf("generating indicator sets")
	indicatorSets, validCandles, err := generateIndicatorSets(indicators, candles)
	if err != nil {
		return errors.Wrapf(err, "error generating indicator sets")
	}
	exchange := NewBacktesterExchange(validCandles, exchangeSlippage, b.source)

	trader := NewTradingStrategy(b.backend,
		exchange, b.strategy, clockwork.NewRealClock())
	trader.setIndicatorSets(indicatorSets)

	startBucket := CandlestickBucket(b.startTime, int64(b.model.TickSizeMinutes))
	for i, sets := range indicatorSets {
		b.resultCalculator.current.indicatorSet = sets
		b.resultCalculator.current.candle = validCandles[i]
		b.resultCalculator.current.time = validCandles[i].StartTime
		b.resultCalculator.current.budget = b.model.Budget
		b.resultCalculator.current.invested = b.model.Invested

		trader.nextTick()
		exchange.NextTick()

		// Only process ticks after we've exhausted the
		// history.
		if validCandles[i].EndTime.After(startBucket) {
			if err := trader.ProcessTick(b.model); err != nil {
				if errors.Cause(err) == ErrNotEnoughData {
					continue
				}
				return errors.Wrapf(err, "error processing tick")
			}
		}

		b.resultCalculator.next()
	}

	logrus.Debugf("finished backtest")

	return nil
}

// BacktestResults contains information about the results of the
// backtest.
type BacktestResults struct {
	BudgetCurrency       string
	InitialBudget        float64
	FinalBudget          float64
	TradeCurrency        string
	InitialCurrencyPrice float64
	FinalCurrencyPrice   float64
	GrossProfit          float64
	GrossLoss            float64
	ProfitTrades         uint
	LossTrades           uint
	PortfolioValuePerDay []float64
}

// https://www.mql5.com/en/articles/1486
func (b *BacktestResults) NetProfit() float64 {
	return b.GrossProfit - b.GrossLoss
}

// https://www.mql5.com/en/articles/1486
func (b *BacktestResults) ProfitFactor() float64 {
	return b.GrossProfit / b.GrossLoss
}

// https://www.mql5.com/en/articles/1486
func (b *BacktestResults) ExpectedPayoff() float64 {
	totalTrades := float64(b.ProfitTrades + b.LossTrades)
	return (float64(b.ProfitTrades)/totalTrades)*(b.GrossProfit/float64(b.ProfitTrades)) -
		(float64(b.LossTrades)/totalTrades)*(b.GrossLoss/float64(b.LossTrades))
}

func (b *BacktestResults) SharpeRatio() float64 {
	increases := []float64{0}
	for i := 1; i < len(b.PortfolioValuePerDay); i++ {
		increases = append(increases, b.PortfolioValuePerDay[i]-b.PortfolioValuePerDay[i-1])
	}

	mean, err := stats.Mean(increases)
	if err != nil {
		panic(err)
	}
	stddev, err := stats.StandardDeviation(increases)
	if err != nil {
		panic(err)
	}

	return math.Pow(365, .5) * mean / stddev
}

// Results returns the results of the backtest.
func (b *Backtester) Results() *BacktestResults {
	results := &BacktestResults{
		InitialBudget:  b.model.InitialBudget,
		BudgetCurrency: b.model.BudgetCurrency,
		TradeCurrency:  b.model.InvestedCurrency,
	}
	if len(b.resultCalculator.results) == 0 {
		return results
	}

	currentDay := b.resultCalculator.results[0].candle.StartTime
	portfolioValue := b.model.InitialBudget
	currentBudget := b.model.InitialBudget
	for _, result := range b.resultCalculator.results {
		if result.budget != 0 {
			portfolioValue = result.budget
		} else {
			portfolioValue = result.invested * result.candle.Close * (1 - exchangeFee - exchangeSlippage)
		}

		if currentDay.After(b.startTime) &&
			result.candle.StartTime.After(currentDay) &&
			result.candle.StartTime.Weekday() != currentDay.Weekday() {
			results.PortfolioValuePerDay = append(results.PortfolioValuePerDay, portfolioValue)
		}

		currentDay = result.candle.StartTime

		if result.marketOrder != nil && result.marketOrder.Side == OrderSell {
			diff := result.marketOrder.FilledSize - currentBudget
			if diff >= 0 {
				results.GrossProfit += diff
				results.ProfitTrades++
			} else {
				results.GrossLoss += -diff
				results.LossTrades++
			}
			currentBudget = TruncateFloat(result.marketOrder.FilledSize, tradeCurrencyPrecision)
		}
		if result.marketOrder != nil && result.marketOrder.Side == OrderBuy &&
			results.InitialCurrencyPrice == 0 {
			results.InitialCurrencyPrice = result.candle.Close
		}
		if result.marketOrder != nil && result.marketOrder.Side == OrderSell {
			results.FinalBudget = TruncateFloat(result.marketOrder.FilledSize, tradeCurrencyPrecision)
			results.FinalCurrencyPrice = result.candle.Close
		}
	}

	return results
}

// ResultsCSV returns a CSV with the granular details about the
// backtest
func (b *Backtester) ResultsCSV() (io.Reader, error) {
	return b.resultCalculator.resultsCSV()
}

type backtestResult struct {
	time         time.Time
	candle       *CandlestickModel
	indicatorSet *IndicatorSet
	marketOrder  *MarketOrderModel
	budget       float64
	invested     float64
}

type backtestResultCalculator struct {
	current *backtestResult
	results []*backtestResult
}

func newBacktestResultCalculator() *backtestResultCalculator {
	return &backtestResultCalculator{
		current: &backtestResult{},
	}
}

func (b *backtestResultCalculator) resultsCSV() (io.Reader, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// This is just used to extract the indicators
	firstRow := b.results[0]

	columns := []string{
		"start_time",
		"end_time",
		"low",
		"high",
		"open",
		"close",
		"volume",
		"bought_size",
		"sold_size",
		"budget",
		"invested",
	}
	for _, set := range firstRow.indicatorSet.Values {
		columns = append(columns, set.IndicatorName)
	}

	if err := writer.Write(columns); err != nil {
		return nil, errors.Wrapf(err, "error writing to csv writer")
	}

	for _, result := range b.results {
		row := []string{
			result.candle.StartTime.Format(time.RFC3339),
			result.candle.EndTime.Format(time.RFC3339),
			fmt.Sprintf("%f", result.candle.Low),
			fmt.Sprintf("%f", result.candle.High),
			fmt.Sprintf("%f", result.candle.Open),
			fmt.Sprintf("%f", result.candle.Close),
			fmt.Sprintf("%f", result.candle.Volume),
		}

		if result.marketOrder != nil {
			if result.marketOrder.Side == OrderBuy {
				row = append(row, fmt.Sprintf("%f", result.marketOrder.FilledSize))
			} else {
				row = append(row, "")
			}
			if result.marketOrder.Side == OrderSell {
				row = append(row, fmt.Sprintf("%f", result.marketOrder.FilledSize))
			} else {
				row = append(row, "")
			}
		} else {
			row = append(row, "", "")
		}

		row = append(row, fmt.Sprintf("%f", result.budget))
		row = append(row, fmt.Sprintf("%f", result.invested))

		for _, set := range result.indicatorSet.Values {
			row = append(row, fmt.Sprintf("%f", set.Value))
		}

		if err := writer.Write(row); err != nil {
			return nil, errors.Wrapf(err, "error writing csv row")
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, errors.Wrapf(err, "error writing to csv")
	}

	return buf, nil
}

func (b *backtestResultCalculator) next() {
	b.results = append(b.results, b.current)
	b.current = &backtestResult{}
}
