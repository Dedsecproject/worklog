package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var (
	dryRunArchive bool
	forceArchive  bool
)

var archiveCmd = &cobra.Command{
	Use:     "archive [YYYY-MM]",
	Aliases: []string{"consolidate"},
	Short:   "Consolidate past months into monthly archive notes",
	Long: `Consolidate past journal entries and completed tasks into monthly archive files
(e.g., archive/YYYY-MM.md).

Unfinished tasks (todo [ ] and in-progress [/]) are preserved in tasks.md as your active backlog.
Current month items stay in tasks.md and journal.md.

Examples:
  worklog archive              # Consolidate all past months
  worklog archive --dry-run    # Preview what would be consolidated without changing files
  worklog archive 2026-10      # Consolidate a specific month
`,
	GroupID: "tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.EnsureDir(); err != nil {
			return err
		}

		now := time.Now()
		var summaries []storage.ArchiveSummary
		archiveSubdir := cfg.GetArchiveDir()

		if len(args) > 0 {
			targetMonth := args[0]
			if _, err := time.Parse("2006-01", targetMonth); err != nil {
				return fmt.Errorf("invalid month format %q; use YYYY-MM", targetMonth)
			}
			summary, err := storage.ConsolidateMonth(cfg.VaultPath, archiveSubdir, targetMonth, dryRunArchive, now)
			if err != nil {
				return err
			}
			if summary != nil {
				summaries = append(summaries, *summary)
			}
		} else {
			var err error
			summaries, err = storage.ConsolidatePastMonths(cfg.VaultPath, archiveSubdir, dryRunArchive, now)
			if err != nil {
				return err
			}
		}

		if len(summaries) == 0 {
			fmt.Println("No past entries or completed tasks to consolidate.")
			return nil
		}

		actionPrefix := "Consolidated"
		if dryRunArchive {
			actionPrefix = "[dry-run] Would consolidate"
		}

		for _, s := range summaries {
			if s.JournalCount == 0 && s.TaskCount == 0 {
				fmt.Printf("Month %s: nothing to consolidate\n", s.Month)
				continue
			}
			relPath, err := filepath.Rel(cfg.VaultPath, s.ArchivePath)
			if err != nil {
				relPath = s.ArchivePath
			}
			ui.PrintSuccess(os.Stdout, "%s %s -> %s (%d journal entries, %d completed tasks)",
				actionPrefix, s.Month, relPath, s.JournalCount, s.TaskCount)
		}

		if !dryRunArchive {
			currentMonth := now.Format("2006-01")
			cfg.LastArchiveCheck = currentMonth
			_ = cfg.Save()
		}

		return nil
	},
}

func init() {
	archiveCmd.Flags().BoolVarP(&dryRunArchive, "dry-run", "n", false, "Preview consolidation without modifying files")
	archiveCmd.Flags().BoolVarP(&forceArchive, "force", "f", false, "Force consolidation check even if already run this month")
	rootCmd.AddCommand(archiveCmd)
}
