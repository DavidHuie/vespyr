package vespyr

import (
	"fmt"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// StrategyStateTryingToBuy is the state for when the strategy
	// is trying to find the right price point in order to buy.
	StrategyStateTryingToBuy = "trying-to-buy"
	// StrategyStateTryingToSell is the state for when the
	// strategy is trying to find the right time to sell.
	StrategyStateTryingToSell = "trying-to-sell"

	// IndicatorEMA refers to the exponential moving average
	// indicator.
	IndicatorEMA = "ema"
	// IndicatorDEMA refers to the EMA difference indicator.
	IndicatorDEMA = "dema"
	// IndicatorRSI refers to the RSI indicator.
	IndicatorRSI = "rsi"
	// IndicatorMACD refers to the MACD indicator.
	IndicatorMACD = "macd"
	// IndicatorMACDWithSignal refers to the MACD with signal
	// indicator.
	IndicatorMACDWithSignal = "macd-with-signal"

	// TradingStrategyEMACrossover is a trading strategy that buys
	// and sells using EMA crossovers.
	TradingStrategyEMACrossover = "ema-crossover"
	// TradingStrategyRSI is a trading strategy that buys and
	// sells based on RSI values.
	TradingStrategyRSI = "rsi"
	// TradingStrategyS1 is the first proprietary Vespyr trading
	// strategy.
	TradingStrategyS1 = "s1"

	// strategyCurrencyPrecision this defines how many decimal
	// places should be used in currency sizes when placing
	// trades.
	tradeCurrencyPrecision = 8

	// minimumBudgetProportion the lowest we're allowed to let our
	// budget drop down to.
	minimumBudgetProportion = .7
)

var (
	// ErrNotEnoughData is returned when there isn't enough data.
	ErrNotEnoughData = errors.New("error: not enough data")
)

// Indicator is an interface to a type that can generate an
// IndicatorValue for a candlestick.
type Indicator interface {
	AddCandlestick(c *CandlestickModel) error
	Value() (*IndicatorValue, error)
	Name() string
}

// IndicatorValue describes the value of an indicator at a specific
// point in time.
type IndicatorValue struct {
	Time          time.Time
	Value         float64
	IndicatorName string
}

// IndicatorSet describes a collection of indicators for a specific
// point in time.
type IndicatorSet struct {
	Time   time.Time
	Values []*IndicatorValue
}

// StrategyInterface describes the exposed by a trading strategy.
type StrategyInterface interface {
	Indicators() []Indicator
	Buy(history []*IndicatorSet, current int) (bool, error)
	Sell(history []*IndicatorSet, current int) (bool, error)
	String() string
	SetTradingStrategy(t *TradingStrategyModel)
}

// TradingStrategy represents a general trading strategy where the
// underlying strategy can indicate either buying or selling.
type TradingStrategy struct {
	backend             Backend
	exchange            Exchange
	strategy            StrategyInterface
	indicators          map[string]Indicator
	indicatorSets       []*IndicatorSet
	indicatorNames      []string
	currentTick         int
	lastCandlestickTime time.Time
	clock               clockwork.Clock
	orderStrategy       OrderStrategy
}

// NewTradingStrategy instantiates a new trading strategy.
func NewTradingStrategy(backend Backend,
	exchange Exchange, strategy StrategyInterface,
	clock clockwork.Clock) *TradingStrategy {
	s := &TradingStrategy{
		backend:     backend,
		exchange:    exchange,
		strategy:    strategy,
		indicators:  make(map[string]Indicator),
		currentTick: -1,
		clock:       clock,
	}
	for _, i := range s.strategy.Indicators() {
		s.indicators[i.Name()] = i
		s.indicatorNames = append(s.indicatorNames, i.Name())
	}
	s.orderStrategy = NewMarketOrderStrategy(exchange, backend)
	return s
}

