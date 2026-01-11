package notion_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/natikgadzhi/notion-based/internal/notion"
)

// TestSmokeTestDatabaseEntries is an integration test that verifies
// the test database has exactly 3 top-level pages.
// This test requires NOTION_TOKEN environment variable to be set.
// Run with: go test -v ./internal/notion -run TestSmokeTest
func TestSmokeTestDatabaseEntries(t *testing.T) {
	// Load .env file if present
	_ = godotenv.Load("../../.env")

	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		t.Skip("NOTION_TOKEN not set, skipping smoke test")
	}

	// Test database URL from config.yaml
	testDatabaseURL := "https://www.notion.so/1e567c00f3f9805dafe3e53f460e93e3?v=1e667c00f3f980848aea000c507d821a"

	// Parse URL to get database ID
	parsed, err := notion.ParseURL(testDatabaseURL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	// Create client
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	client := notion.NewClient(token, logger)
	ctx := context.Background()

	// Detect resource type - should be a database
	resource, err := client.DetectResourceType(ctx, parsed.ID)
	if err != nil {
		t.Fatalf("failed to detect resource type: %v", err)
	}

	if resource.Type != notion.ResourceTypeDatabase {
		t.Fatalf("expected database, got %s", resource.Type)
	}

	// Query database entries
	pages, err := client.QueryDatabase(ctx, resource.ID)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}

	// Verify we have exactly 3 entries
	expectedCount := 3
	if len(pages) != expectedCount {
		t.Errorf("expected %d entries in test database, got %d", expectedCount, len(pages))
		for i, page := range pages {
			t.Logf("  entry %d: %s", i+1, notion.ExtractPageTitle(&page))
		}
	}

	t.Logf("Found %d entries in test database (expected %d)", len(pages), expectedCount)
}
