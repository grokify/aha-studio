// Package main provides the aha-studio CLI entry point.
// aha-studio is the AQL (Aha Query Language) CLI for querying Aha.io data.
// For MCP server functionality, use aha-mcp-server instead.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	aha "github.com/grokify/aha-go"
	"github.com/grokify/mogo/fmt/progress"
	"github.com/spf13/cobra"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/aql/parser"
	"github.com/grokify/aha-studio/aql/validator"
	"github.com/grokify/aha-studio/config"
	"github.com/grokify/aha-studio/executor"
	"github.com/grokify/aha-studio/httpserver"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/repl"
	"github.com/grokify/aha-studio/result"
	"github.com/grokify/aha-studio/sync"
)

var (
	// Version is set by build flags
	Version = "dev"

	// Global config
	cfg *config.Config

	// CLI flags
	outputFormat string
	productID    string
	verbose      bool

	// Exec command flags
	dryRun    bool
	releaseID string
	force     bool

	// Stats flag
	showStats bool

	// Output file
	outputFile string

	// Sync command flags
	syncSince    string
	syncEntities []string

	// Offline query flags
	offlineMode bool
	preferCache bool

	// Serve command flags
	serverAddr     string
	corsOrigins    []string
	noAuth         bool
	serverOffline  bool
	serverPrefer   string
	backgroundSync bool
	syncInterval   string
	defaultProduct string
)

func main() {
	// Load config
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &config.Config{}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "aha-studio",
	Short: "Aha Studio - Query Aha.io with AQL",
	Long: `Aha Studio provides a JQL-like query language (AQL) for querying Aha.io data.

For MCP server functionality, use aha-mcp-server instead.

Environment variables:
  AHA_SUBDOMAIN  Your Aha account subdomain
  AHA_API_KEY    Your Aha API key

Examples:
  aha-studio query "FROM features LIMIT 10"
  aha-studio query "FROM ideas WHERE votes > 10 ORDER BY votes DESC"
  aha-studio query --product PLATFORM "FROM releases"
  aha-studio query -o json "FROM features WHERE status = 'In Progress'"
  aha-studio shell`,
	Version: Version,
}

var queryCmd = &cobra.Command{
	Use:   "query <aql>",
	Short: "Execute an AQL query",
	Long: `Execute an AQL query against the Aha.io API.

AQL Syntax:
  FROM <entity> [WHERE <conditions>] [ORDER BY <field> [ASC|DESC]] [LIMIT <n>]

Entities:
  features, ideas, releases, initiatives, products, users

Operators:
  =, !=, <, <=, >, >=, IN, NOT IN, CONTAINS, LIKE, IS NULL, IS NOT NULL

Logical:
  AND, OR, NOT

Output Formats:
  table     ASCII table (default)
  json      JSON object
  csv       Comma-separated values
  markdown  Markdown table (md)
  yaml      YAML format (yml)
  html      HTML document
  xlsx      Excel workbook (requires -f)

Examples:
  FROM features LIMIT 5
  FROM features WHERE status = "In Progress"
  FROM features WHERE name CONTAINS "API" AND updated_at >= now() - duration("30d")
  FROM ideas WHERE votes > 10 ORDER BY votes DESC LIMIT 10
  FROM ideas WHERE status IN ("New", "Under Review")`,
	Args: cobra.ExactArgs(1),
	RunE: runQuery,
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start interactive REPL",
	Long: `Start an interactive REPL for executing AQL queries.

The REPL provides:
  - Tab completion for keywords, entities, and fields
  - Query history (persisted across sessions)
  - Saved queries
  - Multiple output formats

REPL Commands:
  .help          Show help
  .exit          Exit the REPL
  .output <fmt>  Set output format (table, json, csv)
  .product <id>  Set product context
  .history       Show query history
  .save <name>   Save last query
  .run <name>    Run saved query
  .queries       List saved queries`,
	RunE: runShell,
}

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a saved query",
	Long:  `Run a previously saved query by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSavedQuery,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved queries",
	Long:  `List all saved queries from the config file.`,
	RunE:  listSavedQueries,
}

var saveCmd = &cobra.Command{
	Use:   "save <name> <query>",
	Short: "Save a query",
	Long:  `Save a query with a name for later use.`,
	Args:  cobra.ExactArgs(2),
	RunE:  saveQuery,
}

var execCmd = &cobra.Command{
	Use:   "exec <statement>",
	Short: "Execute an AQL mutation (INSERT, UPDATE, DELETE)",
	Long: `Execute an AQL mutation statement against the Aha.io API.

