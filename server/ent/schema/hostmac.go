package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// HostMAC holds the schema definition for the HostMAC entity.
type HostMAC struct {
	ent.Schema
}

func (HostMAC) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "host_macs"},
	}
}

func (HostMAC) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.String("mac").NotEmpty().
			StructTag(`json:"mac"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.Bool("is_primary").Default(false).
			StructTag(`json:"isPrimary"`),
		field.Bool("is_ignored").Default(false).
			StructTag(`json:"isIgnored"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
	}
}

func (HostMAC) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).Ref("macs").Field("host_id").Unique().Required(),
	}
}
