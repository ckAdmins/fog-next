package services

import (
"context"
"fmt"
"log/slog"
"os/exec"
"time"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/storagenode"
"github.com/nemvince/fog-next/internal/config"
)

// ImageReplicator copies master images to all slave storage nodes using rsync over SSH.
type ImageReplicator struct {
cfg *config.Config
db  *ent.Client
}

func NewImageReplicator(cfg *config.Config, db *ent.Client) *ImageReplicator {
return &ImageReplicator{cfg, db}
}

func (s *ImageReplicator) Name() string { return "ImageReplicator" }

func (s *ImageReplicator) Run(ctx context.Context) error {
ticker := time.NewTicker(s.cfg.Services.ReplicatorInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil
case <-ticker.C:
s.replicate(ctx)
}
}
}

func (s *ImageReplicator) replicate(ctx context.Context) {
groups, err := s.db.StorageGroup.Query().All(ctx)
if err != nil {
slog.Error("replicator: list storage groups", "error", err)
return
}

for _, group := range groups {
nodes, err := s.db.StorageNode.Query().Where(storagenode.StorageGroupIDEQ(group.ID)).All(ctx)
if err != nil {
continue
}

var masterRoot string
type slaveInfo struct{ user, host, root string }
var slaves []slaveInfo

for _, n := range nodes {
if !n.IsEnabled {
continue
}
if n.IsMaster {
masterRoot = n.RootPath
} else {
slaves = append(slaves, slaveInfo{n.SSHUser, n.Hostname, n.RootPath})
}
}

if masterRoot == "" || len(slaves) == 0 {
continue
}

for _, slave := range slaves {
if err := rsyncImages(ctx, masterRoot, slave.user, slave.host, slave.root); err != nil {
slog.Error("replicator: rsync failed",
"to", slave.host, "error", err)
} else {
slog.Info("replicator: sync complete", "to", slave.host)
}
}
}
}

func rsyncImages(ctx context.Context, srcRoot, destUser, destHost, destRoot string) error {
dest := fmt.Sprintf("%s@%s:%s/", destUser, destHost, destRoot)
cmd := exec.CommandContext(ctx,
"rsync", "-azv", "--delete",
"-e", "ssh -o StrictHostKeyChecking=no -o BatchMode=yes",
srcRoot+"/", dest,
)
out, err := cmd.CombinedOutput()
if err != nil {
return fmt.Errorf("rsync: %w\noutput: %s", err, out)
}
return nil
}
