package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ckAdmins/fog-next/ent"
	"github.com/ckAdmins/fog-next/ent/multicastsession"
	enttask "github.com/ckAdmins/fog-next/ent/task"
	"github.com/ckAdmins/fog-next/internal/config"
)

const (
	multicastStatePending = "pending"
	multicastStateRunning = "running"
	multicastStateDone    = "done"
	multicastStateFailed  = "failed"

	multicastStaleTimeout = 15 * time.Minute
)

// sessionTracker holds in-memory readiness and completion state for all
// active multicast sessions, keyed by (sessionID, part, taskID).
type sessionTracker struct {
	mu        sync.RWMutex
	readiness map[uuid.UUID]map[int]map[uuid.UUID]bool // session → part → task → ready
	completed map[uuid.UUID]map[int]map[uuid.UUID]bool // session → part → task → done
}

func newSessionTracker() *sessionTracker {
	return &sessionTracker{
		readiness: make(map[uuid.UUID]map[int]map[uuid.UUID]bool),
		completed: make(map[uuid.UUID]map[int]map[uuid.UUID]bool),
	}
}

// initSession pre-allocates the tracker maps for a session's part range.
func (t *sessionTracker) initSession(sessionID uuid.UUID, clientCount, partCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r := make(map[int]map[uuid.UUID]bool, partCount+1)
	c := make(map[int]map[uuid.UUID]bool, partCount+1)
	for p := 1; p <= partCount; p++ {
		r[p] = make(map[uuid.UUID]bool, clientCount)
		c[p] = make(map[uuid.UUID]bool, clientCount)
	}
	t.readiness[sessionID] = r
	t.completed[sessionID] = c
}

func (t *sessionTracker) removeSession(sessionID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.readiness, sessionID)
	delete(t.completed, sessionID)
}

// signalReady marks a task as ready for a partition and returns true when
// all expected clients have reported ready for that part.
func (t *sessionTracker) signalReady(sessionID, taskID uuid.UUID, part int, clientCount int) (allReady bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.readiness[sessionID]; !ok {
		return false
	}
	if _, ok := t.readiness[sessionID][part]; !ok {
		return false
	}
	t.readiness[sessionID][part][taskID] = true
	return len(t.readiness[sessionID][part]) >= clientCount
}

// signalPartDone marks a task as done for a partition and returns true when
// all clients have completed that part.
func (t *sessionTracker) signalPartDone(sessionID, taskID uuid.UUID, part int, clientCount int) (allDone bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.completed[sessionID]; !ok {
		return false
	}
	if _, ok := t.completed[sessionID][part]; !ok {
		return false
	}
	t.completed[sessionID][part][taskID] = true
	return len(t.completed[sessionID][part]) >= clientCount
}

// MulticastManager manages per-partition udpcast imaging sessions.
type MulticastManager struct {
	cfg     *config.Config
	db      *ent.Client
	tracker *sessionTracker
}

func NewMulticastManager(cfg *config.Config, db *ent.Client) *MulticastManager {
	return &MulticastManager{
		cfg:     cfg,
		db:      db,
		tracker: newSessionTracker(),
	}
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
			m.advanceRunning(ctx)
		}
	}
}

// ---------------------------------------------------------------------------
// Background loop
// ---------------------------------------------------------------------------

func (m *MulticastManager) processQueued(ctx context.Context) {
	sessions, err := m.db.MulticastSession.Query().Where(
		multicastsession.StateEQ(multicastStatePending),
	).All(ctx)
	if err != nil {
		slog.Error("multicast: list pending sessions", "error", err)
		return
	}

	for _, sess := range sessions {
		if err := m.launchSession(ctx, sess); err != nil {
			slog.Error("multicast: launch session", "session", sess.ID, "error", err)
		}
	}
}

