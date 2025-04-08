package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector holds all Prometheus metrics for gobalancer.
type Collector struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	ActiveConns      *prometheus.GaugeVec
	BackendHealth    *prometheus.GaugeVec
	CircuitState     *prometheus.GaugeVec
	UpstreamErrors   *prometheus.CounterVec
}

// New registers all metrics with the default Prometheus registry.
func New() *Collector {
	return &Collector{
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gobalancer_requests_total",
			Help: "Total number of proxied requests.",
		}, []string{"backend", "status_class"}),

		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gobalancer_request_duration_seconds",
			Help:    "Latency of proxied requests.",
			Buckets: prometheus.DefBuckets,
		}, []string{"backend", "route"}),

		ActiveConns: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gobalancer_active_connections",
			Help: "Number of active upstream connections per backend.",
		}, []string{"backend"}),

		BackendHealth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gobalancer_backend_health",
			Help: "Backend health: 0=down, 1=up, 2=probation.",
		}, []string{"backend"}),

		CircuitState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gobalancer_circuit_breaker_state",
			Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open.",
		}, []string{"backend"}),

		UpstreamErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gobalancer_upstream_errors_total",
			Help: "Number of upstream errors per backend and type.",
		}, []string{"backend", "type"}),
	}
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// StatusClass converts an HTTP status code to a class string (2xx, 4xx, 5xx...).
func StatusClass(code int) string {
	switch {
	case code < 200:
		return "1xx"
	case code < 300:
		return "2xx"
	case code < 400:
		return "3xx"
	case code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
