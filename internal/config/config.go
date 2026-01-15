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

// Options contains optional sync behavior settings.
type Options struct {
	// DownloadAttachments controls whether to download Notion-hosted attachments.
	// Defaults to true if not specified.
	DownloadAttachments *bool `yaml:"download_attachments"`
}

// ShouldDownloadAttachments returns whether attachments should be downloaded.
// Defaults to true if not explicitly set.
func (o *Options) ShouldDownloadAttachments() bool {
	if o.DownloadAttachments == nil {
		return true
	}
	return *o.DownloadAttachments
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
