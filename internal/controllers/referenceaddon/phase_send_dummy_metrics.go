package referenceaddon

import (
	"context"
)

func NewPhaseSendDummyMetrics(sampler ResponseSampler, opts ...PhaseSendDummyMetricsOption) *PhaseSendDummyMetrics {
	var cfg PhaseSendDummyMetricsConfig

	cfg.Option(opts...)

	return &PhaseSendDummyMetrics{
		cfg: cfg,

		sampler: sampler,
	}
}

type PhaseSendDummyMetrics struct {
	cfg PhaseSendDummyMetricsConfig

	sampler ResponseSampler
}

func (p *PhaseSendDummyMetrics) Execute(ctx context.Context, req PhaseRequest) PhaseResult {
	p.sampler.RequestSampleResponseData(p.cfg.SampleURLs...)

	return PhaseResultSuccess()
}

type PhaseSendDummyMetricsConfig struct {
	SampleURLs []string
}

func (c *PhaseSendDummyMetricsConfig) Option(opts ...PhaseSendDummyMetricsOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseSendDummyMetrics(c)
	}
}

type PhaseSendDummyMetricsOption interface {
	ConfigurePhaseSendDummyMetrics(*PhaseSendDummyMetricsConfig)
}

type ResponseSampler interface {
	RequestSampleResponseData(urls ...string)
}
