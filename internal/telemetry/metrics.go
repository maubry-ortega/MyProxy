package telemetry

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myproxy_http_requests_total",
		Help: "Total number of HTTP requests proxied",
	}, []string{"domain", "code"})

	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "myproxy_http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"domain"})

	RateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myproxy_rate_limit_hits_total",
		Help: "Total number of rate limit hits",
	}, []string{"domain", "type"})

	ActiveBackends = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "myproxy_active_backends",
		Help: "Number of active (healthy) backends per domain",
	}, []string{"domain"})

	HttpErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "myproxy_http_errors_total",
		Help: "Total number of HTTP 5xx errors",
	}, []string{"domain", "code"})
)

func RecordRequest(domain string, status int, duration float64) {
	HttpRequestsTotal.WithLabelValues(domain, fmt.Sprintf("%d", status)).Inc()
	HttpRequestDuration.WithLabelValues(domain).Observe(duration)
	if status >= 500 {
		HttpErrorsTotal.WithLabelValues(domain, fmt.Sprintf("%d", status)).Inc()
	}
}
