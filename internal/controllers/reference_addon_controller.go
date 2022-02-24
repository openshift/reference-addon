package controllers

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	refapisv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	"github.com/openshift/reference-addon/internal/addonsdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReferenceAddonReconciler struct {
	client.Client
	StatusReporter *addonsdk.StatusReporter
	Log            logr.Logger
	Scheme         *runtime.Scheme
}

func (r *ReferenceAddonReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("referenceaddon", req.NamespacedName.String())

	successfulCondition := metav1.Condition{
		Type:    addonsdk.AddonHealthyConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  "AllComponentsUp",
		Message: "Everything under reference-addon is working perfectly fine",
	}
	failureCondition := metav1.Condition{
		Type:    addonsdk.AddonHealthyConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  "ImproperNaming",
		Message: "The addon resources are improperly named",
	}

	var conditionsToReport []metav1.Condition
	// dummy code to indicate reconciliation of the reference-addon object
	if strings.HasPrefix(req.NamespacedName.Name, "redhat-") {
		log.Info("reconciling for a reference addon object prefixed by redhat- ")
		conditionsToReport = []metav1.Condition{successfulCondition}
	} else if strings.HasPrefix(req.NamespacedName.Name, "reference-addon") {
		log.Info("reconciling for a reference addon object named reference-addon")
		conditionsToReport = []metav1.Condition{successfulCondition}
	} else {
		log.Info("reconciling for a reference addon object not prefixed by redhat- or named reference-addon")
		conditionsToReport = []metav1.Condition{failureCondition}
	}

	if err := r.StatusReporter.SetConditions(ctx, conditionsToReport); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ReferenceAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&refapisv1alpha1.ReferenceAddon{}).
		Complete(r)
}
