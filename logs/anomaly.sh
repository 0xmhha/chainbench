#!/usr/bin/env bash
# logs/anomaly.sh - Detect anomalous patterns in node logs
# Source this file: source logs/anomaly.sh
#
# Public function:
#   detect_anomalies [--json]
#
# Checks performed:
#   1. ROUND_CHANGE storms  - >3 round changes within 10 seconds for the same block
#   2. Block gaps           - block production gap > 3x blockPeriodSeconds
#   3. Peer disconnections  - repeated disconnects from the same peer (>= 3 times)
#   4. Proposer failures    - consecutive proposal timeouts (>= 3 in a row)

# Guard against double-sourcing
[[ -n "${_CB_LOGS_ANOMALY_SH_LOADED:-}" ]] && return 0
readonly _CB_LOGS_ANOMALY_SH_LOADED=1

_CB_ANOMALY_SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source parser functions
# shellcheck source=logs/parser.sh
source "${_CB_ANOMALY_SELF_DIR}/parser.sh"

CHAINBENCH_DIR="${CHAINBENCH_DIR:-$(cd "${_CB_ANOMALY_SELF_DIR}/.." && pwd)}"

# ---- Internal helpers ---------------------------------------------------------

# _cb_anomaly_collect_log_files
# Prints the path to each node log file, one per line.
_cb_anomaly_collect_log_files() {
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

  # Fallback: any .log file under data/logs
  if [[ -d "$logs_dir" ]]; then
    local f
    for f in "${logs_dir}"/*.log; do
      [[ -f "$f" ]] && printf '%s\n' "$f"
    done
  fi
}

# _cb_anomaly_read_block_period
# Reads genesis.overrides.wbft.blockPeriodSeconds from the active profile.
# Prints an integer (default 1 if not found).
_cb_anomaly_read_block_period() {
  local profile_file="${CHAINBENCH_DIR}/state/current-profile.yaml"
  python3 - "$profile_file" <<'PYEOF'
import sys, os, re

profile_file = sys.argv[1]
default = 1

if not os.path.isfile(profile_file):
    print(default)
    sys.exit(0)

with open(profile_file) as fh:
    for line in fh:
        m = re.match(r'\s*blockPeriodSeconds\s*:\s*(\d+)', line)
        if m:
            print(int(m.group(1)))
            sys.exit(0)

print(default)
PYEOF
}

# ---- Core anomaly detection in Python ----------------------------------------

# _cb_anomaly_analyse <block_period_seconds> <events_jsonl_file> <peers_jsonl_file>
# Runs all four anomaly checks and prints a JSON array of anomaly objects.
#
# Each anomaly object has:
#   severity    - "warning" | "critical"
#   check       - short check identifier
#   timestamp   - when the anomaly was detected (timestamp of triggering event)
#   node        - node label
#   description - human-readable description
_cb_anomaly_analyse() {
  local block_period="$1"
  local events_file="$2"
  local peers_file="$3"

  python3 - "$block_period" "$events_file" "$peers_file" <<'PYEOF'
import sys, json, os
from datetime import datetime, timezone, timedelta
from collections import defaultdict

block_period  = int(sys.argv[1])
events_file   = sys.argv[2]
peers_file    = sys.argv[3]

# ---- Utility -----------------------------------------------------------------

def parse_ts(ts_str):
    if not ts_str:
        return None
    ts_str = ts_str.strip()
    fmts = [
        '%Y-%m-%dT%H:%M:%S.%f%z',
        '%Y-%m-%dT%H:%M:%S%z',
        '%Y-%m-%dT%H:%M:%S.%fZ',
        '%Y-%m-%dT%H:%M:%SZ',
        '%Y-%m-%d %H:%M:%S.%f',
        '%Y-%m-%d %H:%M:%S',
    ]
    for fmt in fmts:
        try:
            dt = datetime.strptime(ts_str, fmt)
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt
        except ValueError:
            pass
    return None


def load_jsonl(path):
    rows = []
    if not os.path.isfile(path):
        return rows
    with open(path, errors='replace') as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError:
                pass
    return rows


# ---- Load data ---------------------------------------------------------------

consensus_events = load_jsonl(events_file)
peer_events      = load_jsonl(peers_file)

# Attach parsed datetime to each event
for ev in consensus_events:
    ev['_dt'] = parse_ts(ev.get('timestamp', ''))
for ev in peer_events:
    ev['_dt'] = parse_ts(ev.get('timestamp', ''))

anomalies = []


# ---- Check 1: ROUND_CHANGE storms -------------------------------------------
# >3 round changes within 10 seconds for the same block

STORM_WINDOW_S   = 10
STORM_THRESHOLD  = 3

rc_by_node_block = defaultdict(list)  # (node, block) -> [datetime, ...]

for ev in consensus_events:
    if ev.get('event') != 'ROUND_CHANGE':
        continue
    dt = ev.get('_dt')
    if dt is None:
        continue
    key = (ev.get('node', ''), ev.get('block'))
    rc_by_node_block[key].append((dt, ev.get('timestamp', '')))

for (node, block), occurrences in rc_by_node_block.items():
    occurrences.sort(key=lambda x: x[0])
    # Sliding window
    for i in range(len(occurrences)):
        window = [
            o for o in occurrences[i:]
            if (o[0] - occurrences[i][0]).total_seconds() <= STORM_WINDOW_S
        ]
        if len(window) > STORM_THRESHOLD:
            anomalies.append({
                'severity':    'critical',
                'check':       'ROUND_CHANGE_STORM',
                'timestamp':   occurrences[i][1],
                'node':        node,
                'description': (
                    f'Round change storm: {len(window)} ROUND_CHANGE events '
                    f'for block {block} within {STORM_WINDOW_S}s '
                    f'(threshold: {STORM_THRESHOLD})'
                ),
            })
            break  # Report once per (node, block) pair


# ---- Check 2: Block gaps ----------------------------------------------------
# Gap between consecutive block events > 3x blockPeriodSeconds

GAP_MULTIPLIER = 3
gap_threshold_s = block_period * GAP_MULTIPLIER

# Collect BLOCK_IMPORTED / COMMITTED events per node
block_evs_by_node = defaultdict(list)
for ev in consensus_events:
    if ev.get('event') not in ('BLOCK_IMPORTED', 'COMMITTED', 'BLOCK_SEALED'):
        continue
    dt = ev.get('_dt')
    if dt is None:
        continue
    block_evs_by_node[ev.get('node', '')].append((dt, ev.get('block'), ev.get('timestamp', '')))

for node, evs in block_evs_by_node.items():
    evs.sort(key=lambda x: x[0])
    for i in range(1, len(evs)):
        prev_dt, prev_block, _ = evs[i - 1]
        curr_dt, curr_block, curr_ts = evs[i]
        gap = (curr_dt - prev_dt).total_seconds()
        if gap > gap_threshold_s:
            anomalies.append({
                'severity':    'warning',
                'check':       'BLOCK_GAP',
                'timestamp':   curr_ts,
                'node':        node,
                'description': (
                    f'Block gap detected: {gap:.1f}s between block '
                    f'{prev_block} and {curr_block} '
                    f'(threshold: {gap_threshold_s}s = {GAP_MULTIPLIER}x '
                    f'blockPeriod={block_period}s)'
                ),
            })


# ---- Check 3: Repeated peer disconnections ----------------------------------
# >=3 disconnects from the same peer

DISCONNECT_THRESHOLD = 3

disconnects = defaultdict(list)  # (node, peer_id) -> [timestamp_str, ...]
for ev in peer_events:
    if ev.get('event') != 'PEER_DISCONNECTED':
        continue
    peer_id = ev.get('peer_id') or 'unknown'
    disconnects[(ev.get('node', ''), peer_id)].append(ev.get('timestamp', ''))

for (node, peer_id), timestamps in disconnects.items():
    if len(timestamps) >= DISCONNECT_THRESHOLD:
        anomalies.append({
            'severity':    'warning',
            'check':       'REPEATED_DISCONNECT',
            'timestamp':   timestamps[-1],
            'node':        node,
            'description': (
                f'Peer {peer_id} disconnected {len(timestamps)} times '
                f'(threshold: {DISCONNECT_THRESHOLD})'
            ),
        })


# ---- Check 4: Proposer failures (consecutive ROUND_CHANGE without COMMITTED) -
# >=3 consecutive round changes without an intervening COMMITTED event

PROPOSER_FAILURE_THRESHOLD = 3

events_by_node = defaultdict(list)
for ev in consensus_events:
    dt = ev.get('_dt')
    if dt is None:
        continue
    events_by_node[ev.get('node', '')].append((dt, ev))

for node, evs in events_by_node.items():
    evs.sort(key=lambda x: x[0])
    consecutive_rc = 0
    first_rc_ts    = ''
    for _, ev in evs:
        event_name = ev.get('event', '')
        if event_name == 'ROUND_CHANGE':
            if consecutive_rc == 0:
                first_rc_ts = ev.get('timestamp', '')
            consecutive_rc += 1
            if consecutive_rc >= PROPOSER_FAILURE_THRESHOLD:
                anomalies.append({
                    'severity':    'critical',
                    'check':       'PROPOSER_FAILURE',
                    'timestamp':   first_rc_ts,
                    'node':        node,
                    'description': (
                        f'Proposer failure: {consecutive_rc} consecutive '
                        f'ROUND_CHANGE events without a COMMITTED event '
                        f'(threshold: {PROPOSER_FAILURE_THRESHOLD})'
                    ),
                })
                # Reset after reporting so we do not spam
                consecutive_rc = 0
        elif event_name in ('COMMITTED', 'BLOCK_SEALED', 'BLOCK_IMPORTED'):
            consecutive_rc = 0
            first_rc_ts    = ''


print(json.dumps(anomalies, indent=2))
PYEOF
}

# ---- Formatters ---------------------------------------------------------------

# _cb_anomaly_format_human <anomalies_json>
_cb_anomaly_format_human() {
  local anomalies_json="$1"

  python3 - "$anomalies_json" <<'PYEOF'
import sys, json

anomalies = json.loads(sys.argv[1])

if not anomalies:
    print('[Anomaly Detection] No anomalies detected.')
    sys.exit(0)

print(f'[Anomaly Detection] {len(anomalies)} anomaly/anomalies found:')
print()

SEVERITY_LABEL = {
    'warning':  '[WARN]    ',
    'critical': '[CRITICAL]',
}

for a in anomalies:
    severity = a.get('severity', 'warning')
    label    = SEVERITY_LABEL.get(severity, '[INFO]    ')
    ts       = a.get('timestamp', '')
    node     = a.get('node', '')
    desc     = a.get('description', '')
    print(f'{label} {ts}  node={node}')
    print(f'           {desc}')
    print()
PYEOF
}

# ---- Public entry point -------------------------------------------------------

# detect_anomalies [--json]
#
# Options:
#   --json    Emit a JSON array of anomaly objects instead of text output
detect_anomalies() {
  local json_mode=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json)
        json_mode=1
        shift
        ;;
      --help|-h)
        printf 'Usage: detect_anomalies [--json]\n' >&2
        return 0
        ;;
      *)
        printf '[WARN] Unknown anomaly option: %s (ignoring)\n' "$1" >&2
        shift
        ;;
    esac
  done

  # Collect log files
  local -a log_files=()
  while IFS= read -r lf; do
    [[ -n "$lf" ]] && log_files+=("$lf")
  done < <(_cb_anomaly_collect_log_files)

  if [[ ${#log_files[@]} -eq 0 ]]; then
    printf '[WARN] No node log files found.\n' >&2
    if [[ "$json_mode" == "1" ]]; then
      printf '[]\n'
    fi
    return 0
  fi

  # Gather consensus and peer events into temp files
  local tmp_events tmp_peers
  tmp_events="$(mktemp /tmp/chainbench-anomaly-events-XXXXXX.jsonl)"
  tmp_peers="$(mktemp /tmp/chainbench-anomaly-peers-XXXXXX.jsonl)"

  local lf
  for lf in "${log_files[@]}"; do
    parse_consensus_events "$lf" >> "$tmp_events" 2>/dev/null || true
    parse_peer_events      "$lf" >> "$tmp_peers"  2>/dev/null || true
  done

  local block_period
  block_period="$(_cb_anomaly_read_block_period)"

  local anomalies_json
  anomalies_json="$(_cb_anomaly_analyse "$block_period" "$tmp_events" "$tmp_peers")"

  rm -f "$tmp_events" "$tmp_peers"

  if [[ "$json_mode" == "1" ]]; then
    python3 -c "import sys, json; print(json.dumps(json.loads(sys.argv[1]), indent=2))" \
      "$anomalies_json"
  else
    _cb_anomaly_format_human "$anomalies_json"
  fi
}