Mutation Syntax:
  INSERT INTO <entity> (columns) VALUES (values)
  UPDATE <entity> SET field = value [WHERE conditions]
  DELETE FROM <entity> WHERE conditions

Supported Entities:
  features  - Create and update features (requires --release for INSERT)

Safety Features:
  --dry-run  Show what would happen without making changes
  --force    Skip confirmation prompt

Examples:
  # Create a new feature (dry run)
  aha-studio exec --dry-run --release REL-1 \
    "INSERT INTO features (name, description) VALUES ('New Feature', 'Description')"

  # Update features matching a condition
  aha-studio exec --product PROJ \
    "UPDATE features SET status = 'Done' WHERE name CONTAINS 'MVP'"

  # Delete is not supported by the Aha API`,
	Args: cobra.ExactArgs(1),
	RunE: runExec,
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Aha data to local SQLite database",
	Long: `Sync Aha.io data to a local SQLite database for offline queries and better performance.

The sync command downloads entities from the Aha API and stores them locally
in ~/.aha-studio/cache.db. This enables offline queries and faster response times.

Entities synced:
  features, ideas, releases, initiatives, goals, epics

Examples:
  # Full sync for a product
  aha-studio sync --product PLATFORM

  # Incremental sync (only changes since last sync)
  aha-studio sync --product PLATFORM --since last

  # Sync specific entities only
  aha-studio sync --product PLATFORM --entities features,ideas

  # Sync changes from a specific date
  aha-studio sync --product PLATFORM --since 2024-01-15`,
	RunE: runSync,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	Long:  `Show the sync status for each entity, including last sync time and record counts.`,
	RunE:  runSyncStatus,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server for AQL queries",
	Long: `Start an HTTP server that exposes a REST API for AQL queries.

The server provides the following endpoints:
  GET  /health              Health check
  GET  /api/version         Server version
  POST /api/query           Execute AQL (JSON body)
  GET  /api/query?aql=...   Execute AQL (query string)
  GET  /api/entities        List available entities
  GET  /api/syntax          AQL syntax help
  GET  /api/products        List products
  GET  /api/cache/status    Cache status
  POST /api/sync            Trigger sync
  GET  /api/sync/status     Sync status

Query Modes:
  api           Query live Aha API (default if no cache)
  offline       Query local SQLite cache only
  prefer-cache  Try cache first, fallback to API (default if cache available)

  Set default mode with --prefer or per-request via mode parameter.

Authentication:
  By default, requests require an API key via X-API-Key header,
  Authorization: Bearer <key>, or api_key query parameter.
  Use --no-auth to disable authentication for local development.

Examples:
  aha-studio serve                                # Start on :8080 with prefer-cache
  aha-studio serve --addr :3000                   # Custom port
  aha-studio serve --offline                      # Offline-only mode (no API calls)
  aha-studio serve --prefer api                   # Always use live API
  aha-studio serve --background-sync --product X  # Enable auto-sync
  aha-studio serve --cors "https://app.com"       # Restrict CORS
  aha-studio serve --no-auth                      # Disable auth`,
	RunE: runServe,
}

