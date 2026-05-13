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

// AgentLog holds a single log line forwarded by fos-agent during a task.
type AgentLog struct {
	ent.Schema
}

func (AgentLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "agent_logs"},
	}
}

func (AgentLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("task_id", uuid.UUID{}).
			StructTag(`json:"taskId"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.Time("logged_at").
			StructTag(`json:"loggedAt"`),
		field.String("level").
			StructTag(`json:"level"`),
		field.String("message").
			StructTag(`json:"message"`),
		field.JSON("attrs", map[string]any{}).Optional().
			StructTag(`json:"attrs,omitempty"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
	}
}

func (AgentLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).Ref("agent_logs").Field("task_id").Unique().Required(),
		edge.From("host", Host.Type).Ref("agent_logs").Field("host_id").Unique().Required(),
	}
}
