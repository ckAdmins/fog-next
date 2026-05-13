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

// PendingMAC holds the schema definition for the PendingMAC entity.
type PendingMAC struct {
	ent.Schema
}

func (PendingMAC) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "pending_macs"},
	}
}

func (PendingMAC) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("mac").NotEmpty().Unique().
			StructTag(`json:"mac"`),
		field.UUID("host_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"hostId,omitempty"`),
		field.Time("seen_at").Default(time.Now).
			StructTag(`json:"seenAt"`),
	}
}

func (PendingMAC) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).Ref("pending_macs").Field("host_id").Unique(),
	}
}
