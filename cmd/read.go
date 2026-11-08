package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var (
	todayTag  string
	todayJSON bool

	yesterdayTag  string
	yesterdayJSON bool

	dayTag  string
	dayJSON bool

	weekTag  string
	weekJSON bool

	monthTag  string
	monthJSON bool

	tasksTag  string
	tasksJSON bool
)

// DayOutput represents a single day's entries and tasks for JSON output.
type DayOutput struct {
	Date    string         `json:"date"`
	Entries []*model.Entry `json:"entries"`
	Tasks   []*model.Task  `json:"tasks"`
}

func parseDateInput(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if t, err := time.ParseInLocation("2006-01-02", input, time.Local); err == nil {
		return t, nil
	}
	now := time.Now()
	if t, err := time.ParseInLocation("01/02", input, time.Local); err == nil {
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.ParseInLocation("01-02", input, time.Local); err == nil {
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	return time.Time{}, fmt.Errorf("invalid date format %q; use YYYY-MM-DD or MM/DD", input)
}

func showDate(targetDate time.Time, tagFilter string, asJSON bool) error {
	targetDateStr := targetDate.Format("2006-01-02")
	isToday := targetDateStr == time.Now().Format("2006-01-02")

	entries, err := storage.ScanEntries(cfg.JournalFilePath(), func(e *model.Entry) bool {
		if e.Timestamp.Format("2006-01-02") != targetDateStr {
			return false
		}
		return model.MatchesTag(e.Tags, tagFilter)
	})
	if err != nil {
		return fmt.Errorf("failed reading journal: %w", err)
	}

	tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), func(t *model.Task) bool {
		if !model.MatchesTag(t.Tags, tagFilter) {
			return false
		}
		// When viewing today, always show in-progress tasks
		if isToday && t.Status == model.StatusDoing {
			return true
		}
		if !t.CreatedAt.IsZero() && t.CreatedAt.Format("2006-01-02") == targetDateStr {
			return true
		}
		if t.CompletedAt != nil && !t.CompletedAt.IsZero() && t.CompletedAt.Format("2006-01-02") == targetDateStr {
			return true
		}
		return false
	})
	if err != nil {
		return fmt.Errorf("failed reading tasks: %w", err)
	}

	if asJSON {
		out := DayOutput{
			Date:    targetDateStr,
			Entries: entries,
			Tasks:   tasks,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	ui.RenderDayGroup(os.Stdout, targetDate, entries, tasks, true)
	return nil
}

var todayCmd = &cobra.Command{
	Use:     "today",
	Short:   "Show today's journal entries and tasks (including in-progress tasks)",
	GroupID: "read",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showDate(time.Now(), todayTag, todayJSON)
	},
}

var yesterdayCmd = &cobra.Command{
	Use:     "yesterday",
	Short:   "Show yesterday's entries and tasks",
	GroupID: "read",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showDate(time.Now().AddDate(0, 0, -1), yesterdayTag, yesterdayJSON)
	},
}

var dayCmd = &cobra.Command{
	Use:     "day <YYYY-MM-DD|MM/DD>",
	Short:   "Show entries and tasks for a specific date",
	GroupID: "read",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDate, err := parseDateInput(args[0])
		if err != nil {
			return err
		}
		return showDate(targetDate, dayTag, dayJSON)
	},
}

var weekCmd = &cobra.Command{
	Use:     "week [n]",
	Short:   "Show last n days (default: 7)",
	GroupID: "read",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		days := 7
		if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
				days = n
			}
		}

		now := time.Now()
		startDate := now.AddDate(0, 0, -(days - 1))
		startStr := startDate.Format("2006-01-02")

		entries, err := storage.ScanEntries(cfg.JournalFilePath(), func(e *model.Entry) bool {
			if e.Timestamp.Format("2006-01-02") < startStr {
				return false
			}
			return model.MatchesTag(e.Tags, weekTag)
		})
		if err != nil {
			return err
		}

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), func(t *model.Task) bool {
			if !model.MatchesTag(t.Tags, weekTag) {
				return false
			}
			return (!t.CreatedAt.IsZero() && t.CreatedAt.Format("2006-01-02") >= startStr) || t.Status == model.StatusDoing
		})
		if err != nil {
			return err
		}

		entriesByDate := make(map[string][]*model.Entry)
		for _, e := range entries {
			d := e.Timestamp.Format("2006-01-02")
			entriesByDate[d] = append(entriesByDate[d], e)
		}

		tasksByDate := make(map[string][]*model.Task)
		for _, t := range tasks {
			d := t.CreatedAt.Format("2006-01-02")
			tasksByDate[d] = append(tasksByDate[d], t)
		}

		var dayOutputs []DayOutput
		for i := days - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			dStr := d.Format("2006-01-02")
			dayEntries := entriesByDate[dStr]
			dayTasks := tasksByDate[dStr]

			if len(dayEntries) > 0 || len(dayTasks) > 0 {
				if weekJSON {
					dayOutputs = append(dayOutputs, DayOutput{
						Date:    dStr,
						Entries: dayEntries,
						Tasks:   dayTasks,
					})
				} else {
					ui.RenderDayGroup(os.Stdout, d, dayEntries, dayTasks, true)
				}
			}
		}

		if weekJSON {
			if dayOutputs == nil {
				dayOutputs = []DayOutput{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(dayOutputs)
		}

		return nil
	},
}

