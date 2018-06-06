package vespyr

import (
	"context"
	"fmt"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Bot is a type that runs automated trading strategies at each tick.
type Bot struct {
	tickCheckInterval     time.Duration
	backend               Backend
	exchange              Exchange
	clock                 clockwork.Clock
	currentCandlestick    *CandlestickModel
	tradingStrategyModels []*TradingStrategyModel
	tradingStrategies     []*TradingStrategy
	product               Product
}

// NewBot returns a new instance of Bot.
func NewBot(tickCheckInterval time.Duration, clock clockwork.Clock,
	backend Backend, exchange Exchange,
	product Product) *Bot {
	return &Bot{
		tickCheckInterval: tickCheckInterval,
		backend:           backend,
		exchange:          exchange,
		clock:             clock,
		product:           product,
	}
}

// Run runs the bot until a cancellation signal comes in.
func (b *Bot) Run(ctx context.Context) {
	logrus.Debugf("starting %s bot", b.product)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := b.ProcessTickForNewCandlestick(); err != nil {
			logrus.WithError(err).Errorf("error processing tick for candlestick")
		}

		b.clock.Sleep(b.tickCheckInterval)
	}
}

// ProcessTickForNewCandlestick processes the current tick is there's
// a new candlestick available.
func (b *Bot) ProcessTickForNewCandlestick() error {
	nextCandlestick, err := b.backend.FindMostRecentCandlestick(b.product)
	if err != nil {
		return errors.Wrapf(err, "error finding next candlestick")
	}
	if b.currentCandlestick == nil || nextCandlestick.ID != b.currentCandlestick.ID {
		logrus.Debugf("found a more recent %s candlestick, processing ticks", b.product)
		if err := b.ProcessTick(nextCandlestick.EndTime); err != nil {
			return errors.Wrapf(err, "error processing candlestick")
		}
	}

	b.currentCandlestick = nextCandlestick
	return nil
}

func (b *Bot) initializeStrategies(t time.Time) error {
	logrus.Debugf("initializing %s strategies", b.product)

	strategies, err := b.backend.FindActiveTradingStrategies(b.product)
	if err != nil {
		return errors.Wrapf(err, "error finding active trading strategies")
	}

	b.tradingStrategies = []*TradingStrategy{}
	b.tradingStrategyModels = []*TradingStrategyModel{}

	for _, strategy := range strategies {
		meta, err := strategy.Strategy()
		if err != nil {
			return errors.Wrapf(err, "error extracting strategy from model")
		}

		logrus.Debugf("initializing %s %s strategy: %d", b.product,
			strategy.TradingStrategy, strategy.ID)

		service := NewTradingStrategy(b.backend, b.exchange,
			meta, b.clock)

		historyStart := t.Add(-time.Duration(strategy.HistoryTicks) *
			time.Duration(strategy.TickSizeMinutes) * time.Minute)

		candles, err := b.backend.FindCandlesticks(historyStart, t,
			strategy.Product, int64(strategy.TickSizeMinutes))
		if err != nil {
			return errors.Wrapf(err, "error finding candlesticks for strategy")
		}
		for _, candle := range candles {
			if err := service.SeedIndicators(candle); err != nil {
				return errors.Wrapf(err, "error seeding strategy indicators")
			}
		}

		b.tradingStrategyModels = append(b.tradingStrategyModels, strategy)
		b.tradingStrategies = append(b.tradingStrategies, service)
	}

	return nil
}

// ProcessTick a tick, calling all strategies that have to run.
func (b *Bot) ProcessTick(t time.Time) error {
	if err := b.initializeStrategies(t); err != nil {
		return errors.Wrapf(err, "error initializing strategies")
	}

	for i, model := range b.tradingStrategyModels {
		if model.NextTickAt.IsZero() || !model.DeactivatedAt.IsZero() {
			logrus.Debugf("skipping strategy: %d", model.ID)
			continue
		}
		if model.NextTickAt.Equal(t) || model.NextTickAt.Before(t) {
			logrus.Debugf("processing tick for %s strategy: %d", b.product, model.ID)

			// TODO: processes ticks in parallel so that
			// there isn't latency in running each
			// strategy.
			if err := b.processTick(model, b.tradingStrategies[i], t); err != nil {
				return errors.Wrapf(err, "error processing tick for strategy")
			}
		}
	}

	return nil
}

func (b *Bot) processTick(model *TradingStrategyModel, service *TradingStrategy, t time.Time) error {
	// Seed the service's indicators.
	if !service.LastCandlestickTime().IsZero() &&
		t.After(service.LastCandlestickTime()) &&
		// Ensure that we're processing the complete tick, not a partial one.
		uint(t.Sub(service.LastCandlestickTime()).Minutes())%model.TickSizeMinutes == 0 {
		candles, err := b.backend.FindCandlesticks(service.LastCandlestickTime(), t,
			model.Product, int64(model.TickSizeMinutes))
		if err != nil {
			return errors.Wrapf(err, "error finding candlesticks")
		}
		for _, c := range candles {
			if err := service.SeedIndicators(c); err != nil {
				return errors.Wrapf(err, "error seeding strategy indicators")
			}
		}
	}

	if err := service.ProcessTick(model); err != nil {
		// Deactivate strategy if there isn't enough data.
		if errors.Cause(err) == ErrNotEnoughData {
			model.DeactivatedAt = t
			if uerr := b.backend.UpdateTradingStrategy(model); uerr != nil {
				return errors.Wrapf(uerr, "error updating trading strategy ticks")
			}

			logrus.Infof("deactivating %s strategy for a lack of data", b.product)

			msg := fmt.Sprintf(`ID: %d
Type: %s
Reason: not enough data`, model.ID, service.strategy)
			PostTradesSlackMessage("", slack.PostMessageParameters{
				AsUser: true,
				Attachments: []slack.Attachment{
					{Title: "Strategy Deactivated", Text: msg, Color: "#ff5c3f"},
				},
			})

			return nil
		}

		return errors.Wrapf(err, "error processing strategy tick")
	}

	model.LastTickAt = t
	model.NextTickAt = CandlestickBucket(t, int64(model.TickSizeMinutes)).
		Add(time.Duration(model.TickSizeMinutes) * time.Minute)

	if err := b.backend.UpdateTradingStrategy(model); err != nil {
		return errors.Wrapf(err, "error updating trading strategy ticks")
	}

	return nil
}
