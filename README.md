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

### Start a task and launch an AI agent

```bash
# Launch agent immediately after creating worktree
wt start --agent copilot "add user authentication"

# Or use environment variable
export WT_AGENT=copilot
wt start "add user authentication"

# Or configure a default agent
wt config default_agent copilot
wt start "add user authentication"
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

# With agent
wt start --jira PROJ-123 --agent copilot
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

### Launch an agent on an existing worktree

```bash
# Launch agent on a previously created task
wt agent --agent copilot wt-a1b2c3d4

# Uses WT_AGENT or default_agent if --agent not specified
wt agent wt-a1b2c3d4
```

## AI Agent Integration

`wt` can automatically launch AI agents (like GitHub Copilot CLI or Claude) inside newly created worktrees, placing the agent in the correct context for the task.

### Agent Launch Methods

**1. Explicit flag (highest priority):**
```bash
wt start --agent copilot "implement feature"
wt agent --agent copilot wt-abc123
```

**2. Environment variable:**
```bash
export WT_AGENT=copilot
wt start "implement feature"
```

**3. Configuration default:**
```bash
wt config default_agent copilot
wt start "implement feature"
```

### Passing Arguments to Agents

```bash
wt start --agent copilot --agent-args "--verbose" "implement feature"
wt agent --agent copilot --agent-args "-y" wt-abc123
```

### Agent Aliases

Configure custom agent paths in `~/.wt/config.yaml`:

```yaml
default_agent: copilot
agent_aliases:
  copilot: /usr/local/bin/github-copilot-cli
  claude: ~/.local/bin/claude-cli
```

### Environment Variables Passed to Agents

When an agent is launched, `wt` sets these environment variables:

- `WT_TASK_ID`: The task ID (e.g., `wt-abc123`)
- `WT_TICKET_KEY`: The connected ticket key if available (e.g., `PROJ-123`)
- `WT_TICKET_SUMMARY`: The ticket summary or task description

Agents can use these to provide better context-aware assistance.

### Workflow Examples

**Sequential workflow (create, then launch agent later):**
```bash
# Create worktree for planning
wt start "add authentication"

# Later, launch agent when ready
wt agent --agent copilot wt-abc123
```

**Direct workflow (create and launch in one step):**
```bash
wt start --agent copilot "add authentication"
```

**With Jira integration:**
```bash
# Ticket context automatically passed to agent
wt start --jira PROJ-123 --agent copilot
```

**Fallback behavior:**
- `wt start`: If agent not found, prints warning and continues with worktree creation
- `wt agent`: If agent not found, exits with error (agent launch is the primary purpose)

## Commands

| Command | Description |
|---------|-------------|
| `wt start <description>` | Create a worktree from a task description |
| `wt start --jira <KEY>` | Create a worktree from a Jira ticket |
| `wt start --agent <name>` | Create worktree and launch agent |
| `wt agent <task-id>` | Launch an agent on an existing worktree |
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
default_agent: copilot  # optional: default agent to launch
agent_aliases:          # optional: custom agent paths
  copilot: /usr/local/bin/github-copilot-cli
  claude: ~/.local/bin/claude-cli
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
wt config default_agent copilot
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
