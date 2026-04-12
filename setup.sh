#!/usr/bin/env bash
# setup.sh - One-command setup for chainbench + Claude Code MCP integration
set -euo pipefail

CHAINBENCH_DIR="$(cd "$(dirname "$0")" && pwd)"
MCP_DIR="${CHAINBENCH_DIR}/mcp-server"

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

_check_cmd bash    "Install bash 4.0+"
_check_cmd python3 "Install python3"
_check_cmd curl    "Install curl"
_check_cmd node    "Install Node.js 18+: https://nodejs.org"
_check_cmd npm     "Install npm (comes with Node.js)"

NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
if (( NODE_VERSION < 18 )); then
  echo "ERROR: Node.js 18+ required (found v${NODE_VERSION})"
  exit 1
fi

echo "[OK] Prerequisites: bash, python3, curl, node $(node -v), npm $(npm -v)"

# ---- Build MCP server -------------------------------------------------------

echo ""
echo "[1/3] Building MCP server ..."

cd "${MCP_DIR}"
npm install --silent 2>&1 | tail -1
npm run build --silent 2>&1
cd "${CHAINBENCH_DIR}"

if [[ ! -f "${MCP_DIR}/dist/index.js" ]]; then
  echo "ERROR: MCP server build failed"
  exit 1
fi

echo "  [OK] MCP server built: ${MCP_DIR}/dist/index.js"

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

_register_symlink "chainbench"     "${CHAINBENCH_DIR}/chainbench.sh"
_register_symlink "chainbench-mcp" "${CHAINBENCH_DIR}/bin/chainbench-mcp"

# ---- Summary -----------------------------------------------------------------

echo ""
echo "[3/3] Setup complete!"
echo ""
echo "========================================="
echo "  chainbench is ready"
echo "========================================="
echo ""
echo "  CLI:  chainbench --help"
echo "  MCP:  ${MCP_DIR}/dist/index.js"
echo ""
echo "  Next steps:"
echo "    1. cd <your-chain-project>"
echo "    2. chainbench mcp enable          # register MCP for this project"
echo "    3. chainbench config set chain.binary_path /path/to/gstable"
echo "    4. chainbench init && chainbench start"
echo "    5. bash tests/unit/run.sh         # verify unit tests pass"
echo ""
