package ui

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
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

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape codes from string s.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// VisibleWidth returns the display column count of string s, ignoring ANSI escape codes.
func VisibleWidth(s string) int {
	return utf8.RuneCountInString(StripANSI(s))
}

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

// PrintInfo prints a cyan info message.
func PrintInfo(w io.Writer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(w, "%sℹ%s %s\n", colorCyan, colorReset, msg)
}

// PrintDim prints dimmed text.
func PrintDim(w io.Writer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(w, "%s%s%s", colorDim, msg, colorReset)
}

// PrintHeader prints a section header.
func PrintHeader(w io.Writer, title string) {
	fmt.Fprintf(w, "\n%s%s=== %s ===%s\n", colorBold, colorCyan, title, colorReset)
}

// IndentLines prefixes every non-empty line of s with prefix.
func IndentLines(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// WrapTokens wraps a list of tokens onto lines such that:
// - The first line visual width (including firstLinePrefix) does not exceed maxWidth.
// - Continuation lines start with hangingIndent and their visual width does not exceed maxWidth.
// If a token alone exceeds the available width, it is kept unbroken on the line.
func WrapTokens(tokens []string, firstLinePrefix string, hangingIndent string, maxWidth int) string {
	prefixWidth := VisibleWidth(firstLinePrefix)
	indentWidth := VisibleWidth(hangingIndent)

	if maxWidth <= 0 {
		maxWidth = 80
	}
	if maxWidth < indentWidth+10 {
		maxWidth = indentWidth + 40
	}

	if len(tokens) == 0 {
		return firstLinePrefix
	}

	var lines []string
	var currentLine strings.Builder
	currentLine.WriteString(firstLinePrefix)
	currentVisualLen := prefixWidth
	firstOnLine := true

	for _, token := range tokens {
		tLen := VisibleWidth(token)
		if firstOnLine {
			currentLine.WriteString(token)
			currentVisualLen += tLen
			firstOnLine = false
		} else {
			if currentVisualLen+1+tLen > maxWidth {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(hangingIndent)
				currentLine.WriteString(token)
				currentVisualLen = indentWidth + tLen
			} else {
				currentLine.WriteString(" ")
				currentLine.WriteString(token)
				currentVisualLen += 1 + tLen
			}
		}
	}
	lines = append(lines, currentLine.String())

	return strings.Join(lines, "\n")
}

// FormatTaskLine formats a task with aligned checkbox status, word wrapping, and hanging indents.
func FormatTaskLine(t *model.Task, showLineNumber bool) string {
	return FormatTaskLineWithWidth(t, showLineNumber, GetTerminalWidth())
}

// FormatTaskLineWithWidth formats a task wrapped to the specified maximum column width.
func FormatTaskLineWithWidth(t *model.Task, showLineNumber bool, maxWidth int) string {
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
		fileIndicator := "t:"
		if strings.HasSuffix(t.FilePath, "journal.md") {
			fileIndicator = "j:"
		}
		rawPrefix := fmt.Sprintf("#%s%d", fileIndicator, t.LineNumber)
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

	firstPrefix := linePrefix + coloredBox + timeStr
	hangingIndent := strings.Repeat(" ", indentLen)

	var doneToken string
	if t.CompletedAt != nil {
		doneToken = fmt.Sprintf("%s(done: %s)%s", colorDim, t.CompletedAt.Format("15:04"), colorReset)
	}

	var tagTokens []string
	for _, tag := range t.Tags {
		tagTokens = append(tagTokens, fmt.Sprintf("%s%s%s", colorCyan, tag, colorReset))
	}

	rawParagraphs := strings.Split(t.Text, "\n")
	p0Words := strings.Fields(rawParagraphs[0])
	var p0Tokens []string
	p0Tokens = append(p0Tokens, p0Words...)
	if doneToken != "" {
		p0Tokens = append(p0Tokens, doneToken)
	}
	p0Tokens = append(p0Tokens, tagTokens...)

	res := WrapTokens(p0Tokens, firstPrefix, hangingIndent, maxWidth)

	for i := 1; i < len(rawParagraphs); i++ {
		trimmed := strings.TrimSpace(rawParagraphs[i])
		if trimmed != "" {
			pWords := strings.Fields(trimmed)
			res += "\n" + WrapTokens(pWords, hangingIndent, hangingIndent, maxWidth)
		}
	}

	return res
}

// FormatEntryLine formats a journal entry with aligned bullet, word wrapping, and hanging indents.
func FormatEntryLine(e *model.Entry, showLineNumber bool) string {
	return FormatEntryLineWithWidth(e, showLineNumber, GetTerminalWidth())
}

// FormatEntryLineWithWidth formats a journal entry wrapped to the specified maximum column width.
func FormatEntryLineWithWidth(e *model.Entry, showLineNumber bool, maxWidth int) string {
	var linePrefix string
	indentLen := 0

	if showLineNumber && e.LineNumber > 0 {
		rawPrefix := fmt.Sprintf("#j:%d", e.LineNumber)
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

	firstPrefix := linePrefix + bulletStr + timeStr
	hangingIndent := strings.Repeat(" ", indentLen)

	var tagTokens []string
	for _, tag := range e.Tags {
		tagTokens = append(tagTokens, fmt.Sprintf("%s%s%s", colorCyan, tag, colorReset))
	}

	rawParagraphs := strings.Split(e.Text, "\n")
	p0Words := strings.Fields(rawParagraphs[0])
	var p0Tokens []string
	p0Tokens = append(p0Tokens, p0Words...)
	p0Tokens = append(p0Tokens, tagTokens...)

	res := WrapTokens(p0Tokens, firstPrefix, hangingIndent, maxWidth)

	for i := 1; i < len(rawParagraphs); i++ {
		trimmed := strings.TrimSpace(rawParagraphs[i])
		if trimmed != "" {
			pWords := strings.Fields(trimmed)
			res += "\n" + WrapTokens(pWords, hangingIndent, hangingIndent, maxWidth)
		}
	}

	return res
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

// RenderGroupedTasks prints tasks categorized into Doing, Todo, and Done with aligned hanging indents.
func RenderGroupedTasks(w io.Writer, doing, todo, done []*model.Task, showLineNumber bool) {
	total := len(doing) + len(todo) + len(done)
	PrintHeader(w, fmt.Sprintf("Tasks (%d total)", total))
	availWidth := GetTerminalWidth() - 4

	if len(doing) > 0 {
		fmt.Fprintf(w, "\n  %s%s⚡ In Progress (%d):%s\n", colorBold, colorBlue, len(doing), colorReset)
		for _, t := range doing {
			formatted := FormatTaskLineWithWidth(t, showLineNumber, availWidth)
			fmt.Fprintln(w, IndentLines(formatted, "    "))
		}
	}

	if len(todo) > 0 {
		fmt.Fprintf(w, "\n  %s%s☐ Todo (%d):%s\n", colorBold, colorYellow, len(todo), colorReset)
		for _, t := range todo {
			formatted := FormatTaskLineWithWidth(t, showLineNumber, availWidth)
			fmt.Fprintln(w, IndentLines(formatted, "    "))
		}
	}

	if len(done) > 0 {
		fmt.Fprintf(w, "\n  %s%s✓ Done (%d):%s\n", colorBold, colorGreen, len(done), colorReset)
		for _, t := range done {
			formatted := FormatTaskLineWithWidth(t, showLineNumber, availWidth)
			fmt.Fprintln(w, IndentLines(formatted, "    "))
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

// RenderDayGroup renders both entries and tasks for a given date with aligned hanging indents.
func RenderDayGroup(w io.Writer, date time.Time, entries []*model.Entry, tasks []*model.Task, showLineNumber bool) {
	dateHeading := date.Format("Monday, 2006-01-02")
	fmt.Fprintf(w, "\n%s%s● %s%s\n", colorBold, colorYellow, dateHeading, colorReset)
	availWidth := GetTerminalWidth() - 4

	if len(entries) > 0 {
		fmt.Fprintf(w, "  %sJournal:%s\n", colorBold, colorReset)
		for _, e := range entries {
			formatted := FormatEntryLineWithWidth(e, showLineNumber, availWidth)
			fmt.Fprintln(w, IndentLines(formatted, "    "))
		}
	}

	if len(tasks) > 0 {
		fmt.Fprintf(w, "  %sTasks:%s\n", colorBold, colorReset)
		for _, t := range tasks {
			formatted := FormatTaskLineWithWidth(t, showLineNumber, availWidth)
			fmt.Fprintln(w, IndentLines(formatted, "    "))
		}
	}

	if len(entries) == 0 && len(tasks) == 0 {
		fmt.Fprintf(w, "  %s(no entries or tasks recorded)%s\n", colorDim, colorReset)
	}
}
