// Package main provides the CLI entry point for notion-sync.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information (set at build time)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "notion-sync",
	Short: "Sync Notion pages and databases to Obsidian",
	Long: `notion-sync is a one-way sync tool from Notion to Obsidian.

It fetches pages and databases from Notion, converts them to
Obsidian-flavored markdown, and writes them to your vault.

Notion is the source of truth. Changes made in Obsidian will be
overwritten on the next sync.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("notion-sync %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(syncCmd)
}
