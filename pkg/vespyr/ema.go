package vespyr

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// SMA returns the simple moving average for an index within a slice
// of float64s.
func SMA(period uint, values []float64, index uint) float64 {
	sum := float64(0)
	num := 0

	for i := int(index) - int(period) + 1; i <= int(index); i++ {
		if i < 0 {
			continue
		}
		sum += values[i]
		num++
	}

	return sum / float64(num)
}

// FloatDiff returns a slice with each index containing the difference
// a[i] - b[i].
func FloatDiff(a, b []float64) []float64 {
	var response []float64
	for i, v := range a {
		response = append(response, v-b[i])
	}
	return response
}

// EMAIndicator computes a running value for an exponential moving
// average.
type EMAIndicator struct {
	lastTime time.Time
	lastEMA  float64
	values   []float64
	period   uint
}

// NewEMAIndicator creates a new EMAIndicator.
func NewEMAIndicator(period uint) *EMAIndicator {
	return &EMAIndicator{period: period}
}

// AddCandlestick adds a candlestick to the indicator.
func (e *EMAIndicator) AddCandlestick(c *CandlestickModel) error {
	if c.Volume == 0 {
		return nil
	}

	price, err := c.MeanPrice()
	if err != nil {
		return errors.Wrapf(err, "error calculating mean candlestick price")
	}

	e.lastTime = c.StartTime
	if e.lastEMA == 0 {
		if uint(len(e.values)) < (e.period - 1) {
			e.values = append(e.values, price)
			return nil
		}
		if uint(len(e.values)) == e.period-1 {
			e.values = append(e.values, price)
			e.lastEMA = SMA(e.period, e.values, e.period-1)
			e.values = nil
			return nil
		}
	}

	constant := float64(2) / (float64(e.period) + 1)
	e.lastEMA = constant*(price-e.lastEMA) + e.lastEMA

	return nil
}

// Value returns the last calculated EMA value.
func (e *EMAIndicator) Value() (*IndicatorValue, error) {
	if e.lastEMA == 0 {
		return nil, ErrNotEnoughData
	}
	return &IndicatorValue{
		Time:          e.lastTime,
		Value:         e.lastEMA,
		IndicatorName: e.Name(),
	}, nil
}

// Name returns the name of the indicator.
func (e *EMAIndicator) Name() string {
	return fmt.Sprintf("%s-%d", IndicatorEMA, e.period)
}

// DEMAIndicator is an indicator that tracks the difference in two
// EMAs.
type DEMAIndicator struct {
	lastTime time.Time
	shortEMA *EMAIndicator
	longEMA  *EMAIndicator
}

// NewDEMAIndicator creates a new DEMAIndicator.
func NewDEMAIndicator(shortPeriod, longPeriod uint) *DEMAIndicator {
	return &DEMAIndicator{
		shortEMA: NewEMAIndicator(shortPeriod),
		longEMA:  NewEMAIndicator(longPeriod),
	}
}

// Name returns the name of the indicator.
func (d *DEMAIndicator) Name() string {
	return fmt.Sprintf("%s-%d-%d", IndicatorDEMA, d.shortEMA.period, d.longEMA.period)
}

// AddCandlestick processes a candlestick.
func (d *DEMAIndicator) AddCandlestick(c *CandlestickModel) error {
	if c.Volume == 0 {
		return nil
	}
	if err := d.shortEMA.AddCandlestick(c); err != nil {
		return errors.Wrapf(err, "error adding candlestick to short EMA")
	}
	if err := d.longEMA.AddCandlestick(c); err != nil {
		return errors.Wrapf(err, "error adding candlestick to long EMA")
	}
	d.lastTime = c.StartTime
	return nil
}

// Value returns the DEMA value for the last processed candlestick.
func (d *DEMAIndicator) Value() (*IndicatorValue, error) {
	short, err := d.shortEMA.Value()
	if err != nil {
		return nil, errors.Wrapf(err, "error calculating short EMA")
	}
	long, err := d.longEMA.Value()
	if err != nil {
		return nil, errors.Wrapf(err, "error calculating long EMA")
	}

	value := (short.Value - long.Value) / (short.Value + long.Value) / 2
	return &IndicatorValue{
		Value:         value,
		Time:          d.lastTime,
		IndicatorName: d.Name(),
	}, nil
}
