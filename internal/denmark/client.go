package denmark

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/base"
	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// BaseURL is the CVR API endpoint
	BaseURL = "https://cvrapi.dk/api"

	// SearchCacheTTL for search results (shorter to ensure fresher results)
	SearchCacheTTL = 2 * time.Minute

	// DefaultCacheTTL for company details and other cached responses
	DefaultCacheTTL = 5 * time.Minute

	// DefaultUserAgent is the default user agent for CVR API requests
	DefaultUserAgent = "nordic-registry-mcp-server/1.0 (github.com/olgasafonova/nordic-registry-mcp-server)"
)

// Client provides access to the Danish CVR API
type Client struct {
	*base.Client
	baseURL   string
	userAgent string
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

// WithBaseURL sets a custom base URL (for testing)
func WithBaseURL(url string) ClientOption {
	return func(client *Client) {
		client.baseURL = url
	}
}

// NewClient creates a new CVR API client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		Client:    base.NewClient(),
		baseURL:   BaseURL,
		userAgent: DefaultUserAgent,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// lookupRequest describes a single-company CVR lookup: the upstream query
// params, the cache key/TTL, and the identifier to report when not found.
type lookupRequest struct {
	params     url.Values
	cacheKey   string
	notFoundID string
	ttl        time.Duration
}

// cachedLookup performs a cached single-company CVR request: it returns a
// cached hit under req.cacheKey, otherwise issues the request, treats a zero
// CVR as not-found (reported against req.notFoundID), and caches the hit for
// req.ttl. All single-result Denmark lookups (search, P-number, phone) share
// this flow; only the request descriptor differs.
func (c *Client) cachedLookup(ctx context.Context, req lookupRequest) (*Company, error) {
	if cached, ok := c.Cache.Get(req.cacheKey); ok {
		return cached.(*Company), nil
	}

	var result Company
	if err := c.doRequest(ctx, req.params, &result); err != nil {
		return nil, err
	}

	// Check if we got a valid result
	if result.CVR == 0 {
		return nil, apierrors.NewNotFoundError("denmark", req.notFoundID)
	}

	c.Cache.Set(req.cacheKey, &result, req.ttl)
	return &result, nil
}

// SearchCompany searches for a company by name
func (c *Client) SearchCompany(ctx context.Context, query string) (*Company, error) {
	params := url.Values{}
	params.Set("search", query)
	params.Set("country", "dk")

	return c.cachedLookup(ctx, lookupRequest{
		params:     params,
		cacheKey:   "search:" + params.Encode(),
		notFoundID: query,
		ttl:        SearchCacheTTL,
	})
}

// lookupByParam performs a cached single-key CVR lookup keyed on one query
// parameter, caching under "<cacheKind>:<value>". GetByPNumber and
// SearchByPhone share this flow; only the parameter name and cache prefix
// differ.
func (c *Client) lookupByParam(ctx context.Context, paramKey, value, cacheKind string) (*Company, error) {
	params := url.Values{}
	params.Set(paramKey, value)
	params.Set("country", "dk")

	cacheKey := cacheKind + ":" + value
	return c.cachedLookup(ctx, lookupRequest{
		params:     params,
		cacheKey:   cacheKey,
		notFoundID: cacheKey,
		ttl:        DefaultCacheTTL,
	})
}

// GetByPNumber retrieves a company by production unit P-number
func (c *Client) GetByPNumber(ctx context.Context, pnumber string) (*Company, error) {
	// Normalize P-number - remove spaces and dashes
	pnumber = strings.ReplaceAll(pnumber, " ", "")
	pnumber = strings.ReplaceAll(pnumber, "-", "")

	// Validate after normalization
	if err := ValidatePNumber(pnumber); err != nil {
		return nil, err
	}

	return c.lookupByParam(ctx, "produ", pnumber, "pnumber")
}

// SearchByPhone searches for a company by phone number
func (c *Client) SearchByPhone(ctx context.Context, phone string) (*Company, error) {
	// Normalize phone number - remove spaces and common formatting
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "+45", "")

	// Validate after normalization
	if err := ValidatePhone(phone); err != nil {
		return nil, err
	}

	return c.lookupByParam(ctx, "phone", phone, "phone")
}

// GetCompany retrieves a company by CVR number
func (c *Client) GetCompany(ctx context.Context, cvr string) (*Company, error) {
	cvr = NormalizeCVR(cvr)
	if err := validateCVR(cvr); err != nil {
		return nil, err
	}

	cacheKey := "company:" + cvr
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	// Use deduplication to avoid duplicate requests for the same CVR
	result, _, err := c.Dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		params := url.Values{}
		params.Set("vat", cvr)
		params.Set("country", "dk")

		var company Company
		if err := c.doRequest(ctx, params, &company); err != nil {
			return nil, err
		}

		// Check if we got a valid result
		if company.CVR == 0 {
			return nil, apierrors.NewNotFoundError("denmark", cvr)
		}

		return &company, nil
	})

	if err != nil {
		return nil, err
	}

	company := result.(*Company)
	c.Cache.Set(cacheKey, company, DefaultCacheTTL)
	return company, nil
}

