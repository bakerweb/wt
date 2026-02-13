package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ResolveAgent resolves an agent name to an executable path.
// It first checks the aliases map, then falls back to PATH lookup.
func ResolveAgent(name string, aliases map[string]string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("no agent specified")
	}

	// Check aliases first
	if path, ok := aliases[name]; ok {
		if _, err := exec.LookPath(path); err != nil {
			return "", fmt.Errorf("agent alias %q points to %q which is not found: %w", name, path, err)
		}
		return path, nil
	}

	// Fall back to PATH lookup
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("agent %q not found in PATH: %w", name, err)
	}
	return path, nil
}

// ValidateAgent checks if an agent is available.
func ValidateAgent(name string, aliases map[string]string) error {
	_, err := ResolveAgent(name, aliases)
	return err
}

// LaunchOptions configures agent launch.
type LaunchOptions struct {
	Agent         string
	Args          []string
	WorkDir       string
	TaskID        string
	TicketKey     string
	TicketSummary string
	Aliases       map[string]string
}

// LaunchAgent launches an agent using exec syscall to replace the current process.
func LaunchAgent(opts LaunchOptions) error {
	agentPath, err := ResolveAgent(opts.Agent, opts.Aliases)
	if err != nil {
		return err
	}

	// Change to the working directory
	if opts.WorkDir != "" {
		if err := os.Chdir(opts.WorkDir); err != nil {
			return fmt.Errorf("failed to change directory to %s: %w", opts.WorkDir, err)
		}
	}

	// Set environment variables
	if opts.TaskID != "" {
		os.Setenv("WT_TASK_ID", opts.TaskID)
	}
	if opts.TicketKey != "" {
		os.Setenv("WT_TICKET_KEY", opts.TicketKey)
	}
	if opts.TicketSummary != "" {
		os.Setenv("WT_TICKET_SUMMARY", opts.TicketSummary)
	}

	// Build command arguments
	args := []string{agentPath}
	args = append(args, opts.Args...)

	// Use exec syscall to replace the current process
	// This makes the agent the direct child of the shell
	if err := syscall.Exec(agentPath, args, os.Environ()); err != nil {
		return fmt.Errorf("failed to exec %s: %w", agentPath, err)
	}

	// This line will never be reached if exec succeeds
	return nil
}

// ParseAgentArgs parses a space-separated string of agent arguments.
// Handles quoted strings properly.
func ParseAgentArgs(argsStr string) []string {
	if argsStr == "" {
		return nil
	}

	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range argsStr {
		switch {
		case (r == '"' || r == '\'') && !inQuote:
			inQuote = true
			quoteChar = r
		case r == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
