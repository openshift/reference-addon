package status

import (
	"time"

	"github.com/go-logr/logr"
)

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.Log = w.Log
}

type WithAddonInstanceNamespace string

func (w WithAddonInstanceNamespace) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.AddonInstanceNamespace = string(w)
}

type WithAddonInstanceName string

func (w WithAddonInstanceName) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.AddonInstanceName = string(w)
}

type WithReferenceAddonNamespace string

func (w WithReferenceAddonNamespace) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.ReferenceAddonNamespace = string(w)
}

type WithReferenceAddonName string

func (w WithReferenceAddonName) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.ReferenceAddonName = string(w)
}

type WithHeartbeatInterval time.Duration

func (w WithHeartbeatInterval) ConfigureStatusControllerReconciler(c *StatusControllerReconcilerConfig) {
	c.HeartBeatInterval = time.Duration(w)
}
