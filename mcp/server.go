package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpserver "github.com/plexusone/omniskill/mcp/server"
)

// ServerOptions configures the MCP server.
type ServerOptions struct {
	// Transport is the transport mode: "stdio" or "http"
	Transport string

	// HTTPAddr is the HTTP server address (for http transport)
	HTTPAddr string

	// Config is the Aha configuration
	Config *Config
}

// RunServer starts the MCP server.
func RunServer(ctx context.Context, opts *ServerOptions) error {
	// Create OmniSkill runtime
	rt := mcpserver.New(&mcp.Implementation{
		Name:    "aha-studio",
		Version: "0.3.0",
	}, nil)

	// Create and register Aha skill
	ahaSkill := NewAhaSkill(opts.Config)
	if err := ahaSkill.Init(ctx); err != nil {
		return fmt.Errorf("initializing Aha skill: %w", err)
	}
	rt.RegisterSkillWithPrefix(ahaSkill)

	// Start server based on transport
	switch opts.Transport {
	case "http":
		_, err := rt.ServeHTTP(ctx, &mcpserver.HTTPServerOptions{
			Addr: opts.HTTPAddr,
		})
		return err
	case "stdio":
		fallthrough
	default:
		return rt.ServeStdio(ctx)
	}
}
