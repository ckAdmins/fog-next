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

// Printer holds the schema definition for the Printer entity.
type Printer struct {
	ent.Schema
}

func (Printer) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "printers"},
	}
}

func (Printer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.String("type").Default("local").
			StructTag(`json:"type"`),
		field.String("port").Default("").
			StructTag(`json:"port"`),
		field.String("ip").Default("").
			StructTag(`json:"ip"`),
		field.String("model").Default("").
			StructTag(`json:"model"`),
		field.String("driver").Default("").
			StructTag(`json:"driver"`),
		field.Bool("is_default").Default(false).
			StructTag(`json:"isDefault"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Printer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("assocs", PrinterAssoc.Type),
	}
}
