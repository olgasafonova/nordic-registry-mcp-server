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

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 5 * time.Minute

	// DefaultUserAgent is the default user agent for CVR API requests
	DefaultUserAgent = "nordic-registry-mcp-server/1.0 (github.com/olgasafonova/nordic-registry-mcp-server)"
)

// Client provides access to the Danish CVR API
type Client struct {
	*base.Client
	userAgent string
}

// ClientOption configures the Client (re-export base.ClientOption for compatibility)
type ClientOption = base.ClientOption

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(c *http.Client) ClientOption {
	return base.WithHTTPClient(c)
}

// WithLogger sets a custom logger
func WithLogger(l *slog.Logger) ClientOption {
	return base.WithLogger(l)
}

// WithCache sets a custom cache
func WithCache(c *infra.Cache) ClientOption {
	return base.WithCache(c)
}

// NewClient creates a new CVR API client
func NewClient(opts ...ClientOption) *Client {
	return &Client{
		Client:    base.NewClient(opts...),
		userAgent: DefaultUserAgent,
	}
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

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// GetCompany retrieves a company by CVR number
func (c *Client) GetCompany(ctx context.Context, cvr string) (*Company, error) {
	cvr = normalizeCVR(cvr)
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
	reqURL := BaseURL + "?" + params.Encode()

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
