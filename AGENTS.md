# `worklog` — Agent & Architecture Specification

## Agent Operational Guidelines & Invariants

When working as an AI agent on this codebase, adhere strictly to the following rules:

1. **Storage Integrity (Plain Text Only):**
   - Do **NOT** introduce SQLite, embedded key-value stores, or external databases. Plain Markdown files (`tasks.md` and `journal.md`) are the only source of truth.
   - All entries must remain human-readable and compatible with Obsidian, Dataview, and standard Markdown editors.

2. **Zero-Latency Appends (<15ms):**
   - Appending to `tasks.md` or `journal.md` MUST use `os.O_APPEND|os.O_CREATE|os.O_WRONLY`.
   - Never read, parse, or load the entire file into memory just to append an entry.

3. **Data Safety & Atomic Mutations:**
   - Any destructive or mutating command (`rm`, `edit`, `toggle`, `archive`, `migrate`) MUST create a timestamped backup first using `storage.SaveBackup`.
   - In-place file rewrites must write to a temporary file in the same directory (`.tmp`) and atomically rename it into place (`os.Rename`), preserving `0644` permissions.

4. **Shell Configuration Safety:**
   - **NEVER** edit or inject into shell configuration files (`~/.zshrc`, `~/.bashrc`, `~/.config/fish`, etc.) automatically. Suggest configuration instructions to the user instead.

5. **Testing & Quality Assurance:**
   - Run `go test -v -race -count=1 ./...` to verify all changes before completing a task.
   - Maintain unit tests for new features and keep memory overhead under 2MB during scans.

---

## 1. Executive Summary & Vision
`worklog` is a native terminal CLI tool designed for zero-latency capture of tasks and thoughts to prevent context-switch decay. It writes directly to plain Markdown files (`tasks.md` and `journal.md`) that live inside an Obsidian vault or standalone directory.

* **Primary Principle:** Instant execution (<15ms). Text appends and reads bypass heavy runtime interpreters and desktop application APIs.
* **Storage Model:** Local-first plain text. Compatible with Obsidian's file watcher, Dataview, and standard Markdown viewers.
* **Target Runtime:** Go (compiled native binary).
* **Target Distribution:** Homebrew Tap automated via GoReleaser.

---

## 2. Data Schema & Contracts

All entries are stored in two centralized files located in the target vault folder.

### 2.1. Tasks (`tasks.md`)
Standard GitHub-flavored markdown checkbox syntax:

* **Todo (`todo`):**
  ```markdown
  - [ ] 2026-11-06 14:30 Check reverse proxy logs #devops #homelab
  ```
* **In Progress (`doing`):**
  ```markdown
  - [/] 2026-11-06 14:30 Check reverse proxy logs #devops #homelab
  ```
* **Completed (`done`):**
  ```markdown
  - [x] 2026-11-06 14:30 Check reverse proxy logs (done: 2026-11-06 15:10) #devops #homelab
  ```

#### Parsing Rules:
* Prefix detection: `- [ ] ` (Todo), `- [/] ` (Doing), `- [x] ` or `- [X] ` (Done).
* Timestamp format: `YYYY-MM-DD HH:MM` (ISO 8601-like, 24-hour).
* Tags: Any string token matching `#[a-zA-Z0-9_\-]+`.

### 2.2. Journal (`journal.md`)
Standard bulleted text entries:

* **Entry:**
  ```markdown
  - 2026-11-06 14:35 Discussed microservice migration with team lead #meetings #arch
  ```

#### Parsing Rules:
* Prefix detection: `- `.
* Timestamp follows prefix directly.
* Remaining text constitutes the body and inline tags.

---

## 3. Configuration & Path Discovery

`worklog` resolves its data directory using the following precedence:

1. **CLI Flag Override:** `--path <custom-path>` (highest priority).
2. **Environment Variable:** `WORKLOG_DIR` (e.g., `export WORKLOG_DIR="$HOME/Documents/Vault/worklog"`).
3. **Local Config File:** `~/.config/worklog/config.json`.
4. **Fallback Default:** `~/.local/share/worklog` or `~/worklog/`.

### Config File Structure (`~/.config/worklog/config.json`):
```json
{
  "vault_path": "~/Documents/ObsidianVault/worklog",
  "default_editor": "nvim",
  "date_format": "2006-01-02",
  "time_format": "15:04"
}
```

---

## 4. CLI Command Specification

