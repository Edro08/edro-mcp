package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleEntries(homeDir string) []Entry {
	binPath := filepath.Join(homeDir, ".mcp", "bin", "mysql-mcp")
	return []Entry{
		{
			ProfileName: "mysql-local",
			Command:     binPath,
			Args:        []string{},
			Env: map[string]string{
				"DATABASE_HOST":     "127.0.0.1",
				"DATABASE_PASSWORD": "real-secret-value",
			},
			MaskedEnv: map[string]string{
				"DATABASE_HOST":     "127.0.0.1",
				"DATABASE_PASSWORD": "<resolved from $MYSQL_LOCAL_PASSWORD>",
			},
		},
		{
			ProfileName: "mysql-staging",
			Command:     binPath,
			Args:        []string{},
			Env: map[string]string{
				"DATABASE_HOST":     "staging.internal",
				"DATABASE_PASSWORD": "staging-secret-value",
			},
			MaskedEnv: map[string]string{
				"DATABASE_HOST":     "staging.internal",
				"DATABASE_PASSWORD": "<resolved from $MYSQL_STAGING_PASSWORD>",
			},
		},
	}
}

func TestSyncStandardJSONClientsPreservesExisting(t *testing.T) {
	homeDir := t.TempDir()
	entries := sampleEntries(homeDir)

	for _, clientID := range []string{"antigravity", "claude", "cursor"} {
		t.Run(clientID, func(t *testing.T) {
			spec, err := GetSpec(clientID)
			if err != nil {
				t.Fatalf("GetSpec(%s): %v", clientID, err)
			}
			cfgPath := spec.ConfigPath(homeDir)
			if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			initial := `{
  "customSetting": "keep-me",
  "mcpServers": {
    "chrome-devtools": {
      "command": "npx",
      "args": ["-y", "chrome-devtools-mcp"]
    }
  }
}`
			if err := os.WriteFile(cfgPath, []byte(initial), 0o600); err != nil {
				t.Fatalf("write initial config: %v", err)
			}

			// 1. Dry-run should mask secrets and not modify file or create .bak
			dryRes, err := SyncClient(homeDir, clientID, entries, true)
			if err != nil {
				t.Fatalf("dry-run SyncClient(%s): %v", clientID, err)
			}
			if strings.Contains(dryRes.DryRunDiff, "real-secret-value") {
				t.Errorf("dry-run leaked secret in diff for %s", clientID)
			}
			if !strings.Contains(dryRes.DryRunDiff, "<resolved from $MYSQL_LOCAL_PASSWORD>") {
				t.Errorf("expected masked placeholder in dry-run diff for %s", clientID)
			}
			if _, err := os.Stat(cfgPath + ".bak"); !os.IsNotExist(err) {
				t.Errorf("dry-run should not create .bak file")
			}

			// 2. Real sync creates .bak and preserves customSetting + chrome-devtools
			res, err := SyncClient(homeDir, clientID, entries, false)
			if err != nil {
				t.Fatalf("SyncClient(%s): %v", clientID, err)
			}
			if res.BackupPath == "" {
				t.Errorf("expected backup path for %s", clientID)
			}
			bakBytes, err := os.ReadFile(res.BackupPath)
			if err != nil || string(bakBytes) != initial {
				t.Errorf("backup content mismatch for %s", clientID)
			}

			exists, profiles, err := InspectClientConfig(homeDir, clientID)
			if err != nil || !exists {
				t.Fatalf("InspectClientConfig(%s) failed: exists=%v err=%v", clientID, exists, err)
			}
			expectedProfiles := "chrome-devtools,mysql-local,mysql-staging"
			if strings.Join(profiles, ",") != expectedProfiles {
				t.Errorf("expected profiles %s, got %v", expectedProfiles, profiles)
			}

			updatedBytes, _ := os.ReadFile(cfgPath)
			if !strings.Contains(string(updatedBytes), `"customSetting": "keep-me"`) {
				t.Errorf("expected customSetting to be preserved in %s", clientID)
			}
			if clientID == "claude" && !strings.Contains(string(updatedBytes), `"type": "stdio"`) {
				t.Errorf("expected claude entry to include type=stdio")
			}
		})
	}
}

