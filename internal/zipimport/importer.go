package zipimport

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/natikgadzhi/notion-based/internal/sync"
	"github.com/natikgadzhi/notion-based/internal/writer"
)

// Importer handles importing a Notion zip export to an Obsidian vault.
type Importer struct {
	zipPath          string
	writer           *writer.Writer
	logger           *slog.Logger
	state            *sync.SyncState
	attachmentFolder string
	dryRun           bool
}

// ImportOptions configures the import behavior.
type ImportOptions struct {
	ZipPath          string
	VaultPath        string
	AttachmentFolder string
	StateFile        string
	DryRun           bool
	Force            bool
}

// NewImporter creates a new importer for a Notion zip export.
func NewImporter(opts ImportOptions, logger *slog.Logger) (*Importer, error) {
	// Validate zip file exists
	if _, err := os.Stat(opts.ZipPath); err != nil {
		return nil, fmt.Errorf("zip file not found: %w", err)
	}

	// Create writer
	w := writer.New(opts.VaultPath, opts.AttachmentFolder, opts.DryRun, logger)

	// Load or create state
	var state *sync.SyncState
	if opts.Force {
		state = sync.NewSyncState()
	} else {
		var err error
		state, err = sync.LoadState(opts.StateFile)
		if err != nil {
			return nil, fmt.Errorf("loading state: %w", err)
		}
	}

	return &Importer{
		zipPath:          opts.ZipPath,
		writer:           w,
		logger:           logger,
		state:            state,
		attachmentFolder: opts.AttachmentFolder,
		dryRun:           opts.DryRun,
	}, nil
}

// ImportResult contains the results of an import operation.
type ImportResult struct {
	PagesImported     int
	DatabasesImported int
	AttachmentsCopied int
	Errors            []error
}

