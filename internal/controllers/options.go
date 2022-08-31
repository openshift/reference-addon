package controllers

import "github.com/go-logr/logr"

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureReferenceAddonReconciler(c *ReferenceAddonReconcilerConfig) {
	c.Log = w.Log
}

func (w WithLog) ConfigurePhaseSimulateReconciliation(c *PhaseSimulateReconciliationConfig) {
	c.Log = w.Log
}

func (w WithLog) ConfigurePhaseUninstall(c *PhaseUninstallConfig) {
	c.Log = w.Log
}

type WithAddonNamespace string

func (w WithAddonNamespace) ConfigureConfigMapUninstallSignaler(c *ConfigMapUninstallSignalerConfig) {
	c.AddonNamespace = string(w)
}

func (w WithAddonNamespace) ConfigureReferenceAddonReconciler(c *ReferenceAddonReconcilerConfig) {
	c.AddonNamespace = string(w)
}

func (w WithAddonNamespace) ConfigurePhaseUninstall(c *PhaseUninstallConfig) {
	c.AddonNamespace = string(w)
}

type WithOperatorName string

func (w WithOperatorName) ConfigureConfigMapUninstallSignaler(c *ConfigMapUninstallSignalerConfig) {
	c.OperatorName = string(w)
}

func (w WithOperatorName) ConfigurePhaseUninstall(c *PhaseUninstallConfig) {
	c.OperatorName = string(w)
}

func (w WithOperatorName) ConfigureReferenceAddonReconciler(c *ReferenceAddonReconcilerConfig) {
	c.OperatorName = string(w)
}

type WithDeleteLabel string

func (w WithDeleteLabel) ConfigureConfigMapUninstallSignaler(c *ConfigMapUninstallSignalerConfig) {
	c.DeleteLabel = string(w)
}

func (w WithDeleteLabel) ConfigureReferenceAddonReconciler(c *ReferenceAddonReconcilerConfig) {
	c.DeleteLabel = string(w)
}

type WithNamespace string

func (w WithNamespace) ConfigureListCSVs(c *ListCSVsConfig) {
	c.Namespace = string(w)
}

type WithPrefix string

func (w WithPrefix) ConfigureListCSVs(c *ListCSVsConfig) {
	c.Prefix = string(w)
}

type WithSampleURLs []string

func (w WithSampleURLs) ConfigurePhaseSendDummyMetrics(c *PhaseSendDummyMetricsConfig) {
	c.SampleURLs = []string(w)
}
