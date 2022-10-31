package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	refv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var ErrEmptyOptionValue = errors.New("empty option value")

func validateOptionValue(val string) error {
	if val != "" {
		return nil
	}

	return ErrEmptyOptionValue
}

func generateIngressPolicyName(prefix string) string {
	return fmt.Sprintf("%s-ingress", prefix)
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func newAvailableCondition(reason refv1alpha1.ReferenceAddonAvailableReason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:               refv1alpha1.ReferenceAddonConditionAvailable.String(),
		Status:             reason.Status(),
		Reason:             reason.String(),
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}
}

func enqueueObject(obj types.NamespacedName) source.Func {
	return func(_ context.Context, _ handler.EventHandler, q workqueue.RateLimitingInterface, _ ...predicate.Predicate) error {
		q.Add(reconcile.Request{
			NamespacedName: obj,
		})

		return nil
	}
}

func hasNamePrefix(pfx string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return strings.HasPrefix(obj.GetName(), pfx)
		},
	)
}

func hasName(name string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return obj.GetName() == name
		},
	)
}
