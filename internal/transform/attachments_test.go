package transform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloader_generateFilename(t *testing.T) {
	d := NewDownloader(DownloaderConfig{})

	tests := []struct {
		name    string
		url     string
		wantExt string
	}{
		{
			name:    "simple image URL",
			url:     "https://example.com/images/photo.png",
			wantExt: ".png",
		},
		{
			name:    "URL with query params",
			url:     "https://s3.amazonaws.com/bucket/file.jpg?token=abc123",
			wantExt: ".jpg",
		},
		{
			name:    "notion S3 URL",
			url:     "https://s3.us-west-2.amazonaws.com/secure.notion-static.com/abc/image.webp",
			wantExt: ".webp",
		},
		{
			name:    "URL with encoded characters",
			url:     "https://example.com/My%20Document.pdf",
			wantExt: ".pdf",
		},
		{
			name:    "URL without extension",
			url:     "https://example.com/api/file/abc123",
			wantExt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := d.generateFilename(tt.url)

			// Check that filename is not empty
			if filename == "" {
				t.Error("generateFilename returned empty string")
			}

			// Check extension
			if tt.wantExt != "" {
				if !strings.HasSuffix(filename, tt.wantExt) {
					t.Errorf("expected filename to end with %q, got %q", tt.wantExt, filename)
				}
			}

			// Check that filename contains hash prefix
			if len(filename) < 12 {
				t.Errorf("filename too short, expected hash prefix: %q", filename)
			}
		})
	}
}

func TestDownloader_isDownloadableURL(t *testing.T) {
	d := NewDownloader(DownloaderConfig{})

	tests := []struct {
		url  string
		want bool
	}{
		// Notion URLs - should download
		{"https://s3.us-west-2.amazonaws.com/secure.notion-static.com/abc/image.png", true},
		{"https://prod-files-secure.s3.us-west-2.amazonaws.com/abc/image.jpg", true},
		{"https://www.notion.so/images/page-cover/nasa_tim_peake_spacewalk.jpg", true},

		// Common file hosts - should download
		{"https://i.imgur.com/abc123.png", true},
		{"https://images.unsplash.com/photo-123.jpg", true},
		{"https://raw.githubusercontent.com/user/repo/main/image.png", true},

		// Direct file links - should download
		{"https://example.com/files/document.pdf", true},
		{"https://example.com/assets/image.png", true},
		{"https://example.com/media/video.mp4", true},

		// Non-downloadable URLs
		{"https://www.youtube.com/watch?v=abc123", false},
		{"https://twitter.com/user/status/123", false},
		{"https://example.com/page", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := d.isDownloadableURL(tt.url)
			if got != tt.want {
				t.Errorf("isDownloadableURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestDownloader_Download_Disabled(t *testing.T) {
	d := NewDownloader(DownloaderConfig{
		Enabled: false,
	})

	url := "https://example.com/image.png"
	result, err := d.Download(context.Background(), url)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != url {
		t.Errorf("expected original URL when disabled, got %q", result)
	}
}

func TestDownloader_Download_NonDownloadable(t *testing.T) {
	d := NewDownloader(DownloaderConfig{
		Enabled: true,
	})

	url := "https://www.youtube.com/watch?v=abc123"
	result, err := d.Download(context.Background(), url)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != url {
		t.Errorf("expected original URL for non-downloadable, got %q", result)
	}
}

func TestDownloader_Download_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	d := NewDownloader(DownloaderConfig{
		VaultPath:        tmpDir,
		AttachmentFolder: "_attachments",
		Enabled:          true,
		DryRun:           true,
	})

	url := "https://s3.us-west-2.amazonaws.com/test/image.png"
	result, err := d.Download(context.Background(), url)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should return local path but not create file
	if !strings.HasPrefix(result, "_attachments/") {
		t.Errorf("expected local path prefix, got %q", result)
	}

	// Verify file was NOT created
	fullPath := filepath.Join(tmpDir, result)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("file should not exist in dry-run mode")
	}
}

func TestDownloader_Download_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := NewDownloader(DownloaderConfig{
		VaultPath:        tmpDir,
		AttachmentFolder: "_attachments",
		Enabled:          true,
		DryRun:           false,
	})

	url := server.URL + "/test-image.png"
	result, err := d.Download(context.Background(), url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return local path
	if !strings.HasPrefix(result, "_attachments/") {
		t.Errorf("expected local path prefix, got %q", result)
	}
	if !strings.HasSuffix(result, ".png") {
		t.Errorf("expected .png extension, got %q", result)
	}

	// Verify file was created
	fullPath := filepath.Join(tmpDir, result)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "fake image data" {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestDownloader_Download_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := NewDownloader(DownloaderConfig{
		VaultPath:        tmpDir,
		AttachmentFolder: "_attachments",
		Enabled:          true,
	})

	url := server.URL + "/image.png"

	// First download
	result1, err := d.Download(context.Background(), url)
	if err != nil {
		t.Fatalf("first download failed: %v", err)
	}

	// Second download should use cache
	result2, err := d.Download(context.Background(), url)
	if err != nil {
		t.Fatalf("second download failed: %v", err)
	}

	if result1 != result2 {
		t.Errorf("cached result mismatch: %q != %q", result1, result2)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (cached), got %d", callCount)
	}
}

func TestDownloader_Download_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	attachDir := filepath.Join(tmpDir, "_attachments")
	if err := os.MkdirAll(attachDir, 0o755); err != nil {
		t.Fatalf("failed to create attachments dir: %v", err)
	}

	// Pre-create a file
	existingFile := filepath.Join(attachDir, "existing.png")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0o644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	// Create downloader with URL that would generate "existing.png"
	d := NewDownloader(DownloaderConfig{
		VaultPath:        tmpDir,
		AttachmentFolder: "_attachments",
		Enabled:          true,
	})

	// The filename is based on URL hash, so we need to use the download method
	// and verify the existing file check works for the cache
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("new content"))
	}))
	defer server.Close()

	url := server.URL + "/image.png"

	// Download once
	result1, _ := d.Download(context.Background(), url)

	// Create a new downloader (fresh cache) and download again
	d2 := NewDownloader(DownloaderConfig{
		VaultPath:        tmpDir,
		AttachmentFolder: "_attachments",
		Enabled:          true,
	})

	result2, _ := d2.Download(context.Background(), url)

	// Should return same path
	if result1 != result2 {
		t.Errorf("paths should match: %q != %q", result1, result2)
	}
}

func TestDownloader_Download_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	d := NewDownloader(DownloaderConfig{
		VaultPath:        tmpDir,
		AttachmentFolder: "_attachments",
		Enabled:          true,
	})

	url := server.URL + "/notfound.png"
	_, err := d.Download(context.Background(), url)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestNullDownloader(t *testing.T) {
	d := &NullDownloader{}
	url := "https://example.com/image.png"

	result, err := d.Download(context.Background(), url)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != url {
		t.Errorf("expected original URL, got %q", result)
	}
}

func TestSanitizeFilenameForAttachment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple.png", "simple.png"},
		{"file/with/slashes.jpg", "file-with-slashes.jpg"},
		{"file:with:colons.pdf", "file-with-colons.pdf"},
		{"file?with?questions.doc", "filewithquestions.doc"},
		{"file with spaces.png", "file with spaces.png"},
		{"  spaces  .png", "spaces  .png"},
		{"..dots..", "dots"},
		{"My%20Document.pdf", "My Document.pdf"},
		{"", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilenameForAttachment(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilenameForAttachment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
