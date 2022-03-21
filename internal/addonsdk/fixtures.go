package addonsdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"

	"github.com/openshift/reference-addon/internal/testutil"
)

type StatusReporterOpt func(s *StatusReporter)

func NewTestStatusReporter(t *testing.T, opts ...StatusReporterOpt) *StatusReporter {
	s := &StatusReporter{
		addonName:            "test-reference-addon",
		addonTargetNamespace: "reference-addon",
		latestConditions: []metav1.Condition{
			{
				Type:    AddonHealthyConditionType,
				Status:  metav1.ConditionUnknown,
				Reason:  "NoHealthReported",
				Message: fmt.Sprintf("Addon %q hasn't reported health yet", "test-reference-addon"),
			},
		},
		updateCh:  make(chan updateOptions),
		runnersCh: make(chan bool, 1), //empty channel indicating no reporters running as of now
		log:       testutil.NewLogger(t),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithRunningStatusReporter(s *StatusReporter) {
	s.runnersChMutex.Lock()
	defer s.runnersChMutex.Unlock()
	if len(s.runnersCh) == 0 {
		s.runnersCh <- true
	}
}

func WithClient(client *testutil.Client) StatusReporterOpt {
	return StatusReporterOpt(func(s *StatusReporter) {
		s.addonInstanceInteractor = testutil.NewAddonSdkClientMock(client)
	})
}

func NewStatusReporterRunner(ctx context.Context, s *StatusReporter) {
	for {
		select {
		case <-s.updateCh:
		case <-ctx.Done():
			return
		}
	}
}

func NewTestAddonInstance() addonsv1alpha1.AddonInstance {
	return addonsv1alpha1.AddonInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "addon-instance",
			Namespace: "reference-addon",
		},
		Spec: addonsv1alpha1.AddonInstanceSpec{
			HeartbeatUpdatePeriod: metav1.Duration{Duration: 10 * time.Second},
		},
	}
}
