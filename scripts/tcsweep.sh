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

# A round may need a different environment from the rest. go-wemix is the case
# that forces it: poa reserves three consecutive p2p-side ports per node, so it
# needs the server set written for it, and the default one is refused at the
# place step with "p2p_step must be >= 3". Set TCSWEEP_FLAGS_WEMIX (or _WBFT,
# or _STABLENET) and that round uses it in place of TCSWEEP_FLAGS.
roundFlags() {
  local up name val
  up=$(printf '%s' "$1" | tr '[:lower:]' '[:upper:]')
  name="TCSWEEP_FLAGS_$up"
  val="${!name:-}"
  if [ -n "$val" ]; then printf '%s' "$val"; else printf '%s' "${TCSWEEP_FLAGS:-}"; fi
}

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

# planGroups turns the selected cases into the invocations that will run them.
#
# One invocation per group, because `chainbench run` takes several definitions
# and composes ONE network per distinct composition among them: measured
# 2026-10-07, three rpc cases produced two networks, the two that share a
# composition sharing one. Walking case by case paid for a bring-up every time.
#
# The trade is the one the header paragraph names: cases on a shared network
# see each other's state, so one can decide another's verdict. That is why this
# is a group of a FOLDER rather than of everything, and why the log says which
# cases shared.
#
# tests/tc/common runs three times, one chain at a time, in the order
# stablenet, wemix, wbft. A round overrides the chain-preset with the same
# shape on its chain -- the per-chain presets differ in the chain field and
# nothing else -- and a shape with no sibling on that chain is reported rather
# than silently dropped.
# The cases arrive as arguments, not on a pipe: `python3 -` takes its program
# on stdin, so a pipe into it is the program's own text and sys.stdin is
# already spent. attachTarget carries the same note; this is the same trap.
planGroups() {
  python3 - "$ROOT" "$@" <<'PLAN'
import json, pathlib, sys

root = pathlib.Path(sys.argv[1])
cases = sys.argv[2:]

# ROUNDS is the order the common suite is walked in. One chain's whole suite
# runs before the next one starts, so a round pays for its networks once
# instead of alternating between chains case by case.
ROUNDS = ["stablenet", "wemix", "wbft"]
COMMON = "tests/tc/common/"


def base_preset(spec):
    """The named chain-preset a case ends up on, following extends."""
    try:
        d = json.loads(pathlib.Path(spec).read_text())
    except Exception:
        return None
    cp, seen = d.get("chainPreset"), set()
    while True:
        if isinstance(cp, str):
            if cp in seen:
                return None
            seen.add(cp)
            f = root / "presets/chain" / (cp + ".json")
            if not f.exists():
                return None
            nxt = json.loads(f.read_text())
            if nxt.get("chain"):
                return cp
            cp = nxt.get("extends")
            continue
        if isinstance(cp, dict):
            cp = cp.get("extends")
            continue
        return None


def sibling(preset, chain):
    """The same shape on another chain. The per-chain presets differ in the
    chain field and nothing else, and the binary follows the chain, so the
    shape's name is the chain's name with the prefix swapped."""
    for p in ("stablenet-", "wbft-", "wemix-"):
        if preset.startswith(p):
            cand = chain + "-" + preset[len(p):]
            return cand if (root / "presets/chain" / (cand + ".json")).exists() else None
    return None


groups, unmapped = [], []


def emit(round_name, override, specs):
    if specs:
        groups.append((round_name, override, specs))


def flush(key, specs):
    if key != "common":
        emit("-", "-", specs)
        return
    for chain in ROUNDS:
        if chain == "stablenet":
            # Already what every common case declares, so nothing to override:
            # the engine composes one network per distinct composition.
            emit(chain, "-", specs)
            continue
        by_preset = {}
        for s in specs:
            bp = base_preset(s)
            tgt = sibling(bp, chain) if bp else None
            if not tgt:
                unmapped.append((chain, s, bp or "?"))
                continue
            by_preset.setdefault(tgt, []).append(s)
        for tgt in sorted(by_preset):
            emit(chain, tgt, by_preset[tgt])


cur_key, cur = None, []
for spec in cases:
    rel = str(pathlib.Path(spec).relative_to(root)) if str(spec).startswith(str(root)) else spec
    is_common = COMMON in rel + "/"
    key = ("common" if is_common else str(pathlib.Path(rel).parent))
    if key != cur_key:
        if cur_key is not None:
            flush(cur_key, cur)
        cur_key, cur = key, []
    cur.append(spec)


if cur_key is not None:
    flush(cur_key, cur)

for chain, spec, bp in unmapped:
    print("UNMAPPED\t%s\t%s\t%s" % (chain, bp, spec))
for r, o, specs in groups:
    print("GROUP\t%s\t%s\t%s" % (r, o, " ".join(specs)))
PLAN
}

