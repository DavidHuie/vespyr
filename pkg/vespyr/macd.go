package vespyr

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// MACDIndicator implements the MACD indicator.
type MACDIndicator struct {
	shortEMAPeriod uint
	longEMAPeriod  uint
	shortEMA       *EMAIndicator
	longEMA        *EMAIndicator
	lastTime       time.Time
}

// NewMACDIndicator creates a new MACD indicator.
func NewMACDIndicator(shortPeriod, longPeriod uint) *MACDIndicator {
	return &MACDIndicator{
		shortEMAPeriod: shortPeriod,
		longEMAPeriod:  longPeriod,
		shortEMA:       NewEMAIndicator(shortPeriod),
		longEMA:        NewEMAIndicator(longPeriod),
	}
}

// AddCandlestick adds a candlestick to the indicator.
func (m *MACDIndicator) AddCandlestick(c *CandlestickModel) error {
	if c.Volume == 0 {
		return nil
	}

	if err := m.shortEMA.AddCandlestick(c); err != nil {
		return errors.Wrapf(err, "error adding candlestick to MACD short EMA")
	}
	if err := m.longEMA.AddCandlestick(c); err != nil {
		return errors.Wrapf(err, "error adding candlestick to MACD long EMA")
	}

	m.lastTime = c.StartTime

	return nil
}

// Name return the name of the indicator.
func (m *MACDIndicator) Name() string {
	return fmt.Sprintf("%s-%d-%d", IndicatorMACD, m.shortEMAPeriod, m.longEMAPeriod)
}

// Value value returns the value of the indicator.
func (m *MACDIndicator) Value() (*IndicatorValue, error) {
	if m.lastTime.IsZero() {
		return nil, ErrNotEnoughData
	}

	short, err := m.shortEMA.Value()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting MACD short EMA value")
	}
	long, err := m.longEMA.Value()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting MACD long EMA value")
	}

	return &IndicatorValue{
		Time:          m.lastTime,
		Value:         short.Value - long.Value,
		IndicatorName: m.Name(),
	}, nil
}

// MACDWithSignal is an indicator that tracks the MACD value minus the
// signal EMA.
type MACDWithSignal struct {
	shortEMAPeriod uint
	longEMAPeriod  uint
	signalPeriod   uint
	signalEMA      *EMAIndicator
	macd           *MACDIndicator
	lastTime       time.Time
}

// NewMACDWithSignal returns a new MACDWithSignal.
func NewMACDWithSignal(shortPeriod, longPeriod, signalPeriod uint) *MACDWithSignal {
	return &MACDWithSignal{
		shortEMAPeriod: shortPeriod,
		longEMAPeriod:  longPeriod,
		signalPeriod:   signalPeriod,
		signalEMA:      NewEMAIndicator(signalPeriod),
		macd:           NewMACDIndicator(shortPeriod, longPeriod),
	}
}

// Name return the name of the indicator.
func (m *MACDWithSignal) Name() string {
	return fmt.Sprintf("%s-%d-ema-%d-%d", IndicatorMACDWithSignal,
		m.signalPeriod, m.shortEMAPeriod, m.longEMAPeriod)
}

// AddCandlestick adds a candlestick to the indicator.
func (m *MACDWithSignal) AddCandlestick(c *CandlestickModel) error {
	if c.Volume == 0 {
		return nil
	}

	if err := m.macd.AddCandlestick(c); err != nil {
		return errors.Wrapf(err, "error adding candlestick to MACD")
	}

	value, err := m.macd.Value()
	if err != nil {
		if errors.Cause(err) != ErrNotEnoughData {
			return errors.Wrapf(err, "error getting MACD value")
		}
	}
	if value != nil {
		signalCandlestick := &CandlestickModel{
			Low:    value.Value,
			High:   value.Value,
			Open:   value.Value,
			Close:  value.Value,
			Volume: value.Value,
		}
		if err := m.signalEMA.AddCandlestick(signalCandlestick); err != nil {
			return errors.Wrapf(err, "error adding candlestick to signal EMA")
		}
	}

	m.lastTime = c.StartTime

	return nil
}

// Value value returns the value of the indicator.
func (m *MACDWithSignal) Value() (*IndicatorValue, error) {
	if m.lastTime.IsZero() {
		return nil, ErrNotEnoughData
	}

	signal, err := m.signalEMA.Value()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting signal EMA value")
	}
	macd, err := m.macd.Value()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting MACD value")
	}

	return &IndicatorValue{
		Time:          m.lastTime,
		Value:         macd.Value - signal.Value,
		IndicatorName: m.Name(),
	}, nil
}
