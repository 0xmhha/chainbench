#!/usr/bin/env bash
# setup.sh — bring the local docker servers from nothing to ready-to-test.
#
# The five steps below used to live in README.md as copy-paste blocks, so doing
# them meant assembling eight commands and remembering the order. The order is
# the part that bites: recreating a container empties /data/chainbench/bin, so
# a recreate that is not followed by a binary copy leaves a fleet that composes
# a network and cannot launch it.
#
#   check      docker is up, accounts.env is real
#   generate   gen-env.sh writes build/ (compose, server sets, workspace config,
#              localmap). Nothing in build/ is handwritten.
#   image      docker build of the ubuntu+sshd server image
#   up         docker compose up -d
#   binaries   build each chain's LINUX binary in a golang container and copy it
#              into every server. A darwin build is Mach-O and will not run.
#   verify     every server answers ssh, and holds every binary as an ELF file
#
# Run it with no argument for all of them. Run one step by name to redo that
# step alone -- `binaries` after a recreate, `verify` to check a fleet somebody
# else started.
set -euo pipefail

cd "$(dirname "$0")"

# ---- knobs ---------------------------------------------------------------
# Where the chain sources are checked out. One directory holding one clone per
# chain, named as the table below names them.
CHAIN_SRC_ROOT="${CHAIN_SRC_ROOT:-$HOME/Work/github/chain}"

# gen-env.sh owns the server count and writes it into build/docker-compose.yml.
# This script never declares its own: it asks compose which services exist, so
# `SERVERS=20 ./setup.sh` cannot leave the two disagreeing.
BUILD="${BUILD:-build}"
IMAGE="${IMAGE:-chainbench-server:ubuntu24}"
COMPOSE="$BUILD/docker-compose.yml"

# Where a server keeps the binaries provision launches. It is the workspace
# config's dataRoot + paths.binaries; gen-env.sh writes both, and a change
# there has to be made here too.
REMOTE_BIN_DIR=/data/chainbench/bin

# One line per chain: <name> <repo dir> <build target> <output binary> <builder image>
#
# go-wbft builds ./cmd/gwemix and lands as gwbft: the repository forked from
# go-wemix and kept the command name, while the chain manifest asks for gwbft.
# go-wemix pins golang:1.19 because its go.mod does; the other two declare
# go 1.23 and are built with a current toolchain.
CHAINS=(
  "go-stablenet  go-stablenet  ./cmd/gstable  gstable  golang:1.25"
  "go-wbft       go-wbft       ./cmd/gwemix   gwbft    golang:1.25"
  "go-wemix      go-wemix      ./cmd/gwemix   gwemix   golang:1.19"
)

# A named volume for the module and build caches, so a second build of the same
# chain is quick and a `docker run --rm` does not throw the cache away.
GOCACHE_VOLUME="${GOCACHE_VOLUME:-cbgocache}"

# Where a built binary lands on THIS machine before it is copied in. It has to
# be a path Docker Desktop shares with the VM, and $TMPDIR is not reliably one:
# on macOS `-v /tmp/out:/out` binds the VM's /tmp, so the build writes a file
# the host cannot see and the docker cp that follows finds nothing. Reproduced
# 2026-10-01 -- /out held gstable inside the container while the host's /tmp/out
# was empty. A directory under $HOME is shared, so the artifacts go there.
ARTIFACT_DIR="${ARTIFACT_DIR:-$HOME/cbw/linuxbin}"
# --------------------------------------------------------------------------

RECREATE=0

say()  { printf '\n== %s\n' "$*"; }
info() { printf '   %s\n' "$*"; }
die()  { printf 'setup: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<EOF
usage: ./setup.sh [--recreate] [step [chain...]]

steps:
  (none)     check generate image up binaries verify
  check      docker is up and accounts.env is real
  generate   regenerate build/ with gen-env.sh
  image      build the server image
  up         start the containers
  binaries   build and place chain binaries (all, or only the ones named)
  verify     ssh login and binary presence on every server

options:
  --recreate  recreate containers in the up step. Needed after editing
              gen-env.sh, the Dockerfile, firewall.sh or an account NAME.
              It empties $REMOTE_BIN_DIR, so the binaries step must follow.

environment:
  CHAIN_SRC_ROOT  chain clones live here (now: $CHAIN_SRC_ROOT)
  SERVERS         passed through to gen-env.sh to change the server count
EOF
}

