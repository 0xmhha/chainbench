#!/usr/bin/env bash
# wemix-wbft-handoff.sh — live E2E reproduction of the go-wemix+etcd -> go-wbft
# croissant hardfork handoff, driving the built binaries through the chainbench
# framework (pkg/consensus/upgrade) and the golden profile.
#
# The framework's plan (chainbench upgrade genesis) puts the producer at node
# index 0 and validators after it. The preset, however, carries the producer
# identity as node5 and the four BFT (BLS) validators as node1-4. So each plan
# node is mapped to a preset identity: plan node 1 (producer) <- preset node5;
# plan nodes 2-5 (validators) <- preset node1-4.
#
# node1 (plan) = go-wemix producer (poa + etcd + governance); plan nodes 2-5 =
# go-wbft BFT validators. Every node inits the SAME merged genesis; go-wemix
# mines blocks 0..CROISSANT-1 (PoA, diff=1), then "skips mining due to Croissant"
# and the go-wbft validators take over.
#
# PASSES iff head passes croissant AND a go-wbft validator (not the go-wemix
# producer) mined a post-croissant block.
#
# Requirements (paths overridable via env): WEMIX_BIN, WBFT_BIN, TEMPLATE,
# etcd/jq/python3/curl on PATH.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WEMIX_BIN="${WEMIX_BIN:-/Users/kevin/work/github/wemade/go-wemix/build/bin/gwemix}"
WBFT_BIN="${WBFT_BIN:-/Users/kevin/work/github/wemade/go-wbft/build/bin/gwemix}"
TEMPLATE="${TEMPLATE:-/Users/kevin/work/github/wemade/go-wemix/wemix/scripts/genesis-template.json}"
PROFILE="$REPO/profiles/wemix-upgrade.yaml"
PRESET="$REPO/keys/preset"
WORK="${WORK:-/tmp/wemix-wbft-handoff}"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-bin}"

CHAINID=8285
CROISSANT=20
PRODUCER_ACCT="0xf9593d358b373d354a348c00887b914b408f6984"  # node5 unlockable keystore
STAKE=2000000000000000000000000000
BAL=1000000000000000000000000000

# plan node k (1-based) -> preset node number. producer(plan 1)=preset5.
PRESET_FOR=(_ 5 1 2 3 4)

log(){ printf '\n=== %s ===\n' "$*"; }
cleanup(){ pkill -9 -f "$WORK" 2>/dev/null; }
trap cleanup EXIT
require(){ command -v "$1" >/dev/null 2>&1 || { echo "missing: $1"; exit 2; }; }
require jq; require python3; require curl; require etcd
[ -x "$WEMIX_BIN" ] || { echo "no wemix binary: $WEMIX_BIN"; exit 2; }
[ -x "$WBFT_BIN" ]  || { echo "no wbft binary: $WBFT_BIN"; exit 2; }
[ -x "$CHAINBENCH" ] || { echo "building chainbench"; (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || exit 2; }

# Optional (scenario 2 — state preservation + post-fork operation): a funded key.
# When FAUCET_PK is set its address is funded in genesis (pre-fork state); after
# the handoff we assert that balance survives on the wbft successor and that the
# post-fork chain still processes a tx and a contract deploy/call.
FAUCET_ADDR=""; FAUCET_BAL="$BAL"
if [ -n "${FAUCET_PK:-}" ]; then
  python3 -c "import eth_account" 2>/dev/null || { echo "missing python3 web3/eth-account (needed to fund FAUCET_PK)"; exit 2; }
  FAUCET_ADDR="$(FAUCET_PK="$FAUCET_PK" python3 -c "import os; from eth_account import Account; print(Account.from_key(os.environ['FAUCET_PK']).address)")"
  echo "faucet address (funded in genesis): $FAUCET_ADDR"
fi
RETURNS42="0x600a600c600039600a6000f3602a60005260206000f3"

pkill -9 -f "$WORK" 2>/dev/null; sleep 1
rm -rf "$WORK"; mkdir -p "$WORK"

p2p(){ echo $((30010 + ($1-1)*10)); }   # arg = plan node (1-based)
http(){ echo $((40010 + ($1-1)*10)); }

# preset field for plan node k
preset_pubkey(){ local pk=${PRESET_FOR[$1]}; python3 -c "import json;print([n['publicKey'] for n in json.load(open('$PRESET/metadata.json'))['nodes'] if n['index']==$pk][0])"; }
preset_nodekey(){ local pk=${PRESET_FOR[$1]}
  if [ "$pk" = 5 ]; then cat "$PRESET/node5/nodekey";
  else python3 -c "import json;print([n['nodekey'] for n in json.load(open('$PRESET/metadata.json'))['nodes'] if n['index']==$pk][0])"; fi; }

# --- 1. wemix governance config (producer = plan node 1 = preset node5) -------
log "build wemix governance config"
PROD_ID="0x$(preset_pubkey 1)"
ACCOUNTS_JSON=$(FAUCET_ADDR="$FAUCET_ADDR" python3 - <<PY
import json, os
accts=[{"addr":"$PRODUCER_ACCT","balance":$BAL}]
fa=os.environ.get("FAUCET_ADDR","")
if fa:
    accts.append({"addr":fa,"balance":$BAL})
for n in json.load(open("$PRESET/metadata.json"))["nodes"]:
    accts.append({"addr":n["address"],"balance":$BAL})
print(json.dumps(accts))
PY
)
cat > "$WORK/wemix-config.json" <<JSON
{
  "extraData": "chainbench wemix->wbft handoff E2E",
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
      "id": "$PROD_ID", "ip": "127.0.0.1", "port": $(p2p 1), "bootnode": true }
  ],
  "accounts": $ACCOUNTS_JSON
}
JSON

