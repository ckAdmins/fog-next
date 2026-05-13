package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// GlobalSetting holds the schema definition for the GlobalSetting entity.
type GlobalSetting struct {
	ent.Schema
}

func (GlobalSetting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "global_settings"},
	}
}

func (GlobalSetting) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("key").NotEmpty().Unique().
			StructTag(`json:"key"`),
		field.String("value").Default("").
			StructTag(`json:"value"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.String("category").Default("").
			StructTag(`json:"category"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}
