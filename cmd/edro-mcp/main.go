package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"edro-mcp/internal/builder"
	"edro-mcp/internal/client"
	"edro-mcp/internal/doctor"
	"edro-mcp/internal/manifest"
	"edro-mcp/internal/profile"
	"edro-mcp/internal/ui"
)

type options struct {
	subcommand     string
	clients        string
	profiles       string
	mcps           string
	all            bool
	configPath     string
	binDir         string
	repoRoot       string
	homeDir        string
	dryRun         bool
	resolveEnv     string
	nonInteractive bool
	jsonOutput     bool
	forceBuild     bool
	listFlag       bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, "")
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, overrideHome string) int {
	opts, err := parseArgs(args, stderr)
	if err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if opts.subcommand == "help" {
		printUsage(stdout)
		return 0
	}

	homeDir := overrideHome
	if homeDir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(stderr, "error: resolve user home directory: %v\n", err)
			return 1
		}
		homeDir = h
	}
	opts.homeDir = homeDir

	if opts.binDir == "" {
		opts.binDir = builder.DefaultBinDir(homeDir)
	}
	if opts.repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "error: get working directory: %v\n", err)
			return 1
		}
		opts.repoRoot = detectRepoRoot(cwd)
	}

	policy, err := profile.ParseResolvePolicy(opts.resolveEnv)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	manifests, err := manifest.Discover(opts.repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: discover manifests in %s: %v\n", opts.repoRoot, err)
		return 1
	}

	switch opts.subcommand {
	case "list":
		return runList(opts, manifests, stdout, stderr)
	case "doctor", "status":
		return runDoctor(opts, manifests, stdout, stderr)
	case "build":
		return runBuild(ctx, opts, manifests, stdout, stderr)
	case "install":
		return runInstallOrSync(ctx, opts, manifests, policy, true, stdin, stdout, stderr)
	case "sync":
		return runInstallOrSync(ctx, opts, manifests, policy, opts.forceBuild, stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: subcomando desconocido %q (válidos: install, sync, build, list, doctor, status)\n", opts.subcommand)
		return 2
	}
}

func parseArgs(args []string, stderr io.Writer) (options, error) {
	var opts options
	subcmd := "install"

	var flagArgs []string
	subcmdSet := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !subcmdSet && !strings.HasPrefix(arg, "-") {
			subcmd = strings.ToLower(strings.TrimSpace(arg))
			subcmdSet = true
			continue
		}
		flagArgs = append(flagArgs, arg)
	}

	fs := flag.NewFlagSet("edro-mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printUsage(stderr)
	}

	fs.StringVar(&opts.clients, "clients", "", "Clientes destino separados por comas (antigravity,opencode,codex,claude,cursor o all)")
	fs.StringVar(&opts.profiles, "profiles", "", "Perfiles a sincronizar separados por comas o all")
	fs.StringVar(&opts.mcps, "mcps", "", "IDs de servidores MCP separados por comas (ej. mysql,postgresql)")
	fs.BoolVar(&opts.all, "all", false, "Seleccionar todos los clientes detectados y todos los perfiles habilitados")
	fs.StringVar(&opts.configPath, "config", "", "Ruta explícita al archivo profiles.json")
	fs.StringVar(&opts.binDir, "bin-dir", "", "Directorio destino de binarios compilados (por defecto ~/.mcp/bin)")
	fs.StringVar(&opts.repoRoot, "repo-root", "", "Directorio raíz del repositorio con archivos mcp.manifest.json")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "Previsualizar cambios sin escribir ni modificar archivos en disco")
	fs.StringVar(&opts.resolveEnv, "resolve-env", "auto", "Política de traducción para {env:VAR_NAME}: auto, literal o keep")
	fs.BoolVar(&opts.nonInteractive, "non-interactive", false, "Ejecutar sin prompts interactivos")
	fs.BoolVar(&opts.jsonOutput, "json", false, "Emitir salida estructurada en formato JSON")
	fs.BoolVar(&opts.forceBuild, "build", false, "Forzar compilación de binarios durante sync")
	fs.BoolVar(&opts.listFlag, "list", false, "Listar servidores MCP descubiertos, perfiles y clientes")

	if err := fs.Parse(flagArgs); err != nil {
		return opts, err
	}

	if opts.listFlag {
		subcmd = "list"
	}
	opts.subcommand = subcmd
	return opts, nil
}

