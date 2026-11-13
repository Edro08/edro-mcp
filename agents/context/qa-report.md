# QA Report — Instalador Multi-Agente y Gestor de Perfiles (`edro-mcp`)

- **Date**: 2026-11-11
- **Cycle**: Initial
- **Scope Tested**: [`../specs/prd.md`](../specs/prd.md), [`../specs/design.md`](../specs/design.md)

## 1. Test Environment & Real Checks Executed
| Check / Scenario | Command or Method | Observed Result | Pass/Fail |
| :--- | :--- | :--- | :--- |
| Root Go unit & integration suite | `go test -count=1 ./...` | `ok edro-mcp/cmd/edro-mcp`, `ok edro-mcp/internal/builder`, `ok edro-mcp/internal/client`, `ok edro-mcp/internal/doctor`, `ok edro-mcp/internal/manifest`, `ok edro-mcp/internal/profile`, `ok edro-mcp/internal/ui` (all tests passed, exit code `0`). | PASS |
| Regression suite (`database/mysql` & `database/postgresql`) | `go test -v ./...` in [`../../database/mysql`](../../database/mysql) and [`../../database/postgresql`](../../database/postgresql) | `ok mysql/database` and `ok postgresql/database` (all existing SQL validation tests passed without regression). | PASS |
| Discovery & listing (`list` & `list --json`) | `edro-mcp list` & `edro-mcp list --json` (isolated `HOME=<TEMP_DIR>`) | Discovered `mysql` and `postgresql` manifests from [`../../database/mysql/mcp.manifest.json`](../../database/mysql/mcp.manifest.json) and [`../../database/postgresql/mcp.manifest.json`](../../database/postgresql/mcp.manifest.json), displayed required/optional env vars, loaded 3 profiles (`mysql-local`, `mysql-staging`, `postgres-local`) from [`../../profiles.example.json`](../../profiles.example.json), and listed all 5 supported clients (`antigravity`, `opencode`, `codex`, `claude`, `cursor`). | PASS |
| Centralized binary build (`build --all`) | `edro-mcp build --all --bin-dir <TEMP_DIR>/.mcp/bin` | Compiled and installed `mysql-mcp.exe` (13.17 MB) and `postgresql-mcp.exe` (18.52 MB) into `<TEMP_DIR>/.mcp/bin/` with executable permissions. | PASS |
| Dry-run preview & secret masking (`sync --dry-run`) | `edro-mcp sync --clients all --profiles all --bin-dir <TEMP_DIR>/.mcp/bin --dry-run` | Previewed changes for all 5 clients without creating `.bak` files or modifying disk files; sensitive resolved variables (`MYSQL_LOCAL_PASSWORD`, `POSTGRES_LOCAL_PASSWORD`) were masked as `<resolved from $VAR_NAME>` and missing system variables emitted clear warnings in Spanish. | PASS |
| Non-destructive multi-client sync (`sync --clients all --profiles all`) | `edro-mcp sync --clients all --profiles all --bin-dir <TEMP_DIR>/.mcp/bin` against pre-populated `mcp_config.json`, `opencode.jsonc` (with comments & trailing commas), `config.toml`, `.claude.json`, and `mcp.json` | Created `.bak` backups (`mcp_config.json.bak`, `opencode.jsonc.bak`, `config.toml.bak`, `.claude.json.bak`, `mcp.json.bak`); preserved custom top-level keys, header comments in JSONC, TOML root keys, and external MCP servers (`chrome-devtools`, `external-mcp`, `github`, `filesystem`, `existing-cursor`); injected `mysql-local`, `mysql-staging`, and `postgres-local` with client-specific `{env:VAR}` translations (`{env:VAR}` in OpenCode, `${VAR}` in Claude Code, resolved env or placeholder in Antigravity/Codex/Cursor). | PASS |
| Interactive installer flow (`install`) | `"n`nall`nall`n" \| edro-mcp install --bin-dir <TEMP_DIR>/.mcp/bin` | Displayed Spanish interactive prompts for client and profile multi-selection, deduplicated builds per unique MCP (`mysql` built once for `mysql-local` + `mysql-staging`), and synced all selected clients cleanly (exit code `0`). | PASS |
| Health diagnostics (`doctor` / `status --json`) | `edro-mcp doctor --bin-dir <TEMP_DIR>/.mcp/bin` & `edro-mcp status --bin-dir <TEMP_DIR>/.mcp/bin --json` | Verified binaries exist and are executable, validated required profile env vars, verified syntax of all 5 client configs, and exited with code `0` (`"healthy": true`). | PASS |
| Portability audit | `audit-paths.ps1 -Root .` | `PASS: No hardcoded host paths detected` across the repository. | PASS |

## 2. Acceptance Criteria Coverage
- [x] **AC-01 (Descubrimiento de Manifiestos)**: Verified — `edro-mcp list` and `edro-mcp list --json` discover both [`../../database/mysql/mcp.manifest.json`](../../database/mysql/mcp.manifest.json) and [`../../database/postgresql/mcp.manifest.json`](../../database/postgresql/mcp.manifest.json) and output their required (`DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_NAME`, `DATABASE_USERNAME`, `DATABASE_PASSWORD`) and optional (`DATABASE_PERMISSIONS`) environment variables.
- [x] **AC-02 (Build e Instalación Global)**: Verified — `edro-mcp build --all` compiles `mysql` and `postgresql` into `~/.mcp/bin/` (`<TEMP_DIR>/.mcp/bin/mysql-mcp.exe` and `postgresql-mcp.exe`) with executable permissions and dynamically resolved runtime paths.
- [x] **AC-03 (Soporte Multi-Perfil)**: Verified — `mysql-local` and `mysql-staging` (both referencing `mcp: "mysql"`) along with `postgres-local` (`mcp: "postgresql"`) are registered as independent server entries under their respective profile names and isolated environment maps across all 5 client adapters while deduplicating the `mysql` binary build.
- [x] **AC-04 (Preservación de Configuración Existente)**: Verified — Existing top-level keys (`customKey`, `theme`, `model`, `approval_policy`, `projects`), header comments (`// Header comment to preserve` in `opencode.jsonc`), and external MCP entries (`chrome-devtools`, `external-mcp`, `github`, `filesystem`, `existing-cursor`) remain intact after sync, `.bak` files are created before modification, and `edro-mcp doctor` confirms valid syntax across JSON, JSONC, and TOML.
- [x] **AC-05 (Flujo Interactivo y No Interactivo)**: Verified — Interactive mode (`edro-mcp install`) guides the user in Spanish through client and profile selection, while non-interactive mode (`--clients`, `--profiles`, `--all`, `--dry-run`) completes without requiring `stdin`.
- [x] **AC-06 (Portabilidad Estricta)**: Verified — `audit-paths.ps1` passed with zero findings; all user home and binary paths are resolved dynamically via `os.UserHomeDir()` and `filepath.Join`.

## 3. Defects Found *(if any)*
None.

## 4. Verdict
`STATUS: PASS`
