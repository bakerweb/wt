# Copilot Instructions for `wt`

## Project Overview

`wt` is a task-driven git worktree manager written in Go. It creates and manages isolated git worktrees based on task descriptions or ticket system integrations (Jira, Monday.com, ClickUp). The tool bridges task management systems and developers/AI agents by automatically setting up branch-isolated environments for work.

## Build, Test, and Lint

### Build
```bash
go build -o wt ./cmd/wt  # Build the binary to ./wt
go install ./cmd/wt      # Build and install to $GOPATH/bin
```

### Test
```bash
go test ./...                                              # Run all tests
go test -v ./...                                            # Verbose output
go test -run TestBranchName ./internal/worktree            # Run specific test
go test -cover ./...                                       # Show coverage
```

### Single Package Testing
- Agent: `go test ./internal/agent`
- Config: `go test ./internal/config`
- Worktree: `go test ./internal/worktree`

### Build Release
```bash
# Uses GoReleaser (see .goreleaser.yml)
goreleaser build --single-target  # Build current platform
goreleaser release                # Full release (requires Git tags and GitHub token)
```

## Architecture

### Core Packages

**`internal/cli`** - Entry point and command handling
- Imports from urfave/cli/v2 for command structure
- Defines all CLI commands: `start`, `agent`, `list`, `finish`, `remove`, `switch`, `status`, `connect`, `sync`, `config`, `prune`
- Core pattern: Each command is a function returning `*cli.Command`

**`internal/task`** - Task lifecycle management
- `Manager` type orchestrates task creation and lifecycle
- `StartOptions` configures new tasks with description, repo path, and optional ticket info
- Generates task IDs and manages worktree/branch creation
- Interacts with `worktree` and `config` packages

**`internal/worktree`** - Git worktree and branch operations
- `SanitizeBranchName`: Converts descriptions to valid git branch names (60-char limit, lowercase, hyphens only)
- `BranchName`: Creates full branch name with prefix (e.g., "feature/add-auth")
- `BranchNameFromTicket`: Creates ticket-scoped branch names (e.g., "feature/proj-123-add-auth")
- Wraps git commands via `exec.Command` for worktree operations

**`internal/config`** - Configuration persistence and task tracking
- Stores config in `~/.wt/config.yaml` with user preferences (worktrees base, branch prefix, agent aliases)
- Thread-safe with mutex protection
- Maintains task list for tracking active worktrees
- Supports connector configurations (Jira credentials, etc.)

**`internal/connector`** - Pluggable task management integrations
- `Connector` interface: `GetTicket`, `ListAssigned`, `TransitionTicket`, `Validate`
- Three implementations: `jira`, `monday`, `clickup` (Monday and ClickUp are placeholder/planned)
- `Registry` pattern for managing multiple connectors
- Jira uses basic auth with HTTP client

**`internal/agent`** - AI agent launching and argument parsing
- `ParseAgentArgs`: Splits agent argument strings respecting quotes
- `ResolveAgent`: Finds agent executable in PATH or via aliases
- `ValidateAgent`: Verifies agent is available before launching
- Sets environment variables for agent: `WT_TASK_ID`, `WT_TICKET_KEY`, `WT_TICKET_SUMMARY`

### Data Model

**Task** (in config package)
```go
ID          string    // Generated ID (e.g., "wt-abc123d4")
Description string    // User-provided task description
Worktree    string    // Absolute path to worktree directory
Branch      string    // Git branch name with prefix
RepoPath    string    // Repository root path
Connector   string    // Connector name if ticket-driven
TicketKey   string    // External system ticket key (e.g., "PROJ-123")
Created     time.Time // Creation timestamp
```

## Key Conventions

### Branch Naming
- Branches use a prefix (default "feature") + sanitized description: `feature/add-user-auth`
- For tickets: `feature/proj-123-implement-oauth` (prefix + ticket key + sanitized summary)
- Descriptions are lowercased, special chars become hyphens, max 60 chars
- Ticket names include the ticket key for traceability

### Task ID Generation
- Format: `wt-` + 8 random hex characters (lowercase)
- Generated in `internal/task/task.go` via `generateID()`
- Used as the primary identifier in commands: `wt finish wt-a1b2c3d4`

### Worktree Directory Structure
```
~/worktrees/
├── <repo-name>/
│   ├── add-user-auth/              # One per task
│   ├── implement-oauth-flow/
│   └── ...
```
- Base path configurable via `config worktrees_base`
- Repo isolation prevents conflicts across projects

### Configuration
- Located at `~/.wt/config.yaml`
- YAML format with nested connector configs
- Example connector config structure:
  ```yaml
  connectors:
    jira:
      url: https://yourco.atlassian.net
      email: user@co.com
      api_token: xxxx
  ```

### Error Handling
- CLI errors wrapped with context: `fmt.Errorf("failed to X: %w", err)`
- Package functions return error as second return value (Go convention)
- Main defers error printing to CLI handler in `cmd/wt/main.go`

### Testing Pattern
- Test files paired with source: `file.go` + `file_test.go`
- Table-driven tests for edge cases (e.g., `TestSanitizeBranchName`)
- Test discovery automatic: any `func TestXxx(t *testing.T)` in `*_test.go`
- No test helper frameworks beyond standard `testing` package

## Dependencies

- **urfave/cli/v2** - CLI framework (v2.27.7+)
- **gopkg.in/yaml.v3** - YAML parsing for config (v3.0.1+)
- **Standard library**: `os`, `exec`, `http`, `context`, `sync`, etc.
- Go 1.25.5+ required (see go.mod)

## Integration Points for Development

- **Adding a new connector**: Implement `connector.Connector` interface and register in `cli.buildRegistry()`
- **Adding a new command**: Create function in `internal/cli/` returning `*cli.Command`, add to `cli.Run()`
- **Task state changes**: Modify `Task` struct in `internal/config/` and update persistence logic
- **Git operations**: Use `worktree` package functions; they wrap `exec.Command` for git calls
