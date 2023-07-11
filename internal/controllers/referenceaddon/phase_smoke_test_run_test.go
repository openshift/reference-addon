package referenceaddon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPhaseSmokeTestRunInterface(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(Phase), new(PhaseSmokeTestRun))
}

func TestPhaseSmokeTestRun_Execute(t *testing.T) {
	t.Parallel()

	var (
		tr = true
		f  = false
	)

	for name, tc := range map[string]struct {
		EnableSmokeTest *bool
	}{
		"enablesmoketest 'nil'": {
			EnableSmokeTest: nil,
		},
		"enablesmoketest 'false'": {
			EnableSmokeTest: &f,
		},
		"enablesmoketest 'true'": {
			EnableSmokeTest: &tr,
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tester := &SmokeTesterMock{}

			if tc.EnableSmokeTest != nil {
				if *tc.EnableSmokeTest {
					tester.On("Enable")
				} else {
					tester.On("Disable")
				}
			}

			phase := NewPhaseSmokeTestRun(
				WithSmokeTester{
					Tester: tester},
			)

			res := phase.Execute(context.Background(), PhaseRequest{
				Params: NewPhaseRequestParameters(
					WithEnableSmokeTest{
						Value: tc.EnableSmokeTest,
					},
				),
			})

			assert.Equal(t, PhaseStatusSuccess, res.Status())
			tester.AssertExpectations(t)
		})
	}
}

type SmokeTesterMock struct {
	mock.Mock
}

func (m *SmokeTesterMock) Enable() {
	m.Called()
}

func (m *SmokeTesterMock) Disable() {
	m.Called()
}
