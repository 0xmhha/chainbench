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
# five of them and they belong to the environment, not to this script:
#   TCSWEEP_FLAGS="--server-set env/docker/build/server-set.yaml \
#     --workspace-config env/docker/build/workspace-config.yaml \
#     --docker --all-servers --keys-source generate"
# --all-servers is not optional with a 15-server set: without it, and without a
# --server naming one, the place step refuses because it cannot tell which
# server to use. Leaving it out of this list once sent a reader straight into
# that refusal. Unset, every run goes local and wants the chain binary on PATH.
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

# attachOf prints the attach endpoint a case's chain-preset declares, or nothing
# when the case composes its own network. The declaration is printed rather than
# a yes/no because the caller has to know WHICH variable it names: the sweep can
# supply GSTABLE_RPC, pointing at a network it brings up, and can supply nothing
# else.
#
# Measured 2026-10-02. The five go-stablenet/testnet cases read
# ${GSTABLE_TESTNET_RPC}, which names a network nobody here composes. A yes/no
# answer sent all five through runAttach, so the sweep booted a 15-node network
# for each of them, handed it a variable the preset does not read, watched the
# case refuse on an empty endpoint, and tore the network down again: 194 seconds
# spent to ask nothing, five times.
attachOf() {
  python3 - "$1" "$ROOT" <<'PY'
import json, pathlib, sys
spec, root = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
try:
    d = json.loads(spec.read_text())
except Exception:
    raise SystemExit
cp, seen = d.get("chainPreset"), set()
while True:
    if isinstance(cp, str):
        f = root / "presets/chain" / (cp + ".json")
        if not f.exists() or cp in seen:
            break
        seen.add(cp)
        cp = json.loads(f.read_text())
        continue
    if isinstance(cp, dict):
        rpc = (cp.get("attach") or {}).get("rpc")
        if rpc:
            print(rpc[0] if isinstance(rpc, list) else rpc)
            break
        cp = cp.get("extends")
        continue
    break
PY
}

# attachTarget prints the RPC URL of node1 of the network composed in $1, as
# this machine reaches it, and the key set it was composed from. Under
# --docker the node's own address is translated through the localmap next to
# the server set, the same way chainbench dials it.
attachTarget() {
  # The record is read into a variable and handed over as an argument, not
  # piped. `python3 -` takes the script on stdin and the heredoc below is that
  # stdin, so a pipe into this cannot be read: json.load(sys.stdin) saw the
  # already-consumed heredoc and raised, every time. The failure was a
  # traceback rather than a message because the error was discarded and the
  # empty result then fell through to the preset's default endpoint, so an
  # attach case failed against 127.0.0.1:8600 with nothing saying why.
  local show
  show=$("$BIN" chain show --workspace-dir "$1" --node 1 --json 2>&1) || {
    printf 'attachTarget: chain show failed for %s: %s\n' "$1" "$show" >&2
    return 1
  }
  [ -n "$show" ] || { printf 'attachTarget: chain show said nothing for %s\n' "$1" >&2; return 1; }
  python3 - "$1" "$ROOT" "$show" ${EXTRA[@]+"${EXTRA[@]}"} <<'PY'
import json, pathlib, re, sys
ws, root, extra = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]), sys.argv[4:]
e = json.loads(sys.argv[3])["entries"][0]
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
# consensus case's network, kept up in $2, then stopped. The case is given
# node1's address and the key set the network was composed from, which is what
# the case's own declaration would name on a machine that set one up.
#
# The host case is a common one, so this reaches it under tests/tc/common.
runAttach() {
  local spec=$1 ws=$2 host="$ROOT/tests/tc/common/node/CT-NODE-009-head-hash-agreement.json" up rpc keys
  if ! up=$("$BIN" run "$host" --workspace-dir "$ws" --keep-up ${EXTRA[@]+"${EXTRA[@]}"} 2>&1); then
    printf '%s\nthe network to attach to did not come up\n' "$up"
    "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
    return 2
  fi
  # Checked, not assumed. attachTarget writes its refusal to stderr and returns
  # non-zero; an unchecked read then leaves both variables empty, and the
  # preset's own default endpoint takes over, so the case is answered by
  # whatever happens to be on this machine's 8600. That is the silent failure
  # the comment above attachTarget describes. Its cause was fixed and this half
  # was not, so any other cause brings it straight back.
  local target
  if ! target=$(attachTarget "$ws" 2>&1); then
    printf '%s\nthe address of the network to attach to could not be read\n' "$target"
    "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
    return 2
  fi
  read -r rpc keys <<<"$target"
  if [ -z "$rpc" ]; then
    printf 'attachTarget named no endpoint: %s\n' "$target"
    "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
    return 2
  fi
  GSTABLE_RPC="$rpc" "$BIN" run "$spec" --keys "$keys" 2>&1
  local code=$?
  "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
  return $code
}

