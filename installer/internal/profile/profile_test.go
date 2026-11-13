package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"edro-mcp/internal/manifest"
)

func TestLoadPrecedence(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()

	exampleJSON := `{
  "schemaVersion": "1.0",
  "profiles": {
    "from-example": {
      "mcp": "mysql",
      "enabled": true,
      "env": {"DATABASE_HOST": "127.0.0.1"}
    }
  }
}`
	if err := os.WriteFile(filepath.Join(repoRoot, ExampleProfilesName), []byte(exampleJSON), 0o600); err != nil {
		t.Fatalf("write example: %v", err)
	}

	// 1. Should load fallback profiles.example.json when neither local nor home profiles.json exists.
	cfg, source, err := Load("", repoRoot, homeDir)
	if err != nil {
		t.Fatalf("Load fallback failed: %v", err)
	}
	if filepath.Base(source) != ExampleProfilesName {
		t.Fatalf("expected source %s, got %s", ExampleProfilesName, source)
	}
	if _, ok := cfg.Profiles["from-example"]; !ok {
		t.Fatalf("expected profile from-example")
	}

	// 2. Create ~/.mcp/profiles.json and verify it takes precedence over profiles.example.json.
	userMCPDir := filepath.Join(homeDir, ".mcp")
	_ = os.MkdirAll(userMCPDir, 0o755)
	userJSON := `{
  "schemaVersion": "1.0",
  "profiles": {
    "from-user": {
      "mcp": "postgresql",
      "enabled": true,
      "env": {"DATABASE_HOST": "10.0.0.1"}
    }
  }
}`
	_ = os.WriteFile(filepath.Join(userMCPDir, ProfilesFileName), []byte(userJSON), 0o600)

	cfg, source, err = Load("", repoRoot, homeDir)
	if err != nil {
		t.Fatalf("Load user profiles failed: %v", err)
	}
	if !strings.Contains(source, ".mcp") {
		t.Fatalf("expected user .mcp source, got %s", source)
	}
	if _, ok := cfg.Profiles["from-user"]; !ok {
		t.Fatalf("expected profile from-user")
	}

	// 3. Create <repoRoot>/profiles.json and verify it takes precedence over ~/.mcp/profiles.json.
	localJSON := `{
  "schemaVersion": "1.0",
  "profiles": {
    "from-local": {
      "mcp": "mysql",
      "enabled": true,
      "env": {"DATABASE_HOST": "192.168.1.10"}
    }
  }
}`
	_ = os.WriteFile(filepath.Join(repoRoot, ProfilesFileName), []byte(localJSON), 0o600)

	cfg, source, err = Load("", repoRoot, homeDir)
	if err != nil {
		t.Fatalf("Load local profiles failed: %v", err)
	}
	if source != filepath.Join(repoRoot, ProfilesFileName) {
		t.Fatalf("expected repo local profiles.json, got %s", source)
	}
	if _, ok := cfg.Profiles["from-local"]; !ok {
		t.Fatalf("expected profile from-local")
	}
}

