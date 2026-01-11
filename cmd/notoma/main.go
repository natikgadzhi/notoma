// Package main provides the entry point for the notoma CLI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/sync"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dryRun  bool
	force   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "notoma",
		Short: "Notion to Obsidian sync tool",
		Long:  "notoma syncs content from Notion to an Obsidian vault.",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file path")

	// Sync command
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync content from Notion to Obsidian",
		RunE:  runSync,
	}
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing")
	syncCmd.Flags().BoolVar(&force, "force", false, "ignore state, perform full sync")
	rootCmd.AddCommand(syncCmd)

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		RunE:  runStatus,
	}
	rootCmd.AddCommand(statusCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("notoma version 0.1.0")
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSync(_ *cobra.Command, _ []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	syncer, err := sync.NewSyncer(cfg, logger,
		sync.WithDryRun(dryRun),
		sync.WithForce(force),
	)
	if err != nil {
		return fmt.Errorf("creating syncer: %w", err)
	}

	result, err := syncer.Run(context.Background())
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	logger.Info("sync completed",
		"pages_updated", result.PagesUpdated,
		"pages_skipped", result.PagesSkipped,
		"databases_updated", result.DatabasesUpdated,
		"errors", len(result.Errors),
		"duration", result.Duration,
	)

	if len(result.Errors) > 0 {
		return fmt.Errorf("sync completed with %d errors", len(result.Errors))
	}

	return nil
}

func runStatus(_ *cobra.Command, _ []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	state, err := sync.LoadState(cfg.State.File)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	summary := state.Summary()

	logger.Info("sync status",
		"last_sync", summary.LastSync,
		"pages", summary.PageCount,
		"databases", summary.DatabaseCount,
		"attachments", summary.AttachmentCount,
	)

	return nil
}
