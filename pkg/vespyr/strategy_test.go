package vespyr_test

import (
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTryBuyWithBuy(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToBuy,
		InitialBudget:    500,
		Budget:           500,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
		TradingStrategy:  vespyr.TradingStrategyEMACrossover,
	}
	ema := &vespyr.EMACrossoverStrategy{
		ShortPeriod: 1,
		LongPeriod:  2,
	}
	assert.NoError(t, model.SetStrategy(ema))

	backend := new(vespyr.MockBackend)
	backend.On("CreateMarketOrder", &vespyr.MarketOrderModel{
		ExchangeID:        "asdf",
		TradingStrategyID: model.ID,
		Product:           model.Product,
		Side:              "buy",
		Cost:              500,
		CostCurrency:      "USD",
		FilledSize:        200,
		SizeCurrency:      "USD",
		Fees:              10,
		FeesCurrency:      "USD",
	}).Return(nil)
	backend.On("UpdateTradingStrategy", &vespyr.TradingStrategyModel{
		ID:                  69,
		Product:             vespyr.ProductBTCUSD,
		HistoryTicks:        1,
		State:               vespyr.StrategyStateTryingToSell,
		InitialBudget:       500,
		Budget:              0,
		BudgetCurrency:      vespyr.CurrencyUSD,
		InvestedCurrency:    vespyr.CurrencyBTC,
		Invested:            200,
		TickSizeMinutes:     15,
		TradingStrategy:     vespyr.TradingStrategyEMACrossover,
		TradingStrategyData: model.TradingStrategyData,
	}).Return(nil)

	exchange := new(vespyr.MockExchange)
	exchange.On("CreateMarketOrder", &vespyr.MarketOrder{
		Side:    "buy",
		Cost:    500,
		Product: vespyr.ProductBTCUSD,
	}).Return(&vespyr.CreateMarketOrderResponse{
		ExchangeID:         "asdf",
		FilledSize:         200,
		FilledSizeCurrency: "USD",
		Fees:               10,
		FeesCurrency:       "USD",
	}, nil)

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		ema,
		clock,
	)

	c1 := fakeCandlestick()
	assert.NoError(t, strategy.SeedIndicators(c1))

	c2 := fakeCandlestick()
	c2.Close = c1.Close + 1
	assert.NoError(t, strategy.SeedIndicators(c2))
	assert.NoError(t, strategy.TryBuy(model))
	mock.AssertExpectationsForObjects(t, backend, exchange)
}

func TestTryBuyWithNoop(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToBuy,
		InitialBudget:    500,
		Budget:           500,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}
	ema := &vespyr.EMACrossoverStrategy{
		ShortPeriod: 1,
		LongPeriod:  2,
		UpThreshold: 100,
	}
	assert.NoError(t, model.SetStrategy(ema))

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		ema,
		clock,
	)

	assert.NoError(t, strategy.SeedIndicators(fakeCandlestick()))
	assert.NoError(t, strategy.SeedIndicators(fakeCandlestick()))
	assert.NoError(t, strategy.TryBuy(model))
	mock.AssertExpectationsForObjects(t, backend, exchange)
}

func TestTryBuyAfterSeedingIndicators(t *testing.T) {
	startTime := time.Now()

	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToBuy,
		InitialBudget:    500,
		Budget:           500,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}

	indicator := new(vespyr.MockIndicator)
	indicator.On("AddCandlestick", mock.Anything).Return(nil)
	indicator.On("Value").Return(&vespyr.IndicatorValue{
		Time:          startTime,
		Value:         1234.234,
		IndicatorName: "test",
	}, nil).Once()
	indicator.On("Name").Return("test")

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)

	strategyImpl := new(vespyr.MockStrategyInterface)
	strategyImpl.On("String").Return("impl")
	strategyImpl.On("Indicators").Return([]vespyr.Indicator{indicator})
	strategyImpl.On("Buy", []*vespyr.IndicatorSet{
		&vespyr.IndicatorSet{
			Time: startTime,
			Values: []*vespyr.IndicatorValue{
				&vespyr.IndicatorValue{
					Time:          startTime,
					Value:         1234.234,
					IndicatorName: "test",
				},
			},
		},
	}, 0).Return(false, nil)

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		strategyImpl,
		clock,
	)
	assert.NoError(t, strategy.SeedIndicators(&vespyr.CandlestickModel{StartTime: startTime}))
	assert.NoError(t, strategy.TryBuy(model))
	mock.AssertExpectationsForObjects(t, indicator, backend, exchange, strategyImpl)
}

