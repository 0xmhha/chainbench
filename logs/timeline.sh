#!/usr/bin/env bash
# logs/timeline.sh - Chronological consensus event timeline across all nodes
# Source this file: source logs/timeline.sh
#
# Public function:
#   generate_timeline [--json] [--last <seconds>]
#
# Reads node log files discovered via state/pids.json (or data/logs/*.log
# as a fallback), parses consensus events with logs/parser.sh, merges and
# sorts them by timestamp, then renders a human-readable table or JSON array.

# Guard against double-sourcing
[[ -n "${_CB_LOGS_TIMELINE_SH_LOADED:-}" ]] && return 0
readonly _CB_LOGS_TIMELINE_SH_LOADED=1

_CB_TIMELINE_SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source parser functions
# shellcheck source=logs/parser.sh
source "${_CB_TIMELINE_SELF_DIR}/parser.sh"

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "${_CB_TIMELINE_SELF_DIR}/.." && pwd)}"

# ---- Internal helpers ---------------------------------------------------------

# _cb_timeline_collect_log_files
# Prints the path to each node log file, one per line.
# Prefers reading from state/pids.json; falls back to data/logs/*.log.
_cb_timeline_collect_log_files() {
  local pids_file="${CHAINBENCH_DIR}/state/pids.json"
  local logs_dir="${CHAINBENCH_DIR}/data/logs"

  if [[ -f "$pids_file" ]]; then
    python3 - "$pids_file" "$logs_dir" <<'PYEOF'
import sys, json, os

pids_file = sys.argv[1]
logs_dir  = sys.argv[2]

with open(pids_file) as fh:
    data = json.load(fh)

for node in data.get('nodes', []):
    label = node.get('label', '')
    if not label:
        continue
    candidate = os.path.join(logs_dir, f'{label}.log')
    if os.path.isfile(candidate):
        print(candidate)
PYEOF
  fi

  # Fallback: any .log file in data/logs
  if [[ -d "$logs_dir" ]]; then
    local f
    for f in "${logs_dir}"/*.log; do
      [[ -f "$f" ]] && printf '%s\n' "$f"
    done
  fi
}

# _cb_timeline_merge_events <last_seconds> <log_file> [<log_file> ...]
# Runs parse_consensus_events on each log file, merges all JSON-line events,
# filters to the last <last_seconds> seconds, sorts by timestamp, and prints
# the merged sorted JSON array to stdout.
_cb_timeline_merge_events() {
  local last_seconds="$1"
  shift
  local -a log_files=("$@")

  # Collect all raw JSON lines into one temporary file
  local tmp_events
  tmp_events="$(mktemp /tmp/chainbench-timeline-XXXXXX.jsonl)"

  local lf
  for lf in "${log_files[@]}"; do
    [[ -f "$lf" ]] || continue
    parse_consensus_events "$lf" >> "$tmp_events" 2>/dev/null || true
  done

  python3 - "$tmp_events" "$last_seconds" <<'PYEOF'
import sys, json, os
from datetime import datetime, timezone, timedelta

events_file  = sys.argv[1]
last_seconds = int(sys.argv[2])

events = []
with open(events_file, errors='replace') as fh:
    for line in fh:
        line = line.strip()
        if not line:
            continue
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            pass

os.unlink(events_file)


def parse_ts(ts_str):
    """Parse an ISO-style or syslog-style timestamp; return datetime (UTC)."""
    if not ts_str:
        return None
    ts_str = ts_str.strip()
    formats = [
        '%Y-%m-%dT%H:%M:%S.%f%z',
        '%Y-%m-%dT%H:%M:%S%z',
        '%Y-%m-%dT%H:%M:%S.%fZ',
        '%Y-%m-%dT%H:%M:%SZ',
        '%Y-%m-%d %H:%M:%S.%f',
        '%Y-%m-%d %H:%M:%S',
    ]
    for fmt in formats:
        try:
            dt = datetime.strptime(ts_str, fmt)
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt
        except ValueError:
            pass
    return None


# Parse timestamps and drop events without a parseable timestamp
parsed = []
for ev in events:
    dt = parse_ts(ev.get('timestamp', ''))
    if dt is not None:
        parsed.append((dt, ev))

if not parsed:
    print(json.dumps([]))
    sys.exit(0)

# Determine the cutoff: latest event minus last_seconds
latest_dt = max(dt for dt, _ in parsed)
cutoff_dt = latest_dt - timedelta(seconds=last_seconds)

filtered = [(dt, ev) for dt, ev in parsed if dt >= cutoff_dt]
filtered.sort(key=lambda x: x[0])

# Attach a relative offset field (seconds since first event in window)
if filtered:
    first_dt = filtered[0][0]
    result = []
    for dt, ev in filtered:
        offset = (dt - first_dt).total_seconds()
        enriched = dict(ev)
        enriched['offset_seconds'] = round(offset, 3)
        result.append(enriched)
else:
    result = []

print(json.dumps(result))
PYEOF
}

# ---- Formatters ---------------------------------------------------------------

# _cb_timeline_format_human <json_array>
_cb_timeline_format_human() {
  local json_array="$1"

  python3 - "$json_array" <<'PYEOF'
import sys, json

events = json.loads(sys.argv[1])

if not events:
    print('[Timeline] No consensus events found in the requested window.')
    sys.exit(0)

# Determine window size label
last_offset = events[-1].get('offset_seconds', 0)
print(f'[Consensus Timeline: last {last_offset:.0f} seconds]')
print()

# Column widths
col_offset = 10
col_node   = 10
col_event  = 16
col_detail = 50

header = (
    f"{'Time':>{col_offset}}  "
    f"{'Node':<{col_node}}  "
    f"{'Event':<{col_event}}  "
    f"Detail"
)
print(header)
print('-' * len(header))

for ev in events:
    offset  = ev.get('offset_seconds', 0)
    node    = str(ev.get('node', ''))[:col_node]
    event   = str(ev.get('event', ''))[:col_event]
    block   = ev.get('block')
    round_n = ev.get('round')

    # Build detail string
    parts = []
    if block is not None:
        parts.append(f'block={block}')
    if round_n is not None:
        parts.append(f'round={round_n}')
    detail = ' '.join(parts)

    print(
        f'T+{offset:<{col_offset - 2}.3f}s  '
        f'{node:<{col_node}}  '
        f'{event:<{col_event}}  '
        f'{detail}'
    )
PYEOF
}

# _cb_timeline_format_json <json_array>
_cb_timeline_format_json() {
  # The array is already valid JSON; pretty-print it
  python3 -c "import sys, json; print(json.dumps(json.loads(sys.argv[1]), indent=2))" "$1"
}

# ---- Public entry point -------------------------------------------------------

# generate_timeline [--json] [--last <seconds>]
#
# Options:
#   --json           Emit a JSON array instead of human-readable text
#   --last <N>       Only show events from the last N seconds (default: 30)
generate_timeline() {
  local json_mode=0
  local last_seconds=30

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json)
        json_mode=1
        shift
        ;;
      --last)
        if [[ -z "${2:-}" ]] || ! [[ "${2}" =~ ^[0-9]+$ ]]; then
          printf '[ERROR] --last requires a positive integer\n' >&2
          return 1
        fi
        last_seconds="$2"
        shift 2
        ;;
      --help|-h)
        printf 'Usage: generate_timeline [--json] [--last <seconds>]\n' >&2
        return 0
        ;;
      *)
        printf '[WARN] Unknown timeline option: %s (ignoring)\n' "$1" >&2
        shift
        ;;
    esac
  done

  # Collect log files
  local -a log_files=()
  while IFS= read -r lf; do
    [[ -n "$lf" ]] && log_files+=("$lf")
  done < <(_cb_timeline_collect_log_files)

  if [[ ${#log_files[@]} -eq 0 ]]; then
    printf '[WARN] No node log files found.\n' >&2
    if [[ "$json_mode" == "1" ]]; then
      printf '[]\n'
    fi
    return 0
  fi

  local merged_json
  merged_json="$(_cb_timeline_merge_events "$last_seconds" "${log_files[@]}")"

  if [[ "$json_mode" == "1" ]]; then
    _cb_timeline_format_json "$merged_json"
  else
    _cb_timeline_format_human "$merged_json"
  fi
}
