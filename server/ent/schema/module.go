package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Module holds the schema definition for the Module entity.
type Module struct {
	ent.Schema
}

func (Module) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "modules"},
	}
}

func (Module) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().Unique().
			StructTag(`json:"name"`),
		field.String("short_name").NotEmpty().Unique().
			StructTag(`json:"shortName"`),
		field.Bool("is_default").Default(false).
			StructTag(`json:"isDefault"`),
		field.Bool("is_enabled").Default(true).
			StructTag(`json:"isEnabled"`),
	}
}

func (Module) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("statuses", ModuleStatus.Type),
	}
}
