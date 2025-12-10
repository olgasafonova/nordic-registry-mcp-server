package wiki

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds MediaWiki connection settings
type Config struct {
	// BaseURL is the wiki API endpoint (e.g., https://wiki.example.com/api.php)
	BaseURL string

	// Username for bot password authentication (optional, for editing)
	Username string

	// Password for bot password authentication (optional, for editing)
	Password string

	// Timeout for API requests
	Timeout time.Duration

	// UserAgent identifies the client to the wiki
	UserAgent string

	// MaxRetries for failed requests
	MaxRetries int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	baseURL := os.Getenv("MEDIAWIKI_URL")
	if baseURL == "" {
		return nil, errors.New("MEDIAWIKI_URL environment variable is required")
	}

	timeout := 30 * time.Second
	if t := os.Getenv("MEDIAWIKI_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	maxRetries := 3
	if r := os.Getenv("MEDIAWIKI_MAX_RETRIES"); r != "" {
		if n, err := strconv.Atoi(r); err == nil && n >= 0 {
			maxRetries = n
		}
	}

	userAgent := os.Getenv("MEDIAWIKI_USER_AGENT")
	if userAgent == "" {
		userAgent = "MediaWikiMCPServer/1.0 (https://github.com/olgasafonova/mediawiki-mcp-server)"
	}

	return &Config{
		BaseURL:    baseURL,
		Username:   os.Getenv("MEDIAWIKI_USERNAME"),
		Password:   os.Getenv("MEDIAWIKI_PASSWORD"),
		Timeout:    timeout,
		UserAgent:  userAgent,
		MaxRetries: maxRetries,
	}, nil
}

// HasCredentials returns true if authentication credentials are configured
func (c *Config) HasCredentials() bool {
	return c.Username != "" && c.Password != ""
}
