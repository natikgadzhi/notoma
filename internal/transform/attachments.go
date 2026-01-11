// Package transform handles attachment downloads from Notion.
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
	"path"
	"strings"
	"time"
)

// AttachmentType represents the type of attachment.
type AttachmentType string

const (
	AttachmentTypeImage AttachmentType = "image"
	AttachmentTypeFile  AttachmentType = "file"
	AttachmentTypePDF   AttachmentType = "pdf"
	AttachmentTypeAudio AttachmentType = "audio"
	AttachmentTypeVideo AttachmentType = "video"
)

// Attachment represents a downloaded attachment.
type Attachment struct {
	// OriginalURL is the URL from Notion.
	OriginalURL string

	// LocalPath is the path relative to the attachment folder.
	LocalPath string

	// ContentHash is the SHA-256 hash of the file content.
	ContentHash string

	// Type is the attachment type.
	Type AttachmentType

	// Size is the file size in bytes.
	Size int64

	// DownloadedAt is when the attachment was downloaded.
	DownloadedAt time.Time
}

// AttachmentDownloader handles downloading attachments from Notion.
type AttachmentDownloader struct {
	client           *http.Client
	logger           *slog.Logger
	attachmentFolder string
	dryRun           bool

	// downloaded tracks attachments downloaded in this session.
	// Key is the original URL, value is the attachment info.
	downloaded map[string]*Attachment
}

// NewAttachmentDownloader creates a new attachment downloader.
func NewAttachmentDownloader(attachmentFolder string, dryRun bool, logger *slog.Logger) *AttachmentDownloader {
	return &AttachmentDownloader{
		client: &http.Client{
			Timeout: 5 * time.Minute, // Large files may take time
		},
		logger:           logger,
		attachmentFolder: attachmentFolder,
		dryRun:           dryRun,
		downloaded:       make(map[string]*Attachment),
	}
}

// IsNotionHosted returns true if the URL is hosted by Notion.
// Notion-hosted URLs expire after ~1 hour and need to be downloaded immediately.
func IsNotionHosted(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Host)
	return strings.Contains(host, "notion-static.com") ||
		strings.Contains(host, "notion.so") ||
		strings.Contains(host, "amazonaws.com") && strings.Contains(rawURL, "secure.notion")
}

// Download downloads an attachment from the given URL.
// Returns the local path relative to the attachment folder.
// If dryRun is true, returns a path but doesn't actually download.
// If the URL was already downloaded this session, returns the cached path.
func (d *AttachmentDownloader) Download(ctx context.Context, rawURL string, attachmentType AttachmentType) (*Attachment, error) {
	// Check if already downloaded this session
	if att, ok := d.downloaded[rawURL]; ok {
		return att, nil
	}

	// Generate filename from URL
	filename := d.generateFilename(rawURL, attachmentType)
	localPath := path.Join(d.attachmentFolder, filename)

	if d.dryRun {
		d.logger.Info("would download attachment", "url", rawURL, "path", localPath)
		att := &Attachment{
			OriginalURL:  rawURL,
			LocalPath:    localPath,
			Type:         attachmentType,
			DownloadedAt: time.Now(),
		}
		d.downloaded[rawURL] = att
		return att, nil
	}

	// Download the file
	d.logger.Debug("downloading attachment", "url", rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read body and compute hash
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	hash := sha256.Sum256(body)
	hashStr := hex.EncodeToString(hash[:])

	att := &Attachment{
		OriginalURL:  rawURL,
		LocalPath:    localPath,
		ContentHash:  hashStr,
		Type:         attachmentType,
		Size:         int64(len(body)),
		DownloadedAt: time.Now(),
	}

	// Store in cache
	d.downloaded[rawURL] = att

	d.logger.Debug("downloaded attachment",
		"url", rawURL,
		"path", localPath,
		"size", att.Size,
		"hash", hashStr[:16],
	)

	return att, nil
}

// GetDownloaded returns all attachments downloaded in this session.
func (d *AttachmentDownloader) GetDownloaded() map[string]*Attachment {
	return d.downloaded
}

// GetData retrieves the downloaded data for an attachment.
// This re-downloads the file if needed (for writing to disk separately).
func (d *AttachmentDownloader) GetData(ctx context.Context, rawURL string) ([]byte, error) {
	if d.dryRun {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// generateFilename creates a unique filename for an attachment.
func (d *AttachmentDownloader) generateFilename(rawURL string, attachmentType AttachmentType) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Fall back to hash-based name
		hash := sha256.Sum256([]byte(rawURL))
		return hex.EncodeToString(hash[:8]) + extensionForType(attachmentType)
	}

	// Try to extract original filename from path
	filename := path.Base(parsed.Path)

	// Clean up query parameters from filename
	if idx := strings.Index(filename, "?"); idx > 0 {
		filename = filename[:idx]
	}

	// If filename is empty or just an extension, generate from hash
	if filename == "" || filename == "." || !strings.Contains(filename, ".") {
		hash := sha256.Sum256([]byte(rawURL))
		filename = hex.EncodeToString(hash[:8]) + extensionForType(attachmentType)
	}

	// URL decode the filename
	if decoded, err := url.QueryUnescape(filename); err == nil {
		filename = decoded
	}

	// Ensure unique filenames by adding a hash suffix for Notion-hosted files
	// since they often have generic names
	if IsNotionHosted(rawURL) {
		ext := path.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		hash := sha256.Sum256([]byte(rawURL))
		filename = base + "_" + hex.EncodeToString(hash[:4]) + ext
	}

	return sanitizeAttachmentFilename(filename)
}

// sanitizeAttachmentFilename removes or replaces characters that are
// problematic in filenames.
func sanitizeAttachmentFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", "_",
		"\r", "",
		" ", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)

	// Limit length
	if len(name) > 200 {
		ext := path.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(base) > 200-len(ext) {
			base = base[:200-len(ext)]
		}
		name = base + ext
	}

	return name
}

// extensionForType returns a default file extension for an attachment type.
func extensionForType(t AttachmentType) string {
	switch t {
	case AttachmentTypeImage:
		return ".png"
	case AttachmentTypePDF:
		return ".pdf"
	case AttachmentTypeAudio:
		return ".mp3"
	case AttachmentTypeVideo:
		return ".mp4"
	default:
		return ".bin"
	}
}

// MarkdownPathForAttachment returns the markdown path to reference an attachment.
// This handles the path format for Obsidian.
func MarkdownPathForAttachment(attachmentFolder, localPath string) string {
	// In Obsidian, we use relative paths from the vault root
	// The local path already includes the attachment folder
	return localPath
}
