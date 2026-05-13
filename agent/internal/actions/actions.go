// Package actions implements the fos-agent action dispatcher and each
// supported action: register, deploy, capture, debug, and wipe.
package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"

	fogapi "github.com/nemvince/fos-agent/internal/api"
	"github.com/nemvince/fos-agent/internal/disk"
	"github.com/nemvince/fos-agent/internal/imaging"
	"github.com/nemvince/fos-agent/internal/inventory"
	"github.com/nemvince/fos-agent/internal/partition"
	"github.com/nemvince/fos-agent/internal/tui"
)

// Dispatcher routes the handshake action to the correct handler.
func Dispatch(ctx context.Context, client *fogapi.Client, resp *fogapi.HandshakeResponse, p *tea.Program) error {
	slog.Info("dispatching action", "action", resp.Action)
	switch resp.Action {
	case "deploy", "debug_deploy":
		return Deploy(ctx, client, resp, p)
	case "capture", "debug_capture":
		return Capture(ctx, client, resp, p)
	case "wipe":
		return Wipe(ctx, client, resp, p)
	case "debug":
		return Debug(ctx, resp)
	default:
		slog.Warn("unknown action, dropping to debug shell", "action", resp.Action)
		return Debug(ctx, resp)
	}
}

// ------------------------------------------------------------------
// Register — collect inventory and submit to server (no boot token)
// ------------------------------------------------------------------

func Register(ctx context.Context, client *fogapi.Client, macs []string, p *tea.Program) error {
	slog.Info("action: register")
	if p != nil {
		tui.SendMsg(p, tui.PhaseMsg("Register: collecting hardware inventory..."))
	}
	inv, err := inventory.Collect()
	if err != nil {
		slog.Warn("partial inventory", "err", err)
	}

	req := fogapi.RegisterRequest{
		MACs:      macs,
		CPUModel:  inv.CPUModel,
		CPUCores:  inv.CPUCores,
		RAMBytes:  inv.RAMBytes,
		DiskBytes: inv.DiskBytes,
		UUID:      inv.UUID,
	}
	if p != nil {
		tui.SendMsg(p, tui.PhaseMsg("Register: submitting inventory..."))
	}
	if err := client.Register(ctx, req); err != nil {
		return err
	}
	slog.Info("registration submitted")
	if p != nil {
		tui.SendMsg(p, tui.PhaseMsg("Registration complete"))
	}
	return nil
}

// ------------------------------------------------------------------
// Deploy — restore image to disk
// ------------------------------------------------------------------

