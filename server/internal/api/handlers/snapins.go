package handlers

import (
"fmt"
"io"
"net/http"
"os"
"path/filepath"
"strings"

"github.com/go-chi/chi/v5"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/api/response"
"github.com/nemvince/fog-next/internal/config"
)

var allowedSnapinMIME = map[string]bool{
"application/octet-stream":              true,
"application/x-sh":                      true,
"application/x-bat":                     true,
"application/x-msdos-program":           true,
"application/zip":                       true,
"application/x-tar":                     true,
"application/gzip":                      true,
"application/x-bzip2":                   true,
"application/x-xz":                      true,
"application/vnd.debian.binary-package": true,
"application/x-rpm":                     true,
"text/x-shellscript":                    true,
"text/plain":                             true,
}

type Snapins struct {
db         *ent.Client
snapinPath string
}

func NewSnapins(cfg *config.Config, db *ent.Client) *Snapins {
path := cfg.Storage.SnapinPath
if path == "" {
path = "/opt/fog/snapins"
}
return &Snapins{db: db, snapinPath: path}
}

func (h *Snapins) List(w http.ResponseWriter, r *http.Request) {
snapins, err := h.db.Snapin.Query().Limit(100).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(snapins))
}

func (h *Snapins) Get(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
sn, err := h.db.Snapin.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "snapin")
return
}
response.InternalError(w)
return
}
response.OK(w, sn)
}

func (h *Snapins) Create(w http.ResponseWriter, r *http.Request) {
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
Command     string `json:"command"`
Arguments   string `json:"arguments"`
RunWith     string `json:"runWith"`
IsEnabled   bool   `json:"isEnabled"`
ToReplicate bool   `json:"toReplicate"`
}
if !response.Decode(w, r, &req) {
return
}
if req.Name == "" {
response.BadRequest(w, "name is required")
return
}
sn, err := h.db.Snapin.Create().
SetName(req.Name).
SetDescription(req.Description).
SetCommand(req.Command).
SetArguments(req.Arguments).
SetRunWith(req.RunWith).
SetIsEnabled(req.IsEnabled).
SetToReplicate(req.ToReplicate).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.Created(w, sn)
}

func (h *Snapins) Update(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if _, err := h.db.Snapin.Get(r.Context(), id); err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "snapin")
return
}
response.InternalError(w)
return
}
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
Command     string `json:"command"`
Arguments   string `json:"arguments"`
RunWith     string `json:"runWith"`
IsEnabled   bool   `json:"isEnabled"`
ToReplicate bool   `json:"toReplicate"`
}
if !response.Decode(w, r, &req) {
return
}
updated, err := h.db.Snapin.UpdateOneID(id).
SetName(req.Name).
SetDescription(req.Description).
SetCommand(req.Command).
SetArguments(req.Arguments).
SetRunWith(req.RunWith).
SetIsEnabled(req.IsEnabled).
SetToReplicate(req.ToReplicate).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, updated)
}

func (h *Snapins) Delete(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.Snapin.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

// Upload accepts a multipart file upload for a snapin and stores it on disk.
func (h *Snapins) Upload(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
sn, err := h.db.Snapin.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "snapin")
return
}
response.InternalError(w)
return
}

if err := r.ParseMultipartForm(32 << 20); err != nil {
response.BadRequest(w, "could not parse multipart form")
return
}

file, header, err := r.FormFile("file")
if err != nil {
response.BadRequest(w, "file field is required")
return
}
defer file.Close()

sniff := make([]byte, 512)
n, _ := file.Read(sniff)
detectedMIME := http.DetectContentType(sniff[:n])
baseMIME := strings.TrimSpace(strings.SplitN(detectedMIME, ";", 2)[0])
if !allowedSnapinMIME[baseMIME] {
response.BadRequest(w, fmt.Sprintf("file type not allowed: %s", baseMIME))
return
}

safeName := filepath.Base(filepath.Clean(header.Filename))
if safeName == "." || safeName == ".." || safeName == "" || strings.ContainsAny(safeName, "/\\") {
response.BadRequest(w, "invalid filename")
return
}

destDir := filepath.Join(h.snapinPath, id.String())
if err := os.MkdirAll(destDir, 0750); err != nil {
response.InternalError(w)
return
}

destPath := filepath.Join(destDir, safeName)
if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)+string(os.PathSeparator)) {
response.BadRequest(w, "invalid filename")
return
}

out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
if err != nil {
response.InternalError(w)
return
}
defer out.Close()

body := io.MultiReader(strings.NewReader(string(sniff[:n])), file)
size, err := io.Copy(out, body)
if err != nil {
response.InternalError(w)
return
}

_, err = h.db.Snapin.UpdateOne(sn).
SetFileName(safeName).
SetFilePath(destPath).
SetSizeBytes(size).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, sn)
}
