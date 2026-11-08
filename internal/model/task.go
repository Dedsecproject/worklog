package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type TaskStatus string

const (
	StatusTodo  TaskStatus = "todo"  // "- [ ]"
	StatusDoing TaskStatus = "doing" // "- [/]"
	StatusDone  TaskStatus = "done"  // "- [x]" or "- [X]"
)

const (
	TimestampFormat = "2006-01-02 15:04"
	DateFormat      = "2006-01-02"
	TasksFileName   = "tasks.md"
	JournalFileName = "journal.md"
)

var (
	doneTagRegex     = regexp.MustCompile(`\(done:\s*(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})\)`)
	taskPrefixRegex  = regexp.MustCompile(`^-\s*\[(\s*|\/|x|X|-)\]\s*`)
	mdCommentRegex   = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// StripMarkdownComments removes markdown comments <!-- ... --> from a string.
func StripMarkdownComments(s string) string {
	return mdCommentRegex.ReplaceAllString(s, "")
}

type Task struct {
	FilePath    string
	RawLine     string
	LineNumber  int
	Status      TaskStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
	Text        string
	Tags        []string
}

// Markdown formats the task as a Markdown checkbox entry.
// Undated tasks (zero CreatedAt) are written without a timestamp rather than being stamped "now".
func (t Task) Markdown() string {
	var box string
	switch t.Status {
	case StatusDoing:
		box = "[/]"
	case StatusDone:
		box = "[x]"
	default:
		box = "[ ]"
	}

	dateStr := ""
	if !t.CreatedAt.IsZero() {
		dateStr = t.CreatedAt.Format(TimestampFormat) + " "
	}

	var doneStr string
	if t.CompletedAt != nil {
		doneStr = fmt.Sprintf(" (done: %s)", t.CompletedAt.Format(TimestampFormat))
	}

	tagsStr := ""
	if len(t.Tags) > 0 {
		tagsStr = " " + strings.Join(t.Tags, " ")
	}

	lines := strings.Split(t.Text, "\n")
	res := fmt.Sprintf("- %s %s%s%s%s\n", box, dateStr, lines[0], doneStr, tagsStr)
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			res += fmt.Sprintf("  %s\n", trimmed)
		}
	}
	return res
}

// Toggle advances the task status: todo -> doing -> done -> todo.
// When transitioning to done, CompletedAt is set to now.
// When transitioning away from done, CompletedAt is set to nil.
func (t *Task) Toggle(now time.Time) {
	switch t.Status {
	case StatusTodo:
		t.Status = StatusDoing
		t.CompletedAt = nil
	case StatusDoing:
		t.Status = StatusDone
		t.CompletedAt = &now
	case StatusDone:
		t.Status = StatusTodo
		t.CompletedAt = nil
	default:
		t.Status = StatusTodo
		t.CompletedAt = nil
	}
}

// ParseTask parses a line into a Task.
// Supported prefixes:
// - [ ] (todo)
// - [/] (doing / in-progress)
// - [-] (doing / cancelled)
// - [x] or - [X] (done)
func ParseTask(line string, lineNumber int) (*Task, error) {
	trimmed := strings.TrimRight(line, "\r\n")
	trimmedLeading := strings.TrimSpace(trimmed)

	prefixMatch := taskPrefixRegex.FindStringSubmatch(trimmedLeading)
	if len(prefixMatch) == 0 {
		return nil, fmt.Errorf("line %d: not a valid task format", lineNumber)
	}

	marker := strings.TrimSpace(prefixMatch[1])
	var status TaskStatus
	switch marker {
	case "/", "-":
		status = StatusDoing
	case "x", "X":
		status = StatusDone
	default:
		status = StatusTodo
	}

	content := strings.TrimSpace(trimmedLeading[len(prefixMatch[0]):])
	content = StripMarkdownComments(content)

	createdAt, content := parseLeadingTimestamp(content)

	// Check for (done: YYYY-MM-DD HH:MM)
	var completedAt *time.Time
	if doneMatches := doneTagRegex.FindStringSubmatch(content); len(doneMatches) > 1 {
		if ct, err := time.ParseInLocation(TimestampFormat, doneMatches[1], time.Local); err == nil {
			completedAt = &ct
			content = doneTagRegex.ReplaceAllString(content, "")
		}
	}

	cleanedText, tags := SplitTags(content)

	return &Task{
		RawLine:     line,
		LineNumber:  lineNumber,
		Status:      status,
		CreatedAt:   createdAt,
		CompletedAt: completedAt,
		Text:        cleanedText,
		Tags:        tags,
	}, nil
}