// SeedIndicators seeds the indicators with the current candle.
func (t *TradingStrategy) SeedIndicators(c *CandlestickModel) error {
	var set IndicatorSet
	for _, name := range t.indicatorNames {
		ind := t.indicators[name]
		t.lastCandlestickTime = c.EndTime
		set.Time = c.StartTime
		if err := ind.AddCandlestick(c); err != nil {
			return errors.Wrapf(err, "error seeding indicator: %s", name)
		}
		value, err := ind.Value()
		if err != nil && errors.Cause(err) != ErrNotEnoughData {
			return errors.Wrapf(err, "error getting indicator value: %s", name)
		}
		set.Values = append(set.Values, value)
	}

	// TODO: watch the memory size here.
	t.indicatorSets = append(t.indicatorSets, &set)
	t.nextTick()

	return nil
}

// LastCandlestickTime returns the ending time of the last candlestick
// that was processed.
func (t *TradingStrategy) LastCandlestickTime() time.Time {
	return t.lastCandlestickTime
}

func (t *TradingStrategy) setIndicatorSets(is []*IndicatorSet) {
	t.indicatorSets = is
}

func (t *TradingStrategy) nextTick() {
	t.currentTick++
}

// ProcessTick processes a single trading strategy model tick.
func (t *TradingStrategy) ProcessTick(m *TradingStrategyModel) error {
	switch m.State {
	case StrategyStateTryingToBuy:
		if err := t.TryBuy(m); err != nil {
			return errors.Wrapf(err, "error trying to buy")
		}
		return nil
	case StrategyStateTryingToSell:
		if err := t.TrySell(m); err != nil {
			return errors.Wrapf(err, "error trying to sell")
		}
		return nil
	default:
		return errors.Errorf("error: unknown strategy state: %s", m.State)
	}
}

// ValidateIndicatorSets ensures that the data in the present
// indicator sets is good enough to process.
func ValidateIndicatorSets(historyTicks, currentTick int, indicatorSets []*IndicatorSet) error {
	if len(indicatorSets) == 0 {
		return ErrNotEnoughData
	}
	if currentTick > len(indicatorSets) {
		return ErrNotEnoughData
	}

	// Validate that the latest indicator set has values.
	lastSet := indicatorSets[len(indicatorSets)-1]
	if lastSet == nil {
		logrus.Debugf("missing last indicator set")
		return ErrNotEnoughData
	}
	if len(lastSet.Values) == 0 {
		logrus.Debugf("missing indicator values for indicator set %s", lastSet.Time)
		return ErrNotEnoughData
	}
	for _, v := range lastSet.Values {
		if v == nil {
			logrus.Debugf("missing indicator value for indicator set %s", lastSet.Time)
			return ErrNotEnoughData
		}
	}

	return nil
}

// TryBuy tries to buy a currency if the underlying strategy indicates
// so.
func (t *TradingStrategy) TryBuy(m *TradingStrategyModel) error {
	if err := ValidateIndicatorSets(int(m.HistoryTicks), t.currentTick, t.indicatorSets); err != nil {
		return errors.Wrapf(err, "error validating indicator sets")
	}

	buy, err := t.strategy.Buy(t.indicatorSets, t.currentTick)
	if err != nil {
		return errors.Wrapf(err, "error using strategy")
	}

	if buy {
		response, err := t.orderStrategy.PerformOrder(&PerformOrderArgs{
			Product:         m.Product,
			Side:            OrderBuy,
			Cost:            m.Budget,
			TradingStrategy: m,
		})
		if err != nil {
			return errors.Wrapf(err, "error performing buy order")
		}

		initialBudget := m.Budget
		m.Invested = TruncateFloat(response.FilledSize, tradeCurrencyPrecision)
		m.Budget = 0
		m.State = StrategyStateTryingToSell

		if err := t.backend.UpdateTradingStrategy(m); err != nil {
			return errors.Wrapf(err, "error updating trading strategy model in database")
		}

		slackMsg := fmt.Sprintf(`Type: %s
Product: %s
Strategy ID: %d
Strategy type: %s
Order strategy: %s
Tick size (minutes): %d
Size (%s): %f
Cost (%s): %f
Fees (%s): %f
Implied price (%s): %f`, OrderBuy, m.Product, m.ID, t.strategy, t.orderStrategy,
			m.TickSizeMinutes, response.FilledSizeCurrency,
			response.FilledSize, m.BudgetCurrency, initialBudget,
			response.FeesCurrency, response.Fees, m.BudgetCurrency, initialBudget/response.FilledSize)
		PostTradesSlackMessage("", slack.PostMessageParameters{
			AsUser: true,
			Attachments: []slack.Attachment{
				{Title: "Market Order", Text: slackMsg, Color: "#2afc43"},
			},
		})
	} else {
		msg := fmt.Sprintf("skipped buy with strategy %d: %s", m.ID, t.strategy)
		logrus.Debug(msg)
	}

	return nil
}

