package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/jomei/notionapi"
	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/notion"
	"github.com/natikgadzhi/notion-based/internal/transform"
	"github.com/natikgadzhi/notion-based/internal/writer"
)

// Syncer orchestrates synchronization between Notion and Obsidian.
type Syncer struct {
	cfg        *config.Config
	client     *notion.Client
	writer     *writer.Writer
	downloader *transform.Downloader
	state      *State
	logger     *slog.Logger
	dryRun     bool
	force      bool
}

// SyncerOption is a functional option for configuring the Syncer.
type SyncerOption func(*Syncer)

// WithDryRun enables dry-run mode (no writes).
func WithDryRun(dryRun bool) SyncerOption {
	return func(s *Syncer) {
		s.dryRun = dryRun
	}
}

// WithForce forces a full resync, ignoring state.
func WithForce(force bool) SyncerOption {
	return func(s *Syncer) {
		s.force = force
	}
}

// NewSyncer creates a new Syncer instance.
func NewSyncer(cfg *config.Config, logger *slog.Logger, opts ...SyncerOption) (*Syncer, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Load or create state
	state, err := LoadState(cfg.State.File)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	s := &Syncer{
		cfg:    cfg,
		client: notion.NewClient(cfg.NotionToken, logger),
		state:  state,
		logger: logger,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create writer
	s.writer = writer.New(cfg.Output.VaultPath, cfg.Output.AttachmentFolder, s.dryRun, logger)

	// Create attachment downloader
	s.downloader = transform.NewDownloader(transform.DownloaderConfig{
		VaultPath:        cfg.Output.VaultPath,
		AttachmentFolder: cfg.Output.AttachmentFolder,
		Enabled:          cfg.Options.DownloadAttachments,
		DryRun:           s.dryRun,
		Logger:           logger,
	})

	// Reset state if force sync
	if s.force {
		s.state.Reset()
	}

	return s, nil
}

// SyncResult contains statistics from a sync operation.
type SyncResult struct {
	PagesProcessed        int
	PagesUpdated          int
	PagesSkipped          int
	DatabasesProcessed    int
	DatabasesUpdated      int
	AttachmentsDownloaded int
	Errors                []error
	Duration              time.Duration
}

// Run executes the sync operation.
func (s *Syncer) Run(ctx context.Context) (*SyncResult, error) {
	start := time.Now()
	result := &SyncResult{}

	s.logger.Info("starting sync",
		"roots", len(s.cfg.Sync.Roots),
		"dry_run", s.dryRun,
		"force", s.force,
	)

	for _, root := range s.cfg.Sync.Roots {
		if err := s.syncRoot(ctx, root, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("syncing root %s: %w", root.URL, err))
			s.logger.Error("sync root failed", "url", root.URL, "error", err)
		}
	}

	// Save state (unless dry run)
	if !s.dryRun {
		s.state.MarkSynced()
		if err := s.state.Save(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("saving state: %w", err))
		}
	}

	result.Duration = time.Since(start)

	s.logger.Info("sync complete",
		"pages_processed", result.PagesProcessed,
		"pages_updated", result.PagesUpdated,
		"pages_skipped", result.PagesSkipped,
		"databases_processed", result.DatabasesProcessed,
		"databases_updated", result.DatabasesUpdated,
		"attachments", result.AttachmentsDownloaded,
		"errors", len(result.Errors),
		"duration", result.Duration,
	)

	return result, nil
}

