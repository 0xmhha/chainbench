#!/usr/bin/env bash
# lib/cmd_spec.sh - Look up test specification by RT-ID
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Sub-subcommands:
#   spec lookup <id>         Look up a TC section from the spec document
#   spec config              Show current spec document path

[[ -n "${_CB_CMD_SPEC_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_SPEC_SH_LOADED=1

_CB_LIB_DIR="${_CB_LIB_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
[[ "$(type -t log_info 2>/dev/null)" == "function" ]] || source "${_CB_LIB_DIR}/common.sh"

# Spec document path: configurable via CHAINBENCH_SPEC_DOC env var or state/local-config.yaml
_cb_spec_resolve_path() {
  # 1. Environment variable
  if [[ -n "${CHAINBENCH_SPEC_DOC:-}" && -f "${CHAINBENCH_SPEC_DOC}" ]]; then
    printf '%s\n' "$CHAINBENCH_SPEC_DOC"
    return 0
  fi

  # 2. Local config overlay
  local overlay="${CHAINBENCH_DIR}/state/local-config.yaml"
  if [[ -f "$overlay" ]]; then
    local from_config
    from_config=$(python3 -c "
import sys
try:
    import yaml
    with open(sys.argv[1]) as f:
        d = yaml.safe_load(f) or {}
    print(d.get('spec', {}).get('doc_path', ''))
except Exception:
    print('')
" "$overlay" 2>/dev/null)
    if [[ -n "$from_config" && -f "$from_config" ]]; then
      printf '%s\n' "$from_config"
      return 0
    fi
  fi

  # 3. Not configured
  return 1
}

# _cb_spec_lookup <id> [doc_path]
# Extract a TC section from the spec document by RT-ID.
_cb_spec_lookup() {
  local id="${1:?spec lookup requires a test case ID (e.g. RT-A-2-01)}"
  local doc_path="${2:-}"

  if [[ -z "$doc_path" ]]; then
    doc_path="$(_cb_spec_resolve_path)" || {
      log_error "No spec document configured."
      log_error "Set with: chainbench config set spec.doc_path /path/to/regression-test-spec.md"
      log_error "Or: export CHAINBENCH_SPEC_DOC=/path/to/regression-test-spec.md"
      return 1
    }
  fi

  if [[ ! -f "$doc_path" ]]; then
    log_error "Spec document not found: $doc_path"
    return 1
  fi

  python3 - "$doc_path" "$id" <<'PYEOF'
import sys, json, re

doc_path = sys.argv[1]
target_id = sys.argv[2].strip()

with open(doc_path, encoding='utf-8') as f:
    content = f.read()

# Find the section: #### RT-<ID> — <title>
# Match with optional variant suffix (e.g. RT-A-1-01-A, RT-A-2-05a)
pattern = re.compile(
    r'^####\s+' + re.escape(target_id) + r'\b[^\n]*\n',
    re.MULTILINE
)

match = pattern.search(content)
if not match:
    # Try case-insensitive
    pattern_ci = re.compile(
        r'^####\s+' + re.escape(target_id) + r'\b[^\n]*\n',
        re.MULTILINE | re.IGNORECASE
    )
    match = pattern_ci.search(content)

if not match:
    print(json.dumps({"error": f"TC '{target_id}' not found in {doc_path}"}))
    sys.exit(1)

start = match.start()
header_line = match.group(0).strip()

# Extract title from header
title_match = re.match(r'^####\s+\S+\s*—\s*(.+)', header_line)
title = title_match.group(1).strip() if title_match else ""

# Find the end: next #### or --- or end of file
end_pattern = re.compile(r'^(?:####\s|---\s*$)', re.MULTILINE)
end_match = end_pattern.search(content, match.end())
end = end_match.start() if end_match else len(content)

excerpt = content[match.end():end].strip()

# Extract key fields from the table if present
fields = {}
for line in excerpt.split('\n'):
    m = re.match(r'\|\s*\*\*(\w[\w\s]*)\*\*\s*\|\s*(.+?)\s*\|', line)
    if m:
        fields[m.group(1).strip()] = m.group(2).strip()

result = {
    "id": target_id,
    "title": title,
    "source": f"{doc_path}:{content[:start].count(chr(10)) + 1}",
    "priority": fields.get("우선순위", ""),
    "type": fields.get("유형", ""),
    "prerequisites": fields.get("선행 TC", ""),
    "related": fields.get("연관 TC", ""),
    "code_refs": fields.get("코드 근거", ""),
    "excerpt": excerpt[:2000],  # Cap at 2000 chars for context efficiency
}

print(json.dumps(result, ensure_ascii=False, indent=2))
PYEOF
}

_cb_spec_usage() {
  cat >&2 <<'EOF'
Usage: chainbench spec <subcommand> [args]

Subcommands:
  lookup <id>     Look up a test case by RT-ID (e.g. RT-A-2-01)
  config          Show current spec document path

Configuration:
  chainbench config set spec.doc_path /path/to/regression-test-spec.md
  export CHAINBENCH_SPEC_DOC=/path/to/regression-test-spec.md
EOF
}

_cb_spec_main() {
  if [[ $# -lt 1 ]]; then
    _cb_spec_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    lookup)
      _cb_spec_lookup "$@"
      ;;
    config)
      local path
      path="$(_cb_spec_resolve_path 2>/dev/null)" || path="(not configured)"
      printf 'Spec document: %s\n' "$path"
      ;;
    --help|-h|help)
      _cb_spec_usage
      return 0
      ;;
    *)
      log_error "Unknown spec subcommand: '$subcmd'"
      _cb_spec_usage
      return 1
      ;;
  esac
}

_cb_spec_main "$@"
