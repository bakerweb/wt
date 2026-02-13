package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/bakerweb/wt/internal/agent"
	"github.com/bakerweb/wt/internal/config"
	"github.com/bakerweb/wt/internal/connector"
	"github.com/bakerweb/wt/internal/connector/clickup"
	"github.com/bakerweb/wt/internal/connector/jira"
	"github.com/bakerweb/wt/internal/connector/monday"
	"github.com/bakerweb/wt/internal/task"
	"github.com/bakerweb/wt/internal/worktree"
	"github.com/urfave/cli/v2"
)

var Version = "dev"

// Custom help template with command categories
const appHelpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:

  Task Lifecycle:{{range .VisibleCommands}}{{if eq .Category "lifecycle"}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

  Navigation & Status:{{range .VisibleCommands}}{{if eq .Category "navigation"}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

  Agent Integration:{{range .VisibleCommands}}{{if eq .Category "agent"}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

  Configuration & Integration:{{range .VisibleCommands}}{{if eq .Category "config"}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

  Maintenance:{{range .VisibleCommands}}{{if eq .Category "maintenance"}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

  Other:{{range .VisibleCommands}}{{if eq .Category ""}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}

GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`

func Run(args []string) error {
	app := &cli.App{
		Name:                 "wt",
		Usage:                "Git worktree manager driven by tasks",
		Version:              Version,
		CustomAppHelpTemplate: appHelpTemplate,
		Commands: []*cli.Command{
			startCmd(),
			agentCmd(),
			listCmd(),
			finishCmd(),
			removeCmd(),
			switchCmd(),
			statusCmd(),
			connectCmd(),
			syncCmd(),
			configCmd(),
			pruneCmd(),
		},
	}
	return app.Run(args)
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

func getRepoPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine current directory: %w", err)
	}
	// Walk up to find .git
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a git repository (searched from %s)", cwd)
		}
		dir = parent
	}
}

func buildRegistry(cfg *config.Config) *connector.Registry {
	reg := connector.NewRegistry()
	if cc, ok := cfg.Connectors["jira"]; ok {
		reg.Register(jira.New(cc.URL, cc.Email, cc.APIToken))
	}
	reg.Register(monday.New())
	reg.Register(clickup.New())
	return reg
}

// --- start ---
func startCmd() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Category:  "lifecycle",
		Usage:     "Create a new worktree for a task",
		ArgsUsage: "<task-description>",
		Description: `Create an isolated git worktree for a new task in a separate directory.

   Supports two modes:
     1. From description: wt start "add user authentication"
     2. From Jira ticket: wt start --jira PROJ-123

   Can optionally launch an AI agent immediately with --agent flag.
   Use WT_AGENT environment variable or default_agent config for automatic agent launch.

   Examples:
     wt start "implement oauth flow"
     wt start --jira PROJ-123
     wt start --agent copilot "add user auth"
     wt start --jira PROJ-123 --agent copilot --agent-args "--verbose"`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "jira",
				Usage: "Create worktree from a Jira issue key (e.g. PROJ-123)",
			},
			&cli.StringFlag{
				Name:  "agent",
				Usage: "Launch an agent after creating the worktree (e.g. copilot, claude)",
			},
			&cli.StringFlag{
				Name:  "agent-args",
				Usage: "Arguments to pass to the agent",
			},
		},
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			repoPath, err := getRepoPath()
			if err != nil {
				return err
			}

			mgr := task.NewManager(cfg)
			opts := task.StartOptions{RepoPath: repoPath}

			if jiraKey := c.String("jira"); jiraKey != "" {
				cc, ok := cfg.Connectors["jira"]
				if !ok {
					return fmt.Errorf("jira is not configured; run 'wt connect jira' first")
				}
				client := jira.New(cc.URL, cc.Email, cc.APIToken)
				ticket, err := client.GetTicket(context.Background(), jiraKey)
				if err != nil {
					return fmt.Errorf("failed to fetch jira issue: %w", err)
				}
				opts.Description = ticket.Summary
				opts.Connector = "jira"
				opts.TicketKey = ticket.Key
				opts.TicketTitle = ticket.Summary
				fmt.Printf("📋 Jira: %s - %s\n", ticket.Key, ticket.Summary)
			} else {
				if c.NArg() < 1 {
					return fmt.Errorf("please provide a task description or use --jira <ISSUE-KEY>")
				}
				opts.Description = joinArgs(c)
			}

			t, err := mgr.Start(opts)
			if err != nil {
				return err
			}

			fmt.Printf("✅ Task started: %s\n", t.ID)
			fmt.Printf("   Branch:   %s\n", t.Branch)
			fmt.Printf("   Worktree: %s\n", t.Worktree)

			// Determine agent to launch
			agentName := c.String("agent")
			if agentName == "" {
				agentName = os.Getenv("WT_AGENT")
			}
			if agentName == "" {
				agentName = cfg.DefaultAgent
			}

			// If no agent specified, just print the cd command
			if agentName == "" {
				fmt.Printf("\n   cd %s\n", t.Worktree)
				return nil
			}

			// Validate and launch agent
			if err := agent.ValidateAgent(agentName, cfg.AgentAliases); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Agent %q not found: %v\n", agentName, err)
				fmt.Printf("\n   cd %s\n", t.Worktree)
				return nil
			}

			// Parse agent args
			agentArgs := agent.ParseAgentArgs(c.String("agent-args"))

			fmt.Printf("\n🚀 Launching agent: %s\n", agentName)
			return agent.LaunchAgent(agent.LaunchOptions{
				Agent:         agentName,
				Args:          agentArgs,
				WorkDir:       t.Worktree,
				TaskID:        t.ID,
				TicketKey:     t.TicketKey,
				TicketSummary: opts.TicketTitle,
				Aliases:       cfg.AgentAliases,
			})
		},
	}
}

// --- agent ---
func agentCmd() *cli.Command {
	return &cli.Command{
		Name:      "agent",
		Category:  "agent",
		Usage:     "Launch an agent on an existing worktree",
		ArgsUsage: "<task-id>",
		Description: `Launch an AI agent (like GitHub Copilot CLI or Claude) in an existing task's worktree.

   The agent will be launched with context about the task, including task ID, 
   ticket key (if available), and task description via environment variables.

   Agent selection priority:
     1. --agent flag (highest)
     2. WT_AGENT environment variable
     3. default_agent config setting

   Examples:
     wt agent wt-abc123                    # Uses WT_AGENT or default_agent
     wt agent --agent copilot wt-abc123    # Explicit agent selection
     wt agent --agent copilot --agent-args "-y" wt-abc123`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "agent",
				Usage: "Agent to launch (e.g. copilot, claude). If omitted, uses WT_AGENT env var or default_agent config",
			},
			&cli.StringFlag{
				Name:  "agent-args",
				Usage: "Arguments to pass to the agent",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("please provide a task ID (see 'wt list')")
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			taskID := c.Args().First()
			t, err := cfg.FindTask(taskID)
			if err != nil {
				return err
			}

			// Verify worktree still exists
			if _, err := os.Stat(t.Worktree); err != nil {
				return fmt.Errorf("worktree %s no longer exists: %w", t.Worktree, err)
			}

			// Determine agent to launch
			agentName := c.String("agent")
			if agentName == "" {
				agentName = os.Getenv("WT_AGENT")
			}
			if agentName == "" {
				agentName = cfg.DefaultAgent
			}

			if agentName == "" {
				return fmt.Errorf("no agent specified; use --agent flag, set WT_AGENT env var, or configure default_agent")
			}

			// Validate agent (fail if not found, unlike wt start)
			if err := agent.ValidateAgent(agentName, cfg.AgentAliases); err != nil {
				return fmt.Errorf("agent %q not found: %w", agentName, err)
			}

			// Parse agent args
			agentArgs := agent.ParseAgentArgs(c.String("agent-args"))

			fmt.Printf("🚀 Launching agent %q on task %s\n", agentName, t.ID)
			fmt.Printf("   Worktree: %s\n", t.Worktree)

			ticketSummary := t.TicketKey
			if t.Description != "" {
				ticketSummary = t.Description
			}

			return agent.LaunchAgent(agent.LaunchOptions{
				Agent:         agentName,
				Args:          agentArgs,
				WorkDir:       t.Worktree,
				TaskID:        t.ID,
				TicketKey:     t.TicketKey,
				TicketSummary: ticketSummary,
				Aliases:       cfg.AgentAliases,
			})
		},
	}
}

// --- list ---
func listCmd() *cli.Command {
	return &cli.Command{
		Name:     "list",
		Category: "navigation",
		Usage:    "Show all active tasks and worktrees",
		Aliases:  []string{"ls"},
		Description: `Display a table of all active tasks managed by wt.

   Shows task ID, description, branch name, worktree path, and associated ticket.
   Use task IDs from this output with other commands (finish, remove, switch, agent).

   Example:
     wt list`,
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if len(cfg.Tasks) == 0 {
				fmt.Println("No active tasks.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tDESCRIPTION\tBRANCH\tWORKTREE\tTICKET")
			for _, t := range cfg.Tasks {
				ticket := t.TicketKey
				if ticket == "" {
					ticket = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", t.ID, truncate(t.Description, 40), t.Branch, t.Worktree, ticket)
			}
			return w.Flush()
		},
	}
}

// --- finish ---
func finishCmd() *cli.Command {
	return &cli.Command{
		Name:      "finish",
		Category:  "lifecycle",
		Usage:     "Complete a task, remove worktree and branch",
		ArgsUsage: "<task-id>",
		Description: `Complete a task and clean up all resources.

   This command will:
     1. Remove the worktree directory
     2. Delete the git branch
     3. Remove the task from wt's tracking

   Use this when work is complete and merged. For keeping the branch, use 'wt remove' instead.

   Example:
     wt finish wt-abc123`,
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("please provide a task ID (see 'wt list')")
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			mgr := task.NewManager(cfg)
			t, err := mgr.Finish(c.Args().First())
			if err != nil {
				return err
			}
			fmt.Printf("✅ Task finished: %s\n", t.Description)
			fmt.Printf("   Worktree removed: %s\n", t.Worktree)
			fmt.Printf("   Branch deleted: %s\n", t.Branch)
			return nil
		},
	}
}

// --- remove ---
func removeCmd() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Category:  "lifecycle",
		Usage:     "Remove a worktree but keep the branch",
		Aliases:   []string{"rm"},
		ArgsUsage: "<task-id>",
		Description: `Remove a worktree directory but preserve the git branch.

   Use this when you want to free up disk space but keep the branch for later work.
   The branch can be checked out again or a new worktree created from it.

   Example:
     wt remove wt-abc123`,
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("please provide a task ID (see 'wt list')")
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			mgr := task.NewManager(cfg)
			t, err := mgr.Remove(c.Args().First())
			if err != nil {
				return err
			}
			fmt.Printf("✅ Worktree removed: %s\n", t.Worktree)
			fmt.Printf("   Branch kept: %s\n", t.Branch)
			return nil
		},
	}
}

// --- switch ---
func switchCmd() *cli.Command {
	return &cli.Command{
		Name:      "switch",
		Category:  "navigation",
		Usage:     "Print the path to a task's worktree (use with cd)",
		ArgsUsage: "<task-id>",
		Description: `Print the absolute path to a task's worktree directory.

   Designed to be used with command substitution to change directories:
     cd $(wt switch wt-abc123)

   Example:
     wt switch wt-abc123              # Prints path only
     cd $(wt switch wt-abc123)        # Change to task worktree`,
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("please provide a task ID (see 'wt list')")
			}
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			t, err := cfg.FindTask(c.Args().First())
			if err != nil {
				return err
			}
			// Print just the path so it can be used with: cd $(wt switch <id>)
			fmt.Print(t.Worktree)
			return nil
		},
	}
}

// --- status ---
func statusCmd() *cli.Command {
	return &cli.Command{
		Name:     "status",
		Category: "navigation",
		Usage:    "Show status of the current worktree task",
		Description: `Display detailed information about the task in the current directory.

   Shows task ID, description, branch, worktree path, creation time, and ticket info.
   Only works when run from inside a wt-managed worktree directory.

   Example:
     cd ~/worktrees/myrepo/feature-branch
     wt status`,
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			t, err := cfg.FindTaskByWorktree(cwd)
			if err != nil {
				fmt.Println("Not inside a wt-managed worktree.")
				return nil
			}
			fmt.Printf("Task:      %s\n", t.ID)
			fmt.Printf("Desc:      %s\n", t.Description)
			fmt.Printf("Branch:    %s\n", t.Branch)
			fmt.Printf("Worktree:  %s\n", t.Worktree)
			fmt.Printf("Created:   %s\n", t.Created.Format("2006-01-02 15:04"))
			if t.TicketKey != "" {
				fmt.Printf("Ticket:    %s (%s)\n", t.TicketKey, t.Connector)
			}
			return nil
		},
	}
}

