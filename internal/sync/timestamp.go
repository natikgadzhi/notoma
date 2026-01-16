// Package sync provides synchronization logic between Notion and Obsidian.
package sync

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/natikgadzhi/notion-based/internal/notion"
)

// TimestampUpdater updates Notion pages with sync timestamps.
type TimestampUpdater struct {
	client  *notion.Client
	logger  *slog.Logger
	enabled bool
	dryRun  bool
}

// NewTimestampUpdater creates a new timestamp updater.
// If enabled is false, UpdateAfterSync is a no-op.
func NewTimestampUpdater(client *notion.Client, logger *slog.Logger, enabled bool, dryRun bool) *TimestampUpdater {
	if logger == nil {
		logger = slog.Default()
	}
	return &TimestampUpdater{
		client:  client,
		logger:  logger,
		enabled: enabled,
		dryRun:  dryRun,
	}
}

// UpdateAfterSync updates the last_notoma_sync_at property on a page after successful sync.
// Returns nil if the updater is disabled or in dry-run mode.
// Returns nil with a warning log if the property doesn't exist on the page.
// Returns error only for unexpected failures.
func (u *TimestampUpdater) UpdateAfterSync(ctx context.Context, pageID string) error {
	if !u.enabled {
		return nil
	}

	if u.dryRun {
		u.logger.Debug("would update timestamp (dry-run)", "page_id", pageID)
		return nil
	}

	timestamp := time.Now().UTC()
	err := u.client.UpdatePageTimestamp(ctx, pageID, timestamp)
	if err != nil {
		// Check if it's a property not found error - this is expected and non-fatal
		var propErr *notion.PropertyNotFoundError
		if errors.As(err, &propErr) {
			u.logger.Warn("timestamp property not found on page, skipping update",
				"page_id", pageID,
				"property", propErr.PropertyName,
			)
			return nil
		}

		// Other errors are logged but don't fail the sync
		u.logger.Error("failed to update page timestamp",
			"page_id", pageID,
			"error", err,
		)
		return nil // Don't fail the sync for timestamp update failures
	}

	u.logger.Debug("updated page timestamp", "page_id", pageID, "timestamp", timestamp)
	return nil
}

// IsEnabled returns whether timestamp updates are enabled.
func (u *TimestampUpdater) IsEnabled() bool {
	return u.enabled
}
