package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jomei/notionapi"
	"github.com/lmittmann/tint"
	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/notion"
	"github.com/natikgadzhi/notion-based/internal/sync"
	"github.com/natikgadzhi/notion-based/internal/transform"
	"github.com/natikgadzhi/notion-based/internal/tui"
	"github.com/natikgadzhi/notion-based/internal/writer"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	configPath string
	dryRun     bool
	force      bool
	verbose    bool
	quiet      bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Notion content to Obsidian vault",
	Long: `Sync fetches pages and databases from Notion and converts them
to Obsidian-flavored markdown files in your vault.

By default, it performs incremental sync - only fetching pages
modified since the last sync. Use --force to perform a full resync.`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "path to config file")
	syncCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview changes without writing files")
	syncCmd.Flags().BoolVarP(&force, "force", "f", false, "ignore state and perform full resync")
	syncCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	syncCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "disable TUI, use plain log output")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Determine if we should use TUI mode
	// Use TUI by default if stdout is a TTY and quiet mode is not enabled
	useTUI := !quiet && term.IsTerminal(int(os.Stdout.Fd()))

	// Set up logging
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	var logOutput io.Writer = os.Stderr
	if useTUI {
		// In TUI mode, suppress logs unless verbose
		if !verbose {
			logOutput = io.Discard
		}
	}

	logger := slog.New(tint.NewHandler(logOutput, &tint.Options{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("received shutdown signal, canceling...")
		cancel()
	}()

	// Load configuration
	logger.Info("loading configuration", "path", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if dryRun {
		logger.Info("dry-run mode enabled, no files will be written")
	}

	// Load sync state (or create new if --force or doesn't exist)
	var state *sync.SyncState
	if force {
		logger.Info("force mode enabled, ignoring state and performing full resync")
		state = sync.NewSyncState()
	} else {
		state, err = sync.LoadState(cfg.State.File)
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}
		if state.ResourceCount() > 0 {
			logger.Info("loaded sync state",
				"resources", state.ResourceCount(),
				"entries", state.EntryCount(),
				"last_sync", state.LastSyncTime.Format(time.RFC3339),
			)
		} else {
			logger.Info("no previous sync state found, performing full sync")
		}
	}

	// Create Notion client
	client := notion.NewClient(cfg.NotionToken, logger)

	// Validate connection by fetching current user
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("validating Notion token: %w", err)
	}
	logger.Info("connected to Notion", "bot", user.Name)

	// Create writer
	w := writer.New(cfg.Output.VaultPath, cfg.Output.AttachmentFolder, dryRun, logger)

	// Create attachment downloader if enabled
	var attDownloader *transform.AttachmentDownloader
	if cfg.Options.DownloadAttachments {
		attDownloader = transform.NewAttachmentDownloader(cfg.Output.AttachmentFolder, dryRun, logger)
		logger.Info("attachment downloading enabled", "folder", cfg.Output.AttachmentFolder)
	}

	// Create TUI runner if in TUI mode
	var tuiRunner *tui.Runner
	if useTUI {
		tuiRunner = tui.NewRunner()
		if err := tuiRunner.Start(); err != nil {
			return fmt.Errorf("starting TUI: %w", err)
		}
	}

	// Build list of roots to process
	roots := cfg.Sync.Roots

	// Discover workspace roots if enabled
	if cfg.Sync.DiscoverWorkspaceRoots {
		logger.Info("discovering workspace roots")
		discovered, err := client.DiscoverWorkspaceRoots(ctx)
		if err != nil {
			return fmt.Errorf("discovering workspace roots: %w", err)
		}
		logger.Info("discovered workspace roots", "count", len(discovered))

		// Convert discovered resources to config.Root format
		for _, res := range discovered {
			// Use Notion URL format for discovered roots
			url := fmt.Sprintf("https://notion.so/%s", strings.ReplaceAll(res.ID, "-", ""))
			roots = append(roots, config.Root{
				URL:  url,
				Name: res.Title,
			})
		}
	}

	// Process each root
	var syncErr error
	for _, root := range roots {
		if err := processRoot(ctx, client, w, logger, cfg, root, dryRun, state, tuiRunner, attDownloader); err != nil {
			logger.Error("failed to process root", "url", root.URL, "error", err)
			syncErr = err
			// Continue with other roots
		}
	}

	// Write downloaded attachments to disk
	if attDownloader != nil && !dryRun {
		downloaded := attDownloader.GetDownloaded()
		for url, att := range downloaded {
			// Get attachment data and write it
			data, err := attDownloader.GetData(ctx, url)
			if err != nil {
				logger.Error("failed to download attachment data", "url", url, "error", err)
				continue
			}
			if _, err := w.WriteAttachment(att.LocalPath, data); err != nil {
				logger.Error("failed to write attachment", "path", att.LocalPath, "error", err)
				continue
			}
			// Update attachment state
			state.UpdateAttachmentState(url, att.ContentHash, att.LocalPath, att.Size, "")
		}
		logger.Info("downloaded attachments", "count", len(downloaded))
	}

	// Save state (unless dry-run)
	if !dryRun {
		if err := sync.SaveState(cfg.State.File, state); err != nil {
			logger.Error("failed to save state", "error", err)
			if syncErr == nil {
				syncErr = err
			}
		} else {
			logger.Info("saved sync state", "path", cfg.State.File)
		}
	}

	// Signal completion to TUI
	if tuiRunner != nil {
		tuiRunner.Done(syncErr)
		tuiRunner.Wait()
	} else {
		logger.Info("sync complete")
	}

	return syncErr
}

