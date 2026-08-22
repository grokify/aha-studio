package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Release holds the schema definition for the Release entity, mirroring
// the `releases` table defined in sync/db.go.
type Release struct {
	ent.Schema
}

// Fields of the Release.
func (Release) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product"),
		field.String("reference_num").Optional(),
		field.String("name").Optional(),
		field.String("start_date").Optional(),
		field.String("release_date").Optional(),
		field.Bool("released").Optional().Default(false),
		field.Bool("parking_lot").Optional().Default(false),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Release.
func (Release) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
	}
}
