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

var writeCmd = &cobra.Command{
	Use:    "write <text>",
	Short:  "Append a journal entry directly",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.TrimSpace(strings.Join(args, " "))
		if text == "" {
			return fmt.Errorf("journal entry cannot be empty")
		}

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
	Use:   "note",
	Short: "Compose multi-line entry in $EDITOR",
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
	rootCmd.AddCommand(writeCmd)
	rootCmd.AddCommand(noteCmd)
}
