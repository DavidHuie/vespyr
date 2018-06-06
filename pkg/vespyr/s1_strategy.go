package vespyr

import (
	"fmt"
	"math/rand"

	"github.com/MaxHalford/gago"
	"github.com/sirupsen/logrus"
)

const (
	rsiPeriod = 14
)

// S1Strategy is a custom trading strategy.
type S1Strategy struct {
	EMAShortPeriod       uint    `yaml:"ema_short_period"`
	EMALongPeriod        uint    `yaml:"ema_long_period"`
	EMAUpThreshold       float64 `yaml:"ema_up_threshold"`
	EMADownThreshold     float64 `yaml:"ema_down_threshold"`
	RSIExitThreshold     float64 `yaml:"rsi_exit_threshold"`
	RSIEntranceThreshold float64 `yaml:"rsi_entrance_threshold"`

	strategy *TradingStrategyModel
}

// SetTradingStrategy sets the underlying trading strategy.
func (s *S1Strategy) SetTradingStrategy(t *TradingStrategyModel) {
	s.strategy = t
}

// String returns the string representation of the strategy.
func (s *S1Strategy) String() string {
	return fmt.Sprintf("S1: ema short period: %d, ema long period: %d, ema up threshold: %f, ema down threshold: %f, rsi entrance threshold: %f, rsi exit threshold: %f",
		s.EMAShortPeriod, s.EMALongPeriod, s.EMAUpThreshold, s.EMADownThreshold,
		s.RSIEntranceThreshold, s.RSIExitThreshold,
	)
}

// Indicators returns the indicators returned by the strategy.
func (s *S1Strategy) Indicators() []Indicator {
	var indicators []Indicator
	indicators = append(indicators, NewDEMAIndicator(s.EMAShortPeriod, s.EMALongPeriod))
	indicators = append(indicators, NewEMAIndicator(s.EMAShortPeriod))
	indicators = append(indicators, NewEMAIndicator(s.EMALongPeriod))
	indicators = append(indicators, NewRSIIndicator(rsiPeriod))
	return indicators
}

// Buy determines whether the currency should be bought using the
// indicator history.
func (s *S1Strategy) Buy(history []*IndicatorSet, current int) (bool, error) {
	if len(history)-1 < current {
		return false, ErrNotEnoughData
	}

	// Don't reenter a trade if we've spent multiple ticks on
	// it. This occurs when we exit a trade via RSI while still
	// having short EMA > large EMA.
	if current-1 >= 0 {
		lastDEMA := history[current-1].Values[0]
		lastRSI := history[current-1].Values[3]
		if lastDEMA.Value >= s.EMAUpThreshold && lastRSI.Value >= s.RSIExitThreshold {
			return false, nil
		}
	}

	currentValues := history[current].Values
	rsi := currentValues[3]
	dema := currentValues[0]

	logrus.Debugf("s1 buy rsi: %f, rsi entrance threshold: %f, dema: %f, dema up threshold: %f",
		rsi.Value, s.RSIEntranceThreshold, dema.Value, s.EMAUpThreshold)

	// Enter the trade early if RSI is at the right threshold.
	if rsi == nil {
		return false, ErrNotEnoughData
	}
	if rsi.Value <= s.RSIEntranceThreshold {
		return true, nil
	}

	if dema == nil {
		return false, ErrNotEnoughData
	}
	return (dema.Value > s.EMAUpThreshold), nil
}

// Sell determines whether the currency should be sold using the
// indicator history.
func (s *S1Strategy) Sell(history []*IndicatorSet, current int) (bool, error) {
	if len(history)-1 < current {
		return false, ErrNotEnoughData
	}

	currentValues := history[current].Values
	dema := currentValues[0]
	rsi := currentValues[3]

	logrus.Debugf("s1 sell rsi: %f, rsi exit threshold: %f, dema: %f, dema down threshold: %f",
		rsi.Value, s.RSIExitThreshold, dema.Value, s.EMADownThreshold)

	// Exit the trade early if the RSI value is larger than some
	// threshold.
	if rsi == nil {
		return false, ErrNotEnoughData
	}
	if rsi.Value >= s.RSIExitThreshold {
		return true, nil
	}

	if dema == nil {
		return false, ErrNotEnoughData
	}

	return (dema.Value < s.EMADownThreshold), nil
}