# --- 2. wemix base genesis, then merge the croissant section via chainbench ----
log "wemix base genesis + framework croissant merge"
"$WEMIX_BIN" wemix genesis --data "$WORK/wemix-config.json" --genesis "$TEMPLATE" --out "$WORK/base.json" \
  || { echo "wemix genesis failed"; exit 1; }
jq ".config.chainId=$CHAINID | .config.petersburgBlock=0" "$WORK/base.json" > "$WORK/base2.json"
mv "$WORK/base2.json" "$WORK/base.json"
"$CHAINBENCH" upgrade genesis --profile "$PROFILE" --from-genesis "$WORK/base.json" --out "$WORK/genesis.json" \
  || { echo "chainbench upgrade genesis failed"; exit 1; }

# --- 3. static-nodes + init + provision each node -----------------------------
log "init + provision 5 nodes"
python3 - <<PY > "$WORK/static-nodes.json"
import json
enodes=[]
preset={1:5,2:1,3:2,4:3,5:4}
nodes={n['index']:n for n in json.load(open("$PRESET/metadata.json"))["nodes"]}
for k in range(1,6):
    pk=nodes[preset[k]]['publicKey']; port=30010+(k-1)*10
    enodes.append(f"enode://{pk}@127.0.0.1:{port}?discport=0")
print(json.dumps(enodes, indent=2))
PY

init_node(){ # plan-index binary instancedir
  local i="$1" bin="$2" inst="$3"; local d="$WORK/node$i"
  mkdir -p "$d"
  "$bin" --datadir "$d" init "$WORK/genesis.json" >/dev/null 2>&1 || { echo "init node$i failed"; exit 1; }
  mkdir -p "$d/$inst"
  preset_nodekey "$i" > "$d/$inst/nodekey"
  cp "$WORK/static-nodes.json" "$d/$inst/static-nodes.json"
}
init_node 1 "$WEMIX_BIN" geth                        # producer, go-wemix uses geth/
for i in 2 3 4 5; do init_node "$i" "$WBFT_BIN" gwemix; done   # go-wbft uses gwemix/
# producer's unlockable keystore FILE into the datadir keystore dir (copy the
# file, not the dir, or it nests under an init-created keystore/).
mkdir -p "$WORK/node1/keystore"
cp "$PRESET/node5/keystore/UTC--node5-keystore" "$WORK/node1/keystore/"
echo -n "1" > "$WORK/pw"

