package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAndResolve(t *testing.T) {
	tmpRoot := t.TempDir()

	mysqlDir := filepath.Join(tmpRoot, "database", "mysql")
	pgDir := filepath.Join(tmpRoot, "database", "postgresql")
	skippedAgentDir := filepath.Join(tmpRoot, "agents", "specs")

	for _, dir := range []string{mysqlDir, pgDir, skippedAgentDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	mysqlJSON := `{
  "schemaVersion": "1.0",
  "id": "mysql",
  "name": "MySQL MCP",
  "description": "MySQL test manifest",
  "runtime": "go",
  "build": {"type": "go", "source": ".", "binary": "mysql-mcp"},
  "run": {"command": "{bin_dir}/mysql-mcp{exe_ext}", "args": ["--mode", "{mcp_dir}"]},
  "env": [{"name": "DATABASE_HOST", "required": true, "default": "127.0.0.1"}]
}`
	pgJSON := `{
  "schemaVersion": "1.0",
  "id": "postgresql",
  "name": "PostgreSQL MCP",
  "description": "PostgreSQL test manifest",
  "runtime": "go",
  "build": {"type": "go", "source": ".", "binary": "postgresql-mcp"},
  "run": {"command": "{bin_dir}/postgresql-mcp{exe_ext}", "args": []}
}`

	if err := os.WriteFile(filepath.Join(mysqlDir, ManifestFileName), []byte(mysqlJSON), 0o644); err != nil {
		t.Fatalf("write mysql manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pgDir, ManifestFileName), []byte(pgJSON), 0o644); err != nil {
		t.Fatalf("write pg manifest: %v", err)
	}
	// Should be ignored because "agents" is in skippedDirs
	if err := os.WriteFile(filepath.Join(skippedAgentDir, ManifestFileName), []byte(mysqlJSON), 0o644); err != nil {
		t.Fatalf("write skipped manifest: %v", err)
	}

	manifests, err := Discover(tmpRoot)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(manifests) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(manifests))
	}
	if manifests[0].ID != "mysql" || manifests[1].ID != "postgresql" {
		t.Fatalf("unexpected manifest order/IDs: %+v", manifests)
	}

	binDir := filepath.Join(tmpRoot, ".mcp", "bin")
	cmdWin, argsWin := manifests[0].ResolveCommand(binDir, "windows")
	if filepath.Base(cmdWin) != "mysql-mcp.exe" {
		t.Errorf("expected windows binary mysql-mcp.exe, got %s", cmdWin)
	}
	if len(argsWin) != 2 || argsWin[1] != mysqlDir {
		t.Errorf("unexpected resolved args: %v", argsWin)
	}

	cmdLinux, _ := manifests[0].ResolveCommand(binDir, "linux")
	if filepath.Base(cmdLinux) != "mysql-mcp" {
		t.Errorf("expected linux binary mysql-mcp, got %s", cmdLinux)
	}
}

func TestDiscoverDuplicateID(t *testing.T) {
	tmpRoot := t.TempDir()
	dirA := filepath.Join(tmpRoot, "a")
	dirB := filepath.Join(tmpRoot, "b")
	_ = os.MkdirAll(dirA, 0o755)
	_ = os.MkdirAll(dirB, 0o755)

	content := `{
  "schemaVersion": "1.0",
  "id": "dup",
  "name": "Duplicate",
  "runtime": "go",
  "build": {"type": "go", "source": ".", "binary": "dup-mcp"},
  "run": {"command": "{bin_dir}/dup-mcp{exe_ext}", "args": []}
}`
	_ = os.WriteFile(filepath.Join(dirA, ManifestFileName), []byte(content), 0o644)
	_ = os.WriteFile(filepath.Join(dirB, ManifestFileName), []byte(content), 0o644)

	if _, err := Discover(tmpRoot); err == nil {
		t.Fatal("expected error for duplicate manifest ID, got nil")
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		name string
		m    Manifest
	}{
		{
			name: "wrong schema version",
			m:    Manifest{SchemaVersion: "2.0", ID: "x", Name: "X", Runtime: "go", Build: BuildSpec{Type: "go", Binary: "x"}, Run: RunSpec{Command: "x"}},
		},
		{
			name: "missing id",
			m:    Manifest{SchemaVersion: "1.0", Name: "X", Runtime: "go", Build: BuildSpec{Type: "go", Binary: "x"}, Run: RunSpec{Command: "x"}},
		},
		{
			name: "unsupported build type",
			m:    Manifest{SchemaVersion: "1.0", ID: "x", Name: "X", Runtime: "go", Build: BuildSpec{Type: "unknown"}, Run: RunSpec{Command: "x"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.m.Validate(); err == nil {
				t.Errorf("expected validation error for %s", tc.name)
			}
		})
	}
}
