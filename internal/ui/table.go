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

// FormatTaskLine formats a task with aligned checkbox status and hanging indents for multi-line text.
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

	var linePrefix string
	indentLen := 0

	if showLineNumber && t.LineNumber > 0 {
		fileIndicator := ""
		if strings.HasSuffix(t.FilePath, "journal.md") {
			fileIndicator = "j:"
		}
		rawPrefix := fmt.Sprintf("#%s%d", fileIndicator, t.LineNumber)
		// Fixed 6-character width for line prefix so checkboxes always align
		linePrefix = fmt.Sprintf("%s%-6s%s ", colorGray, rawPrefix, colorReset)
		indentLen += 7
	}

	coloredBox := fmt.Sprintf("%s%s%s ", statusColor, box, colorReset)
	indentLen += 4

	timeStr := "      "
	if !t.CreatedAt.IsZero() {
		timeStr = fmt.Sprintf("%s%s%s ", colorDim, t.CreatedAt.Format("15:04"), colorReset)
	}
	indentLen += 6

	doneStr := ""
	if t.CompletedAt != nil {
		doneStr = fmt.Sprintf(" %s(done: %s)%s", colorDim, t.CompletedAt.Format("15:04"), colorReset)
	}

	tagsStr := ""
	if len(t.Tags) > 0 {
		tagsStr = fmt.Sprintf(" %s%s%s", colorCyan, strings.Join(t.Tags, " "), colorReset)
	}

	lines := strings.Split(t.Text, "\n")
	hangingIndent := strings.Repeat(" ", indentLen)

	var sb strings.Builder
	sb.WriteString(linePrefix)
	sb.WriteString(coloredBox)
	sb.WriteString(timeStr)
	sb.WriteString(lines[0])
	sb.WriteString(doneStr)
	sb.WriteString(tagsStr)

	// Hanging indent for multi-line tasks
	for i := 1; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])
		if trimmedLine != "" {
			sb.WriteString("\n")
			sb.WriteString(hangingIndent)
			sb.WriteString(trimmedLine)
		}
	}

	return sb.String()
}

// FormatEntryLine formats a journal entry with aligned bullet and hanging indents for multi-line text.
func FormatEntryLine(e *model.Entry, showLineNumber bool) string {
	var linePrefix string
	indentLen := 0

	if showLineNumber && e.LineNumber > 0 {
		rawPrefix := fmt.Sprintf("#%d", e.LineNumber)
		linePrefix = fmt.Sprintf("%s%-6s%s ", colorGray, rawPrefix, colorReset)
		indentLen += 7
	}

	bulletStr := fmt.Sprintf("%s•%s ", colorCyan, colorReset)
	indentLen += 2

	timeStr := "      "
	if !e.Timestamp.IsZero() {
		timeStr = fmt.Sprintf("%s%s%s ", colorDim, e.Timestamp.Format("15:04"), colorReset)
	}
	indentLen += 6

	tagsStr := ""
	if len(e.Tags) > 0 {
		tagsStr = fmt.Sprintf(" %s%s%s", colorCyan, strings.Join(e.Tags, " "), colorReset)
	}

	lines := strings.Split(e.Text, "\n")
	hangingIndent := strings.Repeat(" ", indentLen)

	var sb strings.Builder
	sb.WriteString(linePrefix)
	sb.WriteString(bulletStr)
	sb.WriteString(timeStr)
	sb.WriteString(lines[0])
	sb.WriteString(tagsStr)

	// Hanging indent for multi-line entries
	for i := 1; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])
		if trimmedLine != "" {
			sb.WriteString("\n")
			sb.WriteString(hangingIndent)
			sb.WriteString(trimmedLine)
		}
	}

	return sb.String()
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

// RenderGroupedTasks prints tasks categorized into Doing, Todo, and Done.
func RenderGroupedTasks(w io.Writer, doing, todo, done []*model.Task, showLineNumber bool) {
	total := len(doing) + len(todo) + len(done)
	PrintHeader(w, fmt.Sprintf("Tasks (%d total)", total))

	if len(doing) > 0 {
		fmt.Fprintf(w, "\n  %s%s⚡ In Progress (%d):%s\n", colorBold, colorBlue, len(doing), colorReset)
		for _, t := range doing {
			fmt.Fprintf(w, "    %s\n", FormatTaskLine(t, showLineNumber))
		}
	}

	if len(todo) > 0 {
		fmt.Fprintf(w, "\n  %s%s☐ Todo (%d):%s\n", colorBold, colorYellow, len(todo), colorReset)
		for _, t := range todo {
			fmt.Fprintf(w, "    %s\n", FormatTaskLine(t, showLineNumber))
		}
	}

	if len(done) > 0 {
		fmt.Fprintf(w, "\n  %s%s✓ Done (%d):%s\n", colorBold, colorGreen, len(done), colorReset)
		for _, t := range done {
			fmt.Fprintf(w, "    %s\n", FormatTaskLine(t, showLineNumber))
		}
	}

	if total == 0 {
		fmt.Fprintf(w, "%s(no tasks)%s\n", colorDim, colorReset)
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
