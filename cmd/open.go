package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [journal|tasks]",
	Short: "Open the target file in $EDITOR",
	Args:  cobra.MaximumNArgs(1),
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
	Use:   "export [csv|json]",
	Short: "Dump data to stdout in CSV or JSON format",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format := "json"
		if len(args) > 0 {
			format = strings.ToLower(args[0])
		}

		entries, err := storage.ScanEntries(cfg.JournalFilePath(), nil)
		if err != nil {
			return err
		}

		tasks, err := storage.ScanTasks(cfg.TasksFilePath(), nil)
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

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.PrintHeader(os.Stdout, "worklog Configuration")
		fmt.Printf("  Vault Path:      %s\n", cfg.VaultPath)
		fmt.Printf("  Tasks File:      %s\n", cfg.TasksFilePath())
		fmt.Printf("  Journal File:    %s\n", cfg.JournalFilePath())
		fmt.Printf("  Editor:          %s\n", cfg.GetEditor())
		fmt.Printf("  Date Format:     %s\n", cfg.DateFormat)
		fmt.Printf("  Time Format:     %s\n", cfg.TimeFormat)
		if cfg.ConfigPath != "" {
			fmt.Printf("  Config File:     %s\n", cfg.ConfigPath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(configCmd)
}
