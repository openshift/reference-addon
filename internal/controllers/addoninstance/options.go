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
	c.StatusControllerNamespace = string(w)
}

type WithStatusControllerName string

func (w WithStatusControllerName) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.StatusControllerName = string(w)
}

type WithReferenceAddonNamespace string

func (w WithReferenceAddonNamespace) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.ReferenceAddonNamespace = string(w)
}

type WithReferenceAddonName string

func (w WithReferenceAddonName) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.ReferenceAddonName = string(w)
}

type WithRetryAfterTime int

func (w WithRetryAfterTime) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.RetryAfterTime = time.Duration(w)
}
