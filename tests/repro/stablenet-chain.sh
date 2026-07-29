#!/usr/bin/env bash
# stablenet-chain.sh — live smoke test of a go-stablenet chain (scenario 4): boot
# a stablenet network, confirm it produces blocks, then verify transaction
# processing and contract deploy/call. This is the basic end-to-end companion to
# the more specific stablenet-*.sh scenarios (sync gap, basefee, consensus
# lifecycle, forks), mirroring wbft-chain.sh for the stablenet binary.
#
# Needs a real gstable binary (not runnable in CI). Transaction/contract
# verification also needs a genesis-funded key.
#
# Requirements (overridable via env):
#   GSTABLE_BIN  go-stablenet node binary (or gstable on PATH)
#   FAUCET_PK    a genesis-funded private key (env only — never a literal)
#   python3
#
#   GSTABLE_BIN=/path/to/gstable FAUCET_PK=0x... tests/repro/stablenet-chain.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-stablenet-chain-bin}"
WORK="${WORK:-/tmp/stablenet-chain}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"
ADVANCE="${ADVANCE:-10}"
RETURNS42="0x600a600c600039600a6000f3602a60005260206000f3"
DEST="${DEST:-0x000000000000000000000000000000000000dEaD}"
XFER="${XFER:-1000000000000000000}"

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT

command -v python3 >/dev/null || { echo "missing: python3"; exit 2; }
[ -n "$GSTABLE_BIN" ] && [ -x "$GSTABLE_BIN" ] || { echo "no gstable binary (set GSTABLE_BIN=/path/to/gstable)"; exit 2; }
[ -n "${FAUCET_PK:-}" ] || { echo "set FAUCET_PK to a genesis-funded private key (env only)"; exit 2; }
[ -x "$CHAINBENCH" ] || { echo "building chainbench"; (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2; }

cleanup; sleep 1
rm -rf "$WORK"; mkdir -p "$WORK"

rpc_url() {
  python3 -c "import json; ns=json.load(open('$WORK/nodeset.json')); print(next(n['rpc_url'] for n in ns['nodes'] if n['index']==$1))"
}
head_at() {
  local hex
  hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_blockNumber 2>/dev/null | tr -d '"')" || return 0
  python3 -c "s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else -1)"
}
advanced() { local before after; before="$(head_at "$1")"; sleep "$2"; after="$(head_at "$1")"; [ "$after" -gt "$before" ]; }
wait_advancing() { # <url> <timeout_s> — succeeds as soon as head grows past its first sample
  local url="$1" timeout="${2:-45}" start end t=0
  start="$(head_at "$url")"
  while [ "$t" -lt "$timeout" ]; do
    sleep 3; t=$((t + 3))
    end="$(head_at "$url")"
    [ "$end" -gt "$start" ] && return 0
  done
  return 1
}
balance_at() {
  local hex
  hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_getBalance --params "[\"$2\",\"latest\"]" 2>/dev/null | tr -d '"')"
  python3 -c "s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else 0)"
}

log "boot stablenet ($VALIDATORS validators + $ENDPOINTS endpoint)"
"$CHAINBENCH" setup --launch \
  --chain stablenet \
  --binary "$GSTABLE_BIN" \
  --data-dir "$WORK" \
  --keys-dir "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" || { echo "setup --launch failed"; exit 1; }

URL="$(rpc_url 1)"
log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

# 1. block production (poll — WBFT consensus can take ~10-15s to warm up)
if ! wait_advancing "$URL" 45; then echo "FAIL: stablenet chain not producing blocks"; exit 1; fi
log "block production OK (head advancing)"

# 2. transaction processing: value transfer, assert receipt success + credited
log "send value transfer ($XFER wei -> $DEST)"
before="$(balance_at "$URL" "$DEST")"
HASH="$("$CHAINBENCH" tx send --chain stablenet --rpc "$URL" --from-key "$FAUCET_PK" --to "$DEST" --value "$XFER")" \
  || { echo "FAIL: tx send errored"; exit 1; }
echo "tx hash: $HASH"
ST="$("$CHAINBENCH" tx wait --rpc "$URL" --hash "$HASH" 2>/dev/null | awk '/^status:/{print $2}')"
[ "$ST" = "success" ] || { echo "FAIL: tx receipt status=$ST (want success)"; exit 1; }
after="$(balance_at "$URL" "$DEST")"
python3 -c "exit(0 if $after >= $before + $XFER else 1)" || { echo "FAIL: recipient not credited ($before -> $after)"; exit 1; }
log "tx processing OK (receipt success + recipient credited)"

# 3. contract deploy + call (returns 42)
log "deploy returns-42 contract + eth_call"
OUT="$("$CHAINBENCH" contract deploy --chain stablenet --rpc "$URL" --from-key "$FAUCET_PK" --bytecode "$RETURNS42")" \
  || { echo "FAIL: contract deploy errored"; exit 1; }
ADDR="$(echo "$OUT" | awk '/^contract:/{print $2}')"
[ -n "$ADDR" ] || { echo "FAIL: no contract address in deploy output: $OUT"; exit 1; }
echo "contract: $ADDR"
ok=false
for _ in $(seq 1 30); do
  RES="$("$CHAINBENCH" contract call --rpc "$URL" --to "$ADDR" --data 0x 2>/dev/null | tr -d '"')"
  case "$RES" in *2a) ok=true; break;; esac
  sleep 3
done
[ "$ok" = true ] || { echo "FAIL: contract eth_call did not return 42 (last=$RES)"; exit 1; }
log "contract OK (deployed code returns 42)"

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: stablenet chain — block production + tx processing + contract deploy/call"
