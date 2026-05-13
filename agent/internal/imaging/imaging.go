// Package imaging wraps partclone for streaming image capture and restore.
// All image data flows through stdin/stdout — no temporary files are written.
//
// Progress is read from partclone's stderr (parsing the standard
// "Completed: XX.XX%" format).  The agent runs partclone with -F (force
// progress) and -f 1 (one-second refresh) so progress lines are emitted
// regularly even when stderr is a pipe.  musl libc (used by the Buildroot
// toolchain) leaves stderr unbuffered by default, so no stdbuf wrapper
// is required.
package imaging

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ProgressFunc is called with percent complete and bits-per-minute.
type ProgressFunc func(percent int, bpm int64)

// stderrRing is a bounded ring buffer that captures the tail of partclone's
// stderr so it can be included in error messages on failure.
type stderrRing struct {
	buf  []byte
	pos  int
	full bool
}

func newStderrRing(size int) *stderrRing {
	if size < 256 {
		size = 256
	}
	return &stderrRing{buf: make([]byte, size)}
}

func (r *stderrRing) Write(p []byte) (int, error) {
	n := len(p)
	for _, b := range p {
		r.buf[r.pos] = b
		r.pos = (r.pos + 1) % len(r.buf)
	}
	if !r.full && r.pos == 0 {
		r.full = true
	}
	return n, nil
}

func (r *stderrRing) String() string {
	if r.full {
		return string(r.buf[r.pos:]) + string(r.buf[:r.pos])
	}
	return string(r.buf[:r.pos])
}

// runPartclone starts partclone with the given arguments, feeds stdin/src and
// captures stdout/dst, then reads stderr for progress while also buffering
// its tail for error diagnostics.  Returns nil on success, or an error that
// includes the captured stderr tail.
func runPartclone(ctx context.Context, bin string, args []string, src io.Reader, dst io.Writer, progress ProgressFunc, teeStderr bool) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = src
	if dst != nil {
		cmd.Stdout = dst
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", bin, err)
	}

	capture := newStderrRing(4096)
	done := make(chan struct{})
	go func() {
		defer close(done)
		var r io.Reader = io.TeeReader(stderrPipe, capture)
		if teeStderr {
			r = io.TeeReader(r, os.Stderr)
		}
		readStderrProgress(r, progress)
	}()

	waitErr := cmd.Wait()
	<-done

	if waitErr != nil {
		tail := strings.TrimSpace(capture.String())
		if tail == "" {
			tail = "(no stderr output)"
		}
		return fmt.Errorf("%s: %w\nstderr: %s", bin, waitErr, tail)
	}

	// Log the last few lines of stderr even on success — useful for
	// diagnosing silent failures (e.g. partclone exits 0 but produced
	// no data because the source device was empty).
	if tail := strings.TrimSpace(capture.String()); tail != "" {
		slog.Info("partclone completed", "bin", bin, "stderr", tail)
	} else {
		slog.Info("partclone completed", "bin", bin, "stderr", "(empty)")
	}
	return nil
}

// Restore streams src into partclone.restore targeting the given device.
// fs is the filesystem type string passed to partclone (e.g. "ext4", "ntfs").
// If teeStderr is true, raw partclone stderr is also written to os.Stderr
// (useful when no TUI is active and the user is watching the console).
func Restore(ctx context.Context, device, fs string, src io.Reader, progress ProgressFunc, teeStderr bool) error {
	bin := partcloneBin(fs)
	slog.Info("partclone restore", "device", device, "fs", fs, "bin", bin)
	var args []string
	if bin == "partclone.dd" {
		args = []string{"-s", "-", "-O", device, "-F", "-f", "1"}
	} else {
		// -r = restore, -s - = source from stdin, -O = output+overwrite.
		// Matches legacy: partclone.restore -O ${target} -f 1
		args = []string{"-r", "-s", "-", "-O", device, "-F", "-f", "1"}
	}
	return runPartclone(ctx, bin, args, src, nil, progress, teeStderr)
}

