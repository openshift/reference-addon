package phase

import (
	"context"

	refv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Phase interface {
	Execute(ctx context.Context, req Request) Result
}

type Request struct {
	Addon  refv1alpha1.ReferenceAddon
	Params RequestParameters
}

func NewRequestParameters(opts ...RequestParametersOption) RequestParameters {
	var cfg RequestParametersConfig

	cfg.Option(opts...)

	return RequestParameters{
		applyNetworkPolicies: cfg.ApplyNetworkPolicies,
		size:                 cfg.Size,
	}
}

type RequestParameters struct {
	applyNetworkPolicies *bool
	size                 *string
}

func (p *RequestParameters) GetSize() (string, bool) {
	if p.size == nil {
		return "", false
	}

	return *p.size, true
}

func (p *RequestParameters) GetApplyNetworkPolicies() (bool, bool) {
	if p.applyNetworkPolicies == nil {
		return false, false
	}

	return *p.applyNetworkPolicies, true
}

type RequestParametersConfig struct {
	ApplyNetworkPolicies *bool
	Size                 *string
}

func (c *RequestParametersConfig) Option(opts ...RequestParametersOption) {
	for _, opt := range opts {
		opt.ConfigureRequestParameters(c)
	}
}

type WithApplyNetworkPolicies struct{ Value *bool }

func (w WithApplyNetworkPolicies) ConfigureRequestParameters(c *RequestParametersConfig) {
	c.ApplyNetworkPolicies = w.Value
}

type WithSize struct{ Value *string }

func (w WithSize) ConfigureRequestParameters(c *RequestParametersConfig) {
	c.Size = w.Value
}

type RequestParametersOption interface {
	ConfigureRequestParameters(*RequestParametersConfig)
}

func Success(opts ...ResultOption) Result {
	var cfg ResultConfig

	cfg.Option(opts...)

	return Result{
		cfg:    cfg,
		status: StatusSuccess,
	}
}

func Blocking(opts ...ResultOption) Result {
	var cfg ResultConfig

	cfg.Option(opts...)

	return Result{
		cfg:    cfg,
		status: StatusBlocking,
	}
}

func Failure(msg string, opts ...ResultOption) Result {
	var cfg ResultConfig

	cfg.Option(opts...)

	return Result{
		cfg:        cfg,
		status:     StatusFailure,
		failureMsg: msg,
	}
}

func Error(err error, opts ...ResultOption) Result {
	var cfg ResultConfig

	cfg.Option(opts...)

	return Result{
		cfg:    cfg,
		status: StatusError,
		err:    err,
	}
}

type Result struct {
	cfg        ResultConfig
	err        error
	failureMsg string
	status     Status
}

func (r Result) Status() Status {
	return r.status
}

func (r Result) FailureMessage() string {
	return r.failureMsg
}

func (r Result) Error() error {
	return r.err
}

func (r Result) Conditions() []metav1.Condition {
	return r.cfg.Conditions
}

type Status string

func (s Status) String() string {
	return string(s)
}

const (
	StatusBlocking Status = "blocking"
	StatusError    Status = "error"
	StatusFailure  Status = "failure"
	StatusSuccess  Status = "success"
)

type ResultConfig struct {
	Conditions []metav1.Condition
}

func (c *ResultConfig) Option(opts ...ResultOption) {
	for _, opt := range opts {
		opt.ConfigureResult(c)
	}
}

type ResultOption interface {
	ConfigureResult(c *ResultConfig)
}

type WithConditions []metav1.Condition

func (w WithConditions) ConfigureResult(c *ResultConfig) {
	c.Conditions = append(c.Conditions, w...)
}
