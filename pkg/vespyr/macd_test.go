package vespyr_test

import (
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestMACDIndicator(t *testing.T) {
	t.Run("test-simple", func(t *testing.T) {
		indicator := vespyr.NewMACDIndicator(1, 2)
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 15, 50}
		responses := []float64{0, .5, .5, .5, .5, .5, .5, .5, 2.5, 12.5}
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
				assert.EqualError(t, errors.Cause(err), vespyr.ErrNotEnoughData.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, responses[i], value.Value)
					assert.Equal(t, startTime, value.Time)
					assert.Equal(t, "macd-1-2", value.IndicatorName)
				}
			}
		}
	})
}

func TestMACDIndicatorWithSignal(t *testing.T) {
	t.Run("test-simple", func(t *testing.T) {
		indicator := vespyr.NewMACDWithSignal(1, 2, 2)
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 15, 50}
		responses := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0.6666666666666667, 3.5555555555555554}
		isError := []bool{true, true, false, false, false, false, false, false, false, false}
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
			if isError[i] {
				assert.EqualError(t, errors.Cause(err), vespyr.ErrNotEnoughData.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, responses[i], value.Value)
					assert.Equal(t, startTime, value.Time)
					assert.Equal(t, "macd-with-signal-2-ema-1-2", value.IndicatorName)
				}
			}
		}
	})
}
