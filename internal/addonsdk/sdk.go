package addonsdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"go.uber.org/zap"
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

	// the latest condition which the heartbeat reporter would be reporting the periodically
	latestCondition metav1.Condition

	// to control the rate at which the heartbeat reporter would run
	currentInterval time.Duration
	ticker          *time.Ticker

	// for effectively communicating the stop and update signals
	stopperCh chan bool
	updateCh  chan updateOptions

	// for concurrency-safely executing one instance of heartbeat reporter loop
	executeOnce sync.Once

	log logr.Logger
}

// ensure that the `StatusReporter` implements the `addonsdk.statusReporterClient` interface
var _ statusReporterClient = (*StatusReporter)(nil)

// InitializeStatusReporterSingleton sets up a singleton of the type `StatusReporter` (only if it doesn't exist yet) and returns it to the caller.
func InitializeStatusReporterSingleton(addonInstanceInteractor client, addonName string, addonTargetNamespace string) (*StatusReporter, error) {

	zapLog, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize the logger: %+v", err))
	}

	if statusReporterSingleton == nil {
		statusReporterSingletonMutex.Lock()
		defer statusReporterSingletonMutex.Unlock()
		if statusReporterSingleton == nil {
			statusReporterSingleton = &StatusReporter{
				addonInstanceInteractor: addonInstanceInteractor,
				addonName:               addonName,
				addonTargetNamespace:    addonTargetNamespace,
				latestCondition: metav1.Condition{
					Type:    "addons.managed.openshift.io/Healthy",
					Status:  "False",
					Reason:  "NoHeartbeatReported",
					Message: fmt.Sprintf("Addon '%s' hasn't reported any heartbeat yet", addonName),
				},
				stopperCh: make(chan bool),
				updateCh:  make(chan updateOptions),
				log:       zapr.NewLogger(zapLog),
			}

			currentAddonInstance := &addonsv1alpha1.AddonInstance{}
			if err := statusReporterSingleton.addonInstanceInteractor.GetAddonInstance(context.TODO(), types.NamespacedName{Name: "addon-instance", Namespace: statusReporterSingleton.addonTargetNamespace}, currentAddonInstance); err != nil {
				return nil, fmt.Errorf("error occurred while fetching the current heartbeat update period interval")
			}
			statusReporterSingleton.currentInterval = currentAddonInstance.Spec.HeartbeatUpdatePeriod.Duration
		}
	}

	return statusReporterSingleton, nil
}

func GetStatusReporterSingleton() (*StatusReporter, error) {
	if statusReporterSingleton == nil {
		return nil, fmt.Errorf("heartbeat-reporter not found to be initialised. Initialize it by calling `InitializeStatusReporterSingleton(...)`")
	}
	return statusReporterSingleton, nil
}

func (sr StatusReporter) LatestCondition() metav1.Condition {
	return sr.latestCondition
}

func (sr *StatusReporter) Start(ctx context.Context) {
	// ensures to tie only one heartbeat-reporter loop at a time to a StatusReporter object
	sr.executeOnce.Do(func() {
		defer sr.ticker.Stop()
		sr.ticker = time.NewTicker(sr.currentInterval)

		for {
			select {
			case update := <-sr.updateCh:
				// update the interval if the newInterval in the `update` is provided and is not equal to the existing interval
				// synchronize the timer with this new interval
				if update.newAddonInstanceSpec != nil {
					if update.newAddonInstanceSpec.HeartbeatUpdatePeriod.Duration != sr.currentInterval {
						sr.currentInterval = update.newAddonInstanceSpec.HeartbeatUpdatePeriod.Duration
						sr.ticker.Reset(sr.currentInterval)
					}
				}

				if update.newLatestCondition != nil {
					// immediately register a new heartbeat upon receive one from the client/tenant side
					if err := sr.updateAddonInstanceStatus(ctx, *update.newLatestCondition); err != nil {
						sr.log.Error(err, "failed to update the addoninstance status")
						continue
					}
					sr.latestCondition = *update.newLatestCondition
				}
			case <-sr.ticker.C:
				if err := sr.updateAddonInstanceStatus(ctx, sr.latestCondition); err != nil {
					sr.log.Error(err, "failed to report the regular heartbeat")
				}
			case <-ctx.Done():
				return
			case <-sr.stopperCh:
				return
			}
		}
	})
}

func (sr *StatusReporter) Stop() error {
	select {
	case sr.stopperCh <- true:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("failed to stop the reporter: timed out waiting for the reporter to ack the stop signal")
	}
}

func (sr *StatusReporter) SendHeartbeat(ctx context.Context, condition metav1.Condition) error {
	select {
	case sr.updateCh <- updateOptions{newLatestCondition: &condition}: // near-instantly received by the heartbeat reporter loop
		return nil
	case <-ctx.Done():
		return fmt.Errorf("found the provided context to be exhausted")
	}
}

func (sr *StatusReporter) ReportAddonInstanceSpecChange(ctx context.Context, newAddonInstanceSpec *addonsv1alpha1.AddonInstanceSpec) error {
	select {
	case sr.updateCh <- updateOptions{newAddonInstanceSpec: newAddonInstanceSpec}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("found the provided context to be exhausted")
	}
}
