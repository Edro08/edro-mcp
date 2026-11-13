#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${HOME}/.mcp/bin"
mkdir -p "${BIN_DIR}"

CLI_BIN="${BIN_DIR}/edro-mcp"

echo "[edro-mcp] Compilando CLI en ${CLI_BIN}..."
(
  cd "${REPO_ROOT}/installer"
  go build -o "${CLI_BIN}" .
)
chmod +x "${CLI_BIN}"

cd "${REPO_ROOT}"
if [ "$#" -gt 0 ]; then
  exec "${CLI_BIN}" "$@"
else
  exec "${CLI_BIN}" install
fi
