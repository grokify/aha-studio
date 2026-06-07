//nolint:dupl // Test functions have intentionally similar structure for consistency
package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	// Skip if no API key (can't create server)
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}
}

func TestHandleVersion(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/version", s.handleVersion)

	req := httptest.NewRequest("GET", "/api/version", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Server == "" || resp.Studio == "" {
		t.Error("expected non-empty version strings")
	}
}

func TestHandleEntities(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/entities", s.handleEntities)

	req := httptest.NewRequest("GET", "/api/entities", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp EntitiesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Entities) == 0 {
		t.Error("expected non-empty entities list")
	}

	// Check that features entity is present
	found := false
	for _, e := range resp.Entities {
		if e.Name == "features" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'features' entity in list")
	}
}

func TestHandleSyntax(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/syntax", s.handleSyntax)

	req := httptest.NewRequest("GET", "/api/syntax", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp SyntaxResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Syntax == "" {
		t.Error("expected non-empty syntax string")
	}

	if len(resp.Operators) == 0 {
		t.Error("expected non-empty operators list")
	}

	if len(resp.Examples) == 0 {
		t.Error("expected non-empty examples list")
	}
}

func TestHandleQueryGET_MissingAQL(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/query", s.handleQueryGET)

	req := httptest.NewRequest("GET", "/api/query", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "MISSING_AQL" {
		t.Errorf("expected code 'MISSING_AQL', got '%s'", resp.Code)
	}
}

func TestHandleQueryPOST_InvalidJSON(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/query", s.handleQueryPOST)

	req := httptest.NewRequest("POST", "/api/query", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Errorf("expected code 'INVALID_JSON', got '%s'", resp.Code)
	}
}

func TestHandleQueryPOST_MissingAQL(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/query", s.handleQueryPOST)

	body := `{"product": "PLATFORM"}`
	req := httptest.NewRequest("POST", "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "MISSING_AQL" {
		t.Errorf("expected code 'MISSING_AQL', got '%s'", resp.Code)
	}
}

func TestHandleQueryPOST_InvalidAQL(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/query", s.handleQueryPOST)

	body := `{"aql": "INVALID QUERY SYNTAX"}`
	req := httptest.NewRequest("POST", "/api/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "INVALID_AQL" {
		t.Errorf("expected code 'INVALID_AQL', got '%s'", resp.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.NoAuth = true
	cfg.CORSOrigins = []string{"https://example.com"}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create handler with CORS middleware
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/api/query", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d for preflight, got %d", http.StatusNoContent, rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected CORS origin header")
	}

	// Test disallowed origin
	req = httptest.NewRequest("OPTIONS", "/api/query", nil)
	req.Header.Set("Origin", "https://other.com")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header for disallowed origin")
	}
}

func TestAuthMiddleware(t *testing.T) {
	if os.Getenv("AHA_API_KEY") == "" {
		t.Skip("AHA_API_KEY not set")
	}

	cfg := DefaultConfig()
	cfg.APIKey = "test-api-key"
	cfg.NoAuth = false

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create handler with auth middleware
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test request without API key
	req := httptest.NewRequest("GET", "/api/query", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Test request with valid API key in X-API-Key header
	req = httptest.NewRequest("GET", "/api/query", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Test request with valid API key in Authorization header
	req = httptest.NewRequest("GET", "/api/query", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Test request with invalid API key
	req = httptest.NewRequest("GET", "/api/query", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Test health endpoint bypasses auth
	req = httptest.NewRequest("GET", "/health", nil)
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for health endpoint, got %d", http.StatusOK, rec.Code)
	}
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "X-API-Key header",
			setup: func(r *http.Request) {
				r.Header.Set("X-API-Key", "header-key")
			},
			expected: "header-key",
		},
		{
			name: "Authorization Bearer header",
			setup: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer bearer-key")
			},
			expected: "bearer-key",
		},
		{
			name: "Query parameter",
			setup: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("api_key", "query-key")
				r.URL.RawQuery = q.Encode()
			},
			expected: "query-key",
		},
		{
			name:     "No key",
			setup:    func(r *http.Request) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/query", nil)
			tt.setup(req)

			got := extractAPIKey(req)
			if got != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Addr != ":8080" {
		t.Errorf("expected default addr ':8080', got '%s'", cfg.Addr)
	}

	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "*" {
		t.Error("expected default CORS origins to be ['*']")
	}

	if cfg.NoAuth != false {
		t.Error("expected NoAuth to be false by default")
	}
}
