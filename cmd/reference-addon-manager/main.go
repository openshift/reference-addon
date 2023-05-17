package main

import (
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/go-logr/logr"
	av1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	refapis "github.com/openshift/reference-addon/apis"
	ractrl "github.com/openshift/reference-addon/internal/controllers/referenceaddon"
	"github.com/openshift/reference-addon/internal/controllers/status"
	"github.com/openshift/reference-addon/internal/metrics"
	"github.com/openshift/reference-addon/internal/pprof"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

func main() {
	opts := options{
		DeleteLabel:           "api.openshift.com/addon-reference-addon-delete",
		EnableMetricsRecorder: true,
		MetricsAddr:           ":8080",
		OperatorName:          "reference-addon",
		ParameterSecretname:   "addon-reference-addon-parameters",
		ProbeAddr:             ":8081",
		AddonInstanceName:     "addon-instance",
		HeartbeatInterval:     10 * time.Second,
		Zap: zap.Options{
			Development: true,
		},
	}
	if err := opts.Process(); err != nil {
		fmt.Fprintf(os.Stdout, "Unexpected error occurred while processing options: %v\n", err)

		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts.Zap)))

	log := ctrl.Log.WithName("setup")

	log.Info("Setting Up Manager")

	mgr, err := setupManager(log, opts)
	if err != nil {
		fail(log, err, "setting up manager")
	}

	log.Info("Starting Manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fail(log, err, "running manager")
	}
}

func setupManager(log logr.Logger, opts options) (ctrl.Manager, error) {
	log.Info("Registering Metrics")

	if err := metrics.RegisterMetrics(ctrlmetrics.Registry); err != nil {
		return nil, fmt.Errorf("registering metrics: %w", err)
	}

	log.Info("Setting Up Scheme")

	scheme, err := initializeScheme()
	if err != nil {
		return nil, fmt.Errorf("initializing scheme: %w", err)
	}

	log.Info("Initializing Manager")

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("getting config for cluster: %w", err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         opts.MetricsAddr,
		HealthProbeBindAddress:     opts.ProbeAddr,
		Port:                       9443,
		LeaderElectionResourceLock: "leases",
		LeaderElection:             opts.EnableLeaderElection,
		LeaderElectionID:           "8a4hp84a6s.addon-operator-lock",
		Namespace:                  opts.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing manager: %w", err)
	}

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return nil, fmt.Errorf("adding healthz check to manager: %w", err)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		return nil, fmt.Errorf("adding readyz check to manager: %w", err)
	}

	if opts.PprofAddr != "" {
		log.Info("Initializing Pprof")

		if err := mgr.Add(pprof.NewServer(opts.PprofAddr)); err != nil {
			return nil, fmt.Errorf("adding pprof server to manager: %w", err)
		}
	}

	log.Info("Initializing Controllers")

	client := mgr.GetClient()

	r, err := ractrl.NewReferenceAddonReconciler(
		client,
		ractrl.NewSecretParameterGetter(
			client,
			ractrl.WithNamespace(opts.Namespace),
			ractrl.WithName(opts.ParameterSecretname),
		),
		ractrl.WithLog{Log: ctrl.Log.WithName("controller").WithName("referenceaddon")},
		ractrl.WithAddonNamespace(opts.Namespace),
		ractrl.WithAddonParameterSecretName(opts.ParameterSecretname),
		ractrl.WithOperatorName(opts.OperatorName),
		ractrl.WithDeleteLabel(opts.DeleteLabel),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing reference addon controller: %w", err)
	}

	if err := r.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("setting up reference addon controller: %w", err)
	}

	statusctlr, err := status.NewStatusControllerReconciler(
		client,
		status.WithLog{Log: ctrl.Log.WithName("controller").WithName("status")},
		status.WithAddonInstanceNamespace(opts.AddonInstanceNamespace),
		status.WithAddonInstanceName(opts.AddonInstanceName),
		status.WithReferenceAddonNamespace(opts.Namespace),
		status.WithReferenceAddonName(opts.OperatorName),
		status.WithHeartbeatInterval(opts.HeartbeatInterval),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing status controller: %w", err)
	}

	if err := statusctlr.SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("setting up status controller: %w", err)
	}

	return mgr, nil
}

func initializeScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding client-go APIs to scheme: %w", err)
	}

	if err := refapis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding Reference Addon APIs to scheme: %w", err)
	}

	if err := opsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding Operators v1alpha1 APIs to scheme :%w", err)
	}

	if err := av1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding addon-operator v1alpha1 APIs to scheme :%w", err)
	}

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding client-go APIs to scheme :%w", err)
	}

	return scheme, nil
}

func fail(log logr.Logger, err error, msg string) {
	log.Error(err, msg)

	os.Exit(1)
}
