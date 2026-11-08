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
	TasksFileName   = model.TasksFileName
	JournalFileName = model.JournalFileName
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
	return ""
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
		if !e.Timestamp.IsZero() {
			m := e.Timestamp.Format("2006-01")
			if m != "" && m < currentMonth {
				monthSet[m] = true
			}
		}
	}

	tasks, err := ScanTasks(tasksPath, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, t := range tasks {
		if t.Status == model.StatusDone {
			m := CompletionMonth(t)
			if m != "" && m < currentMonth {
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
		if !e.Timestamp.IsZero() && e.Timestamp.Format("2006-01") == targetMonth {
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

// WriteArchiveFile writes or merges tasks and journal entries into an archive markdown file atomically.
func WriteArchiveFile(filePath, month string, tasks []*model.Task, entries []*model.Entry) error {
	existingBytes, err := os.ReadFile(filePath)
	var content string
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if len(existingBytes) > 0 {
		content = string(existingBytes)
	} else {
		content = fmt.Sprintf("# %s Archive\n\n", month)
	}

	// Append tasks to "## Completed Tasks" section
	if len(tasks) > 0 {
		var newTasksSB strings.Builder
		for _, t := range tasks {
			md := t.Markdown()
			if !strings.Contains(content, strings.TrimSpace(md)) {
				newTasksSB.WriteString(md)
			}
		}
		newTasksStr := newTasksSB.String()
		if newTasksStr != "" {
			if idx := strings.Index(content, "## Completed Tasks"); idx != -1 {
				// Find where this section ends (either next "## " or end of string)
				rest := content[idx+len("## Completed Tasks"):]
				if nextHeader := strings.Index(rest, "\n## "); nextHeader != -1 {
					insertPos := idx + len("## Completed Tasks") + nextHeader
					content = content[:insertPos] + "\n" + newTasksStr + content[insertPos:]
				} else {
					if !strings.HasSuffix(content, "\n") {
						content += "\n"
					}
					content += newTasksStr
				}
			} else {
				if !strings.HasSuffix(content, "\n") {
					content += "\n"
				}
				content += "\n## Completed Tasks\n" + newTasksStr
			}
		}
	}

	// Append journal entries to "## Journal Entries" section
	if len(entries) > 0 {
		var newEntriesSB strings.Builder
		for _, e := range entries {
			md := e.Markdown()
			if !strings.Contains(content, strings.TrimSpace(md)) {
				newEntriesSB.WriteString(md)
			}
		}
		newEntriesStr := newEntriesSB.String()
		if newEntriesStr != "" {
			if idx := strings.Index(content, "## Journal Entries"); idx != -1 {
				rest := content[idx+len("## Journal Entries"):]
				if nextHeader := strings.Index(rest, "\n## "); nextHeader != -1 {
					insertPos := idx + len("## Journal Entries") + nextHeader
					content = content[:insertPos] + "\n" + newEntriesStr + content[insertPos:]
				} else {
					if !strings.HasSuffix(content, "\n") {
						content += "\n"
					}
					content += newEntriesStr
				}
			} else {
				if !strings.HasSuffix(content, "\n") {
					content += "\n"
				}
				content += "\n## Journal Entries\n" + newEntriesStr
			}
		}
	}

	// Atomic write
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, "archive-*.tmp")
	if err != nil {
		return fmt.Errorf("failed creating temp archive file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	_ = os.Chmod(tempPath, 0644)
	if _, err := tempFile.WriteString(content); err != nil {
		return err
	}
	_ = tempFile.Sync()
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, filePath)
}
