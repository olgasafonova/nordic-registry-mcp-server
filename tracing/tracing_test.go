package tracing

import (
	"context"
	"errors"
	"os"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestDefaultConfig(t *testing.T) {
	// Clear environment variables for consistent testing
	_ = os.Unsetenv("OTEL_ENVIRONMENT")
	_ = os.Unsetenv("OTEL_ENABLED")
	_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	cfg := DefaultConfig()

	if cfg.ServiceName != "nordic-registry-mcp-server" {
		t.Errorf("Expected ServiceName 'nordic-registry-mcp-server', got %q", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("Expected ServiceVersion '1.0.0', got %q", cfg.ServiceVersion)
	}
	if cfg.Environment != "development" {
		t.Errorf("Expected Environment 'development', got %q", cfg.Environment)
	}
	if cfg.Enabled {
		t.Error("Expected Enabled to be false by default")
	}
	if cfg.OTLPEndpoint != "" {
		t.Errorf("Expected OTLPEndpoint to be empty, got %q", cfg.OTLPEndpoint)
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("Expected SampleRate 1.0, got %f", cfg.SampleRate)
	}
}

func TestDefaultConfig_WithEnvVars(t *testing.T) {
	_ = os.Setenv("OTEL_ENVIRONMENT", "production")
	_ = os.Setenv("OTEL_ENABLED", "true")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	defer func() {
		_ = os.Unsetenv("OTEL_ENVIRONMENT")
		_ = os.Unsetenv("OTEL_ENABLED")
		_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}()

	cfg := DefaultConfig()

	if cfg.Environment != "production" {
		t.Errorf("Expected Environment 'production', got %q", cfg.Environment)
	}
	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.OTLPEndpoint != "localhost:4318" {
		t.Errorf("Expected OTLPEndpoint 'localhost:4318', got %q", cfg.OTLPEndpoint)
	}
}

func TestDefaultConfig_EnabledByEndpoint(t *testing.T) {
	_ = os.Unsetenv("OTEL_ENABLED")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	defer func() {
		_ = os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}()

	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true when OTLP endpoint is set")
	}
}

func TestSetup_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Shutdown should be a no-op function
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestSetup_EnabledWithStdout(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        true,
		OTLPEndpoint:   "", // Empty means stdout exporter
		SampleRate:     1.0,
	}

	shutdown, err := Setup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	// Verify tracing is set up by getting a tracer
	tracer := Tracer()
	if tracer == nil {
		t.Error("Expected tracer to be non-nil")
	}
}

func TestSetup_DifferentSampleRates(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate float64
	}{
		{"always sample", 1.0},
		{"never sample", 0.0},
		{"ratio sample", 0.5},
		{"above 1.0", 1.5},  // Should still work, treated as always
		{"below 0.0", -0.5}, // Should still work, treated as never
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Enabled:        true,
				SampleRate:     tt.sampleRate,
			}

			shutdown, err := Setup(context.Background(), cfg)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			_ = shutdown(context.Background())
		})
	}
}

func TestTracer(t *testing.T) {
	tracer := Tracer()
	if tracer == nil {
		t.Error("Expected tracer to be non-nil")
	}
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()

	newCtx, span := StartSpan(ctx, "test-span")
	defer span.End()

	if newCtx == nil {
		t.Error("Expected context to be non-nil")
	}
	if span == nil {
		t.Error("Expected span to be non-nil")
	}

	// Verify span context is valid (even if not sampled)
	spanCtx := trace.SpanFromContext(newCtx).SpanContext()
	if !spanCtx.TraceID().IsValid() && !spanCtx.SpanID().IsValid() {
		// This is fine if tracing isn't configured, but the span should exist
		if span == nil {
			t.Error("Span should not be nil")
		}
	}
}

func TestAddToolAttributes(t *testing.T) {
	_, span := StartSpan(context.Background(), "test-tool")
	defer span.End()

	// Should not panic
	AddToolAttributes(span, "norway_search_companies", "search")
}

func TestAddRegistryAttributes(t *testing.T) {
	tests := []struct {
		name    string
		country string
		action  string
	}{
		{"norway search", "norway", "search"},
		{"norway get_company", "norway", "get_company"},
		{"empty action", "norway", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, span := StartSpan(context.Background(), "test-registry")
			defer span.End()

			// Should not panic
			AddRegistryAttributes(span, tt.country, tt.action)
		})
	}
}

func TestRecordError(t *testing.T) {
	_, span := StartSpan(context.Background(), "test-error")
	defer span.End()

	// Should not panic with nil error
	RecordError(span, nil)

	// Should not panic with actual error
	RecordError(span, errors.New("test error"))
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     string
		setEnv       bool
	}{
		{
			name:         "env set",
			envKey:       "TEST_GET_ENV_KEY",
			envValue:     "custom-value",
			defaultValue: "default-value",
			expected:     "custom-value",
			setEnv:       true,
		},
		{
			name:         "env not set",
			envKey:       "TEST_GET_ENV_KEY_UNSET",
			defaultValue: "default-value",
			expected:     "default-value",
			setEnv:       false,
		},
		{
			name:         "env empty",
			envKey:       "TEST_GET_ENV_KEY_EMPTY",
			envValue:     "",
			defaultValue: "default-value",
			expected:     "default-value",
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				_ = os.Setenv(tt.envKey, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.envKey) }()
			}

			result := getEnvOrDefault(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTracerName(t *testing.T) {
	if TracerName != "nordic-registry-mcp-server" {
		t.Errorf("Expected TracerName 'nordic-registry-mcp-server', got %q", TracerName)
	}
}
