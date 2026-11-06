#!/usr/bin/env bash
# attach-external.sh — run chainbench's chain-agnostic cases against an external
# chain: one already running somewhere else that chainbench attaches to over RPC
# (it does not launch or manage it), needing no chain binary — just an RPC URL.
#
# This covers the legacy regression "Layer 2 E2E" section. Despite that name it is
# NOT an Ethereum L2 / rollup and needs no L2 support in chainbench: it meant
# generic RPC operations run against some external chain. Those are the common
# read cases below, so this script points them at an external RPC and asserts
# none is skipped.
#
# Sending from a funded account is not covered here. A write case needs a key file
# for that chain and belongs in a case of its own.
#
# Requirements: EXTERNAL_RPC (the external chain's JSON-RPC URL) and EXTERNAL_CHAIN
# (the chain it runs: stablenet, wbft or wemix). There is no default, because the
# chain decides how the run reads the endpoint.
#
#   EXTERNAL_RPC=https://chain.example/rpc EXTERNAL_CHAIN=wbft tests/repro/attach-external.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-attach-external-bin}"
EXTERNAL_CHAIN="${EXTERNAL_CHAIN:-}"

# Read cases that only need the standard RPC namespaces. txpool stays out: an
# external endpoint often does not expose it.
CASES=(
  tests/tc/common/fee/CT-FEE-010-suggested-gas-price.json
  tests/tc/common/fee/CT-FEE-012-fee-history.json
  tests/tc/common/node/CT-NODE-014-sync-complete.json
)

log() { printf '\n=== %s ===\n' "$*"; }

[ -n "${EXTERNAL_RPC:-}" ] || {
  echo "set EXTERNAL_RPC to the external chain's JSON-RPC URL"
  exit 2
}
[ -n "$EXTERNAL_CHAIN" ] || {
  echo "set EXTERNAL_CHAIN to the chain it runs: stablenet, wbft or wemix"
  exit 2
}
[ -x "$CHAINBENCH" ] || {
  echo "building chainbench"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2
}

log "run read cases against external endpoint $EXTERNAL_RPC (chain=$EXTERNAL_CHAIN)"
cd "$REPO" || exit 2
# --no-skips makes a skip fail: it means the endpoint lacks a capability.
if ! "$CHAINBENCH" run --rpc "$EXTERNAL_RPC" --chain "$EXTERNAL_CHAIN" --no-skips "${CASES[@]}"; then
  echo "FAIL: some cases failed or were skipped"
  exit 1
fi
log "PASS: read cases green against the external endpoint"
