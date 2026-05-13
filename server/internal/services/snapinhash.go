package services

import (
"context"
"crypto/sha256"
"fmt"
"io"
"log/slog"
"os"
"path/filepath"
"time"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/config"
)

// SnapinHash periodically computes SHA-256 checksums for all snapin files
// and updates the database.
type SnapinHash struct {
cfg *config.Config
db  *ent.Client
}

func NewSnapinHash(cfg *config.Config, db *ent.Client) *SnapinHash {
return &SnapinHash{cfg, db}
}

func (s *SnapinHash) Name() string { return "SnapinHash" }

func (s *SnapinHash) Run(ctx context.Context) error {
ticker := time.NewTicker(s.cfg.Services.SnapinHashInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil
case <-ticker.C:
s.computeHashes(ctx)
}
}
}

func (s *SnapinHash) computeHashes(ctx context.Context) {
snapins, err := s.db.Snapin.Query().Limit(10000).All(ctx)
if err != nil {
slog.Error("snapinhash: list snapins", "error", err)
return
}

for _, sn := range snapins {
if sn.FilePath == "" {
continue
}

hash, err := sha256File(sn.FilePath)
if err != nil {
slog.Warn("snapinhash: cannot hash file", "path", sn.FilePath, "error", err)
continue
}

if hash != sn.Hash {
if err := s.db.Snapin.UpdateOneID(sn.ID).SetHash(hash).Exec(ctx); err != nil {
slog.Error("snapinhash: update snapin", "name", sn.Name, "error", err)
}
}
}
}

func sha256File(path string) (string, error) {
f, err := os.Open(filepath.Clean(path))
if err != nil {
return "", err
}
defer f.Close()

h := sha256.New()
if _, err := io.Copy(h, f); err != nil {
return "", err
}
return fmt.Sprintf("%x", h.Sum(nil)), nil
}
