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

// setPositiveInt sets key when v is greater than zero.
func setPositiveInt(params url.Values, key string, v int) {
	if v > 0 {
		params.Set(key, strconv.Itoa(v))
	}
}

// setNonEmpty sets key when v is a non-empty string.
func setNonEmpty(params url.Values, key, v string) {
	if v != "" {
		params.Set(key, v)
	}
}

// setBoolPtr sets key when v is non-nil.
func setBoolPtr(params url.Values, key string, v *bool) {
	if v != nil {
		params.Set(key, strconv.FormatBool(*v))
	}
}

// cachedFetch describes a cache-backed GET: where to look in the cache,
// which endpoint to hit on a miss, and how long to keep the result.
type cachedFetch struct {
	key    string
	path   string
	params url.Values
	ttl    time.Duration
}

// getCached fetches a resource through the cache: return the cached value
// when present, otherwise perform the request and cache the result.
func getCached[T any](ctx context.Context, c *Client, req cachedFetch) (*T, error) {
	if cached, ok := c.Cache.Get(req.key); ok {
		return cached.(*T), nil
	}

	var result T
	if err := c.doRequest(ctx, req.path, req.params, &result); err != nil {
		return nil, err
	}

	c.Cache.Set(req.key, &result, req.ttl)
	return &result, nil
}

// updateParams builds the query parameters shared by the registry's
// update feeds.
func updateParams(since time.Time, opts *UpdatesOptions) url.Values {
	params := url.Values{}
	params.Set("dato", since.Format("2006-01-02T15:04:05.000Z"))
	if opts != nil {
		setPositiveInt(params, "size", opts.Size)
	}
	return params
}

// fetchFresh performs an uncached request; update feeds use it because
// they represent real-time data.
func fetchFresh[T any](ctx context.Context, c *Client, path string, params url.Values) (*T, error) {
	var result T
	if err := c.doRequest(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SearchCompanies searches for companies by name or other criteria
func (c *Client) SearchCompanies(ctx context.Context, query string, opts *SearchOptions) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("navn", query)
	opts.apply(params)

	return getCached[SearchResponse](ctx, c, cachedFetch{key: "search:" + params.Encode(), path: "/enheter", params: params, ttl: SearchCacheTTL})
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

// apply copies the set options into params; a nil receiver is a no-op.
func (o *SearchOptions) apply(params url.Values) {
	if o == nil {
		return
	}
	setPositiveInt(params, "page", o.Page)
	setPositiveInt(params, "size", o.Size)
	setNonEmpty(params, "organisasjonsform", o.OrgForm)
	setNonEmpty(params, "kommunenummer", o.Municipality)
	setBoolPtr(params, "registrertIMvaregisteret", o.RegisteredInVAT)
	setBoolPtr(params, "konkurs", o.Bankrupt)
	setBoolPtr(params, "registrertIFrivillighetsregisteret", o.RegisteredInVoluntary)
}

// GetCompany retrieves a company by organization number
func (c *Client) GetCompany(ctx context.Context, orgNumber string) (*Company, error) {
	orgNumber = NormalizeOrgNumber(orgNumber)
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
	orgNumber = NormalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	return getCached[RolesResponse](ctx, c, cachedFetch{key: "roles:" + orgNumber, path: "/enheter/" + orgNumber + "/roller", ttl: DefaultCacheTTL})
}

// GetSubUnits retrieves sub-units (branches) for a parent company
func (c *Client) GetSubUnits(ctx context.Context, parentOrgNumber string) (*SubUnitSearchResponse, error) {
	parentOrgNumber = NormalizeOrgNumber(parentOrgNumber)
	if err := validateOrgNumber(parentOrgNumber); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("overordnetEnhet", parentOrgNumber)

	return getCached[SubUnitSearchResponse](ctx, c, cachedFetch{key: "subunits:" + parentOrgNumber, path: "/underenheter", params: params, ttl: DefaultCacheTTL})
}

// GetSubUnit retrieves a specific sub-unit by organization number
func (c *Client) GetSubUnit(ctx context.Context, orgNumber string) (*SubUnit, error) {
	orgNumber = NormalizeOrgNumber(orgNumber)
	if err := validateOrgNumber(orgNumber); err != nil {
		return nil, err
	}

	return getCached[SubUnit](ctx, c, cachedFetch{key: "subunit:" + orgNumber, path: "/underenheter/" + orgNumber, ttl: DefaultCacheTTL})
}

// GetUpdates retrieves recent updates from the registry
func (c *Client) GetUpdates(ctx context.Context, since time.Time, opts *UpdatesOptions) (*UpdatesResponse, error) {
	return fetchFresh[UpdatesResponse](ctx, c, "/oppdateringer/enheter", updateParams(since, opts))
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
		setPositiveInt(params, "page", opts.Page)
		setPositiveInt(params, "size", opts.Size)
		setNonEmpty(params, "kommunenummer", opts.Municipality)
	}

	return getCached[SubUnitSearchResponse](ctx, c, cachedFetch{key: "search_subunits:" + params.Encode(), path: "/underenheter", params: params, ttl: SearchCacheTTL})
}

// GetMunicipalities retrieves the list of Norwegian municipalities.
// Uses size=500 to fetch all municipalities in one request (Norway has ~365).
// Cached for 24h since this data rarely changes.
func (c *Client) GetMunicipalities(ctx context.Context) (*MunicipalitiesResponse, error) {
	// Request all municipalities at once (default page size is 20)
	params := url.Values{"size": {"500"}}
	return getCached[MunicipalitiesResponse](ctx, c, cachedFetch{key: "municipalities", path: "/kommuner", params: params, ttl: 24 * time.Hour})
}

// GetOrgForms retrieves the list of organization forms (AS, ENK, etc.).
// Cached for 24h since this data rarely changes.
func (c *Client) GetOrgForms(ctx context.Context) (*OrgFormsResponse, error) {
	return getCached[OrgFormsResponse](ctx, c, cachedFetch{key: "orgforms", path: "/organisasjonsformer", ttl: 24 * time.Hour})
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
		on = NormalizeOrgNumber(on)
		if err := validateOrgNumber(on); err != nil {
			return nil, fmt.Errorf("invalid org number %q: %w", on, err)
		}
		normalized = append(normalized, on)
	}

	params := url.Values{}
	params.Set("organisasjonsnummer", strings.Join(normalized, ","))

	return getCached[SearchResponse](ctx, c, cachedFetch{key: "batch:" + strings.Join(normalized, ","), path: "/enheter", params: params, ttl: DefaultCacheTTL})
}

