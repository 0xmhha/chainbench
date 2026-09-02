#!/usr/bin/env bash
# attach-external.sh — run chainbench's chain-agnostic cases against an external
# chain: one already running somewhere else that chainbench attaches to over RPC
# (it does not launch or manage it), needing no chain binary — just an RPC URL.
#
# This covers the legacy regression "Layer 2 E2E" section (z-layer2-e2e,
# RT-Z-01/03/04). Despite that name it is NOT an Ethereum L2 / rollup and needs no
# L2 support in chainbench: "Layer 2 E2E" there just meant generic RPC operations
# (chain state, event logs, contract call, send tx, fee delegation) run against
# some external chain. The reference suite has no runnable z-layer2 scripts — only
# a design note — because those operations are already chainbench's chain-agnostic
# testkit cases (empty ChainCompat, gated only on "rpc"), and `chainbench test
# --rpc` attaches any endpoint. So this script points them at an external RPC and
# asserts they run (skip=0).
#
# Covered here (read/state — no key needed): RT-Z-03 (event logs), RT-Z-04 (chain
# state), and the general RPC well-formedness the cases assert.
# NOT covered (need a funded key on the external chain): RT-Z-02 (send tx) and
# RT-Z-05 (fee delegation) — set CHAINBENCH_FUNDED_KEY to run the write cases.
#
# Requirements: EXTERNAL_RPC (the external chain's JSON-RPC URL). Optional:
# EXTERNAL_CHAIN (label, default "external").
#
#   EXTERNAL_RPC=https://chain.example/rpc tests/repro/attach-external.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-attach-external-bin}"
EXTERNAL_CHAIN="${EXTERNAL_CHAIN:-external}"

# Chain-agnostic, single-endpoint cases (empty ChainCompat, rpc-only). Each runs
# against any attached chain.
CASES=(
  block-progression
  gas-price-positive
  chain-not-syncing
  block-by-hash-consistency
  fee-history-well-formed
  estimate-gas
  logs-query-well-formed
  txpool-status
  block-transactions-field
  txpool-content-well-formed
)

log() { printf '\n=== %s ===\n' "$*"; }

[ -n "${EXTERNAL_RPC:-}" ] || {
  echo "set EXTERNAL_RPC to the external chain's JSON-RPC URL"
  exit 2
}
[ -x "$CHAINBENCH" ] || {
  echo "building chainbench"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2
}

args=()
for c in "${CASES[@]}"; do args+=(--name "$c"); done

log "run chain-agnostic cases against external endpoint $EXTERNAL_RPC (chain=$EXTERNAL_CHAIN)"
if ! out="$("$CHAINBENCH" test --rpc "$EXTERNAL_RPC" --chain "$EXTERNAL_CHAIN" "${args[@]}" 2>&1)"; then
  echo "$out"
  echo "FAIL: some cases reported failures"
  exit 1
fi
echo "$out"
# The selected cases are chain-agnostic; a skip means the endpoint lacks the rpc
# capability or the case did not apply — surface it rather than pass silently.
if echo "$out" | grep -qE 'skip=[1-9]'; then
  echo "FAIL: some cases skipped (endpoint capability problem)"
  exit 1
fi
log "PASS: chain-agnostic read cases green against the external endpoint"

# Write side (RT-Z-02 send tx, RT-Z-05 fee delegation): the chain-agnostic write
# cases need a funded account key on the external chain, supplied via
# CHAINBENCH_FUNDED_KEY (env only — never a literal). Without it, they skip.
if [ -n "${CHAINBENCH_FUNDED_KEY:-}" ]; then
  log "run chain-agnostic write cases (funded key present)"
  if ! wout="$("$CHAINBENCH" test --rpc "$EXTERNAL_RPC" --chain "$EXTERNAL_CHAIN" \
    --name external-value-transfer --name external-fee-delegated-transfer 2>&1)"; then
    echo "$wout"
    echo "FAIL: external-chain write cases reported failures"
    exit 1
  fi
  echo "$wout"
  # A skip here means the funded key did not take or 0x16 is unsupported; the
  # value-transfer case at least must have run.
  if echo "$wout" | grep -qE 'pass=0'; then
    echo "FAIL: no write case ran (funded key not applied?)"
    exit 1
  fi
  log "PASS: external-chain write cases ran with the funded key"
else
  log "SKIP write side: set CHAINBENCH_FUNDED_KEY to run RT-Z-02/RT-Z-05 on the external chain"
fi
