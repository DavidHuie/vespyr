package vespyr

import (
	"context"
	"time"

	"strings"

	coinbase "github.com/DavidHuie/go-coinbase-exchange"
	"github.com/gorilla/websocket"
	"github.com/jonboulle/clockwork"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	gdaxOrderMarket = "market"
	gdaxRetries     = 20
)

// GDAXClient is the interface for an underlying GDAX client.
type GDAXClient interface {
	GetHistoricRates(product string, p ...coinbase.GetHistoricRatesParams) ([]coinbase.HistoricRate, error)
	CreateOrder(*coinbase.Order) (coinbase.Order, error)
	GetOrder(string) (coinbase.Order, error)
}

// GDAXExchange is a client to the GDAX cryptocurrency exchange.
type GDAXExchange struct {
	client GDAXClient
	clock  clockwork.Clock
}

// NewGDAXExchange returns a new instance of GDAXExchange.
func NewGDAXExchange(client GDAXClient, clock clockwork.Clock) *GDAXExchange {
	return &GDAXExchange{
		client: client,
		clock:  clock,
	}
}

func (k *GDAXExchange) EmitsFullCandlesticks() bool {
	return false
}

// GetMessageChan returns a channel that emits exchange messages.
func (g *GDAXExchange) GetMessageChan(ctx context.Context, product Product) (<-chan *ExchangeMessage, error) {
	var wsDialer websocket.Dialer

	wsConn, _, err := wsDialer.Dial("wss://ws-feed.gdax.com", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening websocket connection to GDAX")
	}

	subscribe := map[string]string{
		"type":       "subscribe",
		"product_id": string(product),
	}
	if err := wsConn.WriteJSON(subscribe); err != nil {
		return nil, errors.Wrapf(err, "error writing to websocket")
	}

	c := make(chan *ExchangeMessage)

	go func() {
		for {
			select {
			case <-ctx.Done():
				wsConn.Close()
				return
			default:

			}

			var message coinbase.Message
			if err := wsConn.ReadJSON(&message); err != nil {
				logrus.WithError(err).Errorf("error reading from websocket")
				close(c)
				return
			}

			exchangeMessage := &ExchangeMessage{
				Price:       message.Price,
				ProductType: message.ProductId,
				Size:        message.Size,
				Type:        message.Type,
				Time:        message.Time.Time(),
			}

			c <- exchangeMessage
		}
	}()

	return c, nil
}

func (g *GDAXExchange) StreamCandlesticks(ctx context.Context, product Product) (<-chan *CandlestickModel, error) {
	panic("not implemented")
}

