package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"
	"worklog/internal/config"
	"worklog/internal/model"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var (
	vaultPathFlag string
	cfg           *config.Config
	appVersion    = "dev"
)

// SetVersion sets the application version string.
func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "worklog [command] [args]",
	Short: "worklog - Zero-latency capture for tasks and thoughts",
	Long: `worklog is a native terminal CLI tool designed for zero-latency capture
of tasks and thoughts. It writes directly to plain Markdown files (tasks.md and journal.md)
compatible with Obsidian.

Direct journal write:
  worklog Discussed microservice migration with team lead #arch
`,
	DisableFlagParsing: false,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(vaultPathFlag)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := cfg.EnsureDir(); err != nil {
			return fmt.Errorf("failed to initialize vault directory: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		// Bare text write: worklog <text...>
		text := strings.TrimSpace(strings.Join(args, " "))
		if text == "" {
			return cmd.Help()
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

func isKnownSubcommand(name string) bool {
	for _, c := range rootCmd.Commands() {
		if c.Name() == name || c.HasAlias(name) {
			return true
		}
	}
	return false
}

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	result := make([]string, 0, len(args)+1)
	inserted := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !inserted {
			if arg == "-p" || arg == "--path" {
				result = append(result, arg)
				if i+1 < len(args) {
					i++
					result = append(result, args[i])
				}
				continue
			}

			if strings.HasPrefix(arg, "-") {
				result = append(result, arg)
				continue
			}

			// First non-flag argument
			if !isKnownSubcommand(arg) {
				result = append(result, "write")
			}
			inserted = true
		}

		result = append(result, arg)
	}

	return result
}

func Execute() {
	if len(os.Args) > 1 {
		os.Args = append([]string{os.Args[0]}, normalizeArgs(os.Args[1:])...)
	}
	if err := rootCmd.Execute(); err != nil {
		ui.PrintError(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&vaultPathFlag, "path", "p", "", "Path to the worklog/vault data directory")
}
