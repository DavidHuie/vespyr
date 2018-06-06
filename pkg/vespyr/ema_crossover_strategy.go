package vespyr

import (
	"fmt"
	"math/rand"

	"github.com/MaxHalford/gago"
	"github.com/sirupsen/logrus"
)

// EMACrossoverStrategy is a strategy for that buys and sells based on
// EMA crossovers.
type EMACrossoverStrategy struct {
	ShortPeriod   uint    `yaml:"short_period"`
	LongPeriod    uint    `yaml:"long_period"`
	UpThreshold   float64 `yaml:"up_threshold"`
	DownThreshold float64 `yaml:"down_threshold"`

	strategy *TradingStrategyModel
}

// String returns the string representation of the strategy.
func (e *EMACrossoverStrategy) String() string {
	return fmt.Sprintf("EMA Crossover: short period: %d, long period: %d, up threshold: %f, down threshold %f",
		e.ShortPeriod, e.LongPeriod, e.UpThreshold, e.DownThreshold,
	)
}

// SetTradingStrategy sets the underlying trading strategy.
func (e *EMACrossoverStrategy) SetTradingStrategy(t *TradingStrategyModel) {
	e.strategy = t
}

// Indicators returns the indicators returned by the strategy.
func (e *EMACrossoverStrategy) Indicators() []Indicator {
	var indicators []Indicator
	indicators = append(indicators, NewDEMAIndicator(e.ShortPeriod, e.LongPeriod))
	indicators = append(indicators, NewEMAIndicator(e.ShortPeriod))
	indicators = append(indicators, NewEMAIndicator(e.LongPeriod))
	return indicators
}

// Buy determines whether the currency should be bought using the
// indicator history.
func (e *EMACrossoverStrategy) Buy(history []*IndicatorSet, current int) (bool, error) {
	if len(history)-1 < current {
		return false, ErrNotEnoughData
	}

	currentValues := history[current].Values

	dema := currentValues[0]
	if dema == nil {
		return false, ErrNotEnoughData
	}

	message := fmt.Sprintf("ema crossover (%d, %s) buy dema value: %f, up threshold: %f",
		e.strategy.ID, e.strategy.Product, dema.Value, e.UpThreshold)
	logrus.Debug(message)
	PostStrategyDataToSlack(e, e.strategy, map[string]interface{}{
		currentValues[0].IndicatorName: currentValues[0].Value,
		currentValues[1].IndicatorName: currentValues[1].Value,
		currentValues[2].IndicatorName: currentValues[2].Value,
	})

	return (dema.Value > e.UpThreshold), nil
}

// Sell determines whether the currency should be sold using the
// indicator history.
func (e *EMACrossoverStrategy) Sell(history []*IndicatorSet, current int) (bool, error) {
	if len(history)-1 < current {
		return false, ErrNotEnoughData
	}

	currentValues := history[current].Values

	dema := currentValues[0]
	if dema == nil {
		return false, ErrNotEnoughData
	}

	message := fmt.Sprintf("ema crossover (%d, %s) sell dema value: %f, down threshold: %f",
		e.strategy.ID, e.strategy.Product, dema.Value, e.DownThreshold)
	logrus.Debug(message)
	PostStrategyDataToSlack(e, e.strategy, map[string]interface{}{
		currentValues[0].IndicatorName: currentValues[0].Value,
		currentValues[1].IndicatorName: currentValues[1].Value,
		currentValues[2].IndicatorName: currentValues[2].Value,
	})

	return (dema.Value < e.DownThreshold), nil
}

// Rand creates a random version of the strategy.
func (e *EMACrossoverStrategy) Rand(rng *rand.Rand) {
	e.ShortPeriod = uint(rng.Float64() * 50)
	e.LongPeriod = 2 * e.ShortPeriod
	e.UpThreshold = rng.Float64() * .002
	e.DownThreshold = -e.UpThreshold
}

// Clone returns a clone of the current strategy.
func (e *EMACrossoverStrategy) Clone() StrategyGenome {
	return &EMACrossoverStrategy{
		ShortPeriod:   e.ShortPeriod,
		LongPeriod:    e.LongPeriod,
		UpThreshold:   e.UpThreshold,
		DownThreshold: e.DownThreshold,
	}
}

// Mutate mutates the underlying strategy.
func (e *EMACrossoverStrategy) Mutate(rng *rand.Rand) {
	mutateProb := 0.8

	thresholds := []float64{e.DownThreshold, e.UpThreshold}
	gago.MutNormalFloat64(thresholds, mutateProb, rng)
	e.DownThreshold, e.UpThreshold = thresholds[0], thresholds[1]

	if rng.Float64() < mutateProb {
		x := int(e.ShortPeriod)
		x += int(float64(e.ShortPeriod) * rng.NormFloat64())
		if x > 0 {
			e.ShortPeriod = uint(x) % 100
		}
	}

	if rng.Float64() < mutateProb {
		x := int(e.LongPeriod)
		x += int(float64(e.LongPeriod) * rng.NormFloat64())
		if x > 0 {
			e.LongPeriod = uint(x) % 100
		}
	}

	if e.LongPeriod < e.ShortPeriod {
		e.LongPeriod = e.ShortPeriod
	}
}

// Crossover crosses over an EMACrossoverStrategy with a different
// one.
func (e *EMACrossoverStrategy) Crossover(m StrategyGenome,
	r *rand.Rand) (StrategyGenome, StrategyGenome) {
	mate := m.(*EMACrossoverStrategy)

	p1 := []float64{float64(e.ShortPeriod), float64(e.LongPeriod),
		e.UpThreshold, e.DownThreshold}
	p2 := []float64{float64(mate.ShortPeriod), float64(mate.LongPeriod),
		mate.UpThreshold, mate.DownThreshold}

	c1, c2 := gago.CrossUniformFloat64(p1, p2, r)

	s1 := &EMACrossoverStrategy{
		ShortPeriod:   uint(c1[0]),
		LongPeriod:    uint(c1[1]),
		UpThreshold:   c1[2],
		DownThreshold: c1[3],
	}
	if s1.LongPeriod < s1.ShortPeriod {
		s1.ShortPeriod = s1.LongPeriod
	}

	s2 := &EMACrossoverStrategy{
		ShortPeriod:   uint(c2[0]),
		LongPeriod:    uint(c2[1]),
		UpThreshold:   c2[2],
		DownThreshold: c2[3],
	}
	if s2.LongPeriod < s2.ShortPeriod {
		s2.ShortPeriod = s2.LongPeriod
	}

	return s1, s2
}
