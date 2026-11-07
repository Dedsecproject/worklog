package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultDateFormat = "2006-01-02"
	DefaultTimeFormat = "15:04"
	TasksFileName     = "tasks.md"
	JournalFileName   = "journal.md"
)

type Config struct {
	VaultPath     string `json:"vault_path"`
	DefaultEditor string `json:"default_editor,omitempty"`
	DateFormat    string `json:"date_format,omitempty"`
	TimeFormat    string `json:"time_format,omitempty"`
	ConfigPath    string `json:"-"`
}

// ExpandPath expands leading ~ to the current user's home directory.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return filepath.Clean(path), nil
}

// DefaultConfigPath returns ~/.config/worklog/config.json or $WORKLOG_CONFIG if set.
func DefaultConfigPath() (string, error) {
	if env := os.Getenv("WORKLOG_CONFIG"); env != "" {
		return ExpandPath(env)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "worklog", "config.json"), nil
}

// Load attempts to resolve configuration following precedence:
// 1. CLI flag override (--path)
// 2. WORKLOG_DIR environment variable
// 3. Local config file (~/.config/worklog/config.json)
// 4. Default fallback (~/.local/share/worklog if exists, otherwise ~/worklog)
func Load(flagPath string) (*Config, error) {
	cfg := &Config{
		DateFormat: DefaultDateFormat,
		TimeFormat: DefaultTimeFormat,
	}

	configPath, err := DefaultConfigPath()
	if err == nil {
		cfg.ConfigPath = configPath
		if fileData, err := os.ReadFile(configPath); err == nil {
			var fileCfg Config
			if err := json.Unmarshal(fileData, &fileCfg); err == nil {
				if fileCfg.VaultPath != "" {
					cfg.VaultPath = fileCfg.VaultPath
				}
				if fileCfg.DefaultEditor != "" {
					cfg.DefaultEditor = fileCfg.DefaultEditor
				}
				if fileCfg.DateFormat != "" {
					cfg.DateFormat = fileCfg.DateFormat
				}
				if fileCfg.TimeFormat != "" {
					cfg.TimeFormat = fileCfg.TimeFormat
				}
			}
		}
	}

	// Precedence overrides
	if envDir := os.Getenv("WORKLOG_DIR"); envDir != "" {
		cfg.VaultPath = envDir
	}

	if flagPath != "" {
		cfg.VaultPath = flagPath
	}

	// Fallback if still unset
	if cfg.VaultPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.New("could not determine user home directory")
		}
		localShare := filepath.Join(home, ".local", "share", "worklog")
		if info, err := os.Stat(localShare); err == nil && info.IsDir() {
			cfg.VaultPath = localShare
		} else {
			cfg.VaultPath = filepath.Join(home, "worklog")
		}
	}

	expandedVault, err := ExpandPath(cfg.VaultPath)
	if err != nil {
		return nil, err
	}
	cfg.VaultPath = expandedVault

	return cfg, nil
}

// TasksFilePath returns the absolute path to tasks.md
func (c *Config) TasksFilePath() string {
	return filepath.Join(c.VaultPath, TasksFileName)
}

// JournalFilePath returns the absolute path to journal.md
func (c *Config) JournalFilePath() string {
	return filepath.Join(c.VaultPath, JournalFileName)
}

// EnsureDir guarantees the vault directory and data files exist.
func (c *Config) EnsureDir() error {
	if err := os.MkdirAll(c.VaultPath, 0755); err != nil {
		return err
	}

	tasksPath := c.TasksFilePath()
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		if err := os.WriteFile(tasksPath, []byte(""), 0644); err != nil {
			return err
		}
	}

	journalPath := c.JournalFilePath()
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		if err := os.WriteFile(journalPath, []byte(""), 0644); err != nil {
			return err
		}
	}

	return nil
}

// GetEditor returns the preferred editor
func (c *Config) GetEditor() string {
	if c.DefaultEditor != "" {
		return c.DefaultEditor
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	if env := os.Getenv("VISUAL"); env != "" {
		return env
	}
	return "nano"
}

// SetValue updates a config field by key name.
func (c *Config) SetValue(key, value string) error {
	switch strings.ToLower(key) {
	case "vault_path", "vault", "path", "folder", "dir":
		expanded, err := ExpandPath(value)
		if err != nil {
			return err
		}
		c.VaultPath = expanded
	case "default_editor", "editor":
		c.DefaultEditor = value
	case "date_format":
		c.DateFormat = value
	case "time_format":
		c.TimeFormat = value
	default:
		return errors.New("unknown config key (valid keys: vault_path, default_editor, date_format, time_format)")
	}
	return nil
}

// GetValue returns a config field value by key name.
func (c *Config) GetValue(key string) (string, error) {
	switch strings.ToLower(key) {
	case "vault_path", "vault", "path", "folder", "dir":
		return c.VaultPath, nil
	case "default_editor", "editor":
		return c.DefaultEditor, nil
	case "date_format":
		return c.DateFormat, nil
	case "time_format":
		return c.TimeFormat, nil
	default:
		return "", errors.New("unknown config key (valid keys: vault_path, default_editor, date_format, time_format)")
	}
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	if c.ConfigPath == "" {
		cp, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		c.ConfigPath = cp
	}

	dir := filepath.Dir(c.ConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.ConfigPath, data, 0644)
}