func (m *MulticastManager) launchSession(ctx context.Context, sess *ent.MulticastSession) error {
	img, err := m.db.Image.Get(ctx, sess.ImageID)
	if err != nil {
		return fmt.Errorf("get image: %w", err)
	}

	partitions := parseImagePartitions(img)
	if partitions == nil {
		return fmt.Errorf("image %s has no partition metadata", img.ID)
	}

	// Count associated tasks to know expected client count.
	taskCount, err := m.db.Task.Query().Where(
		enttask.MulticastSessionIDEQ(sess.ID),
		enttask.StateIn(enttask.StateQueued, enttask.StateActive),
	).Count(ctx)
	if err != nil {
		return fmt.Errorf("count session tasks: %w", err)
	}
	if taskCount == 0 {
		slog.Warn("multicast: session has no tasks, marking done", "session", sess.ID)
		_ = m.db.MulticastSession.UpdateOneID(sess.ID).SetState(multicastStateDone).Exec(ctx)
		return nil
	}

	clientCount := taskCount
	if sess.ClientCount > 0 {
		clientCount = sess.ClientCount
	}

	m.tracker.initSession(sess.ID, clientCount, partitions.PartCount)

	now := time.Now()
	if err := m.db.MulticastSession.UpdateOneID(sess.ID).
		SetState(multicastStateRunning).
		SetCurrentPart(1).
		SetStartedAt(now).
		Exec(ctx); err != nil {
		m.tracker.removeSession(sess.ID)
		return fmt.Errorf("update session state: %w", err)
	}

	slog.Info("multicast: session started",
		"session", sess.ID, "image", img.ID,
		"parts", partitions.PartCount, "clients", clientCount)
	return nil
}

// advanceRunning checks running sessions for partitions ready to be sent
// and launches udp-sender when all clients are ready for the current part.
func (m *MulticastManager) advanceRunning(ctx context.Context) {
	sessions, err := m.db.MulticastSession.Query().Where(
		multicastsession.StateEQ(multicastStateRunning),
	).All(ctx)
	if err != nil {
		slog.Error("multicast: list running sessions", "error", err)
		return
	}

	for _, sess := range sessions {
		m.advanceSession(ctx, sess)
	}
}

func (m *MulticastManager) advanceSession(ctx context.Context, sess *ent.MulticastSession) {
	part := sess.CurrentPart
	if part < 1 {
		return
	}

	img, err := m.db.Image.Get(ctx, sess.ImageID)
	if err != nil {
		slog.Error("multicast: get image", "session", sess.ID, "error", err)
		return
	}

	partitions := parseImagePartitions(img)
	if partitions == nil {
		slog.Error("multicast: no partition metadata", "session", sess.ID)
		_ = m.db.MulticastSession.UpdateOneID(sess.ID).SetState(multicastStateFailed).Exec(ctx)
		return
	}

	// Check staleness.
	if sess.StartedAt != nil && time.Since(*sess.StartedAt) > multicastStaleTimeout {
		slog.Warn("multicast: session stale, marking failed", "session", sess.ID)
		_ = m.db.MulticastSession.UpdateOneID(sess.ID).SetState(multicastStateFailed).Exec(ctx)
		m.tracker.removeSession(sess.ID)
		return
	}

	// If we've passed the last partition, mark session done.
	if part > partitions.PartCount {
		now := time.Now()
		_ = m.db.MulticastSession.UpdateOneID(sess.ID).
			SetState(multicastStateDone).
			SetCompletedAt(now).
			Exec(ctx)
		m.tracker.removeSession(sess.ID)
		slog.Info("multicast: session complete", "session", sess.ID)
		return
	}

	clientCount := sess.ClientCount
	if clientCount <= 0 {
		taskCount, err := m.db.Task.Query().Where(
			enttask.MulticastSessionIDEQ(sess.ID),
			enttask.StateIn(enttask.StateQueued, enttask.StateActive, enttask.StateComplete),
		).Count(ctx)
		if err == nil {
			clientCount = taskCount
		}
	}

	// Check if all clients are ready for the current part.
	allReady := false
	m.tracker.mu.RLock()
	if rm, ok := m.tracker.readiness[sess.ID]; ok {
		if pm, ok := rm[part]; ok {
			allReady = len(pm) >= clientCount
		}
	}
	m.tracker.mu.RUnlock()

	if !allReady {
		return
	}

	// Check if we already have a sender running for this part.
	m.tracker.mu.RLock()
	_, senderRunning := m.tracker.completed[sess.ID][part] // use completed[part] as a sentinel for "sender launched"
	m.tracker.mu.RUnlock()

	if !senderRunning {
		m.tracker.mu.Lock()
		m.tracker.completed[sess.ID][part] = make(map[uuid.UUID]bool) // mark sender as launched
		m.tracker.mu.Unlock()

		partFile := filepath.Join(m.cfg.Storage.BasePath, img.Path, partFilename(part))
		port := sess.Portbase + (part * 100)

		slog.Info("multicast: launching udp-sender",
			"session", sess.ID, "part", part, "file", partFile,
			"portbase", port, "minReceivers", clientCount)

		go m.runUdpcast(ctx, sess.ID, partFile, part, port, clientCount)
	}
}

