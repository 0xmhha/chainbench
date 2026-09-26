#!/usr/bin/env bash
# Run every DSL case, one network each, and say what happened.
#
# One network per case on purpose. Cases that share a chain-preset still change
# chain state, so running them together lets one case decide another's verdict —
# measured, and the reason this walks rather than batches. It is slower and it
# is the only shape whose result means anything.
#
# Four verdicts, not two. `chainbench run` already distinguishes them and a
# sweep that folds them lies in both directions: a SKIP answered nothing about
# the case and is not a pass, and a BLOCKED run could not ask the question at
# all, which is not the same as the case answering "no". The exit code says
# which — 0 ran, 1 a case failed, 2 the run could not proceed — and the summary
# line carries the counts.
#
# Usage: scripts/tcsweep.sh <out.log> [pattern]
set -uo pipefail

OUT="${1:?usage: tcsweep.sh <out.log> [pattern]}"
PATTERN="${2:-}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/chainbench"
WS_BASE="${TCSWEEP_WS:-$HOME/cbw/sweep}"
# Extra flags for every `chainbench run`. A sweep against the docker fleet needs
# four of them and they belong to the environment, not to this script:
#   TCSWEEP_FLAGS="--server-set env/docker/build/server-set.yaml \
#     --workspace-config env/docker/build/workspace-config.yaml --docker \
#     --keys-source generate"
read -r -a EXTRA <<<"${TCSWEEP_FLAGS:-}"

[ -x "$BIN" ] || { echo "no $BIN — run make build" >&2; exit 2; }

# chainOf reads the chain a case declares. The directory is not it: one case
# under go-wbft/ declares stablenet, and guessing from the path is what labelled
# it wrongly in an earlier sweep.
chainOf() {
  python3 - "$1" "$ROOT" <<'PY'
import json, pathlib, sys
spec, root = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
try:
    d = json.loads(spec.read_text())
except Exception:
    print("?"); raise SystemExit
cp = d.get("chainPreset")
seen = set()
while True:
    if isinstance(cp, dict):
        if cp.get("chain"):
            print(cp["chain"]); break
        ext = cp.get("extends")
        if not ext or ext in seen:
            print("?"); break
        seen.add(ext)
        cp = ext
        continue
    if isinstance(cp, str):
        f = root / "presets/chain" / (cp + ".json")
        if not f.exists():
            print("?"); break
        cp = json.loads(f.read_text())
        continue
    print("?"); break
PY
}

# attachOf says whether a case declares env.attach: it runs against a network
# that is already up and composes none. `run --workspace-dir` refuses such a
# case, so the sweep brings a network up for it first (runAttach).
attachOf() {
  python3 - "$1" "$ROOT" <<'PY'
import json, pathlib, sys
spec, root = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
try:
    d = json.loads(spec.read_text())
except Exception:
    print(0); raise SystemExit
cp, seen = d.get("chainPreset"), set()
while True:
    if isinstance(cp, str):
        f = root / "presets/chain" / (cp + ".json")
        if not f.exists() or cp in seen:
            print(0); break
        seen.add(cp)
        cp = json.loads(f.read_text())
        continue
    if isinstance(cp, dict):
        if cp.get("attach"):
            print(1); break
        cp = cp.get("extends")
        continue
    print(0); break
PY
}

# attachTarget prints the RPC URL of node1 of the network composed in $1, as
# this machine reaches it, and the key set it was composed from. Under
# --docker the node's own address is translated through the localmap next to
# the server set, the same way chainbench dials it.
attachTarget() {
  "$BIN" chain show --workspace-dir "$1" --node 1 --json 2>/dev/null | python3 - "$1" "$ROOT" ${EXTRA[@]+"${EXTRA[@]}"} <<'PY'
import json, pathlib, re, sys
ws, root, extra = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]), sys.argv[3:]
e = json.load(sys.stdin)["entries"][0]
host, port = e["host"], e["http"]
if "--docker" in extra and "--server-set" in extra:
    lm = pathlib.Path(extra[extra.index("--server-set") + 1]).parent / "localmap.yaml"
    m = re.search(re.escape(host) + r":\s*\n\s*host:\s*(\S+)\s*\n\s*ports:\s*\{([^}]*)\}", lm.read_text())
    if m:
        ports = dict(p.split(":") for p in m.group(2).replace(" ", "").split(","))
        host, port = m.group(1), ports.get(str(port), port)
keys = json.loads((ws / "chain-record.json").read_text()).get("keysDir", "")
if keys and not pathlib.Path(keys).is_absolute():
    keys = str(root / keys)
