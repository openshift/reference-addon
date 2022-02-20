package addonsdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	statusReporterSingleton      *StatusReporter
	statusReporterSingletonMutex = &sync.Mutex{}
)

// a nice heartbeat-reporter implemented by the MT-SRE which our tenants can use
// if they don't like it, they can implement their own heartbeat reporter by creating a type which implements the `addonsdk.statusReporterClient` interface
type StatusReporter struct {
	// object provided by the client/tenants which implements the addonsdk.client interface
	addonInstanceInteractor client
	addonName               string
	addonTargetNamespace    string

	// the latest conditions which the heartbeat reporter would be reporting periodically
	latestConditions []metav1.Condition

	// to control the rate at which the heartbeat reporter would run
	interval time.Duration
	ticker   *time.Ticker

	// for effectively communicating the stop and update signals
	stopperCh chan bool
	updateCh  chan updateOptions

	// for concurrency-safely executing one instance of heartbeat reporter loop
	executeOnce sync.Once

	//for tracking if the heartbeat reporter is running or done running
	doneCh chan bool

	log logr.Logger
}

// ensure that the `StatusReporter` implements the `addonsdk.statusReporterClient` interface
var _ statusReporterClient = (*StatusReporter)(nil)

// InitializeStatusReporterSingleton sets up a singleton of the type `StatusReporter` (only if it doesn't exist yet) and returns it to the caller.
func InitializeStatusReporterSingleton(addonInstanceInteractor client, addonName string, addonTargetNamespace string, logger logr.Logger) *StatusReporter {
	if statusReporterSingleton == nil {
		statusReporterSingletonMutex.Lock()
		defer statusReporterSingletonMutex.Unlock()
		if statusReporterSingleton == nil {
			statusReporterSingleton = &StatusReporter{
				addonInstanceInteractor: addonInstanceInteractor,
				addonName:               addonName,
				addonTargetNamespace:    addonTargetNamespace,
				latestConditions: []metav1.Condition{
					{
						Type:    "addons.managed.openshift.io/Healthy",
						Status:  "False",
						Reason:  "NoHeartbeatReported",
						Message: fmt.Sprintf("Addon '%s' hasn't reported any heartbeat yet", addonName),
					},
				},
				stopperCh: make(chan bool),
				updateCh:  make(chan updateOptions),
				doneCh:    make(chan bool),
				log:       logger,
			}
			// because the heartbeat reporter still hasn't been started
			defer close(statusReporterSingleton.doneCh)
		}
	}

	return statusReporterSingleton
}

func GetStatusReporterSingleton() (*StatusReporter, error) {
	if statusReporterSingleton == nil {
		return nil, fmt.Errorf("heartbeat-reporter not found to be initialised. Initialize it by calling `InitializeStatusReporterSingleton(...)`")
	}
	return statusReporterSingleton, nil
}

func (sr StatusReporter) LatestConditions() []metav1.Condition {
	return sr.latestConditions
}

func (sr *StatusReporter) Start(ctx context.Context) error {
	// ensures to tie only one heartbeat-reporter loop at a time to a StatusReporter object
	var startErr error
	sr.executeOnce.Do(func() {
		sr.doneCh = make(chan bool)
		defer close(sr.doneCh)

		currentAddonInstance := &addonsv1alpha1.AddonInstance{}
		if err := sr.addonInstanceInteractor.GetAddonInstance(context.TODO(), types.NamespacedName{Name: "addon-instance", Namespace: sr.addonTargetNamespace}, currentAddonInstance); err != nil {
			startErr = fmt.Errorf("error occurred while fetching the current heartbeat update period interval: %w", err)
			return
		}
		sr.interval = currentAddonInstance.Spec.HeartbeatUpdatePeriod.Duration
		defer sr.ticker.Stop()
		sr.ticker = time.NewTicker(sr.interval)

		for {
			select {
			case update := <-sr.updateCh:
				// update the interval if the newInterval in the `update` is provided and is not equal to the existing interval
				// synchronize the timer with this new interval
				if update.addonInstance != nil {
					if update.addonInstance.Spec.HeartbeatUpdatePeriod.Duration != sr.interval {
						sr.interval = update.addonInstance.Spec.HeartbeatUpdatePeriod.Duration
						sr.ticker.Reset(sr.interval)
					}
				}

				if update.conditions != nil {
					sr.latestConditions = *update.conditions
				}
			case <-sr.ticker.C:
				if err := sr.updateAddonInstanceStatus(ctx, sr.latestConditions); err != nil {
					sr.log.Error(err, "failed to report the regular heartbeat")
				}
			case <-ctx.Done():
			case <-sr.stopperCh:
			}
		}
	})
	return startErr
}

