package vespyr_test

import (
	"testing"
	"time"

	"math/rand"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSMA(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	assert.Equal(t, float64(1), vespyr.SMA(1, values, 0))
	assert.Equal(t, float64(1), vespyr.SMA(2, values, 0))
	assert.Equal(t, float64(1.5), vespyr.SMA(2, values, 1))
	assert.Equal(t, float64(5.5), vespyr.SMA(10, values, 9))
}

func TestEMAIndicator(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		indicator := vespyr.NewEMAIndicator(5)
		assert.NoError(t, indicator.AddCandlestick(&vespyr.CandlestickModel{
			Open:  100,
			Low:   100,
			High:  100,
			Close: 100,
		}))
		value, err := indicator.Value()
		assert.Nil(t, value)
		assert.EqualError(t, err, vespyr.ErrNotEnoughData.Error())
	})

	t.Run("initial-sma", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		indicator := vespyr.NewEMAIndicator(10)
		responses := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 5.5}
		for i, v := range values {
			startTime := time.Now()
			assert.NoError(t, indicator.AddCandlestick(&vespyr.CandlestickModel{
				StartTime: startTime,
				Open:      v,
				Low:       v,
				High:      v,
				Close:     v,
				Volume:    1,
			}))
			value, err := indicator.Value()
			if responses[i] == 0 {
				assert.EqualError(t, err, vespyr.ErrNotEnoughData.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, responses[i], value.Value)
					assert.Equal(t, startTime, value.Time)
					assert.Equal(t, "ema-10", value.IndicatorName)
				}
			}
		}
	})

	t.Run("ema-1", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		indicator := vespyr.NewEMAIndicator(1)
		responses := values
		for i, v := range values {
			assert.NoError(t, indicator.AddCandlestick(&vespyr.CandlestickModel{
				Open:   v,
				Low:    v,
				High:   v,
				Close:  v,
				Volume: 1,
			}))
			value, err := indicator.Value()
			if assert.NoError(t, err) {
				assert.Equal(t, responses[i], value.Value)
			}
		}
	})

	t.Run("ema-5", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		indicator := vespyr.NewEMAIndicator(5)
		responses := []float64{0, 0, 0, 0, 3, 4, 5, 6, 7, 8}
		for i, v := range values {
			assert.NoError(t, indicator.AddCandlestick(&vespyr.CandlestickModel{
				Close:  v,
				Open:   v,
				Low:    v,
				High:   v,
				Volume: 1,
			}))
			value, err := indicator.Value()
			if responses[i] == 0 {
				assert.EqualError(t, err, vespyr.ErrNotEnoughData.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, responses[i], value.Value)
				}
			}
		}
	})

	t.Run("ema-complex", func(t *testing.T) {
		values := []float64{
			64.75,
			63.79,
			63.73,
			63.73,
			63.55,
			63.19,
			63.91,
			63.85,
			62.95,
			63.37,
			61.33,
			61.51,
			61.87,
		}
		indicator := vespyr.NewEMAIndicator(10)
		responses := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 63.682, 63.254363636363635, 62.93720661157025, 62.7431690458302}
		for i, v := range values {
			assert.NoError(t, indicator.AddCandlestick(&vespyr.CandlestickModel{
				Close:  v,
				Volume: 1,
				Open:   v,
				Low:    v,
				High:   v,
			}))
			value, err := indicator.Value()
			if responses[i] == 0 {
				assert.EqualError(t, err, vespyr.ErrNotEnoughData.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, responses[i], value.Value)
				}
			}
		}
	})
}

func TestDEMAIndicator(t *testing.T) {
	t.Run("simple-dema", func(t *testing.T) {
		dema := vespyr.NewDEMAIndicator(1, 2)

		// TODO: verify these numbers
		values := []float64{10, 10, 9, 8, 11}
		responses := []float64{0, 0,
			-0.009090909090909106,
			-0.013513513513513521,
			0.02014010507880909,
		}
		hasError := []bool{true, false, false, false, false}
		for i, v := range values {
			assert.NoError(
				t,
				dema.AddCandlestick(&vespyr.CandlestickModel{Close: v, Volume: 1}),
			)

			value, err := dema.Value()
			if hasError[i] {
				assert.EqualError(t,
					errors.Cause(err), vespyr.ErrNotEnoughData.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, responses[i], value.Value)
				}
			}

		}

	})
}

func TestEMAMutate(t *testing.T) {
	t.Skip()

	m := vespyr.EMACrossoverStrategy{
		ShortPeriod:   1,
		LongPeriod:    8,
		UpThreshold:   .0025,
		DownThreshold: -.0025,
	}

	r := rand.New(rand.NewSource(1))
	for x := 0; x < 100; x++ {
		m.Mutate(r)
		t.Logf("%v\n", m)
	}
}
