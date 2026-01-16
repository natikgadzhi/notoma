package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/Test-Page-abc123def456abc123def456abc123de"
      name: "Test Page"
output:
  vault_path: "/data/vault"
  attachment_folder: "_attachments"
state:
  file: "/data/state.json"
options:
  download_attachments: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	// Set NOTION_TOKEN env var
	t.Setenv("NOTION_TOKEN", "test-token-123")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify config values
	if len(cfg.Sync.Roots) != 1 {
		t.Errorf("expected 1 root, got %d", len(cfg.Sync.Roots))
	}

	if cfg.Sync.Roots[0].Name != "Test Page" {
		t.Errorf("expected root name 'Test Page', got %q", cfg.Sync.Roots[0].Name)
	}

	if cfg.Output.VaultPath != "/data/vault" {
		t.Errorf("expected vault_path '/data/vault', got %q", cfg.Output.VaultPath)
	}

	if cfg.Output.AttachmentFolder != "_attachments" {
		t.Errorf("expected attachment_folder '_attachments', got %q", cfg.Output.AttachmentFolder)
	}

	if cfg.State.File != "/data/state.json" {
		t.Errorf("expected state file '/data/state.json', got %q", cfg.State.File)
	}

	if !cfg.Options.ShouldDownloadAttachments() {
		t.Error("expected download_attachments to be true")
	}

	if cfg.NotionToken != "test-token-123" {
		t.Errorf("expected NotionToken 'test-token-123', got %q", cfg.NotionToken)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	// Ensure NOTION_TOKEN is not set
	t.Setenv("NOTION_TOKEN", "")

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for missing NOTION_TOKEN")
	}
}

func TestLoad_MissingRoots(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  roots: []
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for empty roots without discover_workspace_roots")
	}
}

func TestLoad_DiscoverWorkspaceRoots(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  discover_workspace_roots: true
output:
  vault_path: "/data/vault"
  attachment_folder: "_attachments"
state:
  file: "/data/state.json"
options:
  download_attachments: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token-123")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Sync.DiscoverWorkspaceRoots {
		t.Error("expected discover_workspace_roots to be true")
	}

	if len(cfg.Sync.Roots) != 0 {
		t.Errorf("expected 0 roots, got %d", len(cfg.Sync.Roots))
	}
}

