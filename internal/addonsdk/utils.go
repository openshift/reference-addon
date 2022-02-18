package addonsdk

import (
	"context"
	"fmt"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type updateOptions struct {
	// for capturing new changes to the addonInstance which the tenants would be watching
	// using this instead of `newInterval *time.Duration` so that:
	// - the client/tenant has to setup a watcher/informer for AddonInstance and just have to straight away input the entire AddonInstance object whenever an update on it is observed
	// - in the future, if we (SDK developers) expect other inputs associated with other fields of AddonInstance Spec, we won't have to nudge the clients/tenants to update their watcher/informer to start a new bunch of arguments when calling `addonsdk.ReportUpdate(...)`.
	//      They can still continue to blindly input the newAddonInstanceSpec whenever a change is observed by their informer/watcher and it would be upto our SDK to take decisions on the basis of whether `heartbeatUpdatePeriod` field changed or some other field changed.
	newAddonInstanceSpec *addonsv1alpha1.AddonInstanceSpec

	// for capturing new heartbeats/conditions to be reported
	newLatestCondition *metav1.Condition
}

func (adihrClient *AddonInstanceHeartbeatReporter) updateAddonInstanceStatus(ctx context.Context, condition metav1.Condition) error {
	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := adihrClient.AddonInstanceInteractor.GetAddonInstance(ctx, types.NamespacedName{Name: "addon-instance", Namespace: adihrClient.AddonTargetNamespace}, addonInstance); err != nil {
		return fmt.Errorf("failed to get the AddonInstance: %w", err)
	}
	currentTime := metav1.Now()
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = currentTime
	}
	// TODO: confirm that it's not worth tracking the ObservedGeneration at per-condition basis
	meta.SetStatusCondition(&(*addonInstance).Status.Conditions, condition)
	addonInstance.Status.ObservedGeneration = (*addonInstance).Generation
	addonInstance.Status.LastHeartbeatTime = metav1.Now()

	if err := adihrClient.AddonInstanceInteractor.UpdateAddonInstanceStatus(ctx, addonInstance); err != nil {
		return fmt.Errorf("failed to set AddonInstance's Condition: %w", err)
	}
	return nil
}
