// Package metrics provides Prometheus metrics for the Nordic Registry MCP server.
// It tracks request counts, latencies, cache performance, and error rates.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Namespace and subsystem for all metrics
const (
	Namespace = "nordic_registry_mcp"
)

var (
	// RequestsTotal counts total MCP tool calls by tool name and status
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "requests_total",
		Help:      "Total number of MCP tool calls",
	}, []string{"tool", "status"})

	// RequestDuration measures request latency distribution
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "request_duration_seconds",
		Help:      "Request latency distribution by tool",
		Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"tool"})

	// RequestInFlight tracks currently executing requests
	RequestInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "requests_in_flight",
		Help:      "Number of requests currently being processed",
	}, []string{"tool"})

	// CacheHits counts cache hits
	CacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "cache_hits_total",
		Help:      "Total cache hit count",
	})

	// CacheMisses counts cache misses
	CacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "cache_misses_total",
		Help:      "Total cache miss count",
	})

	// CacheSize tracks current cache entry count
	CacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "cache_entries",
		Help:      "Current number of cache entries",
	})

	// CacheEvictions counts cache evictions
	CacheEvictions = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "cache_evictions_total",
		Help:      "Total cache eviction count",
	})

	// RegistryAPILatency measures registry API call latency by country and action
	RegistryAPILatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "registry_api_latency_seconds",
		Help:      "Registry API call latency by country and action",
		Buckets:   prometheus.DefBuckets,
	}, []string{"country", "action"})

	// RegistryAPIRequestsTotal counts registry API requests
	RegistryAPIRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "registry_api_requests_total",
		Help:      "Total registry API requests by country, action and status",
	}, []string{"country", "action", "status"})

	// RegistryAPIErrors counts registry API errors by error code
	RegistryAPIErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "registry_api_errors_total",
		Help:      "Registry API errors by country, action and error code",
	}, []string{"country", "action", "error_code"})

	// RegistryAPIRetries counts API request retries
	RegistryAPIRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "registry_api_retries_total",
		Help:      "Registry API retry count by country and action",
	}, []string{"country", "action"})

	// RateLimitRejections counts requests rejected due to rate limiting
	RateLimitRejections = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "rate_limit_rejections_total",
		Help:      "Requests rejected due to rate limiting",
	})

	// RateLimitWaits counts requests that had to wait for rate limiter
	RateLimitWaits = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "rate_limit_waits_total",
		Help:      "Requests that waited for rate limiter semaphore",
	})

	// AuthFailures counts authentication failures
	AuthFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "auth_failures_total",
		Help:      "Authentication failure count by reason",
	}, []string{"reason"})

	// SSRFBlocked counts blocked SSRF attempts
	SSRFBlocked = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "ssrf_blocked_total",
		Help:      "SSRF attempts blocked by security",
	}, []string{"type"})

	// PanicsRecovered counts recovered panics
	PanicsRecovered = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "panics_recovered_total",
		Help:      "Number of panics recovered in tool handlers",
	}, []string{"tool"})

	// HTTPRequestsTotal counts HTTP transport requests
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "http_requests_total",
		Help:      "Total HTTP requests by method and status",
	}, []string{"method", "status"})

	// HTTPRequestDuration measures HTTP request latency
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency distribution",
		Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"method", "path"})

	// ConnectionPoolSize tracks HTTP connection pool size
	ConnectionPoolSize = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "connection_pool_size",
		Help:      "Current size of HTTP connection pool",
	})

	// EditOperations counts write operations by type
	EditOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace,
		Name:      "edit_operations_total",
		Help:      "Edit operations by type and status",
	}, []string{"operation", "status"})

	// ContentSize tracks content sizes processed
	ContentSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace,
		Name:      "content_size_bytes",
		Help:      "Content size distribution in bytes",
		Buckets:   []float64{100, 1000, 10000, 50000, 100000, 250000, 500000, 1000000},
	}, []string{"operation"})
)

// RecordRequest records a completed request with its duration and status
func RecordRequest(tool string, duration float64, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	RequestsTotal.WithLabelValues(tool, status).Inc()
	RequestDuration.WithLabelValues(tool).Observe(duration)
}

// RecordAPICall records a registry API call
func RecordAPICall(country, action string, duration float64, success bool, errorCode string) {
	status := "success"
	if !success {
		status = "error"
	}
	RegistryAPIRequestsTotal.WithLabelValues(country, action, status).Inc()
	RegistryAPILatency.WithLabelValues(country, action).Observe(duration)
	if errorCode != "" {
		RegistryAPIErrors.WithLabelValues(country, action, errorCode).Inc()
	}
}

// RecordCacheAccess records a cache hit or miss
func RecordCacheAccess(hit bool) {
	if hit {
		CacheHits.Inc()
	} else {
		CacheMisses.Inc()
	}
}

// SetCacheSize updates the current cache size gauge
func SetCacheSize(size int64) {
	CacheSize.Set(float64(size))
}
