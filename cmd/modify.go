package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var (
	forceDelete     bool
	rmTaskOnly      bool
	rmJournalOnly   bool
	rmLastFlag      bool
	editTaskOnly    bool
	editJournalOnly bool
)

type TargetItem struct {
	IsTask     bool
	Task       *model.Task
	Entry      *model.Entry
	FilePath   string
	LineNumber int
	RawLine    string
	Display    string
}

func parseSelector(rawSelector string) (cleanSelector string, taskOnly bool, journalOnly bool, isLast bool) {
	s := strings.TrimSpace(rawSelector)
	lower := strings.ToLower(s)

	if lower == "--last" || lower == "last" {
		return "", false, false, true
	}

	// Remove leading '#' if present for prefix parsing
	trimmedHash := strings.TrimPrefix(lower, "#")

	switch {
	case strings.HasPrefix(trimmedHash, "journal:"):
		return strings.TrimPrefix(trimmedHash, "journal:"), false, true, false
	case strings.HasPrefix(trimmedHash, "j:"):
		return strings.TrimPrefix(trimmedHash, "j:"), false, true, false
	case strings.HasPrefix(trimmedHash, "journal#"):
		return strings.TrimPrefix(trimmedHash, "journal#"), false, true, false
	case len(trimmedHash) >= 2 && trimmedHash[0] == 'j' && trimmedHash[1] >= '0' && trimmedHash[1] <= '9':
		return trimmedHash[1:], false, true, false
	case strings.HasPrefix(trimmedHash, "task:") || strings.HasPrefix(trimmedHash, "tasks:"):
		colonIdx := strings.Index(trimmedHash, ":")
		return trimmedHash[colonIdx+1:], true, false, false
	case strings.HasPrefix(trimmedHash, "t:"):
		return strings.TrimPrefix(trimmedHash, "t:"), true, false, false
	case strings.HasPrefix(trimmedHash, "task#"):
		return strings.TrimPrefix(trimmedHash, "task#"), true, false, false
	case len(trimmedHash) >= 2 && trimmedHash[0] == 't' && trimmedHash[1] >= '0' && trimmedHash[1] <= '9':
		return trimmedHash[1:], true, false, false
	}

	if _, err := strconv.Atoi(trimmedHash); err == nil {
		return trimmedHash, false, false, false
	}

	return s, false, false, false
}

func resolveLastItem() (*TargetItem, error) {
	tasks, _ := storage.ScanTasks(cfg.TasksFilePath(), nil)
	entries, _ := storage.ScanEntries(cfg.JournalFilePath(), nil)

	var lastTask *model.Task
	if len(tasks) > 0 {
		lastTask = tasks[len(tasks)-1]
	}

	var lastEntry *model.Entry
	if len(entries) > 0 {
		lastEntry = entries[len(entries)-1]
	}

	if lastTask == nil && lastEntry == nil {
		return nil, fmt.Errorf("no tasks or journal entries found")
	}

	if lastTask != nil && lastEntry == nil {
		return &TargetItem{
			IsTask:     true,
			Task:       lastTask,
			FilePath:   cfg.TasksFilePath(),
			LineNumber: lastTask.LineNumber,
			RawLine:    lastTask.RawLine,
			Display:    fmt.Sprintf("Task (#t:%d):    %s", lastTask.LineNumber, ui.FormatTaskLine(lastTask, false)),
		}, nil
	}

	if lastEntry != nil && lastTask == nil {
		return &TargetItem{
			IsTask:     false,
			Entry:      lastEntry,
			FilePath:   cfg.JournalFilePath(),
			LineNumber: lastEntry.LineNumber,
			RawLine:    lastEntry.RawLine,
			Display:    fmt.Sprintf("Journal (#j:%d): %s", lastEntry.LineNumber, ui.FormatEntryLine(lastEntry, false)),
		}, nil
	}

	// Compare timestamps to pick the most recent one
	if lastEntry.Timestamp.After(lastTask.CreatedAt) {
		return &TargetItem{
			IsTask:     false,
			Entry:      lastEntry,
			FilePath:   cfg.JournalFilePath(),
			LineNumber: lastEntry.LineNumber,
			RawLine:    lastEntry.RawLine,
			Display:    fmt.Sprintf("Journal (#j:%d): %s", lastEntry.LineNumber, ui.FormatEntryLine(lastEntry, false)),
		}, nil
	}

	return &TargetItem{
		IsTask:     true,
		Task:       lastTask,
		FilePath:   cfg.TasksFilePath(),
		LineNumber: lastTask.LineNumber,
		RawLine:    lastTask.RawLine,
		Display:    fmt.Sprintf("Task (#t:%d):    %s", lastTask.LineNumber, ui.FormatTaskLine(lastTask, false)),
	}, nil
}

