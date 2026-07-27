#!/usr/bin/env bash
# stablenet-block-propagation.sh — live reproduction of the block-fetcher
# near-head propagation case (regression a-ethereum a1-07), driving the built
# chainbench CLI against a real gstable binary.
#
# During normal operation each newly-produced block reaches all peers in real
# time via the Block Fetcher (NewBlock / NewBlockHashes), so every node stays
# within a couple of blocks of the producer's head. This boots a stablenet and,
# over several rounds, asserts each node's lag behind node1's head is <= LAG.
#
# Needs a real gstable binary (not runnable in CI), like the other tests/repro
# scripts.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3.
#
#   GSTABLE_BIN=/path/to/gstable tests/repro/stablenet-block-propagation.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-block-propagation-bin}"
WORK="${WORK:-/tmp/stablenet-block-propagation}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"
ROUNDS="${ROUNDS:-5}"
LAG="${LAG:-2}" # max allowed lag behind node1's head

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT

command -v python3 >/dev/null || {
  echo "missing: python3"
  exit 2
}
[ -n "$GSTABLE_BIN" ] && [ -x "$GSTABLE_BIN" ] || {
  echo "no gstable binary (set GSTABLE_BIN=/path/to/gstable)"
  exit 2
}
[ -x "$CHAINBENCH" ] || {
  echo "building chainbench"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2
}

cleanup
sleep 1
rm -rf "$WORK"
mkdir -p "$WORK"

node_indexes() {
  python3 -c "import json; ns=json.load(open('$WORK/nodeset.json')); print(' '.join(str(n['index']) for n in sorted(ns['nodes'], key=lambda x: x['index'])))"
}
rpc_url() {
  python3 -c "import json; ns=json.load(open('$WORK/nodeset.json')); print(next(n['rpc_url'] for n in ns['nodes'] if n['index']==$1))"
}
head_at() {
  local hex
  hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_blockNumber 2>/dev/null | tr -d '"')" || return 0
  python3 -c "s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else -1)"
}

log "boot stablenet ($VALIDATORS validators + $ENDPOINTS endpoint)"
"$CHAINBENCH" setup --launch \
  --chain stablenet \
  --binary "$GSTABLE_BIN" \
  --data-dir "$WORK" \
  --keys-dir "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" || {
  echo "setup --launch failed"
  exit 1
}

INDEXES="$(node_indexes)"
BP_URL="$(rpc_url 1)"

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

log "check near-head propagation over $ROUNDS rounds (lag <= $LAG)"
for r in $(seq 1 "$ROUNDS"); do
  bp="$(head_at "$BP_URL")"
  for idx in $INDEXES; do
    [ "$idx" = 1 ] && continue
    h="$(head_at "$(rpc_url "$idx")")"
    lag=$((bp - h))
    [ "$lag" -lt 0 ] && lag=$((-lag))
    if [ "$lag" -gt "$LAG" ]; then
      echo "FAIL: round $r node$idx lag=$lag > $LAG (bp=$bp node=$h) — block not propagated near-head"
      exit 1
    fi
  done
  printf '  round %s: all nodes within %s of head %s\n' "$r" "$LAG" "$bp"
  sleep 2
done

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: blocks propagate near-head to all nodes (a1-07)"
