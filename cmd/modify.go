package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var forceDelete bool

var rmCmd = &cobra.Command{
	Use:     "rm <selector>",
	Short:   "Delete an entry (interactive confirmation)",
	GroupID: "tools",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := strings.Join(args, " ")

		// Try tasks first
		task, taskErr := storage.FindTaskInFiles(cfg.TasksFilePath(), cfg.JournalFilePath(), selector)
		if taskErr == nil {
			targetFile := task.FilePath
			if targetFile == "" {
				targetFile = cfg.TasksFilePath()
			}
			fmt.Printf("Found Task (#%d): %s\n", task.LineNumber, ui.FormatTaskLine(task, false))
			if !forceDelete {
				if !askConfirmation("Delete this task?") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			_, err := storage.DeleteLine(targetFile, task.LineNumber)
			if err != nil {
				return fmt.Errorf("failed deleting task: %w", err)
			}
			ui.PrintSuccess(os.Stdout, "Task deleted")
			return nil
		}

		// Try journal
		entry, entryErr := storage.FindEntryBySelector(cfg.JournalFilePath(), selector)
		if entryErr == nil {
			fmt.Printf("Found Entry (#%d): %s\n", entry.LineNumber, ui.FormatEntryLine(entry, false))
			if !forceDelete {
				if !askConfirmation("Delete this journal entry?") {
					fmt.Println("Aborted.")
					return nil
				}
			}
			_, err := storage.DeleteLine(cfg.JournalFilePath(), entry.LineNumber)
			if err != nil {
				return fmt.Errorf("failed deleting entry: %w", err)
			}
			ui.PrintSuccess(os.Stdout, "Journal entry deleted")
			return nil
		}

		return fmt.Errorf("no task or journal entry matching selector %q found", selector)
	},
}

var editCmd = &cobra.Command{
	Use:     "edit <selector> [new text]",
	Short:   "Edit entry in place or in $EDITOR",
	GroupID: "tools",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := args[0]
		newText := ""
		if len(args) > 1 {
			newText = strings.Join(args[1:], " ")
		}

		// Try tasks first
		task, taskErr := storage.FindTaskInFiles(cfg.TasksFilePath(), cfg.JournalFilePath(), selector)
		if taskErr == nil {
			targetFile := task.FilePath
			if targetFile == "" {
				targetFile = cfg.TasksFilePath()
			}
			if newText != "" {
				tags := model.ExtractTags(newText)
				cleaned := newText
				for _, t := range tags {
					cleaned = strings.ReplaceAll(cleaned, t, "")
				}
				task.Text = strings.Join(strings.Fields(cleaned), " ")
				task.Tags = tags
				newLine := strings.TrimRight(task.Markdown(), "\n")
				if err := storage.ReplaceLine(targetFile, task.LineNumber, newLine); err != nil {
					return err
				}
				ui.PrintSuccess(os.Stdout, "Task updated: %s", ui.FormatTaskLine(task, true))
				return nil
			}

			// Open in editor
			editedLine, err := editInEditor(task.RawLine)
			if err != nil {
				return err
			}
			if editedLine == "" {
				fmt.Println("No changes made.")
				return nil
			}
			if err := storage.ReplaceLine(targetFile, task.LineNumber, editedLine); err != nil {
				return err
			}
			ui.PrintSuccess(os.Stdout, "Task updated")
			return nil
		}

		// Try journal
		entry, entryErr := storage.FindEntryBySelector(cfg.JournalFilePath(), selector)
		if entryErr == nil {
			if newText != "" {
				tags := model.ExtractTags(newText)
				cleaned := newText
				for _, t := range tags {
					cleaned = strings.ReplaceAll(cleaned, t, "")
				}
				entry.Text = strings.Join(strings.Fields(cleaned), " ")
				entry.Tags = tags
				newLine := strings.TrimRight(entry.Markdown(), "\n")
				if err := storage.ReplaceLine(cfg.JournalFilePath(), entry.LineNumber, newLine); err != nil {
					return err
				}
				ui.PrintSuccess(os.Stdout, "Journal entry updated: %s", ui.FormatEntryLine(entry, true))
				return nil
			}

			// Open in editor
			editedLine, err := editInEditor(entry.RawLine)
			if err != nil {
				return err
			}
			if editedLine == "" {
				fmt.Println("No changes made.")
				return nil
			}
			if err := storage.ReplaceLine(cfg.JournalFilePath(), entry.LineNumber, editedLine); err != nil {
				return err
			}
			ui.PrintSuccess(os.Stdout, "Journal entry updated")
			return nil
		}

		return fmt.Errorf("no task or journal entry matching selector %q found", selector)
	},
}

func askConfirmation(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func editInEditor(initialContent string) (string, error) {
	editor := cfg.GetEditor()
	tempFile, err := os.CreateTemp("", "worklog-edit-*.md")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.WriteString(initialContent); err != nil {
		tempFile.Close()
		return "", err
	}
	tempFile.Close()

	editorParts := strings.Fields(editor)
	editorCmd := exec.Command(editorParts[0], append(editorParts[1:], tempPath)...)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(tempPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func init() {
	rmCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete without confirmation")
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(editCmd)
}
