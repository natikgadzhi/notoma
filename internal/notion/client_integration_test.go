//go:build integration

package notion

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// Integration tests require NOTION_TOKEN environment variable.
// Run with: go test -tags=integration ./internal/notion/...

func TestIntegration_GetCurrentUser(t *testing.T) {
	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		t.Skip("NOTION_TOKEN not set, skipping integration test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(token, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}

	if user == nil {
		t.Fatal("GetCurrentUser returned nil user")
	}

	t.Logf("Connected as: %s (type: %s)", user.Name, user.Type)
}

func TestIntegration_DiscoverWorkspaceRoots(t *testing.T) {
	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		t.Skip("NOTION_TOKEN not set, skipping integration test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(token, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	roots, err := client.DiscoverWorkspaceRoots(ctx)
	if err != nil {
		t.Fatalf("DiscoverWorkspaceRoots failed: %v", err)
	}

	t.Logf("Discovered %d workspace roots:", len(roots))
	for _, root := range roots {
		t.Logf("  - [%s] %s (%s)", root.Type, root.Title, root.ID)
	}

	// Verify we got some results (depends on the workspace, but there should be at least something)
	if len(roots) == 0 {
		t.Log("Warning: no workspace roots found - this may be expected for an empty workspace")
	}

	// Verify all roots have required fields
	for i, root := range roots {
		if root.ID == "" {
			t.Errorf("root[%d].ID is empty", i)
		}
		if root.Type != ResourceTypePage && root.Type != ResourceTypeDatabase {
			t.Errorf("root[%d].Type is invalid: %s", i, root.Type)
		}
	}
}

func TestIntegration_SearchAll(t *testing.T) {
	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		t.Skip("NOTION_TOKEN not set, skipping integration test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(token, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test searching for pages
	pages, err := client.SearchAll(ctx, "page")
	if err != nil {
		t.Fatalf("SearchAll(page) failed: %v", err)
	}
	t.Logf("Found %d pages", len(pages))

	// Test searching for databases
	databases, err := client.SearchAll(ctx, "database")
	if err != nil {
		t.Fatalf("SearchAll(database) failed: %v", err)
	}
	t.Logf("Found %d databases", len(databases))
}
