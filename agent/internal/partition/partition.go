// Package partition wraps sgdisk, sfdisk, and parted for partition table
// operations.  Both GPT and MBR (DOS) tables are handled:
//
//   - GPT  → backup/restore via sgdisk --backup / --load-backup
//   - MBR  → backup via sfdisk -d + raw 512-byte boot sector;
//     restore via dd (boot sector) + sfdisk
//
// The partition table blob exchanged with the server is a JSON envelope so
// the restore side always knows which path to take:
//
//	{"type":"gpt","sgdisk":"<base64>"}
//	{"type":"mbr","sfdisk":"<sfdisk -d output>","mbr":"<base64 of 512 bytes>"}
package partition

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TableType represents the partition table format.
type TableType string

const (
	TableMBR TableType = "mbr"
	TableGPT TableType = "gpt"
)

// tableBlob is the JSON structure stored as part 0 of each image.
type tableBlob struct {
	Type   TableType `json:"type"`
	SGDisk string    `json:"sgdisk,omitempty"` // GPT: base64-encoded sgdisk binary backup
	SFDisk string    `json:"sfdisk,omitempty"` // Text: sfdisk -d output (both GPT and MBR)
	MBRRaw string    `json:"mbr,omitempty"`    // MBR: base64 of raw 512-byte boot sector
}

// DetectTableType returns the partition table type of disk by reading the
// PTTYPE attribute via blkid.  Falls back to GPT on any error.
func DetectTableType(disk string) TableType {
	out, err := exec.Command("blkid", "-p", "-o", "value", "-s", "PTTYPE", disk).Output()
	if err != nil {
		slog.Warn("blkid PTTYPE failed, assuming GPT", "disk", disk, "err", err)
		return TableGPT
	}
	switch strings.TrimSpace(string(out)) {
	case "dos":
		return TableMBR
	default:
		return TableGPT
	}
}

// Backup captures the partition table of disk and returns it as a JSON blob.
// GPT disks use a binary sgdisk backup; MBR disks save the sfdisk table dump
// plus the raw first 512 bytes of the disk (contains GRUB/bootloader code).
func Backup(disk string) ([]byte, error) {
	slog.Info("detecting partition table type", "disk", disk)
	tt := DetectTableType(disk)
	slog.Info("backing up partition table", "disk", disk, "type", tt)

	switch tt {
	case TableMBR:
		return backupMBR(disk)
	default:
		return backupGPT(disk)
	}
}

// Restore writes the partition table blob (produced by Backup) back to disk.
// It dispatches to the appropriate restore path based on the blob type field.
func Restore(disk string, blob []byte) error {
	slog.Info("restoring partition table", "disk", disk)
	var tb tableBlob
	if err := json.Unmarshal(blob, &tb); err != nil {
		// Legacy: raw sgdisk binary blob from older fos-next versions.
		slog.Warn("blob is not JSON, attempting legacy sgdisk restore", "disk", disk)
		err := restoreSGDiskRaw(disk, blob)
		reReadPartitionTable(disk)
		return err
	}
	var err error
	switch tb.Type {
	case TableMBR:
		err = restoreMBR(disk, tb)
	default:
		err = restoreGPT(disk, tb)
	}
	if err != nil {
		return err
	}
	reReadPartitionTable(disk)
	return nil
}

// ExpandLast resizes the last partition on the disk to fill all available
// space using parted's resizepart command.  This only adjusts the partition
// table entry; filesystem resizing is handled by imaging.Expand.
func ExpandLast(disk string) error {
	slog.Info("expanding last partition to fill disk", "disk", disk)
	lastN, err := lastPartitionNumber(disk)
	if err != nil {
		return fmt.Errorf("find last partition: %w", err)
	}
	partN := fmt.Sprintf("%d", lastN)
	out, err := exec.Command("parted", "-s", disk, "resizepart", partN, "100%").CombinedOutput()
	if err != nil {
		return fmt.Errorf("parted resizepart %s: %w\n%s", partN, err, out)
	}
	// Re-read partition table so the kernel sees the new size.
	_ = exec.Command("partprobe", disk).Run()
	settlePartTable()
	return nil
}

// Wipe runs shred over the entire disk to overwrite all data.
func Wipe(disk string, passes int) error {
	slog.Info("wiping disk", "disk", disk, "passes", passes)
	args := []string{"-v", "-n", fmt.Sprintf("%d", passes), disk}
	cmd := exec.Command("shred", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shred: %w", err)
	}
	return nil
}