// runUdpcast runs udp-sender for a single partition and updates completion state.
func (m *MulticastManager) runUdpcast(ctx context.Context, sessionID uuid.UUID, partFile string, part, portbase, minReceivers int) {
	args := []string{
		"--file", partFile,
		"--portbase", fmt.Sprintf("%d", portbase),
		"--min-receivers", fmt.Sprintf("%d", minReceivers),
		"--nokbd",
		"--max-wait", "300",
	}

	cmd := exec.CommandContext(ctx, "udp-sender", args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		slog.Error("multicast: udp-sender failed",
			"session", sessionID, "part", part,
			"error", err, "output", string(out))
		return
	}

	slog.Info("multicast: partition sent", "session", sessionID, "part", part)

	// Advance to next partition once this one is sent.
	_ = m.db.MulticastSession.UpdateOneID(sessionID).
		SetCurrentPart(part + 1).
		Exec(ctx)
}

// ---------------------------------------------------------------------------
// Agent-facing coordination API (called from BootAPI handlers)
// ---------------------------------------------------------------------------

// SignalReady marks a task as ready for a given partition.
// Returns "start" with the portbase if all clients are ready, "wait" otherwise.
func (m *MulticastManager) SignalReady(sessionID, taskID uuid.UUID, part int) (action string, portbase int, err error) {
	sess, err := m.db.MulticastSession.Get(context.Background(), sessionID)
	if err != nil {
		return "", 0, fmt.Errorf("session not found: %w", err)
	}

	clientCount := sess.ClientCount
	if clientCount <= 0 {
		taskCount, err := m.db.Task.Query().Where(
			enttask.MulticastSessionIDEQ(sess.ID),
			enttask.StateIn(enttask.StateQueued, enttask.StateActive),
		).Count(context.Background())
		if err == nil && taskCount > 0 {
			clientCount = taskCount
		}
	}

	allReady := m.tracker.signalReady(sessionID, taskID, part, clientCount)
	portbase = sess.Portbase + (part * 100)

	if allReady {
		return "start", portbase, nil
	}
	return "wait", portbase, nil
}

// SignalPartDone marks a task as done for a given partition.
func (m *MulticastManager) SignalPartDone(sessionID, taskID uuid.UUID, part int) error {
	sess, err := m.db.MulticastSession.Get(context.Background(), sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	clientCount := sess.ClientCount
	if clientCount <= 0 {
		taskCount, err := m.db.Task.Query().Where(
			enttask.MulticastSessionIDEQ(sess.ID),
			enttask.StateIn(enttask.StateQueued, enttask.StateActive, enttask.StateComplete),
		).Count(context.Background())
		if err == nil && taskCount > 0 {
			clientCount = taskCount
		}
	}

	m.tracker.signalPartDone(sessionID, taskID, part, clientCount)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// imagePartitions is the JSONB shape stored in Image.Partitions.
type imagePartitions struct {
	PartCount           int      `json:"partCount"`
	ImageType           string   `json:"imageType"`
	FixedSizePartitions []int    `json:"fixedSizePartitions"`
	PartTypes           []string `json:"partTypes,omitempty"`
	PartNumbers         []int    `json:"partNumbers,omitempty"`
}

func parseImagePartitions(img *ent.Image) *imagePartitions {
	if len(img.Partitions) == 0 {
		return nil
	}
	var ip imagePartitions
	if err := json.Unmarshal(img.Partitions, &ip); err != nil {
		return nil
	}
	return &ip
}

func partFilename(part int) string {
	if part == 0 {
		return "ptable"
	}
	return fmt.Sprintf("part%d", part)
}

// MulticastCoordinator is the interface used by BootAPI handlers to
// communicate agent readiness and completion back to the manager.
type MulticastCoordinator interface {
	SignalReady(sessionID, taskID uuid.UUID, part int) (action string, portbase int, err error)
	SignalPartDone(sessionID, taskID uuid.UUID, part int) error
}

// compile-time interface check
var _ MulticastCoordinator = (*MulticastManager)(nil)