func runList(opts options, manifests []manifest.Manifest, stdout, stderr io.Writer) int {
	cfg, source, err := profile.Load(opts.configPath, opts.repoRoot, opts.homeDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: load profiles: %v\n", err)
		return 1
	}
	clients := client.DetectAll(opts.homeDir)
	if err := ui.RenderList(stdout, manifests, cfg, displayRelativeOrClean(opts.repoRoot, source), clients, opts.jsonOutput); err != nil {
		fmt.Fprintf(stderr, "error: render list: %v\n", err)
		return 1
	}
	return 0
}

func runDoctor(opts options, manifests []manifest.Manifest, stdout, stderr io.Writer) int {
	cfg, source, err := profile.Load(opts.configPath, opts.repoRoot, opts.homeDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: load profiles: %v\n", err)
		return 1
	}
	rep := doctor.Run(manifests, cfg, displayRelativeOrClean(opts.repoRoot, source), opts.binDir, opts.homeDir)
	if err := ui.RenderDoctor(stdout, rep, opts.jsonOutput); err != nil {
		fmt.Fprintf(stderr, "error: render doctor report: %v\n", err)
		return 1
	}
	if !rep.Healthy {
		return 1
	}
	return 0
}

func runBuild(ctx context.Context, opts options, manifests []manifest.Manifest, stdout, stderr io.Writer) int {
	if len(manifests) == 0 {
		fmt.Fprintln(stderr, "error: no se encontraron archivos mcp.manifest.json")
		return 1
	}

	var selected []manifest.Manifest
	if opts.mcps != "" && !strings.EqualFold(strings.TrimSpace(opts.mcps), "all") {
		for _, id := range splitCSV(opts.mcps) {
			m, ok := manifest.FindByID(manifests, id)
			if !ok {
				fmt.Fprintf(stderr, "error: mcp %q no encontrado\n", id)
				return 1
			}
			selected = append(selected, m)
		}
	} else {
		selected = manifests
	}

	results, err := builder.BuildMany(ctx, selected, opts.binDir, opts.dryRun)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if opts.jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"dryRun": opts.dryRun,
			"builds": results,
		})
		return 0
	}

	for _, r := range results {
		prefix := "✔"
		if r.DryRun {
			prefix = "[dry-run]"
		}
		fmt.Fprintf(stdout, "  %s %-12s -> %s\n", prefix, r.ManifestID, r.BinaryPath)
	}
	return 0
}

