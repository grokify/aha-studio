package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Feature holds the schema definition for the Feature entity, mirroring
// the `features` table defined in sync/db.go.
type Feature struct {
	ent.Schema
}

// Fields of the Feature.
func (Feature) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product"),
		field.String("reference_num").Optional(),
		field.String("name").Optional(),
		field.String("description").Optional(),
		field.String("status").Optional(),
		field.String("assigned_to").Optional(),
		field.String("start_date").Optional(),
		field.String("due_date").Optional(),
		field.String("release").Optional(),
		field.String("release_id").Optional(),
		field.String("release_reference_num").Optional(),
		field.String("tags").Optional(),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Feature.
func (Feature) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
		index.Fields("status"),
		index.Fields("updated_at"),
		index.Fields("release_id"),
		index.Fields("release_reference_num"),
	}
}
