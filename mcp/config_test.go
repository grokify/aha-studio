package mcp

import (
	"os"
	"testing"
)

func TestConfigFromEnv(t *testing.T) {
	// Save original env vars
	origSubdomain := os.Getenv("AHA_SUBDOMAIN")
	origAPIKey := os.Getenv("AHA_API_KEY")
	origProduct := os.Getenv("AHA_DEFAULT_PRODUCT")
	origEmail := os.Getenv("AHA_EMAIL")
	origPassword := os.Getenv("AHA_PASSWORD")

	// Restore env vars after test
	defer func() {
		restoreEnv("AHA_SUBDOMAIN", origSubdomain)
		restoreEnv("AHA_API_KEY", origAPIKey)
		restoreEnv("AHA_DEFAULT_PRODUCT", origProduct)
		restoreEnv("AHA_EMAIL", origEmail)
		restoreEnv("AHA_PASSWORD", origPassword)
	}()

	// Set test values
	os.Setenv("AHA_SUBDOMAIN", "test-company")
	os.Setenv("AHA_API_KEY", "test-key-12345")
	os.Setenv("AHA_DEFAULT_PRODUCT", "PROD")
	os.Setenv("AHA_EMAIL", "user@example.com")
	os.Setenv("AHA_PASSWORD", "secret123")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}

	if cfg.Subdomain != "test-company" {
		t.Errorf("Subdomain = %q, want %q", cfg.Subdomain, "test-company")
	}
	if cfg.APIKey != "test-key-12345" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key-12345")
	}
	if cfg.DefaultProduct != "PROD" {
		t.Errorf("DefaultProduct = %q, want %q", cfg.DefaultProduct, "PROD")
	}
	if cfg.BrowserEmail != "user@example.com" {
		t.Errorf("BrowserEmail = %q, want %q", cfg.BrowserEmail, "user@example.com")
	}
	if cfg.BrowserPassword != "secret123" {
		t.Errorf("BrowserPassword = %q, want %q", cfg.BrowserPassword, "secret123")
	}
}

func TestConfigFromEnv_MissingSubdomain(t *testing.T) {
	// Save and clear env vars
	origSubdomain := os.Getenv("AHA_SUBDOMAIN")
	origAPIKey := os.Getenv("AHA_API_KEY")
	defer func() {
		restoreEnv("AHA_SUBDOMAIN", origSubdomain)
		restoreEnv("AHA_API_KEY", origAPIKey)
	}()

	os.Unsetenv("AHA_SUBDOMAIN")
	os.Setenv("AHA_API_KEY", "test-key")

	_, err := ConfigFromEnv()
	if err == nil {
		t.Error("ConfigFromEnv() expected error for missing subdomain, got nil")
	}
}

func TestConfigFromEnv_MissingAPIKey(t *testing.T) {
	// Save and clear env vars
	origSubdomain := os.Getenv("AHA_SUBDOMAIN")
	origAPIKey := os.Getenv("AHA_API_KEY")
	defer func() {
		restoreEnv("AHA_SUBDOMAIN", origSubdomain)
		restoreEnv("AHA_API_KEY", origAPIKey)
	}()

	os.Setenv("AHA_SUBDOMAIN", "test-company")
	os.Unsetenv("AHA_API_KEY")

	_, err := ConfigFromEnv()
	if err == nil {
		t.Error("ConfigFromEnv() expected error for missing API key, got nil")
	}
}

func TestConfigFromEnv_OptionalFieldsEmpty(t *testing.T) {
	// Save original env vars
	origSubdomain := os.Getenv("AHA_SUBDOMAIN")
	origAPIKey := os.Getenv("AHA_API_KEY")
	origProduct := os.Getenv("AHA_DEFAULT_PRODUCT")
	origEmail := os.Getenv("AHA_EMAIL")
	origPassword := os.Getenv("AHA_PASSWORD")

	// Restore env vars after test
	defer func() {
		restoreEnv("AHA_SUBDOMAIN", origSubdomain)
		restoreEnv("AHA_API_KEY", origAPIKey)
		restoreEnv("AHA_DEFAULT_PRODUCT", origProduct)
		restoreEnv("AHA_EMAIL", origEmail)
		restoreEnv("AHA_PASSWORD", origPassword)
	}()

	// Set only required values, clear optional
	os.Setenv("AHA_SUBDOMAIN", "test-company")
	os.Setenv("AHA_API_KEY", "test-key")
	os.Unsetenv("AHA_DEFAULT_PRODUCT")
	os.Unsetenv("AHA_EMAIL")
	os.Unsetenv("AHA_PASSWORD")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}

	if cfg.DefaultProduct != "" {
		t.Errorf("DefaultProduct = %q, want empty", cfg.DefaultProduct)
	}
	if cfg.BrowserEmail != "" {
		t.Errorf("BrowserEmail = %q, want empty", cfg.BrowserEmail)
	}
	if cfg.BrowserPassword != "" {
		t.Errorf("BrowserPassword = %q, want empty", cfg.BrowserPassword)
	}
}

func TestConfig_NewClient(t *testing.T) {
	cfg := &Config{
		Subdomain: "test-company",
		APIKey:    "test-key",
	}

	client, err := cfg.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewClient() returned nil client")
	}
}

// restoreEnv restores an environment variable to its original value.
func restoreEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}
