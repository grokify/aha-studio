package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Idea holds the schema definition for the Idea entity, mirroring the
// `ideas` table defined in sync/db.go.
type Idea struct {
	ent.Schema
}

// Fields of the Idea.
func (Idea) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product"),
		field.String("reference_num").Optional(),
		field.String("name").Optional(),
		field.String("description").Optional(),
		field.String("status").Optional(),
		field.Int("votes").Optional().Default(0),
		field.String("tags").Optional(),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Idea.
func (Idea) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
		index.Fields("status"),
		index.Fields("votes"),
	}
}
