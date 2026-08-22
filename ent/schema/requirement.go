package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Requirement holds the schema definition for the Requirement entity,
// mirroring the `requirements` table defined in sync/db.go.
type Requirement struct {
	ent.Schema
}

// Fields of the Requirement.
func (Requirement) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product"),
		field.String("feature_id").Optional(),
		field.String("reference_num").Optional(),
		field.String("name").Optional(),
		field.String("description").Optional(),
		field.String("status").Optional(),
		field.String("assigned_to").Optional(),
		field.Int("position").Optional().Nillable(),
		field.Float("original_estimate").Optional().Nillable(),
		field.Float("remaining_estimate").Optional().Nillable(),
		field.Float("work_done").Optional().Nillable(),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Requirement.
func (Requirement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
		index.Fields("feature_id"),
		index.Fields("status"),
	}
}
