package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ───────────────────────────────────────────────────────────

var (
	styleHeader    = lipgloss.NewStyle().Bold(true)
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleGreen     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleRed       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleYellow    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleCyan      = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleGray      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	sectionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Shared progress bar model (20-char mini bars for partitions use
	// the same gradient but are rendered with ViewAs).
	progressBar = progress.New(progress.WithScaledGradient("#04B575", "#04B575"))
)

// ── View ─────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.Width < 80 {
		m.Width = 80
	}

	var sections []string

	sections = append(sections, m.renderHeader())
	sections = append(sections, "")
	sections = append(sections, m.renderStatus())
	sections = append(sections, "")
	sections = append(sections, m.renderProgress())
	sections = append(sections, "")

	if len(m.Partitions) > 0 {
		sections = append(sections, m.renderPartitions())
		sections = append(sections, "")
	}

	sections = append(sections, m.renderLogs())

	joined := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return joined
}

// ── header ───────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	left := fmt.Sprintf("fog-agent %s (%s)", m.Version, m.Commit)
	right := m.BuildDate
	if right == "" || right == "unknown" {
		right = time.Now().UTC().Format("2006-01-02T15:04Z")
	}
	return styleHeader.
		Width(m.Width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top,
			left,
			lipgloss.NewStyle().Width(m.Width-len(left)).Align(lipgloss.Right).Render(right),
		))
}

// ── status ───────────────────────────────────────────────────────────

func (m Model) renderStatus() string {
	icon, iconStyle := statusIcon(m.Connected)

	serverLabel := "Server   "
	serverVal := truncate(m.ServerURL, m.Width-35)

	lines := []string{
		joinStatusLine(serverLabel, serverVal, iconStyle.Render(icon), m.Width),
		fmt.Sprintf("Task     %-8s  ID  %s", m.Action, m.TaskID),
		fmt.Sprintf("Image    %s", m.ImageID),
		diskLine(m),
		fmt.Sprintf("Phase    %s", m.Phase),
	}
	return strings.Join(lines, "\n")
}

func statusIcon(connected bool) (string, lipgloss.Style) {
	if connected {
		return "✓ Connected", styleGreen
	}
	return "◌ waiting", styleYellow
}

func joinStatusLine(label, value, icon string, width int) string {
	spacer := width - len(label) - len(value) - len(icon) - 2
	if spacer < 1 {
		spacer = 1
	}
	return label + value + strings.Repeat(" ", spacer) + icon
}

func diskLine(m Model) string {
	if m.DiskDevice == "" {
		return "Disk"
	}
	return fmt.Sprintf("Disk     %s (%s)", m.DiskDevice, formatBytes(m.DiskSize))
}

// ── progress ─────────────────────────────────────────────────────────

func (m Model) renderProgress() string {
	header := sectionStyle.Render("── Progress " + strings.Repeat("─", max(0, m.Width-13)))

	pct := clamp(m.OverallPct, 0, 100)
	bar := progressBar.ViewAs(float64(pct) / 100.0)
	barLine := fmt.Sprintf("[%s] %3d%%", bar, pct)

	parts := make([]string, 0, 3)
	if m.TotalBytes > 0 {
		parts = append(parts, fmt.Sprintf("%s / %s",
			formatBytes(m.DoneBytes), formatBytes(m.TotalBytes)))
	}
	if m.SpeedBpm > 0 {
		parts = append(parts, fmt.Sprintf("%s/min", formatSpeed(m.SpeedBpm)))
	}
	if m.OverallPct > 0 && m.OverallPct < 100 && !m.StartedAt.IsZero() {
		elapsed := time.Since(m.StartedAt)
		totalEst := time.Duration(float64(elapsed) / (float64(m.OverallPct) / 100.0))
		remaining := totalEst - elapsed
		if remaining > 0 {
			parts = append(parts, fmt.Sprintf("ETA %s", remaining.Round(time.Second)))
		}
	}
	statsLine := "  " + strings.Join(parts, "  │  ")

	return header + "\n" + barLine + "\n" + statsLine
}

