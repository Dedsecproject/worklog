package model

import (
	"reflect"
	"testing"
)

func TestSplitTags(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedText string
		expectedTags []string
	}{
		{
			name:         "Standard tags",
			input:        "Meeting with frontend team #meetings #arch",
			expectedText: "Meeting with frontend team",
			expectedTags: []string{"#meetings", "#arch"},
		},
		{
			name:         "Nested tags",
			input:        "Refactor database layer #backend/db/sql",
			expectedText: "Refactor database layer",
			expectedTags: []string{"#backend/db/sql"},
		},
		{
			name:         "Numeric references like PR numbers should NOT be tags",
			input:        "Reviewed and merged PR #142 for deployment",
			expectedText: "Reviewed and merged PR #142 for deployment",
			expectedTags: nil,
		},
		{
			name:         "URL fragments should NOT be tags",
			input:        "Check docs at https://example.com/guide#setup #docs",
			expectedText: "Check docs at https://example.com/guide#setup",
			expectedTags: []string{"#docs"},
		},
		{
			name:         "Sub-tag preservation",
			input:        "Important task #a with another #ab item",
			expectedText: "Important task with another item",
			expectedTags: []string{"#a", "#ab"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotTags := SplitTags(tt.input)
			if gotText != tt.expectedText {
				t.Errorf("gotText = %q, want %q", gotText, tt.expectedText)
			}
			if !reflect.DeepEqual(gotTags, tt.expectedTags) {
				t.Errorf("gotTags = %v, want %v", gotTags, tt.expectedTags)
			}
		})
	}
}
