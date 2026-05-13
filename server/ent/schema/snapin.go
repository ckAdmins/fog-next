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

// Snapin holds the schema definition for the Snapin entity.
type Snapin struct {
	ent.Schema
}

func (Snapin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "snapins"},
	}
}

func (Snapin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.String("file_name").Default("").
			StructTag(`json:"fileName"`),
		field.String("file_path").Default("").
			StructTag(`json:"filePath"`),
		field.String("command").Default("").
			StructTag(`json:"command"`),
		field.String("arguments").Default("").
			StructTag(`json:"arguments"`),
		field.String("run_with").Default("").
			StructTag(`json:"runWith"`),
		field.String("hash").Default("").
			StructTag(`json:"hash"`),
		field.Int64("size_bytes").Default(0).
			StructTag(`json:"sizeBytes"`),
		field.Bool("is_enabled").Default(true).
			StructTag(`json:"isEnabled"`),
		field.Bool("to_replicate").Default(false).
			StructTag(`json:"toReplicate"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.String("created_by").Default("").
			StructTag(`json:"createdBy"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Snapin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("assocs", SnapinAssoc.Type),
		edge.To("snapin_tasks", SnapinTask.Type),
	}
}
