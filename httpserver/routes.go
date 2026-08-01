package httpserver

import (
	"net/http"
)

// registerRoutes sets up all API routes.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health and info endpoints (no auth required)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/version", s.handleVersion)
	mux.HandleFunc("GET /api/openapi.json", s.handleOpenAPI)
	mux.HandleFunc("GET /metrics", s.handleMetrics)

	// Query endpoints
	mux.HandleFunc("GET /api/query", s.handleQueryGET)
	mux.HandleFunc("POST /api/query", s.handleQueryPOST)

	// Reference endpoints
	mux.HandleFunc("GET /api/entities", s.handleEntities)
	mux.HandleFunc("GET /api/syntax", s.handleSyntax)
	mux.HandleFunc("GET /api/products", s.handleProducts)

	// Cache/sync endpoints
	mux.HandleFunc("GET /api/cache/status", s.handleCacheStatus)
	mux.HandleFunc("POST /api/sync", s.handleSync)
	mux.HandleFunc("GET /api/sync/status", s.handleSyncStatus)

	// Saved filters endpoints
	mux.HandleFunc("GET /api/filters", s.handleListFilters)
	mux.HandleFunc("POST /api/filters", s.handleCreateFilter)
	mux.HandleFunc("GET /api/filters/{id}", s.handleGetFilter)
	mux.HandleFunc("PUT /api/filters/{id}", s.handleUpdateFilter)
	mux.HandleFunc("DELETE /api/filters/{id}", s.handleDeleteFilter)

	// Full-text search endpoint
	mux.HandleFunc("GET /api/search", s.handleSearch)

	// Graph endpoints (Neo4j)
	mux.HandleFunc("GET /api/graph/status", s.handleGraphStatus)
	mux.HandleFunc("GET /api/graph/features/{id}/connected", s.handleGraphConnected)
	mux.HandleFunc("GET /api/graph/path", s.handleGraphPath)
	mux.HandleFunc("GET /api/graph/releases/{id}/dependencies", s.handleGraphReleaseDeps)
	mux.HandleFunc("GET /api/graph/initiatives/{id}/impact", s.handleGraphInitiativeImpact)
	mux.HandleFunc("GET /api/graph/products/{id}/overview", s.handleGraphProductOverview)
	mux.HandleFunc("POST /api/graph/cypher", s.handleGraphCypher)
}
