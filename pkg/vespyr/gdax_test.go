package vespyr_test

import (
	"sync"
	"testing"
	"time"

	"github.com/DavidHuie/go-coinbase-exchange"
	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGDAXGetCandlesticks(t *testing.T) {
	gdaxClient := new(vespyr.MockGDAXClient)
	gdax := vespyr.NewGDAXExchange(gdaxClient, clockwork.NewFakeClock())

	mock.AssertExpectationsForObjects(t, gdaxClient)

	start := time.Now().Add(-time.Minute * 2)
	end := start.Add(time.Minute * 2)

	rates := []coinbase.HistoricRate{
		{start, 10, 20, 12, 15, 1},
		{start.Add(time.Minute), 11, 21, 16, 13, 2},
	}

	gdaxClient.On("GetHistoricRates", string(vespyr.ProductBTCUSD), mock.MatchedBy(func(c coinbase.GetHistoricRatesParams) bool {
		return cmp.Equal(c, coinbase.GetHistoricRatesParams{
			Start:       start,
			End:         end,
			Granularity: 60,
		})
	})).Return(rates, nil)

	candleSticks, err := gdax.GetCandlesticks(vespyr.ProductBTCUSD, start, end, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(candleSticks) != 2 {
		t.Fatal("invalid number of candlesticks")
	}

	c1 := candleSticks[0]
	assert.Equal(t, &vespyr.CandlestickModel{
		StartTime: start,
		EndTime:   start.Add(time.Minute),
		Low:       10,
		High:      20,
		Open:      12,
		Close:     15,
		Volume:    1,
		Direction: vespyr.CandlestickDirectionUp,
		Product:   vespyr.ProductBTCUSD,
	}, c1)

	c2 := candleSticks[1]
	assert.Equal(t, &vespyr.CandlestickModel{
		StartTime: start.Add(time.Minute),
		EndTime:   start.Add(2 * time.Minute),
		Low:       11,
		High:      21,
		Open:      16,
		Close:     13,
		Volume:    2,
		Direction: vespyr.CandlestickDirectionDown,
		Product:   vespyr.ProductBTCUSD,
	}, c2)
}

func TestGDAXCreateMarketOrderLive(t *testing.T) {
	t.Skip()

	client := coinbase.NewClient("", "", "")
	exchange := vespyr.NewGDAXExchange(client, clockwork.NewRealClock())

	response, err := exchange.CreateMarketOrder(&vespyr.MarketOrder{
		Product: vespyr.ProductBTCUSD,
		Side:    vespyr.OrderSell,
		Cost:    0.01,
	})

	t.Logf("response: %#v", response)
	t.Logf("err: %#v", err)
}

func TestFakeGDAXCreateMarketOrder(t *testing.T) {
	t.Skip("test is expensive")

	ex := vespyr.NewFakeGDAXExchange(clockwork.NewRealClock())

	response, err := ex.CreateMarketOrder(&vespyr.MarketOrder{
		Product: vespyr.ProductBTCUSD,
		Side:    vespyr.OrderBuy,
		Cost:    10000,
	})
	if assert.NoError(t, err) {
		t.Logf("fake gdax buy response: %#v", response)
	}

	response, err = ex.CreateMarketOrder(&vespyr.MarketOrder{
		Product: vespyr.ProductBTCUSD,
		Side:    vespyr.OrderSell,
		Cost:    2,
	})
	if assert.NoError(t, err) {
		t.Logf("fake gdax sell response: %#v", response)
	}
}

func TestGDAXCreateMarketOrder(t *testing.T) {
	gdaxClient := new(vespyr.MockGDAXClient)
	clock := clockwork.NewFakeClock()
	gdax := vespyr.NewGDAXExchange(gdaxClient, clock)

	mock.AssertExpectationsForObjects(t, gdaxClient)

	t.Run("buy", func(t *testing.T) {
		gdaxClient.On("CreateOrder", &coinbase.Order{
			Type:      "market",
			Side:      "buy",
			ProductId: string(vespyr.ProductBTCUSD),
			Funds:     10000,
		}).Return(coinbase.Order{Id: "order-id"}, nil).Once()

		gdaxClient.On("GetOrder", "order-id").Return(
			coinbase.Order{Status: "pending"}, nil,
		).Once()

		gdaxClient.On("GetOrder", "order-id").Return(
			coinbase.Order{
				Id:         "order-id",
				Status:     "done",
				Settled:    true,
				FilledSize: 2.5,
				FillFees:   25,
			}, nil,
		).Once()

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := gdax.CreateMarketOrder(&vespyr.MarketOrder{
				Product: vespyr.ProductBTCUSD,
				Side:    vespyr.OrderBuy,
				Cost:    10000,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, "order-id", response.ExchangeID)
				assert.Equal(t, float64(25), response.Fees)
				assert.Equal(t, float64(2.5), response.FilledSize)
				assert.Equal(t, vespyr.CurrencyUSD, response.FeesCurrency)
				assert.Equal(t, vespyr.CurrencyBTC, response.FilledSizeCurrency)
			}
		}()

		time.Sleep(time.Millisecond)
		clock.Advance(3 * time.Second)
		wg.Wait()
	})

	t.Run("sell", func(t *testing.T) {
		gdaxClient.On("CreateOrder", &coinbase.Order{
			Type:      "market",
			Side:      "sell",
			ProductId: string(vespyr.ProductBTCUSD),
			Size:      2.5,
		}).Return(coinbase.Order{Id: "order-id"}, nil).Once()

		gdaxClient.On("GetOrder", "order-id").Return(
			coinbase.Order{Status: "pending"}, nil,
		).Once()

		gdaxClient.On("GetOrder", "order-id").Return(
			coinbase.Order{
				Id:            "order-id",
				Status:        "done",
				Settled:       true,
				ExecutedValue: 10000,
				FillFees:      25,
			}, nil,
		).Once()

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := gdax.CreateMarketOrder(&vespyr.MarketOrder{
				Product: vespyr.ProductBTCUSD,
				Side:    vespyr.OrderSell,
				Cost:    2.5,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, "order-id", response.ExchangeID)
				assert.Equal(t, float64(25), response.Fees)
				assert.Equal(t, float64(9975), response.FilledSize)
				assert.Equal(t, vespyr.CurrencyUSD, response.FeesCurrency)
				assert.Equal(t, vespyr.CurrencyUSD, response.FilledSizeCurrency)
			}
		}()

		time.Sleep(time.Millisecond)
		clock.Advance(3 * time.Second)
		wg.Wait()
	})
}
