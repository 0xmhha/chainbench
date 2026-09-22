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
  out=$("$BIN" run "$spec" --workspace-dir "$ws" 2>&1)
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
  # Leaving a network up would let it collide with the next case's ports.
  rm -rf "$ws"
done

{
  echo ""
  echo "=== TOTAL ==="
  printf '{"pass": %d, "fail": %d, "blocked": %d, "skip": %d, "notrun": %d}\n' \
    "$pass" "$fail" "$blocked" "$skip" "$notrun"
  echo "DONE"
} | tee -a "$OUT"
