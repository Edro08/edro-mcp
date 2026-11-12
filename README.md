# edro-mcp · Instalador Multi-Agente y Monorepo de Servidores MCP

`edro-mcp` es un repositorio unificado de servidores **Model Context Protocol (MCP)** locales (`stdio`) acompañado de una herramienta CLI en Go (sin dependencias externas) para descubrir, compilar en `~/.mcp/bin/`, gestionar múltiples perfiles de conexión (`profiles.json`) e inyectar configuraciones de forma segura en los principales agentes y clientes de IA.

## Características Principales

1. **Descubrimiento Agnóstico (`mcp.manifest.json`)**:
   Cada servidor MCP declara su contrato de compilación, ejecución y variables de entorno en un archivo `mcp.manifest.json`.
2. **Binarios Desacoplados del Repositorio (`~/.mcp/bin/`)**:
   Todos los ejecutables se instalan en el directorio global del usuario (`~/.mcp/bin/` en Linux/macOS y `%USERPROFILE%\.mcp\bin\` en Windows), eliminando la necesidad de configurar `cwd` acoplado a la ruta donde clonaste el repositorio.
3. **Gestión Multi-Perfil (`profiles.json`)**:
   Permite registrar múltiples instancias de un mismo servidor MCP (por ejemplo, `mysql-local`, `mysql-staging`, `postgres-local`) sin exponer credenciales en Git.
4. **Sincronización Multi-Cliente con Respaldo Automático (`.bak`)**:
   Inyecta y actualiza tus perfiles preservando configuraciones previas, temas, ajustes y otros servidores MCP externos en:
   - **Antigravity**: `~/.gemini/config/mcp_config.json` (JSON)
   - **OpenCode**: `~/.config/opencode/opencode.jsonc` (JSONC, preservando comentarios de cabecera y ajustes)
   - **Codex**: `~/.codex/config.toml` (TOML, secciones `[mcp_servers.<perfil>]`)
   - **Claude Code**: `~/.claude.json` (JSON)
   - **Cursor**: `~/.cursor/mcp.json` (JSON)

---

## Inicio Rápido

### 1. Configurar Perfiles Locales

Copia la plantilla versionada [`profiles.example.json`](profiles.example.json) a `profiles.json` (archivo ignorado por Git):

```sh
cp profiles.example.json profiles.json
```

Edita `profiles.json` con tus instancias de conexión y define las contraseñas mediante variables de entorno del sistema usando la sintaxis `{env:NOMBRE_VARIABLE}`.

### 2. Ejecutar el Instalador Interactivo ("Preguntar qué uso")

Puedes usar el script de arranque para tu sistema operativo o ejecutar directamente con `go run`:

**Windows (PowerShell):**
```powershell
.\install.ps1
```

**Linux / macOS (Bash):**
```bash
./install.sh
```

**Directamente con Go:**
```sh
go run ./cmd/edro-mcp install
```

El asistente interactivo detectará qué clientes de IA tienes instalados, te permitirá elegir qué clientes y perfiles sincronizar, compilará los binarios necesarios en `~/.mcp/bin/` y creará copias de seguridad `.bak` antes de actualizar cada configuración.

---

## Comandos y Banderas CLI

| Comando | Descripción |
| --- | --- |
| `edro-mcp install` | Descubre manifiestos, compila los MCPs seleccionados hacia `~/.mcp/bin/` y sincroniza perfiles en los clientes elegidos. |
| `edro-mcp sync` | Sincroniza perfiles en los clientes seleccionados (compilando el binario si aún no existe en `~/.mcp/bin/` o si se pasa `--build`). |
| `edro-mcp build` | Compila e instala en `~/.mcp/bin/` todos los MCPs descubiertos o los filtrados por `--mcps`. |
| `edro-mcp list` | Muestra en tabla (o `--json`) los servidores MCP descubiertos, perfiles cargados y clientes detectados. |
| `edro-mcp doctor` | Verifica que los binarios en `~/.mcp/bin/` existan, valida las variables obligatorias de `profiles.json` y comprueba la sintaxis de las configuraciones de los clientes. |

### Ejemplos No Interactivos (Automatización / Scripts)

```sh
# Listar servidores MCP descubiertos, variables requeridas y estado de clientes
go run ./cmd/edro-mcp list

# Previsualizar cambios (--dry-run) sin modificar archivos en disco (oculta secretos resueltos)
go run ./cmd/edro-mcp sync --clients antigravity,opencode,codex --profiles mysql-local,postgres-local --dry-run

# Sincronizar todos los perfiles habilitados en clientes específicos
go run ./cmd/edro-mcp sync --clients opencode,codex --profiles all

# Compilar únicamente los binarios en ~/.mcp/bin/
go run ./cmd/edro-mcp build --all

# Verificar el estado de salud de binarios, perfiles y archivos de configuración
go run ./cmd/edro-mcp doctor
```

---

## Servidores MCP Incluidos

| Servidor | Directorio | Runtime | Descripción |
| --- | --- | --- | --- |
| **MySQL MCP** | [`database/mysql`](database/mysql/README.md) | Go (`mysql-mcp`) | Consultas y operaciones SQL en MySQL con control granular de permisos por variable `DATABASE_PERMISSIONS`. |
| **PostgreSQL MCP** | [`database/postgresql`](database/postgresql/README.md) | Go (`postgresql-mcp`) | Consultas y operaciones SQL en PostgreSQL con control granular de permisos por variable `DATABASE_PERMISSIONS`. |

---

## Cómo Agregar un Nuevo Servidor MCP

1. Crea una carpeta para tu servidor (por ejemplo, `database/redis` o `tools/vector-search`).
2. Añade un archivo `mcp.manifest.json` siguiendo el contrato en [`agents/specs/mcp-manifest.example.json`](agents/specs/mcp-manifest.example.json).
3. Ejecuta `go run ./cmd/edro-mcp list` para verificar que `edro-mcp` lo descubre automáticamente.
