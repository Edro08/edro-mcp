# Fases de desarrollo de edro-mcp

## 🟢 Fases completadas
- **Fase 0 — Inicialización**: Configuración de `AGENTS.md` y estructura `agents/` en la rama `feat/mcp-installer`.
- **Fase 1 — Instalador Multi-Agente y Gestor de Perfiles (`edro-mcp`)**:
  - [x] Definir alcance y criterios de aceptación (`prd.md`).
  - [x] Diseñar contratos CLI, manifiestos y esquemas de perfiles (`design.md`).
  - [x] Definir arquitectura detallada y plan de implementación (`architecture.md` / `fases.md`).
  - [x] **Subfase 1.1 — Manifiestos, Plantillas y Configuración Raíz**:
    - Crear `go.mod` en la raíz del repositorio (`module edro-mcp`, stdlib pura).
    - Crear `database/mysql/mcp.manifest.json` y `database/postgresql/mcp.manifest.json`.
    - Crear `profiles.example.json` en la raíz del repositorio y actualizar `.gitignore` para excluir `/profiles.json`, `*.bak` y binarios compilados.
  - [x] **Subfase 1.2 — Dominio Base (`internal/manifest`, `internal/profile`, `internal/builder`)**:
    - Implementar descubrimiento recursivo, validación de esquema e interpolación de `{bin_dir}` y `{exe_ext}` en `internal/manifest` con pruebas unitarias (`manifest_test.go`).
    - Implementar carga de `profiles.json`, generación de perfiles por defecto desde manifiestos, validación de variables `required` y políticas de resolución `{env:VAR_NAME}` (`auto`, `literal`, `keep`) con enmascaramiento de secretos en `internal/profile` (`profile_test.go`).
    - Implementar compilación e instalación en `~/.mcp/bin/` en `internal/builder` soportando `go`, `command` y `--dry-run` (`builder_test.go`).
  - [x] **Subfase 1.3 — Adaptadores Multi-Cliente (`internal/client`)**:
    - Implementar adaptadores y fusión no destructiva con backup `.bak` para Antigravity (JSON), Claude Code (JSON), Cursor (JSON), OpenCode (JSONC con preservación/tolerancia de comentarios y comas finales) y Codex (TOML por secciones `[mcp_servers.<profile>]`).
    - Crear suite exhaustiva de pruebas unitarias (`client_test.go`) verificando preservación de configuraciones previas y servidores MCP externos (`AC-03`, `AC-04`).
  - [x] **Subfase 1.4 — CLI, Flujo Interactivo, Doctor y Scripts de Bootstrap (`cmd/edro-mcp`, `internal/ui`, `internal/doctor`)**:
    - Implementar subcomandos `install`, `sync`, `build`, `list` y `doctor` (alias `status`) con todas las banderas de `design.md` (`--clients`, `--profiles`, `--mcps`, `--all`, `--config`, `--bin-dir`, `--dry-run`, `--resolve-env`, `--non-interactive`, `--json`).
    - Implementar asistente interactivo en español en `internal/ui` y verificador `internal/doctor`.
    - Crear scripts ligeros de arranque `install.ps1` e `install.sh` en la raíz del repositorio y actualizar `README.md`, `database/mysql/README.md` y `database/postgresql/README.md`.
  - [x] Ejecutar pruebas (`go vet ./...`, `go test ./...`), QA (`STATUS: PASS`), auditoría de seguridad (`STATUS: PASS`) y auditoría de portabilidad (`PASS`).

## 🟡 Fases en progreso
- *(Sin fases en progreso actualmente)*

## ⚪ Fases planificadas
- **Fase 2 — Nuevos servidores MCP y plantillas**: Incorporación de servidores MCP adicionales en otros lenguajes (Python/Node/Go) aprovechando el contrato `mcp.manifest.json`.