# --- 4. launch: producer (go-wemix) + validators (go-wbft), concurrent --------
log "launch nodes (concurrent mixed binaries)"
launch_wbft(){ local i="$1"; local d="$WORK/node$i"
  "$WBFT_BIN" --datadir "$d" --networkid "$CHAINID" --nat none \
    --port "$(p2p "$i")" --http --http.addr 127.0.0.1 --http.port "$(http "$i")" \
    --http.api eth,net,web3,istanbul,admin,miner,txpool \
    --authrpc.port $(( $(http "$i") + 2 )) --syncmode full --mine \
    --verbosity 3 > "$d/node.log" 2>&1 &
}
"$WEMIX_BIN" --datadir "$WORK/node1" --networkid "$CHAINID" --nat none \
  --port "$(p2p 1)" --http --http.addr 127.0.0.1 --http.port "$(http 1)" \
  --http.api eth,net,web3,wemix,admin,miner,txpool,personal \
  --authrpc.port $(( $(http 1) + 2 )) --syncmode full \
  --mine --miner.etherbase "$PRODUCER_ACCT" \
  --unlock "$PRODUCER_ACCT" --password "$WORK/pw" --allow-insecure-unlock \
  --verbosity 3 > "$WORK/node1/node.log" 2>&1 &
for i in 2 3 4 5; do launch_wbft "$i"; done
sleep 8

rpc(){ curl -s "http://127.0.0.1:$1" -H 'content-type:application/json' \
  --data "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$2\",\"params\":${3:-[]}}"; }
hexdec(){ python3 -c "import sys;print(int(sys.argv[1],16))" "$1" 2>/dev/null || echo 0; }

# --- 4b. full-mesh peering via admin_addPeer -----------------------------------
# static-nodes placement differs per binary/instance; force a full mesh so the
# go-wbft validators can exchange WBFT consensus messages with each other (not
# just with the producer), which they need for a 2f+1 quorum post-fork.
log "wire full mesh (admin_addPeer)"
mapfile -t ENODES < <(jq -r '.[]' "$WORK/static-nodes.json")
for k in 1 2 3 4 5; do
  for e in "${ENODES[@]}"; do
    rpc "$(http "$k")" admin_addPeer "[\"$e\"]" >/dev/null
  done
done
sleep 5
for k in 1 2 3 4 5; do printf 'node%d peers=%s\n' "$k" "$(rpc "$(http "$k")" net_peerCount | jq -r '.result // "?"')"; done

# --- 5. deploy governance + etcdInit on the producer --------------------------
log "deploy governance + etcdInit on producer (node1)"
P1=$(http 1)
for w in $(seq 1 20); do [ -S "$WORK/node1/gwemix.ipc" ] && break; sleep 1; done
[ -S "$WORK/node1/gwemix.ipc" ] || { echo "producer IPC never appeared; tail:"; tail -15 "$WORK/node1/node.log"; exit 1; }
"$WEMIX_BIN" wemix deploy-governance --url "$WORK/node1/gwemix.ipc" \
  --password "$WORK/pw" "$WORK/wemix-config.json" \
  "$WORK/node1/keystore/UTC--node5-keystore" > "$WORK/deploy-gov.log" 2>&1 \
  && echo "governance deployed" || echo "deploy-governance nonzero (see log)"
"$WEMIX_BIN" attach "$WORK/node1/gwemix.ipc" --exec "admin.etcdInit()" > "$WORK/etcd-init.log" 2>&1 \
  && echo "etcdInit done" || echo "etcdInit nonzero (see log)"

# --- 6. poll for the handoff --------------------------------------------------
# Poll a go-wbft VALIDATOR (plan node 2), not the producer: the go-wemix producer
# has no wbft engine, so it cannot import post-croissant blocks and stays at the
# fork height. The successor chain advances only on the wbft nodes.
log "poll for handoff (target head > $CROISSANT, on a wbft validator)"
PV=$(http 2)
PRODUCER_MINER=$(python3 -c "print('$PRODUCER_ACCT'.lower())")
post_fork_wbft_miner=""; head=0
for t in $(seq 1 80); do
  bn=$(rpc "$PV" eth_blockNumber | jq -r '.result // "0x0"'); head=$(hexdec "$bn")
  if [ "$head" -gt "$CROISSANT" ]; then
    blk=$(printf '0x%x' $((CROISSANT+1)))
    miner=$(rpc "$PV" eth_getBlockByNumber "[\"$blk\",false]" | jq -r '.result.miner // ""' | tr 'A-Z' 'a-z')
    printf 'head=%s  block %d miner=%s\n' "$head" $((CROISSANT+1)) "$miner"
    if [ -n "$miner" ] && [ "$miner" != "$PRODUCER_MINER" ] && [ "$miner" != "0x0000000000000000000000000000000000000000" ]; then
      post_fork_wbft_miner="$miner"; break; fi
  else
    printf 't=%2ds head=%d peers(node1)=%s\n' $((t*3)) "$head" "$(rpc "$P1" net_peerCount | jq -r '.result // "?"')"
  fi
  sleep 3