// Clone captures device to dst, streaming raw partclone output.
// If teeStderr is true, raw partclone stderr is also written to os.Stderr.
func Clone(ctx context.Context, device, fs string, dst io.Writer, progress ProgressFunc, teeStderr bool) error {
	bin := partcloneBin(fs)
	slog.Info("partclone clone", "device", device, "fs", fs, "bin", bin)
	var args []string
	if bin == "partclone.dd" {
		args = []string{"-s", device, "-o", "-", "-F", "-f", "1"}
	} else {
		// -c = clone, -s = source device, -o - = output to stdout
		args = []string{"-c", "-s", device, "-o", "-", "-F", "-f", "1"}
	}
	return runPartclone(ctx, bin, args, nil, dst, progress, teeStderr)
}

// ------------------------------------------------------------------
// stderr progress parsing
// ------------------------------------------------------------------

// readStderrProgress reads from r (partclone's stderr), splits on \r,
// extracts Completed: XX.XX% and speed (XX.XXGB/min), and calls fn for
// each parsed update.  ANSI escape sequences are stripped before parsing.
func readStderrProgress(r io.Reader, fn ProgressFunc) {
	if fn == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	scanner.Split(scanCRLines)
	for scanner.Scan() {
		line := scanner.Text()
		line = stripANSI(line)
		pct, bpm := parseProgressLine(line)
		if pct >= 0 {
			fn(pct, bpm)
		}
	}
}

// scanCRLines is a bufio.SplitFunc that splits on carriage return (\r).
func scanCRLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// stripANSI removes ANSI escape sequences (CSI codes like \x1b[0m, \x1b[A)
// from s.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

// parseProgressLine extracts percent (0-100) and bits-per-minute from a
// partclone progress line.  Returns (-1, 0) if the line does not contain
// a progress update.
//
// Expected format (partclone non-ncurses, with -F):
//
//	Elapsed: 00:00:01, Remaining: 00:00:00, Completed: 37.42%  1.23GB/min
func parseProgressLine(line string) (percent int, bpm int64) {
	pctRe := regexp.MustCompile(`Completed?:\s*([\d.]+)\s*%`)
	m := pctRe.FindStringSubmatch(line)
	if m == nil {
		return -1, 0
	}
	pct, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return -1, 0
	}
	bpm = parseSpeedFromLine(line)
	return int(pct), bpm
}

// parseSpeedFromLine extracts bits-per-minute from a partclone progress
// line.  Looks for patterns like "1.23GB/min" or "512.34MB/min".
func parseSpeedFromLine(line string) int64 {
	speedRe := regexp.MustCompile(`(\d+\.?\d*)\s*(GB|MB|KB|TB)/min`)
	m := speedRe.FindStringSubmatch(line)
	if m == nil {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	var multiplier float64
	switch m[2] {
	case "KB":
		multiplier = 8 * 1e3
	case "MB":
		multiplier = 8 * 1e6
	case "GB":
		multiplier = 8 * 1e9
	case "TB":
		multiplier = 8 * 1e12
	default:
		return 0
	}
	return int64(val * multiplier)
}

// ------------------------------------------------------------------
// partclone binary selection
// ------------------------------------------------------------------

// partcloneBin maps a filesystem label to the appropriate partclone binary.
func partcloneBin(fs string) string {
	switch strings.ToLower(fs) {
	case "ext2", "ext3", "ext4":
		return "partclone.ext4"
	case "ntfs":
		return "partclone.ntfs"
	case "fat16", "fat32", "vfat":
		return "partclone.fat"
	case "xfs":
		return "partclone.xfs"
	case "btrfs":
		return "partclone.btrfs"
	default:
		return "partclone.dd"
	}
}

// CanShrink reports whether the filesystem type can be non-destructively
// shrunk before capture to produce a smaller image.
func CanShrink(fs string) bool {
	switch strings.ToLower(fs) {
	case "ext2", "ext3", "ext4", "ntfs", "btrfs":
		return true
	default:
		return false
	}
}

// Shrink reduces the filesystem on device to its minimum safe size so that
// less data needs to be transferred during capture.  Returns (true, nil) if
// the filesystem was shrunk, (false, nil) if shrinking is not supported for
// this filesystem (caller should mark the partition as fixed-size), or
// (false, err) on a hard failure.
func Shrink(ctx context.Context, device, fs string) (shrunk bool, err error) {
	switch strings.ToLower(fs) {
	case "ntfs":
		return shrinkNTFS(ctx, device)
	case "ext2", "ext3", "ext4":
		return shrinkEXT(ctx, device)
	case "btrfs":
		return shrinkBTRFS(ctx, device)
	default:
		return false, nil
	}
}

// Expand grows the filesystem on device to fill the entire partition.  It
// is safe to call even if the partition was not previously shrunk.
// Non-fatal: unsupported filesystem types are silently skipped.
func Expand(ctx context.Context, device, fs string) error {
	switch strings.ToLower(fs) {
	case "ntfs":
		return expandNTFS(ctx, device)
	case "ext2", "ext3", "ext4":
		return expandEXT(ctx, device)
	case "btrfs":
		return expandBTRFS(ctx, device)
	case "xfs":
		return expandXFS(ctx, device)
	case "f2fs":
		return expandF2FS(ctx, device)
	default:
		// FAT, swap, dd — nothing to do.
		return nil
	}
}

// ------------------------------------------------------------------
// NTFS
// ------------------------------------------------------------------

func shrinkNTFS(ctx context.Context, device string) (bool, error) {
	// Dry-run first to confirm ntfsresize thinks it can shrink.
	out, err := exec.CommandContext(ctx, "ntfsresize", "-fns", device).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ntfsresize dry-run: %w\n%s", err, out)
	}
	// Actually shrink. -f force, -s 0 (minimum size).
	out, err = exec.CommandContext(ctx, "ntfsresize", "-f", "--size", "0", device).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ntfsresize shrink: %w\n%s", err, out)
	}
	return true, nil
}

