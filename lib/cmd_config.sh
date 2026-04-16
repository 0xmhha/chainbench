#!/usr/bin/env bash
# lib/cmd_config.sh - 'chainbench config' subcommand handler
# Manages machine-local overlay (state/local-config.yaml), git-ignored.
# Subcommands: get, set, unset, list

# Guard against double-sourcing
[[ -n "${_CB_CMD_CONFIG_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_CONFIG_SH_LOADED=1

_CB_LIB_DIR="${_CB_LIB_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
# common.sh may already be sourced by the caller
[[ "$(type -t log_info 2>/dev/null)" == "function" ]] || source "${_CB_LIB_DIR}/common.sh"

readonly _CB_CONFIG_OVERLAY="${CHAINBENCH_DIR}/state/local-config.yaml"
readonly _CB_CONFIG_FIELD_PATTERN='^[a-zA-Z0-9_][a-zA-Z0-9_.]*$'

_cb_config_usage() {
  cat >&2 <<'EOF'
Usage: chainbench config <subcommand> [args]

Subcommands:
  list                         Print the full local overlay
  get  <field>                 Read a field (dot-notation, e.g. chain.binary_path)
  set  <field> <value>         Write a field (JSON-parsed if valid, else string)
  unset <field>                Remove a field

The overlay is stored at state/local-config.yaml (git-ignored).
It is merged on top of the profile YAML during profile loading.
EOF
}

_cb_config_validate_field() {
  local field="$1"
  if [[ -z "$field" ]]; then
    log_error "Field path must not be empty"
    return 1
  fi
  if [[ "$field" == *..* ]]; then
    log_error "Field path must not contain '..': $field"
    return 1
  fi
  if ! [[ "$field" =~ $_CB_CONFIG_FIELD_PATTERN ]]; then
    log_error "Invalid field path: $field (must match ${_CB_CONFIG_FIELD_PATTERN})"
    return 1
  fi
  return 0
}

_cb_config_list() {
  if [[ ! -f "$_CB_CONFIG_OVERLAY" ]]; then
    echo "(empty)"
    return 0
  fi
  cat "$_CB_CONFIG_OVERLAY"
}

_cb_config_get() {
  local field="${1:?config get requires a field}"
  _cb_config_validate_field "$field" || return 1

  if [[ ! -f "$_CB_CONFIG_OVERLAY" ]]; then
    echo "(not found)"
    return 1
  fi

  python3 - "$_CB_CONFIG_OVERLAY" "$field" <<'PYEOF'
import sys, json

overlay_path = sys.argv[1]
field = sys.argv[2]

try:
    import yaml
    with open(overlay_path) as fh:
        data = yaml.safe_load(fh) or {}
except ImportError:
    # Fallback: simple parse
    with open(overlay_path) as fh:
        content = fh.read()
    if not content.strip():
        data = {}
    else:
        import json as j
        # Try JSON first (our atomic write produces YAML, but just in case)
        try:
            data = j.loads(content)
        except Exception:
            data = {}

parts = field.split('.')
node = data
try:
    for p in parts:
        node = node[p]
    if isinstance(node, (dict, list)):
        print(json.dumps(node))
    elif node is None:
        print("null")
    else:
        print(node)
except (KeyError, TypeError):
    print("(not found)")
    sys.exit(1)
PYEOF
}

_cb_config_set() {
  local field="${1:?config set requires a field}"
  local value="${2:?config set requires a value}"
  _cb_config_validate_field "$field" || return 1

  mkdir -p "$(dirname "$_CB_CONFIG_OVERLAY")"

  python3 - "$_CB_CONFIG_OVERLAY" "$field" "$value" <<'PYEOF'
import sys, json, os, tempfile

overlay_path = sys.argv[1]
field = sys.argv[2]
raw_value = sys.argv[3]

# Parse value: try JSON first, then treat as string
try:
    value = json.loads(raw_value)
except (json.JSONDecodeError, ValueError):
    value = raw_value

# Load existing overlay
data = {}
if os.path.isfile(overlay_path):
    with open(overlay_path) as fh:
        content = fh.read()
    if content.strip():
        try:
            import yaml
            data = yaml.safe_load(content) or {}
        except ImportError:
            try:
                data = json.loads(content)
            except (json.JSONDecodeError, ValueError):
                data = {}

# Set nested field
parts = field.split('.')
node = data
for p in parts[:-1]:
    if p not in node or not isinstance(node[p], dict):
        node[p] = {}
    node = node[p]
node[parts[-1]] = value

# Write atomically: temp file + rename
try:
    import yaml
    dump_func = lambda d, fh: yaml.safe_dump(d, fh, default_flow_style=False, allow_unicode=True)
except ImportError:
    dump_func = lambda d, fh: json.dump(d, fh, indent=2, ensure_ascii=False) or fh.write('\n')

fd, tmp_path = tempfile.mkstemp(dir=os.path.dirname(overlay_path), suffix='.yaml.tmp')
try:
    with os.fdopen(fd, 'w') as fh:
        dump_func(data, fh)
    os.rename(tmp_path, overlay_path)
except Exception:
    os.unlink(tmp_path)
    raise
PYEOF
}

_cb_config_unset() {
  local field="${1:?config unset requires a field}"
  _cb_config_validate_field "$field" || return 1

  if [[ ! -f "$_CB_CONFIG_OVERLAY" ]]; then
    log_warn "No overlay file exists, nothing to unset"
    return 0
  fi

  python3 - "$_CB_CONFIG_OVERLAY" "$field" <<'PYEOF'
import sys, json, os, tempfile

overlay_path = sys.argv[1]
field = sys.argv[2]

# Load existing overlay
data = {}
with open(overlay_path) as fh:
    content = fh.read()
if content.strip():
    try:
        import yaml
        data = yaml.safe_load(content) or {}
    except ImportError:
        try:
            data = json.loads(content)
        except (json.JSONDecodeError, ValueError):
            data = {}

# Navigate and delete
parts = field.split('.')
node = data
parents = []
for p in parts[:-1]:
    if p not in node or not isinstance(node[p], dict):
        print(f"WARN: field '{field}' not found", file=sys.stderr)
        sys.exit(0)
    parents.append((node, p))
    node = node[p]

leaf = parts[-1]
if leaf not in node:
    print(f"WARN: field '{field}' not found", file=sys.stderr)
    sys.exit(0)

del node[leaf]

# Cascade-clean empty parent dicts
for parent, key in reversed(parents):
    if isinstance(parent[key], dict) and len(parent[key]) == 0:
        del parent[key]

# Write atomically
try:
    import yaml
    dump_func = lambda d, fh: yaml.safe_dump(d, fh, default_flow_style=False, allow_unicode=True)
except ImportError:
    dump_func = lambda d, fh: json.dump(d, fh, indent=2, ensure_ascii=False) or fh.write('\n')

fd, tmp_path = tempfile.mkstemp(dir=os.path.dirname(overlay_path), suffix='.yaml.tmp')
try:
    with os.fdopen(fd, 'w') as fh:
        dump_func(data, fh)
    os.rename(tmp_path, overlay_path)
except Exception:
    os.unlink(tmp_path)
    raise
PYEOF
}

_cb_config_main() {
  if [[ $# -lt 1 ]]; then
    _cb_config_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    list)   _cb_config_list ;;
    get)    _cb_config_get "$@" ;;
    set)    _cb_config_set "$@" ;;
    unset)  _cb_config_unset "$@" ;;
    --help|-h|help)
      _cb_config_usage
      return 0
      ;;
    *)
      log_error "Unknown config subcommand: '$subcmd'"
      _cb_config_usage
      return 1
      ;;
  esac
}

# Entry point when sourced by chainbench.sh dispatcher
_cb_config_main "$@"
