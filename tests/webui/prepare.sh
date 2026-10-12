#!/usr/bin/env bash
set -euo pipefail
[[ $# == 1 && "$1" =~ ^[a-zA-Z0-9-]+$ ]] || { echo 'requires invocation ID' >&2; exit 2; }
runtime="${WEBUI_RUNTIME_ROOT:-/private/tmp/chainbench-web-ui-e2e}/$1"
mkdir -p "$runtime"
go build -o "$runtime/chainbench-dashboard" ./cmd/chainbench-dashboard
