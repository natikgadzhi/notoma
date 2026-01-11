package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/sync"
	"github.com/spf13/cobra"
)

var statusConfigPath string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync state and tracked resources",
	Long: `Status displays information about the current sync state, including:
- When the last sync occurred
- Number of tracked resources (pages and databases)
- Number of database entries`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVarP(&statusConfigPath, "config", "c", "config.yaml", "path to config file")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(statusConfigPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Load sync state
	state, err := sync.LoadState(cfg.State.File)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Display status
	printStatus(os.Stdout, cfg, state)

	return nil
}

// printStatus outputs the sync state to the given writer.
func printStatus(w io.Writer, cfg *config.Config, state *sync.SyncState) {
	_, _ = fmt.Fprintln(w, "Notoma Sync Status")
	_, _ = fmt.Fprintln(w, "==================")
	_, _ = fmt.Fprintln(w)

	// Config info
	_, _ = fmt.Fprintf(w, "Config file:  %s\n", statusConfigPath)
	_, _ = fmt.Fprintf(w, "State file:   %s\n", cfg.State.File)
	_, _ = fmt.Fprintf(w, "Vault path:   %s\n", cfg.Output.VaultPath)
	_, _ = fmt.Fprintln(w)

	// Last sync time
	if state.LastSyncTime.IsZero() {
		_, _ = fmt.Fprintln(w, "Last sync:    Never")
	} else {
		ago := time.Since(state.LastSyncTime).Round(time.Second)
		_, _ = fmt.Fprintf(w, "Last sync:    %s (%s ago)\n",
			state.LastSyncTime.Format("2006-01-02 15:04:05"),
			formatDuration(ago))
	}
	_, _ = fmt.Fprintln(w)

	// Resource counts
	resourceCount := state.ResourceCount()
	entryCount := state.EntryCount()
	pageCount := 0
	dbCount := 0

	for _, res := range state.Resources {
		switch res.Type {
		case sync.ResourceTypePage:
			pageCount++
		case sync.ResourceTypeDatabase:
			dbCount++
		}
	}

	_, _ = fmt.Fprintln(w, "Summary")
	_, _ = fmt.Fprintln(w, "-------")
	_, _ = fmt.Fprintf(w, "Total resources:   %d\n", resourceCount)
	_, _ = fmt.Fprintf(w, "  Pages:           %d\n", pageCount)
	_, _ = fmt.Fprintf(w, "  Databases:       %d\n", dbCount)
	_, _ = fmt.Fprintf(w, "Database entries:  %d\n", entryCount)

	if resourceCount == 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "No resources synced yet. Run 'notoma sync' to sync content.")
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}
