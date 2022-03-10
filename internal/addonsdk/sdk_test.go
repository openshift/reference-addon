package addonsdk

import (
	"context"
	"testing"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openshift/reference-addon/internal/testutil"
)

func TestAddonSDKReportAddonInstanceSpecChange(t *testing.T) {
	t.Run("fails to report addonInstanceSpec change on a stopped reporter", func(t *testing.T) {
		t.Parallel()
		s := NewTestStatusReporter(t)

		testAddonInstance := addonsv1alpha1.AddonInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "addon-instance",
				Namespace: "reference-addon",
			},
		}

		err := s.ReportAddonInstanceSpecChange(context.TODO(), testAddonInstance)
		assert.EqualError(t, err, "can't report AddonInstance spec change on a stopped StatusReporter")
	})

	t.Run("fails to report addonInstanceSpec change over an exhausted context", func(t *testing.T) {
		t.Parallel()
		s := NewTestStatusReporter(t, WithRunningStatusReporter)

		ctx, cancel := context.WithCancel(context.TODO())
		cancel() // marking the ctx as `Done`

		testAddonInstance := addonsv1alpha1.AddonInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "addon-instance",
				Namespace: "reference-addon",
			},
		}

		err := s.ReportAddonInstanceSpecChange(ctx, testAddonInstance)
		assert.EqualError(t, err, ctx.Err().Error())
	})

	t.Run("succeeds to report addonInstanceSpec change on a running status reporter", func(t *testing.T) {
		t.Parallel()
		s := NewTestStatusReporter(t, WithRunningStatusReporter)

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel() //cleanup

		go func(ctx context.Context) {
			for {
				select {
				case <-s.updateCh:
				case <-ctx.Done():
					return
				}
			}
		}(ctx)

		testAddonInstance := addonsv1alpha1.AddonInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "addon-instance",
				Namespace: "reference-addon",
			},
		}

		err := s.ReportAddonInstanceSpecChange(context.TODO(), testAddonInstance)
		assert.NoError(t, err)
	})
}

func TestAddonSDKSetConditions(t *testing.T) {
	t.Run("error on setting conditions on no running status reporter", func(t *testing.T) {
		t.Parallel()
		s := NewTestStatusReporter(t)

		testConditions := []metav1.Condition{
			{
				Type:    AddonHealthyConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  "AddonWorking",
				Message: "All components of reference-addon seem to be working in harmony",
			},
		}

		err := s.SetConditions(context.TODO(), testConditions)
		assert.EqualError(t, err, "StatusReporter found to be stopped: can't set conditions to a stopped StatusReporter")
	})

	t.Run("successfully set conditions on a running status reporter", func(t *testing.T) {
		t.Parallel()
		c := testutil.NewClient()

		c.On(
			"Get",
			mock.IsType(context.TODO()),
			mock.IsType(types.NamespacedName{}),
			mock.IsType(&addonsv1alpha1.AddonInstance{}),
		).Run(func(args mock.Arguments) {
			adi := &addonsv1alpha1.AddonInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "addon-instance",
					Namespace: "reference-addon",
				},
			}
			adi.DeepCopyInto(args.Get(2).(*addonsv1alpha1.AddonInstance))
		}).Return(nil)

		c.StatusMock.On(
			"Update",
			mock.IsType(context.TODO()),
			mock.IsType(&addonsv1alpha1.AddonInstance{}),
			mock.Anything,
		).Return(nil)

		s := NewTestStatusReporter(t, WithRunningStatusReporter, WithClient(c))

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel() //cleanup

		go func(ctx context.Context) {
			for {
				select {
				case <-s.updateCh:
				case <-ctx.Done():
					return
				}
			}
		}(ctx)

		testConditions := []metav1.Condition{
			{
				Type:    AddonHealthyConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  "AddonWorking",
				Message: "All components of reference-addon seem to be working in harmony",
			},
		}

		err := s.SetConditions(context.TODO(), testConditions)
		assert.NoError(t, err)
		c.AssertNumberOfCalls(t, "Get", 1)
		c.StatusMock.AssertNumberOfCalls(t, "Update", 1)
	})

}
