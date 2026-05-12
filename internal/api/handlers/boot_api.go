package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nemvince/fog-next/ent"
	enthost "github.com/nemvince/fog-next/ent/host"
	"github.com/nemvince/fog-next/ent/hostmac"
	"github.com/nemvince/fog-next/ent/inventory"
	"github.com/nemvince/fog-next/ent/storagenode"
	enttask "github.com/nemvince/fog-next/ent/task"
	"github.com/nemvince/fog-next/internal/api/middleware"
	"github.com/nemvince/fog-next/internal/api/response"
	fogauth "github.com/nemvince/fog-next/internal/auth"
	"github.com/nemvince/fog-next/internal/config"
	"github.com/nemvince/fog-next/internal/ws"
)

// imagePartitions is the JSONB shape stored in Image.Partitions.
type imagePartitions struct {
	PartCount           int      `json:"partCount"`
	ImageType           string   `json:"imageType"`
	FixedSizePartitions []int    `json:"fixedSizePartitions"`
	PartTypes           []string `json:"partTypes,omitempty"`
}

// BootAPI handles unauthenticated and boot-token-authenticated endpoints
// under /fog/api/v1/boot/.
type BootAPI struct {
cfg        *config.Config
db         *ent.Client
hub        *ws.Hub
httpClient *http.Client
}

func NewBootAPI(cfg *config.Config, db *ent.Client, hub *ws.Hub) *BootAPI {
return &BootAPI{
cfg: cfg,
db:  db,
hub: hub,
httpClient: &http.Client{
Timeout: 0, // no timeout — image streams can be large
},
}
}

// ------------------------------------------------------------------
// Handshake — POST /fog/api/v1/boot/handshake  (unauthenticated)
// ------------------------------------------------------------------

type handshakeRequest struct {
MACs []string `json:"macs"`
}

type handshakeResponse struct {
	BootToken           string   `json:"bootToken"`
	TaskID              string   `json:"taskId"`
	Action              string   `json:"action"`
	ImageID             string   `json:"imageId,omitempty"`
	PartCount           int      `json:"partCount,omitempty"`
	TotalBytes          int64    `json:"totalBytes,omitempty"`
	StorageNodeURL      string   `json:"storageNodeUrl,omitempty"`
	ImageType           string   `json:"imageType,omitempty"`
	FixedSizePartitions []int    `json:"fixedSizePartitions,omitempty"`
	PartTypes           []string `json:"partTypes,omitempty"`
}

func (h *BootAPI) Handshake(w http.ResponseWriter, r *http.Request) {
var req handshakeRequest
if !response.Decode(w, r, &req) {
return
}
if len(req.MACs) == 0 {
response.BadRequest(w, "macs is required")
return
}

// Look up host by any of the supplied MACs.
var host *ent.Host
for _, mac := range req.MACs {
normed := normMAC(mac)
if isZeroMAC(normed) {
continue
}
h2, err := h.db.HostMAC.Query().Where(hostmac.MACEQ(normed)).QueryHost().Only(r.Context())
if err == nil {
host = h2
break
}
if !ent.IsNotFound(err) {
slog.Error("handshake host lookup error", "err", err)
response.InternalError(w)
return
}
}

// Unknown host — record the MACs as pending and request registration.
if host == nil {
for _, mac := range req.MACs {
normed := normMAC(mac)
if isZeroMAC(normed) {
continue
}
_ = h.db.PendingMAC.Create().SetMAC(normed).SetSeenAt(time.Now()).Exec(r.Context())
}
response.OK(w, handshakeResponse{Action: "register"})
return
}

// Update last-contact timestamp.
_ = h.db.Host.UpdateOneID(host.ID).SetLastContact(time.Now()).Exec(r.Context())

// Look for an active task.
task, err := h.db.Task.Query().Where(
enttask.HostIDEQ(host.ID),
enttask.StateIn(enttask.StateQueued, enttask.StateActive),
).Order(enttask.ByCreatedAt()).First(r.Context())
if err != nil {
if ent.IsNotFound(err) {
response.OK(w, handshakeResponse{Action: "idle"})
return
}
slog.Error("handshake task lookup error", "err", err)
response.InternalError(w)
return
}

// Issue a boot token scoped to this task.
bootToken, err := fogauth.IssueBootToken(h.cfg.Auth, task.ID, host.ID, string(task.Type))
if err != nil {
slog.Error("boot token issuance failed", "err", err)
response.InternalError(w)
return
}

resp := handshakeResponse{
BootToken: bootToken,
TaskID:    task.ID.String(),
Action:    string(task.Type),
}

	if task.ImageID != nil {
		resp.ImageID = task.ImageID.String()
		img, imgErr := h.db.Image.Get(r.Context(), *task.ImageID)
		if imgErr == nil {
			resp.TotalBytes = img.SizeBytes
			if ip := parseImagePartitions(img); ip != nil {
				resp.PartCount = ip.PartCount
				resp.ImageType = ip.ImageType
				resp.FixedSizePartitions = ip.FixedSizePartitions
				resp.PartTypes = ip.PartTypes
			}
			if task.StorageNodeID != nil {
				node, nodeErr := h.db.StorageNode.Get(r.Context(), *task.StorageNodeID)
				if nodeErr == nil && node.IsEnabled {
					resp.StorageNodeURL = buildStorageNodeURL(node)
				}
			}
		}
	}

// Transition task to active.
now := time.Now()
_ = h.db.Task.UpdateOneID(task.ID).
SetState(enttask.StateActive).
SetStartedAt(now).
Exec(r.Context())

response.OK(w, resp)
}

