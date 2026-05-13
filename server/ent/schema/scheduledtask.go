package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ScheduledTask holds the schema definition for the ScheduledTask entity.
type ScheduledTask struct {
	ent.Schema
}

func (ScheduledTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "scheduled_tasks"},
	}
}

func (ScheduledTask) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").Default("").
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.String("task_type").
			StructTag(`json:"taskType"`),
		field.String("minute").Default("*").
			StructTag(`json:"minute"`),
		field.String("hour").Default("*").
			StructTag(`json:"hour"`),
		field.String("day_of_month").Default("*").
			StructTag(`json:"dayOfMonth"`),
		field.String("month").Default("*").
			StructTag(`json:"month"`),
		field.String("day_of_week").Default("*").
			StructTag(`json:"dayOfWeek"`),
		field.Bool("is_group").Default(false).
			StructTag(`json:"isGroup"`),
		field.UUID("target_id", uuid.UUID{}).
			StructTag(`json:"targetId"`),
		field.Bool("is_shutdown").Default(false).
			StructTag(`json:"isShutdown"`),
		field.Bool("is_active").Default(true).
			StructTag(`json:"isActive"`),
		field.Time("next_run_at").Optional().Nillable().
			StructTag(`json:"nextRunAt,omitempty"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
	}
}