func TestTrySellWithSell(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToSell,
		InitialBudget:    500,
		Budget:           0,
		Invested:         200,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}
	ema := &vespyr.EMACrossoverStrategy{
		ShortPeriod: 1,
		LongPeriod:  2,
	}
	assert.NoError(t, model.SetStrategy(ema))

	backend := new(vespyr.MockBackend)
	backend.On("CreateMarketOrder", &vespyr.MarketOrderModel{
		ExchangeID:        "asdf",
		TradingStrategyID: model.ID,
		Product:           model.Product,
		Side:              "sell",
		Cost:              200,
		CostCurrency:      "BTC",
		FilledSize:        490,
		SizeCurrency:      "USD",
		Fees:              10,
		FeesCurrency:      "USD",
	}).Return(nil)
	backend.On("UpdateTradingStrategy", &vespyr.TradingStrategyModel{
		ID:                  69,
		Product:             vespyr.ProductBTCUSD,
		HistoryTicks:        1,
		State:               vespyr.StrategyStateTryingToBuy,
		InitialBudget:       500,
		Budget:              490,
		BudgetCurrency:      vespyr.CurrencyUSD,
		InvestedCurrency:    vespyr.CurrencyBTC,
		Invested:            0,
		TickSizeMinutes:     15,
		TradingStrategy:     vespyr.TradingStrategyEMACrossover,
		TradingStrategyData: model.TradingStrategyData,
	}).Return(nil)

	exchange := new(vespyr.MockExchange)
	exchange.On("CreateMarketOrder", &vespyr.MarketOrder{
		Side:    "sell",
		Cost:    200,
		Product: vespyr.ProductBTCUSD,
	}).Return(&vespyr.CreateMarketOrderResponse{
		ExchangeID:         "asdf",
		FilledSize:         490,
		FilledSizeCurrency: "USD",
		Fees:               10,
		FeesCurrency:       "USD",
	}, nil)

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		ema,
		clock,
	)

	c1 := fakeCandlestick()
	assert.NoError(t, strategy.SeedIndicators(c1))
	c2 := fakeCandlestick()
	c2.Close = c1.Close - 1
	c2.StartTime = c1.StartTime.Add(time.Minute)
	assert.NoError(t, strategy.SeedIndicators(c2))
	assert.NoError(t, strategy.TrySell(model))
	mock.AssertExpectationsForObjects(t, backend, exchange)
}

func TestTrySellWithSellAndDeactivation(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToSell,
		InitialBudget:    500,
		Budget:           0,
		Invested:         200,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}
	ema := &vespyr.EMACrossoverStrategy{
		ShortPeriod: 1,
		LongPeriod:  2,
	}
	assert.NoError(t, model.SetStrategy(ema))

	clock := clockwork.NewFakeClock()

	backend := new(vespyr.MockBackend)
	backend.On("CreateMarketOrder", &vespyr.MarketOrderModel{
		ExchangeID:        "asdf",
		TradingStrategyID: model.ID,
		Product:           model.Product,
		Side:              "sell",
		Cost:              200,
		CostCurrency:      "BTC",
		FilledSize:        250,
		SizeCurrency:      "USD",
		Fees:              10,
		FeesCurrency:      "USD",
	}).Return(nil)
	backend.On("UpdateTradingStrategy", &vespyr.TradingStrategyModel{
		ID:                  69,
		Product:             vespyr.ProductBTCUSD,
		HistoryTicks:        1,
		State:               vespyr.StrategyStateTryingToBuy,
		InitialBudget:       500,
		Budget:              250,
		BudgetCurrency:      vespyr.CurrencyUSD,
		InvestedCurrency:    vespyr.CurrencyBTC,
		Invested:            0,
		TickSizeMinutes:     15,
		TradingStrategy:     vespyr.TradingStrategyEMACrossover,
		TradingStrategyData: model.TradingStrategyData,
	}).Return(nil).Once()
	backend.On("UpdateTradingStrategy", &vespyr.TradingStrategyModel{
		ID:                  69,
		Product:             vespyr.ProductBTCUSD,
		HistoryTicks:        1,
		State:               vespyr.StrategyStateTryingToBuy,
		InitialBudget:       500,
		Budget:              250,
		BudgetCurrency:      vespyr.CurrencyUSD,
		InvestedCurrency:    vespyr.CurrencyBTC,
		Invested:            0,
		TickSizeMinutes:     15,
		TradingStrategy:     vespyr.TradingStrategyEMACrossover,
		TradingStrategyData: model.TradingStrategyData,
		DeactivatedAt:       clock.Now(),
	}).Return(nil).Once()

	exchange := new(vespyr.MockExchange)
	exchange.On("CreateMarketOrder", &vespyr.MarketOrder{
		Side:    "sell",
		Cost:    200,
		Product: vespyr.ProductBTCUSD,
	}).Return(&vespyr.CreateMarketOrderResponse{
		ExchangeID:         "asdf",
		FilledSize:         250,
		FilledSizeCurrency: "USD",
		Fees:               10,
		FeesCurrency:       "USD",
	}, nil)

	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		ema,
		clock,
	)

	c1 := fakeCandlestick()
	assert.NoError(t, strategy.SeedIndicators(c1))
	c2 := fakeCandlestick()
	c2.Close = c1.Close - 1
	c2.StartTime = c1.StartTime.Add(time.Minute)
	assert.NoError(t, strategy.SeedIndicators(c2))
	assert.NoError(t, strategy.TrySell(model))
	assert.False(t, model.DeactivatedAt.IsZero())
	mock.AssertExpectationsForObjects(t, backend, exchange)
}

