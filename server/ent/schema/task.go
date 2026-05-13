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

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

func (Task) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tasks"},
	}
}

// TaskState enum values.
type TaskState string

const (
	TaskStateQueued   TaskState = "queued"
	TaskStateActive   TaskState = "active"
	TaskStateComplete TaskState = "complete"
	TaskStateFailed   TaskState = "failed"
	TaskStateCanceled TaskState = "canceled"
)

// TaskType enum values.
type TaskType string

const (
	TaskTypeDeploy        TaskType = "deploy"
	TaskTypeCapture       TaskType = "capture"
	TaskTypeMulticast     TaskType = "multicast"
	TaskTypeDebugDeploy   TaskType = "debug_deploy"
	TaskTypeDebugCapture  TaskType = "debug_capture"
	TaskTypeMemTest       TaskType = "memtest"
	TaskTypeWipe          TaskType = "wipe"
	TaskTypeDiskTest      TaskType = "disk_test"
	TaskTypeAVScan        TaskType = "av_scan"
	TaskTypeSnapinInstall TaskType = "snapin_install"
)

func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).Unique().Immutable().
			StructTag(`json:"id"`),
		field.String("name").Default("").
			StructTag(`json:"name"`),
		field.Enum("type").Values(
			string(TaskTypeDeploy), string(TaskTypeCapture), string(TaskTypeMulticast),
			string(TaskTypeDebugDeploy), string(TaskTypeDebugCapture),
			string(TaskTypeMemTest), string(TaskTypeWipe), string(TaskTypeDiskTest),
			string(TaskTypeAVScan), string(TaskTypeSnapinInstall),
		).StorageKey("type").
			StructTag(`json:"type"`),
		field.Enum("state").Values(
			string(TaskStateQueued), string(TaskStateActive), string(TaskStateComplete),
			string(TaskStateFailed), string(TaskStateCanceled),
		).Default(string(TaskStateQueued)).
			StructTag(`json:"state"`),
		field.UUID("host_id", uuid.UUID{}).
			StructTag(`json:"hostId"`),
		field.UUID("image_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"imageId,omitempty"`),
		field.UUID("storage_node_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"storageNodeId,omitempty"`),
		field.UUID("storage_group_id", uuid.UUID{}).Optional().Nillable().
			StructTag(`json:"storageGroupId,omitempty"`),
		field.Bool("is_group").Default(false).
			StructTag(`json:"isGroup"`),
		field.Bool("is_forced").Default(false).
			StructTag(`json:"isForced"`),
		field.Bool("is_shutdown").Default(false).
			StructTag(`json:"isShutdown"`),
		field.Int("percent_complete").Default(0).
			StructTag(`json:"percentComplete"`),
		field.Int64("bits_per_minute").Default(0).
			StructTag(`json:"bitsPerMinute"`),
		field.Int64("bytes_transferred").Default(0).
			StructTag(`json:"bytesTransferred"`),
		field.Time("scheduled_at").Optional().Nillable().
			StructTag(`json:"scheduledAt,omitempty"`),
		field.Time("started_at").Optional().Nillable().
			StructTag(`json:"startedAt,omitempty"`),
		field.Time("completed_at").Optional().Nillable().
			StructTag(`json:"completedAt,omitempty"`),
		field.Time("created_at").Default(time.Now).Immutable().
			StructTag(`json:"createdAt"`),
		field.String("created_by").Default("").
			StructTag(`json:"createdBy"`),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			StructTag(`json:"updatedAt"`),
	}
}

func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("host", Host.Type).Ref("tasks").Field("host_id").Unique().Required(),
		edge.From("image", Image.Type).Ref("tasks").Field("image_id").Unique(),
		edge.From("storage_node", StorageNode.Type).Ref("tasks").Field("storage_node_id").Unique(),
		edge.From("storage_group", StorageGroup.Type).Ref("tasks").Field("storage_group_id").Unique(),
		edge.To("imaging_log", ImagingLog.Type).Unique(),
		edge.To("agent_logs", AgentLog.Type),
	}
}
