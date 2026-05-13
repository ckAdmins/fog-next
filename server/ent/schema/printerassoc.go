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

// PrinterAssoc holds the schema definition for the PrinterAssoc entity.
type PrinterAssoc struct {
	ent.Schema
}

func (PrinterAssoc) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "printer_assocs"},
	}
}

func (PrinterAssoc) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("printer_id", uuid.UUID{}).
			StructTag(`json:"printerId"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.Bool("is_default").Default(false).
			StructTag(`json:"isDefault"`),
	}
}

func (PrinterAssoc) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("printer_id", "host_id").Unique(),
	}
}

func (PrinterAssoc) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("printer", Printer.Type).Ref("assocs").Field("printer_id").Unique().Required(),
	}
}
