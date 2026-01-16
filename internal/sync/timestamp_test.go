package sync

import (
	"context"
	"log/slog"
	"testing"
)

func TestTimestampUpdater_Disabled(t *testing.T) {
	// Create updater with enabled=false
	updater := NewTimestampUpdater(nil, slog.Default(), false, false)

	// Should be disabled
	if updater.IsEnabled() {
		t.Error("expected updater to be disabled")
	}

	// UpdateAfterSync should return nil immediately when disabled
	err := updater.UpdateAfterSync(context.Background(), "test-page-id")
	if err != nil {
		t.Errorf("expected nil error when disabled, got %v", err)
	}
}

func TestTimestampUpdater_DryRun(t *testing.T) {
	// Create updater with dryRun=true
	updater := NewTimestampUpdater(nil, slog.Default(), true, true)

	// Should be enabled
	if !updater.IsEnabled() {
		t.Error("expected updater to be enabled")
	}

	// UpdateAfterSync should return nil in dry-run mode without calling client
	err := updater.UpdateAfterSync(context.Background(), "test-page-id")
	if err != nil {
		t.Errorf("expected nil error in dry-run mode, got %v", err)
	}
}

func TestNewTimestampUpdater_NilLogger(t *testing.T) {
	// Should not panic with nil logger and should set default logger
	updater := NewTimestampUpdater(nil, nil, false, false)

	// Verify updater was created (constructor always returns non-nil)
	// and that calling IsEnabled doesn't panic (indirectly verifies logger is set)
	if updater.IsEnabled() {
		t.Error("expected updater to be disabled")
	}
}