func (sr *StatusReporter) Stop(ctx context.Context) error {
	select {
	case <-sr.doneCh: // will non-blockingly receive whenever doneCh would be closed
		sr.log.Info("status reporter is already stopped")
		return nil
	case sr.stopperCh <- true:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to stop the status reporter: %w", ctx.Err())
	}
}

func (sr *StatusReporter) SetConditions(ctx context.Context, conditions []metav1.Condition) error {
	// immediately register a new heartbeat upon receive one from the client/tenant side
	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := sr.addonInstanceInteractor.GetAddonInstance(ctx, types.NamespacedName{Name: "addon-instance", Namespace: sr.addonTargetNamespace}, addonInstance); err != nil {
		return fmt.Errorf("failed to get the AddonInstance: %w", err)
	}

	// making a deep-copy for current Conditions for rolling back in case of failures
	previousConditions := addonInstance.Status.Conditions

	newConditions := addonInstance.Status.Conditions
	for _, condition := range conditions {
		meta.SetStatusCondition(&newConditions, condition)
	}
	addonInstance.Status.Conditions = newConditions
	addonInstance.Status.ObservedGeneration = addonInstance.Generation
	addonInstance.Status.LastHeartbeatTime = metav1.Now()

	if err := sr.addonInstanceInteractor.UpdateAddonInstanceStatus(ctx, addonInstance); err != nil {
		return fmt.Errorf("failed to update AddonInstance Status: %w", err)
	}

	// rollbackAddonInstanceStatusUpdate() will be called when the `conditions` would fail to be sent on the `sr.updateCh` channel
	rollbackAddonInstanceStatusUpdate := func() error {
		addonInstance.Status.Conditions = previousConditions
		addonInstance.Status.LastHeartbeatTime = metav1.Now()

		if err := sr.addonInstanceInteractor.UpdateAddonInstanceStatus(ctx, addonInstance); err != nil {
			return fmt.Errorf("failed to update AddonInstance Status: %w", err)
		}
		return nil
	}

	select {
	case <-sr.doneCh:
		sr.log.Info("StatusReporter found to be stopped")
		return nil
	case sr.updateCh <- updateOptions{conditions: &conditions}: // near-instantly received by the StatusReporter loop
		return nil
	case <-ctx.Done():
		sr.log.Error(ctx.Err(), "failed to send the heartbeat's Conditions to the StatusReporter")
		sr.log.Info("rolling back the AddonInstance Status update...")
		// not 100% full-proof rollback because in theory, the rollback could still fail.
		// Yet in that case too, eventual consistency would still be preserved because our AddonInstance.Status.Conditions would get re-synchronized to `sr.latestConditions` in the next iteration of the StatusReporter loop.
		if err := rollbackAddonInstanceStatusUpdate(); err != nil {
			return fmt.Errorf("failed to rollback the AddonInstance Status update: %w", err)
		}
		return nil
	}
}

func (sr *StatusReporter) ReportAddonInstanceSpecChange(ctx context.Context, newAddonInstance addonsv1alpha1.AddonInstance) error {
	select {
	case <-sr.doneCh:
		return fmt.Errorf("can't report AddonInstance spec change on a stopped StatusReporter")
	case sr.updateCh <- updateOptions{addonInstance: &newAddonInstance}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to report AddonInstance spec change: %w", ctx.Err())
	}
}
