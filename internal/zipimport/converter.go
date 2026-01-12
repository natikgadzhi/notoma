package zipimport

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// Converter transforms Notion export content to Obsidian format.
type Converter struct {
	// pageTitles maps Notion IDs to page titles for link resolution
	pageTitles map[string]string
	// pageFiles maps Notion IDs to output filenames
	pageFiles map[string]string
	// attachmentFolder is the relative path for attachments in the vault
	attachmentFolder string
}

// NewConverter creates a new Notion to Obsidian converter.
func NewConverter(attachmentFolder string) *Converter {
	return &Converter{
		pageTitles:       make(map[string]string),
		pageFiles:        make(map[string]string),
		attachmentFolder: attachmentFolder,
	}
}

// RegisterPage registers a page for link resolution.
func (c *Converter) RegisterPage(id, title, filename string) {
	if id != "" {
		c.pageTitles[id] = title
		c.pageFiles[id] = filename
	}
	// Also register by title for non-ID links
	c.pageTitles[title] = title
	c.pageFiles[title] = filename
}

// ConvertMarkdown transforms Notion-exported markdown to Obsidian format.
func (c *Converter) ConvertMarkdown(content string, pagePath string) string {
	// Convert internal links
	content = c.convertInternalLinks(content, pagePath)

	// Convert attachment paths
	content = c.convertAttachmentPaths(content, pagePath)

	// Convert callouts (Notion uses different syntax)
	content = convertCallouts(content)

	// Clean up Notion-specific artifacts
	content = cleanupNotionArtifacts(content)

	return content
}

// internalLinkRegex matches markdown links that might be internal Notion links.
// Matches: [text](path) or [text](path "title")
var internalLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// wikiLinkRegex matches wiki-style links: [[text]]
var wikiLinkRegex = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// convertInternalLinks converts Notion internal links to Obsidian wiki-links.
func (c *Converter) convertInternalLinks(content, pagePath string) string {
	// Convert markdown links to internal pages
	content = internalLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := internalLinkRegex.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		text := matches[1]
		href := matches[2]

		// Skip external links
		if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
			return match
		}

		// Decode URL-encoded paths
		decodedHref, err := url.PathUnescape(href)
		if err != nil {
			decodedHref = href
		}

		// Check if it's a link to another markdown file
		if strings.HasSuffix(strings.ToLower(decodedHref), ".md") {
			// Extract target page name
			targetFile := filepath.Base(decodedHref)
			targetName := strings.TrimSuffix(targetFile, filepath.Ext(targetFile))
			targetName = StripNotionID(targetName)

			// Use link text if meaningful, otherwise use target name
			linkText := text
			if linkText == "" || linkText == targetFile {
				linkText = targetName
			}

			return "[[" + targetName + "|" + linkText + "]]"
		}

		// Check if it's a relative path to an attachment
		if isAttachmentPath(decodedHref) {
			return c.convertAttachmentLink(text, decodedHref, pagePath)
		}

		return match
	})

	// Clean up wiki-links that have Notion IDs
	content = wikiLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := wikiLinkRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		linkContent := matches[1]

		// Check for alias syntax [[target|alias]]
		parts := strings.SplitN(linkContent, "|", 2)
		target := parts[0]
		alias := ""
		if len(parts) > 1 {
			alias = parts[1]
		}

		// Strip Notion ID from target
		cleanTarget := StripNotionID(target)
		if cleanTarget == "" {
			cleanTarget = target
		}

		if alias != "" {
			return "[[" + cleanTarget + "|" + alias + "]]"
		}
		return "[[" + cleanTarget + "]]"
	})

	return content
}

// isAttachmentPath checks if a path points to an attachment.
func isAttachmentPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return isAttachment(ext)
}

// convertAttachmentLink converts an attachment link to use the vault attachment folder.
func (c *Converter) convertAttachmentLink(text, href, pagePath string) string {
	// Get just the filename
	filename := filepath.Base(href)

	// Decode URL encoding
	decodedFilename, err := url.PathUnescape(filename)
	if err != nil {
		decodedFilename = filename
	}

	// Strip Notion ID from filename if present
	ext := filepath.Ext(decodedFilename)
	baseName := strings.TrimSuffix(decodedFilename, ext)
	cleanName := StripNotionID(baseName) + ext

	// Build path relative to vault root
	attachmentPath := c.attachmentFolder + "/" + cleanName

	// Determine if it's an image (use ![]() syntax) or file (use []() syntax)
	if isImageFile(ext) {
		if text != "" {
			return "![" + text + "](" + attachmentPath + ")"
		}
		return "![](" + attachmentPath + ")"
	}

	linkText := text
	if linkText == "" {
		linkText = cleanName
	}
	return "[" + linkText + "](" + attachmentPath + ")"
}

// convertAttachmentPaths updates all attachment references in the content.
func (c *Converter) convertAttachmentPaths(content, pagePath string) string {
	// Match image syntax: ![alt](path)
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	content = imageRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := imageRegex.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		alt := matches[1]
		src := matches[2]

		// Skip external URLs
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			return match
		}

		// Decode and clean the path
		decoded, err := url.PathUnescape(src)
		if err != nil {
			decoded = src
		}

		filename := filepath.Base(decoded)
		ext := filepath.Ext(filename)
		baseName := strings.TrimSuffix(filename, ext)
		cleanName := StripNotionID(baseName) + ext

		attachmentPath := c.attachmentFolder + "/" + cleanName
		return "![" + alt + "](" + attachmentPath + ")"
	})

	return content
}

