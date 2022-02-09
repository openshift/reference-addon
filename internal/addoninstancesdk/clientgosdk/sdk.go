package clientgosdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	addonInstanceHeartbeatReporterSingleton      *AddonInstanceHeartbeatReporter
	addonInstanceHeartbeatReporterSingletonMutex = &sync.Mutex{}
)

// a nice heartbeat-reporter implemented by the MT-SRE which our tenants can use
// if they don't like it, they can implement their own heartbeat reporter by creating a type which implements the `AddonInstanceStatusReporterClient` interface
type AddonInstanceHeartbeatReporter struct {
	AddonInstanceInteractor AddonInstanceInteractorClient
	AddonName               string
	AddonTargetNamespace    string

	isRunning      bool
	isRunningMutex *sync.Mutex

	latestCondition      metav1.Condition
	latestConditionMutex *sync.Mutex

	addonInstanceSpecChangeWatcherCh chan bool

	stopperCh chan bool
}

var _ AddonInstanceStatusReporterClient = (*AddonInstanceHeartbeatReporter)(nil)

func InitializeAddonInstanceHeartbeatReporterSingleton(addonInstanceInteractor AddonInstanceInteractorClient, addonName string, addonTargetNamespace string) (*AddonInstanceHeartbeatReporter, error) {
	if addonInstanceHeartbeatReporterSingleton == nil {
		addonInstanceHeartbeatReporterSingletonMutex.Lock()
		defer addonInstanceHeartbeatReporterSingletonMutex.Unlock()
		if addonInstanceHeartbeatReporterSingleton == nil {
			addonInstanceHeartbeatReporterSingleton = &AddonInstanceHeartbeatReporter{
				AddonInstanceInteractor: addonInstanceInteractor,
				AddonName:               addonName,
				AddonTargetNamespace:    addonTargetNamespace,
				isRunning:               false,
				isRunningMutex:          &sync.Mutex{},
				latestCondition: metav1.Condition{
					Type:    "addons.managed.openshift.io/Healthy",
					Status:  "False",
					Reason:  "NoHeartbeatReported",
					Message: fmt.Sprintf("Addon '%s' hasn't reported any heartbeat yet", addonName),
				},
				latestConditionMutex: &sync.Mutex{},
				stopperCh:            make(chan bool),
			}

			var err error
			if addonInstanceHeartbeatReporterSingleton.addonInstanceSpecChangeWatcherCh, err = addonInstanceInteractor.AddonInstanceSpecChangeWatcher(); err != nil {
				return nil, fmt.Errorf("failed to get the AddonInstance's spec-change watcher: %w", err)
			}
		}
	}
	return addonInstanceHeartbeatReporterSingleton, nil
}

func GetAddonInstanceHeartbeatReporterSingleton() (*AddonInstanceHeartbeatReporter, error) {
	if addonInstanceHeartbeatReporterSingleton == nil {
		return nil, fmt.Errorf("heartbeat-reporter not found to be initialised. Initialize it by calling `InitializeAddonInstanceHeartbeatReporterSingleton(...)`")
	}
	return addonInstanceHeartbeatReporterSingleton, nil
}

func (adihrClient AddonInstanceHeartbeatReporter) IsRunning() bool {
	return adihrClient.isRunning
}

// only for informational purpose
func (adihrClient AddonInstanceHeartbeatReporter) LatestCondition() metav1.Condition {
	return adihrClient.latestCondition
}

func (adihrClient *AddonInstanceHeartbeatReporter) ChangeRunningState(desiredState bool) {
	adihrClient.isRunningMutex.Lock()
	adihrClient.isRunning = desiredState
	adihrClient.isRunningMutex.Unlock()
}

func (adihrClient *AddonInstanceHeartbeatReporter) Start(ctx context.Context) error {
	defer func() {
		adihrClient.ChangeRunningState(false)
	}()

	adihrClient.isRunningMutex.Lock()
	if adihrClient.isRunning {
		adihrClient.isRunningMutex.Unlock()
		return fmt.Errorf("already found the heartbeat reporter to be running")
	}
	adihrClient.isRunning = true
	adihrClient.isRunningMutex.Unlock()

	for {
		addonInstance, err := adihrClient.AddonInstanceInteractor.GetAddonInstance(ctx, "addon-instance", adihrClient.AddonTargetNamespace)
		if err != nil {
			return fmt.Errorf("failed to get the AddonInstance resource in the namespace %s: %w", adihrClient.AddonTargetNamespace, err)
		}

		adihrClient.latestConditionMutex.Lock()
		if err := adihrClient.AddonInstanceInteractor.UpdateAddonInstanceStatus(ctx, addonInstance, adihrClient.latestCondition); err != nil {
			adihrClient.latestConditionMutex.Unlock()
			return fmt.Errorf("failed to set AddonInstance's Condition: %w", err)
		}
		adihrClient.latestConditionMutex.Unlock()

		select {
		case <-adihrClient.stopperCh:
			return nil
		case <-ctx.Done():
			return nil
		case <-time.After(addonInstance.Spec.HeartbeatUpdatePeriod.Duration):
			continue
		case <-adihrClient.addonInstanceSpecChangeWatcherCh:
			continue
		}
	}
}

func (adihrClient *AddonInstanceHeartbeatReporter) Stop() error {
	select {
	case adihrClient.stopperCh <- true:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("failed to stop the reporter: timed out waiting for the reporter to ack the stop signal")
	}
}

func (adihrClient *AddonInstanceHeartbeatReporter) SendHeartbeat(ctx context.Context, condition metav1.Condition) error {
	adihrClient.latestConditionMutex.Lock()
	defer adihrClient.latestConditionMutex.Unlock()

	addonInstance, err := adihrClient.AddonInstanceInteractor.GetAddonInstance(ctx, "addon-instance", adihrClient.AddonTargetNamespace)
	if err != nil {
		return fmt.Errorf("failed to get the AddonInstance: %w", err)
	}
	if err := adihrClient.upsertCondition(ctx, addonInstance, condition); err != nil {
		return fmt.Errorf("failed to upsert condition to the AddonInstance: %w", err)
	}
	adihrClient.latestCondition = condition
	return nil
}

func (adihrClient *AddonInstanceHeartbeatReporter) upsertCondition(ctx context.Context, addonInstance *addonsv1alpha1.AddonInstance, condition metav1.Condition) error {
	if err := adihrClient.AddonInstanceInteractor.UpdateAddonInstanceStatus(ctx, addonInstance, condition); err != nil {
		return fmt.Errorf("failed to set AddonInstance's Condition: %w", err)
	}
	return nil
}
