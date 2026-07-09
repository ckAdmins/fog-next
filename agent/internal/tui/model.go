// Package tui provides a Bubble-Tea-based terminal dashboard for the
// fog imaging agent.  It renders a full-screen display showing connection status,
// task info, partition progress, and a scrollable log window.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ── data types shared with message payloads ──────────────────────────

// NetIface holds network interface state.
type NetIface struct {
	Name string
	MAC  string
	IPs  []string
	Up   bool
}

// PartStatus holds per-partition progress.
type PartStatus struct {
	Device    string
	FSType    string
	SizeBytes uint64
	Percent   int
	State     string // "pending", "active", "done", "skipped", "error"
}

// LogEntry is a single log line.
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

// ── message types ────────────────────────────────────────────────────

// PhaseMsg updates the current operation phase text.
type PhaseMsg string

// ConnectedMsg updates the server connection status.
type ConnectedMsg bool

// ServerMsg updates the server URL shown in the header.
type ServerMsg string

// TaskMsg updates task metadata (action, ids).
type TaskMsg struct {
	TaskID  string
	ImageID string
	Action  string
}

// DiskMsg updates the target disk info.
type DiskMsg struct {
	Device    string
	SizeBytes uint64
}

// ProgressMsg updates the overall progress bar.
type ProgressMsg struct {
	Pct        int
	DoneBytes  uint64
	TotalBytes uint64
	SpeedBpm   uint64
}

// PartitionsMsg replaces the entire partition list.
type PartitionsMsg []PartStatus

// PartitionMsg updates a single partition entry at Index.
type PartitionMsg struct {
	Index  int
	Status PartStatus
}

// InterfacesMsg replaces the NIC list.
type InterfacesMsg []NetIface

// InterfaceUpMsg marks a NIC as up or down by name.
type InterfaceUpMsg struct {
	Name string
	Up   bool
}

// InterfaceIPsMsg sets the IP addresses for a NIC by name.
type InterfaceIPsMsg struct {
	Name string
	IPs  []string
}

// PrimaryIPMsg sets the primary (first addressed) IP.
type PrimaryIPMsg string

// LogMsg appends a log entry to the TUI log panel.
type LogMsg struct {
	Level   string
	Message string
}

// VersionMsg sets the version info in the header.
type VersionMsg struct {
	Version   string
	Commit    string
	BuildDate string
}

// QuitMsg tells the TUI program to exit cleanly.
type QuitMsg struct{}

// ── model ────────────────────────────────────────────────────────────

// Model is the Bubble Tea application model.  All state mutations happen
// inside Update(), which runs on a single goroutine — no mutex needed.
type Model struct {
	Version   string
	Commit    string
	BuildDate string

	Interfaces []NetIface
	PrimaryIP  string

	ServerURL string
	Connected bool
	TaskID    string
	ImageID   string
	Action    string

	DiskDevice string
	DiskSize   uint64

	Phase string

	OverallPct int
	DoneBytes  uint64
	TotalBytes uint64
	SpeedBpm   uint64

	Partitions []PartStatus

	Logs []LogEntry

	StartedAt time.Time

	Width  int
	Height int
}

// NewModel returns a Model with sensible defaults.
func NewModel() Model {
	return Model{
		StartedAt: time.Now(),
		Width:     80,
		Height:    24,
	}
}

// Init returns the initial command.  The agent dashboard is
// externally-driven so there is nothing to kick off.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles every incoming message.  When the model is mutated
// Bubble Tea automatically re-renders the view.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case VersionMsg:
		m.Version = msg.Version
		m.Commit = msg.Commit
		m.BuildDate = msg.BuildDate

	case PhaseMsg:
		m.Phase = string(msg)

	case ConnectedMsg:
		m.Connected = bool(msg)

	case ServerMsg:
		m.ServerURL = string(msg)

	case TaskMsg:
		m.TaskID = msg.TaskID
		m.ImageID = msg.ImageID
		m.Action = msg.Action

	case DiskMsg:
		m.DiskDevice = msg.Device
		m.DiskSize = msg.SizeBytes

	case ProgressMsg:
		m.OverallPct = msg.Pct
		m.DoneBytes = msg.DoneBytes
		m.TotalBytes = msg.TotalBytes
		m.SpeedBpm = msg.SpeedBpm

	case PartitionsMsg:
		m.Partitions = make([]PartStatus, len(msg))
		copy(m.Partitions, msg)

	case PartitionMsg:
		if msg.Index >= 0 && msg.Index < len(m.Partitions) {
			m.Partitions[msg.Index] = msg.Status
		}

	case InterfacesMsg:
		m.Interfaces = make([]NetIface, len(msg))
		copy(m.Interfaces, msg)

	case InterfaceUpMsg:
		for i := range m.Interfaces {
			if m.Interfaces[i].Name == msg.Name {
				m.Interfaces[i].Up = msg.Up
				break
			}
		}

	case InterfaceIPsMsg:
		for i := range m.Interfaces {
			if m.Interfaces[i].Name == msg.Name {
				m.Interfaces[i].IPs = make([]string, len(msg.IPs))
				copy(m.Interfaces[i].IPs, msg.IPs)
				break
			}
		}

	case PrimaryIPMsg:
		m.PrimaryIP = string(msg)

	case LogMsg:
		m.Logs = append(m.Logs, LogEntry{
			Time:    time.Now(),
			Level:   msg.Level,
			Message: msg.Message,
		})
		if len(m.Logs) > 200 {
			m.Logs = m.Logs[len(m.Logs)-200:]
		}

	case QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}
