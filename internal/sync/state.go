// Package sync handles synchronization between Notion and Obsidian.
package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ResourceType indicates the type of Notion resource.
type ResourceType string

const (
	ResourceTypePage     ResourceType = "page"
	ResourceTypeDatabase ResourceType = "database"
)

// EntryState tracks the sync state of a database entry (page within a database).
type EntryState struct {
	PageID       string    `json:"page_id"`
	Title        string    `json:"title"`
	LastModified time.Time `json:"last_modified"`
	LocalFile    string    `json:"local_file"`
}

// ResourceState tracks the sync state of a Notion page or database.
type ResourceState struct {
	ID           string                `json:"id"`
	Type         ResourceType          `json:"type"`
	Title        string                `json:"title"`
	LastModified time.Time             `json:"last_modified"`
	LocalPath    string                `json:"local_path"`
	Entries      map[string]EntryState `json:"entries,omitempty"`
}

// AttachmentState tracks the sync state of a downloaded attachment.
type AttachmentState struct {
	// OriginalURL is the URL the attachment was downloaded from.
	// Note: Notion URLs expire, so this is for reference only.
	OriginalURL string `json:"original_url"`

	// URLHash is a hash of the original URL for stable identification.
	URLHash string `json:"url_hash"`

	// ContentHash is the SHA-256 hash of the file content.
	ContentHash string `json:"content_hash"`

	// LocalPath is the path to the downloaded file.
	LocalPath string `json:"local_path"`

	// Size is the file size in bytes.
	Size int64 `json:"size"`

	// PageID is the ID of the page that references this attachment.
	PageID string `json:"page_id,omitempty"`

	// DownloadedAt is when the attachment was downloaded.
	DownloadedAt time.Time `json:"downloaded_at"`
}

// SyncState is the top-level sync state persisted to disk.
type SyncState struct {
	Version      int                         `json:"version"`
	LastSyncTime time.Time                   `json:"last_sync_time"`
	Resources    map[string]ResourceState    `json:"resources"`
	Attachments  map[string]*AttachmentState `json:"attachments,omitempty"`
}

// StateVersion is the current schema version for the state file.
const StateVersion = 1

// NewSyncState creates a new empty sync state.
func NewSyncState() *SyncState {
	return &SyncState{
		Version:     StateVersion,
		Resources:   make(map[string]ResourceState),
		Attachments: make(map[string]*AttachmentState),
	}
}

// LoadState loads sync state from a file.
// If the file doesn't exist, returns a new empty state.
func LoadState(path string) (*SyncState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewSyncState(), nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	// Initialize maps if nil (for backwards compatibility)
	if state.Resources == nil {
		state.Resources = make(map[string]ResourceState)
	}
	for id, res := range state.Resources {
		if res.Entries == nil && res.Type == ResourceTypeDatabase {
			res.Entries = make(map[string]EntryState)
			state.Resources[id] = res
		}
	}
	if state.Attachments == nil {
		state.Attachments = make(map[string]*AttachmentState)
	}

	return &state, nil
}

