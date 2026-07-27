#!/usr/bin/env bash
# stablenet-sync-gap.sh — live reproduction of the endpoint re-sync cases
# (regression a-ethereum a1-02 full-sync, a1-06 downloader-path), driving the
# built chainbench CLI and its per-node lifecycle commands against a real gstable
# binary.
#
# It boots a local stablenet, stops the endpoint node (node5) to open a block
# gap while the validators keep producing, waits for a large gap (>= GAP blocks,
# forcing the downloader path rather than the near-head fetcher), restarts the
# endpoint, and asserts it catches back up: head within 2 blocks, a matching
# block hash at a sampled height, and eth_syncing == false.
#
# This is the a1-* porting from docs/dev/HandOff.md §4 step 3. The lifecycle
# primitives (setup.StopNode/RelaunchNode) and the `node stop/start --index`
# commands are unit-tested; this script is the live end-to-end check, which needs
# a real gstable binary and so runs manually (like tests/repro/wemix-wbft-handoff.sh),
# not in CI.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3.
#
#   GSTABLE_BIN=/path/to/gstable tests/repro/stablenet-sync-gap.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-sync-gap-bin}"
WORK="${WORK:-/tmp/stablenet-sync-gap}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
ENDPOINT_IDX="${ENDPOINT_IDX:-5}" # node index of the endpoint to cycle (validators+1)
GAP="${GAP:-12}"                  # blocks to let the validators advance while node is down
SETTLE="${SETTLE:-15}"           # seconds to let the network boot + peer
SYNC_TIMEOUT="${SYNC_TIMEOUT:-60}" # seconds to wait for re-sync after restart
# Endpoint sync mode: "full" (a1-02/a1-06) or "snap" (a1-03). For snap, use a
# larger gap (>=128) so the snap pivot triggers, e.g. GAP=150 SYNCMODE=snap.
SYNCMODE="${SYNCMODE:-full}"

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

# rpc_url <index> — the RPC URL of node <index> from the saved nodeset.
rpc_url() {
  python3 -c "import json,sys; ns=json.load(open('$WORK/nodeset.json')); print(next(n['rpc_url'] for n in ns['nodes'] if n['index']==$1))"
}
# head <url> — current block number (decimal) at the given RPC URL, or -1.
head_at() {
  local hex
  hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_blockNumber 2>/dev/null | tr -d '"')" || return 0
  python3 -c "import sys; s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else -1)"
}
block_hash() { block_field "$1" "$2" hash; }
# block_field <url> <block-hex> <field> — a header field of the given block.
block_field() {
  "$CHAINBENCH" node rpc --rpc "$1" --method eth_getBlockByNumber \
    --params "[\"$2\", false]" 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin); print((d or {}).get('$3',''))"
}

log "boot stablenet ($VALIDATORS validators + $ENDPOINTS endpoint, endpoint syncmode=$SYNCMODE)"
"$CHAINBENCH" setup --launch \
  --chain stablenet \
  --binary "$GSTABLE_BIN" \
  --data-dir "$WORK" \
  --keys-dir "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" \
  --set "nodes.endpoint_syncmode=$SYNCMODE" || {
  echo "setup --launch failed"
  exit 1
}

BP_URL="$(rpc_url 1)"
EN_URL="$(rpc_url "$ENDPOINT_IDX")"

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

# Endpoint should start roughly in sync with the validators.
en0="$(head_at "$EN_URL")"
bp0="$(head_at "$BP_URL")"
log "pre-stop heads: BP1=$bp0 EN$ENDPOINT_IDX=$en0"

log "stop endpoint node$ENDPOINT_IDX to open a gap"
"$CHAINBENCH" node stop --data-dir "$WORK" --index "$ENDPOINT_IDX" || {
  echo "node stop failed"
  exit 1
}
stop_head="$(head_at "$BP_URL")"

log "wait for validators to advance >= $GAP blocks (downloader path)"
deadline=$((GAP * 3 + 20))
for _ in $(seq 1 "$deadline"); do
  cur="$(head_at "$BP_URL")"
  if [ "$cur" -ge "$((stop_head + GAP))" ]; then break; fi
  sleep 1
done
bp_before="$(head_at "$BP_URL")"
gap=$((bp_before - stop_head))
log "gap created: $gap blocks (BP1 now at $bp_before)"
if [ "$gap" -lt "$GAP" ]; then
  echo "FAIL: only $gap blocks produced, wanted >= $GAP"
  exit 1
fi

log "restart endpoint node$ENDPOINT_IDX"
"$CHAINBENCH" node start --data-dir "$WORK" --index "$ENDPOINT_IDX" || {
  echo "node start failed"
  exit 1
}

log "wait up to ${SYNC_TIMEOUT}s for endpoint to re-sync"
synced=false
for _ in $(seq 1 "$SYNC_TIMEOUT"); do
  en="$(head_at "$EN_URL")"
  bp="$(head_at "$BP_URL")"
  diff=$((bp - en))
  [ "$diff" -lt 0 ] && diff=$((-diff))
  if [ "$en" -ge 0 ] && [ "$diff" -le 2 ]; then
    synced=true
    log "endpoint re-synced: EN$ENDPOINT_IDX=$en BP1=$bp (diff=$diff)"
    break
  fi
  sleep 1
done
if [ "$synced" != true ]; then
  echo "FAIL: endpoint did not re-sync within ${SYNC_TIMEOUT}s"
  exit 1
fi

# Hash agreement at a block produced while the endpoint was down.
sample=$((bp_before - 1))
[ "$sample" -lt 1 ] && sample=1
sample_hex="$(python3 -c "print(hex($sample))")"
bp_hash="$(block_hash "$BP_URL" "$sample_hex")"
en_hash="$(block_hash "$EN_URL" "$sample_hex")"
if [ -z "$bp_hash" ] || [ "$bp_hash" != "$en_hash" ]; then
  echo "FAIL: block $sample hash mismatch (BP=$bp_hash EN=$en_hash)"
  exit 1
fi
log "block $sample hash matches on BP1 and EN$ENDPOINT_IDX: $bp_hash"

# stateRoot agreement + state access (a1-03 snap sync reconstructs the same
# state). The alloc account is a genesis-funded one that no case spends.
bp_root="$(block_field "$BP_URL" "$sample_hex" stateRoot)"
en_root="$(block_field "$EN_URL" "$sample_hex" stateRoot)"
if [ -z "$bp_root" ] || [ "$bp_root" != "$en_root" ]; then
  echo "FAIL: block $sample stateRoot mismatch (BP=$bp_root EN=$en_root)"
  exit 1
fi
en_bal="$("$CHAINBENCH" node rpc --rpc "$EN_URL" --method eth_getBalance \
  --params '["0x71562b71999873db5b286df957af199ec94617f7","latest"]' 2>/dev/null | tr -d '"')"
if [ -z "$en_bal" ] || [ "$en_bal" = "0x0" ]; then
  echo "FAIL: endpoint cannot read alloc account state (balance=$en_bal)"
  exit 1
fi
log "endpoint state access OK (stateRoot matches; alloc balance=$en_bal)"

syncing="$("$CHAINBENCH" node rpc --rpc "$EN_URL" --method eth_syncing 2>/dev/null | tr -d '"')"
if [ "$syncing" != "false" ]; then
  echo "FAIL: endpoint still reports eth_syncing=$syncing"
  exit 1
fi

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: endpoint re-sync green (syncmode=$SYNCMODE, gap=$gap, stateRoot matched)"
