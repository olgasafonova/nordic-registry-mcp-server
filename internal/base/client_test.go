package base

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	defer client.Close()

	if client.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
	if client.Logger == nil {
		t.Error("Logger is nil")
	}
	if client.Cache == nil {
		t.Error("Cache is nil")
	}
	if client.Dedup == nil {
		t.Error("Dedup is nil")
	}
	if client.CircuitBreaker == nil {
		t.Error("CircuitBreaker is nil")
	}
	if client.Semaphore == nil {
		t.Error("Semaphore is nil")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	customHTTP := &http.Client{Timeout: 60 * time.Second}
	customLogger := slog.Default()

	client := NewClient(
		WithHTTPClient(customHTTP),
		WithLogger(customLogger),
	)
	defer client.Close()

	if client.HTTPClient != customHTTP {
		t.Error("custom HTTP client was not set")
	}
	if client.Logger != customLogger {
		t.Error("custom logger was not set")
	}
}

func TestClient_DefaultValues(t *testing.T) {
	client := NewClient()
	defer client.Close()

	if client.HTTPClient.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", client.HTTPClient.Timeout, DefaultTimeout)
	}

	if cap(client.Semaphore) != MaxConcurrentRequests {
		t.Errorf("semaphore capacity = %d, want %d", cap(client.Semaphore), MaxConcurrentRequests)
	}
}

func TestClient_AcquireReleaseSlot(t *testing.T) {
	client := NewClient()
	defer client.Close()

	ctx := context.Background()

	// Acquire a slot
	if err := client.AcquireSlot(ctx); err != nil {
		t.Fatalf("AcquireSlot failed: %v", err)
	}

	// Release it
	client.ReleaseSlot()
}

func TestClient_AcquireSlot_ContextCanceled(t *testing.T) {
	// Create client with only 1 slot
	client := &Client{
		Semaphore: make(chan struct{}, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Fill the slot
	client.Semaphore <- struct{}{}

	// Cancel context immediately
	cancel()

	// Try to acquire - should fail with context error
	err := client.AcquireSlot(ctx)
	if err == nil {
		t.Error("expected error when context is canceled")
	}
}

func TestClient_CircuitBreakerStats(t *testing.T) {
	client := NewClient()
	defer client.Close()

	stats := client.CircuitBreakerStats()
	if stats.State != "closed" {
		t.Errorf("initial circuit breaker state = %q, want 'closed'", stats.State)
	}
}

func TestClient_DedupStats(t *testing.T) {
	client := NewClient()
	defer client.Close()

	stats := client.DedupStats()
	if stats != 0 {
		t.Errorf("initial dedup stats = %d, want 0", stats)
	}
}

func TestClient_CheckCircuitBreaker(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Initially should allow requests
	if err := client.CheckCircuitBreaker(); err != nil {
		t.Errorf("unexpected error from CheckCircuitBreaker: %v", err)
	}
}

func TestClient_CheckCircuitBreaker_Open(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Open the circuit breaker by recording failures
	for range 10 {
		client.RecordFailure()
	}

	// Circuit should now be open
	err := client.CheckCircuitBreaker()
	if err == nil {
		t.Error("expected error when circuit is open")
	}
}

func TestClient_WithCache(t *testing.T) {
	customCache := infra.NewCache(500)
	defer customCache.Close()

	client := NewClient(WithCache(customCache))
	defer client.Close()

	if client.Cache != customCache {
		t.Error("custom cache was not set")
	}
}

func TestClient_RecordSuccess(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Record some failures first
	client.RecordFailure()
	client.RecordFailure()

	// Then record success - should reset consecutive fails
	client.RecordSuccess()

	stats := client.CircuitBreakerStats()
	if stats.ConsecutiveFails != 0 {
		t.Errorf("consecutive fails = %d, want 0 after success", stats.ConsecutiveFails)
	}
}

func TestClient_RecordFailure(t *testing.T) {
	client := NewClient()
	defer client.Close()

	client.RecordFailure()
	client.RecordFailure()
	client.RecordFailure()

	stats := client.CircuitBreakerStats()
	if stats.ConsecutiveFails != 3 {
		t.Errorf("consecutive fails = %d, want 3", stats.ConsecutiveFails)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"longer than max length", 10, "longer tha..."},
		{"", 5, ""},
		{"abc", 0, "..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestReadAndClose(t *testing.T) {
	t.Run("normal response", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader("test response body"))
		resp := &http.Response{
			Body: body,
		}

		data, err := readAndClose(resp)
		if err != nil {
			t.Fatalf("readAndClose failed: %v", err)
		}

		if string(data) != "test response body" {
			t.Errorf("got %q, want 'test response body'", string(data))
		}
	})

	t.Run("empty response", func(t *testing.T) {
		body := io.NopCloser(strings.NewReader(""))
		resp := &http.Response{
			Body: body,
		}

		data, err := readAndClose(resp)
		if err != nil {
			t.Fatalf("readAndClose failed: %v", err)
		}

		if len(data) != 0 {
			t.Errorf("expected empty data, got %d bytes", len(data))
		}
	})
}

func TestDoRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Accept header not set")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	body, statusCode, err := client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 1,
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", statusCode)
	}
	if string(body) != `{"status":"ok"}` {
		t.Errorf("body = %q, want '{\"status\":\"ok\"}'", string(body))
	}
}

