package transform

import (
	"context"
	"strings"
	"testing"

	"github.com/jomei/notionapi"
)

// mockFetcher implements BlockFetcher for testing.
type mockFetcher struct {
	children map[string][]notionapi.Block
}

func (m *mockFetcher) GetBlockChildren(_ context.Context, blockID string) ([]notionapi.Block, error) {
	if m.children == nil {
		return nil, nil
	}
	return m.children[blockID], nil
}

func newRichText(text string) []notionapi.RichText {
	return []notionapi.RichText{{PlainText: text}}
}

func TestParagraphBlock(t *testing.T) {
	block := &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: "block",
			ID:     "test-id",
			Type:   notionapi.BlockTypeParagraph,
		},
		Paragraph: notionapi.Paragraph{
			RichText: newRichText("Hello, world!"),
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello, world!\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestHeadingBlocks(t *testing.T) {
	tests := []struct {
		name     string
		block    notionapi.Block
		expected string
	}{
		{
			name: "heading 1",
			block: &notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading1},
				Heading1:   notionapi.Heading{RichText: newRichText("Heading 1")},
			},
			expected: "# Heading 1\n\n",
		},
		{
			name: "heading 2",
			block: &notionapi.Heading2Block{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading2},
				Heading2:   notionapi.Heading{RichText: newRichText("Heading 2")},
			},
			expected: "## Heading 2\n\n",
		},
		{
			name: "heading 3",
			block: &notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading3},
				Heading3:   notionapi.Heading{RichText: newRichText("Heading 3")},
			},
			expected: "### Heading 3\n\n",
		},
		{
			name: "toggleable heading",
			block: &notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading1},
				Heading1: notionapi.Heading{
					RichText:     newRichText("Toggleable"),
					IsToggleable: true,
				},
			},
			expected: "> [!faq]- Toggleable\n\n",
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.BlocksToMarkdown([]notionapi.Block{tt.block})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestListBlocks(t *testing.T) {
	tests := []struct {
		name     string
		block    notionapi.Block
		expected string
	}{
		{
			name: "bulleted list item",
			block: &notionapi.BulletedListItemBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
				BulletedListItem: notionapi.ListItem{
					RichText: newRichText("Item 1"),
				},
			},
			expected: "- Item 1\n",
		},
		{
			name: "numbered list item",
			block: &notionapi.NumberedListItemBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeNumberedListItem},
				NumberedListItem: notionapi.ListItem{
					RichText: newRichText("Item 1"),
				},
			},
			expected: "1. Item 1\n",
		},
		{
			name: "unchecked todo",
			block: &notionapi.ToDoBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeToDo},
				ToDo: notionapi.ToDo{
					RichText: newRichText("Task"),
					Checked:  false,
				},
			},
			expected: "- [ ] Task\n",
		},
		{
			name: "checked todo",
			block: &notionapi.ToDoBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeToDo},
				ToDo: notionapi.ToDo{
					RichText: newRichText("Done task"),
					Checked:  true,
				},
			},
			expected: "- [x] Done task\n",
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.BlocksToMarkdown([]notionapi.Block{tt.block})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		block    *notionapi.CodeBlock
		expected string
	}{
		{
			name: "code with language",
			block: &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeCode},
				Code: notionapi.Code{
					RichText: newRichText("fmt.Println(\"Hello\")"),
					Language: "go",
				},
			},
			expected: "```go\nfmt.Println(\"Hello\")\n```\n\n",
		},
		{
			name: "code without language",
			block: &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeCode},
				Code: notionapi.Code{
					RichText: newRichText("echo hello"),
					Language: "plain text",
				},
			},
			expected: "```\necho hello\n```\n\n",
		},
		{
			name: "code with caption",
			block: &notionapi.CodeBlock{
				BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeCode},
				Code: notionapi.Code{
					RichText: newRichText("const x = 1"),
					Language: "javascript",
					Caption:  newRichText("Example code"),
				},
			},
			expected: "```javascript\nconst x = 1\n```\n\n*Example code*\n\n",
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.BlocksToMarkdown([]notionapi.Block{tt.block})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestQuoteBlock(t *testing.T) {
	block := &notionapi.QuoteBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeQuote},
		Quote: notionapi.Quote{
			RichText: newRichText("This is a quote."),
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "> This is a quote.\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestCalloutBlock(t *testing.T) {
	emoji := notionapi.Emoji("💡")
	block := &notionapi.CalloutBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeCallout},
		Callout: notionapi.Callout{
			RichText: newRichText("This is a tip."),
			Icon: &notionapi.Icon{
				Type:  "emoji",
				Emoji: &emoji,
			},
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[!tip]") {
		t.Errorf("expected callout type 'tip', got %q", result)
	}
	if !strings.Contains(result, "This is a tip.") {
		t.Errorf("expected callout content, got %q", result)
	}
}

