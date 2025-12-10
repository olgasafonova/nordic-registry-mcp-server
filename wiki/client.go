package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client handles communication with the MediaWiki API
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *slog.Logger

	// Authentication state
	mu          sync.RWMutex
	loggedIn    bool
	csrfToken   string
	tokenExpiry time.Time
}

// NewClient creates a new MediaWiki API client
func NewClient(config *Config, logger *slog.Logger) *Client {
	jar, _ := cookiejar.New(nil)

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Jar:     jar,
		},
		logger: logger,
	}
}

// apiRequest makes a request to the MediaWiki API
func (c *Client) apiRequest(ctx context.Context, params url.Values) (map[string]interface{}, error) {
	params.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.config.UserAgent)

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * 100 * time.Millisecond)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Check for API errors
		if errObj, ok := result["error"].(map[string]interface{}); ok {
			code := errObj["code"].(string)
			info := errObj["info"].(string)
			return nil, fmt.Errorf("API error [%s]: %s", code, info)
		}

		return result, nil
	}

	return nil, lastErr
}

// login authenticates with the wiki using bot password
func (c *Client) login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loggedIn && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	if !c.config.HasCredentials() {
		return fmt.Errorf("no credentials configured. Set MEDIAWIKI_USERNAME and MEDIAWIKI_PASSWORD environment variables")
	}

	// Get login token
	params := url.Values{}
	params.Set("action", "query")
	params.Set("meta", "tokens")
	params.Set("type", "login")

	resp, err := c.apiRequest(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to get login token: %w", err)
	}

	query, ok := resp["query"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	tokens, ok := query["tokens"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no tokens in response")
	}

	loginToken, ok := tokens["logintoken"].(string)
	if !ok {
		return fmt.Errorf("no login token in response")
	}

	// Perform login
	params = url.Values{}
	params.Set("action", "login")
	params.Set("lgname", c.config.Username)
	params.Set("lgpassword", c.config.Password)
	params.Set("lgtoken", loginToken)

	resp, err = c.apiRequest(ctx, params)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	login, ok := resp["login"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected login response")
	}

	result := login["result"].(string)
	if result != "Success" {
		reason := login["reason"]
		if reason != nil {
			return fmt.Errorf("login failed: %s - %v", result, reason)
		}
		return fmt.Errorf("login failed: %s", result)
	}

	c.loggedIn = true
	c.tokenExpiry = time.Now().Add(20 * time.Minute)

	c.logger.Info("Successfully logged in", "username", c.config.Username)

	return nil
}

// getCSRFToken gets a CSRF token for editing
func (c *Client) getCSRFToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.csrfToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.csrfToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	if err := c.login(ctx); err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("action", "query")
	params.Set("meta", "tokens")
	params.Set("type", "csrf")

	resp, err := c.apiRequest(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get CSRF token: %w", err)
	}

	query := resp["query"].(map[string]interface{})
	tokens := query["tokens"].(map[string]interface{})
	csrfToken := tokens["csrftoken"].(string)

	c.mu.Lock()
	c.csrfToken = csrfToken
	c.tokenExpiry = time.Now().Add(20 * time.Minute)
	c.mu.Unlock()

	return csrfToken, nil
}

// truncateContent truncates content if it exceeds the limit
func truncateContent(content string, limit int) (string, bool) {
	if len(content) <= limit {
		return content, false
	}
	return content[:limit] + "\n\n[Content truncated. Original length: " + fmt.Sprint(len(content)) + " characters]", true
}

// normalizeLimit ensures limit is within bounds
func normalizeLimit(limit, defaultVal, maxVal int) int {
	if limit <= 0 {
		return defaultVal
	}
	if limit > maxVal {
		return maxVal
	}
	return limit
}

// normalizeCategoryName ensures category name has proper prefix
func normalizeCategoryName(name string) string {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "Category:") {
		name = "Category:" + name
	}
	return name
}
