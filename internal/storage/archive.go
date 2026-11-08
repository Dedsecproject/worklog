package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"worklog/internal/model"
)

const (
	TasksFileName   = "tasks.md"
	JournalFileName = "journal.md"
)

type ArchiveSummary struct {
	Month        string // e.g. "2026-10"
	JournalCount int
	TaskCount    int
	ArchivePath  string
}

// CompletionMonth returns the YYYY-MM string representing when the task was completed or created.
func CompletionMonth(t *model.Task) string {
	if t.CompletedAt != nil && !t.CompletedAt.IsZero() {
		return t.CompletedAt.Format("2006-01")
	}
	if !t.CreatedAt.IsZero() {
		return t.CreatedAt.Format("2006-01")
	}
	return time.Now().Format("2006-01")
}

// FindPastMonths scans journal.md and tasks.md and returns a sorted slice of past YYYY-MM months (strictly < currentMonth).
func FindPastMonths(journalPath, tasksPath string, now time.Time) ([]string, error) {
	currentMonth := now.Format("2006-01")
	monthSet := make(map[string]bool)

	entries, err := ScanEntries(journalPath, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range entries {
		m := e.Timestamp.Format("2006-01")
		if m < currentMonth {
			monthSet[m] = true
		}
	}

	tasks, err := ScanTasks(tasksPath, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, t := range tasks {
		if t.Status == model.StatusDone {
			m := CompletionMonth(t)
			if m < currentMonth {
				monthSet[m] = true
			}
		}
	}

	var months []string
	for m := range monthSet {
		months = append(months, m)
	}
	sort.Strings(months)
	return months, nil
}

// ConsolidatePastMonths discovers all past months and consolidates them into archiveDir.
func ConsolidatePastMonths(vaultDir, archiveSubdir string, dryRun bool, now time.Time) ([]ArchiveSummary, error) {
	journalPath := filepath.Join(vaultDir, JournalFileName)
	tasksPath := filepath.Join(vaultDir, TasksFileName)

	pastMonths, err := FindPastMonths(journalPath, tasksPath, now)
	if err != nil {
		return nil, err
	}

	var summaries []ArchiveSummary
	for _, m := range pastMonths {
		summary, err := ConsolidateMonth(vaultDir, archiveSubdir, m, dryRun, now)
		if err != nil {
			return summaries, err
		}
		if summary != nil && (summary.JournalCount > 0 || summary.TaskCount > 0) {
			summaries = append(summaries, *summary)
		}
	}

	return summaries, nil
}

// ConsolidateMonth archives entries and completed tasks for a specific YYYY-MM into archive/YYYY-MM.md.
func ConsolidateMonth(vaultDir, archiveSubdir, targetMonth string, dryRun bool, now time.Time) (*ArchiveSummary, error) {
	archiveDir := filepath.Join(vaultDir, archiveSubdir)
	archiveFile := filepath.Join(archiveDir, targetMonth+".md")
	journalPath := filepath.Join(vaultDir, JournalFileName)
	tasksPath := filepath.Join(vaultDir, TasksFileName)

	// 1. Collect journal entries for targetMonth
	var journalToArchive []*model.Entry
	journalLinesToDelete := make(map[int]bool)

	entries, err := ScanEntries(journalPath, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range entries {
		if e.Timestamp.Format("2006-01") == targetMonth {
			journalToArchive = append(journalToArchive, e)
			span := strings.Count(e.RawLine, "\n") + 1
			for l := e.LineNumber; l < e.LineNumber+span; l++ {
				journalLinesToDelete[l] = true
			}
		}
	}

	// 2. Collect completed tasks for targetMonth (PENDING and DOING TASKS ALWAYS STAY AS BACKLOG)
	var tasksToArchive []*model.Task
	taskLinesToDelete := make(map[int]bool)

	tasks, err := ScanTasks(tasksPath, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, t := range tasks {
		if t.Status == model.StatusDone && CompletionMonth(t) == targetMonth {
			tasksToArchive = append(tasksToArchive, t)
			span := strings.Count(t.RawLine, "\n") + 1
			for l := t.LineNumber; l < t.LineNumber+span; l++ {
				taskLinesToDelete[l] = true
			}
		}
	}

	summary := &ArchiveSummary{
		Month:        targetMonth,
		JournalCount: len(journalToArchive),
		TaskCount:    len(tasksToArchive),
		ArchivePath:  archiveFile,
	}

	if len(journalToArchive) == 0 && len(tasksToArchive) == 0 {
		return summary, nil
	}

	if dryRun {
		return summary, nil
	}

	// 3. Write / Merge into archive/targetMonth.md
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	if err := WriteArchiveFile(archiveFile, targetMonth, tasksToArchive, journalToArchive); err != nil {
		return nil, fmt.Errorf("failed writing archive file: %w", err)
	}

	// 4. Atomically remove archived entries from journal.md
	if len(journalLinesToDelete) > 0 {
		err = MutateFile(journalPath, func(lineNum int, line string) (string, bool, error) {
			if journalLinesToDelete[lineNum] {
				return "", false, nil
			}
			return line, true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed updating journal.md after archive: %w", err)
		}
	}

	// 5. Atomically remove archived completed tasks from tasks.md
	if len(taskLinesToDelete) > 0 {
		err = MutateFile(tasksPath, func(lineNum int, line string) (string, bool, error) {
			if taskLinesToDelete[lineNum] {
				return "", false, nil
			}
			return line, true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed updating tasks.md after archive: %w", err)
		}
	}

	return summary, nil
}

// WriteArchiveFile writes or merges tasks and journal entries into an archive markdown file.
func WriteArchiveFile(filePath, month string, tasks []*model.Task, entries []*model.Entry) error {
	existingBytes, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s Archive\n\n", month))
		if len(tasks) > 0 {
			sb.WriteString("## Completed Tasks\n")
			for _, t := range tasks {
				sb.WriteString(t.Markdown())
			}
			sb.WriteString("\n")
		}
		if len(entries) > 0 {
			sb.WriteString("## Journal Entries\n")
			for _, e := range entries {
				sb.WriteString(e.Markdown())
			}
			sb.WriteString("\n")
		}
		return os.WriteFile(filePath, []byte(sb.String()), 0644)
	} else if err != nil {
		return err
	}

	content := string(existingBytes)

	// Append tasks
	if len(tasks) > 0 {
		var tasksSB strings.Builder
		for _, t := range tasks {
			tasksSB.WriteString(t.Markdown())
		}
		if strings.Contains(content, "## Completed Tasks") {
			content = strings.Replace(content, "## Completed Tasks\n", "## Completed Tasks\n"+tasksSB.String(), 1)
		} else {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "\n## Completed Tasks\n" + tasksSB.String()
		}
	}

	// Append journal entries
	if len(entries) > 0 {
		var entriesSB strings.Builder
		for _, e := range entries {
			entriesSB.WriteString(e.Markdown())
		}
		if strings.Contains(content, "## Journal Entries") {
			content = strings.Replace(content, "## Journal Entries\n", "## Journal Entries\n"+entriesSB.String(), 1)
		} else {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += "\n## Journal Entries\n" + entriesSB.String()
		}
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}
