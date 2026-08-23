package sync

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify database was created
	if db.db == nil {
		t.Error("database connection is nil")
	}
}

func TestUpsertFeature(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	data := map[string]any{
		"id":            "FEAT-123",
		"reference_num": "PROJ-123",
		"name":          "Test Feature",
		"status":        "In Progress",
		"created_at":    time.Now(),
	}

	// Insert feature
	err = db.UpsertFeature(context.Background(), "PROJ", data)
	if err != nil {
		t.Fatalf("UpsertFeature() error = %v", err)
	}

	// Update feature
	data["status"] = "Done"
	err = db.UpsertFeature(context.Background(), "PROJ", data)
	if err != nil {
		t.Fatalf("UpsertFeature() second call error = %v", err)
	}
}

func TestUpsertIdea(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	data := map[string]any{
		"id":            "IDEA-123",
		"reference_num": "PROJ-I-123",
		"name":          "Test Idea",
		"votes":         int64(42),
		"created_at":    time.Now(),
	}

	err = db.UpsertIdea(context.Background(), "PROJ", data)
	if err != nil {
		t.Fatalf("UpsertIdea() error = %v", err)
	}
}

func TestSetGetLastSync(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Initially no sync time
	lastSync, err := db.GetLastSync(context.Background(), "features", "PROJ")
	if err != nil {
		t.Fatalf("GetLastSync() error = %v", err)
	}
	if !lastSync.IsZero() {
		t.Errorf("expected zero time for new entity, got %v", lastSync)
	}

	// Set sync time
	now := time.Now().Truncate(time.Second)
	err = db.SetLastSync(context.Background(), "features", "PROJ", now, 100)
	if err != nil {
		t.Fatalf("SetLastSync() error = %v", err)
	}

	// Get sync time
	lastSync, err = db.GetLastSync(context.Background(), "features", "PROJ")
	if err != nil {
		t.Fatalf("GetLastSync() after set error = %v", err)
	}
	if !lastSync.Truncate(time.Second).Equal(now) {
		t.Errorf("GetLastSync() = %v, want %v", lastSync.Truncate(time.Second), now)
	}
}

func TestGetSyncStatus(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Set some sync times
	now := time.Now()
	_ = db.SetLastSync(context.Background(), "features", "PROJ", now, 50)
	_ = db.SetLastSync(context.Background(), "ideas", "PROJ", now, 25)

	// Get status
	status, err := db.GetSyncStatus(context.Background(), "PROJ")
	if err != nil {
		t.Fatalf("GetSyncStatus() error = %v", err)
	}

	if len(status) != 2 {
		t.Errorf("expected 2 status entries, got %d", len(status))
	}

	if s, ok := status["features"]; ok {
		if s.RecordCount != 50 {
			t.Errorf("features record count = %d, want 50", s.RecordCount)
		}
	} else {
		t.Error("missing features status")
	}
}

// TestUpsertFeatureRoundTrip verifies the Ent-backed UpsertFeature actually
// writes the fields it's given (TestUpsertFeature above only asserts
// err == nil), including that a second upsert with a missing optional key
// (release_id) doesn't null out the first upsert's value - see
// mapStringPtr's SetNillableX skip-if-absent behavior in ent_coerce.go.
func TestUpsertFeatureRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	created := time.Now().Truncate(time.Second)

	data := map[string]any{
		"id":            "FEAT-RT-1",
		"reference_num": "PROJ-RT-1",
		"name":          "Round Trip Feature",
		"status":        "In Progress",
		"release_id":    "REL-1",
		"tags":          []string{"a", "b"},
		"created_at":    created,
	}
	if err := db.UpsertFeature(ctx, "PROJ", data); err != nil {
		t.Fatalf("UpsertFeature() error = %v", err)
	}

	got, err := db.ent.Feature.Get(ctx, "FEAT-RT-1")
	if err != nil {
		t.Fatalf("reading back feature: %v", err)
	}
	if got.Name != "Round Trip Feature" || got.Status != "In Progress" || got.ReleaseID != "REL-1" {
		t.Errorf("round-trip mismatch: name=%q status=%q release_id=%q", got.Name, got.Status, got.ReleaseID)
	}
	if got.Tags != "a,b" {
		t.Errorf("tags = %q, want %q", got.Tags, "a,b")
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, created)
	}

	// Second upsert omitting release_id: SetNillableReleaseID(nil) should
	// be a no-op, preserving the value from the first upsert rather than
	// nulling it out.
	data2 := map[string]any{
		"id":     "FEAT-RT-1",
		"name":   "Round Trip Feature",
		"status": "Done",
	}
	if err := db.UpsertFeature(ctx, "PROJ", data2); err != nil {
		t.Fatalf("UpsertFeature() second call error = %v", err)
	}

	got2, err := db.ent.Feature.Get(ctx, "FEAT-RT-1")
	if err != nil {
		t.Fatalf("reading back feature after second upsert: %v", err)
	}
	if got2.Status != "Done" {
		t.Errorf("status = %q, want %q", got2.Status, "Done")
	}
	if got2.ReleaseID != "REL-1" {
		t.Errorf("release_id after partial upsert = %q, want preserved %q", got2.ReleaseID, "REL-1")
	}
}

// TestUpsertInitiativeRoundTrip exercises the Nillable numeric fields
// (value/effort/progress), which TestUpsertFeatureRoundTrip doesn't cover.
func TestUpsertInitiativeRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	data := map[string]any{
		"id":       "INIT-RT-1",
		"name":     "Round Trip Initiative",
		"value":    8.5,
		"effort":   3.0,
		"progress": 42.0,
	}
	if err := db.UpsertInitiative(ctx, "PROJ", data); err != nil {
		t.Fatalf("UpsertInitiative() error = %v", err)
	}

	got, err := db.ent.Initiative.Get(ctx, "INIT-RT-1")
	if err != nil {
		t.Fatalf("reading back initiative: %v", err)
	}
	if got.Value == nil || *got.Value != 8.5 {
		t.Errorf("value = %v, want 8.5", got.Value)
	}
	if got.Effort == nil || *got.Effort != 3.0 {
		t.Errorf("effort = %v, want 3.0", got.Effort)
	}
	if got.Progress == nil || *got.Progress != 42.0 {
		t.Errorf("progress = %v, want 42.0", got.Progress)
	}
}
