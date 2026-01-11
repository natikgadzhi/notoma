package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jomei/notionapi"
	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/notion"
	"github.com/spf13/cobra"
)

var (
	configPath string
	dryRun     bool
	force      bool
	verbose    bool
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
}

func runSync(cmd *cobra.Command, args []string) error {
	// Set up logging
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
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

	if force {
		logger.Info("force mode enabled, performing full resync")
	}

	// Create Notion client
	client := notion.NewClient(cfg.NotionToken, logger)

	// Validate connection by fetching current user
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("validating Notion token: %w", err)
	}
	logger.Info("connected to Notion", "bot", user.Name)

	// Process each root
	for _, root := range cfg.Sync.Roots {
		if err := processRoot(ctx, client, logger, cfg, root, dryRun); err != nil {
			logger.Error("failed to process root", "url", root.URL, "error", err)
			// Continue with other roots
		}
	}

	logger.Info("sync complete")
	return nil
}

func processRoot(ctx context.Context, client *notion.Client, logger *slog.Logger, cfg *config.Config, root config.Root, dryRun bool) error {
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

	// Fetch content info (for both dry-run preview and actual sync)
	switch resource.Type {
	case notion.ResourceTypePage:
		blocks, err := client.GetBlockChildren(ctx, resource.ID)
		if err != nil {
			return fmt.Errorf("fetching page blocks: %w", err)
		}
		if dryRun {
			logger.Info("would sync page", "title", resource.Title, "blocks", len(blocks))
		} else {
			logger.Info("fetched page blocks", "count", len(blocks))
			// TODO: Implement actual sync logic in Phase 2-4
		}

	case notion.ResourceTypeDatabase:
		pages, err := client.QueryDatabase(ctx, resource.ID)
		if err != nil {
			return fmt.Errorf("querying database: %w", err)
		}
		if dryRun {
			logger.Info("would sync database", "title", resource.Title, "entries", len(pages))
			for i, page := range pages {
				if i >= 10 {
					logger.Info("... and more", "remaining", len(pages)-10)
					break
				}
				logger.Info("  entry", "title", extractPageTitle(page))
			}
		} else {
			logger.Info("fetched database entries", "count", len(pages))
			// TODO: Implement actual sync logic in Phase 2-4
		}
	}

	return nil
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
