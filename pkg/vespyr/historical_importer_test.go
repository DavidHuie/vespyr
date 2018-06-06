package vespyr_test

import (
	"testing"
	"time"

	coinbase "github.com/DavidHuie/go-coinbase-exchange"
	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/mock"
)

func TestHistoricalImporter(t *testing.T) {
	gdaxClient := new(vespyr.MockGDAXClient)
	gdax := vespyr.NewGDAXExchange(gdaxClient, clockwork.NewFakeClock())
	backend := new(vespyr.MockBackend)
	importer := vespyr.NewHistoricalImporter(vespyr.ProductBTCUSD, gdax, backend, 2)

	mock.AssertExpectationsForObjects(t, gdaxClient, backend)

	start := time.Now().Add(-time.Minute * 2)
	end := start.Add(time.Minute * 2)

	rates := []coinbase.HistoricRate{
		{start, 10, 20, 12, 15, 1},
		{start.Add(time.Minute), 11, 21, 16, 13, 2},
	}

	gdaxClient.On("GetHistoricRates", string(vespyr.ProductBTCUSD), mock.MatchedBy(func(c coinbase.GetHistoricRatesParams) bool {
		return cmp.Equal(c, coinbase.GetHistoricRatesParams{
			Start:       start,
			End:         start.Add(150 * time.Second * 60),
			Granularity: 60,
		})
	})).Return(rates, nil)

	backend.On("UpsertCandlestick", &vespyr.CandlestickModel{
		StartTime: start,
		EndTime:   start.Add(time.Minute),
		Low:       10,
		High:      20,
		Open:      12,
		Close:     15,
		Volume:    1,
		Direction: vespyr.CandlestickDirectionUp,
		Product:   vespyr.ProductBTCUSD,
	}).Return(nil).Times(1)

	backend.On("UpsertCandlestick", &vespyr.CandlestickModel{
		StartTime: start.Add(time.Minute),
		EndTime:   start.Add(2 * time.Minute),
		Low:       11,
		High:      21,
		Open:      16,
		Close:     13,
		Volume:    2,
		Direction: vespyr.CandlestickDirectionDown,
		Product:   vespyr.ProductBTCUSD,
	}).Return(nil).Times(1)

	importer.Import(start, end)
}
