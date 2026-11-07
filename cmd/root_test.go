package cmd

import (
	"reflect"
	"testing"
)

func TestNormalizeArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Bare text should insert write",
			input:    []string{"Discussed", "architecture", "#arch"},
			expected: []string{"write", "Discussed", "architecture", "#arch"},
		},
		{
			name:     "Flag before bare text should insert write before text",
			input:    []string{"--path", "/tmp/vault", "Discussed", "architecture"},
			expected: []string{"--path", "/tmp/vault", "write", "Discussed", "architecture"},
		},
		{
			name:     "Short flag -p before bare text",
			input:    []string{"-p", "/tmp/vault", "Quick", "entry"},
			expected: []string{"-p", "/tmp/vault", "write", "Quick", "entry"},
		},
		{
			name:     "Known command should not be modified",
			input:    []string{"task", "deploy", "release"},
			expected: []string{"task", "deploy", "release"},
		},
		{
			name:     "Known command with flags should not be modified",
			input:    []string{"--path", "/tmp/vault", "today"},
			expected: []string{"--path", "/tmp/vault", "today"},
		},
		{
			name:     "Help flag should not be modified",
			input:    []string{"--help"},
			expected: []string{"--help"},
		},
		{
			name:     "Empty args",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("normalizeArgs(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
