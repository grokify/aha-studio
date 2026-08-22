package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity, mirroring the
// `users` table defined in sync/db.go. No indexes on this table today.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("first_name").Optional(),
		field.String("last_name").Optional(),
		field.String("email").Optional(),
		field.String("role").Optional(),
		field.Time("created_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}
