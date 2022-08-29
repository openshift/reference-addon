package controllers

import (
	"context"
	"testing"

	"github.com/openshift/reference-addon/internal/controllers/phase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
)

func TestPhaseSimulateReconciliationInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(phase.Phase), new(PhaseSimulateReconciliation))
}

func TestPhaseSimulateReconciliation(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		Request phase.Request
	}{
		"redhat prefixed object": {
			Request: phase.Request{
				Object: types.NamespacedName{
					Name: "redhat-test",
				},
			},
		},
		"reference-addon object": {
			Request: phase.Request{
				Object: types.NamespacedName{
					Name: "reference-addon",
				},
			},
		},
		"non redhat prefixed object": {
			Request: phase.Request{
				Object: types.NamespacedName{
					Name: "test",
				},
			},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := NewPhaseSimulateReconciliation()

			res := p.Execute(context.Background(), tc.Request)
			require.NoError(t, res.Error())

			assert.True(t, res.IsSuccess())
		})
	}
}
