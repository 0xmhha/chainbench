#!/usr/bin/env bash
# stablenet-hardfork-swap.sh — live smoke test of a stablenet BINARY-SWAP hardfork
# (scenario 5): boot a stablenet network on a pre-fork gstable build, produce
# blocks and deploy a contract, then stop that binary and relaunch the same data
# directories on the post-fork build via `chainbench hardfork`. The fork activates
# at the genesis-encoded block (bohoBlock); afterwards block production and tx
# processing must continue, and pre-fork state (the deployed contract) must survive.
#
# This is DISTINCT from stablenet-delayed-fork.sh: that runs a single binary whose
# fork activation is merely delayed. Here two different builds are swapped in place
# — the real "upgrade every node's binary at the fork block" case. It exercises the
# `chainbench hardfork` same-chain swap (to-chain == from-chain, distinct --to-binary).
#
# For a MEANINGFUL swap, PRE_FORK_BIN and POST_FORK_BIN should be different builds
# (pre-fork lacks the fork; post-fork activates it at bohoBlock). If POST_FORK_BIN
# is unset it defaults to PRE_FORK_BIN and the script still exercises the swap
# mechanics (relaunch in place), but the fork-behavior delta is only observable
# with two builds.
#
# Needs real gstable binaries (not runnable in CI). Requirements (env-overridable):
#   PRE_FORK_BIN   pre-fork gstable (or GSTABLE_BIN, or gstable on PATH)
#   POST_FORK_BIN  post-fork gstable (default: PRE_FORK_BIN)
#   FAUCET_PK      a genesis-funded private key (env only — never a literal)
#   BOHO_BLOCK     fork activation block (default 40; must exceed head at swap)
#   python3
#
#   PRE_FORK_BIN=/path/pre/gstable POST_FORK_BIN=/path/post/gstable \
#     FAUCET_PK=0x... tests/repro/stablenet-hardfork-swap.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PRE_FORK_BIN="${PRE_FORK_BIN:-${GSTABLE_BIN:-$(command -v gstable || true)}}"
POST_FORK_BIN="${POST_FORK_BIN:-$PRE_FORK_BIN}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-hardfork-swap-bin}"
WORK="${WORK:-/tmp/stablenet-hardfork-swap}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"
ADVANCE="${ADVANCE:-10}"
BOHO_BLOCK="${BOHO_BLOCK:-40}"
RETURNS42="0x600a600c600039600a6000f3602a60005260206000f3"
DEST="${DEST:-0x000000000000000000000000000000000000dEaD}"
XFER="${XFER:-1000000000000000000}"

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT

command -v python3 >/dev/null || { echo "missing: python3"; exit 2; }
[ -n "$PRE_FORK_BIN" ] && [ -x "$PRE_FORK_BIN" ] || { echo "no gstable binary (set PRE_FORK_BIN or GSTABLE_BIN=/path/to/gstable)"; exit 2; }
[ -x "$POST_FORK_BIN" ] || { echo "no post-fork gstable binary: $POST_FORK_BIN"; exit 2; }
[ -n "${FAUCET_PK:-}" ] || { echo "set FAUCET_PK to a genesis-funded private key (env only)"; exit 2; }
[ -x "$CHAINBENCH" ] || { echo "building chainbench"; (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2; }
[ "$PRE_FORK_BIN" = "$POST_FORK_BIN" ] && echo "note: PRE_FORK_BIN == POST_FORK_BIN — exercising swap mechanics only (no fork-behavior delta)"

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
    sleep 3; t=$((t + 3)); end="$(head_at "$url")"
    [ "$end" -gt "$start" ] && return 0
  done
  return 1
}
wait_cross() { # <url> <target> <timeout_s> — succeeds when head exceeds target
  local url="$1" target="$2" timeout="${3:-90}" t=0 h
  while [ "$t" -lt "$timeout" ]; do
    h="$(head_at "$url")"; [ "$h" -gt "$target" ] && return 0
    sleep 3; t=$((t + 3))
  done
  return 1
}
call42() { # <url> <addr> -> true if eth_call returns 42
  local r; r="$("$CHAINBENCH" contract call --rpc "$1" --to "$2" --data 0x 2>/dev/null | tr -d '"')"
  case "$r" in *2a) return 0;; *) return 1;; esac
}

