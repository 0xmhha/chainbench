#!/usr/bin/env bash
# lib/common.sh - Shared utility functions for chainbench
# Source this file: source lib/common.sh

# ---- Color definitions -------------------------------------------------------

_CB_RED='\033[0;31m'
_CB_YELLOW='\033[0;33m'
_CB_GREEN='\033[0;32m'
_CB_CYAN='\033[0;36m'
_CB_RESET='\033[0m'

# ---- Logging -----------------------------------------------------------------

# log_info <message>
# Prints an informational message unless CHAINBENCH_QUIET=1.
log_info() {
  [[ "${CHAINBENCH_QUIET:-0}" == "1" ]] && return 0
  printf "${_CB_CYAN}[INFO]${_CB_RESET}  %s\n" "$*" >&2
}

# log_warn <message>
# Prints a warning message unless CHAINBENCH_QUIET=1.
log_warn() {
  [[ "${CHAINBENCH_QUIET:-0}" == "1" ]] && return 0
  printf "${_CB_YELLOW}[WARN]${_CB_RESET}  %s\n" "$*" >&2
}

# log_error <message>
# Always prints an error message (not suppressed by --quiet).
log_error() {
  printf "${_CB_RED}[ERROR]${_CB_RESET} %s\n" "$*" >&2
}

# log_success <message>
# Prints a success message unless CHAINBENCH_QUIET=1.
log_success() {
  [[ "${CHAINBENCH_QUIET:-0}" == "1" ]] && return 0
  printf "${_CB_GREEN}[OK]${_CB_RESET}    %s\n" "$*" >&2
}

# ---- Command checks ----------------------------------------------------------

# require_cmd <command> [error_message]
# Exits with code 1 if the command is not found in $PATH.
require_cmd() {
  local cmd="${1:?require_cmd: command name required}"
  local msg="${2:-Required command '$cmd' not found. Please install it.}"

  if ! command -v "$cmd" &>/dev/null; then
    log_error "$msg"
    return 1
  fi
  return 0
}

# ---- Binary resolution -------------------------------------------------------

# resolve_binary <binary_name> [explicit_path]
# Resolves the full path to a binary.
#   1. If explicit_path is non-empty and the file is executable, use it.
#   2. Otherwise search $PATH via `command -v`.
# Prints the resolved path to stdout. Returns 1 on failure.
resolve_binary() {
  local binary_name="${1:?resolve_binary: binary name required}"
  local explicit_path="${2:-}"

  if [[ -n "$explicit_path" ]]; then
    if [[ -x "$explicit_path" ]]; then
      printf '%s\n' "$explicit_path"
      return 0
    else
      log_warn "Explicit binary path '$explicit_path' is not executable, falling back to \$PATH"
    fi
  fi

  local resolved
  resolved="$(command -v "$binary_name" 2>/dev/null)"
  if [[ -z "$resolved" ]]; then
    log_error "Binary '$binary_name' not found in PATH"
    return 1
  fi

  printf '%s\n' "$resolved"
  return 0
}

# ---- Port calculation --------------------------------------------------------

# get_node_port <base_port> <node_index>
# Returns base_port + node_index (0-based).
# Example: get_node_port 30301 2  ->  30303
get_node_port() {
  local base="${1:?get_node_port: base port required}"
  local index="${2:?get_node_port: node index required}"

  if ! [[ "$base"  =~ ^[0-9]+$ ]] || ! [[ "$index" =~ ^[0-9]+$ ]]; then
    log_error "get_node_port: both base_port and node_index must be non-negative integers"
    return 1
  fi

  printf '%d\n' $(( base + index ))
}

# ---- Misc helpers ------------------------------------------------------------

# is_truthy <value>
# Returns 0 (true) when the value is "true", "1", "yes" (case-insensitive).
is_truthy() {
  case "${1,,}" in
    true|1|yes) return 0 ;;
    *) return 1 ;;
  esac
}
