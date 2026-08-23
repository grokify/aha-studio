package omniroadmap

import (
	"fmt"
	"strings"
	"time"

	"github.com/grokify/omniroadmap-core/provider"
)

func canonicalID(sourceID string) string {
	return fmt.Sprintf("%s:%s", providerName, sourceID)
}

// entityTable maps a canonical ItemKind to the cache's entity table name.
func entityTable(kind provider.ItemKind) (string, bool) {
	switch kind {
	case provider.ItemKindFeature:
		return "features", true
	case provider.ItemKindEpic:
		return "epics", true
	case provider.ItemKindInitiative:
		return "initiatives", true
	default:
		return "", false
	}
}

// itemFromRecord converts a flattened cache record (columns + data JSON
// keys, as returned by sync.DB.ListRecords/GetRecord) to a canonical Item.
func itemFromRecord(rec map[string]any, kind provider.ItemKind) provider.Item {
	id := recString(rec, "id")
	item := provider.Item{
		ID:          canonicalID(id),
		Provider:    providerName,
		SourceID:    id,
		SourceRef:   recString(rec, "reference_num"),
		SourceURL:   recString(rec, "url"),
		Kind:        kind,
		Name:        recString(rec, "name"),
		Description: recString(rec, "description"),
		Status:      statusFromName(recString(rec, "status")),
		StartDate:   recDate(rec, "start_date"),
		CreatedAt:   recTime(rec, "created_at"),
		UpdatedAt:   recTime(rec, "updated_at"),
		Tags:        recTags(rec),
	}

	// Features use due_date; initiatives use end_date.
	if d := recDate(rec, "due_date"); d != nil {
		item.DueDate = d
	} else if d := recDate(rec, "end_date"); d != nil {
		item.DueDate = d
	}

	if releaseID := recString(rec, "release_id"); releaseID != "" {
		item.ReleaseID = canonicalID(releaseID)
	}
	if assignee := recString(rec, "assigned_to"); assignee != "" {
		item.Owner = &provider.Person{Name: assignee}
	}
	if progress, ok := recFloat(rec, "progress"); ok {
		item.Progress = &progress
	}
	item.CustomFields = recCustomFields(rec)

	if product := recString(rec, "product"); product != "" {
		item.WorkspaceRef = product
		item.Metadata = map[string]any{"aha-studio.product": product}
	}
	return item
}

// releaseFromRecord converts a flattened releases-table record to a
// canonical Release.
func releaseFromRecord(rec map[string]any) provider.Release {
	id := recString(rec, "id")
	rel := provider.Release{
		ID:          canonicalID(id),
		Provider:    providerName,
		SourceID:    id,
		SourceRef:   recString(rec, "reference_num"),
		SourceURL:   recString(rec, "url"),
		Name:        recString(rec, "name"),
		StartDate:   recDate(rec, "start_date"),
		ReleaseDate: recDate(rec, "release_date"),
		Released:    recBool(rec, "released"),
	}
	meta := map[string]any{}
	if product := recString(rec, "product"); product != "" {
		meta["aha-studio.product"] = product
	}
	if parkingLot := recBool(rec, "parking_lot"); parkingLot {
		meta["aha-studio.parking_lot"] = true
	}
	if len(meta) > 0 {
		rel.Metadata = meta
	}
	return rel
}

// statusFromName normalizes a workflow status name into a canonical
// Status. Only the display name is cached (no Complete flag or Position),
// so Category is a name heuristic; the live "aha" provider gives richer
// status data.
func statusFromName(name string) *provider.Status {
	if name == "" {
		return nil
	}
	category := provider.StatusCategoryInProgress
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "cancel"):
		category = provider.StatusCategoryCanceled
	case strings.Contains(lower, "shipped"),
		strings.Contains(lower, "complete"),
		strings.Contains(lower, "done"),
		strings.Contains(lower, "released"):
		category = provider.StatusCategoryDone
	case strings.Contains(lower, "backlog"),
		strings.Contains(lower, "not started"),
		strings.Contains(lower, "new"),
		strings.Contains(lower, "consideration"),
		strings.Contains(lower, "future"):
		category = provider.StatusCategoryTodo
	}
	return &provider.Status{
		Name:     name,
		Category: category,
		Complete: category == provider.StatusCategoryDone,
	}
}

// recCustomFields extracts the cached custom_fields array (present only
// for detail-synced records) as canonical CustomFields.
func recCustomFields(rec map[string]any) []provider.CustomField {
	raw, ok := rec["custom_fields"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	var out []provider.CustomField
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		cf := provider.CustomField{Value: m["value"]}
		if s, ok := m["key"].(string); ok {
			cf.Key = s
		}
		if s, ok := m["name"].(string); ok {
			cf.Name = s
		}
		if s, ok := m["type"].(string); ok {
			cf.Type = s
		}
		out = append(out, cf)
	}
	return out
}

func recString(rec map[string]any, key string) string {
	if s, ok := rec[key].(string); ok {
		return s
	}
	return ""
}

func recBool(rec map[string]any, key string) bool {
	switch v := rec[key].(type) {
	case bool:
		return v
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func recFloat(rec map[string]any, key string) (float64, bool) {
	switch v := rec[key].(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// recTags handles both storage shapes: the tags column (comma-separated
// string) and a data-JSON array.
func recTags(rec map[string]any) []string {
	switch v := rec["tags"].(type) {
	case string:
		if v == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		tags := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				tags = append(tags, p)
			}
		}
		return tags
	case []any:
		var tags []string
		for _, t := range v {
			if s, ok := t.(string); ok && s != "" {
				tags = append(tags, s)
			}
		}
		return tags
	default:
		return nil
	}
}

// recDate parses a date-only column ("2006-01-02"), tolerating full
// timestamps too.
func recDate(rec map[string]any, key string) *time.Time {
	s := recString(rec, key)
	if s == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// recTime parses a timestamp value, which arrives as time.Time from
// time-typed columns or as a string from data-JSON keys.
func recTime(rec map[string]any, key string) *time.Time {
	switch v := rec[key].(type) {
	case time.Time:
		if v.IsZero() {
			return nil
		}
		return &v
	case string:
		return recDate(rec, key)
	default:
		return nil
	}
}
