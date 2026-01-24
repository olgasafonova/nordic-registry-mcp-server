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
	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// BaseURL is the Brønnøysundregistrene API endpoint
	BaseURL = "https://data.brreg.no/enhetsregisteret/api"

	// SearchCacheTTL for search results (shorter to ensure fresher results)
	SearchCacheTTL = 2 * time.Minute

	// DefaultCacheTTL for company details and other cached responses
	DefaultCacheTTL = 5 * time.Minute
)

// Client provides access to the Norwegian Brønnøysundregistrene API
type Client struct {
	*base.Client
	baseURL string
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

// NewClient creates a new Brønnøysundregistrene client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		Client:  base.NewClient(),
		baseURL: BaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
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
		if opts.RegisteredInVoluntary != nil {
			params.Set("registrertIFrivillighetsregisteret", strconv.FormatBool(*opts.RegisteredInVoluntary))
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

	c.Cache.Set(cacheKey, &result, SearchCacheTTL)
	return &result, nil
}

// SearchOptions configures company search
type SearchOptions struct {
	Page                  int
	Size                  int
	OrgForm               string // Organization form code (AS, ENK, etc.)
	Municipality          string // Kommune number
	RegisteredInVAT       *bool
	Bankrupt              *bool
	RegisteredInVoluntary *bool // Filter for voluntary organizations (Frivillighetsregisteret)
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

// SearchSubUnitsOptions configures subunit search
type SearchSubUnitsOptions struct {
	Page         int
	Size         int
	Municipality string
}

// SearchSubUnits searches for sub-units by name
func (c *Client) SearchSubUnits(ctx context.Context, query string, opts *SearchSubUnitsOptions) (*SubUnitSearchResponse, error) {
	params := url.Values{}
	params.Set("navn", query)

	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			params.Set("size", strconv.Itoa(opts.Size))
		}
		if opts.Municipality != "" {
			params.Set("kommunenummer", opts.Municipality)
		}
	}

	cacheKey := "search_subunits:" + params.Encode()
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*SubUnitSearchResponse), nil
	}

	var result SubUnitSearchResponse
	if err := c.doRequest(ctx, "/underenheter", params, &result); err != nil {
		return nil, err
	}

	c.Cache.Set(cacheKey, &result, SearchCacheTTL)
	return &result, nil
}

// GetMunicipalities retrieves the list of Norwegian municipalities
func (c *Client) GetMunicipalities(ctx context.Context) (*MunicipalitiesResponse, error) {
	cacheKey := "municipalities"
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*MunicipalitiesResponse), nil
	}

	var result MunicipalitiesResponse
	if err := c.doRequest(ctx, "/kommuner", nil, &result); err != nil {
		return nil, err
	}

	// Cache for longer since this data rarely changes
	c.Cache.Set(cacheKey, &result, 24*time.Hour)
	return &result, nil
}

// GetOrgForms retrieves the list of organization forms (AS, ENK, etc.)
func (c *Client) GetOrgForms(ctx context.Context) (*OrgFormsResponse, error) {
	cacheKey := "orgforms"
	if cached, ok := c.Cache.Get(cacheKey); ok {
		return cached.(*OrgFormsResponse), nil
	}

	var result OrgFormsResponse
	if err := c.doRequest(ctx, "/organisasjonsformer", nil, &result); err != nil {
		return nil, err
	}

	// Cache for longer since this data rarely changes
	c.Cache.Set(cacheKey, &result, 24*time.Hour)
	return &result, nil
}

// BatchGetCompanies retrieves multiple companies by organization numbers in one request.
// The API supports up to 2000 org numbers per request.
func (c *Client) BatchGetCompanies(ctx context.Context, orgNumbers []string) (*SearchResponse, error) {
	if len(orgNumbers) == 0 {
		return &SearchResponse{}, nil
	}
	if len(orgNumbers) > 2000 {
		return nil, fmt.Errorf("maximum 2000 organization numbers per request, got %d", len(orgNumbers))
	}

	// Normalize and validate all org numbers
	normalized := make([]string, 0, len(orgNumbers))
	for _, on := range orgNumbers {
		on = normalizeOrgNumber(on)
		if err := validateOrgNumber(on); err != nil {
			return nil, fmt.Errorf("invalid org number %q: %w", on, err)
		}
		normalized = append(normalized, on)
	}

	params := url.Values{}
	params.Set("organisasjonsnummer", strings.Join(normalized, ","))

	cacheKey := "batch:" + strings.Join(normalized, ",")
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

// GetSubUnitUpdates retrieves recent updates to sub-units from the registry
func (c *Client) GetSubUnitUpdates(ctx context.Context, since time.Time, opts *UpdatesOptions) (*SubUnitUpdatesResponse, error) {
	params := url.Values{}
	params.Set("dato", since.Format("2006-01-02T15:04:05.000Z"))

	if opts != nil {
		if opts.Size > 0 {
			params.Set("size", strconv.Itoa(opts.Size))
		}
	}

	// Updates are not cached as they represent real-time data
	var result SubUnitUpdatesResponse
	if err := c.doRequest(ctx, "/oppdateringer/underenheter", params, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// doRequest performs an HTTP request using the base client infrastructure
func (c *Client) doRequest(ctx context.Context, path string, params url.Values, result interface{}) error {
	// Build URL
	reqURL := c.baseURL + path
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
		return apierrors.NewNotFoundError("norway", path)
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
