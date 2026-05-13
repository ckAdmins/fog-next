// fos-agent is the imaging agent for the fos-alpine initramfs.
// It runs as a regular process (not PID 1) on a terminal, launched
// from the fos-autologin wrapper script.  OpenRC handles init,
// networking, udev, and signal reaping — the agent just focuses on
// imaging operations.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbletea"

	"github.com/nemvince/fos-agent/internal/actions"
	fogapi "github.com/nemvince/fos-agent/internal/api"
	"github.com/nemvince/fos-agent/internal/cmdline"
	"github.com/nemvince/fos-agent/internal/netup"
	"github.com/nemvince/fos-agent/internal/tui"
	"github.com/nemvince/fos-agent/internal/version"
)

// Exit codes communicated back to the fos-autologin wrapper.
const (
	exitSuccess    = 0 // done → wrapper will poweroff
	exitError      = 1 // failed → wrapper drops to shell
	exitDebug      = 2 // debug action → wrapper drops to shell
	exitRegisterOK = 3 // register completed → wrapper will poweroff
)

func main() {
	// Step 1: Parse kernel cmdline.
	params, err := cmdline.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fos-agent: cannot read kernel cmdline: %v\n", err)
		os.Exit(exitError)
	}
	if params.FogServer == "" {
		fmt.Fprintf(os.Stderr, "fos-agent: fog_server not set on kernel command line\n")
		os.Exit(exitError)
	}

	// Step 2: Start the Bubble Tea TUI in a goroutine.  It takes over
	// the terminal (alternate screen + raw mode) and renders on every
	// message sent via SendMsg().  Skip the TUI for debug-* actions so
	// the terminal remains readable and partclone stderr is visible.
	var p *tea.Program
	var teaDone chan struct{}
	if params.FogTUI && !strings.HasPrefix(params.FogAction, "debug") {
		p = tui.NewProgram()
		teaDone = make(chan struct{})
		go func() {
			defer close(teaDone)
			defer func() {
				if r := recover(); r != nil {
					kmsgPrintf("fos-agent: TUI panic: %v", r)
					fmt.Fprintf(os.Stderr, "fos-agent: TUI panic: %v\n", r)
				}
			}()
			if _, runErr := p.Run(); runErr != nil {
				kmsgPrintf("fos-agent: TUI error: %v", runErr)
				fmt.Fprintf(os.Stderr, "fos-agent: TUI error: %v\n", runErr)
			}
		}()
		tui.SendMsg(p, tui.VersionMsg{
			Version:   version.Version,
			Commit:    version.Commit,
			BuildDate: version.BuildDate,
		})
		tui.SendMsg(p, tui.LogMsg{Level: "INFO", Message: "fos-agent starting"})
	}

	// Step 3: Set up logging.  When the TUI owns the terminal, log
	// output goes to kmsg and the log file only.  Without a TUI,
	// stderr (which is the terminal) is also a destination.
	remoteHandler := setupLogging(p != nil)

	slog.Info("fos-agent starting",
		"version", version.Version,
		"commit", version.Commit,
		"buildDate", version.BuildDate,
	)

	if rel, err := os.ReadFile("/etc/fos-release"); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(rel)), "\n") {
			if line != "" {
				slog.Info("fos-release", "entry", strings.TrimSpace(line))
			}
		}
	}

	slog.Info("cmdline parsed",
		"server", params.FogServer,
		"action", params.FogAction,
		"host", params.FogHost,
		"tui", params.FogTUI,
		"auto", params.FogAuto,
	)

	if p != nil {
		teeLogging(p)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	slog.Info("signal handlers installed, entering network wait")

	// Step 4: Wait for the network to be ready.  OpenRC handles DHCP,
	// so we just poll until an interface has an IP address.
	netCtx, netCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer netCancel()

	slog.Info("calling WaitForNetwork", "timeout", "3m0s")
	primaryMAC, err := netup.WaitForNetwork(netCtx, p)
	if err != nil {
		slog.Error("network wait failed", "err", err)
		exitAgent(exitError, remoteHandler, p, teaDone)
	}
	slog.Info("network is up", "primaryMAC", primaryMAC)

	// Ensure udev has finished coldplugging all hardware so disks are
	// visible before any imaging action tries to enumerate them.
	slog.Info("waiting for udev to settle...")
	exec.Command("udevadm", "settle", "-t", "30").Run()
	slog.Info("udev settled, proceeding with handshake")

	client := fogapi.New(params.FogServer, p)

	// Collect all MACs for the handshake.
	nics, _ := netup.ListNICs()
	macs := make([]string, 0, len(nics))
	for _, n := range nics {
		macs = append(macs, n.MAC)
	}
	reorder(macs, primaryMAC)

	// Step 5: Handshake with fog-next server.
	slog.Info("starting handshake", "macs", macs)
	resp, err := client.Handshake(ctx, fogapi.HandshakeRequest{MACs: macs})
	if err != nil {
		slog.Error("handshake failed", "err", err)
		exitAgent(exitError, remoteHandler, p, teaDone)
	}
	slog.Info("handshake succeeded",
		"action", resp.Action,
		"taskId", resp.TaskID,
		"imageId", resp.ImageID,
	)

	// Activate remote log forwarding now that we have a boot token and task ID.
	if resp.TaskID != "" {
		remoteHandler.SetClient(client, resp.TaskID)
	}

	// Override action from cmdline if set (e.g. fog_action=debug).
	if params.FogAction != "" && params.FogAction != resp.Action {
		slog.Info("cmdline overrides action", "from", resp.Action, "to", params.FogAction)
		resp.Action = params.FogAction
	}

	// Step 6: Dispatch the action.
	slog.Info("dispatching action", "action", resp.Action)
	switch resp.Action {
	case "register":
		if err := actions.Register(ctx, client, macs, p); err != nil {
			slog.Error("register failed", "err", err)
		}
		exitAgent(exitRegisterOK, remoteHandler, p, teaDone)

	case "debug":
		actions.Debug(ctx, resp)
		exitAgent(exitDebug, remoteHandler, p, teaDone)

	case "debug_deploy":
		slog.Info("running debug_deploy — verbose deploy, drops to shell")
		if err := actions.DebugDeploy(ctx, client, resp, p); err != nil {
			slog.Error("debug_deploy failed", "err", err)
		}
		exitAgent(exitDebug, remoteHandler, p, teaDone)

	case "debug_capture":
		slog.Info("running debug_capture — verbose capture, drops to shell")
		if err := actions.DebugCapture(ctx, client, resp, p); err != nil {
			slog.Error("debug_capture failed", "err", err)
		}
		exitAgent(exitDebug, remoteHandler, p, teaDone)

	default:
		slog.Info("entering action dispatcher", "action", resp.Action)
		if err := actions.Dispatch(ctx, client, resp, p); err != nil {
			slog.Error("action failed", "err", err)
			exitAgent(exitError, remoteHandler, p, teaDone)
		}
		slog.Info("action completed successfully", "action", resp.Action)
		exitAgent(exitSuccess, remoteHandler, p, teaDone)
	}
}

