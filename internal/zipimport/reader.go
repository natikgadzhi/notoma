package zipimport

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipReader extracts and reads Notion export zip files.
type ZipReader struct {
	zipPath string
	tempDir string
}

// NewZipReader creates a new reader for a Notion export zip.
func NewZipReader(zipPath string) *ZipReader {
	return &ZipReader{
		zipPath: zipPath,
	}
}

// Extract extracts the zip file to a temporary directory and returns the path.
func (r *ZipReader) Extract() (string, error) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "notoma-import-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}
	r.tempDir = tempDir

	// Open zip file
	zr, err := zip.OpenReader(r.zipPath)
	if err != nil {
		return "", fmt.Errorf("opening zip file: %w", err)
	}
	defer func() { _ = zr.Close() }()

	// Extract all files
	for _, f := range zr.File {
		if err := extractZipFile(f, tempDir); err != nil {
			return "", fmt.Errorf("extracting %s: %w", f.Name, err)
		}
	}

	return tempDir, nil
}

// Cleanup removes the temporary directory.
func (r *ZipReader) Cleanup() error {
	if r.tempDir != "" {
		return os.RemoveAll(r.tempDir)
	}
	return nil
}

// extractZipFile extracts a single file from the zip archive.
func extractZipFile(f *zip.File, destDir string) error {
	// Sanitize path to prevent zip slip vulnerability
	destPath := filepath.Join(destDir, f.Name)
	if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0755)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Create destination file
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	// Open source file in zip
	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Copy contents with size limit to prevent decompression bombs
	const maxFileSize = 100 * 1024 * 1024 // 100MB per file
	_, err = io.CopyN(destFile, srcFile, maxFileSize)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