func init() {
	// Query command flags
	queryCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (table, json, csv, markdown, yaml, html, xlsx)")
	queryCmd.Flags().StringVarP(&outputFile, "file", "f", "", "Output file path (required for xlsx format)")
	queryCmd.Flags().StringVarP(&productID, "product", "p", "", "Product ID or reference prefix (required for releases)")
	queryCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	queryCmd.Flags().BoolVar(&showStats, "stats", false, "Show query execution statistics")
	queryCmd.Flags().BoolVar(&offlineMode, "offline", false, "Query local SQLite database only (requires prior sync)")
	queryCmd.Flags().BoolVar(&preferCache, "prefer-cache", false, "Try local cache first, fall back to API")

	// Run command flags
	runCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (table, json, csv)")
	runCmd.Flags().StringVarP(&productID, "product", "p", "", "Product ID or reference prefix")

	// Exec command flags
	execCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes")
	execCmd.Flags().StringVarP(&productID, "product", "p", "", "Product ID or reference prefix")
	execCmd.Flags().StringVar(&releaseID, "release", "", "Release ID for creating features")
	execCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	execCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Sync command flags
	syncCmd.Flags().StringVarP(&productID, "product", "p", "", "Product ID to sync (required)")
	syncCmd.Flags().StringVar(&syncSince, "since", "", "Sync changes since: 'last' or date (YYYY-MM-DD)")
	syncCmd.Flags().StringSliceVar(&syncEntities, "entities", nil, "Specific entities to sync (comma-separated)")
	_ = syncCmd.MarkFlagRequired("product")

	// Sync status command flags
	syncStatusCmd.Flags().StringVarP(&productID, "product", "p", "", "Product ID to show status for")

	// Add sync subcommand
	syncCmd.AddCommand(syncStatusCmd)

	// Serve command flags
	serveCmd.Flags().StringVar(&serverAddr, "addr", ":8080", "Listen address")
	serveCmd.Flags().StringSliceVar(&corsOrigins, "cors", []string{"*"}, "CORS origins (comma-separated)")
	serveCmd.Flags().BoolVar(&noAuth, "no-auth", false, "Disable API key authentication")
	serveCmd.Flags().BoolVar(&serverOffline, "offline", false, "Offline-only mode (no API calls)")
	serveCmd.Flags().StringVar(&serverPrefer, "prefer", "prefer-cache", "Default query mode (api, offline, prefer-cache)")
	serveCmd.Flags().BoolVar(&backgroundSync, "background-sync", false, "Enable automatic background sync")
	serveCmd.Flags().StringVar(&syncInterval, "sync-interval", "15m", "Background sync interval (e.g., 15m, 1h)")
	serveCmd.Flags().StringVarP(&defaultProduct, "product", "p", "", "Default product for sync operations")

	// Add commands
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(serveCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	aqlQuery := args[0]
	return executeQuery(aqlQuery)
}

func runShell(cmd *cobra.Command, args []string) error {
	// Create Aha client
	client, err := aha.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Aha client: %w", err)
	}

	// Start REPL
	r := repl.New(client, cfg)
	r.Run()
	return nil
}

func runSavedQuery(cmd *cobra.Command, args []string) error {
	name := args[0]
	query, ok := cfg.GetQuery(name)
	if !ok {
		return fmt.Errorf("query '%s' not found", name)
	}

	fmt.Printf("Running: %s\n\n", query)
	return executeQuery(query)
}

func listSavedQueries(cmd *cobra.Command, args []string) error {
	if cfg == nil || len(cfg.Queries) == 0 {
		fmt.Println("No saved queries.")
		return nil
	}

	fmt.Println("Saved queries:")
	for name, query := range cfg.Queries {
		display := query
		if len(display) > 60 {
			display = display[:57] + "..."
		}
		fmt.Printf("  %s: %s\n", name, display)
	}
	return nil
}

func saveQuery(cmd *cobra.Command, args []string) error {
	name := args[0]
	query := args[1]

	cfg.SaveQuery(name, query)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Query saved as '%s'\n", name)
	return nil
}

