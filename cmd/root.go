package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"
	"worklog/internal/config"
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
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(vaultPathFlag)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Skip initializing vault directories for purely informational commands
		if cmd != nil && (cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version") {
			return nil
		}

		if err := cfg.EnsureDir(); err != nil {
			return fmt.Errorf("failed to initialize vault directory: %w", err)
		}

		// Run automatic monthly consolidation check if month has rolled over
		maybeAutoArchive(cmd)

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func isKnownSubcommand(name string) bool {
	if name == "completion" || name == "help" || name == "version" {
		return true
	}
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

func maybeAutoArchive(cmd *cobra.Command) {
	if cmd != nil && (cmd.Name() == "archive" || cmd.Name() == "consolidate" || cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version") {
		return
	}
	if cfg == nil || !cfg.IsAutoArchiveEnabled() {
		return
	}

	now := time.Now()
	currentMonth := now.Format("2006-01")
	if cfg.LastArchiveCheck == currentMonth {
		return
	}

	// First time encountering a new month: consolidate past months
	summaries, err := storage.ConsolidatePastMonths(cfg.VaultPath, cfg.GetArchiveDir(), false, now)
	if err == nil {
		cfg.LastArchiveCheck = currentMonth
		_ = cfg.Save()
		if len(summaries) > 0 {
			for _, s := range summaries {
				ui.PrintDim(os.Stderr, "ℹ Consolidated %s -> %s/%s.md (%d entries, %d completed tasks)\n",
					s.Month, cfg.GetArchiveDir(), s.Month, s.JournalCount, s.TaskCount)
			}
		}
	}
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
	rootCmd.AddGroup(&cobra.Group{
		ID:    "write",
		Title: "Journal & Writing:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "task",
		Title: "Task Management:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "read",
		Title: "Read & Search:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "tools",
		Title: "Modify, Files & Tools:",
	})

	rootCmd.PersistentFlags().StringVarP(&vaultPathFlag, "path", "p", "", "Path to the worklog/vault data directory")
}
