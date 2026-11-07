package cmd

import (
	"testing"
)

func TestParseSelector(t *testing.T) {
	tests := []struct {
		input           string
		wantClean       string
		wantTaskOnly    bool
		wantJournalOnly bool
		wantLast        bool
	}{
		{"1", "1", false, false, false},
		{"#1", "1", false, false, false},
		{"j:1", "1", false, true, false},
		{"journal:5", "5", false, true, false},
		{"#j:3", "3", false, true, false},
		{"j2", "2", false, true, false},
		{"t:1", "1", true, false, false},
		{"task:4", "4", true, false, false},
		{"tasks:10", "10", true, false, false},
		{"#t:7", "7", true, false, false},
		{"t3", "3", true, false, false},
		{"--last", "", false, false, true},
		{"last", "", false, false, true},
		{"Discussed microservice", "Discussed microservice", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			clean, taskOnly, journalOnly, isLast := parseSelector(tt.input)
			if clean != tt.wantClean {
				t.Errorf("clean = %q, want %q", clean, tt.wantClean)
			}
			if taskOnly != tt.wantTaskOnly {
				t.Errorf("taskOnly = %v, want %v", taskOnly, tt.wantTaskOnly)
			}
			if journalOnly != tt.wantJournalOnly {
				t.Errorf("journalOnly = %v, want %v", journalOnly, tt.wantJournalOnly)
			}
			if isLast != tt.wantLast {
				t.Errorf("isLast = %v, want %v", isLast, tt.wantLast)
			}
		})
	}
}
