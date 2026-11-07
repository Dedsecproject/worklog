package ui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"worklog/internal/model"
)

func TestFormatTaskLineAlignment(t *testing.T) {
	DisableColor()

	now := time.Date(2026, 11, 6, 14, 30, 0, 0, time.Local)
	task1 := &model.Task{
		FilePath:   "/path/to/tasks.md",
		LineNumber: 1,
		Status:     model.StatusTodo,
		CreatedAt:  now,
		Text:       "First line of task",
	}

	task2 := &model.Task{
		FilePath:   "/path/to/journal.md",
		LineNumber: 3,
		Status:     model.StatusDoing,
		CreatedAt:  now,
		Text:       "Task from journal",
	}

	out1 := FormatTaskLine(task1, true)
	out2 := FormatTaskLine(task2, true)

	// Check that checkboxes start at the exact same index
	idxBox1 := strings.Index(out1, "[ ]")
	idxBox2 := strings.Index(out2, "[/]")

	if idxBox1 != idxBox2 {
		t.Errorf("Checkboxes are not aligned: out1 index %d, out2 index %d\nout1: %q\nout2: %q", idxBox1, idxBox2, out1, out2)
	}

	// Check that timestamps start at the exact same index
	idxTime1 := strings.Index(out1, "14:30")
	idxTime2 := strings.Index(out2, "14:30")

	if idxTime1 != idxTime2 {
		t.Errorf("Timestamps are not aligned: out1 index %d, out2 index %d\nout1: %q\nout2: %q", idxTime1, idxTime2, out1, out2)
	}
}

func TestFormatTaskLineHangingIndent(t *testing.T) {
	DisableColor()

	now := time.Date(2026, 11, 6, 14, 30, 0, 0, time.Local)
	task := &model.Task{
		FilePath:   "/path/to/tasks.md",
		LineNumber: 12,
		Status:     model.StatusDoing,
		CreatedAt:  now,
		Text:       "First line of task\nSecond continuation line\nThird continuation line",
		Tags:       []string{"#devops"},
	}

	formatted := FormatTaskLine(task, true)
	lines := strings.Split(formatted, "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), formatted)
	}

	// The text of line 0 starts at index where "First" begins
	textStartCol := strings.Index(lines[0], "First")

	// Lines 1 and 2 should have spaces up to textStartCol
	for i := 1; i < len(lines); i++ {
		trimmedLeading := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
		if trimmedLeading != textStartCol {
			t.Errorf("line %d does not have matching hanging indent: got %d spaces, want %d\nLine: %q", i, trimmedLeading, textStartCol, lines[i])
		}
	}
}

func TestFormatEntryLineHangingIndent(t *testing.T) {
	DisableColor()

	now := time.Date(2026, 11, 6, 14, 35, 0, 0, time.Local)
	entry := &model.Entry{
		LineNumber: 5,
		Timestamp:  now,
		Text:       "Meeting summary\nAction items discussed\nNext steps",
		Tags:       []string{"#meetings"},
	}

	formatted := FormatEntryLine(entry, true)
	lines := strings.Split(formatted, "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), formatted)
	}

	prefixRunes := utf8.RuneCountInString(lines[0][:strings.Index(lines[0], "Meeting")])
	for i := 1; i < len(lines); i++ {
		trimmedLeading := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
		if trimmedLeading != prefixRunes {
			t.Errorf("line %d does not have matching hanging indent: got %d spaces, want %d\nLine: %q", i, trimmedLeading, prefixRunes, lines[i])
		}
	}
}
