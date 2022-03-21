# Status Reporting with AddonInstance API

## Introduction

We provide a dedicated AddonInstance API which has multiple use-cases but one of the more substantial use-cases is to deliver an addon's status/health via it.

Whenever AddonOperator gets installed on an OSD cluster (at the time of its bootstrap itself), it sets up the [`AddonInstance` CRD](https://raw.githubusercontent.com/openshift/addon-operator/main/config/deploy/addons.managed.openshift.io_addoninstances.yaml) as well.

## Purpose

For every `Addon` CR which gets created, an `AddonInstance` CR, with the name "addon-instance", gets created in that addon's target namespace.

For example, upon applying the following Addon CR for reference-addon:

```yaml
apiVersion: addons.managed.openshift.io/v1alpha1
kind: Addon
metadata:
  name: reference-addon
spec:
  displayName: An amazing example addon!
  namespaces:
  - name: reference-addon
  install:
    type: OLMOwnNamespace
    olmOwnNamespace:
      namespace: reference-addon
      packageName: reference-addon
      channel: alpha
      catalogSourceImage: quay.io/osd-addons/reference-addon-index@sha256:58cb1c4478a150dc44e6c179d709726516d84db46e4e130a5227d8b76456b5bd
  upgradePolicy:
    id: 123-456-789
  monitoring:
    federation:
      namespace: "reference-addon"
      matchNames:
      - reference_addon_foos_per_second
      matchLabels:
        prometheus: reference-addon
```

Automatically, an AddonInstance object will be created in the namespace "reference-addon" (targetNamespace of reference-addon):

```sh
❯ kubectl get addoninstance -n reference-addon
NAME             LAST HEARTBEAT   AGE
addon-instance                    7s
```
And this AddonInstance CR would look like this:

```yaml
apiVersion: addons.managed.openshift.io/v1alpha1
kind: AddonInstance
metadata:
  name: addon-instance
  namespace: reference-addon
spec:
  heartbeatUpdatePeriod: 10s
```

## Idea behind status reporting

Any Addon is expected to report its health/status to its corresponding AddonInstance CR periodically. These reported statuses are perceived as heartbeats coming from the addon, meaning that if the addon fails to report for a certain amount of time, it would be considered dead.

**How does a status/heartbeat look in this case?**

Heartbeat = The `metav1.Condition` object with the `type: addons.managed.openshift.io/Healthy`, under the `.status.conditions` of the AddonInstance CR.
```go
metav1.Condition{
  Type: "addons.managed.openshift.io/Healthy",
  Status: "False",
  Reason: "DiskCritical",
  Message: "Disk usage greater than 85%",
}
```

Hence, after reporting the above heartbeat (metav1.Condition), the respective AddonInstance would look like:

```yaml
apiVersion: addons.managed.openshift.io/v1alpha1
kind: AddonInstance
metadata:
  name: addon-instance
  namespace: reference-addon
spec:
  heartbeatUpdatePeriod: 10s
status:
  conditions:
  - type: "addons.managed.openshift.io/Healthy"
    status: "False"
    reason: "DiskCritical"
    message: "Disk usage greater than 85%"
  lastHeartbeatTime: "<timestamp of the moment when the above Condition with the type: addons.managed.openshift.io/Healthy (heartbeat) was reported>"
```

The AddonOperator periodically would check the `AddonInstance` CR and determine everytime how long it's been since the last heartbeat was reported (via `status.lastHeartbeatTime`) and accordingly, it would decide whether to mark the Addon as dead or not.

Marking an addon as dead is, basically, setting the following condition to the Addon's AddonInstance CR:
```yaml
type: "addons.managed.openshift.io/Healthy",
status: "Unknown",
reason: "HeartbeatTimeout",
message: "Addon failed to send heartbeat.",
```

But don't worry, you don't have to implement this entire functionality of periodically reporting heartbeats/statuses from scratch.
That's where our AddonInstance SDK comes into the picture.

## Addon SDK

Addon SDK is a package which you can utilise to easily integrate AddonInstance status reporting with ease and without much hassle. It sets up automatic periodic heartbeat reporter loops, provides support for stopping/restarting those heartbeat reporters at your own wish, sending manual heartbeats in real-time along with synchronising the heartbeat reporter loop with this newly sent heartbeat so that it proceeds to periodically report this new heartbeat from then on.

All you have to do is meet some prerequisites.

## Integrating Addon SDK to your Addon

### Achieving the prerequisites

* *go get the Addon SDK package*

* Give your addon the RBAC permissions to GET `addoninstance` and UPDATE `addoninstance/status` in their targetNamespace.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: reference-addon-addoninstance-role
  namespace: reference-addon ## target namespace
- apiGroups:
  - "addons.managed.openshift.io"
  resources:
  - addoninstances/status
  verbs:
  - get
  - list
  - watch
  - update
  - patch
- apiGroups:
  - "addons.managed.openshift.io"
  resources:
  - addoninstances
  verbs:
  - get
  - list
  - watch
```

* Define a type which implements the `addonsdk.client` interface
```go
type client interface {
	// the following GetAddonInstance method should be backed by a cache
	GetAddonInstance(ctx context.Context, key types.NamespacedName, addonInstance *addonsv1alpha1.AddonInstance) error
	UpdateAddonInstanceStatus(ctx context.Context, addonInstance *addonsv1alpha1.AddonInstance) error
}
```
An object of this type will be used by the AddonSDK to GET any relevant addoninstance resource and UPDATE the status (report the heartbeat) for an AddonInstance resource.
> The AddonSDK is made to be entirely independent of client-go and controller-runtime so as to ensure that any user/client/tenant can integrate this AddonSDK without being forced to install a certain version of client-go/controller-runtime.

To get an example of such a `type`, checkout `cmd/reference-addon-manager/addonsdkclient.go` in reference-addon to see how an implementation of `addonsdk.client` interface for a controller-runtime-compliant operator/addon looks.

### Setting up the heartbeat reporter
In the entrypoint (where `main()` lies) of your addon:
* Instantiate an object of the `type` you just defined in the prerequisites, which implements the `addonsdk.client` interface.
```go
// ref: `cmd/reference-addon-manager/addonsdkclient.go`
addonSdkClient := NewAddonSDKClient(mgr.GetClient())
```
* Using this object and other addon-related details as inputs, instantiate an object of the StatusReporter through the `addonsdk` package.
```go
statusReporter := addonsdk.SetupStatusReporter(addonSdkClient, addonName, addonNamespace, ctrl.Log.WithName("StatusReporter"))

if err != nil {
    setupLog.Error(err, "unable to setup status-reporter")
    os.Exit(1)
}
```
* Start the status reporter alongside starting your operator/addon:
```go
if err := mgr.Add(statusReporter); err != nil {
    setupLog.Error(err, "unable to add status-reporter to manager")
    os.Exit(1)
}
...
...
setupLog.Info("starting manager")
// starts the operator as well as the `statusReporter`
if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
    setupLog.Error(err, "problem running manager")
    os.Exit(1)
}
```

This will start reporting heartbeats/statuses automatically at a periodic rate expected off the respective `AddonInstance` CR (AddonInstance's `.spec.heartbeatUpdatePeriod` seconds).

### Reporting ad-hoc heartbeats in real-time

You would want to report heartbeat/status from different sections of your Addon's codebase.

For achieving that, all you have to do is invoke the `statusReporter.SetConditions(context, conditionsToReport)` method, where `statusReporter` is the object which you setup in the entrypoint of your addon's code via `addonsdk.SetupStatusReporter(...)`

Now, it's upto you how you access that `statusReporter` object var across your addon's codebase.

In the case of reference-addon, we modified the `ReferenceAddonReconciler` (reference-addon's core controller) struct itself to hold a reference to the `statusReporter` object:
```go
type ReferenceAddonReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
    StatusReporter *addonsdk.StatusReporter // here
}
```
and we instantiate the ReferenceAddonReconciler object (with a reference to the statusReporter object) in the entrypoint like this:
```go
	// the following section hooks up a heartbeat reporter with the current addon/operator
	r := controllers.ReferenceAddonReconciler{
		Client:         mgr.GetClient(),
		Log:            ctrl.Log.WithName("controllers").WithName("ReferenceAddon"),
		StatusReporter: statusReporter, // look
	}

	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ReferenceAddon")
		os.Exit(1)
	}
