package vespyr_test

import (
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBotSingleTick(t *testing.T) {
	startTime := vespyr.CandlestickBucket(time.Now(), 1)

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)
	clock := clockwork.NewFakeClock()

	defer mock.AssertExpectationsForObjects(t, backend, exchange)

	backend.On("FindMostRecentCandlestick", vespyr.ProductBTCUSD).Return(
		&vespyr.CandlestickModel{
			ID:        1,
			StartTime: startTime.Add(-time.Minute),
			EndTime:   startTime,
			Low:       2400,
			High:      2800,
			Open:      2500,
			Close:     2600,
			Volume:    4,
			Direction: vespyr.CandlestickDirectionUp,
			Product:   vespyr.ProductBTCUSD,
		}, nil,
	).Once()

	tradingStrategy := &vespyr.TradingStrategyModel{
		ID:               123,
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

	backend.On("FindCandlesticks", startTime.Add(-2*time.Minute), startTime, vespyr.ProductBTCUSD, int64(1)).
		Return([]*vespyr.CandlestickModel{
			&vespyr.CandlestickModel{
				StartTime: startTime.Add(-2 * time.Minute),
				EndTime:   startTime.Add(-1 * time.Minute),
				Low:       2400,
				High:      2800,
				Open:      2500,
				Close:     2600,
				Volume:    4,
				Direction: vespyr.CandlestickDirectionUp,
				Product:   vespyr.ProductBTCUSD,
			},
			&vespyr.CandlestickModel{
				StartTime: startTime.Add(-1 * time.Minute),
				EndTime:   startTime,
				Low:       2400,
				High:      2800,
				Open:      2500,
				Close:     2600,
				Volume:    4,
				Direction: vespyr.CandlestickDirectionUp,
				Product:   vespyr.ProductBTCUSD,
			},
		}, nil).Once()

	assert.NoError(t, tradingStrategy.SetStrategy(&vespyr.EMACrossoverStrategy{
		ShortPeriod:   1,
		LongPeriod:    2,
		UpThreshold:   0,
		DownThreshold: 0,
	}))

	backend.On("FindActiveTradingStrategies", vespyr.ProductBTCUSD).Return([]*vespyr.TradingStrategyModel{tradingStrategy}, nil).Once()

	backend.On("UpdateTradingStrategy", tradingStrategy).Return(nil).Once()

	bot := vespyr.NewBot(time.Second, clock, backend,
		exchange, vespyr.ProductBTCUSD)

	// First attempt
	if err := bot.ProcessTickForNewCandlestick(); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, float64(100), tradingStrategy.Budget)
	assert.Equal(t, float64(0), tradingStrategy.Invested)
	assert.Equal(t, vespyr.CurrencyUSD, tradingStrategy.BudgetCurrency)
	assert.Equal(t, startTime, tradingStrategy.LastTickAt)
	assert.Equal(t, startTime.Add(time.Minute), tradingStrategy.NextTickAt)
}

func TestBotNoData(t *testing.T) {
	startTime := vespyr.CandlestickBucket(time.Now(), 1)

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)
	clock := clockwork.NewFakeClock()

	defer mock.AssertExpectationsForObjects(t, backend, exchange)

	backend.On("FindMostRecentCandlestick", vespyr.ProductBTCUSD).Return(
		&vespyr.CandlestickModel{
			ID:        1,
			StartTime: startTime.Add(-time.Minute),
			EndTime:   startTime,
			Low:       2400,
			High:      2800,
			Open:      2500,
			Close:     2600,
			Volume:    4,
			Direction: vespyr.CandlestickDirectionUp,
			Product:   vespyr.ProductBTCUSD,
		}, nil,
	).Once()

	tradingStrategy := &vespyr.TradingStrategyModel{
		ID:               123,
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

	backend.On("FindCandlesticks", startTime.Add(-2*time.Minute), startTime, vespyr.ProductBTCUSD, int64(1)).
		Return(nil, nil).Once()

	assert.NoError(t, tradingStrategy.SetStrategy(&vespyr.EMACrossoverStrategy{
		ShortPeriod:   1,
		LongPeriod:    2,
		UpThreshold:   0,
		DownThreshold: 0,
	}))

	backend.On("FindActiveTradingStrategies", vespyr.ProductBTCUSD).Return([]*vespyr.TradingStrategyModel{tradingStrategy}, nil).Once()

	backend.On("UpdateTradingStrategy", tradingStrategy).Return(nil).Once()

	bot := vespyr.NewBot(time.Second, clock, backend,
		exchange, vespyr.ProductBTCUSD)

	if err := bot.ProcessTickForNewCandlestick(); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, float64(100), tradingStrategy.Budget)
	assert.Equal(t, float64(0), tradingStrategy.Invested)
	assert.Equal(t, vespyr.CurrencyUSD, tradingStrategy.BudgetCurrency)
	assert.True(t, tradingStrategy.LastTickAt.IsZero())
	assert.Equal(t, startTime, tradingStrategy.NextTickAt)
	assert.Equal(t, startTime, tradingStrategy.DeactivatedAt)
}