func executeQuery(aqlQuery string) error {
	startTime := time.Now()

	// Parse the query
	p := parser.New(aqlQuery)
	query, err := p.Parse()
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Validate the query
	v := validator.New()
	if err := v.Validate(query); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create execution plan
	pl := planner.New()
	plan := pl.Plan(query)

	// Apply config defaults
	if productID == "" && cfg != nil && cfg.Defaults.Product != "" {
		productID = cfg.Defaults.Product
	}
	if productID != "" {
		plan.APIParams.ProductID = productID
	}

	// Resolve output format
	format := result.FormatTable
	if outputFormat != "" {
		var err error
		format, err = result.ParseFormat(outputFormat)
		if err != nil {
			return err
		}
	} else if cfg != nil && cfg.Defaults.Output != "" {
		var err error
		format, err = result.ParseFormat(cfg.Defaults.Output)
		if err != nil {
			return err
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Entity: %s\n", plan.Entity)
		fmt.Fprintf(os.Stderr, "API Params: %+v\n", plan.APIParams)
		fmt.Fprintf(os.Stderr, "Client Filters: %d\n", len(plan.ClientFilters))
		fmt.Fprintf(os.Stderr, "Requires Pagination: %v\n", plan.RequiresPagination)
		if offlineMode {
			fmt.Fprintf(os.Stderr, "Mode: offline (SQLite)\n")
		} else if preferCache {
			fmt.Fprintf(os.Stderr, "Mode: prefer-cache\n")
		}
		fmt.Fprintln(os.Stderr)
	}

	var res *result.Result

	// Handle offline mode
	if offlineMode {
		res, err = executeOfflineQuery(plan)
		if err != nil {
			return fmt.Errorf("offline query error: %w", err)
		}
	} else if preferCache {
		// Try offline first, fall back to API
		res, err = executeOfflineQuery(plan)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Cache miss, falling back to API: %v\n", err)
			}
			res, err = executeAPIQuery(plan)
			if err != nil {
				return fmt.Errorf("execution error: %w", err)
			}
		}
	} else {
		// Normal API execution
		res, err = executeAPIQuery(plan)
		if err != nil {
			return fmt.Errorf("execution error: %w", err)
		}
	}

	executionTime := time.Since(startTime)

	// Handle Excel output separately (requires file path)
	if format == result.FormatExcel {
		if outputFile == "" {
			return fmt.Errorf("xlsx format requires output file: use -f <path.xlsx>")
		}
		excelFormatter := result.NewExcelFormatter()
		if plan.SelectFields != nil {
			excelFormatter.WithFields(plan.SelectFields)
		}
		if err := excelFormatter.FormatToFile(outputFile, res); err != nil {
			return fmt.Errorf("writing Excel file: %w", err)
		}
		fmt.Printf("Wrote %d record(s) to %s\n", res.Count(), outputFile)
	} else {
		// Format and output results to stdout
		formatter := result.NewFormatter(format)
		if plan.SelectFields != nil {
			formatter.WithFields(plan.SelectFields)
		}

		if err := formatter.Format(os.Stdout, res); err != nil {
			return err
		}
	}

	// Show stats if requested
	if showStats {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Query Statistics:\n")
		fmt.Fprintf(os.Stderr, "  Execution time: %v\n", executionTime.Round(time.Millisecond))
		fmt.Fprintf(os.Stderr, "  Records returned: %d\n", res.Count())
		fmt.Fprintf(os.Stderr, "  Client filters: %d\n", len(plan.ClientFilters))
		if plan.RequiresPagination {
			fmt.Fprintf(os.Stderr, "  Pagination: enabled\n")
		}
		if offlineMode {
			fmt.Fprintf(os.Stderr, "  Source: SQLite cache\n")
		} else if preferCache {
			fmt.Fprintf(os.Stderr, "  Source: cache with API fallback\n")
		} else {
			fmt.Fprintf(os.Stderr, "  Source: Aha API\n")
		}
	}

	return nil
}

// executeAPIQuery executes a query against the Aha API.
func executeAPIQuery(plan *planner.Plan) (*result.Result, error) {
	client, err := aha.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Aha client: %w", err)
	}

	// Create progress renderer
	renderer := progress.NewSingleStageRenderer(os.Stderr).WithBarWidth(30).WithTextWidth(40)

	// Progress callback
	progressFn := func(current, total int, message string) {
		renderer.Update(current, total, message)
	}

	exec := executor.New(client).WithProgress(progressFn)
	res, err := exec.Execute(context.Background(), plan)

	// Clear progress line
	renderer.Done("")

	return res, err
}

// executeOfflineQuery executes a query against the local SQLite database.
func executeOfflineQuery(plan *planner.Plan) (*result.Result, error) {
	db, err := sync.Open(sync.DefaultDBPath())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Set product context if specified
	if productID != "" {
		db.SetProduct(productID)
	}

	return db.QueryOffline(plan)
}