// ------------------------------------------------------------------
// Register — POST /fog/api/v1/boot/register  (unauthenticated)
// ------------------------------------------------------------------

type registerRequest struct {
MACs      []string `json:"macs"`
CPUModel  string   `json:"cpuModel"`
CPUCores  int      `json:"cpuCores"`
RAMBytes  int64    `json:"ramBytes"`
DiskBytes int64    `json:"diskBytes"`
UUID      string   `json:"uuid,omitempty"`
}

func (h *BootAPI) Register(w http.ResponseWriter, r *http.Request) {
var req registerRequest
if !response.Decode(w, r, &req) {
return
}
if len(req.MACs) == 0 {
response.BadRequest(w, "macs is required")
return
}

// Check if host already exists for any MAC.
for _, mac := range req.MACs {
normed := normMAC(mac)
if isZeroMAC(normed) {
continue
}
existing, err := h.db.HostMAC.Query().Where(hostmac.MACEQ(normed)).QueryHost().Only(r.Context())
if err == nil {
_ = h.db.Inventory.Create().
SetHostID(existing.ID).
SetCPUModel(req.CPUModel).
SetCPUCores(req.CPUCores).
SetRAMMib(int(req.RAMBytes / 1024 / 1024)).
SetHdSizeGB(int(req.DiskBytes / 1024 / 1024 / 1024)).
SetUUID(req.UUID).
OnConflictColumns(inventory.FieldHostID).
UpdateNewValues().
Exec(r.Context())
response.OK(w, map[string]string{"status": "already_registered"})
return
}
}

// Collect valid (non-zero) MACs.
var validMACs []string
for _, mac := range req.MACs {
normed := normMAC(mac)
if !isZeroMAC(normed) {
validMACs = append(validMACs, normed)
}
}
if len(validMACs) == 0 {
response.BadRequest(w, "no valid macs provided")
return
}

// Create a new disabled host with the first MAC as primary.
newHost, err := h.db.Host.Create().
SetName("pending-" + strings.ReplaceAll(validMACs[0], ":", "")).
SetIsEnabled(false).
Save(r.Context())
if err != nil {
slog.Error("register: create host failed", "err", err)
response.InternalError(w)
return
}

for i, mac := range validMACs {
if err := h.db.HostMAC.Create().
SetHostID(newHost.ID).
SetMAC(mac).
SetIsPrimary(i == 0).
Exec(r.Context()); err != nil {
slog.Warn("register: add MAC failed", "mac", mac, "err", err)
}
}

_ = h.db.Inventory.Create().
SetHostID(newHost.ID).
SetCPUModel(req.CPUModel).
SetCPUCores(req.CPUCores).
SetRAMMib(int(req.RAMBytes / 1024 / 1024)).
SetHdSizeGB(int(req.DiskBytes / 1024 / 1024 / 1024)).
SetUUID(req.UUID).
OnConflictColumns(inventory.FieldHostID).
UpdateNewValues().
Exec(r.Context())

response.OK(w, map[string]string{"status": "registered", "hostId": newHost.ID.String()})
}

