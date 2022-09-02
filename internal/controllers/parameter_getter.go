package controllers

import (
	"context"
	"fmt"

	"github.com/openshift/reference-addon/internal/controllers/phase"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ParameterGetter interface {
	GetParameters(ctx context.Context) (phase.RequestParameters, error)
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
	sizeParameterID = "size"
)

func (s *SecretParameterGetter) GetParameters(ctx context.Context) (phase.RequestParameters, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	key := client.ObjectKey{
		Namespace: s.cfg.Namespace,
		Name:      s.cfg.Name,
	}

	var params phase.RequestParameters

	var secret corev1.Secret

	if err := s.client.Get(ctx, key, &secret); err != nil {
		return params, fmt.Errorf("retrieving addon parameters secret: %w", err)
	}

	if val, ok := secret.Data[sizeParameterID]; ok {
		params.Size = string(val)
	}

	return params, nil
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