func TestDividerBlock(t *testing.T) {
	block := &notionapi.DividerBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeDivider},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "---\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestEquationBlock(t *testing.T) {
	block := &notionapi.EquationBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeEquation},
		Equation: notionapi.Equation{
			Expression: "E = mc^2",
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "$$\nE = mc^2\n$$\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestChildPageBlock(t *testing.T) {
	block := &notionapi.ChildPageBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeChildPage},
		ChildPage: struct {
			Title string `json:"title"`
		}{
			Title: "My Child Page",
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "[[My Child Page]]\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestChildDatabaseBlock(t *testing.T) {
	block := &notionapi.ChildDatabaseBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeChildDatabase},
		ChildDatabase: struct {
			Title string `json:"title"`
		}{
			Title: "My Database",
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "[[My Database]]\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestImageBlock(t *testing.T) {
	block := &notionapi.ImageBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeImage},
		Image: notionapi.Image{
			External: &notionapi.FileObject{
				URL: "https://example.com/image.png",
			},
			Caption: newRichText("My image"),
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "![My image](https://example.com/image.png)\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestBookmarkBlock(t *testing.T) {
	block := &notionapi.BookmarkBlock{
		BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeBookmark},
		Bookmark: notionapi.Bookmark{
			URL:     "https://example.com",
			Caption: newRichText("Example Site"),
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "[Example Site](https://example.com)\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestToggleBlock(t *testing.T) {
	fetcher := &mockFetcher{
		children: map[string][]notionapi.Block{
			"toggle-id": {
				&notionapi.ParagraphBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeParagraph},
					Paragraph:  notionapi.Paragraph{RichText: newRichText("Hidden content")},
				},
			},
		},
	}

	block := &notionapi.ToggleBlock{
		BasicBlock: notionapi.BasicBlock{
			Object:      "block",
			ID:          "toggle-id",
			Type:        notionapi.BlockTypeToggle,
			HasChildren: true,
		},
		Toggle: notionapi.Toggle{
			RichText: newRichText("Click to expand"),
		},
	}

	transformer := NewTransformer(context.Background(), fetcher)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[!faq]- Click to expand") {
		t.Errorf("expected toggle title, got %q", result)
	}
	if !strings.Contains(result, "Hidden content") {
		t.Errorf("expected toggle content, got %q", result)
	}
}

