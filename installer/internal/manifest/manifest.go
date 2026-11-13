package manifest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	ManifestFileName = "mcp.manifest.json"
	SchemaVersion1_0 = "1.0"
)

// BuildSpec defines how an MCP server binary or artifact is built.
type BuildSpec struct {
	Type    string   `json:"type"`
	Source  string   `json:"source,omitempty"`
	Binary  string   `json:"binary"`
	Command []string `json:"command,omitempty"`
}

// RunSpec defines how an installed MCP server is launched by AI clients.
type RunSpec struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// EnvVar describes an environment variable expected or supported by the MCP server.
type EnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
	Sensitive   bool   `json:"sensitive"`
}

// Manifest represents a parsed mcp.manifest.json contract.
type Manifest struct {
	SchemaVersion string    `json:"schemaVersion"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Runtime       string    `json:"runtime"`
	Build         BuildSpec `json:"build"`
	Run           RunSpec   `json:"run"`
	Env           []EnvVar  `json:"env"`

	// Dir is the directory containing mcp.manifest.json (populated at load time).
	Dir string `json:"-"`
	// ManifestPath is the full path to mcp.manifest.json (populated at load time).
	ManifestPath string `json:"-"`
}

var skippedDirs = map[string]bool{
	".git":         true,
	"agents":       true,
	"internal":     true,
	"cmd":          true,
	"node_modules": true,
	"vendor":       true,
	"bin":          true,
	"dist":         true,
}

// Discover walks rootDir recursively to find and validate all mcp.manifest.json files.
func Discover(rootDir string) ([]Manifest, error) {
	cleanRoot := filepath.Clean(rootDir)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return nil, fmt.Errorf("stat root directory %s: %w", cleanRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path %s is not a directory", cleanRoot)
	}

	var manifests []Manifest
	seenIDs := make(map[string]string)

	err = filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}

		if d.IsDir() {
			name := d.Name()
			if path != cleanRoot && (strings.HasPrefix(name, ".") || skippedDirs[name]) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() != ManifestFileName {
			return nil
		}

		m, err := LoadFile(path)
		if err != nil {
			return err
		}

		if prevPath, exists := seenIDs[m.ID]; exists {
			return fmt.Errorf("duplicate manifest id %q found in %s and %s", m.ID, prevPath, path)
		}
		seenIDs[m.ID] = path
		manifests = append(manifests, m)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].ID < manifests[j].ID
	})

	return manifests, nil
}

// LoadFile reads, parses, and validates a single mcp.manifest.json file.
func LoadFile(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	m.ManifestPath = filepath.Clean(path)
	m.Dir = filepath.Dir(m.ManifestPath)

	if err := m.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("validate manifest %s: %w", path, err)
	}

	return m, nil
}

// Validate checks required fields and schema constraints on a Manifest.
func (m Manifest) Validate() error {
	if m.SchemaVersion != SchemaVersion1_0 {
		return fmt.Errorf("unsupported schemaVersion %q (expected %q)", m.SchemaVersion, SchemaVersion1_0)
	}
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("manifest id is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("manifest name is required")
	}
	if strings.TrimSpace(m.Runtime) == "" {
		return fmt.Errorf("manifest runtime is required")
	}
	if strings.TrimSpace(m.Build.Type) == "" {
		return fmt.Errorf("manifest build.type is required")
	}
	switch m.Build.Type {
	case "go":
		if strings.TrimSpace(m.Build.Binary) == "" {
			return fmt.Errorf("manifest build.binary is required for go build")
		}
	case "command":
		if len(m.Build.Command) == 0 {
			return fmt.Errorf("manifest build.command is required for command build")
		}
	case "copy":
		if strings.TrimSpace(m.Build.Source) == "" || strings.TrimSpace(m.Build.Binary) == "" {
			return fmt.Errorf("manifest build.source and build.binary are required for copy build")
		}
	default:
		return fmt.Errorf("unsupported build.type %q", m.Build.Type)
	}
	if strings.TrimSpace(m.Run.Command) == "" {
		return fmt.Errorf("manifest run.command is required")
	}
	return nil
}

// BinaryPath returns the target path inside binDir for the compiled binary.
func (m Manifest) BinaryPath(binDir string, goos string) string {
	if goos == "" {
		goos = runtime.GOOS
	}
	ext := ExeExt(goos)
	binName := m.Build.Binary
	if ext != "" && !strings.HasSuffix(strings.ToLower(binName), ext) {
		binName += ext
	}
	return filepath.Join(filepath.Clean(binDir), binName)
}

// ResolveCommand interpolates {bin_dir}, {exe_ext}, {script_ext}, and {mcp_dir} in Run.Command and Run.Args.
func (m Manifest) ResolveCommand(binDir string, goos string) (string, []string) {
	if goos == "" {
		goos = runtime.GOOS
	}
	cleanBin := filepath.Clean(binDir)
	exeExt := ExeExt(goos)
	scriptExt := ""
	if goos == "windows" {
		scriptExt = ".cmd"
	}

	replaceTokens := func(s string) string {
		out := strings.ReplaceAll(s, "{bin_dir}", cleanBin)
		out = strings.ReplaceAll(out, "{exe_ext}", exeExt)
		out = strings.ReplaceAll(out, "{script_ext}", scriptExt)
		out = strings.ReplaceAll(out, "{mcp_dir}", m.Dir)
		return out
	}

	cmdStr := replaceTokens(m.Run.Command)
	// Normalize path separators when command points into binDir or mcpDir.
	if strings.Contains(m.Run.Command, "{bin_dir}") || strings.Contains(m.Run.Command, "{mcp_dir}") {
		cmdStr = filepath.FromSlash(filepath.ToSlash(cmdStr))
		if goos == "windows" {
			cmdStr = strings.ReplaceAll(cmdStr, "/", `\`)
		} else {
			cmdStr = strings.ReplaceAll(cmdStr, `\`, "/")
		}
	}

	args := make([]string, len(m.Run.Args))
	for i, arg := range m.Run.Args {
		args[i] = replaceTokens(arg)
	}

	return cmdStr, args
}

// FindByID searches a slice of Manifests by ID.
func FindByID(manifests []Manifest, id string) (Manifest, bool) {
	for _, m := range manifests {
		if m.ID == id {
			return m, true
		}
	}
	return Manifest{}, false
}

// ExeExt returns ".exe" on Windows and "" on POSIX operating systems.
func ExeExt(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}
