package finland

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// DefaultBaseURL is the PRH open data API base URL
	DefaultBaseURL = "https://avoindata.prh.fi/opendata-ytj-api/v3"

	// DefaultTimeout for API requests
	DefaultTimeout = 30 * time.Second

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 15 * time.Minute

	// MaxConcurrentRequests limits concurrent requests
	MaxConcurrentRequests = 5
)

// Client is a Finnish PRH API client with caching and resilience
type Client struct {
	baseURL        string
	httpClient     *http.Client
	cache          *infra.Cache
	circuitBreaker *infra.CircuitBreaker
	dedup          *infra.RequestDeduplicator
	semaphore      chan struct{}
	logger         *slog.Logger
}

// ClientOption configures the Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new Finnish PRH API client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:        DefaultBaseURL,
		httpClient:     &http.Client{Timeout: DefaultTimeout},
		cache:          infra.NewCache(1000),
		circuitBreaker: infra.NewCircuitBreaker(),
		dedup:          infra.NewRequestDeduplicator(),
		semaphore:      make(chan struct{}, MaxConcurrentRequests),
		logger:         slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Close releases client resources
func (c *Client) Close() {
	if c.cache != nil {
		c.cache.Close()
	}
}

// CircuitBreakerStats returns the current circuit breaker stats
func (c *Client) CircuitBreakerStats() infra.CircuitBreakerStats {
	return c.circuitBreaker.Stats()
}

// businessIDRegex validates Finnish business ID format (Y-tunnus)
var businessIDRegex = regexp.MustCompile(`^\d{7}-\d$`)

// NormalizeBusinessID cleans and validates a Finnish business ID
func NormalizeBusinessID(id string) (string, error) {
	// Remove spaces and convert to uppercase
	cleaned := strings.TrimSpace(id)
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	// Remove FI prefix if present
	cleaned = strings.TrimPrefix(cleaned, "FI")
	cleaned = strings.TrimPrefix(cleaned, "fi")

	// Validate format
	if !businessIDRegex.MatchString(cleaned) {
		return "", fmt.Errorf("invalid Finnish business ID format: %s (expected format: 1234567-8)", id)
	}

	return cleaned, nil
}

// SearchCompanies searches for companies by name
func (c *Client) SearchCompanies(ctx context.Context, args SearchCompaniesArgs) (*CompanySearchResponse, error) {
	// Build query params
	params := url.Values{}
	params.Set("name", args.Query)

	if args.Location != "" {
		params.Set("location", args.Location)
	}
	if args.CompanyForm != "" {
		params.Set("companyForm", args.CompanyForm)
	}
	if args.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", args.Page))
	}

	cacheKey := "fi:search:" + params.Encode()

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*CompanySearchResponse), nil
	}

	// Deduplicate concurrent requests
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		return c.doSearch(ctx, params)
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*CompanySearchResponse)
	c.cache.Set(cacheKey, resp, DefaultCacheTTL)
	return resp, nil
}

func (c *Client) doSearch(ctx context.Context, params url.Values) (*CompanySearchResponse, error) {
	// Rate limit via semaphore
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while waiting for rate limiter: %w", ctx.Err())
	}

	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		return nil, fmt.Errorf("circuit breaker open: PRH API unavailable")
	}

	reqURL := fmt.Sprintf("%s/companies?%s", c.baseURL, params.Encode())
	c.logger.Debug("PRH API request", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nordic-registry-mcp-server/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("PRH API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("PRH API error: %s", resp.Status)
	}

	c.circuitBreaker.RecordSuccess()

	var result CompanySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetCompany retrieves a company by business ID
func (c *Client) GetCompany(ctx context.Context, businessID string) (*Company, error) {
	normalized, err := NormalizeBusinessID(businessID)
	if err != nil {
		return nil, err
	}

	cacheKey := "fi:company:" + normalized

	// Check cache
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	// Deduplicate concurrent requests
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		return c.doGetCompany(ctx, normalized)
	})
	if err != nil {
		return nil, err
	}

	company := result.(*Company)
	c.cache.Set(cacheKey, company, DefaultCacheTTL)
	return company, nil
}

func (c *Client) doGetCompany(ctx context.Context, businessID string) (*Company, error) {
	// Rate limit via semaphore
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context canceled while waiting for rate limiter: %w", ctx.Err())
	}

	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		return nil, fmt.Errorf("circuit breaker open: PRH API unavailable")
	}

	reqURL := fmt.Sprintf("%s/companies?businessId=%s", c.baseURL, url.QueryEscape(businessID))
	c.logger.Debug("PRH API request", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nordic-registry-mcp-server/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("PRH API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("PRH API error: %s", resp.Status)
	}

	c.circuitBreaker.RecordSuccess()

	var result CompanySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Companies) == 0 {
		return nil, fmt.Errorf("company not found: %s", businessID)
	}

	return &result.Companies[0], nil
}
