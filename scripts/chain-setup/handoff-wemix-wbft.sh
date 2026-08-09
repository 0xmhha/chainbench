#!/usr/bin/env bash
# handoff-wemix-wbft.sh — case 2: go-wemix produces up to the fork, go-wbft
# validators take over after it.
#
# This is the sequence that works, run end to end and verified live: block 100
# is sealed by a go-wbft validator. It is a script rather than prose because the
# ordering is the whole point — see docs/dev/chain-setup/case-2-wemix-to-wbft.md.
#
# `chainbench chain up --case wemix-wbft` still runs the older single-phase
# order and fails; until that is reworked, this script is the way to stand the
# handoff up.
#
# Usage:
#   scripts/chain-setup/handoff-wemix-wbft.sh [data-dir]
#
# Environment:
#   CHAIN_DIR   parent of go-wemix / go-wbft checkouts
#   PROFILE     upgrade profile (fork_block 100 recommended; 20 is too early)
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHAIN_DIR="${CHAIN_DIR:-$HOME/work/github/0xmhha/chain}"
PROFILE="${PROFILE:-$REPO/profiles/wemix-upgrade.yaml}"
# Keep the data dir short: a node's IPC socket path must stay under 104 chars.
D="${1:-/tmp/handoff}"

G="$CHAIN_DIR/go-wemix/build/bin/gwemix"   # producer binary
W="$CHAIN_DIR/go-wbft/build/bin/gwemix"    # successor binary (also named gwemix)
TEMPLATE="$CHAIN_DIR/go-wemix/wemix/scripts/genesis-template.json"

for f in "$G" "$W" "$TEMPLATE" "$PROFILE"; do
  [[ -e "$f" ]] || { echo "missing: $f" >&2; exit 1; }
done

# Validator identity: plan node k+1 takes preset node k (profile plan_order
# [5,1,2,3,4] puts the producer first while the preset lists it last).
addr_for() {
  case "$1" in
    2) echo 0xc17d493883eaa3b4cceb0f214b273392d562f9d8 ;;
    3) echo 0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c ;;
    4) echo 0x8c4a10b9108d49b9d23f764464090831d9c17764 ;;
    5) echo 0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6 ;;
  esac
}

rpc() { # rpc <port> <method> [params-json]
  curl -s -X POST -H 'Content-Type: application/json' \
    --data "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$2\",\"params\":${3:-[]}}" \
    "http://127.0.0.1:$1"
}

step() { printf '\n=== %s ===\n' "$*"; }

rm -rf "$D"

# --- Prepare: genesis + per-node datadir init -------------------------------
# chain up does this part correctly. Capture each node's launch command while
# they are running, then stop them: the datadirs stay.
step "prepare (genesis + datadir init)"
cd "$REPO"
go run ./cmd/chainbench chain up --case wemix-wbft --profile "$PROFILE" \
  --from-binary "$G" --to-binary "$W" --template "$TEMPLATE" \
  --data-dir "$D" --stop-after launch

for pid in $(pgrep -f "datadir $D/node"); do
  args=$(ps -o args= -p "$pid")
  n=$(sed -E "s|.*--datadir $D/node([0-9]+).*|\1|" <<<"$args")
  echo "$args" > "$D/node$n.cmd"
done
go run ./cmd/chainbench chain down --data-dir "$D"
sleep 3

# --- Phase A: bootstrap with the producer ALONE -----------------------------
# With any other node up, etcdInit leaves the cluster empty and the chain stalls.
step "phase A: producer alone"
# shellcheck disable=SC2046  # deliberate word splitting of the captured command
nohup $(cat "$D/node1.cmd") > "$D/n1.log" 2>&1 &
until [[ -S "$D/node1/gwemix.ipc" ]]; do sleep 1; done

KS=$(ls "$D"/node1/keystore/* | head -1)
"$G" wemix deploy-governance --url "$D/node1/gwemix.ipc" \
  --password "$D/password" "$D/wemix-config.json" "$KS"

"$G" attach "$D/node1/gwemix.ipc" --exec 'admin.etcdInit()'

# admin.etcdInit() returns null even on success, so the cluster state is the
# only evidence it worked.
step "phase A: verify the etcd cluster formed"
ETCD=$("$G" attach "$D/node1/gwemix.ipc" --exec 'JSON.stringify(admin.wemixInfo.etcd)')
echo "$ETCD"
grep -q '\\"cluster\\":\\"[^\\"]' <<<"$ETCD" || {
  echo "etcd cluster is empty: the bootstrap did not take. Do not continue." >&2
  exit 1
}

pkill -f "datadir $D/node1 " || true
sleep 3

# --- Phase B: run the whole network -----------------------------------------
step "phase B: launch every node"
for i in 2 3 4 5; do
  A=$(addr_for "$i")
  mkdir -p "$D/node$i/keystore"
  cp "$REPO/keys/preset/node$((i-1))/keystore/"* "$D/node$i/keystore/"
  # Successors must be able to sign, or they cannot seal past the fork.
  echo "$(cat "$D/node$i.cmd") --unlock $A --password $D/password --miner.etherbase $A" \
    > "$D/node$i.full"
done

# shellcheck disable=SC2046
nohup $(cat "$D/node1.cmd") > "$D/n1.log" 2>&1 &
for i in 2 3 4 5; do
  # shellcheck disable=SC2046
  nohup $(cat "$D/node$i.full") > "$D/n$i.log" 2>&1 &
done
sleep 30

# The mesh must be re-wired: these are fresh processes, and validators that
# cannot reach each other never reach quorum.
step "phase B: wire the mesh"
: > "$D/enodes.txt"
for p in 40010 40020 40030 40040 40050; do
  rpc "$p" admin_nodeInfo | python3 -c \
    "import json,sys;print(json.load(sys.stdin)['result']['enode'])" >> "$D/enodes.txt"
done
for p in 40010 40020 40030 40040 40050; do
  while read -r e; do rpc "$p" admin_addPeer "[\"$e\"]" > /dev/null; done < "$D/enodes.txt"
done

step "state"
for p in 40010 40020 40030 40040 40050; do
  printf '  :%s head=%s peers=%s\n' "$p" \
    "$(rpc "$p" eth_blockNumber | sed 's/.*result":"\([^"]*\)".*/\1/')" \
    "$(rpc "$p" net_peerCount   | sed 's/.*result":"\([^"]*\)".*/\1/')"
done

cat <<EOF

Watch the handoff on a VALIDATOR, not the producer: the producer stops at the
block before the fork, which is correct.

  curl -s -X POST -H 'Content-Type: application/json' \\
    --data '{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["0x64",false]}' \\
    http://127.0.0.1:40020

Block <fork> and after must carry a validator address as "miner".
Stop everything with: pkill -f "datadir $D/node"
EOF
