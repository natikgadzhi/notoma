package transform

import (
	"testing"
	"time"

	"github.com/jomei/notionapi"
)

// mockDateConfig implements DateFormatterConfig for testing.
type mockDateConfig struct {
	transformEmpty      bool
	linkDailyNotes      bool
	dailyNotePathPrefix string
	dateFormat          string
}

func (m *mockDateConfig) ShouldTransformEmptyDatetimeToDate() bool { return m.transformEmpty }
func (m *mockDateConfig) ShouldLinkDailyNotes() bool               { return m.linkDailyNotes }
func (m *mockDateConfig) GetDailyNotePathPrefix() string           { return m.dailyNotePathPrefix }
func (m *mockDateConfig) GetDateFormat() string {
	if m.dateFormat == "" {
		return DefaultDateFormat
	}
	return m.dateFormat
}

func TestNewDateFormatter(t *testing.T) {
	tests := []struct {
		name   string
		config DateFormatterConfig
		check  func(*testing.T, *DateFormatter)
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			check: func(t *testing.T, df *DateFormatter) {
				if !df.transformEmptyDatetime {
					t.Error("expected transformEmptyDatetime to be true")
				}
				if df.linkDailyNotes {
					t.Error("expected linkDailyNotes to be false")
				}
				if df.dailyNotePathPrefix != "" {
					t.Errorf("expected empty prefix, got %q", df.dailyNotePathPrefix)
				}
				if df.dateFormat != DefaultDateFormat {
					t.Errorf("expected default format %q, got %q", DefaultDateFormat, df.dateFormat)
				}
			},
		},
		{
			name: "custom config",
			config: &mockDateConfig{
				transformEmpty:      false,
				linkDailyNotes:      true,
				dailyNotePathPrefix: "Days/",
				dateFormat:          "2006-01-02",
			},
			check: func(t *testing.T, df *DateFormatter) {
				if df.transformEmptyDatetime {
					t.Error("expected transformEmptyDatetime to be false")
				}
				if !df.linkDailyNotes {
					t.Error("expected linkDailyNotes to be true")
				}
				if df.dailyNotePathPrefix != "Days/" {
					t.Errorf("expected prefix %q, got %q", "Days/", df.dailyNotePathPrefix)
				}
				if df.dateFormat != "2006-01-02" {
					t.Errorf("expected format %q, got %q", "2006-01-02", df.dateFormat)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := NewDateFormatter(tt.config)
			tt.check(t, df)
		})
	}
}

