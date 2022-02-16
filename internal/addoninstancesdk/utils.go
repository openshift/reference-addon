package addoninstancesdk

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type updateOptions struct {
	newInterval        *time.Duration
	newLatestCondition *metav1.Condition
}

func (adihrClient *AddonInstanceHeartbeatReporter) updateAddonInstanceStatus(ctx context.Context, condition metav1.Condition) error {
	addonInstance, err := adihrClient.AddonInstanceInteractor.GetAddonInstance(ctx, "addon-instance", adihrClient.AddonTargetNamespace)
	if err != nil {
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
