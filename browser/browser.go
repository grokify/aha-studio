// Package browser provides browser automation for Aha! using w3pilot.
package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/plexusone/w3pilot"
)

// Client wraps w3pilot for Aha! browser automation.
type Client struct {
	pilot     *w3pilot.Pilot
	subdomain string
	email     string
	password  string
	headless  bool
	timeout   time.Duration
	loggedIn  bool
}

// Config holds browser client configuration.
type Config struct {
	Subdomain string
	Email     string
	Password  string
	Headless  bool
	Timeout   time.Duration
}

// NewClient creates a new browser automation client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		subdomain: cfg.Subdomain,
		email:     cfg.Email,
		password:  cfg.Password,
		headless:  cfg.Headless,
		timeout:   timeout,
	}
}

// Launch starts the browser.
func (c *Client) Launch(ctx context.Context) error {
	var err error
	if c.headless {
		c.pilot, err = w3pilot.LaunchHeadless(ctx)
	} else {
		c.pilot, err = w3pilot.Launch(ctx)
	}
	return err
}

// Close closes the browser.
func (c *Client) Close(ctx context.Context) error {
	if c.pilot != nil {
		return c.pilot.Quit(ctx)
	}
	return nil
}

// IsLoggedIn returns whether the client is logged in.
func (c *Client) IsLoggedIn() bool {
	return c.loggedIn
}

// BaseURL returns the Aha base URL for this subdomain.
func (c *Client) BaseURL() string {
	return fmt.Sprintf("https://%s.aha.io", c.subdomain)
}

// Login authenticates with Aha! using email and password.
func (c *Client) Login(ctx context.Context) error {
	if c.pilot == nil {
		if err := c.Launch(ctx); err != nil {
			return fmt.Errorf("failed to launch browser: %w", err)
		}
	}

	loginURL := c.BaseURL() + "/session/new"
	if err := c.pilot.Go(ctx, loginURL); err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// Wait for and fill email field
	emailField, err := c.pilot.Find(ctx, "input[name='user[email]']", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to find email field: %w", err)
	}
	if err := emailField.Fill(ctx, c.email, nil); err != nil {
		return fmt.Errorf("failed to fill email: %w", err)
	}

	// Fill password field
	passwordField, err := c.pilot.Find(ctx, "input[name='user[password]']", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to find password field: %w", err)
	}
	if err := passwordField.Fill(ctx, c.password, nil); err != nil {
		return fmt.Errorf("failed to fill password: %w", err)
	}

	// Click submit button
	submitBtn, err := c.pilot.Find(ctx, "input[type='submit'], button[type='submit']", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to find submit button: %w", err)
	}
	if err := submitBtn.Click(ctx, nil); err != nil {
		return fmt.Errorf("failed to click submit: %w", err)
	}

	// Wait for navigation to complete
	if err := c.pilot.WaitForNavigation(ctx, c.timeout); err != nil {
		return fmt.Errorf("failed to wait for login navigation: %w", err)
	}

	// Verify we're logged in by checking for dashboard elements
	_, err = c.pilot.Find(ctx, "[data-testid='nav-products'], .products-menu, #products-menu", &w3pilot.FindOptions{
		Timeout: c.timeout,
	})
	if err != nil {
		return fmt.Errorf("login may have failed - could not verify dashboard: %w", err)
	}

	c.loggedIn = true
	return nil
}

// NavigateTo navigates to a specific URL path within Aha.
func (c *Client) NavigateTo(ctx context.Context, path string) error {
	if c.pilot == nil {
		return fmt.Errorf("browser not launched")
	}
	url := c.BaseURL() + path
	return c.pilot.Go(ctx, url)
}

// Screenshot takes a screenshot and returns PNG bytes.
func (c *Client) Screenshot(ctx context.Context) ([]byte, error) {
	if c.pilot == nil {
		return nil, fmt.Errorf("browser not launched")
	}
	return c.pilot.Screenshot(ctx)
}

// Title returns the current page title.
func (c *Client) Title(ctx context.Context) (string, error) {
	if c.pilot == nil {
		return "", fmt.Errorf("browser not launched")
	}
	return c.pilot.Title(ctx)
}

// URL returns the current page URL.
func (c *Client) URL(ctx context.Context) (string, error) {
	if c.pilot == nil {
		return "", fmt.Errorf("browser not launched")
	}
	return c.pilot.URL(ctx)
}

// Pilot returns the underlying w3pilot instance for advanced operations.
func (c *Client) Pilot() *w3pilot.Pilot {
	return c.pilot
}
