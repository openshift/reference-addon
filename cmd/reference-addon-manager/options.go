package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type options struct {
	DeleteLabel               string
	EnableLeaderElection      bool
	EnableMetricsRecorder     bool
	MetricsAddr               string
	Namespace                 string
	OperatorName              string
	ParameterSecretname       string
	PprofAddr                 string
	ProbeAddr                 string
	StatusControllerName      string
	StatusControllerNamespace string
	RetryAfterTime            time.Duration
	Zap                       zap.Options
}

func (o *options) Process() error {
	o.processFlags()
	o.processSecrets()

	return o.validate()
}

func (o *options) processFlags() {
	flags := flag.CommandLine

	flags.StringVar(
		&o.DeleteLabel,
		"delete-label",
		o.DeleteLabel,
		"Label applied to addon ConfigMap to trigger deletion.",
	)

	flags.BoolVar(
		&o.EnableLeaderElection,
		"enable-leader-election",
		o.EnableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)

	flags.BoolVar(
		&o.EnableMetricsRecorder,
		"enable-metrics-recorder",
		o.EnableMetricsRecorder,
		"Enable recording Addon Metrics.",
	)

	flags.StringVar(
		&o.MetricsAddr,
		"metrics-addr",
		o.MetricsAddr,
		"The address the metric endpoint binds to.",
	)

	flags.StringVar(
		&o.Namespace,
		"namespace",
		o.Namespace,
		"The namepsace in which the operator will run.",
	)

	flags.StringVar(
		&o.OperatorName,
		"operator-name",
		o.OperatorName,
		"The operator's Bundle name.",
	)

	flags.StringVar(
		&o.ParameterSecretname,
		"parameter-secret-name",
		o.ParameterSecretname,
		"The name of the Secret where addon parameters can be retrieved.",
	)

	flags.StringVar(
		&o.PprofAddr,
		"pprof-addr",
		o.PprofAddr,
		"The address the pprof web endpoint binds to.",
	)

	flags.StringVar(
		&o.ProbeAddr,
		"health-probe-bind-address",
		o.ProbeAddr,
		"The address the probe endpoint binds to.",
	)

	flags.StringVar(
		&o.StatusControllerName,
		"addon-instance-name",
		o.StatusControllerName,
		"The name of addon instance operator.",
	)

	flags.StringVar(
		&o.StatusControllerNamespace,
		"addon-instance-namespace",
		o.StatusControllerNamespace,
		"The namespace addon instance exists in.",
	)

	flags.Duration(
		"retry-after-time",
		o.RetryAfterTime,
		"Time between retries for addon instance",
	)

	o.Zap.BindFlags(flags)

	flag.Parse()
}

func (o *options) processSecrets() {
	const (
		scrtsPath              = "/var/run/secrets"
		inClusterNamespacePath = scrtsPath + "/kubernetes.io/serviceaccount/namespace"
	)

	var namespace string

	if ns, err := os.ReadFile(inClusterNamespacePath); err == nil {
		// Avoid applying a garbage value if an error occurred
		namespace = string(ns)
	}

	if o.Namespace == "" {
		o.Namespace = namespace
	}
}

var ErrEmptyValue = errors.New("empty value")

func (o *options) validate() error {
	if o.Namespace == "" {
		return fmt.Errorf("validating namespace: %w", ErrEmptyValue)
	}

	return nil
}