func processRoot(ctx context.Context, client *notion.Client, w *writer.Writer, logger *slog.Logger, cfg *config.Config, root config.Root, dryRun bool, state *sync.SyncState, tuiRunner *tui.Runner, attDownloader *transform.AttachmentDownloader) error {
	// Parse URL to get ID
	parsed, err := notion.ParseURL(root.URL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	name := root.Name
	if name == "" {
		name = parsed.ID[:8] + "..."
	}
	logger.Info("processing root", "name", name, "id", parsed.ID)

	// Detect resource type
	resource, err := client.DetectResourceType(ctx, parsed.ID)
	if err != nil {
		return fmt.Errorf("detecting resource type: %w", err)
	}

	logger.Info("detected resource",
		"type", resource.Type,
		"title", resource.Title,
		"id", resource.ID,
	)

	// Add to TUI if available
	if tuiRunner != nil {
		itemType := tui.TypePage
		if resource.Type == notion.ResourceTypeDatabase {
			itemType = tui.TypeDatabase
		}
		tuiRunner.AddRoot(resource.ID, resource.Title, itemType)
		tuiRunner.SetSyncing(resource.ID)
	}

	var syncErr error
	switch resource.Type {
	case notion.ResourceTypePage:
		syncErr = syncPage(ctx, client, w, logger, resource, name, dryRun, state, tuiRunner, attDownloader)

	case notion.ResourceTypeDatabase:
		syncErr = syncDatabase(ctx, client, w, logger, resource, name, dryRun, state, tuiRunner, attDownloader)
	}

	// Update TUI status
	if tuiRunner != nil {
		if syncErr != nil {
			tuiRunner.SetError(resource.ID, syncErr.Error())
		} else {
			tuiRunner.SetDone(resource.ID)
		}
	}

	return syncErr
}

// syncPage syncs a standalone page to the vault.
func syncPage(ctx context.Context, client *notion.Client, w *writer.Writer, logger *slog.Logger, resource *notion.Resource, folderName string, dryRun bool, state *sync.SyncState, tuiRunner *tui.Runner, attDownloader *transform.AttachmentDownloader) error {
	// Fetch page metadata to get LastEditedTime
	page, err := client.GetPage(ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("fetching page: %w", err)
	}

	lastModified := page.LastEditedTime

	// Check if sync is needed
	if !state.NeedsSync(resource.ID, lastModified) {
		logger.Info("page unchanged, skipping", "title", resource.Title)
		if tuiRunner != nil {
			tuiRunner.SetDone(resource.ID)
		}
		return nil
	}

	// Fetch blocks
	blocks, err := client.GetBlockChildren(ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("fetching page blocks: %w", err)
	}

	filename := sanitizeFilename(resource.Title) + ".md"

	if dryRun {
		logger.Info("would sync page", "title", resource.Title, "blocks", len(blocks))
		return nil
	}

	// Transform blocks to markdown (with attachment downloading if enabled)
	var transformer *transform.Transformer
	if attDownloader != nil {
		transformer = transform.NewTransformerWithAttachments(ctx, client, attDownloader)
	} else {
		transformer = transform.NewTransformer(ctx, client)
	}

	markdown, err := transformer.BlocksToMarkdown(blocks)
	if err != nil {
		return fmt.Errorf("transforming blocks: %w", err)
	}

	// Write markdown file
	if err := w.WriteMarkdown("", filename, markdown); err != nil {
		return fmt.Errorf("writing markdown: %w", err)
	}

	// Update state
	state.SetResource(sync.ResourceState{
		ID:           resource.ID,
		Type:         sync.ResourceTypePage,
		Title:        resource.Title,
		LastModified: lastModified,
		LocalPath:    filename,
	})

	logger.Info("synced page", "title", resource.Title, "file", filename)
	return nil
}

// syncDatabase syncs a database and all its entries to the vault.
func syncDatabase(ctx context.Context, client *notion.Client, w *writer.Writer, logger *slog.Logger, resource *notion.Resource, folderName string, dryRun bool, state *sync.SyncState, tuiRunner *tui.Runner, attDownloader *transform.AttachmentDownloader) error {
	// Fetch database schema
	db, err := client.GetDatabase(ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("fetching database: %w", err)
	}

	schema, err := transform.ParseDatabaseSchema(db)
	if err != nil {
		return fmt.Errorf("parsing database schema: %w", err)
	}

	// Determine folder path for entries
	folder := sanitizeFilename(resource.Title)
	if folderName != "" && !isUUIDPrefix(folderName) {
		folder = sanitizeFilename(folderName)
	}

	// Query database entries
	pages, err := client.QueryDatabase(ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("querying database: %w", err)
	}

	// Initialize or get database state
	dbState := state.GetResource(resource.ID)
	if dbState == nil {
		state.SetResource(sync.ResourceState{
			ID:           resource.ID,
			Type:         sync.ResourceTypeDatabase,
			Title:        resource.Title,
			LastModified: db.LastEditedTime,
			LocalPath:    folder,
			Entries:      make(map[string]sync.EntryState),
		})
		dbState = state.GetResource(resource.ID)
	}

	// Add child entries to TUI
	if tuiRunner != nil {
		for _, page := range pages {
			title := extractPageTitle(page)
			if title == "" {
				title = string(page.ID)[:8] + "..."
			}
			tuiRunner.AddChild(resource.ID, string(page.ID), title, tui.TypePage)
		}
	}

	if dryRun {
		logger.Info("would sync database", "title", resource.Title, "folder", folder, "entries", len(pages))
		for i, page := range pages {
			if i >= 10 {
				logger.Info("... and more", "remaining", len(pages)-10)
				break
			}
			logger.Info("  entry", "title", extractPageTitle(page))
		}
		return nil
	}

	// Generate and write .base file
	baseFile, err := transform.GenerateBaseFile(schema, folder)
	if err != nil {
		return fmt.Errorf("generating base file: %w", err)
	}
	baseContent, err := transform.MarshalBaseFile(baseFile)
	if err != nil {
		return fmt.Errorf("marshaling base file: %w", err)
	}
	if err := w.WriteBase("", resource.Title, baseContent); err != nil {
		return fmt.Errorf("writing base file: %w", err)
	}

	// Ensure folder exists for entries
	if err := w.EnsureFolder(folder); err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}

	// Create transformer (with attachment downloading if enabled)
	var transformer *transform.Transformer
	if attDownloader != nil {
		transformer = transform.NewTransformerWithAttachments(ctx, client, attDownloader)
	} else {
		transformer = transform.NewTransformer(ctx, client)
	}

	// Process each entry
	syncedCount := 0
	skippedCount := 0

	for _, page := range pages {
		pageID := string(page.ID)
		lastModified := page.LastEditedTime

		// Check if entry needs sync
		if !state.NeedsEntrySync(resource.ID, pageID, lastModified) {
			logger.Debug("entry unchanged, skipping", "id", pageID)
			skippedCount++
			if tuiRunner != nil {
				tuiRunner.SetDone(pageID)
			}
			continue
		}

		// Update TUI
		if tuiRunner != nil {
			tuiRunner.SetSyncing(pageID)
		}

		filename, err := syncDatabaseEntry(ctx, client, w, transformer, logger, &page, schema, folder)
		if err != nil {
			logger.Error("failed to sync entry", "id", page.ID, "error", err)
			if tuiRunner != nil {
				tuiRunner.SetError(pageID, err.Error())
			}
			// Continue with other entries
		} else {
			syncedCount++
			// Update entry state
			_ = state.SetEntry(resource.ID, sync.EntryState{
				PageID:       pageID,
				Title:        extractPageTitle(page),
				LastModified: lastModified,
				LocalFile:    filename,
			})
			if tuiRunner != nil {
				tuiRunner.SetDone(pageID)
			}
		}
	}

	// Update database state with latest timestamp
	state.SetResource(sync.ResourceState{
		ID:           resource.ID,
		Type:         sync.ResourceTypeDatabase,
		Title:        resource.Title,
		LastModified: db.LastEditedTime,
		LocalPath:    folder,
		Entries:      dbState.Entries,
	})

	logger.Info("synced database",
		"title", resource.Title,
		"folder", folder,
		"synced", syncedCount,
		"skipped", skippedCount,
		"total", len(pages),
	)
	return nil
}

