package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSyncState(t *testing.T) {
	state := NewSyncState()

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Version != StateVersion {
		t.Errorf("expected version %d, got %d", StateVersion, state.Version)
	}
	if state.Resources == nil {
		t.Error("expected non-nil Resources map")
	}
	if len(state.Resources) != 0 {
		t.Errorf("expected empty Resources, got %d", len(state.Resources))
	}
}

func TestLoadState_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	state, err := LoadState(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if len(state.Resources) != 0 {
		t.Errorf("expected empty state, got %d resources", len(state.Resources))
	}
}

func TestLoadState_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	// Create a valid state file
	original := &SyncState{
		Version:      1,
		LastSyncTime: time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		Resources: map[string]ResourceState{
			"page-123": {
				ID:           "page-123",
				Type:         ResourceTypePage,
				Title:        "Test Page",
				LastModified: time.Date(2025, 1, 10, 11, 0, 0, 0, time.UTC),
				LocalPath:    "Test Page.md",
			},
			"db-456": {
				ID:           "db-456",
				Type:         ResourceTypeDatabase,
				Title:        "My Database",
				LastModified: time.Date(2025, 1, 10, 10, 0, 0, 0, time.UTC),
				LocalPath:    "My Database",
				Entries: map[string]EntryState{
					"entry-1": {
						PageID:       "entry-1",
						Title:        "Entry One",
						LastModified: time.Date(2025, 1, 10, 9, 0, 0, 0, time.UTC),
						LocalFile:    "Entry One.md",
					},
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Load and verify
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("expected version 1, got %d", loaded.Version)
	}
	if len(loaded.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(loaded.Resources))
	}

	page := loaded.Resources["page-123"]
	if page.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", page.Title)
	}
	if page.Type != ResourceTypePage {
		t.Errorf("expected type page, got %s", page.Type)
	}

	db := loaded.Resources["db-456"]
	if len(db.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(db.Entries))
	}
	entry := db.Entries["entry-1"]
	if entry.Title != "Entry One" {
		t.Errorf("expected entry title 'Entry One', got %q", entry.Title)
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := LoadState(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadState_NilMaps(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	// Create a state with nil maps (simulating an older version)
	data := []byte(`{
		"version": 1,
		"last_sync_time": "2025-01-10T12:00:00Z",
		"resources": {
			"db-1": {
				"id": "db-1",
				"type": "database",
				"title": "DB"
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	state, err := LoadState(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify entries map was initialized
	db := state.Resources["db-1"]
	if db.Entries == nil {
		t.Error("expected Entries map to be initialized")
	}
}

func TestSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "state.json")

	state := NewSyncState()
	state.SetResource(ResourceState{
		ID:           "page-123",
		Type:         ResourceTypePage,
		Title:        "Test Page",
		LastModified: time.Now(),
		LocalPath:    "Test Page.md",
	})

	if err := SaveState(path, state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file not created: %v", err)
	}

	// Verify content is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}

	var loaded SyncState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("saved file is not valid JSON: %v", err)
	}

	if len(loaded.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(loaded.Resources))
	}
}

func TestSaveState_NilState(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	err := SaveState(path, nil)
	if err == nil {
		t.Error("expected error for nil state")
	}
}

func TestSaveState_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "state.json")

	// Create initial state
	state1 := NewSyncState()
	state1.SetResource(ResourceState{ID: "v1", Type: ResourceTypePage, Title: "Version 1"})
	if err := SaveState(path, state1); err != nil {
		t.Fatalf("failed to save initial state: %v", err)
	}

	// Save updated state
	state2 := NewSyncState()
	state2.SetResource(ResourceState{ID: "v2", Type: ResourceTypePage, Title: "Version 2"})
	if err := SaveState(path, state2); err != nil {
		t.Fatalf("failed to save updated state: %v", err)
	}

	// Verify no temp file remains
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}

	// Verify final content
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if _, ok := loaded.Resources["v2"]; !ok {
		t.Error("expected v2 resource in final state")
	}
}

func TestGetResource(t *testing.T) {
	state := NewSyncState()
	state.Resources["page-1"] = ResourceState{
		ID:    "page-1",
		Type:  ResourceTypePage,
		Title: "Page One",
	}

	// Existing resource
	res := state.GetResource("page-1")
	if res == nil {
		t.Fatal("expected non-nil resource")
	}
	if res.Title != "Page One" {
		t.Errorf("expected title 'Page One', got %q", res.Title)
	}

	// Non-existent resource
	res = state.GetResource("nonexistent")
	if res != nil {
		t.Error("expected nil for non-existent resource")
	}

	// Nil state
	var nilState *SyncState
	res = nilState.GetResource("page-1")
	if res != nil {
		t.Error("expected nil for nil state")
	}
}

func TestSetResource(t *testing.T) {
	state := NewSyncState()

	res := ResourceState{
		ID:    "page-1",
		Type:  ResourceTypePage,
		Title: "New Page",
	}
	state.SetResource(res)

	if len(state.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(state.Resources))
	}
	if state.Resources["page-1"].Title != "New Page" {
		t.Errorf("unexpected title: %s", state.Resources["page-1"].Title)
	}

	// Update existing
	res.Title = "Updated Page"
	state.SetResource(res)
	if state.Resources["page-1"].Title != "Updated Page" {
		t.Errorf("expected 'Updated Page', got %q", state.Resources["page-1"].Title)
	}
}

// TestDatabaseStateInitAndUpdate tests the pattern used in syncDatabase where we:
// 1. Get resource (may be nil for new databases)
// 2. Set resource if nil to initialize it
// 3. Add entries to the database
// 4. Update the resource with the entries
// This test ensures we don't have nil pointer issues when accessing Entries.
func TestDatabaseStateInitAndUpdate(t *testing.T) {
	state := NewSyncState()
	dbID := "db-123"

	// Step 1: Get resource - should be nil for new database
	dbState := state.GetResource(dbID)
	if dbState != nil {
		t.Fatal("expected nil for new database")
	}

	// Step 2: Initialize if nil (simulating syncDatabase behavior)
	if dbState == nil {
		state.SetResource(ResourceState{
			ID:      dbID,
			Type:    ResourceTypeDatabase,
			Title:   "Test Database",
			Entries: make(map[string]EntryState),
		})
		dbState = state.GetResource(dbID)
	}

	// Verify dbState is now valid
	if dbState == nil {
		t.Fatal("dbState should not be nil after initialization")
	}
	if dbState.Entries == nil {
		t.Fatal("dbState.Entries should not be nil after initialization")
	}

	// Step 3: Add entries (simulating entry sync)
	_ = state.SetEntry(dbID, EntryState{
		PageID: "entry-1",
		Title:  "Entry One",
	})
	_ = state.SetEntry(dbID, EntryState{
		PageID: "entry-2",
		Title:  "Entry Two",
	})

	// Step 4: Update database state using cached dbState.Entries
	// This is the pattern that previously could cause nil pointer dereference
	state.SetResource(ResourceState{
		ID:      dbID,
		Type:    ResourceTypeDatabase,
		Title:   "Test Database Updated",
		Entries: dbState.Entries,
	})

	// Verify final state
	finalState := state.GetResource(dbID)
	if finalState == nil {
		t.Fatal("final state should not be nil")
	}
	if finalState.Title != "Test Database Updated" {
		t.Errorf("expected updated title, got %q", finalState.Title)
	}
	if len(finalState.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(finalState.Entries))
	}
}

func TestRemoveResource(t *testing.T) {
	state := NewSyncState()
	state.Resources["page-1"] = ResourceState{ID: "page-1"}
	state.Resources["page-2"] = ResourceState{ID: "page-2"}

	state.RemoveResource("page-1")

	if len(state.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(state.Resources))
	}
	if _, ok := state.Resources["page-1"]; ok {
		t.Error("page-1 should have been removed")
	}
	if _, ok := state.Resources["page-2"]; !ok {
		t.Error("page-2 should still exist")
	}

	// Removing non-existent should not panic
	state.RemoveResource("nonexistent")
}

func TestGetEntry(t *testing.T) {
	state := NewSyncState()
	state.Resources["db-1"] = ResourceState{
		ID:   "db-1",
		Type: ResourceTypeDatabase,
		Entries: map[string]EntryState{
			"entry-1": {
				PageID: "entry-1",
				Title:  "Entry One",
			},
		},
	}

	// Existing entry
	entry := state.GetEntry("db-1", "entry-1")
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Title != "Entry One" {
		t.Errorf("expected title 'Entry One', got %q", entry.Title)
	}

	// Non-existent entry
	entry = state.GetEntry("db-1", "nonexistent")
	if entry != nil {
		t.Error("expected nil for non-existent entry")
	}

	// Non-existent database
	entry = state.GetEntry("nonexistent", "entry-1")
	if entry != nil {
		t.Error("expected nil for non-existent database")
	}
}

func TestSetEntry(t *testing.T) {
	state := NewSyncState()
	state.Resources["db-1"] = ResourceState{
		ID:      "db-1",
		Type:    ResourceTypeDatabase,
		Entries: make(map[string]EntryState),
	}

	entry := EntryState{
		PageID: "entry-1",
		Title:  "New Entry",
	}
	err := state.SetEntry("db-1", entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := state.GetEntry("db-1", "entry-1")
	if got == nil {
		t.Fatal("expected entry to be set")
	}
	if got.Title != "New Entry" {
		t.Errorf("expected title 'New Entry', got %q", got.Title)
	}

	// Non-existent database
	err = state.SetEntry("nonexistent", entry)
	if err == nil {
		t.Error("expected error for non-existent database")
	}

	// Not a database
	state.Resources["page-1"] = ResourceState{ID: "page-1", Type: ResourceTypePage}
	err = state.SetEntry("page-1", entry)
	if err == nil {
		t.Error("expected error for non-database resource")
	}
}

func TestRemoveEntry(t *testing.T) {
	state := NewSyncState()
	state.Resources["db-1"] = ResourceState{
		ID:   "db-1",
		Type: ResourceTypeDatabase,
		Entries: map[string]EntryState{
			"entry-1": {PageID: "entry-1"},
			"entry-2": {PageID: "entry-2"},
		},
	}

	state.RemoveEntry("db-1", "entry-1")

	db := state.Resources["db-1"]
	if len(db.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(db.Entries))
	}
	if _, ok := db.Entries["entry-1"]; ok {
		t.Error("entry-1 should have been removed")
	}

	// Removing from non-existent database should not panic
	state.RemoveEntry("nonexistent", "entry-1")
}

func TestNeedsSync(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	state := NewSyncState()
	state.Resources["page-1"] = ResourceState{
		ID:           "page-1",
		LastModified: now,
	}

	tests := []struct {
		name         string
		id           string
		lastModified time.Time
		wantSync     bool
	}{
		{"new resource", "page-new", now, true},
		{"unchanged", "page-1", now, false},
		{"older than state", "page-1", earlier, false},
		{"newer than state", "page-1", later, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := state.NeedsSync(tt.id, tt.lastModified)
			if got != tt.wantSync {
				t.Errorf("NeedsSync(%s, %v) = %v, want %v", tt.id, tt.lastModified, got, tt.wantSync)
			}
		})
	}
}

func TestNeedsEntrySync(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	state := NewSyncState()
	state.Resources["db-1"] = ResourceState{
		ID:   "db-1",
		Type: ResourceTypeDatabase,
		Entries: map[string]EntryState{
			"entry-1": {
				PageID:       "entry-1",
				LastModified: now,
			},
		},
	}

	tests := []struct {
		name         string
		entryID      string
		lastModified time.Time
		wantSync     bool
	}{
		{"new entry", "entry-new", now, true},
		{"unchanged", "entry-1", now, false},
		{"older than state", "entry-1", earlier, false},
		{"newer than state", "entry-1", later, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := state.NeedsEntrySync("db-1", tt.entryID, tt.lastModified)
			if got != tt.wantSync {
				t.Errorf("NeedsEntrySync(db-1, %s, %v) = %v, want %v", tt.entryID, tt.lastModified, got, tt.wantSync)
			}
		})
	}
}

func TestDetectDeletedResources(t *testing.T) {
	state := NewSyncState()
	state.Resources["page-1"] = ResourceState{ID: "page-1", Type: ResourceTypePage, Title: "Page One"}
	state.Resources["page-2"] = ResourceState{ID: "page-2", Type: ResourceTypePage, Title: "Page Two"}
	state.Resources["db-1"] = ResourceState{ID: "db-1", Type: ResourceTypeDatabase, Title: "DB One"}

	// Only page-1 and db-1 are currently in Notion
	currentIDs := map[string]bool{
		"page-1": true,
		"db-1":   true,
	}

	deleted := state.DetectDeletedResources(currentIDs)

	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted resource, got %d", len(deleted))
	}
	if deleted[0].ID != "page-2" {
		t.Errorf("expected deleted ID 'page-2', got %q", deleted[0].ID)
	}
	if deleted[0].ChangeType != ChangeTypeDeleted {
		t.Errorf("expected ChangeTypeDeleted, got %s", deleted[0].ChangeType)
	}
}

func TestDetectDeletedEntries(t *testing.T) {
	state := NewSyncState()
	state.Resources["db-1"] = ResourceState{
		ID:   "db-1",
		Type: ResourceTypeDatabase,
		Entries: map[string]EntryState{
			"entry-1": {PageID: "entry-1", Title: "Entry One"},
			"entry-2": {PageID: "entry-2", Title: "Entry Two"},
			"entry-3": {PageID: "entry-3", Title: "Entry Three"},
		},
	}

	// Only entry-1 and entry-3 are currently in the database
	currentEntryIDs := map[string]bool{
		"entry-1": true,
		"entry-3": true,
	}

	deleted := state.DetectDeletedEntries("db-1", currentEntryIDs)

	if len(deleted) != 1 {
		t.Fatalf("expected 1 deleted entry, got %d", len(deleted))
	}
	if deleted[0].PageID != "entry-2" {
		t.Errorf("expected deleted PageID 'entry-2', got %q", deleted[0].PageID)
	}
	if deleted[0].ChangeType != ChangeTypeDeleted {
		t.Errorf("expected ChangeTypeDeleted, got %s", deleted[0].ChangeType)
	}

	// Non-existent database returns nil
	deleted = state.DetectDeletedEntries("nonexistent", currentEntryIDs)
	if deleted != nil {
		t.Errorf("expected nil for non-existent database, got %v", deleted)
	}
}

func TestResourceCount(t *testing.T) {
	state := NewSyncState()
	if state.ResourceCount() != 0 {
		t.Errorf("expected 0, got %d", state.ResourceCount())
	}

	state.Resources["page-1"] = ResourceState{ID: "page-1"}
	state.Resources["page-2"] = ResourceState{ID: "page-2"}
	if state.ResourceCount() != 2 {
		t.Errorf("expected 2, got %d", state.ResourceCount())
	}

	var nilState *SyncState
	if nilState.ResourceCount() != 0 {
		t.Errorf("expected 0 for nil state, got %d", nilState.ResourceCount())
	}
}

func TestEntryCount(t *testing.T) {
	state := NewSyncState()
	if state.EntryCount() != 0 {
		t.Errorf("expected 0, got %d", state.EntryCount())
	}

	state.Resources["page-1"] = ResourceState{ID: "page-1", Type: ResourceTypePage}
	state.Resources["db-1"] = ResourceState{
		ID:   "db-1",
		Type: ResourceTypeDatabase,
		Entries: map[string]EntryState{
			"entry-1": {},
			"entry-2": {},
		},
	}
	state.Resources["db-2"] = ResourceState{
		ID:   "db-2",
		Type: ResourceTypeDatabase,
		Entries: map[string]EntryState{
			"entry-3": {},
		},
	}

	if state.EntryCount() != 3 {
		t.Errorf("expected 3, got %d", state.EntryCount())
	}
}

func TestAllLocalPaths(t *testing.T) {
	state := NewSyncState()
	state.Resources["page-1"] = ResourceState{
		ID:        "page-1",
		LocalPath: "Page One.md",
	}
	state.Resources["db-1"] = ResourceState{
		ID:        "db-1",
		Type:      ResourceTypeDatabase,
		LocalPath: "My Database",
		Entries: map[string]EntryState{
			"entry-1": {LocalFile: "Entry One.md"},
			"entry-2": {LocalFile: "Entry Two.md"},
		},
	}

	paths := state.AllLocalPaths()

	if len(paths) != 4 {
		t.Errorf("expected 4 paths, got %d", len(paths))
	}

	// Check all expected paths are present
	expected := map[string]bool{
		"Page One.md":  false,
		"My Database":  false,
		"Entry One.md": false,
		"Entry Two.md": false,
	}
	for _, p := range paths {
		if _, ok := expected[p]; ok {
			expected[p] = true
		}
	}
	for path, found := range expected {
		if !found {
			t.Errorf("missing expected path: %s", path)
		}
	}
}

func TestAllLocalPaths_Empty(t *testing.T) {
	state := NewSyncState()
	paths := state.AllLocalPaths()
	if len(paths) != 0 {
		t.Errorf("expected empty slice, got %v", paths)
	}

	var nilState *SyncState
	paths = nilState.AllLocalPaths()
	if paths != nil {
		t.Errorf("expected nil for nil state, got %v", paths)
	}
}
