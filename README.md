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

## Servidores MCP Incluidos

| Servidor | Directorio | Runtime | Descripción |
| --- | --- | --- | --- |
| **MySQL MCP** | [`servers/database/mysql`](servers/database/mysql/README.md) | Go (`mysql-mcp`) | Consultas y operaciones SQL en MySQL con control granular de permisos por variable `DATABASE_PERMISSIONS`. |
| **PostgreSQL MCP** | [`servers/database/postgresql`](servers/database/postgresql/README.md) | Go (`postgresql-mcp`) | Consultas y operaciones SQL en PostgreSQL con control granular de permisos por variable `DATABASE_PERMISSIONS`. |


## Inicio Rápido

**Requisitos:** Tener instalados **Git** y **Go** (1.27+).

### 1. Clona el repositorio
```sh
git clone https://github.com/Edro08/edro-mcp.git
```

Entra a la carpeta del proyecto:
```sh
cd edro-mcp
```

### 2. Configura tus perfiles
Copia la plantilla [`profiles.example.json`](profiles.example.json) a `profiles.json` (ignorado por Git) y ajusta tus conexiones:

```sh
cp profiles.example.json profiles.json
```

> Usa `{env:NOMBRE_VARIABLE}` en `profiles.json` para no guardar contraseñas reales en el archivo.

### 3. Ejecuta el instalador
El asistente detectará tus clientes instalados, te preguntará cuáles quieres configurar y compilará los binarios en `~/.mcp/bin/`:

- **Windows (PowerShell):**
  ```powershell
  .\install.ps1
  ```
  Si Windows bloquea la ejecución de scripts:
  ```powershell
  powershell -ExecutionPolicy Bypass -File .\install.ps1
  ```
- **Linux / macOS (Bash):**
  ```bash
  ./install.sh
  ```

---

## Comandos CLI

Puedes pasar cualquier subcomando directamente a `.\install.ps1 <comando>` o `./install.sh <comando>` (o usar `edro-mcp <comando>` si `~/.mcp/bin` está en tu `PATH`):

| Comando | Qué hace |
| --- | --- |
| `install` *(por defecto)* | Asistente interactivo: compila los MCPs en `~/.mcp/bin/` y sincroniza perfiles en los clientes elegidos. |
| `sync` | Sincroniza perfiles en los clientes sin recompilar si el binario ya existe (acepta `--build` y `--dry-run`). |
| `build` | Solo compila e instala los binarios en `~/.mcp/bin/`. |
| `list` | Muestra los servidores MCP descubiertos, perfiles cargados y clientes detectados. |
| `doctor` | Verifica que los binarios existan, las variables obligatorias estén definidas y los archivos de clientes sean válidos. |

### Ejemplos sin interacción (scripts / automatización)

Ver servidores, perfiles y clientes detectados:
```sh
./install.sh list
```

Previsualizar cambios sin modificar archivos (`--dry-run` oculta secretos):
```sh
./install.sh sync --clients antigravity,opencode --profiles mysql-local --dry-run
```

Sincronizar todos los perfiles en clientes específicos:
```sh
./install.sh sync --clients opencode,codex --profiles all
```

Validar el estado de binarios, perfiles y configuraciones:
```sh
./install.sh doctor
```
