package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	refapisv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	"github.com/openshift/reference-addon/internal/utils"
)

type ReferenceAddonReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ReferenceAddonReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("addon", req.NamespacedName.String())

	successfulCondition := metav1.Condition{
		Type:    "addons.managed.openshift.io/Healthy",
		Status:  "True",
		Reason:  "AllComponentsUp",
		Message: "Everything under reference-addon is working perfectly fine",
	}
	failureCondition := metav1.Condition{
		Type:    "addons.managed.openshift.io/Healthy",
		Status:  "False",
		Reason:  "ImproperNaming",
		Message: "The addon resources are improperly named",
	}

	// if the ReferenceAddon object getting reconciled has the name "reference-addon", only then report a successful heartbeat
	if strings.HasPrefix(req.NamespacedName.Name, "reference-addon") || strings.HasPrefix(req.NamespacedName.Name, "redhat-") {
		if err := utils.SetAndSendHeartbeat(r, successfulCondition); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set and sent heartbeat: %w", err)
		}
	} else {
		if err := utils.SetAndSendHeartbeat(r, failureCondition); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set and sent heartbeat: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r ReferenceAddonReconciler) GetAddonName() string {
	return "reference-addon"
}

func (r ReferenceAddonReconciler) GetAddonTargetNamespace() string {
	// assuming ADDON_TARGET_NAMESPACE would be populated via downwards API in the deployment spec of the reference addon
	return os.Getenv("ADDON_TARGET_NAMESPACE")
}

func (r ReferenceAddonReconciler) GetClient() client.Client {
	return r.Client
}

// the following 'HandleAddonInstanceConfigurationChanges' method can be absolutely anything depending how reference-addon would want to deal with AddonInstance's configuration change
func (r *ReferenceAddonReconciler) HandleAddonInstanceConfigurationChanges(newAddonInstanceSpec addonsv1alpha1.AddonInstanceSpec) error {
	fmt.Println("Handling AddonInstance's configuration changes, whooossh!!!")
	return nil
}

func (r *ReferenceAddonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&refapisv1alpha1.ReferenceAddon{}).
		Complete(r)
}
