#!/usr/bin/env bash
# lib/profile.sh - YAML profile parser and environment variable exporter
# Source this file: source lib/profile.sh
#
# Primary entry point: load_profile <profile_name>
# After a successful call all CHAINBENCH_* variables are exported in the
# current shell and CHAINBENCH_PROFILE_JSON points to a temporary JSON file
# containing the fully merged profile.

# Guard against double-sourcing
[[ -n "${_CHAINBENCH_PROFILE_SH_LOADED:-}" ]] && return 0
readonly _CHAINBENCH_PROFILE_SH_LOADED=1

# Source common utilities if not already loaded
_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${_CB_LIB_DIR}/common.sh"

# ---- Internal constants ------------------------------------------------------

readonly _CB_PROFILES_DIR="${_CB_LIB_DIR}/../profiles"

# ---- Python helper -----------------------------------------------------------
# _cb_python_merge_yaml <profile_path> [parent_path]
# Outputs a fully merged JSON document to stdout.
# Uses PyYAML when available, otherwise falls back to a simple parser.

_cb_python_merge_yaml() {
  local profile_path="${1:?_cb_python_merge_yaml: profile path required}"

  python3 - "$profile_path" "$_CB_PROFILES_DIR" <<'PYEOF'
import sys
import json
import os
import re

PROFILE_PATH  = sys.argv[1]
PROFILES_ROOT = sys.argv[2]


# --------------------------------------------------------------------------- #
# YAML loading                                                                 #
# --------------------------------------------------------------------------- #

def _simple_yaml_parse(text):
    """
    Minimal YAML parser for the subset used by chainbench profiles.

    Handles:
      - Indented block mappings (nested dicts)
      - Sequences introduced by '- value' lines
      - Quoted / unquoted scalar values
      - Inline comments (# ...)
      - null / true / false / integer / float scalars
      - Blank lines and pure-comment lines are ignored
    """
    lines = text.splitlines()
    # Annotate each non-empty, non-comment line with its indent level
    tokens = []
    for raw in lines:
        stripped = raw.rstrip()
        if not stripped or stripped.lstrip().startswith('#'):
            continue
        indent = len(stripped) - len(stripped.lstrip())
        content = stripped.lstrip()
        # Strip inline comment (outside quotes)
        content = _strip_inline_comment(content)
        tokens.append((indent, content))

    result, _ = _parse_mapping(tokens, 0, -1)
    return result


def _strip_inline_comment(s):
    """Remove trailing # comment that is not inside quotes."""
    in_single = False
    in_double = False
    for i, ch in enumerate(s):
        if ch == "'" and not in_double:
            in_single = not in_single
        elif ch == '"' and not in_single:
            in_double = not in_double
        elif ch == '#' and not in_single and not in_double:
            return s[:i].rstrip()
    return s


def _cast_scalar(raw):
    """Convert a scalar string to a Python native type."""
    raw = raw.strip()
    # Quoted strings
    if (raw.startswith('"') and raw.endswith('"')) or \
       (raw.startswith("'") and raw.endswith("'")):
        return raw[1:-1]
    lower = raw.lower()
    if lower in ('null', '~', ''):
        return None
    if lower == 'true':
        return True
    if lower == 'false':
        return False
    # Integer
    try:
        return int(raw)
    except ValueError:
        pass
    # Float
    try:
        return float(raw)
    except ValueError:
        pass
    return raw


def _parse_mapping(tokens, start, parent_indent):
    """
    Parse a YAML mapping (dict) starting at tokens[start].
    Returns (dict, next_index).
    Stops when a token with indent <= parent_indent is encountered.
    """
    result = {}
    i = start
    while i < len(tokens):
        indent, content = tokens[i]
        if indent <= parent_indent:
            break
        # Sequence item - should not appear at mapping level, skip
        if content.startswith('- ') or content == '-':
            break
        # Key: value  or  Key:
        if ':' not in content:
            i += 1
            continue
        colon_pos = content.index(':')
        key = content[:colon_pos].strip()
        rest = content[colon_pos + 1:].strip()

        if rest:
            # Inline value on same line
            result[key] = _cast_scalar(rest)
            i += 1
        else:
            # Value continues on next lines
            i += 1
            if i < len(tokens):
                next_indent, next_content = tokens[i]
                if next_indent > indent:
                    if next_content.startswith('- ') or next_content == '-':
                        # Sequence
                        seq, i = _parse_sequence(tokens, i, indent)
                        result[key] = seq
                    else:
                        # Nested mapping
                        sub, i = _parse_mapping(tokens, i, indent)
                        result[key] = sub
                else:
                    result[key] = None
            else:
                result[key] = None
    return result, i


def _parse_sequence(tokens, start, parent_indent):
    """
    Parse a YAML sequence (list) starting at tokens[start].
    Returns (list, next_index).
    """
    result = []
    i = start
    while i < len(tokens):
        indent, content = tokens[i]
        if indent <= parent_indent:
            break
        if content.startswith('- '):
            item_str = content[2:].strip()
            result.append(_cast_scalar(item_str))
            i += 1
        elif content == '-':
            result.append(None)
            i += 1
        else:
            break
    return result, i


def load_yaml_file(path):
    try:
        import yaml
        with open(path) as fh:
            return yaml.safe_load(fh) or {}
    except ImportError:
        with open(path) as fh:
            return _simple_yaml_parse(fh.read()) or {}


# --------------------------------------------------------------------------- #
# Deep merge                                                                   #
# --------------------------------------------------------------------------- #

def deep_merge(base, override):
    """Recursively merge override into base; override wins on conflicts."""
    result = dict(base)
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = value
    return result


# --------------------------------------------------------------------------- #
# Inheritance resolution                                                       #
# --------------------------------------------------------------------------- #

MAX_INHERIT_DEPTH = 10

def find_profile(name, profiles_root):
    """Search profiles/<name>.yaml then profiles/custom/<name>.yaml."""
    candidates = [
        os.path.join(profiles_root, f"{name}.yaml"),
        os.path.join(profiles_root, "custom", f"{name}.yaml"),
    ]
    for c in candidates:
        if os.path.isfile(c):
            return c
    return None


def load_with_inheritance(path, profiles_root, depth=0):
    if depth > MAX_INHERIT_DEPTH:
        raise RuntimeError(f"Inheritance depth exceeded ({MAX_INHERIT_DEPTH}), "
                           "possible circular reference")
    data = load_yaml_file(path)
    parent_name = data.get('inherits')
    if not parent_name:
        return data

    parent_path = find_profile(str(parent_name), profiles_root)
    if parent_path is None:
        raise FileNotFoundError(
            f"Parent profile '{parent_name}' not found in {profiles_root}"
        )

    parent_data = load_with_inheritance(parent_path, profiles_root, depth + 1)
    # Remove meta-only field before merging
    child = {k: v for k, v in data.items() if k != 'inherits'}
    return deep_merge(parent_data, child)


# --------------------------------------------------------------------------- #
# Main                                                                         #
# --------------------------------------------------------------------------- #

try:
    merged = load_with_inheritance(PROFILE_PATH, PROFILES_ROOT)
    print(json.dumps(merged, ensure_ascii=False))
except Exception as exc:
    print(f"ERROR: {exc}", file=sys.stderr)
    sys.exit(1)
PYEOF
}

