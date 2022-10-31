package controllers

import (
	"context"
	"testing"

	"github.com/openshift/reference-addon/internal/controllers/phase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPhaseApplyNetworkPoliciesInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(phase.Phase), new(PhaseApplyNetworkPolicies))
}

func TestPhaseApplyNetworkPolicies(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		ApplyNetworkPolicy *bool
		Policies           []netv1.NetworkPolicy
	}{
		"applyNetworkPolicies unset": {
			ApplyNetworkPolicy: nil,
		},
		"applyNetworkPolicies false/no NetworkPolicies": {
			ApplyNetworkPolicy: boolPtr(false),
		},
		"applyNetworkPolicies false/with NetworkPolicies": {
			ApplyNetworkPolicy: boolPtr(false),
			Policies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
		},
		"applyNetworkPolicies true/no NetworkPolicies": {
			ApplyNetworkPolicy: boolPtr(false),
		},
		"applyNetworkPolicies true/with NetworkPolicies": {
			ApplyNetworkPolicy: boolPtr(true),
			Policies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var m NetworkPolicyClientMock

			switch val := tc.ApplyNetworkPolicy; {
			case val == nil:
			case *val == true:
				argList := []interface{}{mock.Anything, mock.Anything, WithPolicies(tc.Policies)}

				m.
					On("ApplyNetworkPolicies", argList...).
					Return(nil)
			case *val == false:
				argList := make([]interface{}, 0, 1+len(tc.Policies))

				argList = append(argList, mock.Anything)

				for _, p := range tc.Policies {
					argList = append(argList, p)
				}

				m.
					On("RemoveNetworkPolicies", argList...).
					Return(nil)
			}

			p := NewPhaseApplyNetworkPolicies(
				&m,
				WithPolicies(tc.Policies),
			)

			res := p.Execute(context.Background(), phase.Request{
				Params: phase.NewRequestParameters(
					phase.WithApplyNetworkPolicies{Value: tc.ApplyNetworkPolicy},
				),
			})
			require.NoError(t, res.Error())

			assert.Equal(t, phase.StatusSuccess, res.Status())

			m.AssertExpectations(t)
		})
	}
}

type NetworkPolicyClientMock struct {
	mock.Mock
}

func (m *NetworkPolicyClientMock) ApplyNetworkPolicies(ctx context.Context, opts ...ApplyNetorkPoliciesOption) error {
	argList := make([]interface{}, 0, 1+len(opts))

	argList = append(argList, ctx)

	for _, o := range opts {
		argList = append(argList, o)
	}

	args := m.Called(argList...)

	return args.Error(0)
}

func (m *NetworkPolicyClientMock) RemoveNetworkPolicies(ctx context.Context, policies ...netv1.NetworkPolicy) error {
	argList := make([]interface{}, 0, 1+len(policies))

	argList = append(argList, ctx)

	for _, p := range policies {
		argList = append(argList, p)
	}

	args := m.Called(argList...)

	return args.Error(0)
}

func TestNetworkPolicyClientImplInterfaces(t *testing.T) {
	t.Parallel()

	require.Implements(t, new(NetworkPolicyClient), new(NetworkPolicyClientImpl))
}

func TestNetworkPolicyClientImpl_ApplyNetworkPolicies(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		ActualPolicies   []netv1.NetworkPolicy
		DesiredPolicies  []netv1.NetworkPolicy
		ExpectedPolicies []netv1.NetworkPolicy
	}{
		"no existing policies/no new policies": {
			ActualPolicies:   []netv1.NetworkPolicy{},
			DesiredPolicies:  []netv1.NetworkPolicy{},
			ExpectedPolicies: []netv1.NetworkPolicy{},
		},
		"existing policies/no new policies": {
			ActualPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			DesiredPolicies: []netv1.NetworkPolicy{},
			ExpectedPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
		},
		"no existing policies/one new policy": {
			ActualPolicies: []netv1.NetworkPolicy{},
			DesiredPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			ExpectedPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
		},
		"existing policy/one desired policy": {
			ActualPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			DesiredPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			ExpectedPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			objs := make([]client.Object, 0, len(tc.ActualPolicies))

			for _, p := range tc.ActualPolicies {
				objs = append(objs, &p)
			}

			c := fake.
				NewClientBuilder().
				WithObjects(objs...).
				Build()

			npClient := NewNetworkPolicyClientImpl(c)

			require.NoError(t, npClient.ApplyNetworkPolicies(
				context.Background(),
				WithPolicies(tc.DesiredPolicies),
			),
			)

			for _, expected := range tc.ExpectedPolicies {
				assert.NoError(t, c.Get(
					context.Background(),
					client.ObjectKeyFromObject(&expected),
					new(netv1.NetworkPolicy),
				),
				)
			}
		})
	}
}

func TestNetworkPolicyClientImpl_RemoveNetworkPolicies(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		ActualPolicies   []netv1.NetworkPolicy
		DesiredPolicies  []netv1.NetworkPolicy
		ExpectedPolicies []netv1.NetworkPolicy
	}{
		"no existing policies/no new policies": {
			ActualPolicies:   []netv1.NetworkPolicy{},
			DesiredPolicies:  []netv1.NetworkPolicy{},
			ExpectedPolicies: []netv1.NetworkPolicy{},
		},
		"existing policies/no new policies": {
			ActualPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			DesiredPolicies: []netv1.NetworkPolicy{},
			ExpectedPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
		},
		"no existing policies/one new policy": {
			ActualPolicies: []netv1.NetworkPolicy{},
			DesiredPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			ExpectedPolicies: []netv1.NetworkPolicy{},
		},
		"existing policy/one desired policy": {
			ActualPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			DesiredPolicies: []netv1.NetworkPolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test-namespace",
					},
				},
			},
			ExpectedPolicies: []netv1.NetworkPolicy{},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			objs := make([]client.Object, 0, len(tc.ActualPolicies))

			for _, p := range tc.ActualPolicies {
				objs = append(objs, &p)
			}

			c := fake.
				NewClientBuilder().
				WithObjects(objs...).
				Build()

			npClient := NewNetworkPolicyClientImpl(c)

			require.NoError(t, npClient.RemoveNetworkPolicies(
				context.Background(),
				tc.DesiredPolicies...,
			),
			)

			for _, expected := range tc.ExpectedPolicies {
				assert.NoError(t, c.Get(
					context.Background(),
					client.ObjectKeyFromObject(&expected),
					new(netv1.NetworkPolicy),
				),
				)
			}
		})
	}
}
