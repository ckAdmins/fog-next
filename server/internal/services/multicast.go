package services

import (
"context"
"fmt"
"log/slog"
"os/exec"
"path/filepath"
"time"

"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/multicastsession"
"github.com/nemvince/fog-next/internal/config"
)

const (
multicastStatePending = "pending"
multicastStateRunning = "running"
multicastStateDone    = "done"
multicastStateFailed  = "failed"
)

// MulticastManager manages udpcast multicast imaging sessions.
type MulticastManager struct {
cfg *config.Config
db  *ent.Client
}

func NewMulticastManager(cfg *config.Config, db *ent.Client) *MulticastManager {
return &MulticastManager{cfg, db}
}

func (m *MulticastManager) Name() string { return "MulticastManager" }

func (m *MulticastManager) Run(ctx context.Context) error {
ticker := time.NewTicker(m.cfg.Services.MulticastInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil
case <-ticker.C:
m.processQueued(ctx)
}
}
}

func (m *MulticastManager) processQueued(ctx context.Context) {
sessions, err := m.db.MulticastSession.Query().Where(
multicastsession.StateIn(multicastStatePending, multicastStateRunning),
).All(ctx)
if err != nil {
slog.Error("multicast: list pending sessions", "error", err)
return
}

for _, sess := range sessions {
if sess.State == multicastStatePending {
if err := m.launchSession(ctx, sess); err != nil {
slog.Error("multicast: launch session", "session", sess.ID, "error", err)
}
}
}
}

func (m *MulticastManager) launchSession(ctx context.Context, sess *ent.MulticastSession) error {
img, err := m.db.Image.Get(ctx, sess.ImageID)
if err != nil {
return fmt.Errorf("get image: %w", err)
}

imgPath := filepath.Join(m.cfg.Storage.BasePath, img.Path)

if err := m.db.MulticastSession.UpdateOneID(sess.ID).SetState(multicastStateRunning).Exec(ctx); err != nil {
return fmt.Errorf("update session state: %w", err)
}

sessID := sess.ID
go m.runUdpcast(sessID, imgPath)
return nil
}

func (m *MulticastManager) runUdpcast(sessID uuid.UUID, imgPath string) {
ctx := context.Background()
args := []string{
"--file", imgPath,
"--min-receivers", "1",
"--max-wait", "60",
}

cmd := exec.CommandContext(ctx, "udp-sender", args...)
out, err := cmd.CombinedOutput()

newState := multicastStateDone
if err != nil {
slog.Error("multicast: udpcast failed", "session", sessID, "error", err, "output", string(out))
newState = multicastStateFailed
} else {
slog.Info("multicast: session complete", "session", sessID)
}

if updateErr := m.db.MulticastSession.UpdateOneID(sessID).SetState(newState).Exec(ctx); updateErr != nil {
slog.Error("multicast: update final state", "session", sessID, "error", updateErr)
}
}
