package norway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// BaseURL is the Brønnøysundregistrene API endpoint
	BaseURL = "https://data.brreg.no/enhetsregisteret/api"

	// DefaultTimeout for API requests
	DefaultTimeout = 30 * time.Second

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 5 * time.Minute

	// MaxConcurrentRequests limits parallel API calls
	MaxConcurrentRequests = 5
)

// Client provides access to the Norwegian Brønnøysundregistrene API
type Client struct {
	httpClient     *http.Client
	logger         *slog.Logger
	cache          *infra.Cache
	dedup          *infra.RequestDeduplicator
	circuitBreaker *infra.CircuitBreaker
	semaphore      chan struct{}
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

// NewClient creates a new Brønnøysundregistrene client
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

// SearchCompanies searches for companies by name or other criteria
func (c *Client) SearchCompanies(ctx context.Context, query string, opts *SearchOptions) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("navn", query)

	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			params.Set("size", strconv.Itoa(opts.Size))
		}
		if opts.OrgForm != "" {
			params.Set("organisasjonsform", opts.OrgForm)
		}
		if opts.Municipality != "" {
			params.Set("kommunenummer", opts.Municipality)
		}
		if opts.RegisteredInVAT != nil {
			params.Set("registrertIMvaregisteret", strconv.FormatBool(*opts.RegisteredInVAT))
		}
		if opts.Bankrupt != nil {
			params.Set("konkurs", strconv.FormatBool(*opts.Bankrupt))
		}
	}

	cacheKey := "search:" + params.Encode()
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*SearchResponse), nil
	}

	var result SearchResponse
	if err := c.doRequest(ctx, "/enheter", params, &result); err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// SearchOptions configures company search
type SearchOptions struct {
	Page            int
	Size            int
	OrgForm         string // Organization form code (AS, ENK, etc.)
	Municipality    string // Kommune number
	RegisteredInVAT *bool
	Bankrupt        *bool
}

// GetCompany retrieves a company by organization number
func (c *Client) GetCompany(ctx context.Context, orgNumber string) (*Company, error) {
	orgNumber = normalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	cacheKey := "company:" + orgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	// Use deduplication to avoid duplicate requests for the same org number
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		var company Company
		if err := c.doRequest(ctx, "/enheter/"+orgNumber, nil, &company); err != nil {
			return nil, err
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

// GetRoles retrieves board members and other roles for a company
func (c *Client) GetRoles(ctx context.Context, orgNumber string) (*RolesResponse, error) {
	orgNumber = normalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	cacheKey := "roles:" + orgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*RolesResponse), nil
	}

	var result RolesResponse
	if err := c.doRequest(ctx, "/enheter/"+orgNumber+"/roller", nil, &result); err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// GetSubUnits retrieves sub-units (branches) for a parent company
func (c *Client) GetSubUnits(ctx context.Context, parentOrgNumber string) (*SubUnitSearchResponse, error) {
	parentOrgNumber = normalizeOrgNumber(parentOrgNumber)
	if err := validateOrgNumber(parentOrgNumber); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("overordnetEnhet", parentOrgNumber)

	cacheKey := "subunits:" + parentOrgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*SubUnitSearchResponse), nil
	}

	var result SubUnitSearchResponse
	if err := c.doRequest(ctx, "/underenheter", params, &result); err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// GetSubUnit retrieves a specific sub-unit by organization number
func (c *Client) GetSubUnit(ctx context.Context, orgNumber string) (*SubUnit, error) {
	orgNumber = normalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	cacheKey := "subunit:" + orgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		return cached.(*SubUnit), nil
	}

	var result SubUnit
	if err := c.doRequest(ctx, "/underenheter/"+orgNumber, nil, &result); err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// GetUpdates retrieves recent updates from the registry
func (c *Client) GetUpdates(ctx context.Context, since time.Time, opts *UpdatesOptions) (*UpdatesResponse, error) {
	params := url.Values{}
	params.Set("dato", since.Format("2006-01-02T15:04:05.000Z"))

	if opts != nil {
		if opts.Size > 0 {
			params.Set("size", strconv.Itoa(opts.Size))
		}
	}

	// Updates are not cached as they represent real-time data
	var result UpdatesResponse
	if err := c.doRequest(ctx, "/oppdateringer/enheter", params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdatesOptions configures the updates query
type UpdatesOptions struct {
	Size int
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
func (c *Client) doRequest(ctx context.Context, path string, params url.Values, result interface{}) error {
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
	reqURL := BaseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nordic-registry-mcp-server/1.0")

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
			c.logger.Warn("API request failed, retrying",
				"attempt", attempt+1,
				"path", path,
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
			return &NotFoundError{OrgNumber: path}
		}

		if resp.StatusCode >= 400 {
			// Check for rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
					if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
						select {
						case <-time.After(time.Duration(seconds) * time.Second):
						case <-ctx.Done():
							return ctx.Err()
						}
						continue
					}
				}
			}

			// Try to parse API error
			var apiErr APIError
			if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
				lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, apiErr.Message)
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
	OrgNumber string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("organization not found: %s", e.OrgNumber)
}

// normalizeOrgNumber removes spaces and dashes from organization number
func normalizeOrgNumber(orgNumber string) string {
	orgNumber = strings.ReplaceAll(orgNumber, " ", "")
	orgNumber = strings.ReplaceAll(orgNumber, "-", "")
	return orgNumber
}

// validateOrgNumber checks if the organization number is valid
func validateOrgNumber(orgNumber string) error {
	if len(orgNumber) != 9 {
		return fmt.Errorf("organization number must be 9 digits, got %d", len(orgNumber))
	}
	for _, c := range orgNumber {
		if c < '0' || c > '9' {
			return fmt.Errorf("organization number must contain only digits")
		}
	}
	return nil
}
