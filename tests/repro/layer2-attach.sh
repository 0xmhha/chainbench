#!/usr/bin/env bash
# layer2-attach.sh — run chainbench's chain-agnostic cases against an attached
# Layer 2 endpoint (regression z-layer2-e2e RT-Z-01/03/04). The reference suite
# has no z-layer2 scripts — only a design note — because "Layer 2 E2E" is just
# generic RPC operations (chain state, event logs, contract call, send tx, fee
# delegation) run against an L2 chain. chainbench already exposes those as
# chain-agnostic testkit cases (empty ChainCompat, gated only on "rpc"), and
# `chainbench test --rpc` attaches any endpoint — so this script points them at
# an L2 RPC and asserts they run (skip=0).
#
# This attaches to an ALREADY-RUNNING L2, so it needs no chain binary — just an
# L2 RPC URL. It is the L2 verification path from docs/dev/HandOff.md.
#
# Covered here (read/state — no key needed): RT-Z-03 (event logs), RT-Z-04 (chain
# state), and the general RPC well-formedness the L2 cases assert.
# NOT covered (need an L2-funded key): RT-Z-02 (send tx) and RT-Z-05 (fee
# delegation) — provide an L2 account and use `chainbench tx`/the transfer cases
# with the L2 in their ChainCompat.
#
# Requirements: L2_RPC (the Layer 2 JSON-RPC URL). Optional: L2_CHAIN (label,
# default "layer2").
#
#   L2_RPC=https://l2.example/rpc tests/repro/layer2-attach.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-layer2-bin}"
L2_CHAIN="${L2_CHAIN:-layer2}"

# Chain-agnostic, single-endpoint cases (empty ChainCompat, rpc-only). Each runs
# against any attached chain, including an L2.
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

[ -n "${L2_RPC:-}" ] || {
  echo "set L2_RPC to the Layer 2 JSON-RPC URL"
  exit 2
}
[ -x "$CHAINBENCH" ] || {
  echo "building chainbench"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2
}

args=()
for c in "${CASES[@]}"; do args+=(--name "$c"); done

log "run chain-agnostic cases against L2 endpoint $L2_RPC (chain=$L2_CHAIN)"
if ! out="$("$CHAINBENCH" test --rpc "$L2_RPC" --chain "$L2_CHAIN" "${args[@]}" 2>&1)"; then
  echo "$out"
  echo "FAIL: some L2 cases reported failures"
  exit 1
fi
echo "$out"
# The selected cases are chain-agnostic; a skip means the endpoint lacks the rpc
# capability or the case did not apply — surface it rather than pass silently.
if echo "$out" | grep -qE 'skip=[1-9]'; then
  echo "FAIL: some L2 cases skipped (endpoint capability problem)"
  exit 1
fi
log "PASS: chain-agnostic read cases green against the L2 endpoint"

# Write side (RT-Z-02 send tx, RT-Z-05 fee delegation): the chain-agnostic write
# cases need a funded account key on the L2, supplied via CHAINBENCH_FUNDED_KEY
# (env only — never a literal). Without it, they skip.
if [ -n "${CHAINBENCH_FUNDED_KEY:-}" ]; then
  log "run chain-agnostic write cases (funded key present)"
  if ! wout="$("$CHAINBENCH" test --rpc "$L2_RPC" --chain "$L2_CHAIN" \
    --name external-value-transfer --name external-fee-delegated-transfer 2>&1)"; then
    echo "$wout"
    echo "FAIL: L2 write cases reported failures"
    exit 1
  fi
  echo "$wout"
  # A skip here means the funded key did not take or 0x16 is unsupported; the
  # value-transfer case at least must have run.
  if echo "$wout" | grep -qE 'pass=0'; then
    echo "FAIL: no write case ran (funded key not applied?)"
    exit 1
  fi
  log "PASS: L2 write cases ran with the funded key"
else
  log "SKIP write side: set CHAINBENCH_FUNDED_KEY to run RT-Z-02/RT-Z-05 on the L2"
fi
