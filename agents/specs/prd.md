# Product Requirements — Instalador Multi-Agente y Gestor de Perfiles (`edro-mcp`)

## 1. Objective & Problem Statement
Actualmente, los servidores MCP de este repositorio (`database/mysql`, `database/postgresql` y futuros servidores) requieren compilación manual en cada directorio, rutas absolutas acopladas al directorio donde se clonó el repositorio (`cwd` + ejecutable local) y edición manual de los archivos de configuración de cada cliente de IA (OpenCode, Antigravity, Codex, Claude Code, Cursor, etc.), cada uno con un formato y esquema distinto (`JSON`, `JSONC`, `TOML`).

El objetivo de **`edro-mcp`** es proveer una herramienta CLI unificada, portable y agnóstica al lenguaje de implementación de cada MCP que automatice:
1. El descubrimiento y compilación/preparación de cualquier MCP del repositorio mediante manifiestos declarativos (`mcp.manifest.json`).
2. La instalación centralizada de binarios/artefactos ejecutables en `~/.mcp/bin/` para desacoplar la configuración de la ubicación del repositorio clonado.
3. La gestión de múltiples perfiles/instancias por servidor MCP (por ejemplo, `mysql-local`, `mysql-staging`, `postgres-dev`) sin exponer secretos en el control de versiones.
4. La selección interactiva (y por banderas CLI) de qué agentes/clientes de IA utiliza el usuario y la inyección/actualización directa y segura en sus archivos de configuración global sin sobrescribir ni corromper configuraciones existentes.

## 2. Scope
### In Scope
- **Descubrimiento basado en manifiestos (`mcp.manifest.json`)**:
  - Definición de un contrato declarativo por servidor MCP que especifique metadatos (`id`, `name`, `description`), comandos de compilación/preparación multiplataforma, comando de ejecución resultante y esquema de variables de entorno (obligatorias, opcionales y valores por defecto).
  - Incorporación de `mcp.manifest.json` en los servidores existentes (`database/mysql` y `database/postgresql`).
