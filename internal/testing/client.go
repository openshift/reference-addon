package testing

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTestClient(client client.Client) *TestClient {
	return &TestClient{
		client: client,
	}
}

type TestClient struct {
	client client.Client
}

func (c *TestClient) Create(ctx context.Context, obj client.Object, opts ...RequestOption) {
	if err := c.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); errors.IsNotFound(err) {
		ExpectWithOffset(1, c.client.Create(ctx, obj)).Should(Succeed())
		c.EventuallyObjectExists(ctx, obj, opts...)
	}
}

func (c *TestClient) Update(ctx context.Context, obj client.Object, opts ...RequestOption) {
	ExpectWithOffset(1, c.client.Update(ctx, obj)).Should(Succeed())
}

func (c *TestClient) Delete(ctx context.Context, obj client.Object, opts ...RequestOption) {
	ExpectWithOffset(1, client.IgnoreNotFound(c.client.Delete(ctx, obj))).Should(Succeed())
	c.EventuallyObjectDoesNotExist(ctx, obj, opts...)
}

func (c *TestClient) EventuallyObjectExists(ctx context.Context, obj client.Object, opts ...RequestOption) bool {
	var cfg RequestConfig

	cfg.Option(opts...)
	cfg.Default()

	get := func() error {
		return c.client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}

	return EventuallyWithOffset(1, get, fmt.Sprint(cfg.Timeout)).Should(Succeed())
}

func (c *TestClient) EventuallyObjectDoesNotExist(ctx context.Context, obj client.Object, opts ...RequestOption) bool {
	var cfg RequestConfig

	cfg.Option(opts...)
	cfg.Default()

	get := func() error {
		return c.client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}

	return EventuallyWithOffset(1, get, fmt.Sprint(cfg.Timeout)).ShouldNot(Succeed())
}

type RequestConfig struct {
	Timeout time.Duration
}

func (c *RequestConfig) Option(opts ...RequestOption) {
	for _, opt := range opts {
		opt.ConfigureRequest(c)
	}
}

func (c *RequestConfig) Default() {
	if c.Timeout == 0 {
		c.Timeout = 1 * time.Second
	}
}

type RequestOption interface {
	ConfigureRequest(*RequestConfig)
}

type WithTimeout time.Duration

func (w WithTimeout) ConfigureRequest(c *RequestConfig) {
	c.Timeout = time.Duration(w)
}
