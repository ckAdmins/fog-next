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

// StorageGroup holds the schema definition for the StorageGroup entity.
type StorageGroup struct {
	ent.Schema
}

func (StorageGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "storage_groups"},
	}
}

func (StorageGroup) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().Unique().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (StorageGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("nodes", StorageNode.Type),
		edge.To("images", Image.Type),
		edge.To("tasks", Task.Type),
	}
}