// GetCandlesticks returns candlesticks for a specified period and
// granularity from GDAX.
func (g *GDAXExchange) GetCandlesticks(product Product, start, end time.Time, granularitySeconds int) ([]*CandlestickModel, error) {
	intervals := start.Sub(end).Seconds() / float64(granularitySeconds)
	if intervals >= 200 {
		return nil, errors.New("error: too many candlesticks requested of GDAX")
	}

	historicRates, err := g.client.GetHistoricRates(string(product), coinbase.GetHistoricRatesParams{
		Start:       start,
		End:         end,
		Granularity: granularitySeconds,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching historic rates from GDAX")
	}

	var candles []*CandlestickModel
	for _, h := range historicRates {
		direction := CandlestickDirectionUp
		if h.Close < h.Open {
			direction = CandlestickDirectionDown
		}
		candles = append(candles, &CandlestickModel{
			StartTime: h.Time,
			EndTime:   h.Time.Add(time.Second * time.Duration(granularitySeconds)),
			Low:       h.Low,
			High:      h.High,
			Open:      h.Open,
			Close:     h.Close,
			Volume:    h.Volume,
			Direction: direction,
			Product:   product,
		})
	}

	return candles, nil
}

// CreateMarketOrder creates a market order on GDAX.
func (g *GDAXExchange) CreateMarketOrder(args *MarketOrder) (*CreateMarketOrderResponse, error) {
	order := &coinbase.Order{
		Type:      gdaxOrderMarket,
		Side:      args.Side,
		ProductId: string(args.Product),
	}

	response := &CreateMarketOrderResponse{}

	if args.Side == OrderBuy {
		switch args.Product {
		case ProductBTCUSD:
			order.Funds = args.Cost
			response.FilledSizeCurrency = CurrencyBTC
			response.FeesCurrency = CurrencyUSD
		case ProductETHUSD:
			order.Funds = args.Cost
			response.FilledSizeCurrency = CurrencyETH
			response.FeesCurrency = CurrencyUSD
		case ProductLTCUSD:
			order.Funds = args.Cost
			response.FilledSizeCurrency = CurrencyLTC
			response.FeesCurrency = CurrencyUSD
		default:
			return nil, errors.Errorf("error: unrecognized product type: %s", args.Product)
		}

		logrus.Debugf("creating GDAX %s market order of size %f for %s", order.Side,
			args.Cost, args.Product)
	} else if args.Side == OrderSell {
		switch args.Product {
		case ProductBTCUSD:
			order.Size = args.Cost
			response.FilledSizeCurrency = CurrencyUSD
			response.FeesCurrency = CurrencyUSD
		case ProductETHUSD:
			order.Size = args.Cost
			response.FilledSizeCurrency = CurrencyUSD
			response.FeesCurrency = CurrencyUSD
		case ProductLTCUSD:
			order.Size = args.Cost
			response.FilledSizeCurrency = CurrencyUSD
			response.FeesCurrency = CurrencyUSD
		default:
			return nil, errors.Errorf("error: unrecognized product type: %s", args.Product)
		}

		logrus.Debugf("creating GDAX %s market order of size %f for %s", order.Side,
			args.Cost, args.Product)
	}

	logrus.Debugf("GDAX create market order args: %#v", order)

	gdaxResponse, err := g.client.CreateOrder(order)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating GDAX order")
	}

	logrus.Debugf("GDAX create market order response: %#v", gdaxResponse)

	finalizedOrder := false
	orderID := gdaxResponse.Id

RetryLoop:
	for i := 0; i < gdaxRetries; i++ {
		gdaxResponse, err = g.client.GetOrder(orderID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "rate limit") {
				logrus.Warnf("error while getting GDAX order status, retrying: %s", err)
				g.clock.Sleep(time.Second << uint(i))
				continue
			}
			return nil, errors.Wrapf(err, "error fetching GDAX order")
		}

		logrus.Debugf("GDAX get market order response: %#v", gdaxResponse)

		switch gdaxResponse.Status {
		case "open", "pending", "active":
			logrus.Debugf("GDAX market order not filled yet, status: %s", gdaxResponse.Status)
			g.clock.Sleep(time.Second << uint(i))
			continue
		case "done":
			if gdaxResponse.Settled {
				finalizedOrder = true
				break RetryLoop
			}
			logrus.Debugf("GDAX market order done but not settled, retrying")
			g.clock.Sleep(time.Second << uint(i))
			continue
		default:
			return nil, errors.Errorf("error: unknown GDAX order status: %s", gdaxResponse.Status)
		}
	}

	if !finalizedOrder {
		return nil, errors.Errorf("error: could not get market order status after retrying")
	}

	response.ExchangeID = gdaxResponse.Id
	response.Fees = gdaxResponse.FillFees
	if args.Side == OrderBuy {
		response.FilledSize = gdaxResponse.FilledSize
	} else {
		response.FilledSize = gdaxResponse.ExecutedValue - gdaxResponse.FillFees
	}

	return response, nil
}

// FakeGDAXExchange is a GDAX implementation that places mock trades.
type FakeGDAXExchange struct {
	GDAXExchange
}

// NewFakeGDAXExchange returns a new instance of FakeGDAXExchange.
func NewFakeGDAXExchange(clock clockwork.Clock) *FakeGDAXExchange {
	return &FakeGDAXExchange{
		GDAXExchange: *NewGDAXExchange(nil, clock),
	}
}

// CreateMarketOrder mocks a market order at the last market price.
func (f *FakeGDAXExchange) CreateMarketOrder(args *MarketOrder) (*CreateMarketOrderResponse, error) {
	ctx := context.Background()
	messageChan, err := f.GetMessageChan(ctx, args.Product)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting GDAX message channel")
	}

	var message *ExchangeMessage
	for {
		message = <-messageChan
		if message.Type == string(MessageMatch) && string(args.Product) == message.ProductType {
			break
		}
	}

	price := message.Price

	if args.Side == OrderBuy {
		total := (1 - .0025) * args.Cost / price
		fees := .0025 * args.Cost

		return &CreateMarketOrderResponse{
			ExchangeID:         uuid.NewV4().String(),
			FilledSize:         total,
			FilledSizeCurrency: ProductToMetadata[Product(args.Product)].MarketOrderSellCurrency,
			Fees:               fees,
			FeesCurrency:       ProductToMetadata[Product(args.Product)].MarketOrderFeesCurrency,
		}, nil
	}

	// Sell
	total := args.Cost * price * (1 - .0025)
	fees := .0025 * args.Cost * price

	return &CreateMarketOrderResponse{
		ExchangeID:         uuid.NewV4().String(),
		FilledSize:         total,
		FilledSizeCurrency: ProductToMetadata[Product(args.Product)].MarketOrderBuyCurrency,
		Fees:               fees,
		FeesCurrency:       ProductToMetadata[Product(args.Product)].MarketOrderFeesCurrency,
	}, nil
}
