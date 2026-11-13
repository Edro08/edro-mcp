# Design Specification — Instalador Multi-Agente y Gestor de Perfiles (`edro-mcp`)

## 1. Design Artifacts in `agents/specs/`
Artefactos de diseño, contratos de esquema y mockups almacenados junto a esta especificación mediante enlaces relativos:
- UI / CLI Interactive Mockup (HTML): [`cli-mockup.html`](cli-mockup.html)
- Contrato y Ejemplos de Manifiesto MCP (JSON): [`mcp-manifest.example.json`](mcp-manifest.example.json)
- Contrato y Ejemplo de Perfiles Multi-Instancia (JSON): [`profiles.example.json`](profiles.example.json)

---

## 2. Experience & Interface Breakdown

### 2.1 CLI / Console UX & Command Signatures

El binario CLI se invoca como `edro-mcp` (o `go run ./cmd/edro-mcp` desde la raíz del repositorio). Cuando se ejecuta sin argumentos o mediante `edro-mcp install`, inicia el asistente interactivo en terminal ("Preguntar qué uso").

#### Comandos y Subcomandos

| Comando | Descripción | Comportamiento por defecto |
| --- | --- | --- |
| `edro-mcp` / `edro-mcp install` | Descubre manifiestos, compila los MCPs seleccionados hacia `~/.mcp/bin/` e inyecta/actualiza los perfiles en los clientes elegidos. | Interactivo si no se pasan `--clients` ni `--profiles`/`--mcps`/`--all`. |
| `edro-mcp sync` | Sincroniza perfiles hacia los archivos de configuración de los clientes seleccionados (compilando el binario si aún no existe en `~/.mcp/bin/` o si se indica `--build`). | Interactivo si faltan banderas de selección; no interactivo cuando se especifican `--clients` y `--profiles` (o `--all`). |
| `edro-mcp build` | Compila e instala en `~/.mcp/bin/` todos los MCPs descubiertos o los filtrados por `--mcps` / `--all`, sin modificar configuraciones de clientes. | No interactivo si se pasa `--mcps` o `--all`. |
| `edro-mcp list` | Lista en formato tabular (o `--json`) los servidores MCP descubiertos (`mcp.manifest.json`), los perfiles cargados (`profiles.json`) y el estado de detección de clientes. | No interactivo (`stdout`). |
| `edro-mcp doctor` (alias `status`) | Verifica que los binarios en `~/.mcp/bin/` existan y sean ejecutables, valida que las variables obligatorias de los perfiles estén definidas y comprueba la sintaxis de las configuraciones de los clientes detectados. | No interactivo (`stdout`, código de salida `0` si todo es válido, `!= 0` en caso de errores). |

#### Banderas Globales y de Filtrado (CLI Flags)

| Bandera | Tipo / Ejemplo | Descripción |
| --- | --- | --- |
| `--clients` | `string` (CSV) | Lista separada por comas de clientes destino: `antigravity,opencode,codex,claude,cursor` o `all`. |
| `--profiles` | `string` (CSV) | Lista separada por comas de perfiles definidos en `profiles.json` (ej. `mysql-local,mysql-staging,postgres-local`) o `all`. |
| `--mcps` | `string` (CSV) | Lista separada por comas de IDs de servidores MCP (ej. `mysql,postgresql`). Si un MCP no tiene perfil en `profiles.json`, genera un perfil por defecto con el mismo `id` y valores plantilla del manifiesto. |
| `--all` | `bool` | Selecciona automáticamente todos los clientes detectados y todos los perfiles habilitados sin preguntar por `stdin`. |
| `--config` | `string` (ruta) | Ruta explícita al archivo de perfiles. Por defecto busca `./profiles.json`, luego `~/.mcp/profiles.json`, y como respaldo de plantilla `./profiles.example.json`. |
| `--bin-dir` | `string` (ruta) | Directorio destino de binarios compilados (por defecto `~/.mcp/bin`). |
| `--dry-run` | `bool` | Previsualiza los binarios que se compilarían y el resumen de cambios que se inyectarían en cada archivo de cliente sin escribir ni modificar ningún archivo en disco. |
| `--resolve-env` | `string` (`auto` \| `literal` \| `keep`) | Política de traducción para referencias `{env:VAR_NAME}` (por defecto `auto`: preserva `{env:VAR}` en OpenCode, usa `${VAR}` en Claude Code y expande desde el entorno actual del sistema en clientes que requieren valores literales como Antigravity, Codex y Cursor). |
| `--non-interactive` | `bool` | Fuerza la ejecución sin prompts interactivos; falla con código de error claro si faltan parámetros requeridos. |
| `--json` | `bool` | Devuelve la salida de `list`, `doctor` o `--dry-run` en formato JSON estructurado. |

