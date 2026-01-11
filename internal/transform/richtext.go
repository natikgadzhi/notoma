package transform

import (
	"strings"

	"github.com/jomei/notionapi"
)

// RichTextToMarkdown converts Notion rich text array to markdown string.
func RichTextToMarkdown(richText []notionapi.RichText) string {
	var sb strings.Builder
	for _, rt := range richText {
		sb.WriteString(richTextSegmentToMarkdown(rt))
	}
	return sb.String()
}

// richTextSegmentToMarkdown converts a single rich text segment to markdown.
func richTextSegmentToMarkdown(rt notionapi.RichText) string {
	text := rt.PlainText

	// Handle mentions (check if Mention field is populated)
	if rt.Mention != nil {
		text = convertMention(rt.Mention)
	}

	// Handle equations (check if Equation field is populated)
	if rt.Equation != nil {
		return "$" + rt.Equation.Expression + "$"
	}

	// Apply annotations
	if rt.Annotations != nil {
		text = applyAnnotations(text, rt.Annotations)
	}

	// Handle links
	if rt.Href != "" {
		// Check if it's an internal Notion link
		if strings.Contains(rt.Href, "notion.so") || strings.HasPrefix(rt.Href, "/") {
			// For internal links, we'll need to resolve them to wiki-links
			// For now, extract page ID and create a placeholder wiki-link
			text = convertNotionLink(rt.PlainText, rt.Href)
		} else {
			text = "[" + text + "](" + rt.Href + ")"
		}
	}

	return text
}

// convertMention converts a Notion mention to markdown.
func convertMention(mention *notionapi.Mention) string {
	if mention == nil {
		return ""
	}
	switch mention.Type {
	case notionapi.MentionTypePage:
		if mention.Page != nil {
			// TODO: Resolve page title from ID
			return "[[" + string(mention.Page.ID) + "]]"
		}
	case notionapi.MentionTypeDatabase:
		if mention.Database != nil {
			// TODO: Resolve database title from ID
			return "[[" + string(mention.Database.ID) + "]]"
		}
	case notionapi.MentionTypeUser:
		if mention.User != nil {
			return "@" + mention.User.Name
		}
	case notionapi.MentionTypeDate:
		if mention.Date != nil {
			return formatMentionDate(mention.Date)
		}
	case notionapi.MentionTypeTemplateMention:
		if mention.TemplateMention != nil {
			return mention.TemplateMention.TemplateMentionDate
		}
	}
	return ""
}

// formatMentionDate formats a date mention to ISO string.
func formatMentionDate(date *notionapi.DateObject) string {
	if date == nil || date.Start == nil {
		return ""
	}

	start := date.Start.String()
	if date.End != nil {
		return start + " → " + date.End.String()
	}
	return start
}

// applyAnnotations applies rich text annotations to text.
func applyAnnotations(text string, ann *notionapi.Annotations) string {
	if text == "" {
		return text
	}

	// Apply code first (innermost)
	if ann.Code {
		text = "`" + text + "`"
	}

	// Apply strikethrough
	if ann.Strikethrough {
		text = "~~" + text + "~~"
	}

	// Apply italic
	if ann.Italic {
		text = "*" + text + "*"
	}

	// Apply bold
	if ann.Bold {
		text = "**" + text + "**"
	}

	// Apply underline (using HTML since markdown doesn't support it)
	if ann.Underline {
		text = "<u>" + text + "</u>"
	}

	// Handle highlight colors (background colors)
	if isHighlightColor(ann.Color) {
		text = "==" + text + "=="
	}

	return text
}

// isHighlightColor checks if the color is a background/highlight color.
func isHighlightColor(color notionapi.Color) bool {
	switch color {
	case "yellow_background", "blue_background", "green_background",
		"orange_background", "pink_background", "purple_background",
		"red_background", "gray_background", "brown_background":
		return true
	}
	return false
}

// convertNotionLink converts an internal Notion link to a wiki-link or external link.
func convertNotionLink(text, href string) string {
	// Parse the Notion URL to extract page ID
	// For now, just create a wiki-link with the text
	// The actual page title resolution will happen during sync when we have page mappings

	// Clean the text for wiki-link
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		// Try to extract something meaningful from the URL
		cleanText = "link"
	}

	// Return as wiki-link placeholder - will be resolved during sync
	return "[[" + cleanText + "]]"
}

// RichTextToPlain extracts plain text from rich text array.
func RichTextToPlain(richText []notionapi.RichText) string {
	var sb strings.Builder
	for _, rt := range richText {
		sb.WriteString(rt.PlainText)
	}
	return sb.String()
}
