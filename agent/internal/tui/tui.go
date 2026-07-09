package tui

import tea "github.com/charmbracelet/bubbletea"

// NewProgram creates a Bubble Tea Program for the fog imaging agent TUI dashboard.
// The returned program must be run in a goroutine:
//
//	go p.Run()
//
// Messages are sent from the main goroutine via SendMsg(p, msg).  Call
// SendMsg(p, QuitMsg{}) to signal a clean shutdown, then wait for Run() to
// return to ensure the terminal is restored.
func NewProgram() *tea.Program {
	m := NewModel()
	return tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithoutSignalHandler(),
		tea.WithInput(nil), // no keyboard input — dashboard is display-only
	)
}

// SendMsg sends a message to the TUI program.  It uses a goroutine internally
// so that the caller is never blocked if the tea program has already exited.
// Safe to call with a nil program (no-op).
func SendMsg(p *tea.Program, msg tea.Msg) {
	if p == nil {
		return
	}
	go func() { p.Send(msg) }()
}
