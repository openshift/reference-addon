package phase

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

type Phase interface {
	Execute(ctx context.Context, req Request) Result
}

type Request struct {
	Object types.NamespacedName
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

func Success() Result {
	return Result{
		status: StatusSuccess,
	}
}

func Blocking() Result {
	return Result{
		status:   StatusSuccess,
		blocking: true,
	}
}

func Failure(msg string) Result {
	return Result{
		status:     StatusFailure,
		failureMsg: msg,
	}
}

func Error(err error) Result {
	return Result{
		status: StatusError,
		err:    err,
	}
}

type Result struct {
	err        error
	failureMsg string
	status     Status
	blocking   bool
}

func (r Result) IsSuccess() bool {
	return r.status == StatusSuccess
}

func (r Result) IsBlocking() bool {
	return r.blocking
}

func (r Result) FailureMessage() string {
	return r.failureMsg
}

func (r Result) Error() error {
	return r.err
}

type Status string

const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusError   = "error"
)
