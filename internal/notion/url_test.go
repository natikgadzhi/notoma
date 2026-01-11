package notion

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantID    string
		wantRawID string
		wantErr   bool
	}{
		{
			name:      "full URL with title slug",
			input:     "https://www.notion.so/workspace/My-Page-Title-abc123def456789012345678901234ab",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:      "URL with just ID",
			input:     "https://www.notion.so/workspace/abc123def456789012345678901234ab",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:      "database URL with view parameter",
			input:     "https://www.notion.so/workspace/abc123def456789012345678901234ab?v=xyz789",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:      "URL without workspace",
			input:     "https://www.notion.so/abc123def456789012345678901234ab",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:      "raw 32-char hex ID",
			input:     "abc123def456789012345678901234ab",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:      "UUID format ID",
			input:     "abc123de-f456-7890-1234-5678901234ab",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:      "URL with UUID format in path",
			input:     "https://www.notion.so/workspace/abc123de-f456-7890-1234-5678901234ab",
			wantID:    "abc123de-f456-7890-1234-5678901234ab",
			wantRawID: "abc123def456789012345678901234ab",
			wantErr:   false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "URL without valid ID",
			input:   "https://www.notion.so/workspace/some-page",
			wantErr: true,
		},
		{
			name:    "too short hex string",
			input:   "abc123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ID != tt.wantID {
				t.Errorf("ParseURL() ID = %v, want %v", got.ID, tt.wantID)
			}
			if got.RawID != tt.wantRawID {
				t.Errorf("ParseURL() RawID = %v, want %v", got.RawID, tt.wantRawID)
			}
		})
	}
}

func TestFormatAsUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "32-char hex to UUID",
			input: "abc123def456789012345678901234ab",
			want:  "abc123de-f456-7890-1234-5678901234ab",
		},
		{
			name:  "all zeros",
			input: "00000000000000000000000000000000",
			want:  "00000000-0000-0000-0000-000000000000",
		},
		{
			name:  "wrong length returns unchanged",
			input: "abc123",
			want:  "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAsUUID(tt.input); got != tt.want {
				t.Errorf("formatAsUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractRawID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain 32-char hex",
			input: "abc123def456789012345678901234ab",
			want:  "abc123def456789012345678901234ab",
		},
		{
			name:  "UUID format",
			input: "abc123de-f456-7890-1234-5678901234ab",
			want:  "abc123def456789012345678901234ab",
		},
		{
			name:  "title slug with ID",
			input: "My-Page-Title-abc123def456789012345678901234ab",
			want:  "abc123def456789012345678901234ab",
		},
		{
			name:  "no valid ID",
			input: "some-random-text",
			want:  "",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRawID(tt.input); got != tt.want {
				t.Errorf("extractRawID() = %v, want %v", got, tt.want)
			}
		})
	}
}
