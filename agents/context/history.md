# Historial de edro-mcp

> Entradas ordenadas cronológicamente de más reciente a más antigua (orden cronológico descendente).

## 2026-11-11 — Fase 1: Instalador Multi-Agente y Gestor de Perfiles (`edro-mcp`)
- **Detalle de cambios ([`cmd/edro-mcp/main.go`](../../cmd/edro-mcp/main.go), [`internal/manifest/manifest.go`](../../internal/manifest/manifest.go), [`internal/profile/profile.go`](../../internal/profile/profile.go), [`internal/builder/builder.go`](../../internal/builder/builder.go), [`internal/client/client.go`](../../internal/client/client.go), [`internal/ui/ui.go`](../../internal/ui/ui.go), [`internal/doctor/doctor.go`](../../internal/doctor/doctor.go))**:
  - Implementación del CLI `edro-mcp` (100% biblioteca estándar de Go) con comandos `install`, `sync`, `build`, `list` y `doctor`/`status`, modo interactivo en español ("Preguntar qué uso") y banderas para automatización.
  - Creación de manifiestos declarativos agnósticos al lenguaje en [`database/mysql/mcp.manifest.json`](../../database/mysql/mcp.manifest.json) y [`database/postgresql/mcp.manifest.json`](../../database/postgresql/mcp.manifest.json), plantilla [`profiles.example.json`](../../profiles.example.json) y scripts de arranque [`install.ps1`](../../install.ps1) e [`install.sh`](../../install.sh).
  - Compilación e instalación global de binarios en `~/.mcp/bin/` y adaptadores de fusión no destructiva con backup `.bak` para Antigravity, OpenCode (JSONC), Codex (TOML), Claude Code y Cursor.
- **Verificación**:
  - `go vet ./...` y `go test -count=1 ./...` exitosos en la raíz, `database/mysql` y `database/postgresql`.
  - Reportes de verificación E2E ([`qa-report.md`](qa-report.md)) y auditoría de seguridad ([`security-report.md`](security-report.md)) aprobados con `STATUS: PASS`.
  - Escáner de portabilidad `audit-paths` ejecutado con código de salida `0` (`PASS: No hardcoded host paths detected`).

## 2026-11-11 — Fase 0: Inicialización de contexto
- **Detalle de cambios ([`AGENTS.md`](../../AGENTS.md))**:
  - Creación de estructura base de contexto y especificaciones en `agents/`.
- **Verificación**:
  - Estructura validada sin rutas absolutas de host.
