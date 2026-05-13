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

// MulticastSession holds the schema definition for the MulticastSession entity.
type MulticastSession struct {
	ent.Schema
}

func (MulticastSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "multicast_sessions"},
	}
}

func (MulticastSession) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").Default("").
			StructTag(`json:"name"`),
		field.UUID("image_id", uuid.UUID{}).
			StructTag(`json:"imageId"`),
		field.UUID("storage_node_id", uuid.UUID{}).
			StructTag(`json:"storageNodeId"`),
		field.Int("port").
			StructTag(`json:"port"`),
		field.String("interface").Default("").
			StructTag(`json:"interface"`),
		field.Int("client_count").Default(0).
			StructTag(`json:"clientCount"`),
		field.String("state").Default("pending").
			StructTag(`json:"state"`),
		field.Time("started_at").Optional().Nillable().
			StructTag(`json:"startedAt,omitempty"`),
		field.Time("completed_at").Optional().Nillable().
			StructTag(`json:"completedAt,omitempty"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
	}
}

func (MulticastSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("image", Image.Type).Ref("multicast_sessions").Field("image_id").Unique().Required(),
		edge.From("storage_node", StorageNode.Type).Ref("multicast_sessions").Field("storage_node_id").Unique().Required(),
	}
}
