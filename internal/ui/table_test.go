package ui

import (
	"bytes"
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

func TestTaskWordWrapping(t *testing.T) {
	DisableColor()
	defer SetTerminalWidth(0)

	now := time.Date(2026, 11, 6, 14, 0, 0, 0, time.Local)
	task := &model.Task{
		FilePath:   "/path/to/tasks.md",
		LineNumber: 6,
		Status:     model.StatusTodo,
		CreatedAt:  now,
		Text:       "update your recall extension to add the following: a way to see the tags; a sidecard if available of the note [2026-11-02T14]",
		Tags:       []string{"#recall", "#extension"},
	}

	// Test wrapping at 60 columns
	formatted := FormatTaskLineWithWidth(task, true, 60)
	lines := strings.Split(formatted, "\n")

	if len(lines) <= 1 {
		t.Fatalf("expected task to wrap onto multiple lines, but got 1 line: %s", formatted)
	}

	textCol := strings.Index(lines[0], "update")
	if textCol < 0 {
		t.Fatalf("expected text to start with 'update' in line 0")
	}

	for i, line := range lines {
		vWidth := VisibleWidth(line)
		if vWidth > 60 {
			t.Errorf("line %d exceeded maxWidth 60: width=%d, line=%q", i, vWidth, line)
		}
		if i > 0 {
			trimmedLeading := len(line) - len(strings.TrimLeft(line, " "))
			if trimmedLeading != textCol {
				t.Errorf("line %d does not match hanging indent %d, got %d: %q", i, textCol, trimmedLeading, line)
			}
		}
	}
}

func TestJournalEntryWordWrapping(t *testing.T) {
	DisableColor()
	defer SetTerminalWidth(0)

	now := time.Date(2026, 11, 7, 21, 12, 0, 0, time.Local)
	entry := &model.Entry{
		LineNumber: 6,
		Timestamp:  now,
		Text:       "technically speaking, there is a worklog, there is a task, and there is a journal, i just need to configure them to see how I can make them work properly to my needs",
		Tags:       []string{"#reflection"},
	}

	// Test wrapping at 70 columns
	formatted := FormatEntryLineWithWidth(entry, true, 70)
	lines := strings.Split(formatted, "\n")

	if len(lines) <= 1 {
		t.Fatalf("expected journal entry to wrap onto multiple lines, got 1 line: %s", formatted)
	}

	prefixRunes := utf8.RuneCountInString(lines[0][:strings.Index(lines[0], "technically")])

	for i, line := range lines {
		vWidth := VisibleWidth(line)
		if vWidth > 70 {
			t.Errorf("line %d exceeded maxWidth 70: width=%d, line=%q", i, vWidth, line)
		}
		if i > 0 {
			trimmedLeading := len(line) - len(strings.TrimLeft(line, " "))
			if trimmedLeading != prefixRunes {
				t.Errorf("line %d does not match hanging indent %d, got %d: %q", i, prefixRunes, trimmedLeading, line)
			}
		}
	}
}

func TestRenderGroupedTasksAndDayGroupAlignment(t *testing.T) {
	DisableColor()
	SetTerminalWidth(60)
	defer SetTerminalWidth(0)

	now := time.Date(2026, 11, 6, 14, 0, 0, 0, time.Local)
	task := &model.Task{
		FilePath:   "/path/to/tasks.md",
		LineNumber: 1,
		Status:     model.StatusDoing,
		CreatedAt:  now,
		Text:       "This is a very long task description that should wrap nicely and keep its 4 space indent",
	}

	entry := &model.Entry{
		LineNumber: 1,
		Timestamp:  now,
		Text:       "This is a very long journal entry that should wrap nicely and keep its 4 space indent",
	}

	var buf bytes.Buffer
	RenderGroupedTasks(&buf, []*model.Task{task}, nil, nil, true)
	output := buf.String()

	// In RenderGroupedTasks, all task lines should start with at least 4 spaces
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "This is") || strings.Contains(line, "nicely") {
			if !strings.HasPrefix(line, "    ") {
				t.Errorf("expected line in RenderGroupedTasks to have 4-space prefix, got: %q", line)
			}
		}
	}

	var dayBuf bytes.Buffer
	RenderDayGroup(&dayBuf, now, []*model.Entry{entry}, []*model.Task{task}, true)
	dayOutput := dayBuf.String()

	// In RenderDayGroup, all task and journal lines should start with 4 spaces
	for _, line := range strings.Split(dayOutput, "\n") {
		if strings.Contains(line, "This is") || strings.Contains(line, "nicely") {
			if !strings.HasPrefix(line, "    ") {
				t.Errorf("expected line in RenderDayGroup to have 4-space prefix, got: %q", line)
			}
		}
	}
}

func TestVisibleWidthAndStripANSI(t *testing.T) {
	colored := "\033[36m#meetings\033[0m"
	if StripANSI(colored) != "#meetings" {
		t.Errorf("StripANSI failed: got %q", StripANSI(colored))
	}
	if VisibleWidth(colored) != 9 {
		t.Errorf("VisibleWidth failed: got %d, want 9", VisibleWidth(colored))
	}

	plain := "Hello World"
	if VisibleWidth(plain) != 11 {
		t.Errorf("VisibleWidth plain failed: got %d, want 11", VisibleWidth(plain))
	}
}
