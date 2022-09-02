package controllers

import (
	"context"
	"testing"

	"github.com/openshift/reference-addon/internal/controllers/phase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretParameterGetterInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(ParameterGetter), new(SecretParameterGetter))
}

func TestSecretParameterGetter(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		ActualSecret   *corev1.Secret
		Namespace      string
		Name           string
		ExpectedParams phase.RequestParameters
	}{
		"happy path": {
			ActualSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"size": []byte("1"),
				},
			},
			Namespace: "test-namespace",
			Name:      "test",
			ExpectedParams: phase.RequestParameters{
				Size: "1",
			},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := fake.
				NewClientBuilder().
				WithObjects(tc.ActualSecret).
				Build()

			getter := NewSecretParameterGetter(
				client,
				WithNamespace(tc.Namespace),
				WithName(tc.Name),
			)

			params, err := getter.GetParameters(context.Background())
			require.NoError(t, err)

			assert.Equal(t, tc.ExpectedParams, params)
		})
	}
}
