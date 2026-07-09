// Package fos handles downloading the fog-next kernel and initramfs artifacts
// from a release URL during `fog install`. Files are downloaded to a temporary
// location, their SHA-256 checksums are verified against the release's
// sha256sums file, and they are then atomically moved into kernel_path.
package fos

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ckAdmins/fog-next/internal/config"
)

// Downloader fetches fog-next release artifacts and verifies their checksums.
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

// githubRelease is the relevant subset of the GitHub Releases API response.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Download fetches bzImage and init.xz (or whatever is configured), verifies
// checksums, and installs them into kernel_path.
//
// When ReleaseURL points to a GitHub repository the GitHub Releases API is
// queried first to obtain the release tag (for display) and the exact
// download URLs of every asset.  If the API is unreachable the downloader
// falls back to the legacy URL-concatenation behaviour.
func (d *Downloader) Download(ctx context.Context) error {
	base := strings.TrimRight(d.cfg.ReleaseURL, "/")

	// Try the GitHub Releases API for version info and asset URLs.
	if assets, tag, ok := d.fetchRelease(ctx); ok {
		slog.Info("fog-next agent release", "version", tag)

		// Build a map of asset name → download URL from the API response.
		assetURLs := make(map[string]string, len(assets))
		for _, a := range assets {
			assetURLs[a.Name] = a.BrowserDownloadURL
		}

		// Download sha256sums first so we can verify the other assets.
		sumsURL, ok := assetURLs["sha256sums"]
		if !ok {
			return fmt.Errorf("sha256sums not found in release assets")
		}
		slog.Info("fetching fog-next release checksums", "url", sumsURL)
		sums, err := d.fetchChecksums(ctx, sumsURL)
		if err != nil {
			return fmt.Errorf("fetch sha256sums: %w", err)
		}

		files := []string{d.cfg.KernelFile, d.cfg.InitFile}
		for _, name := range files {
			expected, ok := sums[name]
			if !ok {
				return fmt.Errorf("sha256sums has no entry for %q", name)
			}
			dlURL, ok := assetURLs[name]
			if !ok {
				return fmt.Errorf("asset %q not found in release", name)
			}
			slog.Info("downloading", "file", name, "url", dlURL)
			if err := d.fetchAndVerify(ctx, dlURL, name, expected); err != nil {
				return fmt.Errorf("download %s: %w", name, err)
			}
			slog.Info("installed", "file", name, "dest", filepath.Join(d.dest, name))
		}
		return nil
	}

	// Fallback: legacy URL-concatenation behaviour for non-GitHub or
	// API-unreachable scenarios.
	slog.Info("fetching fog-next release checksums", "url", base+"/sha256sums")
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
		dlURL := base + "/" + name
		slog.Info("downloading", "file", name, "url", dlURL)
		if err := d.fetchAndVerify(ctx, dlURL, name, expected); err != nil {
			return fmt.Errorf("download %s: %w", name, err)
		}
		slog.Info("installed", "file", name, "dest", filepath.Join(d.dest, name))
	}
	return nil
}

// fetchRelease calls the GitHub Releases API for the latest release.  On
// success it returns the asset list and the release tag.  The boolean
// indicates whether the API call succeeded — callers must fall back to
// legacy URL behaviour when false.
func (d *Downloader) fetchRelease(ctx context.Context) ([]githubAsset, string, bool) {
	owner, repo := parseGitHubRepo(d.cfg.ReleaseURL)
	if owner == "" {
		return nil, "", false
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, "", false
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", false
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, "", false
	}
	if rel.TagName == "" {
		return nil, "", false
	}
	return rel.Assets, rel.TagName, true
}

// parseGitHubRepo extracts "owner/repo" from a GitHub releases URL.
// Returns empty strings if the URL does not match the expected pattern.
func parseGitHubRepo(rawURL string) (owner, repo string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}
	if u.Host != "github.com" {
		return "", ""
	}
	// Expected path: /{owner}/{repo}/releases/...
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
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
