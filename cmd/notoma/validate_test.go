package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/natikgadzhi/notion-based/internal/config"
)

func TestCheckVaultPath(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantPassed bool
	}{
		{
			name: "valid writable directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			wantPassed: true,
		},
		{
			name: "nonexistent directory",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/to/vault"
			},
			wantPassed: false,
		},
		{
			name: "path is a file not directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
					t.Fatalf("creating test file: %v", err)
				}
				return filePath
			},
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			passed, _ := checkVaultPath(path)
			if passed != tt.wantPassed {
				t.Errorf("checkVaultPath() passed = %v, want %v", passed, tt.wantPassed)
			}
		})
	}
}

func TestCheckStatePath(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantPassed bool
	}{
		{
			name: "state file in existing directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return filepath.Join(dir, "state.json")
			},
			wantPassed: true,
		},
		{
			name: "state file in current directory",
			setup: func(t *testing.T) string {
				return "state.json"
			},
			wantPassed: true,
		},
		{
			name: "state file in creatable nested directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return filepath.Join(dir, "nested", "subdir", "state.json")
			},
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			passed, _ := checkStatePath(path)
			if passed != tt.wantPassed {
				t.Errorf("checkStatePath() passed = %v, want %v", passed, tt.wantPassed)
			}
		})
	}
}

func TestRootDisplayName(t *testing.T) {
	tests := []struct {
		name string
		root config.Root
		want string
	}{
		{
			name: "root with name",
			root: config.Root{
				URL:  "https://notion.so/abc123",
				Name: "My Page",
			},
			want: "My Page",
		},
		{
			name: "root without name short URL",
			root: config.Root{
				URL: "https://notion.so/abc",
			},
			want: "https://notion.so/abc",
		},
		{
			name: "root without name long URL",
			root: config.Root{
				URL: "https://www.notion.so/workspace/My-Long-Page-Title-abc123def456abc123def456abc123de",
			},
			want: "...3def456abc123def456abc123de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rootDisplayName(tt.root)
			if got != tt.want {
				t.Errorf("rootDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateCmd_MissingConfigFile(t *testing.T) {
	// Test that validate command fails gracefully when config file doesn't exist
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"notoma", "validate", "--config", "/nonexistent/config.yaml"}

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestValidateCmd_InvalidConfig(t *testing.T) {
	// Create a temporary invalid config file (missing required fields)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Invalid config - missing vault_path
	configContent := `
sync:
  roots:
    - url: "https://www.notion.so/workspace/abc123def456abc123def456abc123de"
state:
  file: "state.json"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	t.Setenv("NOTION_TOKEN", "test-token")

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"notoma", "validate", "--config", configPath}

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