func TestTrySellWithNoop(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToSell,
		InitialBudget:    500,
		Budget:           0,
		Invested:         200,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}
	ema := &vespyr.EMACrossoverStrategy{
		ShortPeriod: 1,
		LongPeriod:  2,
	}
	assert.NoError(t, model.SetStrategy(ema))

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		ema,
		clock,
	)
	assert.NoError(t, strategy.SeedIndicators(fakeCandlestick()))
	assert.NoError(t, strategy.SeedIndicators(fakeCandlestick()))
	assert.NoError(t, strategy.TrySell(model))
	mock.AssertExpectationsForObjects(t, backend, exchange)
}

func TestTrySellAfterSeedingIndicators(t *testing.T) {
	startTime := time.Now()

	indicator := new(vespyr.MockIndicator)
	indicator.On("AddCandlestick", mock.Anything).Return(nil)
	indicator.On("Value").Return(&vespyr.IndicatorValue{
		Time:          startTime,
		Value:         1234.234,
		IndicatorName: "test",
	}, nil)
	indicator.On("Name").Return("test")

	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToSell,
		InitialBudget:    500,
		Budget:           0,
		Invested:         200,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)

	strategyImpl := new(vespyr.MockStrategyInterface)
	strategyImpl.On("Indicators").Return([]vespyr.Indicator{indicator})
	strategyImpl.On("String").Return("impl")
	strategyImpl.On("Sell", []*vespyr.IndicatorSet{
		&vespyr.IndicatorSet{
			Time: startTime,
			Values: []*vespyr.IndicatorValue{
				&vespyr.IndicatorValue{
					Time:          startTime,
					Value:         1234.234,
					IndicatorName: "test",
				},
			},
		},
		&vespyr.IndicatorSet{
			Time: startTime,
			Values: []*vespyr.IndicatorValue{
				&vespyr.IndicatorValue{
					Time:          startTime,
					Value:         1234.234,
					IndicatorName: "test",
				},
			},
		},
	}, 1).Return(false, nil)

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		strategyImpl,
		clock,
	)

	assert.NoError(t, strategy.SeedIndicators(&vespyr.CandlestickModel{StartTime: startTime}))
	assert.NoError(t, strategy.SeedIndicators(&vespyr.CandlestickModel{StartTime: startTime}))
	assert.NoError(t, strategy.TrySell(model))
	mock.AssertExpectationsForObjects(t, indicator, backend, exchange, strategyImpl)
}

func TestTrySellWithoutHistory(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToSell,
		InitialBudget:    500,
		Budget:           0,
		Invested:         200,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)

	indicator := new(vespyr.MockIndicator)
	indicator.On("Name").Return("test")

	strategyImpl := new(vespyr.MockStrategyInterface)
	strategyImpl.On("Indicators").Return([]vespyr.Indicator{indicator})

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		strategyImpl,
		clock,
	)

	assert.Error(t, strategy.TrySell(model), vespyr.ErrNotEnoughData.Error())
	mock.AssertExpectationsForObjects(t, indicator, backend, exchange, strategyImpl)
}

