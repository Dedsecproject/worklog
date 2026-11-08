package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var entryPrefixRegex = regexp.MustCompile(`^-\s*(\[(\s*|\/|x|X|-)\])?\s*`)

type Entry struct {
	FilePath   string
	RawLine    string
	LineNumber int
	Timestamp  time.Time
	Text       string
	Tags       []string
}

// Markdown formats the entry as a bulleted text entry.
func (e Entry) Markdown() string {
	dateStr := ""
	if !e.Timestamp.IsZero() {
		dateStr = e.Timestamp.Format(TimestampFormat) + " "
	}
	tagsStr := ""
	if len(e.Tags) > 0 {
		tagsStr = " " + strings.Join(e.Tags, " ")
	}

	lines := strings.Split(e.Text, "\n")
	res := fmt.Sprintf("- %s%s%s\n", dateStr, lines[0], tagsStr)
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			res += fmt.Sprintf("  %s\n", trimmed)
		}
	}
	return res
}

// ParseEntry parses a markdown bullet line into an Entry.
// Format: - YYYY-MM-DD HH:MM Text #tags...
// Also supports notes prefixed with checkboxes like - [/] or - [ ] so they never disappear.
func ParseEntry(line string, lineNumber int) (*Entry, error) {
	trimmed := strings.TrimRight(line, "\r\n")
	trimmedLeading := strings.TrimSpace(trimmed)

	if !strings.HasPrefix(trimmedLeading, "-") {
		return nil, fmt.Errorf("line %d: not a valid journal entry format", lineNumber)
	}

	prefixMatch := entryPrefixRegex.FindStringSubmatch(trimmedLeading)
	if len(prefixMatch) == 0 {
		return nil, fmt.Errorf("line %d: not a valid journal entry format", lineNumber)
	}

	content := strings.TrimSpace(trimmedLeading[len(prefixMatch[0]):])
	content = StripMarkdownComments(content)

	timestamp, content := parseLeadingTimestamp(content)

	cleanedText, tags := SplitTags(content)

	return &Entry{
		RawLine:    line,
		LineNumber: lineNumber,
		Timestamp:  timestamp,
		Text:       cleanedText,
		Tags:       tags,
	}, nil
}
