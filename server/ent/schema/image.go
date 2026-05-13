package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Image holds the schema definition for the Image entity.
type Image struct {
	ent.Schema
}

func (Image) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "images"},
	}
}

func (Image) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.String("path").Default("").
			StructTag(`json:"path"`),
		field.UUID("os_type_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"osTypeId,omitempty"`),
		field.UUID("image_type_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"imageTypeId,omitempty"`),
		field.UUID("storage_group_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"storageGroupId,omitempty"`),
		field.Bool("is_enabled").Default(true).
			StructTag(`json:"isEnabled"`),
		field.Bool("to_replicate").Default(false).
			StructTag(`json:"toReplicate"`),
		field.Int64("size_bytes").Default(0).
			StructTag(`json:"sizeBytes"`),
		field.JSON("partitions", json.RawMessage{}).Optional().
			StructTag(`json:"partitions,omitempty"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.String("created_by").Default("").
			StructTag(`json:"createdBy"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Image) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("os_type", OSType.Type).Ref("images").Field("os_type_id").Unique(),
		edge.From("image_type", ImageType.Type).Ref("images").Field("image_type_id").Unique(),
		edge.From("storage_group", StorageGroup.Type).Ref("images").Field("storage_group_id").Unique(),
		edge.To("hosts", Host.Type),
		edge.To("tasks", Task.Type),
		edge.To("imaging_logs", ImagingLog.Type),
		edge.To("multicast_sessions", MulticastSession.Type),
	}
}