// ------------------------------------------------------------------
// Progress — POST /fog/api/v1/boot/progress  (boot-token auth)
// ------------------------------------------------------------------

type progressRequest struct {
TaskID           string `json:"taskId"`
Percent          int    `json:"percent"`
BitsPerMinute    int64  `json:"bitsPerMinute"`
BytesTransferred int64  `json:"bytesTransferred"`
}

func (h *BootAPI) Progress(w http.ResponseWriter, r *http.Request) {
claims := middleware.BootClaimsFrom(r.Context())
if claims == nil {
response.Unauthorized(w)
return
}
var req progressRequest
if !response.Decode(w, r, &req) {
return
}

taskID, err := uuid.Parse(req.TaskID)
if err != nil || taskID != claims.TaskID {
response.BadRequest(w, "task ID mismatch")
return
}

if err := h.db.Task.UpdateOneID(taskID).
SetPercentComplete(req.Percent).
SetBitsPerMinute(req.BitsPerMinute).
SetBytesTransferred(req.BytesTransferred).
Exec(r.Context()); err != nil {
slog.Warn("progress update failed", "task", taskID, "err", err)
}

response.NoContent(w)
}

// ------------------------------------------------------------------
// Complete — POST /fog/api/v1/boot/complete  (boot-token auth)
// ------------------------------------------------------------------

type completeRequest struct {
TaskID  string `json:"taskId"`
Success bool   `json:"success"`
Message string `json:"message,omitempty"`
}

func (h *BootAPI) Complete(w http.ResponseWriter, r *http.Request) {
claims := middleware.BootClaimsFrom(r.Context())
if claims == nil {
response.Unauthorized(w)
return
}
var req completeRequest
if !response.Decode(w, r, &req) {
return
}

taskID, err := uuid.Parse(req.TaskID)
if err != nil || taskID != claims.TaskID {
response.BadRequest(w, "task ID mismatch")
return
}

	t, err := h.db.Task.Get(r.Context(), taskID)
	if err != nil {
		response.NotFound(w, "task")
		return
	}

	now := time.Now()
	newState := enttask.StateFailed
	pct := t.PercentComplete
	if req.Success {
		newState = enttask.StateComplete
		pct = 100
	}

	_ = h.db.Task.UpdateOneID(taskID).
		SetState(newState).
		SetPercentComplete(pct).
		SetCompletedAt(now).
		Exec(r.Context())

	// On capture failure, clean up partial partition files.
	if !req.Success && t.Type == enttask.TypeCapture && t.ImageID != nil {
		h.cleanupImageParts(*t.ImageID)
	}

	// Record imaging log entry.
var duration int64
if t.StartedAt != nil {
duration = int64(now.Sub(*t.StartedAt).Seconds())
}
_ = h.db.ImagingLog.Create().
SetHostID(claims.HostID).
SetTaskID(taskID).
SetTaskType(string(t.Type)).
SetNillableImageID(t.ImageID).
SetSizeBytes(t.BytesTransferred).
SetDuration(duration).
Exec(r.Context())

response.NoContent(w)
}

// ------------------------------------------------------------------
// Download — GET /fog/api/v1/boot/images/{id}/download?part=N  (boot-token auth)
// ------------------------------------------------------------------

func (h *BootAPI) Download(w http.ResponseWriter, r *http.Request) {
claims := middleware.BootClaimsFrom(r.Context())
if claims == nil {
response.Unauthorized(w)
return
}

imageID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}

task, err := h.db.Task.Get(r.Context(), claims.TaskID)
if err != nil || task.ImageID == nil || *task.ImageID != imageID {
response.Forbidden(w)
return
}

partStr := r.URL.Query().Get("part")
partNum, parseErr := strconv.Atoi(partStr)
if parseErr != nil {
response.BadRequest(w, "part parameter must be an integer")
return
}

img, err := h.db.Image.Get(r.Context(), imageID)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "image")
return
}
response.InternalError(w)
return
}

nodeURL, resolveErr := h.resolveStorageNodeURL(r, task, img)
if resolveErr != nil {
h.downloadLocal(w, r, img, partNum)
return
}

upstreamURL := fmt.Sprintf("%s/%s/%s", strings.TrimRight(nodeURL, "/"), img.Path, partFilename(partNum))
h.proxyGet(w, r, upstreamURL)
}

