package zipimport

import (
	"strings"
	"testing"
)

func TestConverter_ConvertInternalLinks(t *testing.T) {
	c := NewConverter("_attachments")
	c.RegisterPage("abc12345def67890", "Target Page", "Target Page.md")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "external link unchanged",
			input: "[Google](https://google.com)",
			want:  "[Google](https://google.com)",
		},
		{
			name:  "internal md link to wiki link",
			input: "[My Link](Target%20Page%20abc12345def67890.md)",
			want:  "[[Target Page|My Link]]",
		},
		{
			name:  "wiki link with ID stripped",
			input: "[[Target Page abc12345def67890]]",
			want:  "[[Target Page]]",
		},
		{
			name:  "wiki link with alias and ID",
			input: "[[Target Page abc12345def67890|my alias]]",
			want:  "[[Target Page|my alias]]",
		},
		{
			name:  "plain wiki link unchanged",
			input: "[[Simple Link]]",
			want:  "[[Simple Link]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.convertInternalLinks(tt.input, "")
			if got != tt.want {
				t.Errorf("convertInternalLinks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConverter_ConvertAttachmentPaths(t *testing.T) {
	c := NewConverter("_attachments")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "external image unchanged",
			input: "![alt](https://example.com/img.png)",
			want:  "![alt](https://example.com/img.png)",
		},
		{
			name:  "local image with ID",
			input: "![photo](images/photo%20abc12345.png)",
			want:  "![photo](_attachments/photo.png)",
		},
		{
			name:  "local image simple path",
			input: "![](./image.jpg)",
			want:  "![](_attachments/image.jpg)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.convertAttachmentPaths(tt.input, "")
			if got != tt.want {
				t.Errorf("convertAttachmentPaths() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertCallouts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tip callout",
			input: "> 💡 This is a tip",
			want:  "> [!tip]\n> This is a tip",
		},
		{
			name:  "regular quote unchanged",
			input: "> Just a regular quote",
			want:  "> Just a regular quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertCallouts(tt.input)
			if got != tt.want {
				t.Errorf("convertCallouts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanupNotionArtifacts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty link removed",
			input: "Some text [](http://example.com) more text",
			want:  "Some text  more text",
		},
		{
			name:  "multiple blank lines normalized",
			input: "Line 1\n\n\n\n\nLine 2",
			want:  "Line 1\n\n\nLine 2",
		},
		{
			name:  "trailing whitespace removed",
			input: "Line with spaces   \nAnother line\t",
			want:  "Line with spaces\nAnother line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanupNotionArtifacts(tt.input)
			if got != tt.want {
				t.Errorf("cleanupNotionArtifacts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertDatabaseToFrontmatter(t *testing.T) {
	row := map[string]string{
		"Name":   "Test Entry",
		"Tags":   "tag1, tag2, tag3",
		"Status": "Active",
		"Date":   "2024-01-15",
	}
	schema := []string{"Name", "Tags", "Status", "Date"}

	result := ConvertDatabaseToFrontmatter(row, schema)

	// Check it starts and ends with YAML delimiters
	if !strings.HasPrefix(result, "---\n") {
		t.Error("frontmatter should start with ---")
	}
	if !strings.HasSuffix(result, "---\n") {
		t.Error("frontmatter should end with ---")
	}

	// Check key fields are present
	if !strings.Contains(result, "name:") {
		t.Error("frontmatter should contain name field")
	}
	if !strings.Contains(result, "status: Active") {
		t.Error("frontmatter should contain status field")
	}
	if !strings.Contains(result, "date: 2024-01-15") {
		t.Error("frontmatter should contain date field")
	}

	// Tags should be converted to list
	if !strings.Contains(result, "tags:\n") {
		t.Error("tags should be formatted as list")
	}
}

func TestSanitizeYAMLKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Name", "name"},
		{"My Field", "my_field"},
		{"UPPERCASE", "uppercase"},
		{"with-dashes", "with_dashes"},
		{"Special!@#Characters", "specialcharacters"},
		{"multiple___underscores", "multiple_underscores"},
		{"__leading_trailing__", "leading_trailing"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeYAMLKey(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeYAMLKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeYAMLValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple text", "simple text"},
		{"text with: colon", "\"text with: colon\""},
		{"text with \"quotes\"", "\"text with \\\"quotes\\\"\""},
		{"-starts with dash", "\"-starts with dash\""},
		{"has #hash", "\"has #hash\""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeYAMLValue(tt.input)
			if got != tt.want {
				t.Errorf("escapeYAMLValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".PNG", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".webp", true},
		{".svg", true},
		{".pdf", false},
		{".mp4", false},
		{".doc", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := isImageFile(tt.ext)
			if got != tt.want {
				t.Errorf("isImageFile(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestConverter_ConvertMarkdown(t *testing.T) {
	c := NewConverter("_attachments")
	c.RegisterPage("abc12345678901234", "Other Page", "Other Page.md")

	input := `# My Page

Some text with a [link](Other%20Page%20abc12345678901234.md) to another page.

![image](photo%20def456789012.png)

> 💡 This is a tip callout

More content here.`

	result := c.ConvertMarkdown(input, "")

	// Check wiki link conversion
	if !strings.Contains(result, "[[Other Page|link]]") {
		t.Error("internal link should be converted to wiki link")
	}

	// Check image path conversion
	if !strings.Contains(result, "_attachments/photo.png") {
		t.Error("image path should be converted to attachment folder")
	}

	// Check callout conversion
	if !strings.Contains(result, "[!tip]") {
		t.Error("callout should be converted to Obsidian format")
	}
}
