// Code generated by mockery v1.0.0
package vespyr

import krakenapi "github.com/DavidHuie/kraken-go-api-client"
import mock "github.com/stretchr/testify/mock"

// MockKrakenClient is an autogenerated mock type for the KrakenClient type
type MockKrakenClient struct {
	mock.Mock
}

// OHLC provides a mock function with given fields: pair, last
func (_m *MockKrakenClient) OHLC(pair string, last ...int64) (*krakenapi.OHLCResponse, error) {
	_va := make([]interface{}, len(last))
	for _i := range last {
		_va[_i] = last[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, pair)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *krakenapi.OHLCResponse
	if rf, ok := ret.Get(0).(func(string, ...int64) *krakenapi.OHLCResponse); ok {
		r0 = rf(pair, last...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*krakenapi.OHLCResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, ...int64) error); ok {
		r1 = rf(pair, last...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
