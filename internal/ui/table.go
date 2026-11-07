package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"worklog/internal/model"
)

var (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

func init() {
	if os.Getenv("NO_COLOR") != "" {
		DisableColor()
	}
}

// DisableColor strips ANSI escape sequences for raw or piped outputs.
func DisableColor() {
	colorReset = ""
	colorBold = ""
	colorDim = ""
	colorRed = ""
	colorGreen = ""
	colorYellow = ""
	colorBlue = ""
	colorCyan = ""
	colorGray = ""
}

// PrintSuccess prints a green success message.
func PrintSuccess(w io.Writer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(w, "%s✓%s %s\n", colorGreen, colorReset, msg)
}

// PrintError prints a red error message.
func PrintError(w io.Writer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(w, "%s✗%s %s\n", colorRed, colorReset, msg)
}

// PrintHeader prints a section header.
func PrintHeader(w io.Writer, title string) {
	fmt.Fprintf(w, "\n%s%s=== %s ===%s\n", colorBold, colorCyan, title, colorReset)
}

// FormatTaskLine formats a task for terminal display.
func FormatTaskLine(t *model.Task, showLineNumber bool) string {
	var box string
	var statusColor string

	switch t.Status {
	case model.StatusDone:
		box = "[x]"
		statusColor = colorGreen
	case model.StatusDoing:
		box = "[/]"
		statusColor = colorBlue
	default:
		box = "[ ]"
		statusColor = colorYellow
	}

	linePrefix := ""
	if showLineNumber && t.LineNumber > 0 {
		linePrefix = fmt.Sprintf("%s#%-3d%s ", colorGray, t.LineNumber, colorReset)
	}

	timeStr := ""
	if !t.CreatedAt.IsZero() {
		timeStr = fmt.Sprintf("%s%s%s ", colorDim, t.CreatedAt.Format("15:04"), colorReset)
	}

	doneStr := ""
	if t.CompletedAt != nil {
		doneStr = fmt.Sprintf(" %s(done: %s)%s", colorDim, t.CompletedAt.Format("15:04"), colorReset)
	}

	tagsStr := ""
	if len(t.Tags) > 0 {
		tagsStr = fmt.Sprintf(" %s%s%s", colorCyan, strings.Join(t.Tags, " "), colorReset)
	}

	return fmt.Sprintf("%s%s%s%s %s%s%s%s", linePrefix, statusColor, box, colorReset, timeStr, t.Text, doneStr, tagsStr)
}

// FormatEntryLine formats a journal entry for terminal display.
func FormatEntryLine(e *model.Entry, showLineNumber bool) string {
	linePrefix := ""
	if showLineNumber && e.LineNumber > 0 {
		linePrefix = fmt.Sprintf("%s#%-3d%s ", colorGray, e.LineNumber, colorReset)
	}

	timeStr := ""
	if !e.Timestamp.IsZero() {
		timeStr = fmt.Sprintf("%s%s%s ", colorDim, e.Timestamp.Format("15:04"), colorReset)
	}

	tagsStr := ""
	if len(e.Tags) > 0 {
		tagsStr = fmt.Sprintf(" %s%s%s", colorCyan, strings.Join(e.Tags, " "), colorReset)
	}

	return fmt.Sprintf("%s%s•%s %s%s%s", linePrefix, colorCyan, colorReset, timeStr, e.Text, tagsStr)
}

// RenderTasks prints a slice of tasks.
func RenderTasks(w io.Writer, tasks []*model.Task, showLineNumber bool) {
	if len(tasks) == 0 {
		fmt.Fprintf(w, "%s(no tasks)%s\n", colorDim, colorReset)
		return
	}
	for _, t := range tasks {
		fmt.Fprintln(w, FormatTaskLine(t, showLineNumber))
	}
}

// RenderEntries prints a slice of journal entries.
func RenderEntries(w io.Writer, entries []*model.Entry, showLineNumber bool) {
	if len(entries) == 0 {
		fmt.Fprintf(w, "%s(no journal entries)%s\n", colorDim, colorReset)
		return
	}
	for _, e := range entries {
		fmt.Fprintln(w, FormatEntryLine(e, showLineNumber))
	}
}

// RenderDayGroup renders both entries and tasks for a given date.
func RenderDayGroup(w io.Writer, date time.Time, entries []*model.Entry, tasks []*model.Task, showLineNumber bool) {
	dateHeading := date.Format("Monday, 2006-01-02")
	fmt.Fprintf(w, "\n%s%s● %s%s\n", colorBold, colorYellow, dateHeading, colorReset)

	if len(entries) > 0 {
		fmt.Fprintf(w, "  %sJournal:%s\n", colorBold, colorReset)
		for _, e := range entries {
			fmt.Fprintf(w, "    %s\n", FormatEntryLine(e, showLineNumber))
		}
	}

	if len(tasks) > 0 {
		fmt.Fprintf(w, "  %sTasks:%s\n", colorBold, colorReset)
		for _, t := range tasks {
			fmt.Fprintf(w, "    %s\n", FormatTaskLine(t, showLineNumber))
		}
	}

	if len(entries) == 0 && len(tasks) == 0 {
		fmt.Fprintf(w, "  %s(no entries or tasks recorded)%s\n", colorDim, colorReset)
	}
}
