// Package repl provides an interactive shell for AQL queries.
package repl

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	aha "github.com/grokify/aha-go"
	"github.com/grokify/aha-studio/aql/parser"
	"github.com/grokify/aha-studio/aql/validator"
	"github.com/grokify/aha-studio/config"
	"github.com/grokify/aha-studio/executor"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// REPL provides an interactive query shell.
type REPL struct {
	client    *aha.Client
	executor  *executor.Executor
	config    *config.Config
	history   *History
	output    result.Format
	productID string
	verbose   bool
}

// New creates a new REPL instance.
func New(client *aha.Client, cfg *config.Config) *REPL {
	r := &REPL{
		client:   client,
		executor: executor.New(client),
		config:   cfg,
		output:   result.FormatTable,
	}

	// Apply config defaults
	if cfg != nil {
		if cfg.Defaults.Output != "" {
			if f, err := result.ParseFormat(cfg.Defaults.Output); err == nil {
				r.output = f
			}
		}
		if cfg.Defaults.Product != "" {
			r.productID = cfg.Defaults.Product
		}
	}

	// Load history
	r.history = NewHistory()
	_ = r.history.Load() // Ignore history load errors

	return r
}

// Run starts the interactive REPL.
func (r *REPL) Run() {
	fmt.Printf("Aha Studio REPL - Connected to %s.aha.io\n", r.client.Subdomain())
	fmt.Println("Type '.help' for commands, '.exit' to quit.")
	fmt.Println()

	p := prompt.New(
		r.execute,
		r.complete,
		prompt.OptionPrefix("aql> "),
		prompt.OptionTitle("Aha Studio"),
		prompt.OptionHistory(r.history.Entries()),
		prompt.OptionPrefixTextColor(prompt.Cyan),
		prompt.OptionPreviewSuggestionTextColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionBGColor(prompt.DarkBlue),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  func(*prompt.Buffer) { fmt.Println("\nUse '.exit' to quit.") },
		}),
	)
	p.Run()
}

// execute handles a line of input.
func (r *REPL) execute(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	// Handle REPL commands
	if strings.HasPrefix(input, ".") {
		r.handleCommand(input)
		return
	}

	// Execute AQL query
	r.executeQuery(input)
}

// handleCommand handles REPL dot commands.
func (r *REPL) handleCommand(input string) {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case ".help", ".h":
		r.showHelp()

	case ".exit", ".quit", ".q":
		if err := r.history.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save history: %v\n", err)
		}
		fmt.Println("Goodbye!")
		os.Exit(0)

	case ".output", ".o":
		if len(parts) < 2 {
			fmt.Printf("Current output format: %s\n", r.output)
			return
		}
		format, err := result.ParseFormat(parts[1])
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		r.output = format
		fmt.Printf("Output format set to: %s\n", format)

	case ".product", ".p":
		if len(parts) < 2 {
			if r.productID == "" {
				fmt.Println("No product context set.")
			} else {
				fmt.Printf("Current product: %s\n", r.productID)
			}
			return
		}
		r.productID = parts[1]
		fmt.Printf("Product context set to: %s\n", r.productID)

	case ".verbose", ".v":
		r.verbose = !r.verbose
		fmt.Printf("Verbose mode: %v\n", r.verbose)

	case ".history":
		entries := r.history.Entries()
		if len(entries) == 0 {
			fmt.Println("No history.")
			return
		}
		for i, entry := range entries {
			fmt.Printf("%3d: %s\n", i+1, entry)
		}

	case ".clear":
		fmt.Print("\033[H\033[2J")

	case ".save":
		if len(parts) < 2 {
			fmt.Println("Usage: .save <name>")
			return
		}
		name := parts[1]
		entries := r.history.Entries()
		if len(entries) == 0 {
			fmt.Println("No query to save.")
			return
		}
		lastQuery := entries[len(entries)-1]
		if r.config != nil {
			r.config.SaveQuery(name, lastQuery)
			if err := r.config.Save(); err != nil {
				fmt.Println("Error saving config:", err)
				return
			}
			fmt.Printf("Query saved as '%s'\n", name)
		} else {
			fmt.Println("Config not available.")
		}

	case ".run":
		if len(parts) < 2 {
			fmt.Println("Usage: .run <name>")
			return
		}
		name := parts[1]
		if r.config == nil {
			fmt.Println("Config not available.")
			return
		}
		query, ok := r.config.Queries[name]
		if !ok {
			fmt.Printf("Query '%s' not found.\n", name)
			return
		}
		fmt.Printf("Running: %s\n", query)
		r.executeQuery(query)

	case ".queries", ".list":
		if r.config == nil || len(r.config.Queries) == 0 {
			fmt.Println("No saved queries.")
			return
		}
		fmt.Println("Saved queries:")
		for name, query := range r.config.Queries {
			// Truncate long queries
			display := query
			if len(display) > 60 {
				display = display[:57] + "..."
			}
			fmt.Printf("  %s: %s\n", name, display)
		}

	case ".delete":
		if len(parts) < 2 {
			fmt.Println("Usage: .delete <name>")
			return
		}
		name := parts[1]
		if r.config == nil {
			fmt.Println("Config not available.")
			return
		}
		if _, ok := r.config.Queries[name]; !ok {
			fmt.Printf("Query '%s' not found.\n", name)
			return
		}
		delete(r.config.Queries, name)
		if err := r.config.Save(); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}
		fmt.Printf("Query '%s' deleted.\n", name)

	default:
		fmt.Printf("Unknown command: %s (type .help for commands)\n", cmd)
	}
}

