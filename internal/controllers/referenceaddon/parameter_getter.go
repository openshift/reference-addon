package referenceaddon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ParameterGetter interface {
	GetParameters(ctx context.Context) (PhaseRequestParameters, error)
}

func NewSecretParameterGetter(client client.Client, opts ...SecretParameteterGetterOption) *SecretParameterGetter {
	var cfg SecretParameterGetterConfig

	cfg.Option(opts...)

	return &SecretParameterGetter{
		cfg: cfg,

		client: client,
	}
}

type SecretParameterGetter struct {
	cfg SecretParameterGetterConfig

	client client.Client
}

const (
	applyNetworkPoliciesID = "applynetworkpolicies"
	sizeParameterID        = "size"
)

func (s *SecretParameterGetter) GetParameters(ctx context.Context) (PhaseRequestParameters, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	key := client.ObjectKey{
		Namespace: s.cfg.Namespace,
		Name:      s.cfg.Name,
	}

	var secret corev1.Secret

	if err := s.client.Get(ctx, key, &secret); err != nil {
		return NewPhaseRequestParameters(), fmt.Errorf("retrieving addon parameters secret: %w", err)
	}

	var opts []PhaseRequestParametersOption

	if val, ok := secret.Data[applyNetworkPoliciesID]; ok {
		b, err := parseBool(string(val))
		if err != nil {
			return NewPhaseRequestParameters(), fmt.Errorf("parsing 'ApplyNetworkPolicies' value: %w", err)
		}

		opts = append(opts, WithApplyNetworkPolicies{Value: &b})
	}

	if val, ok := secret.Data[sizeParameterID]; ok {
		s := string(val)

		opts = append(opts, WithSize{Value: &s})
	}

	return NewPhaseRequestParameters(opts...), nil
}

type SecretParameterGetterConfig struct {
	Namespace string
	Name      string
}

func (c *SecretParameterGetterConfig) Option(opts ...SecretParameteterGetterOption) {
	for _, opt := range opts {
		opt.ConfigureSecretParameterGetter(c)
	}
}

type SecretParameteterGetterOption interface {
	ConfigureSecretParameterGetter(*SecretParameterGetterConfig)
}

var ErrInvalidBoolValue = errors.New("invalid bool value")

func parseBool(maybeBool string) (bool, error) {
	switch strings.ToLower(maybeBool) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, ErrInvalidBoolValue
	}
}
