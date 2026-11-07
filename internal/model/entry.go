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
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	dateStr := ts.Format(TimestampFormat)
	tagsStr := ""
	if len(e.Tags) > 0 {
		tagsStr = " " + strings.Join(e.Tags, " ")
	}

	lines := strings.Split(e.Text, "\n")
	res := fmt.Sprintf("- %s %s%s\n", dateStr, lines[0], tagsStr)
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

	var timestamp time.Time
	if len(content) >= 16 {
		potentialDate := content[:16]
		if t, err := time.ParseInLocation(TimestampFormat, potentialDate, time.Local); err == nil {
			timestamp = t
			content = strings.TrimSpace(content[16:])
		}
	}

	// If no timestamp was found in line, default to current time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	tags := ExtractTags(content)

	cleanedText := content
	for _, tag := range tags {
		cleanedText = strings.ReplaceAll(cleanedText, tag, "")
	}
	cleanedText = strings.Join(strings.Fields(cleanedText), " ")

	return &Entry{
		RawLine:    line,
		LineNumber: lineNumber,
		Timestamp:  timestamp,
		Text:       cleanedText,
		Tags:       tags,
	}, nil
}
