package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type UndoMeta struct {
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Files     []string  `json:"files"`
}

// BackupDir returns the path to <vault>/.worklog/backups
func BackupDir(vaultDir string) string {
	return filepath.Join(vaultDir, ".worklog", "backups")
}

// SaveBackup saves a snapshot of the given files before a mutation occurs.
func SaveBackup(vaultDir string, action string, filePaths ...string) error {
	bDir := BackupDir(vaultDir)
	if err := os.MkdirAll(bDir, 0755); err != nil {
		return err
	}

	var backedUpFiles []string
	for _, fp := range filePaths {
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			continue
		}
		base := filepath.Base(fp)
		dest := filepath.Join(bDir, base+".bak")
		if err := copyFile(fp, dest); err != nil {
			return fmt.Errorf("failed creating backup for %s: %w", base, err)
		}
		backedUpFiles = append(backedUpFiles, base)
	}

	meta := UndoMeta{
		Action:    action,
		Timestamp: time.Now(),
		Files:     backedUpFiles,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(bDir, "meta.json"), metaBytes, 0644)
}

// Undo restores the files from the last backup and clears the backup state.
func Undo(vaultDir string) (*UndoMeta, error) {
	bDir := BackupDir(vaultDir)
	metaPath := filepath.Join(bDir, "meta.json")

	metaBytes, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no undo history available")
	} else if err != nil {
		return nil, fmt.Errorf("failed reading undo metadata: %w", err)
	}

	var meta UndoMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("corrupt undo metadata: %w", err)
	}

	// Restore each file
	for _, base := range meta.Files {
		bakPath := filepath.Join(bDir, base+".bak")
		targetPath := filepath.Join(vaultDir, base)
		if _, err := os.Stat(bakPath); err != nil {
			continue
		}
		if err := copyFile(bakPath, targetPath); err != nil {
			return nil, fmt.Errorf("failed restoring %s: %w", base, err)
		}
		_ = os.Remove(bakPath)
	}

	_ = os.Remove(metaPath)
	return &meta, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
