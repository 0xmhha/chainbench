#!/usr/bin/env bash
# Scans lib/cmd_*.sh for references to the current chain binary name
# ("gstable"). Output lists each hit with file:line:context.
#
# Usage: scripts/inventory/scan-binary-hardcoding.sh [--json]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CMD_DIR="${ROOT}/lib"

format="text"
[[ "${1:-}" == "--json" ]] && format="json"

declare -a rows=()
while IFS= read -r hit; do
  rows+=("$hit")
done < <(grep -n 'gstable' "${CMD_DIR}"/cmd_*.sh || true)

if [[ "$format" == "json" ]]; then
  printf '[\n'
  first=1
  for r in "${rows[@]}"; do
    file="${r%%:*}"; rest="${r#*:}"
    lineno="${rest%%:*}"; text="${rest#*:}"
    # JSON-escape the text (basic: backslashes and quotes)
    esc="${text//\\/\\\\}"; esc="${esc//\"/\\\"}"
    [[ $first -eq 0 ]] && printf ',\n'
    first=0
    printf '  {"file":"%s","line":%s,"text":"%s"}' "${file#${ROOT}/}" "$lineno" "$esc"
  done
  printf '\n]\n'
else
  for r in "${rows[@]}"; do
    printf '%s\n' "${r#${ROOT}/}"
  done
fi
