package controllers

import (
	"context"
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var ErrEmptyOptionValue = errors.New("empty option value")

func ValidateOptionValue(val string) error {
	if val != "" {
		return nil
	}

	return ErrEmptyOptionValue
}

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}

func EnqueueObject(obj types.NamespacedName) source.Func {
	return func(_ context.Context, _ handler.EventHandler, q workqueue.RateLimitingInterface, _ ...predicate.Predicate) error {
		q.Add(reconcile.Request{
			NamespacedName: obj,
		})

		return nil
	}
}

func HasNamePrefix(pfx string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return strings.HasPrefix(obj.GetName(), pfx)
		},
	)
}

func HasName(name string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return obj.GetName() == name
		},
	)
}