- **Almacenamiento global de binarios (`~/.mcp/bin/`)**:
  - Compilación e instalación de ejecutables en el directorio global del usuario (`~/.mcp/bin/` en sistemas POSIX y `%USERPROFILE%\.mcp\bin\` en Windows, resuelto dinámicamente).
- **Gestión de perfiles de conexión/instancia**:
  - Soporte para un archivo local ignorado por Git (`profiles.json` o `~/.mcp/profiles.json`) acompañado de una plantilla versionada (`profiles.example.json`).
  - Capacidad de definir una o múltiples instancias de un mismo MCP (ej. `mysql-local`, `mysql-prod`, `postgres-analytics`) con sus variables de entorno o referencias a variables de entorno del sistema.
- **Modo interactivo ("Preguntar qué uso") y modo no interactivo (CLI flags)**:
  - Flujo interactivo en terminal que consulte al usuario:
    1. Qué clientes/agentes de IA utiliza (selección múltiple).
    2. Qué servidores MCP o perfiles desea compilar y sincronizar.
  - Banderas CLI equivalentes (`--clients`, `--profiles`, `--mcps`, `--all`, `--dry-run`, `--list`) para ejecución automatizada o en scripts.
- **Adaptadores de configuración global multi-cliente (Inyección segura / Merge)**:
  - Soporte inicial para los clientes principales:
    - **Antigravity**: `~/.gemini/config/mcp_config.json` (JSON).
    - **OpenCode**: `~/.config/opencode/opencode.jsonc` (JSONC, preservando o tolerando comentarios y estructura existente).
    - **Codex**: `~/.codex/config.toml` (TOML, sección `[mcp_servers.<profile>]`).
    - **Claude Code / Claude Desktop**: `~/.claude.json` o configuración global equivalente (JSON).
    - **Cursor**: `~/.cursor/mcp.json` (JSON).
  - Creación de copias de seguridad (`.bak`) antes de modificar cualquier archivo de configuración del usuario y fusión no destructiva (respetando otros MCPs y ajustes ajenos a `edro-mcp`).
- **Documentación actualizada en español**:
  - `README.md` raíz del repositorio y actualización de `database/mysql/README.md` y `database/postgresql/README.md` reflejando el nuevo flujo con `edro-mcp`.

### Out of Scope
- Servidores MCP remotos basados en SSE/HTTP o gestión de despliegues en la nube (el alcance actual se centra en servidores MCP locales ejecutados vía `stdio`).
- Interfaz gráfica de usuario (GUI) o aplicación web; toda la interacción se realiza mediante la CLI.
- Almacenamiento o bóveda criptográfica propia de contraseñas; las credenciales sensibles deben seguir inyectándose mediante variables de entorno del sistema o referencias soportadas por cada cliente.
- Modificación de la lógica interna de negocio o herramientas SQL de los servidores `database/mysql` y `database/postgresql`.

## 3. User Flows & Functional Requirements
1. **REQ-01 (Descubrimiento Agnóstico de MCPs)**: El CLI debe escanear el repositorio buscando archivos `mcp.manifest.json` y listar todos los servidores MCP disponibles con su identificador, descripción, lenguaje/runtime y variables de entorno requeridas/opcionales.
2. **REQ-02 (Compilación e Instalación en `~/.mcp/bin/`)**: Para cada MCP seleccionado, el CLI debe ejecutar el paso de build definido en su manifiesto y colocar el binario resultante en `~/.mcp/bin/<binary_name>` (añadiendo `.exe` automáticamente en Windows cuando corresponda), creando el directorio si no existe.
3. **REQ-03 (Gestión de Perfiles Multi-Instancia)**: El usuario debe poder definir perfiles en un archivo de configuración de perfiles (`profiles.json` local ignorado por Git o en `~/.mcp/profiles.json`) a partir de `profiles.example.json`. Cada perfil vincula un nombre de instancia (ej. `mysql-local`) con un `mcp` base (ej. `mysql`) y un mapa de variables de entorno. Si no existe un perfil personalizado para un MCP seleccionado, el instalador debe permitir generar una entrada base con las variables por defecto/plantilla del manifiesto.
4. **REQ-04 (Selección Interactiva y por Flags)**:
   - Al ejecutar `edro-mcp install` o `edro-mcp sync` sin banderas de filtro, el CLI debe preguntar interactivamente al usuario qué agentes/clientes utiliza (mostrando cuáles tienen archivos/directorios detectados en el sistema) y qué perfiles/MCPs desea activar.
   - Al ejecutar con banderas (ej. `edro-mcp sync --clients opencode,antigravity,codex --profiles mysql-local,postgres-dev`), debe ejecutarse sin bloqueos interactivos.
5. **REQ-05 (Inyección Directa y No Destructiva en Configs Globales)**:
   - Para cada cliente seleccionado, el adaptador correspondiente debe traducir cada perfil al esquema exacto esperado por ese cliente (por ejemplo, `mcpServers` en Antigravity/Cursor/Claude, `mcp.servers` en OpenCode, `[mcp_servers.<name>]` en Codex) apuntando al ejecutable en `~/.mcp/bin/` sin requerir `cwd` atado al repositorio.
   - La escritura debe preservar cualquier otra clave de configuración del cliente y cualquier otro servidor MCP no administrado por la operación actual, generando un respaldo previo o escritura atómica segura.
6. **REQ-06 (Modo `--dry-run` y Validación)**: El usuario debe poder previsualizar los cambios que se aplicarían en las configuraciones globales (`--dry-run`) y validar que las variables obligatorias de cada perfil estén declaradas antes de escribir en disco.

## 4. Acceptance Criteria (Testable)
- [ ] **AC-01 (Descubrimiento de Manifiestos)**: Dado el repositorio con `database/mysql/mcp.manifest.json` y `database/postgresql/mcp.manifest.json`, cuando se ejecuta el comando de listado (`edro-mcp list`), entonces se muestran ambos servidores MCP junto con sus variables de entorno obligatorias y opcionales.
- [ ] **AC-02 (Build e Instalación Global)**: Dado un entorno con Go instalado, cuando se solicita compilar/instalar `mysql` y `postgresql`, entonces los binarios se generan e instalan en `~/.mcp/bin/` con permisos de ejecución y sin depender de rutas absolutas codificadas en duro.
- [ ] **AC-03 (Soporte Multi-Perfil)**: Dado un archivo de perfiles con dos instancias del mismo MCP (`mysql-local` y `mysql-staging`), cuando se sincronizan hacia un cliente soportado, entonces ambas entradas se registran de forma independiente bajo sus respectivos nombres de perfil y con sus respectivas variables de entorno.
- [ ] **AC-04 (Preservación de Configuración Existente)**: Dado un archivo de configuración de cliente existente (en formato JSON, JSONC o TOML) que ya contiene ajustes personalizados del usuario y un servidor MCP externo (ej. `chrome-devtools`), cuando `edro-mcp` inyecta o actualiza perfiles de este repositorio, entonces el servidor externo y los ajustes previos permanecen intactos y el archivo resultante es sintácticamente válido.
- [ ] **AC-05 (Flujo Interactivo y No Interactivo)**: Cuando el usuario ejecuta el instalador en modo interactivo, puede seleccionar clientes y perfiles mediante prompts claros en español; cuando lo ejecuta con `--clients` y `--profiles` (y opcionalmente `--dry-run`), la herramienta completa el flujo sin requerir entrada por teclado (`stdin`).
- [ ] **AC-06 (Portabilidad Estricta)**: Ningún archivo versionado del repositorio (código fuente, manifiestos, plantillas, tests o documentación) contiene rutas absolutas del host local (como letras de unidad o nombres de usuario del sistema).

## 5. Open Questions / Constraints
- **Restricción de Portabilidad**: Todas las rutas hacia el directorio del usuario (`~`) y binarios (`~/.mcp/bin/`) deben resolverse en tiempo de ejecución mediante las APIs estándar del sistema operativo (`os.UserHomeDir()`, `filepath.Join`).
- **Restricción de Seguridad**: `profiles.json` debe estar incluido explícitamente en `.gitignore` para evitar que credenciales o datos de conexión locales se suban accidentalmente al repositorio; solo `profiles.example.json` se versiona.
- **Formato de Variables Sensibles entre Clientes**: Mientras OpenCode soporta interpolación `{env:VAR_NAME}` dentro de su archivo, otros clientes (como Antigravity, Cursor o Codex) pasan los valores literales del mapa `env` al proceso hijo. El diseño (`design.md`) definirá cómo los adaptadores manejan referencias a variables de entorno del sistema o placeholders seguros según las capacidades de cada cliente.
