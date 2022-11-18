package referenceaddon

import (
	"context"
	"testing"

	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPhaseUninstallInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(Phase), new(PhaseUninstall))
}

func TestPhaseUninstall(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		AddonNamespace string
		OperatorName   string
		DeleteLabel    string
	}{
		"happy path": {
			AddonNamespace: "test-namespace",
			OperatorName:   "test-operator",
			DeleteLabel:    "addon-test-operator-delete",
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var signaler uninstallSignalerMock

			signaler.
				On("SignalUninstall", mock.Anything).
				Return(true)

			var uninstaller uninstallerMock

			uninstaller.
				On("Uninstall", mock.Anything, tc.AddonNamespace, tc.OperatorName).
				Return(nil)

			p := NewPhaseUninstall(
				&signaler,
				&uninstaller,
				WithAddonNamespace(tc.AddonNamespace),
				WithOperatorName(tc.OperatorName),
			)

			res := p.Execute(context.Background(), PhaseRequest{})
			require.NoError(t, res.Error())

			signaler.AssertExpectations(t)
			uninstaller.AssertExpectations(t)
		})
	}
}

type uninstallSignalerMock struct {
	mock.Mock
}

func (m *uninstallSignalerMock) SignalUninstall(ctx context.Context) bool {
	args := m.Called(ctx)

	return args.Bool(0)
}

type uninstallerMock struct {
	mock.Mock
}

func (m *uninstallerMock) Uninstall(ctx context.Context, namespace, operatorName string) error {
	args := m.Called(ctx, namespace, operatorName)

	return args.Error(0)
}

func TestUninstallSignalerImpl(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		ConfigMap      *corev1.ConfigMap
		AddonNamespace string
		OperatorName   string
		DeleteLabel    string
		AssertResult   assert.BoolAssertionFunc
	}{
		"signal not present": {
			ConfigMap:      &corev1.ConfigMap{},
			AddonNamespace: "test-namespace",
			OperatorName:   "test-operator",
			DeleteLabel:    "test-delete-label",
			AssertResult:   assert.False,
		},
		"signal present": {
			ConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"test-delete-label": "true",
					},
				},
			},
			AddonNamespace: "test-namespace",
			OperatorName:   "test-operator",
			DeleteLabel:    "test-delete-label",
			AssertResult:   assert.True,
		},
		"misconfigured operator name": {
			ConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"test-delete-label": "true",
					},
				},
			},
			AddonNamespace: "test-namespace",
			OperatorName:   "misconfigured",
			DeleteLabel:    "test-delete-label",
			AssertResult:   assert.False,
		},
		"misconfigured addon namespace": {
			ConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"test-delete-label": "true",
					},
				},
			},
			AddonNamespace: "misconfigured",
			OperatorName:   "test-operator",
			DeleteLabel:    "test-delete-label",
			AssertResult:   assert.False,
		},
		"misconfigured delete label": {
			ConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"test-delete-label": "true",
					},
				},
			},
			AddonNamespace: "test-namespace",
			OperatorName:   "test-operator",
			DeleteLabel:    "misconfigured",
			AssertResult:   assert.False,
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := fake.NewClientBuilder().
				WithObjects(tc.ConfigMap).
				Build()

			signaler, err := NewConfigMapUninstallSignaler( //nolint
				client,
				WithAddonNamespace(tc.AddonNamespace),
				WithOperatorName(tc.OperatorName),
				WithDeleteLabel(tc.DeleteLabel),
			)
			require.NoError(t, err)

			tc.AssertResult(t, signaler.SignalUninstall(context.Background()))
		})
	}
}

func TestUninstallImpl(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, opsv1alpha1.AddToScheme(scheme))

	for name, tc := range map[string]struct {
		ActualCSV    opsv1alpha1.ClusterServiceVersion
		CSVNamespace string
		CSVPrefix    string
	}{
		"happy path": {
			ActualCSV: opsv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator.v0.0.0",
					Namespace: "test-namespace",
				},
			},
			CSVNamespace: "test-namespace",
			CSVPrefix:    "test-operator",
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var csvClient csvClientMock
			csvClient.
				On("ListCSVs", mock.Anything, WithNamespace(tc.CSVNamespace), WithPrefix(tc.CSVPrefix)).
				Return([]opsv1alpha1.ClusterServiceVersion{tc.ActualCSV}, nil)
			csvClient.
				On("RemoveCSVs", mock.Anything, tc.ActualCSV).
				Return(nil)

			uninstaller := NewUninstallerImpl(&csvClient)
			require.NoError(t, uninstaller.Uninstall(context.Background(), tc.CSVNamespace, tc.CSVPrefix))
		})
	}
}

type csvClientMock struct {
	mock.Mock
}

func (m *csvClientMock) ListCSVs(ctx context.Context, opts ...ListCSVsOption) ([]opsv1alpha1.ClusterServiceVersion, error) {
	argList := make([]interface{}, 0, len(opts)+1)

	argList = append(argList, ctx)

	for _, opt := range opts {
		argList = append(argList, opt)
	}

	args := m.Called(argList...)

	return args.Get(0).([]opsv1alpha1.ClusterServiceVersion), args.Error(1)
}

func (m *csvClientMock) RemoveCSVs(ctx context.Context, csvs ...opsv1alpha1.ClusterServiceVersion) error {
	argList := make([]interface{}, 0, len(csvs)+1)

	argList = append(argList, ctx)

	for _, csv := range csvs {
		argList = append(argList, csv)
	}

	args := m.Called(argList...)

	return args.Error(0)
}

func TestCSVListerImpl_ListCSVs(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, opsv1alpha1.AddToScheme(scheme))

	for name, tc := range map[string]struct {
		ActualCSV    opsv1alpha1.ClusterServiceVersion
		CSVNamespace string
		CSVPrefix    string
	}{
		"happy path": {
			ActualCSV: opsv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator.v0.0.0",
					Namespace: "test-namespace",
				},
			},
			CSVNamespace: "test-namespace",
			CSVPrefix:    "test-operator",
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&tc.ActualCSV).
				Build()

			lister := NewCSVClientImpl(client)

			csvs, err := lister.ListCSVs(context.Background(), WithNamespace(tc.CSVNamespace), WithPrefix(tc.CSVPrefix))
			require.NoError(t, err)

			assert.Contains(t, csvs, tc.ActualCSV)
		})
	}
}

func TestCSVListerImpl_RemoveCSVs(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, opsv1alpha1.AddToScheme(scheme))

	for name, tc := range map[string]struct {
		ActualCSV    opsv1alpha1.ClusterServiceVersion
		CSVNamespace string
		CSVPrefix    string
	}{
		"happy path": {
			ActualCSV: opsv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-operator.v0.0.0",
					Namespace: "test-namespace",
				},
			},
			CSVNamespace: "test-namespace",
			CSVPrefix:    "test-operator",
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&tc.ActualCSV).
				Build()

			lister := NewCSVClientImpl(c)
			require.NoError(t, lister.RemoveCSVs(context.Background(), tc.ActualCSV))

			key := client.ObjectKeyFromObject(&tc.ActualCSV)
			assert.Error(t, c.Get(context.Background(), key, &tc.ActualCSV))
		})
	}
}
