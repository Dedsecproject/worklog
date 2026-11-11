package storage

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"worklog/internal/model"
)

// TaskFilter is a predicate function for filtering tasks.
type TaskFilter func(t *model.Task) bool

// EntryFilter is a predicate function for filtering journal entries.
type EntryFilter func(e *model.Entry) bool

// ScanTasks streams lines from a file and returns parsed tasks matching the filter.
func ScanTasks(filePath string, filter TaskFilter) ([]*model.Task, error) {
	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allTasks []*model.Task
	var lastTask *model.Task
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") || strings.HasSuffix(trimmed, "-->") {
			continue
		}

		task, err := model.ParseTask(line, lineNumber)
		if err == nil {
			task.FilePath = filePath
			lastTask = task
			allTasks = append(allTasks, task)
		} else if lastTask != nil && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			// Continuation line belonging to the previous task
			lastTask.Text += "\n" + strings.TrimSpace(line)
			lastTask.RawLine += "\n" + line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var tasks []*model.Task
	for _, task := range allTasks {
		if filter == nil || filter(task) {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// ScanAllTasks scans both tasks.md and journal.md to ensure no tasks or checkbox notes are missed.
func ScanAllTasks(tasksFilePath, journalFilePath string, filter TaskFilter) ([]*model.Task, error) {
	tasks, err := ScanTasks(tasksFilePath, filter)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if journalFilePath != "" && journalFilePath != tasksFilePath {
		journalTasks, err := ScanTasks(journalFilePath, filter)
		if err == nil && len(journalTasks) > 0 {
			tasks = append(tasks, journalTasks...)
		}
	}

	return tasks, nil
}

// ScanEntries streams lines from journal.md and returns parsed entries matching the filter.
func ScanEntries(filePath string, filter EntryFilter) ([]*model.Entry, error) {
	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allEntries []*model.Entry
	var lastEntry *model.Entry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") || strings.HasSuffix(trimmed, "-->") {
			continue
		}

		entry, err := model.ParseEntry(line, lineNumber)
		if err == nil {
			entry.FilePath = filePath
			lastEntry = entry
			allEntries = append(allEntries, entry)
		} else if lastEntry != nil && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			// Continuation line belonging to the previous journal entry
			lastEntry.Text += "\n" + strings.TrimSpace(line)
			lastEntry.RawLine += "\n" + line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var entries []*model.Entry
	for _, entry := range allEntries {
		if filter == nil || filter(entry) {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// CollectTags scans both tasks and journal files and returns unique tags and counts.
func CollectTags(tasksFile, journalFile string) ([]string, map[string]int, error) {
	counts := make(map[string]int)

	tasks, err := ScanAllTasks(tasksFile, journalFile, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	for _, t := range tasks {
		for _, tag := range t.Tags {
			counts[tag]++
		}
	}

	entries, err := ScanEntries(journalFile, nil)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	for _, e := range entries {
		for _, tag := range e.Tags {
			counts[tag]++
		}
	}

	uniqueTags := make([]string, 0, len(counts))
	for tag := range counts {
		uniqueTags = append(uniqueTags, tag)
	}
	sort.Strings(uniqueTags)

	return uniqueTags, counts, nil
}

// TailLines returns the last n non-empty lines from the given file.
func TailLines(filePath string, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}

	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ringBuffer []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") || strings.HasSuffix(trimmed, "-->") {
			continue
		}
		ringBuffer = append(ringBuffer, line)
		if len(ringBuffer) > n {
			ringBuffer = ringBuffer[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ringBuffer, nil
}