// ------------------------------------------------------------------
// Upload — PUT /fog/api/v1/boot/images/{id}/upload?part=N  (boot-token auth)
// ------------------------------------------------------------------

func (h *BootAPI) Upload(w http.ResponseWriter, r *http.Request) {
claims := middleware.BootClaimsFrom(r.Context())
if claims == nil {
response.Unauthorized(w)
return
}

imageID, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}

task, err := h.db.Task.Get(r.Context(), claims.TaskID)
if err != nil || task.ImageID == nil || *task.ImageID != imageID {
response.Forbidden(w)
return
}

partStr := r.URL.Query().Get("part")
partNum, parseErr := strconv.Atoi(partStr)
if parseErr != nil {
response.BadRequest(w, "part parameter must be an integer")
return
}

img, err := h.db.Image.Get(r.Context(), imageID)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "image")
return
}
response.InternalError(w)
return
}

rc := http.NewResponseController(w)
_ = rc.SetReadDeadline(time.Time{})
_ = rc.SetWriteDeadline(time.Time{})

nodeURL, resolveErr := h.resolveStorageNodeURL(r, task, img)
if resolveErr != nil {
h.uploadLocal(w, r, img, partNum)
return
}

upstreamURL := fmt.Sprintf("%s/%s/%s", strings.TrimRight(nodeURL, "/"), img.Path, partFilename(partNum))
h.proxyPut(w, r, upstreamURL)
}

// ------------------------------------------------------------------
// ImageMeta — POST /fog/api/v1/boot/images/meta  (boot-token auth)
// ------------------------------------------------------------------

type imageMetaRequest struct {
	TaskID              string   `json:"taskId"`
	ImageID             string   `json:"imageId"`
	ImageType           string   `json:"imageType"`
	FixedSizePartitions []int    `json:"fixedSizePartitions"`
	PartCount           int      `json:"partCount"`
	PartTypes           []string `json:"partTypes,omitempty"`
}

func (h *BootAPI) ImageMeta(w http.ResponseWriter, r *http.Request) {
claims := middleware.BootClaimsFrom(r.Context())
if claims == nil {
response.Unauthorized(w)
return
}

var req imageMetaRequest
if !response.Decode(w, r, &req) {
return
}

taskID, err := uuid.Parse(req.TaskID)
if err != nil || taskID != claims.TaskID {
response.BadRequest(w, "task ID mismatch")
return
}

imageID, err := uuid.Parse(req.ImageID)
if err != nil {
response.BadRequest(w, "invalid image ID")
return
}

task, err := h.db.Task.Get(r.Context(), taskID)
if err != nil || task.ImageID == nil || *task.ImageID != imageID {
response.Forbidden(w)
return
}

	ip := imagePartitions{
		PartCount:           req.PartCount,
		ImageType:           req.ImageType,
		FixedSizePartitions: req.FixedSizePartitions,
		PartTypes:           req.PartTypes,
	}
partJSON, err := json.Marshal(ip)
if err != nil {
response.InternalError(w)
return
}

if err := h.db.Image.UpdateOneID(imageID).SetPartitions(partJSON).Exec(r.Context()); err != nil {
slog.Error("ImageMeta: update image failed", "imageId", imageID, "err", err)
response.InternalError(w)
return
}

slog.Info("ImageMeta saved", "imageId", imageID, "partCount", ip.PartCount, "imageType", ip.ImageType)
response.NoContent(w)
}

// ------------------------------------------------------------------
// HostList — GET /fog/api/v1/boot/hosts  (unauthenticated — for iPXE menu)
// ------------------------------------------------------------------

func (h *BootAPI) HostList(w http.ResponseWriter, r *http.Request) {
hosts, err := h.db.Host.Query().Where(enthost.IsEnabledEQ(true)).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(hosts))
}

// ------------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------------

func (h *BootAPI) proxyGet(w http.ResponseWriter, r *http.Request, upstreamURL string) {
req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstreamURL, nil)
if err != nil {
slog.Error("proxy GET: build request", "url", upstreamURL, "err", err)
response.InternalError(w)
return
}
if rng := r.Header.Get("Range"); rng != "" {
req.Header.Set("Range", rng)
}
resp, err := h.httpClient.Do(req)
if err != nil {
slog.Error("proxy GET: upstream request failed", "url", upstreamURL, "err", err)
response.InternalError(w)
return
}
defer resp.Body.Close()
for _, hdr := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
if v := resp.Header.Get(hdr); v != "" {
w.Header().Set(hdr, v)
}
}
w.WriteHeader(resp.StatusCode)
_, _ = io.Copy(w, resp.Body)
}

