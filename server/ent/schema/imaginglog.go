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

// ImagingLog holds the schema definition for the ImagingLog entity.
type ImagingLog struct {
	ent.Schema
}

func (ImagingLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "imaging_logs"},
	}
}

func (ImagingLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.UUID("task_id", uuid.UUID{}).
			StructTag(`json:"taskId"`),
		field.String("task_type").
			StructTag(`json:"taskType"`),
		field.UUID("image_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"imageId,omitempty"`),
		field.Int64("size_bytes").Default(0).
			StructTag(`json:"sizeBytes"`),
		field.Int64("duration").Default(0).
			StructTag(`json:"duration"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
	}
}

func (ImagingLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).Ref("imaging_logs").Field("host_id").Unique().Required(),
		edge.From("image", Image.Type).Ref("imaging_logs").Field("image_id").Unique(),
		edge.From("task", Task.Type).Ref("imaging_log").Field("task_id").Unique().Required(),
	}
}