// CheckSize verifies that the partition table described by blob fits within
// diskSizeBytes on the target disk. Returns nil if it fits, or an error
// describing the mismatch. The blob is the same JSON envelope produced by
// Backup (part 0 of an image).
func CheckSize(disk string, blob []byte, diskSizeBytes int64) error {
	var tb tableBlob
	if err := json.Unmarshal(blob, &tb); err != nil {
		// Legacy raw sgdisk — can't parse, skip the check.
		return nil
	}

	var totalEndBytes int64
	switch tb.Type {
	case TableMBR:
		totalEndBytes = parseMBREndBytes(tb.SFDisk)
	case TableGPT:
		totalEndBytes = parseGPTEndBytes(tb.SGDisk)
	}

	if totalEndBytes <= 0 {
		return nil
	}
	if totalEndBytes > diskSizeBytes {
		return fmt.Errorf("image partitions extend to %d bytes but target disk %s is only %d bytes",
			totalEndBytes, disk, diskSizeBytes)
	}
	return nil
}

// parseMBREndBytes parses sfdisk -d output and returns the end byte (start+sectors)
// of the last partition. sfdisk output is in 512-byte sectors.
func parseMBREndBytes(sfdiskOut string) int64 {
	var lastEndSector int64
	for _, line := range strings.Split(sfdiskOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "label") || strings.HasPrefix(line, "device") || strings.HasPrefix(line, "unit") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		startStr := strings.TrimSpace(fields[1])
		startStr = strings.TrimPrefix(strings.TrimPrefix(startStr, "start="), " ")
		start, err := strconv.ParseInt(strings.TrimSpace(strings.Split(startStr, ",")[0]), 10, 64)
		if err != nil {
			continue
		}
		sizeStr := strings.TrimSpace(fields[3])
		sizeStr = strings.TrimPrefix(strings.TrimPrefix(sizeStr, "size="), " ")
		size, err := strconv.ParseInt(strings.TrimSpace(strings.Split(sizeStr, ",")[0]), 10, 64)
		if err != nil {
			continue
		}
		endSector := start + size
		if endSector > lastEndSector {
			lastEndSector = endSector
		}
	}
	return lastEndSector * 512
}

