# PostgreSQL MCP

Servidor MCP local (`stdio`) para consultar y operar una base de datos PostgreSQL. Expone `list_tables`, `describe_table` y `select` por defecto; las operaciones de escritura se habilitan explícitamente mediante `DATABASE_PERMISSIONS`.

## Instalación y Configuración Recomendada (`edro-mcp`)

Desde la raíz del repositorio `edro-mcp`, utiliza el instalador unificado para compilar el binario en `~/.mcp/bin/postgresql-mcp` (o `postgresql-mcp.exe` en Windows) y sincronizar tus perfiles (`postgres-local`, etc.) en **Antigravity**, **OpenCode**, **Codex**, **Claude Code** o **Cursor**:

```sh
# Modo interactivo (te preguntará qué clientes y perfiles deseas usar)
go -C installer run . install

# Modo directo por banderas
go -C installer run . sync --clients opencode,antigravity,codex --profiles postgres-local
```

### Ejemplo de Perfil en `profiles.json`

```json
{
  "schemaVersion": "1.0",
  "profiles": {
    "postgres-local": {
      "mcp": "postgresql",
      "description": "Base de datos PostgreSQL local",
      "enabled": true,
      "env": {
        "DATABASE_HOST": "127.0.0.1",
        "DATABASE_PORT": "5432",
        "DATABASE_NAME": "mi_base_de_datos",
        "DATABASE_USERNAME": "mi_usuario",
        "DATABASE_PASSWORD": "{env:POSTGRES_LOCAL_PASSWORD}",
        "DATABASE_PERMISSIONS": "SELECT"
      }
    }
  }
}
```

> Exporta `POSTGRES_LOCAL_PASSWORD` en tu entorno antes de sincronizar o iniciar tu cliente de IA. Nunca guardes contraseñas reales en archivos seguidos por Git.

## Compilación Manual (Opcional)

Si deseas compilar únicamente este módulo desde `servers/database/postgresql`:

```sh
go build -o ~/.mcp/bin/postgresql-mcp .
```

## Variables de Entorno

| Variable | Descripción | Obligatoria |
| --- | --- | --- |
| `DATABASE_HOST` | Host de PostgreSQL, accesible desde donde corre el MCP. | Sí |
| `DATABASE_PORT` | Puerto de PostgreSQL (por ejemplo, `5432`). | Sí |
| `DATABASE_NAME` | Base de datos a utilizar. | Sí |
| `DATABASE_USERNAME` | Usuario de PostgreSQL. | Sí |
| `DATABASE_PASSWORD` | Contraseña de ese usuario (se recomienda `{env:VAR_NAME}`). | Sí |
| `DATABASE_PERMISSIONS` | Operaciones habilitadas, separadas por comas: `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `CREATE`, `ALTER`. Por defecto, `SELECT`. | No |

## Herramientas

| Herramienta | Permiso | Descripción |
| --- | --- | --- |
| `list_tables` | `SELECT` | Lista las tablas de la base de datos. |
| `describe_table` | `SELECT` | Muestra las columnas y sus características. |
| `select` | `SELECT` | Consulta filas con filtros de igualdad y límite. |
| `insert` | `INSERT` | Inserta una fila; devuelve `rows_affected`, no el ID generado. |
| `update` | `UPDATE` | Actualiza filas con un filtro obligatorio. |
| `delete` | `DELETE` | Elimina filas con un filtro obligatorio. |
| `create_table` | `CREATE` | Crea una tabla con columnas de tipos permitidos. |
| `alter_table` | `ALTER` | Añade una columna a una tabla existente. |