# endWorkspace stops the network a workspace holds and removes the workspace
# when the case that ran there passed.
#
# A workspace that saw a failure is kept, with its network stopped: it holds
# the composition id and the per-node paths, which is what a second look needs,
# and `chain rm` on a remote target would take the datadir and the node logs
# with it. If rm cannot finish on a passing one, that workspace is kept too --
# removing it while a datadir is still on a server leaves that datadir with
# nothing left that can name it.
endWorkspace() {
  local ws=$1 passed=$2
  [ -d "$ws" ] || return 0
  "$BIN" chain stop --workspace-dir "$ws" >/dev/null 2>&1
  if [ "$passed" = 1 ]; then
    if "$BIN" chain rm --workspace-dir "$ws" >/dev/null 2>&1; then
      rm -rf "$ws"
      return 0
    fi
    printf '          chain rm did not finish; kept so it can be removed later: %s\n' "$ws" | tee -a "$OUT"
    return 0
  fi
  printf '          kept for inspection: %s\n' "$ws" | tee -a "$OUT"
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
# The plan: one line per group, as "GROUP<tab>round<tab>override<tab>specs",
# plus an UNMAPPED line for a case a round has no preset for. A group shares
# a round and a chain-preset override; its cases still run one at a time.
PLAN=()
while IFS= read -r plan_line; do
  PLAN+=("$plan_line")
done < <(planGroups "${CASES[@]}")

pass=0 fail=0 blocked=0 skip=0 notrun=0 i=0 c=0
# The denominator is case-RUNS, not cases: tests/tc/common is walked once per
# chain, so two cases there are six runs, and counting cases printed [3/2].
# Runs and invocations are the same number, because one case is one
# `chainbench run`, so only the runs are printed. A header reporting both
# would send the reader looking for a batching this walk does not do.
selected=$total
total=0
for plan_line in "${PLAN[@]}"; do
  case "$plan_line" in
    GROUP*)    total=$((total + $(cut -f4 <<<"$plan_line" | wc -w))) ;;
    UNMAPPED*) total=$((total + 1)) ;;
  esac
done
echo "$selected case(s) -> $total run(s) of a case; tests/tc/common is walked once per chain" | tee -a "$OUT"
echo "" | tee -a "$OUT"

# record takes one case's verdict: $1 spec, $2 the run's output, $3 seconds,
# $4 the label to print next to it, $5 the exit code `chainbench run` gave.
record() {
  local spec=$1 out=$2 took=$3 label=$4 code=$5 rel line p f b s2 verdict
  rel="${spec#"$ROOT"/}"
  # Prefer a summary line that names this case; a run of one definition
  # prints only its own, which the fallback reads. The narrowing is what
  # keeps one case's result from being read off another's, and it has to
  # stay for as long as a grouped invocation remains possible.
  line=$(grep -F -- "$rel" <<<"$out" | grep -oE 'pass=[0-9]+ fail=[0-9]+ blocked=[0-9]+ skip=[0-9]+' | tail -1)
  [ -n "$line" ] || line=$(grep -oE 'pass=[0-9]+ fail=[0-9]+ blocked=[0-9]+ skip=[0-9]+' <<<"$out" | tail -1)
  if [ -z "$line" ]; then
    if grep -q 'executable file not found' <<<"$out"; then
      verdict=NOTRUN; notrun=$((notrun + 1))
    else
      verdict=BLOCKED; blocked=$((blocked + 1))
    fi
  else
    read -r p f b s2 <<<"$(sed -E 's/[a-z0-9]+=//g' <<<"$line")"
    if   [ "$f" -gt 0 ]; then verdict=FAIL;    fail=$((fail + 1))
    elif [ "$b" -gt 0 ]; then verdict=BLOCKED; blocked=$((blocked + 1))
    elif [ "$s2" -gt 0 ] && [ "$p" -eq 0 ]; then verdict=SKIP; skip=$((skip + 1))
    # Counts alone do not make a pass. A run can report fail=0 and still end
    # non-zero -- a teardown that could not finish, a refusal after the last
    # verdict -- and calling that a pass hides exactly the failures no case
    # reported. So the code decides this last step.
    elif [ "$code" -eq 0 ]; then verdict=PASS; pass=$((pass + 1))
    else verdict=FAIL; fail=$((fail + 1))
    fi
  fi
  i=$((i + 1))
  printf '[%3d/%3d] %-7s %4ds %-10s %s\n' "$i" "$total" "$verdict" "$took" "$label" "$rel" | tee -a "$OUT"
  [ "$verdict" = PASS ]
}

