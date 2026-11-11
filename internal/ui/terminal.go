//go:build !windows

package ui

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

var terminalWidthOverride = 0

// SetTerminalWidth overrides the terminal width for testing or custom formatting.
// Set to 0 to restore automatic detection.
func SetTerminalWidth(w int) {
	terminalWidthOverride = w
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// GetTerminalWidth returns the terminal width in columns.
// Precedence:
// 1. Explicit override via SetTerminalWidth
// 2. COLUMNS environment variable
// 3. ioctl TIOCGWINSZ on stdout, stderr, or stdin
// 4. Default 80 columns fallback
func GetTerminalWidth() int {
	if terminalWidthOverride > 0 {
		return terminalWidthOverride
	}

	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}

	var ws winsize
	for _, fd := range []uintptr{os.Stdout.Fd(), os.Stderr.Fd(), os.Stdin.Fd()} {
		_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
		if err == 0 && ws.Col > 0 {
			return int(ws.Col)
		}
	}

	return 80
}