---

### 2.2 Flujo Interactivo en Terminal ("Preguntar qué uso")

Cuando el usuario ejecuta `edro-mcp install` (o `edro-mcp sync`) sin banderas de selección:

1. **Inicialización de Perfiles**:
   - Si no existe `./profiles.json` ni `~/.mcp/profiles.json`, pero sí existe `./profiles.example.json`, el asistente ofrece crear `./profiles.json` a partir de la plantilla o usar las plantillas predeterminadas de cada `mcp.manifest.json`.
2. **Paso 1 — Selección Múltiple de Clientes de IA**:
   - Inspecciona el directorio del usuario (`~`) para detectar qué clientes ya están instalados o configurados y los preselecciona por defecto (`[x]`).
   - El usuario puede marcar o desmarcar cualquier cliente (incluso si el archivo aún no existe, en cuyo caso `edro-mcp` creará el directorio padre y el archivo inicial).
3. **Paso 2 — Selección Múltiple de Perfiles / MCPs**:
   - Muestra todos los perfiles disponibles en `profiles.json` indicando a qué servidor MCP pertenecen (`mcp: mysql`, `mcp: postgresql`) y su resumen de conexión no sensible (`host:port/db`).
4. **Paso 3 — Validación de Variables de Entorno**:
   - Verifica que todas las variables marcadas como `required: true` en el manifiesto estén presentes en el perfil seleccionado.
   - Si un perfil usa `{env:VAR_NAME}` y uno de los clientes seleccionados requiere expansión en tiempo de sincronización (`antigravity`, `codex`, `cursor`) pero `VAR_NAME` no está definida en el entorno actual, muestra una advertencia clara u ofrece mantener el placeholder seguro sin abortar el resto de clientes.
5. **Paso 4 — Compilación Global (`~/.mcp/bin/`) e Inyección con Backup**:
   - Compila una sola vez cada MCP único requerido por los perfiles seleccionados (ej. si se eligen `mysql-local` y `mysql-staging`, `mysql` se compila una única vez hacia `~/.mcp/bin/mysql-mcp` o `~/.mcp/bin/mysql-mcp.exe` en Windows).
   - Antes de modificar cualquier archivo de configuración existente del cliente, crea una copia de seguridad `<archivo>.bak`.
   - Fusiona los perfiles seleccionados en el árbol de configuración del cliente preservando todos los demás ajustes y servidores MCP externos.

```text
$ edro-mcp install

╭──────────────────────────────────────────────────────────────╮
│  edro-mcp · Instalador Multi-Agente y Gestor de Perfiles     │
╰──────────────────────────────────────────────────────────────╯
• Manifiestos detectados: 2 (mysql, postgresql)
• Archivo de perfiles:   ./profiles.json (3 perfiles cargados)

? ¿Qué clientes/agentes de IA utilizas? [Espacio: marcar/desmarcar, Enter: confirmar]
  [x] Antigravity   (~/.gemini/config/mcp_config.json)     [detectado]
  [x] OpenCode      (~/.config/opencode/opencode.jsonc)    [detectado]
  [x] Codex         (~/.codex/config.toml)                 [detectado]
  [ ] Claude Code   (~/.claude.json)                       [no detectado, se creará]
  [ ] Cursor        (~/.cursor/mcp.json)                   [no detectado, se creará]

? ¿Qué perfiles de servidor MCP deseas compilar y sincronizar?
  [x] mysql-local     (mcp: mysql      · 127.0.0.1:3306/app_dev)
  [x] mysql-staging   (mcp: mysql      · staging-db.internal:3306/app_staging)
  [x] postgres-local  (mcp: postgresql · 127.0.0.1:5432/analytics_dev)

[1/2] Compilando e instalando binarios en ~/.mcp/bin/...
  ✔ mysql       -> ~/.mcp/bin/mysql-mcp
  ✔ postgresql  -> ~/.mcp/bin/postgresql-mcp

[2/2] Sincronizando perfiles en clientes seleccionados...
  ✔ Antigravity -> ~/.gemini/config/mcp_config.json (backup: mcp_config.json.bak)
  ✔ OpenCode    -> ~/.config/opencode/opencode.jsonc (backup: opencode.jsonc.bak)
  ✔ Codex       -> ~/.codex/config.toml (backup: config.toml.bak)

RESUMEN DE SINCRONIZACIÓN
CLIENTE      ARCHIVO                            PERFILES INYECTADOS                          ESTADO
Antigravity  ~/.gemini/config/mcp_config.json   mysql-local, mysql-staging, postgres-local   OK
OpenCode     ~/.config/opencode/opencode.jsonc  mysql-local, mysql-staging, postgres-local   OK
Codex        ~/.codex/config.toml               mysql-local, mysql-staging, postgres-local   OK
```

