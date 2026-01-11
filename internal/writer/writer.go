// Package writer handles writing synced content to the Obsidian vault.
package writer

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Writer handles writing files to the Obsidian vault.
type Writer struct {
	vaultPath        string
	attachmentFolder string
	dryRun           bool
	logger           *slog.Logger
}

// New creates a new Writer instance.
func New(vaultPath, attachmentFolder string, dryRun bool, logger *slog.Logger) *Writer {
	return &Writer{
		vaultPath:        vaultPath,
		attachmentFolder: attachmentFolder,
		dryRun:           dryRun,
		logger:           logger,
	}
}

// WriteMarkdown writes a markdown file to the vault.
// folderPath is relative to the vault root (can be empty for root).
// filename should include .md extension.
// content is the full file content (frontmatter + body).
func (w *Writer) WriteMarkdown(folderPath, filename, content string) error {
	fullPath := filepath.Join(w.vaultPath, folderPath, filename)

	if w.dryRun {
		w.logger.Info("would write", "path", fullPath, "size", len(content))
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", fullPath, err)
	}

	w.logger.Debug("wrote file", "path", fullPath, "size", len(content))
	return nil
}

// WriteBase writes a .base file for a database.
// folderPath is relative to the vault root.
// name is the database name (without extension).
// content is the YAML content for the .base file.
func (w *Writer) WriteBase(folderPath, name string, content []byte) error {
	filename := name + ".base"
	fullPath := filepath.Join(w.vaultPath, folderPath, filename)

	if w.dryRun {
		w.logger.Info("would write base", "path", fullPath, "size", len(content))
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("writing file %s: %w", fullPath, err)
	}

	w.logger.Debug("wrote base file", "path", fullPath, "size", len(content))
	return nil
}

// EnsureFolder creates a folder in the vault if it doesn't exist.
func (w *Writer) EnsureFolder(folderPath string) error {
	fullPath := filepath.Join(w.vaultPath, folderPath)

	if w.dryRun {
		w.logger.Debug("would ensure folder", "path", fullPath)
		return nil
	}

	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		return fmt.Errorf("creating folder %s: %w", fullPath, err)
	}

	return nil
}
