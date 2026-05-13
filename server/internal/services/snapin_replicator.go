package services

import (
"context"
"log/slog"
"os/exec"
"time"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/storagenode"
"github.com/nemvince/fog-next/internal/config"
)

// SnapinReplicator syncs snapin files from master storage nodes to all slave nodes.
type SnapinReplicator struct {
cfg *config.Config
db  *ent.Client
}

func NewSnapinReplicator(cfg *config.Config, db *ent.Client) *SnapinReplicator {
return &SnapinReplicator{cfg, db}
}

func (sr *SnapinReplicator) Name() string { return "SnapinReplicator" }

func (sr *SnapinReplicator) Run(ctx context.Context) error {
ticker := time.NewTicker(sr.cfg.Services.SnapinReplicatorInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil
case <-ticker.C:
sr.replicate(ctx)
}
}
}

func (sr *SnapinReplicator) replicate(ctx context.Context) {
groups, err := sr.db.StorageGroup.Query().All(ctx)
if err != nil {
slog.Error("snapin-replicator: list groups", "error", err)
return
}

for _, group := range groups {
nodes, err := sr.db.StorageNode.Query().Where(storagenode.StorageGroupIDEQ(group.ID)).All(ctx)
if err != nil {
slog.Error("snapin-replicator: list nodes", "group", group.Name, "error", err)
continue
}

type nodeInfo struct{ name, host, root, sshUser string }
var master *nodeInfo
var slaves []nodeInfo
for _, n := range nodes {
if !n.IsEnabled {
continue
}
if n.IsMaster {
master = &nodeInfo{n.Name, n.Hostname, n.RootPath, n.SSHUser}
} else {
slaves = append(slaves, nodeInfo{n.Name, n.Hostname, n.RootPath, n.SSHUser})
}
}

if master == nil || len(slaves) == 0 {
continue
}

for _, slave := range slaves {
sr.rsyncSnapins(ctx, master.host, master.root, master.sshUser, slave.host, slave.root, slave.sshUser)
}
}
}

func (sr *SnapinReplicator) rsyncSnapins(ctx context.Context, masterHost, masterRoot, masterUser, slaveHost, slaveRoot, slaveUser string) {
srcPath := masterHost + ":" + masterRoot + "/"
if masterUser != "" {
srcPath = masterUser + "@" + srcPath
}

dstPath := slaveHost + ":" + slaveRoot + "/"
if slaveUser != "" {
dstPath = slaveUser + "@" + dstPath
}

cmd := exec.CommandContext(ctx, "rsync",
"-azv", "--delete",
"-e", "ssh -o StrictHostKeyChecking=no -o BatchMode=yes",
srcPath, dstPath,
)

if out, err := cmd.CombinedOutput(); err != nil {
slog.Error("snapin-replicator: rsync failed",
"master", masterHost, "slave", slaveHost, "error", err, "output", string(out))
} else {
slog.Info("snapin-replicator: sync complete", "master", masterHost, "slave", slaveHost)
}
}
