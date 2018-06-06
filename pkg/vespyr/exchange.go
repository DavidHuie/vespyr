package vespyr

import (
	"context"
	"time"
)

// Product is a trading product.
type Product string

// CandlestickDirection refers to a candlestick direction.
type CandlestickDirection string

// ExchangeMessageType refers to a message type that's emitted by an
// exchange.
type ExchangeMessageType string

// ExchangeType defines the type of the exchange.
type ExchangeType string

const (
	ExchangeGDAX   ExchangeType = "gdax"
	ExchangeKraken ExchangeType = "kraken"

	// ProductBTCUSD refers to a Bitcoin trading product with
	// units in Dollars.
	ProductBTCUSD Product = "BTC-USD"
	// ProductETHUSD refers to an Ethereum trading product with
	// units in Dollars.
	ProductETHUSD Product = "ETH-USD"
	// ProductLTCUSD refers to an Litecoin trading product with
	// units in Dollars.
	ProductLTCUSD   Product = "LTC-USD"
	ProductXMRUSD   Product = "XMR-USD"
	ProductBCHUSD   Product = "BCH-USD"
	ProductDashUSD  Product = "DASH-USD"
	ProductZcashUSD Product = "ZEC-USD"
	ProductXRPUSD   Product = "XRP-USD"
	// ProductETCUSD   Product = "ETC-USD"

	// CurrencyUSD refers to US dollars.
	CurrencyUSD = "USD"
	// CurrencyBTC refers to Bitcoin.
	CurrencyBTC = "BTC"
	// CurrencyETH is the Ethereum currency.
	CurrencyETH = "ETH"
	// CurrencyLTC is the Litecoin currency.
	CurrencyLTC = "LTC"
	// CurrencyXMR is the Monero currency.
	CurrencyXMR = "XMR"
	// CurrencyBCH is the Bitcoin Cash currency.
	CurrencyBCH = "BCH"
	// CurrencyDash is the Dash currency.
	CurrencyDash = "DASH"
	// CurrencyZcash is the Zcash currency.
	CurrencyZcash = "ZEC"
	// CurrencyXRP is the Ripple currency.
	CurrencyXRP = "XRP"
	// CurrencyETC is the Ethereum Classic currency.
	CurrencyETC = "ETC"

	// MessageMatch is an exchange message that refers to a match
	// between a buy and sell order.
	MessageMatch ExchangeMessageType = "match"

	// CandlestickDirectionUp refers to a candlestick that points
	// up.
	CandlestickDirectionUp CandlestickDirection = "up"
	// CandlestickDirectionDown refers to a candlestick that points
	// down.
	CandlestickDirectionDown CandlestickDirection = "down"

	// OrderBuy indicates a buy position on an order.
	OrderBuy = "buy"
	// OrderSell indicates a sell position on an order.
	OrderSell = "sell"
)

// ProductMetadata defines metadata about each product.
type ProductMetadata struct {
	ExchangeType            ExchangeType
	MarketOrderBuyCurrency  string
	MarketOrderSellCurrency string
	MarketOrderFeesCurrency string
}

// ProductToMetadata is a singleton with metadata about each product.
var ProductToMetadata = map[Product]*ProductMetadata{
	ProductBTCUSD: &ProductMetadata{
		ExchangeType:            ExchangeGDAX,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyBTC,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductETHUSD: &ProductMetadata{
		ExchangeType:            ExchangeGDAX,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyETH,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductLTCUSD: &ProductMetadata{
		ExchangeType:            ExchangeGDAX,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyLTC,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductXMRUSD: &ProductMetadata{
		ExchangeType:            ExchangeKraken,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyXMR,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductBCHUSD: &ProductMetadata{
		ExchangeType:            ExchangeKraken,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyBCH,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductDashUSD: &ProductMetadata{
		ExchangeType:            ExchangeKraken,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyDash,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductZcashUSD: &ProductMetadata{
		ExchangeType:            ExchangeKraken,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyZcash,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	ProductXRPUSD: &ProductMetadata{
		ExchangeType:            ExchangeKraken,
		MarketOrderBuyCurrency:  CurrencyUSD,
		MarketOrderSellCurrency: CurrencyXRP,
		MarketOrderFeesCurrency: CurrencyUSD,
	},
	// ProductETCUSD: &ProductMetadata{
	// 	ExchangeType:            ExchangeKraken,
	// 	MarketOrderBuyCurrency:  CurrencyUSD,
	// 	MarketOrderSellCurrency: CurrencyETC,
	// 	MarketOrderFeesCurrency: CurrencyUSD,
	// },
}

// ExchangeMessage is emitted by an exchange representing an action
// that occurred on the exchange.
type ExchangeMessage struct {
	Price       float64
	ProductType string
	Size        float64
	Type        string
	Time        time.Time
}

// MarketOrder describes the settings for a MarketOrder.
type MarketOrder struct {
	Product Product
	Side    string
	Cost    float64
}

// NewMarketOrder instantiates a new market order.
func NewMarketOrder(product Product, side string, cost float64) *MarketOrder {
	return &MarketOrder{
		Product: product,
		Side:    side,
		Cost:    cost,
	}
}

// Exchange represents a connection to a trading exchange.
type Exchange interface {
	GetMessageChan(context.Context, Product) (<-chan *ExchangeMessage, error)
	GetCandlesticks(product Product, start, end time.Time, granularity int) ([]*CandlestickModel, error)
	CreateMarketOrder(*MarketOrder) (*CreateMarketOrderResponse, error)
	StreamCandlesticks(ctx context.Context, product Product) (<-chan *CandlestickModel, error)
	EmitsFullCandlesticks() bool
}

// CreateMarketOrderResponse is the create market order response.
type CreateMarketOrderResponse struct {
	ExchangeID         string
	FilledSize         float64
	FilledSizeCurrency string
	Fees               float64
	FeesCurrency       string
}
