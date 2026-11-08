package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var (
	migrateDryRun bool
	migrateJSON   bool
)

var migrateCmd = &cobra.Command{
	Use:     "migrate",
	Short:   "Standardize legacy timestamps and merge legacy worklog.md",
	GroupID: "tools",
	Long: `Migrate detects and upgrades legacy data formats in your worklog vault:
- Standardizes trailing timestamps (e.g. '[2026-10-24T14]') into leading Obsidian timestamps in tasks.md and journal.md.
- Migrates any legacy 'worklog.md' tasks into tasks.md and notes into journal.md without duplicating entries.
- Renames 'worklog.md' to 'worklog.md.migrated'.
- Automatically takes a backup before modifying any files, enabling 'worklog undo'.

Examples:
  worklog migrate --dry-run    # Preview changes without modifying files
  worklog migrate              # Apply migrations with automatic backup
  worklog migrate --json       # Machine-readable output
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !migrateDryRun {
			legacyPath := filepath.Join(cfg.VaultPath, "worklog.md")
			_ = storage.SaveBackup(cfg.VaultPath, "migrate", cfg.JournalFilePath(), cfg.TasksFilePath(), legacyPath)
		}

		stats, err := storage.Migrate(cfg.VaultPath, migrateDryRun)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		if migrateJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(stats)
		}

		if migrateDryRun {
			ui.PrintInfo(os.Stdout, "Migration preview (dry-run, no changes made):")
		} else {
			ui.PrintSuccess(os.Stdout, "Migration completed successfully!")
		}

		fmt.Printf("  • Tasks standardized:   %d\n", stats.TasksStandardized)
		fmt.Printf("  • Journal standardized: %d\n", stats.JournalStandardized)
		if stats.LegacyWorklogFound {
			fmt.Printf("  • Legacy worklog.md:    found (%d tasks, %d journal entries moved)\n",
				stats.WorklogTasksMoved, stats.WorklogEntriesMoved)
		} else {
			fmt.Printf("  • Legacy worklog.md:    none found\n")
		}

		if len(stats.Changes) > 0 {
			fmt.Printf("\nChanges (%d total):\n", len(stats.Changes))
			limit := 15
			if len(stats.Changes) < limit {
				limit = len(stats.Changes)
			}
			for i := 0; i < limit; i++ {
				fmt.Printf("  %s\n", stats.Changes[i])
			}
			if len(stats.Changes) > limit {
				fmt.Printf("  ... and %d more changes\n", len(stats.Changes)-limit)
			}
		} else {
			fmt.Println("\nVault is already up-to-date. No changes needed.")
		}

		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVarP(&migrateDryRun, "dry-run", "n", false, "Preview changes without modifying vault files")
	migrateCmd.Flags().BoolVar(&migrateJSON, "json", false, "Output migration results as JSON")
	rootCmd.AddCommand(migrateCmd)
}
