package norway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/base"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// BaseURL is the Brønnøysundregistrene API endpoint
	BaseURL = "https://data.brreg.no/enhetsregisteret/api"

	// DefaultCacheTTL for cached responses
	DefaultCacheTTL = 5 * time.Minute
)

// Client provides access to the Norwegian Brønnøysundregistrene API
type Client struct {
	*base.Client
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

// NewClient creates a new Brønnøysundregistrene client
func NewClient(opts ...ClientOption) *Client {
	return &Client{Client: base.NewClient(opts...)}
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
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*SearchResponse), nil
	}

	var result SearchResponse
	if err := c.doRequest(ctx, "/enheter", params, &result); err != nil {
		return nil, err
	}

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
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
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*Company), nil
	}

	// Use deduplication to avoid duplicate requests for the same org number
	result, _, err := c.Dedup.Do(ctx, cacheKey, func() (interface{}, error) {
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
	c.Cache.Set(cacheKey, company, DefaultCacheTTL)
	return company, nil
}

// GetRoles retrieves board members and other roles for a company
func (c *Client) GetRoles(ctx context.Context, orgNumber string) (*RolesResponse, error) {
	orgNumber = normalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	cacheKey := "roles:" + orgNumber
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*RolesResponse), nil
	}

	var result RolesResponse
	if err := c.doRequest(ctx, "/enheter/"+orgNumber+"/roller", nil, &result); err != nil {
		return nil, err
	}

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
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
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*SubUnitSearchResponse), nil
	}

	var result SubUnitSearchResponse
	if err := c.doRequest(ctx, "/underenheter", params, &result); err != nil {
		return nil, err
	}

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
	return &result, nil
}

// GetSubUnit retrieves a specific sub-unit by organization number
func (c *Client) GetSubUnit(ctx context.Context, orgNumber string) (*SubUnit, error) {
	orgNumber = normalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	cacheKey := "subunit:" + orgNumber
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*SubUnit), nil
	}

	var result SubUnit
	if err := c.doRequest(ctx, "/underenheter/"+orgNumber, nil, &result); err != nil {
		return nil, err
	}

	c.Cache.Set(cacheKey, &result, DefaultCacheTTL)
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

// doRequest performs an HTTP request using the base client infrastructure
func (c *Client) doRequest(ctx context.Context, path string, params url.Values, result interface{}) error {
	// Build URL
	reqURL := BaseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	body, statusCode, err := c.Client.DoRequest(ctx, base.RequestConfig{URL: reqURL})
	if err != nil {
		return err
	}

	// Handle HTTP errors
	if statusCode == http.StatusNotFound {
		c.RecordSuccess()
		return &NotFoundError{OrgNumber: path}
	}

	if statusCode >= 400 {
		// Try to parse API error
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			c.RecordSuccess() // Client errors don't indicate service issues
			return fmt.Errorf("API error %d: %s", statusCode, apiErr.Message)
		}
		c.RecordSuccess()
		return fmt.Errorf("API error %d: %s", statusCode, string(body))
	}

	// Parse success response
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.RecordSuccess()
	return nil
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
