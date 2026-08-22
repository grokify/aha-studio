package sync

import "time"

// The sync layer's Upsert* methods take a loosely-typed map[string]any
// because the data originates from Aha's API (via sync.go), which is
// inherently semi-structured (custom fields, optional API fields). These
// helpers safely extract typed values for Ent's typed builder methods,
// mirroring the safe `data[key].(T)` type assertions the hand-written SQL
// version used (defaulting rather than panicking on a missing/wrong-type
// key).

// mapID extracts the required "id" field - a missing/wrong-type key
// defaults to "".
func mapID(data map[string]any) string {
	s, _ := data["id"].(string)
	return s
}

// mapInt is for fields with a schema-level Default(0) that should always
// be explicitly set on upsert (unlike optional fields using the *Ptr
// helpers below, a missing key here legitimately means zero, e.g. "votes"
// - not "leave whatever was there").
func mapInt(data map[string]any, key string) int {
	switch v := data[key].(type) {
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

// The *Ptr helpers feed Ent's generated SetNillableX(*T) builder methods
// (generated for every Optional() field regardless of whether the schema
// field itself is marked .Nillable()): when the sync payload doesn't
// contain a key, SetNillableX(nil) is a no-op, so upserting doesn't
// overwrite a previously-known-good value with a zero value just because
// this sync round's payload happened to omit that field.

func mapStringPtr(data map[string]any, key string) *string {
	s, ok := data[key].(string)
	if !ok {
		return nil
	}
	return &s
}

func mapFloat64Ptr(data map[string]any, key string) *float64 {
	f, ok := data[key].(float64)
	if !ok {
		return nil
	}
	return &f
}

func mapIntPtr(data map[string]any, key string) *int {
	var v int
	switch x := data[key].(type) {
	case int64:
		v = int(x)
	case int:
		v = x
	default:
		return nil
	}
	return &v
}

// mapMapSlice extracts a []map[string]any field (e.g. IdeaUser's embedded
// idea_organizations refs) - a missing/wrong-type key defaults to nil, which
// Ent's SetIdeaOrganizations([]map[string]any(nil)) stores as an empty JSON
// array, not an error.
func mapMapSlice(data map[string]any, key string) []map[string]any {
	v, _ := data[key].([]map[string]any)
	return v
}

func mapTimePtr(data map[string]any, key string) *time.Time {
	t, ok := data[key].(time.Time)
	if !ok {
		return nil
	}
	return &t
}
