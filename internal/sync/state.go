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

// State tracks synchronization state for incremental updates.
type State struct {
	// Version is the state file format version for future migrations.
	Version int `json:"version"`

	// LastSync is when the last successful sync completed.
	LastSync time.Time `json:"last_sync"`

	// Pages tracks the state of individual pages.
	Pages map[string]*PageState `json:"pages"`

	// Databases tracks the state of databases.
	Databases map[string]*DatabaseState `json:"databases"`

	// Attachments tracks downloaded attachments by URL hash.
	Attachments map[string]*AttachmentState `json:"attachments"`

	// path is the file path where state is persisted (not serialized).
	path string
}

// PageState tracks the sync state of a single page.
type PageState struct {
	// ID is the Notion page ID.
	ID string `json:"id"`

	// Title is the page title (for logging/debugging).
	Title string `json:"title"`

	// LastEdited is the page's last_edited_time from Notion.
	LastEdited time.Time `json:"last_edited"`

	// ContentHash is a hash of the transformed content.
	// Used to detect if regeneration produced different output.
	ContentHash string `json:"content_hash,omitempty"`

	// OutputPath is the relative path to the output file in the vault.
	OutputPath string `json:"output_path"`

	// SyncedAt is when we last synced this page.
	SyncedAt time.Time `json:"synced_at"`
}

// DatabaseState tracks the sync state of a database.
type DatabaseState struct {
	// ID is the Notion database ID.
	ID string `json:"id"`

	// Title is the database title.
	Title string `json:"title"`

	// LastEdited is the database's last_edited_time from Notion.
	LastEdited time.Time `json:"last_edited"`

	// OutputFolder is the relative path to the output folder in the vault.
	OutputFolder string `json:"output_folder"`

	// SyncedAt is when we last synced this database.
	SyncedAt time.Time `json:"synced_at"`

	// EntryCount is the number of entries in the database.
	EntryCount int `json:"entry_count"`
}

// AttachmentState tracks a downloaded attachment.
type AttachmentState struct {
	// URLHash is the SHA256 hash of the original URL (used as key).
	URLHash string `json:"url_hash"`

	// OriginalURL is the original remote URL.
	OriginalURL string `json:"original_url"`

	// LocalPath is the path relative to the vault.
	LocalPath string `json:"local_path"`

	// DownloadedAt is when the attachment was downloaded.
	DownloadedAt time.Time `json:"downloaded_at"`

	// Size is the file size in bytes.
	Size int64 `json:"size"`
}

const stateVersion = 1

// NewState creates a new empty state.
func NewState(path string) *State {
	return &State{
		Version:     stateVersion,
		Pages:       make(map[string]*PageState),
		Databases:   make(map[string]*DatabaseState),
		Attachments: make(map[string]*AttachmentState),
		path:        path,
	}
}

// LoadState loads state from a file, or creates a new state if the file doesn't exist.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewState(path), nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	state.path = path

	// Initialize maps if nil (for older state files)
	if state.Pages == nil {
		state.Pages = make(map[string]*PageState)
	}
	if state.Databases == nil {
		state.Databases = make(map[string]*DatabaseState)
	}
	if state.Attachments == nil {
		state.Attachments = make(map[string]*AttachmentState)
	}

	return &state, nil
}

// Save persists the state to disk.
func (s *State) Save() error {
	// Ensure parent directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}

	return nil
}

// GetPage returns the state for a page, or nil if not tracked.
func (s *State) GetPage(id string) *PageState {
	return s.Pages[id]
}

// SetPage updates the state for a page.
func (s *State) SetPage(ps *PageState) {
	s.Pages[ps.ID] = ps
}

// GetDatabase returns the state for a database, or nil if not tracked.
func (s *State) GetDatabase(id string) *DatabaseState {
	return s.Databases[id]
}

// SetDatabase updates the state for a database.
func (s *State) SetDatabase(ds *DatabaseState) {
	s.Databases[ds.ID] = ds
}

// GetAttachment returns the state for an attachment by URL hash.
func (s *State) GetAttachment(urlHash string) *AttachmentState {
	return s.Attachments[urlHash]
}

// SetAttachment updates the state for an attachment.
func (s *State) SetAttachment(as *AttachmentState) {
	s.Attachments[as.URLHash] = as
}

// NeedsSync checks if a page needs to be synced based on its last_edited time.
func (s *State) NeedsSync(pageID string, lastEdited time.Time) bool {
	ps := s.GetPage(pageID)
	if ps == nil {
		return true
	}
	return lastEdited.After(ps.LastEdited)
}

// NeedsDatabaseSync checks if a database needs to be synced.
func (s *State) NeedsDatabaseSync(databaseID string, lastEdited time.Time) bool {
	ds := s.GetDatabase(databaseID)
	if ds == nil {
		return true
	}
	return lastEdited.After(ds.LastEdited)
}

// MarkSynced updates the last sync time.
func (s *State) MarkSynced() {
	s.LastSync = time.Now()
}

// Path returns the state file path.
func (s *State) Path() string {
	return s.path
}

// Reset clears all tracked state (for force sync).
func (s *State) Reset() {
	s.Pages = make(map[string]*PageState)
	s.Databases = make(map[string]*DatabaseState)
	s.Attachments = make(map[string]*AttachmentState)
	s.LastSync = time.Time{}
}

// Summary returns a summary of the current state.
func (s *State) Summary() StateSummary {
	return StateSummary{
		LastSync:        s.LastSync,
		PageCount:       len(s.Pages),
		DatabaseCount:   len(s.Databases),
		AttachmentCount: len(s.Attachments),
	}
}

// StateSummary provides a high-level overview of sync state.
type StateSummary struct {
	LastSync        time.Time
	PageCount       int
	DatabaseCount   int
	AttachmentCount int
}
