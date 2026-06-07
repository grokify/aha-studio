// Package httpserver provides an HTTP API server for AQL queries.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	aha "github.com/grokify/aha-go"
	"github.com/grokify/aha-studio/graph"
	"github.com/grokify/aha-studio/studio"
	"github.com/grokify/aha-studio/sync"
)

// Version is the HTTP server version.
const Version = "0.2.0"

// QueryMode specifies how queries are executed.
type QueryMode string

const (
	// QueryModeAPI queries the live Aha API (default).
	QueryModeAPI QueryMode = "api"

	// QueryModeOffline queries only the local SQLite cache.
	QueryModeOffline QueryMode = "offline"

	// QueryModePreferCache tries the local cache first, falls back to API.
	QueryModePreferCache QueryMode = "prefer-cache"
)

// IsValid returns true if the query mode is valid.
func (m QueryMode) IsValid() bool {
	switch m {
	case QueryModeAPI, QueryModeOffline, QueryModePreferCache:
		return true
	case "": // empty is valid, defaults to server setting
		return true
	}
	return false
}

// Config configures the HTTP server.
type Config struct {
	// Addr is the address to listen on (e.g., ":8080").
	Addr string

	// CORSOrigins specifies allowed CORS origins.
	// Use "*" to allow all origins.
	CORSOrigins []string

	// APIKey is the API key for authenticating requests.
	// If empty and NoAuth is false, uses AHA_API_KEY from environment.
	APIKey string

	// NoAuth disables API key authentication (for local development).
	NoAuth bool

	// Logger is the server logger. If nil, a default logger is used.
	Logger *slog.Logger

	// DefaultQueryMode is the default query mode when not specified in request.
	// If empty, defaults to "prefer-cache" when cache is available, otherwise "api".
	DefaultQueryMode QueryMode

	// OfflineOnly restricts the server to offline-only mode (no API calls).
	OfflineOnly bool

	// BackgroundSync enables automatic background sync.
	BackgroundSync bool

	// SyncInterval is the interval for background sync (default: 15 minutes).
	SyncInterval time.Duration

	// DefaultProduct is the default product for sync operations.
	DefaultProduct string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Addr:             ":8080",
		CORSOrigins:      []string{"*"},
		NoAuth:           false,
		DefaultQueryMode: QueryModePreferCache,
		SyncInterval:     15 * time.Minute,
	}
}

// Server is the HTTP API server for AQL queries.
type Server struct {
	config      Config
	httpServer  *http.Server
	studio      *studio.Studio
	client      *aha.Client
	db          *sync.DB
	syncer      *sync.Syncer
	scheduler   *sync.Scheduler
	graphClient *graph.Client
	logger      *slog.Logger
}

// New creates a new HTTP server with the given configuration.
func New(cfg Config) (*Server, error) {
	var client *aha.Client
	var studioInstance *studio.Studio

	// Only create Aha client if not in offline-only mode
	if !cfg.OfflineOnly {
		var err error
		client, err = aha.NewClient()
		if err != nil {
			return nil, fmt.Errorf("creating Aha client: %w", err)
		}
		studioInstance = studio.New(client)
	}

	// Set up logger
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	// Open SQLite database for caching
	db, err := sync.Open(sync.DefaultDBPath())
	if err != nil {
		logger.Warn("failed to open cache database, cache features disabled", "error", err)
	}

	// Create syncer if we have both client and db
	var syncer *sync.Syncer
	if client != nil && db != nil {
		syncer = sync.NewSyncer(db, client)
	}

	// Try to create graph client if Neo4j is configured
	var graphClient *graph.Client
	if os.Getenv("NEO4J_PASSWORD") != "" {
		graphCfg := graph.ConfigFromEnv()
		var err error
		graphClient, err = graph.NewClient(graphCfg)
		if err != nil {
			logger.Warn("failed to create graph client, graph features disabled", "error", err)
		} else {
			// Verify connectivity
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if verifyErr := graphClient.Verify(ctx); verifyErr != nil {
				logger.Warn("failed to connect to Neo4j, graph features disabled", "error", verifyErr)
				graphClient = nil
			} else {
				logger.Info("connected to Neo4j", "uri", graphCfg.URI)
			}
			cancel()
		}
	}

	// Apply defaults
	if cfg.DefaultQueryMode == "" {
		if db != nil {
			cfg.DefaultQueryMode = QueryModePreferCache
		} else {
			cfg.DefaultQueryMode = QueryModeAPI
		}
	}

	// Validate offline-only mode
	if cfg.OfflineOnly && db == nil {
		return nil, errors.New("offline-only mode requires a working cache database")
	}

	s := &Server{
		config:      cfg,
		studio:      studioInstance,
		client:      client,
		db:          db,
		syncer:      syncer,
		graphClient: graphClient,
		logger:      logger,
	}

	// Set up routes
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Build middleware chain
	var handler http.Handler = mux
	handler = s.corsMiddleware(handler)
	if !cfg.NoAuth {
		handler = s.authMiddleware(handler)
	}
	handler = s.loggingMiddleware(handler)
	handler = s.recoveryMiddleware(handler)

	s.httpServer = &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s, nil
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}

	// Start background sync if enabled
	if s.config.BackgroundSync && s.syncer != nil {
		interval := s.config.SyncInterval
		if interval == 0 {
			interval = 15 * time.Minute
		}
		schedulerCfg := sync.SchedulerConfig{
			Interval: interval,
			Product:  s.config.DefaultProduct,
			Logger:   s.logger,
		}
		s.scheduler = sync.NewScheduler(s.syncer, schedulerCfg)
		s.scheduler.Start(context.Background())
		s.logger.Info("background sync started", "interval", interval, "product", s.config.DefaultProduct)
	}

	s.logger.Info("HTTP server starting",
		"addr", listener.Addr().String(),
		"auth", !s.config.NoAuth,
		"cors", s.config.CORSOrigins,
		"mode", s.config.DefaultQueryMode,
		"offline_only", s.config.OfflineOnly,
		"cache_enabled", s.db != nil,
	)

	if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ListenAndServe starts the server and blocks until shutdown.
func (s *Server) ListenAndServe() error {
	return s.Start()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down")

	// Stop background sync
	if s.scheduler != nil {
		s.scheduler.Stop()
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Warn("error closing cache database", "error", err)
		}
	}

	// Close graph client
	if s.graphClient != nil {
		if err := s.graphClient.Close(ctx); err != nil {
			s.logger.Warn("error closing graph client", "error", err)
		}
	}

	return s.httpServer.Shutdown(ctx)
}

// Run starts the server and handles graceful shutdown on SIGINT/SIGTERM.
func (s *Server) Run() error {
	// Channel for shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Channel for server errors
	errCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := s.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-stop:
		s.logger.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		return err
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.Shutdown(ctx)
}

// CacheEnabled returns true if the cache database is available.
func (s *Server) CacheEnabled() bool {
	return s.db != nil
}

// GraphEnabled returns true if the Neo4j graph client is available.
func (s *Server) GraphEnabled() bool {
	return s.graphClient != nil
}

// EffectiveQueryMode returns the query mode to use, resolving empty to default.
func (s *Server) EffectiveQueryMode(requested QueryMode) QueryMode {
	if requested == "" {
		return s.config.DefaultQueryMode
	}
	// In offline-only mode, force offline
	if s.config.OfflineOnly {
		return QueryModeOffline
	}
	return requested
}