// --- connect ---
func connectCmd() *cli.Command {
	return &cli.Command{
		Name:      "connect",
		Category:  "config",
		Usage:     "Configure a task management connector",
		ArgsUsage: "<connector-name>",
		Description: `Configure integration with external task management systems.

   Currently supports Jira with planned support for Monday.com and ClickUp.
   Once configured, use 'wt start --jira <KEY>' to create worktrees from tickets.

   Example:
     wt connect jira --url https://company.atlassian.net --email user@company.com --token TOKEN`,
		Subcommands: []*cli.Command{
			{
				Name:  "jira",
				Usage: "Configure Jira integration",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "url", Usage: "Jira base URL (e.g. https://yourco.atlassian.net)", Required: true},
					&cli.StringFlag{Name: "email", Usage: "Your Jira email address", Required: true},
					&cli.StringFlag{Name: "token", Usage: "Jira API token", Required: true},
					&cli.StringFlag{Name: "project", Usage: "Default Jira project key"},
				},
				Action: func(c *cli.Context) error {
					cfg, err := loadConfig()
					if err != nil {
						return err
					}
					client := jira.New(c.String("url"), c.String("email"), c.String("token"))
					fmt.Print("Validating Jira credentials... ")
					if err := client.Validate(context.Background()); err != nil {
						fmt.Println("❌")
						return fmt.Errorf("validation failed: %w", err)
					}
					fmt.Println("✅")

					if err := cfg.SetConnector("jira", config.ConnectorConfig{
						URL:      c.String("url"),
						Email:    c.String("email"),
						APIToken: c.String("token"),
						Project:  c.String("project"),
					}); err != nil {
						return err
					}
					fmt.Println("Jira connector configured successfully.")
					return nil
				},
			},
		},
	}
}