// exitAgent flushes any remaining log entries to the server, tears down
// the TUI (if active), and calls os.Exit.  Always use this instead of a
// bare os.Exit so error logs are visible on the fog-next frontend.
func exitAgent(code int, rh *fogapi.RemoteHandler, p *tea.Program, done chan struct{}) {
	if rh != nil {
		rh.FlushAll()
	}
	doQuit(p, done)
	os.Exit(code)
}

// doQuit signals the TUI program to quit and blocks (with a timeout) for
// the terminal to be restored.  Safe to call with a nil program (no-op).
func doQuit(p *tea.Program, done chan struct{}) {
	if p == nil {
		return
	}
	tui.SendMsg(p, tui.QuitMsg{})
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			slog.Warn("timed out waiting for TUI to quit")
			kmsgPrintf("fos-agent: timeout waiting for TUI shutdown")
		}
	}
}

// kmsgPrintf writes a message directly to /dev/kmsg.  Useful for
// diagnostics when the TUI owns the terminal and stderr is invisible.
func kmsgPrintf(format string, args ...any) {
	f, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", args...)
}

// setupLogging configures slog output.  When the TUI is active we skip
// stderr (the terminal) to avoid corrupting TUI frames.  Logs always go
// to /dev/kmsg and /var/log/fos-agent.log.
func setupLogging(tuiActive bool) *fogapi.RemoteHandler {
	var writers []io.Writer

	if !tuiActive {
		writers = append(writers, os.Stderr)
	}

	if kmsg, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0); err == nil {
		writers = append(writers, kmsg)
	}

	if err := os.MkdirAll("/var/log", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "fos-agent: /var/log: %v\n", err)
	}
	logFile, logErr := os.OpenFile("/var/log/fos-agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logErr != nil {
		fmt.Fprintf(os.Stderr, "fos-agent: /var/log/fos-agent.log: %v\n", logErr)
	} else {
		writers = append(writers, logFile)
	}

	var w io.Writer
	switch len(writers) {
	case 0:
		w = io.Discard
	case 1:
		w = writers[0]
	default:
		w = io.MultiWriter(writers...)
	}

	consoleHandler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	rh := fogapi.NewRemoteHandler(consoleHandler)
	slog.SetDefault(slog.New(rh))
	return rh
}

// teeLogging wraps the current slog handler with one that also sends log
// entries to the Bubble Tea TUI via program.Send().
func teeLogging(p *tea.Program) {
	existing := slog.Default().Handler()
	wrapped := &tuiLogHandler{delegate: existing, program: p}
	slog.SetDefault(slog.New(wrapped))
}

// tuiLogHandler wraps a slog.Handler and also sends entries to the TUI.
type tuiLogHandler struct {
	delegate slog.Handler
	program  *tea.Program
}

func (h *tuiLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.delegate.Enabled(ctx, level)
}

func (h *tuiLogHandler) Handle(ctx context.Context, r slog.Record) error {
	r2 := r.Clone()
	msg := r2.Message
	r2.Attrs(func(a slog.Attr) bool {
		msg += " " + a.Key + "=" + a.Value.String()
		return true
	})
	tui.SendMsg(h.program, tui.LogMsg{Level: r2.Level.String(), Message: msg})
	return h.delegate.Handle(ctx, r)
}

func (h *tuiLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &tuiLogHandler{delegate: h.delegate.WithAttrs(attrs), program: h.program}
}

func (h *tuiLogHandler) WithGroup(name string) slog.Handler {
	return &tuiLogHandler{delegate: h.delegate.WithGroup(name), program: h.program}
}

// reorder moves the element matching primary to the front of the slice.
func reorder(macs []string, primary string) {
	for i, m := range macs {
		if m == primary && i != 0 {
			macs[0], macs[i] = macs[i], macs[0]
			return
		}
	}
}
