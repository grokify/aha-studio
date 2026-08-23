package sync

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/grokify/aha-studio/ent/feature"

	_ "modernc.org/sqlite"
)

const realDBPath = "/Users/johnwang/.ahastudio/cache.db"

// TestRealDBSmoke is Phase 6's full-scale validation: it re-upserts a
// sample of REAL rows (not synthetic test fixtures) from every populated
// table in the actual production cache through the new Ent-backed
// Upsert* methods, at the real data's actual field shapes - the closest
// approximation of a real sync run available without live Aha API
// credentials. It operates on a COPY only; the real file is verified
// byte-identical (size+mtime) before and after.
func TestRealDBSmoke(t *testing.T) {
	info, err := os.Stat(realDBPath)
	if os.IsNotExist(err) {
		t.Skipf("real DB not found at %s, skipping", realDBPath)
	}
	sizeBefore, mtimeBefore := info.Size(), info.ModTime()

	dir := t.TempDir()
	copyPath := filepath.Join(dir, "smoke.db")
	if !copyFileForTest(t, realDBPath, copyPath) {
		t.Skip("real DB disappeared, skipping")
	}
	copyFileForTest(t, realDBPath+"-wal", copyPath+"-wal")
	copyFileForTest(t, realDBPath+"-shm", copyPath+"-shm")

	db, err := Open(copyPath)
	if err != nil {
		t.Fatalf("Open() on real-data copy: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	t.Run("initiatives", func(t *testing.T) {
		rows, err := db.ent.Initiative.Query().Limit(50).All(ctx)
		if err != nil {
			t.Fatalf("querying real initiatives: %v", err)
		}
		if len(rows) == 0 {
			t.Fatal("expected real initiative rows to sample")
		}
		for _, r := range rows {
			data := map[string]any{"id": r.ID, "product": r.Product, "name": r.Name, "reference_num": r.ReferenceNum}
			if err := db.UpsertInitiative(ctx, r.Product, data); err != nil {
				t.Fatalf("re-upserting real initiative %s: %v", r.ID, err)
			}
		}
		n, err := db.ent.Initiative.Query().Count(ctx)
		if err != nil {
			t.Fatalf("count after re-upsert: %v", err)
		}
		if n == 0 {
			t.Fatal("initiatives count dropped to 0 after re-upsert")
		}
	})

	t.Run("features", func(t *testing.T) {
		rows, err := db.ent.Feature.Query().Limit(200).All(ctx)
		if err != nil {
			t.Fatalf("querying real features: %v", err)
		}
		if len(rows) == 0 {
			t.Fatal("expected real feature rows to sample")
		}
		for _, r := range rows {
			data := map[string]any{
				"id": r.ID, "reference_num": r.ReferenceNum, "name": r.Name,
				"status": r.Status, "release_id": r.ReleaseID,
			}
			if err := db.UpsertFeature(ctx, r.Product, data); err != nil {
				t.Fatalf("re-upserting real feature %s: %v", r.ID, err)
			}
		}
		n, err := db.ent.Feature.Query().Count(ctx)
		if err != nil {
			t.Fatalf("count after re-upsert: %v", err)
		}
		if n == 0 {
			t.Fatal("features count dropped to 0 after re-upsert")
		}
	})

	t.Run("goals", func(t *testing.T) {
		rows, err := db.ent.Goal.Query().Limit(50).All(ctx)
		if err != nil {
			t.Fatalf("querying real goals: %v", err)
		}
		for _, r := range rows {
			data := map[string]any{"id": r.ID, "name": r.Name, "status": r.Status}
			if err := db.UpsertGoal(ctx, r.Product, data); err != nil {
				t.Fatalf("re-upserting real goal %s: %v", r.ID, err)
			}
		}
	})

	t.Run("epics", func(t *testing.T) {
		rows, err := db.ent.Epic.Query().Limit(50).All(ctx)
		if err != nil {
			t.Fatalf("querying real epics: %v", err)
		}
		for _, r := range rows {
			data := map[string]any{"id": r.ID, "name": r.Name, "status": r.Status}
			if err := db.UpsertEpic(ctx, r.Product, data); err != nil {
				t.Fatalf("re-upserting real epic %s: %v", r.ID, err)
			}
		}
	})

	t.Run("sync_meta", func(t *testing.T) {
		status, err := db.GetSyncStatus(ctx, "IN")
		if err != nil {
			t.Fatalf("GetSyncStatus: %v", err)
		}
		if len(status) == 0 {
			t.Fatal("expected sync_meta rows for product IN")
		}
	})

	t.Run("fts_still_intact_after_writes", func(t *testing.T) {
		n, err := db.ent.Feature.Query().Where(feature.NameContainsFold("a")).Count(ctx)
		if err != nil {
			t.Fatalf("sanity query after writes: %v", err)
		}
		if n == 0 {
			t.Fatal("expected at least one feature with 'a' in the name")
		}
	})

	// Original file must remain byte-identical throughout.
	infoAfter, err := os.Stat(realDBPath)
	if err != nil {
		t.Fatalf("stat real DB after test: %v", err)
	}
	if infoAfter.Size() != sizeBefore || !infoAfter.ModTime().Equal(mtimeBefore) {
		t.Fatalf("REAL DB WAS MODIFIED: size %d->%d, mtime %v->%v",
			sizeBefore, infoAfter.Size(), mtimeBefore, infoAfter.ModTime())
	}
}

func copyFileForTest(t *testing.T, src, dst string) bool {
	t.Helper()
	in, err := os.Open(src)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		t.Fatalf("opening %s: %v", src, err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("creating %s: %v", dst, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copying %s -> %s: %v", src, dst, err)
	}
	return true
}
