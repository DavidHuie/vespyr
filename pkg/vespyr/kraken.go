package vespyr

import (
	"context"
	"time"

	"github.com/DavidHuie/kraken-go-api-client"
	"github.com/jonboulle/clockwork"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	krakenSleepBetweenCandles = 15 * time.Second
)

// KrakenClient is the interface needed out of a Kraken client.
type KrakenClient interface {
	OHLC(pair string, last ...int64) (*krakenapi.OHLCResponse, error)
	// AddOrder(pair string, direction string, orderType string, volume string, args map[string]string) (*krakenapi.AddOrderResponse, error)
	// QueryOrders(txids string, args map[string]string) (*krakenapi.QueryOrdersResponse, error)
}

// KrakenExchange is a client to the Kraken cryptocurrency exchange.
type KrakenExchange struct {
	client KrakenClient
	clock  clockwork.Clock
}

// NewKrakenExchange creates a new instance of KrakenExchange.
func NewKrakenExchange(client KrakenClient, clock clockwork.Clock) *KrakenExchange {
	return &KrakenExchange{
		client: client,
		clock:  clock,
	}
}

var productToKrakenType = map[Product]string{
	ProductXMRUSD:   krakenapi.XXMRZUSD,
	ProductBCHUSD:   krakenapi.BCHUSD,
	ProductDashUSD:  krakenapi.DASHUSD,
	ProductZcashUSD: krakenapi.XZECZUSD,
	ProductXRPUSD:   krakenapi.XXRPZUSD,
	// ProductETCUSD:   krakenapi.XETCXUSD,
}

// EmitsFullCandlesticks returns whether the exchange emits full
// candlesticks.
func (k *KrakenExchange) EmitsFullCandlesticks() bool {
	return true
}

// StreamCandlesticks provides a stream of candlesticks for the
// product from the exchange.
func (k *KrakenExchange) StreamCandlesticks(ctx context.Context, product Product) (<-chan *CandlestickModel, error) {
	c := make(chan *CandlestickModel)
	krakenType := productToKrakenType[product]
	response, err := k.client.OHLC(krakenType)
	if err != nil {
		return nil, errors.Wrapf(err, "error making initial Kraken OHLC call")
	}

	go func() {
		lastID := response.Last
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			response, err := k.client.OHLC(krakenType, lastID)
			if err != nil {
				logrus.WithError(err).Errorf("error making Kraken OHLC call")
				k.clock.Sleep(krakenSleepBetweenCandles)
				continue
			}

			if len(response.OHLC) == 0 {
				logrus.Warnf("Kraken OHLC response did not have any candlesticks")
				k.clock.Sleep(krakenSleepBetweenCandles)
				continue
			}

			// If there's just a single candlestick, it's
			// referring to the current time period.
			if len(response.OHLC) == 1 {
				k.clock.Sleep(krakenSleepBetweenCandles)
				continue
			}

			for i := 0; i < len(response.OHLC)-1; i++ {
				candle := response.OHLC[i]

				model := &CandlestickModel{
					StartTime: candle.Time,
					EndTime:   candle.Time.Add(time.Minute),
					Low:       candle.Low,
					High:      candle.High,
					Open:      candle.Open,
					Close:     candle.Close,
					Volume:    candle.Volume,
					Product:   product,
				}

				if candle.Volume > 0 {
					if model.Close >= model.Open {
						model.Direction = CandlestickDirectionUp
					} else {
						model.Direction = CandlestickDirectionDown
					}
				}

				c <- model
			}

			lastID = response.Last

			k.clock.Sleep(krakenSleepBetweenCandles)
		}

	}()

	return c, nil
}

func (k *KrakenExchange) GetMessageChan(ctx context.Context, product Product) (<-chan *ExchangeMessage, error) {
	panic("not implemented")
}

func (k *KrakenExchange) GetCandlesticks(product Product, start time.Time, end time.Time, granularity int) ([]*CandlestickModel, error) {
	panic("not implemented")
}

func (k *KrakenExchange) CreateMarketOrder(*MarketOrder) (*CreateMarketOrderResponse, error) {
	panic("not implemented")
}

type FakeKrakenExchange struct {
	KrakenExchange
}

func NewFakeKrakenExchange(client KrakenClient, clock clockwork.Clock) *FakeKrakenExchange {
	return &FakeKrakenExchange{
		KrakenExchange: *NewKrakenExchange(client, clock),
	}
}
