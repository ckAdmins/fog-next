package handlers

import (
"net/http"

"github.com/go-chi/chi/v5"
"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/groupmember"
"github.com/nemvince/fog-next/internal/api/response"
)

type Groups struct{ db *ent.Client }

func NewGroups(db *ent.Client) *Groups { return &Groups{db} }

func (h *Groups) List(w http.ResponseWriter, r *http.Request) {
groups, err := h.db.Group.Query().Limit(100).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(groups))
}

func (h *Groups) Get(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
g, err := h.db.Group.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "group")
return
}
response.InternalError(w)
return
}
response.OK(w, g)
}

func (h *Groups) Create(w http.ResponseWriter, r *http.Request) {
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
}
if !response.Decode(w, r, &req) {
return
}
if req.Name == "" {
response.BadRequest(w, "name is required")
return
}
g, err := h.db.Group.Create().SetName(req.Name).SetDescription(req.Description).Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, g)
}

func (h *Groups) Update(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if _, err := h.db.Group.Get(r.Context(), id); err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "group")
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
g, err := h.db.Group.UpdateOneID(id).SetName(req.Name).SetDescription(req.Description).Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, g)
}

func (h *Groups) Delete(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.Group.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

func (h *Groups) ListMembers(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
members, err := h.db.GroupMember.Query().Where(groupmember.GroupIDEQ(id)).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(members))
}

func (h *Groups) AddMember(w http.ResponseWriter, r *http.Request) {
groupID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
var body struct {
HostID uuid.UUID `json:"hostId"`
}
if !response.Decode(w, r, &body) {
return
}
gm, err := h.db.GroupMember.Create().SetGroupID(groupID).SetHostID(body.HostID).Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, gm)
}

func (h *Groups) RemoveMember(w http.ResponseWriter, r *http.Request) {
groupID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
hostID, ok := parseUUID(w, chi.URLParam(r, "hostId"))
if !ok {
return
}
if _, err := h.db.GroupMember.Delete().Where(
groupmember.GroupIDEQ(groupID),
groupmember.HostIDEQ(hostID),
).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}