func runInstallOrSync(
	ctx context.Context,
	opts options,
	manifests []manifest.Manifest,
	policy profile.ResolvePolicy,
	alwaysBuild bool,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	if len(manifests) == 0 {
		fmt.Fprintln(stderr, "error: no se encontraron servidores MCP (mcp.manifest.json)")
		return 1
	}

	hasFilterFlags := opts.clients != "" || opts.profiles != "" || opts.mcps != "" || opts.all
	interactive := !opts.nonInteractive && !hasFilterFlags

	// Offer to initialize profiles.json from profiles.example.json in interactive mode if missing
	if interactive && opts.configPath == "" && !profile.HasUserProfiles(opts.repoRoot, opts.homeDir) {
		examplePath := filepath.Join(opts.repoRoot, profile.ExampleProfilesName)
		if _, err := os.Stat(examplePath); err == nil && !opts.dryRun {
			createLocal, promptErr := ui.PromptInitProfiles(stdin, stdout)
			if promptErr != nil {
				fmt.Fprintf(stderr, "error: %v\n", promptErr)
				return 1
			}
			if createLocal {
				if createdPath, copyErr := profile.CopyExampleToLocal(opts.repoRoot); copyErr == nil {
					fmt.Fprintf(stdout, "  ✔ Creado %s\n\n", displayRelativeOrClean(opts.repoRoot, createdPath))
				}
			}
		}
	}

	cfg, source, err := profile.Load(opts.configPath, opts.repoRoot, opts.homeDir)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// If no profiles exist in config at all, synthesize default profiles from manifests
	if len(cfg.Profiles) == 0 {
		for _, m := range manifests {
			cfg.Profiles[m.ID] = profile.SynthesizeDefault(m)
		}
	}

	statuses := client.DetectAll(opts.homeDir)

	if interactive && !opts.jsonOutput {
		ui.PrintBanner(stdout, manifests, displayRelativeOrClean(opts.repoRoot, source), len(cfg.Profiles))
	}

	// 1. Resolve target clients
	selectedClients, err := resolveSelectedClients(opts, statuses, interactive, stdin, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// 2. Resolve target profiles
	selectedProfiles, err := resolveSelectedProfiles(opts, cfg, manifests, interactive, stdin, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// 3. Validate required variables on selected profiles and collect manifests to build
	var manifestsToBuild []manifest.Manifest
	for _, profName := range selectedProfiles {
		p := cfg.Profiles[profName]
		m, found := manifest.FindByID(manifests, p.MCP)
		if !found {
			fmt.Fprintf(stderr, "error: el perfil %q referencia un MCP inexistente %q\n", profName, p.MCP)
			return 1
		}
		if valErrs := profile.ValidateProfile(profName, p, m); len(valErrs) > 0 {
			for _, ve := range valErrs {
				fmt.Fprintf(stderr, "error de validación: %s\n", ve)
			}
			return 1
		}

		binPath := m.BinaryPath(opts.binDir, runtime.GOOS)
		_, statErr := os.Stat(binPath)
		if alwaysBuild || os.IsNotExist(statErr) {
			manifestsToBuild = append(manifestsToBuild, m)
		}
	}

	// 4. Build required MCP binaries into ~/.mcp/bin/
	var buildResults []builder.BuildResult
	if len(manifestsToBuild) > 0 {
		if !opts.jsonOutput {
			fmt.Fprintf(stdout, "\n[1/2] Compilando e instalando binarios en %s...\n", opts.binDir)
		}
		buildResults, err = builder.BuildMany(ctx, manifestsToBuild, opts.binDir, opts.dryRun)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		if !opts.jsonOutput {
			for _, br := range buildResults {
				mark := "✔"
				if br.DryRun {
					mark = "[dry-run]"
				}
				fmt.Fprintf(stdout, "  %s %-12s -> %s\n", mark, br.ManifestID, br.BinaryPath)
			}
		}
	}

	// 5. Synchronize selected profiles into each selected client
	if !opts.jsonOutput {
		fmt.Fprintln(stdout, "\n[2/2] Sincronizando perfiles en clientes seleccionados...")
	}

	var syncResults []client.SyncResult
	var allWarnings []string
	seenWarnings := make(map[string]bool)

	for _, clientID := range selectedClients {
		entries, warnings, err := client.BuildEntriesForClient(
			selectedProfiles,
			cfg,
			manifests,
			opts.binDir,
			clientID,
			policy,
			os.LookupEnv,
		)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		for _, w := range warnings {
			if !seenWarnings[w] {
				seenWarnings[w] = true
				allWarnings = append(allWarnings, w)
			}
		}

		sr, err := client.SyncClient(opts.homeDir, clientID, entries, opts.dryRun)
		if err != nil {
			fmt.Fprintf(stderr, "error sincronizando %s: %v\n", clientID, err)
			return 1
		}
		sr.Warnings = warnings
		syncResults = append(syncResults, sr)

		if !opts.jsonOutput {
			mark := "✔"
			suffix := ""
			if sr.DryRun {
				mark = "[dry-run]"
			} else if sr.BackupPath != "" {
				suffix = fmt.Sprintf(" (backup: %s)", filepath.Base(sr.BackupPath))
			} else if sr.Created {
				suffix = " (creado)"
			}
			fmt.Fprintf(stdout, "  %s %-12s -> %s%s\n", mark, sr.ClientName, sr.DisplayRel, suffix)
		}
	}

	if err := ui.RenderSyncSummary(stdout, buildResults, syncResults, allWarnings, opts.dryRun, opts.jsonOutput); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

func resolveSelectedClients(
	opts options,
	statuses []client.ClientStatus,
	interactive bool,
	stdin io.Reader,
	stdout io.Writer,
) ([]string, error) {
	if opts.clients != "" {
		if strings.EqualFold(strings.TrimSpace(opts.clients), "all") {
			all := make([]string, len(statuses))
			for i, st := range statuses {
				all[i] = st.Spec.ID
			}
			return all, nil
		}
		var out []string
		seen := make(map[string]bool)
		for _, token := range splitCSV(opts.clients) {
			spec, err := client.GetSpec(token)
			if err != nil {
				return nil, err
			}
			if !seen[spec.ID] {
				seen[spec.ID] = true
				out = append(out, spec.ID)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("debes especificar al menos un cliente en --clients")
		}
		return out, nil
	}

	if opts.all {
		var detected []string
		for _, st := range statuses {
			if st.Detected {
				detected = append(detected, st.Spec.ID)
			}
		}
		if len(detected) == 0 {
			for _, st := range statuses {
				detected = append(detected, st.Spec.ID)
			}
		}
		return detected, nil
	}

	if interactive {
		return ui.PromptSelectClients(stdin, stdout, statuses)
	}

	return nil, fmt.Errorf("modo no interactivo requiere especificar --clients o --all")
}

func resolveSelectedProfiles(
	opts options,
	cfg *profile.Config,
	manifests []manifest.Manifest,
	interactive bool,
	stdin io.Reader,
	stdout io.Writer,
) ([]string, error) {
	var out []string
	seen := make(map[string]bool)

	addProfile := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}

	if opts.mcps != "" {
		var targetMCPs []manifest.Manifest
		if strings.EqualFold(strings.TrimSpace(opts.mcps), "all") {
			targetMCPs = manifests
		} else {
			for _, id := range splitCSV(opts.mcps) {
				m, ok := manifest.FindByID(manifests, id)
				if !ok {
					return nil, fmt.Errorf("mcp %q no encontrado", id)
				}
				targetMCPs = append(targetMCPs, m)
			}
		}

		for _, m := range targetMCPs {
			matchedAny := false
			for _, profName := range cfg.SortedNames() {
				p := cfg.Profiles[profName]
				if p.MCP == m.ID && p.Enabled {
					addProfile(profName)
					matchedAny = true
				}
			}
			if !matchedAny {
				cfg.Profiles[m.ID] = profile.SynthesizeDefault(m)
				addProfile(m.ID)
			}
		}
	}

	if opts.profiles != "" {
		if strings.EqualFold(strings.TrimSpace(opts.profiles), "all") {
			for _, name := range cfg.SortedNames() {
				if cfg.Profiles[name].Enabled {
					addProfile(name)
				}
			}
		} else {
			for _, name := range splitCSV(opts.profiles) {
				if _, ok := cfg.Profiles[name]; !ok {
					return nil, fmt.Errorf("perfil %q no encontrado en configuración", name)
				}
				addProfile(name)
			}
		}
	}

	if len(out) > 0 {
		return out, nil
	}

	if opts.all || (opts.clients != "" && !interactive) {
		for _, name := range cfg.SortedNames() {
			if cfg.Profiles[name].Enabled {
				addProfile(name)
			}
		}
		if len(out) == 0 {
			for _, name := range cfg.SortedNames() {
				addProfile(name)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no hay perfiles disponibles")
		}
		return out, nil
	}

	if interactive {
		return ui.PromptSelectProfiles(stdin, stdout, cfg, manifests)
	}

	return nil, fmt.Errorf("modo no interactivo requiere especificar --profiles, --mcps o --all")
}

func detectRepoRoot(startDir string) string {
	curr := filepath.Clean(startDir)
	for {
		if _, err := os.Stat(filepath.Join(curr, profile.ExampleProfilesName)); err == nil {
			return curr
		}
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			data, _ := os.ReadFile(filepath.Join(curr, "go.mod"))
			if strings.Contains(string(data), "module edro-mcp") {
				return curr
			}
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return filepath.Clean(startDir)
}

func displayRelativeOrClean(baseDir, targetPath string) string {
	if targetPath == "" {
		return ""
	}
	if baseDir != "" {
		if rel, err := filepath.Rel(baseDir, targetPath); err == nil && !strings.HasPrefix(rel, "..") {
			return "./" + filepath.ToSlash(rel)
		}
	}
	return targetPath
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return out
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, `Uso: edro-mcp [comando] [banderas]

Comandos:
  install            Descubre manifiestos, compila binarios en ~/.mcp/bin/ y sincroniza perfiles (por defecto)
  sync               Sincroniza perfiles hacia las configuraciones globales de los clientes seleccionados
  build              Compila e instala en ~/.mcp/bin/ los servidores MCP descubiertos
  list               Lista los servidores MCP descubiertos, perfiles cargados y estado de clientes
  doctor (o status)  Verifica binarios en ~/.mcp/bin/, perfiles y sintaxis de configuraciones de clientes

Banderas:
  --clients string       Lista CSV de clientes: antigravity,opencode,codex,claude,cursor o all
  --profiles string      Lista CSV de perfiles definidos en profiles.json o all
  --mcps string          Lista CSV de IDs de servidores MCP (ej. mysql,postgresql)
  --all                  Seleccionar todos los clientes detectados y todos los perfiles habilitados
  --config string        Ruta explícita al archivo de perfiles (profiles.json)
  --bin-dir string       Directorio destino de binarios compilados (por defecto ~/.mcp/bin)
  --dry-run              Previsualizar cambios sin escribir ni modificar archivos en disco
  --resolve-env string   Política para referencias {env:VAR_NAME}: auto, literal o keep (por defecto "auto")
  --non-interactive      Forzar ejecución sin prompts interactivos
  --json                 Emitir salida en formato JSON estructurado`)
}
