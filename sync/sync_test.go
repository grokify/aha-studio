package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	aha "github.com/grokify/aha-go"
)

func TestCustomFieldsToMaps(t *testing.T) {
	tests := []struct {
		name string
		in   []aha.CustomField
		want int
	}{
		{name: "nil", in: nil, want: 0},
		{name: "empty", in: []aha.CustomField{}, want: 0},
		{
			name: "two fields",
			in: []aha.CustomField{
				{Key: "initiative_tags", Name: "Tags", Value: []byte(`["Platform_Initiative"]`), Type: "array"},
				{Key: "priority", Name: "Priority", Value: []byte(`"high"`), Type: "string"},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := customFieldsToMaps(tt.in)
			if len(got) != tt.want {
				t.Fatalf("len() = %d, want %d", len(got), tt.want)
			}
			if tt.want == 0 {
				if got != nil {
					t.Errorf("got = %+v, want nil", got)
				}
				return
			}
			if got[0]["key"] != tt.in[0].Key || got[0]["name"] != tt.in[0].Name || got[0]["type"] != tt.in[0].Type {
				t.Errorf("got[0] = %+v, want key/name/type from %+v", got[0], tt.in[0])
			}
		})
	}
}

func TestFeatureDetailToMap(t *testing.T) {
	release := &aha.Release{ID: "rel-1", Name: "Q1 2026"}
	assignee := &aha.User{FirstName: "Ada", LastName: "Lovelace"}
	status := &aha.WorkflowStatus{Name: "In Progress"}
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	f := &aha.Feature{
		ID:             "feat-1",
		ReferenceNum:   "PROJ-1",
		Name:           "Test Feature",
		Description:    "A description",
		URL:            "https://test.aha.io/features/PROJ-1",
		CreatedAt:      created,
		WorkflowStatus: status,
		Release:        release,
		AssignedTo:     assignee,
		Tags:           []string{"tag-a", "tag-b"},
		CustomFields: []aha.CustomField{
			{Key: "initiative_tags", Name: "Tags", Value: []byte(`["Platform_Initiative"]`), Type: "array"},
		},
	}

	got := featureDetailToMap(f)

	wantFields := map[string]any{
		"id":            "feat-1",
		"reference_num": "PROJ-1",
		"name":          "Test Feature",
		"description":   "A description",
		"status":        "In Progress",
		"release":       "Q1 2026",
		"release_id":    "rel-1",
		"assigned_to":   "Ada Lovelace",
	}
	for k, want := range wantFields {
		if got[k] != want {
			t.Errorf("[%s] = %v, want %v", k, got[k], want)
		}
	}

	cf, ok := got["custom_fields"].([]map[string]any)
	if !ok || len(cf) != 1 {
		t.Fatalf("custom_fields = %+v, want 1 entry", got["custom_fields"])
	}
	if cf[0]["key"] != "initiative_tags" {
		t.Errorf("custom_fields[0][key] = %v, want initiative_tags", cf[0]["key"])
	}
}

func TestInitiativeDetailToMap(t *testing.T) {
	status := &aha.WorkflowStatus{Name: "Active"}
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	i := &aha.Initiative{
		ID:             "init-1",
		ReferenceNum:   "PROJ-S-1",
		Name:           "Test Initiative",
		Description:    "An initiative",
		Value:          10,
		Effort:         5,
		Progress:       0.5,
		URL:            "https://test.aha.io/initiatives/PROJ-S-1",
		CreatedAt:      created,
		WorkflowStatus: status,
		CustomFields: []aha.CustomField{
			{Key: "initiative_tags", Name: "Tags", Value: []byte(`["Platform_Initiative"]`), Type: "array"},
		},
	}

	got := initiativeDetailToMap(i)

	wantFields := map[string]any{
		"id":            "init-1",
		"reference_num": "PROJ-S-1",
		"name":          "Test Initiative",
		"description":   "An initiative",
		"status":        "Active",
		"value":         float64(10),
		"effort":        float64(5),
		"progress":      0.5,
	}
	for k, want := range wantFields {
		if got[k] != want {
			t.Errorf("[%s] = %v, want %v", k, got[k], want)
		}
	}

	cf, ok := got["custom_fields"].([]map[string]any)
	if !ok || len(cf) != 1 {
		t.Fatalf("custom_fields = %+v, want 1 entry", got["custom_fields"])
	}
}

