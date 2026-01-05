package denmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// BaseURL is the CVR API endpoint
	BaseURL = "https://cvrapi.dk/api"

	// DefaultTimeout for API requests
	DefaultTimeout = 30 * time.Second

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 5 * time.Minute

	// MaxConcurrentRequests limits parallel API calls
	MaxConcurrentRequests = 5
)

// Client provides access to the Danish CVR API
type Client struct {
	httpClient     *http.Client
	logger         *slog.Logger
	cache          *infra.Cache
	dedup          *infra.RequestDeduplicator
	circuitBreaker *infra.CircuitBreaker
	semaphore      chan struct{}
	userAgent      string
}

// ClientOption configures the Client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = c
	}
}

// WithLogger sets a custom logger
func WithLogger(l *slog.Logger) ClientOption {
	return func(client *Client) {
		client.logger = l
	}
}

// WithCache sets a custom cache
func WithCache(c *infra.Cache) ClientOption {
	return func(client *Client) {
		client.cache = c
	}
}

// WithUserAgent sets a custom user agent (CVR API requires contact info)
func WithUserAgent(ua string) ClientOption {
	return func(client *Client) {
		client.userAgent = ua
	}
}

// NewClient creates a new CVR API client
func NewClient(opts ...ClientOption) *Client {
	// Configure HTTP transport for optimal performance
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

	c := &Client{
		httpClient: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
		},
		logger:         slog.Default(),
		cache:          infra.NewCache(1000),
		dedup:          infra.NewRequestDeduplicator(),
		circuitBreaker: infra.NewCircuitBreaker(),
		semaphore:      make(chan struct{}, MaxConcurrentRequests),
		userAgent:      "nordic-registry-mcp-server/1.0 (github.com/olgasafonova/nordic-registry-mcp-server)",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Close releases resources held by the client
func (c *Client) Close() {
	if c.cache != nil {
		c.cache.Close()
	}
}

// SearchCompany searches for a company by name
func (c *Client) SearchCompany(ctx context.Context, query string) (*Company, error) {
	params := url.Values{}
	params.Set("search", query)
	params.Set("country", "dk")

	cacheKey := "search:" + params.Encode()
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	var result Company
	if err := c.doRequest(ctx, params, &result); err != nil {
		return nil, err
	}

	// Check if we got a valid result
	if result.CVR == "" {
		return nil, &NotFoundError{Query: query}
	}

	c.cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// GetCompany retrieves a company by CVR number
func (c *Client) GetCompany(ctx context.Context, cvr string) (*Company, error) {
	cvr = normalizeCVR(cvr)
	if err := validateCVR(cvr); err != nil {
		return nil, err
	}

	cacheKey := "company:" + cvr
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	// Use deduplication to avoid duplicate requests for the same CVR
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		params := url.Values{}
		params.Set("vat", cvr)
		params.Set("country", "dk")

		var company Company
		if err := c.doRequest(ctx, params, &company); err != nil {
			return nil, err
		}

		// Check if we got a valid result
		if company.CVR == "" {
			return nil, &NotFoundError{CVR: cvr}
		}

		return &company, nil
	})

	if err != nil {
		return nil, err
	}

	company := result.(*Company)
	c.cache.Set(cacheKey, company, DefaultCacheTTL)
	return company, nil
}

// CircuitBreakerStats returns the current circuit breaker state
func (c *Client) CircuitBreakerStats() infra.CircuitBreakerStats {
	return c.circuitBreaker.Stats()
}

// DedupStats returns the number of in-flight deduplicated requests
func (c *Client) DedupStats() int {
	return c.dedup.Stats()
}

// doRequest performs an HTTP request with circuit breaker, rate limiting, and retries
func (c *Client) doRequest(ctx context.Context, params url.Values, result interface{}) error {
	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		stats := c.circuitBreaker.Stats()
		return &infra.ErrCircuitOpen{
			State:    stats.State,
			RetryAt:  stats.LastFailure.Add(30 * time.Second),
			Failures: stats.ConsecutiveFails,
		}
	}

	// Rate limiting via semaphore
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return fmt.Errorf("context canceled while waiting for rate limiter: %w", ctx.Err())
	}

	// Build URL
	reqURL := BaseURL + "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Execute with retry
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("context canceled during backoff: %w", ctx.Err())
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			c.logger.Warn("CVR API request failed, retrying",
				"attempt", attempt+1,
				"error", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Handle HTTP errors
		if resp.StatusCode == http.StatusNotFound {
			c.circuitBreaker.RecordSuccess()
			return &NotFoundError{Query: params.Get("search")}
		}

		if resp.StatusCode >= 400 {
			// Try to parse API error
			var apiErr APIError
			if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
				// Check for "not found" type errors
				if apiErr.T == 1 || strings.Contains(strings.ToLower(apiErr.Error), "not found") {
					c.circuitBreaker.RecordSuccess()
					return &NotFoundError{Query: params.Get("search"), CVR: params.Get("vat")}
				}
				lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.String())
			} else {
				lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			}

			// Don't retry client errors except rate limiting
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
				c.circuitBreaker.RecordSuccess() // Client errors don't indicate service issues
				return lastErr
			}
			continue
		}

		// Check for API error in successful response (CVR API sometimes returns 200 with error)
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
			if apiErr.T == 1 || strings.Contains(strings.ToLower(apiErr.Error), "not found") {
				c.circuitBreaker.RecordSuccess()
				return &NotFoundError{Query: params.Get("search"), CVR: params.Get("vat")}
			}
		}

		// Parse success response
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		c.circuitBreaker.RecordSuccess()
		return nil
	}

	c.circuitBreaker.RecordFailure()
	return lastErr
}

// NotFoundError indicates a company was not found
type NotFoundError struct {
	Query string
	CVR   string
}

func (e *NotFoundError) Error() string {
	if e.CVR != "" {
		return fmt.Sprintf("company not found with CVR: %s", e.CVR)
	}
	return fmt.Sprintf("company not found: %s", e.Query)
}

// normalizeCVR removes spaces and dashes from CVR number
func normalizeCVR(cvr string) string {
	cvr = strings.ReplaceAll(cvr, " ", "")
	cvr = strings.ReplaceAll(cvr, "-", "")
	cvr = strings.ReplaceAll(cvr, "DK", "")
	cvr = strings.ReplaceAll(cvr, "dk", "")
	return cvr
}

// validateCVR checks if the CVR number is valid
func validateCVR(cvr string) error {
	if len(cvr) != 8 {
		return fmt.Errorf("CVR number must be 8 digits, got %d", len(cvr))
	}
	for _, c := range cvr {
		if c < '0' || c > '9' {
			return fmt.Errorf("CVR number must contain only digits")
		}
	}
	return nil
}