for plan_line in "${PLAN[@]}"; do
  kind=$(cut -f1 <<<"$plan_line")
  if [ "$kind" = UNMAPPED ]; then
    chain=$(cut -f2 <<<"$plan_line"); base=$(cut -f3 <<<"$plan_line"); spec=$(cut -f4 <<<"$plan_line")
    i=$((i + 1)); blocked=$((blocked + 1))
    printf '[%3d/%3d] %-7s %4ds %-10s %s\n' "$i" "$total" BLOCKED 0 "$chain" "${spec#"$ROOT"/}" | tee -a "$OUT"
    printf '          %s has no %s sibling, so this round cannot compose it\n' "$base" "$chain" | tee -a "$OUT"
    continue
  fi
  round=$(cut -f2 <<<"$plan_line"); override=$(cut -f3 <<<"$plan_line")
  SPECS=(); while IFS= read -r one; do [ -n "$one" ] && SPECS+=("$one"); done < <(cut -f4 <<<"$plan_line" | tr ' ' '\n')
  [ ${#SPECS[@]} -gt 0 ] || continue
  # For the group's own header line only. A common round names the chain it
  # forces, so it stands for every case in the group; outside common the
  # cases in a folder declare the same chain today, and the line is a
  # heading, not a verdict. Each case's own label is read below.
  glabel="$round"; [ "$glabel" = "-" ] && glabel="$(chainOf "${SPECS[0]}")"
  OVR=(); [ "$override" != "-" ] && OVR=(--chain-preset "$override")

  # One case, one network. The header paragraph says why and it was measured;
  # what follows is the second measurement, of ignoring it.
  #
  # Carrying one workspace across the cases on a chain, with --keep-up, made
  # the next case's preflight answer "reuse" and skip the bring-up: 89s to 66s,
  # 51s to 28s, 48s to 6s. It also broke two cases that had passed for as long
  # as they have existed.
  #
  # CT-NODE-005 reads snap sync. A client turns snap on only for a node whose
  # head is 0 and switches anything with a block to full, so the case needs a
  # chain with nothing in it -- and the case before it on that preset had left
  # one with blocks. CT-NODE-002 wants 15 generated identities; the group
  # before it on that chain had generated 7, and the shared keys directory
  # still held those.
  #
  # Both are the same mistake: "keep the chain to save the bring-up" was read
  # as "a chain may be shared by anything on the same chain id", and what a
  # case needs of a chain is more than its id. Bisected 2026-10-07 over
  # common/node, 15 cases: main 15/15, the fixes alone 15/15, this sharing
  # 40 pass / 3 fail / 2 blocked.
  #
  # Reuse can come back when a case can SAY it needs a fresh chain. Today that
  # requirement lives in CT-NODE-005's prose, where no sweep can read it.
  #
  # execution.chain is not that lever, so do not reach for it: the comparison
  # stage reads the preflight verdict and nothing else, and `fresh` reuses a
  # matching network exactly as reuse-if-matching does (measured 48/23/13
  # against 48/24/13). What the mode adds is the live per-node baseline that
  # partial reconciliation needs, which a whole-network reuse never asks for.
  printf '          %s: %d case(s)%s\n' "$glabel" "${#SPECS[@]}" \
    "$([ "$override" != "-" ] && echo " on $override")" | tee -a "$OUT"
  for spec in "${SPECS[@]}"; do
    c=$((c + 1)); ws="$WS_BASE/c$c"; rm -rf "$ws"
    start=$SECONDS
    # Per case as well, and for a harder reason than the endpoint. A common
    # round overrides the chain-preset, so there the round IS the chain; a
    # folder outside common does not, and each case names its own. The label
    # also picks the round's flags, so one answer for a mixed folder would
    # run a wemix case against the server set written for stablenet and be
    # refused at place with "p2p_step must be >= 3". No folder mixes chains
    # today; nothing stops one from doing so tomorrow.
    label="$round"; [ "$label" = "-" ] && label="$(chainOf "$spec")"
    read -r -a EXTRA <<<"$(roundFlags "$label")"
    # Asked per case, never per group: a group is a folder and a folder may
    # hold both kinds. tests/tc/basic holds 07-basic-wbft-consensus, which
    # composes its own network, next to 08-attached-chain-produces, which
    # attaches. One answer for the whole group sends whichever of the two did
    # not sort first down the other's path, and `run --workspace-dir` refuses
    # an attach case outright.
    endpoint="$(attachOf "$spec")"
    if [ -n "$endpoint" ]; then
      if [ "${endpoint#*GSTABLE_RPC}" != "$endpoint" ]; then
        out=$(runAttach "$spec" "$ws")
      else
        out=$("$BIN" run "$spec" 2>&1)
      fi
    else
      out=$("$BIN" run "$spec" --workspace-dir "$ws" \
        ${OVR[@]+"${OVR[@]}"} ${EXTRA[@]+"${EXTRA[@]}"} 2>&1)
    fi
    code=$?
    if record "$spec" "$out" $((SECONDS - start)) "$label" "$code"; then
      endWorkspace "$ws" 1
    else
      tail -3 <<<"$out" | sed 's/^/          /' | tee -a "$OUT"
      endWorkspace "$ws" 0
    fi
  done

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
