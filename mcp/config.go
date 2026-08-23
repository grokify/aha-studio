package mcp

import (
	"fmt"
	"os"
	"strconv"

	aha "github.com/grokify/aha-go"
)

// Config holds configuration for the Aha MCP skill.
type Config struct {
	// Subdomain is the Aha.io account subdomain.
	Subdomain string

	// APIKey is the Aha.io API key.
	APIKey string

	// DefaultProduct is the default product ID for queries.
	DefaultProduct string

	// BrowserEmail is the email for browser automation login.
	BrowserEmail string

	// BrowserPassword is the password for browser automation login.
	BrowserPassword string

	// DBPath is the path to the SQLite cache database for offline queries.
	// If empty, cache-based tools will return an error.
	DBPath string

	// SyncRequestsPerSecond throttles the dedicated client used by the
	// sync_data tool (does not affect other tools' shared client). Default
	// is 10 req/s if unset/zero (see NewClient).
	SyncRequestsPerSecond float64
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() (*Config, error) {
	subdomain := os.Getenv("AHA_SUBDOMAIN")
	if subdomain == "" {
		return nil, fmt.Errorf("AHA_SUBDOMAIN environment variable not set")
	}

	apiKey := os.Getenv("AHA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("AHA_API_KEY environment variable not set")
	}

	syncRPS := 10.0
	if v := os.Getenv("AHA_SYNC_RPS"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			syncRPS = parsed
		}
	}

	return &Config{
		Subdomain:             subdomain,
		APIKey:                apiKey,
		DefaultProduct:        os.Getenv("AHA_DEFAULT_PRODUCT"),
		BrowserEmail:          os.Getenv("AHA_EMAIL"),
		BrowserPassword:       os.Getenv("AHA_PASSWORD"),
		DBPath:                os.Getenv("AHA_DB_PATH"),
		SyncRequestsPerSecond: syncRPS,
	}, nil
}

// NewClient creates an Aha API client from the config.
func (c *Config) NewClient() (*aha.Client, error) {
	return aha.NewClient(
		aha.WithSubdomain(c.Subdomain),
		aha.WithAPIKey(c.APIKey),
	)
}

// NewSyncClient creates an Aha API client throttled for bulk sync
// operations, using SyncRequestsPerSecond (defaulting to 10 req/s if unset).
func (c *Config) NewSyncClient() (*aha.Client, error) {
	rps := c.SyncRequestsPerSecond
	if rps <= 0 {
		rps = 10
	}
	return aha.NewClient(
		aha.WithSubdomain(c.Subdomain),
		aha.WithAPIKey(c.APIKey),
		aha.WithRequestsPerSecond(rps),
	)
}
