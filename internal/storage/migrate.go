package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"worklog/internal/model"
)

// MigrationStats summarizes the actions taken (or planned) during migration.
type MigrationStats struct {
	LegacyWorklogFound  bool     `json:"legacy_worklog_found"`
	WorklogTasksMoved   int      `json:"worklog_tasks_moved"`
	WorklogEntriesMoved int      `json:"worklog_entries_moved"`
	TasksStandardized   int      `json:"tasks_standardized"`
	JournalStandardized int      `json:"journal_standardized"`
	Changes             []string `json:"changes"`
}

// Migrate standardizes legacy timestamps/formats in tasks.md and journal.md,
// and migrates entries from legacy worklog.md into tasks.md and journal.md.
func Migrate(vaultDir string, dryRun bool) (*MigrationStats, error) {
	stats := &MigrationStats{}

	tasksPath := filepath.Join(vaultDir, model.TasksFileName)
	journalPath := filepath.Join(vaultDir, model.JournalFileName)
	legacyPath := filepath.Join(vaultDir, "worklog.md")

	// 1. Standardize tasks.md
	if err := standardizeFile(tasksPath, true, stats, dryRun); err != nil {
		return nil, fmt.Errorf("failed standardizing tasks: %w", err)
	}

	// 2. Standardize journal.md
	if err := standardizeFile(journalPath, false, stats, dryRun); err != nil {
		return nil, fmt.Errorf("failed standardizing journal: %w", err)
	}

	// 3. Migrate legacy worklog.md if present
	if _, err := os.Stat(legacyPath); err == nil {
		stats.LegacyWorklogFound = true
		if err := migrateLegacyWorklog(vaultDir, legacyPath, tasksPath, journalPath, stats, dryRun); err != nil {
			return nil, fmt.Errorf("failed migrating legacy worklog.md: %w", err)
		}
	}

	return stats, nil
}

func standardizeFile(filePath string, isTasks bool, stats *MigrationStats, dryRun bool) error {
	f, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	var newLines []string
	changedCount := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		origLine := scanner.Text()
		trimmed := strings.TrimSpace(origLine)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "<!--") {
			newLines = append(newLines, origLine)
			continue
		}

		if isTasks {
			task, err := model.ParseTask(origLine, lineNum)
			if err == nil {
				formatted := strings.TrimRight(task.Markdown(), "\r\n")
				if formatted != origLine {
					stats.Changes = append(stats.Changes, fmt.Sprintf("%s:L%d: %s -> %s", filepath.Base(filePath), lineNum, origLine, formatted))
					changedCount++
					newLines = append(newLines, formatted)
					continue
				}
			}
		} else {
			entry, err := model.ParseEntry(origLine, lineNum)
			if err == nil {
				formatted := strings.TrimRight(entry.Markdown(), "\r\n")
				if formatted != origLine {
					stats.Changes = append(stats.Changes, fmt.Sprintf("%s:L%d: %s -> %s", filepath.Base(filePath), lineNum, origLine, formatted))
					changedCount++
					newLines = append(newLines, formatted)
					continue
				}
			}
		}

		newLines = append(newLines, origLine)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if isTasks {
		stats.TasksStandardized += changedCount
	} else {
		stats.JournalStandardized += changedCount
	}

	if changedCount > 0 && !dryRun {
		return rewriteFileAtomically(filePath, newLines)
	}

	return nil
}

func migrateLegacyWorklog(vaultDir, legacyPath, tasksPath, journalPath string, stats *MigrationStats, dryRun bool) error {
	f, err := os.Open(legacyPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Existing task keys to prevent duplicates
	existingTasks, _ := ScanTasks(tasksPath, nil)
	taskSet := make(map[string]bool)
	for _, t := range existingTasks {
		taskSet[strings.ToLower(strings.TrimSpace(t.Text))] = true
	}

	// Existing entry keys to prevent duplicates
	existingEntries, _ := ScanEntries(journalPath, nil)
	entrySet := make(map[string]bool)
	for _, e := range existingEntries {
		entrySet[strings.ToLower(strings.TrimSpace(e.Text))] = true
	}

	var newTasks []*model.Task
	var newEntries []*model.Entry

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "<!--") {
			continue
		}

		// Try parsing as task first
		task, err := model.ParseTask(line, lineNum)
		if err == nil {
			key := strings.ToLower(strings.TrimSpace(task.Text))
			if !taskSet[key] {
				taskSet[key] = true
				newTasks = append(newTasks, task)
				stats.WorklogTasksMoved++
				stats.Changes = append(stats.Changes, fmt.Sprintf("worklog.md -> tasks.md: %s", strings.TrimRight(task.Markdown(), "\r\n")))
			}
			continue
		}

		// Try parsing as entry
		entry, err := model.ParseEntry(line, lineNum)
		if err == nil {
			key := strings.ToLower(strings.TrimSpace(entry.Text))
			if !entrySet[key] {
				entrySet[key] = true
				newEntries = append(newEntries, entry)
				stats.WorklogEntriesMoved++
				stats.Changes = append(stats.Changes, fmt.Sprintf("worklog.md -> journal.md: %s", strings.TrimRight(entry.Markdown(), "\r\n")))
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if !dryRun {
		// Append new tasks to tasks.md
		for _, t := range newTasks {
			if err := AppendTask(tasksPath, t); err != nil {
				return err
			}
		}

		// Append new entries to journal.md
		for _, e := range newEntries {
			if err := AppendEntry(journalPath, e); err != nil {
				return err
			}
		}

		// Rename legacy worklog.md to worklog.md.migrated
		migratedName := legacyPath + ".migrated"
		if err := os.Rename(legacyPath, migratedName); err != nil {
			return fmt.Errorf("failed to rename %s to %s: %w", legacyPath, migratedName, err)
		}
	}

	return nil
}

func rewriteFileAtomically(filePath string, lines []string) error {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	origPerm := os.FileMode(0644)
	if info, err := os.Stat(filePath); err == nil && info.Mode().Perm() != 0 {
		origPerm = info.Mode().Perm()
	}

	tempFile, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		if _, err := os.Stat(tempPath); err == nil {
			_ = os.Remove(tempPath)
		}
	}()

	_ = os.Chmod(tempPath, origPerm)

	writer := bufio.NewWriter(tempFile)
	for _, l := range lines {
		if _, err := writer.WriteString(l + "\n"); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, filePath)
}
