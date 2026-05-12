// Package fos handles downloading the fos-next kernel and initramfs artifacts
// from a release URL during `fog install`. Files are downloaded to a temporary
// location, their SHA-256 checksums are verified against the release's
// sha256sums file, and they are then atomically moved into kernel_path.
package fos

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nemvince/fog-next/internal/config"
)

// Downloader fetches fos-next release artifacts and verifies their checksums.
type Downloader struct {
	cfg    config.FOSConfig
	dest   string // kernel_path from StorageConfig
	client *http.Client
}

// New creates a Downloader using the provided configs.
func New(fosCfg config.FOSConfig, kernelPath string) *Downloader {
	return &Downloader{
		cfg:  fosCfg,
		dest: kernelPath,
		client: &http.Client{
			Timeout: 10 * time.Minute, // large initramfs on slow links
		},
	}
}

// Download fetches bzImage and init.xz (or whatever is configured), verifies
// checksums, and installs them into kernel_path. It prints progress to stdout.
// If any step fails the destination files are not touched.
func (d *Downloader) Download(ctx context.Context) error {
	base := strings.TrimRight(d.cfg.ReleaseURL, "/")

	// Best-effort version detection — follows the GitHub latest→tag redirect
	// and extracts the release tag. Non-GitHub URLs or any error are silently
	// ignored.
	if tag := resolveTag(ctx, d.client, base+"/sha256sums"); tag != "" {
		slog.Info("fos-next release", "version", tag)
	}

	slog.Info("fetching fos-next release checksums", "url", base+"/sha256sums")
	sums, err := d.fetchChecksums(ctx, base+"/sha256sums")
	if err != nil {
		return fmt.Errorf("fetch sha256sums: %w", err)
	}

	files := []string{d.cfg.KernelFile, d.cfg.InitFile}
	for _, name := range files {
		expected, ok := sums[name]
		if !ok {
			return fmt.Errorf("sha256sums has no entry for %q", name)
		}
		url := base + "/" + name
		slog.Info("downloading", "file", name, "url", url)
		if err := d.fetchAndVerify(ctx, url, name, expected); err != nil {
			return fmt.Errorf("download %s: %w", name, err)
		}
		slog.Info("installed", "file", name, "dest", filepath.Join(d.dest, name))
	}
	return nil
}

// fetchChecksums downloads and parses a sha256sums file into a map of
// filename → hex digest.
func (d *Downloader) fetchChecksums(ctx context.Context, url string) (map[string]string, error) {
	resp, err := d.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sums := make(map[string]string)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// sha256sum format: "<hex>  <filename>" (two spaces) or "<hex> <filename>"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		digest := fields[0]
		name := filepath.Base(fields[1]) // strip any leading "./" or path
		sums[name] = digest
	}
	return sums, scanner.Err()
}

// fetchAndVerify downloads url to a temp file, verifies its SHA-256 digest
// against expected, then atomically renames it into place under d.dest/name.
func (d *Downloader) fetchAndVerify(ctx context.Context, url, name, expected string) error {
	if err := os.MkdirAll(d.dest, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", d.dest, err)
	}

	// Write to a sibling temp file so the rename is atomic on the same filesystem.
	tmp, err := os.CreateTemp(d.dest, ".fos-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		_ = os.Remove(tmpPath) // clean up if we didn't rename
	}()

	resp, err := d.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	h := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, h), resp.Body)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", name, got, expected)
	}
	slog.Info("checksum OK", "file", name, "bytes", written, "sha256", got)

	dest := filepath.Join(d.dest, name)
	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}
	return nil
}

// get performs an HTTP GET and returns a non-2xx status as an error.
func (d *Downloader) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP %s fetching %s", resp.Status, url)
	}
	return resp, nil
}

// resolveTag makes a lightweight HEAD request to url and, if the final URL
// (after following redirects) matches the GitHub releases download pattern
// /.../releases/download/<tag>/..., returns the tag. Returns "" on any
// failure or non-GitHub URL.
func resolveTag(ctx context.Context, client *http.Client, url string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	resp.Body.Close()

	// Path after redirects: /owner/repo/releases/download/v1.2.3/sha256sums
	parts := strings.Split(strings.Trim(resp.Request.URL.Path, "/"), "/")
	for i, p := range parts {
		if p == "download" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
