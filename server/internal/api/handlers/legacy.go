package handlers

import (
"fmt"
"net/http"
"strings"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/hostmac"
"github.com/nemvince/fog-next/ent/snapinassoc"
enttask "github.com/nemvince/fog-next/ent/task"
"github.com/nemvince/fog-next/internal/api/response"
"github.com/nemvince/fog-next/internal/config"
)

// Legacy handles the FOG 1.x client communication protocol for backwards
// compatibility while the FOG client base migrates to the new API.
type Legacy struct {
cfg *config.Config
db  *ent.Client
}

func NewLegacy(cfg *config.Config, db *ent.Client) *Legacy {
return &Legacy{cfg, db}
}

// Register accepts a MAC address from a booting host and ensures it exists
// in the database. Unknown MACs are added to pending_macs for admin review.
func (h *Legacy) Register(w http.ResponseWriter, r *http.Request) {
mac := strings.ToLower(strings.TrimSpace(r.FormValue("mac")))
if mac == "" || mac == "00:00:00:00:00:00" {
http.Error(w, "bad request", http.StatusBadRequest)
return
}

_, err := h.db.HostMAC.Query().Where(hostmac.MACEQ(mac)).QueryHost().Only(r.Context())
if err == nil {
fmt.Fprint(w, "#!ok")
return
}
if !ent.IsNotFound(err) {
response.InternalError(w)
return
}

_ = h.db.PendingMAC.Create().SetMAC(mac).Exec(r.Context())
fmt.Fprint(w, "#!reg")
}

// HostInfo returns the task configuration for a known host identified by MAC.
func (h *Legacy) HostInfo(w http.ResponseWriter, r *http.Request) {
mac := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mac")))
if mac == "" {
http.Error(w, "bad request", http.StatusBadRequest)
return
}

host, err := h.db.HostMAC.Query().Where(hostmac.MACEQ(mac)).QueryHost().Only(r.Context())
if err != nil {
if ent.IsNotFound(err) {
fmt.Fprint(w, "#!err=Host not found")
return
}
response.InternalError(w)
return
}

task, err := h.db.Task.Query().Where(
enttask.HostIDEQ(host.ID),
enttask.StateIn(enttask.StateQueued, enttask.StateActive),
).Order(enttask.ByCreatedAt()).First(r.Context())
if err != nil {
if ent.IsNotFound(err) {
fmt.Fprint(w, "#!noTask")
return
}
response.InternalError(w)
return
}

var nodeHost, nodeRoot string
if task.StorageNodeID != nil {
node, err := h.db.StorageNode.Get(r.Context(), *task.StorageNodeID)
if err == nil {
nodeHost = node.Hostname
nodeRoot = node.RootPath
}
}

var imageID string
if task.ImageID != nil {
imageID = task.ImageID.String()
}
fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
task.ID, task.Type, imageID, nodeHost, nodeRoot)
}

// Progress receives imaging progress updates from the legacy client.
func (h *Legacy) Progress(w http.ResponseWriter, r *http.Request) {
taskID := r.FormValue("taskid")
id, ok := parseUUID(w, taskID)
if !ok {
return
}

t, err := h.db.Task.Get(r.Context(), id)
if err != nil {
fmt.Fprint(w, "#!err=Task not found")
return
}

newState := enttask.StateActive
pct := t.PercentComplete
if p := r.FormValue("pct"); p != "" {
var v int
if _, err := fmt.Sscanf(p, "%d", &v); err == nil {
pct = v
}
}
if pct >= 100 {
newState = enttask.StateComplete
}

_ = h.db.Task.UpdateOneID(id).SetPercentComplete(pct).SetState(newState).Exec(r.Context())
fmt.Fprint(w, "#!ok")
}

// Jobs returns pending snapin tasks for a host (legacy format).
func (h *Legacy) Jobs(w http.ResponseWriter, r *http.Request) {
mac := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mac")))
if mac == "" {
http.Error(w, "bad request", http.StatusBadRequest)
return
}

host, err := h.db.HostMAC.Query().Where(hostmac.MACEQ(mac)).QueryHost().Only(r.Context())
if err != nil {
fmt.Fprint(w, "#!noJobs")
return
}

snapins, err := h.db.SnapinAssoc.Query().Where(snapinassoc.HostIDEQ(host.ID)).QuerySnapin().All(r.Context())
if err != nil || len(snapins) == 0 {
fmt.Fprint(w, "#!noJobs")
return
}

for _, sn := range snapins {
fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", sn.ID, sn.FileName, sn.Command, sn.Arguments)
}
}
