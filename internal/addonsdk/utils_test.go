package addonsdk

import (
	"context"
	"errors"
	"fmt"
	"testing"

	addonsv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openshift/reference-addon/internal/testutil"
)

func TestUpdateAddonInstanceStatus(t *testing.T) {
	type errLevel string
	var (
		get    errLevel = "Get"
		update errLevel = "Update"
		never  errLevel = "None"
	)

	tests := []struct {
		name   string
		errOn  errLevel
		errMsg string
	}{
		{
			name:   "everything runs successfully",
			errOn:  never,
			errMsg: "",
		},
		{
			name:   "fails on GET addonInstance",
			errOn:  get,
			errMsg: "couldn't GET the addonInstance",
		},
		{
			name:   "fails on UPDATE addonInstance status",
			errOn:  update,
			errMsg: "couldn't UPDATE the addonInstance status",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := testutil.NewClient()

			var getReturnArguments, updateReturnArguments error

			if test.errOn == get {
				getReturnArguments = errors.New(test.errMsg)
			}

			if test.errOn == update {
				updateReturnArguments = errors.New(test.errMsg)
			}

			c.On(
				"Get",
				mock.IsType(context.TODO()),
				mock.IsType(types.NamespacedName{}),
				mock.IsType(&addonsv1alpha1.AddonInstance{}),
			).Run(func(args mock.Arguments) {
				adi := NewTestAddonInstance()
				adi.DeepCopyInto(args.Get(2).(*addonsv1alpha1.AddonInstance))
			}).Return(getReturnArguments)

			c.StatusMock.On(
				"Update",
				mock.IsType(context.TODO()),
				mock.IsType(&addonsv1alpha1.AddonInstance{}),
				mock.Anything,
			).Return(updateReturnArguments)

			testStatusReporter := NewTestStatusReporter(t, WithClient(c))
			testConditions := []metav1.Condition{
				{
					Type:    AddonHealthyConditionType,
					Status:  metav1.ConditionTrue,
					Reason:  "AddonWorking",
					Message: "All components of reference-addon seem to be working in harmony",
				},
			}

			err := testStatusReporter.updateAddonInstanceStatus(context.TODO(), testConditions)

			// Atleast "Get" will be called no matter what
			// no need to go as granular as `c.AssertCalled(t, "Get", ...)` it's implied that the mock would get called with the correct set of args considering there's only one `c.On("Get")` definition
			c.AssertNumberOfCalls(t, "Get", 1)

			switch test.errOn {
			case never:
				assert.NoError(t, err)
				c.StatusMock.AssertNumberOfCalls(t, "Update", 1)
			case get:
				expectedErrStr := fmt.Sprintf("failed to get the AddonInstance: %s", test.errMsg)
				assert.EqualError(t, err, expectedErrStr)
				c.StatusMock.AssertNumberOfCalls(t, "Update", 0)
			case update:
				expectedErr := fmt.Errorf("failed to update AddonInstance Status: %w", errors.New(test.errMsg))
				assert.EqualError(t, err, expectedErr.Error())
				c.StatusMock.AssertNumberOfCalls(t, "Update", 1)
			}
		})
	}
}
