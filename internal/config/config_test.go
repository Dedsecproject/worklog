package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"~", home},
		{"~/test/path", filepath.Join(home, "test/path")},
		{"/var/log", "/var/log"},
	}

	for _, tt := range tests {
		got, err := ExpandPath(tt.input)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLoadPrecedence(t *testing.T) {
	// Test CLI flag overrides all
	t.Setenv("WORKLOG_DIR", "/from/env")
	cfg, err := Load("/from/flag")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.VaultPath != "/from/flag" {
		t.Errorf("got %q, want /from/flag", cfg.VaultPath)
	}

	// Test Env var overrides default
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.VaultPath != "/from/env" {
		t.Errorf("got %q, want /from/env", cfg.VaultPath)
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		VaultPath: filepath.Join(tempDir, "vault"),
	}

	if err := cfg.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if _, err := os.Stat(cfg.TasksFilePath()); os.IsNotExist(err) {
		t.Errorf("tasks.md was not created")
	}

	if _, err := os.Stat(cfg.JournalFilePath()); os.IsNotExist(err) {
		t.Errorf("journal.md was not created")
	}
}

func TestSetAndGetValue(t *testing.T) {
	cfg := &Config{}

	if err := cfg.SetValue("editor", "hx"); err != nil {
		t.Fatalf("SetValue editor failed: %v", err)
	}
	if cfg.DefaultEditor != "hx" {
		t.Errorf("got %q, want hx", cfg.DefaultEditor)
	}

	val, err := cfg.GetValue("default_editor")
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if val != "hx" {
		t.Errorf("got %q, want hx", val)
	}

	// Test auto_archive
	if err := cfg.SetValue("auto_archive", "false"); err != nil {
		t.Fatalf("SetValue auto_archive failed: %v", err)
	}
	if cfg.IsAutoArchiveEnabled() {
		t.Errorf("expected auto_archive to be false")
	}
	val, _ = cfg.GetValue("auto_archive")
	if val != "false" {
		t.Errorf("got %q, want false", val)
	}

	// Test archive_dir
	if err := cfg.SetValue("archive_dir", "monthly"); err != nil {
		t.Fatalf("SetValue archive_dir failed: %v", err)
	}
	if cfg.GetArchiveDir() != "monthly" {
		t.Errorf("got %q, want monthly", cfg.GetArchiveDir())
	}

	if err := cfg.SetValue("invalid_key", "val"); err == nil {
		t.Errorf("expected error on invalid key, got nil")
	}
}

func TestSaveDoesNotOverwritePersistedVaultWithFlag(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")
	t.Setenv("WORKLOG_CONFIG", configFile)

	// Save an original config with vault /saved/vault
	initialCfg := &Config{
		ConfigPath: configFile,
	}
	_ = initialCfg.SetValue("vault_path", "/saved/vault")
	if err := initialCfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Now load with a CLI override flag: --path /override/path
	loadedCfg, err := Load("/override/path")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loadedCfg.VaultPath != "/override/path" {
		t.Errorf("expected runtime VaultPath to be /override/path, got %q", loadedCfg.VaultPath)
	}

	// Simulate auto-archive saving config
	loadedCfg.LastArchiveCheck = "2026-11"
	if err := loadedCfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Re-load without flag and verify the persisted vault path is STILL /saved/vault!
	reloadedCfg, err := Load("")
	if err != nil {
		t.Fatalf("reloaded Load failed: %v", err)
	}
	if reloadedCfg.VaultPath != "/saved/vault" {
		t.Errorf("CLI flag override leaked into config.json! Got %q, want /saved/vault", reloadedCfg.VaultPath)
	}
}
