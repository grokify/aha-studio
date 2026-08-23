//nolint:dupl // Test functions follow similar table-driven test patterns
package mcp

import (
	"context"
	"strings"
	"testing"

	aha "github.com/grokify/aha-go"
)

// TestHandler_Query_ParamValidation tests Query parameter validation.
func TestHandler_Query_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing query parameter",
			params:    map[string]any{},
			wantError: "query parameter is required",
		},
		{
			name:      "empty query parameter",
			params:    map[string]any{"query": ""},
			wantError: "query parameter is required",
		},
		{
			name:      "wrong type for query",
			params:    map[string]any{"query": 123},
			wantError: "query parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.Query(context.Background(), tt.params)
			if err == nil {
				t.Errorf("Query() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Query() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetFeature_ParamValidation tests GetFeature parameter validation.
func TestHandler_GetFeature_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing reference parameter",
			params:    map[string]any{},
			wantError: "reference parameter is required",
		},
		{
			name:      "empty reference parameter",
			params:    map[string]any{"reference": ""},
			wantError: "reference parameter is required",
		},
		{
			name:      "wrong type for reference",
			params:    map[string]any{"reference": 123},
			wantError: "reference parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetFeature(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetFeature() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetFeature() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetIdea_ParamValidation tests GetIdea parameter validation.
func TestHandler_GetIdea_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing reference parameter",
			params:    map[string]any{},
			wantError: "reference parameter is required",
		},
		{
			name:      "empty reference parameter",
			params:    map[string]any{"reference": ""},
			wantError: "reference parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetIdea(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetIdea() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetIdea() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetRelease_ParamValidation tests GetRelease parameter validation.
func TestHandler_GetRelease_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing reference parameter",
			params:    map[string]any{},
			wantError: "reference parameter is required",
		},
		{
			name:      "empty reference parameter",
			params:    map[string]any{"reference": ""},
			wantError: "reference parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetRelease(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetRelease() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetRelease() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetInitiative_ParamValidation tests GetInitiative parameter validation.
func TestHandler_GetInitiative_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing reference parameter",
			params:    map[string]any{},
			wantError: "reference parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetInitiative(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetInitiative() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetInitiative() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetEpic_ParamValidation tests GetEpic parameter validation.
func TestHandler_GetEpic_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing epic_id parameter",
			params:    map[string]any{},
			wantError: "epic_id parameter is required",
		},
		{
			name:      "empty epic_id parameter",
			params:    map[string]any{"epic_id": ""},
			wantError: "epic_id parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetEpic(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetEpic() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetEpic() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetGoal_ParamValidation tests GetGoal parameter validation.
func TestHandler_GetGoal_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing goal_id parameter",
			params:    map[string]any{},
			wantError: "goal_id parameter is required",
		},
		{
			name:      "empty goal_id parameter",
			params:    map[string]any{"goal_id": ""},
			wantError: "goal_id parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetGoal(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetGoal() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetGoal() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_CreateFeature_ParamValidation tests CreateFeature parameter validation.
func TestHandler_CreateFeature_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing release_id parameter",
			params:    map[string]any{"name": "Test Feature"},
			wantError: "release_id parameter is required",
		},
		{
			name:      "missing name parameter",
			params:    map[string]any{"release_id": "REL-1"},
			wantError: "name parameter is required",
		},
		{
			name:      "empty name parameter",
			params:    map[string]any{"release_id": "REL-1", "name": ""},
			wantError: "name parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.CreateFeature(context.Background(), tt.params)
			if err == nil {
				t.Errorf("CreateFeature() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("CreateFeature() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_ChangeFeatureStatus_ParamValidation tests ChangeFeatureStatus parameter validation.
func TestHandler_ChangeFeatureStatus_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing feature_id parameter",
			params:    map[string]any{"status": "In Progress"},
			wantError: "feature_id parameter is required",
		},
		{
			name:      "missing status parameter",
			params:    map[string]any{"feature_id": "FEAT-123"},
			wantError: "status parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.ChangeFeatureStatus(context.Background(), tt.params)
			if err == nil {
				t.Errorf("ChangeFeatureStatus() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("ChangeFeatureStatus() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_AssignFeatureRelease_ParamValidation tests AssignFeatureRelease parameter validation.
func TestHandler_AssignFeatureRelease_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing feature_id parameter",
			params:    map[string]any{"release_id": "REL-1"},
			wantError: "feature_id parameter is required",
		},
		{
			name:      "missing release_id parameter",
			params:    map[string]any{"feature_id": "FEAT-123"},
			wantError: "release_id parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.AssignFeatureRelease(context.Background(), tt.params)
			if err == nil {
				t.Errorf("AssignFeatureRelease() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("AssignFeatureRelease() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_AddFeatureComment_ParamValidation tests AddFeatureComment parameter validation.
func TestHandler_AddFeatureComment_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing feature_id parameter",
			params:    map[string]any{"body": "Test comment"},
			wantError: "feature_id parameter is required",
		},
		{
			name:      "missing body parameter",
			params:    map[string]any{"feature_id": "FEAT-123"},
			wantError: "body parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.AddFeatureComment(context.Background(), tt.params)
			if err == nil {
				t.Errorf("AddFeatureComment() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("AddFeatureComment() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GraphQuery_ParamValidation tests GraphQuery parameter validation.
// Note: GraphQuery checks Neo4j config first, so this test verifies the
// handler returns an error when Neo4j is not configured.
func TestHandler_GraphQuery_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "neo4j not configured",
			params:    map[string]any{"cypher": "MATCH (n) RETURN n LIMIT 1"},
			wantError: "Neo4j not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GraphQuery(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GraphQuery() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GraphQuery() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_ListInitiativeFeatures_ParamValidation tests ListInitiativeFeatures parameter validation.
func TestHandler_ListInitiativeFeatures_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing initiative_id parameter",
			params:    map[string]any{},
			wantError: "initiative_id parameter is required",
		},
		{
			name:      "empty initiative_id parameter",
			params:    map[string]any{"initiative_id": ""},
			wantError: "initiative_id parameter is required",
		},
		{
			name:      "wrong type for initiative_id",
			params:    map[string]any{"initiative_id": 123},
			wantError: "initiative_id parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.ListInitiativeFeatures(context.Background(), tt.params)
			if err == nil {
				t.Errorf("ListInitiativeFeatures() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("ListInitiativeFeatures() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_GetInitiativeWithFeatures_ParamValidation tests GetInitiativeWithFeatures parameter validation.
func TestHandler_GetInitiativeWithFeatures_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing initiative_id parameter",
			params:    map[string]any{},
			wantError: "initiative_id parameter is required",
		},
		{
			name:      "empty initiative_id parameter",
			params:    map[string]any{"initiative_id": ""},
			wantError: "initiative_id parameter is required",
		},
		{
			name:      "wrong type for initiative_id",
			params:    map[string]any{"initiative_id": 123},
			wantError: "initiative_id parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.GetInitiativeWithFeatures(context.Background(), tt.params)
			if err == nil {
				t.Errorf("GetInitiativeWithFeatures() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("GetInitiativeWithFeatures() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_ListInitiativesByTag_ParamValidation tests ListInitiativesByTag parameter validation.
func TestHandler_ListInitiativesByTag_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing product_id parameter",
			params:    map[string]any{"tag": "Platform_Initiative"},
			wantError: "product_id parameter is required",
		},
		{
			name:      "empty product_id parameter",
			params:    map[string]any{"product_id": "", "tag": "Platform_Initiative"},
			wantError: "product_id parameter is required",
		},
		{
			name:      "missing tag parameter",
			params:    map[string]any{"product_id": "PROD"},
			wantError: "tag parameter is required",
		},
		{
			name:      "empty tag parameter",
			params:    map[string]any{"product_id": "PROD", "tag": ""},
			wantError: "tag parameter is required",
		},
		{
			name:      "wrong type for product_id",
			params:    map[string]any{"product_id": 123, "tag": "Platform_Initiative"},
			wantError: "product_id parameter is required",
		},
		{
			name:      "wrong type for tag",
			params:    map[string]any{"product_id": "PROD", "tag": 123},
			wantError: "tag parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.ListInitiativesByTag(context.Background(), tt.params)
			if err == nil {
				t.Errorf("ListInitiativesByTag() expected error containing %q, got nil", tt.wantError)
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("ListInitiativesByTag() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestInitiativeHasTag tests the initiativeHasTag helper function.
func TestInitiativeHasTag(t *testing.T) {
	tests := []struct {
		name string
		init *aha.Initiative
		tag  string
		want bool
	}{
		{
			name: "no custom fields",
			init: &aha.Initiative{},
			tag:  "Platform_Initiative",
			want: false,
		},
		{
			name: "matching tag",
			init: &aha.Initiative{
				CustomFields: []aha.CustomField{
					{Key: "initiative_tags", Name: "Tags", Value: []byte(`["Platform_Initiative","Other"]`), Type: "array"},
				},
			},
			tag:  "Platform_Initiative",
			want: true,
		},
		{
			name: "case-insensitive match",
			init: &aha.Initiative{
				CustomFields: []aha.CustomField{
					{Key: "initiative_tags", Name: "Tags", Value: []byte(`["platform_initiative"]`), Type: "array"},
				},
			},
			tag:  "Platform_Initiative",
			want: true,
		},
		{
			name: "no matching tag",
			init: &aha.Initiative{
				CustomFields: []aha.CustomField{
					{Key: "initiative_tags", Name: "Tags", Value: []byte(`["Other_Tag"]`), Type: "array"},
				},
			},
			tag:  "Platform_Initiative",
			want: false,
		},
		{
			name: "non-tag custom field ignored",
			init: &aha.Initiative{
				CustomFields: []aha.CustomField{
					{Key: "priority", Name: "Priority", Value: []byte(`"high"`), Type: "string"},
				},
			},
			tag:  "high",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := initiativeHasTag(tt.init, tt.tag)
			if got != tt.want {
				t.Errorf("initiativeHasTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHandler_SyncData_NotConfigured tests SyncData when sync is not configured.
func TestHandler_SyncData_NotConfigured(t *testing.T) {
	h := NewToolHandlers(&Config{
		Subdomain: "test",
		APIKey:    "test-key",
	})

	_, err := h.SyncData(context.Background(), map[string]any{"product": "PROD"})
	if err == nil {
		t.Fatal("SyncData() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sync not configured") {
		t.Errorf("SyncData() error = %q, want error containing %q", err.Error(), "sync not configured")
	}
}

// TestNewToolHandlers tests NewToolHandlers creation.
func TestNewToolHandlers(t *testing.T) {
	cfg := &Config{
		Subdomain:      "test",
		APIKey:         "test-key",
		DefaultProduct: "PROD",
	}

	h := NewToolHandlers(cfg)
	if h == nil {
		t.Fatal("NewToolHandlers() returned nil")
	}
	if h.config != cfg {
		t.Error("NewToolHandlers() config not set correctly")
	}
}
