#!/usr/bin/env bash
set -euo pipefail
criterion=''
output='chainbench-out/web-ui-acceptance'
live=false
capture=false
while (($#)); do
  case "$1" in
    --criterion) criterion="$2"; shift 2 ;;
    --output) output="$2"; shift 2 ;;
    --require-live) live=true; shift ;;
    --capture) capture=true; shift ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[[ "$live" == true ]] || { echo 'requires --require-live' >&2; exit 2; }
case "$criterion" in
  WEB-01|WEB-03|WEB-04|WEB-05|WEB-08|WEB-09|WEB-10) ;;
  *) echo "unsupported criterion: $criterion" >&2; exit 2 ;;
esac
export PYTHONDONTWRITEBYTECODE=1
if [[ "$capture" == true ]]; then
  python3 tests/webui/capture.py "$criterion" "$output"
else
  python3 tests/webui/recheck.py "$criterion" "$output"
fi