// executeQuery parses and executes an AQL query.
func (r *REPL) executeQuery(input string) {
	// Add to history
	r.history.Add(input)

	// Parse
	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		fmt.Println("Parse error:", err)
		return
	}

	// Validate
	v := validator.New()
	if err := v.Validate(query); err != nil {
		fmt.Println("Validation error:", err)
		return
	}

	// Plan
	pl := planner.New()
	plan := pl.Plan(query)

	// Set product context
	if r.productID != "" {
		plan.APIParams.ProductID = r.productID
	}

	if r.verbose {
		fmt.Printf("Entity: %s\n", plan.Entity)
		fmt.Printf("API Params: %+v\n", plan.APIParams)
		fmt.Printf("Client Filters: %d\n", len(plan.ClientFilters))
		fmt.Println()
	}

	// Execute
	res, err := r.executor.Execute(context.Background(), plan)
	if err != nil {
		fmt.Println("Execution error:", err)
		return
	}

	// Format output
	formatter := result.NewFormatter(r.output)
	if plan.SelectFields != nil {
		formatter.WithFields(plan.SelectFields)
	}
	if err := formatter.Format(os.Stdout, res); err != nil {
		fmt.Println("Format error:", err)
	}
}

// showHelp displays help information.
func (r *REPL) showHelp() {
	help := `
Aha Studio REPL Commands:

  .help, .h          Show this help
  .exit, .quit, .q   Exit the REPL
  .output, .o <fmt>  Set output format (table, json, csv)
  .product, .p <id>  Set product context
  .verbose, .v       Toggle verbose mode
  .history           Show query history
  .clear             Clear screen
  .save <name>       Save last query with a name
  .run <name>        Run a saved query
  .queries, .list    List saved queries
  .delete <name>     Delete a saved query

AQL Syntax:

  [SELECT <fields/aggregates>]
  FROM <entity>
  [JOIN <entity> ON <condition>]
  [WHERE <conditions>]
  [GROUP BY <fields>]
  [HAVING <aggregate_condition>]
  [ORDER BY <field> [ASC|DESC]]
  [LIMIT <n>]

Entities: features, ideas, releases, initiatives

Aggregate Functions: COUNT(*), COUNT(field), SUM(field), AVG(field), MIN(field), MAX(field)

Examples:
  FROM features LIMIT 10
  FROM ideas WHERE votes > 5 ORDER BY votes DESC
  FROM features WHERE status = "In Progress"
  SELECT status, COUNT(*) FROM features GROUP BY status
  SELECT COUNT(*), AVG(votes) FROM ideas
`
	fmt.Println(help)
}
