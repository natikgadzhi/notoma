// Package config handles loading and validation of notoma configuration.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Root represents a Notion page or database to sync.
type Root struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name,omitempty"`
}

// OutputConfig specifies where synced content should be written.
type OutputConfig struct {
	VaultPath        string `yaml:"vault_path"`
	AttachmentFolder string `yaml:"attachment_folder"`
}

// StateConfig specifies where sync state is stored.
type StateConfig struct {
	File string `yaml:"file"`
}

// DatesConfig contains settings for date formatting in output.
type DatesConfig struct {
	// TransformEmptyDatetimeToDate controls whether datetimes with midnight time
	// (e.g., 2026-01-02T00:00:00Z) are converted to date-only format.
	// Defaults to true if not specified.
	TransformEmptyDatetimeToDate *bool `yaml:"transform_empty_datetime_to_date"`

	// DateFormat specifies the Go time format string for date-only values.
	// Defaults to "02-01-2006" (DD-MM-YYYY) if not specified.
	// Common formats:
	//   - "02-01-2006" = DD-MM-YYYY (default)
	//   - "2006-01-02" = YYYY-MM-DD (ISO)
	//   - "01/02/2006" = MM/DD/YYYY (US)
	//   - "02/01/2006" = DD/MM/YYYY (EU)
	DateFormat string `yaml:"date_format"`

	// LinkDailyNotes controls whether dates are wrapped in Obsidian links.
	// When true, dates become links like [15-01-2026](Days/15-01-2026.md).
	// Defaults to false if not specified.
	LinkDailyNotes *bool `yaml:"link_daily_notes"`

	// DailyNotePathPrefix is the folder path prefix for daily note links.
	// Example: "Days/" results in links like [15-01-2026](Days/15-01-2026.md).
	// Only used when LinkDailyNotes is true.
	DailyNotePathPrefix string `yaml:"daily_note_path_prefix"`
}

// ShouldTransformEmptyDatetimeToDate returns whether empty datetimes should be
// converted to date-only format. Defaults to true if not explicitly set.
func (d *DatesConfig) ShouldTransformEmptyDatetimeToDate() bool {
	if d == nil || d.TransformEmptyDatetimeToDate == nil {
		return true
	}
	return *d.TransformEmptyDatetimeToDate
}

// GetDateFormat returns the date format string.
// Defaults to "02-01-2006" (DD-MM-YYYY) if not set.
func (d *DatesConfig) GetDateFormat() string {
	if d == nil || d.DateFormat == "" {
		return "02-01-2006" // DD-MM-YYYY default
	}
	return d.DateFormat
}

// ShouldLinkDailyNotes returns whether dates should be wrapped in Obsidian links.
// Defaults to false if not explicitly set.
func (d *DatesConfig) ShouldLinkDailyNotes() bool {
	if d == nil || d.LinkDailyNotes == nil {
		return false
	}
	return *d.LinkDailyNotes
}

// GetDailyNotePathPrefix returns the path prefix for daily note links.
// Returns empty string if not set.
func (d *DatesConfig) GetDailyNotePathPrefix() string {
	if d == nil {
		return ""
	}
	return d.DailyNotePathPrefix
}

// Options contains optional sync behavior settings.
type Options struct {
	// DownloadAttachments controls whether to download Notion-hosted attachments.
	// Defaults to true if not specified.
	DownloadAttachments *bool `yaml:"download_attachments"`

	// Dates contains date formatting configuration.
	Dates *DatesConfig `yaml:"dates"`

	// UpdateNotionTimestamp controls whether to update synced pages in Notion
	// with a last_notoma_sync_at property containing the sync timestamp.
	// This helps users identify stale pages. Defaults to false if not specified.
	// Note: The Notion database/page must have a "last_notoma_sync_at" date property.
	UpdateNotionTimestamp *bool `yaml:"update_notion_timestamp"`
}

// ShouldDownloadAttachments returns whether attachments should be downloaded.
// Defaults to true if not explicitly set.
func (o *Options) ShouldDownloadAttachments() bool {
	if o.DownloadAttachments == nil {
		return true
	}
	return *o.DownloadAttachments
}

// GetDatesConfig returns the dates configuration, or nil if not set.
func (o *Options) GetDatesConfig() *DatesConfig {
	return o.Dates
}

// ShouldUpdateNotionTimestamp returns whether to update synced pages in Notion
// with a sync timestamp. Defaults to false if not explicitly set.
func (o *Options) ShouldUpdateNotionTimestamp() bool {
	if o.UpdateNotionTimestamp == nil {
		return false
	}
	return *o.UpdateNotionTimestamp
}

// SyncConfig contains the list of roots to sync.
type SyncConfig struct {
	// Roots is a list of specific Notion pages/databases to sync.
	// Can be empty if DiscoverWorkspaceRoots is true.
	Roots []Root `yaml:"roots,omitempty"`

	// DiscoverWorkspaceRoots when true, automatically discovers all
	// root-level pages and databases in the workspace (pages/databases
	// whose parent is the workspace itself, not nested under other pages).
	// This uses the Notion Search API to find all items shared with the
	// integration and filters for those at the workspace root level.
	DiscoverWorkspaceRoots bool `yaml:"discover_workspace_roots,omitempty"`
}

// Config is the top-level configuration structure.
type Config struct {
	Sync    SyncConfig   `yaml:"sync"`
	Output  OutputConfig `yaml:"output"`
	State   StateConfig  `yaml:"state"`
	Options Options      `yaml:"options"`

	// NotionToken is loaded from environment, not from config file.
	NotionToken string `yaml:"-"`
}

// Load reads configuration from a YAML file and environment variables.
// NOTION_TOKEN is loaded from environment only (not from config file).
// If a .env file exists in the current directory, it will be loaded first.
func Load(path string) (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Load NOTION_TOKEN from environment
	cfg.NotionToken = os.Getenv("NOTION_TOKEN")

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration has all required fields.
func (c *Config) Validate() error {
	var errs []error

	// Either roots must be specified, or discover_workspace_roots must be true
	if len(c.Sync.Roots) == 0 && !c.Sync.DiscoverWorkspaceRoots {
		errs = append(errs, errors.New("either sync.roots or sync.discover_workspace_roots is required"))
	}

	for i, root := range c.Sync.Roots {
		if root.URL == "" {
			errs = append(errs, fmt.Errorf("root %d: url is required", i+1))
		}
	}

	if c.Output.VaultPath == "" {
		errs = append(errs, errors.New("output.vault_path is required"))
	}

	if c.State.File == "" {
		errs = append(errs, errors.New("state.file is required"))
	}

	if c.NotionToken == "" {
		errs = append(errs, errors.New("NOTION_TOKEN environment variable is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
