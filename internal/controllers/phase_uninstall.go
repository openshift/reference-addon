package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/openshift/reference-addon/internal/controllers/phase"
	opsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPhaseUninstall(signaler UninstallSignaler, uninstaller Uninstaller, opts ...PhaseUninstallOption) *PhaseUninstall {
	var cfg PhaseUninstallConfig

	cfg.Option(opts...)
	cfg.Default()

	return &PhaseUninstall{
		cfg: cfg,

		signaler:    signaler,
		uninstaller: uninstaller,
	}
}

type PhaseUninstall struct {
	cfg PhaseUninstallConfig

	signaler    UninstallSignaler
	uninstaller Uninstaller
}

func (p *PhaseUninstall) Execute(ctx context.Context, req phase.Request) phase.Result {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !p.signaler.SignalUninstall(ctx) {
		return phase.Success()
	}

	if err := p.uninstaller.Uninstall(ctx, p.cfg.AddonNamespace, p.cfg.OperatorName); err != nil {
		return phase.Error(fmt.Errorf("uninstalling addon: %w", err))
	}

	return phase.Blocking()
}

type PhaseUninstallConfig struct {
	Log logr.Logger

	AddonNamespace string
	OperatorName   string
}

func (c *PhaseUninstallConfig) Option(opts ...PhaseUninstallOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseUninstall(c)
	}
}

func (c *PhaseUninstallConfig) Default() {
	if c.Log == nil {
		c.Log = logr.Discard()
	}
}

type PhaseUninstallOption interface {
	ConfigurePhaseUninstall(*PhaseUninstallConfig)
}

type Uninstaller interface {
	Uninstall(ctx context.Context, namespace, operatorName string) error
}

func NewUninstallerImpl(client CSVClient, opts ...UninstallerImplOption) *UninstallerImpl {
	var cfg UninstallerImplConfig

	cfg.Option(opts...)
	cfg.Default()

	return &UninstallerImpl{
		cfg: cfg,

		client: client,
	}
}

type UninstallerImpl struct {
	cfg UninstallerImplConfig

	client CSVClient
}

func (u UninstallerImpl) Uninstall(ctx context.Context, namespace, operatorName string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	csvs, err := u.client.ListCSVs(ctx, WithNamespace(namespace), WithPrefix(operatorName))
	if err != nil {
		return fmt.Errorf("listing ClusterServiceVersions with name %q: %w", operatorName, err)
	}

	if err := u.client.RemoveCSVs(ctx, csvs...); err != nil {
		return fmt.Errorf("removing csvs: %w", err)
	}

	return nil
}

type UninstallerImplConfig struct {
	Log logr.Logger
}

func (c *UninstallerImplConfig) Option(opts ...UninstallerImplOption) {
	for _, opt := range opts {
		opt.ConfigureUninstallerImpl(c)
	}
}

func (c *UninstallerImplConfig) Default() {
	if c.Log == nil {
		c.Log = logr.Discard()
	}
}

type UninstallerImplOption interface {
	ConfigureUninstallerImpl(*UninstallerImplConfig)
}

type CSVClient interface {
	ListCSVs(ctx context.Context, opts ...ListCSVsOption) ([]opsv1alpha1.ClusterServiceVersion, error)
	RemoveCSVs(ctx context.Context, csvs ...opsv1alpha1.ClusterServiceVersion) error
}

type ListCSVsConfig struct {
	Namespace string
	Prefix    string
}

func (c *ListCSVsConfig) Option(opts ...ListCSVsOption) {
	for _, opt := range opts {
		opt.ConfigureListCSVs(c)
	}
}

type ListCSVsOption interface {
	ConfigureListCSVs(*ListCSVsConfig)
}

func NewCSVClientImpl(client client.Client, opts ...CSVClientOption) *CSVClientImpl {
	var cfg CSVClientImplConfig

	cfg.Option(opts...)
	cfg.Default()

	return &CSVClientImpl{
		cfg: cfg,

		client: client,
	}
}

type CSVClientImpl struct {
	cfg CSVClientImplConfig

	client client.Client
}

