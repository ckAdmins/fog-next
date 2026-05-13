package handlers

import (
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"path/filepath"

"github.com/go-chi/chi/v5"
"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/api/response"
"github.com/nemvince/fog-next/internal/config"
)

type Images struct {
db  *ent.Client
cfg *config.StorageConfig
}

func NewImages(db *ent.Client, cfg *config.StorageConfig) *Images { return &Images{db, cfg} }

func (h *Images) List(w http.ResponseWriter, r *http.Request) {
images, err := h.db.Image.Query().Limit(100).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(images))
}

func (h *Images) Get(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
img, err := h.db.Image.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "image")
return
}
response.InternalError(w)
return
}
response.OK(w, img)
}

func (h *Images) Create(w http.ResponseWriter, r *http.Request) {
var req struct {
Name           string     `json:"name"`
Description    string     `json:"description"`
Path           string     `json:"path"`
IsEnabled      bool       `json:"isEnabled"`
ToReplicate    bool       `json:"toReplicate"`
StorageGroupID *uuid.UUID `json:"storageGroupId"`
}
if !response.Decode(w, r, &req) {
return
}
if req.Name == "" {
response.BadRequest(w, "name is required")
return
}
img, err := h.db.Image.Create().
SetName(req.Name).
SetDescription(req.Description).
SetPath(req.Path).
SetIsEnabled(req.IsEnabled).
SetToReplicate(req.ToReplicate).
SetNillableStorageGroupID(req.StorageGroupID).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, img)
}

func (h *Images) Update(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
existing, err := h.db.Image.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "image")
return
}
response.InternalError(w)
return
}
var req struct {
Name           string          `json:"name"`
Description    string          `json:"description"`
Path           string          `json:"path"`
IsEnabled      bool            `json:"isEnabled"`
ToReplicate    bool            `json:"toReplicate"`
StorageGroupID *uuid.UUID      `json:"storageGroupId"`
}
if !response.Decode(w, r, &req) {
return
}
// Preserve partition metadata — only the agent writes it.
updated, err := h.db.Image.UpdateOneID(id).
SetName(req.Name).
SetDescription(req.Description).
SetPath(req.Path).
SetIsEnabled(req.IsEnabled).
SetToReplicate(req.ToReplicate).
SetNillableStorageGroupID(req.StorageGroupID).
SetPartitions(existing.Partitions). // preserve
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, updated)
}

func (h *Images) Delete(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.Image.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

// Download streams the image archive to the client.
func (h *Images) Download(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
img, err := h.db.Image.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "image")
return
}
response.InternalError(w)
return
}

imgPath := filepath.Join(h.cfg.BasePath, filepath.Clean(img.Path))
f, err := os.Open(imgPath) // #nosec G304 — path is server-controlled
if err != nil {
if os.IsNotExist(err) {
response.NotFound(w, "image file")
return
}
response.InternalError(w)
return
}
defer f.Close()

w.Header().Set("Content-Type", "application/octet-stream")
w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(imgPath)))
http.ServeContent(w, r, filepath.Base(imgPath), img.UpdatedAt, f)
}

// Upload receives a streamed image archive and writes it to disk.
func (h *Images) Upload(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
img, err := h.db.Image.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "image")
return
}
response.InternalError(w)
return
}

maxBytes := h.cfg.MaxUploadBytes
if maxBytes <= 0 {
maxBytes = 100 << 30
}
r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

imgPath := filepath.Join(h.cfg.BasePath, filepath.Clean(img.Path))
if err := os.MkdirAll(filepath.Dir(imgPath), 0o750); err != nil {
response.InternalError(w)
return
}

tmp, err := os.CreateTemp(filepath.Dir(imgPath), ".upload-*")
if err != nil {
response.InternalError(w)
return
}
tmpName := tmp.Name()
defer func() { _ = os.Remove(tmpName) }()

n, err := io.Copy(tmp, r.Body)
tmp.Close()
if err != nil {
response.InternalError(w)
return
}

if err := os.Rename(tmpName, imgPath); err != nil {
response.InternalError(w)
return
}

_ = h.db.Image.UpdateOneID(img.ID).SetSizeBytes(n).Exec(r.Context())
response.OK(w, map[string]int64{"bytesReceived": n})
}

// SetPartitions is called by the boot API after a successful capture to
// record partition metadata (stored as raw JSON).
func (h *Images) SetPartitions(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
var raw json.RawMessage
if !response.Decode(w, r, &raw) {
return
}
if err := h.db.Image.UpdateOneID(id).SetPartitions(raw).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}
