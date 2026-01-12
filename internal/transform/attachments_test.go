package transform

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsNotionHosted(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "prod-files-secure S3 URL",
			url:      "https://prod-files-secure.s3.us-west-2.amazonaws.com/b3c89895-9b25-481d-a83d-2b78ea6c744b/1b9010af-65cb-4021-a378-b4a5a045fad9/file.vcf",
			expected: true,
		},
		{
			name:     "prod-files-secure S3 URL with query params",
			url:      "https://prod-files-secure.s3.us-west-2.amazonaws.com/abc123/image.png?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=test",
			expected: true,
		},
		{
			name:     "secure.notion-static.com URL",
			url:      "https://secure.notion-static.com/abc123/file.pdf",
			expected: true,
		},
		{
			name:     "notion.so URL",
			url:      "https://www.notion.so/abc123/image.png",
			expected: true,
		},
		{
			name:     "external URL - imgur",
			url:      "https://i.imgur.com/abc123.png",
			expected: false,
		},
		{
			name:     "external URL - github",
			url:      "https://raw.githubusercontent.com/user/repo/image.png",
			expected: false,
		},
		{
			name:     "external URL - generic S3",
			url:      "https://my-bucket.s3.amazonaws.com/file.png",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotionHosted(tt.url)
			if result != tt.expected {
				t.Errorf("IsNotionHosted(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestSanitizeAttachmentFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "image.png",
			expected: "image.png",
		},
		{
			name:     "filename with spaces",
			input:    "my image.png",
			expected: "my_image.png",
		},
		{
			name:     "filename with special chars",
			input:    "file:name?.png",
			expected: "file_name_.png",
		},
		{
			name:     "filename with slashes",
			input:    "path/to/file.png",
			expected: "path_to_file.png",
		},
		{
			name:     "very long filename",
			input:    strings.Repeat("a", 250) + ".png",
			expected: strings.Repeat("a", 196) + ".png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeAttachmentFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeAttachmentFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtensionForType(t *testing.T) {
	tests := []struct {
		attachType AttachmentType
		expected   string
	}{
		{AttachmentTypeImage, ".png"},
		{AttachmentTypePDF, ".pdf"},
		{AttachmentTypeAudio, ".mp3"},
		{AttachmentTypeVideo, ".mp4"},
		{AttachmentTypeFile, ".bin"},
	}

	for _, tt := range tests {
		t.Run(string(tt.attachType), func(t *testing.T) {
			result := extensionForType(tt.attachType)
			if result != tt.expected {
				t.Errorf("extensionForType(%q) = %q, want %q", tt.attachType, result, tt.expected)
			}
		})
	}
}

func TestAttachmentDownloader_DryRun(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	downloader := NewAttachmentDownloader("_attachments", true, logger)

	// In dry-run mode, download should succeed without actually downloading
	att, err := downloader.Download(context.Background(), "https://example.com/image.png", AttachmentTypeImage)
	if err != nil {
		t.Errorf("Download() in dry-run mode returned error: %v", err)
	}
	if att == nil {
		t.Fatal("Download() returned nil attachment")
	}
	if att.LocalPath == "" {
		t.Error("Download() returned attachment with empty LocalPath")
	}

	// Should be in downloaded cache
	downloaded := downloader.GetDownloaded()
	if len(downloaded) != 1 {
		t.Errorf("GetDownloaded() returned %d items, want 1", len(downloaded))
	}
}

func TestAttachmentDownloader_Download(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake png data"))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	downloader := NewAttachmentDownloader("_attachments", false, logger)

	// Download should succeed
	att, err := downloader.Download(context.Background(), server.URL+"/test.png", AttachmentTypeImage)
	if err != nil {
		t.Errorf("Download() returned error: %v", err)
	}
	if att == nil {
		t.Fatal("Download() returned nil attachment")
	}
	if att.ContentHash == "" {
		t.Error("Download() returned attachment with empty ContentHash")
	}
	if att.Size == 0 {
		t.Error("Download() returned attachment with zero Size")
	}

	// Second download of same URL should return cached result
	att2, err := downloader.Download(context.Background(), server.URL+"/test.png", AttachmentTypeImage)
	if err != nil {
		t.Errorf("Second Download() returned error: %v", err)
	}
	if att2 != att {
		t.Error("Second Download() should return same cached attachment")
	}
}

func TestAttachmentDownloader_DownloadError(t *testing.T) {
	// Create a test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	downloader := NewAttachmentDownloader("_attachments", false, logger)

	// Download should fail
	_, err := downloader.Download(context.Background(), server.URL+"/notfound.png", AttachmentTypeImage)
	if err == nil {
		t.Error("Download() should return error for 404 response")
	}
}

func TestAttachmentDownloader_GenerateFilename(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	downloader := NewAttachmentDownloader("_attachments", true, logger)

	tests := []struct {
		name       string
		url        string
		attachType AttachmentType
		wantSuffix string
	}{
		{
			name:       "URL with filename",
			url:        "https://example.com/images/photo.jpg",
			attachType: AttachmentTypeImage,
			wantSuffix: ".jpg",
		},
		{
			name:       "URL without extension",
			url:        "https://example.com/file",
			attachType: AttachmentTypeImage,
			wantSuffix: ".png", // default for image type
		},
		{
			name:       "Notion-hosted URL",
			url:        "https://secure.notion-static.com/abc/image.png",
			attachType: AttachmentTypeImage,
			wantSuffix: ".png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := downloader.generateFilename(tt.url, tt.attachType)
			if !strings.HasSuffix(result, tt.wantSuffix) {
				t.Errorf("generateFilename(%q, %q) = %q, want suffix %q", tt.url, tt.attachType, result, tt.wantSuffix)
			}
		})
	}
}
