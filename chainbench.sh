#!/usr/bin/env bash
# chainbench - Local chain sandbox test bench for go-stablenet
# Single entry point for all chain lifecycle, testing, and analysis operations.
set -euo pipefail

# ---- Resolve CHAINBENCH_DIR -------------------------------------------------
# Follow symlinks so the path points to the real install directory,
# not the symlink location (e.g. /usr/local/bin).
_SOURCE="$0"
while [[ -L "${_SOURCE}" ]]; do
  _DIR="$(cd "$(dirname "${_SOURCE}")" && pwd)"
  _SOURCE="$(readlink "${_SOURCE}")"
  # Handle relative symlink targets
  [[ "${_SOURCE}" != /* ]] && _SOURCE="${_DIR}/${_SOURCE}"
done
CHAINBENCH_DIR="$(cd "$(dirname "${_SOURCE}")" && pwd)"
unset _SOURCE _DIR
export CHAINBENCH_DIR

# ---- Parse global flags -----------------------------------------------------
CHAINBENCH_QUIET="${CHAINBENCH_QUIET:-0}"
CHAINBENCH_PROFILE="${CHAINBENCH_PROFILE:-default}"

_CB_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --quiet)
      CHAINBENCH_QUIET=1; shift ;;
    --profile)
      CHAINBENCH_PROFILE="${2:?--profile requires a value}"; shift 2 ;;
    --profile=*)
      CHAINBENCH_PROFILE="${1#--profile=}"; shift ;;
    --help|-h)
      _CB_SHOW_HELP=1; shift ;;
    *)
      _CB_ARGS+=("$1"); shift ;;
  esac
done

export CHAINBENCH_QUIET
export CHAINBENCH_PROFILE

# ---- Source common library ---------------------------------------------------
source "${CHAINBENCH_DIR}/lib/common.sh"

# ---- Help --------------------------------------------------------------------
_cb_show_usage() {
  cat <<'EOF'
chainbench - Local chain sandbox test bench for go-stablenet

Usage: chainbench <command> [options]

Commands:
  init       Initialize chain from profile (genesis + TOML + datadir)
  start      Start all nodes
  stop       Stop all nodes
  restart    Stop, clean, init, and start with same profile
  status     Show node status
  clean      Remove node data (keeps config/profiles)
  node       Control individual nodes (stop/start/log/rpc)
  test       Run built-in tests
  log        Analyze node logs (timeline/anomaly/search)
  profile    Manage profiles (list/show/create)
  report     Show test results
  remote     Manage remote chain RPC connections
  mcp        Enable/disable MCP server for a project
  uninstall  Remove chainbench installation

Global Options:
  --profile <name>   Profile to use for init (default: default)
  --quiet            Suppress decorative output
  --help             Show this help

Examples:
  chainbench init --profile default
  chainbench start
  chainbench test run basic/consensus
  chainbench node stop 3
  chainbench stop
  chainbench status
  chainbench remote add eth-main https://eth.llamarpc.com --type mainnet
  chainbench remote info eth-main --json
  chainbench test run remote --remote eth-main
EOF
}

# ---- Dispatch ----------------------------------------------------------------
if [[ "${_CB_SHOW_HELP:-0}" -eq 1 && ${#_CB_ARGS[@]} -eq 0 ]]; then
  _cb_show_usage
  exit 0
fi

if [[ ${#_CB_ARGS[@]} -eq 0 ]]; then
  _cb_show_usage
  exit 1
fi

_CB_SUBCOMMAND="${_CB_ARGS[0]}"
set -- "${_CB_ARGS[@]:1}"

_CB_CMD_FILE="${CHAINBENCH_DIR}/lib/cmd_${_CB_SUBCOMMAND}.sh"

# Dynamic command dispatch: any lib/cmd_<name>.sh is a valid subcommand
if [[ "${_CB_SUBCOMMAND}" == "uninstall" ]]; then
  exec bash "${CHAINBENCH_DIR}/uninstall.sh"
elif [[ -f "${_CB_CMD_FILE}" ]]; then
  source "${_CB_CMD_FILE}"
else
  log_error "Unknown command: ${_CB_SUBCOMMAND}"
  echo ""
  echo "Available commands:"
  for _cmd_file in "${CHAINBENCH_DIR}/lib/cmd_"*.sh; do
    [[ -f "$_cmd_file" ]] || continue
    _cmd_name=$(basename "$_cmd_file" | sed 's/cmd_//;s/\.sh//')
    _cmd_desc=$(head -3 "$_cmd_file" | grep -m1 '^# Command:' | sed 's/# Command: [^ ]* — //' || true)
    printf "  %-12s %s\n" "$_cmd_name" "${_cmd_desc:-(no description)}"
  done
  echo ""
  echo "Run 'chainbench <command> --help' for details."
  exit 1
fi
