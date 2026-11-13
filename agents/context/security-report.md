# Security Audit Report — Instalador Multi-Agente y Gestor de Perfiles (`edro-mcp`)

- **Date**: 2026-11-11
- **Cycle**: Initial
- **Scope Audited**: [`../specs/architecture.md`](../specs/architecture.md), [`../specs/prd.md`](../specs/prd.md), [`../specs/design.md`](../specs/design.md), [`qa-report.md`](qa-report.md), Go CLI codebase (`cmd/edro-mcp`, `internal/*`), bootstrap scripts (`install.ps1`, `install.sh`), MCP manifests (`database/mysql/mcp.manifest.json`, `database/postgresql/mcp.manifest.json`), and module dependency graphs (`go.mod`).

## 1. Automated Scans Executed
| Tool / Scan | Command | Result |
| :--- | :--- | :--- |
| Static analysis (`go vet` — root CLI) | `go vet ./...` | Clean (exit code `0`) |
| Static analysis (`go vet` — MCP servers) | `go vet ./...` in `../../database/mysql` and `../../database/postgresql` | Clean (exit code `0`) |
| Dependency / Vuln scan | `govulncheck ./...` (fallback: `go.mod` dependency graph audit) | Clean — root `go.mod` uses Go standard library only (zero third-party dependencies); `database/mysql` and `database/postgresql` remain isolated with pinned driver/SDK versions |
| Secret & credential scan | Regex scan across all tracked/untracked repository files for API keys, private keys, and literal credentials | Clean — zero hardcoded secrets or credentials; all profile templates use `{env:VAR_NAME}` placeholders |
| Portability / Host-path audit | `audit-paths.ps1 -Root .` | Clean (`Exit 0` — `PASS: No hardcoded host paths detected`) |

## 2. Security & Hardening Checklist
- [x] AuthN / AuthZ boundaries enforced (local CLI operates strictly within user-owned `~/.mcp/bin` and user-level client configs; database MCP servers enforce `DATABASE_PERMISSIONS` SQL statement whitelisting)
- [x] Input validation & injection protection (command execution uses `exec.CommandContext` with argument slices and no shell interpolation; JSON/JSONC/TOML config mergers use `encoding/json` and `strconv.Quote` to prevent config/section injection)
- [x] No hardcoded secrets, tokens, or sensitive data in logs (`--dry-run` masks resolved host variables as `<resolved from $VAR_NAME>` and masks `sensitive: true` literals as `********`; `.gitignore` excludes `/profiles.json`, `.env`, `*.bak`, `*.tmp`, and binaries)
- [x] Hardening applied (least-privilege file permissions `0o600` on `profiles.json`, `.bak` backups, and `.tmp` atomic config writes; `0o755` on directories and compiled binaries; clean error wrapping without leaking secret values)

## 3. Findings *(if any)*
| ID | Severity | Location | Impact | Required Remediation |
| :--- | :--- | :--- | :--- | :--- |
| `SEC-01` | Low | [`../../internal/manifest/manifest.go`](../../internal/manifest/manifest.go#L147-L196) | `Manifest.Validate()` verifies `build.binary` is non-empty, and `BinaryPath()` joins it with `binDir`. If a user loads an untrusted external manifest containing `../` segments in `build.binary`, the output path could resolve outside `binDir`. | Optional defense-in-depth: validate `filepath.Base(m.Build.Binary) == m.Build.Binary` inside `Manifest.Validate()`. |
| `SEC-02` | Low | [`../../internal/ui/ui.go`](../../internal/ui/ui.go#L208-L218) | `RenderList` in `--json` mode serializes `cfg.Profiles` directly. While `{env:VAR_NAME}` references are safe, a user who manually places a plaintext password in `profiles.json` instead of `{env:VAR_NAME}` would see it echoed in `edro-mcp list --json`. | Optional defense-in-depth: apply manifest sensitivity masking to `cfg.Profiles` before encoding `list --json` output. |

## 4. Verdict
`STATUS: PASS`