```
We use this `r` variable across the codebase to report heartbeats by calling `r.StatusReporter.SetConditions(..., ...)`.

For example, under the `internal/controllers/reference_addon_controller.go`

```go
func (r *ReferenceAddonReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("referenceaddon", req.NamespacedName.String())

	successfulCondition := metav1.Condition{
		Type:    addonsdk.AddonHealthyConditionType,
		Status:  metav1.ConditionTrue,
		Reason:  "ReferenceAddonSpecProperty",
		Message: "spec.reportSuccessCondition found to be 'true'",
	}
	failureCondition := metav1.Condition{
		Type:    addonsdk.AddonHealthyConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  "ReferenceAddonSpecProperty",
		Message: "spec.reportSuccessCondition found to be 'false'",
	}
    ...
    ...
    ...
    conditionsToReport := []metav1.Condition{successfulCondition}
    if (somethingWeirdIsHappening) {
        conditionsToReport := []metav1.Condition{failureCondition}
    }

    // reporting the new heartbeat
	if err := r.StatusReporter.SetConditions(ctx, conditionsToReport); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
```

### Reporting AddonInstance .spec level changes

There might be situations when the `.spec` section of the AddonInstance CR corresponding to your Addon changes.

It can be any sort of change. For example, it can be an increase in `.spec.heartbeatUpdatePeriod` to make heartbeat reporting more relaxed because the existing one was hogging the cluster unnecessarily.

So, in that case, it's the Addon's responsibility to signal the `statusReporter` that the AddonInstance changed.

The `statusReporter.ReportAddonInstanceSpecChange(context, newAddonInstanceObj)` is for signalling that.

It's upto you how you implement a watcher which watches those kind of changes in the `AddonInstance` CR and accordingly signal them.
You can write your own reconciler or even informers for that matter.

For example, in reference-addon, we've written a controller (reconciler) which is meant to watch for `AddonInstance` spec-level changes in the AddonInstance CR with the name "addon-instance" and namespace "reference-addon". Whenever it captures a change in the AddonInstance's spec, it GETs that new AddonInstance object and calls `statusReporter.ReportAddonInstanceSpecChange(context, newAddonInstanceObj)` for signalling that.

The following piece of code ensures to call `Reconcile()` method whenever the AddonInstance spec change is observed or is Created (for capturing events like Informer's Cache Syncs):
```go
func (r *AddonInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	addonInstanceSpecChangePredicate := predicate.Funcs{
		// capture updates to the .spec
		UpdateFunc: func(e event.UpdateEvent) bool {
			// ignore updates to .status in which case metadata.generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		// capture CREATE events too for moments like Cache-syncs
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		// no need to capture DELETE or any other event
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&addonsv1alpha1.AddonInstance{}, builder.WithPredicates(addonInstanceConfigurationChangePredicate)).
		Complete(r)
}
```

```go
func (r *AddonInstanceReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Ignore if the AddonInstance who .spec changed doesn't have the name "addon-instance"
	if req.NamespacedName.Name != "addon-instance" {
		return ctrl.Result{}, nil
	}

  	// GET that latest AddonInstance for which we just observed a change
	newAddonInstance := &addonsv1alpha1.AddonInstance{}
	if err := r.Get(ctx, req.NamespacedName, newAddonInstance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Log.Info("reporting addon-instance spec change to status-reporter")

  	// Report this new AddonInstance spec change to the status
	if err := r.StatusReporter.ReportAddonInstanceSpecChange(ctx, *newAddonInstance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
```