# ---- Profile location --------------------------------------------------------

# _cb_find_profile_path <name>
# Prints the path to the YAML file. Returns 1 if not found.
_cb_find_profile_path() {
  local name="${1:?_cb_find_profile_path: profile name required}"

  local candidates=(
    "${_CB_PROFILES_DIR}/${name}.yaml"
    "${_CB_PROFILES_DIR}/custom/${name}.yaml"
  )

  local p
  for p in "${candidates[@]}"; do
    if [[ -f "$p" ]]; then
      printf '%s\n' "$p"
      return 0
    fi
  done

  log_error "Profile '$name' not found. Searched:"
  for p in "${candidates[@]}"; do
    log_error "  $p"
  done
  return 1
}

# ---- JSON field extraction ---------------------------------------------------

# _cb_jq_get <json_file> <jq_filter> [default_value]
# Extracts a value from a JSON file using python3 (avoids jq dependency).
_cb_jq_get() {
  local json_file="${1:?_cb_jq_get: json file required}"
  local filter="${2:?_cb_jq_get: jq filter required}"
  local default="${3:-}"

  python3 - "$json_file" "$filter" "$default" <<'PYEOF'
import sys
import json

json_file = sys.argv[1]
filter_expr = sys.argv[2]   # e.g. ".nodes.validators"
default_val = sys.argv[3] if len(sys.argv) > 3 else ""

with open(json_file) as fh:
    data = json.load(fh)

# Navigate the dot-separated path (skip leading dot)
parts = [p for p in filter_expr.lstrip('.').split('.') if p]
node = data
try:
    for part in parts:
        node = node[part]
    if node is None:
        print(default_val)
    elif isinstance(node, bool):
        print(str(node).lower())
    elif isinstance(node, list):
        # Space-separated for bash arrays
        print(' '.join(str(x) for x in node))
    else:
        print(node)
except (KeyError, TypeError):
    print(default_val)
PYEOF
}

