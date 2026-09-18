#!/usr/bin/env bash
# handoff-wemix-wbft.sh — case 2: go-wemix produces up to the croissant fork,
# go-wbft validators take over after it.
#
# This used to be 148 lines of hand-ordered phases, because the composer could
# not express a network that runs two builds and hands production from one to
# the other. It can now: the env declares which build each node runs and where
# the fork is, and the cross-fork step does the handover. What is left here is
# the one command that runs it, kept because two documents point at this path.
#
# The old body called `chain up --case/--profile/--from-binary/--to-binary/
# --template/--data-dir/--stop-after` and `chain down`. None of those exist any
# more — only `--to-binary`, and on `hardfork` rather than `chain up` — so the
# script had not been runnable for some time and nothing said so. The CLI-flag
# check in cmd/chainbench/cliflags_test.go is what found it.
#
# Usage:
#   scripts/chain-setup/handoff-wemix-wbft.sh [workspace-dir]
#
# Environment:
#   GWEMIX_BIN         go-wemix build (the producer, up to the fork)
#   GWBFT_BIN          go-wbft build (the successors; its make target is also
#                      named gwemix)
#   GOWEMIX_TEMPLATE   go-wemix's own wemix/scripts/genesis-template.json
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHAIN_DIR="${CHAIN_DIR:-$HOME/Work/github/chain}"

# Keep the workspace short: a node's IPC socket path must stay under 104 chars.
WS="${1:-/tmp/handoff}"

export GWEMIX_BIN="${GWEMIX_BIN:-$CHAIN_DIR/go-wemix/build/bin/gwemix}"
export GWBFT_BIN="${GWBFT_BIN:-$CHAIN_DIR/go-wbft/build/bin/gwemix}"
export GOWEMIX_TEMPLATE="${GOWEMIX_TEMPLATE:-$CHAIN_DIR/go-wemix/wemix/scripts/genesis-template.json}"

for v in GWEMIX_BIN GWBFT_BIN GOWEMIX_TEMPLATE; do
  [ -f "${!v}" ] || { echo "$v is not a file: ${!v}" >&2; exit 1; }
done

rm -rf "$WS"
exec go run ./cmd/chainbench run \
  --workspace-dir "$WS" \
  --binary "$GWEMIX_BIN" \
  --no-skips \
  "$REPO/tests/tc/go-wemix/hardfork/01-croissant-successors-take-over.json"
