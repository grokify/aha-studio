package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Initiative holds the schema definition for the Initiative entity,
// mirroring the `initiatives` table defined in sync/db.go. See
// /Users/johnwang/.claude/plans/sprightly-waddling-platypus.md.
type Initiative struct {
	ent.Schema
}

// Fields of the Initiative.
func (Initiative) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product"),
		field.String("reference_num").Optional(),
		field.String("name").Optional(),
		field.String("description").Optional(),
		field.String("status").Optional(),
		field.Float("value").Optional().Nillable(),
		field.Float("effort").Optional().Nillable(),
		field.Float("progress").Optional().Nillable(),
		field.String("start_date").Optional(),
		field.String("end_date").Optional(),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Initiative.
func (Initiative) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
	}
}
