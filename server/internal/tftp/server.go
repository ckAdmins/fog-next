// Package tftp provides an embedded TFTP server for PXE/iPXE booting.
// It serves boot files (iPXE binaries, memdisk, kernels) from a configurable
// root directory, enabling BIOS and UEFI network boot without an external
// tftp-hpa or xinetd installation.
package tftp

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	tftp "github.com/pin/tftp/v3"

	"github.com/nemvince/fog-next/internal/config"
)

// Server wraps the pin/tftp server with FOG-specific configuration.
type Server struct {
	cfg  *config.Config
	tftp *tftp.Server
}

// New creates a configured TFTP server that serves files from cfg.TFTP.RootDir.
func New(cfg *config.Config) *Server {
	s := &Server{cfg: cfg}
	s.tftp = tftp.NewServer(s.readHandler, nil)
	s.tftp.EnableSinglePort()
	return s
}

// ListenAndServe binds to the configured UDP address and blocks until the
// server encounters a fatal error.  It is safe to call from a goroutine.
func (s *Server) ListenAndServe() error {
	if !s.cfg.TFTP.Enabled {
		slog.Info("tftp server disabled by config")
		// Block forever to keep the goroutine alive without consuming CPU.
		select {}
	}
	slog.Info("tftp server listening", "addr", s.cfg.TFTP.Listen)
	return s.tftp.ListenAndServe(s.cfg.TFTP.Listen)
}

// readHandler is called by the pin/tftp library for every incoming TFTP GET
// request.  It returns a file reader for the requested path within RootDir.
// The special file "autoexec.ipxe" is generated dynamically from the server
// config so iPXE firmware auto-chains to the FOG boot menu without needing a
// static file on disk.
func (s *Server) readHandler(filename string, rf io.ReaderFrom) error {
	// autoexec.ipxe: iPXE executes this automatically after loading.
	// Generate it from config so it always points at the correct base_url.
	if filepath.Clean("/"+filename) == "/autoexec.ipxe" {
		script := fmt.Sprintf("#!ipxe\nchain %s/fog/boot?mac=${net0/mac}\n", s.cfg.Server.BaseURL)
		buf := bytes.NewBufferString(script)
		rf.(tftp.OutgoingTransfer).SetSize(int64(buf.Len()))
		n, err := rf.ReadFrom(buf)
		if err != nil {
			slog.Error("tftp: autoexec send error", "error", err)
			return err
		}
		slog.Debug("tftp: served autoexec.ipxe", "bytes", n)
		return nil
	}

	// Sanitise the path — prevent directory traversal attacks.
	clean := filepath.Clean("/" + filename)
	full := filepath.Join(s.cfg.TFTP.RootDir, clean)

	file, err := os.Open(filepath.Clean(full))
	if err != nil {
		slog.Warn("tftp: file not found", "file", filename, "error", err)
		return err
	}
	defer file.Close()


	stat, err := file.Stat()
	if err == nil {
		slog.Debug("tftp: file opened", "file", filename, "size", stat.Size(), "error", err)
		rf.(tftp.OutgoingTransfer).SetSize(stat.Size())
	} else {
		// Stat errors are non-fatal for TFTP; the transfer can proceed without a size.
		slog.Warn("tftp: file stat error", "file", filename, "error", err)
	}

	n, err := rf.ReadFrom(file)
	if err != nil {
		slog.Error("tftp: read error", "file", filename, "error", err)
		return err
		
	}
	slog.Debug("tftp: served", "file", filename, "bytes", n)
	return nil
}
