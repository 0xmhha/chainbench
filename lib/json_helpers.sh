#!/usr/bin/env bash
# lib/json_helpers.sh - JSON CRUD utilities for chainbench
# Thin shell wrappers over a single python backend (scripts/json_backend.py).
# python3 is already a hard requirement (profile parsing), so this needs no jq.
#
# Usage: source "${CHAINBENCH_DIR}/lib/json_helpers.sh"

# Guard against double-sourcing
[[ -n "${_CB_JSON_HELPERS_LOADED:-}" ]] && return 0
readonly _CB_JSON_HELPERS_LOADED=1

# Resolve scripts/ relative to this file so callers can source from anywhere.
_CB_JSON_SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../scripts" && pwd)"
readonly _CB_JSON_BACKEND_PY="${_CB_JSON_SCRIPTS_DIR}/json_backend.py"

# ---- Read operations ---------------------------------------------------------

# cb_json_read <file> <dot.path> [default]
# Read a value from a JSON file using dot-path notation.
# Example: cb_json_read state/pids.json "nodes.1.http_port" "8545"
# Output: value string. Prints default (or empty) if field not found.
# Exit code: 0 (success), 1 (file not found or parse failure)
cb_json_read() {
  local file="${1:?cb_json_read: file required}"
  local path="${2:?cb_json_read: dot.path required}"
  local default="${3:-}"

  if [[ ! -f "$file" ]]; then
    [[ -n "$default" ]] && printf '%s\n' "$default"
    return 1
  fi

  python3 "$_CB_JSON_BACKEND_PY" read "$file" "$path" "$default"
}

# cb_json_read_stdin <dot.path> [default]
# Read a value from JSON on stdin using dot-path notation.
# Example: echo '{"result":"0x1a"}' | cb_json_read_stdin "result"
cb_json_read_stdin() {
  local path="${1:?cb_json_read_stdin: dot.path required}"
  local default="${2:-}"

  python3 "$_CB_JSON_BACKEND_PY" read-stdin "$path" "$default"
}

# cb_json_array_len <file> <dot.path>
# Return the length of a JSON array at the given path.
# Exit code: 0 (success), 1 (not an array or path not found)
cb_json_array_len() {
  local file="${1:?cb_json_array_len: file required}"
  local path="${2:?cb_json_array_len: dot.path required}"

  python3 "$_CB_JSON_BACKEND_PY" array-len "$file" "$path" 2>/dev/null \
    || { printf '0\n'; return 1; }
}

# ---- Write operations --------------------------------------------------------

# cb_json_write <file> <dot.path> <value>
# Write a value to a JSON file at the given dot-path (atomic replace).
# Value is auto-typed: integers, booleans, null are stored as JSON types.
# Exit code: 0 (success), 1 (failure)
cb_json_write() {
  local file="${1:?cb_json_write: file required}"
  local path="${2:?cb_json_write: dot.path required}"
  local value="$3"

  python3 "$_CB_JSON_BACKEND_PY" write "$file" "$path" "$value"
}

# cb_json_merge <base_file> <override_json_string>
# Deep-merge override JSON into base file (atomic replace).
cb_json_merge() {
  local file="${1:?cb_json_merge: file required}"
  local override="${2:?cb_json_merge: override JSON required}"

  python3 "$_CB_JSON_BACKEND_PY" merge "$file" "$override"
}

# ---- Conversion utilities ----------------------------------------------------

# cb_hex_to_dec <hex_string>
# Convert hex to decimal. Supports "0x1a" and "1a" formats.
# Output: decimal string. Exit code: 0 (success), 1 (invalid hex)
cb_hex_to_dec() {
  local hex="${1:?cb_hex_to_dec: hex string required}"
  hex="${hex#0x}"
  hex="${hex#0X}"

  # Try pure bash first (faster for small values)
  if [[ "$hex" =~ ^[0-9a-fA-F]+$ ]] && (( ${#hex} <= 15 )); then
    printf '%d\n' "0x${hex}" 2>/dev/null && return 0
  fi

  # Fall back to python3 for large values
  python3 -c "print(int('${hex}', 16))" 2>/dev/null || return 1
}

# cb_dec_to_hex <decimal_string>
# Convert decimal to hex with 0x prefix.
cb_dec_to_hex() {
  local dec="${1:?cb_dec_to_hex: decimal string required}"
  printf '0x%x\n' "$dec" 2>/dev/null || {
    python3 -c "print(hex(int('${dec}')))" 2>/dev/null || return 1
  }
}

# ---- RPC response parsing ----------------------------------------------------

# cb_json_get_result <json_string>
# Extract .result field from a JSON-RPC response.
# Prints error to stderr and returns 1 if response contains .error.
cb_json_get_result() {
  local json="${1:?cb_json_get_result: json string required}"

  python3 "$_CB_JSON_BACKEND_PY" get-result "$json"
}

# cb_json_has_error <json_string>
# Check if a JSON-RPC response contains an error.
# Exit code: 0 (has error), 1 (no error)
cb_json_has_error() {
  local json="${1:?cb_json_has_error: json string required}"

  python3 "$_CB_JSON_BACKEND_PY" has-error "$json"
}
