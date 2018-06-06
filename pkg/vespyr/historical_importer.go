package vespyr

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	importerBatchSize                 = 150
	importerMaxRetries                = 20
	gdaxCandlestickGranularitySeconds = 60
)

// HistoricalImporter is a type that can import candle stick
// information into the backend from a specific period of time.
type HistoricalImporter struct {
	product     Product
	exchange    Exchange
	backend     Backend
	concurrency int
}

// NewHistoricalImporter creates a new HistoricalImporter.
func NewHistoricalImporter(product Product, e Exchange, b Backend, concurrency int) *HistoricalImporter {
	return &HistoricalImporter{
		product:     product,
		exchange:    e,
		backend:     b,
		concurrency: concurrency,
	}
}

func (h *HistoricalImporter) processBatch(start, end time.Time) error {
	candlesticks, err := h.exchange.GetCandlesticks(h.product,
		start, end, gdaxCandlestickGranularitySeconds)
	if err != nil {
		return errors.Wrapf(err, "error fetching candlesticks")
	}

	for _, c := range candlesticks {
		if err := h.backend.UpsertCandlestick(c); err != nil {
			return errors.Wrapf(err, "error upserting candlestick")
		}
	}

	return nil
}

func (h *HistoricalImporter) worker(c <-chan *importerMsg) {
	for msg := range c {
		logrus.Debugf("processing import message: %#v", msg)

		retries := 0
		for {
			if retries >= importerMaxRetries {
				logrus.WithField("HistoricalImporter", msg).
					Errorf("error too many retries for importer")
				break
			}

			if err := h.processBatch(msg.start, msg.end); err != nil {
				logrus.WithError(err).Warnf("error processing batch, retrying")
				time.Sleep(time.Second << uint(retries))
				retries++
			} else {
				break
			}
		}
	}
}

type importerMsg struct {
	start, end  time.Time
	granularity int
}

// Import imports a time range into a backend.
func (h *HistoricalImporter) Import(start, end time.Time) {
	// Start workers
	c := make(chan *importerMsg)
	wg := &sync.WaitGroup{}
	for i := 0; i < h.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.worker(c)
		}()
	}

	logrus.Infof("starting import from %s to %s", start, end)

	for i := start; i.Before(end) || i.Equal(end); i = i.Add(importerBatchSize * time.Minute) {
		logrus.Infof("processing batch at time: %s", i)

		msg := &importerMsg{
			start: i,
			end:   i.Add(importerBatchSize * time.Minute),
		}
		c <- msg
	}

	close(c)
	wg.Wait()
}
