package sweden

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

const (
	// API endpoints
	defaultBaseURL  = "https://gw.api.bolagsverket.se/vardefulla-datamangder/v1"
	defaultTokenURL = "https://portal.api.bolagsverket.se/oauth2/token" // #nosec G101 -- public OAuth endpoint URL, not credentials

	// OAuth2 scopes
	scopeRead = "vardefulla-datamangder:read"
	scopePing = "vardefulla-datamangder:ping"

	// Environment variable names
	envClientID     = "BOLAGSVERKET_CLIENT_ID"
	envClientSecret = "BOLAGSVERKET_CLIENT_SECRET" // #nosec G101 -- env var name, not actual secret

	// Timeouts and retries
	defaultTimeout     = 30 * time.Second
	tokenRefreshMargin = 5 * time.Minute // Refresh token 5 minutes before expiry

	// Cache TTLs
	companyCacheTTL  = 15 * time.Minute
	documentCacheTTL = 30 * time.Minute

	// Size limits
	maxDocumentSize = 100 * 1024 * 1024 // 100 MB - prevents memory exhaustion from malicious responses
)

// Client provides access to the Swedish business registry (Bolagsverket).
type Client struct {
	httpClient   *http.Client
	baseURL      string
	tokenURL     string
	clientID     string
	clientSecret string

	// Token management
	tokenMu     sync.RWMutex
	accessToken string
	tokenExpiry time.Time

	// Resilience infrastructure
	cache          *infra.Cache
	circuitBreaker *infra.CircuitBreaker
	dedup          *infra.RequestDeduplicator
}

// ClientOption configures the client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTokenURL sets a custom token URL (useful for testing).
func WithTokenURL(url string) ClientOption {
	return func(c *Client) {
		c.tokenURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithCredentials sets OAuth2 credentials directly (instead of from env vars).
func WithCredentials(clientID, clientSecret string) ClientOption {
	return func(c *Client) {
		c.clientID = clientID
		c.clientSecret = clientSecret
	}
}

// NewClient creates a new Sweden registry client.
// Credentials are read from BOLAGSVERKET_CLIENT_ID and BOLAGSVERKET_CLIENT_SECRET
// environment variables unless provided via WithCredentials option.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient:     &http.Client{Timeout: defaultTimeout},
		baseURL:        defaultBaseURL,
		tokenURL:       defaultTokenURL,
		clientID:       os.Getenv(envClientID),
		clientSecret:   os.Getenv(envClientSecret),
		cache:          infra.NewCache(500),
		circuitBreaker: infra.NewCircuitBreaker(),
		dedup:          infra.NewRequestDeduplicator(),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.clientID == "" || c.clientSecret == "" {
		return nil, fmt.Errorf("sweden: missing OAuth2 credentials; set %s and %s environment variables", envClientID, envClientSecret)
	}

	return c, nil
}

// Close releases resources and clears sensitive data from memory.
func (c *Client) Close() {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Clear token from memory
	c.accessToken = ""
	c.tokenExpiry = time.Time{}

	// Close cache
	if c.cache != nil {
		c.cache.Close()
	}
}

// IsConfigured returns true if OAuth2 credentials are available.
func IsConfigured() bool {
	return os.Getenv(envClientID) != "" && os.Getenv(envClientSecret) != ""
}

// getToken returns a valid access token, refreshing if necessary.
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Add(tokenRefreshMargin).Before(c.tokenExpiry) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// Need to refresh token
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" && time.Now().Add(tokenRefreshMargin).Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// Request new token
	token, err := c.refreshToken(ctx)
	if err != nil {
		return "", err
	}

	return token, nil
}

// refreshToken requests a new OAuth2 access token.
func (c *Client) refreshToken(ctx context.Context) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", scopeRead+" "+scopePing)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("sweden: creating token request: %w", err)
	}

	// Basic auth with client credentials
	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sweden: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sweden: token request returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("sweden: decoding token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.accessToken, nil
}

// doRequest performs an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any) ([]byte, error) {
	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		return nil, errors.New("sweden: circuit breaker open")
	}

	token, err := c.getToken(ctx)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("sweden: marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	reqURL := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("sweden: creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.circuitBreaker.RecordFailure()
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Detail != "" {
			return nil, fmt.Errorf("sweden: API error %d: %s - %s", apiErr.Status, apiErr.Title, apiErr.Detail)
		}
		return nil, fmt.Errorf("sweden: request returned %d: %s", resp.StatusCode, string(respBody))
	}

	c.circuitBreaker.RecordSuccess()
	return respBody, nil
}