// GetSubUnitUpdates retrieves recent updates to sub-units from the registry
func (c *Client) GetSubUnitUpdates(ctx context.Context, since time.Time, opts *UpdatesOptions) (*SubUnitUpdatesResponse, error) {
	return fetchFresh[SubUnitUpdatesResponse](ctx, c, "/oppdateringer/underenheter", updateParams(since, opts))
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
		// Try to parse API error envelope first (Brønnøysundregistrene
		// returns a documented JSON error shape; preserve apiErr.Message
		// because it's the user-facing diagnostic).
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			c.RecordSuccess() // Client errors don't indicate service issues
			return fmt.Errorf("API error %d: %s", statusCode, apiErr.Message)
		}
		c.RecordSuccess()
		// SECURITY (HG-2): unparsed body fallback — truncate to bound the
		// blast radius. The body comes from a public registry API so it
		// isn't credentials, but a 4xx HTML page from a Cloudflare/proxy
		// or an upstream stack trace can still push unbounded content
		// (10 MB MaxResponseSize) into the MCP error.
		return fmt.Errorf("API error %d: %s", statusCode, truncateBody(body))
	}

	// Parse success response
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	c.RecordSuccess()
	return nil
}

// NormalizeOrgNumber removes spaces and dashes from a Norwegian organization number.
// This allows users to input "923 609 016" or "923-609-016" which get normalized to "923609016".
func NormalizeOrgNumber(orgNumber string) string {
	orgNumber = strings.TrimSpace(orgNumber)
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

// maxBodyInError caps how many bytes of an unparsed upstream body land in
// caller-facing error messages. See HG-2 in rules/review-patterns.md.
const maxBodyInError = 256

// truncateBody bounds the blast radius of a non-JSON-envelope upstream body
// in error messages. Brønnøysundregistrene normally returns a documented
// JSON error envelope handled above the fallback; this guard keeps errant
// HTML 4xx pages, Cloudflare interstitials, and upstream stack traces from
// flowing unbounded into the MCP caller's error string.
func truncateBody(body []byte) string {
	if len(body) <= maxBodyInError {
		return string(body)
	}
	return string(body[:maxBodyInError]) + "..."
}
