package referenceaddon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPhaseSendDummyMetricsInterface(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(Phase), new(PhaseSendDummyMetrics))
}

func TestPhaseSendDummyMetrics(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		SampleURLs []string
	}{
		"happy path": {
			SampleURLs: []string{"https://fake.io"},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			argList := make([]interface{}, 0, len(tc.SampleURLs))

			for _, url := range tc.SampleURLs {
				argList = append(argList, url)
			}

			var sampler ResponseSamplerMock
			sampler.
				On("RequestSampleResponseData", argList...).
				Return()

			p := NewPhaseSendDummyMetrics(&sampler, WithSampleURLs(tc.SampleURLs))

			res := p.Execute(context.Background(), PhaseRequest{})
			require.NoError(t, res.Error())

			assert.Equal(t, PhaseStatusSuccess, res.Status())
		})
	}
}

type ResponseSamplerMock struct {
	mock.Mock
}

func (r *ResponseSamplerMock) RequestSampleResponseData(urls ...string) {
	argList := make([]interface{}, 0, len(urls))

	for _, url := range urls {
		argList = append(argList, url)
	}

	r.Called(argList...)
}
