package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <term>",
	Short: "Full-text regex search across tasks and journal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := args[0]
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return fmt.Errorf("invalid regular expression: %w", err)
		}

		entries, err := storage.ScanEntries(cfg.JournalFilePath(), func(e *model.Entry) bool {
			return re.MatchString(e.Text) || re.MatchString(e.RawLine)
		})
		if err != nil {
			return err
		}

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), func(t *model.Task) bool {
			return re.MatchString(t.Text) || re.MatchString(t.RawLine)
		})
		if err != nil {
			return err
		}

		fmt.Printf("Search results for %q (Journal: %d, Tasks: %d)\n", pattern, len(entries), len(tasks))

		if len(entries) > 0 {
			ui.PrintHeader(os.Stdout, "Journal Matches")
			ui.RenderEntries(os.Stdout, entries, true)
		}

		if len(tasks) > 0 {
			ui.PrintHeader(os.Stdout, "Task Matches")
			ui.RenderTasks(os.Stdout, tasks, true)
		}

		if len(entries) == 0 && len(tasks) == 0 {
			fmt.Println("No matches found.")
		}

		return nil
	},
}

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List unique tags across both files",
	RunE: func(cmd *cobra.Command, args []string) error {
		tags, counts, err := storage.CollectTags(cfg.TasksFilePath(), cfg.JournalFilePath())
		if err != nil {
			return err
		}

		if len(tags) == 0 {
			fmt.Println("No tags found.")
			return nil
		}

		ui.PrintHeader(os.Stdout, fmt.Sprintf("Tags (%d unique)", len(tags)))
		for _, tag := range tags {
			fmt.Printf("  %-20s %4d\n", tag, counts[tag])
		}

		return nil
	},
}

var tailCmd = &cobra.Command{
	Use:   "tail [n] [journal|tasks]",
	Short: "Show last n lines (default: 10, journal)",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		n := 10
		target := "journal"

		if len(args) >= 1 {
			if count, err := strconv.Atoi(args[0]); err == nil && count > 0 {
				n = count
			} else {
				target = strings.ToLower(args[0])
			}
		}
		if len(args) >= 2 {
			target = strings.ToLower(args[1])
		}

		targetFile := cfg.JournalFilePath()
		if target == "tasks" || target == "task" {
			targetFile = cfg.TasksFilePath()
		}

		lines, err := storage.TailLines(targetFile, n)
		if err != nil {
			return err
		}

		if len(lines) == 0 {
			fmt.Printf("(no entries in %s)\n", target)
			return nil
		}

		for _, line := range lines {
			fmt.Println(line)
		}

		return nil
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Metrics breakdown (item count by date/status)",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := storage.ScanEntries(cfg.JournalFilePath(), nil)
		if err != nil {
			return err
		}

		tasks, err := storage.ScanAllTasks(cfg.TasksFilePath(), cfg.JournalFilePath(), nil)
		if err != nil {
			return err
		}

		todoCount := 0
		doingCount := 0
		doneCount := 0

		activeDays := make(map[string]bool)

		for _, e := range entries {
			activeDays[e.Timestamp.Format("2006-01-02")] = true
		}

		for _, t := range tasks {
			activeDays[t.CreatedAt.Format("2006-01-02")] = true
			switch t.Status {
			case model.StatusTodo:
				todoCount++
			case model.StatusDoing:
				doingCount++
			case model.StatusDone:
				doneCount++
			}
		}

		totalTasks := len(tasks)
		completionRate := 0.0
		if totalTasks > 0 {
			completionRate = (float64(doneCount) / float64(totalTasks)) * 100
		}

		tags, counts, _ := storage.CollectTags(cfg.TasksFilePath(), cfg.JournalFilePath())

		type tagPair struct {
			tag   string
			count int
		}
		var tagPairs []tagPair
		for tag, count := range counts {
			tagPairs = append(tagPairs, tagPair{tag: tag, count: count})
		}
		sort.Slice(tagPairs, func(i, j int) bool {
			return tagPairs[i].count > tagPairs[j].count
		})

		ui.PrintHeader(os.Stdout, "Worklog Metrics & Stats")
		fmt.Printf("  Active Days:          %d\n", len(activeDays))
		fmt.Printf("  Journal Entries:      %d\n", len(entries))
		fmt.Printf("  Total Tasks:          %d\n", totalTasks)
		fmt.Printf("    • Todo:             %d\n", todoCount)
		fmt.Printf("    • Doing:            %d\n", doingCount)
		fmt.Printf("    • Done:             %d\n", doneCount)
		fmt.Printf("  Completion Rate:      %.1f%%\n", completionRate)
		fmt.Printf("  Unique Tags:          %d\n", len(tags))

		if len(tagPairs) > 0 {
			fmt.Println("\n  Top Tags:")
			limit := 5
			if len(tagPairs) < limit {
				limit = len(tagPairs)
			}
			for i := 0; i < limit; i++ {
				fmt.Printf("    %-18s %d\n", tagPairs[i].tag, tagPairs[i].count)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(tagsCmd)
	rootCmd.AddCommand(tailCmd)
	rootCmd.AddCommand(statsCmd)
}
