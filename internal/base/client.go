// Package base provides shared client infrastructure for Nordic registry APIs.
package base

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
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

// RequestConfig configures a single HTTP request
type RequestConfig struct {
	URL       string
	UserAgent string
	MaxRetry  int // defaults to 3
}

// DoRequest performs an HTTP request with circuit breaker, rate limiting, and retries.
// Returns the response body on success. The caller handles response parsing.
func (c *Client) DoRequest(ctx context.Context, cfg RequestConfig) ([]byte, int, error) {
	// Check circuit breaker
	if err := c.CheckCircuitBreaker(); err != nil {
		return nil, 0, err
	}

	// Rate limiting via semaphore
	if err := c.AcquireSlot(ctx); err != nil {
		return nil, 0, err
	}
	defer c.ReleaseSlot()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	} else {
		req.Header.Set("User-Agent", "nordic-registry-mcp-server/1.0")
	}

	maxRetry := cfg.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3
	}

	// Execute with retry
	var lastErr error
	for attempt := 0; attempt < maxRetry; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, 0, fmt.Errorf("context canceled during backoff: %w", ctx.Err())
			}
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			c.Logger.Warn("API request failed, retrying",
				"attempt", attempt+1,
				"url", cfg.URL,
				"error", err)
			continue
		}

		body, err := readAndClose(resp)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Handle rate limiting with Retry-After header
		if resp.StatusCode == http.StatusTooManyRequests {
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
					select {
					case <-time.After(time.Duration(seconds) * time.Second):
					case <-ctx.Done():
						return nil, 0, ctx.Err()
					}
					continue
				}
			}
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		// Server errors (5xx) should be retried
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, truncate(string(body), 200))
			continue
		}

		// Return body and status code for caller to handle
		return body, resp.StatusCode, nil
	}

	c.CircuitBreaker.RecordFailure()
	return nil, 0, lastErr
}

// RecordSuccess records a successful request with the circuit breaker
func (c *Client) RecordSuccess() {
	c.CircuitBreaker.RecordSuccess()
}

// RecordFailure records a failed request with the circuit breaker
func (c *Client) RecordFailure() {
	c.CircuitBreaker.RecordFailure()
}

// readAndClose reads the response body and closes it
func readAndClose(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return body, err
}

// truncate shortens a string to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