var monthCmd = &cobra.Command{
	Use:     "month [YYYY-MM]",
	Short:   "Show calendar month breakdown",
	GroupID: "read",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		monthPrefix := time.Now().Format("2006-01")
		if len(args) > 0 {
			monthPrefix = args[0]
		}

		entries, err := storage.ScanEntries(cfg.JournalFilePath(), func(e *model.Entry) bool {
			if !strings.HasPrefix(e.Timestamp.Format("2006-01-02"), monthPrefix) {
				return false
			}
			return model.MatchesTag(e.Tags, monthTag)
		})
		if err != nil {
			return err
		}

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), func(t *model.Task) bool {
			if !model.MatchesTag(t.Tags, monthTag) {
				return false
			}
			return !t.CreatedAt.IsZero() && strings.HasPrefix(t.CreatedAt.Format("2006-01-02"), monthPrefix)
		})
		if err != nil {
			return err
		}

		entriesByDate := make(map[string][]*model.Entry)
		for _, e := range entries {
			d := e.Timestamp.Format("2006-01-02")
			entriesByDate[d] = append(entriesByDate[d], e)
		}

		tasksByDate := make(map[string][]*model.Task)
		for _, t := range tasks {
			d := t.CreatedAt.Format("2006-01-02")
			tasksByDate[d] = append(tasksByDate[d], t)
		}

		targetTime, err := time.ParseInLocation("2006-01", monthPrefix, time.Local)
		if err != nil {
			return fmt.Errorf("invalid month format; use YYYY-MM")
		}

		firstDay := time.Date(targetTime.Year(), targetTime.Month(), 1, 0, 0, 0, 0, time.Local)
		lastDay := firstDay.AddDate(0, 1, -1)

		var dayOutputs []DayOutput
		hasAny := false
		for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
			dStr := d.Format("2006-01-02")
			dayEntries := entriesByDate[dStr]
			dayTasks := tasksByDate[dStr]
			if len(dayEntries) > 0 || len(dayTasks) > 0 {
				hasAny = true
				if monthJSON {
					dayOutputs = append(dayOutputs, DayOutput{
						Date:    dStr,
						Entries: dayEntries,
						Tasks:   dayTasks,
					})
				} else {
					ui.RenderDayGroup(os.Stdout, d, dayEntries, dayTasks, true)
				}
			}
		}

		if monthJSON {
			if dayOutputs == nil {
				dayOutputs = []DayOutput{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(dayOutputs)
		}

		if !hasAny {
			fmt.Printf("No activity recorded for %s.\n", monthPrefix)
		}
		return nil
	},
}

var tasksListCmd = &cobra.Command{
	Use:     "tasks [todo|doing|done|all]",
	Short:   "List tasks filtered/grouped by status",
	GroupID: "task",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filterStatus := "all"
		if len(args) > 0 {
			arg := strings.ToLower(args[0])
			switch arg {
			case "todo", "doing", "in-progress", "inprogress", "done", "all":
				filterStatus = arg
			default:
				if strings.HasPrefix(arg, "#") || tasksTag == "" {
					tasksTag = arg
				}
			}
		}

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), func(t *model.Task) bool {
			if !model.MatchesTag(t.Tags, tasksTag) {
				return false
			}

			switch filterStatus {
			case "todo":
				return t.Status == model.StatusTodo
			case "doing", "in-progress", "inprogress":
				return t.Status == model.StatusDoing
			case "done":
				return t.Status == model.StatusDone
			default:
				return true
			}
		})
		if err != nil {
			return err
		}

		if tasksJSON {
			if tasks == nil {
				tasks = []*model.Task{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(tasks)
		}

		if filterStatus == "all" && len(tasks) > 0 {
			var doingTasks, todoTasks, doneTasks []*model.Task
			for _, t := range tasks {
				switch t.Status {
				case model.StatusDoing:
					doingTasks = append(doingTasks, t)
				case model.StatusTodo:
					todoTasks = append(todoTasks, t)
				case model.StatusDone:
					doneTasks = append(doneTasks, t)
				}
			}
			ui.RenderGroupedTasks(os.Stdout, doingTasks, todoTasks, doneTasks, true)
			return nil
		}

		ui.PrintHeader(os.Stdout, fmt.Sprintf("Tasks (%s: %d)", filterStatus, len(tasks)))
		ui.RenderTasks(os.Stdout, tasks, true)
		return nil
	},
}

func init() {
	todayCmd.Flags().StringVarP(&todayTag, "tag", "t", "", "Filter entries and tasks by tag")
	todayCmd.Flags().BoolVar(&todayJSON, "json", false, "Output results as JSON")

	yesterdayCmd.Flags().StringVarP(&yesterdayTag, "tag", "t", "", "Filter entries and tasks by tag")
	yesterdayCmd.Flags().BoolVar(&yesterdayJSON, "json", false, "Output results as JSON")

	dayCmd.Flags().StringVarP(&dayTag, "tag", "t", "", "Filter entries and tasks by tag")
	dayCmd.Flags().BoolVar(&dayJSON, "json", false, "Output results as JSON")

	weekCmd.Flags().StringVarP(&weekTag, "tag", "t", "", "Filter entries and tasks by tag")
	weekCmd.Flags().BoolVar(&weekJSON, "json", false, "Output results as JSON")

	monthCmd.Flags().StringVarP(&monthTag, "tag", "t", "", "Filter entries and tasks by tag")
	monthCmd.Flags().BoolVar(&monthJSON, "json", false, "Output results as JSON")

	tasksListCmd.Flags().StringVarP(&tasksTag, "tag", "t", "", "Filter tasks by tag")
	tasksListCmd.Flags().BoolVar(&tasksJSON, "json", false, "Output results as JSON")

	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(yesterdayCmd)
	rootCmd.AddCommand(dayCmd)
	rootCmd.AddCommand(weekCmd)
	rootCmd.AddCommand(monthCmd)
	rootCmd.AddCommand(tasksListCmd)
}
