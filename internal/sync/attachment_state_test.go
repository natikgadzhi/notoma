package sync

import (
	"testing"
)

func TestSyncState_AttachmentState(t *testing.T) {
	state := NewSyncState()

	// Initially no attachments
	if state.HasAttachment("https://example.com/image.png") {
		t.Error("HasAttachment() should return false for non-existent attachment")
	}

	// Add attachment
	state.UpdateAttachmentState(
		"https://example.com/image.png",
		"abc123hash",
		"_attachments/image.png",
		12345,
		"page-1",
	)

	// Now attachment should exist
	if !state.HasAttachment("https://example.com/image.png") {
		t.Error("HasAttachment() should return true after UpdateAttachmentState")
	}

	// Get attachment by URL
	att := state.GetAttachmentByURL("https://example.com/image.png")
	if att == nil {
		t.Fatal("GetAttachmentByURL() returned nil")
	}
	if att.ContentHash != "abc123hash" {
		t.Errorf("AttachmentState.ContentHash = %q, want %q", att.ContentHash, "abc123hash")
	}
	if att.LocalPath != "_attachments/image.png" {
		t.Errorf("AttachmentState.LocalPath = %q, want %q", att.LocalPath, "_attachments/image.png")
	}
	if att.Size != 12345 {
		t.Errorf("AttachmentState.Size = %d, want %d", att.Size, 12345)
	}
	if att.PageID != "page-1" {
		t.Errorf("AttachmentState.PageID = %q, want %q", att.PageID, "page-1")
	}

	// Get local path
	localPath := state.GetAttachmentLocalPath("https://example.com/image.png")
	if localPath != "_attachments/image.png" {
		t.Errorf("GetAttachmentLocalPath() = %q, want %q", localPath, "_attachments/image.png")
	}

	// Non-existent attachment local path
	localPath = state.GetAttachmentLocalPath("https://example.com/nonexistent.png")
	if localPath != "" {
		t.Errorf("GetAttachmentLocalPath() for non-existent = %q, want empty", localPath)
	}
}

func TestSyncState_GetAttachmentsByPage(t *testing.T) {
	state := NewSyncState()

	// Add attachments for different pages
	state.UpdateAttachmentState("url1", "hash1", "path1", 100, "page-1")
	state.UpdateAttachmentState("url2", "hash2", "path2", 200, "page-1")
	state.UpdateAttachmentState("url3", "hash3", "path3", 300, "page-2")

	// Get attachments for page-1
	page1Atts := state.GetAttachmentsByPage("page-1")
	if len(page1Atts) != 2 {
		t.Errorf("GetAttachmentsByPage(page-1) returned %d items, want 2", len(page1Atts))
	}

	// Get attachments for page-2
	page2Atts := state.GetAttachmentsByPage("page-2")
	if len(page2Atts) != 1 {
		t.Errorf("GetAttachmentsByPage(page-2) returned %d items, want 1", len(page2Atts))
	}

	// Get attachments for non-existent page
	page3Atts := state.GetAttachmentsByPage("page-3")
	if len(page3Atts) != 0 {
		t.Errorf("GetAttachmentsByPage(page-3) returned %d items, want 0", len(page3Atts))
	}
}

func TestSyncState_DetectOrphanedAttachments(t *testing.T) {
	state := NewSyncState()

	// Add attachments for different pages
	state.UpdateAttachmentState("url1", "hash1", "path1", 100, "page-1")
	state.UpdateAttachmentState("url2", "hash2", "path2", 200, "page-2")
	state.UpdateAttachmentState("url3", "hash3", "path3", 300, "page-3")

	// Current pages only include page-1 and page-3
	currentPageIDs := map[string]bool{
		"page-1": true,
		"page-3": true,
	}

	orphaned := state.DetectOrphanedAttachments(currentPageIDs)
	if len(orphaned) != 1 {
		t.Errorf("DetectOrphanedAttachments() returned %d items, want 1", len(orphaned))
	}
	if len(orphaned) > 0 && orphaned[0].PageID != "page-2" {
		t.Errorf("Orphaned attachment PageID = %q, want %q", orphaned[0].PageID, "page-2")
	}
}