---

### 2.3 Contrato de `mcp.manifest.json` (Descubrimiento Agnóstico al Lenguaje)

Cada servidor MCP del repositorio declara un archivo `mcp.manifest.json` en su directorio raíz (por ejemplo, `database/mysql/mcp.manifest.json` y `database/postgresql/mcp.manifest.json`). Consulta el ejemplo completo en [`mcp-manifest.example.json`](mcp-manifest.example.json).

#### Estructura y Campos del Manifiesto

```json
{
  "schemaVersion": "1.0",
  "id": "mysql",
  "name": "MySQL MCP Server",
  "description": "Servidor MCP local (stdio) para consultar una base MySQL.",
  "runtime": "go",
  "build": {
    "type": "go",
    "source": ".",
    "binary": "mysql-mcp"
  },
  "run": {
    "command": "{bin_dir}/mysql-mcp{exe_ext}",
    "args": []
  },
  "env": [
    {
      "name": "DATABASE_HOST",
      "description": "Host de MySQL, accesible desde donde corre el MCP.",
      "required": true,
      "default": "127.0.0.1",
      "sensitive": false
    }
  ]
}
```

#### Estrategias de Compilación y Ejecución Soportadas (`runtime` / `build.type`)

1. **`go` (`build.type = "go"`)**:
   - Ejecuta `go build -o <bin_dir>/<binary><exe_ext> <source>` con `cmd.Dir` ubicado en el directorio donde reside `mcp.manifest.json`.
   - En Windows, `{exe_ext}` se resuelve automáticamente a `.exe`; en Linux/macOS se resuelve a cadena vacía `""`.
2. **`command` / Scripts / Python / Node (`build.type = "command"` o `"copy"`)**:
   - Permite a futuros MCPs escritos en Python, Node.js, Rust o binarios precompilados declarar un comando personalizado `build.command` (ej. `["uv", "build"]` o `["npm", "run", "build"]`) o copiar un wrapper ejecutable hacia `{bin_dir}/<binary>`.
   - Tokens interpolados dinámicamente en `run.command` y `run.args`:
     - `{bin_dir}`: Ruta absoluta resuelta en tiempo de ejecución hacia `~/.mcp/bin` (usando `os.UserHomeDir()`).
     - `{exe_ext}`: `.exe` en Windows, `""` en sistemas POSIX.
     - `{mcp_dir}`: Directorio del manifiesto (solo si un runtime interpretado necesita referenciar recursos estáticos, aunque por defecto todos los binarios compilados corren desde `{bin_dir}` sin depender del repositorio ni de `cwd`).

---

### 2.4 Contrato de `profiles.json` y Traducción de `{env:VAR_NAME}`

Los perfiles desacoplan la definición del servidor MCP (`mysql`, `postgresql`) de las instancias concretas de conexión (`mysql-local`, `mysql-staging`, `postgres-local`). Consulta el ejemplo completo en [`profiles.example.json`](profiles.example.json).

- **Archivos y Precedencia**:
  1. Bandera explícita `--config <ruta>`.
  2. `./profiles.json` en la raíz del repositorio (**ignorado en `.gitignore`** para proteger credenciales).
  3. `~/.mcp/profiles.json` en el directorio global del usuario.
  4. `./profiles.example.json` (plantilla versionada en Git sin secretos reales).
