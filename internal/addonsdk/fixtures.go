package addonsdk

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
