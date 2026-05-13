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

// GroupMember holds the schema definition for the GroupMember entity.
type GroupMember struct {
	ent.Schema
}

func (GroupMember) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_members"},
	}
}

func (GroupMember) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("group_id", uuid.UUID{}).
			StructTag(`json:"groupId"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
	}
}

func (GroupMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "host_id").Unique(),
	}
}

func (GroupMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).Ref("members").Field("group_id").Unique().Required(),
		edge.From("host", Host.Type).Ref("group_members").Field("host_id").Unique().Required(),
	}
}