func Deploy(ctx context.Context, client *fogapi.Client, resp *fogapi.HandshakeResponse, p *tea.Program) error {
	slog.Info("action: deploy", "imageId", resp.ImageID, "parts", resp.PartCount)

	targetDisk, err := disk.Primary()
	if err != nil {
		return reportFail(ctx, client, resp.TaskID, "primary disk detection failed: "+err.Error())
	}

	isResizable := resp.ImageType == "resizable"
	fixedSet := make(map[int]bool)
	for _, n := range resp.FixedSizePartitions {
		fixedSet[n] = true
	}

	targetDiskBytes := uint64(disk.SysfsSize(targetDisk))

	if p != nil {
		tui.SendMsg(p, tui.DiskMsg{Device: targetDisk, SizeBytes: targetDiskBytes})
		tui.SendMsg(p, tui.PhaseMsg("Deploy: restoring partition table..."))
	}

	ptable, _, err := client.DownloadPart(ctx, resp.ImageID, 0, 0)
	if err != nil {
		return reportFail(ctx, client, resp.TaskID, "partition table download failed: "+err.Error())
	}
	defer ptable.Close()
	tableBytes, err := readAll(ptable)
	if err != nil {
		return reportFail(ctx, client, resp.TaskID, "reading partition table: "+err.Error())
	}
	if err := partition.Restore(targetDisk, tableBytes); err != nil {
		return reportFail(ctx, client, resp.TaskID, "restoring partition table: "+err.Error())
	}

	// After GPT restore + UUIDs, the kernel may not have finished sizing
	// partition devices yet. Poll until every expected partition has a
	// non-zero size so partclone doesn't fail with "0 MB" target.
	partNums := make([]int, resp.PartCount)
	for i := 0; i < resp.PartCount; i++ {
		partNums[i] = partNumFromMeta(resp.PartNumbers, i)
	}
	if err := disk.WaitForPartitions(targetDisk, partNums, 10*time.Second); err != nil {
		return reportFail(ctx, client, resp.TaskID, "partition devices not ready: "+err.Error())
	}

	// Check for disk size mismatch: if the image partitions extend past the
	// target disk and the image is not resizable, fail early with a clear
	// message rather than hitting a cryptic partclone error later.
	if !isResizable && targetDiskBytes > 0 {
		if mismatchErr := partition.CheckSize(targetDisk, tableBytes, int64(targetDiskBytes)); mismatchErr != nil {
			return reportFail(ctx, client, resp.TaskID, "disk size mismatch: "+mismatchErr.Error())
		}
	}

	totalBytes := resp.TotalBytes
	hasProgram := p != nil

	if hasProgram {
		parts := make([]tui.PartStatus, resp.PartCount)
		for i := 0; i < resp.PartCount; i++ {
			num := partNumFromMeta(resp.PartNumbers, i)
			dev := disk.PartitionDevice(targetDisk, num)
			fs := "?"
			if i < len(resp.PartTypes) {
				fs = resp.PartTypes[i]
			}
			parts[i] = tui.PartStatus{Device: dev, FSType: fs, SizeBytes: 0, State: "pending"}
		}
		tui.SendMsg(p, tui.PartitionsMsg(parts))
	}

	var transferred int64
	progressFn := makeProgressFn(ctx, client, resp.TaskID, &transferred, totalBytes)

	type partState struct {
		dev        string
		restoredFS string
		displayFS  string
		contentLen int64
	}
	restoredParts := make([]partState, resp.PartCount)

	for part := 1; part <= resp.PartCount; part++ {
		num := partNumFromMeta(resp.PartNumbers, part-1)
		dev := disk.PartitionDevice(targetDisk, num)
		displayFS := "?"
		if part-1 < len(resp.PartTypes) && resp.PartTypes[part-1] != "" {
			displayFS = resp.PartTypes[part-1]
		}
		slog.Info("deploying partition", "part", num, "device", dev)

		if hasProgram {
			tui.SendMsg(p, tui.PhaseMsg(fmt.Sprintf("Deploy: restoring partition %d/%d (%s)", part, resp.PartCount, dev)))
			tui.SendMsg(p, tui.PartitionMsg{Index: part - 1, Status: tui.PartStatus{Device: dev, FSType: displayFS, SizeBytes: 0, Percent: 0, State: "active"}})
		}

		var partErr error
		var resumeOffset int64
		for attempt := 0; attempt < 3; attempt++ {
			body, contentLen, dlErr := client.DownloadPart(ctx, resp.ImageID, part, resumeOffset)
			if dlErr != nil {
				partErr = dlErr
				slog.Warn("download failed, retrying", "part", part, "attempt", attempt, "err", dlErr)
				time.Sleep(5 * time.Second)
				continue
			}

			peek := make([]byte, 1)
			n, _ := io.ReadFull(body, peek)
			full := io.MultiReader(bytes.NewReader(peek[:n]), body)

			if n == 1 && peek[0] == '{' {
				raw, readErr := io.ReadAll(full)
				body.Close()
				if readErr != nil {
					partErr = readErr
					slog.Warn("reading part sentinel failed, retrying", "part", part, "attempt", attempt, "err", readErr)
					time.Sleep(5 * time.Second)
					continue
				}
				var meta partMeta
				_ = json.Unmarshal(raw, &meta)
				if meta.Type == "swap" {
					slog.Info("recreating swap partition", "part", part, "device", dev, "uuid", meta.UUID)
					if mkErr := makeSwap(dev, meta.UUID); mkErr != nil {
						partErr = mkErr
						slog.Warn("mkswap failed, retrying", "part", part, "attempt", attempt, "err", mkErr)
						time.Sleep(5 * time.Second)
						continue
					}
					if hasProgram {
						tui.SendMsg(p, tui.PartitionMsg{Index: part - 1, Status: tui.PartStatus{Device: dev, FSType: "swap", SizeBytes: uint64(contentLen), Percent: 100, State: "skipped"}})
					}
					restoredParts[part-1] = partState{dev: dev, restoredFS: "swap", displayFS: "swap", contentLen: contentLen}
				}
				partErr = nil
				break
			}

			fs := displayFS // use saved type from capture metadata for correct partclone binary
			if fs == "?" || fs == "" || fs == "dd" {
				fs = detectFilesystem(dev)
			}
			if hasProgram {
				tui.SendMsg(p, tui.PartitionMsg{Index: part - 1, Status: tui.PartStatus{Device: dev, FSType: displayFS, SizeBytes: uint64(contentLen), Percent: 0, State: "active"}})
			}

			wrappedFn := progressFn
			if hasProgram {
				partIdx := part - 1
				wrappedFn = func(pct int, bpm int64) {
					progressFn(pct, bpm)
					tui.SendMsg(p, tui.PartitionMsg{Index: partIdx, Status: tui.PartStatus{Device: dev, FSType: displayFS, SizeBytes: uint64(contentLen), Percent: pct, State: "active"}})
					tui.SendMsg(p, tui.ProgressMsg{Pct: pct, DoneBytes: uint64(transferred), TotalBytes: uint64(totalBytes), SpeedBpm: uint64(bpm)})
				}
			}

			slog.Info("starting partclone restore", "part", part, "device", dev, "fs", fs, "attempt", attempt)
			imgErr := imaging.Restore(ctx, dev, fs, full, wrappedFn, !hasProgram)
			body.Close()
			if imgErr != nil {
				partErr = imgErr
				slog.Warn("restore failed, retrying", "part", part, "attempt", attempt, "err", imgErr)
				time.Sleep(5 * time.Second)
				continue
			}
			slog.Info("partclone restore complete", "part", part, "device", dev)

			restoredFS := probeFilesystem(dev)
			// Fall back to the known type from capture if probe failed.
			if restoredFS == "dd" && displayFS != "?" && displayFS != "dd" {
				slog.Warn("filesystem probe returned dd, using saved type from capture", "part", part, "detected", restoredFS, "saved", displayFS)
				restoredFS = displayFS
			}
			if strings.ToLower(restoredFS) == "ntfs" {
				imaging.NtfsFixDirty(dev)
			}
			restoredParts[part-1] = partState{dev: dev, restoredFS: restoredFS, displayFS: displayFS, contentLen: contentLen}
			partErr = nil
			if hasProgram {
				tui.SendMsg(p, tui.PartitionMsg{Index: part - 1, Status: tui.PartStatus{Device: dev, FSType: displayFS, SizeBytes: uint64(contentLen), Percent: 100, State: "done"}})
			}
			break
		}
		if partErr != nil {
			return reportFail(ctx, client, resp.TaskID,
				"partition "+strconv.Itoa(part)+" restore failed after 3 attempts: "+partErr.Error())
		}
	}

	// Now that all partition data has been written, expand the last partition
	// table entry (if resizable) and then expand filesystems.
	if isResizable {
		if err := partition.ExpandLast(targetDisk); err != nil {
			slog.Warn("expand last partition table entry failed (non-fatal)", "err", err)
		}

		for part := 1; part <= resp.PartCount; part++ {
			num := partNumFromMeta(resp.PartNumbers, part-1)
			if fixedSet[num] {
				continue
			}
			ps := restoredParts[part-1]
			if ps.restoredFS == "" || ps.restoredFS == "swap" {
				continue
			}
			slog.Info("expanding filesystem after deploy", "part", num, "device", ps.dev, "fs", ps.restoredFS)
			if expErr := imaging.Expand(ctx, ps.dev, ps.restoredFS); expErr != nil {
				slog.Warn("filesystem expand failed (non-fatal)", "part", num, "err", expErr)
			}
		}
	}

	if hasProgram {
		tui.SendMsg(p, tui.PhaseMsg("Deploy: complete"))
	}

	return client.Complete(ctx, fogapi.CompleteRequest{
		TaskID:  resp.TaskID,
		Success: true,
	})
}

