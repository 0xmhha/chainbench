#!/usr/bin/env bash
# Prints every adapter_* function defined in lib/adapters/*.sh with its file,
# line, and implementation status (real vs. stub).
#
# Usage: scripts/inventory/list-adapter-functions.sh [--json]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ADAPTERS_DIR="${ROOT}/lib/adapters"

format="text"
[[ "${1:-}" == "--json" ]] && format="json"

declare -a rows=()
for f in "${ADAPTERS_DIR}"/*.sh; do
  chain="$(basename "$f" .sh)"
  while IFS= read -r line; do
    lineno="${line%%:*}"
    fn="${line#*:}"
    fn="${fn%%(*}"
    fn="${fn## }"; fn="${fn%% }"
    status="real"
    if grep -q "_cb_${chain}_not_implemented" "$f" && \
       grep -q "^${fn}()[[:space:]]*{[[:space:]]*_cb_${chain}_not_implemented" "$f"; then
      status="stub"
    fi
    rows+=("${chain}|${fn}|${f#${ROOT}/}:${lineno}|${status}")
  done < <(grep -n "^adapter_[a-zA-Z_]*()" "$f" || true)
done

if [[ "$format" == "json" ]]; then
  printf '[\n'
  first=1
  for r in "${rows[@]}"; do
    IFS='|' read -r chain fn loc status <<<"$r"
    [[ $first -eq 0 ]] && printf ',\n'
    first=0
    printf '  {"chain":"%s","function":"%s","location":"%s","status":"%s"}' "$chain" "$fn" "$loc" "$status"
  done
  printf '\n]\n'
else
  printf '%-10s %-40s %-50s %s\n' CHAIN FUNCTION LOCATION STATUS
  for r in "${rows[@]}"; do
    IFS='|' read -r chain fn loc status <<<"$r"
    printf '%-10s %-40s %-50s %s\n' "$chain" "$fn" "$loc" "$status"
  done
fi
