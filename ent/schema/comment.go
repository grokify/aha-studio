package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Comment holds the schema definition for the Comment entity, mirroring
// the `comments` table defined in sync/db.go. Note: unlike most other
// entities, `product` is nullable here (no NOT NULL in the original
// schema).
type Comment struct {
	ent.Schema
}

// Fields of the Comment.
func (Comment) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product").Optional(),
		field.String("commentable_type").Optional(),
		field.String("commentable_id").Optional(),
		field.String("body").Optional(),
		field.String("user_id").Optional(),
		field.String("url").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}

// Indexes of the Comment.
func (Comment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product"),
		index.Fields("commentable_type", "commentable_id"),
		index.Fields("user_id"),
	}
}
