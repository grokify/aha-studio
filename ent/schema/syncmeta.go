package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SyncMeta holds the schema definition for the SyncMeta entity, mirroring
// the `sync_meta` table defined in sync/db.go. Original schema has a
// composite natural key (entity, product), no surrogate id column; Ent
// has no native composite-PK support for regular node schemas, so this
// uses the same workaround as Relationship: default int-autoincrement ID
// + a unique composite index. Table name is explicitly pinned via
// entsql.Annotation since Ent's default pluralization of "SyncMeta"
// would produce "sync_metas", not the existing "sync_meta".
type SyncMeta struct {
	ent.Schema
}

// Annotations of the SyncMeta.
func (SyncMeta) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sync_meta"},
	}
}

// Fields of the SyncMeta.
func (SyncMeta) Fields() []ent.Field {
	return []ent.Field{
		field.String("entity"),
		field.String("product"),
		field.Time("last_sync"),
		field.Int("record_count").Optional().Default(0),
	}
}

// Indexes of the SyncMeta.
func (SyncMeta) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("entity", "product").Unique(),
	}
}