func (c *CSVClientImpl) ListCSVs(ctx context.Context, opts ...ListCSVsOption) ([]opsv1alpha1.ClusterServiceVersion, error) {
	var cfg ListCSVsConfig

	cfg.Option(opts...)

	var listOptions []client.ListOption

	if cfg.Namespace != "" {
		listOptions = append(listOptions, client.InNamespace(cfg.Namespace))
	}

	var csvs opsv1alpha1.ClusterServiceVersionList

	if err := c.client.List(ctx, &csvs, listOptions...); err != nil {
		return nil, fmt.Errorf("listing ClusterServiceVersions: %w", err)
	}

	var res []opsv1alpha1.ClusterServiceVersion

	for _, csv := range csvs.Items {
		if !strings.HasPrefix(csv.Name, cfg.Prefix) {
			continue
		}

		res = append(res, csv)
	}

	return res, nil
}

func (c *CSVClientImpl) RemoveCSVs(ctx context.Context, csvs ...opsv1alpha1.ClusterServiceVersion) error {
	var finalErr error

	for _, csv := range csvs {
		c.cfg.Log.Info("attempting to delete 'ClusterServiceVersion'")

		if err := c.client.Delete(ctx, &csv); err != nil && !errors.IsNotFound(err) {
			c.cfg.Log.Error(err, "failed to delete 'ClusterServiceVersion'")

			multierr.AppendInto(&finalErr, fmt.Errorf("deleting CSV %q: %w", csv.Name, err))
		} else {
			c.cfg.Log.Info("successfully deleted 'ClusterServiceVersion'")
		}
	}

	return finalErr
}

type CSVClientImplConfig struct {
	Log logr.Logger
}

func (c *CSVClientImplConfig) Option(opts ...CSVClientOption) {
	for _, opt := range opts {
		opt.ConfigureCSVClientImpl(c)
	}
}

func (c *CSVClientImplConfig) Default() {
	if c.Log == nil {
		c.Log = logr.Discard()
	}
}

type CSVClientOption interface {
	ConfigureCSVClientImpl(*CSVClientImplConfig)
}

type UninstallSignaler interface {
	SignalUninstall(ctx context.Context) bool
}

func NewConfigMapUninstallSignaler(client client.Client, opts ...ConfigMapUninstallSignalerOption) (*ConfigMapUninstallSignaler, error) {
	var cfg ConfigMapUninstallSignalerConfig

	cfg.Option(opts...)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &ConfigMapUninstallSignaler{
		cfg: cfg,

		client: client,
	}, nil
}

type ConfigMapUninstallSignaler struct {
	cfg ConfigMapUninstallSignalerConfig

	client client.Client
}

func (s *ConfigMapUninstallSignaler) SignalUninstall(ctx context.Context) bool {
	tgt := types.NamespacedName{
		Namespace: s.cfg.AddonNamespace,
		Name:      s.cfg.OperatorName,
	}

	var cm corev1.ConfigMap

	if err := s.client.Get(ctx, tgt, &cm); err != nil {
		return false
	}

	_, ok := cm.Labels[s.cfg.DeleteLabel]

	return ok
}

type ConfigMapUninstallSignalerConfig struct {
	AddonNamespace string
	OperatorName   string
	DeleteLabel    string
}

func (c *ConfigMapUninstallSignalerConfig) Option(opts ...ConfigMapUninstallSignalerOption) {
	for _, opt := range opts {
		opt.ConfigureConfigMapUninstallSignaler(c)
	}
}

func (c *ConfigMapUninstallSignalerConfig) Validate() error {
	var finalErr error

	if err := validateOptionValue(c.AddonNamespace); err != nil {
		multierr.AppendInto(&finalErr, fmt.Errorf("validating AddonNamespace: %w", err))
	}

	if err := validateOptionValue(c.OperatorName); err != nil {
		multierr.AppendInto(&finalErr, fmt.Errorf("validating OperatorName: %w", err))
	}

	if err := validateOptionValue(c.DeleteLabel); err != nil {
		multierr.AppendInto(&finalErr, fmt.Errorf("validating DeleteLabel: %w", err))
	}

	return finalErr
}

type ConfigMapUninstallSignalerOption interface {
	ConfigureConfigMapUninstallSignaler(*ConfigMapUninstallSignalerConfig)
}
