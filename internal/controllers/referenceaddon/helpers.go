package referenceaddon

import (
	"fmt"

	refv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateIngressPolicyName(prefix string) string {
	return fmt.Sprintf("%s-ingress", prefix)
}

func newAvailableCondition(reason refv1alpha1.ReferenceAddonAvailableReason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:               refv1alpha1.ReferenceAddonConditionAvailable.String(),
		Status:             reason.Status(),
		Reason:             reason.String(),
		Message:            msg,
		LastTransitionTime: metav1.Now(),
	}
}
