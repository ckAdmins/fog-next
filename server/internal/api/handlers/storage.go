package handlers

import (
"net/http"

"github.com/go-chi/chi/v5"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/storagenode"
"github.com/nemvince/fog-next/internal/api/response"
)

type storageGroupResponse struct {
*ent.StorageGroup
Nodes []*ent.StorageNode `json:"nodes"`
}

type Storage struct{ db *ent.Client }

func NewStorage(db *ent.Client) *Storage { return &Storage{db} }

func (h *Storage) ListGroups(w http.ResponseWriter, r *http.Request) {
groups, err := h.db.StorageGroup.Query().All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(groups))
}

func (h *Storage) GetGroup(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
sg, err := h.db.StorageGroup.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "storage group")
return
}
response.InternalError(w)
return
}
nodes, _ := h.db.StorageNode.Query().Where(storagenode.StorageGroupIDEQ(id)).All(r.Context())
response.OK(w, storageGroupResponse{StorageGroup: sg, Nodes: nodes})
}

func (h *Storage) CreateGroup(w http.ResponseWriter, r *http.Request) {
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
}
if !response.Decode(w, r, &req) {
return
}
sg, err := h.db.StorageGroup.Create().
SetName(req.Name).
SetDescription(req.Description).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, sg)
}

func (h *Storage) UpdateGroup(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if _, err := h.db.StorageGroup.Get(r.Context(), id); err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "storage group")
return
}
response.InternalError(w)
return
}
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
}
if !response.Decode(w, r, &req) {
return
}
updated, err := h.db.StorageGroup.UpdateOneID(id).
SetName(req.Name).
SetDescription(req.Description).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, updated)
}

func (h *Storage) DeleteGroup(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.StorageGroup.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

func (h *Storage) ListNodes(w http.ResponseWriter, r *http.Request) {
groupID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
nodes, err := h.db.StorageNode.Query().Where(storagenode.StorageGroupIDEQ(groupID)).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(nodes))
}

func (h *Storage) GetNode(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
node, err := h.db.StorageNode.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "storage node")
return
}
response.InternalError(w)
return
}
response.OK(w, node)
}

func (h *Storage) CreateNode(w http.ResponseWriter, r *http.Request) {
groupID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
Hostname    string `json:"hostname"`
RootPath    string `json:"rootPath"`
IsEnabled   bool   `json:"isEnabled"`
IsMaster    bool   `json:"isMaster"`
MaxClients  int    `json:"maxClients"`
SSHUser     string `json:"sshUser"`
WebRoot     string `json:"webRoot"`
}
if !response.Decode(w, r, &req) {
return
}
sn, err := h.db.StorageNode.Create().
SetName(req.Name).
SetDescription(req.Description).
SetStorageGroupID(groupID).
SetHostname(req.Hostname).
SetRootPath(req.RootPath).
SetIsEnabled(req.IsEnabled).
SetIsMaster(req.IsMaster).
SetMaxClients(req.MaxClients).
SetSSHUser(req.SSHUser).
SetWebRoot(req.WebRoot).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, sn)
}

func (h *Storage) UpdateNode(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if _, err := h.db.StorageNode.Get(r.Context(), id); err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "storage node")
return
}
response.InternalError(w)
return
}
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
Hostname    string `json:"hostname"`
RootPath    string `json:"rootPath"`
IsEnabled   bool   `json:"isEnabled"`
IsMaster    bool   `json:"isMaster"`
MaxClients  int    `json:"maxClients"`
SSHUser     string `json:"sshUser"`
WebRoot     string `json:"webRoot"`
}
if !response.Decode(w, r, &req) {
return
}
updated, err := h.db.StorageNode.UpdateOneID(id).
SetName(req.Name).
SetDescription(req.Description).
SetHostname(req.Hostname).
SetRootPath(req.RootPath).
SetIsEnabled(req.IsEnabled).
SetIsMaster(req.IsMaster).
SetMaxClients(req.MaxClients).
SetSSHUser(req.SSHUser).
SetWebRoot(req.WebRoot).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, updated)
}

func (h *Storage) DeleteNode(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.StorageNode.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}