print(f"http://{host}:{port} {keys}")
PY
}

# runAttach runs an attach case against a network it brings up for it: the
# basic consensus case's network, kept up in $2, then stopped. The case is
# given node1's address and the key set the network was composed from, which
# is what the case's own declaration would name on a machine that set one up.
runAttach() {
  local spec=$1 ws=$2 host="$ROOT/tests/tc/basic/01-basic-consensus.json" up rpc keys
  if ! up=$("$BIN" run "$host" --workspace-dir "$ws" --keep-up ${EXTRA[@]+"${EXTRA[@]}"} 2>&1); then
    printf '%s\nthe network to attach to did not come up\n' "$up"
    "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
    return 2
  fi
  read -r rpc keys < <(attachTarget "$ws")
  CHAINBENCH_RPC="$rpc" "$BIN" run "$spec" --keys "$keys" 2>&1
  local code=$?
  "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
  return $code
}

mapfile -t CASES < <(find "$ROOT/tests/tc" -name '*.json' | sort | { [ -n "$PATTERN" ] && grep "$PATTERN" || cat; })
total=${#CASES[@]}
: > "$OUT"
echo "$total cases, one network each" | tee -a "$OUT"
echo "" | tee -a "$OUT"

pass=0 fail=0 blocked=0 skip=0 notrun=0 i=0
for spec in "${CASES[@]}"; do
  i=$((i + 1))
  rel="${spec#"$ROOT"/}"
  chain="$(chainOf "$spec")"

  ws="$WS_BASE/c$i"
  rm -rf "$ws"
  start=$SECONDS
  if [ "$(attachOf "$spec")" = 1 ]; then
    out=$(runAttach "$spec" "$ws")
  else
    out=$("$BIN" run "$spec" --workspace-dir "$ws" ${EXTRA[@]+"${EXTRA[@]}"} 2>&1)
  fi
  code=$?
  took=$((SECONDS - start))

  # The summary line is the run's own count. Its absence means the run never
  # got as far as one, which is a setup failure and not a verdict about a case.
  line=$(grep -oE 'pass=[0-9]+ fail=[0-9]+ blocked=[0-9]+ skip=[0-9]+' <<<"$out" | tail -1)
  if [ -z "$line" ]; then
    if grep -q 'executable file not found' <<<"$out"; then
      verdict=NOTRUN; notrun=$((notrun + 1))
    else
      verdict=BLOCKED; blocked=$((blocked + 1))
    fi
  else
    read -r p f b s <<<"$(sed -E 's/[a-z]+=//g' <<<"$line")"
    case 1 in
      $((f > 0)))            verdict=FAIL;    fail=$((fail + 1)) ;;
      $((b > 0)))            verdict=BLOCKED; blocked=$((blocked + 1)) ;;
      $((s > 0 && p == 0)))  verdict=SKIP;    skip=$((skip + 1)) ;;
      $((code == 0)))        verdict=PASS;    pass=$((pass + 1)) ;;
      *)                     verdict=FAIL;    fail=$((fail + 1)) ;;
    esac
  fi

  printf '[%3d/%3d] %-7s %4ds %-10s %s\n' "$i" "$total" "$verdict" "$took" "$chain" "$rel" | tee -a "$OUT"
  if [ "$verdict" != PASS ]; then
    # The last lines hold the refusal, including the state it failed in.
    tail -3 <<<"$out" | sed 's/^/          /' | tee -a "$OUT"
  fi
  # Stop always: a network left running holds the next case's ports. Remove
  # only what passed. On a remote target `chain rm` deletes the datadir and
  # the node logs on the server, and `rm -rf $ws` takes the record that could
  # ask for them later — so for a case that did not pass, both stay and the
  # sweep says where. The session under ~/.chainbench already holds the logs
  # gathered at the moment of failure; what is kept here is the machine state
  # behind them, which is what a second look needs.
  if [ -n "${TCSWEEP_FLAGS:-}" ]; then
    "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
  fi
  if [ "$verdict" = PASS ]; then
    [ -n "${TCSWEEP_FLAGS:-}" ] && "$BIN" chain rm --workspace-dir "$ws" >/dev/null 2>&1
    rm -rf "$ws"
  else
    printf '          kept for inspection: %s\n' "$ws" | tee -a "$OUT"
  fi
done

{
  echo ""
  echo "=== TOTAL ==="
  printf '{"pass": %d, "fail": %d, "blocked": %d, "skip": %d, "notrun": %d}\n' \
    "$pass" "$fail" "$blocked" "$skip" "$notrun"
  echo "DONE"
} | tee -a "$OUT"
