package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"edro-mcp/internal/client"
	"edro-mcp/internal/manifest"
	"edro-mcp/internal/profile"
)

func TestDoctorHealthyAndUnhealthy(t *testing.T) {
	homeDir := t.TempDir()
	binDir := filepath.Join(homeDir, ".mcp", "bin")
	_ = os.MkdirAll(binDir, 0o755)

	m := manifest.Manifest{
		SchemaVersion: "1.0",
		ID:            "mysql",
		Name:          "MySQL MCP",
		Runtime:       "go",
		Build:         manifest.BuildSpec{Type: "go", Binary: "mysql-mcp"},
		Run:           manifest.RunSpec{Command: "{bin_dir}/mysql-mcp{exe_ext}"},
		Env: []manifest.EnvVar{
			{Name: "DATABASE_HOST", Required: true},
		},
	}

	cfg := &profile.Config{
		SchemaVersion: "1.0",
		Profiles: map[string]profile.Profile{
			"mysql-local": {
				MCP:     "mysql",
				Enabled: true,
				Env:     map[string]string{"DATABASE_HOST": "127.0.0.1"},
			},
		},
	}

	// 1. Before creating the binary, doctor should report unhealthy
	rep1 := Run([]manifest.Manifest{m}, cfg, "profiles.json", binDir, homeDir)
	if rep1.Healthy {
		t.Fatal("expected doctor to report unhealthy when binary is missing")
	}

	// 2. Create binary and valid client config -> doctor should report healthy
	binPath := m.BinaryPath(binDir, runtime.GOOS)
	if err := os.WriteFile(binPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	agSpec, _ := client.GetSpec("antigravity")
	agPath := agSpec.ConfigPath(homeDir)
	_ = os.MkdirAll(filepath.Dir(agPath), 0o755)
	_ = os.WriteFile(agPath, []byte(`{"mcpServers":{"mysql-local":{"command":"x"}}}`), 0o600)

	rep2 := Run([]manifest.Manifest{m}, cfg, "profiles.json", binDir, homeDir)
	if !rep2.Healthy {
		t.Fatalf("expected doctor to be healthy, got issues: %v", rep2.Issues)
	}

	// 3. Corrupt client config -> doctor should detect syntax error
	_ = os.WriteFile(agPath, []byte(`{invalid-json`), 0o600)
	rep3 := Run([]manifest.Manifest{m}, cfg, "profiles.json", binDir, homeDir)
	if rep3.Healthy {
		t.Fatal("expected doctor to report unhealthy when client config has invalid JSON")
	}
}
