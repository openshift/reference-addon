package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/openshift/reference-addon/internal/controllers/phase"
	"go.uber.org/multierr"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPhaseApplyNetworkPolicies(client NetworkPolicyClient, opts ...PhaseApplyNetworkPoliciesOption) *PhaseApplyNetworkPolicies {
	var cfg PhaseApplyNetworkPoliciesConfig

	cfg.Option(opts...)
	cfg.Default()

	return &PhaseApplyNetworkPolicies{
		cfg: cfg,

		client: client,
	}
}

type PhaseApplyNetworkPolicies struct {
	cfg PhaseApplyNetworkPoliciesConfig

	client NetworkPolicyClient
}

func (p *PhaseApplyNetworkPolicies) Execute(ctx context.Context, req phase.Request) phase.Result {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	applyNetworkPolicies, ok := req.Params.GetApplyNetworkPolicies()
	if !ok {
		return phase.Success()
	}

	if !applyNetworkPolicies {
		return p.ensureNetworkPoliciesRemoved(ctx)
	}

	return p.ensureNetworkPoliciesApplied(ctx)
}

func (p *PhaseApplyNetworkPolicies) ensureNetworkPoliciesRemoved(ctx context.Context) phase.Result {
	p.cfg.Log.Info("removing NetworkPolicies", "count", len(p.cfg.Policies))

	if err := p.client.RemoveNetworkPolicies(ctx, p.cfg.Policies...); err != nil {
		return phase.Error(fmt.Errorf("deleting NetworkPolicies: %w", err))
	}

	p.cfg.Log.Info("successfully removed NetworkPolicies", "count", len(p.cfg.Policies))

	return phase.Success()
}

func (p *PhaseApplyNetworkPolicies) ensureNetworkPoliciesApplied(ctx context.Context) phase.Result {
	p.cfg.Log.Info("applying NetworkPolicies", "count", len(p.cfg.Policies))

	if err := p.client.ApplyNetworkPolicies(ctx, p.cfg.Policies...); err != nil {
		return phase.Error(fmt.Errorf("applying NetworkPolicies: %w", err))
	}

	p.cfg.Log.Info("successfully applied NetworkPolicies", "count", len(p.cfg.Policies))

	return phase.Success()
}

type PhaseApplyNetworkPoliciesConfig struct {
	Log logr.Logger

	Policies []netv1.NetworkPolicy
}

func (c *PhaseApplyNetworkPoliciesConfig) Option(opts ...PhaseApplyNetworkPoliciesOption) {
	for _, opt := range opts {
		opt.ConfigurePhaseApplyNetworkPolicies(c)
	}
}

func (c *PhaseApplyNetworkPoliciesConfig) Default() {
	if c.Log == nil {
		c.Log = logr.Discard()
	}
}

type PhaseApplyNetworkPoliciesOption interface {
	ConfigurePhaseApplyNetworkPolicies(*PhaseApplyNetworkPoliciesConfig)
}

type NetworkPolicyClient interface {
	ApplyNetworkPolicies(ctx context.Context, policies ...netv1.NetworkPolicy) error
	RemoveNetworkPolicies(ctx context.Context, policies ...netv1.NetworkPolicy) error
}

func NewNetworkPolicyClientImpl(client client.Client) *NetworkPolicyClientImpl {
	return &NetworkPolicyClientImpl{
		client: client,
	}
}

type NetworkPolicyClientImpl struct {
	client client.Client
}

func (c *NetworkPolicyClientImpl) ApplyNetworkPolicies(ctx context.Context, policies ...netv1.NetworkPolicy) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var finalErr error

	for _, policy := range policies {
		if err := c.createOrUpdatePolicy(ctx, policy); err != nil {
			multierr.AppendInto(&finalErr, fmt.Errorf("creating/updating NetworkPolicy %q: %w", policy.Name, err))
		}
	}

	return finalErr
}

func (c *NetworkPolicyClientImpl) createOrUpdatePolicy(ctx context.Context, policy netv1.NetworkPolicy) error {
	actualPolicy := &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policy.Name,
			Namespace: policy.Namespace,
		},
	}

	_, err := ctrl.CreateOrUpdate(ctx, c.client, actualPolicy, func() error {
		actualPolicy.Labels = labels.Merge(actualPolicy.Labels, policy.Labels)
		actualPolicy.Spec = policy.Spec

		return nil
	})

	return err
}

func (c *NetworkPolicyClientImpl) RemoveNetworkPolicies(ctx context.Context, policies ...netv1.NetworkPolicy) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var finalErr error

	for _, policy := range policies {
		if err := c.client.Delete(ctx, &policy); err != nil && !errors.IsNotFound(err) {
			multierr.AppendInto(&finalErr, fmt.Errorf("deleting NetworkPolicy %q: %w", policy.Name, err))
		}
	}

	return finalErr
}
