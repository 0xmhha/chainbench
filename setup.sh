#!/usr/bin/env bash
# setup.sh - One-command setup for chainbench + Claude Code MCP integration.
# Builds the Go binaries (CLI, MCP server, dashboard) and registers them on PATH.
set -euo pipefail

CHAINBENCH_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${CHAINBENCH_DIR}/bin"

echo "========================================="
echo "  chainbench setup"
echo "========================================="
echo ""

# ---- Check prerequisites ----------------------------------------------------

_check_cmd() {
  if ! command -v "$1" &>/dev/null; then
    echo "ERROR: $1 is required but not found."
    echo "  $2"
    exit 1
  fi
}

_check_cmd go      "Install Go 1.25+: https://go.dev/dl"
_check_cmd python3 "Install python3 (used by the reproduction scripts)"
_check_cmd curl    "Install curl"

echo "[OK] Prerequisites: go $(go version | awk '{print $3}'), python3, curl"

# ---- Build the Go binaries --------------------------------------------------

echo ""
echo "[1/3] Building chainbench binaries ..."

mkdir -p "${BIN_DIR}"
cd "${CHAINBENCH_DIR}"
go build -o "${BIN_DIR}/chainbench"     ./cmd/chainbench
go build -o "${BIN_DIR}/chainbench-mcp" ./cmd/chainbench-mcp
go build -o "${BIN_DIR}/chainbenchd"    ./cmd/chainbenchd

for b in chainbench chainbench-mcp chainbenchd; do
  if [[ ! -x "${BIN_DIR}/${b}" ]]; then
    echo "ERROR: build failed: ${BIN_DIR}/${b}"
    exit 1
  fi
done
echo "  [OK] Built: chainbench, chainbench-mcp, chainbenchd"

# ---- Register chainbench in PATH ---------------------------------------------

echo ""
echo "[2/3] Registering chainbench in \$PATH ..."

SYMLINK_DIR="/usr/local/bin"

_register_symlink() {
  local name="$1"
  local target="$2"
  local symlink="${SYMLINK_DIR}/${name}"

  if command -v "${name}" &>/dev/null; then
    local existing
    existing="$(command -v "${name}")"
    if [[ "$(readlink -f "${existing}" 2>/dev/null || realpath "${existing}" 2>/dev/null)" == "${target}" ]]; then
      echo "  [OK] Already registered: ${existing} → $(basename "${target}")"
      return 0
    else
      echo "  [WARN] '${name}' already exists at ${existing}"
      echo "         Skipping symlink creation. Remove it manually if needed."
      return 0
    fi
  fi

  if [[ -w "${SYMLINK_DIR}" ]]; then
    ln -sf "${target}" "${symlink}"
    echo "  [OK] Created symlink: ${symlink} → $(basename "${target}")"
  else
    echo "  Creating symlink requires sudo ..."
    sudo ln -sf "${target}" "${symlink}" && \
      echo "  [OK] Created symlink: ${symlink} → $(basename "${target}")" || \
      echo "  [WARN] Failed. Add manually: ln -s ${target} ${symlink}"
  fi
}

_register_symlink "chainbench"     "${BIN_DIR}/chainbench"
_register_symlink "chainbench-mcp" "${BIN_DIR}/chainbench-mcp"

# ---- Summary -----------------------------------------------------------------

echo ""
echo "[3/3] Setup complete!"
echo ""
echo "========================================="
echo "  chainbench is ready"
echo "========================================="
echo ""
echo "  CLI:  chainbench --help"
echo "  MCP:  chainbench-mcp        (Go stdio server; register in .mcp.json as \"command\": \"chainbench-mcp\")"
echo ""
echo "  Next steps:"
echo "    1. cd <your-chain-project>"
echo "    2. chainbench chains                       # list registered chains"
echo "    3. chainbench setup --chain stablenet --launch --binary /path/to/gstable"
echo "    4. chainbench verify --data-dir <dir>      # confirm block production"
echo ""
