package services

import (
"context"
"log/slog"
"os"
"path/filepath"
"time"

"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/internal/config"
)

// ImageSize periodically walks the storage root and updates size_bytes on each Image.
type ImageSize struct {
cfg *config.Config
db  *ent.Client
}

func NewImageSize(cfg *config.Config, db *ent.Client) *ImageSize {
return &ImageSize{cfg, db}
}

func (s *ImageSize) Name() string { return "ImageSize" }

func (s *ImageSize) Run(ctx context.Context) error {
ticker := time.NewTicker(s.cfg.Services.ImageSizeInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil
case <-ticker.C:
s.updateSizes(ctx)
}
}
}

func dirSize(path string) (int64, error) {
var total int64
err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
if err != nil {
return nil
}
if !d.IsDir() {
info, err := d.Info()
if err == nil {
total += info.Size()
}
}
return nil
})
return total, err
}

func (s *ImageSize) updateSizes(ctx context.Context) {
images, err := s.db.Image.Query().Limit(10000).All(ctx)
if err != nil {
slog.Error("imagesize: list images", "error", err)
return
}

for _, img := range images {
if img.Path == "" {
continue
}

imgPath := filepath.Join(s.cfg.Storage.BasePath, img.Path)
size, err := dirSize(imgPath)
if err != nil {
slog.Warn("imagesize: cannot measure", "path", imgPath, "error", err)
continue
}

if img.SizeBytes != size {
if err := s.db.Image.UpdateOneID(img.ID).SetSizeBytes(size).Exec(ctx); err != nil {
slog.Error("imagesize: update image", "name", img.Name, "error", err)
}
}
}
}
