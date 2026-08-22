package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IdeaEndorsement holds the schema definition for the IdeaEndorsement entity,
// mirroring an idea's endorsements ("votes" in the Aha UI) as returned by
// GET /ideas/:idea_id/endorsements. Unlike Relationship, each endorsement
// carries Aha's own natural id, so this follows Comment's template (natural
// string PK, parent-ref column, flattened nested-object fields) rather than
// Relationship's synthetic-key/composite-unique-index shape.
type IdeaEndorsement struct {
	ent.Schema
}

// Fields of the IdeaEndorsement.
func (IdeaEndorsement) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("product").Optional(),
		field.String("idea_id").Optional(),
		field.Int("weight").Optional().Default(1),
		// value/link are always null in every sample seen so far; true type
		// is unconfirmed. Stored as nullable strings for forward-compat.
		field.String("value").Optional(),
		field.String("link").Optional(),

		// Flattened from the "endorsed_by_portal_user" nested object.
		field.String("portal_user_id").Optional(),
		field.String("portal_user_name").Optional(),
		field.String("portal_user_email").Optional(),
		field.Time("portal_user_created_at").Optional().Nillable(),

		// Flattened from the "endorsed_by_idea_user" nested object.
		field.String("idea_user_id").Optional(),
		field.String("idea_user_name").Optional(),
		field.String("idea_user_email").Optional(),
		field.String("idea_user_title").Optional(),
		field.Time("idea_user_created_at").Optional().Nillable(),

		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
	}
}

// Indexes of the IdeaEndorsement.
func (IdeaEndorsement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("idea_id"),
		index.Fields("portal_user_email"),
		index.Fields("product", "idea_id"),
	}
}