# Selection, without mapfile: that is bash 4 and macOS ships bash 3.2, where it
# is not a builtin and not an error either -- `mapfile -t X < <(echo a)` prints
# "command not found" and leaves X empty, so the sweep reported "0 cases" and
# (before the exit contract above) ended 0. A machine with no newer bash on PATH
# ran nothing and said so in a way that reads like success.
#
# The pattern is applied with grep and its result is checked, rather than
# `grep || cat`: grep has already consumed the stream by the time cat runs, so
# the fallback could only ever produce nothing, and a typo in the pattern was
# indistinguishable from a clean sweep.
CASES=()
while IFS= read -r case_path; do
  CASES+=("$case_path")
done < <(
  if [ -n "$PATTERN" ]; then
    find "$ROOT/tests/tc" -name '*.json' | sort | grep -- "$PATTERN"
  else
    find "$ROOT/tests/tc" -name '*.json' | sort
  fi
)
total=${#CASES[@]}
: > "$OUT"
if [ "$total" -eq 0 ]; then
  if [ -n "$PATTERN" ]; then
    echo "no case matched $PATTERN" | tee -a "$OUT" >&2
  else
    echo "no case found under $ROOT/tests/tc" | tee -a "$OUT" >&2
  fi
  exit 2
fi
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
  endpoint="$(attachOf "$spec")"
  # Whether this case got a workspace at all. The one below does not, and
  # calling `chain stop --workspace-dir` on a path nothing composed CREATES it,
  # holding a chain-record.json and a process.json for a network that never
  # existed -- which the sweep then offered "for inspection".
  composed=1
  if [ -z "$endpoint" ]; then
    out=$("$BIN" run "$spec" --workspace-dir "$ws" ${EXTRA[@]+"${EXTRA[@]}"} 2>&1)
  elif [ "${endpoint#*GSTABLE_RPC}" != "$endpoint" ]; then
    # The one endpoint this sweep can supply: a network it composes itself.
    out=$(runAttach "$spec" "$ws")
  else
    # An endpoint naming a network nobody here composes. Composing one for it
    # would cost a full bring-up and answer nothing, so the case is run as
    # written: if the operator exported the variable it names, it reaches their
    # network, and if they did not, the preset refuses in a second and says
    # which variable is empty.
    composed=0
    out=$("$BIN" run "$spec" 2>&1)
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
  # Always, as the paragraph above says. It used to run only with
  # TCSWEEP_FLAGS set, so a local sweep never stopped anything -- and the one
  # moment the guard exists for is a run that died with its nodes still up,
  # which is exactly when nothing else will stop them.
  if [ "$composed" -eq 1 ]; then
    "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
  fi
  if [ "$verdict" = PASS ] && [ "$composed" -eq 1 ]; then
    # If rm cannot finish, the workspace stays. It holds the composition id and
    # the per-node paths, and deleting it while a datadir is still on a server
    # leaves that datadir with nothing left that can name it.
    if "$BIN" chain rm --workspace-dir "$ws" >/dev/null 2>&1; then
      rm -rf "$ws"
    else
      printf '          chain rm did not finish; kept so it can be removed later: %s\n' "$ws" | tee -a "$OUT"
    fi
  elif [ -d "$ws" ]; then
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

# The exit code the header promises. It was never written, so every sweep ended
# 0: the 197-case run that reported fail=1 blocked=15 exited 0, and so did a
# pattern that matched nothing. Anything that reads the code rather than the log
# -- CI, a wrapper, `&&` -- was told the sweep passed.
#
# A failure outranks a block: a case that answered "no" is a result, and one
# that could not be asked is not, so a sweep with both reports the result.
if [ "$fail" -gt 0 ]; then
  exit 1
fi
if [ $((blocked + notrun)) -gt 0 ]; then
  exit 2
fi
exit 0