func TestTableBlock(t *testing.T) {
	fetcher := &mockFetcher{
		children: map[string][]notionapi.Block{
			"table-id": {
				&notionapi.TableRowBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableRowBlock},
					TableRow: notionapi.TableRow{
						Cells: [][]notionapi.RichText{
							newRichText("Header 1"),
							newRichText("Header 2"),
						},
					},
				},
				&notionapi.TableRowBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableRowBlock},
					TableRow: notionapi.TableRow{
						Cells: [][]notionapi.RichText{
							newRichText("Cell 1"),
							newRichText("Cell 2"),
						},
					},
				},
			},
		},
	}

	block := &notionapi.TableBlock{
		BasicBlock: notionapi.BasicBlock{
			Object:      "block",
			ID:          "table-id",
			Type:        notionapi.BlockTypeTableBlock,
			HasChildren: true,
		},
		Table: notionapi.Table{
			TableWidth:      2,
			HasColumnHeader: true,
		},
	}

	transformer := NewTransformer(context.Background(), fetcher)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check table structure
	if !strings.Contains(result, "| Header 1 | Header 2 |") {
		t.Errorf("expected table header, got %q", result)
	}
	if !strings.Contains(result, "| --- | --- |") {
		t.Errorf("expected table separator, got %q", result)
	}
	if !strings.Contains(result, "| Cell 1 | Cell 2 |") {
		t.Errorf("expected table row, got %q", result)
	}
}

func TestNestedListItems(t *testing.T) {
	fetcher := &mockFetcher{
		children: map[string][]notionapi.Block{
			"parent-id": {
				&notionapi.BulletedListItemBlock{
					BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
					BulletedListItem: notionapi.ListItem{
						RichText: newRichText("Nested item"),
					},
				},
			},
		},
	}

	block := &notionapi.BulletedListItemBlock{
		BasicBlock: notionapi.BasicBlock{
			Object:      "block",
			ID:          "parent-id",
			Type:        notionapi.BlockTypeBulletedListItem,
			HasChildren: true,
		},
		BulletedListItem: notionapi.ListItem{
			RichText: newRichText("Parent item"),
		},
	}

	transformer := NewTransformer(context.Background(), fetcher)
	result, err := transformer.BlocksToMarkdown([]notionapi.Block{block})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "- Parent item") {
		t.Errorf("expected parent item, got %q", result)
	}
	if !strings.Contains(result, "Nested item") {
		t.Errorf("expected nested item, got %q", result)
	}
}

func TestSkippedBlocks(t *testing.T) {
	blocks := []notionapi.Block{
		&notionapi.BreadcrumbBlock{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeBreadcrumb},
		},
		&notionapi.TableOfContentsBlock{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTableOfContents},
		},
		&notionapi.TemplateBlock{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeTemplate},
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// These blocks should produce empty or minimal output
	if strings.Contains(result, "breadcrumb") || strings.Contains(result, "table_of_contents") {
		t.Errorf("expected skipped blocks to not appear in output, got %q", result)
	}
}

func TestMixedBlockTypes(t *testing.T) {
	blocks := []notionapi.Block{
		&notionapi.Heading1Block{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeHeading1},
			Heading1:   notionapi.Heading{RichText: newRichText("Title")},
		},
		&notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeParagraph},
			Paragraph:  notionapi.Paragraph{RichText: newRichText("Some text.")},
		},
		&notionapi.BulletedListItemBlock{
			BasicBlock:       notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
			BulletedListItem: notionapi.ListItem{RichText: newRichText("Item 1")},
		},
		&notionapi.BulletedListItemBlock{
			BasicBlock:       notionapi.BasicBlock{Type: notionapi.BlockTypeBulletedListItem},
			BulletedListItem: notionapi.ListItem{RichText: newRichText("Item 2")},
		},
		&notionapi.DividerBlock{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeDivider},
		},
	}

	transformer := NewTransformer(context.Background(), nil)
	result, err := transformer.BlocksToMarkdown(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that all blocks are present
	expectations := []string{
		"# Title",
		"Some text.",
		"- Item 1",
		"- Item 2",
		"---",
	}

	for _, exp := range expectations {
		if !strings.Contains(result, exp) {
			t.Errorf("expected %q in output, got %q", exp, result)
		}
	}
}

func TestBlocksToMarkdownSimple(t *testing.T) {
	blocks := []notionapi.Block{
		&notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{Type: notionapi.BlockTypeParagraph},
			Paragraph:  notionapi.Paragraph{RichText: newRichText("Test")},
		},
	}

	result, err := BlocksToMarkdownSimple(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Test\n\n"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
