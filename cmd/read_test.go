package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"worklog/internal/config"
	"worklog/internal/model"
	"worklog/internal/storage"
)

func setupTestVault(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	cfg = &config.Config{
		VaultPath: tempDir,
	}

	today := time.Now()
	todayStr := today.Format("2006-01-02")

	// Create test entries
	_ = storage.AppendEntry(cfg.JournalFilePath(), &model.Entry{
		Timestamp: today,
		Text:      "Frontend meeting with team",
		Tags:      []string{"#frontend", "#meetings"},
	})
	_ = storage.AppendEntry(cfg.JournalFilePath(), &model.Entry{
		Timestamp: today,
		Text:      "Backend db migration planning",
		Tags:      []string{"#backend/db"},
	})

	// Create test tasks
	_ = storage.AppendTask(cfg.TasksFilePath(), &model.Task{
		Status:    model.StatusDoing,
		CreatedAt: today,
		Text:      "Refactor auth logic",
		Tags:      []string{"#backend", "#auth"},
	})
	completedTime := today
	_ = storage.AppendTask(cfg.TasksFilePath(), &model.Task{
		Status:      model.StatusDone,
		CreatedAt:   today,
		CompletedAt: &completedTime,
		Text:        "Deploy v0.3.0 release",
		Tags:        []string{"#devops"},
	})
	_ = storage.AppendTask(cfg.TasksFilePath(), &model.Task{
		Status:    model.StatusTodo,
		CreatedAt: today,
		Text:      "Write documentation",
		Tags:      []string{"#docs"},
	})

	_ = todayStr
	return tempDir
}

func TestReadTagFiltering(t *testing.T) {
	setupTestVault(t)

	// Filter tasks by #backend (should match #backend/db or #backend)
	tasks, err := storage.ScanTasks(cfg.TasksFilePath(), func(task *model.Task) bool {
		return model.MatchesTag(task.Tags, "backend")
	})
	if err != nil {
		t.Fatalf("ScanTasks failed: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Text != "Refactor auth logic" {
		t.Errorf("expected 1 task matching 'backend', got %d", len(tasks))
	}

	// Filter journal entries by #backend (should match #backend/db via prefix)
	entries, err := storage.ScanEntries(cfg.JournalFilePath(), func(entry *model.Entry) bool {
		return model.MatchesTag(entry.Tags, "backend")
	})
	if err != nil {
		t.Fatalf("ScanEntries failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Text != "Backend db migration planning" {
		t.Errorf("expected 1 entry matching 'backend', got %d", len(entries))
	}
}

func TestJSONSerialization(t *testing.T) {
	setupTestVault(t)

	// Test DayOutput serialization
	tasks, _ := storage.ScanTasks(cfg.TasksFilePath(), nil)
	entries, _ := storage.ScanEntries(cfg.JournalFilePath(), nil)

	out := DayOutput{
		Date:    time.Now().Format("2006-01-02"),
		Entries: entries,
		Tasks:   tasks,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(out); err != nil {
		t.Fatalf("failed encoding DayOutput: %v", err)
	}

	var decoded DayOutput
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed unmarshaling DayOutput: %v", err)
	}

	if len(decoded.Entries) != 2 {
		t.Errorf("expected 2 entries in JSON, got %d", len(decoded.Entries))
	}
	if len(decoded.Tasks) != 3 {
		t.Errorf("expected 3 tasks in JSON, got %d", len(decoded.Tasks))
	}
}

func TestUndoCommandFlow(t *testing.T) {
	tmpDir := setupTestVault(t)

	// Save backup
	if err := storage.SaveBackup(tmpDir, "rm test", cfg.TasksFilePath()); err != nil {
		t.Fatalf("SaveBackup failed: %v", err)
	}

	// Delete file
	if err := os.Remove(cfg.TasksFilePath()); err != nil {
		t.Fatalf("failed removing tasks file: %v", err)
	}

	// Run undo
	meta, err := storage.Undo(tmpDir)
	if err != nil {
		t.Fatalf("Undo failed: %v", err)
	}
	if meta.Action != "rm test" {
		t.Errorf("meta.Action = %q, want 'rm test'", meta.Action)
	}

	// Tasks file should be restored
	if _, err := os.Stat(cfg.TasksFilePath()); os.IsNotExist(err) {
		t.Errorf("tasks file not restored after undo")
	}
}

func TestMigrateCommandFlow(t *testing.T) {
	tmpDir := setupTestVault(t)

	legacyPath := filepath.Join(tmpDir, "worklog.md")
	if err := os.WriteFile(legacyPath, []byte("- [ ] 2026-10-10 10:00 Old legacy item\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stats, err := storage.Migrate(tmpDir, false)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if !stats.LegacyWorklogFound {
		t.Errorf("expected LegacyWorklogFound to be true")
	}
	if stats.WorklogTasksMoved != 1 {
		t.Errorf("stats.WorklogTasksMoved = %d, want 1", stats.WorklogTasksMoved)
	}
}
