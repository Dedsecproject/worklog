package config

import (
	"os"
	"path/filepath"
	"strings"
)

// InstallCompletionToFile appends the completion source line to the target rcFile
// only if "worklog completion" is not already present.
func InstallCompletionToFile(rcFile string, shellName string) (bool, error) {
	if rcFile == "" {
		return false, nil
	}

	content, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	// If already present, do not duplicate
	if strings.Contains(string(content), "worklog completion") {
		return false, nil
	}

	var completionLine string
	switch shellName {
	case "zsh":
		completionLine = "source <(worklog completion zsh 2>/dev/null)"
	case "bash":
		completionLine = "source <(worklog completion bash 2>/dev/null)"
	case "fish":
		completionLine = "worklog completion fish | source"
	default:
		completionLine = "source <(worklog completion zsh 2>/dev/null)"
	}

	block := "\n# worklog shell completion\n" + completionLine + "\n"
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if _, err := f.WriteString(block); err != nil {
		return false, err
	}

	return true, nil
}

// AutoInstallCompletion detects the user's shell and ensures completion is installed in the rc file.
func AutoInstallCompletion() (installed bool, rcPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, "", err
	}

	shell := os.Getenv("SHELL")
	var rcFile string
	var shellName string

	switch {
	case strings.Contains(shell, "zsh"):
		shellName = "zsh"
		rcFile = filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		shellName = "bash"
		rcFile = filepath.Join(home, ".bashrc")
	case strings.Contains(shell, "fish"):
		shellName = "fish"
		rcFile = filepath.Join(home, ".config", "fish", "config.fish")
	default:
		shellName = "zsh"
		rcFile = filepath.Join(home, ".zshrc")
	}

	added, err := InstallCompletionToFile(rcFile, shellName)
	return added, rcFile, err
}
