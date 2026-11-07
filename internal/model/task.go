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
)

var (
	tagRegex     = regexp.MustCompile(`#[a-zA-Z0-9_\-]+`)
	doneTagRegex = regexp.MustCompile(`\(done:\s*(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})\)`)
)

type Task struct {
	RawLine     string
	LineNumber  int
	Status      TaskStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
	Text        string
	Tags        []string
}

// Markdown formats the task as a Markdown checkbox entry.
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

	dateStr := t.CreatedAt.Format(TimestampFormat)
	var doneStr string
	if t.CompletedAt != nil {
		doneStr = fmt.Sprintf(" (done: %s)", t.CompletedAt.Format(TimestampFormat))
	}

	tagsStr := ""
	if len(t.Tags) > 0 {
		tagsStr = " " + strings.Join(t.Tags, " ")
	}

	return fmt.Sprintf("- %s %s %s%s%s\n", box, dateStr, t.Text, doneStr, tagsStr)
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

// ExtractTags finds all hashtags matching #[a-zA-Z0-9_\-]+
func ExtractTags(s string) []string {
	matches := tagRegex.FindAllString(s, -1)
	if len(matches) == 0 {
		return nil
	}
	// Deduplicate preserving order
	seen := make(map[string]bool)
	var tags []string
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			tags = append(tags, match)
		}
	}
	return tags
}

// ParseTask parses a line into a Task.
// Supported prefixes:
// - [ ] (todo)
// - [/] (doing)
// - [x] or - [X] (done)
func ParseTask(line string, lineNumber int) (*Task, error) {
	trimmed := strings.TrimRight(line, "\r\n")
	trimmedLeading := strings.TrimSpace(trimmed)

	var status TaskStatus
	var content string

	switch {
	case strings.HasPrefix(trimmedLeading, "- [ ] "):
		status = StatusTodo
		content = strings.TrimPrefix(trimmedLeading, "- [ ] ")
	case strings.HasPrefix(trimmedLeading, "- [/] "):
		status = StatusDoing
		content = strings.TrimPrefix(trimmedLeading, "- [/] ")
	case strings.HasPrefix(trimmedLeading, "- [x] ") || strings.HasPrefix(trimmedLeading, "- [X] "):
		status = StatusDone
		if strings.HasPrefix(trimmedLeading, "- [x] ") {
			content = strings.TrimPrefix(trimmedLeading, "- [x] ")
		} else {
			content = strings.TrimPrefix(trimmedLeading, "- [X] ")
		}
	default:
		return nil, fmt.Errorf("line %d: not a valid task format", lineNumber)
	}

	content = strings.TrimSpace(content)

	// Check if content starts with a timestamp: YYYY-MM-DD HH:MM (16 chars)
	var createdAt time.Time
	if len(content) >= 16 {
		potentialDate := content[:16]
		if t, err := time.ParseInLocation(TimestampFormat, potentialDate, time.Local); err == nil {
			createdAt = t
			content = strings.TrimSpace(content[16:])
		}
	}

	// Check for (done: YYYY-MM-DD HH:MM)
	var completedAt *time.Time
	if doneMatches := doneTagRegex.FindStringSubmatch(content); len(doneMatches) > 1 {
		if ct, err := time.ParseInLocation(TimestampFormat, doneMatches[1], time.Local); err == nil {
			completedAt = &ct
			content = doneTagRegex.ReplaceAllString(content, "")
		}
	}

	// Extract tags from the content
	tags := ExtractTags(content)

	// Clean tags from main text so Text only holds the core description
	cleanedText := content
	for _, tag := range tags {
		cleanedText = strings.ReplaceAll(cleanedText, tag, "")
	}
	cleanedText = strings.Join(strings.Fields(cleanedText), " ")

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
