#!/usr/bin/env bash
# wemix-chain.sh — live smoke test of a PURE go-wemix chain (scenario 1): a
# wemix+etcd network running entirely on go-wemix (poa consensus + governance +
# etcd), with NO croissant handoff. It boots a producer, deploys governance and
# initializes etcd, confirms block production, then verifies tx processing and
# contract deploy/call.
#
# This is the wemix half of wemix-wbft-handoff.sh WITHOUT the wbft successor: the
# producer mines the whole chain (PoA, diff=1). The genesis has no croissant fork,
# so the go-wemix engine stays in charge throughout.
#
# Requirements (paths/env overridable): WEMIX_BIN, TEMPLATE (wemix genesis
# template), etcd/jq/python3 on PATH, python3 web3/eth-account (to derive and fund
# the FAUCET_PK address), and FAUCET_PK (a private key — env only, never a literal;
# its address is funded in the genesis so it can send tx / deploy contracts).
#
#   WEMIX_BIN=/path/gwemix TEMPLATE=/path/genesis-template.json FAUCET_PK=0x... \
#     tests/repro/wemix-chain.sh
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WEMIX_BIN="${WEMIX_BIN:-$(command -v gwemix || true)}"
TEMPLATE="${TEMPLATE:-}"
PRESET="$REPO/keys/preset"
WORK="${WORK:-/tmp/wemix-chain}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-wemix-chain-bin}"

CHAINID="${CHAINID:-8285}"
PRODUCER_ACCT="0xf9593d358b373d354a348c00887b914b408f6984" # preset node5 unlockable keystore
STAKE=2000000000000000000000000000
BAL=1000000000000000000000000000
RETURNS42="0x600a600c600039600a6000f3602a60005260206000f3"
DEST="${DEST:-0x000000000000000000000000000000000000dEaD}"
XFER="${XFER:-1000000000000000000}"

log() { printf '\n=== %s ===\n' "$*"; }
cleanup() { pkill -9 -f "$WORK" 2>/dev/null || true; }
trap cleanup EXIT
require() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1"; exit 2; }; }

require jq; require python3; require etcd
[ -n "$WEMIX_BIN" ] && [ -x "$WEMIX_BIN" ] || { echo "no wemix binary (set WEMIX_BIN=/path/to/gwemix)"; exit 2; }
[ -n "$TEMPLATE" ] && [ -f "$TEMPLATE" ] || { echo "set TEMPLATE to the wemix genesis template json"; exit 2; }
[ -n "${FAUCET_PK:-}" ] || { echo "set FAUCET_PK to a private key (env only; its address is funded in genesis)"; exit 2; }
python3 -c "import eth_account" 2>/dev/null || { echo "missing python3 web3/eth-account (to derive the faucet address)"; exit 2; }
[ -x "$CHAINBENCH" ] || { echo "building chainbench"; (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2; }

cleanup; sleep 1
rm -rf "$WORK"; mkdir -p "$WORK"

p2p() { echo 30010; }
http() { echo 40010; }
FAUCET_ADDR="$(FAUCET_PK="$FAUCET_PK" python3 -c "import os; from eth_account import Account; print(Account.from_key(os.environ['FAUCET_PK']).address)")"
log "faucet address (funded in genesis): $FAUCET_ADDR"

# --- 1. wemix governance config (single producer = preset node5) --------------
log "build wemix governance config (pure wemix, no croissant)"
PROD_ID="0x$(python3 -c "import json;print([n['publicKey'] for n in json.load(open('$PRESET/metadata.json'))['nodes'] if n['index']==5][0])")"
ACCOUNTS_JSON="$(FAUCET_ADDR="$FAUCET_ADDR" python3 - <<PY
import json, os
accts=[{"addr":"$PRODUCER_ACCT","balance":$BAL},
       {"addr":os.environ["FAUCET_ADDR"],"balance":$BAL}]
for n in json.load(open("$PRESET/metadata.json"))["nodes"]:
    accts.append({"addr":n["address"],"balance":$BAL})
print(json.dumps(accts))
PY
)"
cat > "$WORK/wemix-config.json" <<JSON
{
  "extraData": "chainbench pure wemix E2E",
  "staker": "$PRODUCER_ACCT", "ecosystem": "$PRODUCER_ACCT",
  "maintenance": "$PRODUCER_ACCT", "feecollector": "$PRODUCER_ACCT",
  "env": {
    "ballotDurationMin": 86400, "ballotDurationMax": 604800,
    "stakingMin": 1500000000000000000000000, "stakingMax": 1500000000000000000000000000,
    "MaxIdleBlockInterval": 5, "blockCreationTime": 1000,
    "blockRewardAmount": 1000000000000000000, "maxPriorityFeePerGas": 100000000000,
    "rewardDistributionMethod": [4000,1000,2500,2500],
    "maxBaseFee": 50000000000000, "blockGasLimit": 105000000,
    "baseFeeMaxChangeRate": 55, "gasTargetPercentage": 30
  },
  "members": [
    { "addr": "$PRODUCER_ACCT", "stake": $STAKE, "name": "node1",
      "id": "$PROD_ID", "ip": "127.0.0.1", "port": $(p2p), "bootnode": true }
  ],
  "accounts": $ACCOUNTS_JSON
}
JSON