// ------------------------------------------------------------------
// Capture — clone partitions and upload to server
// ------------------------------------------------------------------

func Capture(ctx context.Context, client *fogapi.Client, resp *fogapi.HandshakeResponse, p *tea.Program) error {
	slog.Info("action: capture", "imageId", resp.ImageID)

	targetDisk, err := disk.Primary()
	if err != nil {
		return reportFail(ctx, client, resp.TaskID, "primary disk detection failed: "+err.Error())
	}

	if p != nil {
		tui.SendMsg(p, tui.PhaseMsg("Capture: backing up partition table..."))
	}

	tableBytes, err := partition.Backup(targetDisk)
	if err != nil {
		return reportFail(ctx, client, resp.TaskID, "partition table backup failed: "+err.Error())
	}
	if err := client.UploadPart(ctx, resp.ImageID, 0, bytesReader(tableBytes)); err != nil {
		return reportFail(ctx, client, resp.TaskID, "partition table upload failed: "+err.Error())
	}

	hasProgram := p != nil

	parts := disk.DiscoverPartitions(targetDisk)
	if len(parts) == 0 {
		return reportFail(ctx, client, resp.TaskID, "no partitions found on disk")
	}

	type partInfo struct {
		dev         string
		fs          string
		size        uint64
		isSwap      bool
		swapUUID    string
		actualNum   int // actual kernel partition number (e.g. 14, not sequential index)
	}
	info := make([]partInfo, len(parts))
	var partTypes []string
	var partNumbers []int
	for i, dev := range parts {
		fs := detectFilesystem(dev)
		sz := disk.PartitionSize(dev)
		n := disk.PartitionNumber(dev)
		pi := partInfo{dev: dev, fs: fs, size: sz, actualNum: n}
		partTypes = append(partTypes, fs)
		partNumbers = append(partNumbers, n)
		if fs == "swap" {
			pi.isSwap = true
			pi.swapUUID = readPartitionUUID(dev)
		}
		if pi.fs == "ntfs" && imaging.IsBitlockerEncrypted(pi.dev) {
			return reportFail(ctx, client, resp.TaskID,
				"Bitlocker encryption detected on partition "+strconv.Itoa(n)+" ("+pi.dev+") — cannot capture encrypted volume")
		}
		info[i] = pi
	}

	if hasProgram {
		tui.SendMsg(p, tui.DiskMsg{Device: targetDisk, SizeBytes: uint64(disk.SysfsSize(targetDisk))})
		partStatuses := make([]tui.PartStatus, len(parts))
		for i, pi := range info {
			partStatuses[i] = tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, State: "pending"}
		}
		tui.SendMsg(p, tui.PartitionsMsg(partStatuses))
	}

	// Shrink filesystems once before cloning. Track which partitions actually
	// shrunk so FixedSizePartitions is accurate. Shrinking is unconditional —
	// legacy FOS always shrinks. The imageType controls whether deploy expands.
	var fixedSizePartitions []int
	for _, pi := range info {
		partNum := pi.actualNum
		if pi.isSwap || !imaging.CanShrink(pi.fs) {
			fixedSizePartitions = append(fixedSizePartitions, partNum)
			continue
		}
		if pi.fs == "ntfs" {
			imaging.CleanNTFS(ctx, pi.dev)
		}
		slog.Info("shrinking filesystem before capture", "part", partNum, "fs", pi.fs)
		shrunk, shrinkErr := imaging.Shrink(ctx, pi.dev, pi.fs)
		if shrinkErr != nil || !shrunk {
			slog.Warn("filesystem shrink failed, marking as fixed-size", "part", partNum, "err", shrinkErr)
			fixedSizePartitions = append(fixedSizePartitions, partNum)
		}
	}

	// Default to resizable — only stay fixed if the image was explicitly
	// tagged as fixed (e.g. from a previous capture).
	imageType := "resizable"
	if resp.ImageType == "fixed" {
		imageType = "fixed"
	}
	if err := client.SetImageMeta(ctx, fogapi.ImageMetaRequest{
		TaskID:              resp.TaskID,
		ImageID:             resp.ImageID,
		ImageType:           imageType,
		FixedSizePartitions: fixedSizePartitions,
		PartCount:           len(parts),
		PartTypes:           partTypes,
		PartNumbers:         partNumbers,
	}); err != nil {
		return reportFail(ctx, client, resp.TaskID, "SetImageMeta failed: "+err.Error())
	}

	for i, pi := range info {
		seqNum := i + 1 // sequential index for file part name (part1, part2, ...)
		partNum := pi.actualNum
		slog.Info("capturing partition", "part", partNum, "device", pi.dev, "fs", pi.fs, "size", pi.size)

		if hasProgram {
			tui.SendMsg(p, tui.PhaseMsg(fmt.Sprintf("Capture: cloning partition %d/%d (%s)", seqNum, len(parts), pi.fs)))
		}

		var partErr error
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				slog.Warn("capture partition failed, retrying", "part", partNum, "attempt", attempt, "err", partErr)
				if hasProgram {
					tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, State: "active"}})
				}
				time.Sleep(5 * time.Second)
			}

			if pi.isSwap {
				sentinel, _ := json.Marshal(partMeta{Type: "swap", UUID: pi.swapUUID})
				if upErr := client.UploadPart(ctx, resp.ImageID, seqNum, bytes.NewReader(sentinel)); upErr != nil {
					partErr = upErr
					slog.Warn("swap sentinel upload failed, retrying", "part", partNum, "attempt", attempt, "err", upErr)
					continue
				}
				partErr = nil
				if hasProgram {
					tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: "swap", SizeBytes: pi.size, Percent: 100, State: "skipped"}})
				}
				break
			}

			if hasProgram {
				tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, State: "active"}})
			}

			progressFn := func(pct int, bpm int64) {
				if hasProgram {
					tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, Percent: pct, State: "active"}})
					tui.SendMsg(p, tui.ProgressMsg{Pct: pct, DoneBytes: 0, TotalBytes: 0, SpeedBpm: uint64(bpm)})
				}
				_ = client.ReportProgress(ctx, fogapi.ProgressRequest{
					TaskID:        resp.TaskID,
					Percent:       pct,
					BitsPerMinute: bpm,
				})
			}
			pr, pw := syncPipe()
			errCh := make(chan error, 1)
			slog.Info("starting clone goroutine", "part", partNum, "device", pi.dev, "fs", pi.fs, "attempt", attempt)
			go func() {
				err := imaging.Clone(ctx, pi.dev, pi.fs, pw, progressFn, !hasProgram)
				pw.CloseWithError(err)
				errCh <- err
			}()
			slog.Info("uploading part data", "part", partNum, "imageId", resp.ImageID)
			if upErr := client.UploadPart(ctx, resp.ImageID, seqNum, pr); upErr != nil {
				if hasProgram {
					tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, State: "error"}})
				}
				partErr = upErr
				slog.Warn("upload failed, retrying", "part", partNum, "attempt", attempt, "err", upErr)
				continue
			}
			slog.Info("upload complete, waiting for clone", "part", partNum)
			if err := <-errCh; err != nil {
				if hasProgram {
					tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, State: "error"}})
				}
				partErr = err
				slog.Warn("partclone failed, retrying", "part", partNum, "attempt", attempt, "err", err)
				continue
			}
			slog.Info("partition capture complete", "part", partNum, "device", pi.dev)
			if hasProgram {
				tui.SendMsg(p, tui.PartitionMsg{Index: i, Status: tui.PartStatus{Device: pi.dev, FSType: pi.fs, SizeBytes: pi.size, Percent: 100, State: "done"}})
			}
			partErr = nil
			break
		}
		if partErr != nil {
			return reportFail(ctx, client, resp.TaskID,
				"partition "+strconv.Itoa(partNum)+" capture failed after 3 attempts: "+partErr.Error())
		}
	}

	if hasProgram {
		tui.SendMsg(p, tui.PhaseMsg("Capture: complete"))
	}

	return client.Complete(ctx, fogapi.CompleteRequest{
		TaskID:  resp.TaskID,
		Success: true,
	})
}

