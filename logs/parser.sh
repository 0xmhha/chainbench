#!/usr/bin/env bash
# logs/parser.sh - Log parsing functions for gstable/WBFT consensus events
# Source this file: source logs/parser.sh
#
# Public functions:
#   parse_consensus_events <log_file>   - Extract WBFT consensus events as JSON lines
#   parse_block_events     <log_file>   - Extract block import/commit events as JSON lines
#   parse_peer_events      <log_file>   - Extract peer connect/disconnect events as JSON lines

# Guard against double-sourcing
[[ -n "${_CB_LOGS_PARSER_SH_LOADED:-}" ]] && return 0
readonly _CB_LOGS_PARSER_SH_LOADED=1

# ---- Helpers ------------------------------------------------------------------

# _cb_parser_require_file <log_file>
# Returns 1 with an error message if the file does not exist.
_cb_parser_require_file() {
  local log_file="${1:?_cb_parser_require_file: log_file required}"
  if [[ ! -f "$log_file" ]]; then
    printf '[ERROR] Log file not found: %s\n' "$log_file" >&2
    return 1
  fi
}

# ---- parse_consensus_events ---------------------------------------------------

# parse_consensus_events <log_file>
#
# Scans <log_file> for WBFT consensus-related log lines and emits one JSON
# object per match on stdout.
#
# Recognised patterns (case-insensitive on the event keyword):
#   - NEW ROUND / PREPREPARED / PREPARED / COMMITTED  (ROUND STATE lines)
#   - ROUND CHANGE
#   - PROPOSE / PREPREPARE / PREPARE / COMMIT  (individual message types)
#   - Block sealed / Block imported
#
# Output fields per object:
#   timestamp  - ISO-8601 string extracted from the log line
#   node       - basename of the log file (without .log extension)
#   event      - normalised event name (e.g. NEW_ROUND, PREPREPARED, …)
#   block      - block number as integer, or null
#   round      - round number as integer, or null
#   details    - raw line content after the event keyword
parse_consensus_events() {
  local log_file="${1:?parse_consensus_events: log_file required}"
  _cb_parser_require_file "$log_file" || return 1

  python3 - "$log_file" <<'PYEOF'
import sys
import re
import json
import os

log_file = sys.argv[1]
node_name = os.path.splitext(os.path.basename(log_file))[0]

# Consensus keyword -> normalised event name
# Order matters: more specific patterns first
CONSENSUS_PATTERNS = [
    # ROUND STATE transitions
    (re.compile(r'(?i)NEW[_\s]ROUND'),       'NEW_ROUND'),
    (re.compile(r'(?i)PREPREPARED'),          'PREPREPARED'),
    (re.compile(r'(?i)PREPARED'),             'PREPARED'),
    (re.compile(r'(?i)COMMITTED'),            'COMMITTED'),
    (re.compile(r'(?i)ROUND[_\s]CHANGE'),     'ROUND_CHANGE'),
    # Message types
    (re.compile(r'(?i)\bPREPREPARE\b'),       'PREPREPARE'),
    (re.compile(r'(?i)\bPREPARE\b'),          'PREPARE'),
    (re.compile(r'(?i)\bPROPOSE\b'),          'PROPOSE'),
    (re.compile(r'(?i)\bCOMMIT\b'),           'COMMIT'),
    # Block lifecycle
    (re.compile(r'(?i)block\s+sealed'),       'BLOCK_SEALED'),
    (re.compile(r'(?i)block\s+imported'),     'BLOCK_IMPORTED'),
]

# Capture timestamp at the start of a log line (various formats)
# Handles:
#   2024-01-15T10:23:45.123Z
#   2024-01-15 10:23:45.123
#   Jan 15 10:23:45
TS_RE = re.compile(
    r'^(?P<ts>'
    r'\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?'
    r'|[A-Za-z]{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}'
    r')'
)

# Extract block number from the line
BLOCK_RE  = re.compile(r'(?i)block[=:#\s]+(\d+)|number[=:#\s]+(\d+)|#(\d+)\b')
ROUND_RE  = re.compile(r'(?i)round[=:#\s]+(\d+)')


def extract_int(match):
    """Return the first non-None group from a regex match as int, else None."""
    if match is None:
        return None
    for g in match.groups():
        if g is not None:
            return int(g)
    return None


try:
    with open(log_file, errors='replace') as fh:
        for raw_line in fh:
            line = raw_line.rstrip('\n')

            ts_match = TS_RE.match(line)
            timestamp = ts_match.group('ts') if ts_match else ''

            event_name = None
            for pattern, name in CONSENSUS_PATTERNS:
                if pattern.search(line):
                    event_name = name
                    break

            if event_name is None:
                continue

            block = extract_int(BLOCK_RE.search(line))
            round_num = extract_int(ROUND_RE.search(line))

            # Details: everything after the first consensus keyword match
            details = line.strip()

            obj = {
                'timestamp': timestamp,
                'node':      node_name,
                'event':     event_name,
                'block':     block,
                'round':     round_num,
                'details':   details,
            }
            print(json.dumps(obj))

except OSError as exc:
    print(f'ERROR reading {log_file}: {exc}', file=sys.stderr)
    sys.exit(1)
PYEOF
}

