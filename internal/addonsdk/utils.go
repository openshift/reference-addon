package addonsdk

import (
	"context"
	"fmt"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type updateOptions struct {
	// for capturing new changes to the addonInstance which the tenants would be watching
	addonInstance *addonsv1alpha1.AddonInstance

	// for capturing new heartbeats/conditions to be reported
	conditions *[]metav1.Condition
}

func (sr *StatusReporter) updateAddonInstanceStatus(ctx context.Context, conditions []metav1.Condition) error {
	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := sr.addonInstanceInteractor.GetAddonInstance(ctx, types.NamespacedName{Name: "addon-instance", Namespace: sr.addonTargetNamespace}, addonInstance); err != nil {
		return fmt.Errorf("failed to get the AddonInstance: %w", err)
	}

	// TODO(doc): we should point clients to use the helper methods from apimachinery, when interacting with object conditions:
	// Ref: https://github.com/kubernetes/apimachinery/blob/57f2a0733447cfd41294477d833cce6580faaca3/pkg/api/meta/conditions.go#L30
	addonInstance.Status.Conditions = conditions
	addonInstance.Status.ObservedGeneration = addonInstance.Generation
	addonInstance.Status.LastHeartbeatTime = metav1.Now()

	if err := sr.addonInstanceInteractor.UpdateAddonInstanceStatus(ctx, addonInstance); err != nil {
		return fmt.Errorf("failed to update AddonInstance Status: %w", err)
	}
	return nil
}
