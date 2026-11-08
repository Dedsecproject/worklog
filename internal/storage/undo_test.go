package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUndoFlow(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "tasks.md")
	file2 := filepath.Join(tmpDir, "journal.md")

	origContent1 := "- [ ] 2026-11-08 10:00 Original task 1\n- [ ] 2026-11-08 11:00 Original task 2\n"
	origContent2 := "- 2026-11-08 10:30 Original journal entry\n"

	if err := os.WriteFile(file1, []byte(origContent1), 0644); err != nil {
		t.Fatalf("failed writing file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(origContent2), 0644); err != nil {
		t.Fatalf("failed writing file2: %v", err)
	}

	// 1. Undo without prior backup should error
	if _, err := Undo(tmpDir); err == nil {
		t.Fatalf("expected error when undoing with no backup, got nil")
	}

	// 2. Save backup
	if err := SaveBackup(tmpDir, "test-action", file1, file2); err != nil {
		t.Fatalf("failed saving backup: %v", err)
	}

	// 3. Mutate files
	if err := os.WriteFile(file1, []byte("- [x] Mutated task\n"), 0644); err != nil {
		t.Fatalf("failed mutating file1: %v", err)
	}
	if err := os.Remove(file2); err != nil {
		t.Fatalf("failed removing file2: %v", err)
	}

	// 4. Undo
	meta, err := Undo(tmpDir)
	if err != nil {
		t.Fatalf("undo failed: %v", err)
	}

	if meta.Action != "test-action" {
		t.Errorf("meta.Action = %q, want 'test-action'", meta.Action)
	}

	// Verify file1 content restored
	restored1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("failed reading restored file1: %v", err)
	}
	if string(restored1) != origContent1 {
		t.Errorf("file1 content mismatch:\ngot: %q\nwant: %q", string(restored1), origContent1)
	}

	// Verify file2 restored
	restored2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("failed reading restored file2: %v", err)
	}
	if string(restored2) != origContent2 {
		t.Errorf("file2 content mismatch:\ngot: %q\nwant: %q", string(restored2), origContent2)
	}

	// 5. Subsequent Undo should fail (history cleared)
	if _, err := Undo(tmpDir); err == nil {
		t.Errorf("subsequent undo should have failed, but succeeded")
	}
}