// TrySell tries to sell a currency if the underlying strategy
// indicates so.
func (t *TradingStrategy) TrySell(m *TradingStrategyModel) error {
	if err := ValidateIndicatorSets(int(m.HistoryTicks), t.currentTick, t.indicatorSets); err != nil {
		return errors.Wrapf(err, "error validating indicator sets")
	}

	sell, err := t.strategy.Sell(t.indicatorSets, t.currentTick)
	if err != nil {
		return errors.Wrapf(err, "error using strategy")
	}

	if sell {
		response, err := t.orderStrategy.PerformOrder(&PerformOrderArgs{
			Product:         m.Product,
			Side:            OrderSell,
			Cost:            m.Invested,
			TradingStrategy: m,
		})
		if err != nil {
			return errors.Wrapf(err, "error performing sell order")
		}

		initialInvested := m.Invested
		m.Invested = 0
		m.Budget = TruncateFloat(response.FilledSize, tradeCurrencyPrecision)
		m.State = StrategyStateTryingToBuy

		if err := t.backend.UpdateTradingStrategy(m); err != nil {
			return errors.Wrapf(err, "error updating ema crossover model in database")
		}

		slackMsg := fmt.Sprintf(`Type: %s
Product: %s
Strategy ID: %d
Strategy type: %s
Order strategy: %s
Tick size (minutes): %d
Size (%s): %f
Cost (%s): %f
Fees (%s): %f
Implied price (%s): %f
Current profit %%: %f`, OrderSell, m.Product, m.ID, t.strategy, t.orderStrategy, m.TickSizeMinutes,
			response.FilledSizeCurrency, response.FilledSize,
			m.InvestedCurrency, initialInvested,
			response.FeesCurrency, response.Fees, m.BudgetCurrency, response.FilledSize/initialInvested,
			100*(response.FilledSize-m.InitialBudget)/m.InitialBudget)
		PostTradesSlackMessage("", slack.PostMessageParameters{
			AsUser: true,
			Attachments: []slack.Attachment{
				{Title: "Market Order", Text: slackMsg, Color: "#4286f4"},
			},
		})

		prop := m.Budget / m.InitialBudget
		if prop < minimumBudgetProportion {
			logrus.Infof("deactivating strategy %d for falling below minimum budget proportion %f: %f", m.ID, prop, m.Budget)

			msg := fmt.Sprintf(`ID: %d
Type: %s
Reason: budget %f %s fell below minimum proportion %f`, m.ID, t.strategy,
				m.Budget, m.BudgetCurrency, minimumBudgetProportion)
			PostTradesSlackMessage("", slack.PostMessageParameters{
				AsUser: true,
				Attachments: []slack.Attachment{
					{Title: "Strategy Deactivated", Text: msg, Color: "#ff5c3f"},
				},
			})

			m.DeactivatedAt = t.clock.Now()
			if err := t.backend.UpdateTradingStrategy(m); err != nil {
				return errors.Wrapf(err, "error updating strategy after deactivating")
			}
		}
	} else {
		msg := fmt.Sprintf("skipped sell with strategy %d: %s", m.ID, t.strategy)
		logrus.Debug(msg)
	}

	return nil
}
