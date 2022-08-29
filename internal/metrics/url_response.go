package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	availability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reference_addon_sample_availability",
			Help: "external url availability 0-not available and 1-available.",
		},
		[]string{"url"},
	)
	responseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reference_addon_sample_response_time",
			Help: "external url response time taken.",
		},
		[]string{"url"},
	)
)

// RegisterMetrics must register metrics in given registry collector
func RegisterMetrics() {
	ctrlmetrics.Registry.MustRegister(availability)
	ctrlmetrics.Registry.MustRegister(responseTime)
}

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
