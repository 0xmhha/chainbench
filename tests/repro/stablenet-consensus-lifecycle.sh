#!/usr/bin/env bash
# stablenet-consensus-lifecycle.sh — live reproduction of the WBFT consensus
# liveness/lifecycle cases (regression b-wbft b-08 quorum-deficient, b-09
# round-change, b-10 post-round-change parentHash, a-ethereum a1-04 restart),
# driving the built chainbench CLI and its per-node lifecycle commands against a
# real gstable binary.
#
# With 4 validators, WBFT tolerates f=1 fault and needs 2f+1=3 to commit:
#   - stopping 1 validator (3 remain, >= quorum) → consensus continues (b-09),
#     and the chain stays linked (b-10); restarting it resumes (a1-04).
#   - stopping 2 validators (2 remain, < quorum) → block production halts (b-08);
#     restarting them recovers.
#
# Needs a real gstable binary (not runnable in CI), like the other tests/repro
# scripts.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3.
#
#   GSTABLE_BIN=/path/to/gstable tests/repro/stablenet-consensus-lifecycle.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-consensus-lifecycle-bin}"
WORK="${WORK:-/tmp/stablenet-consensus-lifecycle}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"
ADVANCE="${ADVANCE:-10}" # seconds to observe head movement
HALT_WINDOW="${HALT_WINDOW:-12}" # seconds to confirm production halted

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

rpc_url() {
  python3 -c "import json; ns=json.load(open('$WORK/nodeset.json')); print(next(n['rpc_url'] for n in ns['nodes'] if n['index']==$1))"
}
head_at() {
  local hex
  hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_blockNumber 2>/dev/null | tr -d '"')" || return 0
  python3 -c "s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else -1)"
}
block_field() {
  "$CHAINBENCH" node rpc --rpc "$1" --method eth_getBlockByNumber \
    --params "[\"$2\", false]" 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin); print((d or {}).get('$3',''))"
}
advanced() { # <url> <seconds> — true if head grew over the window
  local before after
  before="$(head_at "$1")"
  sleep "$2"
  after="$(head_at "$1")"
  [ "$after" -gt "$before" ]
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

BP_URL="$(rpc_url 1)"
OBS_URL="$(rpc_url "$((VALIDATORS + 1))")" # an endpoint, to observe when node1 is down

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

if ! advanced "$OBS_URL" "$ADVANCE"; then
  echo "FAIL: baseline — head not advancing before any stop"
  exit 1
fi
log "baseline head advancing OK"

# b-09: stop one validator; the remaining 3 (>= quorum 3) keep producing.
log "stop validator node1 (round change; 3 of 4 remain)"
"$CHAINBENCH" node stop --data-dir "$WORK" --index 1 || {
  echo "node stop failed"
  exit 1
}
if ! advanced "$OBS_URL" "$ADVANCE"; then
  echo "FAIL: b-09 — consensus halted with only 1 validator down"
  exit 1
fi
log "b-09 round-change OK: production continues with node1 down"

# b-10: the chain stays linked across the round change (parentHash chain).
tip="$(head_at "$OBS_URL")"
ok=true
for n in "$tip" "$((tip - 1))" "$((tip - 2))"; do
  [ "$n" -lt 1 ] && continue
  h_child="$(block_field "$OBS_URL" "$(python3 -c "print(hex($n))")" parentHash)"
  h_parent="$(block_field "$OBS_URL" "$(python3 -c "print(hex($n - 1))")" hash)"
  if [ -z "$h_parent" ] || [ "$h_child" != "$h_parent" ]; then
    ok=false
    echo "parentHash mismatch at block $n: child.parentHash=$h_child parent.hash=$h_parent"
  fi
done
[ "$ok" = true ] || {
  echo "FAIL: b-10 — parentHash chain broken after round change"
  exit 1
}
log "b-10 parentHash chain intact"

# a1-04: restart node1; it rejoins and production continues.
log "restart validator node1"
"$CHAINBENCH" node start --data-dir "$WORK" --index 1 || {
  echo "node start failed"
  exit 1
}
if ! advanced "$OBS_URL" "$ADVANCE"; then
  echo "FAIL: a1-04 — head not advancing after node1 restart"
  exit 1
fi
log "a1-04 restart OK: node1 rejoined, production continues"

# b-08: stop two validators (2 remain, < quorum 3) → production halts.
log "stop validators node1 + node2 (quorum deficient; 2 of 4 remain)"
"$CHAINBENCH" node stop --data-dir "$WORK" --index 1 || true
"$CHAINBENCH" node stop --data-dir "$WORK" --index 2 || true
if advanced "$OBS_URL" "$HALT_WINDOW"; then
  echo "FAIL: b-08 — production continued below quorum (should halt)"
  exit 1
fi
log "b-08 quorum-deficient OK: production halted"

# recovery: restart both → production resumes.
log "restart node1 + node2 (recover quorum)"
"$CHAINBENCH" node start --data-dir "$WORK" --index 1 || true
"$CHAINBENCH" node start --data-dir "$WORK" --index 2 || true
if ! advanced "$OBS_URL" "$((ADVANCE + 10))"; then
  echo "FAIL: recovery — production did not resume after restoring quorum"
  exit 1
fi

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: round-change (b-09) + parentHash (b-10) + restart (a1-04) + quorum-halt/recovery (b-08)"
