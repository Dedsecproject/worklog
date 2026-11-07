package model

import (
	"testing"
	"time"
)

func TestParseTask(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantStatus   TaskStatus
		wantText     string
		wantTags     []string
		wantDone     bool
		expectErr    bool
	}{
		{
			name:       "Todo task with tags",
			input:      "- [ ] 2026-11-06 14:30 Check reverse proxy logs #devops #homelab",
			wantStatus: StatusTodo,
			wantText:   "Check reverse proxy logs",
			wantTags:   []string{"#devops", "#homelab"},
			wantDone:   false,
		},
		{
			name:       "Doing task",
			input:      "- [/] 2026-11-06 14:30 Check reverse proxy logs",
			wantStatus: StatusDoing,
			wantText:   "Check reverse proxy logs",
			wantTags:   nil,
			wantDone:   false,
		},
		{
			name:       "Completed task with done timestamp",
			input:      "- [x] 2026-11-06 14:30 Check reverse proxy logs (done: 2026-11-06 15:10) #devops #homelab",
			wantStatus: StatusDone,
			wantText:   "Check reverse proxy logs",
			wantTags:   []string{"#devops", "#homelab"},
			wantDone:   true,
		},
		{
			name:       "Completed task uppercase X",
			input:      "- [X] 2026-11-06 14:30 Task finished",
			wantStatus: StatusDone,
			wantText:   "Task finished",
			wantTags:   nil,
			wantDone:   false,
		},
		{
			name:      "Invalid line",
			input:     "just a random note",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := ParseTask(tt.input, 1)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if task.Status != tt.wantStatus {
				t.Errorf("got status %v, want %v", task.Status, tt.wantStatus)
			}
			if task.Text != tt.wantText {
				t.Errorf("got text %q, want %q", task.Text, tt.wantText)
			}
			if (task.CompletedAt != nil) != tt.wantDone {
				t.Errorf("got completedAt %v, wantDone %v", task.CompletedAt, tt.wantDone)
			}
			if len(task.Tags) != len(tt.wantTags) {
				t.Errorf("got %d tags, want %d", len(task.Tags), len(tt.wantTags))
			}
		})
	}
}

func TestTaskToggle(t *testing.T) {
	now := time.Date(2026, 11, 6, 15, 0, 0, 0, time.Local)
	task := &Task{
		Status:    StatusTodo,
		CreatedAt: now,
		Text:      "Test task",
	}

	// todo -> doing
	task.Toggle(now)
	if task.Status != StatusDoing {
		t.Errorf("expected StatusDoing, got %s", task.Status)
	}
	if task.CompletedAt != nil {
		t.Errorf("expected nil CompletedAt")
	}

	// doing -> done
	task.Toggle(now)
	if task.Status != StatusDone {
		t.Errorf("expected StatusDone, got %s", task.Status)
	}
	if task.CompletedAt == nil {
		t.Errorf("expected non-nil CompletedAt")
	}

	// done -> todo
	task.Toggle(now)
	if task.Status != StatusTodo {
		t.Errorf("expected StatusTodo, got %s", task.Status)
	}
	if task.CompletedAt != nil {
		t.Errorf("expected nil CompletedAt after returning to todo")
	}
}

func TestTaskMarkdown(t *testing.T) {
	createdAt := time.Date(2026, 11, 6, 14, 30, 0, 0, time.Local)
	completedAt := time.Date(2026, 11, 6, 15, 10, 0, 0, time.Local)

	task := Task{
		Status:      StatusDone,
		CreatedAt:   createdAt,
		CompletedAt: &completedAt,
		Text:        "Check reverse proxy logs",
		Tags:        []string{"#devops", "#homelab"},
	}

	expected := "- [x] 2026-11-06 14:30 Check reverse proxy logs (done: 2026-11-06 15:10) #devops #homelab\n"
	if got := task.Markdown(); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
