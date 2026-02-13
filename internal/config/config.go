package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".wt"
	configFile = "config.yaml"
)

// Config represents the top-level configuration for wt.
type Config struct {
	WorktreesBase string                       `yaml:"worktrees_base"`
	DefaultBranch string                       `yaml:"default_branch"`
	BranchPrefix  string                       `yaml:"branch_prefix"`
	DefaultAgent  string                       `yaml:"default_agent,omitempty"`
	AgentAliases  map[string]string            `yaml:"agent_aliases,omitempty"`
	Connectors    map[string]ConnectorConfig   `yaml:"connectors,omitempty"`
	Tasks         []Task                       `yaml:"tasks,omitempty"`

	path string `yaml:"-"`
	mu   sync.Mutex `yaml:"-"`
}

// ConnectorConfig stores settings for a task management connector.
type ConnectorConfig struct {
	URL      string `yaml:"url,omitempty"`
	Email    string `yaml:"email,omitempty"`
	APIToken string `yaml:"api_token,omitempty"`
	Project  string `yaml:"project,omitempty"`
}

// Task represents an active worktree task.
type Task struct {
	ID          string    `yaml:"id"`
	Description string    `yaml:"description"`
	Worktree    string    `yaml:"worktree"`
	Branch      string    `yaml:"branch"`
	RepoPath    string    `yaml:"repo_path"`
	Connector   string    `yaml:"connector,omitempty"`
	TicketKey   string    `yaml:"ticket_key,omitempty"`
	Created     time.Time `yaml:"created"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		WorktreesBase: filepath.Join(home, "worktrees"),
		DefaultBranch: "main",
		BranchPrefix:  "feature",
		DefaultAgent:  "",
		AgentAliases:  make(map[string]string),
		Connectors:    make(map[string]ConnectorConfig),
		Tasks:         []Task{},
	}
}

// ConfigDir returns the path to the wt config directory.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configDir), nil
}

// Load reads the config from disk, or returns defaults if none exists.
func Load() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, configFile)
	cfg := DefaultConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Connectors == nil {
		cfg.Connectors = make(map[string]ConnectorConfig)
	}
	if cfg.Tasks == nil {
		cfg.Tasks = []Task{}
	}
	if cfg.AgentAliases == nil {
		cfg.AgentAliases = make(map[string]string)
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		dir, err := ConfigDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, configFile)
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(c.path, data, 0o644)
}

// AddTask adds a task and persists the config.
func (c *Config) AddTask(t Task) error {
	c.Tasks = append(c.Tasks, t)
	return c.Save()
}

// RemoveTask removes a task by ID and persists the config.
func (c *Config) RemoveTask(id string) error {
	for i, t := range c.Tasks {
		if t.ID == id {
			c.Tasks = append(c.Tasks[:i], c.Tasks[i+1:]...)
			return c.Save()
		}
	}
	return fmt.Errorf("task %q not found", id)
}

// FindTask finds a task by ID.
func (c *Config) FindTask(id string) (*Task, error) {
	for i := range c.Tasks {
		if c.Tasks[i].ID == id {
			return &c.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %q not found", id)
}

// FindTaskByWorktree finds a task whose worktree path matches the given directory.
func (c *Config) FindTaskByWorktree(dir string) (*Task, error) {
	for i := range c.Tasks {
		if c.Tasks[i].Worktree == dir {
			return &c.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("no task found for worktree %q", dir)
}

// SetConnector stores connector configuration.
func (c *Config) SetConnector(name string, cc ConnectorConfig) error {
	c.Connectors[name] = cc
	return c.Save()
}
