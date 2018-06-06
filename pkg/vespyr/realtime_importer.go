package vespyr

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	dbCandlestickBucketSize = 1
)

// RealtimeImporter imports exchange data into a backend.
type RealtimeImporter struct {
	mutex    sync.Mutex
	candles  map[int64]*CandlestickBuilder
	backend  Backend
	product  Product
	exchange Exchange
}

// NewRealtimeImporter instantiates a new RealtimeImporter.
func NewRealtimeImporter(p Product, b Backend, e Exchange) *RealtimeImporter {
	return &RealtimeImporter{
		product:  p,
		backend:  b,
		exchange: e,
		candles:  make(map[int64]*CandlestickBuilder),
	}
}

// Start begins the import process.
func (i *RealtimeImporter) Start(flushInterval time.Duration) {
	if i.exchange.EmitsFullCandlesticks() {
		i.StartProcessingCandlesticks()
	} else {
		i.StartProcessingMessages(flushInterval)
	}
}

// StartProcessingCandlesticks processes candlesticks as they're
// emitted by the exchange.
func (i *RealtimeImporter) StartProcessingCandlesticks() {
	logrus.Printf("starting realtime importer: %s", i.product)

	for {
		c, err := i.exchange.StreamCandlesticks(context.Background(), i.product)
		if err != nil {
			logrus.WithError(err).Errorf("error getting exchange message channel for product %s", i.product)
			time.Sleep(5 * time.Second)
			continue
		}

		for candlestick := range c {
			logrus.Infof("flushing candlestick(%s) with volume %f for product %s", candlestick.StartTime,
				candlestick.Volume, i.product)

			if err := i.backend.UpsertCandlestick(candlestick); err != nil {
				logrus.WithError(err).Errorf("error upserting candlestick")
				continue
			}
		}
	}
}

// StartProcessingMessages begins the import process with messages.
func (i *RealtimeImporter) StartProcessingMessages(flushInterval time.Duration) {
	logrus.Printf("starting realtime importer: %s", i.product)

	ticker := time.NewTicker(flushInterval)
	go func() {
		for {
			<-ticker.C
			i.Flush()
		}
	}()

	for {
		c, err := i.exchange.GetMessageChan(context.Background(), i.product)
		if err != nil {
			logrus.WithError(err).Errorf("error getting exchange message channel")
			time.Sleep(5 * time.Second)
			continue
		}

		for msg := range c {
			i.ProcessExchangeMessage(msg)
		}
	}
}

// ProcessExchangeMessage processes an exchange message, sending the
// message to the appropriate candlestick.
func (i *RealtimeImporter) ProcessExchangeMessage(msg *ExchangeMessage) {
	bucket := CandlestickBucket(msg.Time, dbCandlestickBucketSize)
	unixBucket := bucket.Unix()

	i.mutex.Lock()
	if i.candles[unixBucket] == nil {
		i.candles[unixBucket] = NewCandlestickBuilder(
			i.product,
			bucket,
			bucket.Add(time.Minute),
		)
	}
	i.mutex.Unlock()

	candle, ok := i.candles[unixBucket]
	if !ok {
		logrus.Errorf("error: did not find candle bucket for %v", unixBucket)
	}

	candle.ProcessMessage(msg)
}

// Flush flushes all populated candlesticks to the backend.
func (i *RealtimeImporter) Flush() {
	// Sort keys
	var buckets []int
	for bucket := range i.candles {
		buckets = append(buckets, int(bucket))
	}
	sort.Ints(buckets)

	// We need at least two buckets in order to flush since the
	// last one is the one that's being processed.
	if len(buckets) < 2 {
		return
	}

	// Flush all except last key to backend
	for j := 0; j < len(buckets)-1; j++ {
		bucket := int64(buckets[j])
		builder, ok := i.candles[bucket]
		if !ok {
			logrus.Errorf("error: did not find candle bucket for %v", bucket)
		}

		candle := builder.Build()

		if candle.Volume > 0 {
			logrus.Infof("flushing candlestick(%s) with volume %f for product %s", candle.StartTime,
				candle.Volume, i.product)
			if err := i.backend.UpsertCandlestick(candle); err != nil {
				logrus.WithError(err).Errorf("error upserting candlestick")
				continue
			}
		}

		i.mutex.Lock()
		delete(i.candles, int64(buckets[j]))
		i.mutex.Unlock()
	}
}
