package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task [todo|doing|done] <text>",
	Short: "Add a task or manage task state",
	Long: `Add a new task (defaulting to todo) or toggle an existing task.

Examples:
  worklog task Deploy new release #devops
  worklog task doing Investigating memory leak
  worklog task done Reviewed PR #123
  worklog task toggle 1
  worklog task toggle memory
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		status := model.StatusTodo
		taskTextArgs := args

		switch strings.ToLower(args[0]) {
		case "todo":
			status = model.StatusTodo
			taskTextArgs = args[1:]
		case "doing":
			status = model.StatusDoing
			taskTextArgs = args[1:]
		case "done":
			status = model.StatusDone
			taskTextArgs = args[1:]
		}

		rawText := strings.TrimSpace(strings.Join(taskTextArgs, " "))
		if rawText == "" {
			return fmt.Errorf("task description cannot be empty")
		}

		now := time.Now()
		var completedAt *time.Time
		if status == model.StatusDone {
			completedAt = &now
		}

		tags := model.ExtractTags(rawText)
		cleanedText := rawText
		for _, tag := range tags {
			cleanedText = strings.ReplaceAll(cleanedText, tag, "")
		}
		cleanedText = strings.Join(strings.Fields(cleanedText), " ")

		task := &model.Task{
			Status:      status,
			CreatedAt:   now,
			CompletedAt: completedAt,
			Text:        cleanedText,
			Tags:        tags,
		}

		if err := storage.AppendTask(cfg.TasksFilePath(), task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		ui.PrintSuccess(os.Stdout, "Task created: %s", ui.FormatTaskLine(task, false))
		return nil
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle <query|index>",
	Short: "Cycle task state: todo -> doing -> done -> todo",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := strings.Join(args, " ")
		task, err := storage.FindTaskBySelector(cfg.TasksFilePath(), selector)
		if err != nil {
			return err
		}

		now := time.Now()
		updated, err := storage.ToggleTask(cfg.TasksFilePath(), task.LineNumber, now)
		if err != nil {
			return err
		}

		ui.PrintSuccess(os.Stdout, "Task updated: %s", ui.FormatTaskLine(updated, true))
		return nil
	},
}

var doneCmd = &cobra.Command{
	Use:   "done <text>",
	Short: "Add an already-completed task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rawText := strings.TrimSpace(strings.Join(args, " "))
		now := time.Now()
		tags := model.ExtractTags(rawText)
		cleanedText := rawText
		for _, tag := range tags {
			cleanedText = strings.ReplaceAll(cleanedText, tag, "")
		}
		cleanedText = strings.Join(strings.Fields(cleanedText), " ")

		task := &model.Task{
			Status:      model.StatusDone,
			CreatedAt:   now,
			CompletedAt: &now,
			Text:        cleanedText,
			Tags:        tags,
		}

		if err := storage.AppendTask(cfg.TasksFilePath(), task); err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}

		ui.PrintSuccess(os.Stdout, "Task completed: %s", ui.FormatTaskLine(task, false))
		return nil
	},
}

func init() {
	taskCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(doneCmd)
}