func TestLoad_CombinedRootsAndDiscovery(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Both explicit roots and discover_workspace_roots can be used together
	configContent := `
sync:
  discover_workspace_roots: true
  roots:
    - url: "https://www.notion.so/workspace/Test-Page-abc123def456abc123def456abc123de"
      name: "Explicit Root"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Sync.DiscoverWorkspaceRoots {
		t.Error("expected discover_workspace_roots to be true")
	}

	if len(cfg.Sync.Roots) != 1 {
		t.Errorf("expected 1 explicit root, got %d", len(cfg.Sync.Roots))
	}
}

func TestLoad_MissingVaultPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
output:
  attachment_folder: "_attachments"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for missing vault_path")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Setenv("NOTION_TOKEN", "test-token")

	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestOptions_ShouldDownloadAttachments_DefaultsToTrue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Config without download_attachments specified
	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
  attachment_folder: "_attachments"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should default to true when not specified
	if !cfg.Options.ShouldDownloadAttachments() {
		t.Error("expected ShouldDownloadAttachments() to default to true when not specified")
	}

	// The underlying pointer should be nil
	if cfg.Options.DownloadAttachments != nil {
		t.Error("expected DownloadAttachments to be nil when not specified")
	}
}

func TestOptions_ShouldDownloadAttachments_ExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Config with download_attachments explicitly set to false
	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
options:
  download_attachments: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should be false when explicitly set to false
	if cfg.Options.ShouldDownloadAttachments() {
		t.Error("expected ShouldDownloadAttachments() to be false when explicitly set to false")
	}
}

func TestDatesConfig_Defaults(t *testing.T) {
	// Test nil DatesConfig uses defaults
	var cfg *DatesConfig = nil

	if !cfg.ShouldTransformEmptyDatetimeToDate() {
		t.Error("expected ShouldTransformEmptyDatetimeToDate to default to true")
	}
	if cfg.ShouldLinkDailyNotes() {
		t.Error("expected ShouldLinkDailyNotes to default to false")
	}
	if cfg.GetDailyNotePathPrefix() != "" {
		t.Errorf("expected GetDailyNotePathPrefix to default to empty, got %q", cfg.GetDailyNotePathPrefix())
	}
	if cfg.GetDateFormat() != "02-01-2006" {
		t.Errorf("expected GetDateFormat to default to '02-01-2006', got %q", cfg.GetDateFormat())
	}
}

func TestDatesConfig_CustomValues(t *testing.T) {
	trueVal := true
	falseVal := false

	cfg := &DatesConfig{
		TransformEmptyDatetimeToDate: &falseVal,
		LinkDailyNotes:               &trueVal,
		DailyNotePathPrefix:          "Days/",
		DateFormat:                   "2006-01-02",
	}

	if cfg.ShouldTransformEmptyDatetimeToDate() {
		t.Error("expected ShouldTransformEmptyDatetimeToDate to be false")
	}
	if !cfg.ShouldLinkDailyNotes() {
		t.Error("expected ShouldLinkDailyNotes to be true")
	}
	if cfg.GetDailyNotePathPrefix() != "Days/" {
		t.Errorf("expected GetDailyNotePathPrefix to be 'Days/', got %q", cfg.GetDailyNotePathPrefix())
	}
	if cfg.GetDateFormat() != "2006-01-02" {
		t.Errorf("expected GetDateFormat to be '2006-01-02', got %q", cfg.GetDateFormat())
	}
}

func TestLoad_WithDatesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/Test-Page-abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
options:
  dates:
    transform_empty_datetime_to_date: false
    date_format: "2006-01-02"
    link_daily_notes: true
    daily_note_path_prefix: "Journal/"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token-123")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	dates := cfg.Options.GetDatesConfig()
	if dates == nil {
		t.Fatal("expected dates config to be present")
	}

	if dates.ShouldTransformEmptyDatetimeToDate() {
		t.Error("expected ShouldTransformEmptyDatetimeToDate to be false")
	}
	if !dates.ShouldLinkDailyNotes() {
		t.Error("expected ShouldLinkDailyNotes to be true")
	}
	if dates.GetDailyNotePathPrefix() != "Journal/" {
		t.Errorf("expected prefix 'Journal/', got %q", dates.GetDailyNotePathPrefix())
	}
	if dates.GetDateFormat() != "2006-01-02" {
		t.Errorf("expected format '2006-01-02', got %q", dates.GetDateFormat())
	}
}

func TestLoad_WithoutDatesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/Test-Page-abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token-123")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	dates := cfg.Options.GetDatesConfig()
	if dates != nil {
		t.Error("expected dates config to be nil when not specified")
	}

	// The Options methods should still return defaults via the DatesConfig methods
	// which handle nil receivers
}

func TestOptions_ShouldUpdateNotionTimestamp_DefaultsToFalse(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Config without update_notion_timestamp specified
	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should default to false when not specified
	if cfg.Options.ShouldUpdateNotionTimestamp() {
		t.Error("expected ShouldUpdateNotionTimestamp() to default to false when not specified")
	}

	// The underlying pointer should be nil
	if cfg.Options.UpdateNotionTimestamp != nil {
		t.Error("expected UpdateNotionTimestamp to be nil when not specified")
	}
}

func TestOptions_ShouldUpdateNotionTimestamp_ExplicitTrue(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Config with update_notion_timestamp explicitly set to true
	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
output:
  vault_path: "/data/vault"
state:
  file: "/data/state.json"
options:
  update_notion_timestamp: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should be true when explicitly set to true
	if !cfg.Options.ShouldUpdateNotionTimestamp() {
		t.Error("expected ShouldUpdateNotionTimestamp() to be true when explicitly set to true")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with roots",
			config: Config{
				Sync: SyncConfig{
					Roots: []Root{{URL: "https://notion.so/test"}},
				},
				Output: OutputConfig{
					VaultPath: "/data/vault",
				},
				State: StateConfig{
					File: "/data/state.json",
				},
				NotionToken: "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with discover_workspace_roots",
			config: Config{
				Sync: SyncConfig{
					DiscoverWorkspaceRoots: true,
				},
				Output: OutputConfig{
					VaultPath: "/data/vault",
				},
				State: StateConfig{
					File: "/data/state.json",
				},
				NotionToken: "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with both roots and discover_workspace_roots",
			config: Config{
				Sync: SyncConfig{
					Roots:                  []Root{{URL: "https://notion.so/test"}},
					DiscoverWorkspaceRoots: true,
				},
				Output: OutputConfig{
					VaultPath: "/data/vault",
				},
				State: StateConfig{
					File: "/data/state.json",
				},
				NotionToken: "test-token",
			},
			wantErr: false,
		},
		{
			name: "missing roots and discover_workspace_roots",
			config: Config{
				Sync: SyncConfig{
					Roots: []Root{},
				},
				Output: OutputConfig{
					VaultPath: "/data/vault",
				},
				State: StateConfig{
					File: "/data/state.json",
				},
				NotionToken: "test-token",
			},
			wantErr: true,
		},
		{
			name: "empty root URL",
			config: Config{
				Sync: SyncConfig{
					Roots: []Root{{URL: "", Name: "Test"}},
				},
				Output: OutputConfig{
					VaultPath: "/data/vault",
				},
				State: StateConfig{
					File: "/data/state.json",
				},
				NotionToken: "test-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
