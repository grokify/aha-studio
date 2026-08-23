package omniroadmap

import (
	"path/filepath"
	"testing"

	studiosync "github.com/grokify/aha-studio/sync"
	omniroadmap "github.com/grokify/omniroadmap-core"
	"github.com/grokify/omniroadmap-core/provider"
	"github.com/grokify/omniroadmap-core/provider/providertest"
)

// newSeededProvider opens a temp-file cache DB and seeds it through the
// same upsert path the real syncer uses.
func newSeededProvider(t *testing.T, opts ...Option) *Provider {
	t.Helper()
	db, err := studiosync.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatalf("sync.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := t.Context()
	seed := []struct {
		upsert func() error
	}{
		{func() error {
			return db.UpsertFeature(ctx, "PROJ", map[string]any{
				"id":            "feat-1",
				"reference_num": "PROJ-1",
				"name":          "SSO Integration",
				"description":   "Add single sign-on",
				"status":        "In development",
				"assigned_to":   "Alice",
				"start_date":    "2026-09-01",
				"due_date":      "2026-09-30",
				"release_id":    "rel-1",
				"url":           "https://test.aha.io/features/PROJ-1",
				"custom_fields": []map[string]any{
					{"key": "priority_moscow", "name": "MoSCoW", "value": "Must Have", "type": "string"},
					{"key": "rice_reach", "name": "Reach", "value": "5000", "type": "number"},
				},
			})
		}},
		{func() error {
			return db.UpsertFeature(ctx, "PROJ", map[string]any{
				"id": "feat-2", "reference_num": "PROJ-2", "name": "Dark mode", "status": "Backlog",
			})
		}},
		{func() error {
			return db.UpsertInitiative(ctx, "PROJ", map[string]any{
				"id": "init-1", "reference_num": "PROJ-S-1", "name": "Enterprise Push", "status": "Shipped",
			})
		}},
		{func() error {
			return db.UpsertEpic(ctx, "PROJ", map[string]any{
				"id": "epic-1", "reference_num": "PROJ-E-1", "name": "Auth Epic", "status": "Cancelled",
			})
		}},
		{func() error {
			return db.UpsertRelease(ctx, "PROJ", map[string]any{
				"id": "rel-1", "reference_num": "PROJ-R-1", "name": "Q3 Release",
				"release_date": "2026-09-30", "released": true,
			})
		}},
	}
	for _, s := range seed {
		if err := s.upsert(); err != nil {
			t.Fatalf("seeding: %v", err)
		}
	}
	return NewProvider(db, opts...)
}

func TestConformance(t *testing.T) {
	p := newSeededProvider(t)
	providertest.RunAll(t, providertest.Config{
		Provider:     p,
		TestItemID:   "PROJ-1", // by reference number
		TestItemKind: provider.ItemKindFeature,
	})
}

func TestListItems_AllKinds(t *testing.T) {
	p := newSeededProvider(t)

	resp, err := p.ListItems(t.Context(), &provider.ListItemsRequest{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(resp.Items) != 4 {
		t.Fatalf("len(Items) = %d, want 4 (2 features + 1 initiative + 1 epic)", len(resp.Items))
	}

	kinds := map[provider.ItemKind]int{}
	for _, item := range resp.Items {
		kinds[item.Kind]++
	}
	if kinds[provider.ItemKindFeature] != 2 || kinds[provider.ItemKindInitiative] != 1 || kinds[provider.ItemKindEpic] != 1 {
		t.Errorf("kind counts = %v", kinds)
	}
}

func TestGetItem_CustomFieldsSurvive(t *testing.T) {
	p := newSeededProvider(t)

	item, err := p.GetItem(t.Context(), &provider.GetItemRequest{
		Kind: provider.ItemKindFeature,
		ID:   "feat-1",
	})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.ID != "aha-studio:feat-1" {
		t.Errorf("ID = %q, want aha-studio:feat-1", item.ID)
	}
	if item.SourceRef != "PROJ-1" {
		t.Errorf("SourceRef = %q, want PROJ-1", item.SourceRef)
	}
	if item.WorkspaceRef != "PROJ" {
		t.Errorf("WorkspaceRef = %q, want PROJ", item.WorkspaceRef)
	}
	if item.Status == nil || item.Status.Category != provider.StatusCategoryInProgress {
		t.Errorf("Status = %+v, want in_progress", item.Status)
	}
	if item.ReleaseID != "aha-studio:rel-1" {
		t.Errorf("ReleaseID = %q, want aha-studio:rel-1", item.ReleaseID)
	}
	if item.StartDate == nil || item.StartDate.Format("2006-01-02") != "2026-09-01" {
		t.Errorf("StartDate = %v, want 2026-09-01", item.StartDate)
	}
	if item.Owner == nil || item.Owner.Name != "Alice" {
		t.Errorf("Owner = %+v, want Alice", item.Owner)
	}

	if len(item.CustomFields) != 2 {
		t.Fatalf("len(CustomFields) = %d, want 2 (detail-synced record)", len(item.CustomFields))
	}
	byKey := map[string]provider.CustomField{}
	for _, cf := range item.CustomFields {
		byKey[cf.Key] = cf
	}
	if byKey["priority_moscow"].Value != "Must Have" {
		t.Errorf("priority_moscow = %v, want Must Have", byKey["priority_moscow"].Value)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	p := newSeededProvider(t)
	_, err := p.GetItem(t.Context(), &provider.GetItemRequest{
		Kind: provider.ItemKindFeature,
		ID:   "nope",
	})
	if !omniroadmap.IsNotFound(err) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestListReleases(t *testing.T) {
	p := newSeededProvider(t)
	resp, err := p.ListReleases(t.Context(), &provider.ListReleasesRequest{})
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(resp.Releases) != 1 {
		t.Fatalf("len(Releases) = %d, want 1", len(resp.Releases))
	}
	rel := resp.Releases[0]
	if !rel.Released {
		t.Error("Released = false, want true")
	}
	if rel.ReleaseDate == nil || rel.ReleaseDate.Format("2006-01-02") != "2026-09-30" {
		t.Errorf("ReleaseDate = %v, want 2026-09-30", rel.ReleaseDate)
	}
}

func TestListStatuses_DerivedFromItems(t *testing.T) {
	p := newSeededProvider(t)
	resp, err := p.ListStatuses(t.Context(), &provider.ListStatusesRequest{})
	if err != nil {
		t.Fatalf("ListStatuses: %v", err)
	}

	categories := map[string]provider.StatusCategory{}
	for _, s := range resp.Statuses {
		categories[s.Name] = s.Category
	}
	if categories["In development"] != provider.StatusCategoryInProgress {
		t.Errorf("In development = %q, want in_progress", categories["In development"])
	}
	if categories["Backlog"] != provider.StatusCategoryTodo {
		t.Errorf("Backlog = %q, want todo", categories["Backlog"])
	}
	if categories["Shipped"] != provider.StatusCategoryDone {
		t.Errorf("Shipped = %q, want done", categories["Shipped"])
	}
	if categories["Cancelled"] != provider.StatusCategoryCanceled {
		t.Errorf("Cancelled = %q, want canceled", categories["Cancelled"])
	}
}

func TestListCustomFieldDefinitions_Unsupported(t *testing.T) {
	p := newSeededProvider(t)
	_, err := p.ListCustomFieldDefinitions(t.Context(), &provider.ListCustomFieldDefinitionsRequest{})
	if !omniroadmap.IsUnsupportedOperation(err) {
		t.Errorf("err = %v, want ErrUnsupportedOperation", err)
	}
}

func TestListItems_ProductScoping(t *testing.T) {
	p := newSeededProvider(t, WithProduct("OTHER"))
	resp, err := p.ListItems(t.Context(), &provider.ListItemsRequest{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0 for a product with no records", len(resp.Items))
	}
}
