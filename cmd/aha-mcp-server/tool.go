package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/plexusone/omniskill/skill"
	"github.com/spf13/cobra"

	ahamcp "github.com/grokify/aha-studio/mcp"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Invoke MCP tools directly from the CLI",
	Long: `Invoke Aha MCP tools directly without a running MCP server session.

This is useful for AI agent sessions that cannot register new or updated MCP
tools mid-session (an agent's MCP servers can only be started once, at
session launch). Instead of restarting the session to pick up a tool that
was added or changed, an agent can shell out to run any registered tool
directly by name.`,
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available tools",
	Args:  cobra.NoArgs,
	RunE:  runToolList,
}

var toolSchemaCmd = &cobra.Command{
	Use:   "schema <tool-name>",
	Short: "Print the JSON Schema for a tool's parameters",
	Args:  cobra.ExactArgs(1),
	RunE:  runToolSchema,
}

var toolCallCmd = &cobra.Command{
	Use:   "call <tool-name> [json-params]",
	Short: "Call a tool by name with JSON parameters",
	Long: `Call a tool by name with JSON parameters and print the JSON result.

Parameters are a single JSON object, provided either as the second argument
or via stdin (omit the argument, or pass "-", to read from stdin).

Examples:
  aha-mcp-server tool call get_feature '{"reference":"FEAT-123"}'
  echo '{"initiative_id":"PROD-S-34"}' | aha-mcp-server tool call list_initiative_features`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runToolCall,
}

func init() {
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolSchemaCmd)
	toolCmd.AddCommand(toolCallCmd)
	rootCmd.AddCommand(toolCmd)
}

// loadTools builds the Aha skill and returns its tools indexed by name.
func loadTools(ctx context.Context) (map[string]skill.Tool, error) {
	cfg, err := buildConfig()
	if err != nil {
		return nil, err
	}

	s := ahamcp.NewAhaSkill(cfg)
	if err := s.Init(ctx); err != nil {
		return nil, fmt.Errorf("initializing skill: %w", err)
	}

	tools := make(map[string]skill.Tool, len(s.Tools()))
	for _, t := range s.Tools() {
		tools[t.Name()] = t
	}
	return tools, nil
}

// lookupTool loads all tools and returns the one matching name, or an error
// listing the failure if the tool isn't registered.
func lookupTool(ctx context.Context, name string) (skill.Tool, error) {
	tools, err := loadTools(ctx)
	if err != nil {
		return nil, err
	}

	t, ok := tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q (run 'aha-mcp-server tool list' to see available tools)", name)
	}
	return t, nil
}

func runToolList(cmd *cobra.Command, args []string) error {
	tools, err := loadTools(cmd.Context())
	if err != nil {
		return err
	}

	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Printf("%-40s %s\n", name, tools[name].Description())
	}
	return nil
}

func runToolSchema(cmd *cobra.Command, args []string) error {
	t, err := lookupTool(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	schema := skill.ParametersToJSONSchema(t.Parameters())
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(schema)
}

func runToolCall(cmd *cobra.Command, args []string) error {
	name := args[0]

	rawParams := ""
	if len(args) == 2 && args[1] != "-" {
		rawParams = args[1]
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading params from stdin: %w", err)
		}
		rawParams = strings.TrimSpace(string(data))
	}

	params := map[string]any{}
	if rawParams != "" {
		if err := json.Unmarshal([]byte(rawParams), &params); err != nil {
			return fmt.Errorf("parsing JSON params: %w", err)
		}
	}

	t, err := lookupTool(cmd.Context(), name)
	if err != nil {
		return err
	}

	result, err := t.Call(cmd.Context(), params)
	if err != nil {
		return fmt.Errorf("calling tool %q: %w", name, err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