# --- 2. wemix base genesis (NO croissant merge — pure wemix) -------------------
log "wemix genesis (chainId=$CHAINID)"
"$WEMIX_BIN" wemix genesis --data "$WORK/wemix-config.json" --genesis "$TEMPLATE" --out "$WORK/base.json" \
  || { echo "wemix genesis failed"; exit 1; }
jq ".config.chainId=$CHAINID | .config.petersburgBlock=0" "$WORK/base.json" > "$WORK/genesis.json"

# --- 3. init + provision the producer -----------------------------------------
log "init producer node1"
d="$WORK/node1"; mkdir -p "$d"
"$WEMIX_BIN" --datadir "$d" init "$WORK/genesis.json" >/dev/null 2>&1 || { echo "init failed"; exit 1; }
mkdir -p "$d/geth"
cat "$PRESET/node5/nodekey" > "$d/geth/nodekey"
mkdir -p "$d/keystore"; cp "$PRESET/node5/keystore/UTC--node5-keystore" "$d/keystore/"
echo -n "1" > "$WORK/pw"

# --- 4. launch producer (go-wemix, mining) ------------------------------------
log "launch producer"
"$WEMIX_BIN" --datadir "$d" --networkid "$CHAINID" --nat none \
  --port "$(p2p)" --http --http.addr 127.0.0.1 --http.port "$(http)" \
  --http.api eth,net,web3,wemix,admin,miner,txpool,personal \
  --authrpc.port $(( $(http) + 2 )) --syncmode full \
  --mine --miner.etherbase "$PRODUCER_ACCT" \
  --unlock "$PRODUCER_ACCT" --password "$WORK/pw" --allow-insecure-unlock \
  --verbosity 3 > "$d/node.log" 2>&1 &

URL="http://127.0.0.1:$(http)"
for _ in $(seq 1 20); do [ -S "$d/gwemix.ipc" ] && break; sleep 1; done
[ -S "$d/gwemix.ipc" ] || { echo "producer IPC never appeared; tail:"; tail -15 "$d/node.log"; exit 1; }

# --- 5. deploy governance + etcdInit ------------------------------------------
log "deploy governance + etcdInit"
"$WEMIX_BIN" wemix deploy-governance --url "$d/gwemix.ipc" --password "$WORK/pw" \
  "$WORK/wemix-config.json" "$d/keystore/UTC--node5-keystore" > "$WORK/deploy-gov.log" 2>&1 \
  && echo "governance deployed" || { echo "deploy-governance failed (see $WORK/deploy-gov.log)"; tail -10 "$WORK/deploy-gov.log"; exit 1; }
"$WEMIX_BIN" attach "$d/gwemix.ipc" --exec "admin.etcdInit()" > "$WORK/etcd-init.log" 2>&1 \
  && echo "etcdInit done" || { echo "etcdInit failed (see $WORK/etcd-init.log)"; tail -10 "$WORK/etcd-init.log"; exit 1; }

head_at() {
  local hex; hex="$("$CHAINBENCH" node rpc --rpc "$1" --method eth_blockNumber 2>/dev/null | tr -d '"')" || return 0
  python3 -c "s='$hex'.strip(); print(int(s,16) if s.startswith('0x') else -1)"
}

# --- 6. block production -------------------------------------------------------
log "wait for block production"
ok=false
for _ in $(seq 1 30); do
  h="$(head_at "$URL")"
  [ "$h" -ge 2 ] && { ok=true; echo "head=$h"; break; }
  sleep 3
done
[ "$ok" = true ] || { echo "FAIL: wemix chain not producing blocks"; tail -20 "$d/node.log"; exit 1; }
log "block production OK"

# --- 7. tx processing ----------------------------------------------------------
log "send value transfer ($XFER wei -> $DEST)"
HASH="$("$CHAINBENCH" tx send --chain wemix --rpc "$URL" --from-key "$FAUCET_PK" --to "$DEST" --value "$XFER")" \
  || { echo "FAIL: tx send errored"; exit 1; }
echo "tx hash: $HASH"
ST="$("$CHAINBENCH" tx wait --rpc "$URL" --hash "$HASH" 2>/dev/null | awk '/^status:/{print $2}')"
[ "$ST" = "success" ] || { echo "FAIL: tx receipt status=$ST (want success)"; exit 1; }
log "tx processing OK"

# --- 8. contract deploy + call -------------------------------------------------
log "deploy returns-42 contract + eth_call"
OUT="$("$CHAINBENCH" contract deploy --chain wemix --rpc "$URL" --from-key "$FAUCET_PK" --bytecode "$RETURNS42")" \
  || { echo "FAIL: contract deploy errored"; exit 1; }
ADDR="$(echo "$OUT" | awk '/^contract:/{print $2}')"
[ -n "$ADDR" ] || { echo "FAIL: no contract address: $OUT"; exit 1; }
ok=false
for _ in $(seq 1 30); do
  RES="$("$CHAINBENCH" contract call --rpc "$URL" --to "$ADDR" --data 0x 2>/dev/null | tr -d '"')"
  case "$RES" in *2a) ok=true; break;; esac
  sleep 3
done
[ "$ok" = true ] || { echo "FAIL: contract eth_call did not return 42 (last=$RES)"; exit 1; }
log "contract OK (deployed code returns 42 at $ADDR)"

log "PASS: pure wemix chain — governance/etcd bootstrap + block production + tx + contract"
