#!/usr/bin/env bash
# capture-cli.sh — record what a chain binary actually accepts.
#
# The binary is the ground truth for its own surface: every command, every
# subcommand, every flag, as built. Reading the chain's source answers a
# different question (what a flag *does*), and that analysis lives in the
# cli-graph.md next to this output — see docs/chain-analysis/README.md.
#
# Usage:
#   scripts/chain-analysis/capture-cli.sh <chain-repo> <binary> <out-dir>
#
# Example:
#   scripts/chain-analysis/capture-cli.sh ~/work/github/0xmhha/chain/go-wemix \
#       ~/work/github/0xmhha/chain/go-wemix/build/bin/gwemix \
#       docs/chain-analysis/gwemix
set -euo pipefail

repo=${1:?chain repo path}
bin=${2:?binary path}
out=${3:?output directory}

[ -x "$bin" ] || { echo "not executable: $bin" >&2; exit 1; }
mkdir -p "$out"

commit=$(git -C "$repo" rev-parse --short HEAD 2>/dev/null || echo "unknown")
commit_date=$(git -C "$repo" log -1 --format=%ad --date=short 2>/dev/null || echo "unknown")
version=$("$bin" version 2>/dev/null | head -20 || true)

surface="$out/cli-surface.txt"
{
  echo "# CLI surface of $(basename "$bin"), captured from the built binary."
  echo "#"
  echo "# chain repo:   $repo"
  echo "# chain commit: $commit ($commit_date)"
  echo "# captured:     $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "#"
  echo "# Regenerate with scripts/chain-analysis/capture-cli.sh. If the chain"
  echo "# commit above is not the one you are building against, this file is"
  echo "# stale and so is anything derived from it."
  echo
  echo "## version"
  echo "$version" | sed 's/^/    /'
  echo
  echo "## top level"
  "$bin" --help 2>&1 | sed 's/^/    /'
} > "$surface"

# Walk one level of subcommands: the top-level help lists them, and each has
# its own flag set. Two levels covers every command these chains define.
commands=$("$bin" --help 2>&1 |
  awk '/^[A-Z ]*COMMANDS:/{c=1;next} /^[A-Z ]+:/{c=0} c&&NF{print $1}' |
  tr -d ',' | grep -Ev '^-|^$|^h$' | sort -u || true)

for cmd in $commands; do
  {
    echo
    echo "## $cmd"
    "$bin" "$cmd" --help 2>&1 | sed 's/^/    /' || true
  } >> "$surface"
  subs=$("$bin" "$cmd" --help 2>&1 |
    awk '/^[A-Z ]*COMMANDS:/{c=1;next} /^[A-Z ]+:/{c=0} c&&NF{print $1}' |
    tr -d ',' | grep -Ev '^-|^$|^h$' | sort -u || true)
  for sub in $subs; do
    {
      echo
      echo "## $cmd $sub"
      "$bin" "$cmd" "$sub" --help 2>&1 | sed 's/^/    /' || true
    } >> "$surface"
  done
done

flags="$out/cli-flags.txt"
grep -oE '^\s+(--[a-zA-Z0-9._-]+)' "$surface" | tr -d ' ' | sort -u > "$flags"

echo "$surface: $(grep -c '^## ' "$surface") command surface(s), $(wc -l < "$flags") distinct flags (chain $commit)"