# ---- Validation --------------------------------------------------------------

# _cb_validate_profile_json <json_file>
# Checks that mandatory fields are present. Returns 1 on failure.
_cb_validate_profile_json() {
  local json_file="${1:?_cb_validate_profile_json: json file required}"
  local errors=0

  local required_fields=(
    ".chain.binary"
    ".nodes.validators"
    ".ports.base_p2p"
    ".ports.base_http"
  )

  local field
  for field in "${required_fields[@]}"; do
    local val
    val="$(_cb_jq_get "$json_file" "$field")"
    if [[ -z "$val" ]]; then
      log_error "Profile validation: required field '$field' is missing or empty"
      (( errors++ ))
    fi
  done

  [[ $errors -eq 0 ]]
}

# ---- Export ------------------------------------------------------------------

# _cb_export_profile_vars <json_file> <profile_name>
# Reads the merged JSON and exports CHAINBENCH_* environment variables.
_cb_export_profile_vars() {
  local json_file="${1:?_cb_export_profile_vars: json file required}"
  local profile_name="${2:?_cb_export_profile_vars: profile name required}"

  # Helper: read a field, fall back to default, then export as CHAINBENCH_VAR
  _cb_set_var() {
    local var_name="$1"
    local field="$2"
    local default="${3:-}"
    local value
    value="$(_cb_jq_get "$json_file" "$field" "$default")"
    export "${var_name}=${value}"
  }

  # Profile metadata
  export CHAINBENCH_PROFILE_NAME="$profile_name"
  export CHAINBENCH_PROFILE_JSON="$json_file"

  # chain.*
  _cb_set_var CHAINBENCH_BINARY            ".chain.binary"       "gstable"
  _cb_set_var CHAINBENCH_BINARY_PATH       ".chain.binary_path"  ""
  _cb_set_var CHAINBENCH_NETWORK_ID        ".chain.network_id"   ""
  _cb_set_var CHAINBENCH_CHAIN_ID          ".chain.chain_id"     ""

  # data.*
  _cb_set_var CHAINBENCH_DATA_DIR          ".data.directory"     "data"

  # genesis.*
  _cb_set_var CHAINBENCH_GENESIS_TEMPLATE  ".genesis.template"   ""

  # nodes.*
  _cb_set_var CHAINBENCH_VALIDATORS        ".nodes.validators"   "1"
  _cb_set_var CHAINBENCH_ENDPOINTS         ".nodes.endpoints"    "0"
  _cb_set_var CHAINBENCH_VERBOSITY         ".nodes.verbosity"    "3"
  _cb_set_var CHAINBENCH_EN_VERBOSITY      ".nodes.en_verbosity" "3"
  _cb_set_var CHAINBENCH_GCMODE            ".nodes.gcmode"       "full"
  _cb_set_var CHAINBENCH_CACHE             ".nodes.cache"        "1024"
  _cb_set_var CHAINBENCH_EXTRA_FLAGS       ".nodes.extra_flags"  ""

  # Computed total node count
  export CHAINBENCH_TOTAL_NODES=$(( CHAINBENCH_VALIDATORS + CHAINBENCH_ENDPOINTS ))

  # keys.*
  _cb_set_var CHAINBENCH_KEYS_MODE         ".keys.mode"          "static"
  _cb_set_var CHAINBENCH_KEYS_SOURCE       ".keys.source"        "keys/preset"

  # ports.*
  _cb_set_var CHAINBENCH_BASE_P2P          ".ports.base_p2p"     "30301"
  _cb_set_var CHAINBENCH_BASE_HTTP         ".ports.base_http"    "8501"
  _cb_set_var CHAINBENCH_BASE_WS           ".ports.base_ws"      "9501"
  _cb_set_var CHAINBENCH_BASE_AUTH         ".ports.base_auth"    "8551"
  _cb_set_var CHAINBENCH_BASE_METRICS      ".ports.base_metrics" "6061"

  # logging.*
  _cb_set_var CHAINBENCH_LOG_ROTATION      ".logging.rotation"   "true"
  _cb_set_var CHAINBENCH_LOG_MAX_SIZE      ".logging.max_size"   "10M"
  _cb_set_var CHAINBENCH_LOG_MAX_FILES     ".logging.max_files"  "5"
  _cb_set_var CHAINBENCH_LOG_DIR           ".logging.directory"  "data/logs"

  # tests.*
  _cb_set_var CHAINBENCH_AUTO_RUN_TESTS    ".tests.auto_run"     ""

  unset -f _cb_set_var
}

