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

type RequestParameters struct {
	Size string
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
