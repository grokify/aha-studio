//go:build integration

package mcp_test

import (
	"context"
	"os"
	"testing"

	"github.com/grokify/aha-studio/mcp"
)

// TestGetResourceByID_Integration tests that getResourceByID-based handlers
// correctly construct API URLs (no doubled /api/v1 path).
//
// Run with: go test -tags=integration -v ./mcp/...
//
// Requires: AHA_SUBDOMAIN and AHA_API_KEY environment variables.
// Optionally set test IDs: AHA_TEST_EPIC_ID, AHA_TEST_GOAL_ID, etc.
func TestGetResourceByID_Integration(t *testing.T) {
	subdomain := os.Getenv("AHA_SUBDOMAIN")
	apiKey := os.Getenv("AHA_API_KEY")

	if subdomain == "" || apiKey == "" {
		t.Skip("AHA_SUBDOMAIN and AHA_API_KEY must be set for integration tests")
	}

	h := mcp.NewToolHandlers(&mcp.Config{
		Subdomain: subdomain,
		APIKey:    apiKey,
	})

	ctx := context.Background()

	// Test cases for getResourceByID-based handlers
	// These all use the same underlying getResourceByID helper that had the doubled /api/v1 bug
	tests := []struct {
		name     string
		handler  func(context.Context, map[string]any) (any, error)
		paramKey string
		envVar   string
		fallback string // fallback ID to try if env var not set
	}{
		{
			name:     "GetEpic",
			handler:  h.GetEpic,
			paramKey: "epic_id",
			envVar:   "AHA_TEST_EPIC_ID",
			fallback: "", // no fallback - skip if not set
		},
		{
			name:     "GetGoal",
			handler:  h.GetGoal,
			paramKey: "goal_id",
			envVar:   "AHA_TEST_GOAL_ID",
			fallback: "",
		},
		{
			name:     "GetUser",
			handler:  h.GetUser,
			paramKey: "user_id",
			envVar:   "AHA_TEST_USER_ID",
			fallback: "",
		},
		{
			name:     "GetWorkflow",
			handler:  h.GetWorkflow,
			paramKey: "workflow_id",
			envVar:   "AHA_TEST_WORKFLOW_ID",
			fallback: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := os.Getenv(tt.envVar)
			if id == "" {
				id = tt.fallback
			}
			if id == "" {
				t.Skipf("%s not set, skipping %s test", tt.envVar, tt.name)
			}

			result, err := tt.handler(ctx, map[string]any{tt.paramKey: id})
			if err != nil {
				t.Fatalf("%s(%s) returned error: %v", tt.name, id, err)
			}

			// Result should be a map with status_code
			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("%s(%s) returned non-map: %T", tt.name, id, result)
			}

			statusCode, ok := resultMap["status_code"].(int)
			if !ok {
				t.Fatalf("%s(%s) missing status_code in response", tt.name, id)
			}

			// The key test: we should NOT get a 404 with HTML error page
			// (which would indicate doubled /api/v1 path)
			if statusCode == 404 {
				// Check if response contains HTML (indicates wrong URL path)
				for _, val := range resultMap {
					if s, ok := val.(string); ok && len(s) > 0 && s[0] == '<' {
						t.Errorf("%s(%s) returned 404 with HTML response - likely doubled /api/v1 in URL path", tt.name, id)
						t.Logf("Response contains HTML: %s...", s[:min(100, len(s))])
						return
					}
				}
				// 404 but not HTML - could be legitimately not found
				t.Logf("%s(%s) returned 404 (may be valid if ID doesn't exist)", tt.name, id)
			} else if statusCode != 200 {
				t.Logf("%s(%s) returned status %d", tt.name, id, statusCode)
			} else {
				t.Logf("%s(%s) returned 200 OK", tt.name, id)
			}
		})
	}
}

// TestGetFeature_Integration tests GetFeature which uses GraphQL (genqlient).
func TestGetFeature_Integration(t *testing.T) {
	subdomain := os.Getenv("AHA_SUBDOMAIN")
	apiKey := os.Getenv("AHA_API_KEY")

	if subdomain == "" || apiKey == "" {
		t.Skip("AHA_SUBDOMAIN and AHA_API_KEY must be set for integration tests")
	}

	featureRef := os.Getenv("AHA_TEST_FEATURE_REF")
	if featureRef == "" {
		t.Skip("AHA_TEST_FEATURE_REF not set")
	}

	h := mcp.NewToolHandlers(&mcp.Config{
		Subdomain: subdomain,
		APIKey:    apiKey,
	})

	result, err := h.GetFeature(context.Background(), map[string]any{
		"reference": featureRef,
	})
	if err != nil {
		t.Fatalf("GetFeature(%s) returned error: %v", featureRef, err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("GetFeature(%s) returned non-map: %T", featureRef, result)
	}

	// Check for error in response
	if errMsg, hasErr := resultMap["error"]; hasErr {
		t.Errorf("GetFeature(%s) returned error: %v", featureRef, errMsg)
		return
	}

	// Should have feature data
	if _, hasFeature := resultMap["feature"]; !hasFeature {
		t.Errorf("GetFeature(%s) missing 'feature' in response: %v", featureRef, resultMap)
	} else {
		t.Logf("GetFeature(%s) returned feature data successfully", featureRef)
	}
}

// TestGetIdea_Integration tests GetIdea which uses GraphQL (genqlient).
func TestGetIdea_Integration(t *testing.T) {
	subdomain := os.Getenv("AHA_SUBDOMAIN")
	apiKey := os.Getenv("AHA_API_KEY")

	if subdomain == "" || apiKey == "" {
		t.Skip("AHA_SUBDOMAIN and AHA_API_KEY must be set for integration tests")
	}

	ideaRef := os.Getenv("AHA_TEST_IDEA_REF")
	if ideaRef == "" {
		t.Skip("AHA_TEST_IDEA_REF not set")
	}

	h := mcp.NewToolHandlers(&mcp.Config{
		Subdomain: subdomain,
		APIKey:    apiKey,
	})

	result, err := h.GetIdea(context.Background(), map[string]any{
		"reference": ideaRef,
	})
	if err != nil {
		t.Fatalf("GetIdea(%s) returned error: %v", ideaRef, err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("GetIdea(%s) returned non-map: %T", ideaRef, result)
	}

	if errMsg, hasErr := resultMap["error"]; hasErr {
		t.Errorf("GetIdea(%s) returned error: %v", ideaRef, errMsg)
		return
	}

	if _, hasIdea := resultMap["idea"]; !hasIdea {
		t.Errorf("GetIdea(%s) missing 'idea' in response: %v", ideaRef, resultMap)
	} else {
		t.Logf("GetIdea(%s) returned idea data successfully", ideaRef)
	}
}