func TestTryBuyWithoutHistory(t *testing.T) {
	model := &vespyr.TradingStrategyModel{
		ID:               69,
		Product:          vespyr.ProductBTCUSD,
		HistoryTicks:     1,
		State:            vespyr.StrategyStateTryingToBuy,
		InitialBudget:    500,
		Budget:           0,
		Invested:         200,
		BudgetCurrency:   vespyr.CurrencyUSD,
		InvestedCurrency: vespyr.CurrencyBTC,
		TickSizeMinutes:  15,
	}

	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)

	indicator := new(vespyr.MockIndicator)
	indicator.On("Name").Return("test")

	strategyImpl := new(vespyr.MockStrategyInterface)
	strategyImpl.On("Indicators").Return([]vespyr.Indicator{indicator})

	clock := clockwork.NewFakeClock()
	strategy := vespyr.NewTradingStrategy(
		backend,
		exchange,
		strategyImpl,
		clock,
	)

	assert.Error(t, strategy.TryBuy(model), vespyr.ErrNotEnoughData)
	mock.AssertExpectationsForObjects(t, indicator, backend, exchange, strategyImpl)
}

func TestValidateIndicatorSets(t *testing.T) {
	t.Run("not-enough-history", func(t *testing.T) {
		var sets []*vespyr.IndicatorSet
		assert.Error(t, vespyr.ErrNotEnoughData, vespyr.ValidateIndicatorSets(2, 0, sets))
	})

	t.Run("correct-tick-times", func(t *testing.T) {
		startTime := time.Now()
		sets := []*vespyr.IndicatorSet{
			&vespyr.IndicatorSet{
				Time: startTime.Add(-time.Minute),
				Values: []*vespyr.IndicatorValue{
					&vespyr.IndicatorValue{},
				},
			},
			&vespyr.IndicatorSet{
				Time: startTime,
				Values: []*vespyr.IndicatorValue{
					&vespyr.IndicatorValue{},
				},
			},
		}
		assert.NoError(t, vespyr.ValidateIndicatorSets(2, 1, sets))

		sets = append(sets, &vespyr.IndicatorSet{})
		assert.EqualError(t, vespyr.ErrNotEnoughData, vespyr.ValidateIndicatorSets(2, 2, sets).Error())
	})

	t.Run("no-indicator-values", func(t *testing.T) {
		startTime := time.Now()
		sets := []*vespyr.IndicatorSet{
			&vespyr.IndicatorSet{
				Time: startTime.Add(-time.Minute),
				Values: []*vespyr.IndicatorValue{
					nil,
				},
			},
			&vespyr.IndicatorSet{
				Time: startTime,
			},
		}
		assert.EqualError(t, vespyr.ValidateIndicatorSets(2, 1, sets), vespyr.ErrNotEnoughData.Error())
	})

	t.Run("missing-values-last-set", func(t *testing.T) {
		startTime := time.Now()
		sets := []*vespyr.IndicatorSet{
			&vespyr.IndicatorSet{
				Time: startTime.Add(-time.Minute),
				Values: []*vespyr.IndicatorValue{
					nil,
				},
			},
			&vespyr.IndicatorSet{
				Time: startTime,
				Values: []*vespyr.IndicatorValue{
					nil,
				},
			},
		}
		assert.EqualError(t, vespyr.ValidateIndicatorSets(2, 1, sets), vespyr.ErrNotEnoughData.Error())
	})

	t.Run("indicator-values", func(t *testing.T) {
		startTime := time.Now()
		sets := []*vespyr.IndicatorSet{
			&vespyr.IndicatorSet{
				Time: startTime.Add(-time.Minute),
				Values: []*vespyr.IndicatorValue{
					&vespyr.IndicatorValue{},
				},
			},
			&vespyr.IndicatorSet{
				Time: startTime,
				Values: []*vespyr.IndicatorValue{
					&vespyr.IndicatorValue{},
				},
			},
		}
		assert.NoError(t, vespyr.ValidateIndicatorSets(2, 1, sets))
	})

}