// Rand creates a random version of the strategy.
func (s *S1Strategy) Rand(rng *rand.Rand) {
	s.EMAShortPeriod = uint(rng.Float64() * 50)
	s.EMALongPeriod = 2 * s.EMAShortPeriod
	s.EMAUpThreshold = rng.Float64() * .002
	s.EMADownThreshold = -s.EMAUpThreshold
	s.RSIExitThreshold = rng.Float64() * 100
	s.RSIEntranceThreshold = rng.Float64() * 100
}

// Clone returns a clone of the current strategy.
func (s *S1Strategy) Clone() StrategyGenome {
	return &S1Strategy{
		EMAShortPeriod:       s.EMAShortPeriod,
		EMALongPeriod:        s.EMALongPeriod,
		EMAUpThreshold:       s.EMAUpThreshold,
		EMADownThreshold:     s.EMADownThreshold,
		RSIExitThreshold:     s.RSIExitThreshold,
		RSIEntranceThreshold: s.RSIEntranceThreshold,
	}
}

// Mutate mutates the underlying strategy.
func (s *S1Strategy) Mutate(rng *rand.Rand) {
	mutateProb := 0.8

	thresholds := []float64{s.EMADownThreshold, s.EMAUpThreshold}
	gago.MutNormalFloat64(thresholds, mutateProb, rng)
	s.EMADownThreshold, s.EMAUpThreshold = thresholds[0], thresholds[1]

	if rng.Float64() < mutateProb {
		x := int(s.EMAShortPeriod)
		x += int(float64(s.EMAShortPeriod) * rng.NormFloat64())
		if x > 0 {
			s.EMAShortPeriod = uint(x) % 100
		}
	}

	if rng.Float64() < mutateProb {
		x := int(s.EMALongPeriod)
		x += int(float64(s.EMALongPeriod) * rng.NormFloat64())
		if x > 0 {
			s.EMALongPeriod = uint(x) % 100
		}
	}

	if s.EMALongPeriod < s.EMAShortPeriod {
		s.EMALongPeriod = s.EMAShortPeriod
	}

	if rng.Float64() < mutateProb {
		x := s.RSIExitThreshold
		x += s.RSIExitThreshold * rng.NormFloat64()
		if x >= 0 {
			s.RSIExitThreshold = x
		}
		if x > 100 {
			s.RSIExitThreshold = 100
		}
	}

	if rng.Float64() < mutateProb {
		x := s.RSIEntranceThreshold
		x += s.RSIEntranceThreshold * rng.NormFloat64()
		if x >= 0 {
			s.RSIEntranceThreshold = x
		}
		if x > 100 {
			s.RSIEntranceThreshold = 100
		}
	}
}

// Crossover crosses over an S1Strategy with a different
// ons.
func (s *S1Strategy) Crossover(m StrategyGenome,
	r *rand.Rand) (StrategyGenome, StrategyGenome) {
	mate := m.(*S1Strategy)

	p1 := []float64{float64(s.EMAShortPeriod), float64(s.EMALongPeriod),
		s.EMAUpThreshold, s.EMADownThreshold,
		s.RSIExitThreshold, s.RSIEntranceThreshold}
	p2 := []float64{float64(mate.EMAShortPeriod), float64(mate.EMALongPeriod),
		mate.EMAUpThreshold, mate.EMADownThreshold,
		mate.RSIExitThreshold, s.RSIEntranceThreshold}

	c1, c2 := gago.CrossUniformFloat64(p1, p2, r)

	s1 := &S1Strategy{
		EMAShortPeriod:       uint(c1[0]),
		EMALongPeriod:        uint(c1[1]),
		EMAUpThreshold:       c1[2],
		EMADownThreshold:     c1[3],
		RSIExitThreshold:     c1[4],
		RSIEntranceThreshold: c1[5],
	}
	if s1.EMALongPeriod < s1.EMAShortPeriod {
		s1.EMAShortPeriod = s1.EMALongPeriod
	}

	s2 := &S1Strategy{
		EMAShortPeriod:       uint(c2[0]),
		EMALongPeriod:        uint(c2[1]),
		EMAUpThreshold:       c2[2],
		EMADownThreshold:     c2[3],
		RSIExitThreshold:     c2[4],
		RSIEntranceThreshold: c2[5],
	}
	if s2.EMALongPeriod < s2.EMAShortPeriod {
		s2.EMAShortPeriod = s2.EMALongPeriod
	}

	return s1, s2
}
