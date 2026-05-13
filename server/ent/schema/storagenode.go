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

// StorageNode holds the schema definition for the StorageNode entity.
type StorageNode struct {
	ent.Schema
}

func (StorageNode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "storage_nodes"},
	}
}

func (StorageNode) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.UUID("storage_group_id", uuid.UUID{}).
			StructTag(`json:"storageGroupId"`),
		field.String("hostname").
			StructTag(`json:"hostname"`),
		field.String("root_path").Default("").
			StructTag(`json:"rootPath"`),
		field.Bool("is_enabled").Default(true).
			StructTag(`json:"isEnabled"`),
		field.Bool("is_master").Default(false).
			StructTag(`json:"isMaster"`),
		field.Int("max_clients").Default(10).
			StructTag(`json:"maxClients"`),
		field.String("ssh_user").Default("fog").
			StructTag(`json:"sshUser"`),
		field.String("web_root").Default("/fog").
			StructTag(`json:"webRoot"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (StorageNode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("storage_group", StorageGroup.Type).Ref("nodes").Field("storage_group_id").Unique().Required(),
		edge.To("tasks", Task.Type),
		edge.To("multicast_sessions", MulticastSession.Type),
	}
}
