package vespyr

import (
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PerformOrderArgs are the arguments to the PerformOrder method.
type PerformOrderArgs struct {
	Product         Product
	Side            string
	Cost            float64
	Timeout         time.Duration
	TradingStrategy *TradingStrategyModel
	Candlestick     *CandlestickModel
}

// PerformOrderResponse is the response to PerformOrder.
type PerformOrderResponse struct {
	FilledSize         float64
	FilledSizeCurrency string
	Fees               float64
	FeesCurrency       string
}

// OrderStrategy is an interface for buying or selling a currency.
type OrderStrategy interface {
	PerformOrder(*PerformOrderArgs) (*PerformOrderResponse, error)
	String() string
}

// MarketOrderStrategy executes a buy/sell order using a single market
// order.
type MarketOrderStrategy struct {
	exchange Exchange
	backend  Backend
}

// NewMarketOrderStrategy creates a new market order strategy.
func NewMarketOrderStrategy(exchange Exchange, backend Backend) *MarketOrderStrategy {
	return &MarketOrderStrategy{
		exchange: exchange,
		backend:  backend,
	}
}

// String returns the string representation of the strategy.
func (o *MarketOrderStrategy) String() string {
	return "MarketOrderStrategy"
}

// PerformOrder creates the market order to buy or sell the currency
// amount.
func (o *MarketOrderStrategy) PerformOrder(args *PerformOrderArgs) (*PerformOrderResponse, error) {
	if args.Side != OrderBuy && args.Side != OrderSell {
		return nil, errors.Errorf("error: unknown order side: %s", args.Side)
	}

	order := NewMarketOrder(
		args.Product,
		args.Side,
		args.Cost,
	)

	response, err := o.exchange.CreateMarketOrder(order)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating market order with exchange")
	}

	var costCurrency string
	if args.Side == OrderBuy {
		costCurrency = ProductToMetadata[order.Product].MarketOrderBuyCurrency
	} else {
		costCurrency = ProductToMetadata[order.Product].MarketOrderSellCurrency
	}

	model := &MarketOrderModel{
		ExchangeID:        response.ExchangeID,
		TradingStrategyID: args.TradingStrategy.ID,
		Product:           order.Product,
		Side:              order.Side,
		Cost:              order.Cost,
		CostCurrency:      costCurrency,
		FilledSize:        response.FilledSize,
		SizeCurrency:      response.FilledSizeCurrency,
		Fees:              response.Fees,
		FeesCurrency:      response.FeesCurrency,
	}
	if err := o.backend.CreateMarketOrder(model); err != nil {
		return nil, errors.Wrapf(err, "error creating market order in database")
	}

	logrus.Infof("made %s market order for %f %s with strategy %d costing %f %s with %f %s in fees",
		args.Side, response.FilledSize, response.FilledSizeCurrency, args.TradingStrategy.ID,
		args.Cost, costCurrency, response.Fees, response.FeesCurrency)

	return &PerformOrderResponse{
		FilledSize:         response.FilledSize,
		FilledSizeCurrency: response.FilledSizeCurrency,
		Fees:               response.Fees,
		FeesCurrency:       response.FeesCurrency,
	}, nil
}

// // LimitOrderSelloffStrategy is a order strategy that locates the
// // current market price and creates a limit order at that price. If
// // after a timeout the currency has not been exchanged, then a market
// // order is placed for the remaining amount.
// type LimitOrderSelloffStrategy struct {
// 	exchange Exchange
// 	backend  Backend
// }

// // PerformOrder performs the limit order sell off.
// func (l *LimitOrderSelloffStrategy) PerformOrder(args *PerformOrderArgs) (*PerformOrderResponse, error) {
// 	candle := args.Candlestick

// 	for {
// 		price, err := args.Candlestick.MeanPrice()
// 		if err != nil {
// 			return nil, errors.Wrapf(err, "error fetch candlestick mean price")
// 		}

// 	}

// 	// Set limit order price
// 	// Place limit order
// 	// Check limit order after one minute
// 	// If order not completely processed, fetch new candle and repeat
// 	// If timeout passes, perform market order
// }