// ------------------------------------------------------------------
// Wipe — overwrite disk with random data
// ------------------------------------------------------------------

func Wipe(ctx context.Context, client *fogapi.Client, resp *fogapi.HandshakeResponse, p *tea.Program) error {
	slog.Info("action: wipe")
	targetDisk, err := disk.Primary()
	if err != nil {
		return reportFail(ctx, client, resp.TaskID, "primary disk detection failed: "+err.Error())
	}
	if p != nil {
		tui.SendMsg(p, tui.DiskMsg{Device: targetDisk, SizeBytes: 0})
		tui.SendMsg(p, tui.PhaseMsg("Wipe: shredding disk (this may take a while)..."))
	}
	if err := partition.Wipe(targetDisk, 1); err != nil {
		return reportFail(ctx, client, resp.TaskID, "wipe failed: "+err.Error())
	}
	return client.Complete(ctx, fogapi.CompleteRequest{TaskID: resp.TaskID, Success: true})
}

// ------------------------------------------------------------------
// Debug — print banner and return.
// ------------------------------------------------------------------

func Debug(ctx context.Context, resp *fogapi.HandshakeResponse) error {
	slog.Info("action: debug — returning to shell")
	printDebugBanner(resp)
	return nil
}

// ------------------------------------------------------------------
// DebugDeploy / DebugCapture — verbosely run deploy or capture, then
// always drop to shell (exit code 2) regardless of success.
// ------------------------------------------------------------------

