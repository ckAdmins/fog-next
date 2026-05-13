package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// SnapinTask holds the schema definition for the SnapinTask entity.
type SnapinTask struct {
	ent.Schema
}

func (SnapinTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "snapin_tasks"},
	}
}

func (SnapinTask) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("snapin_job_id", uuid.UUID{}).
			StructTag(`json:"snapinJobId"`),
		field.UUID("snapin_id", uuid.UUID{}).
			StructTag(`json:"snapinId"`),
		field.String("state").Default("queued").
			StructTag(`json:"state"`),
		field.Int("exit_code").Optional().Nillable().
			StructTag(`json:"exitCode,omitempty"`),
		field.Time("started_at").Optional().Nillable().
			StructTag(`json:"startedAt,omitempty"`),
		field.Time("completed_at").Optional().Nillable().
			StructTag(`json:"completedAt,omitempty"`),
	}
}

func (SnapinTask) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("job", SnapinJob.Type).Ref("tasks").Field("snapin_job_id").Unique().Required(),
		edge.From("snapin", Snapin.Type).Ref("snapin_tasks").Field("snapin_id").Unique().Required(),
	}
}