func expandNTFS(ctx context.Context, device string) error {
	// ntfsresize with no --size fills the partition automatically.
	out, err := exec.CommandContext(ctx, "ntfsresize", "-f", "-b", "-P", device).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ntfsresize expand: %w\n%s", err, out)
	}
	return nil
}

// NtfsFixDirty clears the NTFS dirty bit and journal on a freshly-restored
// NTFS partition.  Without this, Windows will run chkdsk on the first boot
// after every deploy.  Best-effort — errors are logged, not returned.
func NtfsFixDirty(device string) {
	// -b: clear bad sector list, -d: clear dirty flag
	out, err := exec.Command("ntfsfix", "-b", "-d", device).CombinedOutput()
	if err != nil {
		slog.Warn("ntfsfix failed (non-fatal)", "device", device, "err", err, "output", string(out))
	} else {
		slog.Info("ntfsfix: cleared dirty bit", "device", device)
	}
}

// ------------------------------------------------------------------
// EXT2/3/4
// ------------------------------------------------------------------

func shrinkEXT(ctx context.Context, device string) (bool, error) {
	// Force fsck before resize to ensure a clean filesystem.
	if out, err := exec.CommandContext(ctx, "e2fsck", "-fy", device).CombinedOutput(); err != nil {
		return false, fmt.Errorf("e2fsck before shrink: %w\n%s", err, out)
	}
	// Shrink to minimum size.
	out, err := exec.CommandContext(ctx, "resize2fs", "-M", device).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("resize2fs shrink: %w\n%s", err, out)
	}
	return true, nil
}

func expandEXT(ctx context.Context, device string) error {
	// resize2fs with no size argument fills the partition.
	out, err := exec.CommandContext(ctx, "resize2fs", device).CombinedOutput()
	if err != nil {
		return fmt.Errorf("resize2fs expand: %w\n%s", err, out)
	}
	return nil
}

// ------------------------------------------------------------------
// BTRFS
// ------------------------------------------------------------------

// btrfsMountPoint is the temporary mount used for btrfs resize operations.
const btrfsMountPoint = "/tmp/fog-btrfs-resize"

func shrinkBTRFS(ctx context.Context, device string) (bool, error) {
	if err := os.MkdirAll(btrfsMountPoint, 0o700); err != nil {
		return false, fmt.Errorf("btrfs shrink: mkdir: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "mount", "-t", "btrfs", device, btrfsMountPoint).CombinedOutput(); err != nil {
		return false, fmt.Errorf("btrfs shrink: mount: %w\n%s", err, out)
	}
	defer func() {
		_ = exec.CommandContext(ctx, "umount", btrfsMountPoint).Run()
	}()
	// btrfs resize to minimum.  "min" keyword requires btrfs-progs >= 5.10.
	out, err := exec.CommandContext(ctx, "btrfs", "filesystem", "resize", "1:min", btrfsMountPoint).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("btrfs resize min: %w\n%s", err, out)
	}
	return true, nil
}

