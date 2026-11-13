package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"edro-mcp/internal/manifest"
)

const (
	SchemaVersion1_0    = "1.0"
	ProfilesFileName    = "profiles.json"
	ExampleProfilesName = "profiles.example.json"
)

// ResolvePolicy determines how {env:VAR_NAME} placeholders are translated per client.
type ResolvePolicy string

const (
	PolicyAuto    ResolvePolicy = "auto"
	PolicyLiteral ResolvePolicy = "literal"
	PolicyKeep    ResolvePolicy = "keep"
)

// ParseResolvePolicy validates and normalizes a --resolve-env flag string.
func ParseResolvePolicy(raw string) (ResolvePolicy, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return PolicyAuto, nil
	case "literal":
		return PolicyLiteral, nil
	case "keep":
		return PolicyKeep, nil
	default:
		return "", fmt.Errorf("invalid resolve-env policy %q (valid: auto, literal, keep)", raw)
	}
}

// Profile defines a named connection instance of an MCP server.
type Profile struct {
	MCP         string            `json:"mcp"`
	Description string            `json:"description,omitempty"`
	Enabled     bool              `json:"enabled"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env"`
}

// Config represents the top-level structure of profiles.json.
type Config struct {
	SchemaVersion string             `json:"schemaVersion"`
	Profiles      map[string]Profile `json:"profiles"`
}

var envRefRegex = regexp.MustCompile(`^\{env:([A-Za-z_][A-Za-z0-9_]*)\}$`)

// HasUserProfiles checks whether either <repoRoot>/profiles.json or <homeDir>/.mcp/profiles.json exists.
func HasUserProfiles(repoRoot, homeDir string) bool {
	if repoRoot != "" {
		if _, err := os.Stat(filepath.Join(repoRoot, ProfilesFileName)); err == nil {
			return true
		}
	}
	if homeDir != "" {
		if _, err := os.Stat(filepath.Join(homeDir, ".mcp", ProfilesFileName)); err == nil {
			return true
		}
	}
	return false
}

// CopyExampleToLocal copies <repoRoot>/profiles.example.json to <repoRoot>/profiles.json if it does not exist.
func CopyExampleToLocal(repoRoot string) (string, error) {
	src := filepath.Join(repoRoot, ExampleProfilesName)
	dst := filepath.Join(repoRoot, ProfilesFileName)

	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", dst, err)
	}
	return dst, nil
}

// Load loads the profiles configuration according to precedence:
// 1. explicitPath (--config)
// 2. <repoRoot>/profiles.json
// 3. <homeDir>/.mcp/profiles.json
// 4. <repoRoot>/profiles.example.json
func Load(explicitPath, repoRoot, homeDir string) (*Config, string, error) {
	if strings.TrimSpace(explicitPath) != "" {
		cfg, err := LoadFile(explicitPath)
		return cfg, filepath.Clean(explicitPath), err
	}

	var candidates []string
	if repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, ProfilesFileName))
	}
	if homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".mcp", ProfilesFileName))
	}
	if repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, ExampleProfilesName))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			cfg, loadErr := LoadFile(candidate)
			return cfg, filepath.Clean(candidate), loadErr
		}
	}

	// Return an empty valid config if no file exists anywhere.
	return &Config{
		SchemaVersion: SchemaVersion1_0,
		Profiles:      make(map[string]Profile),
	}, "", nil
}

// LoadFile reads and validates a profiles JSON file from disk.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profiles file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse profiles file %s: %w", path, err)
	}

	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = SchemaVersion1_0
	}
	if cfg.SchemaVersion != SchemaVersion1_0 {
		return nil, fmt.Errorf("unsupported profiles schemaVersion %q in %s", cfg.SchemaVersion, path)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	for name, p := range cfg.Profiles {
		if strings.TrimSpace(p.MCP) == "" {
			return nil, fmt.Errorf("profile %q in %s is missing required field \"mcp\"", name, path)
		}
		if p.Env == nil {
			p.Env = make(map[string]string)
			cfg.Profiles[name] = p
		}
	}

	return &cfg, nil
}

// Save writes a Config to path with 0600 permissions.
func Save(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("cannot save nil profile config")
	}
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = SchemaVersion1_0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write profiles file %s: %w", path, err)
	}
	return nil
}

// SortedNames returns the profile keys sorted alphabetically.
func (c *Config) SortedNames() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SynthesizeDefault builds a default Profile from a Manifest's default environment variables.
func SynthesizeDefault(m manifest.Manifest) Profile {
	env := make(map[string]string, len(m.Env))
	for _, ev := range m.Env {
		env[ev.Name] = ev.Default
	}
	return Profile{
		MCP:         m.ID,
		Description: m.Description,
		Enabled:     true,
		Env:         env,
	}
}

