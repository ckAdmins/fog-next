package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ImageType holds the schema definition for the ImageType entity.
type ImageType struct {
	ent.Schema
}

func (ImageType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "image_types"},
	}
}

func (ImageType) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().Unique().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
	}
}

func (ImageType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("images", Image.Type),
	}
}
