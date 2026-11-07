package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"worklog/internal/model"
)

func TestAppendAndScan(t *testing.T) {
	tempDir := t.TempDir()
	tasksFile := filepath.Join(tempDir, "tasks.md")
	journalFile := filepath.Join(tempDir, "journal.md")

	now := time.Date(2026, 11, 6, 14, 30, 0, 0, time.Local)

	task1 := &model.Task{
		Status:    model.StatusTodo,
		CreatedAt: now,
		Text:      "Task one",
		Tags:      []string{"#devops"},
	}
	task2 := &model.Task{
		Status:    model.StatusDoing,
		CreatedAt: now.Add(time.Minute),
		Text:      "Task two",
		Tags:      []string{"#frontend"},
	}

	if err := AppendTask(tasksFile, task1); err != nil {
		t.Fatalf("AppendTask failed: %v", err)
	}
	if err := AppendTask(tasksFile, task2); err != nil {
		t.Fatalf("AppendTask failed: %v", err)
	}

	tasks, err := ScanTasks(tasksFile, nil)
	if err != nil {
		t.Fatalf("ScanTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Text != "Task one" || tasks[0].Status != model.StatusTodo {
		t.Errorf("task 0 mismatch: %+v", tasks[0])
	}
	if tasks[1].Text != "Task two" || tasks[1].Status != model.StatusDoing {
		t.Errorf("task 1 mismatch: %+v", tasks[1])
	}

	entry1 := &model.Entry{
		Timestamp: now,
		Text:      "Journal entry one",
		Tags:      []string{"#devops"},
	}
	if err := AppendEntry(journalFile, entry1); err != nil {
		t.Fatalf("AppendEntry failed: %v", err)
	}

	entries, err := ScanEntries(journalFile, nil)
	if err != nil {
		t.Fatalf("ScanEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	tags, counts, err := CollectTags(tasksFile, journalFile)
	if err != nil {
		t.Fatalf("CollectTags failed: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 unique tags, got %d", len(tags))
	}
	if counts["#devops"] != 2 {
		t.Errorf("expected count for #devops to be 2, got %d", counts["#devops"])
	}
}

func TestAtomicMutations(t *testing.T) {
	tempDir := t.TempDir()
	tasksFile := filepath.Join(tempDir, "tasks.md")

	now := time.Date(2026, 11, 6, 14, 30, 0, 0, time.Local)
	task1 := &model.Task{Status: model.StatusTodo, CreatedAt: now, Text: "Task to toggle"}
	task2 := &model.Task{Status: model.StatusTodo, CreatedAt: now, Text: "Task to delete"}

	_ = AppendTask(tasksFile, task1)
	_ = AppendTask(tasksFile, task2)

	// Test FindTaskBySelector
	found, err := FindTaskBySelector(tasksFile, "toggle")
	if err != nil {
		t.Fatalf("FindTaskBySelector failed: %v", err)
	}
	if found.Text != "Task to toggle" {
		t.Errorf("unexpected task: %v", found)
	}

	// Toggle task 1 (at line 1)
	updated, err := ToggleTask(tasksFile, 1, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("ToggleTask failed: %v", err)
	}
	if updated.Status != model.StatusDoing {
		t.Errorf("expected StatusDoing, got %s", updated.Status)
	}

	// Delete task 2 (at line 2)
	removed, err := DeleteLine(tasksFile, 2)
	if err != nil {
		t.Fatalf("DeleteLine failed: %v", err)
	}
	if !strings.Contains(removed, "Task to delete") {
		t.Errorf("unexpected removed line: %s", removed)
	}

	// Verify only 1 task remains
	tasks, err := ScanTasks(tasksFile, nil)
	if err != nil {
		t.Fatalf("ScanTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task remaining, got %d", len(tasks))
	}
}

func TestTailLines(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "test.txt")

	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	tail, err := TailLines(file, 3)
	if err != nil {
		t.Fatalf("TailLines failed: %v", err)
	}
	if len(tail) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(tail))
	}
	if tail[0] != "line 3" || tail[2] != "line 5" {
		t.Errorf("unexpected tail contents: %v", tail)
	}
}

func TestMultilineTaskScanning(t *testing.T) {
	tempDir := t.TempDir()
	tasksFile := filepath.Join(tempDir, "tasks.md")

	content := "- [ ] 2026-11-06 14:30 Main task description\n  Continuation line one\n  Continuation line two\n- [/] 2026-11-06 14:40 Second task\n"
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	tasks, err := ScanTasks(tasksFile, nil)
	if err != nil {
		t.Fatalf("ScanTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if !strings.Contains(tasks[0].Text, "Continuation line one") {
		t.Errorf("expected continuation in task 0, got %q", tasks[0].Text)
	}
	if !strings.Contains(tasks[0].Text, "Continuation line two") {
		t.Errorf("expected continuation 2 in task 0, got %q", tasks[0].Text)
	}
}

func TestCommentSkipping(t *testing.T) {
	tempDir := t.TempDir()
	journalFile := filepath.Join(tempDir, "journal.md")

	content := `<!-- Global comment header -->
- 2026-11-06 14:30 First valid entry
<!-- - 2026-11-06 14:35 Commented out entry -->
- 2026-11-06 14:40 Second valid entry
<!--
multi-line
comment
-->
`
	if err := os.WriteFile(journalFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	entries, err := ScanEntries(journalFile, nil)
	if err != nil {
		t.Fatalf("ScanEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Text != "First valid entry" {
		t.Errorf("entry 0 got %q, want 'First valid entry'", entries[0].Text)
	}
	if entries[1].Text != "Second valid entry" {
		t.Errorf("entry 1 got %q, want 'Second valid entry'", entries[1].Text)
	}
}
