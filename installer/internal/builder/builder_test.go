package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"edro-mcp/internal/manifest"
)

func TestBuildGoAndDeduplicate(t *testing.T) {
	tmpDir := t.TempDir()
	mcpDir := filepath.Join(tmpDir, "dummy-mcp")
	binDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(mcpDir, 0o755); err != nil {
		t.Fatalf("mkdir mcpDir: %v", err)
	}

	goMod := "module dummy\n\ngo 1.27\n"
	mainGo := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(mcpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mcpDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	m := manifest.Manifest{
		SchemaVersion: "1.0",
		ID:            "dummy",
		Name:          "Dummy MCP",
		Runtime:       "go",
		Dir:           mcpDir,
		Build: manifest.BuildSpec{
			Type:   "go",
			Source: ".",
			Binary: "dummy-mcp",
		},
		Run: manifest.RunSpec{
			Command: "{bin_dir}/dummy-mcp{exe_ext}",
		},
	}

	// 1. Test dry-run does not create binary
	dryRes, err := Build(context.Background(), m, binDir, true)
	if err != nil {
		t.Fatalf("dry-run Build failed: %v", err)
	}
	if !dryRes.DryRun {
		t.Errorf("expected DryRun=true")
	}
	if _, err := os.Stat(dryRes.BinaryPath); !os.IsNotExist(err) {
		t.Errorf("expected binary not to exist after dry-run")
	}

	// 2. Test BuildMany deduplication and real compilation
	results, err := BuildMany(context.Background(), []manifest.Manifest{m, m}, binDir, false)
	if err != nil {
		t.Fatalf("BuildMany failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 deduplicated build result, got %d", len(results))
	}
	if _, err := os.Stat(results[0].BinaryPath); err != nil {
		t.Fatalf("expected compiled binary at %s: %v", results[0].BinaryPath, err)
	}
}

func TestBuildCopy(t *testing.T) {
	tmpDir := t.TempDir()
	mcpDir := filepath.Join(tmpDir, "script-mcp")
	binDir := filepath.Join(tmpDir, "bin")
	_ = os.MkdirAll(mcpDir, 0o755)

	scriptFile := filepath.Join(mcpDir, "runner.sh")
	_ = os.WriteFile(scriptFile, []byte("#!/bin/sh\necho ok\n"), 0o644)

	m := manifest.Manifest{
		SchemaVersion: "1.0",
		ID:            "script-mcp",
		Name:          "Script MCP",
		Runtime:       "sh",
		Dir:           mcpDir,
		Build: manifest.BuildSpec{
			Type:   "copy",
			Source: "runner.sh",
			Binary: "script-mcp",
		},
		Run: manifest.RunSpec{
			Command: "{bin_dir}/script-mcp{exe_ext}",
		},
	}

	res, err := Build(context.Background(), m, binDir, false)
	if err != nil {
		t.Fatalf("copy Build failed: %v", err)
	}
	if _, err := os.Stat(res.BinaryPath); err != nil {
		t.Fatalf("expected copied artifact at %s: %v", res.BinaryPath, err)
	}
}