// syncDatabaseEntry syncs a single database entry.
// Returns the filename written and any error.
func syncDatabaseEntry(ctx context.Context, client *notion.Client, w *writer.Writer, transformer *transform.Transformer, logger *slog.Logger, page *notionapi.Page, schema *transform.DatabaseSchema, folder string) (string, error) {
	// Extract entry data for frontmatter
	entry, err := transform.ExtractEntryData(page, schema)
	if err != nil {
		return "", fmt.Errorf("extracting entry data: %w", err)
	}

	// Fetch page content blocks
	blocks, err := client.GetBlockChildren(ctx, string(page.ID))
	if err != nil {
		return "", fmt.Errorf("fetching entry blocks: %w", err)
	}

	// Transform blocks to markdown
	markdown, err := transformer.BlocksToMarkdown(blocks)
	if err != nil {
		return "", fmt.Errorf("transforming blocks: %w", err)
	}

	// Build complete entry with frontmatter
	dbEntry, err := transform.BuildDatabaseEntry(entry, markdown)
	if err != nil {
		return "", fmt.Errorf("building entry: %w", err)
	}

	// Write the file
	content := dbEntry.Frontmatter + "\n" + dbEntry.Content
	if err := w.WriteMarkdown(folder, dbEntry.Filename, content); err != nil {
		return "", fmt.Errorf("writing entry: %w", err)
	}

	logger.Debug("synced entry", "title", entry.Title, "file", dbEntry.Filename)
	return dbEntry.Filename, nil
}

// isUUIDPrefix checks if the name looks like a truncated UUID (e.g., "1e567c00...").
func isUUIDPrefix(name string) bool {
	if len(name) < 8 {
		return false
	}
	// Check if first 8 chars are hex
	for _, c := range name[:8] {
		isDigit := c >= '0' && c <= '9'
		isHexLower := c >= 'a' && c <= 'f'
		if !isDigit && !isHexLower {
			return false
		}
	}
	return true
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

// extractPageTitle extracts the title from a page's properties.
func extractPageTitle(page notionapi.Page) string {
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
