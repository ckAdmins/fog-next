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

// SnapinJob holds the schema definition for the SnapinJob entity.
type SnapinJob struct {
	ent.Schema
}

func (SnapinJob) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "snapin_jobs"},
	}
}

func (SnapinJob) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.String("state").Default("queued").
			StructTag(`json:"state"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (SnapinJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).Ref("snapin_jobs").Field("host_id").Unique().Required(),
		edge.To("tasks", SnapinTask.Type),
	}
}
