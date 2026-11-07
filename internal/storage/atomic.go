package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"worklog/internal/model"
)

// MutateFile streams lines from filePath, applies the mutator, writes to a temp file,
// and atomically replaces the original file.
func MutateFile(filePath string, mutator func(lineNum int, line string) (newLine string, keep bool, err error)) error {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	tempFile, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		if _, err := os.Stat(tempPath); err == nil {
			_ = os.Remove(tempPath)
		}
	}()

	srcFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	scanner := bufio.NewScanner(srcFile)
	writer := bufio.NewWriter(tempFile)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		newLine, keep, err := mutator(lineNum, line)
		if err != nil {
			return err
		}

		if keep {
			if !strings.HasSuffix(newLine, "\n") {
				newLine += "\n"
			}
			if _, err := writer.WriteString(newLine); err != nil {
				return fmt.Errorf("failed writing to temp file: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading source file: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed flushing temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed closing temp file: %w", err)
	}

	// Atomic replace
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed atomic rename: %w", err)
	}

	return nil
}

// FindTaskInFiles searches tasksFilePath first, and journalFilePath if not found.
func FindTaskInFiles(tasksFilePath, journalFilePath string, selector string) (*model.Task, error) {
	task, err := FindTaskBySelector(tasksFilePath, selector)
	if err == nil {
		return task, nil
	}
	if journalFilePath != "" && journalFilePath != tasksFilePath {
		jTask, jErr := FindTaskBySelector(journalFilePath, selector)
		if jErr == nil {
			return jTask, nil
		}
	}
	return nil, err
}

// FindTaskBySelector searches for a task matching line number, #number, or text substring.
// If multiple text matches exist, it returns the most recent one (last in file).
func FindTaskBySelector(filePath string, selector string) (*model.Task, error) {
	tasks, err := ScanTasks(filePath, nil)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found")
	}

	cleanSelector := strings.TrimSpace(selector)
	lineTarget := -1

	// Check if selector is #line or plain integer
	trimmedNum := strings.TrimPrefix(cleanSelector, "#")
	if num, err := strconv.Atoi(trimmedNum); err == nil {
		lineTarget = num
	}

	if lineTarget > 0 {
		// First check exact line number match
		for _, t := range tasks {
			if t.LineNumber == lineTarget {
				return t, nil
			}
		}
		// If lineTarget is within 1..len(tasks), support 1-based index
		if lineTarget <= len(tasks) {
			return tasks[lineTarget-1], nil
		}
		return nil, fmt.Errorf("no task found at line or index %d", lineTarget)
	}

	// Text query search: search case-insensitively
	lowerQuery := strings.ToLower(cleanSelector)
	var matchingTasks []*model.Task
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Text), lowerQuery) || strings.Contains(strings.ToLower(t.RawLine), lowerQuery) {
			matchingTasks = append(matchingTasks, t)
		}
	}

	if len(matchingTasks) == 0 {
		return nil, fmt.Errorf("no task matching %q found", selector)
	}

	// Return the most recent matching task
	return matchingTasks[len(matchingTasks)-1], nil
}

// FindEntryBySelector searches for a journal entry matching line number or text substring.
func FindEntryBySelector(filePath string, selector string) (*model.Entry, error) {
	entries, err := ScanEntries(filePath, nil)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no journal entries found")
	}

	cleanSelector := strings.TrimSpace(selector)
	lineTarget := -1

	trimmedNum := strings.TrimPrefix(cleanSelector, "#")
	if num, err := strconv.Atoi(trimmedNum); err == nil {
		lineTarget = num
	}

	if lineTarget > 0 {
		for _, e := range entries {
			if e.LineNumber == lineTarget {
				return e, nil
			}
		}
		if lineTarget <= len(entries) {
			return entries[lineTarget-1], nil
		}
		return nil, fmt.Errorf("no journal entry found at line or index %d", lineTarget)
	}

	lowerQuery := strings.ToLower(cleanSelector)
	var matchingEntries []*model.Entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Text), lowerQuery) || strings.Contains(strings.ToLower(e.RawLine), lowerQuery) {
			matchingEntries = append(matchingEntries, e)
		}
	}

	if len(matchingEntries) == 0 {
		return nil, fmt.Errorf("no journal entry matching %q found", selector)
	}

	return matchingEntries[len(matchingEntries)-1], nil
}

// ToggleTask updates the task at targetLineNum to its next status cycle.
func ToggleTask(filePath string, targetLineNum int, now time.Time) (*model.Task, error) {
	var updatedTask *model.Task

	err := MutateFile(filePath, func(lineNum int, line string) (string, bool, error) {
		if lineNum != targetLineNum {
			return line, true, nil
		}

		task, err := model.ParseTask(line, lineNum)
		if err != nil {
			return line, true, fmt.Errorf("line %d is not a valid task: %w", lineNum, err)
		}

		task.Toggle(now)
		updatedTask = task
		return strings.TrimRight(task.Markdown(), "\n"), true, nil
	})

	if err != nil {
		return nil, err
	}
	if updatedTask == nil {
		return nil, fmt.Errorf("task at line %d not found", targetLineNum)
	}

	return updatedTask, nil
}

// DeleteLine removes the specified line number from filePath.
func DeleteLine(filePath string, targetLineNum int) (string, error) {
	var removedLine string

	err := MutateFile(filePath, func(lineNum int, line string) (string, bool, error) {
		if lineNum == targetLineNum {
			removedLine = line
			return "", false, nil // delete line
		}
		return line, true, nil
	})

	if err != nil {
		return "", err
	}
	if removedLine == "" {
		return "", fmt.Errorf("line %d not found", targetLineNum)
	}

	return removedLine, nil
}

// ReplaceLine replaces the line at targetLineNum with newLine.
func ReplaceLine(filePath string, targetLineNum int, newLine string) error {
	found := false
	err := MutateFile(filePath, func(lineNum int, line string) (string, bool, error) {
		if lineNum == targetLineNum {
			found = true
			return newLine, true, nil
		}
		return line, true, nil
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("line %d not found", targetLineNum)
	}
	return nil
}
