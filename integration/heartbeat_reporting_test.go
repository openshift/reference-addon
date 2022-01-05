package integration_test

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/reference-addon/integration"

	"time"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"

	referenceaddonv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
)

func (s *integrationTestSuite) TestReferenceAddonHeartbeatReporting() {
	tests := []struct {
		name                 string
		referenceAddonObject referenceaddonv1alpha1.ReferenceAddon
		expectedHeartbeat    v1.Condition
	}{
		{
			name: "ProperlyNamed",
			referenceAddonObject: referenceaddonv1alpha1.ReferenceAddon{
				Spec: referenceaddonv1alpha1.ReferenceAddonSpec{},
				ObjectMeta: v1.ObjectMeta{
					Name:      "reference-addon",
					Namespace: "reference-addon",
				},
			},
			expectedHeartbeat: v1.Condition{
				Type:    "addons.managed.openshift.io/Healthy",
				Status:  "True",
				Reason:  "AllComponentsUp",
				Message: "Everything under reference-addon is working perfectly fine",
			},
		},
		{
			name: "ImproperlyNamed",
			referenceAddonObject: referenceaddonv1alpha1.ReferenceAddon{
				Spec: referenceaddonv1alpha1.ReferenceAddonSpec{},
				ObjectMeta: v1.ObjectMeta{
					Name:      "something-yada-yada",
					Namespace: "reference-addon",
				},
			},
			expectedHeartbeat: v1.Condition{
				Type:    "addons.managed.openshift.io/Healthy",
				Status:  "False",
				Reason:  "ImproperNaming",
				Message: "The addon resources are improperly named",
			},
		},
	}

	for _, test := range tests {
		test := test
		s.Run(test.name, func() {
			ctx := context.Background()
			referenceAddonObject := test.referenceAddonObject
			s.Require().NoError(integration.Client.Create(ctx, &referenceAddonObject))
			s.T().Cleanup(func() {
				s.referenceAddonCleanup(&referenceAddonObject, ctx)

			})
			time.Sleep(2 * time.Second)
			// check that there is an addonInstance in the target namespace.
			addonInstance := &addonsv1alpha1.AddonInstance{}
			err := integration.Client.Get(ctx, client.ObjectKey{
				Name:      addonsv1alpha1.DefaultAddonInstanceName,
				Namespace: "reference-addon",
			}, addonInstance)
			s.Require().NoError(err)

			currentAddonInstanceCondition := meta.FindStatusCondition(addonInstance.Status.Conditions, "addons.managed.openshift.io/Healthy")
			s.Require().NotNil(currentAddonInstanceCondition)
			s.Assert().Equal(test.expectedHeartbeat, v1.Condition{
				Type:    currentAddonInstanceCondition.Type,
				Status:  currentAddonInstanceCondition.Status,
				Reason:  currentAddonInstanceCondition.Reason,
				Message: currentAddonInstanceCondition.Message,
			})
		})
	}
}
