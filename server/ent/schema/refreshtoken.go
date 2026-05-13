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

// RefreshToken holds the schema definition for the RefreshToken entity.
type RefreshToken struct {
	ent.Schema
}

func (RefreshToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "refresh_tokens"},
	}
}

func (RefreshToken) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("user_id", uuid.UUID{}).
			StructTag(`json:"userId"`),
		field.String("token_hash").Unique().Sensitive(),
		field.Time("expires_at").
			StructTag(`json:"expiresAt"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("revoked_at").Optional().Nillable().
			StructTag(`json:"revokedAt,omitempty"`),
	}
}

func (RefreshToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("refresh_tokens").Field("user_id").Unique().Required(),
	}
}
