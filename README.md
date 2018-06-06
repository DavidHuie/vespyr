# vespyr

Vespyr is a cryptocurrency trading bot for the GDAX exchange that uses
technical analysis strategies.

Vespyr continuously saves price information to a Postgres database. It
then periodically runs a model and executes trades accordingly.

## Installation

```bash
go install github.com/DavidHuie/vespyr/cmd/vespyr
```

## Usage

```text
$ vespyr -h

vespyr is a currency trading engine

Usage:
  vespyr [flags]
  vespyr [command]

Available Commands:
  backtest-ema-crossover backtest an EMA crossover strategy
  backtest-rsi           backtest the rsi strategy
  backtest-s1            backtest the s1 strategy
  bot                    run the automated trading bot
  create-ema             creates an ema trading strategy
  create-s1              creates an s1 trading strategy
  help                   Help about any command
  import                 import historical data
  migrate                migrate the database
  optimize-strategy      optimizes a genetic algorithm
  realtime-import        import data in realtime
  rollback               rollback the database

Flags:
  -c, --config-file string            an optional configuration file
      --gdax-api-key string           the GDAX API key
      --gdax-api-secret string        the GDAX API secret
      --gdax-passphrase string        the GDAX API passphrase
  -h, --help                          help for vespyr
      --postgres string               the postgres URI (default "postgres://vespyr:password@localhost:5432/vespyr?sslmode=disable")
      --rollbar-token string          the Rollbar API token
      --slack-data-channel string     the data Slack channel (default "#data-dev")
      --slack-token string            the Slack API token
      --slack-trades-channel string   the trades Slack channel (default "#trades-dev")
      --use-fake-exchange             use a fake exchange

Use "vespyr [command] --help" for more information about a command.
```

More docs coming soon!
