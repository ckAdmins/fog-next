package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SnapinAssoc holds the schema definition for the SnapinAssoc entity.
type SnapinAssoc struct {
	ent.Schema
}

func (SnapinAssoc) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "snapin_assocs"},
	}
}

func (SnapinAssoc) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("snapin_id", uuid.UUID{}).
			StructTag(`json:"snapinId"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
	}
}

func (SnapinAssoc) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("snapin_id", "host_id").Unique(),
	}
}

func (SnapinAssoc) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("snapin", Snapin.Type).Ref("assocs").Field("snapin_id").Unique().Required(),
		edge.From("host", Host.Type).Ref("snapin_assocs").Field("host_id").Unique().Required(),
	}
}
