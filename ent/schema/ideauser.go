package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IdeaUser holds the schema definition for the IdeaUser entity, mirroring
// GET /idea_users -- voter identities, account-wide (not scoped to a single
// idea, like Comment/IdeaEndorsement are). idea_organizations membership is
// stored as a JSON array of light {id, name} refs rather than a join table:
// membership is small (0-2 orgs per sample seen) and only ever read
// alongside its parent IdeaUser row, matching the "JSON catch-all for
// less-critical relational data" convention already used by Comment.data.
type IdeaUser struct {
	ent.Schema
}

// Fields of the IdeaUser.
func (IdeaUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("name").Optional(),
		field.String("email").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.JSON("idea_organizations", []map[string]any{}).Optional(),
	}
}

// Indexes of the IdeaUser.
func (IdeaUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email"),
	}
}
