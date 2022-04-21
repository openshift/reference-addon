package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	externalURLs = []string{"https://httpstat.us/503", "https://httpstat.us/200"}

	availability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reference_addon_sample_availability",
			Help: "external url availability.",
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

func RequestSampleResponseData() {
	for _, externalURL := range externalURLs {
		status, timeTaken := callExternalURL(externalURL)
		availability.WithLabelValues(externalURL).Set(status)
		responseTime.WithLabelValues(externalURL).Set(timeTaken)
	}
}

func callExternalURL(externalURL string) (float64, float64) {
	start := time.Now()
	response, err := http.Get(externalURL)
	if err != nil {
		return 0, 0
	}
	defer response.Body.Close()
	timeTaken := time.Since(start).Milliseconds()
	status := 0
	if response.StatusCode == 200 {
		status = 1
	}
	return float64(status), float64(timeTaken)
}
