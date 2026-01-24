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

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/base"
	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// DefaultBaseURL is the PRH open data API base URL
	DefaultBaseURL = "https://avoindata.prh.fi/opendata-ytj-api/v3"

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 15 * time.Minute
)

// Client is a Finnish PRH API client with caching and resilience
type Client struct {
	*base.Client
	baseURL string
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

// NewClient creates a new Finnish PRH API client
func NewClient(opts ...ClientOption) *Client {
	return &Client{
		Client:  base.NewClient(opts...),
		baseURL: DefaultBaseURL,
	}
}

// WithBaseURL returns a Client with a custom base URL (for testing)
func (c *Client) WithBaseURL(url string) *Client {
	c.baseURL = url
	return c
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
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*CompanySearchResponse), nil
	}

	// Deduplicate concurrent requests
	result, _, err := c.Dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		return c.doSearch(ctx, params)
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*CompanySearchResponse)
	c.Cache.Set(cacheKey, resp, DefaultCacheTTL)
	return resp, nil
}

func (c *Client) doSearch(ctx context.Context, params url.Values) (*CompanySearchResponse, error) {
	reqURL := fmt.Sprintf("%s/companies?%s", c.baseURL, params.Encode())

	body, statusCode, err := c.Client.DoRequest(ctx, base.RequestConfig{URL: reqURL})
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		c.RecordSuccess() // Client errors don't indicate service issues
		return nil, fmt.Errorf("PRH API error: %d", statusCode)
	}

	var result CompanySearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.RecordSuccess()
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
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	// Deduplicate concurrent requests
	result, _, err := c.Dedup.Do(ctx, cacheKey, func() (interface{}, error) {
		return c.doGetCompany(ctx, normalized)
	})
	if err != nil {
		return nil, err
	}

	company := result.(*Company)
	c.Cache.Set(cacheKey, company, DefaultCacheTTL)
	return company, nil
}

func (c *Client) doGetCompany(ctx context.Context, businessID string) (*Company, error) {
	reqURL := fmt.Sprintf("%s/companies?businessId=%s", c.baseURL, url.QueryEscape(businessID))

	body, statusCode, err := c.Client.DoRequest(ctx, base.RequestConfig{URL: reqURL})
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		c.RecordSuccess() // Client errors don't indicate service issues
		return nil, fmt.Errorf("PRH API error: %d", statusCode)
	}

	var result CompanySearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Companies) == 0 {
		c.RecordSuccess()
		return nil, apierrors.NewNotFoundError("finland", businessID)
	}

	c.RecordSuccess()
	return &result.Companies[0], nil
}
