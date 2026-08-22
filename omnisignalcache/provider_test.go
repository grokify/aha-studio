package omnisignalcache

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	studiosync "github.com/grokify/aha-studio/sync"
	"github.com/plexusone/omnisignal"
	"github.com/plexusone/omnisignal/metrics"
	"github.com/plexusone/signal-spec/pkg/signal"
)

func openTestDB(t *testing.T) *studiosync.DB {
	t.Helper()
	db, err := studiosync.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedIdeaWithVoters(t *testing.T, db *studiosync.DB, product string) {
	t.Helper()
	ctx := context.Background()

	if err := db.UpsertIdea(ctx, product, map[string]any{
		"id": "IDEA-1", "reference_num": "PROJ-I-1", "name": "Add SSO support",
		"votes": int64(3), "created_at": time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("UpsertIdea() error = %v", err)
	}

	if err := db.UpsertIdeaOrganization(ctx, map[string]any{
		"id": "ORG-1", "name": "Acme Corp", "email_domains": "acme.com",
	}); err != nil {
		t.Fatalf("UpsertIdeaOrganization() error = %v", err)
	}
	if err := db.UpsertIdeaOrganization(ctx, map[string]any{
		"id": "ORG-2", "name": "Globex", "email_domains": "globex.com",
	}); err != nil {
		t.Fatalf("UpsertIdeaOrganization() error = %v", err)
	}

	voters := []struct{ id, email string }{
		{"END-1", "alice@acme.com"},
		{"END-2", "bob@acme.com"},
		{"END-3", "carol@globex.com"},
	}
	for _, v := range voters {
		if err := db.UpsertIdeaEndorsement(ctx, product, map[string]any{
			"id": v.id, "idea_id": "IDEA-1", "portal_user_email": v.email,
		}); err != nil {
			t.Fatalf("UpsertIdeaEndorsement(%s) error = %v", v.id, err)
		}
	}
}

func TestFetch_VoterTiering(t *testing.T) {
	db := openTestDB(t)
	seedIdeaWithVoters(t, db, "PROJ")

	p := NewProvider(db, WithProduct("PROJ"))
	defer func() { _ = p.Close() }()

	signals, err := p.Fetch(context.Background(), omnisignal.FetchOptions{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("len(signals) = %d, want 1", len(signals))
	}
	sig := signals[0]

	if sig.Metadata["aha_voter_count"] != 3 {
		t.Errorf("aha_voter_count = %v, want 3", sig.Metadata["aha_voter_count"])
	}
	if sig.Metadata["aha_voter_org_count"] != 2 {
		t.Errorf("aha_voter_org_count = %v, want 2", sig.Metadata["aha_voter_org_count"])
	}

	hist, ok := sig.Metadata["aha_voter_domain_histogram"].(map[string]int)
	if !ok || hist["acme.com"] != 2 || hist["globex.com"] != 1 {
		t.Errorf("aha_voter_domain_histogram = %+v, want acme.com=2 globex.com=1", sig.Metadata["aha_voter_domain_histogram"])
	}

	refs, ok := sig.Metadata[signal.MetaCustomers].([]string)
	if !ok || len(refs) != 2 {
		t.Fatalf("MetaCustomers = %v, want 2 resolved refs", sig.Metadata[signal.MetaCustomers])
	}

	// No raw voter PII in metadata.
	for k, v := range sig.Metadata {
		if s, ok := v.(string); ok && (s == "alice@acme.com" || s == "bob@acme.com" || s == "carol@globex.com") {
			t.Errorf("metadata[%s] = %q leaks a raw voter email", k, s)
		}
	}
}

func TestFetch_CustomerMappingOverride(t *testing.T) {
	db := openTestDB(t)
	seedIdeaWithVoters(t, db, "PROJ")

	p := NewProvider(db, WithProduct("PROJ"), WithCustomerMappings(map[string]string{
		"Acme Corp": "customer:acme-001",
	}))
	defer func() { _ = p.Close() }()

	signals, err := p.Fetch(context.Background(), omnisignal.FetchOptions{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	refs, _ := signals[0].Metadata[signal.MetaCustomers].([]string)
	var found bool
	for _, r := range refs {
		if r == "customer:acme-001" {
			found = true
		}
	}
	if !found {
		t.Errorf("MetaCustomers = %v, want customer:acme-001 (from override mapping)", refs)
	}
}

func TestFetch_NoProduct(t *testing.T) {
	db := openTestDB(t)
	p := NewProvider(db)
	defer func() { _ = p.Close() }()

	if _, err := p.Fetch(context.Background(), omnisignal.FetchOptions{}); err == nil {
		t.Fatal("expected error when WithProduct is unset")
	}
}

func TestFetch_FeedsReachMetric(t *testing.T) {
	db := openTestDB(t)
	seedIdeaWithVoters(t, db, "PROJ")

	p := NewProvider(db, WithProduct("PROJ"))
	defer func() { _ = p.Close() }()

	signals, err := p.Fetch(context.Background(), omnisignal.FetchOptions{})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	result, err := metrics.Compute(context.Background(), "reach", signals, metrics.Options{})
	if err != nil {
		t.Fatalf("metrics.Compute(reach) error = %v", err)
	}
	if result.Value != 2 {
		t.Errorf("reach = %v, want 2 (Acme Corp + Globex, distinct customer refs)", result.Value)
	}
}

func TestEmailDomain(t *testing.T) {
	tests := []struct{ email, want string }{
		{"alice@acme.com", "acme.com"},
		{"Alice@ACME.com", "acme.com"},
		{"not-an-email", ""},
		{"", ""},
		{"trailing@", ""},
	}
	for _, tt := range tests {
		if got := emailDomain(tt.email); got != tt.want {
			t.Errorf("emailDomain(%q) = %q, want %q", tt.email, got, tt.want)
		}
	}
}