func TestSyncState_CleanupOrphanedAttachments(t *testing.T) {
	state := NewSyncState()

	// Add attachments for different pages
	state.UpdateAttachmentState("url1", "hash1", "path1", 100, "page-1")
	state.UpdateAttachmentState("url2", "hash2", "path2", 200, "page-2")
	state.UpdateAttachmentState("url3", "hash3", "path3", 300, "page-3")

	// Current pages only include page-1 and page-3
	currentPageIDs := map[string]bool{
		"page-1": true,
		"page-3": true,
	}

	removed := state.CleanupOrphanedAttachments(currentPageIDs)
	if len(removed) != 1 {
		t.Errorf("CleanupOrphanedAttachments() returned %d paths, want 1", len(removed))
	}
	if len(removed) > 0 && removed[0] != "path2" {
		t.Errorf("Removed path = %q, want %q", removed[0], "path2")
	}

	// Attachment should be removed from state
	if state.HasAttachment("url2") {
		t.Error("Orphaned attachment should be removed from state")
	}

	// Other attachments should remain
	if !state.HasAttachment("url1") {
		t.Error("Non-orphaned attachment url1 should remain")
	}
	if !state.HasAttachment("url3") {
		t.Error("Non-orphaned attachment url3 should remain")
	}
}

func TestSyncState_RemoveAttachment(t *testing.T) {
	state := NewSyncState()

	state.UpdateAttachmentState("url1", "hash1", "path1", 100, "page-1")
	if !state.HasAttachment("url1") {
		t.Fatal("Attachment should exist after UpdateAttachmentState")
	}

	state.RemoveAttachment("url1")
	if state.HasAttachment("url1") {
		t.Error("Attachment should not exist after RemoveAttachment")
	}
}

func TestSyncState_AttachmentNeedsRedownload(t *testing.T) {
	state := NewSyncState()

	// New attachment needs download
	if !state.AttachmentNeedsRedownload("url1", "") {
		t.Error("AttachmentNeedsRedownload() should return true for new attachment")
	}

	// Add attachment
	state.UpdateAttachmentState("url1", "hash1", "path1", 100, "page-1")

	// Same hash - no redownload needed
	if state.AttachmentNeedsRedownload("url1", "hash1") {
		t.Error("AttachmentNeedsRedownload() should return false for same hash")
	}

	// Different hash - redownload needed
	if !state.AttachmentNeedsRedownload("url1", "hash2") {
		t.Error("AttachmentNeedsRedownload() should return true for different hash")
	}

	// Empty hash on check - no redownload (can't compare)
	if state.AttachmentNeedsRedownload("url1", "") {
		t.Error("AttachmentNeedsRedownload() should return false when new hash is empty")
	}
}

func TestSyncState_AttachmentStats(t *testing.T) {
	state := NewSyncState()

	// Add some attachments
	state.UpdateAttachmentState("url1", "hash1", "path1", 100, "page-1")
	state.UpdateAttachmentState("url2", "hash2", "path2", 200, "page-1")
	state.UpdateAttachmentState("url3", "hash3", "path3", 300, "page-2")

	stats := state.GetAttachmentStats()
	if stats.TotalCount != 3 {
		t.Errorf("AttachmentStats.TotalCount = %d, want 3", stats.TotalCount)
	}
	if stats.TotalSize != 600 {
		t.Errorf("AttachmentStats.TotalSize = %d, want 600", stats.TotalSize)
	}
}

func TestHashURL(t *testing.T) {
	// Same URL should produce same hash
	hash1 := HashURL("https://example.com/image.png")
	hash2 := HashURL("https://example.com/image.png")
	if hash1 != hash2 {
		t.Error("HashURL() should produce same hash for same URL")
	}

	// Different URLs should produce different hashes
	hash3 := HashURL("https://example.com/other.png")
	if hash1 == hash3 {
		t.Error("HashURL() should produce different hash for different URL")
	}

	// Hash should be hex string
	if len(hash1) != 64 { // SHA-256 produces 32 bytes = 64 hex chars
		t.Errorf("HashURL() length = %d, want 64", len(hash1))
	}
}
