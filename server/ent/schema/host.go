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

// Host holds the schema definition for the Host entity.
type Host struct {
	ent.Schema
}

func (Host) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hosts"},
	}
}

func (Host) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").NotEmpty().
			StructTag(`json:"name"`),
		field.String("description").Default("").
			StructTag(`json:"description"`),
		field.String("ip").Default("").
			StructTag(`json:"ip"`),
		field.UUID("image_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"imageId,omitempty"`),
		field.String("kernel").Default("").
			StructTag(`json:"kernel"`),
		field.String("init").Default("").
			StructTag(`json:"init"`),
		field.String("kernel_args").Default("").
			StructTag(`json:"kernelArgs"`),
		field.Bool("is_enabled").Default(true).
			StructTag(`json:"isEnabled"`),
		field.Bool("use_aad").Default(false).
			StructTag(`json:"useAad"`),
		field.Bool("use_wol").Default(false).
			StructTag(`json:"useWol"`),
		field.Time("last_contact").Optional().Nillable().
			StructTag(`json:"lastContact,omitempty"`),
		field.Time("deployed_at").Optional().Nillable().
			StructTag(`json:"deployedAt,omitempty"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Host) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("image", Image.Type).Ref("hosts").Field("image_id").Unique(),
		edge.To("macs", HostMAC.Type),
		edge.To("inventory", Inventory.Type).Unique(),
		edge.To("pending_macs", PendingMAC.Type),
		edge.To("group_members", GroupMember.Type),
		edge.To("snapin_assocs", SnapinAssoc.Type),
		edge.To("tasks", Task.Type),
		edge.To("imaging_logs", ImagingLog.Type),
		edge.To("snapin_jobs", SnapinJob.Type),
		edge.To("agent_logs", AgentLog.Type),
	}
}
