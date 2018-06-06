package vespyr

import (
	"fmt"
	"math/rand"

	"github.com/MaxHalford/gago"
)

// RSIStrategy is a strategy for that buys and sells based on the RSI
// values.
type RSIStrategy struct {
	Period        uint    `yaml:"period"`
	BuyThreshold  float64 `yaml:"buy_threshold"`
	SellThreshold float64 `yaml:"sell_threshold"`

	strategy *TradingStrategyModel
}

// String returns the string representation of the strategy.
func (e *RSIStrategy) String() string {
	return fmt.Sprintf("RSIStrategy-p(%d)-(%f)-(%f)",
		e.Period, e.BuyThreshold, e.SellThreshold,
	)
}

// SetTradingStrategy sets the underlying trading strategy.
func (e *RSIStrategy) SetTradingStrategy(t *TradingStrategyModel) {
	e.strategy = t
}

// Indicators returns the indicators returned by the strategy.
func (e *RSIStrategy) Indicators() []Indicator {
	var indicators []Indicator
	indicators = append(indicators, NewRSIIndicator(e.Period))
	return indicators
}

// Buy determines whether the currency should be bought using the
// indicator history.
func (e *RSIStrategy) Buy(history []*IndicatorSet, current int) (bool, error) {
	if len(history)-1 < current {
		return false, ErrNotEnoughData
	}

	currentValues := history[current].Values

	rsi := currentValues[0]
	if rsi == nil {
		return false, ErrNotEnoughData
	}

	return rsi.Value <= e.BuyThreshold, nil
}

// Sell determines whether the currency should be sold using the
// indicator history.
func (e *RSIStrategy) Sell(history []*IndicatorSet, current int) (bool, error) {
	if len(history)-1 < current {
		return false, ErrNotEnoughData
	}

	currentValues := history[current].Values

	rsi := currentValues[0]
	if rsi == nil {
		return false, ErrNotEnoughData
	}

	return rsi.Value >= e.SellThreshold, nil
}

// Rand creates a random version of the strategy.
func (e *RSIStrategy) Rand(rng *rand.Rand) {
	e.Period = uint(rng.Float64() * 50)
	e.SellThreshold = rng.Float64() * 100
	e.BuyThreshold = e.SellThreshold / 2
}

// Clone returns a clone of the current strategy.
func (e *RSIStrategy) Clone() StrategyGenome {
	return &RSIStrategy{
		Period:        e.Period,
		BuyThreshold:  e.BuyThreshold,
		SellThreshold: e.SellThreshold,
	}
}

// Mutate mutates the underlying strategy.
func (e *RSIStrategy) Mutate(rng *rand.Rand) {
	mutateProb := 0.8

	if rng.Float64() < mutateProb {
		x := float64(e.Period)
		x += x * rng.NormFloat64()
		if x < 0 {
			e.Period = 0
		} else {
			e.Period = uint(x)
		}
	}

	if rng.Float64() < mutateProb {
		x := e.BuyThreshold
		x += x * rng.NormFloat64()
		e.BuyThreshold = x
		if x > 100 {
			e.BuyThreshold = 100
		}
		if x < 0 {
			e.BuyThreshold = 0
		}
	}

	if rng.Float64() < mutateProb {
		x := e.SellThreshold
		x += x * rng.NormFloat64()
		e.SellThreshold = x
		if x > 100 {
			e.SellThreshold = 100
		}
		if x < 0 {
			e.SellThreshold = 0
		}
	}
}

// Crossover crosses over an RSIStrategy with a different one.
func (e *RSIStrategy) Crossover(m StrategyGenome,
	r *rand.Rand) (StrategyGenome, StrategyGenome) {
	mate := m.(*RSIStrategy)

	p1 := []float64{float64(e.Period), e.BuyThreshold, e.SellThreshold}
	p2 := []float64{float64(mate.Period), mate.BuyThreshold, mate.SellThreshold}

	c1, c2 := gago.CrossUniformFloat64(p1, p2, r)

	s1 := &RSIStrategy{
		Period:        uint(c1[0]),
		BuyThreshold:  c1[1],
		SellThreshold: c1[2],
	}
	s2 := &RSIStrategy{
		Period:        uint(c2[0]),
		BuyThreshold:  c2[1],
		SellThreshold: c2[2],
	}

	return s1, s2
}
