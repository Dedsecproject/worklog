package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"worklog/internal/model"
)

func TestArchiveAndConsolidation(t *testing.T) {
	tempDir := t.TempDir()
	journalFile := filepath.Join(tempDir, JournalFileName)
	tasksFile := filepath.Join(tempDir, TasksFileName)

	// Past month: 2026-10
	pastDate := time.Date(2026, 10, 15, 10, 30, 0, 0, time.Local)
	pastDoneDate := time.Date(2026, 10, 16, 14, 0, 0, 0, time.Local)

	// Current month: 2026-11
	currentDate := time.Date(2026, 11, 7, 9, 0, 0, 0, time.Local)
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.Local)

	// Setup journal entries:
	// 1. Past single-line entry
	// 2. Past multi-line entry
	// 3. Current month entry
	e1 := &model.Entry{Timestamp: pastDate, Text: "October past entry", Tags: []string{"#oct"}}
	e2 := &model.Entry{Timestamp: pastDate.Add(time.Hour), Text: "October multi-line entry\nContinuation line A\nContinuation line B"}
	e3 := &model.Entry{Timestamp: currentDate, Text: "November current entry", Tags: []string{"#nov"}}

	if err := AppendEntry(journalFile, e1); err != nil {
		t.Fatalf("AppendEntry failed: %v", err)
	}
	if err := AppendEntry(journalFile, e2); err != nil {
		t.Fatalf("AppendEntry failed: %v", err)
	}
	if err := AppendEntry(journalFile, e3); err != nil {
		t.Fatalf("AppendEntry failed: %v", err)
	}

	// Setup tasks:
	// 1. Past completed task -> SHOULD BE ARCHIVED
	// 2. Past pending task -> SHOULD STAY IN TASKS.MD (Backlog!)
	// 3. Past in-progress task -> SHOULD STAY IN TASKS.MD (Active work!)
	// 4. Current month completed task -> SHOULD STAY IN TASKS.MD (Not past month!)
	// 5. Current month todo task -> SHOULD STAY IN TASKS.MD
	t1 := &model.Task{Status: model.StatusDone, CreatedAt: pastDate, CompletedAt: &pastDoneDate, Text: "Completed October task"}
	t2 := &model.Task{Status: model.StatusTodo, CreatedAt: pastDate, Text: "Unfinished October backlog task"}
	t3 := &model.Task{Status: model.StatusDoing, CreatedAt: pastDate, Text: "In-progress October task"}
	t4 := &model.Task{Status: model.StatusDone, CreatedAt: currentDate, CompletedAt: &currentDate, Text: "Completed November task"}
	t5 := &model.Task{Status: model.StatusTodo, CreatedAt: currentDate, Text: "Todo November task"}

	for _, task := range []*model.Task{t1, t2, t3, t4, t5} {
		if err := AppendTask(tasksFile, task); err != nil {
			t.Fatalf("AppendTask failed: %v", err)
		}
	}

	// 1. Test FindPastMonths
	pastMonths, err := FindPastMonths(journalFile, tasksFile, now)
	if err != nil {
		t.Fatalf("FindPastMonths failed: %v", err)
	}
	if len(pastMonths) != 1 || pastMonths[0] != "2026-10" {
		t.Fatalf("expected pastMonths = ['2026-10'], got %v", pastMonths)
	}

	// 2. Test Dry Run
	drySummaries, err := ConsolidatePastMonths(tempDir, "archive", true, now)
	if err != nil {
		t.Fatalf("ConsolidatePastMonths dry-run failed: %v", err)
	}
	if len(drySummaries) != 1 {
		t.Fatalf("expected 1 summary in dry-run, got %d", len(drySummaries))
	}
	if drySummaries[0].JournalCount != 2 || drySummaries[0].TaskCount != 1 {
		t.Errorf("dry-run mismatch: journal=%d, tasks=%d", drySummaries[0].JournalCount, drySummaries[0].TaskCount)
	}

	// Verify archive file was NOT created during dry run
	archiveFilePath := filepath.Join(tempDir, "archive", "2026-10.md")
	if _, err := os.Stat(archiveFilePath); !os.IsNotExist(err) {
		t.Fatalf("archive file should not exist after dry run")
	}

	// 3. Test Live Consolidation
	summaries, err := ConsolidatePastMonths(tempDir, "archive", false, now)
	if err != nil {
		t.Fatalf("ConsolidatePastMonths failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	// 4. Verify Archive File Content
	archiveBytes, err := os.ReadFile(archiveFilePath)
	if err != nil {
		t.Fatalf("failed reading archive file: %v", err)
	}
	archiveContent := string(archiveBytes)

	if !strings.Contains(archiveContent, "# 2026-10 Archive") {
		t.Errorf("missing header in archive file")
	}
	if !strings.Contains(archiveContent, "Completed October task") {
		t.Errorf("missing completed task in archive file")
	}
	if !strings.Contains(archiveContent, "October past entry") {
		t.Errorf("missing October entry in archive file")
	}
	if !strings.Contains(archiveContent, "Continuation line B") {
		t.Errorf("missing continuation line in archive file")
	}
	if strings.Contains(archiveContent, "November current entry") {
		t.Errorf("November entry should NOT be in October archive")
	}

	// 5. Verify Remaining Journal Entries
	remainingEntries, err := ScanEntries(journalFile, nil)
	if err != nil {
		t.Fatalf("ScanEntries failed: %v", err)
	}
	if len(remainingEntries) != 1 {
		t.Fatalf("expected 1 remaining journal entry, got %d", len(remainingEntries))
	}
	if remainingEntries[0].Text != "November current entry" {
		t.Errorf("unexpected remaining entry: %q", remainingEntries[0].Text)
	}

	// 6. Verify Remaining Tasks (Backlog & Current Tasks!)
	remainingTasks, err := ScanTasks(tasksFile, nil)
	if err != nil {
		t.Fatalf("ScanTasks failed: %v", err)
	}
	// Expected: t2 (past todo backlog), t3 (past doing), t4 (current done), t5 (current todo) = 4 tasks!
	if len(remainingTasks) != 4 {
		t.Fatalf("expected 4 remaining tasks, got %d", len(remainingTasks))
	}

	var foundUnfinishedOctober, foundDoingOctober, foundNovemberDone, foundNovemberTodo bool
	for _, remaining := range remainingTasks {
		switch remaining.Text {
		case "Unfinished October backlog task":
			foundUnfinishedOctober = true
		case "In-progress October task":
			foundDoingOctober = true
		case "Completed November task":
			foundNovemberDone = true
		case "Todo November task":
			foundNovemberTodo = true
		case "Completed October task":
			t.Errorf("Completed October task should have been removed from tasks.md!")
		}
	}

	if !foundUnfinishedOctober {
		t.Errorf("Unfinished October backlog task missing from tasks.md!")
	}
	if !foundDoingOctober {
		t.Errorf("In-progress October task missing from tasks.md!")
	}
	if !foundNovemberDone {
		t.Errorf("Completed November task missing from tasks.md!")
	}
	if !foundNovemberTodo {
		t.Errorf("Todo November task missing from tasks.md!")
	}

	// 7. Verify Second Consolidation Run (No past months left)
	pastMonthsSecond, err := FindPastMonths(journalFile, tasksFile, now)
	if err != nil {
		t.Fatalf("FindPastMonths second run failed: %v", err)
	}
	if len(pastMonthsSecond) != 0 {
		t.Errorf("expected 0 past months on second run, got %v", pastMonthsSecond)
	}
}
