package pkg

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	externalURLs = []string{"https://httpstat.us/503", "https://httpstat.us/200"}

	urlResponseCode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reference_addon_sample_response_code",
			Help: "external url response.",
		},
		[]string{"url"},
	)
	urlResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reference_addon_sample_response_time",
			Help: "external url response time.",
		},
		[]string{"url"},
	)
)

// RegisterMetrics must register metrics in given registry collector
func RegisterMetrics() {
	ctrlmetrics.Registry.MustRegister(urlResponseCode)
	ctrlmetrics.Registry.MustRegister(urlResponseTime)
}

func AddURLResponseMetrics() {
	for _, externalURL := range externalURLs {
		status, responseTime := callExternalURL(externalURL)
		urlResponseCode.WithLabelValues(externalURL).Set(status)
		urlResponseTime.WithLabelValues(externalURL).Set(responseTime)
	}
}

func callExternalURL(externalURL string) (float64, float64) {
	start := time.Now()
	response, err := http.Get(externalURL)
	if err != nil {
		return 0, 0
	}
	responseTime := time.Since(start).Milliseconds()
	status := 0
	if response.StatusCode == 200 {
		status = 1
	}
	return float64(status), float64(responseTime)
}
