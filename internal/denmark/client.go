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

// SearchCompany searches for a company by name
func (c *Client) SearchCompany(ctx context.Context, query string) (*Company, error) {
	params := url.Values{}
	params.Set("search", query)
	params.Set("country", "dk")

	cacheKey := "search:" + params.Encode()
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	var result Company
	if err := c.doRequest(ctx, params, &result); err != nil {
		return nil, err
	}

	// Check if we got a valid result
	if result.CVR == 0 {
		return nil, apierrors.NewNotFoundError("denmark", query)
	}

	c.Cache.Set(cacheKey, &result, SearchCacheTTL)
	return &result, nil
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

	params := url.Values{}
	params.Set("produ", pnumber)
	params.Set("country", "dk")

	cacheKey := "pnumber:" + pnumber
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	var result Company
	if err := c.doRequest(ctx, params, &result); err != nil {
		return nil, err
	}

	// Check if we got a valid result
	if result.CVR == 0 {
		return nil, apierrors.NewNotFoundError("denmark", "pnumber:"+pnumber)
	}

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
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

	params := url.Values{}
	params.Set("phone", phone)
	params.Set("country", "dk")

	cacheKey := "phone:" + phone
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	var result Company
	if err := c.doRequest(ctx, params, &result); err != nil {
		return nil, err
	}

	// Check if we got a valid result
	if result.CVR == 0 {
		return nil, apierrors.NewNotFoundError("denmark", "phone:"+phone)
	}

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
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
	// Build URL
	reqURL := c.baseURL + "?" + params.Encode()

	body, statusCode, err := c.Client.DoRequest(ctx, base.RequestConfig{
		URL:       reqURL,
		UserAgent: c.userAgent,
	})
	if err != nil {
		return err
	}

	// Handle HTTP errors
	if statusCode == http.StatusNotFound {
		c.RecordSuccess()
		return apierrors.NewNotFoundError("denmark", notFoundIdentifier(params))
	}

	if statusCode >= 400 {
		// Try to parse API error
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
			// Check for "not found" type errors
			if apiErr.T == 1 || strings.Contains(strings.ToLower(apiErr.Error), "not found") {
				c.RecordSuccess()
				return apierrors.NewNotFoundError("denmark", notFoundIdentifier(params))
			}
			c.RecordSuccess() // Client errors don't indicate service issues
			return fmt.Errorf("API error %d: %s", statusCode, apiErr.String())
		}
		c.RecordSuccess()
		return fmt.Errorf("API error %d: %s", statusCode, string(body))
	}

	// Check for API error in successful response (CVR API sometimes returns 200 with error)
	var apiErr APIError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		if apiErr.T == 1 || strings.Contains(strings.ToLower(apiErr.Error), "not found") {
			c.RecordSuccess()
			return apierrors.NewNotFoundError("denmark", notFoundIdentifier(params))
		}
	}

	// Parse success response
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.RecordSuccess()
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
