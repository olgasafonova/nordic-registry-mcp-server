package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestRecordRequest(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		duration float64
		success  bool
		wantStatus string
	}{
		{
			name:       "successful request",
			tool:       "test_tool",
			duration:   0.5,
			success:    true,
			wantStatus: "success",
		},
		{
			name:       "failed request",
			tool:       "test_tool",
			duration:   1.0,
			success:    false,
			wantStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record the request
			RecordRequest(tt.tool, tt.duration, tt.success)

			// Verify counter was incremented
			counter, err := RequestsTotal.GetMetricWithLabelValues(tt.tool, tt.wantStatus)
			if err != nil {
				t.Fatalf("failed to get metric: %v", err)
			}

			var m dto.Metric
			if err := counter.Write(&m); err != nil {
				t.Fatalf("failed to write metric: %v", err)
			}

			if m.Counter.GetValue() < 1 {
				t.Error("expected counter to be incremented")
			}
		})
	}
}

func TestRecordAPICall(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		duration  float64
		success   bool
		errorCode string
	}{
		{
			name:      "successful API call",
			action:    "query",
			duration:  0.1,
			success:   true,
			errorCode: "",
		},
		{
			name:      "failed API call with error code",
			action:    "edit",
			duration:  0.5,
			success:   false,
			errorCode: "protectedpage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordAPICall(tt.action, tt.duration, tt.success, tt.errorCode)

			// Verify request counter
			status := "success"
			if !tt.success {
				status = "error"
			}
			counter, err := WikiAPIRequestsTotal.GetMetricWithLabelValues(tt.action, status)
			if err != nil {
				t.Fatalf("failed to get metric: %v", err)
			}

			var m dto.Metric
			if err := counter.Write(&m); err != nil {
				t.Fatalf("failed to write metric: %v", err)
			}

			if m.Counter.GetValue() < 1 {
				t.Error("expected counter to be incremented")
			}

			// Verify error counter if error code provided
			if tt.errorCode != "" {
				errCounter, err := WikiAPIErrors.GetMetricWithLabelValues(tt.action, tt.errorCode)
				if err != nil {
					t.Fatalf("failed to get error metric: %v", err)
				}

				var em dto.Metric
				if err := errCounter.Write(&em); err != nil {
					t.Fatalf("failed to write error metric: %v", err)
				}

				if em.Counter.GetValue() < 1 {
					t.Error("expected error counter to be incremented")
				}
			}
		})
	}
}

func TestRecordCacheAccess(t *testing.T) {
	// Get initial values
	initialHits := getCounterValue(t, CacheHits)
	initialMisses := getCounterValue(t, CacheMisses)

	// Record a hit
	RecordCacheAccess(true)
	if getCounterValue(t, CacheHits) != initialHits+1 {
		t.Error("expected cache hits to increment")
	}

	// Record a miss
	RecordCacheAccess(false)
	if getCounterValue(t, CacheMisses) != initialMisses+1 {
		t.Error("expected cache misses to increment")
	}
}

func TestSetCacheSize(t *testing.T) {
	SetCacheSize(100)

	var m dto.Metric
	if err := CacheSize.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	if m.Gauge.GetValue() != 100 {
		t.Errorf("expected cache size 100, got %v", m.Gauge.GetValue())
	}

	SetCacheSize(50)
	if err := CacheSize.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	if m.Gauge.GetValue() != 50 {
		t.Errorf("expected cache size 50, got %v", m.Gauge.GetValue())
	}
}

func TestMetricsRegistered(t *testing.T) {
	// Verify all metrics are registered by checking they can be collected
	metrics := []prometheus.Collector{
		RequestsTotal,
		RequestDuration,
		RequestInFlight,
		CacheHits,
		CacheMisses,
		CacheSize,
		CacheEvictions,
		WikiAPILatency,
		WikiAPIRequestsTotal,
		WikiAPIErrors,
		WikiAPIRetries,
		RateLimitRejections,
		RateLimitWaits,
		AuthFailures,
		SSRFBlocked,
		PanicsRecovered,
		HTTPRequestsTotal,
		HTTPRequestDuration,
		ConnectionPoolSize,
		EditOperations,
		ContentSize,
	}

	for i, m := range metrics {
		if m == nil {
			t.Errorf("metric at index %d is nil", i)
		}
	}
}

func TestNamespace(t *testing.T) {
	if Namespace != "mediawiki_mcp" {
		t.Errorf("expected namespace 'mediawiki_mcp', got '%s'", Namespace)
	}
}

// Helper to get counter value
func getCounterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.Counter.GetValue()
}
