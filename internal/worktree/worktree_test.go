package worktree

import "testing"

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"add user authentication", "add-user-authentication"},
		{"Fix Bug #123: Login Fails!", "fix-bug-123-login-fails"},
		{"  spaces everywhere  ", "spaces-everywhere"},
		{"UPPERCASE-Mixed", "uppercase-mixed"},
		{"special@chars&here", "special-chars-here"},
		{"a-very-long-description-that-exceeds-the-sixty-character-limit-by-quite-a-bit", "a-very-long-description-that-exceeds-the-sixty-character-lim"},
		{"---leading-trailing---", "leading-trailing"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBranchName(t *testing.T) {
	tests := []struct {
		prefix      string
		description string
		expected    string
	}{
		{"feature", "add login", "feature/add-login"},
		{"", "add login", "add-login"},
		{"fix", "memory leak in parser", "fix/memory-leak-in-parser"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := BranchName(tt.prefix, tt.description)
			if got != tt.expected {
				t.Errorf("BranchName(%q, %q) = %q, want %q", tt.prefix, tt.description, got, tt.expected)
			}
		})
	}
}

func TestBranchNameFromTicket(t *testing.T) {
	tests := []struct {
		prefix   string
		ticket   string
		summary  string
		expected string
	}{
		{"feature", "PROJ-123", "implement oauth flow", "feature/proj-123-implement-oauth-flow"},
		{"", "BUG-456", "fix crash on startup", "bug-456-fix-crash-on-startup"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := BranchNameFromTicket(tt.prefix, tt.ticket, tt.summary)
			if got != tt.expected {
				t.Errorf("BranchNameFromTicket(%q, %q, %q) = %q, want %q", tt.prefix, tt.ticket, tt.summary, got, tt.expected)
			}
		})
	}
}
