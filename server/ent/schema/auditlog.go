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

// AuditLog holds the schema definition for the AuditLog entity.
type AuditLog struct {
	ent.Schema
}

func (AuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "audit_logs"},
	}
}

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("user_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"userId,omitempty"`),
		field.String("username").Default("").
			StructTag(`json:"username"`),
		field.String("action").
			StructTag(`json:"action"`),
		field.String("resource").
			StructTag(`json:"resource"`),
		field.String("resource_id").Default("").
			StructTag(`json:"resourceId"`),
		field.String("details").Default("").
			StructTag(`json:"details"`),
		field.String("ip_address").Default("").
			StructTag(`json:"ipAddress"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
	}
}

func (AuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("audit_logs").Field("user_id").Unique(),
	}
}
