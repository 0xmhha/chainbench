#!/usr/bin/env bash
# lib/test_meta.sh - YAML-in-comment frontmatter parser for test scripts
#
# Usage: source lib/test_meta.sh
#        cb_parse_meta <script_path>
#
# Parses the block between "# ---chainbench-meta---" and "# ---end-meta---"
# in a test script, strips the "# " prefix, and outputs JSON to stdout.
# Returns "{}" if the block is absent or unparseable.

[[ -n "${_CB_TEST_META_LOADED:-}" ]] && return 0
readonly _CB_TEST_META_LOADED=1

# cb_parse_meta <script_path>
# Outputs a JSON object to stdout.
cb_parse_meta() {
  local script="${1:-}"

  if [[ -z "$script" || ! -f "$script" ]]; then
    echo "{}"
    return 0
  fi

  local yaml_block
  yaml_block=$(awk '/^# ---chainbench-meta---$/,/^# ---end-meta---$/' "$script" \
    | grep -v '^# ---.*meta---$' \
    | sed 's/^# //' \
    | sed 's/^#$//')

  if [[ -z "$yaml_block" ]]; then
    echo "{}"
    return 0
  fi

  printf '%s' "$yaml_block" | python3 -c "
import sys, json
try:
    import yaml
    data = yaml.safe_load(sys.stdin.read()) or {}
except ImportError:
    # Fallback: minimal key-value parse for simple cases
    data = {}
    for line in sys.stdin.read().splitlines():
        line = line.strip()
        if ':' in line and not line.startswith('-'):
            key, _, val = line.partition(':')
            key = key.strip()
            val = val.strip()
            if val:
                data[key] = val
except Exception:
    data = {}
print(json.dumps(data, ensure_ascii=False))
" 2>/dev/null || echo "{}"
}
