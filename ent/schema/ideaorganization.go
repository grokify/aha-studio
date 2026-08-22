package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IdeaOrganization holds the schema definition for the IdeaOrganization
// entity, mirroring GET /idea_organizations -- Aha's own CRM-style
// customer/account records, account-wide (not scoped to a single idea).
// email_domains is Aha's own authoritative domain-to-organization mapping;
// revenue and endorsements_count only appear on the single-GET response, so
// they're populated only when synced with SyncOptions.Detailed=true (the
// bulk list response omits them).
type IdeaOrganization struct {
	ent.Schema
}

// Fields of the IdeaOrganization.
func (IdeaOrganization) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("name").Optional(),
		field.String("reference_num").Optional(),
		field.String("url").Optional(),
		field.String("email_domains").Optional(),
		field.Float("revenue").Optional().Nillable(),
		field.Int("endorsements_count").Optional().Default(0),
		field.Time("created_at").Optional().Nillable(),
		field.Time("updated_at").Optional().Nillable(),
	}
}

// Indexes of the IdeaOrganization.
func (IdeaOrganization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email_domains"),
		index.Fields("reference_num"),
	}
}