// --- sync ---
func syncCmd() *cli.Command {
	return &cli.Command{
		Name:     "sync",
		Category: "config",
		Usage:    "Fetch assigned tickets from a connected system",
		Description: `List tickets assigned to you from a connected task management system.

   Shows ticket key, summary, and current status. Requires a configured connector.
   Use 'wt connect' first to set up integration with Jira, Monday.com, or ClickUp.

   Examples:
     wt sync                    # Defaults to jira
     wt sync --connector jira   # Explicit connector`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "connector", Aliases: []string{"c"}, Value: "jira", Usage: "Connector to sync from"},
		},
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			reg := buildRegistry(cfg)
			name := c.String("connector")
			conn, ok := reg.Get(name)
			if !ok {
				return fmt.Errorf("connector %q not found; available: %v", name, reg.List())
			}

			fmt.Printf("Syncing from %s...\n", name)
			tickets, err := conn.ListAssigned(context.Background())
			if err != nil {
				return err
			}
			if len(tickets) == 0 {
				fmt.Println("No assigned tickets found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "KEY\tSUMMARY\tSTATUS")
			for _, t := range tickets {
				fmt.Fprintf(w, "%s\t%s\t%s\n", t.Key, truncate(t.Summary, 50), t.Status)
			}
			return w.Flush()
		},
	}
}