- **Esquema de `profiles.json`**:
  - `schemaVersion`: `"1.0"`
  - `profiles`: Mapa `<nombre_perfil> -> { mcp, description, enabled, args, env }`.
  - El `<nombre_perfil>` (ej. `mysql-local`) es la clave exacta con la que se registra el servidor MCP en cada cliente de IA, permitiendo coexistir múltiples conexiones al mismo motor de base de datos.

#### Traducción de Referencias de Variables de Entorno (`{env:VAR_NAME}`)

Para evitar almacenar contraseñas en texto plano dentro de `profiles.json`, cualquier valor dentro de `env` puede declararse con la sintaxis canónica `{env:VAR_NAME}` (por ejemplo, `"DATABASE_PASSWORD": "{env:MYSQL_LOCAL_PASSWORD}"`). Cada adaptador traduce esta expresión según las capacidades nativas del cliente:

| Cliente | Formato Nativo de Variables en Archivo | Traducción de `"{env:VAR_NAME}"` (modo `--resolve-env=auto`) |
| --- | --- | --- |
| **OpenCode** | Soporta interpolación nativa `{env:VAR_NAME}` en `opencode.jsonc`. | Se conserva literalmente `"{env:VAR_NAME}"` en el archivo `opencode.jsonc`. |
| **Claude Code** | Soporta expansión `${VAR_NAME}` en `.claude.json`. | Se traduce a `"${VAR_NAME}"` (o se resuelve desde `os.Getenv("VAR_NAME")` si se indica `--resolve-env=literal`). |
| **Antigravity** | Pasa los pares clave-valor de `env` literalmente al proceso `stdio`. | Si `VAR_NAME` existe en el entorno del sistema al ejecutar `edro-mcp`, inyecta el valor resuelto; si no está definida en el entorno, conserva `"{env:VAR_NAME}"` y emite un aviso en consola indicando qué variable exportar antes de sincronizar. |
| **Codex** | Pasa la tabla `[mcp_servers.<id>.env]` literalmente al proceso `stdio`. | Igual que Antigravity: resuelve `os.Getenv("VAR_NAME")` si está presente en el entorno; de lo contrario conserva `"{env:VAR_NAME}"` con advertencia. |
| **Cursor** | Pasa el objeto `env` literalmente al proceso `stdio`. | Igual que Antigravity y Codex: resuelve `os.Getenv("VAR_NAME")` si está presente en el entorno; de lo contrario conserva `"{env:VAR_NAME}"` con advertencia. |

> **Nota de Diseño para Binarios Propios (`mysql-mcp`, `postgresql-mcp`)**: En modo `--dry-run`, `edro-mcp` **nunca** imprime secretos resueltos en consola; en su lugar muestra `<resolved from $VAR_NAME>` o `{env:VAR_NAME}` enmascarado.

---

### 2.5 Contratos de Adaptadores de Salida por Cliente

Todos los adaptadores implementan una interfaz común de lectura, respaldo (`.bak`), fusión no destructiva (merge) y escritura atómica:
- **Regla de Oro de Preservación**: Ningún adaptador puede eliminar ni sobrescribir otras propiedades del archivo de configuración del usuario ni otros servidores MCP externos (por ejemplo, `chrome-devtools`, `filesystem`, `github`) que no formen parte del conjunto de perfiles sincronizados en la ejecución actual.
- **Sin `cwd` acoplado al repo**: Como el ejecutable reside en `~/.mcp/bin/<binary>`, ningún adaptador escribe la propiedad `cwd` apuntando al repositorio clonado.
- En todos los ejemplos siguientes, `<HOME>` representa el directorio de usuario resuelto dinámicamente en tiempo de ejecución (`os.UserHomeDir()`), nunca una ruta codificada en duro.

#### 1. Antigravity (`~/.gemini/config/mcp_config.json`)
- **Formato**: JSON estándar (indentación de 2 espacios).
- **Ruta del mapa de servidores**: Raíz `-> "mcpServers" -> "<profile_name>"`.
- **Estructura inyectada**:
```json
{
  "mcpServers": {
    "mysql-local": {
      "command": "<HOME>/.mcp/bin/mysql-mcp",
      "args": [],
      "env": {
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PORT": "3306",
        "DATABASE_NAME": "app_dev",
        "DATABASE_USERNAME": "dev_user",
        "DATABASE_PASSWORD": "<valor_o_referencia>",
        "DATABASE_PERMISSIONS": "SELECT,INSERT,UPDATE"
      }
    }
  }
}
```