func TestResolveEnvPoliciesAndMasking(t *testing.T) {
	m := manifest.Manifest{
		ID: "mysql",
		Env: []manifest.EnvVar{
			{Name: "DATABASE_HOST", Required: true, Sensitive: false},
			{Name: "DATABASE_PASSWORD", Required: true, Sensitive: true},
		},
	}

	rawEnv := map[string]string{
		"DATABASE_HOST":     "127.0.0.1",
		"DATABASE_PASSWORD": "{env:TEST_DB_SECRET}",
	}

	mockLookup := func(key string) (string, bool) {
		if key == "TEST_DB_SECRET" {
			return "s3cr3t-value", true
		}
		return "", false
	}

	// OpenCode under auto policy preserves {env:TEST_DB_SECRET}
	resOC, maskOC, warnOC := ResolveEnv(rawEnv, m, "opencode", PolicyAuto, mockLookup)
	if len(warnOC) != 0 {
		t.Errorf("expected no warnings for opencode, got %v", warnOC)
	}
	if resOC["DATABASE_PASSWORD"] != "{env:TEST_DB_SECRET}" {
		t.Errorf("expected opencode to preserve {env:...}, got %s", resOC["DATABASE_PASSWORD"])
	}
	if maskOC["DATABASE_PASSWORD"] != "{env:TEST_DB_SECRET}" {
		t.Errorf("unexpected masked value for opencode: %s", maskOC["DATABASE_PASSWORD"])
	}

	// Claude under auto policy translates to ${TEST_DB_SECRET}
	resClaude, maskClaude, warnClaude := ResolveEnv(rawEnv, m, "claude", PolicyAuto, mockLookup)
	if len(warnClaude) != 0 {
		t.Errorf("expected no warnings for claude, got %v", warnClaude)
	}
	if resClaude["DATABASE_PASSWORD"] != "${TEST_DB_SECRET}" || maskClaude["DATABASE_PASSWORD"] != "${TEST_DB_SECRET}" {
		t.Errorf("expected claude to use ${TEST_DB_SECRET}, got resolved=%s masked=%s", resClaude["DATABASE_PASSWORD"], maskClaude["DATABASE_PASSWORD"])
	}

	// Antigravity under auto policy resolves from lookupEnv and masks secret in dry-run
	resAG, maskAG, warnAG := ResolveEnv(rawEnv, m, "antigravity", PolicyAuto, mockLookup)
	if len(warnAG) != 0 {
		t.Errorf("expected no warnings when env exists, got %v", warnAG)
	}
	if resAG["DATABASE_PASSWORD"] != "s3cr3t-value" {
		t.Errorf("expected resolved secret for antigravity, got %s", resAG["DATABASE_PASSWORD"])
	}
	if maskAG["DATABASE_PASSWORD"] != "<resolved from $TEST_DB_SECRET>" {
		t.Errorf("expected masked placeholder, got %s", maskAG["DATABASE_PASSWORD"])
	}

	// Antigravity when env var is missing keeps placeholder and emits warning
	emptyLookup := func(string) (string, bool) { return "", false }
	resMissing, _, warnMissing := ResolveEnv(rawEnv, m, "antigravity", PolicyAuto, emptyLookup)
	if len(warnMissing) == 0 {
		t.Errorf("expected warning when env var is missing for antigravity")
	}
	if resMissing["DATABASE_PASSWORD"] != "{env:TEST_DB_SECRET}" {
		t.Errorf("expected fallback placeholder when env var missing, got %s", resMissing["DATABASE_PASSWORD"])
	}

	// Literal sensitive password should be masked in masked map
	literalEnv := map[string]string{
		"DATABASE_HOST":     "127.0.0.1",
		"DATABASE_PASSWORD": "plain-text-password",
	}
	resLit, maskLit, _ := ResolveEnv(literalEnv, m, "antigravity", PolicyAuto, emptyLookup)
	if resLit["DATABASE_PASSWORD"] != "plain-text-password" {
		t.Errorf("expected literal password in resolved map")
	}
	if maskLit["DATABASE_PASSWORD"] != "********" {
		t.Errorf("expected literal sensitive var to be masked as ********, got %s", maskLit["DATABASE_PASSWORD"])
	}
}

func TestValidateProfile(t *testing.T) {
	m := manifest.Manifest{
		ID: "mysql",
		Env: []manifest.EnvVar{
			{Name: "DATABASE_HOST", Required: true},
			{Name: "DATABASE_NAME", Required: true},
			{Name: "DATABASE_PERMISSIONS", Required: false},
		},
	}

	incomplete := Profile{
		MCP: "mysql",
		Env: map[string]string{
			"DATABASE_HOST": "127.0.0.1",
		},
	}
	errs := ValidateProfile("mysql-local", incomplete, m)
	if len(errs) != 1 || !strings.Contains(errs[0], "DATABASE_NAME") {
		t.Fatalf("expected missing DATABASE_NAME error, got %v", errs)
	}
}
