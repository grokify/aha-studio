package sync

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	aha "github.com/grokify/aha-go"
)

func TestUpsertIdeaUser(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	data := map[string]any{
		"id":                 "IU-1",
		"name":               "Ada Lovelace",
		"email":              "ada@example.com",
		"idea_organizations": []map[string]any{{"id": "ORG-1", "name": "Example Corp"}},
	}
	if err := db.UpsertIdeaUser(ctx, data); err != nil {
		t.Fatalf("UpsertIdeaUser() error = %v", err)
	}

	// Idempotent update, not a duplicate row.
	data["name"] = "Ada L."
	if err := db.UpsertIdeaUser(ctx, data); err != nil {
		t.Fatalf("UpsertIdeaUser() second call error = %v", err)
	}

	var count int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM idea_users WHERE id = ?`, "IU-1").Scan(&count); err != nil {
		t.Fatalf("querying idea_users: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (upsert should not duplicate)", count)
	}
}

func TestUpsertIdeaOrganization(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	data := map[string]any{
		"id":                 "ORG-1",
		"name":               "Example Corp",
		"reference_num":      "ACCOUNT-O-1",
		"email_domains":      "example.com,example.io",
		"revenue":            1500000.0,
		"endorsements_count": int64(4),
	}
	if err := db.UpsertIdeaOrganization(ctx, data); err != nil {
		t.Fatalf("UpsertIdeaOrganization() error = %v", err)
	}

	var name, domains string
	var revenue float64
	if err := db.db.QueryRow(`SELECT name, email_domains, revenue FROM idea_organizations WHERE id = ?`, "ORG-1").
		Scan(&name, &domains, &revenue); err != nil {
		t.Fatalf("querying idea_organizations: %v", err)
	}
	if name != "Example Corp" || domains != "example.com,example.io" || revenue != 1500000.0 {
		t.Errorf("got name=%q domains=%q revenue=%v, want Example Corp/example.com,example.io/1500000", name, domains, revenue)
	}
}

func TestGetIdeaOrganizationsByDomain(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := db.UpsertIdeaOrganization(ctx, map[string]any{
		"id": "ORG-1", "name": "Example Corp", "email_domains": "example.com, example.io",
	}); err != nil {
		t.Fatalf("UpsertIdeaOrganization() error = %v", err)
	}
	if err := db.UpsertIdeaOrganization(ctx, map[string]any{
		"id": "ORG-2", "name": "No Domain Corp",
	}); err != nil {
		t.Fatalf("UpsertIdeaOrganization() error = %v", err)
	}

	byDomain, err := db.GetIdeaOrganizationsByDomain(ctx)
	if err != nil {
		t.Fatalf("GetIdeaOrganizationsByDomain() error = %v", err)
	}

	if len(byDomain) != 2 {
		t.Fatalf("len(byDomain) = %d, want 2 (both domains from the multi-domain org)", len(byDomain))
	}
	if byDomain["example.com"].Name != "Example Corp" {
		t.Errorf("byDomain[example.com] = %+v, want Example Corp", byDomain["example.com"])
	}
	if byDomain["example.io"].Name != "Example Corp" {
		t.Errorf("byDomain[example.io] = %+v, want Example Corp", byDomain["example.io"])
	}
	if _, ok := byDomain[""]; ok {
		t.Error("byDomain should not contain an empty-string key for the org with no email_domains")
	}
}

func TestSyncer_SyncIdeaOrganizations_Detailed(t *testing.T) {
	var listCalls, getCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/idea_organizations":
			listCalls++
			_, _ = w.Write([]byte(`{
				"idea_organizations": [
					{"id": "ORG-1", "name": "Example Corp", "created_at": "2026-01-01T00:00:00Z"}
				],
				"pagination": {"total_records": 1, "total_pages": 1, "current_page": 1}
			}`))
		case "/idea_organizations/ORG-1":
			getCalls++
			_, _ = w.Write([]byte(`{
				"idea_organization": {
					"id": "ORG-1", "name": "Example Corp", "reference_num": "ACCOUNT-O-1",
					"email_domains": "example.com", "revenue": 500000, "endorsements_count": 2
				}
			}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := aha.NewClient(aha.WithSubdomain("test"), aha.WithAPIKey("test-key"), aha.WithBaseURL(server.URL))
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
		Entities: []string{"idea_organizations"},
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
		t.Errorf("getCalls = %d, want 1 (Detailed=true should fetch each org)", getCalls)
	}

	var domains string
	if err := db.db.QueryRow(`SELECT email_domains FROM idea_organizations WHERE id = ?`, "ORG-1").Scan(&domains); err != nil {
		t.Fatalf("querying stored org: %v", err)
	}
	if domains != "example.com" {
		t.Errorf("email_domains = %q, want %q (only populated when Detailed=true)", domains, "example.com")
	}
}

func TestSyncer_SyncIdeaOrganizations_NotDetailed(t *testing.T) {
	var getCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/idea_organizations":
			_, _ = w.Write([]byte(`{
				"idea_organizations": [
					{"id": "ORG-1", "name": "Example Corp", "created_at": "2026-01-01T00:00:00Z"}
				],
				"pagination": {"total_records": 1, "total_pages": 1, "current_page": 1}
			}`))
		default:
			getCalls++
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := aha.NewClient(aha.WithSubdomain("test"), aha.WithAPIKey("test-key"), aha.WithBaseURL(server.URL))
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
	if _, err := syncer.SyncAll(context.Background(), SyncOptions{Entities: []string{"idea_organizations"}}); err != nil {
		t.Fatalf("SyncAll() error = %v", err)
	}

	if getCalls != 0 {
		t.Errorf("getCalls = %d, want 0 (Detailed=false should never fetch per-org detail)", getCalls)
	}

	var domains sql.NullString
	if err := db.db.QueryRow(`SELECT email_domains FROM idea_organizations WHERE id = ?`, "ORG-1").Scan(&domains); err != nil {
		t.Fatalf("querying stored org: %v", err)
	}
	if domains.Valid {
		t.Errorf("email_domains = %q, want NULL (not fetched without Detailed=true)", domains.String)
	}
}