// Import performs the import operation.
func (i *Importer) Import(ctx context.Context) (*ImportResult, error) {
	result := &ImportResult{}

	// Extract zip file
	i.logger.Info("extracting zip file", "path", i.zipPath)
	reader := NewZipReader(i.zipPath)
	extractDir, err := reader.Extract()
	if err != nil {
		return nil, fmt.Errorf("extracting zip: %w", err)
	}
	defer func() {
		if cleanupErr := reader.Cleanup(); cleanupErr != nil {
			i.logger.Warn("failed to cleanup temp directory", "error", cleanupErr)
		}
	}()

	i.logger.Info("zip extracted", "dir", extractDir)

	// Parse the export
	i.logger.Info("parsing Notion export")
	export, err := Parse(extractDir)
	if err != nil {
		return nil, fmt.Errorf("parsing export: %w", err)
	}

	i.logger.Info("parsed export",
		"pages", len(export.Pages),
		"databases", len(export.Databases),
		"attachments", len(export.Attachments),
	)

	// Create converter and register all pages for link resolution
	converter := NewConverter(i.attachmentFolder)
	for _, page := range export.Pages {
		filename := sanitizeFilename(page.Title) + ".md"
		converter.RegisterPage(page.ID, page.Title, filename)
	}

	// Import pages
	for _, page := range export.Pages {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		if err := i.importPage(page, converter); err != nil {
			i.logger.Error("failed to import page", "title", page.Title, "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("page %s: %w", page.Title, err))
			continue
		}
		result.PagesImported++
	}

	// Import databases (including their entries)
	for _, db := range export.Databases {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		if err := i.importDatabase(db, converter); err != nil {
			i.logger.Error("failed to import database", "title", db.Title, "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("database %s: %w", db.Title, err))
			continue
		}
		result.DatabasesImported++
	}

	// Copy attachments
	for relPath, absPath := range export.Attachments {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		if err := i.copyAttachment(relPath, absPath); err != nil {
			i.logger.Error("failed to copy attachment", "path", relPath, "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("attachment %s: %w", relPath, err))
			continue
		}
		result.AttachmentsCopied++
	}

	i.logger.Info("import complete",
		"pages", result.PagesImported,
		"databases", result.DatabasesImported,
		"attachments", result.AttachmentsCopied,
		"errors", len(result.Errors),
	)

	return result, nil
}

// importPage imports a single page to the vault.
func (i *Importer) importPage(page *ExportedPage, converter *Converter) error {
	// Skip pages that are database entries (handled separately)
	for _, db := range i.state.AllDatabaseIDs() {
		if strings.HasPrefix(page.RelPath, db+"/") {
			return nil
		}
	}

	filename := sanitizeFilename(page.Title) + ".md"

	// Determine folder based on parent path
	folder := ""
	if page.ParentPath != "" {
		// Clean up parent path (remove Notion IDs)
		parts := strings.Split(page.ParentPath, string(os.PathSeparator))
		var cleanParts []string
		for _, part := range parts {
			cleanPart := StripNotionID(part)
			if cleanPart != "" {
				cleanParts = append(cleanParts, sanitizeFilename(cleanPart))
			}
		}
		if len(cleanParts) > 0 {
			folder = filepath.Join(cleanParts...)
		}
	}

	// Convert content
	content := converter.ConvertMarkdown(page.Content, page.RelPath)

	if i.dryRun {
		i.logger.Info("would import page", "title", page.Title, "folder", folder, "file", filename)
		return nil
	}

	// Write the file
	if err := i.writer.WriteMarkdown(folder, filename, content); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Update state
	localPath := filename
	if folder != "" {
		localPath = folder + "/" + filename
	}
	i.state.SetResource(sync.ResourceState{
		ID:        page.ID,
		Type:      sync.ResourceTypePage,
		Title:     page.Title,
		LocalPath: localPath,
	})

	i.logger.Debug("imported page", "title", page.Title, "file", localPath)
	return nil
}

// importDatabase imports a database and its entries to the vault.
func (i *Importer) importDatabase(db *ExportedDatabase, converter *Converter) error {
	folder := sanitizeFilename(db.Title)

	if i.dryRun {
		i.logger.Info("would import database",
			"title", db.Title,
			"folder", folder,
			"entries", len(db.Rows),
		)
		return nil
	}

	// Create database folder
	if err := i.writer.EnsureFolder(folder); err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}

	// Generate .base file from CSV headers
	baseContent := generateBaseFromCSV(db.Headers, folder)
	if err := i.writer.WriteBase("", db.Title, []byte(baseContent)); err != nil {
		return fmt.Errorf("writing base file: %w", err)
	}

	// Import each entry
	for idx, row := range db.Rows {
		// Try to find a title column
		title := findTitleValue(row, db.Headers)
		if title == "" {
			title = fmt.Sprintf("Entry %d", idx+1)
		}

		filename := sanitizeFilename(title) + ".md"

		// Generate frontmatter from row data
		frontmatter := ConvertDatabaseToFrontmatter(row, db.Headers)

		// Find matching page content if it exists
		content := ""
		for _, entry := range db.Entries {
			if StripNotionID(entry.Title) == title || entry.Title == title {
				content = converter.ConvertMarkdown(entry.Content, entry.RelPath)
				break
			}
		}

		fullContent := frontmatter + "\n" + content
		if err := i.writer.WriteMarkdown(folder, filename, fullContent); err != nil {
			return fmt.Errorf("writing entry %s: %w", title, err)
		}
	}

	// Update state
	i.state.SetResource(sync.ResourceState{
		ID:        db.ID,
		Type:      sync.ResourceTypeDatabase,
		Title:     db.Title,
		LocalPath: folder,
	})

	i.logger.Debug("imported database", "title", db.Title, "entries", len(db.Rows))
	return nil
}

// copyAttachment copies an attachment to the vault.
func (i *Importer) copyAttachment(relPath, absPath string) error {
	// Clean filename
	filename := filepath.Base(relPath)
	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)
	cleanName := StripNotionID(baseName) + ext

	if i.dryRun {
		i.logger.Info("would copy attachment", "source", relPath, "dest", cleanName)
		return nil
	}

	// Read source file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Write to vault
	destPath := i.attachmentFolder + "/" + cleanName
	if _, err := i.writer.WriteAttachment(destPath, data); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	i.logger.Debug("copied attachment", "file", cleanName)
	return nil
}

// GetState returns the current sync state.
func (i *Importer) GetState() *sync.SyncState {
	return i.state
}

// generateBaseFromCSV creates a .base file content from CSV headers.
func generateBaseFromCSV(headers []string, folder string) string {
	var sb strings.Builder
	sb.WriteString("version: 1\n")
	sb.WriteString("folder: " + folder + "\n")
	sb.WriteString("properties:\n")

	for _, header := range headers {
		key := sanitizeYAMLKey(header)
		propType := inferPropertyType(header)
		sb.WriteString("  - name: " + key + "\n")
		sb.WriteString("    source: frontmatter\n")
		sb.WriteString("    type: " + propType + "\n")
	}

	return sb.String()
}

// inferPropertyType guesses the property type from the header name.
func inferPropertyType(header string) string {
	headerLower := strings.ToLower(header)

	switch {
	case headerLower == "name" || headerLower == "title":
		return "text"
	case strings.Contains(headerLower, "date") || strings.Contains(headerLower, "time"):
		return "date"
	case strings.Contains(headerLower, "tag") || headerLower == "tags":
		return "list"
	case strings.Contains(headerLower, "check") || strings.Contains(headerLower, "done"):
		return "checkbox"
	case strings.Contains(headerLower, "number") || strings.Contains(headerLower, "count") || strings.Contains(headerLower, "amount"):
		return "number"
	case strings.Contains(headerLower, "url") || strings.Contains(headerLower, "link"):
		return "text"
	default:
		return "text"
	}
}

// findTitleValue finds a suitable title value from a row.
func findTitleValue(row map[string]string, headers []string) string {
	// Priority order for title columns
	titleKeys := []string{"Name", "Title", "name", "title", "NAME", "TITLE"}

	for _, key := range titleKeys {
		if val, ok := row[key]; ok && val != "" {
			return val
		}
	}

	// Fall back to first non-empty value
	for _, header := range headers {
		if val := row[header]; val != "" {
			return val
		}
	}

	return ""
}

// sanitizeFilename makes a string safe for use as a filename.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
		"\n", " ",
		"\r", "",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)

	if len(name) > 200 {
		name = name[:200]
	}

	return name
}
