// Package transform provides attachment downloading capabilities.
package transform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// AttachmentDownloader handles downloading and storing attachments from Notion.
type AttachmentDownloader interface {
	// Download downloads an attachment from the given URL and returns the local path.
	// The path is relative to the vault root (e.g., "_attachments/abc123.png").
	// Returns empty string and nil error if downloading is disabled.
	Download(ctx context.Context, remoteURL string) (string, error)
}

// Downloader implements AttachmentDownloader.
type Downloader struct {
	vaultPath        string
	attachmentFolder string
	enabled          bool
	dryRun           bool
	httpClient       *http.Client
	logger           *slog.Logger
	downloaded       map[string]string // URL -> local path cache
}

// DownloaderConfig holds configuration for the downloader.
type DownloaderConfig struct {
	VaultPath        string
	AttachmentFolder string
	Enabled          bool
	DryRun           bool
	Logger           *slog.Logger
}

// NewDownloader creates a new attachment downloader.
func NewDownloader(cfg DownloaderConfig) *Downloader {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Downloader{
		vaultPath:        cfg.VaultPath,
		attachmentFolder: cfg.AttachmentFolder,
		enabled:          cfg.Enabled,
		dryRun:           cfg.DryRun,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		logger:     cfg.Logger,
		downloaded: make(map[string]string),
	}
}

// Download downloads an attachment from the given URL.
// Returns the local path relative to the vault (for use in markdown).
// If downloading is disabled or in dry-run mode, returns the original URL.
func (d *Downloader) Download(ctx context.Context, remoteURL string) (string, error) {
	if !d.enabled {
		return remoteURL, nil
	}

	// Skip external URLs that are not Notion-hosted
	if !d.isDownloadableURL(remoteURL) {
		return remoteURL, nil
	}

	// Check cache first
	if localPath, ok := d.downloaded[remoteURL]; ok {
		return localPath, nil
	}

	// Generate filename from URL
	filename := d.generateFilename(remoteURL)
	localPath := filepath.Join(d.attachmentFolder, filename)
	fullPath := filepath.Join(d.vaultPath, localPath)

	if d.dryRun {
		d.logger.Info("would download", "url", remoteURL, "to", localPath)
		d.downloaded[remoteURL] = localPath
		return localPath, nil
	}

	// Check if file already exists (skip re-download)
	if _, err := os.Stat(fullPath); err == nil {
		d.logger.Debug("attachment already exists", "path", localPath)
		d.downloaded[remoteURL] = localPath
		return localPath, nil
	}

	// Ensure attachment folder exists
	attachDir := filepath.Join(d.vaultPath, d.attachmentFolder)
	if err := os.MkdirAll(attachDir, 0o755); err != nil {
		return "", fmt.Errorf("creating attachment folder: %w", err)
	}

	// Download the file
	if err := d.downloadFile(ctx, remoteURL, fullPath); err != nil {
		return "", fmt.Errorf("downloading %s: %w", remoteURL, err)
	}

	d.logger.Debug("downloaded attachment", "url", remoteURL, "to", localPath)
	d.downloaded[remoteURL] = localPath
	return localPath, nil
}

// isDownloadableURL checks if a URL should be downloaded.
// We download Notion-hosted files and common external file hosting.
func (d *Downloader) isDownloadableURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Host)

	// Always download Notion-hosted files
	if strings.Contains(host, "notion") ||
		strings.Contains(host, "s3.us-west-2.amazonaws.com") ||
		strings.Contains(host, "secure.notion-static.com") ||
		strings.Contains(host, "prod-files-secure") {
		return true
	}

	// Download common image/file hosts
	downloadableHosts := []string{
		"imgur.com",
		"i.imgur.com",
		"unsplash.com",
		"images.unsplash.com",
		"pbs.twimg.com",
		"github.com",
		"raw.githubusercontent.com",
		"user-images.githubusercontent.com",
	}

	for _, h := range downloadableHosts {
		if strings.Contains(host, h) {
			return true
		}
	}

	// Check for common file extensions that indicate a direct file link
	ext := strings.ToLower(path.Ext(u.Path))
	downloadableExts := []string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".mp3", ".wav", ".ogg", ".flac",
		".mp4", ".webm", ".mov", ".avi",
		".zip", ".tar", ".gz", ".rar",
	}

	for _, e := range downloadableExts {
		if ext == e {
			return true
		}
	}

	return false
}

// generateFilename creates a unique filename from a URL.
// Format: {hash}_{original_filename}.{ext}
// The hash ensures uniqueness, the original filename helps with readability.
func (d *Downloader) generateFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// Fallback to hash-only filename
		return d.hashURL(rawURL)
	}

	// Extract original filename from path
	originalFilename := path.Base(u.Path)
	if originalFilename == "" || originalFilename == "/" || originalFilename == "." {
		originalFilename = "file"
	}

	// Sanitize the filename
	originalFilename = sanitizeFilenameForAttachment(originalFilename)

	// Get extension
	ext := path.Ext(originalFilename)
	baseName := strings.TrimSuffix(originalFilename, ext)

	// Truncate long base names
	if len(baseName) > 50 {
		baseName = baseName[:50]
	}

	// Create short hash for uniqueness
	hash := d.hashURL(rawURL)[:12]

	if ext != "" {
		return fmt.Sprintf("%s_%s%s", hash, baseName, ext)
	}
	return fmt.Sprintf("%s_%s", hash, baseName)
}

// hashURL creates a SHA256 hash of the URL (truncated).
func (d *Downloader) hashURL(rawURL string) string {
	h := sha256.New()
	h.Write([]byte(rawURL))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// downloadFile downloads a file from a URL to a local path.
func (d *Downloader) downloadFile(ctx context.Context, rawURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set a reasonable user agent
	req.Header.Set("User-Agent", "Notoma/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Copy content
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// Clean up partial file on error
		_ = os.Remove(destPath)
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// sanitizeFilenameForAttachment removes or replaces characters that are
// problematic in filenames.
func sanitizeFilenameForAttachment(name string) string {
	// Decode URL-encoded characters first
	decoded, err := url.QueryUnescape(name)
	if err == nil {
		name = decoded
	}

	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
		"\n", "",
		"\r", "",
		"\t", "",
	)
	name = replacer.Replace(name)

	// Remove leading/trailing whitespace and dots
	name = strings.Trim(name, " .")

	// Ensure name is not empty
	if name == "" {
		name = "file"
	}

	return name
}

// NullDownloader is a no-op downloader that returns original URLs.
// Useful for testing or when attachments shouldn't be downloaded.
type NullDownloader struct{}

// Download returns the original URL without downloading.
func (n *NullDownloader) Download(_ context.Context, remoteURL string) (string, error) {
	return remoteURL, nil
}
