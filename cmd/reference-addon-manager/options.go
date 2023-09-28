package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type options struct {
	DeleteLabel            string
	EnableLeaderElection   bool
	EnableMetricsRecorder  bool
	MetricsAddr            string
	MetricsCertDir         string
	Namespace              string
	OperatorName           string
	ParameterSecretname    string
	PprofAddr              string
	ProbeAddr              string
	AddonInstanceName      string
	AddonInstanceNamespace string
	HeartbeatInterval      time.Duration
	Zap                    zap.Options
}

func (o *options) Process() error {
	o.processFlags()
	o.processSecrets()
	o.applyValuesFromOptions()

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
		&o.MetricsCertDir,
		"metrics-cert-dir",
		o.MetricsCertDir,
		strings.Join([]string{
			"The directory containing the TLS certificate (tls.crt) and key (tls.key) for secure metrics serviing.",
			"If unset metrics will be served without TLS.",
		}, " "),
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
		&o.AddonInstanceName,
		"addon-instance-name",
		o.AddonInstanceName,
		"The name of addon instance operator.",
	)

	flags.StringVar(
		&o.AddonInstanceNamespace,
		"addon-instance-namespace",
		o.AddonInstanceNamespace,
		"The namespace addon instance exists in.",
	)

	flags.DurationVar(
		&o.HeartbeatInterval,
		"heartbeat-interval",
		o.HeartbeatInterval,
		"Time between heartbeats sent to addon instance",
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

func (o *options) applyValuesFromOptions() {
	if o.AddonInstanceNamespace == "" {
		o.AddonInstanceNamespace = o.Namespace
	}
}

var ErrEmptyValue = errors.New("empty value")

func (o *options) validate() error {
	if o.Namespace == "" {
		return fmt.Errorf("validating namespace: %w", ErrEmptyValue)
	}

	return nil
}
