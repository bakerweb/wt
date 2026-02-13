package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

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

func Run(args []string) error {
	app := &cli.App{
		Name:    "wt",
		Usage:   "Git worktree manager driven by tasks",
		Version: Version,
		Commands: []*cli.Command{
			startCmd(),
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
		Usage:     "Create a new worktree for a task",
		ArgsUsage: "<task-description>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "jira",
				Usage: "Create worktree from a Jira issue key (e.g. PROJ-123)",
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
			fmt.Printf("\n   cd %s\n", t.Worktree)
			return nil
		},
	}
}

// --- list ---
func listCmd() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "Show all active tasks and worktrees",
		Aliases: []string{"ls"},
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
		Usage:     "Complete a task, remove worktree and branch",
		ArgsUsage: "<task-id>",
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
		Usage:     "Remove a worktree but keep the branch",
		Aliases:   []string{"rm"},
		ArgsUsage: "<task-id>",
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
		Usage:     "Print the path to a task's worktree (use with cd)",
		ArgsUsage: "<task-id>",
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
		Name:  "status",
		Usage: "Show status of the current worktree task",
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
		Usage:     "Configure a task management connector",
		ArgsUsage: "<connector-name>",
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
		Name:  "sync",
		Usage: "Fetch assigned tickets from a connected system",
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
		Usage:     "View or set configuration values",
		ArgsUsage: "[key] [value]",
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if c.NArg() == 0 {
				fmt.Printf("worktrees_base: %s\n", cfg.WorktreesBase)
				fmt.Printf("default_branch: %s\n", cfg.DefaultBranch)
				fmt.Printf("branch_prefix:  %s\n", cfg.BranchPrefix)
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
		Name:  "prune",
		Usage: "Clean up stale worktree references",
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
