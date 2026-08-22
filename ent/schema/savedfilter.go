package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SavedFilter holds the schema definition for the SavedFilter entity,
// mirroring the `saved_filters` table defined in sync/db.go. No `data`
// JSON column on this table. `created_at`/`updated_at` had
// `DEFAULT CURRENT_TIMESTAMP` in the original SQL; Ent v0.14 doesn't
// portably push that into DDL, so the default moves to the Go layer here
// (CreateFilter/UpdateFilter already set timestamps manually today, so
// this is like-for-like, not new behavior).
type SavedFilter struct {
	ent.Schema
}

// Fields of the SavedFilter.
func (SavedFilter) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("name").Unique(),
		field.String("aql"),
		field.String("product").Optional(),
		field.String("description").Optional(),
		field.Bool("is_favorite").Optional().Default(false),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now),
	}
}

// Indexes of the SavedFilter.
func (SavedFilter) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_favorite"),
		index.Fields("product"),
	}
}
