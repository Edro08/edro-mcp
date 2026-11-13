package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"edro-mcp/internal/manifest"
	"edro-mcp/internal/profile"
)

// Entry represents a single resolved MCP profile ready to be injected into a client config.
type Entry struct {
	ProfileName string            `json:"profileName"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	MaskedEnv   map[string]string `json:"maskedEnv,omitempty"`
}

// SyncResult describes the result of synchronizing one or more entries into a client configuration file.
type SyncResult struct {
	ClientID   string   `json:"clientId"`
	ClientName string   `json:"clientName"`
	ConfigPath string   `json:"configPath"`
	DisplayRel string   `json:"displayRel"`
	BackupPath string   `json:"backupPath,omitempty"`
	Profiles   []string `json:"profiles"`
	Created    bool     `json:"created"`
	DryRun     bool     `json:"dryRun"`
	DryRunDiff string   `json:"dryRunDiff,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// Spec defines the metadata and file paths for a supported AI client.
type Spec struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Format        string   `json:"format"`
	RelConfigPath string   `json:"relConfigPath"`
	DisplayPath   string   `json:"displayPath"`
	DetectPaths   []string `json:"-"`
}

// ClientStatus reports whether a client is detected on the user's machine.
type ClientStatus struct {
	Spec       Spec   `json:"spec"`
	ConfigPath string `json:"configPath"`
	Detected   bool   `json:"detected"`
	FileExists bool   `json:"fileExists"`
}

var registry = []Spec{
	{
		ID:            "antigravity",
		Name:          "Antigravity",
		Format:        "json",
		RelConfigPath: filepath.FromSlash(".gemini/config/mcp_config.json"),
		DisplayPath:   "~/.gemini/config/mcp_config.json",
		DetectPaths:   []string{".gemini/config/mcp_config.json", ".gemini"},
	},
	{
		ID:            "opencode",
		Name:          "OpenCode",
		Format:        "jsonc",
		RelConfigPath: filepath.FromSlash(".config/opencode/opencode.jsonc"),
		DisplayPath:   "~/.config/opencode/opencode.jsonc",
		DetectPaths:   []string{".config/opencode/opencode.jsonc", ".config/opencode"},
	},
	{
		ID:            "codex",
		Name:          "Codex",
		Format:        "toml",
		RelConfigPath: filepath.FromSlash(".codex/config.toml"),
		DisplayPath:   "~/.codex/config.toml",
		DetectPaths:   []string{".codex/config.toml", ".codex"},
	},
	{
		ID:            "claude",
		Name:          "Claude Code",
		Format:        "json",
		RelConfigPath: filepath.FromSlash(".claude.json"),
		DisplayPath:   "~/.claude.json",
		DetectPaths:   []string{".claude.json", ".claude"},
	},
	{
		ID:            "cursor",
		Name:          "Cursor",
		Format:        "json",
		RelConfigPath: filepath.FromSlash(".cursor/mcp.json"),
		DisplayPath:   "~/.cursor/mcp.json",
		DetectPaths:   []string{".cursor/mcp.json", ".cursor"},
	},
}

// AllSpecs returns all supported client specifications in canonical order.
func AllSpecs() []Spec {
	out := make([]Spec, len(registry))
	copy(out, registry)
	return out
}

// GetSpec looks up a client specification by ID (case-insensitive, supports "claude-code" alias).
func GetSpec(clientID string) (Spec, error) {
	id := strings.ToLower(strings.TrimSpace(clientID))
	if id == "claude-code" {
		id = "claude"
	}
	for _, s := range registry {
		if s.ID == id {
			return s, nil
		}
	}
	return Spec{}, fmt.Errorf("unsupported client %q (valid: antigravity, opencode, codex, claude, cursor)", clientID)
}

// ConfigPath returns the full OS path to the client's global configuration file inside homeDir.
func (s Spec) ConfigPath(homeDir string) string {
	return filepath.Join(filepath.Clean(homeDir), s.RelConfigPath)
}

