package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var (
	standupDaysFlag int
	standupCopyFlag bool
)

var standupCmd = &cobra.Command{
	Use:     "standup",
	Short:   "Generate daily standup summary (done, in-progress, up next)",
	GroupID: "read",
	Long: `Generate a clean Markdown standup summary containing:
- Tasks completed yesterday (or since Friday if today is Monday)
- Tasks currently in progress
- Tasks up next

Use --copy / -c to copy the report directly to your clipboard for Slack or Teams.

Examples:
  worklog standup            # Print standup report to terminal
  worklog standup --copy     # Print and copy to clipboard
  worklog standup --days 3   # Include tasks completed in the last 3 days
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()

		days := standupDaysFlag
		if days <= 0 {
			if now.Weekday() == time.Monday {
				days = 3 // Look back to Friday
			} else {
				days = 1 // Yesterday
			}
		}

		sinceDate := now.AddDate(0, 0, -days)
		sinceDateStr := sinceDate.Format("2006-01-02")

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), nil)
		if err != nil {
			return fmt.Errorf("failed reading tasks: %w", err)
		}

		var doneTasks []*model.Task
		var doingTasks []*model.Task
		var todoTasks []*model.Task

		for _, t := range tasks {
			switch t.Status {
			case model.StatusDone:
				if t.CompletedAt != nil && !t.CompletedAt.IsZero() {
					if t.CompletedAt.Format("2006-01-02") >= sinceDateStr {
						doneTasks = append(doneTasks, t)
					}
				} else if !t.CreatedAt.IsZero() && t.CreatedAt.Format("2006-01-02") >= sinceDateStr {
					doneTasks = append(doneTasks, t)
				}
			case model.StatusDoing:
				doingTasks = append(doingTasks, t)
			case model.StatusTodo:
				todoTasks = append(todoTasks, t)
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("## 🚀 Standup Summary — %s\n\n", now.Format("Monday, Jan 2")))

		sb.WriteString("### ✅ Completed Recently\n")
		if len(doneTasks) == 0 {
			sb.WriteString("*(no tasks completed)*\n")
		} else {
			for _, t := range doneTasks {
				tagsStr := ""
				if len(t.Tags) > 0 {
					tagsStr = " " + strings.Join(t.Tags, " ")
				}
				lines := strings.Split(t.Text, "\n")
				sb.WriteString(fmt.Sprintf("- %s%s\n", lines[0], tagsStr))
				for i := 1; i < len(lines); i++ {
					trimmed := strings.TrimSpace(lines[i])
					if trimmed != "" {
						sb.WriteString(fmt.Sprintf("  %s\n", trimmed))
					}
				}
			}
		}
		sb.WriteString("\n")

		sb.WriteString("### ⚡ Working on Today\n")
		if len(doingTasks) == 0 {
			sb.WriteString("*(none in progress)*\n")
		} else {
			for _, t := range doingTasks {
				tagsStr := ""
				if len(t.Tags) > 0 {
					tagsStr = " " + strings.Join(t.Tags, " ")
				}
				lines := strings.Split(t.Text, "\n")
				sb.WriteString(fmt.Sprintf("- %s%s\n", lines[0], tagsStr))
				for i := 1; i < len(lines); i++ {
					trimmed := strings.TrimSpace(lines[i])
					if trimmed != "" {
						sb.WriteString(fmt.Sprintf("  %s\n", trimmed))
					}
				}
			}
		}
		sb.WriteString("\n")

		sb.WriteString("### 📋 Up Next\n")
		limit := 5
		if len(todoTasks) < limit {
			limit = len(todoTasks)
		}
		if limit == 0 {
			sb.WriteString("*(backlog empty)*\n")
		} else {
			for i := 0; i < limit; i++ {
				t := todoTasks[i]
				tagsStr := ""
				if len(t.Tags) > 0 {
					tagsStr = " " + strings.Join(t.Tags, " ")
				}
				lines := strings.Split(t.Text, "\n")
				sb.WriteString(fmt.Sprintf("- %s%s\n", lines[0], tagsStr))
				for j := 1; j < len(lines); j++ {
					trimmed := strings.TrimSpace(lines[j])
					if trimmed != "" {
						sb.WriteString(fmt.Sprintf("  %s\n", trimmed))
					}
				}
			}
		}

		rendered := sb.String()
		fmt.Println(rendered)

		if standupCopyFlag {
			if err := copyToClipboard(rendered); err != nil {
				ui.PrintError(os.Stderr, "Could not copy to clipboard: %v", err)
			} else {
				ui.PrintSuccess(os.Stderr, "Standup copied to clipboard!")
			}
		}

		return nil
	},
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch {
	case commandExists("pbcopy"):
		cmd = exec.Command("pbcopy")
	case commandExists("wl-copy"):
		cmd = exec.Command("wl-copy")
	case commandExists("xclip"):
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("no clipboard tool found (pbcopy, wl-copy, or xclip)")
	}

	cmd.Stdin = strings.NewReader(text)
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func init() {
	standupCmd.Flags().IntVarP(&standupDaysFlag, "days", "d", 0, "Number of past days to look back for completed tasks")
	standupCmd.Flags().BoolVarP(&standupCopyFlag, "copy", "c", false, "Copy generated standup summary to system clipboard")
	rootCmd.AddCommand(standupCmd)
}