func runSync(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := sync.Open(sync.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Create Aha client
	client, err := aha.NewClient()
	if err != nil {
		return fmt.Errorf("creating Aha client: %w", err)
	}

	// Create progress renderer
	renderer := progress.NewSingleStageRenderer(os.Stderr).WithBarWidth(30).WithTextWidth(40)

	// Progress callback
	progressFn := func(current, total int, message string) {
		renderer.Update(current, total, message)
	}

	// Create syncer with GraphQL support for release_id capture
	subdomain := os.Getenv("AHA_SUBDOMAIN")
	apiKey := os.Getenv("AHA_API_KEY")
	syncer := sync.NewSyncerWithGraphQL(db, client, subdomain, apiKey).WithProgress(progressFn)

	// Build sync options
	opts := sync.SyncOptions{
		Product:  productID,
		Entities: syncEntities,
	}

	// Handle --since flag
	if syncSince != "" {
		if syncSince == "last" {
			opts.Incremental = true
		} else {
			// Parse as date
			t, err := time.Parse("2006-01-02", syncSince)
			if err != nil {
				return fmt.Errorf("invalid date format for --since: %w (expected YYYY-MM-DD)", err)
			}
			opts.Since = t
		}
	}

	fmt.Printf("Syncing product %s...\n", productID)
	if opts.Incremental {
		fmt.Println("Mode: incremental (since last sync)")
	} else if !opts.Since.IsZero() {
		fmt.Printf("Mode: changes since %s\n", opts.Since.Format("2006-01-02"))
	} else {
		fmt.Println("Mode: full sync")
	}
	fmt.Println()

	// Execute sync
	startTime := time.Now()
	results, err := syncer.SyncAll(context.Background(), opts)

	// Clear progress line
	renderer.Done("")

	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	totalRecords := 0
	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("  %-12s ERROR: %v\n", r.Entity, r.Error)
		} else {
			fmt.Printf("  %-12s %d records (%v)\n", r.Entity, r.RecordCount, r.Duration.Round(time.Millisecond))
			totalRecords += r.RecordCount
		}
	}

	fmt.Printf("\nSync complete: %d total records in %v\n", totalRecords, time.Since(startTime).Round(time.Millisecond))
	fmt.Printf("Database: %s\n", sync.DefaultDBPath())

	return nil
}

