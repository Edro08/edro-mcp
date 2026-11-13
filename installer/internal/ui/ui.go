package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"edro-mcp/internal/builder"
	"edro-mcp/internal/client"
	"edro-mcp/internal/doctor"
	"edro-mcp/internal/manifest"
	"edro-mcp/internal/profile"
)

// PrintBanner renders the CLI welcome box and detected metadata in Spanish.
func PrintBanner(out io.Writer, manifests []manifest.Manifest, profilesSource string, profileCount int) {
	ids := make([]string, len(manifests))
	for i, m := range manifests {
		ids[i] = m.ID
	}
	srcDisplay := profilesSource
	if srcDisplay == "" {
		srcDisplay = "(ninguno, se usarán plantillas)"
	}

	fmt.Fprintln(out, "╭──────────────────────────────────────────────────────────────╮")
	fmt.Fprintln(out, "│  edro-mcp · Instalador Multi-Agente y Gestor de Perfiles     │")
	fmt.Fprintln(out, "╰──────────────────────────────────────────────────────────────╯")
	fmt.Fprintf(out, "• Manifiestos detectados: %d (%s)\n", len(manifests), strings.Join(ids, ", "))
	fmt.Fprintf(out, "• Archivo de perfiles:   %s (%d perfiles cargados)\n\n", srcDisplay, profileCount)
}

// PromptInitProfiles asks the user whether to initialize ./profiles.json from ./profiles.example.json.
func PromptInitProfiles(in io.Reader, out io.Writer) (bool, error) {
	reader := bufio.NewReader(in)
	fmt.Fprint(out, "? No se encontró profiles.json local. ¿Deseas crearlo a partir de profiles.example.json? [S/n]: ")
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("read user input: %w", err)
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	if ans == "" || ans == "s" || ans == "si" || ans == "sí" || ans == "y" || ans == "yes" {
		return true, nil
	}
	return false, nil
}

// PromptSelectClients displays the interactive client checklist in Spanish and parses the user's selection.
func PromptSelectClients(in io.Reader, out io.Writer, statuses []client.ClientStatus) ([]string, error) {
	fmt.Fprintln(out, "? ¿Qué clientes/agentes de IA utilizas?")
	var defaults []string
	for i, st := range statuses {
		mark := " "
		statusLabel := "[no detectado, se creará]"
		if st.Detected {
			mark = "x"
			statusLabel = "[detectado]"
			defaults = append(defaults, st.Spec.ID)
		}
		fmt.Fprintf(out, "  %d) [%s] %-13s (%-34s) %s\n", i+1, mark, st.Spec.Name, st.Spec.DisplayPath, statusLabel)
	}
	if len(defaults) == 0 {
		for _, st := range statuses {
			defaults = append(defaults, st.Spec.ID)
		}
	}

	fmt.Fprint(out, "Selección [Enter: usar marcados, números ej. 1,2,3 o 'all']: ")
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read client selection: %w", err)
	}

	raw := strings.TrimSpace(line)
	if raw == "" {
		return defaults, nil
	}
	if isAllToken(raw) {
		all := make([]string, len(statuses))
		for i, st := range statuses {
			all[i] = st.Spec.ID
		}
		return all, nil
	}

	var selected []string
	seen := make(map[string]bool)
	for _, token := range splitTokens(raw) {
		if idx, err := strconv.Atoi(token); err == nil {
			if idx < 1 || idx > len(statuses) {
				return nil, fmt.Errorf("índice de cliente fuera de rango: %d", idx)
			}
			id := statuses[idx-1].Spec.ID
			if !seen[id] {
				seen[id] = true
				selected = append(selected, id)
			}
			continue
		}
		spec, err := client.GetSpec(token)
		if err != nil {
			return nil, err
		}
		if !seen[spec.ID] {
			seen[spec.ID] = true
			selected = append(selected, spec.ID)
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no se seleccionó ningún cliente")
	}
	return selected, nil
}

