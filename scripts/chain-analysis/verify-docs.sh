#!/usr/bin/env bash
# verify-docs.sh — cross-check a chain's cli-graph.md against its built binary.
#
# Neither source nor help output is sufficient alone, and this is why:
#
#   source only  lists flags that are defined but never registered on a
#                command, so the doc promises something the binary rejects
#                (go-stablenet's --docroot: defined at flags.go:187, read at
#                :1726, on no command's flag list).
#   help only    misses flags that are registered but hidden, which are still
#                accepted (--vmodule, --log.json, --discv4).
#
# So each flag the document names but help does not show is asked of the binary
# directly, and reported as HIDDEN (accepted, document is right) or ABSENT
# (rejected, document is wrong).
#
# Usage: scripts/chain-analysis/verify-docs.sh <binary> <chain-dir>
#   e.g. scripts/chain-analysis/verify-docs.sh ~/.../gstable docs/chain-analysis/gstable
set -uo pipefail

bin=${1:?binary path}
dir=${2:?chain analysis directory}
doc="$dir/cli-graph.md"
seen="$dir/cli-flags.txt"

[ -x "$bin" ] || { echo "not executable: $bin" >&2; exit 1; }
[ -f "$doc" ] && [ -f "$seen" ] || { echo "missing $doc or $seen (run capture-cli.sh first)" >&2; exit 1; }

# Only backticked flags: prose mentions a flag in negative statements too
# ("there is no --chainid flag"), and those are not claims that it exists.
documented=$(grep -oE '`--[a-zA-Z0-9._-]+`' "$doc" | tr -d '`' | sort -u)
# Flags the document asserts are NOT accepted. They are checked in the other
# direction: if the binary takes one, the note is wrong and that is worth
# knowing, because it is the kind of note a reader trusts and stops looking.
notaccepted=$(sed -n '/^```not-accepted$/,/^```$/p' "$doc" | grep -oE '^--[a-zA-Z0-9._-]+' | sort -u)
hidden=0 absent=0 wrongnote=0

for flag in $notaccepted; do
  out=$("$bin" "$flag" --help 2>&1 | head -2)
  if echo "$out" | grep -qiE "not defined|flag provided|Incorrect Usage"; then
    continue
  fi
  echo "WRONG   $flag — documented as not accepted, but the binary takes it"
  wrongnote=$((wrongnote + 1))
done
for flag in $documented; do
  grep -qx -- "$flag" "$seen" && continue          # visible in help; nothing to check
  echo "$notaccepted" | grep -qx -- "$flag" && continue   # documented as absent, checked above
  [ ${#flag} -gt 3 ] || continue
  out=$("$bin" "$flag" --help 2>&1 | head -2)
  if echo "$out" | grep -qiE "not defined|flag provided|Incorrect Usage"; then
    echo "ABSENT  $flag — documented but the binary rejects it"
    absent=$((absent + 1))
  else
    echo "HIDDEN  $flag — accepted but not in --help"
    hidden=$((hidden + 1))
  fi
done

echo "$(basename "$dir"): $hidden hidden, $absent absent, $wrongnote wrong note(s) (of $(echo "$documented" | wc -w | tr -d ' ') documented)"
[ "$absent" -eq 0 ] && [ "$wrongnote" -eq 0 ]
