package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Relationship holds the schema definition for the Relationship entity, a
// generic directed-edge table for offline joins between entity types
// (`from_type`/`from_id` -> `rel_type` -> `to_type`/`to_id`), mirroring
// the `relationships` table defined in sync/db.go.
//
// Unlike most other tables, this one has a surrogate
// `INTEGER PRIMARY KEY AUTOINCREMENT` id in the original schema, so no
// `field.String("id")` override here — Ent's default int-autoincrement ID
// matches it directly. Uniqueness is enforced via the composite index
// below (mirroring the original UNIQUE(from_type, from_id, rel_type,
// to_type, to_id) constraint), same pattern as visionstudio's
// initiativedependency.go et al.
type Relationship struct {
	ent.Schema
}

// Fields of the Relationship.
func (Relationship) Fields() []ent.Field {
	return []ent.Field{
		field.String("from_type"),
		field.String("from_id"),
		field.String("rel_type"),
		field.String("to_type"),
		field.String("to_id"),
		field.String("product").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Indexes of the Relationship.
func (Relationship) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("from_type", "from_id", "rel_type", "to_type", "to_id").Unique(),
		index.Fields("from_type", "from_id"),
		index.Fields("to_type", "to_id"),
		index.Fields("rel_type"),
		index.Fields("product"),
	}
}
