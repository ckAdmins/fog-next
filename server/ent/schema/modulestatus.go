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

// ModuleStatus holds the schema definition for the ModuleStatus entity.
type ModuleStatus struct {
	ent.Schema
}

func (ModuleStatus) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "module_status"},
	}
}

func (ModuleStatus) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.UUID("module_id", uuid.UUID{}).
			StructTag(`json:"moduleId"`),
		field.Bool("is_on").Default(true).
			StructTag(`json:"isOn"`),
	}
}

func (ModuleStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("host_id", "module_id").Unique(),
	}
}

func (ModuleStatus) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("module", Module.Type).Ref("statuses").Field("module_id").Unique().Required(),
	}
}
