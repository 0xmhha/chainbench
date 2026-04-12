#!/usr/bin/env bash
# lib/common.sh - Shared utility functions for chainbench
# Source this file: source lib/common.sh

# ---- Color definitions (shared across all modules) ---------------------------
# Supports NO_COLOR (https://no-color.org/) and CHAINBENCH_NO_COLOR

if [[ -n "${NO_COLOR:-}" || -n "${CHAINBENCH_NO_COLOR:-}" ]] || ! [[ -t 2 ]]; then
  CB_RED="" CB_GREEN="" CB_YELLOW="" CB_CYAN="" CB_BOLD="" CB_RESET=""
else
  CB_RED='\033[0;31m'
  CB_GREEN='\033[0;32m'
  CB_YELLOW='\033[0;33m'
  CB_CYAN='\033[0;36m'
  CB_BOLD='\033[1m'
  CB_RESET='\033[0m'
fi

# Backward compatibility aliases (used by existing code)
_CB_RED="$CB_RED"
_CB_YELLOW="$CB_YELLOW"
_CB_GREEN="$CB_GREEN"
_CB_CYAN="$CB_CYAN"
_CB_RESET="$CB_RESET"

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
# Resolves the full path to a binary using the following priority:
#   1. explicit_path from profile (binary_path field)
#   2. Git repository root's build/bin/<binary> (works from any subdirectory)
#   3. $PWD/build/bin/<binary> (fallback for non-git projects)
#   4. $PATH lookup (last resort, with warning about potential conflicts)
# Prints the resolved absolute path to stdout. Returns 1 on failure.
resolve_binary() {
  local binary_name="${1:?resolve_binary: binary name required}"
  local explicit_path="${2:-}"

  # 1. Explicit path from profile
  if [[ -n "$explicit_path" ]]; then
    if [[ -x "$explicit_path" ]]; then
      printf '%s\n' "$explicit_path"
      return 0
    else
      log_warn "Explicit binary path '$explicit_path' is not executable, trying auto-detection"
    fi
  fi

  # 2. Auto-detect from git repository root's build/bin/
  local git_root
  git_root="$(git rev-parse --show-toplevel 2>/dev/null)" || true
  if [[ -n "$git_root" && -x "${git_root}/build/bin/${binary_name}" ]]; then
    log_info "Auto-detected binary at ${git_root}/build/bin/${binary_name}"
    printf '%s\n' "${git_root}/build/bin/${binary_name}"
    return 0
  fi

  # 3. Fallback: check $PWD/build/bin/ (for non-git projects)
  if [[ -x "${PWD}/build/bin/${binary_name}" ]]; then
    log_info "Auto-detected binary at ${PWD}/build/bin/${binary_name}"
    printf '%s\n' "${PWD}/build/bin/${binary_name}"
    return 0
  fi

  # 4. Last resort: $PATH lookup (with conflict warning)
  local resolved
  resolved="$(command -v "$binary_name" 2>/dev/null)"
  if [[ -n "$resolved" ]]; then
    log_warn "Using '$binary_name' from \$PATH ($resolved) — set binary_path in profile to avoid conflicts"
    printf '%s\n' "$resolved"
    return 0
  fi

  log_error "Binary '$binary_name' not found"
  log_error "Tried: explicit_path, git-root/build/bin/, \$PWD/build/bin/, \$PATH"
  return 1
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

# ---- Runtime override parser -------------------------------------------------

# _cb_parse_runtime_overrides <remaining_array_ref> [args...]
# Shared CLI flag parser for --binary-path and --logrot-path.
# Consumes recognized flags and exports the corresponding CHAINBENCH_* env vars.
# Unknown flags are appended to the nameref array for the caller to handle.
# Exits 2 on validation errors (missing value, non-absolute path).
_cb_parse_runtime_overrides() {
  local -n _remaining_ref="$1"
  shift

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --binary-path)
        if [[ $# -lt 2 || "$2" == --* ]]; then
          log_error "--binary-path requires a value (absolute path)"
          exit 2
        fi
        if [[ "$2" != /* ]]; then
          log_error "--binary-path must be an absolute path (got: '$2')"
          exit 2
        fi
        export CHAINBENCH_BINARY_PATH="$2"
        shift 2
        ;;
      --binary-path=*)
        local _val="${1#--binary-path=}"
        if [[ -z "$_val" ]]; then
          log_error "--binary-path requires a value (absolute path)"
          exit 2
        fi
        if [[ "$_val" != /* ]]; then
          log_error "--binary-path must be an absolute path (got: '$_val')"
          exit 2
        fi
        export CHAINBENCH_BINARY_PATH="$_val"
        shift
        ;;
      --logrot-path)
        if [[ $# -lt 2 || "$2" == --* ]]; then
          log_error "--logrot-path requires a value (absolute path)"
          exit 2
        fi
        if [[ "$2" != /* ]]; then
          log_error "--logrot-path must be an absolute path (got: '$2')"
          exit 2
        fi
        export CHAINBENCH_LOGROT_PATH="$2"
        shift 2
        ;;
      --logrot-path=*)
        local _val="${1#--logrot-path=}"
        if [[ -z "$_val" ]]; then
          log_error "--logrot-path requires a value (absolute path)"
          exit 2
        fi
        if [[ "$_val" != /* ]]; then
          log_error "--logrot-path must be an absolute path (got: '$_val')"
          exit 2
        fi
        export CHAINBENCH_LOGROT_PATH="$_val"
        shift
        ;;
      *)
        _remaining_ref+=("$1")
        shift
        ;;
    esac
  done
}

# ---- Logrot resolution ------------------------------------------------------

# resolve_logrot <binary_path> [explicit_logrot_path]
# Resolves the full path to the logrot binary using the priority chain:
#   1. Explicit logrot_path (from profile or CLI)
#   2. dirname(binary_path)/logrot
#   3. git-root/build/bin/logrot
#   4. Auto-build from source if cmd/logrot/main.go exists
#   5. $PATH lookup (with warning)
#   6. Empty string (non-fatal, caller falls back to plain >>)
# Prints the resolved absolute path to stdout, or empty string.
resolve_logrot() {
  local binary_path="${1:-}"
  local explicit_path="${2:-}"

  # 1. Explicit path from profile/CLI
  if [[ -n "$explicit_path" && -x "$explicit_path" ]]; then
    printf '%s\n' "$explicit_path"
    return 0
  elif [[ -n "$explicit_path" ]]; then
    log_warn "Explicit logrot path '$explicit_path' is not executable, trying auto-detection"
  fi

  # 2. Same directory as the chain binary
  if [[ -n "$binary_path" ]]; then
    local sibling
    sibling="$(dirname "$binary_path")/logrot"
    if [[ -x "$sibling" ]]; then
      printf '%s\n' "$sibling"
      return 0
    fi
  fi

  # 3. git-root/build/bin/logrot
  local git_root
  git_root="$(git rev-parse --show-toplevel 2>/dev/null)" || true
  if [[ -n "$git_root" && -x "${git_root}/build/bin/logrot" ]]; then
    printf '%s\n' "${git_root}/build/bin/logrot"
    return 0
  fi

  # 4. Auto-build from source
  if [[ -n "$git_root" ]]; then
    local built
    built="$(_cb_build_logrot_from_source "$git_root")" || true
    if [[ -n "$built" && -x "$built" ]]; then
      log_info "Built logrot from source: $built"
      printf '%s\n' "$built"
      return 0
    fi
  fi

  # 5. $PATH lookup
  local resolved
  resolved="$(command -v logrot 2>/dev/null)" || true
  if [[ -n "$resolved" ]]; then
    log_warn "Using logrot from \$PATH ($resolved)"
    printf '%s\n' "$resolved"
    return 0
  fi

  # 6. Not found
  printf ''
  return 0
}

# _cb_build_logrot_from_source <git_root>
# Attempts to build logrot from cmd/logrot/main.go if present.
# Prints the built binary path on success, empty on failure.
_cb_build_logrot_from_source() {
  local git_root="$1"
  local main_go="${git_root}/cmd/logrot/main.go"
  local out="${git_root}/build/bin/logrot"

  if [[ ! -f "$main_go" ]]; then
    return 0
  fi

  if ! command -v go &>/dev/null; then
    log_info "logrot source found but 'go' not available, skipping build"
    return 0
  fi

  local log_file="${CHAINBENCH_DIR:-${git_root}}/state/logrot-build.log"
  mkdir -p "$(dirname "$log_file")" "$(dirname "$out")"

  if (cd "$git_root" && go build -o "$out" ./cmd/logrot) >"$log_file" 2>&1; then
    printf '%s\n' "$out"
    return 0
  else
    log_warn "logrot build failed, see $log_file"
    return 0
  fi
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
