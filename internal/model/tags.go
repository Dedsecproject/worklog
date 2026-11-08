package model

import (
	"regexp"
	"strings"
	"time"
)

// tagRegex matches Obsidian-style tags:
//   - must be preceded by start-of-text or a character that cannot be part of a word/URL/entity
//   - may contain letters, digits, '_', '-' and '/' (nested tags like #project/alpha)
//   - must contain at least one letter or '_' (so "#142" in "PR #142" is NOT a tag)
var tagRegex = regexp.MustCompile(`(?:^|[^\p{L}\p{N}_/&:#])(#[\p{L}\p{N}_\-/]*[\p{L}_][\p{L}\p{N}_\-/]*)`)

// SplitTags extracts unique tags from text and returns the text with those tags removed.
// Whitespace is normalized per line; line breaks are preserved and lines left empty are dropped.
func SplitTags(text string) (string, []string) {
	matches := tagRegex.FindAllStringSubmatchIndex(text, -1)

	var tags []string
	seen := make(map[string]bool)
	var sb strings.Builder
	last := 0
	for _, m := range matches {
		start, end := m[2], m[3] // capture group 1: the tag itself
		tag := strings.TrimRight(text[start:end], "/-")
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
		sb.WriteString(text[last:start])
		last = end
	}
	sb.WriteString(text[last:])

	var lines []string
	for _, line := range strings.Split(sb.String(), "\n") {
		if normalized := strings.Join(strings.Fields(line), " "); normalized != "" {
			lines = append(lines, normalized)
		}
	}
	return strings.Join(lines, "\n"), tags
}

// ExtractTags returns the unique tags found in s.
func ExtractTags(s string) []string {
	_, tags := SplitTags(s)
	return tags
}

// NewEntry builds a journal entry from raw user input, moving tags out of the text.
func NewEntry(raw string, now time.Time) *Entry {
	text, tags := SplitTags(strings.TrimSpace(raw))
	return &Entry{Timestamp: now, Text: text, Tags: tags}
}

// NewTask builds a task from raw user input, moving tags out of the text.
// Done tasks get CompletedAt = now.
func NewTask(raw string, status TaskStatus, now time.Time) *Task {
	text, tags := SplitTags(strings.TrimSpace(raw))
	t := &Task{Status: status, CreatedAt: now, Text: text, Tags: tags}
	if status == StatusDone {
		completed := now
		t.CompletedAt = &completed
	}
	return t
}

// legacyStampRegex matches legacy timestamp stamps like "[2026-10-24T14]" or "[2026-10-24T14:05]".
var legacyStampRegex = regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2})T(\d{2})(?::(\d{2}))?\]`)

// MatchesTag checks if any tag in tags matches filterTag.
// Matches are case-insensitive and match Obsidian nested tags (e.g. #project matches #project/alpha).
func MatchesTag(tags []string, filterTag string) bool {
	if filterTag == "" {
		return true
	}
	cleanFilter := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(filterTag), "#"))
	if cleanFilter == "" {
		return true
	}
	for _, t := range tags {
		cleanT := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(t), "#"))
		if cleanT == cleanFilter || strings.HasPrefix(cleanT, cleanFilter+"/") {
			return true
		}
	}
	return false
}

// parseLeadingTimestamp extracts a timestamp from the start of content ("YYYY-MM-DD HH:MM ..."),
// falling back to a legacy "[YYYY-MM-DDTHH]" suffix or inline stamp. If neither is present it returns
// the zero time — callers must treat a zero time as "undated", never as "now".
func parseLeadingTimestamp(content string) (time.Time, string) {
	if len(content) >= 16 {
		if t, err := time.ParseInLocation(TimestampFormat, content[:16], time.Local); err == nil {
			return t, strings.TrimSpace(content[16:])
		}
	}
	if loc := legacyStampRegex.FindStringSubmatchIndex(content); loc != nil {
		dateStr := content[loc[2]:loc[3]]
		hourStr := content[loc[4]:loc[5]]
		minuteStr := "00"
		if loc[6] != -1 && loc[7] != -1 {
			minuteStr = content[loc[6]:loc[7]]
		}
		if t, err := time.ParseInLocation(TimestampFormat, dateStr+" "+hourStr+":"+minuteStr, time.Local); err == nil {
			cleaned := strings.TrimSpace(content[:loc[0]] + " " + content[loc[1]:])
			return t, cleaned
		}
	}
	return time.Time{}, content
}
