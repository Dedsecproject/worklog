package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"worklog/internal/model"
	"worklog/internal/storage"
)

func TestArchiveCommand(t *testing.T) {
	tempDir := t.TempDir()
	journalFile := filepath.Join(tempDir, storage.JournalFileName)
	tasksFile := filepath.Join(tempDir, storage.TasksFileName)

	pastDate := time.Date(2026, 10, 15, 10, 0, 0, 0, time.Local)
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.Local)

	// Seed past entry & past done task
	_ = storage.AppendEntry(journalFile, &model.Entry{Timestamp: pastDate, Text: "Past October note"})
	_ = storage.AppendTask(tasksFile, &model.Task{Status: model.StatusDone, CreatedAt: pastDate, CompletedAt: &pastDate, Text: "Past October done task"})
	_ = storage.AppendTask(tasksFile, &model.Task{Status: model.StatusTodo, CreatedAt: pastDate, Text: "Past October unfinished backlog"})

	// Current month
	_ = storage.AppendEntry(journalFile, &model.Entry{Timestamp: now, Text: "Current November note"})
	_ = storage.AppendTask(tasksFile, &model.Task{Status: model.StatusTodo, CreatedAt: now, Text: "Current November todo"})

	// Run archive via storage directly using now
	summaries, err := storage.ConsolidatePastMonths(tempDir, "archive", false, now)
	if err != nil {
		t.Fatalf("ConsolidatePastMonths failed: %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Month != "2026-10" {
		t.Errorf("expected month '2026-10', got %q", summaries[0].Month)
	}
	if summaries[0].JournalCount != 1 || summaries[0].TaskCount != 1 {
		t.Errorf("expected 1 entry, 1 task, got j=%d, t=%d", summaries[0].JournalCount, summaries[0].TaskCount)
	}

	// Verify archive file exists
	archivePath := filepath.Join(tempDir, "archive", "2026-10.md")
	bytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("archive file not found: %v", err)
	}
	content := string(bytes)
	if !strings.Contains(content, "Past October note") {
		t.Errorf("archive missing entry")
	}
	if !strings.Contains(content, "Past October done task") {
		t.Errorf("archive missing done task")
	}

	// Verify backlog remains in tasks.md
	tasks, _ := storage.ScanTasks(tasksFile, nil)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks in tasks.md, got %d", len(tasks))
	}
	foundBacklog := false
	for _, task := range tasks {
		if task.Text == "Past October unfinished backlog" {
			foundBacklog = true
		}
	}
	if !foundBacklog {
		t.Errorf("unfinished backlog task missing from tasks.md")
	}
}
