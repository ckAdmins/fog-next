package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// OSType holds the schema definition for the OSType entity.
type OSType struct {
	ent.Schema
}

func (OSType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "os_types"},
	}
}

func (OSType) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().Unique().
			StructTag(`json:"name"`),
	}
}

func (OSType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("images", Image.Type),
	}
}
