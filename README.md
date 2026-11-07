# worklog

> **Instant terminal logger (<15ms) for tasks and thoughts with native Obsidian compatibility.**

`worklog` is a native terminal CLI tool designed for zero-latency capture of tasks and ideas to prevent context-switch decay. It writes directly to plain Markdown files (`tasks.md` and `journal.md`) that live inside an Obsidian vault or standalone directory.

---

## Features

- **⚡ Zero-Latency (<15ms):** Compiled Go binary using direct append-only disk writes (`O_APPEND`).
- **📝 Obsidian & Local-First:** Writes standard GitHub-flavored Markdown checkboxes and bullet entries. Fully compatible with Obsidian's file watcher, Dataview, and plain text editors.
- **🛡️ Atomic Mutations:** Safe temp-file rewrites and atomic renames prevent partial writes and file corruption.
- **🔍 Smart Disambiguation:** Differentiates between tasks (`#t:1`) and journal entries (`#j:1`) so modifications and deletions are unambiguous.
- **⚡ In-Progress Tracking:** Notes and tasks marked with `[/]` are automatically highlighted in a dedicated dashboard view.
- **✨ Multi-Line Hanging Indents:** Beautifully aligned checkboxes and hanging indents for multi-line tasks and notes.
- **🐚 Automatic Shell Completion:** Zero-config shell completion for Zsh, Bash, and Fish without duplicate entries.

---

## Installation

### Option 1: Homebrew (macOS & Linux)

```bash
brew install <your-github-username>/tap/worklog
```

### Option 2: Pre-compiled Binaries (GitHub Releases)

Download the pre-compiled binary for your operating system and architecture from the [Releases](https://github.com/<your-github-username>/worklog/releases) page:
- **macOS Apple Silicon (M1/M2/M3/M4):** `worklog_*_darwin_arm64.tar.gz`
- **macOS Intel:** `worklog_*_darwin_amd64.tar.gz`
- **Linux 64-bit:** `worklog_*_linux_amd64.tar.gz`
- **Linux ARM64 / Raspberry Pi:** `worklog_*_linux_arm64.tar.gz`

Extract and move the `worklog` binary to your path:
```bash
tar -xzf worklog_*_darwin_arm64.tar.gz
sudo mv worklog /usr/local/bin/
```

### Option 3: From Source (`go install`)

```bash
go install github.com/<your-github-username>/worklog@latest
```

---

## Quick Start

### 1. Direct Journal Logging

Any bare text passed to `worklog` is instantly appended to `journal.md`:

```bash
# Zero-latency capture
worklog "Discussed microservice migration with team lead #meetings #arch"

# Or explicitly via the journal command
worklog journal "Finished database migration script"

# View recent journal entries
worklog journal
```

### 2. Task Management

```bash
# Add a todo task (default)
worklog task "Deploy staging cluster #devops"

# Start an in-progress task
worklog task doing "Investigating memory leak"
# Or using the shortcut:
worklog doing "Investigating memory leak"

# Log an already-completed task
worklog done "Reviewed PR #142"

# View in-progress tasks
worklog doing

# View all tasks (grouped with in-progress at the top)
worklog tasks

# Cycle task state (todo -> doing -> done -> todo)
worklog task toggle 1
```

### 3. Read & Search Views

```bash
worklog today           # Today's journal entries + all in-progress & today's tasks
worklog yesterday       # Yesterday's summary
worklog day 2026-11-06  # Entries and tasks for a specific date
worklog week 7          # Last 7 days breakdown
worklog month           # Current month breakdown
worklog search "cache"  # Full-text regex search across tasks and journal
worklog tags            # List all hashtags and frequencies
worklog stats           # Metrics, active days, and completion rate
```

### 4. Editing & Deleting

```bash
# Delete by unambiguous prefix
worklog rm j:1          # Delete journal entry #j:1
worklog rm t:1          # Delete task #t:1
worklog rm --last       # Delete the most recently logged entry

# If line numbers overlap across files, worklog prompts with an interactive menu:
worklog rm 1

# Edit in place or in $EDITOR
worklog edit j:1 "Updated note text"
worklog edit t:1        # Opens task in $EDITOR
```

### 5. Configuration & Vault Path

```bash
# View active configuration
worklog config

# Point worklog to your Obsidian vault directory
worklog config vault_path ~/Documents/ObsidianVault/worklog

# Change default editor
worklog config editor nvim

# Open config.json directly in $EDITOR
worklog config --edit
```

---

## License

[MIT](LICENSE) © 2026 Luis Collymoore
