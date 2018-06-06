package vespyr

import (
	"fmt"
	"sync"
	"time"
)

type CacheingBackend struct {
	Backend
	sync.Mutex
	cache map[string]interface{}
}

func NewCacheingBackend(b Backend) Backend {
	return &CacheingBackend{
		Backend: b,
		cache:   make(map[string]interface{}),
	}
}

func (c *CacheingBackend) FindCandlesticks(s time.Time, e time.Time,
	p Product, t int64) ([]*CandlestickModel, error) {
	key := fmt.Sprintf("find-candlesticks-%s-%s-%s-%d", s, e, p, t)
	type responseStruct struct {
		models []*CandlestickModel
		err    error
	}

	if value, ok := c.cache[key]; ok {
		response := value.(*responseStruct)
		return response.models, response.err
	}

	c.Lock()
	defer c.Unlock()

	candles, err := c.Backend.FindCandlesticks(s, e, p, t)
	response := &responseStruct{candles, err}
	c.cache[key] = response

	return response.models, response.err
}
