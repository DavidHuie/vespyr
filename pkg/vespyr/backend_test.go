package vespyr_test

import (
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/assert"
)

func TestBackend(t *testing.T) {
	runner, err := vespyr.GetRunner()
	if err != nil {
		t.Fatal(err)
	}

	backend := runner.Backend

	startTime := time.Now()

	t.Run("CandlestickModel", func(t *testing.T) {
		c := &vespyr.CandlestickModel{
			StartTime: startTime,
			EndTime:   startTime.Add(time.Minute),
			Low:       2800,
			High:      3000,
			Open:      2900,
			Close:     2950,
			Volume:    100,
			Direction: vespyr.CandlestickDirectionUp,
			Product:   vespyr.ProductBTCUSD,
		}

		assert.NoError(t, backend.UpsertCandlestick(c))
		id := c.ID

		c.Close = 10000
		assert.NoError(t, backend.UpsertCandlestick(c))

		candle, err := backend.FindCandlestickByID(id)
		assert.NoError(t, err)
		assert.Equal(t, id, candle.ID)
		assert.Equal(t, float64(10000), candle.Close)

		recent, err := backend.FindMostRecentCandlestick(vespyr.ProductBTCUSD)
		assert.NoError(t, err)
		assert.Equal(t, candle, recent)

		candles, err := backend.FindCandlesticks(startTime,
			startTime.Add(time.Minute), vespyr.ProductBTCUSD, 60)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(candles))
		assert.True(t, candles[0].Volume > 0)
	})

	t.Run("MarketOrderModel", func(t *testing.T) {
		ts := &vespyr.TradingStrategyModel{
			NextTickAt:       startTime,
			Product:          vespyr.ProductBTCUSD,
			HistoryTicks:     2,
			State:            vespyr.StrategyStateTryingToBuy,
			InitialBudget:    100,
			Budget:           100,
			BudgetCurrency:   vespyr.CurrencyUSD,
			InvestedCurrency: vespyr.CurrencyBTC,
			TickSizeMinutes:  1,
		}
		assert.NoError(t, backend.CreateTradingStrategy(ts))

		m := &vespyr.MarketOrderModel{
			TradingStrategyID: ts.ID,
			ExchangeID:        "order-id",
			Product:           vespyr.ProductBTCUSD,
			Side:              vespyr.OrderBuy,
			Cost:              100,
			CostCurrency:      vespyr.CurrencyUSD,
			FilledSize:        .01,
			SizeCurrency:      vespyr.CurrencyBTC,
			Fees:              .0001,
			FeesCurrency:      vespyr.CurrencyUSD,
		}
		assert.NoError(t, backend.CreateMarketOrder(m))

		_, err := backend.FindMarketOrderByID(m.ID)
		assert.NoError(t, err)
	})

	t.Run("TradingStrategyModel", func(t *testing.T) {
		ts := &vespyr.TradingStrategyModel{
			NextTickAt:       startTime,
			Product:          vespyr.ProductBTCUSD,
			HistoryTicks:     2,
			State:            vespyr.StrategyStateTryingToBuy,
			InitialBudget:    100,
			Budget:           100,
			BudgetCurrency:   vespyr.CurrencyUSD,
			InvestedCurrency: vespyr.CurrencyBTC,
			TickSizeMinutes:  1,
		}
		assert.NoError(t, backend.CreateTradingStrategy(ts))

		ts.State = "new-state"
		assert.NoError(t, backend.UpdateTradingStrategy(ts))

		foundTS, err := backend.FindTradingStrategyByID(ts.ID)
		assert.NoError(t, err)
		assert.Equal(t, "new-state", foundTS.State)

		results, err := backend.FindActiveTradingStrategies(vespyr.ProductBTCUSD)
		assert.NoError(t, err)
		assert.True(t, len(results) > 0)
	})
}
