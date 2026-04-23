#!/usr/bin/env bash
# lib/network_client.sh — bash client for chainbench-net.
#
# Usage:
#   source "${CHAINBENCH_DIR}/lib/network_client.sh"
#   data=$(cb_net_call "network.load" '{"name":"local"}') || exit $?
#
# Binary resolution (first match wins):
#   1. $CHAINBENCH_NET_BIN
#   2. $CHAINBENCH_DIR/bin/chainbench-net
#   3. $CHAINBENCH_DIR/network/bin/chainbench-net
#   4. `command -v chainbench-net` on PATH
#
# Exit codes:
#   0 — success. data JSON on stdout.
#   1 — API error. "<code>: <message>" on stderr.
#   2 — spawn/parse failure. diagnostic on stderr.
#
# Dependencies: jq.

# Guard against double-sourcing.
if [[ -n "${_CB_NET_CLIENT_LOADED:-}" ]]; then
  return 0
fi
readonly _CB_NET_CLIENT_LOADED=1

if ! command -v jq >/dev/null 2>&1; then
  echo "network_client: WARNING: jq not found on PATH — cb_net_call will fail" >&2
fi

# _cb_net_binary prints the resolved path to chainbench-net or returns 1.
_cb_net_binary() {
  if [[ -n "${CHAINBENCH_NET_BIN:-}" && -x "${CHAINBENCH_NET_BIN}" ]]; then
    echo "${CHAINBENCH_NET_BIN}"
    return 0
  fi
  if [[ -n "${CHAINBENCH_DIR:-}" ]]; then
    if [[ -x "${CHAINBENCH_DIR}/bin/chainbench-net" ]]; then
      echo "${CHAINBENCH_DIR}/bin/chainbench-net"
      return 0
    fi
    if [[ -x "${CHAINBENCH_DIR}/network/bin/chainbench-net" ]]; then
      echo "${CHAINBENCH_DIR}/network/bin/chainbench-net"
      return 0
    fi
  fi
  if command -v chainbench-net >/dev/null 2>&1; then
    command -v chainbench-net
    return 0
  fi
  return 1
}

# _cb_net_spawn <envelope_json>
# Pipes envelope_json to chainbench-net run and prints its stdout stream.
# Returns the subprocess exit code.
_cb_net_spawn() {
  local envelope="$1"
  local bin
  if ! bin=$(_cb_net_binary); then
    echo "network_client: chainbench-net binary not found" >&2
    return 2
  fi
  printf '%s\n' "$envelope" | "$bin" run
}

# _cb_net_parse_result
# Reads NDJSON from stdin, finds the last type=result line, and emits:
#   ok=true  — prints .data JSON to stdout, returns 0
#   ok=false — prints "<code>: <message>" to stderr, returns 1
#   no terminator — prints diagnostic to stderr, returns 2
_cb_net_parse_result() {
  local result_line
  result_line=$(grep '^{' | awk '/"type":"result"/ { last = $0 } END { print last }')
  if [[ -z "$result_line" ]]; then
    echo "network_client: no result terminator in stream" >&2
    return 2
  fi
  local ok
  ok=$(printf '%s' "$result_line" | jq -r '.ok // false')
  if [[ "$ok" == "true" ]]; then
    printf '%s' "$result_line" | jq -c '.data // {}'
    return 0
  fi
  local code msg
  code=$(printf '%s' "$result_line" | jq -r '.error.code // "UNKNOWN"')
  msg=$(printf '%s' "$result_line" | jq -r '.error.message // ""')
  echo "${code}: ${msg}" >&2
  return 1
}

# cb_net_call <command> [args_json]
# See header comment for semantics.
cb_net_call() {
  local command="${1:?cb_net_call: command required}"
  # Bash closes ${...:-...} at the first literal '}', so '{}' as a default
  # must be built outside the expansion.
  local args_json="${2:-}"
  if [[ -z "$args_json" ]]; then
    args_json='{}'
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "network_client: jq not available" >&2
    return 2
  fi

  local envelope
  if ! envelope=$(jq -cn --arg c "$command" --argjson a "$args_json" \
    '{command:$c,args:$a}' 2>/dev/null); then
    echo "network_client: invalid args_json: ${args_json}" >&2
    return 2
  fi

  local stream
  local spawn_rc
  stream=$(_cb_net_spawn "$envelope")
  spawn_rc=$?

  if [[ $spawn_rc -eq 2 ]]; then
    # _cb_net_spawn already printed a diagnostic.
    return 2
  fi

  # Parse regardless of non-zero exit — the binary emits a result terminator
  # for every exit path (protocol/invalid/upstream/internal).
  printf '%s\n' "$stream" | _cb_net_parse_result
}
