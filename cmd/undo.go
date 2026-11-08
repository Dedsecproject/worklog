package cmd

import (
	"os"
	"worklog/internal/storage"
	"worklog/internal/ui"

	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:     "undo",
	Short:   "Undo the last modifying operation (rm, edit, archive)",
	GroupID: "tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		meta, err := storage.Undo(cfg.VaultPath)
		if err != nil {
			return err
		}

		ui.PrintSuccess(os.Stdout, "Undid: %s (restored %d file(s))", meta.Action, len(meta.Files))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
}
