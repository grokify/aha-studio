package sync

import (
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
	err = db.UpsertFeature("PROJ", data)
	if err != nil {
		t.Fatalf("UpsertFeature() error = %v", err)
	}

	// Update feature
	data["status"] = "Done"
	err = db.UpsertFeature("PROJ", data)
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

	err = db.UpsertIdea("PROJ", data)
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
	lastSync, err := db.GetLastSync("features", "PROJ")
	if err != nil {
		t.Fatalf("GetLastSync() error = %v", err)
	}
	if !lastSync.IsZero() {
		t.Errorf("expected zero time for new entity, got %v", lastSync)
	}

	// Set sync time
	now := time.Now().Truncate(time.Second)
	err = db.SetLastSync("features", "PROJ", now, 100)
	if err != nil {
		t.Fatalf("SetLastSync() error = %v", err)
	}

	// Get sync time
	lastSync, err = db.GetLastSync("features", "PROJ")
	if err != nil {
		t.Fatalf("GetLastSync() after set error = %v", err)
	}
	if lastSync.Truncate(time.Second) != now {
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
	_ = db.SetLastSync("features", "PROJ", now, 50)
	_ = db.SetLastSync("ideas", "PROJ", now, 25)

	// Get status
	status, err := db.GetSyncStatus("PROJ")
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
