package vespyr

import (
	"math/rand"
	"time"

	"github.com/MaxHalford/gago"
	"github.com/sirupsen/logrus"
)

// StrategyGenome defines the methods that should be implemented in order for
// an strategy to be optimized genetically.
type StrategyGenome interface {
	StrategyInterface
	Rand(rng *rand.Rand)
	Clone() StrategyGenome
	Mutate(rng *rand.Rand)
	Crossover(m StrategyGenome, r *rand.Rand) (StrategyGenome, StrategyGenome)
}

// BacktesterGenomeFactory generates strategy genomes to be tested by
// gago.
type BacktesterGenomeFactory struct {
	startTime time.Time
	endTime   time.Time
	model     *TradingStrategyModel
	backend   Backend
}

// NewBacktesterGenomeFactory creates a new NewBacktesterGenomeFactory.
func NewBacktesterGenomeFactory(startTime, endTime time.Time, model *TradingStrategyModel,
	backend Backend) (*BacktesterGenomeFactory, error) {
	cache := NewCacheingBackend(backend)
	return &BacktesterGenomeFactory{
		startTime: startTime,
		endTime:   endTime,
		model:     model,
		backend:   cache,
	}, nil
}

// Generate creates a new Genome for backtesting.
func (b *BacktesterGenomeFactory) Generate(rng *rand.Rand) gago.Genome {
	strategy, err := b.model.Strategy()
	if err != nil {
		logrus.Errorf("error fetching strategy from model: %s", err)
		return nil
	}

	genome := strategy.(StrategyGenome)
	genome.Rand(rng)

	return &BacktesterGenome{
		startTime: b.startTime,
		endTime:   b.endTime,
		backend:   b.backend,
		model:     b.model,
		strategy:  genome,
	}
}

// BacktesterGenome wraps a StrategyGenome. BacktesterGenome evaluates
// a StrategyGenome in a backtest when it's evaluated.
type BacktesterGenome struct {
	startTime time.Time
	endTime   time.Time
	backend   Backend
	logger    logrus.FieldLogger
	model     *TradingStrategyModel
	strategy  StrategyGenome
	evaluated bool
}

// Evaluate runs a backtest and returns the score.
func (b *BacktesterGenome) Evaluate() float64 {
	logrus.Debugf("evaluating strategy: %#v", b.strategy)

	model := b.model.Copy()
	if err := model.SetStrategy(b.strategy); err != nil {
		logrus.Errorf("error setting model strategy: %s", err)
		return 0
	}

	logrus.Debugf("evaluating model: %#v", model)

	backtester, err := NewBacktester(b.startTime, b.endTime,
		model, b.backend, rand.NewSource(time.Now().Unix()))
	if err != nil {
		logrus.Errorf("error creating backtester: %s", err)
		return 0
	}
	if err := backtester.Backtest(); err != nil {
		logrus.Errorf("error running backtest: %s", err)
		return 0
	}

	results := backtester.Results()
	if results.FinalBudget == 0 || results.FinalCurrencyPrice == 0 {
		return 0
	}

	return -results.SharpeRatio()
}

// Mutate mutates the underlying strategy.
func (b *BacktesterGenome) Mutate(rng *rand.Rand) {
	b.strategy.Mutate(rng)
}

// Crossover crosses over the underlying strategy with the passed in
// genomes.
func (b *BacktesterGenome) Crossover(genome gago.Genome, rng *rand.Rand) (gago.Genome, gago.Genome) {
	cross := genome.(*BacktesterGenome)
	c1, c2 := b.strategy.Crossover(cross.strategy, rng)

	c1Genome := &BacktesterGenome{
		startTime: b.startTime,
		endTime:   b.endTime,
		backend:   b.backend,
		model:     b.model,
		strategy:  c1,
	}

	c2Genome := &BacktesterGenome{
		startTime: b.startTime,
		endTime:   b.endTime,
		backend:   b.backend,
		model:     b.model,
		strategy:  c2,
	}

	return c1Genome, c2Genome
}

// Clone clones the BacktesterGenome.
func (b *BacktesterGenome) Clone() gago.Genome {
	return &BacktesterGenome{
		startTime: b.startTime,
		endTime:   b.endTime,
		backend:   b.backend,
		model:     b.model,
		strategy:  b.strategy.Clone(),
	}

}
