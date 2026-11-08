package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"worklog/internal/config"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:     "open [journal|tasks]",
	Short:   "Open the target file in $EDITOR",
	GroupID: "tools",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "journal"
		if len(args) > 0 {
			target = strings.ToLower(args[0])
		}

		filePath := cfg.JournalFilePath()
		if target == "tasks" || target == "task" {
			filePath = cfg.TasksFilePath()
		}

		editor := cfg.GetEditor()
		editorParts := strings.Fields(editor)
		editorCmd := exec.Command(editorParts[0], append(editorParts[1:], filePath)...)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		return editorCmd.Run()
	},
}

var exportCmd = &cobra.Command{
	Use:     "export [csv|json]",
	Short:   "Dump data to stdout in CSV or JSON format",
	GroupID: "tools",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format := "json"
		if len(args) > 0 {
			format = strings.ToLower(args[0])
		}

		entries, err := storage.ScanEntries(cfg.JournalFilePath(), nil)
		if err != nil {
			return err
		}

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), nil)
		if err != nil {
			return err
		}

		if format == "csv" {
			w := csv.NewWriter(os.Stdout)
			defer w.Flush()

			// Write tasks
			_ = w.Write([]string{"TYPE", "STATUS", "CREATED_AT", "COMPLETED_AT", "TEXT", "TAGS"})
			for _, t := range tasks {
				completed := ""
				if t.CompletedAt != nil {
					completed = t.CompletedAt.Format("2006-01-02 15:04")
				}
				_ = w.Write([]string{
					"task",
					string(t.Status),
					t.CreatedAt.Format("2006-01-02 15:04"),
					completed,
					t.Text,
					strings.Join(t.Tags, " "),
				})
			}

			// Write journal entries
			for _, e := range entries {
				_ = w.Write([]string{
					"journal",
					"",
					e.Timestamp.Format("2006-01-02 15:04"),
					"",
					e.Text,
					strings.Join(e.Tags, " "),
				})
			}
			return nil
		}

		type exportPayload struct {
			Tasks   []*model.Task  `json:"tasks"`
			Journal []*model.Entry `json:"journal"`
		}

		payload := exportPayload{
			Tasks:   tasks,
			Journal: entries,
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(payload)
	},
}

var (
	configEditFlag   bool
	configVaultFlag  string
	configEditorFlag string
)

var configCmd = &cobra.Command{
	Use:     "config [key] [value]",
	Short:   "Show or update configuration settings",
	GroupID: "tools",
	Long: `Show or update current configuration settings.

Examples:
  worklog config                               # Display current configuration
  worklog config vault_path ~/Documents/Vault  # Update vault / storage directory
  worklog config editor nvim                   # Update default editor
  worklog config vault_path                    # Show current vault path
  worklog config --edit                        # Open config.json in $EDITOR
`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Open in editor flag
		if configEditFlag {
			if cfg.ConfigPath == "" {
				cp, err := config.DefaultConfigPath()
				if err != nil {
					return err
				}
				cfg.ConfigPath = cp
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			editor := cfg.GetEditor()
			editorParts := strings.Fields(editor)
			editorCmd := exec.Command(editorParts[0], append(editorParts[1:], cfg.ConfigPath)...)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			return editorCmd.Run()
		}

		// Update via flags
		updated := false
		if configVaultFlag != "" {
			if err := cfg.SetValue("vault_path", configVaultFlag); err != nil {
				return err
			}
			updated = true
		}
		if configEditorFlag != "" {
			if err := cfg.SetValue("default_editor", configEditorFlag); err != nil {
				return err
			}
			updated = true
		}

		// Update via positional args: worklog config <key> <value>
		if len(args) == 2 {
			key, val := args[0], args[1]
			if err := cfg.SetValue(key, val); err != nil {
				return err
			}
			updated = true
		} else if len(args) == 1 && !updated {
			// Query single key: worklog config <key>
			val, err := cfg.GetValue(args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		}

		if updated {
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed saving config: %w", err)
			}
			if err := cfg.EnsureDir(); err != nil {
				return fmt.Errorf("failed initializing vault directory: %w", err)
			}
			ui.PrintSuccess(os.Stdout, "Configuration updated and saved to %s", cfg.ConfigPath)
		}

		// Always display configuration
		ui.PrintHeader(os.Stdout, "worklog Configuration")
		fmt.Printf("  Vault Path:      %s\n", cfg.VaultPath)
		fmt.Printf("  Tasks File:      %s\n", cfg.TasksFilePath())
		fmt.Printf("  Journal File:    %s\n", cfg.JournalFilePath())
		fmt.Printf("  Editor:          %s\n", cfg.GetEditor())
		fmt.Printf("  Date Format:     %s\n", cfg.DateFormat)
		fmt.Printf("  Time Format:     %s\n", cfg.TimeFormat)
		fmt.Printf("  Auto Archive:    %t\n", cfg.IsAutoArchiveEnabled())
		fmt.Printf("  Archive Dir:     %s\n", cfg.ArchiveDirPath())
		if cfg.ConfigPath != "" {
			fmt.Printf("  Config File:     %s\n", cfg.ConfigPath)
		}

		if !updated {
			fmt.Printf("\n  %sTo modify settings:%s\n", "\033[2m", "\033[0m")
			fmt.Printf("    worklog config vault_path <path>\n")
			fmt.Printf("    worklog config auto_archive <true|false>\n")
			fmt.Printf("    worklog config archive_dir <folder>\n")
			fmt.Printf("    worklog config editor <name>\n")
			fmt.Printf("    worklog config --edit\n")
		}

		return nil
	},
}

func init() {
	configCmd.Flags().BoolVarP(&configEditFlag, "edit", "e", false, "Open config file in $EDITOR")
	configCmd.Flags().StringVar(&configVaultFlag, "vault", "", "Set vault / storage directory path")
	configCmd.Flags().StringVar(&configEditorFlag, "editor", "", "Set default editor")
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(configCmd)
}
