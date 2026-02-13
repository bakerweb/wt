package worktree

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizeBranchName converts a description into a valid git branch name.
func SanitizeBranchName(description string) string {
	s := strings.ToLower(strings.TrimSpace(description))
	// Replace any non-alphanumeric characters (except hyphens) with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// Limit length
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// BranchName generates a full branch name from a prefix and description.
func BranchName(prefix, description string) string {
	sanitized := SanitizeBranchName(description)
	if prefix == "" {
		return sanitized
	}
	return prefix + "/" + sanitized
}

// BranchNameFromTicket generates a branch name from a ticket key and summary.
func BranchNameFromTicket(prefix, ticketKey, summary string) string {
	sanitized := SanitizeBranchName(summary)
	key := strings.ToLower(ticketKey)
	name := key + "-" + sanitized
	if len(name) > 60 {
		name = name[:60]
		name = strings.TrimRight(name, "-")
	}
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// RepoName extracts the repository name from a git repo path.
func RepoName(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %s", repoPath)
	}
	return filepath.Base(strings.TrimSpace(string(out))), nil
}

// Create creates a new git worktree at the specified path with the given branch.
func Create(repoPath, worktreePath, branch string) error {
	// Create the new branch and worktree in one step
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", "-b", branch, worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree: %s\n%s", err, string(out))
	}
	return nil
}

// CreateFromExistingBranch creates a worktree from an existing branch.
func CreateFromExistingBranch(repoPath, worktreePath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create worktree: %s\n%s", err, string(out))
	}
	return nil
}

// Remove removes a git worktree.
func Remove(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", worktreePath, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove worktree: %s\n%s", err, string(out))
	}
	return nil
}

// List lists all worktrees for a repository.
func List(repoPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}
	return parseWorktreeList(string(out)), nil
}

// DeleteBranch deletes a local git branch.
func DeleteBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "branch", "-D", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch %q: %s\n%s", branch, err, string(out))
	}
	return nil
}

// BranchExists checks if a branch already exists.
func BranchExists(repoPath, branch string) bool {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", branch)
	return cmd.Run() == nil
}

// Prune removes stale worktree administrative files.
func Prune(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "prune")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %s\n%s", err, string(out))
	}
	return nil
}

// DefaultBranch detects the default branch of a repository.
func DefaultBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	ref := strings.TrimSpace(string(out))
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// WorktreeInfo holds parsed worktree information.
type WorktreeInfo struct {
	Path   string
	HEAD   string
	Branch string
	Bare   bool
}

func parseWorktreeList(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	var current WorktreeInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch ")
		} else if line == "bare" {
			current.Bare = true
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}
	return worktrees
}
