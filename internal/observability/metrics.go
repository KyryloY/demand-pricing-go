package observability

import "github.com/prometheus/client_golang/prometheus"

var HTTPRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{Name: "demand_pricing_http_requests_total", Help: "Total HTTP requests handled by the API."},
	[]string{"method", "route", "status"},
)

func init() {
	prometheus.MustRegister(HTTPRequests)
}