func expandBTRFS(ctx context.Context, device string) error {
	if err := os.MkdirAll(btrfsMountPoint, 0o700); err != nil {
		return fmt.Errorf("btrfs expand: mkdir: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "mount", "-t", "btrfs", device, btrfsMountPoint).CombinedOutput(); err != nil {
		return fmt.Errorf("btrfs expand: mount: %w\n%s", err, out)
	}
	defer func() {
		_ = exec.CommandContext(ctx, "umount", btrfsMountPoint).Run()
	}()
	out, err := exec.CommandContext(ctx, "btrfs", "filesystem", "resize", "max", btrfsMountPoint).CombinedOutput()
	if err != nil {
		return fmt.Errorf("btrfs resize max: %w\n%s", err, out)
	}
	return nil
}

// ------------------------------------------------------------------
// XFS (can grow but not shrink)
// ------------------------------------------------------------------

func expandXFS(ctx context.Context, device string) error {
	mp := "/tmp/fog-xfs-grow"
	if err := os.MkdirAll(mp, 0o700); err != nil {
		return fmt.Errorf("xfs grow: mkdir: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "mount", "-t", "xfs", device, mp).CombinedOutput(); err != nil {
		return fmt.Errorf("xfs grow: mount: %w\n%s", err, out)
	}
	defer func() {
		_ = exec.CommandContext(ctx, "umount", mp).Run()
	}()
	out, err := exec.CommandContext(ctx, "xfs_growfs", mp).CombinedOutput()
	if err != nil {
		return fmt.Errorf("xfs_growfs: %w\n%s", err, out)
	}
	return nil
}

// ------------------------------------------------------------------
// F2FS (can grow but not shrink)
// ------------------------------------------------------------------

func expandF2FS(ctx context.Context, device string) error {
	out, err := exec.CommandContext(ctx, "resize.f2fs", device).CombinedOutput()
	if err != nil {
		return fmt.Errorf("resize.f2fs: %w\n%s", err, out)
	}
	return nil
}

// ------------------------------------------------------------------
// Bitlocker detection
// ------------------------------------------------------------------

// IsBitlockerEncrypted returns true if the partition appears to be a
// Bitlocker-encrypted volume.  The check reads the NTFS boot sector magic
// bytes at offset 3 (which Bitlocker replaces with "-FVE-FS-").
func IsBitlockerEncrypted(device string) bool {
	// Prefer blkid probe (fastest, no reads beyond the boot sector).
	out, err := exec.Command("blkid", "-p", "-o", "value", "-s", "TYPE", device).Output()
	if err == nil && strings.TrimSpace(string(out)) == "BitLocker" {
		return true
	}
	// Fallback: read the OEM ID field at bytes 3–10 of the boot sector.
	f, err := os.Open(device)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 11)
	if _, err := f.Read(buf); err != nil {
		return false
	}
	return string(buf[3:11]) == "-FVE-FS-"
}

// ------------------------------------------------------------------
// NTFS pre-capture cleanup
// ------------------------------------------------------------------

// CleanNTFS mounts the NTFS partition read-write and removes pagefile.sys
// and hiberfil.sys to reduce the image size.  It is a best-effort operation;
// errors are logged but do not abort the capture.
func CleanNTFS(ctx context.Context, device string) {
	mp := "/tmp/fog-ntfs-clean"
	if err := os.MkdirAll(mp, 0o700); err != nil {
		slog.Warn("ntfs clean: mkdir failed", "err", err)
		return
	}
	out, err := exec.CommandContext(ctx, "ntfs-3g", "-o", "remove_hiberfile", device, mp).CombinedOutput()
	if err != nil {
		slog.Warn("ntfs clean: mount failed", "device", device, "err", err, "output", string(out))
		return
	}
	defer func() {
		if out, err := exec.CommandContext(ctx, "umount", mp).CombinedOutput(); err != nil {
			slog.Warn("ntfs clean: umount failed", "err", err, "output", string(out))
		}
	}()
	for _, name := range []string{"pagefile.sys", "hiberfil.sys", "swapfile.sys"} {
		path := mp + "/" + name
		if err := os.Remove(path); err == nil {
			slog.Info("ntfs clean: removed", "file", name)
		}
	}
}
