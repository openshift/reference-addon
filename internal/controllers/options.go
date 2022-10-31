package controllers

import (
	"github.com/go-logr/logr"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureReferenceAddonReconciler(c *ReferenceAddonReconcilerConfig) {
	c.Log = w.Log
}

func (w WithLog) ConfigurePhaseApplyNetworkPolicies(c *PhaseApplyNetworkPoliciesConfig) {
	c.Log = w.Log
}

func (w WithLog) ConfigurePhaseUninstall(c *PhaseUninstallConfig) {
	c.Log = w.Log
}

func (w WithLog) ConfigureUninstallerImpl(c *UninstallerImplConfig) {
	c.Log = w.Log
}

func (w WithLog) ConfigureCSVClientImpl(c *CSVClientImplConfig) {
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

type WithAddonParameterSecretName string

func (w WithAddonParameterSecretName) ConfigureReferenceAddonReconciler(c *ReferenceAddonReconcilerConfig) {
	c.AddonParameterSecretname = string(w)
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

type WithName string

func (w WithName) ConfigureSecretParameterGetter(c *SecretParameterGetterConfig) {
	c.Name = string(w)
}

type WithNamespace string

func (w WithNamespace) ConfigureSecretParameterGetter(c *SecretParameterGetterConfig) {
	c.Namespace = string(w)
}

func (w WithNamespace) ConfigureListCSVs(c *ListCSVsConfig) {
	c.Namespace = string(w)
}

type WithOwner struct{ Owner metav1.Object }

func (w WithOwner) ConfigureApplyNetworkPolicies(c *ApplyNetorkPoliciesConfig) {
	c.Owner = w.Owner
}

type WithPolicies []netv1.NetworkPolicy

func (w WithPolicies) ConfigurePhaseApplyNetworkPolicies(c *PhaseApplyNetworkPoliciesConfig) {
	c.Policies = append(c.Policies, w...)
}

func (w WithPolicies) ConfigureApplyNetworkPolicies(c *ApplyNetorkPoliciesConfig) {
	c.Policies = append(c.Policies, w...)
}

type WithPrefix string

func (w WithPrefix) ConfigureListCSVs(c *ListCSVsConfig) {
	c.Prefix = string(w)
}

type WithSampleURLs []string

func (w WithSampleURLs) ConfigurePhaseSendDummyMetrics(c *PhaseSendDummyMetricsConfig) {
	c.SampleURLs = []string(w)
}
