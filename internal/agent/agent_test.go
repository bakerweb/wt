package agent

import (
	"testing"
)

func TestParseAgentArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single arg",
			input:    "--help",
			expected: []string{"--help"},
		},
		{
			name:     "multiple args",
			input:    "--verbose --output json",
			expected: []string{"--verbose", "--output", "json"},
		},
		{
			name:     "double quoted arg",
			input:    `--message "hello world"`,
			expected: []string{"--message", "hello world"},
		},
		{
			name:     "single quoted arg",
			input:    `--message 'hello world'`,
			expected: []string{"--message", "hello world"},
		},
		{
			name:     "mixed quotes and spaces",
			input:    `--flag1 "value 1" --flag2 'value 2' --flag3 value3`,
			expected: []string{"--flag1", "value 1", "--flag2", "value 2", "--flag3", "value3"},
		},
		{
			name:     "extra spaces",
			input:    "  --flag   value  ",
			expected: []string{"--flag", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAgentArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d args, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("arg %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestResolveAgent(t *testing.T) {
	tests := []struct {
		name        string
		agentName   string
		aliases     map[string]string
		expectError bool
	}{
		{
			name:        "empty agent name",
			agentName:   "",
			aliases:     nil,
			expectError: true,
		},
		{
			name:        "known command from PATH",
			agentName:   "echo",
			aliases:     nil,
			expectError: false,
		},
		{
			name:        "unknown command",
			agentName:   "nonexistent-agent-12345",
			aliases:     nil,
			expectError: true,
		},
		{
			name:      "alias to known command",
			agentName: "myagent",
			aliases: map[string]string{
				"myagent": "echo",
			},
			expectError: false,
		},
		{
			name:      "alias to unknown command",
			agentName: "myagent",
			aliases: map[string]string{
				"myagent": "nonexistent-12345",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveAgent(tt.agentName, tt.aliases)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAgent(t *testing.T) {
	tests := []struct {
		name        string
		agentName   string
		aliases     map[string]string
		expectError bool
	}{
		{
			name:        "valid agent",
			agentName:   "echo",
			aliases:     nil,
			expectError: false,
		},
		{
			name:        "invalid agent",
			agentName:   "nonexistent-agent-12345",
			aliases:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgent(tt.agentName, tt.aliases)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
