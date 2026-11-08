package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"worklog/internal/model"
)

func TestMigrateStandardization(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, model.TasksFileName)
	journalFile := filepath.Join(tmpDir, model.JournalFileName)

	legacyTasks := `- [/] Refactor auth flow [2026-10-24T14] #backend
- [x] Write unit tests [2026-10-24T15:30]
- [ ] Undated future task #roadmap
`
	legacyJournal := `- Fixed broken API endpoint [2026-10-24T16] #bug
- 2026-11-08 10:00 Standard note already #notes
`

	if err := os.WriteFile(tasksFile, []byte(legacyTasks), 0644); err != nil {
		t.Fatalf("failed writing tasks file: %v", err)
	}
	if err := os.WriteFile(journalFile, []byte(legacyJournal), 0644); err != nil {
		t.Fatalf("failed writing journal file: %v", err)
	}

	// 1. Dry run
	dryStats, err := Migrate(tmpDir, true)
	if err != nil {
		t.Fatalf("dry run migrate failed: %v", err)
	}
	if dryStats.TasksStandardized != 2 {
		t.Errorf("dryStats.TasksStandardized = %d, want 2", dryStats.TasksStandardized)
	}
	if dryStats.JournalStandardized != 1 {
		t.Errorf("dryStats.JournalStandardized = %d, want 1", dryStats.JournalStandardized)
	}

	// Verify files unchanged in dry run
	tContent, _ := os.ReadFile(tasksFile)
	if string(tContent) != legacyTasks {
		t.Errorf("dry run modified tasks file")
	}

	// 2. Real migration
	realStats, err := Migrate(tmpDir, false)
	if err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if realStats.TasksStandardized != 2 {
		t.Errorf("realStats.TasksStandardized = %d, want 2", realStats.TasksStandardized)
	}

	// Check migrated tasks file content
	newTContent, _ := os.ReadFile(tasksFile)
	lines := strings.Split(strings.TrimSpace(string(newTContent)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "- [/] 2026-10-24 14:00 Refactor auth flow #backend" {
		t.Errorf("unexpected line 0: %q", lines[0])
	}
	if lines[1] != "- [x] 2026-10-24 15:30 Write unit tests" {
		t.Errorf("unexpected line 1: %q", lines[1])
	}
	if lines[2] != "- [ ] Undated future task #roadmap" {
		t.Errorf("unexpected line 2: %q", lines[2])
	}

	// Check migrated journal file content
	newJContent, _ := os.ReadFile(journalFile)
	jLines := strings.Split(strings.TrimSpace(string(newJContent)), "\n")
	if len(jLines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(jLines))
	}
	if jLines[0] != "- 2026-10-24 16:00 Fixed broken API endpoint #bug" {
		t.Errorf("unexpected journal line 0: %q", jLines[0])
	}
}

func TestMigrateLegacyWorklogFile(t *testing.T) {
	tmpDir := t.TempDir()

	tasksFile := filepath.Join(tmpDir, model.TasksFileName)
	journalFile := filepath.Join(tmpDir, model.JournalFileName)
	legacyFile := filepath.Join(tmpDir, "worklog.md")

	// Pre-existing tasks and journal
	if err := os.WriteFile(tasksFile, []byte("- [ ] 2026-11-08 09:00 Existing task\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalFile, []byte("- 2026-11-08 09:30 Existing entry\n"), 0644); err != nil {
		t.Fatal(err)
	}

	legacyContent := `# Old Worklog Notes
- [ ] 2026-11-08 09:00 Existing task
- [x] 2026-10-20 12:00 Old completed task
- 2026-11-08 09:30 Existing entry
- 2026-10-20 12:30 Old journal note
`
	if err := os.WriteFile(legacyFile, []byte(legacyContent), 0644); err != nil {
		t.Fatal(err)
	}

	stats, err := Migrate(tmpDir, false)
	if err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	if !stats.LegacyWorklogFound {
		t.Errorf("expected LegacyWorklogFound to be true")
	}
	if stats.WorklogTasksMoved != 1 {
		t.Errorf("stats.WorklogTasksMoved = %d, want 1 (deduped existing)", stats.WorklogTasksMoved)
	}
	if stats.WorklogEntriesMoved != 1 {
		t.Errorf("stats.WorklogEntriesMoved = %d, want 1 (deduped existing)", stats.WorklogEntriesMoved)
	}

	// Legacy file renamed
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Errorf("worklog.md should have been renamed")
	}
	if _, err := os.Stat(legacyFile + ".migrated"); os.IsNotExist(err) {
		t.Errorf("worklog.md.migrated should exist")
	}

	// Check final tasks
	tasks, err := ScanTasks(tasksFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks in tasks.md, got %d", len(tasks))
	}

	// Check final journal
	entries, err := ScanEntries(journalFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries in journal.md, got %d", len(entries))
	}
}
