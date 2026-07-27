#!/usr/bin/env bash
# stablenet-basefee-dynamics.sh — live reproduction of the anzeon baseFee
# dynamics cases (regression c-anzeon c-03 increase, c-05 decrease), driving the
# built chainbench CLI against a real gstable binary.
#
# baseFee reacts to block gas usage: usage > 20% raises the next block's baseFee
# (~+2%), usage < 6% lowers it (~-2%). This boots a stablenet, bursts many
# transfers from the faucet to push at least one block over 20% and asserts a
# later block's baseFee rose, then lets the chain idle and asserts baseFee fell.
#
# These are inherently LOAD- and TIMING-sensitive (you cannot force a block to a
# precise usage band), so this lives in the manual repro tier — not as a testkit
# case — and needs a real gstable binary (not runnable in CI). The regular-account
# gas-tip forcing case (c-01) is ported as a deterministic testkit case
# (tests/anzeon/gastip.go); c-02 (authorized free tip) needs an authorized
# account key and is deferred.
#
# Requirements (overridable via env): GSTABLE_BIN (or gstable on PATH), python3
# with web3/eth_account, and FAUCET_PK (the genesis-funded faucet private key —
# supplied via env so no key literal lives in the repo).
#
#   GSTABLE_BIN=/path/to/gstable FAUCET_PK=0x... tests/repro/stablenet-basefee-dynamics.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GSTABLE_BIN="${GSTABLE_BIN:-$(command -v gstable || true)}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-basefee-bin}"
WORK="${WORK:-/tmp/stablenet-basefee-dynamics}"
VALIDATORS="${VALIDATORS:-4}"
ENDPOINTS="${ENDPOINTS:-1}"
SETTLE="${SETTLE:-15}"
IDLE="${IDLE:-20}" # seconds to idle so low-usage blocks lower the baseFee

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT

command -v python3 >/dev/null || {
  echo "missing: python3"
  exit 2
}
[ -n "${FAUCET_PK:-}" ] || {
  echo "set FAUCET_PK to the genesis-funded faucet private key"
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

RPC_URL="$(python3 -c "import json; ns=json.load(open('$WORK/nodeset.json')); print(next(n['rpc_url'] for n in ns['nodes'] if n['index']==1))")"

log "settle ${SETTLE}s for boot + peering"
sleep "$SETTLE"

log "burst load to push a block over 20% gas usage, then observe baseFee"
FAUCET_PK="$FAUCET_PK" RPC_URL="$RPC_URL" python3 <<'PYEOF'
import os, time
from web3 import Web3

w3 = Web3(Web3.HTTPProvider(os.environ["RPC_URL"]))
acct = w3.eth.account.from_key(os.environ["FAUCET_PK"])
to = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

limit = w3.eth.get_block("latest")["gasLimit"]
target = limit * 25 // 100
n = max(1, target * 3 // 21000 // 2)  # ~1.5x margin over 20%
base = w3.eth.get_block("latest")["baseFeePerGas"]
chain_id = w3.eth.chain_id
nonce = w3.eth.get_transaction_count(acct.address, "pending")

start_block = w3.eth.block_number
base_before = base
for i in range(n):
    tx = {"nonce": nonce + i, "to": to, "value": 1, "gas": 21000, "chainId": chain_id,
          "maxFeePerGas": base + 100_000_000_000_000, "maxPriorityFeePerGas": 27_600_000_000_000, "type": 2}
    w3.eth.send_raw_transaction(acct.sign_transaction(tx).raw_transaction)

# Scan the next blocks for one over 20% usage and check its successor's baseFee.
increased = False
for bn in range(start_block + 1, start_block + 12):
    while w3.eth.block_number < bn + 1:
        time.sleep(1)
    blk = w3.eth.get_block(bn)
    usage = blk["gasUsed"] * 100 / limit
    if usage > 20:
        nxt = w3.eth.get_block(bn + 1)["baseFeePerGas"]
        print(f"block {bn}: usage={usage:.1f}% baseFee {blk['baseFeePerGas']} -> next {nxt}")
        if nxt > blk["baseFeePerGas"]:
            increased = True
            break
assert increased, "no block over 20% usage raised the next baseFee (increase c-03)"
print("c-03 baseFee increase: OK")
PYEOF
[ $? -eq 0 ] || {
  echo "FAIL: baseFee increase (c-03)"
  exit 1
}

log "idle ${IDLE}s so low-usage blocks lower the baseFee"
peak="$("$CHAINBENCH" node rpc --rpc "$RPC_URL" --method eth_getBlockByNumber --params '["latest",false]' 2>/dev/null | python3 -c "import json,sys; print(int(json.load(sys.stdin)['baseFeePerGas'],16))")"
sleep "$IDLE"
low="$("$CHAINBENCH" node rpc --rpc "$RPC_URL" --method eth_getBlockByNumber --params '["latest",false]' 2>/dev/null | python3 -c "import json,sys; print(int(json.load(sys.stdin)['baseFeePerGas'],16))")"
log "baseFee after idle: peak=$peak low=$low"
if [ "$low" -ge "$peak" ]; then
  echo "FAIL: baseFee did not decrease while idle (decrease c-05): peak=$peak low=$low"
  exit 1
fi

"$CHAINBENCH" stop --data-dir "$WORK" >/dev/null 2>&1 || true
log "PASS: baseFee increase (c-03) + decrease (c-05) observed"
