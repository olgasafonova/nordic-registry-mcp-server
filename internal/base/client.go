// Package base provides shared client infrastructure for Nordic registry APIs.
package base

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// DefaultTimeout for API requests
	DefaultTimeout = 30 * time.Second

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 5 * time.Minute

	// MaxConcurrentRequests limits parallel API calls
	MaxConcurrentRequests = 5
)

// Client provides common HTTP client infrastructure with caching, rate limiting,
// circuit breaking, and request deduplication.
type Client struct {
	HTTPClient     *http.Client
	Logger         *slog.Logger
	Cache          *infra.Cache
	Dedup          *infra.RequestDeduplicator
	CircuitBreaker *infra.CircuitBreaker
	Semaphore      chan struct{}
}

// ClientOption configures the Client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.HTTPClient = c
	}
}

// WithLogger sets a custom logger
func WithLogger(l *slog.Logger) ClientOption {
	return func(client *Client) {
		client.Logger = l
	}
}

// WithCache sets a custom cache
func WithCache(c *infra.Cache) ClientOption {
	return func(client *Client) {
		client.Cache = c
	}
}

// NewClient creates a new base client with default settings
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		HTTPClient:     newHTTPClient(DefaultTimeout),
		Logger:         slog.Default(),
		Cache:          infra.NewCache(1000),
		Dedup:          infra.NewRequestDeduplicator(),
		CircuitBreaker: infra.NewCircuitBreaker(),
		Semaphore:      make(chan struct{}, MaxConcurrentRequests),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Close releases resources held by the client
func (c *Client) Close() {
	if c.Cache != nil {
		c.Cache.Close()
	}
}

// CircuitBreakerStats returns the current circuit breaker state
func (c *Client) CircuitBreakerStats() infra.CircuitBreakerStats {
	return c.CircuitBreaker.Stats()
}

// DedupStats returns the number of in-flight deduplicated requests
func (c *Client) DedupStats() int {
	return c.Dedup.Stats()
}

// AcquireSlot blocks until a request slot is available or context is canceled
func (c *Client) AcquireSlot(ctx context.Context) error {
	select {
	case c.Semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context canceled while waiting for rate limiter: %w", ctx.Err())
	}
}

// ReleaseSlot releases a request slot
func (c *Client) ReleaseSlot() {
	<-c.Semaphore
}

// CheckCircuitBreaker returns nil if requests are allowed, or an error if the circuit is open
func (c *Client) CheckCircuitBreaker() error {
	if !c.CircuitBreaker.Allow() {
		stats := c.CircuitBreaker.Stats()
		return &infra.ErrCircuitOpen{
			State:    stats.State,
			RetryAt:  stats.LastFailure.Add(30 * time.Second),
			Failures: stats.ConsecutiveFails,
		}
	}
	return nil
}

// newHTTPClient creates an HTTP client with optimized transport settings
func newHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableCompression:    false,
		ForceAttemptHTTP2:     true,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
