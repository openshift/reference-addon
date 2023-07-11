package referenceaddon

import (
	"context"

	refv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Phase interface {
	Execute(ctx context.Context, req PhaseRequest) PhaseResult
}

type PhaseRequest struct {
	Addon  refv1alpha1.ReferenceAddon
	Params PhaseRequestParameters
}

func NewPhaseRequestParameters(opts ...PhaseRequestParametersOption) PhaseRequestParameters {
	var cfg PhaseRequestParametersConfig

	cfg.Option(opts...)

	return PhaseRequestParameters{
		applyNetworkPolicies: cfg.ApplyNetworkPolicies,
		enableSmokeTest:      cfg.EnableSmokeTest,
		size:                 cfg.Size,
	}
}

type PhaseRequestParameters struct {
	applyNetworkPolicies *bool
	enableSmokeTest      *bool
	size                 *string
}

func (p *PhaseRequestParameters) GetSize() (string, bool) {
	if p.size == nil {
		return "", false
	}

	return *p.size, true
}

func (p *PhaseRequestParameters) GetEnableSmokeTest() (bool, bool) {
	if p.enableSmokeTest == nil {
		return false, false
	}

	return *p.enableSmokeTest, true
}

func (p *PhaseRequestParameters) GetApplyNetworkPolicies() (bool, bool) {
	if p.applyNetworkPolicies == nil {
		return false, false
	}

	return *p.applyNetworkPolicies, true
}

type PhaseRequestParametersConfig struct {
	ApplyNetworkPolicies *bool
	EnableSmokeTest      *bool
	Size                 *string
}

func (c *PhaseRequestParametersConfig) Option(opts ...PhaseRequestParametersOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseRequestParameters(c)
	}
}

type WithApplyNetworkPolicies struct{ Value *bool }

func (w WithApplyNetworkPolicies) ConfigurePhaseRequestParameters(c *PhaseRequestParametersConfig) {
	c.ApplyNetworkPolicies = w.Value
}

type WithEnableSmokeTest struct{ Value *bool }

func (w WithEnableSmokeTest) ConfigurePhaseRequestParameters(c *PhaseRequestParametersConfig) {
	c.EnableSmokeTest = w.Value
}

type WithSize struct{ Value *string }

func (w WithSize) ConfigurePhaseRequestParameters(c *PhaseRequestParametersConfig) {
	c.Size = w.Value
}

type PhaseRequestParametersOption interface {
	ConfigurePhaseRequestParameters(*PhaseRequestParametersConfig)
}

func PhaseResultSuccess(opts ...PhaseResultOption) PhaseResult {
	var cfg PhaseResultConfig

	cfg.Option(opts...)

	return PhaseResult{
		cfg:    cfg,
		status: PhaseStatusSuccess,
	}
}

func PhaseResultBlocking(opts ...PhaseResultOption) PhaseResult {
	var cfg PhaseResultConfig

	cfg.Option(opts...)

	return PhaseResult{
		cfg:    cfg,
		status: PhaseStatusBlocking,
	}
}

func PhaseResultFailure(msg string, opts ...PhaseResultOption) PhaseResult {
	var cfg PhaseResultConfig

	cfg.Option(opts...)

	return PhaseResult{
		cfg:        cfg,
		status:     PhaseStatusFailure,
		failureMsg: msg,
	}
}

func PhaseResultError(err error, opts ...PhaseResultOption) PhaseResult {
	var cfg PhaseResultConfig

	cfg.Option(opts...)

	return PhaseResult{
		cfg:    cfg,
		status: PhaseStatusError,
		err:    err,
	}
}

type PhaseResult struct {
	cfg        PhaseResultConfig
	err        error
	failureMsg string
	status     PhaseStatus
}

func (r PhaseResult) Status() PhaseStatus {
	return r.status
}

func (r PhaseResult) FailureMessage() string {
	return r.failureMsg
}

func (r PhaseResult) Error() error {
	return r.err
}

func (r PhaseResult) Conditions() []metav1.Condition {
	return r.cfg.Conditions
}

type PhaseStatus string

func (s PhaseStatus) String() string {
	return string(s)
}

const (
	PhaseStatusBlocking PhaseStatus = "blocking"
	PhaseStatusError    PhaseStatus = "error"
	PhaseStatusFailure  PhaseStatus = "failure"
	PhaseStatusSuccess  PhaseStatus = "success"
)

type PhaseResultConfig struct {
	Conditions []metav1.Condition
}

func (c *PhaseResultConfig) Option(opts ...PhaseResultOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseResult(c)
	}
}

type PhaseResultOption interface {
	ConfigurePhaseResult(c *PhaseResultConfig)
}

type WithConditions []metav1.Condition

func (w WithConditions) ConfigurePhaseResult(c *PhaseResultConfig) {
	c.Conditions = append(c.Conditions, w...)
}
