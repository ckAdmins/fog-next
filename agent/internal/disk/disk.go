// Package disk provides block-device enumeration for the fog imaging agent.
// It replaces the previous hard-coded candidate list in actions.go with a
// proper lsblk-based approach that handles SATA, NVMe, MMC, virtio, and any
// other disk type the kernel exposes.
package disk

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Disk holds minimal information about a block device.
type Disk struct {
	// Path is the kernel device path, e.g. /dev/sda or /dev/nvme0n1.
	Path string
	// Bytes is the device capacity in bytes (0 if unknown).
	Bytes int64
}

// List returns all physical block devices (type "disk") with a non-zero size,
// ordered by their kernel enumeration sequence (sda before sdb, nvme0n1 before
// nvme1n1, etc.).  Removable flash/USB devices are included — the caller
// decides whether to filter them.
func List() ([]Disk, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(1 * time.Second)
		}
		disks, err := listOnce()
		if err == nil && len(disks) > 0 {
			return disks, nil
		}
		lastErr = err
		slog.Warn("disk scan retrying", "attempt", attempt+1, "err", err)
	}
	return nil, lastErr
}

func listOnce() ([]Disk, error) {
	// lsblk -dpno NAME,TYPE,SIZE
	out, err := exec.Command("lsblk", "-dpno", "NAME,TYPE,SIZE").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	var disks []Disk
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name, devType, sizeStr := fields[0], fields[1], fields[2]
		if devType != "disk" {
			continue
		}
		sz, _ := parseSize(sizeStr)
		if sz == 0 {
			continue
		}
		disks = append(disks, Disk{Path: name, Bytes: sz})
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("no physical disks found")
	}
	return disks, nil
}

// Primary returns the first physical disk, preferring the boot disk if it can
// be identified. The selection priority matches fos behaviour:
//  1. Largest disk (most likely the OS target in a simple machine).
//  2. Enumeration order as a tiebreaker.
//
// For multi-disk imaging use List() directly.
func Primary() (string, error) {
	disks, err := List()
	if err != nil {
		return "", err
	}
	// Sort by size descending, then by name ascending as tiebreaker.
	sort.Slice(disks, func(i, j int) bool {
		if disks[i].Bytes != disks[j].Bytes {
			return disks[i].Bytes > disks[j].Bytes
		}
		return disks[i].Path < disks[j].Path
	})
	slog.Info("primary disk selected", "disk", disks[0].Path, "bytes", disks[0].Bytes)
	return disks[0].Path, nil
}

// PartitionDevice returns the kernel path for partition number n on disk.
// It handles:
//   - SATA/SCSI/virtio  /dev/sdaN       →  /dev/sda1
//   - NVMe              /dev/nvme0n1    →  /dev/nvme0n1p1
//   - MMC/eMMC          /dev/mmcblk0    →  /dev/mmcblk0p1
//   - Loop              /dev/loop0      →  /dev/loop0p1
func PartitionDevice(disk string, n int) string {
	base := filepath.Base(disk)
	// Devices whose name ends in a digit need the "p" separator:
	// nvme0n1, mmcblk0, loop0
	if len(base) > 0 && base[len(base)-1] >= '0' && base[len(base)-1] <= '9' {
		return disk + "p" + strconv.Itoa(n)
	}
	return disk + strconv.Itoa(n)
}

// DiscoverPartitions returns an ordered list of partition device paths that
// exist on disk. It checks up to maxParts partitions.
func DiscoverPartitions(disk string) []string {
	const maxParts = 128
	var parts []string
	for i := 1; i <= maxParts; i++ {
		dev := PartitionDevice(disk, i)
		if _, err := os.Stat(dev); err == nil {
			parts = append(parts, dev)
		}
	}
	return parts
}

// PartitionNumber extracts the partition number from a device path like
// /dev/sda14 → 14, /dev/nvme0n1p2 → 2, /dev/mmcblk0p15 → 15.
// Returns 0 if the number cannot be determined.
func PartitionNumber(dev string) int {
	base := filepath.Base(dev)
	// Remove leading non-digit prefix like "sda", "nvme0n1p", "mmcblk0p".
	// Scan backwards for digits.
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] >= '0' && base[i] <= '9' {
			end := i
			start := i
			for start > 0 && base[start-1] >= '0' && base[start-1] <= '9' {
				start--
			}
			n, _ := strconv.Atoi(base[start : end+1])
			return n
		}
	}
	return 0
}

// parseSize parses lsblk SIZE strings like "500G", "256M", "128K", "1T", or a
// raw byte count.  Returns 0 on error.
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0, nil
	}
	multipliers := map[byte]int64{
		'K': 1 << 10,
		'M': 1 << 20,
		'G': 1 << 30,
		'T': 1 << 40,
		'P': 1 << 50,
	}
	last := s[len(s)-1]
	if mult, ok := multipliers[last]; ok {
		val, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, err
		}
		return int64(val * float64(mult)), nil
	}
	// Raw byte count (lsblk -b outputs raw bytes)
	val, err := strconv.ParseInt(s, 10, 64)
	return val, err
}

// SysfsSize returns the disk size in bytes by reading
// /sys/block/<dev>/size (512-byte sectors).
func SysfsSize(disk string) int64 {
	name := filepath.Base(disk)
	data, err := os.ReadFile("/sys/block/" + name + "/size")
	if err != nil {
		return 0
	}
	sectors, _ := strconv.ParseInt(strings.TrimSpace(string(bytes.TrimSpace(data))), 10, 64)
	return sectors * 512
}

// PartitionSize returns the partition size in bytes via lsblk -bno SIZE.
func PartitionSize(dev string) uint64 {
	out, err := exec.Command("lsblk", "-bno", "SIZE", dev).Output()
	if err != nil {
		return 0
	}
	sz, _ := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	return sz
}

// WaitForPartitions blocks until every expected partition device on disk
// reports a non-zero size via blockdev --getsize64.  After a GPT restore
// the kernel may not have finished setting up device nodes yet; polling
// prevents partclone from seeing a 0-sized device.
func WaitForPartitions(disk string, partNumbers []int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allReady := true
		for _, n := range partNumbers {
			dev := PartitionDevice(disk, n)
			out, err := exec.Command("blockdev", "--getsize64", dev).Output()
			if err != nil {
				allReady = false
				break
			}
			sz, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
			if sz <= 0 {
				allReady = false
				break
			}
		}
		if allReady {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for partition devices on %s", disk)
}