func DebugDeploy(ctx context.Context, client *fogapi.Client, resp *fogapi.HandshakeResponse, p *tea.Program) error {
	slog.Info("===== debug_deploy: starting deploy with verbose output =====")
	slog.Info("handshake details",
		"taskId", resp.TaskID,
		"imageId", resp.ImageID,
		"partCount", resp.PartCount,
		"totalBytes", resp.TotalBytes,
		"imageType", resp.ImageType,
	)
	err := Deploy(ctx, client, resp, p)
	if err != nil {
		slog.Error("debug_deploy: deploy FAILED", "err", err)
	} else {
		slog.Info("debug_deploy: deploy completed successfully")
	}
	return err
}

func DebugCapture(ctx context.Context, client *fogapi.Client, resp *fogapi.HandshakeResponse, p *tea.Program) error {
	slog.Info("===== debug_capture: starting capture with verbose output =====")
	slog.Info("handshake details",
		"taskId", resp.TaskID,
		"imageId", resp.ImageID,
		"imageType", resp.ImageType,
	)
	err := Capture(ctx, client, resp, p)
	if err != nil {
		slog.Error("debug_capture: capture FAILED", "err", err)
	} else {
		slog.Info("debug_capture: capture completed successfully")
	}
	return err
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

func reportFail(ctx context.Context, client *fogapi.Client, taskID, msg string) error {
	slog.Error("task failed", "msg", msg)
	_ = client.Complete(ctx, fogapi.CompleteRequest{TaskID: taskID, Success: false, Message: msg})
	return fmt.Errorf("%s", msg)
}

// partNumFromMeta returns the actual kernel partition number for the
// sequential index idx (0-based). Falls back to idx+1 for backwards
// compatibility with images captured before PartNumbers was added.
func partNumFromMeta(partNumbers []int, idx int) int {
	if idx < len(partNumbers) && partNumbers[idx] > 0 {
		return partNumbers[idx]
	}
	return idx + 1
}

func makeProgressFn(ctx context.Context, client *fogapi.Client, taskID string, transferred *int64, totalBytes int64) imaging.ProgressFunc {
	return func(pct int, bpm int64) {
		req := fogapi.ProgressRequest{
			TaskID:           taskID,
			Percent:          pct,
			BitsPerMinute:    bpm,
			BytesTransferred: *transferred,
		}
		if err := client.ReportProgress(ctx, req); err != nil {
			slog.Warn("progress report failed", "err", err)
		}
	}
}

func detectFilesystem(dev string) string {
	// 1. blkid -po udev — probe mode reads the device directly, bypassing
	//    kernel buffer cache and blkid cache.  This is what legacy FOS uses
	//    in fsTypeSetting() and works on freshly-written partitions.
	out, err := exec.Command("blkid", "-po", "udev", dev).Output()
	if err == nil {
		fs := parseBlkidUdevFS(string(out))
		if fs != "" {
			slog.Info("detected filesystem", "dev", dev, "fs", fs, "via", "blkid-probe")
			return fs
		}
		slog.Debug("blkid probe: no FS_TYPE", "dev", dev, "raw", strings.TrimSpace(string(out)))
	} else {
		slog.Debug("blkid -po udev failed", "dev", dev, "err", err)
	}

	// 2. lsblk (util-linux) — clean single-value output, reads from kernel.
	out, err = exec.Command("lsblk", "-no", "FSTYPE", dev).Output()
	if err == nil {
		fs := strings.TrimSpace(string(out))
		slog.Debug("lsblk fs probe", "dev", dev, "raw", fs)
		if fs != "" {
			slog.Info("detected filesystem", "dev", dev, "fs", fs, "via", "lsblk")
			return fs
		}
	} else {
		slog.Debug("lsblk fs probe failed", "dev", dev, "err", err)
	}

	// 3. blkid without -p — reads cached metadata, fallback for busybox.
	out, err = exec.Command("blkid", dev).Output()
	if err == nil {
		slog.Debug("blkid raw output", "dev", dev, "raw", strings.TrimSpace(string(out)))
		fs := parseBlkidType(string(out))
		if fs != "" {
			slog.Info("detected filesystem", "dev", dev, "fs", fs, "via", "blkid")
			return fs
		}
	} else {
		slog.Debug("blkid probe failed", "dev", dev, "err", err)
	}

	// 4. file -s — detects filesystem from magic bytes in the block device.
	out, err = exec.Command("file", "-s", dev).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		slog.Debug("file -s raw output", "dev", dev, "raw", raw)
		fs := parseFileFSType(raw)
		if fs != "" {
			slog.Info("detected filesystem", "dev", dev, "fs", fs, "via", "file -s")
			return fs
		}
	} else {
		slog.Debug("file -s probe failed", "dev", dev, "err", err)
	}

	slog.Warn("filesystem detection failed, falling back to dd", "dev", dev)
	return "dd"
}

