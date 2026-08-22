package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Goal holds the schema definition for the Goal entity, mirroring the
// `goals` table defined in sync/db.go.
type Goal struct {
	ent.Schema
}

// Fields of the Goal.
func (Goal) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product"),
		field.String("reference_num").Optional(),
		field.String("name").Optional(),
		field.String("description").Optional(),
		field.String("status").Optional(),
		field.Float("progress").Optional().Nillable(),
		field.String("start_date").Optional(),
		field.String("end_date").Optional(),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Goal.
func (Goal) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
	}
}
