// Command aha-mcp-server runs an Aha! MCP server that exposes tools for
// managing product management data from Aha!
//
// For AQL queries and CLI functionality, use aha-studio instead.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	runtime "github.com/plexusone/omniskill/mcp/server"
	"github.com/spf13/cobra"

	ahamcp "github.com/grokify/aha-studio/mcp"
)

const (
	serverName    = "aha-mcp-server"
	serverVersion = "0.8.0"
)

var (
	// Server flags
	httpAddr string

	// Credential flags
	subdomain string
	apiKey    string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "aha-mcp-server",
	Short: "MCP server for Aha! product management",
	Long: `An MCP (Model Context Protocol) server for Aha! product management.

This server exposes 34 tools for AI assistants like Claude Desktop to interact
with Aha.io data including features, ideas, releases, initiatives, and more.

For AQL queries and CLI functionality, use aha-studio instead.

Environment variables:
  AHA_SUBDOMAIN        Aha account subdomain (required)
  AHA_API_KEY          Aha API key (required)
  AHA_DEFAULT_PRODUCT  Default product ID for queries
  AHA_EMAIL            Email for browser automation (optional)
  AHA_PASSWORD         Password for browser automation (optional)`,
	Example: `  # Start stdio server (for Claude Desktop)
  aha-mcp-server

  # Start HTTP server
  aha-mcp-server --http :8080

  # With explicit credentials
  aha-mcp-server --subdomain mycompany --api-key xxx`,
	SilenceUsage: true,
	RunE:         runServer,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n", serverName, serverVersion)
	},
}

func init() {
	// Server flags
	rootCmd.Flags().StringVar(&httpAddr, "http", "",
		"HTTP server address (e.g., :8080). If not set, uses stdio transport.")

	// Credential flags
	rootCmd.PersistentFlags().StringVar(&subdomain, "subdomain", "",
		"Aha! subdomain (env: AHA_SUBDOMAIN)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "",
		"Aha! API key (env: AHA_API_KEY)")

	rootCmd.AddCommand(versionCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	// Apply environment defaults
	if subdomain == "" {
		subdomain = os.Getenv("AHA_SUBDOMAIN")
		if subdomain == "" {
			subdomain = os.Getenv("AHA_DOMAIN") // fallback
		}
	}
	if apiKey == "" {
		apiKey = os.Getenv("AHA_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("AHA_API_TOKEN") // fallback
		}
	}

	// Validate credentials
	if subdomain == "" {
		return fmt.Errorf("subdomain required: use --subdomain or set AHA_SUBDOMAIN")
	}
	if apiKey == "" {
		return fmt.Errorf("API key required: use --api-key or set AHA_API_KEY")
	}

	// Create config
	cfg := &ahamcp.Config{
		Subdomain:       subdomain,
		APIKey:          apiKey,
		DefaultProduct:  os.Getenv("AHA_DEFAULT_PRODUCT"),
		BrowserEmail:    os.Getenv("AHA_EMAIL"),
		BrowserPassword: os.Getenv("AHA_PASSWORD"),
	}

	// Create and initialize skill
	ctx := context.Background()
	skill := ahamcp.NewAhaSkill(cfg)
	if err := skill.Init(ctx); err != nil {
		return fmt.Errorf("initializing skill: %w", err)
	}
	defer skill.Close()

	// Create OmniSkill runtime
	rt := runtime.New(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)
	rt.RegisterSkill(skill)

	// Start server
	if httpAddr != "" {
		fmt.Fprintf(os.Stderr, "Starting MCP server on %s\n", httpAddr)
		_, err := rt.ServeHTTP(ctx, &runtime.HTTPServerOptions{
			Addr: httpAddr,
		})
		return err
	}

	// Default to stdio
	return rt.ServeStdio(ctx)
}
