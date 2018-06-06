package vespyr

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Backend is the interface to the database.
type Backend interface {
	// Candlesticks
	UpsertCandlestick(*CandlestickModel) error
	FindCandlesticks(time.Time, time.Time, Product, int64) ([]*CandlestickModel, error)
	FindMostRecentCandlestick(Product) (*CandlestickModel, error)
	FindCandlestickByID(int64) (*CandlestickModel, error)

	// Market orders
	CreateMarketOrder(*MarketOrderModel) error
	FindMarketOrderByID(int64) (*MarketOrderModel, error)

	// Trading strategies
	FindTradingStrategyByID(int64) (*TradingStrategyModel, error)
	CreateTradingStrategy(*TradingStrategyModel) error
	UpdateTradingStrategy(*TradingStrategyModel) error
	FindActiveTradingStrategies(Product) ([]*TradingStrategyModel, error)
}

// DBConn contains the supported backend operations.
type DBConn struct {
	conn *pg.DB
}

// NewDBConn returns a new DBConn.
func NewDBConn(conn *pg.DB) *DBConn {
	return &DBConn{
		conn: conn,
	}
}

// UpsertCandlestick upserts a candlestick.
func (d *DBConn) UpsertCandlestick(c *CandlestickModel) error {
	_, err := d.conn.Model(c).
		OnConflict("(start_time, product) DO UPDATE").
		Set("low = ?low, high = ?high, open = ?open, close = ?close, volume = ?volume, direction = ?direction, product = ?product, created_at = ?created_at, updated_at = ?updated_at").
		Insert()
	return errors.Wrapf(err, "error upserting candlestick")
}

func (d *DBConn) FindCandlestickByID(id int64) (*CandlestickModel, error) {
	m := &CandlestickModel{ID: id}
	if err := d.conn.Select(m); err != nil {
		return nil, errors.Wrapf(err, "error finding candlestick")
	}
	return m, nil
}

func (d *DBConn) FindMostRecentCandlestick(p Product) (*CandlestickModel, error) {
	var c CandlestickModel
	if err := d.conn.Model(&c).Order("start_time DESC").Where("product = ?", p).Limit(1).Select(); err != nil {
		return nil, errors.Wrapf(err, "error finding most recent candlestick")
	}
	return &c, nil
}

// FindCandlesticks locates candlesticks within a range and reprojects
// them.
func (d *DBConn) FindCandlesticks(startTime, endTime time.Time,
	product Product, tickSizeMinutes int64) ([]*CandlestickModel, error) {
	var candles []*CandlestickModel
	if err := d.conn.Model(&candles).
		Where("start_time >= ? AND end_time <= ? AND product = ?", startTime, endTime, string(product)).
		Order("start_time ASC").
		Select(); err != nil {
		return nil, errors.Wrapf(err, "error finding candlesticks")
	}

	logrus.Debugf("found %d candlesticks", len(candles))

	projections, err := ReprojectCandlesticks(candles, product, tickSizeMinutes)
	if err != nil {
		return nil, errors.Wrapf(err, "error projecting candlesticks")
	}

	logrus.Debugf("projected to %d candlesticks", len(projections))

	return projections, nil
}

func (d *DBConn) FindMarketOrderByID(id int64) (*MarketOrderModel, error) {
	m := &MarketOrderModel{ID: id}
	if err := d.conn.Select(m); err != nil {
		return nil, errors.Wrapf(err, "error finding market order")
	}
	return m, nil
}

func (d *DBConn) CreateMarketOrder(m *MarketOrderModel) error {
	_, err := d.conn.Model(m).Insert()
	return errors.Wrapf(err, "error inserting market order")
}

func (d *DBConn) FindTradingStrategyByID(id int64) (*TradingStrategyModel, error) {
	ts := &TradingStrategyModel{ID: id}
	if err := d.conn.Select(ts); err != nil {
		return nil, errors.Wrapf(err, "error finding trading strategy")
	}
	return ts, nil
}

func (d *DBConn) CreateTradingStrategy(m *TradingStrategyModel) error {
	_, err := d.conn.Model(m).Insert()
	return errors.Wrapf(err, "error inserting trading strategy")
}

func (d *DBConn) UpdateTradingStrategy(m *TradingStrategyModel) error {
	_, err := d.conn.Model(m).Update()
	return errors.Wrapf(err, "error updating trading strategy")
}

func (d *DBConn) FindActiveTradingStrategies(p Product) ([]*TradingStrategyModel, error) {
	var strategies []*TradingStrategyModel
	if err := d.conn.Model(&strategies).Where("product = ? AND deactivated_at IS NULL", p).Select(); err != nil {
		return nil, errors.Wrapf(err, "error finding active strategies")
	}
	return strategies, nil
}
