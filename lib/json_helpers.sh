#!/usr/bin/env bash
# lib/json_helpers.sh - JSON CRUD utilities for chainbench
# Provides a unified interface for JSON read/write/transform operations.
# Uses jq when available, falls back to python3.
#
# Usage: source "${CHAINBENCH_DIR}/lib/json_helpers.sh"

# Guard against double-sourcing
[[ -n "${_CB_JSON_HELPERS_LOADED:-}" ]] && return 0
readonly _CB_JSON_HELPERS_LOADED=1

# ---- Backend detection -------------------------------------------------------

_CB_JSON_BACKEND="python3"
if command -v jq &>/dev/null; then
  _CB_JSON_BACKEND="jq"
fi

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

  case "$_CB_JSON_BACKEND" in
    jq)
      local jq_path
      jq_path=$(_cb_dot_to_jq "$path")
      local result
      result=$(jq -r "${jq_path} // empty" "$file" 2>/dev/null)
      if [[ -z "$result" ]]; then
        printf '%s\n' "$default"
      else
        printf '%s\n' "$result"
      fi
      ;;
    python3)
      python3 - "$file" "$path" "$default" <<'PYEOF'
import sys, json

file_path, dot_path, default_val = sys.argv[1], sys.argv[2], sys.argv[3]

try:
    with open(file_path) as fh:
        data = json.load(fh)
except Exception:
    print(default_val)
    sys.exit(1)

parts = [p for p in dot_path.split('.') if p]
node = data
try:
    for part in parts:
        if isinstance(node, dict):
            node = node[part]
        elif isinstance(node, list):
            node = node[int(part)]
        else:
            raise KeyError(part)
    if node is None:
        print(default_val)
    elif isinstance(node, bool):
        print(str(node).lower())
    elif isinstance(node, (dict, list)):
        print(json.dumps(node))
    else:
        print(node)
except (KeyError, IndexError, TypeError, ValueError):
    print(default_val)
PYEOF
      ;;
  esac
}

# cb_json_read_stdin <dot.path> [default]
# Read a value from JSON on stdin using dot-path notation.
# Example: echo '{"result":"0x1a"}' | cb_json_read_stdin "result"
cb_json_read_stdin() {
  local path="${1:?cb_json_read_stdin: dot.path required}"
  local default="${2:-}"

  local input
  input=$(cat)

  case "$_CB_JSON_BACKEND" in
    jq)
      local jq_path
      jq_path=$(_cb_dot_to_jq "$path")
      local result
      result=$(printf '%s' "$input" | jq -r "${jq_path} // empty" 2>/dev/null)
      if [[ -z "$result" ]]; then
        printf '%s\n' "$default"
      else
        printf '%s\n' "$result"
      fi
      ;;
    python3)
      python3 -c "
import sys, json
data = json.loads(sys.argv[1])
parts = [p for p in sys.argv[2].split('.') if p]
node = data
try:
    for p in parts:
        node = node[p] if isinstance(node, dict) else node[int(p)]
    if node is None:
        print(sys.argv[3])
    elif isinstance(node, bool):
        print(str(node).lower())
    elif isinstance(node, (dict, list)):
        print(json.dumps(node))
    else:
        print(node)
except (KeyError, IndexError, TypeError, ValueError):
    print(sys.argv[3])
" "$input" "$path" "$default"
      ;;
  esac
}

# cb_json_array_len <file> <dot.path>
# Return the length of a JSON array at the given path.
# Exit code: 0 (success), 1 (not an array or path not found)
cb_json_array_len() {
  local file="${1:?cb_json_array_len: file required}"
  local path="${2:?cb_json_array_len: dot.path required}"

  case "$_CB_JSON_BACKEND" in
    jq)
      local jq_path
      jq_path=$(_cb_dot_to_jq "$path")
      jq -r "${jq_path} | length" "$file" 2>/dev/null || { printf '0\n'; return 1; }
      ;;
    python3)
      python3 -c "
import sys, json
with open(sys.argv[1]) as f:
    data = json.load(f)
parts = [p for p in sys.argv[2].split('.') if p]
node = data
for p in parts:
    node = node[p] if isinstance(node, dict) else node[int(p)]
print(len(node) if isinstance(node, (list, dict)) else 0)
" "$file" "$path" 2>/dev/null || { printf '0\n'; return 1; }
      ;;
  esac
}

# ---- Write operations --------------------------------------------------------

# cb_json_write <file> <dot.path> <value>
# Write a value to a JSON file at the given dot-path (in-place).
# Value is auto-typed: integers, booleans, null are stored as JSON types.
# Exit code: 0 (success), 1 (failure)
cb_json_write() {
  local file="${1:?cb_json_write: file required}"
  local path="${2:?cb_json_write: dot.path required}"
  local value="$3"

  case "$_CB_JSON_BACKEND" in
    jq)
      local jq_path
      jq_path=$(_cb_dot_to_jq "$path")
      local jq_value
      jq_value=$(_cb_auto_type_jq "$value")
      local tmp
      tmp=$(mktemp)
      if jq "${jq_path} = ${jq_value}" "$file" > "$tmp" 2>/dev/null; then
        mv "$tmp" "$file"
      else
        rm -f "$tmp"
        return 1
      fi
      ;;
    python3)
      python3 - "$file" "$path" "$value" <<'PYEOF'
import sys, json, os

file_path = sys.argv[1]
dot_path  = sys.argv[2]
raw_value = sys.argv[3]

