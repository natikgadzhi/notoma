package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	state := NewState("/tmp/test.json")

	if state.Version != stateVersion {
		t.Errorf("expected version %d, got %d", stateVersion, state.Version)
	}
	if len(state.Pages) != 0 {
		t.Errorf("expected empty pages map, got %d entries", len(state.Pages))
	}
	if len(state.Databases) != 0 {
		t.Errorf("expected empty databases map, got %d entries", len(state.Databases))
	}
	if len(state.Attachments) != 0 {
		t.Errorf("expected empty attachments map, got %d entries", len(state.Attachments))
	}
	if state.Path() != "/tmp/test.json" {
		t.Errorf("expected path /tmp/test.json, got %s", state.Path())
	}
}

func TestLoadState_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Version != stateVersion {
		t.Errorf("expected version %d, got %d", stateVersion, state.Version)
	}
}

func TestState_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create and populate state
	state := NewState(statePath)
	state.SetPage(&PageState{
		ID:          "page-1",
		Title:       "Test Page",
		LastEdited:  time.Now(),
		ContentHash: "abc123",
		OutputPath:  "test.md",
		SyncedAt:    time.Now(),
	})
	state.SetDatabase(&DatabaseState{
		ID:           "db-1",
		Title:        "Test Database",
		LastEdited:   time.Now(),
		OutputFolder: "database",
		SyncedAt:     time.Now(),
		EntryCount:   5,
	})
	state.SetAttachment(&AttachmentState{
		URLHash:      "hash123",
		OriginalURL:  "https://example.com/image.png",
		LocalPath:    "_attachments/image.png",
		DownloadedAt: time.Now(),
		Size:         1024,
	})
	state.MarkSynced()

	// Save
	if err := state.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load into new state
	loaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Verify
	if loaded.Version != stateVersion {
		t.Errorf("version mismatch: expected %d, got %d", stateVersion, loaded.Version)
	}
	if len(loaded.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(loaded.Pages))
	}
	if len(loaded.Databases) != 1 {
		t.Errorf("expected 1 database, got %d", len(loaded.Databases))
	}
	if len(loaded.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(loaded.Attachments))
	}

	page := loaded.GetPage("page-1")
	if page == nil {
		t.Fatal("expected to find page")
	}
	if page.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", page.Title)
	}
	if page.ContentHash != "abc123" {
		t.Errorf("expected hash 'abc123', got %q", page.ContentHash)
	}

	db := loaded.GetDatabase("db-1")
	if db == nil {
		t.Fatal("expected to find database")
	}
	if db.Title != "Test Database" {
		t.Errorf("expected title 'Test Database', got %q", db.Title)
	}
	if db.EntryCount != 5 {
		t.Errorf("expected 5 entries, got %d", db.EntryCount)
	}

	att := loaded.GetAttachment("hash123")
	if att == nil {
		t.Fatal("expected to find attachment")
	}
	if att.OriginalURL != "https://example.com/image.png" {
		t.Errorf("expected URL 'https://example.com/image.png', got %q", att.OriginalURL)
	}
}

func TestState_NeedsSync(t *testing.T) {
	state := NewState("")

	// New page always needs sync
	if !state.NeedsSync("page-1", time.Now()) {
		t.Error("new page should need sync")
	}

	// Add page with old timestamp
	oldTime := time.Now().Add(-1 * time.Hour)
	state.SetPage(&PageState{
		ID:         "page-1",
		LastEdited: oldTime,
	})

	// Same timestamp - no sync needed
	if state.NeedsSync("page-1", oldTime) {
		t.Error("same timestamp should not need sync")
	}

	// Older timestamp - no sync needed
	if state.NeedsSync("page-1", oldTime.Add(-1*time.Minute)) {
		t.Error("older timestamp should not need sync")
	}

	// Newer timestamp - needs sync
	if !state.NeedsSync("page-1", time.Now()) {
		t.Error("newer timestamp should need sync")
	}
}

func TestState_NeedsDatabaseSync(t *testing.T) {
	state := NewState("")

	// New database always needs sync
	if !state.NeedsDatabaseSync("db-1", time.Now()) {
		t.Error("new database should need sync")
	}

	// Add database with old timestamp
	oldTime := time.Now().Add(-1 * time.Hour)
	state.SetDatabase(&DatabaseState{
		ID:         "db-1",
		LastEdited: oldTime,
	})

	// Same timestamp - no sync needed
	if state.NeedsDatabaseSync("db-1", oldTime) {
		t.Error("same timestamp should not need sync")
	}

	// Newer timestamp - needs sync
	if !state.NeedsDatabaseSync("db-1", time.Now()) {
		t.Error("newer timestamp should need sync")
	}
}

func TestState_Reset(t *testing.T) {
	state := NewState("")
	state.SetPage(&PageState{ID: "page-1"})
	state.SetDatabase(&DatabaseState{ID: "db-1"})
	state.SetAttachment(&AttachmentState{URLHash: "hash1"})
	state.MarkSynced()

	state.Reset()

	if len(state.Pages) != 0 {
		t.Errorf("expected 0 pages after reset, got %d", len(state.Pages))
	}
	if len(state.Databases) != 0 {
		t.Errorf("expected 0 databases after reset, got %d", len(state.Databases))
	}
	if len(state.Attachments) != 0 {
		t.Errorf("expected 0 attachments after reset, got %d", len(state.Attachments))
	}
	if !state.LastSync.IsZero() {
		t.Error("expected zero LastSync after reset")
	}
}

func TestState_Summary(t *testing.T) {
	state := NewState("")
	state.SetPage(&PageState{ID: "page-1"})
	state.SetPage(&PageState{ID: "page-2"})
	state.SetDatabase(&DatabaseState{ID: "db-1"})
	state.SetAttachment(&AttachmentState{URLHash: "hash1"})
	state.SetAttachment(&AttachmentState{URLHash: "hash2"})
	state.SetAttachment(&AttachmentState{URLHash: "hash3"})
	state.MarkSynced()

	summary := state.Summary()

	if summary.PageCount != 2 {
		t.Errorf("expected 2 pages, got %d", summary.PageCount)
	}
	if summary.DatabaseCount != 1 {
		t.Errorf("expected 1 database, got %d", summary.DatabaseCount)
	}
	if summary.AttachmentCount != 3 {
		t.Errorf("expected 3 attachments, got %d", summary.AttachmentCount)
	}
	if summary.LastSync.IsZero() {
		t.Error("expected non-zero LastSync")
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	if err := os.WriteFile(statePath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadState(statePath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestState_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "subdir", "nested", "state.json")

	state := NewState(statePath)
	err := state.Save()
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state file should exist")
	}
}
