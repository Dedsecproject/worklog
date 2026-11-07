package storage

import (
	"os"
	"path/filepath"
	"worklog/internal/model"
)

// AppendEntry writes a journal entry to the target file using O_APPEND.
func AppendEntry(filePath string, entry *model.Entry) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry.Markdown())
	return err
}

// AppendTask writes a task to the target file using O_APPEND.
func AppendTask(filePath string, task *model.Task) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(task.Markdown())
	return err
}
