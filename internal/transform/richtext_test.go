package transform

import (
	"testing"

	"github.com/jomei/notionapi"
)

func TestRichTextToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		richText []notionapi.RichText
		want     string
	}{
		{
			name:     "empty",
			richText: nil,
			want:     "",
		},
		{
			name: "plain text",
			richText: []notionapi.RichText{
				{PlainText: "Hello, world!"},
			},
			want: "Hello, world!",
		},
		{
			name: "multiple segments",
			richText: []notionapi.RichText{
				{PlainText: "Hello, "},
				{PlainText: "world!"},
			},
			want: "Hello, world!",
		},
		{
			name: "bold text",
			richText: []notionapi.RichText{
				{
					PlainText:   "bold",
					Annotations: &notionapi.Annotations{Bold: true},
				},
			},
			want: "**bold**",
		},
		{
			name: "italic text",
			richText: []notionapi.RichText{
				{
					PlainText:   "italic",
					Annotations: &notionapi.Annotations{Italic: true},
				},
			},
			want: "*italic*",
		},
		{
			name: "bold and italic",
			richText: []notionapi.RichText{
				{
					PlainText:   "both",
					Annotations: &notionapi.Annotations{Bold: true, Italic: true},
				},
			},
			want: "***both***",
		},
		{
			name: "strikethrough",
			richText: []notionapi.RichText{
				{
					PlainText:   "deleted",
					Annotations: &notionapi.Annotations{Strikethrough: true},
				},
			},
			want: "~~deleted~~",
		},
		{
			name: "code",
			richText: []notionapi.RichText{
				{
					PlainText:   "code",
					Annotations: &notionapi.Annotations{Code: true},
				},
			},
			want: "`code`",
		},
		{
			name: "underline",
			richText: []notionapi.RichText{
				{
					PlainText:   "underlined",
					Annotations: &notionapi.Annotations{Underline: true},
				},
			},
			want: "<u>underlined</u>",
		},
		{
			name: "highlight",
			richText: []notionapi.RichText{
				{
					PlainText:   "highlighted",
					Annotations: &notionapi.Annotations{Color: "yellow_background"},
				},
			},
			want: "==highlighted==",
		},
		{
			name: "external link",
			richText: []notionapi.RichText{
				{
					PlainText: "link",
					Href:      "https://example.com",
				},
			},
			want: "[link](https://example.com)",
		},
		{
			name: "external link with empty title",
			richText: []notionapi.RichText{
				{
					PlainText: "",
					Href:      "https://docs.google.com/document/d/123",
				},
			},
			want: "https://docs.google.com/document/d/123",
		},
		{
			name: "mixed formatting",
			richText: []notionapi.RichText{
				{PlainText: "This is "},
				{
					PlainText:   "bold",
					Annotations: &notionapi.Annotations{Bold: true},
				},
				{PlainText: " and "},
				{
					PlainText:   "italic",
					Annotations: &notionapi.Annotations{Italic: true},
				},
				{PlainText: " text."},
			},
			want: "This is **bold** and *italic* text.",
		},
		{
			name: "all annotations",
			richText: []notionapi.RichText{
				{
					PlainText: "all",
					Annotations: &notionapi.Annotations{
						Bold:          true,
						Italic:        true,
						Strikethrough: true,
						Code:          true,
					},
				},
			},
			want: "***~~`all`~~***", // code -> strikethrough -> italic -> bold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RichTextToMarkdown(tt.richText)
			if got != tt.want {
				t.Errorf("RichTextToMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRichTextToPlain(t *testing.T) {
	tests := []struct {
		name     string
		richText []notionapi.RichText
		want     string
	}{
		{
			name:     "empty",
			richText: nil,
			want:     "",
		},
		{
			name: "plain text",
			richText: []notionapi.RichText{
				{PlainText: "Hello"},
			},
			want: "Hello",
		},
		{
			name: "multiple segments",
			richText: []notionapi.RichText{
				{PlainText: "Hello, "},
				{PlainText: "world!"},
			},
			want: "Hello, world!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RichTextToPlain(tt.richText)
			if got != tt.want {
				t.Errorf("RichTextToPlain() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsHighlightColor(t *testing.T) {
	tests := []struct {
		color notionapi.Color
		want  bool
	}{
		{"yellow_background", true},
		{"blue_background", true},
		{"red_background", true},
		{"yellow", false},
		{"blue", false},
		{"default", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.color), func(t *testing.T) {
			got := isHighlightColor(tt.color)
			if got != tt.want {
				t.Errorf("isHighlightColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

func TestConvertMention(t *testing.T) {
	pageID := notionapi.ObjectID("abc123")
	dbID := notionapi.ObjectID("def456")

	tests := []struct {
		name    string
		mention *notionapi.Mention
		want    string
	}{
		{
			name:    "nil mention",
			mention: nil,
			want:    "",
		},
		{
			name: "page mention",
			mention: &notionapi.Mention{
				Type: notionapi.MentionTypePage,
				Page: &notionapi.PageMention{ID: pageID},
			},
			want: "[[abc123]]",
		},
		{
			name: "database mention",
			mention: &notionapi.Mention{
				Type:     notionapi.MentionTypeDatabase,
				Database: &notionapi.DatabaseMention{ID: dbID},
			},
			want: "[[def456]]",
		},
		{
			name: "user mention",
			mention: &notionapi.Mention{
				Type: notionapi.MentionTypeUser,
				User: &notionapi.User{Name: "John Doe"},
			},
			want: "@John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMention(tt.mention)
			if got != tt.want {
				t.Errorf("convertMention() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmojiToCalloutType(t *testing.T) {
	tests := []struct {
		emoji string
		want  string
	}{
		{"💡", "tip"},
		{"⚠️", "warning"},
		{"❗", "important"},
		{"❓", "question"},
		{"✅", "success"},
		{"❌", "failure"},
		{"ℹ️", "info"},
		{"🔥", "danger"},
		{"🚨", "danger"},
		{"⭐", "tip"},
		{"📘", "note"},
		{"📗", "tip"},
		{"📙", "example"},
		{"📕", "warning"},
		{"🎯", "important"},
		{"🔖", "quote"},
		{"unknown", "note"},
		{"", "note"},
	}

	for _, tt := range tests {
		t.Run(tt.emoji, func(t *testing.T) {
			got := emojiToCalloutType(tt.emoji)
			if got != tt.want {
				t.Errorf("emojiToCalloutType(%q) = %q, want %q", tt.emoji, got, tt.want)
			}
		})
	}
}
