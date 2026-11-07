package cmd

import (
	"reflect"
	"testing"
)

func TestParseNoteContent(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedText string
		expectedTags []string
	}{
		{
			name:         "Only header comment - should return empty",
			input:        "<!-- Enter your note below. Markdown comments (like this) will be ignored. -->\n\n",
			expectedText: "",
			expectedTags: nil,
		},
		{
			name: "Header comment with single line note and tag",
			input: `<!-- Enter your note below. Markdown comments (like this) will be ignored. -->

Quick note about deployment #ops`,
			expectedText: "Quick note about deployment",
			expectedTags: []string{"#ops"},
		},
		{
			name: "Multi-line note preserving line breaks",
			input: `<!-- Enter your note below. -->

Meeting with team #meeting
- Discussed Q4 goals
- Approved architecture proposal`,
			expectedText: "Meeting with team\n- Discussed Q4 goals\n- Approved architecture proposal",
			expectedTags: []string{"#meeting"},
		},
		{
			name: "Internal comments and markdown headings",
			input: `<!-- comment at top -->
# Retrospective #quarterly
<!-- inline comment block
that spans multiple lines -->
- Great progress on CLI
<!-- another comment -->
- Next step: Homebrew tap`,
			expectedText: "# Retrospective\n- Great progress on CLI\n- Next step: Homebrew tap",
			expectedTags: []string{"#quarterly"},
		},
		{
			name: "Trailing unclosed comment",
			input: `Note content
<!-- unclosed comment at bottom`,
			expectedText: "Note content",
			expectedTags: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotTags := parseNoteContent(tt.input)
			if gotText != tt.expectedText {
				t.Errorf("parseNoteContent() gotText = %q, want %q", gotText, tt.expectedText)
			}
			if !reflect.DeepEqual(gotTags, tt.expectedTags) {
				t.Errorf("parseNoteContent() gotTags = %v, want %v", gotTags, tt.expectedTags)
			}
		})
	}
}