log "boot stablenet on PRE-fork binary (bohoBlock=$BOHO_BLOCK)"
"$CHAINBENCH" setup --launch \
  --chain stablenet \
  --binary "$PRE_FORK_BIN" \
  --data-dir "$WORK" \
  --keys-dir "$REPO/keys/preset" \
  --validators "$VALIDATORS" --endpoints "$ENDPOINTS" \
  --set "genesis.overrides.bohoBlock=$BOHO_BLOCK" || { echo "setup --launch failed"; exit 1; }

URL="$(rpc_url 1)"
log "settle ${SETTLE}s"; sleep "$SETTLE"

# 1. pre-fork block production (poll — WBFT consensus can take ~10-15s to warm up)
if ! wait_advancing "$URL" 45; then echo "FAIL: pre-fork chain not producing blocks"; exit 1; fi
log "pre-fork block production OK"

# 2. deploy a contract BEFORE the swap; its state must survive the binary swap + fork
log "deploy returns-42 contract pre-fork"
OUT="$("$CHAINBENCH" contract deploy --chain stablenet --rpc "$URL" --from-key "$FAUCET_PK" --bytecode "$RETURNS42")" \
  || { echo "FAIL: pre-fork contract deploy errored"; exit 1; }
ADDR="$(echo "$OUT" | awk '/^contract:/{print $2}')"
[ -n "$ADDR" ] || { echo "FAIL: no contract address: $OUT"; exit 1; }
ok=false; for _ in $(seq 1 30); do call42 "$URL" "$ADDR" && { ok=true; break; }; sleep 3; done
[ "$ok" = true ] || { echo "FAIL: pre-fork contract did not return 42"; exit 1; }
log "pre-fork contract deployed at $ADDR (returns 42)"

# 3. swap must happen BEFORE the fork block, or the pre-fork binary crosses it alone
HEAD="$(head_at "$URL")"
if [ "$HEAD" -ge "$BOHO_BLOCK" ]; then
  echo "FAIL: head ($HEAD) already reached bohoBlock ($BOHO_BLOCK) before swap — raise BOHO_BLOCK"; exit 1
fi
log "head=$HEAD < bohoBlock=$BOHO_BLOCK — swap now"

# 4. BINARY SWAP: stop pre-fork nodes, relaunch same datadirs on the post-fork build
log "chainbench hardfork: stop pre-fork binary, relaunch post-fork binary in place"
"$CHAINBENCH" hardfork \
  --data-dir "$WORK" \
  --to-chain stablenet \
  --to-binary "$POST_FORK_BIN" \
  --block "$BOHO_BLOCK" \
  --dry-run=false || { echo "FAIL: hardfork swap errored"; exit 1; }

URL="$(rpc_url 1)" # ports preserved, but reload in case
log "settle for relaunch"; sleep 8

# 5. post-fork block production must continue and cross the fork block (poll)
if ! wait_cross "$URL" "$BOHO_BLOCK" 120; then
  echo "FAIL: post-fork head did not cross bohoBlock ($BOHO_BLOCK) after swap (last=$(head_at "$URL"))"; exit 1
fi
HEAD="$(head_at "$URL")"
log "post-fork production OK (head=$HEAD > bohoBlock=$BOHO_BLOCK)"

# 6. pre-fork state survives: the contract still returns 42 after the swap + fork
call42 "$URL" "$ADDR" || { echo "FAIL: pre-fork contract state lost across the swap/fork"; exit 1; }
log "state preserved OK (pre-fork contract still returns 42 post-fork)"

# 7. post-fork tx processing
log "post-fork value transfer"
HASH="$("$CHAINBENCH" tx send --chain stablenet --rpc "$URL" --from-key "$FAUCET_PK" --to "$DEST" --value "$XFER")" \
  || { echo "FAIL: post-fork tx send errored"; exit 1; }
ST="$("$CHAINBENCH" tx wait --rpc "$URL" --hash "$HASH" 2>/dev/null | awk '/^status:/{print $2}')"
[ "$ST" = "success" ] || { echo "FAIL: post-fork tx status=$ST (want success)"; exit 1; }
log "post-fork tx processing OK"

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: stablenet binary-swap hardfork — pre-fork production + swap + post-fork production/state/tx"
