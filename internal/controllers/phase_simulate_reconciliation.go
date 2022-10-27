package controllers

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/openshift/reference-addon/internal/controllers/phase"
)

func NewPhaseSimulateReconciliation(opts ...PhaseSimulateReconciliationOption) *PhaseSimulateReconciliation {
	var cfg PhaseSimulateReconciliationConfig

	cfg.Option(opts...)
	cfg.Default()

	return &PhaseSimulateReconciliation{
		cfg: cfg,
	}
}

type PhaseSimulateReconciliation struct {
	cfg PhaseSimulateReconciliationConfig
}

func (p *PhaseSimulateReconciliation) Execute(ctx context.Context, req phase.Request) phase.Result {
	log := p.cfg.Log.WithValues("addon", req.Object.String())

	applyNetworkPolicies, _ := req.Params.GetApplyNetworkPolicies()
	size, _ := req.Params.GetSize()

	log.Info(
		"reconciling with addon parameters",
		"ApplyNetworkPolicies", applyNetworkPolicies,
		"Size", size,
	)

	// dummy code to indicate reconciliation of the reference-addon object
	if strings.HasPrefix(req.Object.Name, "redhat-") {
		log.Info("reconciling for a reference addon object prefixed by redhat- ")
	} else if strings.HasPrefix(req.Object.Name, "reference-addon") {
		log.Info("reconciling for a reference addon object named reference-addon")
	} else {
		log.Info("reconciling for a reference addon object not prefixed by redhat- or named reference-addon")
	}

	return phase.Success()
}

type PhaseSimulateReconciliationConfig struct {
	Log logr.Logger
}

func (c *PhaseSimulateReconciliationConfig) Option(opts ...PhaseSimulateReconciliationOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseSimulateReconciliation(c)
	}
}

func (c *PhaseSimulateReconciliationConfig) Default() {
	if c.Log.GetSink() == nil {
		c.Log = logr.Discard()
	}
}

type PhaseSimulateReconciliationOption interface {
	ConfigurePhaseSimulateReconciliation(*PhaseSimulateReconciliationConfig)
}