// probeFilesystem flushes kernel buffers and probes the device for its
// filesystem type. Used after partclone writes data to ensure we read from
// disk, not from stale kernel cache.
func probeFilesystem(dev string) string {
	_ = exec.Command("blockdev", "--flushbufs", dev).Run()
	return detectFilesystem(dev)
}

// parseBlkidUdevFS extracts FS_TYPE= from blkid -po udev output.
// Example input: "ID_FS_TYPE=ext4\nID_PART_ENTRY_TYPE=..."
func parseBlkidUdevFS(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		const prefix = "FS_TYPE="
		if strings.HasPrefix(line, prefix) {
			return line[len(prefix):]
		}
	}
	return ""
}

// parseBlkidType extracts TYPE="..." from a busybox blkid output line.
func parseBlkidType(line string) string {
	const needle = `TYPE="`
	idx := strings.Index(line, needle)
	if idx < 0 {
		return ""
	}
	start := idx + len(needle)
	end := strings.IndexByte(line[start:], '"')
	if end < 0 {
		return ""
	}
	return line[start : start+end]
}

// parseFileFSType extracts the filesystem name from file(1) -s output.
// Example inputs and extracted names:
//
//	/dev/sda1: Linux rev 1.0 ext4 filesystem data ... → "ext4"
//	/dev/sda1: DOS/MBR boot sector ... NTFS ...       → "ntfs"
//	/dev/sda1: SGI XFS filesystem ...                  → "xfs"
//	/dev/sda1: BTRFS ...                               → "btrfs"
//	/dev/sda1: Linux swap file ...                     → "swap"
//	/dev/sda1: FAT (32 bit) ...                        → "fat32"
//	/dev/sda1: data                                    → ""
func parseFileFSType(line string) string {
	// Strip the "device: " prefix.
	if idx := strings.IndexByte(line, ':'); idx >= 0 {
		line = strings.TrimSpace(line[idx+1:])
	}
	line = strings.ToLower(line)
	switch {
	case strings.Contains(line, "ext4"), strings.Contains(line, "ext3"), strings.Contains(line, "ext2"):
		return "ext4"
	case strings.Contains(line, "ntfs"):
		return "ntfs"
	case strings.Contains(line, "xfs"):
		return "xfs"
	case strings.Contains(line, "btrfs"):
		return "btrfs"
	case strings.Contains(line, "fat"), strings.Contains(line, "vfat"):
		return "vfat"
	case strings.Contains(line, "swap"):
		return "swap"
	case strings.Contains(line, "f2fs"):
		return "f2fs"
	}
	return ""
}

func printDebugBanner(resp *fogapi.HandshakeResponse) {
	slog.Info("=== fos-agent debug mode ===")
	slog.Info("taskId", "value", resp.TaskID)
	slog.Info("action", "value", resp.Action)
	slog.Info("imageId", "value", resp.ImageID)
}
