package ui

import (
	"bytes"
	"strings"
	"testing"

	"edro-mcp/internal/client"
	"edro-mcp/internal/manifest"
	"edro-mcp/internal/profile"
)

func TestPromptSelectClientsAndProfiles(t *testing.T) {
	statuses := []client.ClientStatus{
		{
			Spec:     client.Spec{ID: "antigravity", Name: "Antigravity", DisplayPath: "~/.gemini/config/mcp_config.json"},
			Detected: true,
		},
		{
			Spec:     client.Spec{ID: "opencode", Name: "OpenCode", DisplayPath: "~/.config/opencode/opencode.jsonc"},
			Detected: false,
		},
	}

	// 1. Pressing Enter selects pre-detected clients
	var out bytes.Buffer
	selected, err := PromptSelectClients(strings.NewReader("\n"), &out, statuses)
	if err != nil {
		t.Fatalf("PromptSelectClients default failed: %v", err)
	}
	if strings.Join(selected, ",") != "antigravity" {
		t.Errorf("expected default antigravity, got %v", selected)
	}

	// 2. Selecting by number "1,2"
	out.Reset()
	selected, err = PromptSelectClients(strings.NewReader("1,2\n"), &out, statuses)
	if err != nil {
		t.Fatalf("PromptSelectClients 1,2 failed: %v", err)
	}
	if strings.Join(selected, ",") != "antigravity,opencode" {
		t.Errorf("expected antigravity,opencode, got %v", selected)
	}

	// 3. Selecting profiles by number
	cfg := &profile.Config{
		SchemaVersion: "1.0",
		Profiles: map[string]profile.Profile{
			"mysql-local":    {MCP: "mysql", Enabled: true, Env: map[string]string{"DATABASE_HOST": "127.0.0.1"}},
			"postgres-local": {MCP: "postgresql", Enabled: true, Env: map[string]string{"DATABASE_HOST": "127.0.0.1"}},
		},
	}
	manifests := []manifest.Manifest{
		{ID: "mysql"},
		{ID: "postgresql"},
	}

	out.Reset()
	profSelected, err := PromptSelectProfiles(strings.NewReader("2\n"), &out, cfg, manifests)
	if err != nil {
		t.Fatalf("PromptSelectProfiles failed: %v", err)
	}
	if strings.Join(profSelected, ",") != "postgres-local" {
		t.Errorf("expected postgres-local, got %v", profSelected)
	}
}
