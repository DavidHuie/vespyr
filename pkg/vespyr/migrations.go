package vespyr

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/mattes/migrate/source"
	"github.com/sirupsen/logrus"
)

func migrations() {
	cm := new(CodeMigrations)

	cm.AddMigration(new(Migration).SetUp(`
BEGIN;
CREATE TABLE candlesticks (
  id serial,
  start_time timestamptz,
  end_time timestamptz,
  low double precision,
  high double precision,
  open double precision,
  close double precision,
  volume double precision,
  direction text,
  product text
);
COMMIT;
`).SetDown(`
BEGIN;
DROP TABLE candlesticks;
COMMIT;`))

	cm.AddMigration(new(Migration).SetUp(`
BEGIN;
TRUNCATE TABLE candlesticks;
CREATE UNIQUE INDEX candlesticks_start_time_idx ON candlesticks (start_time);
COMMIT;
`).SetDown(`
BEGIN;
DROP INDEX candlesticks_start_time_idx;
COMMIT;`))

	cm.AddMigration(new(Migration).SetUp(`
BEGIN;
CREATE TABLE trading_strategies (
  id serial PRIMARY KEY,
  created_at timestamptz NOT NULL,
  updated_at timestamptz,
  deactivated_at timestamptz,
  last_tick_at timestamptz,
  next_tick_at timestamptz,
  product text,
  history_ticks integer,
  state text,
  initial_budget double precision,
  budget double precision,
  budget_currency text,
  invested double precision,
  invested_currency text,
  tick_size_minutes integer,
  trading_strategy text,
  trading_strategy_data bytea
);
CREATE TABLE market_orders (
  id serial PRIMARY KEY,
  created_at timestamptz NOT NULL,
  updated_at timestamptz,
  trading_strategy_id integer REFERENCES trading_strategies,
  exchange_id text,
  product text,
  side text,
  cost double precision,
  cost_currency text,
  filled_size double precision,
  size_currency text,
  fees double precision,
  fees_currency text
);
DROP INDEX candlesticks_start_time_idx;
CREATE UNIQUE INDEX candlesticks_start_time_product_idx ON candlesticks (start_time, product);
ALTER TABLE candlesticks ADD COLUMN created_at timestamptz;
ALTER TABLE candlesticks ADD COLUMN updated_at timestamptz;
COMMIT;
`).SetDown(`
BEGIN;
DROP TABLE market_orders;
DROP TABLE trading_strategies;
DROP INDEX candlesticks_start_time_product_idx;
CREATE UNIQUE INDEX candlesticks_start_time_idx ON candlesticks (start_time);
ALTER TABLE candlesticks DROP COLUMN created_at;
ALTER TABLE candlesticks DROP COLUMN updated_at;
COMMIT;`))

	cm.AddMigration(new(Migration).SetUp(`
BEGIN;
ALTER TABLE candlesticks ADD PRIMARY KEY (id);
COMMIT;
`).SetDown(`
BEGIN;
ALTER TABLE candlesticks DROP CONSTRAINT candlesticks_pkey;
COMMIT;`))

	source.Register("code", cm)
}

// Migration defines a database migration.
type Migration struct {
	up   string
	down string
}

// SetUp sets the up migration.
func (m *Migration) SetUp(s string) *Migration {
	m.up = s
	return m
}

// Up returns the up migration.
func (m *Migration) Up() string {
	return m.up
}

// SetDown sets the down migration.
func (m *Migration) SetDown(s string) *Migration {
	m.down = s
	return m
}

// Down returns the down migration.
func (m *Migration) Down() string {
	return m.down
}

// CodeMigrations is a type that stores migrations as part of the
// code.
type CodeMigrations struct {
	migrations []*Migration
}

// AddMigration adds a migration to the set of migrations.
func (c *CodeMigrations) AddMigration(m *Migration) {
	c.migrations = append(c.migrations, m)
}

// Open opens the migration driver.
func (c *CodeMigrations) Open(url string) (source.Driver, error) {
	return c, nil
}

// Close closes the migration driver.
func (c *CodeMigrations) Close() error {
	return nil
}

// First returns the first migration.
func (c *CodeMigrations) First() (version uint, err error) {
	if len(c.migrations) == 0 {
		return 0, os.ErrNotExist
	}
	return 1, nil
}

// Prev returns the previous migration.
func (c *CodeMigrations) Prev(version uint) (prevVersion uint, err error) {
	v := version - 1
	if v >= uint(len(c.migrations))+1 || v < 1 {
		return 0, os.ErrNotExist
	}
	return v, nil
}

// Next returns the next migration.
func (c *CodeMigrations) Next(version uint) (nextVersion uint, err error) {
	v := version + 1
	if v >= uint(len(c.migrations))+1 || v < 1 {
		return 0, os.ErrNotExist
	}
	return v, nil
}

func identifier(t string, version uint) string {
	return fmt.Sprintf("code-migration-%s-%v", t, version)
}

// ReadUp returns the Up migration for the version.
func (c *CodeMigrations) ReadUp(version uint) (io.ReadCloser, string, error) {
	if version >= uint(len(c.migrations)+1) || version < 1 {
		return nil, identifier("up", version), os.ErrNotExist
	}
	migration := c.migrations[version-1]
	return ioutil.NopCloser(bytes.NewBufferString(migration.Up())), identifier("up", version), nil
}

// ReadDown returns the Down migration for the version.
func (c *CodeMigrations) ReadDown(version uint) (io.ReadCloser, string, error) {
	if version >= uint(len(c.migrations)+1) || version < 1 {
		return nil, identifier("down", version), os.ErrNotExist
	}
	migration := c.migrations[version-1]
	return ioutil.NopCloser(bytes.NewBufferString(migration.Down())), identifier("down", version), nil
}

type migrationLogger struct{}

func (m *migrationLogger) Printf(format string, v ...interface{}) {
	logrus.Printf(format, v...)
}

func (m *migrationLogger) Verbose() bool {
	return false
}

func init() {
	migrations()
}
