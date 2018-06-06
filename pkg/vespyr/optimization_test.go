package vespyr_test

import (
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/MaxHalford/gago"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBacktesterGenome(t *testing.T) {
	t.Skip("test is expensive")

	model := &vespyr.TradingStrategyModel{
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

	endTime := time.Now()
	startTime := endTime.Add(-time.Minute * 30)

	backend := new(vespyr.MockBackend)
	backend.On("FindCandlesticks", startTime.Add(-15*time.Minute),
		endTime, model.Product, int64(model.TickSizeMinutes)).
		Return([]*vespyr.CandlestickModel{
			&vespyr.CandlestickModel{Close: 10, Volume: 1},
			&vespyr.CandlestickModel{Close: 10, Volume: 1},
			&vespyr.CandlestickModel{Close: 9, Volume: 1},
			&vespyr.CandlestickModel{Close: 8, Volume: 1},
			&vespyr.CandlestickModel{Close: 11, Volume: 1},
			&vespyr.CandlestickModel{Close: 11, Volume: 1},
			&vespyr.CandlestickModel{Close: 9, Volume: 1},
			&vespyr.CandlestickModel{Close: 8, Volume: 1},
			&vespyr.CandlestickModel{Close: 10, Volume: 1},
			&vespyr.CandlestickModel{Close: 10, Volume: 1},
			&vespyr.CandlestickModel{Close: 9, Volume: 1},
			&vespyr.CandlestickModel{Close: 8, Volume: 1},
			&vespyr.CandlestickModel{Close: 11, Volume: 1},
			&vespyr.CandlestickModel{Close: 11, Volume: 1},
			&vespyr.CandlestickModel{Close: 9, Volume: 1},
			&vespyr.CandlestickModel{Close: 8, Volume: 1},
		}, nil)
	defer mock.AssertExpectationsForObjects(t, backend)

	factory, err := vespyr.NewBacktesterGenomeFactory(startTime, endTime, model,
		backend)
	assert.NoError(t, err)

	ga := gago.Generational(factory.Generate)
	ga.Initialize()

	for i := 0; i < 100; i++ {
		assert.NoError(t, ga.Enhance())
	}

	assert.True(t, ga.Best.Fitness < 0)
}
