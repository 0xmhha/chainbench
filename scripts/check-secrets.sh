#!/usr/bin/env bash
# check-secrets.sh — block committing secrets. Scans the staged changes (or all
# tracked files with --all) for high-signal secret patterns and secret-prone new
# files, and exits non-zero if any are found.
#
#   bash scripts/check-secrets.sh          # scan staged changes (pre-commit)
#   bash scripts/check-secrets.sh --all    # scan all tracked files
#
# Install as a pre-commit hook:
#   ln -s ../../scripts/check-secrets.sh .git/hooks/pre-commit
#
# The only intentional key material is presets/keys/ (TEST FIXTURE ONLY) and the
# public test addresses in tests/env/*.env; real secrets belong in the gitignored
# tests/env/secret/ (docs/SECURITY_KEY_HANDLING.md).
set -uo pipefail

# The scan root is the tree this script ships in, resolved from its own
# location rather than from git. Here the two are the same; they are not
# wherever this tree is vendored into a larger repository, and that has
# happened. ALLOW_RE below is written in paths relative to THIS tree, so a
# toplevel that is not this tree would fail to exempt our own fixtures while
# judging code we do not own.
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" || exit 2

# Definite-secret content patterns (no legitimate use in this repo).
CONTENT_RE='AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{50,}|xox[baprs]-[A-Za-z0-9-]{10,}|sk-[A-Za-z0-9]{20,}|-----BEGIN (RSA|OPENSSH|EC|DSA|PGP) PRIVATE KEY-----|AWS_SECRET_ACCESS_KEY[[:space:]]*=[[:space:]]*[A-Za-z0-9/+]{20,}'

# Secret-prone new file paths (allowlist the intentional test fixtures).
FILE_RE='(\.pem|\.key|\.p12|\.pfx|\.keystore|(^|/)id_rsa|(^|/)id_ed25519|(^|/)\.env(\.[^/]+)?)$'
ALLOW_RE='^(presets/keys/|tests/env/[^/]+\.env$|tests/env/secret\.example/)'

if [ "${1:-}" = "--all" ]; then
  files=$(git ls-files)
else
  # --relative, because git prints staged paths from the repository root and
  # this script tests them against the working directory. The two agree here
  # and the flag changes nothing; where they do not, every path would miss the
  # `[ -f "$f" ]` guard below and the scan would pass by finding nothing to
  # read.
  files=$(git diff --cached --name-only --relative --diff-filter=ACM)
fi
[ -z "$files" ] && { echo "check-secrets: nothing to scan"; exit 0; }

fail=0
while IFS= read -r f; do
  [ -f "$f" ] || continue
  # skip session transcripts / this scanner itself
  case "$f" in
    docs/dev/session-data/*|scripts/check-secrets.sh) continue ;;
  esac

  # secret-prone file paths not on the allowlist
  if printf '%s' "$f" | grep -qiE "$FILE_RE" && ! printf '%s' "$f" | grep -qE "$ALLOW_RE"; then
    echo "SECRET-PRONE FILE: $f (private-key/credential path — do not commit; use tests/env/secret/)"
    fail=1
  fi

  # secret content
  if grep -HnIE "$CONTENT_RE" "$f" >/dev/null 2>&1; then
    echo "SECRET CONTENT in $f:"
    grep -nIE "$CONTENT_RE" "$f" | sed 's/^/    /' | head -3
    fail=1
  fi
done <<< "$files"

if [ "$fail" -ne 0 ]; then
  echo ""
  echo "check-secrets: potential secrets found (see above). Commit blocked."
  echo "  - real keys/credentials -> tests/env/secret/ (gitignored)"
  echo "  - if this is a false positive, adjust scripts/check-secrets.sh"
  exit 1
fi
echo "check-secrets: OK"
