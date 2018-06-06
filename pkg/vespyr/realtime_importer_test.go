package vespyr_test

import (
	"testing"

	"time"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/mock"
)

func TestProcessExchangeMessage(t *testing.T) {
	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)
	importer := vespyr.NewRealtimeImporter(
		vespyr.ProductBTCUSD,
		backend,
		exchange,
	)

	mock.AssertExpectationsForObjects(t, backend, exchange)

	currentTime := time.Now()

	importer.ProcessExchangeMessage(&vespyr.ExchangeMessage{
		Time:        currentTime,
		ProductType: string(vespyr.ProductBTCUSD),
		Size:        2,
		Price:       2800,
		Type:        string(vespyr.MessageMatch),
	})

	importer.ProcessExchangeMessage(&vespyr.ExchangeMessage{
		Time:        currentTime.Add(time.Minute),
		ProductType: string(vespyr.ProductBTCUSD),
		Size:        3,
		Price:       2900,
		Type:        string(vespyr.MessageMatch),
	})

	backend.On("UpsertCandlestick", &vespyr.CandlestickModel{
		StartTime: vespyr.CandlestickBucket(currentTime, 1),
		EndTime:   vespyr.CandlestickBucket(currentTime, 1).Add(time.Minute),
		Low:       2800,
		High:      2800,
		Open:      2800,
		Close:     2800,
		Volume:    2,
		Direction: vespyr.CandlestickDirectionUp,
		Product:   vespyr.ProductBTCUSD,
	}).Return(nil)

	importer.Flush()

	importer.ProcessExchangeMessage(&vespyr.ExchangeMessage{
		Time:        currentTime.Add(2 * time.Minute),
		ProductType: string(vespyr.ProductBTCUSD),
		Size:        4,
		Price:       3000,
		Type:        string(vespyr.MessageMatch),
	})

	backend.On("UpsertCandlestick", &vespyr.CandlestickModel{
		StartTime: vespyr.CandlestickBucket(currentTime, 1).Add(time.Minute),
		EndTime:   vespyr.CandlestickBucket(currentTime, 1).Add(2 * time.Minute),
		Low:       2900,
		High:      2900,
		Open:      2900,
		Close:     2900,
		Volume:    3,
		Direction: vespyr.CandlestickDirectionUp,
		Product:   vespyr.ProductBTCUSD,
	}).Return(nil)

	importer.Flush()
}

func TestProcessExchangeMessageNoVolume(t *testing.T) {
	backend := new(vespyr.MockBackend)
	exchange := new(vespyr.MockExchange)
	importer := vespyr.NewRealtimeImporter(
		vespyr.ProductBTCUSD,
		backend,
		exchange,
	)

	mock.AssertExpectationsForObjects(t, backend, exchange)

	currentTime := time.Now()

	importer.ProcessExchangeMessage(&vespyr.ExchangeMessage{
		Time:        currentTime,
		ProductType: string(vespyr.ProductBTCUSD),
		Size:        0,
		Price:       2800,
		Type:        string(vespyr.MessageMatch),
	})

	importer.ProcessExchangeMessage(&vespyr.ExchangeMessage{
		Time:        currentTime.Add(time.Minute),
		ProductType: string(vespyr.ProductBTCUSD),
		Size:        3,
		Price:       2900,
		Type:        string(vespyr.MessageMatch),
	})

	importer.Flush()
}