// DetectAll checks which clients are installed/configured in homeDir.
func DetectAll(homeDir string) []ClientStatus {
	cleanHome := filepath.Clean(homeDir)
	statuses := make([]ClientStatus, 0, len(registry))

	for _, s := range registry {
		cfgPath := s.ConfigPath(cleanHome)
		_, fileErr := os.Stat(cfgPath)
		fileExists := fileErr == nil

		detected := fileExists
		if !detected {
			for _, rel := range s.DetectPaths {
				p := filepath.Join(cleanHome, filepath.FromSlash(rel))
				if _, err := os.Stat(p); err == nil {
					detected = true
					break
				}
			}
		}

		statuses = append(statuses, ClientStatus{
			Spec:       s,
			ConfigPath: cfgPath,
			Detected:   detected,
			FileExists: fileExists,
		})
	}
	return statuses
}

// BuildEntriesForClient resolves commands, arguments, and environment variables for the requested profiles.
func BuildEntriesForClient(
	profileNames []string,
	cfg *profile.Config,
	manifests []manifest.Manifest,
	binDir string,
	clientID string,
	policy profile.ResolvePolicy,
	lookupEnv func(string) (string, bool),
) ([]Entry, []string, error) {
	var entries []Entry
	var allWarnings []string

	for _, profName := range profileNames {
		p, ok := cfg.Profiles[profName]
		if !ok {
			return nil, nil, fmt.Errorf("profile %q not found", profName)
		}
		m, found := manifest.FindByID(manifests, p.MCP)
		if !found {
			return nil, nil, fmt.Errorf("manifest %q referenced by profile %q not found", p.MCP, profName)
		}

		cmdStr, baseArgs := m.ResolveCommand(binDir, runtime.GOOS)
		args := make([]string, 0, len(baseArgs)+len(p.Args))
		args = append(args, baseArgs...)
		args = append(args, p.Args...)

		resolvedEnv, maskedEnv, warnings := profile.ResolveEnv(p.Env, m, clientID, policy, lookupEnv)
		allWarnings = append(allWarnings, warnings...)

		entries = append(entries, Entry{
			ProfileName: profName,
			Command:     cmdStr,
			Args:        args,
			Env:         resolvedEnv,
			MaskedEnv:   maskedEnv,
		})
	}

	return entries, allWarnings, nil
}

// SyncClient merges the provided entries into the target client's global configuration file.
// When dryRun is false and the file already exists, a .bak backup is created before atomic replacement.
func SyncClient(homeDir string, clientID string, entries []Entry, dryRun bool) (SyncResult, error) {
	spec, err := GetSpec(clientID)
	if err != nil {
		return SyncResult{}, err
	}

	cfgPath := spec.ConfigPath(homeDir)
	existingBytes, readErr := os.ReadFile(cfgPath)
	fileExisted := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return SyncResult{}, fmt.Errorf("read %s config %s: %w", spec.ID, cfgPath, readErr)
	}

	profileNames := make([]string, len(entries))
	for i, e := range entries {
		profileNames[i] = e.ProfileName
	}

	// Render masked preview for dryRun (and validation)
	maskedBytes, err := renderMergedConfig(spec.ID, existingBytes, entries, true)
	if err != nil {
		return SyncResult{}, fmt.Errorf("merge %s config (%s): %w", spec.ID, cfgPath, err)
	}

	res := SyncResult{
		ClientID:   spec.ID,
		ClientName: spec.Name,
		ConfigPath: cfgPath,
		DisplayRel: spec.DisplayPath,
		Profiles:   profileNames,
		Created:    !fileExisted,
		DryRun:     dryRun,
	}

	if dryRun {
		res.DryRunDiff = string(maskedBytes)
		return res, nil
	}

	finalBytes, err := renderMergedConfig(spec.ID, existingBytes, entries, false)
	if err != nil {
		return SyncResult{}, fmt.Errorf("merge %s config (%s): %w", spec.ID, cfgPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return SyncResult{}, fmt.Errorf("create parent directory for %s: %w", cfgPath, err)
	}

	if fileExisted {
		bakPath := cfgPath + ".bak"
		if err := os.WriteFile(bakPath, existingBytes, 0o600); err != nil {
			return SyncResult{}, fmt.Errorf("write backup file %s: %w", bakPath, err)
		}
		res.BackupPath = bakPath
	}

	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, finalBytes, 0o600); err != nil {
		return SyncResult{}, fmt.Errorf("write temporary config %s: %w", tmpPath, err)
	}
	if err := replaceFileAtomically(tmpPath, cfgPath); err != nil {
		_ = os.Remove(tmpPath)
		return SyncResult{}, fmt.Errorf("commit config %s: %w", cfgPath, err)
	}

	return res, nil
}

