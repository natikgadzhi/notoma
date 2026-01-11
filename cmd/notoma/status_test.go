package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/sync"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"one minute", 1 * time.Minute, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"one hour", 1 * time.Hour, "1h"},
		{"hours", 3 * time.Hour, "3h"},
		{"hours and minutes", 3*time.Hour + 30*time.Minute, "3h 30m"},
		{"one day", 24 * time.Hour, "1d"},
		{"days", 3 * 24 * time.Hour, "3d"},
		{"days and hours", 3*24*time.Hour + 5*time.Hour, "3d 5h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestPrintStatus_EmptyState(t *testing.T) {
	var buf bytes.Buffer

	cfg := &config.Config{
		Output: config.OutputConfig{
			VaultPath: "/tmp/vault",
		},
		State: config.StateConfig{
			File: "/tmp/state.json",
		},
	}
	state := sync.NewSyncState()

	// Set the global variable used by printStatus
	oldConfigPath := statusConfigPath
	statusConfigPath = "config.yaml"
	defer func() { statusConfigPath = oldConfigPath }()

	printStatus(&buf, cfg, state)

	output := buf.String()

	// Check for expected sections
	if !strings.Contains(output, "Notoma Sync Status") {
		t.Error("expected header in output")
	}
	if !strings.Contains(output, "Last sync:    Never") {
		t.Error("expected 'Never' for last sync time")
	}
	if !strings.Contains(output, "Total resources:   0") {
		t.Error("expected 0 total resources")
	}
	if !strings.Contains(output, "No resources synced yet") {
		t.Error("expected message about no resources synced")
	}
}

func TestPrintStatus_WithResources(t *testing.T) {
	tmpDir := t.TempDir()
	var buf bytes.Buffer

	cfg := &config.Config{
		Output: config.OutputConfig{
			VaultPath: tmpDir,
		},
		State: config.StateConfig{
			File: filepath.Join(tmpDir, "state.json"),
		},
	}

	state := sync.NewSyncState()
	state.LastSyncTime = time.Now().Add(-2 * time.Hour)
	state.SetResource(sync.ResourceState{
		ID:           "page-1",
		Type:         sync.ResourceTypePage,
		Title:        "Test Page",
		LocalPath:    "Page.md",
		LastModified: time.Now().Add(-3 * time.Hour),
	})
	state.SetResource(sync.ResourceState{
		ID:           "db-1",
		Type:         sync.ResourceTypeDatabase,
		Title:        "Test Database",
		LocalPath:    "Database",
		LastModified: time.Now().Add(-4 * time.Hour),
		Entries: map[string]sync.EntryState{
			"entry-1": {
				PageID:    "entry-1",
				Title:     "Entry One",
				LocalFile: "Entry One.md",
			},
		},
	})

	// Set the global variable used by printStatus
	oldConfigPath := statusConfigPath
	statusConfigPath = "config.yaml"
	defer func() { statusConfigPath = oldConfigPath }()

	printStatus(&buf, cfg, state)

	output := buf.String()

	// Check for expected content
	if !strings.Contains(output, "Total resources:   2") {
		t.Errorf("expected 2 total resources in output:\n%s", output)
	}
	if !strings.Contains(output, "Pages:           1") {
		t.Errorf("expected 1 page in output:\n%s", output)
	}
	if !strings.Contains(output, "Databases:       1") {
		t.Errorf("expected 1 database in output:\n%s", output)
	}
	if !strings.Contains(output, "Database entries:  1") {
		t.Errorf("expected 1 database entry in output:\n%s", output)
	}
	// Should not show "No resources synced" message
	if strings.Contains(output, "No resources synced yet") {
		t.Errorf("should not show 'no resources synced' message when resources exist:\n%s", output)
	}
}

func TestPrintStatus_ShowsLastSyncTime(t *testing.T) {
	var buf bytes.Buffer

	cfg := &config.Config{
		Output: config.OutputConfig{
			VaultPath: "/tmp/vault",
		},
		State: config.StateConfig{
			File: "/tmp/state.json",
		},
	}

	state := sync.NewSyncState()
	state.LastSyncTime = time.Now().Add(-30 * time.Minute)

	oldConfigPath := statusConfigPath
	statusConfigPath = "config.yaml"
	defer func() { statusConfigPath = oldConfigPath }()

	printStatus(&buf, cfg, state)

	output := buf.String()

	// Should show the time ago
	if !strings.Contains(output, "30m ago") {
		t.Errorf("expected '30m ago' in output:\n%s", output)
	}
}