// isImageFile checks if an extension is for an image file.
func isImageFile(ext string) bool {
	imageExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".webp": true, ".svg": true, ".bmp": true, ".ico": true,
	}
	return imageExts[strings.ToLower(ext)]
}

// notionCalloutRegex matches Notion's callout/admonition syntax.
// Notion exports callouts as blockquotes with emoji prefixes.
// Uses Unicode Symbol_Other category (\p{So}) which covers most emoji symbols.
var notionCalloutRegex = regexp.MustCompile(`(?m)^>\s*(\p{So}|[⚠️ℹ️✅❌❗❓⭐⚙️])\s*(.*)$`)

// convertCallouts converts Notion callout syntax to Obsidian callouts.
func convertCallouts(content string) string {
	// Map common emojis to Obsidian callout types
	emojiToCallout := map[string]string{
		"💡":  "tip",
		"⚠️": "warning",
		"❗":  "important",
		"ℹ️": "info",
		"📝":  "note",
		"✅":  "success",
		"❌":  "failure",
		"🔥":  "danger",
		"❓":  "question",
		"📌":  "note",
		"🎯":  "important",
		"💬":  "quote",
		"📖":  "note",
		"🚨":  "danger",
		"⭐":  "tip",
	}

	lines := strings.Split(content, "\n")
	var result []string
	inCallout := false
	currentCalloutType := "note"

	for i, line := range lines {
		if matches := notionCalloutRegex.FindStringSubmatch(line); len(matches) == 3 {
			emoji := matches[1]
			text := matches[2]

			calloutType := "note"
			if ct, ok := emojiToCallout[emoji]; ok {
				calloutType = ct
			}

			if !inCallout {
				// Start new callout
				result = append(result, fmt.Sprintf("> [!%s]", calloutType))
				inCallout = true
				currentCalloutType = calloutType
			}
			result = append(result, "> "+text)
		} else if inCallout && strings.HasPrefix(line, "> ") {
			// Continue callout
			result = append(result, line)
		} else {
			// End callout if we were in one
			if inCallout {
				inCallout = false
				_ = currentCalloutType // Used for tracking
			}
			result = append(result, line)
		}

		// Look ahead for callout continuation
		if inCallout && i+1 < len(lines) && !strings.HasPrefix(lines[i+1], ">") {
			inCallout = false
		}
	}

	return strings.Join(result, "\n")
}

// cleanupNotionArtifacts removes Notion-specific artifacts from the content.
func cleanupNotionArtifacts(content string) string {
	// Remove Notion page IDs from headings
	headingRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s+[a-f0-9]{32}\s*$`)
	content = headingRegex.ReplaceAllString(content, "$1 $2")

	// Remove empty links that Notion sometimes exports
	emptyLinkRegex := regexp.MustCompile(`\[\]\([^)]+\)`)
	content = emptyLinkRegex.ReplaceAllString(content, "")

	// Normalize multiple blank lines to max 2
	multiBlankRegex := regexp.MustCompile(`\n{4,}`)
	content = multiBlankRegex.ReplaceAllString(content, "\n\n\n")

	// Remove trailing whitespace from lines
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")

	return content
}

// ConvertDatabaseToFrontmatter converts a CSV row to YAML frontmatter.
func ConvertDatabaseToFrontmatter(row map[string]string, schema []string) string {
	var sb strings.Builder
	sb.WriteString("---\n")

	for _, header := range schema {
		value := row[header]
		if value == "" {
			continue
		}

		// Sanitize header for YAML key
		key := sanitizeYAMLKey(header)

		// Handle different value types
		if strings.Contains(value, ",") && !strings.Contains(value, "\n") {
			// Might be a multi-value field (tags, multi-select)
			values := strings.Split(value, ",")
			sb.WriteString(key + ":\n")
			for _, v := range values {
				v = strings.TrimSpace(v)
				if v != "" {
					sb.WriteString("  - " + escapeYAMLValue(v) + "\n")
				}
			}
		} else {
			// Single value
			sb.WriteString(key + ": " + escapeYAMLValue(value) + "\n")
		}
	}

	sb.WriteString("---\n")
	return sb.String()
}

// sanitizeYAMLKey converts a string to a valid YAML key.
func sanitizeYAMLKey(s string) string {
	// Convert to lowercase and replace spaces/special chars with underscores
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		if r == ' ' || r == '-' {
			return '_'
		}
		return -1
	}, s)

	// Remove consecutive underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	return strings.Trim(s, "_")
}

// escapeYAMLValue escapes a value for safe inclusion in YAML.
func escapeYAMLValue(s string) string {
	// Check if quoting is needed
	needsQuotes := false

	// Check for special characters that require quoting
	specialChars := []string{":", "#", "[", "]", "{", "}", ",", "&", "*", "!", "|", ">", "'", "\"", "%", "@", "`"}
	for _, char := range specialChars {
		if strings.Contains(s, char) {
			needsQuotes = true
			break
		}
	}

	// Check if it starts with special characters
	if len(s) > 0 {
		firstChar := s[0]
		if firstChar == '-' || firstChar == '?' || firstChar == ' ' {
			needsQuotes = true
		}
	}

	// Check for newlines
	if strings.Contains(s, "\n") {
		// Use literal block style for multi-line
		lines := strings.Split(s, "\n")
		var sb strings.Builder
		sb.WriteString("|\n")
		for _, line := range lines {
			sb.WriteString("  " + line + "\n")
		}
		return sb.String()
	}

	if needsQuotes {
		// Escape double quotes and wrap in double quotes
		s = strings.ReplaceAll(s, "\"", "\\\"")
		return "\"" + s + "\""
	}

	return s
}