// syncRoot syncs a single root (page or database).
func (s *Syncer) syncRoot(ctx context.Context, root config.Root, result *SyncResult) error {
	// Parse the URL to get the ID
	parsed, err := notion.ParseURL(root.URL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	// Detect resource type
	resource, err := s.client.DetectResourceType(ctx, parsed.ID)
	if err != nil {
		return fmt.Errorf("detecting resource type: %w", err)
	}

	name := root.Name
	if name == "" {
		name = resource.Title
	}

	s.logger.Info("syncing root",
		"name", name,
		"type", resource.Type,
		"id", resource.ID,
	)

	switch resource.Type {
	case notion.ResourceTypePage:
		return s.syncPage(ctx, resource.ID, "", result)
	case notion.ResourceTypeDatabase:
		return s.syncDatabase(ctx, resource.ID, name, result)
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}
}

// syncPage syncs a single page.
func (s *Syncer) syncPage(ctx context.Context, pageID, folder string, result *SyncResult) error {
	result.PagesProcessed++

	// Fetch the page
	page, err := s.client.GetPage(ctx, pageID)
	if err != nil {
		return fmt.Errorf("fetching page: %w", err)
	}

	lastEdited := page.LastEditedTime

	// Check if sync is needed (unless force)
	if !s.force && !s.state.NeedsSync(pageID, lastEdited) {
		result.PagesSkipped++
		s.logger.Debug("page up to date, skipping", "id", pageID)
		return nil
	}

	// Get page title
	title := extractPageTitle(page)
	if title == "" {
		title = "Untitled"
	}

	s.logger.Debug("syncing page", "id", pageID, "title", title)

	// Fetch blocks
	blocks, err := s.client.GetBlockChildren(ctx, pageID)
	if err != nil {
		return fmt.Errorf("fetching blocks: %w", err)
	}

	// Transform to markdown
	transformer := transform.NewTransformer(ctx, s.client, transform.WithDownloader(s.downloader))
	content, err := transformer.BlocksToMarkdown(blocks)
	if err != nil {
		return fmt.Errorf("transforming blocks: %w", err)
	}

	// Generate frontmatter
	frontmatter := generateFrontmatter(page)
	fullContent := frontmatter + content

	// Calculate content hash
	contentHash := hashContent(fullContent)

	// Check if content changed
	existingState := s.state.GetPage(pageID)
	if existingState != nil && existingState.ContentHash == contentHash {
		result.PagesSkipped++
		s.logger.Debug("content unchanged, skipping write", "id", pageID)
		// Still update the sync time
		existingState.SyncedAt = time.Now()
		s.state.SetPage(existingState)
		return nil
	}

	// Write the file
	filename := sanitizeFilename(title) + ".md"
	if err := s.writer.WriteMarkdown(folder, filename, fullContent); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Update state
	outputPath := filename
	if folder != "" {
		outputPath = folder + "/" + filename
	}

	s.state.SetPage(&PageState{
		ID:          pageID,
		Title:       title,
		LastEdited:  lastEdited,
		ContentHash: contentHash,
		OutputPath:  outputPath,
		SyncedAt:    time.Now(),
	})

	result.PagesUpdated++
	s.logger.Info("synced page", "title", title, "path", outputPath)

	return nil
}

// syncDatabase syncs a database and all its entries.
func (s *Syncer) syncDatabase(ctx context.Context, databaseID, name string, result *SyncResult) error {
	result.DatabasesProcessed++

	// Fetch database metadata
	db, err := s.client.GetDatabase(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("fetching database: %w", err)
	}

	lastEdited := db.LastEditedTime

	// Use provided name or database title
	if name == "" {
		name = extractDatabaseTitle(db)
		if name == "" {
			name = "Database"
		}
	}

	folderName := sanitizeFilename(name)

	s.logger.Debug("syncing database", "id", databaseID, "name", name)

	// Ensure output folder exists
	if err := s.writer.EnsureFolder(folderName); err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}

	// Parse database schema
	schema, err := transform.ParseDatabaseSchema(db)
	if err != nil {
		return fmt.Errorf("parsing database schema: %w", err)
	}

	// Generate and write .base file
	baseFile, err := transform.GenerateBaseFile(schema, folderName)
	if err != nil {
		return fmt.Errorf("generating base file: %w", err)
	}
	baseContent, err := transform.MarshalBaseFile(baseFile)
	if err != nil {
		return fmt.Errorf("serializing base file: %w", err)
	}

	if err := s.writer.WriteBase(folderName, folderName, baseContent); err != nil {
		return fmt.Errorf("writing base file: %w", err)
	}

	// Query all pages in the database
	pages, err := s.client.QueryDatabase(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("querying database: %w", err)
	}

	s.logger.Debug("found database entries", "count", len(pages))

	// Sync each entry
	for _, page := range pages {
		if err := s.syncDatabaseEntry(ctx, &page, schema, folderName, result); err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("syncing entry %s: %w", page.ID, err))
			s.logger.Error("failed to sync entry", "id", page.ID, "error", err)
		}
	}

	// Update database state
	s.state.SetDatabase(&DatabaseState{
		ID:           databaseID,
		Title:        name,
		LastEdited:   lastEdited,
		OutputFolder: folderName,
		SyncedAt:     time.Now(),
		EntryCount:   len(pages),
	})

	result.DatabasesUpdated++
	s.logger.Info("synced database", "name", name, "entries", len(pages))

	return nil
}

