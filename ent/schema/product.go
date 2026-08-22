package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Product holds the schema definition for the Product entity, mirroring
// the `products` table defined in sync/db.go. No indexes on this table
// today.
type Product struct {
	ent.Schema
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("reference_prefix").Optional(),
		field.String("name").Optional(),
		field.Bool("product_line").Optional().Default(false),
		field.Bool("has_ideas").Optional().Default(false),
		field.Time("created_at").Optional().Nillable(),
		field.JSON("data", map[string]any{}).Optional(),
	}
}
