package vespyr_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCalculatePriceWithSlippage(t *testing.T) {
	source := rand.NewSource(1)
	for i := 0; i < 100; i++ {
		orderSlippage := .05
		newPrice := vespyr.CalculatePriceWithSlippage(source, 15, orderSlippage)
		assert.True(t, newPrice < 15*(1+orderSlippage))
		assert.True(t, newPrice > 15*(1-orderSlippage))
	}
}

func TestBacktesterExchangeCreateMarketOrder(t *testing.T) {
	t.Run("buy", func(t *testing.T) {
		candles := []*vespyr.CandlestickModel{
			&vespyr.CandlestickModel{
				Close: 4400,
			},
		}
		exchange := vespyr.NewBacktesterExchange(candles, .05, rand.NewSource(1))
		exchange.NextTick()

		response, err := exchange.CreateMarketOrder(&vespyr.MarketOrder{
			Product: vespyr.ProductBTCUSD,
			Side:    vespyr.OrderBuy,
			Cost:    8800,
		})
		if assert.NoError(t, err) {
			assert.Equal(t, float64(2.057195212480436), response.FilledSize)
			assert.Equal(t, float64(22), response.Fees)
		}
	})
	t.Run("sell", func(t *testing.T) {
		candles := []*vespyr.CandlestickModel{
			&vespyr.CandlestickModel{
				Close: 4200,
			},
		}
		exchange := vespyr.NewBacktesterExchange(candles, .05, rand.NewSource(1))
		exchange.NextTick()

		response, err := exchange.CreateMarketOrder(&vespyr.MarketOrder{
			Product: vespyr.ProductBTCUSD,
			Side:    vespyr.OrderSell,
			Cost:    2,
		})
		if assert.NoError(t, err) {
			assert.Equal(t, float64(8125.677572350939), response.FilledSize)
			assert.Equal(t, float64(20.3651066976214), response.Fees)
		}
	})
}

func TestBacktest(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToBuy,
		InitialBudget:    500,
		Budget:           500,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}
	strategy := &vespyr.EMACrossoverStrategy{
		ShortPeriod: 1,
		LongPeriod:  5,
	}

	assert.NoError(t, model.SetStrategy(strategy))

	endTime := time.Now()
	startTime := endTime.Add(-time.Minute * 30)

	backend := new(vespyr.MockBackend)
	backend.On("FindCandlesticks", startTime.Add(-15*time.Minute), endTime, model.Product, int64(model.TickSizeMinutes)).
		Return([]*vespyr.CandlestickModel{
			&vespyr.CandlestickModel{Close: 10, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 10, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 9, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 8, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 11, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 11, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 9, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
			&vespyr.CandlestickModel{Close: 8, Volume: 1, EndTime: startTime.Add(100 * time.Minute)},
		}, nil)
	defer mock.AssertExpectationsForObjects(t, backend)

	backtester, err := vespyr.NewBacktester(startTime, endTime,
		model, backend, rand.NewSource(1))

	assert.NoError(t, err)
	assert.NoError(t, backtester.Backtest())

	results := backtester.Results()

	assert.Equal(t, &vespyr.BacktestResults{
		BudgetCurrency:       vespyr.CurrencyUSD,
		InitialBudget:        500,
		FinalBudget:          405.67262907,
		TradeCurrency:        vespyr.CurrencyBTC,
		InitialCurrencyPrice: 11,
		FinalCurrencyPrice:   9,
		GrossProfit:          0,
		GrossLoss:            94.32737092755264,
		ProfitTrades:         0,
		LossTrades:           1,
	}, results)
}
