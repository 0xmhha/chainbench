#!/usr/bin/env bash
# tests/lib/report.sh - Test report generation for chainbench
# Usage: source tests/lib/report.sh
#        generate_report [text|json|markdown]

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

# Color codes (consistent with lib/common.sh)
_REPORT_GREEN='\033[0;32m'
_REPORT_RED='\033[0;31m'
_REPORT_CYAN='\033[0;36m'
_REPORT_YELLOW='\033[0;33m'
_REPORT_BOLD='\033[1m'
_REPORT_RESET='\033[0m'

# ---------------------------------------------------------------------------
# Internal: load all result JSON files from state/results/
# ---------------------------------------------------------------------------

# _collect_results
# Prints a JSON array of all result objects found in state/results/*.json.
_collect_results() {
  local results_dir="${CHAINBENCH_DIR}/state/results"
  python3 -c "
import json, os, sys, glob

results_dir = sys.argv[1]
pattern = os.path.join(results_dir, '*.json')
files = sorted(glob.glob(pattern))

results = []
for path in files:
    try:
        with open(path) as f:
            data = json.load(f)
        data.setdefault('_file', os.path.basename(path))
        results.append(data)
    except Exception as e:
        pass  # skip malformed files silently

print(json.dumps(results))
" "$results_dir"
}

# ---------------------------------------------------------------------------
# Report formats
# ---------------------------------------------------------------------------

# _report_json <results_json>
# Outputs a machine-readable JSON summary.
_report_json() {
  local results_json="${1}"
  python3 -c "
import json, sys

results = json.loads(sys.argv[1])
total_tests  = len(results)
total_pass   = sum(1 for r in results if r.get('status') == 'passed')
total_fail   = total_tests - total_pass
total_asserts_pass = sum(r.get('pass', 0) for r in results)
total_asserts_fail = sum(r.get('fail', 0) for r in results)

summary = {
    'summary': {
        'total_tests':   total_tests,
        'passed':        total_pass,
        'failed':        total_fail,
        'assertions': {
            'passed': total_asserts_pass,
            'failed': total_asserts_fail,
        },
    },
    'tests': results,
}
print(json.dumps(summary, indent=2))
" "$results_json"
}

# _report_markdown <results_json>
# Outputs a human-readable Markdown table.
_report_markdown() {
  local results_json="${1}"
  python3 -c "
import json, sys

results = json.loads(sys.argv[1])
total_tests  = len(results)
total_pass   = sum(1 for r in results if r.get('status') == 'passed')
total_fail   = total_tests - total_pass

lines = []
lines.append('# Chainbench Test Report')
lines.append('')
lines.append('## Summary')
lines.append('')
lines.append(f'- Total tests : {total_tests}')
lines.append(f'- Passed      : {total_pass}')
lines.append(f'- Failed      : {total_fail}')
lines.append('')
lines.append('## Results')
lines.append('')
lines.append('| Test | Status | Pass | Fail | Duration | Timestamp |')
lines.append('|------|--------|-----:|-----:|----------|-----------|')

for r in results:
    name      = r.get('test', r.get('_file', 'unknown'))
    status    = r.get('status', '-')
    passed    = r.get('pass', 0)
    failed    = r.get('fail', 0)
    duration  = str(r.get('duration', '-')) + 's'
    ts        = r.get('timestamp', '-')
    marker    = 'PASS' if status == 'passed' else 'FAIL'
    lines.append(f'| {name} | {marker} | {passed} | {failed} | {duration} | {ts} |')

if any(r.get('failures') for r in results):
    lines.append('')
    lines.append('## Failure Details')
    for r in results:
        failures = r.get('failures', [])
        if not failures:
            continue
        name = r.get('test', r.get('_file', 'unknown'))
        lines.append('')
        lines.append(f'### {name}')
        for f in failures:
            lines.append(f'- {f}')

print('\n'.join(lines))
" "$results_json"
}

# _report_text <results_json>
# Outputs a simple human-readable text summary (default).
_report_text() {
  local results_json="${1}"
  python3 -c "
import json, sys

results = json.loads(sys.argv[1])
total_tests  = len(results)
total_pass   = sum(1 for r in results if r.get('status') == 'passed')
total_fail   = total_tests - total_pass
total_asserts_pass = sum(r.get('pass', 0) for r in results)
total_asserts_fail = sum(r.get('fail', 0) for r in results)

print('=== Chainbench Test Report ===')
print(f'Tests   : {total_tests} total, {total_pass} passed, {total_fail} failed')
print(f'Asserts : {total_asserts_pass + total_asserts_fail} total,'
      f' {total_asserts_pass} passed, {total_asserts_fail} failed')
print()

col_w = max((len(r.get('test', '')) for r in results), default=4)
col_w = max(col_w, 4)
header = f\"{'Test':<{col_w}}  {'Status':<8}  {'Pass':>5}  {'Fail':>5}  {'Dur':>6}\"
print(header)
print('-' * len(header))

for r in results:
    name     = r.get('test', r.get('_file', 'unknown'))
    status   = 'PASS' if r.get('status') == 'passed' else 'FAIL'
    passed   = r.get('pass', 0)
    failed   = r.get('fail', 0)
    duration = str(r.get('duration', '-')) + 's'
    print(f'{name:<{col_w}}  {status:<8}  {passed:>5}  {failed:>5}  {duration:>6}')

if any(r.get('failures') for r in results):
    print()
    print('--- Failures ---')
    for r in results:
        failures = r.get('failures', [])
        if not failures:
            continue
        print(f\"{r.get('test', 'unknown')}:\")
        for f in failures:
            print(f'  - {f}')
" "$results_json"
}

# ---------------------------------------------------------------------------
# Public entry point
# ---------------------------------------------------------------------------

# generate_report [format]
#
# Reads all JSON files from state/results/ and prints a report.
# format: text (default) | json | markdown
generate_report() {
  local format="${1:-text}"

  local results_dir="${CHAINBENCH_DIR}/state/results"
  if [[ ! -d "$results_dir" ]]; then
    printf "${_REPORT_YELLOW}[REPORT]${_REPORT_RESET}  No results directory found: %s\n" "$results_dir" >&2
    return 1
  fi

  local results_json
  results_json=$(_collect_results)

  local count
  count=$(python3 -c "import json,sys; print(len(json.loads(sys.argv[1])))" "$results_json")

  if [[ "$count" -eq 0 ]]; then
    printf "${_REPORT_YELLOW}[REPORT]${_REPORT_RESET}  No result files found in %s\n" "$results_dir" >&2
    return 0
  fi

  printf "${_REPORT_CYAN}[REPORT]${_REPORT_RESET}  Generating %s report from %d result(s)...\n" \
    "$format" "$count" >&2

  case "$format" in
    json)
      _report_json "$results_json"
      ;;
    markdown|md)
      _report_markdown "$results_json"
      ;;
    text|*)
      _report_text "$results_json"
      ;;
  esac
}
