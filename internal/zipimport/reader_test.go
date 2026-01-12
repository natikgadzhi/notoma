package zipimport

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestZipReader_Extract(t *testing.T) {
	// Create a test zip file
	zipPath := filepath.Join(t.TempDir(), "test.zip")

	// Create zip file with test content
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zw := zip.NewWriter(zipFile)

	// Add a markdown file
	mdWriter, err := zw.Create("Test Page abc12345.md")
	if err != nil {
		t.Fatalf("failed to create md entry: %v", err)
	}
	if _, err := mdWriter.Write([]byte("# Test Page\n\nSome content here.")); err != nil {
		t.Fatalf("failed to write md content: %v", err)
	}

	// Add a subfolder with a file
	subWriter, err := zw.Create("Subfolder/Child Page def67890.md")
	if err != nil {
		t.Fatalf("failed to create sub entry: %v", err)
	}
	if _, err := subWriter.Write([]byte("# Child Page\n\nNested content.")); err != nil {
		t.Fatalf("failed to write sub content: %v", err)
	}

	// Add a CSV file
	csvWriter, err := zw.Create("Database abc12345.csv")
	if err != nil {
		t.Fatalf("failed to create csv entry: %v", err)
	}
	if _, err := csvWriter.Write([]byte("Name,Tags,Date\nEntry 1,tag1,2024-01-01\nEntry 2,\"tag2,tag3\",2024-01-02")); err != nil {
		t.Fatalf("failed to write csv content: %v", err)
	}

	// Add an image
	imgWriter, err := zw.Create("image.png")
	if err != nil {
		t.Fatalf("failed to create img entry: %v", err)
	}
	// Write fake PNG header
	if _, err := imgWriter.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
		t.Fatalf("failed to write img content: %v", err)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}

	// Test extraction
	reader := NewZipReader(zipPath)
	extractDir, err := reader.Extract()
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}
	defer func() { _ = reader.Cleanup() }()

	// Verify extracted files
	tests := []struct {
		path    string
		wantLen int
	}{
		{"Test Page abc12345.md", 31},
		{"Subfolder/Child Page def67890.md", 29},
		{"Database abc12345.csv", 69},
		{"image.png", 8},
	}

	for _, tt := range tests {
		fullPath := filepath.Join(extractDir, tt.path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", tt.path, err)
			continue
		}
		if len(data) != tt.wantLen {
			t.Errorf("%s: got %d bytes, want %d", tt.path, len(data), tt.wantLen)
		}
	}
}

func TestZipReader_ZipSlipPrevention(t *testing.T) {
	// Create a malicious zip file with path traversal
	zipPath := filepath.Join(t.TempDir(), "malicious.zip")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zw := zip.NewWriter(zipFile)

	// Try to add file with path traversal
	maliciousWriter, err := zw.Create("../../../etc/passwd")
	if err != nil {
		t.Fatalf("failed to create malicious entry: %v", err)
	}
	if _, err := maliciousWriter.Write([]byte("malicious content")); err != nil {
		t.Fatalf("failed to write malicious content: %v", err)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}

	// Extraction should fail or sanitize the path
	reader := NewZipReader(zipPath)
	_, err = reader.Extract()
	if err == nil {
		t.Error("expected error for zip slip attack, got nil")
		_ = reader.Cleanup()
	}
}

func TestZipReader_Cleanup(t *testing.T) {
	// Create a simple test zip
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zw := zip.NewWriter(zipFile)
	w, _ := zw.Create("test.txt")
	if _, err := w.Write([]byte("test")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}

	reader := NewZipReader(zipPath)
	extractDir, err := reader.Extract()
	if err != nil {
		t.Fatalf("failed to extract: %v", err)
	}

	// Verify temp dir exists
	if _, err := os.Stat(extractDir); err != nil {
		t.Fatalf("extract dir should exist: %v", err)
	}

	// Cleanup
	if err := reader.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Verify temp dir is removed
	if _, err := os.Stat(extractDir); !os.IsNotExist(err) {
		t.Error("extract dir should be removed after cleanup")
	}
}