#### 2. OpenCode (`~/.config/opencode/opencode.jsonc`)
- **Formato**: JSONC (JSON con soporte de comentarios `//` y `/* */`).
- **Estrategia de Merge JSONC**: El adaptador tolera comentarios y comas finales al parsear, preserva claves superiores existentes (`$schema`, `theme`, `model`, `provider`, etc.) y actualiza únicamente las entradas correspondientes dentro de `"mcp" -> "servers" -> "<profile_name>"`, preservando comentarios de cabecera o realizando edición quirúrgica / AST sobre el documento.
- **Ruta del mapa de servidores**: Raíz `-> "mcp" -> "servers" -> "<profile_name>"`.
- **Estructura inyectada**:
```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "mysql-local": {
        "type": "local",
        "command": ["<HOME>/.mcp/bin/mysql-mcp"],
        "environment": {
          "DATABASE_HOST": "127.0.0.1",
          "DATABASE_PORT": "3306",
          "DATABASE_NAME": "app_dev",
          "DATABASE_USERNAME": "dev_user",
          "DATABASE_PASSWORD": "{env:MYSQL_LOCAL_PASSWORD}",
          "DATABASE_PERMISSIONS": "SELECT,INSERT,UPDATE"
        }
      }
    }
  }
}
```

#### 3. Codex (`~/.codex/config.toml`)
- **Formato**: TOML.
- **Estrategia de Merge TOML**: Parsea el documento TOML existente (o actualiza por bloques de sección), preservando claves de nivel raíz (`model`, `approval_policy`, etc.) y otras tablas `[mcp_servers.<otro>]`, reemplazando o insertando únicamente `[mcp_servers.<profile_name>]` y `[mcp_servers.<profile_name>.env]`.
- **Ruta de tabla**: `[mcp_servers.<profile_name>]` y subtabla `[mcp_servers.<profile_name>.env]`.
- **Estructura inyectada**:
```toml
[mcp_servers.mysql-local]
command = "<HOME>/.mcp/bin/mysql-mcp"
args = []

[mcp_servers.mysql-local.env]
DATABASE_HOST = "127.0.0.1"
DATABASE_PORT = "3306"
DATABASE_NAME = "app_dev"
DATABASE_USERNAME = "dev_user"
DATABASE_PASSWORD = "<valor_o_referencia>"
DATABASE_PERMISSIONS = "SELECT,INSERT,UPDATE"
```

#### 4. Claude Code (`~/.claude.json`)
- **Formato**: JSON estándar (indentación de 2 espacios).
- **Ruta del mapa de servidores**: Raíz `-> "mcpServers" -> "<profile_name>"` (preservando todo el estado global de proyectos, permisos y preferencias de Claude Code almacenado en `~/.claude.json`).
- **Estructura inyectada**:
```json
{
  "mcpServers": {
    "mysql-local": {
      "type": "stdio",
      "command": "<HOME>/.mcp/bin/mysql-mcp",
      "args": [],
      "env": {
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PORT": "3306",
        "DATABASE_NAME": "app_dev",
        "DATABASE_USERNAME": "dev_user",
        "DATABASE_PASSWORD": "${MYSQL_LOCAL_PASSWORD}",
        "DATABASE_PERMISSIONS": "SELECT,INSERT,UPDATE"
      }
    }
  }
}
```

#### 5. Cursor (`~/.cursor/mcp.json`)
- **Formato**: JSON estándar (indentación de 2 espacios).
- **Ruta del mapa de servidores**: Raíz `-> "mcpServers" -> "<profile_name>"`.
- **Estructura inyectada**:
```json
{
  "mcpServers": {
    "mysql-local": {
      "command": "<HOME>/.mcp/bin/mysql-mcp",
      "args": [],
      "env": {
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PORT": "3306",
        "DATABASE_NAME": "app_dev",
        "DATABASE_USERNAME": "dev_user",
        "DATABASE_PASSWORD": "<valor_o_referencia>",
        "DATABASE_PERMISSIONS": "SELECT,INSERT,UPDATE"
      }
    }
  }
}
```

---

## 3. User Review & Decisions (`Gate 1`)
- **Date**: 2026-11-11
- **Approved by User**: Yes
- **Agreed Adjustments**:
  - Diseño y contratos aprobados en Gate 1. Carpeta raíz del repositorio renombrada a `edro-mcp`.
