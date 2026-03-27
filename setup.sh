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

# ---- Generate MCP config ----------------------------------------------------

echo ""
echo "[2/3] Generating MCP configuration ..."

MCP_ENTRY="${MCP_DIR}/dist/index.js"

# Detect Claude Code settings location
CLAUDE_SETTINGS_DIR="${HOME}/.claude"
CLAUDE_SETTINGS_FILE="${CLAUDE_SETTINGS_DIR}/settings.local.json"

echo ""
echo "  Where do you want to register the MCP server?"
echo ""
echo "  1) Global  — ${CLAUDE_SETTINGS_FILE}"
echo "     (Available in all Claude Code sessions)"
echo ""
echo "  2) Project — .mcp.json in current directory"
echo "     (Available only when Claude Code is in this directory)"
echo ""
echo "  3) Skip    — I'll configure manually"
echo ""
printf "  Choice [1/2/3]: "
read -r _choice

case "${_choice}" in
  1)
    # Global registration
    mkdir -p "${CLAUDE_SETTINGS_DIR}"

    if [[ -f "${CLAUDE_SETTINGS_FILE}" ]]; then
      # Merge into existing settings
      python3 -c "
import json, sys

settings_file = '${CLAUDE_SETTINGS_FILE}'
try:
    with open(settings_file) as f:
        settings = json.load(f)
except (FileNotFoundError, json.JSONDecodeError):
    settings = {}

if 'mcpServers' not in settings:
    settings['mcpServers'] = {}

settings['mcpServers']['chainbench'] = {
    'command': 'node',
    'args': ['${MCP_ENTRY}'],
    'env': {
        'CHAINBENCH_DIR': '${CHAINBENCH_DIR}'
    }
}

with open(settings_file, 'w') as f:
    json.dump(settings, f, indent=2)
    f.write('\n')

print(f'  [OK] Registered in {settings_file}')
"
    else
      # Create new settings file
      cat > "${CLAUDE_SETTINGS_FILE}" <<JSONEOF
{
  "mcpServers": {
    "chainbench": {
      "command": "node",
      "args": ["${MCP_ENTRY}"],
      "env": {
        "CHAINBENCH_DIR": "${CHAINBENCH_DIR}"
      }
    }
  }
}
JSONEOF
      echo "  [OK] Created ${CLAUDE_SETTINGS_FILE}"
    fi
    ;;

  2)
    # Project-level registration
    _MCP_JSON="${CHAINBENCH_DIR}/.mcp.json"
    cat > "${_MCP_JSON}" <<JSONEOF
{
  "mcpServers": {
    "chainbench": {
      "command": "node",
      "args": ["${MCP_ENTRY}"],
      "env": {
        "CHAINBENCH_DIR": "${CHAINBENCH_DIR}"
      }
    }
  }
}
JSONEOF
    echo "  [OK] Created ${_MCP_JSON}"
    ;;

  3|*)
    echo "  Skipped. To configure manually, add to your MCP settings:"
    echo ""
    echo "  {"
    echo "    \"mcpServers\": {"
    echo "      \"chainbench\": {"
    echo "        \"command\": \"node\","
    echo "        \"args\": [\"${MCP_ENTRY}\"],"
    echo "        \"env\": {"
    echo "          \"CHAINBENCH_DIR\": \"${CHAINBENCH_DIR}\""
    echo "        }"
    echo "      }"
    echo "    }"
    echo "  }"
    ;;
esac

# ---- Summary -----------------------------------------------------------------

echo ""
echo "[3/3] Setup complete!"
echo ""
echo "========================================="
echo "  chainbench is ready"
echo "========================================="
echo ""
echo "  CLI:  ${CHAINBENCH_DIR}/chainbench.sh"
echo "  MCP:  ${MCP_ENTRY}"
echo ""
echo "  Next steps:"
echo "    1. Edit profiles/default.yaml to set chain.binary_path"
echo "    2. Run: ./chainbench.sh init && ./chainbench.sh start"
echo "    3. In Claude Code: restart or run /mcp to load the MCP server"
echo ""
echo "  Quick test:"
echo "    ./chainbench.sh --help"
echo ""
