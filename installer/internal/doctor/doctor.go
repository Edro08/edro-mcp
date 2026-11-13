package doctor

import (
	"fmt"
	"os"
	"runtime"

	"edro-mcp/internal/client"
	"edro-mcp/internal/manifest"
	"edro-mcp/internal/profile"
)

// BinaryCheck holds the health status of a single MCP server binary inside binDir.
type BinaryCheck struct {
	ManifestID string `json:"manifestId"`
	BinaryPath string `json:"binaryPath"`
	Exists     bool   `json:"exists"`
	Executable bool   `json:"executable"`
	Issue      string `json:"issue,omitempty"`
}

// ProfileCheck holds the validation result of a single profile in profiles.json.
type ProfileCheck struct {
	Name    string   `json:"name"`
	MCP     string   `json:"mcp"`
	Enabled bool     `json:"enabled"`
	Valid   bool     `json:"valid"`
	Issues  []string `json:"issues,omitempty"`
}

// ClientCheck holds the syntax and detection check for a single AI client configuration.
type ClientCheck struct {
	ClientID   string   `json:"clientId"`
	ClientName string   `json:"clientName"`
	ConfigPath string   `json:"configPath"`
	DisplayRel string   `json:"displayRel"`
	Detected   bool     `json:"detected"`
	FileExists bool     `json:"fileExists"`
	Valid      bool     `json:"valid"`
	Profiles   []string `json:"profiles,omitempty"`
	Issue      string   `json:"issue,omitempty"`
}

// Report aggregates all health checks for edro-mcp doctor / status.
type Report struct {
	Healthy      bool           `json:"healthy"`
	BinDir       string         `json:"binDir"`
	ProfilesPath string         `json:"profilesPath"`
	Binaries     []BinaryCheck  `json:"binaries"`
	Profiles     []ProfileCheck `json:"profiles"`
	Clients      []ClientCheck  `json:"clients"`
	Issues       []string       `json:"issues,omitempty"`
}

// Run executes all health and configuration checks across binaries, profiles, and detected AI clients.
func Run(manifests []manifest.Manifest, cfg *profile.Config, profilesPath, binDir, homeDir string) Report {
	report := Report{
		Healthy:      true,
		BinDir:       binDir,
		ProfilesPath: profilesPath,
	}

	// 1. Check binaries in binDir
	for _, m := range manifests {
		binPath := m.BinaryPath(binDir, runtime.GOOS)
		info, err := os.Stat(binPath)
		bc := BinaryCheck{
			ManifestID: m.ID,
			BinaryPath: binPath,
		}
		if err != nil {
			bc.Exists = false
			bc.Executable = false
			bc.Issue = fmt.Sprintf("binario no encontrado para %q en %s (ejecuta 'edro-mcp build')", m.ID, binPath)
			report.Healthy = false
			report.Issues = append(report.Issues, bc.Issue)
		} else {
			bc.Exists = true
			if info.IsDir() {
				bc.Executable = false
				bc.Issue = fmt.Sprintf("la ruta del binario %s es un directorio", binPath)
				report.Healthy = false
				report.Issues = append(report.Issues, bc.Issue)
			} else if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
				bc.Executable = false
				bc.Issue = fmt.Sprintf("el binario %s no tiene permisos de ejecución", binPath)
				report.Healthy = false
				report.Issues = append(report.Issues, bc.Issue)
			} else {
				bc.Executable = true
			}
		}
		report.Binaries = append(report.Binaries, bc)
	}

	// 2. Check profiles
	if cfg != nil {
		for _, name := range cfg.SortedNames() {
			p := cfg.Profiles[name]
			pc := ProfileCheck{
				Name:    name,
				MCP:     p.MCP,
				Enabled: p.Enabled,
				Valid:   true,
			}
			m, found := manifest.FindByID(manifests, p.MCP)
			if !found {
				issue := fmt.Sprintf("el perfil %q referencia un MCP desconocido %q", name, p.MCP)
				pc.Valid = false
				pc.Issues = append(pc.Issues, issue)
				report.Healthy = false
				report.Issues = append(report.Issues, issue)
			} else {
				errs := profile.ValidateProfile(name, p, m)
				if len(errs) > 0 {
					pc.Valid = false
					pc.Issues = append(pc.Issues, errs...)
					report.Healthy = false
					report.Issues = append(report.Issues, errs...)
				}
			}
			report.Profiles = append(report.Profiles, pc)
		}
	}

	// 3. Check client configurations
	statuses := client.DetectAll(homeDir)
	for _, st := range statuses {
		cc := ClientCheck{
			ClientID:   st.Spec.ID,
			ClientName: st.Spec.Name,
			ConfigPath: st.ConfigPath,
			DisplayRel: st.Spec.DisplayPath,
			Detected:   st.Detected,
			FileExists: st.FileExists,
			Valid:      true,
		}
		if st.FileExists {
			_, profs, err := client.InspectClientConfig(homeDir, st.Spec.ID)
			if err != nil {
				cc.Valid = false
				cc.Issue = fmt.Sprintf("error de sintaxis en %s (%s): %v", st.Spec.Name, st.Spec.DisplayPath, err)
				report.Healthy = false
				report.Issues = append(report.Issues, cc.Issue)
			} else {
				cc.Profiles = profs
			}
		}
		report.Clients = append(report.Clients, cc)
	}

	return report
}