```
Usage: worklog <command> [args]

Write:
  <text>                              Append a journal entry directly
  note                                Compose multi-line entry in $EDITOR
  task [todo|doing|done] <text>       Add a task (default: todo)
  done <text>                         Add an already-completed task
  task toggle <query|index>           Cycle: todo -> doing -> done -> todo
  rm <selector>                       Delete an entry (interactive confirmation)
  edit <selector> [new text]          Edit entry in place or prompt

Read:
  today                               Show today's journal entries and tasks
  yesterday                           Show yesterday's entries and tasks
  day <YYYY-MM-DD|MM/DD>              Show entries and tasks for a date
  week [n]                            Show last n days (default: 7)
  month [YYYY-MM]                     Show calendar month breakdown
  tasks [todo|doing|done|all]         List tasks filtered/grouped by status
  tags                                List unique tags across both files
  search <term>                       Full-text regex search across tasks and journal
  tail [n] [journal|tasks]            Show last n lines (default: 10, journal)
  stats                               Metrics breakdown (item count by date/status)

Files / Misc:
  open [journal|tasks]                Open the target file in $EDITOR
  export [csv|json]                   Dump data to stdout
  config                              Show or update current configuration
  help                                Display help information
```

---

## 5. Project Directory Structure

```text
worklog/
├── .github/
│   └── workflows/
│       └── release.yml        # GoReleaser pipeline for GitHub Releases & Homebrew
├── cmd/
│   ├── root.go                # Cobra root command, flag definitions
│   ├── write.go               # Bare text write, note command
│   ├── task.go                # task [status], done, toggle commands
│   ├── read.go                # today, yesterday, day, week, month, tasks commands
│   ├── search.go              # search, tags, tail, stats
│   ├── modify.go              # rm, edit commands
│   └── open.go                # open in $EDITOR
├── internal/
│   ├── config/
│   │   └── config.go          # Path discovery, file initialization
│   ├── model/
│   │   ├── entry.go           # Journal entry struct and parser
│   │   └── task.go            # Task struct and parser
│   ├── storage/
│   │   ├── append.go          # O_APPEND fast disk writers
│   │   ├── scanner.go         # bufio.Scanner line parsers
│   │   └── atomic.go          # Safe in-place rewrite using temp files
│   └── ui/
│       └── table.go           # Clean terminal formatting & coloring
├── .goreleaser.yaml           # Distribution config
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
└── main.go
```

---

## 6. Core Go Data Models (`internal/model`)

### 6.1. Task Model (`internal/model/task.go`)
```go
package model

import (
	"fmt"
	"strings"
	"time"
)

type TaskStatus string

const (
	StatusTodo  TaskStatus = "todo"  // "- [ ]"
	StatusDoing TaskStatus = "doing" // "- [/]"
	StatusDone  TaskStatus = "done"  // "- [x]"
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

	dateStr := t.CreatedAt.Format("2006-01-02 15:04")
	var doneStr string
	if t.CompletedAt != nil {
		doneStr = fmt.Sprintf(" (done: %s)", t.CompletedAt.Format("2006-01-02 15:04"))
	}

	tagsStr := ""
	if len(t.Tags) > 0 {
		tagsStr = " " + strings.Join(t.Tags, " ")
	}

	return fmt.Sprintf("- %s %s %s%s%s\n", box, dateStr, t.Text, doneStr, tagsStr)
}
```

### 6.2. Journal Entry Model (`internal/model/entry.go`)
```go
package model

import (
	"fmt"
	"strings"
	"time"
)

type Entry struct {
	RawLine    string
	LineNumber int
	Timestamp  time.Time
	Text       string
	Tags       []string
}

func (e Entry) Markdown() string {
	dateStr := e.Timestamp.Format("2006-01-02 15:04")
	tagsStr := ""
	if len(e.Tags) > 0 {
		tagsStr = " " + strings.Join(e.Tags, " ")
	}
	return fmt.Sprintf("- %s %s%s\n", dateStr, e.Text, tagsStr)
}
```

---

## 7. Storage Engine Principles (`internal/storage`)

1. **Instant Appends:** 
   * Writes use `os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)`.
   * Never read or parse the entire file into memory just to append an entry.
2. **Safe In-Place Mutation (`rm`, `edit`, `toggle`):**
   * Never modify files in place directly to avoid truncation loss.
   * Write to a temporary file (`tasks.md.tmp`) in the same directory.
   * Atomically replace using `os.Rename(tempFile, targetFile)`. This guarantees Obsidian never encounters a half-written file.
3. **Line Scanning:**
   * Read operations stream lines using `bufio.NewScanner(file)` to keep memory usage under 2MB even on multi-year worklogs.

---

## 8. Distribution Specification (GoReleaser + Homebrew)

### `.goreleaser.yaml`
```yaml
version: 2
project_name: worklog

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

brews:
  - repository:
      owner: <your-github-username>
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    folder: Formula
    homepage: "https://github.com/<your-github-username>/worklog"
    description: "Instant terminal logger for tasks and thoughts with Obsidian compatibility"
    license: "MIT"
```

### Installation Target:
```bash
brew install <your-github-username>/tap/worklog
```