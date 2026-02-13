package task

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bakerweb/wt/internal/config"
	"github.com/bakerweb/wt/internal/worktree"
)

// Manager handles task lifecycle operations.
type Manager struct {
	Config *config.Config
}

// NewManager creates a new task manager.
func NewManager(cfg *config.Config) *Manager {
	return &Manager{Config: cfg}
}

// StartOptions configures a new task.
type StartOptions struct {
	Description string
	RepoPath    string
	Connector   string
	TicketKey   string
	TicketTitle string
}

// Start creates a new task with an associated worktree.
func (m *Manager) Start(opts StartOptions) (*config.Task, error) {
	repoName, err := worktree.RepoName(opts.RepoPath)
	if err != nil {
		return nil, err
	}

	id := generateID()
	prefix := m.Config.BranchPrefix

	var branch string
	if opts.TicketKey != "" {
		title := opts.TicketTitle
		if title == "" {
			title = opts.Description
		}
		branch = worktree.BranchNameFromTicket(prefix, opts.TicketKey, title)
	} else {
		branch = worktree.BranchName(prefix, opts.Description)
	}

	// Check if branch already exists
	if worktree.BranchExists(opts.RepoPath, branch) {
		return nil, fmt.Errorf("branch %q already exists; use a different description or remove the existing branch", branch)
	}

	wtPath := filepath.Join(m.Config.WorktreesBase, repoName, worktree.SanitizeBranchName(opts.Description))

	if err := os.MkdirAll(filepath.Dir(wtPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create worktree directory: %w", err)
	}

	if err := worktree.Create(opts.RepoPath, wtPath, branch); err != nil {
		return nil, err
	}

	task := config.Task{
		ID:          id,
		Description: opts.Description,
		Worktree:    wtPath,
		Branch:      branch,
		RepoPath:    opts.RepoPath,
		Connector:   opts.Connector,
		TicketKey:   opts.TicketKey,
		Created:     time.Now(),
	}

	if err := m.Config.AddTask(task); err != nil {
		return nil, fmt.Errorf("task created but failed to save: %w", err)
	}

	return &task, nil
}

// Finish removes the worktree and cleans up the task.
func (m *Manager) Finish(id string) (*config.Task, error) {
	task, err := m.Config.FindTask(id)
	if err != nil {
		return nil, err
	}

	if err := worktree.Remove(task.RepoPath, task.Worktree); err != nil {
		return nil, fmt.Errorf("failed to remove worktree: %w", err)
	}

	if err := worktree.DeleteBranch(task.RepoPath, task.Branch); err != nil {
		// Non-fatal: branch might have been merged/deleted already
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	if err := m.Config.RemoveTask(id); err != nil {
		return nil, err
	}

	return task, nil
}

// Remove removes a worktree but keeps the branch.
func (m *Manager) Remove(id string) (*config.Task, error) {
	task, err := m.Config.FindTask(id)
	if err != nil {
		return nil, err
	}

	if err := worktree.Remove(task.RepoPath, task.Worktree); err != nil {
		return nil, fmt.Errorf("failed to remove worktree: %w", err)
	}

	if err := m.Config.RemoveTask(id); err != nil {
		return nil, err
	}

	return task, nil
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("wt-%x", b)
}
