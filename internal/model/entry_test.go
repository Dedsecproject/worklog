package model

import (
	"testing"
	"time"
)

func TestParseEntry(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantText  string
		wantTags  []string
		expectErr bool
	}{
		{
			name:      "Valid entry with tags",
			input:     "- 2026-11-06 14:35 Discussed microservice migration with team lead #meetings #arch",
			wantText:  "Discussed microservice migration with team lead",
			wantTags:  []string{"#meetings", "#arch"},
			expectErr: false,
		},
		{
			name:      "Entry without tags",
			input:     "- 2026-11-06 14:35 Quick note here",
			wantText:  "Quick note here",
			wantTags:  nil,
			expectErr: false,
		},
		{
			name:      "Entry with checkbox prefix is preserved",
			input:     "- [/] 2026-11-06 14:35 In progress note",
			wantText:  "In progress note",
			wantTags:  nil,
			expectErr: false,
		},
		{
			name:      "Invalid format",
			input:     "just plain text",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := ParseEntry(tt.input, 1)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if entry.Text != tt.wantText {
				t.Errorf("got text %q, want %q", entry.Text, tt.wantText)
			}
			if len(entry.Tags) != len(tt.wantTags) {
				t.Errorf("got %d tags, want %d", len(entry.Tags), len(tt.wantTags))
			}
		})
	}
}

func TestEntryMarkdown(t *testing.T) {
	ts := time.Date(2026, 11, 6, 14, 35, 0, 0, time.Local)
	entry := Entry{
		Timestamp: ts,
		Text:      "Discussed microservice migration with team lead",
		Tags:      []string{"#meetings", "#arch"},
	}

	expected := "- 2026-11-06 14:35 Discussed microservice migration with team lead #meetings #arch\n"
	if got := entry.Markdown(); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
