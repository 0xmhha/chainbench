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
  WEB-01|WEB-02|WEB-03|WEB-04|WEB-05|WEB-06|WEB-07|WEB-08|WEB-09|WEB-10|WEB-11|WEB-12|WEB-13|WEB-14) ;;
  *) echo "unsupported criterion: $criterion" >&2; exit 2 ;;
esac
export PYTHONDONTWRITEBYTECODE=1
proof_status=0
if [[ "$capture" == true ]]; then
  python3 tests/webui/capture.py "$criterion" "$output" || proof_status=$?
else
  python3 tests/webui/recheck.py "$criterion" "$output" || proof_status=$?
fi
storage_status=0
python3 tests/webui/after_proof_storage.py "$criterion" "$output" || storage_status=$?
if ((proof_status != 0)); then
  exit "$proof_status"
fi
exit "$storage_status"