# ---- Temporary file management -----------------------------------------------

# _cb_cleanup_profile_json
# Removes the temporary JSON file if it was created by this session.
_cb_cleanup_profile_json() {
  if [[ -n "${CHAINBENCH_PROFILE_JSON:-}" && -f "${CHAINBENCH_PROFILE_JSON}" ]]; then
    rm -f "${CHAINBENCH_PROFILE_JSON}"
    unset CHAINBENCH_PROFILE_JSON
  fi
}

# Register cleanup unless caller opts out
if [[ "${CHAINBENCH_NO_CLEANUP:-0}" != "1" ]]; then
  trap _cb_cleanup_profile_json EXIT
fi

# ---- Public API --------------------------------------------------------------

# load_profile <profile_name>
# Main entry point. Resolves the profile (with inheritance), validates it,
# exports all CHAINBENCH_* variables, and sets CHAINBENCH_PROFILE_JSON.
load_profile() {
  local profile_name="${1:?load_profile: profile name required}"

  require_cmd python3 "python3 is required to parse YAML profiles" || return 1

  # Locate the YAML source file
  local profile_path
  profile_path="$(_cb_find_profile_path "$profile_name")" || return 1

  log_info "Loading profile '$profile_name' from $profile_path"

  # Create a temporary file to hold the merged JSON
  local json_tmp
  json_tmp="$(mktemp /tmp/chainbench-profile-XXXXXX.json)" || {
    log_error "Failed to create temporary file for profile JSON"
    return 1
  }

  # Parse YAML -> JSON (handles inheritance internally)
  local python_output
  if ! python_output="$(_cb_python_merge_yaml "$profile_path")"; then
    log_error "Failed to parse profile '$profile_name'"
    rm -f "$json_tmp"
    return 1
  fi

  # Validate that we got non-empty JSON
  if [[ -z "$python_output" ]]; then
    log_error "Profile parser produced no output for '$profile_name'"
    rm -f "$json_tmp"
    return 1
  fi

  printf '%s\n' "$python_output" > "$json_tmp"

  # Validate required fields
  if ! _cb_validate_profile_json "$json_tmp"; then
    rm -f "$json_tmp"
    return 1
  fi

  # Export environment variables
  _cb_export_profile_vars "$json_tmp" "$profile_name"

  log_success "Profile '$profile_name' loaded: ${CHAINBENCH_VALIDATORS}v + ${CHAINBENCH_ENDPOINTS}en (total ${CHAINBENCH_TOTAL_NODES} nodes)"
  return 0
}