func TestDateFormatter_FormatDate(t *testing.T) {
	midnightDate := notionapi.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	dateTime := notionapi.Date(time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC))

	tests := []struct {
		name   string
		config *mockDateConfig
		date   *notionapi.Date
		want   string
	}{
		{
			name: "midnight date with default format",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			date: &midnightDate,
			want: "15-01-2026",
		},
		{
			name: "midnight date with ISO format",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     "2006-01-02",
			},
			date: &midnightDate,
			want: "2026-01-15",
		},
		{
			name: "midnight date with US format",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     "01/02/2006",
			},
			date: &midnightDate,
			want: "01/15/2026",
		},
		{
			name: "midnight date with transform disabled",
			config: &mockDateConfig{
				transformEmpty: false,
			},
			date: &midnightDate,
			want: "2026-01-15T00:00:00Z",
		},
		{
			name: "non-midnight datetime always returns ISO",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			date: &dateTime,
			want: "2026-01-15T14:30:00Z",
		},
		{
			name:   "nil date returns empty",
			config: &mockDateConfig{},
			date:   nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := NewDateFormatter(tt.config)
			got := df.FormatDate(tt.date)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateFormatter_FormatDateWithLink(t *testing.T) {
	midnightDate := notionapi.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name   string
		config *mockDateConfig
		date   *notionapi.Date
		want   string
	}{
		{
			name: "with daily note link and prefix",
			config: &mockDateConfig{
				transformEmpty:      true,
				linkDailyNotes:      true,
				dailyNotePathPrefix: "Days/",
				dateFormat:          DefaultDateFormat,
			},
			date: &midnightDate,
			want: "[15-01-2026](Days/15-01-2026.md)",
		},
		{
			name: "with daily note link no prefix",
			config: &mockDateConfig{
				transformEmpty:      true,
				linkDailyNotes:      true,
				dailyNotePathPrefix: "",
				dateFormat:          DefaultDateFormat,
			},
			date: &midnightDate,
			want: "[15-01-2026](15-01-2026.md)",
		},
		{
			name: "with daily note link custom format",
			config: &mockDateConfig{
				transformEmpty:      true,
				linkDailyNotes:      true,
				dailyNotePathPrefix: "Journal/",
				dateFormat:          "2006-01-02",
			},
			date: &midnightDate,
			want: "[2026-01-15](Journal/2026-01-15.md)",
		},
		{
			name: "link disabled",
			config: &mockDateConfig{
				transformEmpty: true,
				linkDailyNotes: false,
				dateFormat:     DefaultDateFormat,
			},
			date: &midnightDate,
			want: "15-01-2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := NewDateFormatter(tt.config)
			got := df.FormatDate(tt.date)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateFormatter_FormatDateRange(t *testing.T) {
	startDate := notionapi.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	endDate := notionapi.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
	startDateTime := notionapi.Date(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	endDateTime := notionapi.Date(time.Date(2026, 1, 20, 18, 0, 0, 0, time.UTC))

	tests := []struct {
		name   string
		config *mockDateConfig
		start  *notionapi.Date
		end    *notionapi.Date
		want   string
	}{
		{
			name: "date range without end",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			start: &startDate,
			end:   nil,
			want:  "15-01-2026",
		},
		{
			name: "date range with end",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			start: &startDate,
			end:   &endDate,
			want:  "15-01-2026/20-01-2026",
		},
		{
			name: "datetime range",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			start: &startDateTime,
			end:   &endDateTime,
			want:  "2026-01-15T10:00:00Z/2026-01-20T18:00:00Z",
		},
		{
			name: "date range with links",
			config: &mockDateConfig{
				transformEmpty:      true,
				linkDailyNotes:      true,
				dailyNotePathPrefix: "Days/",
				dateFormat:          DefaultDateFormat,
			},
			start: &startDate,
			end:   &endDate,
			want:  "[15-01-2026](Days/15-01-2026.md)/[20-01-2026](Days/20-01-2026.md)",
		},
		{
			name:   "nil start returns empty",
			config: &mockDateConfig{},
			start:  nil,
			end:    &endDate,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := NewDateFormatter(tt.config)
			got := df.FormatDateRange(tt.start, tt.end)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDateFormatter_FormatDateObject(t *testing.T) {
	startDate := notionapi.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	endDate := notionapi.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name   string
		config *mockDateConfig
		date   *notionapi.DateObject
		want   string
	}{
		{
			name: "date object without end (mentions use arrow)",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			date: &notionapi.DateObject{
				Start: &startDate,
			},
			want: "15-01-2026",
		},
		{
			name: "date object with end uses arrow separator",
			config: &mockDateConfig{
				transformEmpty: true,
				dateFormat:     DefaultDateFormat,
			},
			date: &notionapi.DateObject{
				Start: &startDate,
				End:   &endDate,
			},
			want: "15-01-2026 → 20-01-2026",
		},
		{
			name:   "nil date object",
			config: &mockDateConfig{},
			date:   nil,
			want:   "",
		},
		{
			name:   "date object with nil start",
			config: &mockDateConfig{},
			date: &notionapi.DateObject{
				Start: nil,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := NewDateFormatter(tt.config)
			got := df.FormatDateObject(tt.date)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultDateFormatter(t *testing.T) {
	df := DefaultDateFormatter()

	if !df.transformEmptyDatetime {
		t.Error("expected transformEmptyDatetime to be true")
	}
	if df.linkDailyNotes {
		t.Error("expected linkDailyNotes to be false")
	}
	if df.dailyNotePathPrefix != "" {
		t.Errorf("expected empty prefix, got %q", df.dailyNotePathPrefix)
	}
	if df.dateFormat != DefaultDateFormat {
		t.Errorf("expected format %q, got %q", DefaultDateFormat, df.dateFormat)
	}
}
