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

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users"},
	}
}

// UserRole enum values.
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleMobile   UserRole = "mobile"
	UserRoleReadOnly UserRole = "readonly"
)

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("username").NotEmpty().Unique().
			StructTag(`json:"username"`),
		field.String("password_hash").Sensitive(),
		field.Enum("role").Values(
			string(UserRoleAdmin), string(UserRoleMobile), string(UserRoleReadOnly),
		).Default(string(UserRoleReadOnly)).
			StructTag(`json:"role"`),
		field.String("email").Default("").
			StructTag(`json:"email"`),
		field.Bool("is_active").Default(true).
			StructTag(`json:"isActive"`),
		field.String("api_token").Default("").Sensitive(),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.String("created_by").Default("").
			StructTag(`json:"createdBy"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
		field.Time("last_login_at").Optional().Nillable().
			StructTag(`json:"lastLoginAt,omitempty"`),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("refresh_tokens", RefreshToken.Type),
		edge.To("audit_logs", AuditLog.Type),
	}
}
