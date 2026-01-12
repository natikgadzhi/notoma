package main

import (
	"testing"

	"github.com/jomei/notionapi"
)

func TestExtractChildPages(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []notionapi.Block
		expected []childPageInfo
	}{
		{
			name:     "empty blocks",
			blocks:   []notionapi.Block{},
			expected: nil,
		},
		{
			name: "no child pages",
			blocks: []notionapi.Block{
				&notionapi.ParagraphBlock{
					BasicBlock: notionapi.BasicBlock{
						Object: "block",
						ID:     "para-1",
						Type:   notionapi.BlockTypeParagraph,
					},
				},
				&notionapi.Heading1Block{
					BasicBlock: notionapi.BasicBlock{
						Object: "block",
						ID:     "h1-1",
						Type:   notionapi.BlockTypeHeading1,
					},
				},
			},
			expected: nil,
		},
		{
			name: "single child page",
			blocks: []notionapi.Block{
				&notionapi.ParagraphBlock{
					BasicBlock: notionapi.BasicBlock{ID: "para-1", Type: notionapi.BlockTypeParagraph},
				},
				&notionapi.ChildPageBlock{
					BasicBlock: notionapi.BasicBlock{
						Object: "block",
						ID:     "child-page-1",
						Type:   notionapi.BlockTypeChildPage,
					},
					ChildPage: struct {
						Title string `json:"title"`
					}{
						Title: "My Child Page",
					},
				},
			},
			expected: []childPageInfo{
				{id: "child-page-1", title: "My Child Page"},
			},
		},
		{
			name: "multiple child pages",
			blocks: []notionapi.Block{
				&notionapi.ChildPageBlock{
					BasicBlock: notionapi.BasicBlock{
						ID:   "child-1",
						Type: notionapi.BlockTypeChildPage,
					},
					ChildPage: struct {
						Title string `json:"title"`
					}{
						Title: "First Child",
					},
				},
				&notionapi.ParagraphBlock{
					BasicBlock: notionapi.BasicBlock{ID: "para-1", Type: notionapi.BlockTypeParagraph},
				},
				&notionapi.ChildPageBlock{
					BasicBlock: notionapi.BasicBlock{
						ID:   "child-2",
						Type: notionapi.BlockTypeChildPage,
					},
					ChildPage: struct {
						Title string `json:"title"`
					}{
						Title: "Second Child",
					},
				},
				&notionapi.ChildPageBlock{
					BasicBlock: notionapi.BasicBlock{
						ID:   "child-3",
						Type: notionapi.BlockTypeChildPage,
					},
					ChildPage: struct {
						Title string `json:"title"`
					}{
						Title: "Third Child",
					},
				},
			},
			expected: []childPageInfo{
				{id: "child-1", title: "First Child"},
				{id: "child-2", title: "Second Child"},
				{id: "child-3", title: "Third Child"},
			},
		},
		{
			name: "child pages mixed with child databases",
			blocks: []notionapi.Block{
				&notionapi.ChildPageBlock{
					BasicBlock: notionapi.BasicBlock{
						ID:   "page-1",
						Type: notionapi.BlockTypeChildPage,
					},
					ChildPage: struct {
						Title string `json:"title"`
					}{
						Title: "Child Page",
					},
				},
				&notionapi.ChildDatabaseBlock{
					BasicBlock: notionapi.BasicBlock{
						ID:   "db-1",
						Type: notionapi.BlockTypeChildDatabase,
					},
					ChildDatabase: struct {
						Title string `json:"title"`
					}{
						Title: "Child Database",
					},
				},
			},
			expected: []childPageInfo{
				{id: "page-1", title: "Child Page"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractChildPages(tt.blocks)

			if len(result) != len(tt.expected) {
				t.Fatalf("got %d child pages, want %d", len(result), len(tt.expected))
			}

			for i, got := range result {
				want := tt.expected[i]
				if got.id != want.id {
					t.Errorf("child[%d].id = %q, want %q", i, got.id, want.id)
				}
				if got.title != want.title {
					t.Errorf("child[%d].title = %q, want %q", i, got.title, want.title)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "My Page",
			expected: "My Page",
		},
		{
			name:     "name with slashes",
			input:    "Parent/Child",
			expected: "Parent-Child",
		},
		{
			name:     "name with special chars",
			input:    "File: Test?",
			expected: "File- Test",
		},
		{
			name:     "name with newlines",
			input:    "Line1\nLine2",
			expected: "Line1 Line2",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  Spaced  ",
			expected: "Spaced",
		},
		{
			name:     "very long name",
			input:    string(make([]byte, 250)),
			expected: string(make([]byte, 200)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsUUIDPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid UUID prefix",
			input:    "1e567c00...",
			expected: true,
		},
		{
			name:     "valid hex only",
			input:    "abcdef12",
			expected: true,
		},
		{
			name:     "too short",
			input:    "abc",
			expected: false,
		},
		{
			name:     "not hex",
			input:    "MyPageNa",
			expected: false,
		},
		{
			name:     "mixed case hex fails (uppercase)",
			input:    "ABCDEF12",
			expected: false,
		},
		{
			name:     "real page title",
			input:    "My Notes Page",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUUIDPrefix(tt.input)
			if result != tt.expected {
				t.Errorf("isUUIDPrefix(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
