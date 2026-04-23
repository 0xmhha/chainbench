#!/usr/bin/env bash
# Fake chainbench dispatcher for Sprint 2b.2 tests.
# Supports: chainbench-stub.sh node stop <N>
# Exits 0 on success, 1 if node num is the literal "fail" sentinel,
# 2 on unknown subcommand.

set -u

subcmd="${1:-}"
action="${2:-}"

case "$subcmd $action" in
  "node stop")
    node="${3:-}"
    if [[ -z "$node" ]]; then
      echo "missing node num" >&2
      exit 1
    fi
    if [[ "$node" == "fail" ]]; then
      echo "stub: forced failure for testing" >&2
      exit 1
    fi
    echo "stub: node $node stopped"
    exit 0
    ;;
  *)
    echo "stub: unknown command: $*" >&2
    exit 2
    ;;
esac