# Auto-type the value
def cast(v):
    if v == "null" or v == "":
        return None
    if v == "true":
        return True
    if v == "false":
        return False
    try:
        return int(v)
    except ValueError:
        pass
    try:
        return float(v)
    except ValueError:
        pass
    return v

with open(file_path) as fh:
    data = json.load(fh)

parts = [p for p in dot_path.split('.') if p]
node = data
for p in parts[:-1]:
    if isinstance(node, dict):
        node = node.setdefault(p, {})
    elif isinstance(node, list):
        node = node[int(p)]

last = parts[-1]
if isinstance(node, dict):
    node[last] = cast(raw_value)
elif isinstance(node, list):
    node[int(last)] = cast(raw_value)

with open(file_path, 'w') as fh:
    json.dump(data, fh, indent=2)
    fh.write('\n')
PYEOF
      ;;
  esac
}

# cb_json_merge <base_file> <override_json_string>
# Deep-merge override JSON into base file (in-place).
cb_json_merge() {
  local file="${1:?cb_json_merge: file required}"
  local override="${2:?cb_json_merge: override JSON required}"

  case "$_CB_JSON_BACKEND" in
    jq)
      local tmp
      tmp=$(mktemp)
      if jq --argjson override "$override" '. * $override' "$file" > "$tmp" 2>/dev/null; then
        mv "$tmp" "$file"
      else
        rm -f "$tmp"
        return 1
      fi
      ;;
    python3)
      python3 - "$file" "$override" <<'PYEOF'
import sys, json

def deep_merge(base, over):
    result = dict(base)
    for k, v in over.items():
        if k in result and isinstance(result[k], dict) and isinstance(v, dict):
            result[k] = deep_merge(result[k], v)
        else:
            result[k] = v
    return result

file_path = sys.argv[1]
override  = json.loads(sys.argv[2])

with open(file_path) as fh:
    base = json.load(fh)

merged = deep_merge(base, override)

with open(file_path, 'w') as fh:
    json.dump(merged, fh, indent=2)
    fh.write('\n')
PYEOF
      ;;
  esac
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

  case "$_CB_JSON_BACKEND" in
    jq)
      local has_error
      has_error=$(printf '%s' "$json" | jq -r 'has("error") and (.error != null)' 2>/dev/null)
      if [[ "$has_error" == "true" ]]; then
        local err_msg
        err_msg=$(printf '%s' "$json" | jq -r '.error.message // "unknown error"' 2>/dev/null)
        printf 'RPC error: %s\n' "$err_msg" >&2
        return 1
      fi
      printf '%s' "$json" | jq -r '.result // empty' 2>/dev/null
      ;;
    python3)
      python3 -c "
import sys, json
try:
    d = json.loads(sys.argv[1])
    if 'error' in d and d['error'] is not None:
        msg = d['error'].get('message', 'unknown error') if isinstance(d['error'], dict) else str(d['error'])
        print(f'RPC error: {msg}', file=sys.stderr)
        sys.exit(1)
    result = d.get('result')
    if result is None:
        pass
    elif isinstance(result, bool):
        print(str(result).lower())
    elif isinstance(result, (dict, list)):
        print(json.dumps(result))
    else:
        print(result)
except (json.JSONDecodeError, KeyError) as e:
    print(f'JSON parse error: {e}', file=sys.stderr)
    sys.exit(1)
" "$json"
      ;;
  esac
}

# cb_json_has_error <json_string>
# Check if a JSON-RPC response contains an error.
# Exit code: 0 (has error), 1 (no error)
cb_json_has_error() {
  local json="${1:?cb_json_has_error: json string required}"

  case "$_CB_JSON_BACKEND" in
    jq)
      printf '%s' "$json" | jq -e 'has("error") and (.error != null)' &>/dev/null
      ;;
    python3)
      python3 -c "
import sys, json
d = json.loads(sys.argv[1])
sys.exit(0 if ('error' in d and d['error'] is not None) else 1)
" "$json"
      ;;
  esac
}

# ---- Internal helpers --------------------------------------------------------

# _cb_dot_to_jq <dot.path>
# Convert dot-path to jq path expression.
# "nodes.1.http_port" → '.nodes["1"].http_port'
# "1.http_port"       → '.["1"].http_port'
# ""                  → '.'
_cb_dot_to_jq() {
  local path="$1"
  local result=""
  local IFS='.'

  for part in $path; do
    if [[ "$part" =~ ^[0-9]+$ ]]; then
      result="${result}[\"${part}\"]"
    else
      result="${result}.${part}"
    fi
  done

  # If the path started with a numeric key, result begins with '[' — prepend '.'
  # to form a valid jq expression. Empty path → '.' (identity).
  if [[ -z "$result" ]]; then
    result="."
  elif [[ "$result" == \[* ]]; then
    result=".${result}"
  fi

  printf '%s\n' "$result"
}

# _cb_auto_type_jq <value>
# Convert a string value to a jq-compatible literal.
_cb_auto_type_jq() {
  local val="$1"
  case "$val" in
    null|"")   printf 'null\n' ;;
    true)      printf 'true\n' ;;
    false)     printf 'false\n' ;;
    *)
      if [[ "$val" =~ ^-?[0-9]+$ ]]; then
        printf '%s\n' "$val"
      elif [[ "$val" =~ ^-?[0-9]*\.[0-9]+$ ]]; then
        printf '%s\n' "$val"
      else
        printf '"%s"\n' "$val"
      fi
      ;;
  esac
}
