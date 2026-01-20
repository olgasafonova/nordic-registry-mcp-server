package base

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"
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