// SaveState persists the sync state to a file.
// Creates parent directories if they don't exist.
func SaveState(path string, state *SyncState) error {
	if state == nil {
		return errors.New("state is nil")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	// Set version and update timestamp
	state.Version = StateVersion
	state.LastSyncTime = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	// Write atomically by using a temp file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

// Save is an alias for SaveState that operates on the receiver.
func (s *SyncState) Save(path string) error {
	return SaveState(path, s)
}

// GetResource returns the state of a resource by ID, or nil if not found.
func (s *SyncState) GetResource(id string) *ResourceState {
	if s == nil || s.Resources == nil {
		return nil
	}
	res, ok := s.Resources[id]
	if !ok {
		return nil
	}
	return &res
}

// SetResource updates or adds a resource to the state.
func (s *SyncState) SetResource(res ResourceState) {
	if s.Resources == nil {
		s.Resources = make(map[string]ResourceState)
	}
	s.Resources[res.ID] = res
}

// RemoveResource removes a resource from the state.
func (s *SyncState) RemoveResource(id string) {
	if s.Resources != nil {
		delete(s.Resources, id)
	}
}

// GetEntry returns the state of a database entry, or nil if not found.
func (s *SyncState) GetEntry(databaseID, entryID string) *EntryState {
	res := s.GetResource(databaseID)
	if res == nil || res.Entries == nil {
		return nil
	}
	entry, ok := res.Entries[entryID]
	if !ok {
		return nil
	}
	return &entry
}

// SetEntry updates or adds an entry to a database resource.
func (s *SyncState) SetEntry(databaseID string, entry EntryState) error {
	res := s.GetResource(databaseID)
	if res == nil {
		return fmt.Errorf("database %s not found in state", databaseID)
	}
	if res.Type != ResourceTypeDatabase {
		return fmt.Errorf("resource %s is not a database", databaseID)
	}
	if res.Entries == nil {
		res.Entries = make(map[string]EntryState)
	}
	res.Entries[entry.PageID] = entry
	s.Resources[databaseID] = *res
	return nil
}

// RemoveEntry removes an entry from a database resource.
func (s *SyncState) RemoveEntry(databaseID, entryID string) {
	res := s.GetResource(databaseID)
	if res == nil || res.Entries == nil {
		return
	}
	delete(res.Entries, entryID)
	s.Resources[databaseID] = *res
}

// ChangeType indicates how a resource has changed.
type ChangeType string

const (
	ChangeTypeNew      ChangeType = "new"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
)

// ResourceChange describes a detected change to a resource.
type ResourceChange struct {
	ID         string
	Type       ResourceType
	ChangeType ChangeType
	Title      string
}

// EntryChange describes a detected change to a database entry.
type EntryChange struct {
	DatabaseID string
	PageID     string
	ChangeType ChangeType
	Title      string
}

// NeedsSync determines if a page needs syncing based on its last modified time.
// Returns true if the page is new or has been modified since last sync.
func (s *SyncState) NeedsSync(id string, lastModified time.Time) bool {
	res := s.GetResource(id)
	if res == nil {
		// New resource
		return true
	}
	// Modified if the Notion timestamp is newer
	return lastModified.After(res.LastModified)
}

// NeedsEntrySync determines if a database entry needs syncing.
func (s *SyncState) NeedsEntrySync(databaseID, entryID string, lastModified time.Time) bool {
	entry := s.GetEntry(databaseID, entryID)
	if entry == nil {
		// New entry
		return true
	}
	// Modified if the Notion timestamp is newer
	return lastModified.After(entry.LastModified)
}

// DetectDeletedResources finds resources in state that are no longer in Notion.
// currentIDs should contain all resource IDs currently present in Notion (from config roots).
func (s *SyncState) DetectDeletedResources(currentIDs map[string]bool) []ResourceChange {
	var deleted []ResourceChange
	for id, res := range s.Resources {
		if !currentIDs[id] {
			deleted = append(deleted, ResourceChange{
				ID:         id,
				Type:       res.Type,
				ChangeType: ChangeTypeDeleted,
				Title:      res.Title,
			})
		}
	}
	return deleted
}

// DetectDeletedEntries finds entries in a database that are no longer present.
// currentEntryIDs should contain all entry IDs currently in the database.
func (s *SyncState) DetectDeletedEntries(databaseID string, currentEntryIDs map[string]bool) []EntryChange {
	res := s.GetResource(databaseID)
	if res == nil || res.Entries == nil {
		return nil
	}

	var deleted []EntryChange
	for entryID, entry := range res.Entries {
		if !currentEntryIDs[entryID] {
			deleted = append(deleted, EntryChange{
				DatabaseID: databaseID,
				PageID:     entryID,
				ChangeType: ChangeTypeDeleted,
				Title:      entry.Title,
			})
		}
	}
	return deleted
}

// ResourceCount returns the number of resources in the state.
func (s *SyncState) ResourceCount() int {
	if s == nil || s.Resources == nil {
		return 0
	}
	return len(s.Resources)
}

// EntryCount returns the total number of entries across all databases.
func (s *SyncState) EntryCount() int {
	if s == nil || s.Resources == nil {
		return 0
	}
	count := 0
	for _, res := range s.Resources {
		if res.Type == ResourceTypeDatabase && res.Entries != nil {
			count += len(res.Entries)
		}
	}
	return count
}

// AllLocalPaths returns a list of all local file paths tracked in the state.
// Useful for cleanup or verification.
func (s *SyncState) AllLocalPaths() []string {
	if s == nil || s.Resources == nil {
		return nil
	}
	var paths []string
	for _, res := range s.Resources {
		if res.LocalPath != "" {
			paths = append(paths, res.LocalPath)
		}
		for _, entry := range res.Entries {
			if entry.LocalFile != "" {
				paths = append(paths, entry.LocalFile)
			}
		}
	}
	return paths
}
