// Package graph provides Neo4j graph database integration for Aha Studio.
// It enables relationship-aware queries and graph analytics on Aha.io data.
package graph

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps a Neo4j driver connection.
type Client struct {
	driver  neo4j.DriverWithContext
	uri     string
	product string
}

// Config holds Neo4j connection configuration.
type Config struct {
	URI      string
	Username string
	Password string
	Database string
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() *Config {
	return &Config{
		URI:      getEnvOrDefault("NEO4J_URI", "neo4j://localhost:7687"),
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: os.Getenv("NEO4J_PASSWORD"),
		Database: getEnvOrDefault("NEO4J_DATABASE", "neo4j"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// NewClient creates a new Neo4j client.
func NewClient(cfg *Config) (*Client, error) {
	auth := neo4j.BasicAuth(cfg.Username, cfg.Password, "")
	driver, err := neo4j.NewDriverWithContext(cfg.URI, auth)
	if err != nil {
		return nil, fmt.Errorf("creating neo4j driver: %w", err)
	}

	return &Client{
		driver: driver,
		uri:    cfg.URI,
	}, nil
}

// Close closes the Neo4j driver connection.
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// Verify verifies connectivity to Neo4j.
func (c *Client) Verify(ctx context.Context) error {
	return c.driver.VerifyConnectivity(ctx)
}

// SetProduct sets the current product context for queries.
func (c *Client) SetProduct(product string) {
	c.product = product
}

// Session returns a new Neo4j session.
func (c *Client) Session(ctx context.Context) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
}

// ReadSession returns a new Neo4j session for read-only operations.
func (c *Client) ReadSession(ctx context.Context) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
}

// ExecuteWrite executes a write transaction.
func (c *Client) ExecuteWrite(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := c.Session(ctx)
	defer session.Close(ctx)
	return session.ExecuteWrite(ctx, work)
}

// ExecuteRead executes a read transaction.
func (c *Client) ExecuteRead(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := c.ReadSession(ctx)
	defer session.Close(ctx)
	return session.ExecuteRead(ctx, work)
}

// Run executes a Cypher query with parameters.
func (c *Client) Run(ctx context.Context, cypher string, params map[string]any) (neo4j.ResultWithContext, error) {
	session := c.Session(ctx)
	defer session.Close(ctx)
	return session.Run(ctx, cypher, params)
}