// parseGPTEndBytes decodes a base64 sgdisk backup and returns the byte offset
// of the end of the last partition. sgdisk backup format:
//
//	offset 0:    512 bytes GPT protective MBR (ignored)
//	offset 512:  92 bytes GPT header
//	offset 1024: partition entries (128 bytes each, up to 128 entries)
//
// Each partition entry at offset 32: StartingLBA (u64 LE), offset 40: EndingLBA (u64 LE).
// sgdisk always uses 512-byte logical sectors.
func parseGPTEndBytes(sgdiskBase64 string) int64 {
	raw, err := base64.StdEncoding.DecodeString(sgdiskBase64)
	if err != nil {
		return 0
	}
	if len(raw) < 1024+128 {
		return 0
	}

	partEntryOffset := 1024
	var lastEndLBA uint64
	for i := 0; i < 128; i++ {
		off := partEntryOffset + i*128
		if off+128 > len(raw) {
			break
		}
		empty := true
		for _, b := range raw[off : off+16] {
			if b != 0 {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		startLBA := binary.LittleEndian.Uint64(raw[off+32:])
		endLBA := binary.LittleEndian.Uint64(raw[off+40:])
		if endLBA > startLBA && endLBA > lastEndLBA {
			lastEndLBA = endLBA
		}
	}

	if lastEndLBA == 0 {
		return 0
	}
	return int64((lastEndLBA + 1) * 512)
}

// ------------------------------------------------------------------
// GPT helpers
// ------------------------------------------------------------------

func backupGPT(disk string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "sgdisk-backup-*")
	if err != nil {
		return nil, fmt.Errorf("sgdisk backup: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpName)
	if out, err := exec.Command("sgdisk", "--backup="+tmpName, disk).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("sgdisk backup: %w\n%s", err, out)
	}
	raw, err := os.ReadFile(tmpName)
	if err != nil {
		return nil, fmt.Errorf("sgdisk backup: read temp file: %w", err)
	}
	sfdiskOut, err := exec.Command("sfdisk", "-d", disk).Output()
	if err != nil {
		slog.Warn("sfdisk dump failed (non-fatal)", "disk", disk, "err", err)
	}
	tb := tableBlob{
		Type:   TableGPT,
		SGDisk: base64.StdEncoding.EncodeToString(raw),
		SFDisk: string(sfdiskOut),
	}
	return json.Marshal(tb)
}

func restoreGPT(disk string, tb tableBlob) error {
	raw, err := base64.StdEncoding.DecodeString(tb.SGDisk)
	if err != nil {
		return fmt.Errorf("sgdisk restore: base64 decode: %w", err)
	}
	if err := restoreSGDiskRaw(disk, raw); err != nil {
		return err
	}
	restoreGPTUUIDs(disk, tb.SFDisk)
	return nil
}

func restoreSGDiskRaw(disk string, raw []byte) error {
	zapOut, zapErr := exec.Command("sgdisk", "--zap-all", disk).CombinedOutput()
	if zapErr != nil {
		slog.Warn("sgdisk --zap-all returned non-zero (ignored)", "disk", disk, "err", zapErr, "output", string(zapOut))
	}

	tmp, err := os.CreateTemp("", "sgdisk-restore-*")
	if err != nil {
		return fmt.Errorf("sgdisk restore: create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("sgdisk restore: write temp file: %w", err)
	}
	tmp.Close()
	out, err := exec.Command("sgdisk", "--load-backup="+tmp.Name(), disk).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sgdisk restore: %w\n%s", err, out)
	}
	return nil
}

// restoreGPTUUIDs parses sfdisk -d text output and explicitly sets the GPT
// disk GUID, partition type GUIDs, and partition UUIDs. This is a safety net
// matching the legacy FOS restoreUUIDInformation behaviour.  sgdisk
// --load-backup should already preserve these, but in some initramfs
// environments the backup load can silently drop GUID fields.
//
// All sgdisk operations are batched into a single invocation so the GPT is
// modified atomically and the kernel only re-reads the partition table once.
func restoreGPTUUIDs(disk, sfdiskOut string) {
	if sfdiskOut == "" {
		slog.Warn("restoreGPTUUIDs: no sfdisk data in blob, UUIDs not restored")
		return
	}
	slog.Info("restoring GPT UUIDs from sfdisk data", "disk", disk)

	var args []string

	// Disk GUID: label-id: <GUID>
	if i := strings.Index(sfdiskOut, "label-id:"); i >= 0 {
		rest := strings.TrimSpace(sfdiskOut[i+len("label-id:"):])
		if end := strings.IndexAny(rest, "\n\r "); end > 0 {
			diskuuid := strings.ToLower(rest[:end])
			if diskuuid != "" {
				args = append(args, "-U", diskuuid)
			}
		}
	}
	if len(args) == 0 {
		slog.Warn("restoreGPTUUIDs: no label-id found in sfdisk data")
	}

	// Per-partition type and UUID GUIDs.
	for _, line := range strings.Split(sfdiskOut, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "start=") {
			continue
		}
		colonIdx := strings.IndexByte(line, ':')
		if colonIdx < 0 {
			continue
		}
		devPart := strings.TrimSpace(line[:colonIdx])
		partNum := extractPartNumber(devPart)
		if partNum == "" {
			continue
		}
		rest := line[colonIdx+1:]

		if ptype := extractSFdiskField(rest, "type="); ptype != "" {
			ptype = strings.ToLower(ptype)
			args = append(args, "-t", partNum+":"+ptype)
		}
		if puuid := extractSFdiskField(rest, "uuid="); puuid != "" {
			puuid = strings.ToLower(puuid)
			args = append(args, "-u", partNum+":"+puuid)
		}
	}

	if len(args) == 0 {
		return
	}

	args = append(args, disk)
	slog.Info("restoring GPT UUIDs atomically", "disk", disk, "args", args)
	out, err := exec.Command("sgdisk", args...).CombinedOutput()
	if err != nil {
		slog.Warn("sgdisk batch UUID restore failed", "disk", disk, "err", err, "out", string(out))
	} else {
		slog.Info("GPT UUIDs restored successfully", "disk", disk)
	}
}

// extractSFdiskField extracts a value after a "key=" prefix from sfdisk output.
// The value ends at a comma, space, or end-of-string.
func extractSFdiskField(rest, key string) string {
	idx := strings.Index(rest, key)
	if idx < 0 {
		return ""
	}
	val := rest[idx+len(key):]
	end := strings.IndexAny(val, ", \t\n\r")
	if end > 0 {
		val = val[:end]
	}
	return strings.TrimSpace(val)
}

// extractPartNumber returns the partition number from a device name like
// /dev/sda3, /dev/nvme0n1p4, /dev/mmcblk0p2.
func extractPartNumber(dev string) string {
	for i := len(dev) - 1; i >= 0; i-- {
		if dev[i] >= '0' && dev[i] <= '9' {
			start := i
			for start > 0 && dev[start-1] >= '0' && dev[start-1] <= '9' {
				start--
			}
			return dev[start : i+1]
		}
	}
	return ""
}

// ------------------------------------------------------------------
// MBR helpers
// ------------------------------------------------------------------

func backupMBR(disk string) ([]byte, error) {
	// 1. Dump the partition table via sfdisk (human-readable text format).
	sfdiskOut, err := exec.Command("sfdisk", "-d", disk).Output()
	if err != nil {
		return nil, fmt.Errorf("sfdisk dump: %w", err)
	}

	// 2. Save the raw first 512 bytes which contain the bootloader (GRUB stage 1
	//    or other MBR code) in the first 446 bytes, followed by the partition
	//    table and magic signature.
	f, err := os.Open(disk)
	if err != nil {
		return nil, fmt.Errorf("open disk for MBR read: %w", err)
	}
	defer f.Close()
	mbrRaw := make([]byte, 512)
	if _, err := f.Read(mbrRaw); err != nil {
		return nil, fmt.Errorf("read MBR: %w", err)
	}

	tb := tableBlob{
		Type:   TableMBR,
		SFDisk: string(sfdiskOut),
		MBRRaw: base64.StdEncoding.EncodeToString(mbrRaw),
	}
	return json.Marshal(tb)
}

func restoreMBR(disk string, tb tableBlob) error {
	// 1. Write only the bootloader bytes (first 446 bytes) back to the disk.
	//    sfdisk will write the correct partition table in the next step.
	mbrRaw, err := base64.StdEncoding.DecodeString(tb.MBRRaw)
	if err != nil {
		return fmt.Errorf("MBR restore: base64 decode: %w", err)
	}
	if len(mbrRaw) < 446 {
		return fmt.Errorf("MBR restore: boot sector too short (%d bytes)", len(mbrRaw))
	}
	f, err := os.OpenFile(disk, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open disk for MBR write: %w", err)
	}
	if _, err := f.Write(mbrRaw[:446]); err != nil {
		f.Close()
		return fmt.Errorf("write bootloader bytes: %w", err)
	}
	f.Close()

	// 2. Restore the partition table via sfdisk.
	cmd := exec.Command("sfdisk", "--force", disk)
	cmd.Stdin = strings.NewReader(tb.SFDisk)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sfdisk restore: %w\n%s", err, out)
	}

	// 3. Re-read the partition table.
	_ = exec.Command("partprobe", disk).Run()
	settlePartTable()
	return nil
}

