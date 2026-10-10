"""Bounded forced-command gate for exclusively owned SSH fault fixtures."""
import math
from pathlib import Path
import re
import shlex


def prepare_ssh_gate(runtime, candidate_sha256, timeout_seconds=60):
    if not re.fullmatch(r'[0-9a-f]{64}', candidate_sha256):
        raise ValueError('a verified candidate checksum is required')
    if not 0 < timeout_seconds <= 60:
        raise ValueError('the owned command pause must be bounded to sixty seconds')
    directory = Path(runtime).resolve() / 'ssh'
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    gate = directory / 'gate.sh'
    quote = lambda name: shlex.quote(str(directory / name))
    steps = math.ceil(timeout_seconds / .05)
    script = f'''#!/bin/sh
set -u
umask 077
command=${{SSH_ORIGINAL_COMMAND:-}}
[ -n "$command" ] || exit 2
case "$command" in
 *{candidate_sha256}*)
  case "$command" in
   *"cat >"*)
    if mkdir {quote('gate-once')} 2>/dev/null; then
     /bin/sh -c "$command"
     result=$?
     [ "$result" -eq 0 ] || exit "$result"
     printf '%s %s\\n' "$$" "$PPID" > {quote('gate-copied')}
     remaining={steps}
     while [ ! -f {quote('gate-release')} ]; do
      [ "$remaining" -gt 0 ] || exit 124
      remaining=$((remaining - 1))
      sleep .05
     done
     exit 0
    fi
   ;;
  esac
 ;;
esac
exec /bin/sh -c "$command"
'''
    gate.write_text(script)
    gate.chmod(0o700)
    return gate
