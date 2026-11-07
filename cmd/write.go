package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var journalCmd = &cobra.Command{
	Use:     "journal [text]",
	Aliases: []string{"log", "j", "write"},
	Short:   "List journal entries or append a new entry",
	Long: `List recent journal entries, or append a new entry directly.

Examples:
  worklog journal                                  # List recent journal entries
  worklog journal Met with team lead #meetings     # Append a new journal entry
  worklog "Met with team lead #meetings"           # Direct append shortcut (no command name needed)
`,
	GroupID: "write",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			entries, err := storage.ScanEntries(cfg.JournalFilePath(), nil)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("(no journal entries)")
				return nil
			}

			displayCount := 20
			if len(entries) > displayCount {
				entries = entries[len(entries)-displayCount:]
			}

			ui.PrintHeader(os.Stdout, fmt.Sprintf("Recent Journal Entries (%d)", len(entries)))
			ui.RenderEntries(os.Stdout, entries, true)
			return nil
		}

		text := strings.TrimSpace(strings.Join(args, " "))
		now := time.Now()
		tags := model.ExtractTags(text)
		cleanedText := text
		for _, tag := range tags {
			cleanedText = strings.ReplaceAll(cleanedText, tag, "")
		}
		cleanedText = strings.Join(strings.Fields(cleanedText), " ")

		entry := &model.Entry{
			Timestamp: now,
			Text:      cleanedText,
			Tags:      tags,
		}

		if err := storage.AppendEntry(cfg.JournalFilePath(), entry); err != nil {
			return fmt.Errorf("failed to append entry: %w", err)
		}

		ui.PrintSuccess(os.Stdout, "Journal entry logged (%s)", now.Format("15:04"))
		return nil
	},
}

var noteCmd = &cobra.Command{
	Use:     "note",
	Short:   "Compose multi-line entry in $EDITOR",
	GroupID: "write",
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := cfg.GetEditor()

		tempFile, err := os.CreateTemp("", "worklog-note-*.md")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tempPath := tempFile.Name()
		defer os.Remove(tempPath)

		header := "# Enter your note below. Lines starting with '#' will be ignored unless they are hashtags.\n\n"
		if _, err := tempFile.WriteString(header); err != nil {
			tempFile.Close()
			return err
		}
		tempFile.Close()

		editorParts := strings.Fields(editor)
		editorCmd := exec.Command(editorParts[0], append(editorParts[1:], tempPath)...)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		data, err := os.ReadFile(tempPath)
		if err != nil {
			return fmt.Errorf("failed reading note: %w", err)
		}

		var lines []string
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || (strings.HasPrefix(line, "#") && !strings.Contains(line, " ")) {
				// skip comments or empty lines
				if strings.HasPrefix(line, "# ") {
					continue
				}
			}
			if line != "" {
				lines = append(lines, line)
			}
		}

		if len(lines) == 0 {
			fmt.Println("Empty note, nothing logged.")
			return nil
		}

		combinedText := strings.Join(lines, " ")
		now := time.Now()
		tags := model.ExtractTags(combinedText)
		cleanedText := combinedText
		for _, tag := range tags {
			cleanedText = strings.ReplaceAll(cleanedText, tag, "")
		}
		cleanedText = strings.Join(strings.Fields(cleanedText), " ")

		entry := &model.Entry{
			Timestamp: now,
			Text:      cleanedText,
			Tags:      tags,
		}

		if err := storage.AppendEntry(cfg.JournalFilePath(), entry); err != nil {
			return fmt.Errorf("failed to append note: %w", err)
		}

		ui.PrintSuccess(os.Stdout, "Note logged (%s)", now.Format("15:04"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(journalCmd)
	rootCmd.AddCommand(noteCmd)
}
