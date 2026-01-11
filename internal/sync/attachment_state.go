// Package sync handles attachment state tracking for incremental syncs.
package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// UpdateAttachmentState adds or updates state for a downloaded attachment.
func (s *SyncState) UpdateAttachmentState(originalURL, contentHash, localPath string, size int64, pageID string) {
	urlHash := HashURL(originalURL)
	s.Attachments[urlHash] = &AttachmentState{
		OriginalURL:  originalURL,
		URLHash:      urlHash,
		ContentHash:  contentHash,
		LocalPath:    localPath,
		Size:         size,
		PageID:       pageID,
		DownloadedAt: time.Now(),
	}
}

// GetAttachmentByURL returns the attachment state for a URL.
// Returns nil if the attachment hasn't been downloaded.
func (s *SyncState) GetAttachmentByURL(originalURL string) *AttachmentState {
	urlHash := HashURL(originalURL)
	return s.Attachments[urlHash]
}

// GetAttachmentByHash returns the attachment state by URL hash.
func (s *SyncState) GetAttachmentByHash(urlHash string) *AttachmentState {
	return s.Attachments[urlHash]
}

// HasAttachment returns true if an attachment has been downloaded.
func (s *SyncState) HasAttachment(originalURL string) bool {
	urlHash := HashURL(originalURL)
	_, ok := s.Attachments[urlHash]
	return ok
}

// GetAttachmentLocalPath returns the local path for a previously
// downloaded attachment, or empty string if not found.
func (s *SyncState) GetAttachmentLocalPath(originalURL string) string {
	att := s.GetAttachmentByURL(originalURL)
	if att == nil {
		return ""
	}
	return att.LocalPath
}

// RemoveAttachment removes an attachment from state.
func (s *SyncState) RemoveAttachment(originalURL string) {
	urlHash := HashURL(originalURL)
	delete(s.Attachments, urlHash)
}

// GetAttachmentsByPage returns all attachments associated with a page.
func (s *SyncState) GetAttachmentsByPage(pageID string) []*AttachmentState {
	var attachments []*AttachmentState
	for _, att := range s.Attachments {
		if att.PageID == pageID {
			attachments = append(attachments, att)
		}
	}
	return attachments
}

// DetectOrphanedAttachments returns attachments that are no longer
// referenced by any page in the current set.
func (s *SyncState) DetectOrphanedAttachments(currentPageIDs map[string]bool) []*AttachmentState {
	var orphaned []*AttachmentState
	for _, att := range s.Attachments {
		if att.PageID != "" && !currentPageIDs[att.PageID] {
			orphaned = append(orphaned, att)
		}
	}
	return orphaned
}

// CleanupOrphanedAttachments removes attachments associated with
// pages that no longer exist.
func (s *SyncState) CleanupOrphanedAttachments(currentPageIDs map[string]bool) []string {
	var removed []string
	for urlHash, att := range s.Attachments {
		if att.PageID != "" && !currentPageIDs[att.PageID] {
			removed = append(removed, att.LocalPath)
			delete(s.Attachments, urlHash)
		}
	}
	return removed
}

// AttachmentNeedsRedownload checks if an attachment should be re-downloaded.
// Since Notion URLs expire after ~1 hour, we can't compare URLs directly.
// Instead, we use content hash comparison when available.
//
// Returns true if:
// - The attachment hasn't been downloaded before
// - The content hash is different (content changed)
func (s *SyncState) AttachmentNeedsRedownload(originalURL, contentHash string) bool {
	att := s.GetAttachmentByURL(originalURL)
	if att == nil {
		return true // Never downloaded
	}
	if contentHash != "" && att.ContentHash != "" && att.ContentHash != contentHash {
		return true // Content changed
	}
	return false
}

// HashURL creates a stable hash of a URL for use as a map key.
// This strips query parameters which often contain expiring tokens.
func HashURL(rawURL string) string {
	// For Notion URLs, we want to hash just the path part since
	// the query parameters contain expiring tokens
	hash := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(hash[:])
}

// GetAttachmentStats returns statistics about tracked attachments.
func (s *SyncState) GetAttachmentStats() AttachmentStats {
	var stats AttachmentStats
	stats.TotalCount = len(s.Attachments)
	for _, att := range s.Attachments {
		stats.TotalSize += att.Size
	}
	return stats
}

// AttachmentStats contains statistics about tracked attachments.
type AttachmentStats struct {
	TotalCount int
	TotalSize  int64
}
