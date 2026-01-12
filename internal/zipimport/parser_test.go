package zipimport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTitleAndID(t *testing.T) {
	tests := []struct {
		filename  string
		wantTitle string
		wantID    string
	}{
		{
			filename:  "My Page abc123def456789012345678901234",
			wantTitle: "My Page",
			wantID:    "abc123def456789012345678901234",
		},
		{
			filename:  "Simple Title",
			wantTitle: "Simple Title",
			wantID:    "",
		},
		{
			filename:  "Page With Numbers 123 abc1234567890123456",
			wantTitle: "Page With Numbers 123",
			wantID:    "abc1234567890123456",
		},
		{
			filename:  "Short ID Page abcd1234",
			wantTitle: "Short ID Page",
			wantID:    "abcd1234",
		},
		{
			filename:  "   Spaces   abc1234567890123456",
			wantTitle: "Spaces",
			wantID:    "abc1234567890123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			gotTitle, gotID := extractTitleAndID(tt.filename)
			if gotTitle != tt.wantTitle {
				t.Errorf("title = %q, want %q", gotTitle, tt.wantTitle)
			}
			if gotID != tt.wantID {
				t.Errorf("id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

func TestStripNotionID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Page Title abc123def456789012345678901234", "Page Title"},
		{"Page Title abcd12345678", "Page Title"},
		{"No ID Here", "No ID Here"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StripNotionID(tt.input)
			if got != tt.want {
				t.Errorf("StripNotionID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsAttachment(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".pdf", true},
		{".doc", true},
		{".mp3", true},
		{".mp4", true},
		{".md", false},
		{".csv", false},
		{".txt", true},
		{".html", false},
		{".go", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := isAttachment(tt.ext); got != tt.want {
				t.Errorf("isAttachment(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	// Create a mock export directory structure
	rootDir := t.TempDir()

	// Create markdown files (IDs must be at least 8 hex chars)
	mdContent := "# Test Page\n\nSome content here."
	if err := os.WriteFile(filepath.Join(rootDir, "Test Page abc12345678.md"), []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to create md file: %v", err)
	}

	// Create subfolder with nested page
	subDir := filepath.Join(rootDir, "Parent Page def12345678")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	childContent := "# Child Page\n\nNested content."
	if err := os.WriteFile(filepath.Join(subDir, "Child Page 1234567890ab.md"), []byte(childContent), 0644); err != nil {
		t.Fatalf("failed to create child md file: %v", err)
	}

	// Create CSV file (database)
	csvContent := "Name,Tags,Date\nEntry 1,tag1,2024-01-01\nEntry 2,\"tag2,tag3\",2024-01-02"
	if err := os.WriteFile(filepath.Join(rootDir, "My Database 1234567890abcdef.csv"), []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to create csv file: %v", err)
	}

	// Create an image file
	imgContent := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
	if err := os.WriteFile(filepath.Join(rootDir, "image abc12345678.png"), imgContent, 0644); err != nil {
		t.Fatalf("failed to create image file: %v", err)
	}

	// Parse the export
	export, err := Parse(rootDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify pages
	if len(export.Pages) != 2 {
		t.Errorf("got %d pages, want 2", len(export.Pages))
	}

	// Find root page
	var rootPage *ExportedPage
	for _, p := range export.Pages {
		if p.Title == "Test Page" {
			rootPage = p
			break
		}
	}
	if rootPage == nil {
		t.Error("root page not found")
	} else {
		if rootPage.ID != "abc12345678" {
			t.Errorf("root page ID = %q, want %q", rootPage.ID, "abc12345678")
		}
		if rootPage.ParentPath != "" {
			t.Errorf("root page parent = %q, want empty", rootPage.ParentPath)
		}
	}

	// Find child page
	var childPage *ExportedPage
	for _, p := range export.Pages {
		if p.Title == "Child Page" {
			childPage = p
			break
		}
	}
	if childPage == nil {
		t.Error("child page not found")
	} else {
		if childPage.ParentPath == "" {
			t.Error("child page should have parent path")
		}
	}

	// Verify databases
	if len(export.Databases) != 1 {
		t.Errorf("got %d databases, want 1", len(export.Databases))
	} else {
		db := export.Databases[0]
		if db.Title != "My Database" {
			t.Errorf("database title = %q, want %q", db.Title, "My Database")
		}
		if len(db.Headers) != 3 {
			t.Errorf("database headers = %d, want 3", len(db.Headers))
		}
		if len(db.Rows) != 2 {
			t.Errorf("database rows = %d, want 2", len(db.Rows))
		}
	}

	// Verify attachments
	if len(export.Attachments) != 1 {
		t.Errorf("got %d attachments, want 1", len(export.Attachments))
	}
}

func TestParseMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()

	content := "# My Page\n\nSome **bold** content."
	filePath := filepath.Join(tmpDir, "My Page abc12345678.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	page, err := parseMarkdownFile(filePath, "My Page abc12345678.md")
	if err != nil {
		t.Fatalf("parseMarkdownFile failed: %v", err)
	}

	if page.Title != "My Page" {
		t.Errorf("title = %q, want %q", page.Title, "My Page")
	}
	if page.ID != "abc12345678" {
		t.Errorf("ID = %q, want %q", page.ID, "abc12345678")
	}
	if page.Content != content {
		t.Errorf("content mismatch")
	}
}

func TestParseCSVFile(t *testing.T) {
	tmpDir := t.TempDir()

	content := "Name,Status,Date\nItem 1,Active,2024-01-01\nItem 2,Done,2024-01-02"
	filePath := filepath.Join(tmpDir, "My DB abc12345678.csv")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	db, err := parseCSVFile(filePath, "My DB abc12345678.csv")
	if err != nil {
		t.Fatalf("parseCSVFile failed: %v", err)
	}

	if db.Title != "My DB" {
		t.Errorf("title = %q, want %q", db.Title, "My DB")
	}
	if db.ID != "abc12345678" {
		t.Errorf("ID = %q, want %q", db.ID, "abc12345678")
	}
	if len(db.Headers) != 3 {
		t.Errorf("headers = %d, want 3", len(db.Headers))
	}
	if len(db.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(db.Rows))
	}

	// Check row data
	if db.Rows[0]["Name"] != "Item 1" {
		t.Errorf("first row Name = %q, want %q", db.Rows[0]["Name"], "Item 1")
	}
	if db.Rows[1]["Status"] != "Done" {
		t.Errorf("second row Status = %q, want %q", db.Rows[1]["Status"], "Done")
	}
}
