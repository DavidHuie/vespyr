package vespyr_test

import (
	"testing"
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/assert"
)

func TestRSIIndicator(t *testing.T) {
	f := func(x float64) *float64 {
		return &x
	}

	t.Run("rsi-simple", func(t *testing.T) {
		r := vespyr.NewRSIIndicator(2)

		values := []float64{1, 1, 4, 6, 2, 100}
		responses := []*float64{nil, nil, f(100), f(100), f(5.882352941176464), f(99.70935513169846)}

		for i, v := range values {
			assert.NoError(t, r.AddCandlestick(&vespyr.CandlestickModel{
				Volume:    1,
				StartTime: time.Now(),
				Close:     v,
			}))

			value, err := r.Value()

			response := responses[i]
			if response == nil {
				assert.EqualError(t, err, vespyr.ErrNotEnoughData.Error(), "case %d", i)
			} else {
				assert.Equal(t, *response, value.Value)
				assert.NoError(t, err)
			}
		}
	})

	// http://cns.bu.edu/~gsc/CN710/fincast/Technical%20_indicators/Relative%20Strength%20Index%20(RSI).htm
	t.Run("rsi-complex", func(t *testing.T) {
		r := vespyr.NewRSIIndicator(14)

		values := []float64{
			46.1250,
			47.1250,
			46.4375,
			46.9375,
			44.9375,
			44.25,
			44.6250,
			45.7500,
			47.8125,
			47.5625,
			47,
			44.5625,
			46.3125,
			47.6875,
			46.6875,
			45.6875,
			43.0625,
			43.5625,
			44.8750,
			43.6875,
		}
		responses := []*float64{
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			f(51.77865612648221),
			f(48.47708511243952),
			f(41.073449472180485),
			f(42.863429113074524),
			f(47.38184957507224),
			f(43.992110594000444),
		}

		for i, v := range values {
			assert.NoError(t, r.AddCandlestick(&vespyr.CandlestickModel{
				Volume:    1,
				StartTime: time.Now(),
				Close:     v,
			}))

			value, err := r.Value()

			response := responses[i]
			if response == nil {
				assert.EqualError(t, err, vespyr.ErrNotEnoughData.Error())
			} else {
				assert.Equal(t, *response, value.Value)
				assert.NoError(t, err)
			}
		}
	})
}
