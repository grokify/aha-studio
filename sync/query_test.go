package sync

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/aql/parser"
	"github.com/grokify/aha-studio/aql/validator"
	"github.com/grokify/aha-studio/planner"
)

// runAQL parses, validates, plans, and executes an AQL query against db,
// mirroring what cmd/aha-studio's executeQuery / httpserver's
// executeOfflineQuery do in production.
func runAQL(t *testing.T, ctx context.Context, db *DB, aql string) []map[string]any {
	t.Helper()
	q, err := parser.New(aql).Parse()
	if err != nil {
		t.Fatalf("parsing %q: %v", aql, err)
	}
	if err := validator.New().Validate(q); err != nil {
		t.Fatalf("validating %q: %v", aql, err)
	}
	plan := planner.New().Plan(q)
	res, err := db.QueryOffline(ctx, plan)
	if err != nil {
		t.Fatalf("QueryOffline(%q): %v", aql, err)
	}
	var out []map[string]any
	for _, rec := range res.Records {
		out = append(out, map[string]any(rec))
	}
	return out
}

func TestQueryOffline_PlainFieldFilters(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	// Note: "status"/"name" have an APIParam mapping in schema/entities.go
	// (workflow_status / q), so the planner routes filters on them into
	// plan.APIParams rather than plan.ClientFilters - only meaningful for
	// live-API queries, not QueryOffline. Use "reference_num" (no
	// APIParam) so these filters actually exercise buildSQL/filterToSQL's
	// WHERE-clause path, which is what this test is for.
	seed := []map[string]any{
		{"id": "INIT-1", "reference_num": "ALPHA-1"},
		{"id": "INIT-2", "reference_num": "BETA-1"},
		{"id": "INIT-3", "reference_num": "ALPHA-PROJECT-1"},
	}
	for _, d := range seed {
		if err := db.UpsertInitiative(ctx, "PROJ", d); err != nil {
			t.Fatalf("seeding %v: %v", d["id"], err)
		}
	}
	db.SetProduct("PROJ")

	t.Run("EQ", func(t *testing.T) {
		got := runAQL(t, ctx, db, `FROM initiatives WHERE reference_num = "BETA-1"`)
		if len(got) != 1 || got[0]["id"] != "INIT-2" {
			t.Errorf("EQ filter: got %+v", got)
		}
	})

	t.Run("IN", func(t *testing.T) {
		got := runAQL(t, ctx, db, `FROM initiatives WHERE id IN ("INIT-1", "INIT-2")`)
		if len(got) != 2 {
			t.Errorf("IN filter: expected 2 rows, got %d: %+v", len(got), got)
		}
	})

	t.Run("CONTAINS", func(t *testing.T) {
		got := runAQL(t, ctx, db, `FROM initiatives WHERE reference_num CONTAINS "PROJECT"`)
		if len(got) != 1 || got[0]["id"] != "INIT-3" {
			t.Errorf("CONTAINS filter: got %+v", got)
		}
	})
}

// TestQueryOffline_ReleaseReferenceNumFilter is the actual downstream ask
// this migration exists to unblock: filtering features by a SET of
// release *reference numbers* (e.g. "REL-1"), not Aha's internal release
// ID - previously impossible since sync.go never captured
// Release.ReferenceNum into the local features table.
func TestQueryOffline_ReleaseReferenceNumFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	seed := []map[string]any{
		{"id": "FEAT-1", "release_id": "internal-1", "release_reference_num": "REL-1"},
		{"id": "FEAT-2", "release_id": "internal-2", "release_reference_num": "REL-2"},
		{"id": "FEAT-3", "release_id": "internal-3", "release_reference_num": "REL-3"},
	}
	for _, d := range seed {
		if err := db.UpsertFeature(ctx, "PROJ", d); err != nil {
			t.Fatalf("seeding %v: %v", d["id"], err)
		}
	}
	db.SetProduct("PROJ")

	got := runAQL(t, ctx, db, `FROM features WHERE release_reference_num IN ("REL-1", "REL-3")`)
	gotIDs := map[string]bool{}
	for _, r := range got {
		gotIDs[r["id"].(string)] = true
	}
	if len(got) != 2 || !gotIDs["FEAT-1"] || !gotIDs["FEAT-3"] {
		t.Errorf("release_reference_num IN filter: got %+v", got)
	}
}

func TestQueryOffline_CustomFieldFilter(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Mirrors the real shape sync.go's customFieldsToMaps produces: an
	// array of {key, name, type, value} objects under data.custom_fields.
	highPriority := map[string]any{
		"id":   "INIT-HIGH",
		"name": "High priority initiative",
		"custom_fields": []map[string]any{
			{"key": "priority", "name": "Priority", "type": "string", "value": "High"},
		},
	}
	lowPriority := map[string]any{
		"id":   "INIT-LOW",
		"name": "Low priority initiative",
		"custom_fields": []map[string]any{
			{"key": "priority", "name": "Priority", "type": "string", "value": "Low"},
		},
	}
	noCustomFields := map[string]any{
		"id":   "INIT-NONE",
		"name": "No custom fields",
	}
	for _, d := range []map[string]any{highPriority, lowPriority, noCustomFields} {
		if err := db.UpsertInitiative(ctx, "PROJ", d); err != nil {
			t.Fatalf("seeding %v: %v", d["id"], err)
		}
	}
	db.SetProduct("PROJ")

	got := runAQL(t, ctx, db, `FROM initiatives WHERE custom.priority = "High"`)
	if len(got) != 1 || got[0]["id"] != "INIT-HIGH" {
		t.Fatalf("custom.priority = High: got %+v", got)
	}

	gotNone := runAQL(t, ctx, db, `FROM initiatives WHERE custom.priority = "Nonexistent"`)
	if len(gotNone) != 0 {
		t.Fatalf("custom.priority = Nonexistent: expected 0 rows, got %+v", gotNone)
	}
}

// TestBuildSQL_RejectsUnknownField confirms the schema.GetEntity allow-list
// added to buildSQL/filterToSQL rejects a field name that isn't part of
// the entity's declared schema, rather than interpolating it into SQL -
// closing the gap the design review flagged (SELECT/WHERE/ORDER BY all
// used to accept any string field.Field/OrderBy.Field/SelectFields entry
// unchecked at the SQL-build layer).
func TestBuildSQL_RejectsUnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	malicious := "id; DROP TABLE initiatives; --"

	t.Run("WHERE", func(t *testing.T) {
		plan := &planner.Plan{
			Entity: ast.EntityInitiatives,
			ClientFilters: []planner.Filter{
				{Field: malicious, Op: ast.OpEQ, Value: &ast.Value{Type: ast.ValueTypeString, String: "x"}},
			},
		}
		if _, _, err := db.buildSQL(plan); err == nil {
			t.Error("expected buildSQL to reject an unknown WHERE field, got nil error")
		}
	})

	t.Run("ORDER BY", func(t *testing.T) {
		plan := &planner.Plan{
			Entity:  ast.EntityInitiatives,
			OrderBy: &planner.OrderBy{Field: malicious},
		}
		if _, _, err := db.buildSQL(plan); err == nil {
			t.Error("expected buildSQL to reject an unknown ORDER BY field, got nil error")
		}
	})

	t.Run("SELECT", func(t *testing.T) {
		plan := &planner.Plan{
			Entity:       ast.EntityInitiatives,
			SelectFields: []string{malicious},
		}
		if _, _, err := db.buildSQL(plan); err == nil {
			t.Error("expected buildSQL to reject an unknown SELECT field, got nil error")
		}
	})
}
