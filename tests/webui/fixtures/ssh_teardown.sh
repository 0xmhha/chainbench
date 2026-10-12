#!/usr/bin/env bash
set -euo pipefail
[[ $# == 1 && "$1" =~ ^[a-zA-Z0-9-]+$ ]] || exit 2
runtime="${WEBUI_RUNTIME_ROOT:-/private/tmp/chainbench-web-ui-e2e}/$1/ssh"
if [[ -f "$runtime/fixture.pid" ]]; then
  pid=$(cat "$runtime/fixture.pid")
  # Only stop the fixture daemon that still has this invocation's configuration.
  if ps -p "$pid" -o args= | rg -q -- "$runtime/sshd_config"; then kill "$pid"; fi
fi

rm -f "$runtime/client" "$runtime/unauthorized" "$runtime/host" "$runtime/authorized_keys"