// ── partitions ───────────────────────────────────────────────────────

// FSDisplayName returns a human-readable filesystem label for display in the TUI.
// Raw/internal names like "dd", "?", or "" are mapped to descriptive strings.
func FSDisplayName(raw string) string {
	if raw == "" || raw == "?" {
		return "unknown"
	}
	switch strings.ToLower(raw) {
	case "dd":
		return "raw"
	case "hfsplus", "hfs+":
		return "HFS+"
	case "vfat", "fat12", "fat16", "fat32":
		return "FAT"
	case "ext2":
		return "ext2"
	case "ext3":
		return "ext3"
	case "ext4", "ext4dev", "extfs":
		return "ext4"
	default:
		return raw
	}
}

const maxPartRows = 16

func (m Model) renderPartitions() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("── Partitions " + strings.Repeat("─", max(0, m.Width-15))))
	b.WriteString("\n")

	for i, p := range m.Partitions {
		if i >= maxPartRows {
			more := len(m.Partitions) - maxPartRows
			b.WriteString(fmt.Sprintf("  ... and %d more partition(s)\n", more))
			break
		}
		renderPartitionRow(&b, p)
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderPartitionRow(b *strings.Builder, p PartStatus) {
	dev := fmt.Sprintf("%-16s", p.Device)
	fs := fmt.Sprintf("%-8s", FSDisplayName(p.FSType))
	size := fmt.Sprintf("%-10s", formatBytes(p.SizeBytes))

	pct := clamp(p.Percent, 0, 100)
	miniBar := progressBar.ViewAs(float64(pct) / 100.0)
	bar := fmt.Sprintf("[%s]", miniBar)

	state, stateStyle := partitionState(p.State)

	b.WriteString(fmt.Sprintf("  %s %s %s %s %3d%%  %s\n",
		dev, fs, size, bar, pct,
		stateStyle.Render(state),
	))
}

func partitionState(state string) (string, lipgloss.Style) {
	switch state {
	case "done":
		return "✓", styleGreen
	case "active":
		return "⟳", styleCyan
	case "error":
		return "✗", styleRed
	case "skipped":
		return "⊘", styleYellow
	default:
		return "○", styleGray
	}
}

// ── logs ─────────────────────────────────────────────────────────────

func (m Model) renderLogs() string {
	header := sectionStyle.Render("── Log " + strings.Repeat("─", max(0, m.Width-8)))

	// Calculate how many lines remain for logs.  This is approximate
	// since we can't know the exact row count until after layout, but
	// the TUI fills the screen and overflow scrolls naturally.
	logCap := max(2, (m.Height/5)*3) // ~60% of screen
	entries := m.Logs
	start := 0
	if len(entries) > logCap {
		start = len(entries) - logCap
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	shown := 0
	for i := start; i < len(entries) && shown < logCap; i++ {
		b.WriteString("  ")
		b.WriteString(entries[i].Time.Format("15:04:05"))
		b.WriteString("  ")
		b.WriteString(truncate(entries[i].Message, max(0, m.Width-12)))
		b.WriteString("\n")
		shown++
	}
	return strings.TrimRight(b.String(), "\n")
}

// ── helpers ──────────────────────────────────────────────────────────

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 1 {
		return ""
	}
	return s[:maxLen-1] + string('…')
}

func formatBytes(n uint64) string {
	switch {
	case n >= 1<<40:
		return fmt.Sprintf("%.1f TiB", float64(n)/(1<<40))
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func formatSpeed(bpm uint64) string {
	mbpm := float64(bpm) / (8 * 1024 * 1024)
	if mbpm >= 1024 {
		return fmt.Sprintf("%.1f GiB/min", mbpm/1024)
	}
	return fmt.Sprintf("%.1f MiB/min", mbpm)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
