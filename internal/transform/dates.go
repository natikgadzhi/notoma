// Package transform provides conversion from Notion to Obsidian formats.
// dates.go handles date formatting with configurable options.
package transform

import (
	"fmt"
	"time"

	"github.com/jomei/notionapi"
)

// DateFormatterConfig holds the configuration for date formatting.
// This interface is implemented by config.DatesConfig.
type DateFormatterConfig interface {
	ShouldTransformEmptyDatetimeToDate() bool
	ShouldLinkDailyNotes() bool
	GetDailyNotePathPrefix() string
	GetDateFormat() string
}

// DefaultDateFormat is the default date format (DD-MM-YYYY).
const DefaultDateFormat = "02-01-2006"

// DateFormatter formats dates according to configuration.
type DateFormatter struct {
	transformEmptyDatetime bool
	linkDailyNotes         bool
	dailyNotePathPrefix    string
	dateFormat             string
}

// NewDateFormatter creates a DateFormatter from configuration.
// If cfg is nil, uses default values (transform=true, link=false, format=DD-MM-YYYY).
func NewDateFormatter(cfg DateFormatterConfig) *DateFormatter {
	df := &DateFormatter{
		transformEmptyDatetime: true,
		linkDailyNotes:         false,
		dailyNotePathPrefix:    "",
		dateFormat:             DefaultDateFormat,
	}

	if cfg != nil {
		df.transformEmptyDatetime = cfg.ShouldTransformEmptyDatetimeToDate()
		df.linkDailyNotes = cfg.ShouldLinkDailyNotes()
		df.dailyNotePathPrefix = cfg.GetDailyNotePathPrefix()
		df.dateFormat = cfg.GetDateFormat()
	}

	return df
}

// DefaultDateFormatter returns a DateFormatter with default settings.
func DefaultDateFormatter() *DateFormatter {
	return &DateFormatter{
		transformEmptyDatetime: true,
		linkDailyNotes:         false,
		dailyNotePathPrefix:    "",
		dateFormat:             DefaultDateFormat,
	}
}

// FormatDate formats a Notion Date for Obsidian frontmatter.
// If the time is midnight (00:00:00) and transformation is enabled,
// returns date-only format (DD-MM-YYYY).
// Otherwise, returns full datetime format.
func (df *DateFormatter) FormatDate(d *notionapi.Date) string {
	if d == nil {
		return ""
	}

	t := time.Time(*d)

	// Check if time is midnight (date-only in Notion)
	if df.transformEmptyDatetime && t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
		return df.formatDateOnly(t)
	}

	// Return full datetime in ISO 8601 format
	return d.String()
}

// formatDateOnly formats a time as date-only using the configured format.
func (df *DateFormatter) formatDateOnly(t time.Time) string {
	dateStr := t.Format(df.dateFormat)

	if df.linkDailyNotes {
		return df.wrapInLink(dateStr)
	}

	return dateStr
}

// wrapInLink wraps a date string in an Obsidian markdown link.
// Example: "15-01-2026" -> "[15-01-2026](Days/15-01-2026.md)"
func (df *DateFormatter) wrapInLink(dateStr string) string {
	return fmt.Sprintf("[%s](%s%s.md)", dateStr, df.dailyNotePathPrefix, dateStr)
}

// FormatDateRange formats a date range (start/end) for Obsidian.
func (df *DateFormatter) FormatDateRange(start, end *notionapi.Date) string {
	if start == nil {
		return ""
	}

	startStr := df.FormatDate(start)
	if end == nil {
		return startStr
	}

	return startStr + "/" + df.FormatDate(end)
}

// FormatDateObject formats a DateObject (used in mentions) for Obsidian.
// For mentions, we format differently: use → separator for ranges.
func (df *DateFormatter) FormatDateObject(date *notionapi.DateObject) string {
	if date == nil || date.Start == nil {
		return ""
	}

	startStr := df.FormatDate(date.Start)
	if date.End == nil {
		return startStr
	}

	return startStr + " → " + df.FormatDate(date.End)
}
