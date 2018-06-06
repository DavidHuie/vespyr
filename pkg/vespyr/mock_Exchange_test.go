// Code generated by mockery v1.0.0
package vespyr

import context "context"
import mock "github.com/stretchr/testify/mock"
import time "time"

// MockExchange is an autogenerated mock type for the Exchange type
type MockExchange struct {
	mock.Mock
}

// CreateMarketOrder provides a mock function with given fields: _a0
func (_m *MockExchange) CreateMarketOrder(_a0 *MarketOrder) (*CreateMarketOrderResponse, error) {
	ret := _m.Called(_a0)

	var r0 *CreateMarketOrderResponse
	if rf, ok := ret.Get(0).(func(*MarketOrder) *CreateMarketOrderResponse); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*CreateMarketOrderResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*MarketOrder) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EmitsFullCandlesticks provides a mock function with given fields:
func (_m *MockExchange) EmitsFullCandlesticks() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// GetCandlesticks provides a mock function with given fields: product, start, end, granularity
func (_m *MockExchange) GetCandlesticks(product Product, start time.Time, end time.Time, granularity int) ([]*CandlestickModel, error) {
	ret := _m.Called(product, start, end, granularity)

	var r0 []*CandlestickModel
	if rf, ok := ret.Get(0).(func(Product, time.Time, time.Time, int) []*CandlestickModel); ok {
		r0 = rf(product, start, end, granularity)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*CandlestickModel)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(Product, time.Time, time.Time, int) error); ok {
		r1 = rf(product, start, end, granularity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMessageChan provides a mock function with given fields: _a0, _a1
func (_m *MockExchange) GetMessageChan(_a0 context.Context, _a1 Product) (<-chan *ExchangeMessage, error) {
	ret := _m.Called(_a0, _a1)

	var r0 <-chan *ExchangeMessage
	if rf, ok := ret.Get(0).(func(context.Context, Product) <-chan *ExchangeMessage); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *ExchangeMessage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, Product) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StreamCandlesticks provides a mock function with given fields: ctx, product
func (_m *MockExchange) StreamCandlesticks(ctx context.Context, product Product) (<-chan *CandlestickModel, error) {
	ret := _m.Called(ctx, product)

	var r0 <-chan *CandlestickModel
	if rf, ok := ret.Get(0).(func(context.Context, Product) <-chan *CandlestickModel); ok {
		r0 = rf(ctx, product)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *CandlestickModel)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, Product) error); ok {
		r1 = rf(ctx, product)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