func (h *BootAPI) proxyPut(w http.ResponseWriter, r *http.Request, upstreamURL string) {
req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, upstreamURL, r.Body)
if err != nil {
slog.Error("proxy PUT: build request", "url", upstreamURL, "err", err)
response.InternalError(w)
return
}
req.Header.Set("Content-Type", "application/octet-stream")
resp, err := h.httpClient.Do(req)
if err != nil {
slog.Error("proxy PUT: upstream request failed", "url", upstreamURL, "err", err)
response.InternalError(w)
return
}
defer resp.Body.Close()
w.WriteHeader(resp.StatusCode)
}

func (h *BootAPI) downloadLocal(w http.ResponseWriter, r *http.Request, img *ent.Image, partNum int) {
base := filepath.Clean(h.cfg.Storage.BasePath)
rel := filepath.Join(base, filepath.FromSlash(img.Path), partFilename(partNum))
if !strings.HasPrefix(rel, base+string(filepath.Separator)) {
response.BadRequest(w, "invalid image path")
return
}
http.ServeFile(w, r, rel)
}

func (h *BootAPI) uploadLocal(w http.ResponseWriter, r *http.Request, img *ent.Image, partNum int) {
	base := filepath.Clean(h.cfg.Storage.BasePath)
	dir := filepath.Join(base, filepath.FromSlash(img.Path))
	if !strings.HasPrefix(dir+string(filepath.Separator), base+string(filepath.Separator)) {
		response.BadRequest(w, "invalid image path")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("upload local: mkdir", "dir", dir, "err", err)
		response.InternalError(w)
		return
	}
	dest := filepath.Join(dir, partFilename(partNum))

	// Resume support: if offset=N is provided, append to existing file.
	offsetStr := r.URL.Query().Get("offset")
	var offset int64
	if offsetStr != "" {
		var parseErr error
		offset, parseErr = strconv.ParseInt(offsetStr, 10, 64)
		if parseErr != nil || offset < 0 {
			response.BadRequest(w, "invalid offset parameter")
			return
		}
	}

	if offset > 0 {
		fi, statErr := os.Stat(dest)
		if statErr != nil {
			response.BadRequest(w, "resume requested but target file does not exist")
			return
		}
		if fi.Size() != offset {
			response.BadRequest(w, fmt.Sprintf("resume offset %d does not match existing file size %d", offset, fi.Size()))
			return
		}
		f, openErr := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND, 0o644)
		if openErr != nil {
			slog.Error("upload local: open for append", "dest", dest, "err", openErr)
			response.InternalError(w)
			return
		}
		defer f.Close()
		if _, copyErr := io.Copy(f, r.Body); copyErr != nil {
			slog.Warn("upload local: append body", "path", dest, "err", copyErr)
			response.InternalError(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		slog.Error("upload local: create temp", "dir", dir, "err", err)
		response.InternalError(w)
		return
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, r.Body); err != nil {
		tmp.Close()
		slog.Warn("upload local: read body", "path", dest, "err", err)
		response.InternalError(w)
		return
	}
	tmp.Close()

	if err := os.Rename(tmpName, dest); err != nil {
		slog.Error("upload local: rename", "tmp", tmpName, "dest", dest, "err", err)
		response.InternalError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BootAPI) resolveStorageNodeURL(r *http.Request, task *ent.Task, img *ent.Image) (string, error) {
if task.StorageNodeID != nil {
node, err := h.db.StorageNode.Get(r.Context(), *task.StorageNodeID)
if err == nil && node.IsEnabled {
return buildStorageNodeURL(node), nil
}
}
if img.StorageGroupID != nil {
node, err := h.db.StorageNode.Query().Where(
storagenode.StorageGroupIDEQ(*img.StorageGroupID),
storagenode.IsMasterEQ(true),
storagenode.IsEnabledEQ(true),
).First(r.Context())
if err == nil {
return buildStorageNodeURL(node), nil
}
}
return "", fmt.Errorf("no available storage node for image %s", img.ID)
}

func buildStorageNodeURL(n *ent.StorageNode) string {
base := "http://" + n.Hostname
if n.WebRoot != "" {
return base + "/" + strings.Trim(n.WebRoot, "/")
}
return base
}

func partFilename(part int) string {
if part == 0 {
return "ptable"
}
return "part" + strconv.Itoa(part)
}

func parseImagePartitions(img *ent.Image) *imagePartitions {
	if len(img.Partitions) == 0 {
		return nil
	}
	var ip imagePartitions
	if err := json.Unmarshal(img.Partitions, &ip); err != nil {
		slog.Warn("parseImagePartitions: malformed JSONB", "imageId", img.ID, "err", err)
		return nil
	}
	return &ip
}

// cleanupImageParts removes all partition files and the ptable for an image
// directory. Used when a capture fails to avoid leaving incomplete data on disk.
func (h *BootAPI) cleanupImageParts(imageID uuid.UUID) {
	ctx := context.Background()
	img, err := h.db.Image.Get(ctx, imageID)
	if err != nil {
		slog.Warn("cleanupImageParts: image not found", "imageId", imageID, "err", err)
		return
	}
	base := filepath.Clean(h.cfg.Storage.BasePath)
	dir := filepath.Join(base, filepath.FromSlash(img.Path))
	if !strings.HasPrefix(dir+string(filepath.Separator), base+string(filepath.Separator)) {
		slog.Warn("cleanupImageParts: path traversal attempt", "dir", dir)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Warn("cleanupImageParts: readDir failed", "dir", dir, "err", err)
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "ptable" || strings.HasPrefix(name, "part") {
			fp := filepath.Join(dir, name)
			if rmErr := os.Remove(fp); rmErr != nil {
				slog.Warn("cleanupImageParts: remove failed", "file", fp, "err", rmErr)
			}
		}
	}
	slog.Info("cleanupImageParts: cleaned up failed capture parts", "imageId", imageID, "dir", dir)
}

func normMAC(mac string) string {
return strings.ToLower(strings.TrimSpace(mac))
}

func isZeroMAC(mac string) bool {
return mac == "00:00:00:00:00:00"
}

// ------------------------------------------------------------------
// Logs — POST /fog/api/v1/boot/logs  (boot-token auth)
// ------------------------------------------------------------------

type agentLogEntry struct {
Time  string         `json:"time"`
Level string         `json:"level"`
Msg   string         `json:"msg"`
Attrs map[string]any `json:"attrs,omitempty"`
}

type logsRequest struct {
TaskID  string          `json:"taskId"`
Entries []agentLogEntry `json:"entries"`
}

type agentLogPayload struct {
TaskID  string          `json:"taskId"`
HostID  string          `json:"hostId"`
Entries []agentLogEntry `json:"entries"`
}

func (h *BootAPI) Logs(w http.ResponseWriter, r *http.Request) {
claims := middleware.BootClaimsFrom(r.Context())
if claims == nil {
response.Unauthorized(w)
return
}

var req logsRequest
if !response.Decode(w, r, &req) {
return
}

taskID, err := uuid.Parse(req.TaskID)
if err != nil || taskID != claims.TaskID {
response.BadRequest(w, "task ID mismatch")
return
}

if len(req.Entries) == 0 {
response.NoContent(w)
return
}

builders := make([]*ent.AgentLogCreate, 0, len(req.Entries))
for _, e := range req.Entries {
t, parseErr := time.Parse(time.RFC3339Nano, e.Time)
if parseErr != nil {
t = time.Now()
}
b := h.db.AgentLog.Create().
SetTaskID(claims.TaskID).
SetHostID(claims.HostID).
SetLoggedAt(t).
SetLevel(e.Level).
SetMessage(e.Msg)
if len(e.Attrs) > 0 {
b = b.SetAttrs(e.Attrs)
}
builders = append(builders, b)
}

if err := h.db.AgentLog.CreateBulk(builders...).Exec(r.Context()); err != nil {
slog.Error("agent log bulk insert failed", "task", claims.TaskID, "err", err)
response.InternalError(w)
return
}

if h.hub != nil {
h.hub.Broadcast(ws.Event{
Type: ws.EventAgentLog,
Payload: agentLogPayload{
TaskID:  claims.TaskID.String(),
HostID:  claims.HostID.String(),
Entries: req.Entries,
},
At: time.Now(),
})
}

response.NoContent(w)
}
