package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"edro-mcp/internal/manifest"
)

// BuildResult describes the outcome of building a single MCP server.
type BuildResult struct {
	ManifestID string `json:"manifestId"`
	BinaryPath string `json:"binaryPath"`
	Command    string `json:"command"`
	DryRun     bool   `json:"dryRun"`
}

// DefaultBinDir returns the default binary directory (~/.mcp/bin) using homeDir.
func DefaultBinDir(homeDir string) string {
	return filepath.Join(filepath.Clean(homeDir), ".mcp", "bin")
}

// BuildMany compiles and installs a slice of manifests into binDir, deduplicating by Manifest.ID.
func BuildMany(ctx context.Context, manifests []manifest.Manifest, binDir string, dryRun bool) ([]BuildResult, error) {
	seen := make(map[string]bool, len(manifests))
	var results []BuildResult

	for _, m := range manifests {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true

		res, err := Build(ctx, m, binDir, dryRun)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

// Build compiles or copies the artifact for a single Manifest into binDir.
func Build(ctx context.Context, m manifest.Manifest, binDir string, dryRun bool) (BuildResult, error) {
	if err := m.Validate(); err != nil {
		return BuildResult{}, fmt.Errorf("invalid manifest %q: %w", m.ID, err)
	}

	cleanBinDir := filepath.Clean(binDir)
	outputBinary := m.BinaryPath(cleanBinDir, runtime.GOOS)

	switch m.Build.Type {
	case "go":
		source := strings.TrimSpace(m.Build.Source)
		if source == "" {
			source = "."
		}
		cmdStr := fmt.Sprintf("go build -o %s %s", outputBinary, source)
		res := BuildResult{
			ManifestID: m.ID,
			BinaryPath: outputBinary,
			Command:    cmdStr,
			DryRun:     dryRun,
		}
		if dryRun {
			return res, nil
		}

		if err := os.MkdirAll(cleanBinDir, 0o755); err != nil {
			return res, fmt.Errorf("create bin directory %s: %w", cleanBinDir, err)
		}

		tmpOutput := outputBinary + ".tmp"
		_ = os.Remove(tmpOutput)

		cmd := exec.CommandContext(ctx, "go", "build", "-o", tmpOutput, source)
		cmd.Dir = m.Dir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			_ = os.Remove(tmpOutput)
			errMsg := strings.TrimSpace(stderr.String())
			if errMsg != "" {
				return res, fmt.Errorf("build go mcp %q: %w (%s)", m.ID, err, errMsg)
			}
			return res, fmt.Errorf("build go mcp %q: %w", m.ID, err)
		}

		if err := os.Chmod(tmpOutput, 0o755); err != nil {
			_ = os.Remove(tmpOutput)
			return res, fmt.Errorf("chmod binary %s: %w", tmpOutput, err)
		}
		if err := atomicReplace(tmpOutput, outputBinary); err != nil {
			_ = os.Remove(tmpOutput)
			return res, fmt.Errorf("install binary %s: %w", outputBinary, err)
		}
		return res, nil

	case "command":
		cmdStr := strings.Join(m.Build.Command, " ")
		res := BuildResult{
			ManifestID: m.ID,
			BinaryPath: outputBinary,
			Command:    cmdStr,
			DryRun:     dryRun,
		}
		if dryRun {
			return res, nil
		}

		if err := os.MkdirAll(cleanBinDir, 0o755); err != nil {
			return res, fmt.Errorf("create bin directory %s: %w", cleanBinDir, err)
		}

		cmd := exec.CommandContext(ctx, m.Build.Command[0], m.Build.Command[1:]...)
		cmd.Dir = m.Dir
		cmd.Env = append(os.Environ(),
			"EDRO_MCP_BIN_DIR="+cleanBinDir,
			"EDRO_MCP_OUTPUT="+outputBinary,
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			errMsg := strings.TrimSpace(stderr.String())
			if errMsg != "" {
				return res, fmt.Errorf("run build command for %q: %w (%s)", m.ID, err, errMsg)
			}
			return res, fmt.Errorf("run build command for %q: %w", m.ID, err)
		}

		// If the command produced a file in m.Dir matching Build.Source or Build.Binary, copy it to outputBinary.
		if _, err := os.Stat(outputBinary); os.IsNotExist(err) {
			candidate := m.Build.Source
			if candidate == "" {
				candidate = m.Build.Binary
			}
			if candidate != "" {
				srcPath := filepath.Join(m.Dir, candidate)
				if _, statErr := os.Stat(srcPath); statErr == nil {
					if copyErr := copyExecutable(srcPath, outputBinary); copyErr != nil {
						return res, copyErr
					}
				}
			}
		}
		return res, nil

	case "copy":
		srcPath := filepath.Join(m.Dir, m.Build.Source)
		res := BuildResult{
			ManifestID: m.ID,
			BinaryPath: outputBinary,
			Command:    fmt.Sprintf("copy %s -> %s", srcPath, outputBinary),
			DryRun:     dryRun,
		}
		if dryRun {
			return res, nil
		}
		if err := os.MkdirAll(cleanBinDir, 0o755); err != nil {
			return res, fmt.Errorf("create bin directory %s: %w", cleanBinDir, err)
		}
		if err := copyExecutable(srcPath, outputBinary); err != nil {
			return res, err
		}
		return res, nil

	default:
		return BuildResult{}, fmt.Errorf("unsupported build type %q", m.Build.Type)
	}
}

func copyExecutable(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source artifact %s: %w", srcPath, err)
	}
	tmpPath := dstPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o755); err != nil {
		return fmt.Errorf("write temporary artifact %s: %w", tmpPath, err)
	}
	if err := atomicReplace(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("install artifact %s: %w", dstPath, err)
	}
	return nil
}

func atomicReplace(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// On Windows, os.Rename may fail if dst already exists and has certain attributes.
	_ = os.Remove(dst)
	return os.Rename(src, dst)
}
