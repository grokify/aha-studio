package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/grokify/aha-studio/aql/parser"
	"github.com/grokify/aha-studio/aql/validator"
	"github.com/grokify/aha-studio/graph"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
	"github.com/grokify/aha-studio/studio"
	"github.com/grokify/aha-studio/sync"
)

// QueryRequest is the request body for POST /api/query.
type QueryRequest struct {
	AQL     string    `json:"aql"`
	Product string    `json:"product,omitempty"`
	Mode    QueryMode `json:"mode,omitempty"` // api, offline, prefer-cache
}

// QueryResponse is the response for query endpoints.
type QueryResponse struct {
	Entity  string          `json:"entity"`
	Count   int             `json:"count"`
	Records []result.Record `json:"records"`
	Source  string          `json:"source"` // "api", "cache", or "offline"
	Error   string          `json:"error,omitempty"`
}

// VersionResponse is the response for /api/version.
type VersionResponse struct {
	Server       string `json:"server"`
	Studio       string `json:"studio"`
	CacheEnabled bool   `json:"cache_enabled"`
	DefaultMode  string `json:"default_mode"`
}

// EntityInfo describes an available entity.
type EntityInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Fields      []string `json:"fields"`
}

// EntitiesResponse is the response for /api/entities.
type EntitiesResponse struct {
	Entities []EntityInfo `json:"entities"`
}

// ProductInfo describes an available product.
type ProductInfo struct {
	ID            string `json:"id"`
	ReferenceNum  string `json:"reference_num"`
	Name          string `json:"name"`
	ProductLineID string `json:"product_line_id,omitempty"`
}

// ProductsResponse is the response for /api/products.
type ProductsResponse struct {
	Products []ProductInfo `json:"products"`
}

// SyntaxResponse is the response for /api/syntax.
type SyntaxResponse struct {
	Syntax    string   `json:"syntax"`
	Operators []string `json:"operators"`
	Examples  []string `json:"examples"`
}

// SyncRequest is the request body for POST /api/sync.
type SyncRequest struct {
	Product     string   `json:"product"`
	Incremental bool     `json:"incremental,omitempty"`
	Entities    []string `json:"entities,omitempty"`
}

// SyncResponse is the response for POST /api/sync.
type SyncResponse struct {
	Product string             `json:"product"`
	Results []SyncEntityResult `json:"results"`
	Error   string             `json:"error,omitempty"`
}

// SyncEntityResult contains the result of syncing a single entity.
type SyncEntityResult struct {
	Entity      string `json:"entity"`
	RecordCount int    `json:"record_count"`
	Duration    string `json:"duration"`
	Error       string `json:"error,omitempty"`
}

// SyncStatusResponse is the response for GET /api/sync/status.
type SyncStatusResponse struct {
	Product  string                    `json:"product"`
	Entities map[string]EntitySyncInfo `json:"entities"`
}

// EntitySyncInfo contains sync status for a single entity.
type EntitySyncInfo struct {
	LastSync    string `json:"last_sync"`
	RecordCount int    `json:"record_count"`
}

