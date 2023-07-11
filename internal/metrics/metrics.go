package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func RegisterMetrics(reg prometheus.Registerer) error {
	if err := reg.Register(availability); err != nil {
		return fmt.Errorf("registering 'availability' metric: %w", err)
	}

	if err := reg.Register(responseTime); err != nil {
		return fmt.Errorf("registering 'responseTime' metric: %w", err)
	}

	if err := reg.Register(smokeTest); err != nil {
		return fmt.Errorf("registering 'smokeTest' metric: %w", err)
	}

	return nil
}

var (
	availability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricPrefix + "sample_availability",
			Help: "external url availability 0-not available and 1-available.",
		},
		[]string{"url"},
	)
	responseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricPrefix + "sample_response_time",
			Help: "external url response time taken.",
		},
		[]string{"url"},
	)
	smokeTest = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: metricPrefix + "smoke_test",
			Help: "smoke test for testing end-to-end metrics flow",
		},
	)
)

const metricPrefix = "reference_addon_"

func NewResponseSamplerImpl() *ResponseSamplerImpl {
	return &ResponseSamplerImpl{}
}

type ResponseSamplerImpl struct{}

func (r *ResponseSamplerImpl) RequestSampleResponseData(urls ...string) {
	for _, url := range urls {
		status, timeTaken := callExternalURL(url)

		availability.WithLabelValues(url).Set(status)
		responseTime.WithLabelValues(url).Set(timeTaken)
	}
}

func callExternalURL(externalURL string) (float64, float64) {
	start := time.Now()

	res, err := http.Get(externalURL)
	if err != nil {
		return 0, 0
	}
	defer res.Body.Close()

	status := 0

	if res.StatusCode == 200 {
		status = 1
	}

	return float64(status), float64(time.Since(start).Milliseconds())
}

func NewSmokeTester() *SmokeTester {
	return &SmokeTester{}
}

type SmokeTester struct{}

func (t *SmokeTester) Enable() {
	smokeTest.Set(1)
}

func (t *SmokeTester) Disable() {
	smokeTest.Set(0)
}
