package vespyr

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// RSIIndicator calculates the RSI indicator.
type RSIIndicator struct {
	iterations      uint
	periods         uint
	lastValue       float64
	sumGains        float64
	sumLosses       float64
	lastAverageGain float64
	lastAverageLoss float64
	currentRSI      float64
	lastTime        time.Time
}

// NewRSIIndicator returns a new RSIIndicator.
func NewRSIIndicator(periods uint) *RSIIndicator {
	return &RSIIndicator{
		periods: periods,
	}
}

func rsiFromRS(rs float64) float64 {
	return 100 - 100/(1+rs)
}

// AddCandlestick adds a candlestick to the indicator.
func (r *RSIIndicator) AddCandlestick(c *CandlestickModel) error {
	if c.Volume == 0 {
		return nil
	}

	price, err := c.MeanPrice()
	if err != nil {
		return errors.Wrapf(err, "error calculating mean price")
	}

	defer func() {
		r.iterations++
		r.lastValue = price
		r.lastTime = c.StartTime
	}()

	if r.iterations == 0 {
		return nil
	}

	diff := price - r.lastValue
	if diff >= 0 {
		r.sumGains += diff
	} else {
		r.sumLosses -= diff
	}

	if r.iterations < r.periods {
		return nil
	}

	if r.iterations == r.periods {
		r.lastAverageGain = r.sumGains / float64(r.periods)
		r.lastAverageLoss = r.sumLosses / float64(r.periods)

		if r.lastAverageGain == 0 && r.lastAverageLoss == 0 {
			r.currentRSI = 50
			return nil
		}
		if r.lastAverageGain == 0 {
			r.currentRSI = 0
			return nil
		}
		if r.lastAverageLoss == 0 {
			r.currentRSI = 100
			return nil
		}

		rs := r.lastAverageGain / r.lastAverageLoss
		r.currentRSI = rsiFromRS(rs)
		return nil
	}

	var averageGain float64
	if diff >= 0 {
		averageGain = (r.lastAverageGain*(float64(r.periods)-1) + diff) / 14
	} else {
		averageGain = r.lastAverageGain * (float64(r.periods) - 1) / 14
	}

	var averageLoss float64
	if diff < 0 {
		averageLoss = (r.lastAverageLoss*(float64(r.periods)-1) - diff) / 14
	} else {
		averageLoss = r.lastAverageLoss * (float64(r.periods) - 1) / 14
	}

	r.lastAverageGain = averageGain
	r.lastAverageLoss = averageLoss

	if r.lastAverageGain == 0 && r.lastAverageLoss == 0 {
		r.currentRSI = 50
		return nil
	}
	if r.lastAverageGain == 0 {
		r.currentRSI = 0
		return nil
	}
	if r.lastAverageLoss == 0 {
		r.currentRSI = 100
		return nil
	}

	r.currentRSI = rsiFromRS(averageGain / averageLoss)

	return nil
}

// Name returns the name of the indicator
func (r *RSIIndicator) Name() string {
	return fmt.Sprintf("%s-%d", IndicatorRSI, r.periods)
}

// Value returns the last calculated RSI value.
func (r *RSIIndicator) Value() (*IndicatorValue, error) {
	if r.iterations <= r.periods {
		return nil, ErrNotEnoughData
	}
	return &IndicatorValue{
		Time:          r.lastTime,
		Value:         r.currentRSI,
		IndicatorName: r.Name(),
	}, nil
}