// CacheStatusResponse is the response for GET /api/cache/status.
type CacheStatusResponse struct {
	Enabled      bool   `json:"enabled"`
	DatabasePath string `json:"database_path"`
	DefaultMode  string `json:"default_mode"`
	OfflineOnly  bool   `json:"offline_only"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// handleHealth handles GET /health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleVersion handles GET /api/version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, VersionResponse{
		Server:       Version,
		Studio:       studio.Version,
		CacheEnabled: s.db != nil,
		DefaultMode:  string(s.config.DefaultQueryMode),
	})
}

// handleQueryGET handles GET /api/query?aql=...
func (s *Server) handleQueryGET(w http.ResponseWriter, r *http.Request) {
	aql := r.URL.Query().Get("aql")
	if aql == "" {
		writeError(w, http.StatusBadRequest, "aql parameter is required", "MISSING_AQL")
		return
	}

	product := r.URL.Query().Get("product")
	mode := QueryMode(r.URL.Query().Get("mode"))

	s.executeQuery(w, r, aql, product, mode)
}

// handleQueryPOST handles POST /api/query.
func (s *Server) handleQueryPOST(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}

	if req.AQL == "" {
		writeError(w, http.StatusBadRequest, "aql field is required", "MISSING_AQL")
		return
	}

	s.executeQuery(w, r, req.AQL, req.Product, req.Mode)
}

// executeQuery executes an AQL query and writes the response.
func (s *Server) executeQuery(w http.ResponseWriter, r *http.Request, aql, product string, requestedMode QueryMode) {
	ctx := r.Context()

	// Validate mode
	if !requestedMode.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid mode: must be api, offline, or prefer-cache", "INVALID_MODE")
		return
	}

	// Resolve effective mode
	mode := s.EffectiveQueryMode(requestedMode)

	// Parse and validate the query first
	p := parser.New(aql)
	query, err := p.Parse()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_AQL")
		return
	}

	v := validator.New()
	if err := v.Validate(query); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "INVALID_AQL")
		return
	}

	// Create execution plan
	pl := planner.New()
	plan := pl.Plan(query)

	// Apply product context
	if product != "" {
		plan.APIParams.ProductID = product
	}

	var res *result.Result
	var source string

	switch mode {
	case QueryModeOffline:
		res, source, err = s.executeOfflineQuery(ctx, plan, product)

	case QueryModePreferCache:
		res, source, err = s.executePreferCacheQuery(ctx, plan, product, aql)

	default: // QueryModeAPI
		res, source, err = s.executeAPIQuery(ctx, aql, product)
	}

	if err != nil {
		s.logger.Error("query execution failed", "aql", aql, "mode", mode, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error(), "EXECUTION_ERROR")
		return
	}

	// Build response
	response := QueryResponse{
		Entity:  string(res.Entity),
		Count:   res.Count(),
		Records: res.Records,
		Source:  source,
	}

	writeJSON(w, http.StatusOK, response)
}

// executeAPIQuery executes a query against the live Aha API.
func (s *Server) executeAPIQuery(ctx context.Context, aql, product string) (*result.Result, string, error) {
	if s.studio == nil {
		return nil, "", errOfflineOnlyMode
	}

	opts := studio.QueryOptions{
		ProductID: product,
	}

	res, err := s.studio.QueryWithOptions(ctx, aql, opts)
	if err != nil {
		return nil, "", err
	}

	return res, "api", nil
}

// executeOfflineQuery executes a query against the local SQLite cache.
func (s *Server) executeOfflineQuery(ctx context.Context, plan *planner.Plan, product string) (*result.Result, string, error) {
	if s.db == nil {
		return nil, "", errCacheNotAvailable
	}

	// Set product context
	if product != "" {
		s.db.SetProduct(product)
	}

	res, err := s.db.QueryOffline(plan)
	if err != nil {
		return nil, "", err
	}

	return res, "cache", nil
}

// executePreferCacheQuery tries the cache first, falls back to API.
func (s *Server) executePreferCacheQuery(ctx context.Context, plan *planner.Plan, product, aql string) (*result.Result, string, error) {
	// Try cache first
	if s.db != nil {
		if product != "" {
			s.db.SetProduct(product)
		}

		res, err := s.db.QueryOffline(plan)
		if err == nil && res.Count() > 0 {
			return res, "cache", nil
		}

		// Log cache miss
		s.logger.Debug("cache miss, falling back to API", "aql", aql)
	}

	// Fall back to API
	return s.executeAPIQuery(ctx, aql, product)
}

// handleEntities handles GET /api/entities.
func (s *Server) handleEntities(w http.ResponseWriter, r *http.Request) {
	entities := []EntityInfo{
		{
			Name:        "features",
			Description: "Product features and requirements",
			Fields:      []string{"id", "reference_num", "name", "description", "status", "created_at", "updated_at", "workflow_status", "assigned_to", "product_id"},
		},
		{
			Name:        "ideas",
			Description: "Customer ideas and feedback",
			Fields:      []string{"id", "reference_num", "name", "description", "status", "votes", "created_at", "updated_at", "workflow_status", "categories"},
		},
		{
			Name:        "releases",
			Description: "Product releases and versions",
			Fields:      []string{"id", "reference_num", "name", "release_date", "released", "created_at", "updated_at", "product_id"},
		},
		{
			Name:        "initiatives",
			Description: "Strategic initiatives",
			Fields:      []string{"id", "reference_num", "name", "description", "status", "created_at", "updated_at"},
		},
		{
			Name:        "goals",
			Description: "Business goals and objectives",
			Fields:      []string{"id", "reference_num", "name", "description", "status", "created_at", "updated_at"},
		},
		{
			Name:        "epics",
			Description: "Large bodies of work",
			Fields:      []string{"id", "reference_num", "name", "description", "status", "created_at", "updated_at"},
		},
		{
			Name:        "products",
			Description: "Products in your Aha account",
			Fields:      []string{"id", "reference_prefix", "name", "product_line_id", "created_at", "updated_at"},
		},
		{
			Name:        "users",
			Description: "Users in your Aha account",
			Fields:      []string{"id", "email", "name", "created_at", "updated_at"},
		},
		{
			Name:        "requirements",
			Description: "Feature requirements",
			Fields:      []string{"id", "reference_num", "name", "description", "status", "created_at", "updated_at", "feature_id"},
		},
		{
			Name:        "comments",
			Description: "Comments on records",
			Fields:      []string{"id", "body", "created_at", "updated_at", "user_id", "commentable_type", "commentable_id"},
		},
		{
			Name:        "tags",
			Description: "Tags for organizing records",
			Fields:      []string{"id", "name", "color"},
		},
	}

	writeJSON(w, http.StatusOK, EntitiesResponse{Entities: entities})
}

// handleSyntax handles GET /api/syntax.
func (s *Server) handleSyntax(w http.ResponseWriter, r *http.Request) {
	response := SyntaxResponse{
		Syntax: `FROM <entity> [SELECT <fields>] [WHERE <conditions>] [ORDER BY <field> [ASC|DESC]] [LIMIT <n>]`,
		Operators: []string{
			"=", "!=", "<", "<=", ">", ">=",
			"IN", "NOT IN",
			"CONTAINS", "LIKE",
			"IS NULL", "IS NOT NULL",
			"AND", "OR", "NOT",
		},
		Examples: []string{
			"FROM features LIMIT 10",
			"FROM features WHERE status = 'In Progress'",
			"FROM ideas WHERE votes > 10 ORDER BY votes DESC",
			"FROM ideas WHERE name CONTAINS 'API' LIMIT 5",
			"FROM features WHERE updated_at >= now() - duration('30d')",
			"FROM releases WHERE product_id = 'PLATFORM'",
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// handleProducts handles GET /api/products.
func (s *Server) handleProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.studio == nil {
		writeError(w, http.StatusServiceUnavailable, "API not available in offline mode", "OFFLINE_MODE")
		return
	}

	// Query products using AQL
	res, err := s.studio.QueryWithOptions(ctx, "FROM products", studio.QueryOptions{})
	if err != nil {
		s.logger.Error("failed to list products", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list products", "EXECUTION_ERROR")
		return
	}

	products := make([]ProductInfo, 0, len(res.Records))
	for _, rec := range res.Records {
		products = append(products, ProductInfo{
			ID:            getString(rec, "id"),
			ReferenceNum:  getString(rec, "reference_prefix"),
			Name:          getString(rec, "name"),
			ProductLineID: getString(rec, "product_line_id"),
		})
	}

	writeJSON(w, http.StatusOK, ProductsResponse{Products: products})
}

// handleCacheStatus handles GET /api/cache/status.
func (s *Server) handleCacheStatus(w http.ResponseWriter, r *http.Request) {
	response := CacheStatusResponse{
		Enabled:      s.db != nil,
		DatabasePath: sync.DefaultDBPath(),
		DefaultMode:  string(s.config.DefaultQueryMode),
		OfflineOnly:  s.config.OfflineOnly,
	}

	writeJSON(w, http.StatusOK, response)
}

// handleSync handles POST /api/sync.
func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if s.syncer == nil {
		writeError(w, http.StatusServiceUnavailable, "sync not available (requires API access and cache)", "SYNC_UNAVAILABLE")
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}

	if req.Product == "" {
		// Use default product if available
		if s.config.DefaultProduct != "" {
			req.Product = s.config.DefaultProduct
		} else {
			writeError(w, http.StatusBadRequest, "product is required", "MISSING_PRODUCT")
			return
		}
	}

	ctx := r.Context()

	opts := sync.SyncOptions{
		Product:     req.Product,
		Incremental: req.Incremental,
		Entities:    req.Entities,
	}

	s.logger.Info("starting sync", "product", req.Product, "incremental", req.Incremental, "entities", req.Entities)

	results, err := s.syncer.SyncAll(ctx, opts)
	if err != nil {
		s.logger.Error("sync failed", "product", req.Product, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error(), "SYNC_ERROR")
		return
	}

	// Convert results
	entityResults := make([]SyncEntityResult, 0, len(results))
	for _, r := range results {
		entityResult := SyncEntityResult{
			Entity:      r.Entity,
			RecordCount: r.RecordCount,
			Duration:    r.Duration.Round(time.Millisecond).String(),
		}
		if r.Error != nil {
			entityResult.Error = r.Error.Error()
		}
		entityResults = append(entityResults, entityResult)
	}

	s.logger.Info("sync completed", "product", req.Product, "entities", len(entityResults))

	writeJSON(w, http.StatusOK, SyncResponse{
		Product: req.Product,
		Results: entityResults,
	})
}

// handleSyncStatus handles GET /api/sync/status.
func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available", "CACHE_UNAVAILABLE")
		return
	}

	product := r.URL.Query().Get("product")
	if product == "" {
		product = s.config.DefaultProduct
	}
	if product == "" {
		writeError(w, http.StatusBadRequest, "product query parameter is required", "MISSING_PRODUCT")
		return
	}

	status, err := s.db.GetSyncStatus(product)
	if err != nil {
		s.logger.Error("failed to get sync status", "product", product, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error(), "STATUS_ERROR")
		return
	}

	// Convert to response format
	entities := make(map[string]EntitySyncInfo)
	for name, info := range status {
		entities[name] = EntitySyncInfo{
			LastSync:    info.LastSync,
			RecordCount: info.RecordCount,
		}
	}

	writeJSON(w, http.StatusOK, SyncStatusResponse{
		Product:  product,
		Entities: entities,
	})
}

// getString safely extracts a string from a record.
func getString(rec result.Record, key string) string {
	if v, ok := rec[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Log error but can't change response at this point
		_ = err
	}
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// FilterRequest is the request body for creating/updating a saved filter.
type FilterRequest struct {
	Name        string `json:"name"`
	AQL         string `json:"aql"`
	Product     string `json:"product,omitempty"`
	Description string `json:"description,omitempty"`
	IsFavorite  bool   `json:"is_favorite,omitempty"`
}

// FilterResponse is the response for a single saved filter.
type FilterResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AQL         string `json:"aql"`
	Product     string `json:"product,omitempty"`
	Description string `json:"description,omitempty"`
	IsFavorite  bool   `json:"is_favorite"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// FiltersResponse is the response for listing saved filters.
type FiltersResponse struct {
	Filters []FilterResponse `json:"filters"`
}

// handleListFilters handles GET /api/filters.
func (s *Server) handleListFilters(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available, filters require cache database", "CACHE_UNAVAILABLE")
		return
	}

	filters, err := s.db.ListFilters()
	if err != nil {
		s.logger.Error("failed to list filters", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list filters", "DB_ERROR")
		return
	}

	response := make([]FilterResponse, 0, len(filters))
	for _, f := range filters {
		response = append(response, filterToResponse(f))
	}

	writeJSON(w, http.StatusOK, FiltersResponse{Filters: response})
}

// handleCreateFilter handles POST /api/filters.
func (s *Server) handleCreateFilter(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available, filters require cache database", "CACHE_UNAVAILABLE")
		return
	}

	var req FilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required", "MISSING_NAME")
		return
	}
	if req.AQL == "" {
		writeError(w, http.StatusBadRequest, "aql is required", "MISSING_AQL")
		return
	}

	// Validate AQL syntax
	p := parser.New(req.AQL)
	query, err := p.Parse()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid AQL: "+err.Error(), "INVALID_AQL")
		return
	}
	v := validator.New()
	if err := v.Validate(query); err != nil {
		writeError(w, http.StatusBadRequest, "invalid AQL: "+err.Error(), "INVALID_AQL")
		return
	}

	// Check for duplicate name
	existing, err := s.db.GetFilterByName(req.Name)
	if err != nil {
		s.logger.Error("failed to check for duplicate filter", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create filter", "DB_ERROR")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "filter with this name already exists", "DUPLICATE_NAME")
		return
	}

	filter := &sync.SavedFilter{
		Name:        req.Name,
		AQL:         req.AQL,
		Product:     req.Product,
		Description: req.Description,
		IsFavorite:  req.IsFavorite,
	}

	if err := s.db.CreateFilter(filter); err != nil {
		s.logger.Error("failed to create filter", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create filter", "DB_ERROR")
		return
	}

	s.logger.Info("filter created", "id", filter.ID, "name", filter.Name)

	writeJSON(w, http.StatusCreated, filterToResponse(*filter))
}

// handleGetFilter handles GET /api/filters/{id}.
func (s *Server) handleGetFilter(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available, filters require cache database", "CACHE_UNAVAILABLE")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "filter ID is required", "MISSING_ID")
		return
	}

	filter, err := s.db.GetFilter(id)
	if err != nil {
		s.logger.Error("failed to get filter", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get filter", "DB_ERROR")
		return
	}
	if filter == nil {
		writeError(w, http.StatusNotFound, "filter not found", "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, filterToResponse(*filter))
}

// handleUpdateFilter handles PUT /api/filters/{id}.
func (s *Server) handleUpdateFilter(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available, filters require cache database", "CACHE_UNAVAILABLE")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "filter ID is required", "MISSING_ID")
		return
	}

	// Check if filter exists
	existing, err := s.db.GetFilter(id)
	if err != nil {
		s.logger.Error("failed to get filter for update", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update filter", "DB_ERROR")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "filter not found", "NOT_FOUND")
		return
	}

	var req FilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}

	// Validate AQL if provided
	if req.AQL != "" {
		p := parser.New(req.AQL)
		query, err := p.Parse()
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid AQL: "+err.Error(), "INVALID_AQL")
			return
		}
		v := validator.New()
		if err := v.Validate(query); err != nil {
			writeError(w, http.StatusBadRequest, "invalid AQL: "+err.Error(), "INVALID_AQL")
			return
		}
		existing.AQL = req.AQL
	}

	// Update fields if provided
	if req.Name != "" {
		// Check for duplicate name if name is changing
		if req.Name != existing.Name {
			dup, err := s.db.GetFilterByName(req.Name)
			if err != nil {
				s.logger.Error("failed to check for duplicate filter", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to update filter", "DB_ERROR")
				return
			}
			if dup != nil {
				writeError(w, http.StatusConflict, "filter with this name already exists", "DUPLICATE_NAME")
				return
			}
		}
		existing.Name = req.Name
	}
	existing.Product = req.Product
	existing.Description = req.Description
	existing.IsFavorite = req.IsFavorite

	if err := s.db.UpdateFilter(existing); err != nil {
		s.logger.Error("failed to update filter", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update filter", "DB_ERROR")
		return
	}

	s.logger.Info("filter updated", "id", id, "name", existing.Name)

	writeJSON(w, http.StatusOK, filterToResponse(*existing))
}

// handleDeleteFilter handles DELETE /api/filters/{id}.
func (s *Server) handleDeleteFilter(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available, filters require cache database", "CACHE_UNAVAILABLE")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "filter ID is required", "MISSING_ID")
		return
	}

	if err := s.db.DeleteFilter(id); err != nil {
		if err.Error() == "sql: no rows in result set" {
			writeError(w, http.StatusNotFound, "filter not found", "NOT_FOUND")
			return
		}
		s.logger.Error("failed to delete filter", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete filter", "DB_ERROR")
		return
	}

	s.logger.Info("filter deleted", "id", id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// filterToResponse converts a SavedFilter to FilterResponse.
func filterToResponse(f sync.SavedFilter) FilterResponse {
	return FilterResponse{
		ID:          f.ID,
		Name:        f.Name,
		AQL:         f.AQL,
		Product:     f.Product,
		Description: f.Description,
		IsFavorite:  f.IsFavorite,
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   f.UpdatedAt.Format(time.RFC3339),
	}
}

// SearchRequest is the request for full-text search.
type SearchRequest struct {
	Query    string   `json:"query"`
	Entities []string `json:"entities,omitempty"` // optional filter by entity types
	Limit    int      `json:"limit,omitempty"`    // default 100
}

// SearchResult represents a single search hit.
type SearchResult struct {
	Entity      string  `json:"entity"`
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// SearchResponse is the response for full-text search.
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
}

// handleSearch handles GET /api/search.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "search requires cache database", "CACHE_UNAVAILABLE")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required", "MISSING_QUERY")
		return
	}

	// Parse entity filter
	var entities []string
	if e := r.URL.Query().Get("entities"); e != "" {
		entities = splitEntities(e)
	}

	// Parse limit
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parseInt(l); err == nil && n > 0 {
			limit = n
		}
	}

	results, err := s.db.FullTextSearch(query, entities, limit)
	if err != nil {
		s.logger.Error("search failed", "query", query, "error", err)
		writeError(w, http.StatusInternalServerError, "search failed", "SEARCH_ERROR")
		return
	}

	searchResults := make([]SearchResult, 0, len(results))
	for _, r := range results {
		searchResults = append(searchResults, SearchResult{
			Entity:      getString(r, "entity"),
			ID:          getString(r, "id"),
			Name:        getString(r, "name"),
			Description: getString(r, "description"),
			Score:       getFloat(r, "score"),
		})
	}

	writeJSON(w, http.StatusOK, SearchResponse{
		Query:   query,
		Results: searchResults,
		Count:   len(searchResults),
	})
}

// splitEntities splits a comma-separated list of entity types.
func splitEntities(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// parseInt parses an integer from a string.
func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errString("invalid integer")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// getFloat safely extracts a float64 from a map.
func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// GraphStatusResponse is the response for GET /api/graph/status.
type GraphStatusResponse struct {
	Enabled bool   `json:"enabled"`
	URI     string `json:"uri,omitempty"`
}

// GraphQueryRequest is the request for POST /api/graph/query.
type GraphQueryRequest struct {
	Cypher string         `json:"cypher"`
	Params map[string]any `json:"params,omitempty"`
}

// handleGraphStatus handles GET /api/graph/status.
func (s *Server) handleGraphStatus(w http.ResponseWriter, r *http.Request) {
	response := GraphStatusResponse{
		Enabled: s.graphClient != nil,
	}
	writeJSON(w, http.StatusOK, response)
}

// handleGraphConnected handles GET /api/graph/features/{id}/connected.
func (s *Server) handleGraphConnected(w http.ResponseWriter, r *http.Request) {
	if s.graphClient == nil {
		writeError(w, http.StatusServiceUnavailable, "graph database not configured", "GRAPH_UNAVAILABLE")
		return
	}

	entityID := r.PathValue("id")
	if entityID == "" {
		writeError(w, http.StatusBadRequest, "entity ID is required", "MISSING_ID")
		return
	}

	entityType := r.URL.Query().Get("type")
	if entityType == "" {
		entityType = "Feature"
	}

	depth := 2
	if d := r.URL.Query().Get("depth"); d != "" {
		if n, err := parseInt(d); err == nil && n > 0 && n <= 5 {
			depth = n
		}
	}

	ctx := r.Context()
	result, err := s.graphClient.FindConnectedFeatures(ctx, graph.NodeLabel(entityType), entityID, depth)
	if err != nil {
		s.logger.Error("graph query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "graph query failed", "GRAPH_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGraphPath handles GET /api/graph/path.
func (s *Server) handleGraphPath(w http.ResponseWriter, r *http.Request) {
	if s.graphClient == nil {
		writeError(w, http.StatusServiceUnavailable, "graph database not configured", "GRAPH_UNAVAILABLE")
		return
	}

	fromType := r.URL.Query().Get("from_type")
	fromID := r.URL.Query().Get("from_id")
	toType := r.URL.Query().Get("to_type")
	toID := r.URL.Query().Get("to_id")

	if fromID == "" || toID == "" {
		writeError(w, http.StatusBadRequest, "from_id and to_id are required", "MISSING_PARAMS")
		return
	}

	if fromType == "" {
		fromType = "Feature"
	}
	if toType == "" {
		toType = "Feature"
	}

	ctx := r.Context()
	result, err := s.graphClient.FindPath(ctx, graph.NodeLabel(fromType), fromID, graph.NodeLabel(toType), toID)
	if err != nil {
		s.logger.Error("graph path query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "graph query failed", "GRAPH_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGraphReleaseDeps handles GET /api/graph/releases/{id}/dependencies.
func (s *Server) handleGraphReleaseDeps(w http.ResponseWriter, r *http.Request) {
	if s.graphClient == nil {
		writeError(w, http.StatusServiceUnavailable, "graph database not configured", "GRAPH_UNAVAILABLE")
		return
	}

	releaseID := r.PathValue("id")
	if releaseID == "" {
		writeError(w, http.StatusBadRequest, "release ID is required", "MISSING_ID")
		return
	}

	ctx := r.Context()
	result, err := s.graphClient.GetReleaseDependencies(ctx, releaseID)
	if err != nil {
		s.logger.Error("graph release dependencies query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "graph query failed", "GRAPH_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGraphInitiativeImpact handles GET /api/graph/initiatives/{id}/impact.
func (s *Server) handleGraphInitiativeImpact(w http.ResponseWriter, r *http.Request) {
	if s.graphClient == nil {
		writeError(w, http.StatusServiceUnavailable, "graph database not configured", "GRAPH_UNAVAILABLE")
		return
	}

	initiativeID := r.PathValue("id")
	if initiativeID == "" {
		writeError(w, http.StatusBadRequest, "initiative ID is required", "MISSING_ID")
		return
	}

	ctx := r.Context()
	result, err := s.graphClient.GetInitiativeImpact(ctx, initiativeID)
	if err != nil {
		s.logger.Error("graph initiative impact query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "graph query failed", "GRAPH_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGraphProductOverview handles GET /api/graph/products/{id}/overview.
func (s *Server) handleGraphProductOverview(w http.ResponseWriter, r *http.Request) {
	if s.graphClient == nil {
		writeError(w, http.StatusServiceUnavailable, "graph database not configured", "GRAPH_UNAVAILABLE")
		return
	}

	productID := r.PathValue("id")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "product ID is required", "MISSING_ID")
		return
	}

	ctx := r.Context()
	result, err := s.graphClient.GetProductOverview(ctx, productID)
	if err != nil {
		s.logger.Error("graph product overview query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "graph query failed", "GRAPH_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleGraphCypher handles POST /api/graph/cypher (raw Cypher queries).
func (s *Server) handleGraphCypher(w http.ResponseWriter, r *http.Request) {
	if s.graphClient == nil {
		writeError(w, http.StatusServiceUnavailable, "graph database not configured", "GRAPH_UNAVAILABLE")
		return
	}

	var req GraphQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", "INVALID_JSON")
		return
	}

	if req.Cypher == "" {
		writeError(w, http.StatusBadRequest, "cypher query is required", "MISSING_CYPHER")
		return
	}

	ctx := r.Context()
	result, err := s.graphClient.RunCypher(ctx, req.Cypher, req.Params)
	if err != nil {
		s.logger.Error("cypher query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "cypher query failed: "+err.Error(), "CYPHER_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Error variables for common errors.
var (
	errOfflineOnlyMode   = errString("API not available in offline-only mode")
	errCacheNotAvailable = errString("cache database not available")
)

type errString string

func (e errString) Error() string { return string(e) }
