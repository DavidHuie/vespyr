// Code generated by mockery v1.0.0
package vespyr

import mock "github.com/stretchr/testify/mock"
import rand "math/rand"

// MockStrategyGenome is an autogenerated mock type for the StrategyGenome type
type MockStrategyGenome struct {
	mock.Mock
}

// Buy provides a mock function with given fields: history, current
func (_m *MockStrategyGenome) Buy(history []*IndicatorSet, current int) (bool, error) {
	ret := _m.Called(history, current)

	var r0 bool
	if rf, ok := ret.Get(0).(func([]*IndicatorSet, int) bool); ok {
		r0 = rf(history, current)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]*IndicatorSet, int) error); ok {
		r1 = rf(history, current)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Clone provides a mock function with given fields:
func (_m *MockStrategyGenome) Clone() StrategyGenome {
	ret := _m.Called()

	var r0 StrategyGenome
	if rf, ok := ret.Get(0).(func() StrategyGenome); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(StrategyGenome)
		}
	}

	return r0
}

// Crossover provides a mock function with given fields: m, r
func (_m *MockStrategyGenome) Crossover(m StrategyGenome, r *rand.Rand) (StrategyGenome, StrategyGenome) {
	ret := _m.Called(m, r)

	var r0 StrategyGenome
	if rf, ok := ret.Get(0).(func(StrategyGenome, *rand.Rand) StrategyGenome); ok {
		r0 = rf(m, r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(StrategyGenome)
		}
	}

	var r1 StrategyGenome
	if rf, ok := ret.Get(1).(func(StrategyGenome, *rand.Rand) StrategyGenome); ok {
		r1 = rf(m, r)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(StrategyGenome)
		}
	}

	return r0, r1
}

// Indicators provides a mock function with given fields:
func (_m *MockStrategyGenome) Indicators() []Indicator {
	ret := _m.Called()

	var r0 []Indicator
	if rf, ok := ret.Get(0).(func() []Indicator); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Indicator)
		}
	}

	return r0
}

// Mutate provides a mock function with given fields: rng
func (_m *MockStrategyGenome) Mutate(rng *rand.Rand) {
	_m.Called(rng)
}

// Rand provides a mock function with given fields: rng
func (_m *MockStrategyGenome) Rand(rng *rand.Rand) {
	_m.Called(rng)
}

// Sell provides a mock function with given fields: history, current
func (_m *MockStrategyGenome) Sell(history []*IndicatorSet, current int) (bool, error) {
	ret := _m.Called(history, current)

	var r0 bool
	if rf, ok := ret.Get(0).(func([]*IndicatorSet, int) bool); ok {
		r0 = rf(history, current)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]*IndicatorSet, int) error); ok {
		r1 = rf(history, current)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetTradingStrategy provides a mock function with given fields: t
func (_m *MockStrategyGenome) SetTradingStrategy(t *TradingStrategyModel) {
	_m.Called(t)
}

// String provides a mock function with given fields:
func (_m *MockStrategyGenome) String() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
