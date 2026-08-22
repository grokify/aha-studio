package sync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	aha "github.com/grokify/aha-go"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestUpsertIdeaEndorsement(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	data := map[string]any{
		"id":                     "END-1",
		"idea_id":                "IDEA-1",
		"weight":                 int64(1),
		"portal_user_id":         "U-1",
		"portal_user_name":       "Ada Lovelace",
		"portal_user_email":      "ada@example.com",
		"portal_user_created_at": time.Now(),
		"idea_user_id":           "U-2",
		"idea_user_name":         "Ada Lovelace",
		"idea_user_email":        "ada@example.com",
		"created_at":             time.Now(),
	}

	if err := db.UpsertIdeaEndorsement(ctx, "PROJ", data); err != nil {
		t.Fatalf("UpsertIdeaEndorsement() error = %v", err)
	}

	// Upsert again (idempotent update, not a duplicate row).
	data["portal_user_name"] = "Ada L."
	if err := db.UpsertIdeaEndorsement(ctx, "PROJ", data); err != nil {
		t.Fatalf("UpsertIdeaEndorsement() second call error = %v", err)
	}

	count, err := db.CountIdeaEndorsements(ctx, "IDEA-1")
	if err != nil {
		t.Fatalf("CountIdeaEndorsements() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountIdeaEndorsements() = %d, want 1 (upsert should not duplicate)", count)
	}
}

func TestGetIdeaVoteCounts(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := db.UpsertIdea(ctx, "PROJ", map[string]any{"id": "IDEA-1", "votes": int64(5)}); err != nil {
		t.Fatalf("UpsertIdea() error = %v", err)
	}
	if err := db.UpsertIdea(ctx, "PROJ", map[string]any{"id": "IDEA-2", "votes": int64(0)}); err != nil {
		t.Fatalf("UpsertIdea() error = %v", err)
	}

	counts, err := db.GetIdeaVoteCounts(ctx, "PROJ")
	if err != nil {
		t.Fatalf("GetIdeaVoteCounts() error = %v", err)
	}
	if counts["IDEA-1"] != 5 {
		t.Errorf("counts[IDEA-1] = %d, want 5", counts["IDEA-1"])
	}
	if counts["IDEA-2"] != 0 {
		t.Errorf("counts[IDEA-2] = %d, want 0", counts["IDEA-2"])
	}
}

func TestGetVoterEmailDomainStatistics(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	voters := []struct {
		id, idea, email string
	}{
		{"END-1", "IDEA-1", "alice@acme.com"},
		{"END-2", "IDEA-1", "bob@acme.com"},
		{"END-3", "IDEA-1", "carol@example.com"},
		{"END-4", "IDEA-2", "dave@example.com"},
	}
	for _, v := range voters {
		data := map[string]any{
			"id":                v.id,
			"idea_id":           v.idea,
			"portal_user_email": v.email,
			"created_at":        time.Now(),
		}
		if err := db.UpsertIdeaEndorsement(ctx, "PROJ", data); err != nil {
			t.Fatalf("UpsertIdeaEndorsement(%s) error = %v", v.id, err)
		}
	}

	t.Run("product-wide", func(t *testing.T) {
		stats, err := db.GetVoterEmailDomainStatistics(ctx, "PROJ", "", 10)
		if err != nil {
			t.Fatalf("GetVoterEmailDomainStatistics() error = %v", err)
		}
		if stats.TotalVoters != 4 {
			t.Errorf("TotalVoters = %d, want 4", stats.TotalVoters)
		}
		if stats.UniqueDomains != 2 {
			t.Errorf("UniqueDomains = %d, want 2", stats.UniqueDomains)
		}
		if len(stats.ByDomain) != 2 || stats.ByDomain[0].Domain != "acme.com" || stats.ByDomain[0].Count != 2 {
			t.Errorf("ByDomain = %+v, want acme.com=2 first (highest count)", stats.ByDomain)
		}
	})

	t.Run("scoped to one idea", func(t *testing.T) {
		stats, err := db.GetVoterEmailDomainStatistics(ctx, "PROJ", "IDEA-2", 10)
		if err != nil {
			t.Fatalf("GetVoterEmailDomainStatistics() error = %v", err)
		}
		if stats.TotalVoters != 1 {
			t.Errorf("TotalVoters = %d, want 1", stats.TotalVoters)
		}
		if len(stats.ByDomain) != 1 || stats.ByDomain[0].Domain != "example.com" {
			t.Errorf("ByDomain = %+v, want only example.com", stats.ByDomain)
		}
	})
}

func TestEndorsementToMap(t *testing.T) {
	now := time.Now()
	e := aha.IdeaEndorsement{
		ID:     "END-1",
		IdeaID: "IDEA-1",
		Weight: 1,
		EndorsedByPortalUser: &aha.EndorsementPortalUser{
			ID: "U-1", Name: "Ada Lovelace", Email: "ada@example.com", CreatedAt: now,
		},
		EndorsedByIdeaUser: &aha.EndorsementIdeaUser{
			ID: "U-2", Name: "Ada Lovelace", Email: "ada@example.com", CreatedAt: now, Title: "Engineer",
		},
	}

	rec := endorsementToMap(e)

	if rec["id"] != "END-1" {
		t.Errorf("id = %v, want END-1", rec["id"])
	}
	if rec["portal_user_email"] != "ada@example.com" {
		t.Errorf("portal_user_email = %v, want ada@example.com", rec["portal_user_email"])
	}
	if rec["idea_user_title"] != "Engineer" {
		t.Errorf("idea_user_title = %v, want Engineer", rec["idea_user_title"])
	}
	if _, ok := rec["value"]; ok {
		t.Errorf("value should be omitted when empty, got %v", rec["value"])
	}
}