// ConnectionSummary returns a concise non-sensitive summary such as "127.0.0.1:3306/app_dev".
func ConnectionSummary(p Profile) string {
	host := p.Env["DATABASE_HOST"]
	port := p.Env["DATABASE_PORT"]
	dbName := p.Env["DATABASE_NAME"]

	if host != "" {
		target := host
		if port != "" {
			target += ":" + port
		}
		if dbName != "" {
			target += "/" + dbName
		}
		return target
	}
	if p.Description != "" {
		return p.Description
	}
	return p.MCP
}

// ValidateProfile checks that all required manifest env vars exist in p.Env and are non-empty.
func ValidateProfile(profileName string, p Profile, m manifest.Manifest) []string {
	var errs []string
	for _, ev := range m.Env {
		if !ev.Required {
			continue
		}
		val, ok := p.Env[ev.Name]
		if !ok || strings.TrimSpace(val) == "" {
			errs = append(errs, fmt.Sprintf("el perfil %q (mcp: %s) requiere la variable obligatoria %s", profileName, m.ID, ev.Name))
		}
	}
	return errs
}

// ExtractEnvRef returns (varName, true) if val matches "{env:VAR_NAME}".
func ExtractEnvRef(val string) (string, bool) {
	matches := envRefRegex.FindStringSubmatch(strings.TrimSpace(val))
	if len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

// ResolveEnv translates profile environment variables for a specific target client and policy.
// It returns:
// - resolved: map to write into the actual client config file
// - masked: map safe for console / --dry-run display (never exposes resolved or sensitive secrets)
// - warnings: Spanish warnings if an env var needed expansion from the host environment but was not set
func ResolveEnv(
	rawEnv map[string]string,
	m manifest.Manifest,
	clientID string,
	policy ResolvePolicy,
	lookupEnv func(string) (string, bool),
) (map[string]string, map[string]string, []string) {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if policy == "" {
		policy = PolicyAuto
	}

	sensitiveVars := make(map[string]bool, len(m.Env))
	for _, ev := range m.Env {
		if ev.Sensitive {
			sensitiveVars[ev.Name] = true
		}
	}

	resolved := make(map[string]string, len(rawEnv))
	masked := make(map[string]string, len(rawEnv))
	var warnings []string

	keys := make([]string, 0, len(rawEnv))
	for k := range rawEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	clientLower := strings.ToLower(strings.TrimSpace(clientID))

	for _, k := range keys {
		rawVal := rawEnv[k]
		varName, isRef := ExtractEnvRef(rawVal)
		if !isRef {
			resolved[k] = rawVal
			if sensitiveVars[k] && rawVal != "" {
				masked[k] = "********"
			} else {
				masked[k] = rawVal
			}
			continue
		}

		switch policy {
		case PolicyKeep:
			if clientLower == "claude" {
				resolved[k] = fmt.Sprintf("${%s}", varName)
				masked[k] = fmt.Sprintf("${%s}", varName)
			} else {
				resolved[k] = rawVal
				masked[k] = rawVal
			}

		case PolicyLiteral:
			if envVal, ok := lookupEnv(varName); ok && envVal != "" {
				resolved[k] = envVal
				masked[k] = fmt.Sprintf("<resolved from $%s>", varName)
			} else {
				resolved[k] = rawVal
				masked[k] = rawVal
				warnings = append(warnings, fmt.Sprintf("variable de entorno %s no definida en el sistema para %s (se conservó %s)", varName, clientID, rawVal))
			}

		default: // PolicyAuto
			switch clientLower {
			case "opencode":
				resolved[k] = rawVal
				masked[k] = rawVal
			case "claude":
				resolved[k] = fmt.Sprintf("${%s}", varName)
				masked[k] = fmt.Sprintf("${%s}", varName)
			default: // antigravity, codex, cursor
				if envVal, ok := lookupEnv(varName); ok && envVal != "" {
					resolved[k] = envVal
					masked[k] = fmt.Sprintf("<resolved from $%s>", varName)
				} else {
					resolved[k] = rawVal
					masked[k] = rawVal
					warnings = append(warnings, fmt.Sprintf("variable de entorno %s no definida en el sistema para el cliente %s; exporta %s antes de sincronizar o actualiza la configuración", varName, clientID, varName))
				}
			}
		}
	}

	return resolved, masked, warnings
}
