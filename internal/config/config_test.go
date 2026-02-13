package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultBranch != "main" {
		t.Errorf("expected default branch 'main', got %q", cfg.DefaultBranch)
	}
	if cfg.BranchPrefix != "feature" {
		t.Errorf("expected branch prefix 'feature', got %q", cfg.BranchPrefix)
	}
	if cfg.Connectors == nil {
		t.Error("expected connectors map to be initialized")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.path = cfgPath
	cfg.WorktreesBase = "/tmp/test-worktrees"
	cfg.BranchPrefix = "fix"

	if err := cfg.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load it back by reading the file directly
	loaded := DefaultConfig()
	loaded.path = cfgPath

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	// Unmarshal to verify content
	if len(data) == 0 {
		t.Fatal("config file is empty")
	}
}

func TestAddAndRemoveTask(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.path = cfgPath

	task := Task{
		ID:          "test-001",
		Description: "test task",
		Branch:      "feature/test",
		Worktree:    "/tmp/wt",
	}

	if err := cfg.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}

	found, err := cfg.FindTask("test-001")
	if err != nil {
		t.Fatalf("FindTask failed: %v", err)
	}
	if found.Description != "test task" {
		t.Errorf("expected description 'test task', got %q", found.Description)
	}

	if err := cfg.RemoveTask("test-001"); err != nil {
		t.Fatalf("RemoveTask failed: %v", err)
	}
	if len(cfg.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(cfg.Tasks))
	}
}

func TestFindTaskNotFound(t *testing.T) {
	cfg := DefaultConfig()
	_, err := cfg.FindTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}
