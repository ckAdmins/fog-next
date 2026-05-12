package services

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/nemvince/fog-next/ent"
	"github.com/nemvince/fog-next/ent/hostmac"
	"github.com/nemvince/fog-next/ent/scheduledtask"
	enttask "github.com/nemvince/fog-next/ent/task"
	"github.com/nemvince/fog-next/internal/config"
	"github.com/robfig/cron/v3"
)

// TaskScheduler polls for queued tasks, sends WOL packets, and runs cron-based
// ScheduledTask entries.
type TaskScheduler struct {
cfg *config.Config
db  *ent.Client
}

func NewTaskScheduler(cfg *config.Config, db *ent.Client) *TaskScheduler {
return &TaskScheduler{cfg, db}
}

func (s *TaskScheduler) Name() string { return "TaskScheduler" }

func (s *TaskScheduler) Run(ctx context.Context) error {
ticker := time.NewTicker(s.cfg.Services.SchedulerInterval)
defer ticker.Stop()

cr := cron.New()
cr.Start()
defer cr.Stop()

if err := s.registerScheduledTasks(ctx, cr); err != nil {
slog.Warn("failed to register scheduled tasks on startup", "error", err)
}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.processQueued(ctx)
			s.cancelStaleTasks(ctx)
		}
	}
}

func (s *TaskScheduler) processQueued(ctx context.Context) {
	tasks, err := s.db.Task.Query().Where(enttask.StateEQ(enttask.StateQueued)).All(ctx)
	if err != nil {
		slog.Error("scheduler: list queued tasks", "error", err)
		return
	}

	for _, task := range tasks {
		macs, err := s.db.HostMAC.Query().Where(hostmac.HostIDEQ(task.HostID)).All(ctx)
		if err == nil {
			for _, m := range macs {
				if m.IsPrimary {
					if err := sendWOL(m.MAC); err != nil {
						slog.Warn("scheduler: WOL failed", "mac", m.MAC, "error", err)
					}
				}
			}
		}
	}
}

// cancelStaleTasks cancels queued tasks that have exceeded the TaskTimeout
// without the host ever handshaking.
func (s *TaskScheduler) cancelStaleTasks(ctx context.Context) {
	if s.cfg.Services.TaskTimeout <= 0 {
		return
	}
	deadline := time.Now().Add(-s.cfg.Services.TaskTimeout)
	n, err := s.db.Task.Update().
		Where(
			enttask.StateEQ(enttask.StateQueued),
			enttask.CreatedAtLT(deadline),
		).
		SetState(enttask.StateCanceled).
		Save(ctx)
	if err != nil {
		slog.Warn("scheduler: cancel stale tasks failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("scheduler: cancelled stale queued tasks", "count", n)
	}
}

func (s *TaskScheduler) registerScheduledTasks(ctx context.Context, cr *cron.Cron) error {
tasks, err := s.db.ScheduledTask.Query().Where(scheduledtask.IsActiveEQ(true)).All(ctx)
if err != nil {
return err
}
for _, st := range tasks {
st := st
spec := fmt.Sprintf("%s %s %s %s %s", st.Minute, st.Hour, st.DayOfMonth, st.Month, st.DayOfWeek)
_, err := cr.AddFunc(spec, func() {
s.execScheduledTask(context.Background(), st)
})
if err != nil {
slog.Warn("scheduler: invalid cron spec", "task", st.Name, "spec", spec, "error", err)
}
}
return nil
}

func (s *TaskScheduler) execScheduledTask(ctx context.Context, st *ent.ScheduledTask) {
if err := s.db.Task.Create().
SetName(st.Name).
SetType(enttask.Type(st.TaskType)).
SetState(enttask.StateQueued).
SetHostID(st.TargetID).
SetIsGroup(st.IsGroup).
SetIsShutdown(st.IsShutdown).
SetCreatedBy("scheduler").
Exec(ctx); err != nil {
slog.Error("scheduler: create scheduled task", "name", st.Name, "error", err)
}
}

// sendWOL sends a Wake-on-LAN magic packet to the given MAC address.
func sendWOL(macStr string) error {
mac, err := net.ParseMAC(macStr)
if err != nil {
return fmt.Errorf("parse mac %q: %w", macStr, err)
}

packet := make([]byte, 6+16*6)
for i := 0; i < 6; i++ {
packet[i] = 0xFF
}
for i := 0; i < 16; i++ {
copy(packet[6+i*6:], mac)
}

conn, err := net.Dial("udp", "255.255.255.255:9")
if err != nil {
return fmt.Errorf("dial broadcast: %w", err)
}
defer conn.Close()

if _, err := conn.Write(packet); err != nil {
return fmt.Errorf("write WOL packet: %w", err)
}
return nil
}
