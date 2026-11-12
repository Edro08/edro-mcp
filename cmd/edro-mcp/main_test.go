package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) (repoRoot string, homeDir string, binDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	homeDir = t.TempDir()
	binDir = filepath.Join(homeDir, ".mcp", "bin")

	// Create dummy Go MCP server so `build` and `install` run fast without network
	mysqlDir := filepath.Join(repoRoot, "database", "mysql")
	if err := os.MkdirAll(mysqlDir, 0o755); err != nil {
		t.Fatalf("mkdir mysql: %v", err)
	}
	_ = os.WriteFile(filepath.Join(mysqlDir, "go.mod"), []byte("module dummy-mysql\n\ngo 1.27\n"), 0o644)
	_ = os.WriteFile(filepath.Join(mysqlDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	manifestJSON := `{
  "schemaVersion": "1.0",
  "id": "mysql",
  "name": "MySQL MCP Server",
  "description": "Test MySQL MCP",
  "runtime": "go",
  "build": {"type": "go", "source": ".", "binary": "mysql-mcp"},
  "run": {"command": "{bin_dir}/mysql-mcp{exe_ext}", "args": []},
  "env": [
    {"name": "DATABASE_HOST", "required": true, "default": "127.0.0.1"},
    {"name": "DATABASE_PASSWORD", "required": true, "default": "{env:MYSQL_LOCAL_PASSWORD}", "sensitive": true}
  ]
}`
	_ = os.WriteFile(filepath.Join(mysqlDir, "mcp.manifest.json"), []byte(manifestJSON), 0o644)

	profilesJSON := `{
  "schemaVersion": "1.0",
  "profiles": {
    "mysql-local": {
      "mcp": "mysql",
      "description": "Local MySQL",
      "enabled": true,
      "env": {
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PASSWORD": "{env:MYSQL_LOCAL_PASSWORD}"
      }
    },
    "mysql-staging": {
      "mcp": "mysql",
      "description": "Staging MySQL",
      "enabled": true,
      "env": {
        "DATABASE_HOST": "staging.internal",
        "DATABASE_PASSWORD": "{env:MYSQL_LOCAL_PASSWORD}"
      }
    }
  }
}`
	_ = os.WriteFile(filepath.Join(repoRoot, "profiles.example.json"), []byte(profilesJSON), 0o600)
	return repoRoot, homeDir, binDir
}

func TestCLIListAndNonInteractiveSyncAndDoctor(t *testing.T) {
	repoRoot, homeDir, binDir := setupTestRepo(t)
	t.Setenv("MYSQL_LOCAL_PASSWORD", "super-secret-pass")

	// 1. Test `list` command
	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		[]string{"list", "--repo-root", repoRoot, "--bin-dir", binDir},
		strings.NewReader(""),
		&stdout,
		&stderr,
		homeDir,
	)
	if code != 0 {
		t.Fatalf("list exited with %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "mysql-mcp") || !strings.Contains(stdout.String(), "DATABASE_HOST") {
		t.Errorf("unexpected list output:\n%s", stdout.String())
	}

	// 2. Test `--dry-run` masks secrets
	stdout.Reset()
	stderr.Reset()
	code = run(
		context.Background(),
		[]string{
			"sync",
			"--repo-root", repoRoot,
			"--bin-dir", binDir,
			"--clients", "antigravity,opencode,codex",
			"--profiles", "mysql-local,mysql-staging",
			"--dry-run",
		},
		strings.NewReader(""),
		&stdout,
		&stderr,
		homeDir,
	)
	if code != 0 {
		t.Fatalf("dry-run sync exited with %d: %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "super-secret-pass") {
		t.Fatalf("dry-run output leaked resolved secret:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "<resolved from $MYSQL_LOCAL_PASSWORD>") {
		t.Errorf("expected masked env placeholder in dry-run output:\n%s", stdout.String())
	}

	// 3. Test real `install` with flags and followed by `doctor`
	stdout.Reset()
	stderr.Reset()
	code = run(
		context.Background(),
		[]string{
			"install",
			"--repo-root", repoRoot,
			"--bin-dir", binDir,
			"--clients", "antigravity,opencode,codex,claude,cursor",
			"--profiles", "mysql-local,mysql-staging",
		},
		strings.NewReader(""),
		&stdout,
		&stderr,
		homeDir,
	)
	if code != 0 {
		t.Fatalf("install exited with %d: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(
		context.Background(),
		[]string{"doctor", "--repo-root", repoRoot, "--bin-dir", binDir},
		strings.NewReader(""),
		&stdout,
		&stderr,
		homeDir,
	)
	if code != 0 {
		t.Fatalf("doctor exited with %d:\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
}

func TestCLIInteractiveFlow(t *testing.T) {
	repoRoot, homeDir, binDir := setupTestRepo(t)
	t.Setenv("MYSQL_LOCAL_PASSWORD", "interactive-secret")

	// Simulate user input:
	// 1. "s\n" -> create profiles.json from profiles.example.json
	// 2. "1,2\n" -> select Antigravity and OpenCode
	// 3. "1\n" -> select mysql-local
	userInput := "s\n1,2\n1\n"
	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		[]string{"install", "--repo-root", repoRoot, "--bin-dir", binDir},
		strings.NewReader(userInput),
		&stdout,
		&stderr,
		homeDir,
	)
	if code != 0 {
		t.Fatalf("interactive install exited with %d: %s\nstdout:\n%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "profiles.json")); err != nil {
		t.Errorf("expected local profiles.json to be created during interactive init: %v", err)
	}
}