func renderMergedConfig(clientID string, existing []byte, entries []Entry, useMasked bool) ([]byte, error) {
	switch clientID {
	case "antigravity", "cursor":
		return mergeStandardJSON(existing, entries, false, useMasked)
	case "claude":
		return mergeStandardJSON(existing, entries, true, useMasked)
	case "opencode":
		return mergeOpenCodeJSONC(existing, entries, useMasked)
	case "codex":
		return mergeCodexTOML(existing, entries, useMasked)
	default:
		return nil, fmt.Errorf("unsupported client id %q", clientID)
	}
}

func mergeStandardJSON(existing []byte, entries []Entry, includeStdioType bool, useMasked bool) ([]byte, error) {
	root := make(map[string]any)
	if len(strings.TrimSpace(string(existing))) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("invalid json syntax: %w", err)
		}
	}

	mcpServers := ensureMap(root, "mcpServers")
	for _, entry := range entries {
		envMap := entry.Env
		if useMasked && entry.MaskedEnv != nil {
			envMap = entry.MaskedEnv
		}
		args := entry.Args
		if args == nil {
			args = []string{}
		}
		serverObj := map[string]any{
			"command": entry.Command,
			"args":    args,
			"env":     envMap,
		}
		if includeStdioType {
			serverObj["type"] = "stdio"
		}
		mcpServers[entry.ProfileName] = serverObj
	}

	out, err := marshalJSONNoEscape(root)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return out, nil
}

func mergeOpenCodeJSONC(existing []byte, entries []Entry, useMasked bool) ([]byte, error) {
	root, header, err := parseJSONCObject(existing)
	if err != nil {
		return nil, err
	}
	if _, hasSchema := root["$schema"]; !hasSchema && len(root) == 0 {
		root["$schema"] = "https://opencode.ai/config.json"
	}

	mcpObj := ensureMap(root, "mcp")
	serversObj := ensureMap(mcpObj, "servers")

	for _, entry := range entries {
		envMap := entry.Env
		if useMasked && entry.MaskedEnv != nil {
			envMap = entry.MaskedEnv
		}
		cmdSlice := make([]string, 0, 1+len(entry.Args))
		cmdSlice = append(cmdSlice, entry.Command)
		cmdSlice = append(cmdSlice, entry.Args...)

		serversObj[entry.ProfileName] = map[string]any{
			"type":        "local",
			"command":     cmdSlice,
			"environment": envMap,
		}
	}

	out, err := marshalJSONNoEscape(root)
	if err != nil {
		return nil, fmt.Errorf("marshal opencode jsonc: %w", err)
	}
	if header != "" {
		return []byte(header + string(out)), nil
	}
	return out, nil
}

func marshalJSONNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// InspectClientConfig validates the syntax of a client's configuration file and returns the list
// of MCP server profile names currently registered in it.
func InspectClientConfig(homeDir string, clientID string) (bool, []string, error) {
	spec, err := GetSpec(clientID)
	if err != nil {
		return false, nil, err
	}
	cfgPath := spec.ConfigPath(homeDir)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}

	switch spec.ID {
	case "antigravity", "claude", "cursor":
		var root map[string]any
		if err := json.Unmarshal(data, &root); err != nil {
			return true, nil, fmt.Errorf("invalid json in %s: %w", cfgPath, err)
		}
		servers, ok := root["mcpServers"].(map[string]any)
		if !ok {
			return true, nil, nil
		}
		return true, sortedMapKeys(servers), nil

	case "opencode":
		root, _, err := parseJSONCObject(data)
		if err != nil {
			return true, nil, fmt.Errorf("invalid jsonc in %s: %w", cfgPath, err)
		}
		mcpMap, ok := root["mcp"].(map[string]any)
		if !ok {
			return true, nil, nil
		}
		servers, ok := mcpMap["servers"].(map[string]any)
		if !ok {
			return true, nil, nil
		}
		return true, sortedMapKeys(servers), nil

	case "codex":
		profiles, err := extractTOMLMCPServers(data)
		if err != nil {
			return true, nil, fmt.Errorf("invalid toml in %s: %w", cfgPath, err)
		}
		return true, profiles, nil

	default:
		return false, nil, fmt.Errorf("unsupported client %q", spec.ID)
	}
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok && existing != nil {
		return existing
	}
	created := make(map[string]any)
	parent[key] = created
	return created
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func replaceFileAtomically(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	_ = os.Remove(dst)
	return os.Rename(src, dst)
}
