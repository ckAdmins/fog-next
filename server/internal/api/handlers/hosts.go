package handlers

import (
"net/http"

"github.com/go-chi/chi/v5"
"github.com/google/uuid"
"github.com/ckAdmins/fog-next/ent"
enthost "github.com/ckAdmins/fog-next/ent/host"
"github.com/ckAdmins/fog-next/ent/hostmac"
enttask "github.com/ckAdmins/fog-next/ent/task"
"github.com/ckAdmins/fog-next/internal/api/response"
"github.com/ckAdmins/fog-next/internal/plugins"
)

type Hosts struct {
db      *ent.Client
plugins *plugins.Registry
}

func NewHosts(db *ent.Client, reg *plugins.Registry) *Hosts { return &Hosts{db, reg} }

type hostResponse struct {
*ent.Host
MACs []*ent.HostMAC `json:"macs"`
}

func (h *Hosts) List(w http.ResponseWriter, r *http.Request) {
q := r.URL.Query()
search := q.Get("q")
query := h.db.Host.Query().Limit(50)
if search != "" {
query = query.Where(enthost.NameContainsFold(search))
}
if c := q.Get("cursor"); c != "" {
if id, err := uuid.Parse(c); err == nil {
query = query.Where(enthost.IDGT(id))
}
}
hosts, err := query.All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(hosts))
}

func (h *Hosts) Get(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
host, err := h.db.Host.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "host")
return
}
response.InternalError(w)
return
}
macs, _ := h.db.HostMAC.Query().Where(hostmac.HostIDEQ(id)).All(r.Context())
response.OK(w, hostResponse{Host: host, MACs: macs})
}

func (h *Hosts) Create(w http.ResponseWriter, r *http.Request) {
var req struct {
Name        string     `json:"name"`
Description string     `json:"description"`
IP          string     `json:"ip"`
ImageID     *uuid.UUID `json:"imageId"`
KernelArgs  string     `json:"kernelArgs"`
IsEnabled   bool       `json:"isEnabled"`
}
if !response.Decode(w, r, &req) {
return
}
if req.Name == "" {
response.BadRequest(w, "name is required")
return
}
savedHost, err := h.db.Host.Create().
SetName(req.Name).
SetDescription(req.Description).
SetNillableIP(&req.IP).
SetNillableImageID(req.ImageID).
SetKernelArgs(req.KernelArgs).
SetIsEnabled(req.IsEnabled).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
if err := h.plugins.OnHostRegister(r.Context(), savedHost); err != nil {
_ = h.db.Host.DeleteOneID(savedHost.ID).Exec(r.Context())
response.Error(w, http.StatusUnprocessableEntity, "plugin rejected host", err.Error())
return
}
response.Created(w, savedHost)
}

func (h *Hosts) Update(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if _, err := h.db.Host.Get(r.Context(), id); err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "host")
return
}
response.InternalError(w)
return
}
var req struct {
Name        string     `json:"name"`
Description string     `json:"description"`
IP          *string    `json:"ip"`
ImageID     *uuid.UUID `json:"imageId"`
KernelArgs  string     `json:"kernelArgs"`
IsEnabled   bool       `json:"isEnabled"`
UseWol      bool       `json:"useWol"`
}
if !response.Decode(w, r, &req) {
return
}
updated, err := h.db.Host.UpdateOneID(id).
SetName(req.Name).
SetDescription(req.Description).
SetNillableIP(req.IP).
SetNillableImageID(req.ImageID).
SetKernelArgs(req.KernelArgs).
SetIsEnabled(req.IsEnabled).
SetUseWol(req.UseWol).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, updated)
}

func (h *Hosts) Delete(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.Host.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

func (h *Hosts) ListMACs(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
macs, err := h.db.HostMAC.Query().Where(hostmac.HostIDEQ(id)).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(macs))
}

func (h *Hosts) AddMAC(w http.ResponseWriter, r *http.Request) {
hostID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
var req struct {
MAC       string `json:"mac"`
IsPrimary bool   `json:"isPrimary"`
}
if !response.Decode(w, r, &req) {
return
}
if req.MAC == "" {
response.BadRequest(w, "mac is required")
return
}
if req.MAC == "00:00:00:00:00:00" {
response.BadRequest(w, "invalid mac address")
return
}
mac, err := h.db.HostMAC.Create().
SetHostID(hostID).
SetMAC(req.MAC).
SetIsPrimary(req.IsPrimary).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, mac)
}

func (h *Hosts) DeleteMAC(w http.ResponseWriter, r *http.Request) {
macID, ok := parseUUID(w, chi.URLParam(r, "macId"))
if !ok {
return
}
if err := h.db.HostMAC.DeleteOneID(macID).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

func (h *Hosts) GetInventory(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
inv, err := h.db.Host.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "host")
return
}
response.InternalError(w)
return
}
inventory, err := inv.QueryInventory().Only(r.Context())
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "inventory")
return
}
response.InternalError(w)
return
}
response.OK(w, inventory)
}

func (h *Hosts) GetActiveTask(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
	task, err := h.db.Task.Query().Where(
		enttask.HostIDEQ(id),
		enttask.StateIn(
			enttask.StateQueued,
			enttask.StateActive,
			enttask.StateComplete,
			enttask.StateFailed,
			enttask.StateCanceled,
		),
	).Order(enttask.ByCreatedAt()).First(r.Context())
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

func (h *Hosts) ListPendingMACs(w http.ResponseWriter, r *http.Request) {
macs, err := h.db.PendingMAC.Query().Limit(200).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(macs))
}