// NormalizeOrgNumber normalizes a Swedish organization number.
// Swedish org numbers are 10 digits (NNNNNNNNNN) or 12 digits for personnummer.
// This removes any dashes or spaces.
func NormalizeOrgNumber(orgNumber string) string {
	// Remove common separators
	orgNumber = strings.ReplaceAll(orgNumber, "-", "")
	orgNumber = strings.ReplaceAll(orgNumber, " ", "")
	return strings.TrimSpace(orgNumber)
}

// GetCompany retrieves company information by organization number.
func (c *Client) GetCompany(ctx context.Context, orgNumber string) (*OrganisationerSvar, error) {
	orgNumber = NormalizeOrgNumber(orgNumber)
	if orgNumber == "" {
		return nil, errors.New("sweden: organization number is required")
	}

	// Check cache
	cacheKey := "company:" + orgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		if result, ok := cached.(*OrganisationerSvar); ok {
			return result, nil
		}
	}

	// Deduplicate concurrent requests
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (any, error) {
		reqBody := OrganisationerBegaran{
			Identitetsbeteckning: orgNumber,
		}

		respBody, err := c.doRequest(ctx, http.MethodPost, "/organisationer", reqBody)
		if err != nil {
			return nil, err
		}

		var result OrganisationerSvar
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("sweden: decoding company response: %w", err)
		}

		// Cache the result
		c.cache.Set(cacheKey, &result, companyCacheTTL)

		return &result, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*OrganisationerSvar), nil
}

// GetDocumentList retrieves the list of annual reports for a company.
func (c *Client) GetDocumentList(ctx context.Context, orgNumber string) (*DokumentlistaSvar, error) {
	orgNumber = NormalizeOrgNumber(orgNumber)
	if orgNumber == "" {
		return nil, errors.New("sweden: organization number is required")
	}

	// Check cache
	cacheKey := "doclist:" + orgNumber
	if cached, ok := c.cache.Get(cacheKey); ok {
		if result, ok := cached.(*DokumentlistaSvar); ok {
			return result, nil
		}
	}

	// Deduplicate concurrent requests
	result, _, err := c.dedup.Do(ctx, cacheKey, func() (any, error) {
		reqBody := DokumentlistaBegaran{
			Identitetsbeteckning: orgNumber,
		}

		respBody, err := c.doRequest(ctx, http.MethodPost, "/dokumentlista", reqBody)
		if err != nil {
			return nil, err
		}

		var result DokumentlistaSvar
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("sweden: decoding document list response: %w", err)
		}

		// Cache the result
		c.cache.Set(cacheKey, &result, documentCacheTTL)

		return &result, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*DokumentlistaSvar), nil
}

// DownloadDocument downloads an annual report by document ID.
// Returns the raw ZIP file contents.
func (c *Client) DownloadDocument(ctx context.Context, documentID string) ([]byte, error) {
	if documentID == "" {
		return nil, errors.New("sweden: document ID is required")
	}

	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		return nil, errors.New("sweden: circuit breaker open")
	}

	token, err := c.getToken(ctx)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, err
	}

	reqURL := c.baseURL + "/dokument/" + url.PathEscape(documentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("sweden: creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/zip")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.circuitBreaker.RecordFailure()
		body, _ := io.ReadAll(resp.Body)
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Detail != "" {
			return nil, fmt.Errorf("sweden: API error %d: %s - %s", apiErr.Status, apiErr.Title, apiErr.Detail)
		}
		return nil, fmt.Errorf("sweden: request returned %d: %s", resp.StatusCode, string(body))
	}

	// Limit response size to prevent memory exhaustion
	limitedReader := io.LimitReader(resp.Body, maxDocumentSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: reading document: %w", err)
	}
	if len(data) > maxDocumentSize {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("sweden: document exceeds maximum size of %d bytes", maxDocumentSize)
	}

	c.circuitBreaker.RecordSuccess()
	return data, nil
}

// IsAlive checks if the API is available.
func (c *Client) IsAlive(ctx context.Context) (bool, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/isalive", nil)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(respBody)) == "OK", nil
}

// CircuitBreakerStatus returns the current circuit breaker state.
func (c *Client) CircuitBreakerStatus() string {
	return c.circuitBreaker.State().String()
}

// CircuitBreakerStats returns detailed circuit breaker statistics.
func (c *Client) CircuitBreakerStats() infra.CircuitBreakerStats {
	return c.circuitBreaker.Stats()
}

// CacheSize returns the current number of cached entries.
func (c *Client) CacheSize() int64 {
	return c.cache.Size()
}
