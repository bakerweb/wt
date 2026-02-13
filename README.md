# wt — Git Worktree Manager

Task-driven git worktree management for developers and AI agents.

`wt` creates and manages git worktrees driven by task descriptions or ticket system integrations (Jira, Monday.com, ClickUp).

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/bakerweb/wt/main/scripts/install.sh | bash
```

The installer will download the latest binary to `~/.local/bin/wt`. If `~/.local/bin` is not in your PATH, the installer will show you how to add it.

Or build from source:

```bash
go install github.com/bakerweb/wt/cmd/wt@latest
```

## Quick Start

### Start a task from a description

```bash
cd your-repo
wt start "add user authentication"
# ✅ Task started: wt-a1b2c3d4
#    Branch:   feature/add-user-authentication
#    Worktree: ~/worktrees/your-repo/add-user-authentication
#
#    cd ~/worktrees/your-repo/add-user-authentication
```

### Start a task from a Jira ticket

```bash
# First, connect Jira
wt connect jira --url https://yourco.atlassian.net --email you@co.com --token YOUR_API_TOKEN

# Start from a ticket
wt start --jira PROJ-123
# ✅ Task started: wt-e5f6g7h8
#    Branch:   feature/proj-123-implement-oauth-flow
#    Worktree: ~/worktrees/your-repo/implement-oauth-flow
```

### List active tasks

```bash
wt list
# ID          DESCRIPTION                  BRANCH                                WORKTREE                                   TICKET
# wt-a1b2c3d4 add user authentication      feature/add-user-authentication       ~/worktrees/your-repo/add-user-auth...    -
# wt-e5f6g7h8 implement oauth flow         feature/proj-123-implement-oauth-flow ~/worktrees/your-repo/implement-oau...    PROJ-123
```

### Switch to a task worktree

```bash
cd $(wt switch wt-a1b2c3d4)
```

### Finish a task

```bash
wt finish wt-a1b2c3d4
# ✅ Task finished: add user authentication
#    Worktree removed: ~/worktrees/your-repo/add-user-authentication
#    Branch deleted: feature/add-user-authentication
```

## Commands

| Command | Description |
|---------|-------------|
| `wt start <description>` | Create a worktree from a task description |
| `wt start --jira <KEY>` | Create a worktree from a Jira ticket |
| `wt list` | Show all active tasks and worktrees |
| `wt switch <task-id>` | Print worktree path (use with `cd`) |
| `wt status` | Show current worktree task info |
| `wt finish <task-id>` | Remove worktree and delete branch |
| `wt remove <task-id>` | Remove worktree but keep branch |
| `wt connect jira` | Configure Jira integration |
| `wt sync` | Fetch assigned tickets from connected system |
| `wt config [key] [val]` | View or set configuration |
| `wt prune` | Clean up stale worktree references |
| `wt version` | Show version |

## Configuration

Config is stored in `~/.wt/config.yaml`:

```yaml
worktrees_base: ~/worktrees
default_branch: main
branch_prefix: feature
connectors:
  jira:
    url: https://yourco.atlassian.net
    email: you@co.com
    api_token: YOUR_TOKEN
    project: PROJ
```

Set values with:

```bash
wt config worktrees_base ~/my-worktrees
wt config branch_prefix feat
```

## Supported Connectors

| Connector | Status |
|-----------|--------|
| Jira | ✅ Supported |
| Monday.com | 🔜 Planned |
| ClickUp | 🔜 Planned |

## Requirements

- git >= 2.20
- A git repository to work in

## Uninstall

```bash
rm ~/.local/bin/wt
rm -rf ~/.wt  # optional: remove config and task history
```

Or use the uninstall script:

```bash
curl -fsSL https://raw.githubusercontent.com/bakerweb/wt/main/scripts/install.sh | bash -s -- --uninstall
```

## License

MIT