# ---- server list ---------------------------------------------------------
# Asked of compose rather than counted here: build/docker-compose.yml is what
# made the containers, so it is the first answer to "which servers".
#
# build/ is gitignored and a lost or cleared build/ does not stop the fleet --
# the containers keep running. So when the file is gone, fall back to the
# running containers, which is what `binaries` and `verify` actually act on,
# and say which answer is being used: refilling binaries should not require
# regenerating a config the fleet no longer needs.
servers() {
  if [ -f "$COMPOSE" ]; then
    docker compose -f "$COMPOSE" config --services | sort -V
    return
  fi
  local running
  running=$(docker ps --format '{{.Names}}' | sed -n 's/^chainbench-\(server[0-9]*\)$/\1/p' | sort -V)
  [ -n "$running" ] || die "$COMPOSE missing and no chainbench-server container is running -- run ./setup.sh generate && ./setup.sh up"
  printf '%s\n' "$running"
}

# Said once, where it is noticed, rather than on every call of servers().
warn_no_compose() {
  [ -f "$COMPOSE" ] && return 0
  info "note: $COMPOSE is missing, so the server list comes from the running containers."
  info "      tests need build/server-set.yaml and build/workspace-config.yaml -- regenerate with ./setup.sh generate"
}

container_of() { printf 'chainbench-%s\n' "$1"; }

# ---- steps ---------------------------------------------------------------
step_check() {
  say "check"
  docker info >/dev/null 2>&1 || die "docker is not running"
  info "docker $(docker info --format '{{.ServerVersion}}')"

  # gen-env.sh refuses a placeholder password too. Checking here as well is so
  # the message arrives before anything is generated, not halfway through.
  [ -f accounts.env ] || die "accounts.env missing -- cp accounts.env.sample accounts.env and set a real password"
  if diff -q accounts.env accounts.env.sample >/dev/null 2>&1; then
    die "accounts.env is still the sample -- open it and set a real password"
  fi
  info "accounts.env present and edited"

  [ -d "$CHAIN_SRC_ROOT" ] || die "CHAIN_SRC_ROOT $CHAIN_SRC_ROOT does not exist"
  info "chain sources: $CHAIN_SRC_ROOT"
}

step_generate() {
  say "generate build/"
  ./gen-env.sh
  info "$(ls "$BUILD" | tr '\n' ' ')"
}

step_image() {
  say "build image $IMAGE"
  docker build -t "$IMAGE" .
}

# accounts.env is a bind mount, and compose does not read a mount's CONTENTS
# when deciding whether a container is up to date -- so `up -d` leaves a running
# container with the accounts it was started with. setup-accounts.sh applies the
# password on every start, so a restart is enough for a password change (an
# account NAME change needs --recreate, since the compose command carries it).
accounts_newer_than_containers() {
  local s c started
  for s in $(servers); do
    c=$(container_of "$s")
    started=$(docker inspect -f '{{.State.StartedAt}}' "$c" 2>/dev/null) || return 0
    # Both sides as epoch seconds; date -j is BSD, -d is GNU.
    local f_epoch c_epoch
    f_epoch=$(stat -f %m accounts.env 2>/dev/null || stat -c %Y accounts.env)
    # StartedAt is UTC, so the parse has to be too -- read as local time it is
    # off by the zone offset, which on this machine made a container look nine
    # hours older than it is.
    c_epoch=$(TZ=UTC date -j -f '%Y-%m-%dT%H:%M:%S' "${started%.*}" +%s 2>/dev/null \
           || date -d "$started" +%s 2>/dev/null) || return 0
    [ "$f_epoch" -gt "$c_epoch" ] && return 0
  done
  return 1
}

step_up() {
  if [ "$RECREATE" -eq 1 ]; then
    say "start containers (recreating)"
    docker compose -f "$COMPOSE" up -d --force-recreate
    # Stated rather than left to be discovered: a recreated container starts
    # with an empty bin directory, and a network that cannot launch reads as a
    # chainbench bug rather than a missing file.
    info "containers recreated -- $REMOTE_BIN_DIR is empty, the binaries step refills it"
  else
    say "start containers"
    docker compose -f "$COMPOSE" up -d
    if accounts_newer_than_containers; then
      info "accounts.env is newer than a running container -- restarting so the accounts are reapplied"
      docker compose -f "$COMPOSE" restart
    fi
  fi
  info "$(docker compose -f "$COMPOSE" ps --services --filter status=running | wc -l | tr -d ' ') running"
}

