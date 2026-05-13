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

// Inventory holds the schema definition for the Inventory entity.
type Inventory struct {
	ent.Schema
}

func (Inventory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "inventory"},
	}
}

func (Inventory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.UUID("host_id", uuid.UUID{}).Unique().
			StructTag(`json:"hostId"`),
		field.String("cpu_model").Default("").
			StructTag(`json:"cpuModel"`),
		field.Int("cpu_cores").Default(0).
			StructTag(`json:"cpuCores"`),
		field.Int("cpu_freq_mhz").Default(0).
			StructTag(`json:"cpuFreqMhz"`),
		field.Int("ram_mib").Default(0).
			StructTag(`json:"ramMib"`),
		field.String("hd_model").Default("").
			StructTag(`json:"hdModel"`),
		field.Int("hd_size_gb").Default(0).
			StructTag(`json:"hdSizeGb"`),
		field.String("manufacturer").Default("").
			StructTag(`json:"manufacturer"`),
		field.String("product").Default("").
			StructTag(`json:"product"`),
		field.String("serial").Default("").
			StructTag(`json:"serial"`),
		field.String("uuid").Default("").
			StructTag(`json:"uuid"`),
		field.String("bios_version").Default("").
			StructTag(`json:"biosVersion"`),
		field.String("primary_mac").Default("").
			StructTag(`json:"primaryMac"`),
		field.String("os_name").Default("").
			StructTag(`json:"osName"`),
		field.String("os_version").Default("").
			StructTag(`json:"osVersion"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Inventory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).Ref("inventory").Field("host_id").Unique().Required(),
	}
}
