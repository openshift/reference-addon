package main

import (
	"context"
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

func main() {
	opts := options{
		DeleteLabel:           "api.openshift.com/addon-reference-addon-delete",
		EnableMetricsRecorder: true,
		MetricsAddr:           ":8080",
		OperatorName:          "reference-addon",
		ParameterSecretname:   "addon-reference-addon-parameters",
		ProbeAddr:             ":8081",
	}

	if err := opts.Process(); err != nil {
		setupLog.Error(err, "processing options")
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
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
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// -----
	// PPROF
	// -----
	if len(opts.PprofAddr) > 0 {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		s := &http.Server{Addr: opts.PprofAddr, Handler: mux}
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
			controllers.WithNamespace(opts.Namespace),
			controllers.WithName(opts.ParameterSecretname),
		),
		controllers.WithLog{Log: ctrl.Log.WithName("controllers").WithName("ReferenceAddon")},
		controllers.WithAddonNamespace(opts.Namespace),
		controllers.WithAddonParameterSecretName(opts.ParameterSecretname),
		controllers.WithOperatorName(opts.OperatorName),
		controllers.WithDeleteLabel(opts.DeleteLabel),
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
