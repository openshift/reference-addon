package main

import (
	"context"
	"flag"
	"net/http"
	"net/http/pprof"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	refapis "github.com/openshift/reference-addon/apis"
	"github.com/openshift/reference-addon/internal/addonsdk"
	"github.com/openshift/reference-addon/internal/controllers"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = refapis.AddToScheme(scheme)
}

func main() {
	var (
		metricsAddr          string
		pprofAddr            string
		enableLeaderElection bool
	)
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&pprofAddr, "pprof-addr", "", "The address the pprof web endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         metricsAddr,
		Port:                       9443,
		LeaderElectionResourceLock: "leases",
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "8a4hp84a6s.addon-operator-lock",
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

	// fetching the following from the Downwards Api in case of reference-addon.
	// Can be hardcoded or supplied in any other way too depending on the addon developers.
	addonName := os.Getenv("ADDON_NAME")
	addonNamespace := os.Getenv("ADDON_NAMESPACE")

	// Setup the StatusReporter
	addonSdkClient := NewAddonSDKClient(mgr.GetClient())
	statusReporter, err := addonsdk.InitializeStatusReporterSingleton(addonSdkClient, addonName, addonNamespace)
	if err != nil {
		setupLog.Error(err, "unable to setup StatusReporter")
		os.Exit(1)
	}

	if err := mgr.Add(statusReporter); err != nil {
		setupLog.Error(err, "unable to start status-reporter")
		os.Exit(1)
	}

	// the following section hooks up a heartbeat reporter with the current addon/operator
	r := controllers.ReferenceAddonReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("ReferenceAddon"),
		Scheme:         mgr.GetScheme(),
		StatusReporter: statusReporter,
	}

	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ReferenceAddon")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
