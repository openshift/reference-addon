package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
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

	// dummy code to indicate reconciliation of the reference-addon object
	if req.NamespacedName.Name != "reference-addon" {
		log.Info("doing nothing to a ReferenceAddon object not with the name reference-addon")
		return ctrl.Result{}, nil
	}

	log.Info("reconciling for a ReferenceAddon object with the name reference-addon")
	refado := &refapisv1alpha1.ReferenceAddon{}
	if err := r.Get(ctx, req.NamespacedName, refado); err != nil {
		// don't report anything new if the CR if found to be deleted/non-existent
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
	}
	conditionsToReport := []metav1.Condition{failureCondition}
	if refado.Spec.ReportSuccessfulStatus {
		conditionsToReport = []metav1.Condition{successfulCondition}
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