# build_one <repo dir> <target> <output> <image> -> prints the local artifact path
build_one() {
  local repo="$1" target="$2" out="$3" image="$4"
  local src="$CHAIN_SRC_ROOT/$repo"
  [ -d "$src" ] || die "$src does not exist (CHAIN_SRC_ROOT=$CHAIN_SRC_ROOT)"

  mkdir -p "$ARTIFACT_DIR"
  # The source is mounted read-only so a build cannot write into the clone.
  docker run --rm \
    -v "$src:/src:ro" -v "$ARTIFACT_DIR:/out" \
    -v "$GOCACHE_VOLUME:/gocache" -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
    -w /src "$image" go build -o "/out/$out" "$target"

  # Checked on the host, not in the container: a build that wrote somewhere the
  # host cannot see succeeds inside and leaves nothing here, which is exactly
  # the mount trap above and reads as a chainbench problem two steps later.
  [ -f "$ARTIFACT_DIR/$out" ] || die "$repo: built $out is not visible at $ARTIFACT_DIR -- is that path shared with Docker Desktop?"
  printf '%s\n' "$ARTIFACT_DIR/$out"
}

# place_one <local artifact> <output name>
place_one() {
  local artifact="$1" out="$2" s c
  for s in $(servers); do
    c=$(container_of "$s")
    docker exec -u root "$c" mkdir -p "$REMOTE_BIN_DIR"
    docker cp "$artifact" "$c:$REMOTE_BIN_DIR/$out"
    # docker cp keeps the host file's mode, and a server launches this directly.
    docker exec -u root "$c" chmod 755 "$REMOTE_BIN_DIR/$out"
  done
}

step_binaries() {
  local want=("$@") line name repo target out image artifact
  warn_no_compose
  for line in "${CHAINS[@]}"; do
    # shellcheck disable=SC2086
    set -- $line
    name="$1" repo="$2" target="$3" out="$4" image="$5"
    if [ "${#want[@]}" -gt 0 ] && ! printf '%s\n' "${want[@]}" | grep -qx "$name"; then
      continue
    fi
    say "binary $out ($name, $image)"
    artifact=$(build_one "$repo" "$target" "$out" "$image")
    # An ELF check on this machine, before 15 copies of a Mach-O file go out.
    case "$(head -c4 "$artifact" | od -An -c | tr -d ' ')" in
      *ELF*) ;;
      *) die "$name: built artifact is not ELF -- the build did not run in a linux container" ;;
    esac
    info "built $(du -h "$artifact" | cut -f1)"
    place_one "$artifact" "$out"
    info "placed on $(servers | wc -l | tr -d ' ') server(s)"
  done
}

step_verify() {
  say "verify"
  local s c user fails=0 missing
  warn_no_compose

  # The harness logs in as the first account of accounts.env; gen-env.sh writes
  # that same account into the server set, so verifying it is verifying the
  # login every run uses.
  user=$(sed -n 's/^DEV_ACCOUNTS=["'"'"']*\([^: ]*\).*/\1/p' accounts.env | head -1)
  [ -n "$user" ] || die "could not read the first account from accounts.env"

  for s in $(servers); do
    c=$(container_of "$s")
    if ! docker exec "$c" id "$user" >/dev/null 2>&1; then
      printf '   %-20s account %s missing\n' "$c" "$user"; fails=1; continue
    fi
    missing=""
    for line in "${CHAINS[@]}"; do
      # shellcheck disable=SC2086
      set -- $line
      if ! docker exec "$c" sh -c "head -c4 $REMOTE_BIN_DIR/$4 2>/dev/null | od -An -c | tr -d ' ' | grep -q ELF"; then
        missing="$missing $4"
      fi
    done
    if [ -n "$missing" ]; then
      printf '   %-20s missing or not ELF:%s\n' "$c" "$missing"; fails=1
    else
      printf '   %-20s ok (%s, binaries present)\n' "$c" "$user"
    fi
  done

  [ "$fails" -eq 0 ] || die "verify failed -- run ./setup.sh binaries"

  # What a binary is cannot be read off the file: gstable and gwbft stamp a Git
  # Commit and gwemix does not, so the version line is printed for the record
  # rather than compared against anything.
  for line in "${CHAINS[@]}"; do
    # shellcheck disable=SC2086
    set -- $line
    printf '   %-10s %s\n' "$4" "$(docker exec "$(container_of "$(servers | head -1)")" "$REMOTE_BIN_DIR/$4" version 2>/dev/null | sed -n '2p' | tr -s ' ' || echo '(no version output)')"
  done
  info "all servers ready"
}

# ---- main ----------------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --recreate) RECREATE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) break ;;
  esac
done

case "${1:-all}" in
  all)
    step_check; step_generate; step_image; step_up; step_binaries; step_verify
    say "done"
    info "next: see README.md for the run commands"
    ;;
  check)    step_check ;;
  generate) step_generate ;;
  image)    step_image ;;
  up)       step_up ;;
  binaries) shift; step_binaries "$@" ;;
  verify)   step_verify ;;
  *) usage; exit 1 ;;
esac
