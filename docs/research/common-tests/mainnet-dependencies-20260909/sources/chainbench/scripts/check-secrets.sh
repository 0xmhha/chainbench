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
# The only intentional key material is keys/preset/ (TEST FIXTURE ONLY) and the
# public test addresses in tests/env/*.env; real secrets belong in the gitignored
# tests/env/secret/ (docs/SECURITY_KEY_HANDLING.md).
set -uo pipefail

cd "$(git rev-parse --show-toplevel)" || exit 2

# Definite-secret content patterns (no legitimate use in this repo).
CONTENT_RE='AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{50,}|xox[baprs]-[A-Za-z0-9-]{10,}|sk-[A-Za-z0-9]{20,}|-----BEGIN (RSA|OPENSSH|EC|DSA|PGP) PRIVATE KEY-----|AWS_SECRET_ACCESS_KEY[[:space:]]*=[[:space:]]*[A-Za-z0-9/+]{20,}'

# Secret-prone new file paths (allowlist the intentional test fixtures).
FILE_RE='(\.pem|\.key|\.p12|\.pfx|\.keystore|(^|/)id_rsa|(^|/)id_ed25519|(^|/)\.env(\.[^/]+)?)$'
ALLOW_RE='^(keys/preset/|tests/env/[^/]+\.env$|tests/env/secret\.example/)'

if [ "${1:-}" = "--all" ]; then
  files=$(git ls-files)
else
  files=$(git diff --cached --name-only --diff-filter=ACM)
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
