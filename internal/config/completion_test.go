package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCompletionToFile(t *testing.T) {
	tempDir := t.TempDir()
	rcFile := filepath.Join(tempDir, ".zshrc")

	// First install: file does not exist yet
	installed, err := InstallCompletionToFile(rcFile, "zsh")
	if err != nil {
		t.Fatalf("First install failed: %v", err)
	}
	if !installed {
		t.Errorf("expected installed = true on first run")
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "worklog completion zsh") {
		t.Errorf("expected completion line in file, got: %s", string(content))
	}

	// Second install: should detect existing line and NOT duplicate
	installed2, err := InstallCompletionToFile(rcFile, "zsh")
	if err != nil {
		t.Fatalf("Second install failed: %v", err)
	}
	if installed2 {
		t.Errorf("expected installed = false on second run (duplicate detected)")
	}

	content2, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	occurrences := strings.Count(string(content2), "worklog completion zsh")
	if occurrences != 1 {
		t.Errorf("expected exactly 1 occurrence, got %d:\n%s", occurrences, string(content2))
	}
}