func TestDoRequest_CustomUserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	_, _, _ = client.DoRequest(context.Background(), RequestConfig{
		URL:       server.URL,
		UserAgent: "custom-agent/1.0",
		MaxRetry:  1,
	})

	if receivedUA != "custom-agent/1.0" {
		t.Errorf("User-Agent = %q, want 'custom-agent/1.0'", receivedUA)
	}
}

func TestDoRequest_DefaultUserAgent(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	_, _, _ = client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 1,
	})

	if receivedUA != "nordic-registry-mcp-server/1.0" {
		t.Errorf("User-Agent = %q, want 'nordic-registry-mcp-server/1.0'", receivedUA)
	}
}

func TestDoRequest_CircuitOpen(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Open the circuit breaker
	for range 10 {
		client.RecordFailure()
	}

	_, _, err := client.DoRequest(context.Background(), RequestConfig{
		URL: "http://example.com",
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}
}

func TestDoRequest_ServerError_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	body, statusCode, err := client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 5,
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", statusCode)
	}
	if string(body) != "success" {
		t.Errorf("body = %q, want 'success'", string(body))
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestDoRequest_RateLimited(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, statusCode, err := client.DoRequest(ctx, RequestConfig{
		URL:      server.URL,
		MaxRetry: 3,
	})

	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", statusCode)
	}
	if string(body) != "success" {
		t.Errorf("body = %q, want 'success'", string(body))
	}
}

func TestDoRequest_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := client.DoRequest(ctx, RequestConfig{
		URL:      server.URL,
		MaxRetry: 1,
	})

	if err == nil {
		t.Error("expected error when context is canceled")
	}
}

func TestDoRequest_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("always fails"))
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	_, _, err := client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 2,
	})

	if err == nil {
		t.Error("expected error when all retries fail")
	}
}

func TestDoRequest_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	body, statusCode, err := client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 1,
	})

	// 404 should not be retried - it's returned immediately
	if err != nil {
		t.Fatalf("DoRequest failed: %v", err)
	}
	if statusCode != http.StatusNotFound {
		t.Errorf("status code = %d, want 404", statusCode)
	}
	if string(body) != "not found" {
		t.Errorf("body = %q, want 'not found'", string(body))
	}
}

func TestDoRequest_DefaultMaxRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	// MaxRetry = 0 should default to 3
	_, _, _ = client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 0, // defaults to 3
	})

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3 (default)", attempts)
	}
}

func TestDoRequest_RateLimited_InvalidRetryAfter(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "invalid")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	_, _, err := client.DoRequest(context.Background(), RequestConfig{
		URL:      server.URL,
		MaxRetry: 3,
	})

	// Should still succeed after retry (invalid Retry-After is ignored)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDoRequest_RateLimited_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := client.DoRequest(ctx, RequestConfig{
		URL:      server.URL,
		MaxRetry: 2,
	})

	if err == nil {
		t.Error("expected error when context is canceled during Retry-After wait")
	}
}

func TestDoRequest_BackoffContextCanceled(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := client.DoRequest(ctx, RequestConfig{
		URL:      server.URL,
		MaxRetry: 10,
	})

	// Should fail due to context cancellation during backoff
	if err == nil {
		t.Error("expected error when context is canceled during backoff")
	}
}

func TestReadAndClose_ResponseTooLarge(t *testing.T) {
	// Create a response larger than MaxResponseSize
	largeData := make([]byte, MaxResponseSize+100)
	body := io.NopCloser(bytes.NewReader(largeData))
	resp := &http.Response{
		Body: body,
	}

	_, err := readAndClose(resp)
	if err == nil {
		t.Error("expected error for oversized response")
	}
}

func TestReadAndClose_ReadError(t *testing.T) {
	body := io.NopCloser(&errorReader{})
	resp := &http.Response{
		Body: body,
	}

	_, err := readAndClose(resp)
	if err == nil {
		t.Error("expected error when read fails")
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
