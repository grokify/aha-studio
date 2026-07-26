package mcp

import (
	"fmt"
	"os"

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

	return &Config{
		Subdomain:       subdomain,
		APIKey:          apiKey,
		DefaultProduct:  os.Getenv("AHA_DEFAULT_PRODUCT"),
		BrowserEmail:    os.Getenv("AHA_EMAIL"),
		BrowserPassword: os.Getenv("AHA_PASSWORD"),
		DBPath:          os.Getenv("AHA_DB_PATH"),
	}, nil
}

// NewClient creates an Aha API client from the config.
func (c *Config) NewClient() (*aha.Client, error) {
	return aha.NewClient(
		aha.WithSubdomain(c.Subdomain),
		aha.WithAPIKey(c.APIKey),
	)
}