// doRequest performs an HTTP request using the base client infrastructure
func (c *Client) doRequest(ctx context.Context, params url.Values, result interface{}) error {
	reqURL := c.baseURL + "?" + params.Encode()

	body, statusCode, err := c.Client.DoRequest(ctx, base.RequestConfig{
		URL:       reqURL,
		UserAgent: c.userAgent,
	})
	if err != nil {
		return err
	}

	if errResp := c.classifyError(statusCode, body, params); errResp != nil {
		return errResp
	}

	// 200 with embedded CVR error envelope (the CVR API occasionally
	// returns success status alongside an error body).
	if notFound := c.checkEmbeddedNotFound(body, params); notFound != nil {
		return notFound
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.RecordSuccess()
	return nil
}

// classifyError translates HTTP error status codes into Denmark-specific
// errors. Returns nil when the status code is in the success range, allowing
// the caller to proceed with response parsing.
func (c *Client) classifyError(statusCode int, body []byte, params url.Values) error {
	if statusCode == http.StatusNotFound {
		c.RecordSuccess()
		return apierrors.NewNotFoundError("denmark", notFoundIdentifier(params))
	}

	if statusCode < 400 {
		return nil
	}

	// CVR returns a documented JSON error envelope; preserve apiErr.String()
	// because it's the user-facing diagnostic.
	var apiErr APIError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		if apiErr.T == 1 || strings.Contains(strings.ToLower(apiErr.Error), "not found") {
			c.RecordSuccess()
			return apierrors.NewNotFoundError("denmark", notFoundIdentifier(params))
		}
		c.RecordSuccess() // Client errors don't indicate service issues
		return fmt.Errorf("API error %d: %s", statusCode, apiErr.String())
	}

	c.RecordSuccess()
	// SECURITY (HG-2): unparsed body fallback — truncate to bound the
	// blast radius. CVR's documented envelope is handled above; this
	// path catches HTML 4xx pages, proxy errors, and any non-JSON
	// upstream response. Body comes from a public registry so it
	// isn't credentials, but unbounded HTML / stack traces still leak
	// into the MCP caller's error otherwise.
	return fmt.Errorf("API error %d: %s", statusCode, truncateBody(body))
}

// checkEmbeddedNotFound looks for a CVR "not found" error embedded in a 2xx
// response body and returns the corresponding error. Returns nil if the body
// is a normal success response.
func (c *Client) checkEmbeddedNotFound(body []byte, params url.Values) error {
	var apiErr APIError
	if json.Unmarshal(body, &apiErr) != nil || apiErr.Error == "" {
		return nil
	}
	if apiErr.T == 1 || strings.Contains(strings.ToLower(apiErr.Error), "not found") {
		c.RecordSuccess()
		return apierrors.NewNotFoundError("denmark", notFoundIdentifier(params))
	}
	return nil
}

// notFoundIdentifier extracts the best identifier from request params
func notFoundIdentifier(params url.Values) string {
	if cvr := params.Get("vat"); cvr != "" {
		return cvr
	}
	return params.Get("search")
}

// NormalizeCVR removes spaces, dashes, and DK prefix from a Danish CVR number.
// This allows users to input "DK-24256790", "DK24256790", or "24 25 67 90" which get normalized to "24256790".
func NormalizeCVR(cvr string) string {
	cvr = strings.TrimSpace(cvr)
	cvr = strings.ReplaceAll(cvr, " ", "")
	cvr = strings.ReplaceAll(cvr, "-", "")
	cvr = strings.TrimPrefix(cvr, "DK")
	cvr = strings.TrimPrefix(cvr, "dk")
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

// maxBodyInError caps how many bytes of an unparsed upstream body land in
// caller-facing error messages. See HG-2 in rules/review-patterns.md.
const maxBodyInError = 256

// truncateBody bounds the blast radius of a non-JSON-envelope upstream body
// in error messages. CVR normally returns a documented JSON error envelope
// handled above the fallback; this guard keeps errant HTML 4xx pages,
// upstream stack traces, and proxy interstitials from flowing unbounded
// into the MCP caller's error string.
func truncateBody(body []byte) string {
	if len(body) <= maxBodyInError {
		return string(body)
	}
	return string(body[:maxBodyInError]) + "..."
}
