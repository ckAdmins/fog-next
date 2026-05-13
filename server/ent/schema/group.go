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

// Group holds the schema definition for the Group entity.
type Group struct {
	ent.Schema
}

func (Group) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "groups"},
	}
}

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.String("created_by").Default("").
			StructTag(`json:"createdBy"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("members", GroupMember.Type),
	}
}
