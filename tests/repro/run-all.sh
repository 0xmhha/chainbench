#!/usr/bin/env bash
# run-all.sh — orchestrate every tests/repro live-verification script.
#
# Runs each repro script and classifies it by exit code:
#   0  PASS   ran and its assertions were green
#   2  SKIP   a prerequisite (chain binary / tool / env) was missing — see reason
#   *  FAIL   ran and an assertion or the network failed
#
# Exits non-zero iff at least one script FAILED. Missing prerequisites never fail
# the run, so a partial environment (e.g. only GSTABLE_BIN, no WEMIX_BIN) reports
# exactly which scenarios it could and could not exercise. This is the single
# entry point for the B-track live verification described in tests/repro/README.md.
#
# Usage:
#   # everything the environment can run:
#   GSTABLE_BIN=/path/to/gstable tests/repro/run-all.sh
#   # a subset:
#   GSTABLE_BIN=/path/to/gstable tests/repro/run-all.sh stablenet-sync-gap.sh
#
# Env:
#   CHAINBENCH  reuse a prebuilt CLI (default: build once to /tmp)
#   REBUILD=1   force a rebuild of the CLI
#   LOGDIR      per-script logs (default: /tmp/chainbench-repro-logs)
# Plus the per-script vars (GSTABLE_BIN, WEMIX_BIN, WBFT_BIN, TEMPLATE, L2_RPC,
# FAUCET_PK, CHAINBENCH_FUNDED_KEY) documented in README.md — passed through.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="$REPO/tests/repro"
CHAINBENCH="${CHAINBENCH:-/tmp/chainbench-repro-bin}"
LOGDIR="${LOGDIR:-/tmp/chainbench-repro-logs}"
mkdir -p "$LOGDIR"

# Build the CLI once so the child scripts reuse it instead of each rebuilding.
if [ ! -x "$CHAINBENCH" ] || [ "${REBUILD:-0}" = 1 ]; then
  echo "building chainbench CLI -> $CHAINBENCH"
  (cd "$REPO" && go build -o "$CHAINBENCH" ./cmd/chainbench) || { echo "run-all: go build failed"; exit 2; }
fi
export CHAINBENCH

# Which scripts: explicit args, or every *.sh except this one and LOCAL-* captures.
SCRIPTS=()
if [ "$#" -gt 0 ]; then
  for a in "$@"; do SCRIPTS+=("$(basename "$a")"); done
else
  for s in "$DIR"/*.sh; do
    b="$(basename "$s")"
    case "$b" in run-all.sh | LOCAL-*) continue ;; esac
    SCRIPTS+=("$b")
  done
fi

pass=0 skip=0 fail=0
ROWS=()
for s in "${SCRIPTS[@]}"; do
  if [ ! -f "$DIR/$s" ]; then
    ROWS+=("$(printf '%-6s %-34s %s' "MISS" "$s" "no such script in tests/repro")")
    fail=$((fail + 1))
    continue
  fi
  log="$LOGDIR/${s%.sh}.log"
  echo ">>> $s"
  bash "$DIR/$s" >"$log" 2>&1
  code=$?
  case $code in
  0) status=PASS; pass=$((pass + 1)); reason="" ;;
  2) status=SKIP; skip=$((skip + 1))
     reason="$(grep -iE 'no .*binary|missing|requires|needs?|not found|set [A-Z_]{3,}|provide|L2_RPC|FAUCET_PK' "$log" | head -1 | sed 's/^[[:space:]]*//' | cut -c1-60)" ;;
  *) status=FAIL; fail=$((fail + 1)); reason="exit=$code; see $log" ;;
  esac
  ROWS+=("$(printf '%-6s %-34s %s' "$status" "$s" "$reason")")
done

echo
echo "================== tests/repro summary =================="
for r in "${ROWS[@]}"; do echo "$r"; done
echo "---------------------------------------------------------"
echo "PASS=$pass  SKIP=$skip  FAIL=$fail   (per-script logs: $LOGDIR)"
[ "$fail" -eq 0 ]
