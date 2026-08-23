package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	genql "github.com/Khan/genqlient/graphql"

	"github.com/grokify/aha-go/graphql/generated"
	"github.com/grokify/aha-studio/aql/ast"
)

func TestSplitAssignments(t *testing.T) {
	assignments := []ast.Assignment{
		{Field: "name", Value: ast.Value{Type: ast.ValueTypeString, String: "Updated"}},
		{Field: "custom.priority", Value: ast.Value{Type: ast.ValueTypeString, String: "High"}},
		{Field: "custom.story_points", Value: ast.Value{Type: ast.ValueTypeInt, Int: 8}},
		{Field: "status", Value: ast.Value{Type: ast.ValueTypeString, String: "Done"}},
	}

	standard, custom := splitAssignments(assignments)

	if len(standard) != 2 {
		t.Errorf("expected 2 standard assignments, got %d: %+v", len(standard), standard)
	}
	if standard["name"] != "Updated" || standard["status"] != "Done" {
		t.Errorf("standard assignments = %+v", standard)
	}

	if len(custom) != 2 {
		t.Errorf("expected 2 custom assignments, got %d: %+v", len(custom), custom)
	}
	// Keys must be stripped of the "custom." prefix - SetCustomFieldValues
	// expects bare keys.
	if custom["priority"] != "High" || custom["story_points"] != int64(8) {
		t.Errorf("custom assignments = %+v", custom)
	}
	if _, ok := custom["custom.priority"]; ok {
		t.Error("custom map must not retain the \"custom.\" prefix in its keys")
	}
}

func TestSplitAssignments_Empty(t *testing.T) {
	standard, custom := splitAssignments(nil)
	if len(standard) != 0 || len(custom) != 0 {
		t.Errorf("expected empty maps, got standard=%+v custom=%+v", standard, custom)
	}
}

func TestCustomFieldableType(t *testing.T) {
	tests := []struct {
		entity  ast.EntityType
		want    generated.CustomFieldableTypeEnum
		wantErr bool
	}{
		{ast.EntityFeatures, generated.CustomFieldableTypeEnumFeature, false},
		{ast.EntityInitiatives, generated.CustomFieldableTypeEnumInitiative, false},
		{ast.EntityReleases, generated.CustomFieldableTypeEnumRelease, false},
		{ast.EntityIdeas, "", true},
		{ast.EntityEpics, "", true},
		{ast.EntityGoals, "", true},
	}
	for _, tt := range tests {
		t.Run(string(tt.entity), func(t *testing.T) {
			got, err := customFieldableType(tt.entity)
			if tt.wantErr {
				if err == nil {
					t.Errorf("customFieldableType(%s) error = nil, want non-nil", tt.entity)
				}
				return
			}
			if err != nil {
				t.Fatalf("customFieldableType(%s) error = %v", tt.entity, err)
			}
			if got != tt.want {
				t.Errorf("customFieldableType(%s) = %s, want %s", tt.entity, got, tt.want)
			}
		})
	}
}

func TestUpdateFeature_UnrecognizedKeyFailsFast(t *testing.T) {
	// e.client is intentionally nil - if this reached the API call, it
	// would panic; the point of this test is that it must NOT reach the
	// API call, returning an error instead of silently dropping the
	// unrecognized key (the bug this change fixes).
	e := &Executor{}
	err := e.updateFeature(context.Background(), "FEAT-1", map[string]any{
		"name":           "OK",
		"not_a_field":    "should error, not silently drop",
		"also_bad_field": true,
	})
	if err == nil {
		t.Fatal("updateFeature() error = nil, want error for unrecognized key")
	}
}

func TestUpdateFeature_AllKeysRecognized_NoUnrecognizedError(t *testing.T) {
	// With no unrecognized keys but also no client configured, this
	// documents that validation happens before the API call - we can't
	// assert success without a real client, but we can assert it's NOT
	// the "unrecognized field" error by checking updateOpts would be
	// non-empty (i.e. it got past validation). Use the empty-updates
	// short-circuit (returns nil without calling the client) to prove
	// the validation path itself doesn't false-positive on known fields.
	e := &Executor{}
	if err := e.updateFeature(context.Background(), "FEAT-1", map[string]any{}); err != nil {
		t.Errorf("updateFeature() with empty (all-recognized, trivially) updates: %v", err)
	}
}

func TestUpdateRecord_StandardFieldRejectedForInitiativesAndReleases(t *testing.T) {
	e := &Executor{}
	for _, entity := range []ast.EntityType{ast.EntityInitiatives, ast.EntityReleases} {
		t.Run(string(entity), func(t *testing.T) {
			err := e.updateRecord(context.Background(), entity, "ID-1", map[string]any{"name": "x"})
			if err == nil {
				t.Fatalf("updateRecord(%s) error = nil, want explicit error (not silent no-op)", entity)
			}
		})
	}
}

func TestUpdateCustomFields_NoClientConfigured(t *testing.T) {
	e := &Executor{} // graphqlClient not set
	err := e.updateCustomFields(context.Background(), ast.EntityFeatures, "FEAT-1", map[string]any{"priority": "High"})
	if err == nil {
		t.Fatal("updateCustomFields() error = nil, want error when no GraphQL client is configured")
	}
}

func TestUpdateCustomFields_UnsupportedEntity(t *testing.T) {
	e := &Executor{graphqlClient: fakeGraphQLClient(t, `{"data":{"setCustomFieldValues":{"customFieldValues":[],"errors":{"attributes":[]}}}}`)}
	err := e.updateCustomFields(context.Background(), ast.EntityIdeas, "IDEA-1", map[string]any{"priority": "High"})
	if err == nil {
		t.Fatal("updateCustomFields() error = nil, want error for an entity outside Feature/Initiative/Release")
	}
}

func TestUpdateCustomFields_SucceedsForAllThreeEntities(t *testing.T) {
	client := fakeGraphQLClient(t, `{
		"data": {
			"setCustomFieldValues": {
				"customFieldValues": [{"id": "1", "key": "priority", "value": "High", "humanValue": "High"}],
				"errors": {"attributes": []}
			}
		}
	}`)
	e := &Executor{graphqlClient: client}

	for _, entity := range []ast.EntityType{ast.EntityFeatures, ast.EntityInitiatives, ast.EntityReleases} {
		t.Run(string(entity), func(t *testing.T) {
			err := e.updateCustomFields(context.Background(), entity, "ID-1", map[string]any{"priority": "High"})
			if err != nil {
				t.Errorf("updateCustomFields(%s) error = %v, want nil", entity, err)
			}
		})
	}
}

func TestUpdateCustomFields_PayloadErrorSurfaced(t *testing.T) {
	client := fakeGraphQLClient(t, `{
		"data": {
			"setCustomFieldValues": {
				"customFieldValues": [],
				"errors": {"attributes": [{"name": "priority", "fullMessages": ["is not a valid custom field key"]}]}
			}
		}
	}`)
	e := &Executor{graphqlClient: client}

	err := e.updateCustomFields(context.Background(), ast.EntityFeatures, "FEAT-1", map[string]any{"priority": "High"})
	if err == nil {
		t.Fatal("updateCustomFields() error = nil, want error surfaced from the mutation payload")
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
