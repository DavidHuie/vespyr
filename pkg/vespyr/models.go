package vespyr

import (
	"time"

	"github.com/go-pg/pg/orm"
	"github.com/montanaflynn/stats"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// CandlestickModel is a candlestick's representation in a database.
type CandlestickModel struct {
	tableName struct{} `sql:"candlesticks"`
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	StartTime time.Time
	EndTime   time.Time
	Low       float64
	High      float64
	Open      float64
	Close     float64
	Volume    float64
	Direction CandlestickDirection
	Product   Product
}

// MeanPrice returns the mean price for the candlestick period.
func (m *CandlestickModel) MeanPrice() (float64, error) {
	price, err := stats.Mean([]float64{
		m.Open,
		m.High,
		m.Low,
		m.Close,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "error calculating candlestick mean")
	}
	return price, nil
}

func (m *CandlestickModel) BeforeInsert(db orm.DB) error {
	m.CreatedAt = time.Now()
	return nil
}

func (m *CandlestickModel) BeforeUpdate(db orm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// MarketOrderModel contains metadata about market orders that were
// placed.
type MarketOrderModel struct {
	tableName         struct{} `sql:"market_orders"`
	ID                int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	TradingStrategyID int64
	ExchangeID        string
	Product           Product
	Side              string
	Cost              float64
	CostCurrency      string
	FilledSize        float64
	SizeCurrency      string
	Fees              float64
	FeesCurrency      string
}

func (m *MarketOrderModel) BeforeInsert(db orm.DB) error {
	m.CreatedAt = time.Now()
	return nil
}

func (m *MarketOrderModel) BeforeUpdate(db orm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// TradingStrategyModel contains metadata for a trading strategy.
type TradingStrategyModel struct {
	tableName           struct{} `sql:"trading_strategies"`
	ID                  int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeactivatedAt       time.Time
	LastTickAt          time.Time
	NextTickAt          time.Time
	Product             Product
	HistoryTicks        uint
	State               string
	InitialBudget       float64
	Budget              float64
	BudgetCurrency      string
	Invested            float64
	InvestedCurrency    string
	TickSizeMinutes     uint
	TradingStrategy     string
	TradingStrategyData []byte
}

func (m *TradingStrategyModel) BeforeInsert(db orm.DB) error {
	m.CreatedAt = time.Now()
	return nil
}

func (m *TradingStrategyModel) BeforeUpdate(db orm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// Copy returns a copy of the current model.
func (t *TradingStrategyModel) Copy() *TradingStrategyModel {
	return &TradingStrategyModel{
		ID:                  t.ID,
		Product:             t.Product,
		HistoryTicks:        t.HistoryTicks,
		State:               t.State,
		InitialBudget:       t.InitialBudget,
		Budget:              t.Budget,
		BudgetCurrency:      t.BudgetCurrency,
		Invested:            t.Invested,
		InvestedCurrency:    t.InvestedCurrency,
		TickSizeMinutes:     t.TickSizeMinutes,
		TradingStrategy:     t.TradingStrategy,
		TradingStrategyData: t.TradingStrategyData,
	}
}

// SetStrategy sets the underlying trading strategy.
func (t *TradingStrategyModel) SetStrategy(s StrategyInterface) error {
	switch s.(type) {
	case *EMACrossoverStrategy:
		t.TradingStrategy = TradingStrategyEMACrossover
	case *RSIStrategy:
		t.TradingStrategy = TradingStrategyRSI
	case *S1Strategy:
		t.TradingStrategy = TradingStrategyS1
	default:
		return errors.Errorf("error: unknown trading strategy")
	}

	s.SetTradingStrategy(t)

	b, err := yaml.Marshal(s)
	if err != nil {
		return errors.Wrapf(err, "error YAML marshalling trading strategy: %s", t.TradingStrategy)
	}
	t.TradingStrategyData = b

	return nil
}

// Strategy returns the underlying trading strategy.
func (t *TradingStrategyModel) Strategy() (StrategyInterface, error) {
	switch t.TradingStrategy {
	case TradingStrategyEMACrossover:
		var strategy EMACrossoverStrategy
		if err := yaml.Unmarshal(t.TradingStrategyData, &strategy); err != nil {
			return nil, errors.Wrapf(err, "error YAML unmarshaling trading strategy data")
		}
		strategy.SetTradingStrategy(t)
		return &strategy, nil
	case TradingStrategyRSI:
		var strategy RSIStrategy
		if err := yaml.Unmarshal(t.TradingStrategyData, &strategy); err != nil {
			return nil, errors.Wrapf(err, "error YAML unmarshaling trading strategy data")
		}
		strategy.SetTradingStrategy(t)
		return &strategy, nil
	case TradingStrategyS1:
		var strategy S1Strategy
		if err := yaml.Unmarshal(t.TradingStrategyData, &strategy); err != nil {
			return nil, errors.Wrapf(err, "error YAML unmarshaling trading strategy data")
		}
		strategy.SetTradingStrategy(t)
		return &strategy, nil
	default:
		return nil, errors.Errorf("error: unknown trading strategy: %s", t.TradingStrategy)
	}
}
