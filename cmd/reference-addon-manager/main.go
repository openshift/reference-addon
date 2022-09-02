package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	refapis "github.com/openshift/reference-addon/apis"
	"github.com/openshift/reference-addon/internal/controllers"
	"github.com/openshift/reference-addon/internal/metrics"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = refapis.AddToScheme(scheme)
	_ = opsv1alpha1.AddToScheme(scheme)

	// Register metrics with the global Prometheus registry
	metrics.RegisterMetrics()
}

const (
	addonNamespace           = "redhat-reference-addon"
	operatorName             = "reference-addon"
	deleteLabel              = "api.openshift.com/addon-reference-addon-delete"
	addonParameterSecretname = "addon-reference-addon-parameters"
)

func main() {
	var (
		metricsAddr           string
		pprofAddr             string
		probeAddr             string
		enableLeaderElection  bool
		enableMetricsRecorder bool
	)
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&pprofAddr, "pprof-addr", "", "The address the pprof web endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	flag.BoolVar(&enableMetricsRecorder, "enable-metrics-recorder", true, "Enable recording Addon Metrics")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         metricsAddr,
		HealthProbeBindAddress:     probeAddr,
		Port:                       9443,
		LeaderElectionResourceLock: "leases",
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "8a4hp84a6s.addon-operator-lock",
		Namespace:                  addonNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// -----
	// PPROF
	// -----
	if len(pprofAddr) > 0 {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		s := &http.Server{Addr: pprofAddr, Handler: mux}
		err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
			errCh := make(chan error)
			defer func() {
				for range errCh {
				} // drain errCh for GC
			}()
			go func() {
				defer close(errCh)
				errCh <- s.ListenAndServe()
			}()

			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				s.Close()
				return nil
			}
		}))
		if err != nil {
			setupLog.Error(err, "unable to create pprof server")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(fmt.Errorf("unable to set up health check: %w", err), "setting up manager")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(fmt.Errorf("unable to set up ready check: %w", err), "setting up manager")
		os.Exit(1)
	}

	client := mgr.GetClient()

	// the following section hooks up a heartbeat reporter with the current addon/operator
	r, err := controllers.NewReferenceAddonReconciler(
		client,
		controllers.NewSecretParameterGetter(
			client,
			controllers.WithNamespace(addonNamespace),
			controllers.WithName(addonParameterSecretname),
		),
		controllers.WithLog{Log: ctrl.Log.WithName("controllers").WithName("ReferenceAddon")},
		controllers.WithAddonNamespace(addonNamespace),
		controllers.WithAddonParameterSecretName(addonParameterSecretname),
		controllers.WithOperatorName(operatorName),
		controllers.WithDeleteLabel(deleteLabel),
	)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ReferenceAddon")
		os.Exit(1)
	}

	if err := r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup controller", "controller", "ReferenceAddon")
		os.Exit(1)
	}

	// ensure at least one data point for sample metrics
	sampler := metrics.NewResponseSamplerImpl()
	sampler.RequestSampleResponseData("https://httpstat.us/503", "https://httpstat.us/200")

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
