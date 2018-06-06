package vespyr_test

import (
	"testing"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMarketOrderStrategyPerformOrder(t *testing.T) {
	exchange := new(vespyr.MockExchange)
	backend := new(vespyr.MockBackend)
	defer mock.AssertExpectationsForObjects(t, exchange, backend)

	t.Run("buy", func(t *testing.T) {
		ts := &vespyr.TradingStrategyModel{
			ID: 69,
		}
		args := &vespyr.PerformOrderArgs{
			Product:         vespyr.ProductBTCUSD,
			Side:            vespyr.OrderBuy,
			Cost:            5000,
			TradingStrategy: ts,
		}

		exchange.On("CreateMarketOrder", &vespyr.MarketOrder{
			Product: vespyr.ProductBTCUSD,
			Side:    vespyr.OrderBuy,
			Cost:    5000,
		}).Return(&vespyr.CreateMarketOrderResponse{
			ExchangeID:         "exchange-id",
			FilledSize:         1,
			FilledSizeCurrency: vespyr.CurrencyBTC,
			Fees:               2.00,
			FeesCurrency:       vespyr.CurrencyUSD,
		}, nil)

		backend.On("CreateMarketOrder", &vespyr.MarketOrderModel{
			ExchangeID:        "exchange-id",
			TradingStrategyID: 69,
			Product:           vespyr.ProductBTCUSD,
			Side:              vespyr.OrderBuy,
			Cost:              5000,
			CostCurrency:      vespyr.CurrencyUSD,
			FilledSize:        1,
			SizeCurrency:      vespyr.CurrencyBTC,
			Fees:              2,
			FeesCurrency:      vespyr.CurrencyUSD,
		}).Return(nil)

		strategy := vespyr.NewMarketOrderStrategy(exchange, backend)
		response, err := strategy.PerformOrder(args)
		assert.NoError(t, err)

		assert.Equal(t, &vespyr.PerformOrderResponse{
			FilledSize:         1,
			FilledSizeCurrency: vespyr.CurrencyBTC,
			Fees:               2,
			FeesCurrency:       vespyr.CurrencyUSD,
		}, response)
	})

	t.Run("sell", func(t *testing.T) {
		ts := &vespyr.TradingStrategyModel{
			ID: 69,
		}
		args := &vespyr.PerformOrderArgs{
			Product:         vespyr.ProductBTCUSD,
			Side:            vespyr.OrderSell,
			Cost:            2,
			TradingStrategy: ts,
		}

		exchange.On("CreateMarketOrder", &vespyr.MarketOrder{
			Product: vespyr.ProductBTCUSD,
			Side:    vespyr.OrderSell,
			Cost:    2,
		}).Return(&vespyr.CreateMarketOrderResponse{
			ExchangeID:         "exchange-id",
			FilledSize:         10000,
			FilledSizeCurrency: vespyr.CurrencyUSD,
			Fees:               2.00,
			FeesCurrency:       vespyr.CurrencyUSD,
		}, nil)

		backend.On("CreateMarketOrder", &vespyr.MarketOrderModel{
			ExchangeID:        "exchange-id",
			TradingStrategyID: 69,
			Product:           vespyr.ProductBTCUSD,
			Side:              vespyr.OrderSell,
			Cost:              2,
			CostCurrency:      vespyr.CurrencyBTC,
			FilledSize:        10000,
			SizeCurrency:      vespyr.CurrencyUSD,
			Fees:              2,
			FeesCurrency:      vespyr.CurrencyUSD,
		}).Return(nil)

		strategy := vespyr.NewMarketOrderStrategy(exchange, backend)
		response, err := strategy.PerformOrder(args)
		assert.NoError(t, err)

		assert.Equal(t, &vespyr.PerformOrderResponse{
			FilledSize:         10000,
			FilledSizeCurrency: vespyr.CurrencyUSD,
			Fees:               2,
			FeesCurrency:       vespyr.CurrencyUSD,
		}, response)
	})
}