// ------------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------------

// settlePartTable asks udevd to settle after a partition-table change so that
// device nodes (e.g. /dev/sda1) are present before the caller proceeds.
// Falls back to a 500 ms sleep if udevadm is not available (e.g. during tests).
func settlePartTable() {
	if err := exec.Command("udevadm", "settle", "--timeout=10").Run(); err != nil {
		time.Sleep(500 * time.Millisecond)
	}
}

// reReadPartitionTable forces the kernel to re-read the partition table on
// disk and waits for udev to create device nodes. Matches legacy runPartprobe:
// udevadm settle + blockdev --rereadpt.
func reReadPartitionTable(disk string) {
	slog.Info("re-reading partition table", "disk", disk)
	settlePartTable()
	if out, err := exec.Command("blockdev", "--rereadpt", disk).CombinedOutput(); err != nil {
		slog.Warn("blockdev --rereadpt failed (non-fatal)", "disk", disk, "err", err, "out", string(out))
	}
	_ = exec.Command("partprobe", disk).Run()
	settlePartTable()
}

// lastPartitionNumber uses `parted -m -s <disk> print` to find the highest
// partition number on disk.
func lastPartitionNumber(disk string) (int, error) {
	out, err := exec.Command("parted", "-m", "-s", disk, "print").Output()
	if err != nil {
		return 0, fmt.Errorf("parted print: %w", err)
	}
	last := 0
	for _, line := range strings.Split(string(out), "\n") {
		// Machine-readable line format: number:start:end:size:fs:name:flags;
		fields := strings.SplitN(line, ":", 2)
		if len(fields) < 2 {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(fields[0], "%d", &n); err == nil && n > last {
			last = n
		}
	}
	if last == 0 {
		return 0, fmt.Errorf("no partitions found on %s", disk)
	}
	return last, nil
}