// PromptSelectProfiles displays the interactive profile checklist in Spanish and parses the user's selection.
func PromptSelectProfiles(in io.Reader, out io.Writer, cfg *profile.Config, manifests []manifest.Manifest) ([]string, error) {
	names := cfg.SortedNames()
	if len(names) == 0 {
		return nil, fmt.Errorf("no hay perfiles disponibles para seleccionar")
	}

	fmt.Fprintln(out, "\n? ¿Qué perfiles de servidor MCP deseas compilar y sincronizar?")
	var defaults []string
	for i, name := range names {
		p := cfg.Profiles[name]
		mark := " "
		if p.Enabled {
			mark = "x"
			defaults = append(defaults, name)
		}
		summary := profile.ConnectionSummary(p)
		fmt.Fprintf(out, "  %d) [%s] %-16s (mcp: %-10s · %s)\n", i+1, mark, name, p.MCP, summary)
	}
	if len(defaults) == 0 {
		defaults = append(defaults, names...)
	}

	fmt.Fprint(out, "Selección [Enter: usar marcados, números ej. 1,2 o 'all']: ")
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read profile selection: %w", err)
	}

	raw := strings.TrimSpace(line)
	if raw == "" {
		return defaults, nil
	}
	if isAllToken(raw) {
		return names, nil
	}

	var selected []string
	seen := make(map[string]bool)
	for _, token := range splitTokens(raw) {
		if idx, err := strconv.Atoi(token); err == nil {
			if idx < 1 || idx > len(names) {
				return nil, fmt.Errorf("índice de perfil fuera de rango: %d", idx)
			}
			profName := names[idx-1]
			if !seen[profName] {
				seen[profName] = true
				selected = append(selected, profName)
			}
			continue
		}
		if _, ok := cfg.Profiles[token]; ok {
			if !seen[token] {
				seen[token] = true
				selected = append(selected, token)
			}
			continue
		}
		// Allow specifying an MCP ID directly if it matches a manifest
		if m, found := manifest.FindByID(manifests, token); found {
			cfg.Profiles[m.ID] = profile.SynthesizeDefault(m)
			if !seen[m.ID] {
				seen[m.ID] = true
				selected = append(selected, m.ID)
			}
			continue
		}
		return nil, fmt.Errorf("perfil desconocido %q", token)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no se seleccionó ningún perfil")
	}
	return selected, nil
}

