package main

import (
	"context"

	"fmt"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"github.com/openshift/reference-addon/internal/referenceaddoninteractor"
	addoninstanceclientgosdk "github.com/openshift/reference-addon/internal/addoninstancesdk/clientgosdk"
	"go.uber.org/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}

	logger := zapr.NewLogger(zapLog)

	kubeConfigAbsolutePath := "<your-kubeconfig-path>" // for building from in-cluster config and not kube-config path, set it as ""
	
	// the following interactor is being implemented by the tenants which implement relevant methods which will be later used by the heartbeatreporter (provided by the SDK) to CRUD on AddonInstance's Status
	interactor, err := referenceaddoninteractor.InitializeReferenceAddonInteractorSingleton(kubeConfigAbsolutePath)
	if err != nil {
		logger.Error(err, "unable to setup AddonInstance Interactor")
		os.Exit(1)
	}
	logger.Info("reference-addon interactor setup successfully!")

	// the following heartbeat reporter is the one which is (going to be) provided by the SDK. If the tenant doesn't like it, they can choose to implement their heartbeatReporter by making it implement the addoninstanceclientgosdk.AddonInstanceStatusReporterClient interface
	heartbeatReporter, err := addoninstanceclientgosdk.InitializeAddonInstanceHeartbeatReporterSingleton(interactor, "reference-addon", "reference-addon")
	if err != nil {
		logger.Error(err, "unable to initialize heartbeat reporter")
		os.Exit(1)
	}
	logger.Info("heartbeat-reporter setup successfully")

	go func() {
		if err := heartbeatReporter.Start(context.TODO()); err != nil {
			logger.Error(err, "unable to start heartbeat reporter")
			os.Exit(1)
		}
		logger.Info("received stop signal!")
	}()

	logger.Info("latest Heartbeat condition: ", "condition", heartbeatReporter.LatestCondition())

	time.Sleep(2 * time.Second)
	logger.Info("dummy Heartbeat Initiated")
	if err := dummySendHeartbeat(); err != nil {
		logger.Error(err, "failed to send dummy heartbeat")
	}

	logger.Info("dummy heartbeat sent succcessfully")
	logger.Info("latest Heartbeat condition: ", "condition", heartbeatReporter.LatestCondition())

	logger.Info("giving 30 seconds for heartbeat reporter to work the way it should...")
	time.Sleep(30 * time.Second)

	logger.Info("stopping the reporter for 30 seconds!")
	if err := heartbeatReporter.Stop(); err != nil {
		logger.Error(err, "unable to stop heartbeat reporter")
		os.Exit(1)
	}
	logger.Info("stopped")
	time.Sleep(30 * time.Second)

	logger.Info("starting the reporter again")

	go func() {
		if err := heartbeatReporter.Start(context.TODO()); err != nil {
			logger.Error(err, "unable to start heartbeat reporter: %w")
			os.Exit(1)
		}
		logger.Info("stop-signal returned again!")
	}()

	time.Sleep(15 * time.Second)
	logger.Info("cleaning up...")
	if err := heartbeatReporter.Stop(); err != nil {
		logger.Error(err, "unable to stop heartbeat reporter")
		os.Exit(1)
	}
	time.Sleep(1*time.Second)
	logger.Info("good bye!")
}

func dummySendHeartbeat() error {
	heartbeatReporter, err := addoninstanceclientgosdk.GetAddonInstanceHeartbeatReporterSingleton()
	if err != nil {
		return err
	}

	err = heartbeatReporter.SendHeartbeat(context.TODO(), metav1.Condition{
		Type:    "addons.managed.openshift.io/Healthy",
		Status:  "True",
		Reason:  "CustomReason",
		Message: "Foo-all-great-bar",
	})
	if err != nil {
		return fmt.Errorf("error occurred while updating the addon instance's status: %v", err)
	}
	return nil
}
