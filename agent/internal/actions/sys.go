package actions

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// syncPipe returns a synchronised io.PipeReader/Writer pair.
func syncPipe() (io.ReadCloser, *io.PipeWriter) {
	return io.Pipe()
}

// bytesReader wraps a byte slice in an io.Reader.
func bytesReader(b []byte) io.Reader {
	return io.LimitReader(bytesReaderOf(b), int64(len(b)))
}

type byteSliceReader struct {
	data []byte
	pos  int
}

func (r *byteSliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func bytesReaderOf(b []byte) io.Reader {
	return &byteSliceReader{data: b}
}

// readAll is a thin wrapper around io.ReadAll.
func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// partMeta is the JSON sentinel uploaded in place of partclone data for
// partition types that do not need to be cloned (currently only swap).
type partMeta struct {
	Type string `json:"type"`
	UUID string `json:"uuid,omitempty"`
}

// readPartitionUUID reads the UUID of a partition using blkid.
func readPartitionUUID(dev string) string {
	out, err := exec.Command("blkid", "-s", "UUID", "-o", "value", dev).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// makeSwap initialises a swap partition, preserving the original UUID if known.
func makeSwap(dev, uuid string) error {
	var cmd *exec.Cmd
	if uuid != "" {
		cmd = exec.Command("mkswap", "-U", uuid, dev)
	} else {
		cmd = exec.Command("mkswap", dev)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkswap %s: %w\n%s", dev, err, out)
	}
	return nil
}