// RenderList outputs the discovered manifests, profiles, and client detection status.
func RenderList(
	out io.Writer,
	manifests []manifest.Manifest,
	cfg *profile.Config,
	profilesSource string,
	clients []client.ClientStatus,
	asJSON bool,
) error {
	if asJSON {
		payload := map[string]any{
			"manifests":      manifests,
			"profilesSource": profilesSource,
			"profiles":       cfg.Profiles,
			"clients":        clients,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVIDORES MCP DESCUBIERTOS")
	fmt.Fprintln(tw, "ID\tRUNTIME\tBINARIO\tVARIABLES OBLIGATORIAS\tVARIABLES OPCIONALES\tDESCRIPCIÓN")
	for _, m := range manifests {
		var reqVars, optVars []string
		for _, ev := range m.Env {
			if ev.Required {
				reqVars = append(reqVars, ev.Name)
			} else {
				optVars = append(optVars, ev.Name)
			}
		}
		reqStr := strings.Join(reqVars, ",")
		if reqStr == "" {
			reqStr = "-"
		}
		optStr := strings.Join(optVars, ",")
		if optStr == "" {
			optStr = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", m.ID, m.Runtime, m.Build.Binary, reqStr, optStr, m.Description)
	}
	_ = tw.Flush()

	fmt.Fprintf(out, "\nPERFILES CARGADOS (%s)\n", profilesSource)
	tw = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PERFIL\tMCP\tHABILITADO\tCONEXIÓN / RESUMEN")
	for _, name := range cfg.SortedNames() {
		p := cfg.Profiles[name]
		fmt.Fprintf(tw, "%s\t%s\t%v\t%s\n", name, p.MCP, p.Enabled, profile.ConnectionSummary(p))
	}
	_ = tw.Flush()

	fmt.Fprintln(out, "\nCLIENTES DE IA SOPORTADOS")
	tw = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCLIENTE\tFORMATO\tARCHIVO GLOBAL\tESTADO")
	for _, c := range clients {
		state := "no detectado"
		if c.Detected {
			state = "detectado"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", c.Spec.ID, c.Spec.Name, strings.ToUpper(c.Spec.Format), c.Spec.DisplayPath, state)
	}
	return tw.Flush()
}

// RenderDoctor outputs the health check report in tabular format or JSON.
func RenderDoctor(out io.Writer, rep doctor.Report, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ESTADO DE BINARIOS (~/.mcp/bin)")
	fmt.Fprintln(tw, "MCP\tRUTA\tEXISTE\tEJECUTABLE\tESTADO")
	for _, b := range rep.Binaries {
		status := "OK"
		if b.Issue != "" {
			status = "ERROR: " + b.Issue
		}
		fmt.Fprintf(tw, "%s\t%s\t%v\t%v\t%s\n", b.ManifestID, b.BinaryPath, b.Exists, b.Executable, status)
	}
	_ = tw.Flush()

	fmt.Fprintf(out, "\nVALIDACIÓN DE PERFILES (%s)\n", rep.ProfilesPath)
	tw = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PERFIL\tMCP\tHABILITADO\tVÁLIDO\tDETALLE")
	for _, p := range rep.Profiles {
		detail := "OK"
		if !p.Valid {
			detail = strings.Join(p.Issues, "; ")
		}
		fmt.Fprintf(tw, "%s\t%s\t%v\t%v\t%s\n", p.Name, p.MCP, p.Enabled, p.Valid, detail)
	}
	_ = tw.Flush()

	fmt.Fprintln(out, "\nCONFIGURACIONES DE CLIENTES")
	tw = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CLIENTE\tARCHIVO\tDETECTADO\tSINTAXIS\tSERVIDORES REGISTRADOS")
	for _, c := range rep.Clients {
		syntax := "OK"
		if !c.FileExists {
			syntax = "(sin archivo)"
		} else if !c.Valid {
			syntax = "ERROR: " + c.Issue
		}
		profs := strings.Join(c.Profiles, ", ")
		if profs == "" {
			profs = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%v\t%s\t%s\n", c.ClientName, c.DisplayRel, c.Detected, syntax, profs)
	}
	_ = tw.Flush()

	if rep.Healthy {
		fmt.Fprintln(out, "\n✔ Diagnóstico completado: todos los cheques pasaron correctamente.")
	} else {
		fmt.Fprintf(out, "\n✖ Diagnóstico completado con %d problema(s).\n", len(rep.Issues))
	}
	return nil
}

// RenderSyncSummary outputs the final build and synchronization summary (or JSON / dry-run preview).
func RenderSyncSummary(
	out io.Writer,
	buildResults []builder.BuildResult,
	syncResults []client.SyncResult,
	warnings []string,
	dryRun bool,
	asJSON bool,
) error {
	if asJSON {
		payload := map[string]any{
			"dryRun":   dryRun,
			"builds":   buildResults,
			"clients":  syncResults,
			"warnings": warnings,
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	if len(warnings) > 0 {
		fmt.Fprintln(out, "\nADVERTENCIAS:")
		for _, w := range warnings {
			fmt.Fprintf(out, "  ! %s\n", w)
		}
	}

	if len(syncResults) > 0 {
		fmt.Fprintln(out, "\nRESUMEN DE SINCRONIZACIÓN")
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "CLIENTE\tARCHIVO\tPERFILES INYECTADOS\tESTADO")
		for _, sr := range syncResults {
			state := "OK"
			if sr.DryRun {
				state = "DRY-RUN"
			} else if sr.BackupPath != "" {
				state = fmt.Sprintf("OK (backup: %s)", filepath.Base(sr.BackupPath))
			} else if sr.Created {
				state = "OK (creado)"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", sr.ClientName, sr.DisplayRel, strings.Join(sr.Profiles, ", "), state)
		}
		_ = tw.Flush()

		if dryRun {
			for _, sr := range syncResults {
				fmt.Fprintf(out, "\n--- Previsualización Dry-Run: %s (%s) ---\n%s", sr.ClientName, sr.DisplayRel, sr.DryRunDiff)
			}
		}
	}

	return nil
}

func isAllToken(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return lower == "all" || lower == "todos" || lower == "*"
}

func splitTokens(s string) []string {
	f := func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == ';'
	}
	return strings.FieldsFunc(s, f)
}
