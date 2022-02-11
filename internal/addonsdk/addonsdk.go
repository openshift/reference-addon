package addonsdk

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type kubeClient interface {
	Get(ctx context.Context, key types.NamespacedName, addonInstance *addonsv1alpha1.AddonInstance) error
	UpdateStatus(ctx context.Context, addonInstance *addonsv1alpha1.AddonInstance) error
}

type StatusReporter struct {
	client kubeClient
	config StatusReporterConfig

	// doneCh will be closed when the main worker is shut down
	doneCh chan struct{}

	ticker           *time.Ticker
	tickerInterval   time.Duration
	latestConditions []metav1.Condition
	updateCh         chan updateEvent
}

var (
	_ StatusReporterOption = (WithAddonName)("")
	_ StatusReporterOption = (WithAddonNamespace)("")
)

type StatusReporterOption interface {
	ApplyToStatusReporter(c *StatusReporterConfig)
}

type WithAddonName string

func (n WithAddonName) ApplyToStatusReporter(c *StatusReporterConfig) {
	c.addonName = string(n)
}

type WithAddonNamespace string

func (n WithAddonNamespace) ApplyToStatusReporter(c *StatusReporterConfig) {
	c.addonNamespace = string(n)
}

type StatusReporterConfig struct {
	logger                    logr.Logger
	addonName, addonNamespace string
}

func (c *StatusReporterConfig) Default() {
	if c.logger == nil {
		c.logger = logr.Discard()
	}

	// TODO:
	// Can we make sure this does not collide?
	// Do we want to inject these via the Addon Operator and the Subscription?
	if len(c.addonName) == 0 {
		c.addonName = os.Getenv("ADDON_NAME")
	}
	if len(c.addonNamespace) == 0 {
		c.addonNamespace = os.Getenv("ADDON_NAMESPACE")
	}
}

type updateEvent struct {
	interval   time.Duration
	conditions []metav1.Condition
}

func test() {
	NewStatusReporter(nil, WithAddonNamespace("xxx"), WithAddonName("xxx"))
}

func NewStatusReporter(client kubeClient, opts ...StatusReporterOption) *StatusReporter {
	r := &StatusReporter{
		doneCh:   make(chan struct{}),
		updateCh: make(chan updateEvent),
	}
	for _, opt := range opts {
		opt.ApplyToStatusReporter(&r.config)
	}
	r.config.Default()
	return r
}

// Needs to be setup with an external watcher for the AddonInstance API to
// inform this reporter about changes to the AddonInstance.Spec.
func (r *StatusReporter) HandleAddonInstanceUpdate(addonInstance *addonsv1alpha1.AddonInstance) error {
	select {
	case r.updateCh <- updateEvent{interval: addonInstance.Spec.HeartbeatUpdatePeriod.Duration}:
	case <-r.doneCh:
		// we expect this method to be called by a reconciler/watcher via a queue
		// there is no need to requeue something when we are shutting down
		// so we don't return an error here.
		return nil
	}
	return nil
}

// SetConditions will override conditions of an AddonInstance to report new status.
func (r *StatusReporter) SetConditions(ctx context.Context, conditions []metav1.Condition) error {
	if err := updateAddonInstance(ctx, r.client, types.NamespacedName{
		Name:      r.config.addonName,
		Namespace: r.config.addonNamespace,
	}, conditions); err != nil {
		return err
	}

	select {
	case <-r.doneCh:
		// still send an update to the api server,
		// but no need to update the worker because it's already done.
	case r.updateCh <- updateEvent{conditions: conditions}:
		// case <-ctx.Done():
		// 	// canceled
		// 	return ctx.Err()
	}

	return nil
}

// Implementing controller-runtime Runnable interface
func (r *StatusReporter) Start(ctx context.Context) error {
	// close the update channel last,
	// so concurrent senders will already read from the closed done
	defer close(r.updateCh)

	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := client.Get(ctx, key, addonInstance); err != nil {
		return fmt.Errorf("getting AddonInstance to initialize heartbeat: %w", err)
	}
	period := addonInstance.Spec.HeartbeatUpdatePeriod.Duration

	r.tickerInterval = period
	r.ticker = time.NewTicker(period)
	defer r.ticker.Stop()

	defer close(r.doneCh)

	for {
		select {
		case update := <-r.updateCh:
			// Condition updates
			if update.conditions != nil {
				r.latestConditions = update.conditions
				// reset ticker because we have just sent an update,
				// so we can reset the clock.
				r.ticker.Reset(r.tickerInterval)
			}

			// Interval updates
			if update.interval == 0 ||
				update.interval == r.tickerInterval {
				continue
			}

			// TODO: log interval update
			r.tickerInterval = update.interval
			r.ticker.Reset(update.interval)

		case <-r.ticker.C:
			// TODO: log debug ticker triggered
			if err := updateAddonInstance(ctx, r.client, types.NamespacedName{
				Name:      r.config.addonName,
				Namespace: r.config.addonNamespace,
			}, r.latestConditions); err != nil {
				// only log error don't return
				// return err
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func updateAddonInstance(
	ctx context.Context,
	client kubeClient,
	key types.NamespacedName,
	conditions []metav1.Condition,
) error {
	addonInstance := &addonsv1alpha1.AddonInstance{}
	if err := client.Get(ctx, key, addonInstance); err != nil {
		return fmt.Errorf("getting AddonInstance prior to update: %w", err)
	}

	addonInstance.Status.Conditions = conditions
	addonInstance.Status.LastHeartbeatTime = metav1.Now()
	addonInstance.Status.ObservedGeneration = addonInstance.Generation

	if err := client.UpdateStatus(ctx, addonInstance); err != nil {
		return fmt.Errorf("updating AddonInstance status: %w", err)
	}
	return nil
}