done

log "result"
grep -h "skips mining due to Croissant" "$WORK"/node1/node.log 2>/dev/null | head -1 || echo "(no croissant-skip log on producer yet)"
echo "final head=$head  post-fork miner=$post_fork_wbft_miner  producer=$PRODUCER_MINER"
if [ "$head" -gt "$CROISSANT" ] && [ -n "$post_fork_wbft_miner" ]; then
  echo "handoff reproduced (chain passed croissant; go-wbft validator produced post-fork)"

  # scenario 2: pre-fork state must survive on the wbft successor, and the
  # post-fork chain must still process a tx and a contract. Gated on FAUCET_PK.
  if [ -n "${FAUCET_PK:-}" ]; then
    WURL="http://127.0.0.1:$PV" # a go-wbft validator (plan node 2)
    bn_hex(){ "$CHAINBENCH" node rpc --rpc "$1" --method eth_getBalance --params "[\"$2\",\"latest\"]" 2>/dev/null | tr -d '"'; }
    log "state preservation: faucet genesis balance on the wbft successor"
    bal=$(python3 -c "s='$(bn_hex "$WURL" "$FAUCET_ADDR")'.strip(); print(int(s,16) if s.startswith('0x') else -1)")
    if [ "$bal" != "$FAUCET_BAL" ]; then
      echo "FAIL: pre-fork genesis state not preserved (faucet balance $bal != $FAUCET_BAL)"; exit 1
    fi
    echo "state preserved OK (faucet balance intact across handoff)"

    log "post-fork tx on the wbft successor"
    HASH="$("$CHAINBENCH" tx send --chain wbft --rpc "$WURL" --from-key "$FAUCET_PK" --to 0x000000000000000000000000000000000000dEaD --value 1000000000000000000)" \
      || { echo "FAIL: post-fork tx send errored"; exit 1; }
    ST="$("$CHAINBENCH" tx wait --rpc "$WURL" --hash "$HASH" 2>/dev/null | awk '/^status:/{print $2}')"
    [ "$ST" = "success" ] || { echo "FAIL: post-fork tx status=$ST (want success)"; exit 1; }

    log "post-fork contract deploy/call on the wbft successor"
    OUT="$("$CHAINBENCH" contract deploy --chain wbft --rpc "$WURL" --from-key "$FAUCET_PK" --bytecode "$RETURNS42")" \
      || { echo "FAIL: post-fork contract deploy errored"; exit 1; }
    CADDR="$(echo "$OUT" | awk '/^contract:/{print $2}')"
    cok=false
    for _ in $(seq 1 30); do
      r="$("$CHAINBENCH" contract call --rpc "$WURL" --to "$CADDR" --data 0x 2>/dev/null | tr -d '"')"
      case "$r" in *2a) cok=true; break;; esac; sleep 3
    done
    [ "$cok" = true ] || { echo "FAIL: post-fork contract did not return 42 (last=$r)"; exit 1; }
    echo "PASS: handoff + pre-fork state preserved + post-fork tx/contract OK"; exit 0
  fi

  echo "PASS: handoff reproduced (set FAUCET_PK to also verify state preservation + post-fork tx/contract)"; exit 0
fi
echo "FAIL: handoff not observed"
echo "--- producer(node1) tail ---"; tail -25 "$WORK/node1/node.log"
echo "--- node2 (wbft) tail ---"; tail -25 "$WORK/node2/node.log"
exit 1