func resolveTarget(rawSelector string, forceTask, forceJournal, force bool) (*TargetItem, error) {
	cleanSelector, prefixTask, prefixJournal, isLast := parseSelector(rawSelector)
	if isLast {
		return resolveLastItem()
	}

	taskOnly := forceTask || prefixTask
	journalOnly := forceJournal || prefixJournal

	var candidates []TargetItem

	// Search Tasks
	if !journalOnly {
		task, err := storage.FindTaskBySelector(cfg.TasksFilePath(), cleanSelector)
		if err == nil && task != nil {
			candidates = append(candidates, TargetItem{
				IsTask:     true,
				Task:       task,
				FilePath:   cfg.TasksFilePath(),
				LineNumber: task.LineNumber,
				RawLine:    task.RawLine,
				Display:    fmt.Sprintf("Task (#t:%d):    %s", task.LineNumber, ui.FormatTaskLine(task, false)),
			})
		}
	}

	// Search Journal
	if !taskOnly {
		entry, err := storage.FindEntryBySelector(cfg.JournalFilePath(), cleanSelector)
		if err == nil && entry != nil {
			candidates = append(candidates, TargetItem{
				IsTask:     false,
				Entry:      entry,
				FilePath:   cfg.JournalFilePath(),
				LineNumber: entry.LineNumber,
				RawLine:    entry.RawLine,
				Display:    fmt.Sprintf("Journal (#j:%d): %s", entry.LineNumber, ui.FormatEntryLine(entry, false)),
			})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no task or journal entry matching %q found", rawSelector)
	}

	if len(candidates) == 1 {
		return &candidates[0], nil
	}

	if force {
		return nil, fmt.Errorf("ambiguous selector %q matches multiple items; please specify 'j:%s' for journal or 't:%s' for task", rawSelector, cleanSelector, cleanSelector)
	}

	// Multiple candidates matched: ask user to disambiguate
	fmt.Printf("\nMultiple items match %q:\n", rawSelector)
	for i, c := range candidates {
		fmt.Printf("  [%d] %s\n", i+1, c.Display)
	}

	fmt.Printf("Select an item (1-%d) or 'q' to cancel: ", len(candidates))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("selection aborted")
	}
	input = strings.TrimSpace(input)
	if strings.ToLower(input) == "q" || input == "" {
		return nil, fmt.Errorf("operation cancelled")
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(candidates) {
		return nil, fmt.Errorf("invalid choice %q; operation cancelled", input)
	}

	return &candidates[idx-1], nil
}

var rmCmd = &cobra.Command{
	Use:     "rm <selector>",
	Short:   "Delete an entry (interactive confirmation)",
	GroupID: "tools",
	Long: `Delete a task or journal entry by line number, prefix, or text match.

Selectors:
  worklog rm 1                # Matches line 1 (prompts if both task and journal have line 1)
  worklog rm j:1              # Specifically deletes line 1 in journal (aliases: journal:1, j1, #j:1)
  worklog rm t:1              # Specifically deletes line 1 in tasks (aliases: task:1, t1, #t:1)
  worklog rm --last           # Deletes the most recent entry or task
  worklog rm -j 1             # Flag option: target journal only
  worklog rm -t 1             # Flag option: target tasks only
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if rmLastFlag {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("requires a selector or --last flag")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		rawSelector := ""
		if rmLastFlag {
			rawSelector = "--last"
		} else if len(args) > 0 {
			rawSelector = strings.Join(args, " ")
		}
		item, err := resolveTarget(rawSelector, rmTaskOnly, rmJournalOnly, forceDelete)
		if err != nil {
			return err
		}

		fmt.Printf("Found %s\n", item.Display)
		if !forceDelete {
			prompt := "Delete this item?"
			if item.IsTask {
				prompt = "Delete this task?"
			} else {
				prompt = "Delete this journal entry?"
			}
			if !askConfirmation(prompt) {
				fmt.Println("Aborted.")
				return nil
			}
		}

		_, err = storage.DeleteLine(item.FilePath, item.LineNumber)
		if err != nil {
			return fmt.Errorf("failed deleting line: %w", err)
		}

		if item.IsTask {
			ui.PrintSuccess(os.Stdout, "Task deleted (#%d)", item.LineNumber)
		} else {
			ui.PrintSuccess(os.Stdout, "Journal entry deleted (#%d)", item.LineNumber)
		}
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:     "edit <selector> [new text]",
	Short:   "Edit entry in place or in $EDITOR",
	GroupID: "tools",
	Long: `Edit a task or journal entry in place or in $EDITOR.

Selectors:
  worklog edit 1 [new text]         # Edit line 1 (prompts if ambiguous)
  worklog edit j:1 [new text]       # Specifically edit journal line 1
  worklog edit t:1 [new text]       # Specifically edit task line 1
  worklog edit 1                    # Opens item in $EDITOR
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawSelector := args[0]
		newText := ""
		if len(args) > 1 {
			newText = strings.Join(args[1:], " ")
		}

		item, err := resolveTarget(rawSelector, editTaskOnly, editJournalOnly, false)
		if err != nil {
			return err
		}

		if item.IsTask {
			task := item.Task
			if newText != "" {
				cleanedText, tags := model.SplitTags(newText)
				task.Text = cleanedText
				task.Tags = tags
				newLine := strings.TrimRight(task.Markdown(), "\n")
				if err := storage.ReplaceLine(item.FilePath, item.LineNumber, newLine); err != nil {
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
			if err := storage.ReplaceLine(item.FilePath, item.LineNumber, editedLine); err != nil {
				return err
			}
			ui.PrintSuccess(os.Stdout, "Task updated")
			return nil
		}

		// Edit journal entry
		entry := item.Entry
		if newText != "" {
			cleanedText, tags := model.SplitTags(newText)
			entry.Text = cleanedText
			entry.Tags = tags
			newLine := strings.TrimRight(entry.Markdown(), "\n")
			if err := storage.ReplaceLine(item.FilePath, item.LineNumber, newLine); err != nil {
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
		if err := storage.ReplaceLine(item.FilePath, item.LineNumber, editedLine); err != nil {
			return err
		}
		ui.PrintSuccess(os.Stdout, "Journal entry updated")
		return nil
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
	rmCmd.Flags().BoolVarP(&rmTaskOnly, "task", "t", false, "Target tasks only")
	rmCmd.Flags().BoolVarP(&rmJournalOnly, "journal", "j", false, "Target journal only")
	rmCmd.Flags().BoolVar(&rmLastFlag, "last", false, "Delete the most recent entry or task")

	editCmd.Flags().BoolVarP(&editTaskOnly, "task", "t", false, "Target tasks only")
	editCmd.Flags().BoolVarP(&editJournalOnly, "journal", "j", false, "Target journal only")

	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(editCmd)
}
