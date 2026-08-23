package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	genql "github.com/Khan/genqlient/graphql"

	"github.com/grokify/aha-go/graphql/generated"
)

func TestCustomFieldableTypeFromString(t *testing.T) {
	tests := []struct {
		in      string
		want    generated.CustomFieldableTypeEnum
		wantErr bool
	}{
		{"feature", generated.CustomFieldableTypeEnumFeature, false},
		{"Feature", generated.CustomFieldableTypeEnumFeature, false},
		{"initiative", generated.CustomFieldableTypeEnumInitiative, false},
		{"release", generated.CustomFieldableTypeEnumRelease, false},
		{"idea", "", true},
		{"", "", true},
		{"bogus", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := customFieldableTypeFromString(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Errorf("customFieldableTypeFromString(%q) error = nil, want non-nil", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("customFieldableTypeFromString(%q) error = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("customFieldableTypeFromString(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

// TestHandler_SetCustomFieldValues_ParamValidation follows the same
// no-Init, param-validation-only convention as the other
// TestHandler_*_ParamValidation tests in handlers_test.go.
func TestHandler_SetCustomFieldValues_ParamValidation(t *testing.T) {
	h := NewToolHandlers(&Config{Subdomain: "test", APIKey: "test-key"})

	tests := []struct {
		name      string
		params    map[string]any
		wantError string
	}{
		{
			name:      "missing entity_type",
			params:    map[string]any{"record_id": "FEAT-123", "custom_fields": map[string]any{"priority": "High"}},
			wantError: "entity_type must be one of",
		},
		{
			name: "invalid entity_type",
			params: map[string]any{
				"entity_type": "epic", "record_id": "FEAT-123",
				"custom_fields": map[string]any{"priority": "High"},
			},
			wantError: "entity_type must be one of",
		},
		{
			name:      "missing record_id",
			params:    map[string]any{"entity_type": "feature", "custom_fields": map[string]any{"priority": "High"}},
			wantError: "record_id parameter is required",
		},
		{
			name:      "missing custom_fields",
			params:    map[string]any{"entity_type": "feature", "record_id": "FEAT-123"},
			wantError: "custom_fields parameter is required",
		},
		{
			name: "empty custom_fields",
			params: map[string]any{
				"entity_type": "feature", "record_id": "FEAT-123", "custom_fields": map[string]any{},
			},
			wantError: "custom_fields parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.SetCustomFieldValues(context.Background(), tt.params)
			if err == nil {
				t.Fatalf("SetCustomFieldValues() expected error containing %q, got nil", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("SetCustomFieldValues() error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestHandler_SetCustomFieldValues_Success bypasses Init (like the rest
// of this file) but sets h.graphqlClient directly to a fake, to also
// cover the success and payload-error-surfaced paths, not just param
// validation.
func TestHandler_SetCustomFieldValues_Success(t *testing.T) {
	h := NewToolHandlers(&Config{Subdomain: "test", APIKey: "test-key"})
	h.graphqlClient = fakeGraphQLClient(t, `{
		"data": {
			"setCustomFieldValues": {
				"customFieldValues": [{"id": "1", "key": "priority", "value": "High", "humanValue": "High"}],
				"errors": {"attributes": []}
			}
		}
	}`)

	got, err := h.SetCustomFieldValues(context.Background(), map[string]any{
		"entity_type":   "initiative",
		"record_id":     "PROJ-I-1",
		"custom_fields": map[string]any{"priority": "High"},
	})
	if err != nil {
		t.Fatalf("SetCustomFieldValues() error = %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("SetCustomFieldValues() returned %T, want map[string]any", got)
	}
	if m["record_id"] != "PROJ-I-1" {
		t.Errorf("record_id = %v", m["record_id"])
	}
}

func TestHandler_SetCustomFieldValues_PayloadErrorSurfaced(t *testing.T) {
	h := NewToolHandlers(&Config{Subdomain: "test", APIKey: "test-key"})
	h.graphqlClient = fakeGraphQLClient(t, `{
		"data": {
			"setCustomFieldValues": {
				"customFieldValues": [],
				"errors": {"attributes": [{"name": "priority", "fullMessages": ["is not a valid custom field key"]}]}
			}
		}
	}`)

	_, err := h.SetCustomFieldValues(context.Background(), map[string]any{
		"entity_type":   "feature",
		"record_id":     "FEAT-123",
		"custom_fields": map[string]any{"priority": "High"},
	})
	if err == nil {
		t.Fatal("SetCustomFieldValues() error = nil, want error surfaced from the mutation payload")
	}
}

func fakeGraphQLClient(t *testing.T, responseBody string) genql.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(srv.Close)
	return genql.NewClient(srv.URL, http.DefaultClient)
}
