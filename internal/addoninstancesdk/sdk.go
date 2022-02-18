package addoninstancesdk

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
	// object provided by the client/tenants which implements the AddonInstanceInteractorClient interface
	AddonInstanceInteractor addonInstanceInteractorClient
	AddonName               string
	AddonTargetNamespace    string

	// to "concurrent-safely" track whether there's a heartbeat reporter running or not
	isRunning      bool
	isRunningMutex *sync.Mutex

	// the latest condition which the heartbeat reporter would be reporting the periodically
	latestCondition metav1.Condition

	// to control the rate at which the heartbeat reporter would run
	currentInterval time.Duration
	ticker          *time.Ticker

	// for effectively communicating the stop and update signals
	stopperCh chan bool
	updateCh  chan updateOptions
}

// ensure that the `AddonInstanceHeartbeatReporter` implements the `AddonInstanceStatusReporterClient` interface
var _ addonInstanceStatusReporterClient = (*AddonInstanceHeartbeatReporter)(nil)

// InitializeAddonInstanceHeartbeatReporterSingleton sets up a singleton of the type `AddonInstanceHeartbeatReporter` (only if it doesn't exist yet) and returns it to the caller.
func InitializeAddonInstanceHeartbeatReporterSingleton(addonInstanceInteractor addonInstanceInteractorClient, addonName string, addonTargetNamespace string) (*AddonInstanceHeartbeatReporter, error) {
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
				stopperCh: make(chan bool),
				updateCh:  make(chan updateOptions),
			}

			currentAddonInstance, err := addonInstanceHeartbeatReporterSingleton.AddonInstanceInteractor.GetAddonInstance(context.TODO(), "addon-instance", addonInstanceHeartbeatReporterSingleton.AddonTargetNamespace)
			if err != nil {
				return nil, fmt.Errorf("error occurred while fetching the current heartbeat update period interval")
			}
			addonInstanceHeartbeatReporterSingleton.currentInterval = currentAddonInstance.Spec.HeartbeatUpdatePeriod.Duration
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

func (adihrClient AddonInstanceHeartbeatReporter) LatestCondition() metav1.Condition {
	return adihrClient.latestCondition
}

func (adihrClient *AddonInstanceHeartbeatReporter) changeRunningState(desiredState bool) {
	adihrClient.isRunningMutex.Lock()
	adihrClient.isRunning = desiredState
	adihrClient.isRunningMutex.Unlock()
}

func (adihrClient *AddonInstanceHeartbeatReporter) Start(ctx context.Context) error {
	defer func() {
		adihrClient.changeRunningState(false)
	}()

	// not allow the client/tenant to start multiple heartbeat reporter concurrently and cause data races
	adihrClient.isRunningMutex.Lock()
	if adihrClient.isRunning {
		adihrClient.isRunningMutex.Unlock()
		return fmt.Errorf("already found the heartbeat reporter to be running")
	}
	adihrClient.isRunning = true
	adihrClient.isRunningMutex.Unlock()

	adihrClient.ticker = time.NewTicker(adihrClient.currentInterval)
	defer adihrClient.ticker.Stop()

	for {
		select {
		case update := <-adihrClient.updateCh:
			if update.newLatestCondition != nil {
				// immediately register a new heartbeat upon receive one from the client/tenant side
				if err := adihrClient.updateAddonInstanceStatus(ctx, *update.newLatestCondition); err != nil {
					return fmt.Errorf("failed to update the addoninstance status: %w", err)
				}
				adihrClient.latestCondition = *update.newLatestCondition
			}

			// update the interval if the newInterval in the `update` is provided and is not equal to the existing interval
			// synchronize the timer with this new interval
			if update.newAddonInstanceSpec != nil {
				if update.newAddonInstanceSpec.HeartbeatUpdatePeriod.Duration != adihrClient.currentInterval {
					adihrClient.currentInterval = update.newAddonInstanceSpec.HeartbeatUpdatePeriod.Duration
					adihrClient.ticker.Reset(adihrClient.currentInterval)
				}
			}
		case <-adihrClient.ticker.C:
			if err := adihrClient.updateAddonInstanceStatus(ctx, adihrClient.latestCondition); err != nil {
				return fmt.Errorf("failed to report the regular heartbeat: %w", err)
			}
		case <-ctx.Done():
			return fmt.Errorf("provided context exhausted")
		case <-adihrClient.stopperCh:
			return nil
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
	select {
	case adihrClient.updateCh <- updateOptions{newLatestCondition: &condition}: // near-instantly received by the heartbeat reporter loop
		return nil
	case <-ctx.Done():
		return fmt.Errorf("found the provided context to be exhausted")
	}
}

func (adihrClient *AddonInstanceHeartbeatReporter) ReportAddonInstanceSpecChange(ctx context.Context, newAddonInstanceSpec *addonsv1alpha1.AddonInstanceSpec) error {
	select {
	case adihrClient.updateCh <- updateOptions{newAddonInstanceSpec: newAddonInstanceSpec}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("found the provided context to be exhausted")
	}
}
