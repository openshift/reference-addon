package referenceaddon

import (
	"context"

	"github.com/go-logr/logr"
)

func NewPhaseSmokeTestRun(opts ...PhaseSmokeTestRunOption) *PhaseSmokeTestRun {
	var cfg PhaseSmokeTestRunConfig

	cfg.Option(opts...)
	cfg.Default()

	return &PhaseSmokeTestRun{
		cfg: cfg,
	}
}

type PhaseSmokeTestRun struct {
	cfg PhaseSmokeTestRunConfig
}

func (p *PhaseSmokeTestRun) Execute(_ context.Context, req PhaseRequest) PhaseResult {
	enableSmokeTest, ok := req.Params.GetEnableSmokeTest()
	if !ok {
		p.cfg.Log.V(1).Info("'EnableSmokeTest' parameter not set")

		return PhaseResultSuccess()
	}

	if enableSmokeTest {
		p.cfg.SmokeTester.Enable()

		p.cfg.Log.Info("enabling smoke test")
	} else {
		p.cfg.SmokeTester.Disable()

		p.cfg.Log.Info("disabling smoke test")
	}

	return PhaseResultSuccess()
}

type PhaseSmokeTestRunConfig struct {
	Log logr.Logger

	SmokeTester SmokeTester
}

func (c *PhaseSmokeTestRunConfig) Option(opts ...PhaseSmokeTestRunOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseSmokeTestRun(c)
	}
}

func (c *PhaseSmokeTestRunConfig) Default() {
	if c.Log.GetSink() == nil {
		c.Log = logr.Discard()
	}
}

type PhaseSmokeTestRunOption interface {
	ConfigurePhaseSmokeTestRun(*PhaseSmokeTestRunConfig)
}

type SmokeTester interface {
	Enable()
	Disable()
}
