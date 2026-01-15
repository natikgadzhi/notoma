package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jomei/notionapi"
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
	dryRun bool
	force  bool
	quiet  bool // quiet disables TUI and shows plain log output
)

// syncContext holds dependencies for sync operations, reducing parameter count.
type syncContext struct {
	ctx           context.Context
	client        *notion.Client
	workerPool    *notion.WorkerPool
	writer        *writer.Writer
	logger        *slog.Logger
	state         *sync.SyncState
	tuiRunner     *tui.Runner
	attDownloader *transform.AttachmentDownloader
	dryRun        bool
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Notion content to Obsidian vault",
	Long: `Sync fetches pages and databases from Notion and converts them
to Obsidian-flavored markdown files in your vault.

By default, it performs incremental sync - only fetching pages
modified since the last sync. Use --force to perform a full resync.

When running in a terminal, a TUI progress display is shown by default.
Use --quiet to disable the TUI and show plain log output instead.
Use --verbose to enable debug logging (shown alongside TUI or in quiet mode).`,
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

	// Set up logging - suppress in TUI mode unless verbose
	var logOutput io.Writer = os.Stderr
	if useTUI && !verbose {
		logOutput = io.Discard
	}
	logger := setupLogger(logOutput, verbose)

	// Set up context with signal handling
	ctx, cancel := setupSignalHandler(logger)
	defer cancel()

	// Load configuration
	logger.Info("loading configuration", "path", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if dryRun {
		logger.Info("dry-run mode enabled, no files will be written")
	}

	// Compute config hash for change detection
	configHash := sync.ComputeConfigHash(sync.ConfigSettings{
		DownloadAttachments: cfg.Options.ShouldDownloadAttachments(),
		AttachmentFolder:    cfg.Output.AttachmentFolder,
	})

	// Load sync state (or create new if --force or doesn't exist)
	var state *sync.SyncState
	if force {
		logger.Info("force mode enabled, ignoring state and performing full resync")
		state = sync.NewSyncState()
		state.UpdateConfigHash(configHash)
	} else {
		state, err = sync.LoadState(cfg.State.File)
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		// Check if config changed since last sync
		if state.CheckConfigChanged(configHash) {
			logger.Info("config changed since last sync, invalidating state for full resync")
			state.InvalidateForConfigChange(configHash)
		} else {
			// Update hash for new state files or unchanged config
			state.UpdateConfigHash(configHash)
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

	// Create worker pool for parallel fetching (5 concurrent workers)
	workerPool := notion.DefaultWorkerPool(client)

	// Note: OnStart callback will be set after TUI runner is created

	// Validate connection by fetching current user
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("validating Notion token: %w", err)
	}
	logger.Info("connected to Notion", "bot", user.Name)

	// Create writer
	w := writer.New(cfg.Output.VaultPath, cfg.Output.AttachmentFolder, dryRun, logger)

	// Create attachment downloader if enabled (defaults to true)
	var attDownloader *transform.AttachmentDownloader
	if cfg.Options.ShouldDownloadAttachments() {
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

		// Set up worker pool callback to mark items as syncing when work actually starts
		workerPool.SetOnStart(func(pageID string) {
			tuiRunner.SetSyncing(pageID)
		})
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

	// Create sync context to pass to sync functions
	sc := &syncContext{
		ctx:           ctx,
		client:        client,
		workerPool:    workerPool,
		writer:        w,
		logger:        logger,
		state:         state,
		tuiRunner:     tuiRunner,
		attDownloader: attDownloader,
		dryRun:        dryRun,
	}

	// Process each root
	var syncErr error
	for _, root := range roots {
		if err := processRoot(sc, root); err != nil {
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

func processRoot(sc *syncContext, root config.Root) error {
	// Parse URL to get ID
	parsed, err := notion.ParseURL(root.URL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	name := root.Name
	if name == "" {
		name = parsed.ID[:8] + "..."
	}
	sc.logger.Info("processing root", "name", name, "id", parsed.ID)

	// Detect resource type
	resource, err := sc.client.DetectResourceType(sc.ctx, parsed.ID)
	if err != nil {
		return fmt.Errorf("detecting resource type: %w", err)
	}

	sc.logger.Info("detected resource",
		"type", resource.Type,
		"title", resource.Title,
		"id", resource.ID,
	)

	// Add to TUI if available (starts as pending, worker pool OnStart marks as syncing)
	if sc.tuiRunner != nil {
		itemType := tui.TypePage
		if resource.Type == notion.ResourceTypeDatabase {
			itemType = tui.TypeDatabase
		}
		sc.tuiRunner.AddRoot(resource.ID, resource.Title, resource.Icon, itemType)
	}

	var syncErr error
	switch resource.Type {
	case notion.ResourceTypePage:
		// Pages are written directly to the vault root as flat md files
		syncErr = syncPage(sc, resource, "")

	case notion.ResourceTypeDatabase:
		syncErr = syncDatabase(sc, resource, name)
	}

	// Update TUI status
	if sc.tuiRunner != nil {
		if syncErr != nil {
			sc.tuiRunner.SetError(resource.ID, syncErr.Error())
		} else {
			sc.tuiRunner.SetDone(resource.ID)
		}
	}

	return syncErr
}

// syncPage syncs a standalone page to the vault.
// folderPath specifies where to write the page (empty for vault root).
func syncPage(sc *syncContext, resource *notion.Resource, folderPath string) error {
	return syncPageRecursive(sc, resource, folderPath, make(map[string]bool))
}

// syncPageRecursive syncs a page and all its child pages recursively.
// visited tracks already-synced page IDs to prevent infinite loops.
func syncPageRecursive(sc *syncContext, resource *notion.Resource, folderPath string, visited map[string]bool) error {
	// Check for cycles
	if visited[resource.ID] {
		sc.logger.Debug("skipping already visited page", "id", resource.ID, "title", resource.Title)
		return nil
	}
	visited[resource.ID] = true

	// Mark as syncing for root-level pages (not going through worker pool)
	if sc.tuiRunner != nil {
		sc.tuiRunner.SetSyncing(resource.ID)
	}

	// Fetch page metadata to get LastEditedTime
	page, err := sc.client.GetPage(sc.ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("fetching page: %w", err)
	}

	lastModified := page.LastEditedTime

	// Check if sync is needed
	if !sc.state.NeedsSync(resource.ID, lastModified) {
		sc.logger.Info("page unchanged, skipping", "title", resource.Title)
		if sc.tuiRunner != nil {
			sc.tuiRunner.SetDone(resource.ID)
		}
		return nil
	}

	// Fetch blocks
	blocks, err := sc.client.GetBlockChildren(sc.ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("fetching page blocks: %w", err)
	}

	filename := sanitizeFilename(resource.Title) + ".md"
	localPath := filename
	if folderPath != "" {
		localPath = folderPath + "/" + filename
	}

	if sc.dryRun {
		sc.logger.Info("would sync page", "title", resource.Title, "blocks", len(blocks), "path", localPath)
	} else {
		// Transform blocks to markdown (with attachment downloading if enabled)
		var transformer *transform.Transformer
		if sc.attDownloader != nil {
			transformer = transform.NewTransformerWithAttachments(sc.ctx, sc.client, sc.attDownloader)
		} else {
			transformer = transform.NewTransformer(sc.ctx, sc.client)
		}

		markdown, err := transformer.BlocksToMarkdown(blocks)
		if err != nil {
			return fmt.Errorf("transforming blocks: %w", err)
		}

		// Write markdown file
		if err := sc.writer.WriteMarkdown(folderPath, filename, markdown); err != nil {
			return fmt.Errorf("writing markdown: %w", err)
		}

		// Update state
		sc.state.SetResource(sync.ResourceState{
			ID:           resource.ID,
			Type:         sync.ResourceTypePage,
			Title:        resource.Title,
			LastModified: lastModified,
			LocalPath:    localPath,
		})

		sc.logger.Info("synced page", "title", resource.Title, "file", localPath)
	}

	// Extract and recursively sync child pages
	childPages := extractChildPages(blocks)
	if len(childPages) > 0 {
		// Child pages go in the same folder as parent (flat structure for Obsidian)
		childFolder := folderPath

		sc.logger.Debug("found child pages", "parent", resource.Title, "count", len(childPages), "folder", childFolder)

		// Build map of child info and list of IDs to fetch
		childMap := make(map[string]childPageInfo)
		var childIDs []string
		for _, child := range childPages {
			// Skip already visited pages
			if visited[child.id] {
				sc.logger.Debug("skipping already visited child", "id", child.id)
				continue
			}
			childMap[child.id] = child
			childIDs = append(childIDs, child.id)

			// Add child to TUI if available (starts as pending, worker pool OnStart marks as syncing)
			// Note: child pages from blocks don't have icon data, pass empty for default
			if sc.tuiRunner != nil {
				sc.tuiRunner.AddChild(resource.ID, child.id, child.title, "", tui.TypePage)
			}
		}

		// Fetch child page data (metadata + blocks) in parallel
		if len(childIDs) > 0 {
			results := sc.workerPool.FetchPagesWithBlocksParallel(sc.ctx, childIDs)

			// Process results and collect discovered grandchildren for recursive processing
			type pendingChild struct {
				resource    *notion.Resource
				blocks      []notionapi.Block
				lastModTime time.Time
			}
			var pending []pendingChild

			for result := range results {
				child := childMap[result.PageID]

				if result.Err != nil {
					sc.logger.Error("failed to fetch child page", "id", result.PageID, "error", result.Err)
					if sc.tuiRunner != nil {
						sc.tuiRunner.SetError(result.PageID, result.Err.Error())
					}
					continue
				}

				// Mark as visited
				visited[result.PageID] = true

				pending = append(pending, pendingChild{
					resource: &notion.Resource{
						ID:    result.PageID,
						Type:  notion.ResourceTypePage,
						Title: child.title,
					},
					blocks:      result.Blocks,
					lastModTime: result.Page.LastEditedTime,
				})
			}

			// Process pending children (write files and recurse into grandchildren)
			for _, p := range pending {
				if err := processChildPageWithBlocks(sc, p.resource, p.blocks, p.lastModTime, childFolder, visited); err != nil {
					sc.logger.Error("failed to sync child page", "parent", resource.Title, "child", p.resource.Title, "error", err)
					if sc.tuiRunner != nil {
						sc.tuiRunner.SetError(p.resource.ID, err.Error())
					}
				} else if sc.tuiRunner != nil {
					sc.tuiRunner.SetDone(p.resource.ID)
				}
			}
		}
	}

	return nil
}

// processChildPageWithBlocks processes a child page with pre-fetched blocks.
// It writes the markdown file and recursively processes grandchildren.
func processChildPageWithBlocks(sc *syncContext, resource *notion.Resource, blocks []notionapi.Block, lastModified time.Time, folderPath string, visited map[string]bool) error {
	// Check if sync is needed
	if !sc.state.NeedsSync(resource.ID, lastModified) {
		sc.logger.Info("child page unchanged, skipping", "title", resource.Title)
		return nil
	}

	filename := sanitizeFilename(resource.Title) + ".md"
	localPath := filename
	if folderPath != "" {
		localPath = folderPath + "/" + filename
	}

	if sc.dryRun {
		sc.logger.Info("would sync child page", "title", resource.Title, "blocks", len(blocks), "path", localPath)
	} else {
		// Transform blocks to markdown (with attachment downloading if enabled)
		var transformer *transform.Transformer
		if sc.attDownloader != nil {
			transformer = transform.NewTransformerWithAttachments(sc.ctx, sc.client, sc.attDownloader)
		} else {
			transformer = transform.NewTransformer(sc.ctx, sc.client)
		}

		markdown, err := transformer.BlocksToMarkdown(blocks)
		if err != nil {
			return fmt.Errorf("transforming blocks: %w", err)
		}

		// Write markdown file
		if err := sc.writer.WriteMarkdown(folderPath, filename, markdown); err != nil {
			return fmt.Errorf("writing markdown: %w", err)
		}

		// Update state
		sc.state.SetResource(sync.ResourceState{
			ID:           resource.ID,
			Type:         sync.ResourceTypePage,
			Title:        resource.Title,
			LastModified: lastModified,
			LocalPath:    localPath,
		})

		sc.logger.Info("synced child page", "title", resource.Title, "file", localPath)
	}

	// Extract and recursively sync grandchildren
	childPages := extractChildPages(blocks)
	if len(childPages) > 0 {
		// Grandchildren go in the same folder as parent (flat structure for Obsidian)
		childFolder := folderPath

		// Recursively sync grandchildren using the same parallel approach
		// Build map of grandchild info and list of IDs to fetch
		childMap := make(map[string]childPageInfo)
		var childIDs []string
		for _, child := range childPages {
			// Skip already visited pages
			if visited[child.id] {
				sc.logger.Debug("skipping already visited grandchild", "id", child.id)
				continue
			}
			childMap[child.id] = child
			childIDs = append(childIDs, child.id)

			// Add to TUI (starts as pending, worker pool OnStart marks as syncing)
			if sc.tuiRunner != nil {
				sc.tuiRunner.AddChild(resource.ID, child.id, child.title, "", tui.TypePage)
			}
		}

		// Fetch grandchild data in parallel
		if len(childIDs) > 0 {
			results := sc.workerPool.FetchPagesWithBlocksParallel(sc.ctx, childIDs)

			for result := range results {
				child := childMap[result.PageID]

				if result.Err != nil {
					sc.logger.Error("failed to fetch grandchild page", "id", result.PageID, "error", result.Err)
					if sc.tuiRunner != nil {
						sc.tuiRunner.SetError(result.PageID, result.Err.Error())
					}
					continue
				}

				// Mark as visited
				visited[result.PageID] = true

				grandchildResource := &notion.Resource{
					ID:    result.PageID,
					Type:  notion.ResourceTypePage,
					Title: child.title,
				}

				// Recursively process grandchild
				if err := processChildPageWithBlocks(sc, grandchildResource, result.Blocks, result.Page.LastEditedTime, childFolder, visited); err != nil {
					sc.logger.Error("failed to sync grandchild page", "parent", resource.Title, "child", child.title, "error", err)
					if sc.tuiRunner != nil {
						sc.tuiRunner.SetError(result.PageID, err.Error())
					}
				} else if sc.tuiRunner != nil {
					sc.tuiRunner.SetDone(result.PageID)
				}
			}
		}
	}

	return nil
}

// childPageInfo holds basic info about a child page found in blocks.
type childPageInfo struct {
	id    string
	title string
}

// extractChildPages scans blocks for ChildPageBlock items and returns their IDs and titles.
func extractChildPages(blocks []notionapi.Block) []childPageInfo {
	var children []childPageInfo
	for _, block := range blocks {
		if cpb, ok := block.(*notionapi.ChildPageBlock); ok {
			children = append(children, childPageInfo{
				id:    string(cpb.ID),
				title: cpb.ChildPage.Title,
			})
		}
	}
	return children
}

// syncDatabase syncs a database and all its entries to the vault.
func syncDatabase(sc *syncContext, resource *notion.Resource, folderName string) error {
	// Mark root database as syncing
	if sc.tuiRunner != nil {
		sc.tuiRunner.SetSyncing(resource.ID)
	}

	// Fetch database schema
	db, err := sc.client.GetDatabase(sc.ctx, resource.ID)
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
	pages, err := sc.client.QueryDatabase(sc.ctx, resource.ID)
	if err != nil {
		return fmt.Errorf("querying database: %w", err)
	}

	// Initialize or get database state
	dbState := sc.state.GetResource(resource.ID)
	if dbState == nil {
		sc.state.SetResource(sync.ResourceState{
			ID:           resource.ID,
			Type:         sync.ResourceTypeDatabase,
			Title:        resource.Title,
			LastModified: db.LastEditedTime,
			LocalPath:    folder,
			Entries:      make(map[string]sync.EntryState),
		})
		dbState = sc.state.GetResource(resource.ID)
	}

	// Add child entries to TUI
	if sc.tuiRunner != nil {
		for _, page := range pages {
			title := notion.ExtractPageTitle(&page)
			if title == "" {
				title = string(page.ID)[:8] + "..."
			}
			icon := notion.ExtractPageIcon(&page)
			sc.tuiRunner.AddChild(resource.ID, string(page.ID), title, icon, tui.TypePage)
		}
	}

	if sc.dryRun {
		sc.logger.Info("would sync database", "title", resource.Title, "folder", folder, "entries", len(pages))
		for i, page := range pages {
			if i >= 10 {
				sc.logger.Info("... and more", "remaining", len(pages)-10)
				break
			}
			sc.logger.Info("  entry", "title", notion.ExtractPageTitle(&page))
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
	if err := sc.writer.WriteBase("", resource.Title, baseContent); err != nil {
		return fmt.Errorf("writing base file: %w", err)
	}

	// Ensure folder exists for entries
	if err := sc.writer.EnsureFolder(folder); err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}

	// Create transformer (with attachment downloading if enabled)
	var transformer *transform.Transformer
	if sc.attDownloader != nil {
		transformer = transform.NewTransformerWithAttachments(sc.ctx, sc.client, sc.attDownloader)
	} else {
		transformer = transform.NewTransformer(sc.ctx, sc.client)
	}

	// Filter pages that need syncing and build lookup map
	var pagesToSync []string
	pageMap := make(map[string]*notionapi.Page)
	syncedCount := 0
	skippedCount := 0

	for i := range pages {
		page := &pages[i]
		pageID := string(page.ID)
		pageMap[pageID] = page

		// Check if entry needs sync
		if !sc.state.NeedsEntrySync(resource.ID, pageID, page.LastEditedTime) {
			sc.logger.Debug("entry unchanged, skipping", "id", pageID)
			skippedCount++
			if sc.tuiRunner != nil {
				sc.tuiRunner.SetDone(pageID)
			}
			continue
		}

		pagesToSync = append(pagesToSync, pageID)
		// Note: Item stays as pending until worker pool OnStart callback marks it as syncing
	}

	sc.logger.Info("fetching blocks in parallel",
		"to_sync", len(pagesToSync),
		"skipped", skippedCount,
	)

	// Fetch blocks in parallel using worker pool
	if len(pagesToSync) > 0 {
		results := sc.workerPool.FetchBlocksParallel(sc.ctx, pagesToSync)

		// Process results as they arrive
		for result := range results {
			page := pageMap[result.PageID]
			pageID := result.PageID

			if result.Err != nil {
				sc.logger.Error("failed to fetch blocks", "id", pageID, "error", result.Err)
				if sc.tuiRunner != nil {
					sc.tuiRunner.SetError(pageID, result.Err.Error())
				}
				continue
			}

			// Process the entry with pre-fetched blocks
			filename, err := syncDatabaseEntryWithBlocks(sc, transformer, page, result.Blocks, schema, folder)
			if err != nil {
				sc.logger.Error("failed to sync entry", "id", pageID, "error", err)
				if sc.tuiRunner != nil {
					sc.tuiRunner.SetError(pageID, err.Error())
				}
				continue
			}

			syncedCount++
			// Update entry state
			_ = sc.state.SetEntry(resource.ID, sync.EntryState{
				PageID:       pageID,
				Title:        notion.ExtractPageTitle(page),
				LastModified: page.LastEditedTime,
				LocalFile:    filename,
			})
			if sc.tuiRunner != nil {
				sc.tuiRunner.SetDone(pageID)
			}
		}
	}

	// Update database state with latest timestamp
	sc.state.SetResource(sync.ResourceState{
		ID:           resource.ID,
		Type:         sync.ResourceTypeDatabase,
		Title:        resource.Title,
		LastModified: db.LastEditedTime,
		LocalPath:    folder,
		Entries:      dbState.Entries,
	})

	sc.logger.Info("synced database",
		"title", resource.Title,
		"folder", folder,
		"synced", syncedCount,
		"skipped", skippedCount,
		"total", len(pages),
	)
	return nil
}

// syncDatabaseEntryWithBlocks syncs a single database entry using pre-fetched blocks.
// Returns the filename written and any error.
func syncDatabaseEntryWithBlocks(sc *syncContext, transformer *transform.Transformer, page *notionapi.Page, blocks []notionapi.Block, schema *transform.DatabaseSchema, folder string) (string, error) {
	// Extract entry data for frontmatter
	entry, err := transform.ExtractEntryData(page, schema)
	if err != nil {
		return "", fmt.Errorf("extracting entry data: %w", err)
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
	if err := sc.writer.WriteMarkdown(folder, dbEntry.Filename, content); err != nil {
		return "", fmt.Errorf("writing entry: %w", err)
	}

	sc.logger.Debug("synced entry", "title", entry.Title, "file", dbEntry.Filename)
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
