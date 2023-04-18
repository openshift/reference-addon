package addoninstance

import (
	"time"

	"github.com/go-logr/logr"
)

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.Log = w.Log
}

type WithStatusControllerNamespace string

func (w WithStatusControllerNamespace) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.statusControllerNamespace = string(w)
}

type WithStatusControllerName string

func (w WithStatusControllerName) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.statusControllerName = string(w)
}

type WithReferenceAddonNamespace string

func (w WithReferenceAddonNamespace) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.referenceAddonNamespace = string(w)
}

type WithReferenceAddonName string

func (w WithReferenceAddonName) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.referenceAddonName = string(w)
}

type WithRetryAfterTime int

func (w WithRetryAfterTime) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.retryAfterTime = time.Duration(w)
}
