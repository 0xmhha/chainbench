#!/usr/bin/env bash
# wbft-chain.sh — live smoke test of a FRESH go-wbft chain (scenario 3): boot a
# wbft network from genesis (static bootstrap, WBFT consensus family), confirm it
# produces blocks, then verify transaction processing and contract deploy/call.
#
# Unlike the wemix->wbft handoff (which starts on go-wemix+etcd and hands off at
# croissant), this boots go-wbft directly from block 0 with its validators in the
# genesis — the "wbft from scratch" case.
#
# Needs a real gwbft binary (not runnable in CI), like the other tests/repro
# scripts. Transaction/contract verification also needs a genesis-funded key.
#
# Requirements (overridable via env):
#   WBFT_BIN     go-wbft node binary (or gwbft on PATH)
#   FAUCET_PK    a genesis-funded private key (env only — never a literal)
#   python3
#
#   WBFT_BIN=/path/to/gwbft FAUCET_PK=0x... tests/repro/wbft-chain.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WBFT_BIN="${WBFT_BIN:-$(command -v gwbft || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-wbft-chain-bin}"
WORK="${WORK:-/tmp/wbft-chain}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"
ADVANCE="${ADVANCE:-10}"
# minimal "return 42" contract: any eth_call returns 0x..2a (shared fixture,
# tests/wbft/accounts/contract_roundtrip.go).
RETURNS42="0x600a600c600039600a6000f3602a60005260206000f3"
DEST="${DEST:-0x000000000000000000000000000000000000dEaD}"
XFER="${XFER:-1000000000000000000}" # 1e18 wei

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT

command -v python3 >/dev/null || { echo "missing: python3"; exit 2; }
[ -n "$WBFT_BIN" ] && [ -x "$WBFT_BIN" ] || { echo "no gwbft binary (set WBFT_BIN=/path/to/gwbft)"; exit 2; }
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
advanced() { # <url> <seconds> — true if head grew over the window
  local before after
  before="$(head_at "$1")"; sleep "$2"; after="$(head_at "$1")"
  [ "$after" -gt "$before" ]
}
balance_at() { # <url> <addr> -> decimal wei
  local hex
  hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_getBalance --params "[\"$2\",\"latest\"]" 2>/dev/null | tr -d '"')"
  python3 -c "s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else 0)"
}

log "boot fresh wbft chain ($VALIDATORS validators + $ENDPOINTS endpoint)"
"$CHAINBENCH" setup --launch \
  --chain wbft \
  --binary "$WBFT_BIN" \
  --data-dir "$WORK" \
  --keys-dir "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" || { echo "setup --launch failed"; exit 1; }

URL="$(rpc_url 1)"
log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

# 1. block production
if ! advanced "$URL" "$ADVANCE"; then
  echo "FAIL: wbft chain not producing blocks"; exit 1
fi
log "block production OK (head advancing)"

# 2. transaction processing: value transfer, assert receipt success + credited
log "send value transfer ($XFER wei -> $DEST)"
before="$(balance_at "$URL" "$DEST")"
HASH="$("$CHAINBENCH" tx send --chain wbft --rpc "$URL" --from-key "$FAUCET_PK" --to "$DEST" --value "$XFER")" \
  || { echo "FAIL: tx send errored"; exit 1; }
echo "tx hash: $HASH"
ST="$("$CHAINBENCH" tx wait --rpc "$URL" --hash "$HASH" 2>/dev/null | awk '/^status:/{print $2}')"
[ "$ST" = "success" ] || { echo "FAIL: tx receipt status=$ST (want success)"; exit 1; }
after="$(balance_at "$URL" "$DEST")"
python3 -c "exit(0 if $after >= $before + $XFER else 1)" || { echo "FAIL: recipient not credited ($before -> $after)"; exit 1; }
log "tx processing OK (receipt success + recipient credited)"

# 3. contract deploy + call (returns 42)
log "deploy returns-42 contract + eth_call"
OUT="$("$CHAINBENCH" contract deploy --chain wbft --rpc "$URL" --from-key "$FAUCET_PK" --bytecode "$RETURNS42")" \
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
log "PASS: fresh wbft chain — block production + tx processing + contract deploy/call"