// syncDatabaseEntry syncs a single database entry (page within a database).
func (s *Syncer) syncDatabaseEntry(ctx context.Context, page *notionapi.Page, schema *transform.DatabaseSchema, folder string, result *SyncResult) error {
	result.PagesProcessed++
	pageID := string(page.ID)

	lastEdited := page.LastEditedTime

	// Check if sync is needed
	if !s.force && !s.state.NeedsSync(pageID, lastEdited) {
		result.PagesSkipped++
		return nil
	}

	// Extract entry data from page
	entryData, err := transform.ExtractEntryData(page, schema)
	if err != nil {
		return fmt.Errorf("extracting entry data: %w", err)
	}

	// Fetch and transform content blocks
	blocks, err := s.client.GetBlockChildren(ctx, pageID)
	if err != nil {
		return fmt.Errorf("fetching blocks: %w", err)
	}

	transformer := transform.NewTransformer(ctx, s.client, transform.WithDownloader(s.downloader))
	markdownContent, err := transformer.BlocksToMarkdown(blocks)
	if err != nil {
		return fmt.Errorf("transforming blocks: %w", err)
	}

	// Build complete entry (frontmatter + content)
	entry, err := transform.BuildDatabaseEntry(entryData, markdownContent)
	if err != nil {
		return fmt.Errorf("building entry: %w", err)
	}

	fullContent := entry.Frontmatter + entry.Content
	contentHash := hashContent(fullContent)

	// Check if content changed
	existingState := s.state.GetPage(pageID)
	if existingState != nil && existingState.ContentHash == contentHash {
		result.PagesSkipped++
		existingState.SyncedAt = time.Now()
		s.state.SetPage(existingState)
		return nil
	}

	// Write file
	if err := s.writer.WriteMarkdown(folder, entry.Filename, fullContent); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Update state
	outputPath := folder + "/" + entry.Filename
	title := entryData.Title

	s.state.SetPage(&PageState{
		ID:          pageID,
		Title:       title,
		LastEdited:  lastEdited,
		ContentHash: contentHash,
		OutputPath:  outputPath,
		SyncedAt:    time.Now(),
	})

	result.PagesUpdated++

	return nil
}

// State returns the current sync state.
func (s *Syncer) State() *State {
	return s.state
}

// extractPageTitle extracts the title from a page's properties.
func extractPageTitle(page *notionapi.Page) string {
	if page == nil || page.Properties == nil {
		return ""
	}

	for _, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok {
			var result string
			for _, rt := range titleProp.Title {
				result += rt.PlainText
			}
			return result
		}
	}
	return ""
}

// extractDatabaseTitle extracts the title from a database.
func extractDatabaseTitle(db *notionapi.Database) string {
	if db == nil {
		return ""
	}
	var result string
	for _, rt := range db.Title {
		result += rt.PlainText
	}
	return result
}

// generateFrontmatter creates YAML frontmatter for a page.
func generateFrontmatter(page *notionapi.Page) string {
	return fmt.Sprintf("---\nnotion_id: %s\n---\n\n", page.ID)
}

// hashContent creates a SHA256 hash of content.
func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

// sanitizeFilename removes or replaces problematic characters from filenames.
func sanitizeFilename(name string) string {
	// Use the transform package's sanitizer
	return transform.SanitizeFilename(name)
}