# ---- parse_block_events -------------------------------------------------------

# parse_block_events <log_file>
#
# Extracts block import and commit events from <log_file>.
# Emits one JSON object per event on stdout.
#
# Output fields:
#   timestamp  - ISO-8601 string
#   node       - basename without extension
#   block      - block number as integer, or null
#   hash       - block hash string, or null
#   miner      - miner/coinbase address, or null
#   txs        - transaction count as integer, or null
parse_block_events() {
  local log_file="${1:?parse_block_events: log_file required}"
  _cb_parser_require_file "$log_file" || return 1

  python3 - "$log_file" <<'PYEOF'
import sys
import re
import json
import os

log_file  = sys.argv[1]
node_name = os.path.splitext(os.path.basename(log_file))[0]

# Match lines that contain block import or commit keywords
BLOCK_LINE_RE = re.compile(r'(?i)(?:block\s+(?:imported|committed|added|sealed)|imported\s+new\s+(?:block|chain))')

TS_RE    = re.compile(
    r'^(?P<ts>'
    r'\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?'
    r'|[A-Za-z]{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}'
    r')'
)
NUM_RE    = re.compile(r'(?i)number[=:\s]+(\d+)|#(\d+)\b|block[=:\s]+(\d+)')
HASH_RE   = re.compile(r'(?i)hash[=:\s]+(0x[0-9a-fA-F]+)')
MINER_RE  = re.compile(r'(?i)(?:miner|coinbase|author)[=:\s]+(0x[0-9a-fA-F]+)')
TXS_RE    = re.compile(r'(?i)txs[=:\s]+(\d+)|transactions[=:\s]+(\d+)')


def first_group(match):
    if match is None:
        return None
    for g in match.groups():
        if g is not None:
            return g
    return None


def first_int(match):
    val = first_group(match)
    return int(val) if val is not None else None


try:
    with open(log_file, errors='replace') as fh:
        for raw_line in fh:
            line = raw_line.rstrip('\n')

            if not BLOCK_LINE_RE.search(line):
                continue

            ts_match  = TS_RE.match(line)
            timestamp = ts_match.group('ts') if ts_match else ''

            obj = {
                'timestamp': timestamp,
                'node':      node_name,
                'block':     first_int(NUM_RE.search(line)),
                'hash':      first_group(HASH_RE.search(line)),
                'miner':     first_group(MINER_RE.search(line)),
                'txs':       first_int(TXS_RE.search(line)),
            }
            print(json.dumps(obj))

except OSError as exc:
    print(f'ERROR reading {log_file}: {exc}', file=sys.stderr)
    sys.exit(1)
PYEOF
}

# ---- parse_peer_events --------------------------------------------------------

# parse_peer_events <log_file>
#
# Extracts peer connect / disconnect events from <log_file>.
# Emits one JSON object per event on stdout.
#
# Output fields:
#   timestamp  - ISO-8601 string
#   node       - basename without extension
#   event      - PEER_CONNECTED | PEER_DISCONNECTED
#   peer_id    - enode or peer ID string, or null
#   reason     - disconnect reason string, or null
parse_peer_events() {
  local log_file="${1:?parse_peer_events: log_file required}"
  _cb_parser_require_file "$log_file" || return 1

  python3 - "$log_file" <<'PYEOF'
import sys
import re
import json
import os

log_file  = sys.argv[1]
node_name = os.path.splitext(os.path.basename(log_file))[0]

TS_RE = re.compile(
    r'^(?P<ts>'
    r'\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?'
    r'|[A-Za-z]{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}'
    r')'
)
CONNECT_RE    = re.compile(r'(?i)peer\s+(?:connected|added|handshake)')
DISCONNECT_RE = re.compile(r'(?i)peer\s+(?:disconnected|dropped|removed|lost)')

# enode://... or shortened hex peer ID
PEER_ID_RE = re.compile(r'(enode://[^\s]+|peer[=:\s]+([0-9a-fA-F]{8,}))')
REASON_RE  = re.compile(r'(?i)reason[=:\s]+"?([^",\n]+)"?')


def extract_peer_id(line):
    m = PEER_ID_RE.search(line)
    if m is None:
        return None
    # Return the full enode if present, otherwise the hex id group
    return m.group(1) if m.group(1).startswith('enode://') else m.group(2)


try:
    with open(log_file, errors='replace') as fh:
        for raw_line in fh:
            line = raw_line.rstrip('\n')

            if CONNECT_RE.search(line):
                event = 'PEER_CONNECTED'
            elif DISCONNECT_RE.search(line):
                event = 'PEER_DISCONNECTED'
            else:
                continue

            ts_match  = TS_RE.match(line)
            timestamp = ts_match.group('ts') if ts_match else ''

            reason_match = REASON_RE.search(line)
            reason       = reason_match.group(1).strip() if reason_match else None

            obj = {
                'timestamp': timestamp,
                'node':      node_name,
                'event':     event,
                'peer_id':   extract_peer_id(line),
                'reason':    reason,
            }
            print(json.dumps(obj))

except OSError as exc:
    print(f'ERROR reading {log_file}: {exc}', file=sys.stderr)
    sys.exit(1)
PYEOF
}
