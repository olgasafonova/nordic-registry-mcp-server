// Package base provides shared client infrastructure for Nordic registry APIs.
package base

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
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
	MaxConcurrentRequests = 15

	// MaxResponseSize limits response body size to prevent memory exhaustion (10 MB)
	MaxResponseSize = 10 * 1024 * 1024
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
	if err := c.CheckCircuitBreaker(); err != nil {
		return nil, 0, err
	}

	if err := c.AcquireSlot(ctx); err != nil {
		return nil, 0, err
	}
	defer c.ReleaseSlot()

	req, err := c.buildRequest(ctx, cfg)
	if err != nil {
		return nil, 0, err
	}

	maxRetry := cfg.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3
	}

	var lastErr error
	for attempt := 0; attempt < maxRetry; attempt++ {
		if err := c.waitBeforeAttempt(ctx, attempt); err != nil {
			return nil, 0, err
		}

		body, status, retryErr, fatal := c.executeAttempt(ctx, req, cfg, attempt)
		if fatal != nil {
			return nil, 0, fatal
		}
		if retryErr != nil {
			lastErr = retryErr
			continue
		}
		return body, status, nil
	}

	c.CircuitBreaker.RecordFailure()
	return nil, 0, lastErr
}

// buildRequest constructs the HTTP request with default headers.
func (c *Client) buildRequest(ctx context.Context, cfg RequestConfig) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	} else {
		req.Header.Set("User-Agent", "nordic-registry-mcp-server/1.0")
	}
	return req, nil
}

// waitBeforeAttempt sleeps with exponential backoff + jitter before retries.
// Returns an error only when the context is canceled mid-backoff.
func (c *Client) waitBeforeAttempt(ctx context.Context, attempt int) error {
	if attempt == 0 {
		return nil
	}
	baseBackoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
	jitter := time.Duration(rand.Int63n(int64(baseBackoff / 2))) // #nosec G404 -- jitter for timing, not security
	backoff := baseBackoff + jitter
	select {
	case <-time.After(backoff):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context canceled during backoff: %w", ctx.Err())
	}
}

// executeAttempt performs a single HTTP attempt. Returns body/status on success,
// a retryErr if the attempt should be retried, or a fatal error to abort the loop.
func (c *Client) executeAttempt(ctx context.Context, req *http.Request, cfg RequestConfig, attempt int) (body []byte, status int, retryErr, fatal error) {
	resp, err := c.HTTPClient.Do(req) // #nosec G704 -- URL constructed from hardcoded base + validated input
	if err != nil {
		c.Logger.Warn("API request failed, retrying",
			"attempt", attempt+1,
			"url", cfg.URL,
			"error", err)
		return nil, 0, fmt.Errorf("request failed: %w", err), nil
	}

	body, err = readAndClose(resp)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err), nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if fatal := handleRateLimit(ctx, resp); fatal != nil {
			return nil, 0, nil, fatal
		}
		return nil, 0, fmt.Errorf("rate limited (429)"), nil
	}

	if resp.StatusCode >= 500 {
		return nil, 0, fmt.Errorf("server error %d: %s", resp.StatusCode, truncate(string(body), 200)), nil
	}

	return body, resp.StatusCode, nil, nil
}

// handleRateLimit honors a Retry-After header if present. Returns a fatal
// error only when the context is canceled while waiting; nil otherwise (so the
// caller falls through to retry on the next loop iteration).
func handleRateLimit(ctx context.Context, resp *http.Response) error {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return nil
	}
	seconds, parseErr := strconv.Atoi(retryAfter)
	if parseErr != nil {
		return nil
	}
	select {
	case <-time.After(time.Duration(seconds) * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RecordSuccess records a successful request with the circuit breaker
func (c *Client) RecordSuccess() {
	c.CircuitBreaker.RecordSuccess()
}

// RecordFailure records a failed request with the circuit breaker
func (c *Client) RecordFailure() {
	c.CircuitBreaker.RecordFailure()
}

// readAndClose reads the response body (limited to MaxResponseSize) and closes it
func readAndClose(resp *http.Response) ([]byte, error) {
	// Limit response size to prevent memory exhaustion
	limited := io.LimitReader(resp.Body, MaxResponseSize+1)
	body, err := io.ReadAll(limited)
	_ = resp.Body.Close()

	if err != nil {
		return nil, err
	}

	// Check if we hit the limit (read more than MaxResponseSize)
	if len(body) > MaxResponseSize {
		return nil, fmt.Errorf("response exceeds maximum size of %d bytes", MaxResponseSize)
	}

	return body, nil
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
