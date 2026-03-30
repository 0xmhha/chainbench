#!/usr/bin/env bash
# lib/cmd_mcp.sh - Enable/disable chainbench MCP server for a project
# Usage:
#   chainbench mcp enable  [--target <dir>]   Register MCP server in target project
#   chainbench mcp disable [--target <dir>]   Remove MCP server from target project
#   chainbench mcp status  [--target <dir>]   Check if MCP is enabled for target project

_MCP_SUBCOMMAND="${1:-}"
shift 2>/dev/null || true

# ---- Parse options -----------------------------------------------------------

_MCP_TARGET_DIR="${PWD}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      _MCP_TARGET_DIR="${2:?--target requires a directory path}"; shift 2 ;;
    --target=*)
      _MCP_TARGET_DIR="${1#--target=}"; shift ;;
    *)
      log_error "Unknown option: $1"; exit 1 ;;
  esac
done

# Resolve to absolute path
_MCP_TARGET_DIR="$(cd "${_MCP_TARGET_DIR}" 2>/dev/null && pwd)" || {
  log_error "Target directory does not exist: ${_MCP_TARGET_DIR}"
  exit 1
}

_MCP_JSON_FILE="${_MCP_TARGET_DIR}/.mcp.json"
_MCP_ENTRY="${CHAINBENCH_DIR}/mcp-server/dist/index.js"

# ---- Helper ------------------------------------------------------------------

_mcp_show_usage() {
  cat <<'EOF'
Usage: chainbench mcp <subcommand> [options]

Subcommands:
  enable     Register chainbench MCP server in target project's .mcp.json
  disable    Remove chainbench MCP server from target project's .mcp.json
  status     Check if MCP is enabled for target project

Options:
  --target <dir>   Target project directory (default: current directory)

Examples:
  chainbench mcp enable
  chainbench mcp enable --target /path/to/my-chain-project
  chainbench mcp disable
  chainbench mcp status
EOF
}

# ---- Subcommands -------------------------------------------------------------

_mcp_ensure_built() {
  if [[ ! -f "${_MCP_ENTRY}" ]]; then
    log_error "MCP server not built. Run setup.sh first or:"
    log_error "  cd ${CHAINBENCH_DIR}/mcp-server && npm install && npm run build"
    exit 1
  fi
}

_mcp_enable() {
  _mcp_ensure_built

  local chainbench_dir_escaped
  chainbench_dir_escaped=$(python3 -c "import json; print(json.dumps('${CHAINBENCH_DIR}'))" | tr -d '"')

  if [[ -f "${_MCP_JSON_FILE}" ]]; then
    # Merge into existing .mcp.json
    python3 -c "
import json, sys

mcp_file = '${_MCP_JSON_FILE}'
with open(mcp_file) as f:
    config = json.load(f)

if 'mcpServers' not in config:
    config['mcpServers'] = {}

config['mcpServers']['chainbench'] = {
    'command': 'node',
    'args': ['${_MCP_ENTRY}'],
    'env': {
        'CHAINBENCH_DIR': '${CHAINBENCH_DIR}'
    }
}

with open(mcp_file, 'w') as f:
    json.dump(config, f, indent=2)
    f.write('\n')
" || {
      log_error "Failed to update ${_MCP_JSON_FILE}"
      exit 1
    }
    log_success "Updated ${_MCP_JSON_FILE} (merged chainbench MCP server)"
  else
    # Create new .mcp.json
    python3 -c "
import json

config = {
    'mcpServers': {
        'chainbench': {
            'command': 'node',
            'args': ['${_MCP_ENTRY}'],
            'env': {
                'CHAINBENCH_DIR': '${CHAINBENCH_DIR}'
            }
        }
    }
}

with open('${_MCP_JSON_FILE}', 'w') as f:
    json.dump(config, f, indent=2)
    f.write('\n')
" || {
      log_error "Failed to create ${_MCP_JSON_FILE}"
      exit 1
    }
    log_success "Created ${_MCP_JSON_FILE}"
  fi

  echo ""
  log_info "Restart Claude Code or run /mcp to load the MCP server"
}

_mcp_disable() {
  if [[ ! -f "${_MCP_JSON_FILE}" ]]; then
    log_info "No .mcp.json found in ${_MCP_TARGET_DIR} — nothing to do"
    return 0
  fi

  python3 -c "
import json, os

mcp_file = '${_MCP_JSON_FILE}'
with open(mcp_file) as f:
    config = json.load(f)

servers = config.get('mcpServers', {})
if 'chainbench' not in servers:
    print('chainbench MCP server not found in .mcp.json — nothing to do')
    raise SystemExit(0)

del servers['chainbench']

if not servers:
    # No other MCP servers remain — delete the file
    os.remove(mcp_file)
    print('DELETED')
else:
    config['mcpServers'] = servers
    with open(mcp_file, 'w') as f:
        json.dump(config, f, indent=2)
        f.write('\n')
    print('UPDATED')
" 2>/dev/null | while read -r result; do
    case "${result}" in
      DELETED)
        log_success "Removed ${_MCP_JSON_FILE} (no other MCP servers)" ;;
      UPDATED)
        log_success "Removed chainbench from ${_MCP_JSON_FILE} (other servers preserved)" ;;
      *)
        log_info "${result}" ;;
    esac
  done
}

_mcp_status() {
  if [[ ! -f "${_MCP_JSON_FILE}" ]]; then
    echo "  MCP:     disabled (no .mcp.json)"
    echo "  Target:  ${_MCP_TARGET_DIR}"
    return 0
  fi

  local has_chainbench
  has_chainbench=$(python3 -c "
import json
with open('${_MCP_JSON_FILE}') as f:
    config = json.load(f)
print('yes' if 'chainbench' in config.get('mcpServers', {}) else 'no')
" 2>/dev/null)

  if [[ "${has_chainbench}" == "yes" ]]; then
    echo "  MCP:     enabled"
    echo "  Config:  ${_MCP_JSON_FILE}"
    echo "  Server:  ${_MCP_ENTRY}"
    echo "  Target:  ${_MCP_TARGET_DIR}"
  else
    echo "  MCP:     disabled (chainbench not in .mcp.json)"
    echo "  Target:  ${_MCP_TARGET_DIR}"
  fi
}

# ---- Dispatch ----------------------------------------------------------------

case "${_MCP_SUBCOMMAND}" in
  enable)   _mcp_enable ;;
  disable)  _mcp_disable ;;
  status)   _mcp_status ;;
  --help|-h|"")
    _mcp_show_usage ;;
  *)
    log_error "Unknown mcp subcommand: ${_MCP_SUBCOMMAND}"
    echo ""
    _mcp_show_usage
    exit 1
    ;;
esac
