#!/bin/sh
# setup-accounts.sh — create the shared dev accounts inside a server container.
#
# Runs at container start (gen-env.sh wires it into the compose command),
# reading /etc/chainbench/accounts.env mounted from env/docker/accounts.env.
# Idempotent: existing accounts are kept and the password is (re)applied on
# every start, so an accounts.env edit needs no image rebuild.
#
# Fails loudly on a missing or malformed file: the harness login is the first
# account in it, so a fleet without these accounts is broken either way — a
# silent skip would only move the failure to an opaque chown/auth error later.
set -eu

ACCOUNTS_ENV=/etc/chainbench/accounts.env

die() { echo "setup-accounts: $*" >&2; exit 1; }

[ -f "$ACCOUNTS_ENV" ] || die "$ACCOUNTS_ENV missing (env/docker/accounts.env not mounted?)"
. "$ACCOUNTS_ENV"

[ -n "${DEV_ACCOUNTS:-}" ] || die "DEV_ACCOUNTS is empty in $ACCOUNTS_ENV"
[ -n "${DEV_ACCOUNTS_PASSWORD:-}" ] || die "DEV_ACCOUNTS_PASSWORD missing in $ACCOUNTS_ENV"

# DEV_ACCOUNTS is a space-separated list of name:uid pairs.
for entry in $DEV_ACCOUNTS; do
    case $entry in
        *:*) ;;
        *) die "entry '$entry' is not name:uid" ;;
    esac
    name=${entry%%:*}
    uid=${entry##*:}
    [ -n "$name" ] || die "entry '$entry' has an empty name"
    case $uid in
        '' | *[!0-9]*) die "entry '$entry' has a non-numeric uid" ;;
    esac
    id "$name" >/dev/null 2>&1 \
        || useradd -m -u "$uid" -s /bin/bash -G sudo "$name"
    # printf, not echo: /bin/sh is dash here and its echo eats backslashes.
    printf '%s:%s\n' "$name" "$DEV_ACCOUNTS_PASSWORD" | chpasswd
done
