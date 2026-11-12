# PostgreSQL MCP

Servidor MCP local (stdio) para consultar una base PostgreSQL. Expone `list_tables`, `describe_table` y `select` por defecto; las operaciones de escritura se habilitan por separado.

## Build

Desde `database/postgresql`:

```sh
go build -o postgresql-mcp.exe .
```

## Configuración en OpenCode

Para usar el MCP desde cualquier proyecto, agrega este servidor a la configuración global `~/.config/opencode/opencode.jsonc`. Ajusta la ruta absoluta de `cwd` a donde compilaste el ejecutable y los datos de conexión a tu PostgreSQL:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "postgresql": {
        "type": "local",
        "command": ["./postgresql-mcp.exe"],
        "cwd": "C:/ruta/al/proyecto/database/postgresql",
        "environment": {
          "DATABASE_HOST": "127.0.0.1",
          "DATABASE_PORT": "5432",
          "DATABASE_NAME": "mi_base_de_datos",
          "DATABASE_USERNAME": "mi_usuario",
          "DATABASE_PASSWORD": "{env:POSTGRES_MCP_PASSWORD}",
          "DATABASE_PERMISSIONS": "SELECT"
        }
      }
    }
  }
}
```

Define `POSTGRES_MCP_PASSWORD` en el entorno **antes de iniciar OpenCode**. No guardes contraseñas reales en el JSON.

| Variable | Descripción | Obligatoria |
| --- | --- | --- |
| `DATABASE_HOST` | Host de PostgreSQL, accesible desde donde corre el MCP. | Sí |
| `DATABASE_PORT` | Puerto de PostgreSQL (por ejemplo, `5432`). | Sí |
| `DATABASE_NAME` | Base de datos a utilizar. | Sí |
| `DATABASE_USERNAME` | Usuario de PostgreSQL. | Sí |
| `DATABASE_PASSWORD` | Contraseña de ese usuario. | Sí |
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
