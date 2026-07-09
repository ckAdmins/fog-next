package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ckAdmins/fog-next/ent"
	"github.com/ckAdmins/fog-next/ent/storagenode"
	enttask "github.com/ckAdmins/fog-next/ent/task"
	"github.com/ckAdmins/fog-next/internal/api/middleware"
	"github.com/ckAdmins/fog-next/internal/api/response"
	"github.com/ckAdmins/fog-next/internal/plugins"
)

type Tasks struct {
db      *ent.Client
plugins *plugins.Registry
}

func NewTasks(db *ent.Client, reg *plugins.Registry) *Tasks { return &Tasks{db, reg} }

func (h *Tasks) List(w http.ResponseWriter, r *http.Request) {
q := r.URL.Query()
query := h.db.Task.Query().Limit(100)
if s := q.Get("state"); s != "" {
query = query.Where(enttask.StateEQ(enttask.State(s)))
}
if t := q.Get("type"); t != "" {
query = query.Where(enttask.TypeEQ(enttask.Type(t)))
}
tasks, err := query.All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(tasks))
}

func (h *Tasks) Get(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
task, err := h.db.Task.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "task")
return
}
response.InternalError(w)
return
}
response.OK(w, task)
}

func (h *Tasks) Create(w http.ResponseWriter, r *http.Request) {
claims := middleware.ClaimsFrom(r.Context())
var req struct {
HostID         uuid.UUID       `json:"hostId"`
Type           enttask.Type    `json:"type"`
Name           string          `json:"name"`
ImageID        *uuid.UUID      `json:"imageId"`
StorageGroupID *uuid.UUID      `json:"storageGroupId"`
IsGroup        bool            `json:"isGroup"`
IsShutdown     bool            `json:"isShutdown"`
}
if !response.Decode(w, r, &req) {
return
}
if req.HostID == uuid.Nil {
response.BadRequest(w, "hostId is required")
return
}
if req.Type == "" {
response.BadRequest(w, "type is required")
return
}

needsImage := req.Type == enttask.TypeDeploy ||
req.Type == enttask.TypeCapture ||
req.Type == enttask.TypeMulticast ||
req.Type == enttask.TypeDebugDeploy ||
req.Type == enttask.TypeDebugCapture

	if needsImage {
		host, err := h.db.Host.Get(r.Context(), req.HostID)
		if err != nil {
			if ent.IsNotFound(err) {
				response.NotFound(w, "host")
			} else {
				response.InternalError(w)
			}
			return
		}
		if req.ImageID == nil && host.ImageID != nil {
			req.ImageID = host.ImageID
		}
		if req.ImageID == nil {
			response.BadRequest(w, "host has no image assigned and no imageId provided")
			return
		}
		if req.StorageGroupID == nil {
			img, err := h.db.Image.Get(r.Context(), *req.ImageID)
			if err != nil {
				if ent.IsNotFound(err) {
					response.NotFound(w, "image")
				} else {
					response.InternalError(w)
				}
				return
			}
			req.StorageGroupID = img.StorageGroupID
		}
	}

	// Reject if host already has a queued or active task.
	existing, err := h.db.Task.Query().Where(
		enttask.HostIDEQ(req.HostID),
		enttask.StateIn(enttask.StateQueued, enttask.StateActive),
	).First(r.Context())
	if err == nil {
		response.Conflict(w, "host already has a "+string(existing.State)+" task ("+existing.ID.String()+")")
		return
	}
	if !ent.IsNotFound(err) {
		response.InternalError(w)
		return
	}

	// Resolve storage node synchronously (was async in the scheduler).
	var storageNodeID *uuid.UUID
	if req.StorageGroupID != nil {
		node, err := h.db.StorageNode.Query().Where(
			storagenode.StorageGroupIDEQ(*req.StorageGroupID),
			storagenode.IsMasterEQ(true),
		).First(r.Context())
		if err == nil {
			storageNodeID = &node.ID
		}
	}

	var createdBy string
if claims != nil {
createdBy = claims.Username
}

// Build a bare struct for plugin inspection before persisting.
tStruct := &ent.Task{
Name:           req.Name,
Type:           req.Type,
State:          enttask.StateQueued,
HostID:         req.HostID,
ImageID:        req.ImageID,
StorageGroupID: req.StorageGroupID,
IsGroup:        req.IsGroup,
IsShutdown:     req.IsShutdown,
CreatedBy:      createdBy,
}
if err := h.plugins.BeforeTaskCreate(r.Context(), tStruct); err != nil {
response.Error(w, http.StatusUnprocessableEntity, "plugin rejected task", err.Error())
return
}

	savedTask, err := h.db.Task.Create().
		SetName(tStruct.Name).
		SetType(tStruct.Type).
		SetState(enttask.StateQueued).
		SetHostID(tStruct.HostID).
		SetNillableImageID(tStruct.ImageID).
		SetNillableStorageGroupID(tStruct.StorageGroupID).
		SetNillableStorageNodeID(storageNodeID).
		SetIsGroup(tStruct.IsGroup).
		SetIsShutdown(tStruct.IsShutdown).
		SetCreatedBy(tStruct.CreatedBy).
		Save(r.Context())
if err != nil {
response.InternalError(w)
return
}

writeAudit(r.Context(), h.db, r, "create", "task", savedTask.ID.String(), string(savedTask.Type))
response.Created(w, savedTask)
}

func (h *Tasks) Cancel(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.Task.UpdateOneID(id).SetState(enttask.StateCanceled).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
writeAudit(r.Context(), h.db, r, "cancel", "task", id.String(), "")
response.NoContent(w)
}

type progressUpdate struct {
Percent          int   `json:"percent"`
BitsPerMinute    int64 `json:"bitsPerMinute"`
BytesTransferred int64 `json:"bytesTransferred"`
}

func (h *Tasks) UpdateProgress(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
t, err := h.db.Task.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "task")
return
}
response.InternalError(w)
return
}

var prog progressUpdate
if !response.Decode(w, r, &prog) {
return
}

newState := t.State
if prog.Percent >= 100 {
newState = enttask.StateComplete
} else if t.State == enttask.StateQueued {
newState = enttask.StateActive
}

saved, err := h.db.Task.UpdateOneID(id).
SetPercentComplete(prog.Percent).
SetBitsPerMinute(prog.BitsPerMinute).
SetBytesTransferred(prog.BytesTransferred).
SetState(newState).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}

if saved.State == enttask.StateComplete || saved.State == enttask.StateFailed {
_ = h.plugins.AfterTaskComplete(r.Context(), saved)
}
response.OK(w, saved)
}
