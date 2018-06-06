package vespyr_test

import (
	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
)

func fakeCandlestick() *vespyr.CandlestickModel {
	return &vespyr.CandlestickModel{
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Minute),
		Low:       2000,
		High:      3000,
		Open:      2500,
		Close:     2800,
		Volume:    3,
		Direction: vespyr.CandlestickDirectionDown,
		Product:   vespyr.ProductBTCUSD,
	}
}
