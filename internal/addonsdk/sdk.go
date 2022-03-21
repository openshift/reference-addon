package addonsdk

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	statusReporterSingleton *StatusReporter
	initializeSingletonOnce sync.Once
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
	updateCh chan updateOptions

	//for tracking if the heartbeat reporter runner
	runnersCh      chan bool
	runnersChMutex sync.RWMutex

	log logr.Logger
}

// SetupStatusReporter sets up a singleton of the type `statusReporter` (only if it doesn't exist yet) and returns it to the caller.
func SetupStatusReporter(addonInstanceInteractor client, addonName string, addonTargetNamespace string, logger logr.Logger) *StatusReporter {
	initializeSingletonOnce.Do(func() {
		statusReporterSingleton = &StatusReporter{
			addonInstanceInteractor: addonInstanceInteractor,
			addonName:               addonName,
			addonTargetNamespace:    addonTargetNamespace,
			latestConditions: []metav1.Condition{
				{
					Type:    AddonHealthyConditionType,
					Status:  metav1.ConditionUnknown,
					Reason:  "NoHealthReported",
					Message: fmt.Sprintf("Addon %q hasn't reported health yet", addonName),
				},
			},
			updateCh:  make(chan updateOptions),
			runnersCh: make(chan bool, 1),
			log:       logger,
		}
	})
	return statusReporterSingleton
}

func (sr *StatusReporter) Start(ctx context.Context) error {
	sr.runnersChMutex.Lock()
	select {
	// sr.runnersCh is a buffered channel of capacity 1, which ensures that only one "non-blocking send" (case sr.runnersCh <- true) can happen to it successfully when it's empty, else it will be blocked till either ctx.Done() happens or the `sr.runnersCh` gets freed ("released by the previous caller").
	// this ensures concurrency-safely executing only one status reporter loop at a time
	case sr.runnersCh <- true:
		// defer freeing up the sr.runnersCh for the next caller "in queue" to successfully start the status reporter loop
		sr.runnersChMutex.Unlock()
		defer func() {
			sr.runnersChMutex.Lock()
			defer sr.runnersChMutex.Unlock()
			<-sr.runnersCh
		}()

		currentAddonInstance := &addonsv1alpha1.AddonInstance{}
		if err := sr.addonInstanceInteractor.GetAddonInstance(ctx, types.NamespacedName{Name: "addon-instance", Namespace: sr.addonTargetNamespace}, currentAddonInstance); err != nil {
			sr.log.Error(err, "failed to fetch the current heartbeat update period interval")
			sr.log.Info("initialising the status reporter with the default interval", "period", defaultHeartbeatUpdatePeriod)
			sr.interval = defaultHeartbeatUpdatePeriod
		} else {
			sr.interval = currentAddonInstance.Spec.HeartbeatUpdatePeriod.Duration
		}

		defer sr.ticker.Stop()
		sr.ticker = time.NewTicker(sr.interval)

		for {
			select {
			case update := <-sr.updateCh:
				// update the interval if the newInterval in the `update` is provided and is not equal to the existing interval
				// synchronize the timer with this new interval
				if !reflect.DeepEqual(update.addonInstance, addonsv1alpha1.AddonInstance{}) {
					if update.addonInstance.Spec.HeartbeatUpdatePeriod.Duration != sr.interval {
						sr.interval = update.addonInstance.Spec.HeartbeatUpdatePeriod.Duration
						sr.ticker.Reset(sr.interval)
					}
				}

				if update.conditions != nil {
					sr.latestConditions = update.conditions
				}
			case <-sr.ticker.C:
				if err := sr.updateAddonInstanceStatus(ctx, sr.latestConditions); err != nil {
					sr.log.Error(err, "failed to report the regular heartbeat")
				}
			case <-ctx.Done():
				sr.log.Info("received signal to stop the status reporter", "reason", ctx.Err())
				return nil
			}
		}
	case <-ctx.Done():
		sr.runnersChMutex.Unlock()
		return ctx.Err()
	}
}

func (sr *StatusReporter) SetConditions(ctx context.Context, conditions []metav1.Condition) error {
	sr.runnersChMutex.RLock()
	defer sr.runnersChMutex.RUnlock()
	if len(sr.runnersCh) == 0 {
		return fmt.Errorf("StatusReporter found to be stopped: can't set conditions to a stopped StatusReporter")
	}
	select {
	case sr.updateCh <- updateOptions{conditions: conditions}: // near-instantly received by the StatusReporter loop
		if err := sr.updateAddonInstanceStatus(ctx, conditions); err != nil {
			sr.log.Error(err, "failed to set Conditions[] on AddonInstance, please wait for the next iteration of the status reporter loop to register these Conditions[]")
		}
		return nil
	case <-ctx.Done():
		sr.log.Error(ctx.Err(), "failed to send the heartbeat's Conditions to the StatusReporter")
		return nil
	}
}

func (sr *StatusReporter) ReportAddonInstanceSpecChange(ctx context.Context, newAddonInstance addonsv1alpha1.AddonInstance) error {
	sr.runnersChMutex.RLock()
	defer sr.runnersChMutex.RUnlock()
	if len(sr.runnersCh) == 0 {
		return fmt.Errorf("can't report AddonInstance spec change on a stopped StatusReporter")
	}
	select {
	case sr.updateCh <- updateOptions{addonInstance: newAddonInstance}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