// TestSyncer_SyncInitiatives_Detailed exercises the full detailed-sync path
// against a fake Aha REST server: list-product-initiatives, then a
// per-initiative GetInitiative call merging custom fields, persisted into a
// real temp SQLite DB.
//
//nolint:dupl // Mirrors TestSyncer_SyncFeatures_Detailed but exercises a distinct entity/API shape
func TestSyncer_SyncInitiatives_Detailed(t *testing.T) {
	var listCalls, getCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/products/PROJ/initiatives":
			listCalls++
			_, _ = w.Write([]byte(`{
				"initiatives": [
					{"id": "init-1", "reference_num": "PROJ-S-1", "name": "Initiative One", "created_at": "2026-01-01T00:00:00Z"}
				],
				"pagination": {"total_records": 1, "total_pages": 1, "current_page": 1}
			}`))
		case "/initiatives/init-1":
			getCalls++
			_, _ = w.Write([]byte(`{
				"initiative": {
					"id": "init-1",
					"reference_num": "PROJ-S-1",
					"name": "Initiative One",
					"created_at": "2026-01-01T00:00:00Z",
					"custom_fields": [
						{"key": "initiative_tags", "name": "Tags", "type": "array", "value": ["Platform_Initiative"]}
					]
				}
			}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := aha.NewClient(
		aha.WithSubdomain("test"),
		aha.WithAPIKey("test-key"),
		aha.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	syncer := NewSyncer(db, client)
	results, err := syncer.SyncAll(context.Background(), SyncOptions{
		Product:  "PROJ",
		Entities: []string{"initiatives"},
		Detailed: true,
	})
	if err != nil {
		t.Fatalf("SyncAll() error = %v", err)
	}
	if len(results) != 1 || results[0].Error != nil || results[0].RecordCount != 1 {
		t.Fatalf("SyncAll() results = %+v", results)
	}

	if listCalls != 1 {
		t.Errorf("listCalls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Errorf("getCalls = %d, want 1 (detailed mode should fetch each record)", getCalls)
	}

	var rawData string
	if err := db.db.QueryRow(`SELECT data FROM initiatives WHERE id = ?`, "init-1").Scan(&rawData); err != nil {
		t.Fatalf("querying stored initiative: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(rawData), &stored); err != nil {
		t.Fatalf("unmarshaling stored data: %v", err)
	}
	if _, ok := stored["custom_fields"]; !ok {
		t.Errorf("stored initiative data missing custom_fields: %+v", stored)
	}
}

// TestSyncer_SyncFeatures_Detailed exercises the REST detailed-sync path
// (no GraphQL client configured) for features.
//
//nolint:dupl // Mirrors TestSyncer_SyncInitiatives_Detailed but exercises a distinct entity/API shape
func TestSyncer_SyncFeatures_Detailed(t *testing.T) {
	var listCalls, getCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/features":
			listCalls++
			_, _ = w.Write([]byte(`{
				"features": [
					{"id": "feat-1", "reference_num": "PROJ-1", "name": "Feature One", "created_at": "2026-01-01T00:00:00Z"}
				],
				"pagination": {"total_records": 1, "total_pages": 1, "current_page": 1}
			}`))
		case "/features/feat-1":
			getCalls++
			_, _ = w.Write([]byte(`{
				"feature": {
					"id": "feat-1",
					"reference_num": "PROJ-1",
					"name": "Feature One",
					"created_at": "2026-01-01T00:00:00Z",
					"custom_fields": [
						{"key": "priority", "name": "Priority", "type": "string", "value": "high"}
					]
				}
			}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := aha.NewClient(
		aha.WithSubdomain("test"),
		aha.WithAPIKey("test-key"),
		aha.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	syncer := NewSyncer(db, client) // no GraphQL client => REST path
	results, err := syncer.SyncAll(context.Background(), SyncOptions{
		Product:  "PROJ",
		Entities: []string{"features"},
		Detailed: true,
	})
	if err != nil {
		t.Fatalf("SyncAll() error = %v", err)
	}
	if len(results) != 1 || results[0].Error != nil || results[0].RecordCount != 1 {
		t.Fatalf("SyncAll() results = %+v", results)
	}

	if listCalls != 1 {
		t.Errorf("listCalls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Errorf("getCalls = %d, want 1 (detailed mode should fetch each record)", getCalls)
	}

	var rawData string
	if err := db.db.QueryRow(`SELECT data FROM features WHERE id = ?`, "feat-1").Scan(&rawData); err != nil {
		t.Fatalf("querying stored feature: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(rawData), &stored); err != nil {
		t.Fatalf("unmarshaling stored data: %v", err)
	}
	if _, ok := stored["custom_fields"]; !ok {
		t.Errorf("stored feature data missing custom_fields: %+v", stored)
	}
}