func TestSyncOpenCodeJSONCPreservesCommentsAndSettings(t *testing.T) {
	homeDir := t.TempDir()
	entries := sampleEntries(homeDir)

	spec, _ := GetSpec("opencode")
	cfgPath := spec.ConfigPath(homeDir)
	_ = os.MkdirAll(filepath.Dir(cfgPath), 0o755)

	initialJSONC := `// Global OpenCode configuration comment
/* Multi-line header block */
{
  "$schema": "https://opencode.ai/config.json",
  "theme": "tokyonight",
  // Existing MCP servers
  "mcp": {
    "servers": {
      "external-mcp": {
        "type": "local",
        "command": ["npx", "external-mcp"],
      },
    },
  },
}
`
	if err := os.WriteFile(cfgPath, []byte(initialJSONC), 0o600); err != nil {
		t.Fatalf("write initial jsonc: %v", err)
	}

	res, err := SyncClient(homeDir, "opencode", entries, false)
	if err != nil {
		t.Fatalf("SyncClient(opencode) failed: %v", err)
	}
	if res.BackupPath == "" {
		t.Errorf("expected .bak file for opencode")
	}

	updated, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read updated opencode.jsonc: %v", err)
	}
	text := string(updated)

	if !strings.HasPrefix(text, "// Global OpenCode configuration comment") {
		t.Errorf("expected leading comment to be preserved, got:\n%s", text)
	}
	if !strings.Contains(text, `"theme": "tokyonight"`) {
		t.Errorf("expected theme setting to be preserved")
	}

	exists, profiles, err := InspectClientConfig(homeDir, "opencode")
	if err != nil || !exists {
		t.Fatalf("InspectClientConfig(opencode) failed: %v", err)
	}
	if strings.Join(profiles, ",") != "external-mcp,mysql-local,mysql-staging" {
		t.Errorf("unexpected profiles in opencode: %v", profiles)
	}
}

func TestSyncCodexTOMLPreservesSectionsAndUpdatesIdempotently(t *testing.T) {
	homeDir := t.TempDir()
	entries := sampleEntries(homeDir)

	spec, _ := GetSpec("codex")
	cfgPath := spec.ConfigPath(homeDir)
	_ = os.MkdirAll(filepath.Dir(cfgPath), 0o755)

	initialTOML := `# Codex configuration
model = "gpt-5-codex"
approval_policy = "on-request"

[mcp_servers.chrome-devtools]
command = "npx"
args = ["-y", "chrome-devtools-mcp"]

[mcp_servers.mysql-local]
command = "old-command"
args = ["--old"]

[mcp_servers.mysql-local.env]
OLD_VAR = "1"

[projects."workspace"]
trust_level = "trusted"
`
	if err := os.WriteFile(cfgPath, []byte(initialTOML), 0o600); err != nil {
		t.Fatalf("write initial toml: %v", err)
	}

	res, err := SyncClient(homeDir, "codex", entries, false)
	if err != nil {
		t.Fatalf("SyncClient(codex) failed: %v", err)
	}
	if res.BackupPath == "" {
		t.Errorf("expected .bak file for codex")
	}

	updated, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read updated config.toml: %v", err)
	}
	text := string(updated)

	if !strings.Contains(text, `model = "gpt-5-codex"`) {
		t.Errorf("expected root model key to be preserved")
	}
	if !strings.Contains(text, `[mcp_servers.chrome-devtools]`) {
		t.Errorf("expected external mcp_servers.chrome-devtools section to be preserved")
	}
	if !strings.Contains(text, `[projects."workspace"]`) {
		t.Errorf("expected projects section to be preserved")
	}
	if strings.Contains(text, "old-command") || strings.Contains(text, "OLD_VAR") {
		t.Errorf("expected old mysql-local section to be replaced, got:\n%s", text)
	}

	exists, profiles, err := InspectClientConfig(homeDir, "codex")
	if err != nil || !exists {
		t.Fatalf("InspectClientConfig(codex) failed: %v", err)
	}
	if strings.Join(profiles, ",") != "chrome-devtools,mysql-local,mysql-staging" {
		t.Errorf("unexpected profiles in codex: %v", profiles)
	}
}
