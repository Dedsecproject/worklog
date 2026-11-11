//go:build windows

package ui

import (
	"os"
	"strconv"
)

var terminalWidthOverride = 0

// SetTerminalWidth overrides the terminal width for testing or custom formatting.
func SetTerminalWidth(w int) {
	terminalWidthOverride = w
}

// GetTerminalWidth returns the terminal width in columns for windows.
func GetTerminalWidth() int {
	if terminalWidthOverride > 0 {
		return terminalWidthOverride
	}

	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}

	return 80
}
