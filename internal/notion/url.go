// Package notion provides a wrapper around the Notion API client
// with rate limiting and URL parsing utilities.
package notion

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ParsedURL contains the extracted ID from a Notion URL.
type ParsedURL struct {
	// ID is the 32-character hex ID formatted as UUID (8-4-4-4-12).
	ID string
	// RawID is the original 32-character hex ID without formatting.
	RawID string
}

// hexIDPattern matches a 32-character hex string.
var hexIDPattern = regexp.MustCompile(`[a-f0-9]{32}`)

// ParseURL extracts the page or database ID from a Notion share URL.
//
// Supported formats:
//   - https://www.notion.so/{workspace}/{title}-{id}
//   - https://www.notion.so/{workspace}/{id}
//   - https://www.notion.so/{workspace}/{id}?v={view_id}
//   - https://www.notion.so/{id}
//   - {id} (raw 32-char hex or UUID)
//
// The ID is extracted and formatted as UUID (8-4-4-4-12).
func ParseURL(notionURL string) (*ParsedURL, error) {
	input := strings.TrimSpace(notionURL)
	if input == "" {
		return nil, fmt.Errorf("empty URL")
	}

	// Check if it's already a raw ID (32-char hex or UUID format)
	rawID := extractRawID(input)
	if rawID != "" {
		return &ParsedURL{
			ID:    formatAsUUID(rawID),
			RawID: rawID,
		}, nil
	}

	// Parse as URL
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Extract ID from path
	path := strings.TrimPrefix(u.Path, "/")
	segments := strings.Split(path, "/")

	// Try to find a 32-char hex ID in the path segments
	for i := len(segments) - 1; i >= 0; i-- {
		segment := segments[i]
		rawID = extractRawID(segment)
		if rawID != "" {
			return &ParsedURL{
				ID:    formatAsUUID(rawID),
				RawID: rawID,
			}, nil
		}
	}

	return nil, fmt.Errorf("no valid Notion ID found in URL: %s", notionURL)
}

// extractRawID extracts a 32-character hex ID from a string.
// The ID can be:
//   - A plain 32-char hex string: abc123def456...
//   - A UUID format: abc123de-f456-7890-abcd-ef1234567890
//   - At the end of a title slug: My-Page-Title-abc123def456...
func extractRawID(s string) string {
	// Remove dashes to handle UUID format
	noDashes := strings.ReplaceAll(s, "-", "")

	// If the whole string (without dashes) is exactly 32 hex chars
	if len(noDashes) == 32 && hexIDPattern.MatchString(noDashes) {
		return noDashes
	}

	// Try to find a 32-char hex ID at the end (after a dash in title slugs)
	if match := hexIDPattern.FindString(s); match != "" {
		return match
	}

	// Check if removing dashes from the last 36 chars gives us a 32-char ID
	// (handles UUID format at end of URL path)
	if len(s) >= 36 {
		last36 := s[len(s)-36:]
		noDashes := strings.ReplaceAll(last36, "-", "")
		if len(noDashes) == 32 && hexIDPattern.MatchString(noDashes) {
			return noDashes
		}
	}

	return ""
}

// formatAsUUID formats a 32-character hex string as UUID (8-4-4-4-12).
func formatAsUUID(rawID string) string {
	if len(rawID) != 32 {
		return rawID
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		rawID[0:8],
		rawID[8:12],
		rawID[12:16],
		rawID[16:20],
		rawID[20:32],
	)
}
