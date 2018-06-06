package vespyr

import (
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	candlestickBaseTime = time.Unix(1223424000, 0)

	// ErrMissingCandlestick is returned when a candlestick is missing from a set
	ErrMissingCandlestick = errors.New("error: a candlestick is missing")
)

// CandlestickBucket returns the candlestick bucket based on the base
// time.
func CandlestickBucket(t time.Time, tickSizeMinutes int64) time.Time {
	divide := int64(t.Sub(candlestickBaseTime).Seconds()) / (tickSizeMinutes * 60)
	return candlestickBaseTime.Add(time.Duration(divide) *
		time.Duration(tickSizeMinutes) * time.Minute)
}

// ValidateCandlesticks ensures that the range of candlesticks is
// valid.
func ValidateCandlesticks(candles []*CandlestickModel, tickSizeMinutes int64) error {
	seen := map[time.Time]bool{}
	for _, c := range candles {
		seen[c.StartTime] = true
	}

	initial := candles[0].StartTime
	for range candles {
		if _, ok := seen[initial]; !ok {
			return ErrMissingCandlestick
		}
		initial = initial.Add(time.Duration(tickSizeMinutes) * time.Minute)
	}

	return nil
}

// ReprojectCandlesticks generates sets of candlesticks using a
// different period. The candlesticks must be passed with ascending
// start time.
func ReprojectCandlesticks(candles []*CandlestickModel, product Product, tickSizeMinutes int64) ([]*CandlestickModel, error) {
	if len(candles) == 0 {
		return nil, errors.New("error: no candles provided")
	}

	start := CandlestickBucket(candles[0].StartTime, tickSizeMinutes)
	end := CandlestickBucket(candles[len(candles)-1].StartTime, tickSizeMinutes)

	var buckets []int
	builders := make(map[time.Time]*CandlestickBuilder)
	for i := start; i.Before(end) || i.Equal(end); i = i.Add(time.Duration(tickSizeMinutes) * time.Minute) {
		buckets = append(buckets, int(i.Unix()))
		builders[i] = NewCandlestickBuilder(product, i,
			i.Add(time.Duration(tickSizeMinutes)*time.Minute))
	}

	sort.Ints(buckets)

	for _, c := range candles {
		bucket := CandlestickBucket(c.StartTime, tickSizeMinutes)
		if builder, ok := builders[bucket]; ok {
			builder.ProcessCandlestickModel(c)
		} else {
			return nil, errors.Errorf("error: missing candlestick builder for bucket: %s", c.StartTime)
		}
	}

	var response []*CandlestickModel
	for _, b := range buckets {
		bucket := time.Unix(int64(b), 0)
		candlestick := builders[bucket].Build()
		response = append(response, candlestick)
	}

	return response, nil
}

// CandlestickBuilder builds a candlestick from a stream of exchange
// messages.
type CandlestickBuilder struct {
	mutex       sync.Mutex
	productType Product
	startTime   time.Time
	endTime     time.Time
	low         float64
	high        float64
	open        float64
	close       float64
	volume      float64
}

// NewCandlestickBuilder returns a new CandlestickBuilder.
func NewCandlestickBuilder(productType Product, start, end time.Time) *CandlestickBuilder {
	return &CandlestickBuilder{
		startTime:   start,
		endTime:     end,
		productType: productType,
	}
}

// Build returns a complete Candlestick.
func (c *CandlestickBuilder) Build() *CandlestickModel {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cs := &CandlestickModel{
		StartTime: c.startTime,
		EndTime:   c.endTime,
		Low:       c.low,
		High:      c.high,
		Open:      c.open,
		Close:     c.close,
		Volume:    c.volume,
		Product:   c.productType,
	}
	if c.close >= c.open {
		cs.Direction = CandlestickDirectionUp
	} else {
		cs.Direction = CandlestickDirectionDown
	}

	return cs
}

// ProcessCandlestickModel adds a candlestick to the builder. This
// should be used for reprojecting a set of candlesticks.
func (c *CandlestickBuilder) ProcessCandlestickModel(e *CandlestickModel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if e.Product != c.productType {
		return
	}
	if e.StartTime.After(c.endTime) {
		return
	}
	if e.StartTime.Before(c.startTime) {
		return
	}
	if e.EndTime.Before(c.startTime) {
		return
	}
	if e.EndTime.After(c.endTime) {
		return
	}

	c.volume += e.Volume

	// Initializers
	if c.low == 0 {
		c.low = e.Low
	}
	if c.high == 0 {
		c.high = e.High
	}
	if c.open == 0 {
		c.open = e.Open
	}
	if c.close == 0 {
		c.close = e.Close
	}

	// Setters
	if e.Low < c.low {
		c.low = e.Low
	}
	if e.High > c.high {
		c.high = e.High
	}
	c.close = e.Close
}

// ProcessMessage processes an exchange message.
func (c *CandlestickBuilder) ProcessMessage(e *ExchangeMessage) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if e.Type != string(MessageMatch) {
		return
	}
	if e.ProductType != string(c.productType) {
		return
	}
	if e.Time.After(c.endTime) {
		return
	}
	if e.Time.Before(c.startTime) {
		return
	}

	c.volume += e.Size

	// Initializers
	if c.low == 0 {
		c.low = e.Price
	}
	if c.high == 0 {
		c.high = e.Price
	}
	if c.open == 0 {
		c.open = e.Price
	}
	if c.close == 0 {
		c.close = e.Price
	}

	// Setters
	if e.Price < c.low {
		c.low = e.Price
	}
	if e.Price > c.high {
		c.high = e.Price
	}
	c.close = e.Price
}