// --- config ---
func configCmd() *cli.Command {
	return &cli.Command{
		Name:      "config",
		Category:  "config",
		Usage:     "View or set configuration values",
		ArgsUsage: "[key] [value]",
		Description: `View or modify wt configuration settings.

   Configuration is stored in ~/.wt/config.yaml.

   Available keys:
     worktrees_base  - Base directory for worktrees (default: ~/worktrees)
     default_branch  - Main branch name (default: main)
     branch_prefix   - Prefix for new branches (default: feature)
     default_agent   - Default AI agent to launch

   Examples:
     wt config                              # Show all settings
     wt config worktrees_base               # Show specific value
     wt config worktrees_base ~/my-trees   # Set value`,
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if c.NArg() == 0 {
				fmt.Printf("worktrees_base: %s\n", cfg.WorktreesBase)
				fmt.Printf("default_branch: %s\n", cfg.DefaultBranch)
				fmt.Printf("branch_prefix:  %s\n", cfg.BranchPrefix)
				if cfg.DefaultAgent != "" {
					fmt.Printf("default_agent:  %s\n", cfg.DefaultAgent)
				}
				if len(cfg.AgentAliases) > 0 {
					fmt.Printf("agent_aliases:\n")
					for k, v := range cfg.AgentAliases {
						fmt.Printf("  %s: %s\n", k, v)
					}
				}
				fmt.Printf("connectors:     %v\n", connectorNames(cfg))
				return nil
			}
			key := c.Args().Get(0)
			if c.NArg() == 1 {
				switch key {
				case "worktrees_base":
					fmt.Println(cfg.WorktreesBase)
				case "default_branch":
					fmt.Println(cfg.DefaultBranch)
				case "branch_prefix":
					fmt.Println(cfg.BranchPrefix)
				case "default_agent":
					fmt.Println(cfg.DefaultAgent)
				default:
					return fmt.Errorf("unknown config key: %s", key)
				}
				return nil
			}
			value := c.Args().Get(1)
			switch key {
			case "worktrees_base":
				cfg.WorktreesBase = value
			case "default_branch":
				cfg.DefaultBranch = value
			case "branch_prefix":
				cfg.BranchPrefix = value
			case "default_agent":
				cfg.DefaultAgent = value
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

// --- prune ---
func pruneCmd() *cli.Command {
	return &cli.Command{
		Name:     "prune",
		Category: "maintenance",
		Usage:    "Clean up stale worktree references",
		Description: `Remove stale git worktree administrative files.

   Cleans up references to worktrees that have been manually deleted or moved.
   This runs 'git worktree prune' in the repository.

   Example:
     wt prune`,
		Action: func(c *cli.Context) error {
			repoPath, err := getRepoPath()
			if err != nil {
				return err
			}
			if err := worktree.Prune(repoPath); err != nil {
				return err
			}
			fmt.Println("✅ Pruned stale worktree references.")
			return nil
		},
	}
}

// --- helpers ---

func joinArgs(c *cli.Context) string {
	args := make([]string, c.NArg())
	for i := 0; i < c.NArg(); i++ {
		args[i] = c.Args().Get(i)
	}
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func connectorNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Connectors))
	for k := range cfg.Connectors {
		names = append(names, k)
	}
	if len(names) == 0 {
		return []string{"(none)"}
	}
	return names
}