func runSyncStatus(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := sync.Open(sync.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Get product ID
	product := productID
	if product == "" && cfg != nil && cfg.Defaults.Product != "" {
		product = cfg.Defaults.Product
	}
	if product == "" {
		return fmt.Errorf("product ID required (use --product or set defaults.product in config)")
	}

	// Get sync status
	status, err := db.GetSyncStatus(product)
	if err != nil {
		return fmt.Errorf("getting sync status: %w", err)
	}

	if len(status) == 0 {
		fmt.Printf("No sync data found for product %s\n", product)
		fmt.Println("Run 'aha-studio sync --product " + product + "' to sync data.")
		return nil
	}

	fmt.Printf("Sync status for product %s:\n\n", product)
	fmt.Printf("  %-12s %-20s %s\n", "ENTITY", "LAST SYNC", "RECORDS")
	fmt.Printf("  %-12s %-20s %s\n", "------", "---------", "-------")

	for entity, s := range status {
		fmt.Printf("  %-12s %-20s %d\n", entity, s.LastSync, s.RecordCount)
	}

	fmt.Printf("\nDatabase: %s\n", sync.DefaultDBPath())

	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	// Parse sync interval
	interval, err := time.ParseDuration(syncInterval)
	if err != nil {
		return fmt.Errorf("invalid sync interval: %w", err)
	}

	// Parse query mode
	var queryMode httpserver.QueryMode
	switch serverPrefer {
	case "api":
		queryMode = httpserver.QueryModeAPI
	case "offline":
		queryMode = httpserver.QueryModeOffline
	case "prefer-cache":
		queryMode = httpserver.QueryModePreferCache
	default:
		return fmt.Errorf("invalid --prefer value: %s (must be api, offline, or prefer-cache)", serverPrefer)
	}

	serverCfg := httpserver.Config{
		Addr:             serverAddr,
		CORSOrigins:      corsOrigins,
		NoAuth:           noAuth,
		OfflineOnly:      serverOffline,
		DefaultQueryMode: queryMode,
		BackgroundSync:   backgroundSync,
		SyncInterval:     interval,
		DefaultProduct:   defaultProduct,
	}

	server, err := httpserver.New(serverCfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return server.Run()
}

func runExec(cmd *cobra.Command, args []string) error {
	statement := args[0]

	// Parse the statement
	p := parser.New(statement)
	stmt, err := p.ParseStatement()
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Prepare mutation options
	opts := executor.MutationOptions{
		DryRun:    dryRun,
		ProductID: productID,
		ReleaseID: releaseID,
	}

	// Apply config defaults for product
	if opts.ProductID == "" && cfg != nil && cfg.Defaults.Product != "" {
		opts.ProductID = cfg.Defaults.Product
	}

	// Create Aha client
	client, err := aha.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Aha client: %w", err)
	}

	exec := executor.New(client)
	ctx := context.Background()

	var res *executor.MutationResult

	switch s := stmt.(type) {
	case *ast.Query:
		// This is a SELECT query, not a mutation
		return fmt.Errorf("use 'aha-studio query' for SELECT queries")

	case *ast.InsertStatement:
		if verbose {
			fmt.Fprintf(os.Stderr, "Operation: INSERT\n")
			fmt.Fprintf(os.Stderr, "Entity: %s\n", s.Entity)
			fmt.Fprintf(os.Stderr, "Columns: %v\n", s.Columns)
			fmt.Fprintf(os.Stderr, "Dry Run: %v\n\n", dryRun)
		}

		if !dryRun && !force {
			if !confirmAction("INSERT into " + string(s.Entity)) {
				fmt.Println("Aborted.")
				return nil
			}
		}

		res, err = exec.ExecuteInsert(ctx, s, opts)

	case *ast.UpdateStatement:
		if verbose {
			fmt.Fprintf(os.Stderr, "Operation: UPDATE\n")
			fmt.Fprintf(os.Stderr, "Entity: %s\n", s.Entity)
			fmt.Fprintf(os.Stderr, "Assignments: %d\n", len(s.Assignments))
			fmt.Fprintf(os.Stderr, "Has WHERE: %v\n", s.Where != nil)
			fmt.Fprintf(os.Stderr, "Dry Run: %v\n\n", dryRun)
		}

		if !dryRun && !force {
			if !confirmAction("UPDATE " + string(s.Entity)) {
				fmt.Println("Aborted.")
				return nil
			}
		}

		res, err = exec.ExecuteUpdate(ctx, s, opts)

	case *ast.DeleteStatement:
		if verbose {
			fmt.Fprintf(os.Stderr, "Operation: DELETE\n")
			fmt.Fprintf(os.Stderr, "Entity: %s\n", s.Entity)
			fmt.Fprintf(os.Stderr, "Has WHERE: %v\n", s.Where != nil)
			fmt.Fprintf(os.Stderr, "Dry Run: %v\n\n", dryRun)
		}

		if !dryRun && !force {
			if !confirmAction("DELETE from " + string(s.Entity)) {
				fmt.Println("Aborted.")
				return nil
			}
		}

		res, err = exec.ExecuteDelete(ctx, s, opts)

	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}

	if err != nil {
		return fmt.Errorf("execution error: %w", err)
	}

	// Display results
	return displayMutationResult(res)
}

func confirmAction(action string) bool {
	fmt.Printf("About to %s. Continue? [y/N]: ", action)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func displayMutationResult(res *executor.MutationResult) error {
	if res.DryRun {
		fmt.Println("=== DRY RUN (no changes made) ===")
	}

	fmt.Printf("Operation: %s\n", res.Operation)
	fmt.Printf("Entity: %s\n", res.Entity)
	fmt.Printf("Affected: %d record(s)\n", res.AffectedCount)

	if len(res.IDs) > 0 {
		fmt.Printf("IDs: %s\n", strings.Join(res.IDs, ", "))
	}

	if len(res.Records) > 0 && res.DryRun {
		fmt.Println("\nRecords that would be affected:")
		formatter := result.NewFormatter(result.FormatTable)
		queryResult := &result.Result{
			Records: res.Records,
		}
		if err := formatter.Format(os.Stdout, queryResult); err != nil {
			return err
		}
	}

	if len(res.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range res.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	return nil
}
