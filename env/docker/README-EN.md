# env/docker — local docker standing in for remote servers

English version of [`README.md`](README.md). Both describe the same tree; when
they disagree the Korean one is the original.

These are fake servers that let you exercise chainbench's remote code paths
without real remote hosts. Design and rationale are in
[`docs/dev/docker-remote-design.md`](../../docs/dev/docker-remote-design.md);
the work status is worklist §1g, track R.

- A container is an empty ubuntu server plus sshd. **Access looks exactly like a
  real server**: the first account in `accounts.env` (`devuser1` by default) with
  the shared password, no key login, no root login, and **sudo asks for that
  password** (not NOPASSWD — exercising that flow is the point). Real servers are
  also reached through a provisioned dev account, so the shape matches. No chain
  binary is baked in; provisioning puts it there, as on a real server. The image
  carries no accounts at all — every account is injected at start.
- 15 servers by default (`server1`–`server15`). **server15 is meant to host the
  pn node**, but the server layer does not distinguish it; roles come from the
  netmap assignment.
- Published ports bind to **127.0.0.1 only**. Nothing outside this machine can
  reach them.
- **The firewall is shaped like a real server's**: what the allowlist names is
  open and the rest of inbound is DROPped. What each container's `firewall.sh`
  opens is TCP 10022, 8501-8504, 8601-8604, 8701-8704, 30301-30313, 6060, 3000,
  3001, 9100, 9090, 1099, 5901, 5044, 9200, and UDP 30303. The p2p band is the
  wide one because `firewall.sh` computes it from the slot count: `30301 + SLOTS
  x 3`, which opens 13 ports at the default settings. Every other band is one
  port per slot. **How many ports are open is not how many a set uses** — what
  gets used is the server set's decision.
- **sshd listens on 10022**. The port layout follows the same scheme as a real
  server: p2p 30301 with step 1, http 8601, ws 8701, auth 8501, metrics 6060.

## Using it

```bash
cd env/docker
cp accounts.env.sample accounts.env            # open it, set a real password
./setup.sh                                     # everything else
```

`setup.sh` runs the six steps below in order. Name a step to redo just that one
— `./setup.sh binaries` after recreating containers, `./setup.sh verify` to
inspect servers someone else brought up. `--recreate` rebuilds the containers,
which empties `bin/`, so the binaries step has to follow.

| Step | What it does |
|---|---|
| `check` | Is docker up, and is `accounts.env` something other than the sample |
| `generate` | Regenerate `build/` with `gen-env.sh` |
| `image` | Build the server image |
| `up` | Start the containers |
| `binaries` | Build each chain's **Linux** binary in a golang container and push it to every server |
| `verify` | Per server: the accounts exist and all three binaries are ELF |

By hand it looks like this. This is what `setup.sh` does.

```bash
cd env/docker
cp accounts.env.sample accounts.env            # open it, set a real password
./gen-env.sh                                   # generates all of build/ (nothing is hand-written)
docker build -t chainbench-server:ubuntu24 .
docker compose -f build/docker-compose.yml up -d
ssh -p 2201 devuser1@127.0.0.1 hostname   # password: the accounts.env value -> server1
```

Running `gen-env.sh` without `accounts.env` — or with the password still at the
sample value `change-me` — stops with a message, so a sudo account is never
brought up on a placeholder password. The file is gitignored and must not be
committed.

`gen-env.sh` writes five files into `build/`. None of them is hand-written.

| File | What |
|---|---|
| `docker-compose.yml` | 15 servers, published ports bound to loopback only |
| `server-set.yaml` | For stablenet and wbft. `p2p_step` 1, 4 slots per server |
| `server-set-wemix.yaml` | For poa. `p2p_step` 3, one node per server |
| `workspace-config.yaml` | `dataRoot=/data/chainbench`, `paths.binaries=bin` |
| `localmap.yaml` | The address translation table `--docker` uses |

## If you change a script, regenerate and recreate

`build/` is the output of `gen-env.sh`, and the containers are made from that
output. So editing a script **does not change a running container.** All three
steps have to happen.

```bash
cd env/docker
./gen-env.sh                                                   # regenerate build/
docker compose -f build/docker-compose.yml up -d --force-recreate
# recreating empties /data/chainbench/bin/, so push the binaries again (below)
```

Skipping this on 2026-09-30 cost two websocket cases. `gen-env.sh` had been
fixed on 2026-09-19 to publish the ws band, but `build/docker-compose.yml` was
still the 2026-09-10 file, and the containers were built from that old compose,
so 8701 was never opened. `ws-subscribe-new-heads` and `ws-subscribe-logs` fell
over with `dial ws://127.0.0.1:8701: connection refused`, and finding out why
took a while. What a running container publishes right now is one command away:
`docker port chainbench-server1`.

## Push the binaries first

`./setup.sh binaries` does exactly what this section describes (name a chain for
just one: `./setup.sh binaries go-wemix`). Below is what it does and why.

Each container needs the target chain's **Linux** binary in
`/data/chainbench/bin/`. A build made on a Mac is Mach-O and will not run. Build
it in a golang container and copy it in (settled on 2026-09-29).

**Keep the output directory under your home.** Do not use `/tmp` — on macOS
Docker Desktop, `-v /tmp/out:/out` mounts **the VM's `/tmp`**, not the host's.
The build succeeds and the `docker cp` on the next line, which runs on the host,
cannot find the file. Reproduced on 2026-10-01: `/out` inside the container held
`gstable` while the host's `/tmp/out` was empty.

```bash
OUT=~/cbw/linuxbin && mkdir -p "$OUT"

# Mount the repo read-only so the output cannot overwrite an existing build/bin.
# go-stablenet
docker run --rm -v ~/Work/github/chain/go-stablenet:/src:ro -v "$OUT":/out \
  -v cbgocache:/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -w /src golang:1.25 go build -o /out/gstable ./cmd/gstable

# go-wbft — build ./cmd/gwemix and install it as gwbft. The repo forked from
# go-wemix and kept the command name; the manifest asks for gwbft.
docker run --rm -v ~/Work/github/chain/go-wbft:/src:ro -v "$OUT":/out \
  -v cbgocache:/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -w /src golang:1.25 go build -o /out/gwbft ./cmd/gwemix

# go-wemix — its go.mod asks for 1.19.
docker run --rm -v ~/Work/github/chain/go-wemix:/src:ro -v "$OUT":/out \
  -v cbgocache:/gocache -e GOCACHE=/gocache/build -e GOMODCACHE=/gocache/mod \
  -w /src golang:1.19 go build -o /out/gwemix ./cmd/gwemix

for i in $(seq 1 15); do
  docker exec -u root chainbench-server$i mkdir -p /data/chainbench/bin
  for b in gstable gwbft gwemix; do
    docker cp "$OUT/$b" chainbench-server$i:/data/chainbench/bin/$b
    docker exec -u root chainbench-server$i chmod 755 /data/chainbench/bin/$b
  done
done

docker exec chainbench-server1 /data/chainbench/bin/gstable version | head -3
```

### The two cases that need a second build

`signature-compat-across-swap` and `boho-crossed-by-restart` measure **swapping
one build for another**. Each names its second build `gstable-hardfork` and
`gstable-postfork` respectively. A name is the path on the target, so two names
that are both `gstable` swap one file for itself and look like a swap -- which
is what happened until 2026-10-06.

Without them in place the run stops at BLOCKED before a network is composed, and
says which declaration's path was empty. Add the name to the `for b in gstable
gwbft gwemix` loop above and build it into `$OUT`. Which commit to build from is
in [`tests/tc/HOW-TO-USE.md`](../../tests/tc/HOW-TO-USE.md) section 3.2.

That last line tells you what actually landed. `gstable` and `gwbft` print a
`Git Commit`; `gwemix` does not, because that build does not embed one — the
binary alone cannot be traced back, so note the HEAD you built from somewhere
else.

## Which server set

go-wemix uses `server-set-wemix.yaml`. poa claims three consecutive ports next to
p2p, so `p2p_step` has to be 3; the default set is 1 and gets refused.

Spreading nodes over several machines needs `--all-servers`. The set has 15
servers, so naming none is refused and naming one uses only that server's 4
slots. A case that splits the network with firewall rules needs a different
machine per group, so the flag is mandatory there.

## Running tests across the 15

`--all-servers` spreads one node per server across all 15, and `--docker`
translates dials through the localmap. The standard 15-node smoke test
(7 bp, 7 en, 1 pn):

```bash
# Each container needs the target chain's Linux binary in /data/chainbench/bin/.
bin/chainbench run \
  --workspace-dir <ws> \
  --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers \
  --keys <ws>/genkeys \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json
```

The `chain-up-15` env block declares the 15-node topology. It does not declare a
binary path: which binary is answered by the chain manifest, by name, and where
that name lives is answered by the `dataRoot` and `paths.binaries` of the
`workspace-config.yaml` passed above. Put together they give
`/data/chainbench/bin/gstable`. A case that writes the path directly only runs in
this docker environment.

Keys are generated rather than taken from a preset, because 15 are needed and the
preset has 5. A generated set declares only the topology's bp count (7) as
validators.

### Running the whole sweep on these 15

`scripts/tcsweep.sh` runs against local binaries by default. Set `TCSWEEP_FLAGS`
and the same sweep points at these containers — on a machine with no local chain
binaries that is the only path. Every case gets a `chain stop` (the ports have to
be released for the next case), and **only the ones that passed** go on to
`chain rm`. A case that did not pass keeps the container's datadir and node logs
and the local workspace, and prints those paths: what you need to look at a
failure is the machine state behind it, not the evidence in `~/.chainbench`. The
ones that passed are removed because leaving them makes the next case's genesis
collide with `incompatible genesis`.

**You cannot run everything in one pass. It takes three.** The server set differs
per chain, and `tcsweep.sh` applies one `TCSWEEP_FLAGS` to every case.

```bash
cd <chainbench directory>      # where env/ and scripts/ are visible
make build

export TCSWEEP_FLAGS="--server-set $PWD/env/docker/build/server-set.yaml \
  --workspace-config $PWD/env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate"

# Pass 1 — 177 stablenet + 8 wbft. Excludes go-stablenet/testnet (see below).
scripts/tcsweep.sh ~/cbw/sweep-1.log \
  'tests/tc/basic\|tests/tc/common\|tests/tc/go-wbft\|go-stablenet/regression\|go-stablenet/post-v1.0.0-change\|go-stablenet/hardfork\|go-stablenet/vocabulary'

# Pass 2 — 7 go-wemix cases. Different server set and a different startup budget.
TCSWEEP_FLAGS="--server-set $PWD/env/docker/build/server-set-wemix.yaml \
  --workspace-config $PWD/env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys-source generate --node-monitor-timeout 5m" \
scripts/tcsweep.sh ~/cbw/sweep-2.log 'tests/tc/go-wemix'
```

Pass 2 is separate for the reason given under "Which server set" above: poa needs
`p2p_step` 3 while the default set is 1, so it is refused at the place step, and
it also boots slowly enough to need a raised `--node-monitor-timeout`.

**Pass 3 is not docker.** The 5 cases under `tests/tc/go-stablenet/testnet/`
attach to a network we did not bring up. `tcsweep.sh`'s `runAttach` supplies only
`GSTABLE_RPC` to an attach case, while these five need `GSTABLE_TESTNET_RPC`, so
mixing them into pass 1 turns them all BLOCKED. How to run them is in
[`tests/tc/go-stablenet/testnet/README.md`](../../tests/tc/go-stablenet/testnet/README.md).

`--server-set` without `--server` is refused at the place step: it cannot know
which of the 15 you meant. `--all-servers` spreads one node per server and
removes the question.

A quick case takes about 30 seconds (measured on these 15 on 2026-10-01 with
CT-RPC-001). Something like CT-FAULT-004, which boots 9 nodes, splits them and
rejoins them, takes minutes — so run pass 1 under `nohup` or `tmux` and watch it
with `tail -f`.

The gate on the Go test side is `CHAINBENCH_DOCKER_SERVERS`:

```bash
CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test -p 1 -timeout 60m ./...
```

**A live test fails on a missing fixture rather than skipping.** The top of each
test's comment says what to plant with `docker exec ...`. Run it without planting
and it looks like a regression.

### 15-server smoke test for all three families (stablenet, wbft, go-wemix)

All three families have been verified on the 15 (verified at the time with
4 bp + 11 en; the current standard is 7 bp, 7 en, 1 pn):

```bash
# stablenet (wbft family)  — server-set.yaml, default gates
bin/chainbench run --workspace-dir <ws> --server-set env/docker/build/server-set.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys <ws>/genkeys --keys-source generate \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json

# wbft                     — server-set.yaml, default gates
bin/chainbench run ... --chain-preset wbft-bp7-en7-pn1 tests/tc/common/node/CT-NODE-002-startup-15-nodes.json

# go-wemix (poa)           — needs server-set-wemix.yaml and a raised gate budget
bin/chainbench run --workspace-dir <ws> --server-set env/docker/build/server-set-wemix.yaml \
  --workspace-config env/docker/build/workspace-config.yaml \
  --docker --all-servers --keys <ws>/genkeys --keys-source generate \
  --node-monitor-timeout 5m \
  --chain-preset wemix-bp7-en7-pn1 \
  tests/tc/common/node/CT-NODE-002-startup-15-nodes.json
```

go-wemix (poa) differs from stablenet/wbft in two ways:

1. **A different server set.** poa claims three consecutive ports per node next
   to p2p (p2p + etcd), so it needs `p2p step >= 3`. `gen-env.sh` emits
   `server-set-wemix.yaml` (slots 1, p2p step 3) for exactly this. Bringing wemix
   up on the ordinary `server-set.yaml` (step 1) is refused at the place step
   with `p2p_step must be >= 3`.
2. **A raised gate budget.** poa comes up in the order producer boot (4 bp) →
   governance deployment → etcd formation → endpoint (11 en) join, and the sync
   of a late-joining endpoint takes longer than nodemonitor's default 90s wait.
   `--node-monitor-timeout 5m` stops the gate from killing a network that is
   still forming.

**Before rerunning with a different chain, remove the old composition.** With a
different genesis (different keys, different family), a leftover datadir or
genesis blocks init with `incompatible genesis`. `chain rm` works on remote
targets too (since 2026-09-11), so there is no need to empty the containers by
hand:

```bash
bin/chainbench chain stop --workspace-dir <ws>   # rm refuses while nodes run
bin/chainbench chain rm   --workspace-dir <ws>
```

**It removes only its own composition** — other compositions on the same server,
and `bin/`, are untouched (paths are split by composition id, and deletion is
confined to the target's dataRoot). Verified live: stablenet 3 nodes → stop → rm
→ wbft 3 nodes came up on the same servers **with no manual cleanup** and
progressed to block 8.

Empty them by hand only when the workspace is lost and `chain rm` cannot be run:

```bash
for i in $(seq 1 15); do docker exec chainbench-server$i sh -c \
  'cd /data/chainbench && find . -maxdepth 1 -mindepth 1 ! -name bin -exec rm -rf {} +'; done
```

Generated files (`build/`, gitignored):

| File | Contents |
|---|---|
| `docker-compose.yml` | server1..N. Fixed bridge addresses from 172.30.0.11, ssh 22→2201+, rpc 8600→18601+, metrics 6060→16061+ |
| `server-set.yaml` | **Server set v2 with real addresses** — the same shape as an operational server set. slots 4, p2p step 1 |
| `server-set-wemix.yaml` | Same servers, same credentials, only the poa knobs differ — slots 1, p2p step 3 |
| `workspace-config.yaml` | The target `dataRoot` and the per-purpose directories. The server set no longer carries `dataRoot`, so this must be passed alongside it |
| `localmap.yaml` | Real address → loopback published port. Applied only with `--docker` (R1) |

> **A port with no mapping goes out unchanged, silently.** `AddrMap` rewrites the
> host only and leaves an unmapped port as it is, so dialling a port that was
> never published produces a plausible-looking `127.0.0.1:<container port>` that
> answers nothing. metrics 6060 was exactly that (fixed 2026-09-11). When you
> start using a new port, add it to both the compose publish list and the
> localmap.

To change the server count, run `SERVERS=20 ./gen-env.sh` and bring compose back
up.

## Watching a run

What to look at while a sweep is going. Everything below only reads, so none of
it disturbs a running sweep.

### How far along

```bash
tail -f ~/cbw/sweep-1.log
```

One line per case. There are four verdicts — `PASS`, `FAIL` (the case was
wrong), `BLOCKED` (we never got to ask), `NOTRUN` (no binary found). **Seeing
`NOTRUN` means `TCSWEEP_FLAGS` was not passed**: every run then falls back to
local and looks for the chain binary on this machine's PATH, and it is only in
the containers.

Which case is running right now is answered by the process list.

```bash
ps -eo etime,args | grep '[b]in/chainbench run' | sed -E 's#(--workspace-dir [^ ]*).*#\1#'
```

### Did the chain really go to docker

```bash
W=~/cbw/sweep/c11                                  # the sweep uses cN
bin/chainbench chain status --workspace-dir $W     # how far through the 8 stages
bin/chainbench chain show   --workspace-dir $W     # the node table
```

**Whether it fell back to local is answered directly by `host` in `show`.**
`172.30.0.x` means it went to a container; a Mac path means local. There are six
ways to pick a node — `--node`, `--label` (`node7`, `en2`), `--host`, `--addr`,
`--port`, and `--json`.

### What a node is saying

```bash
bin/chainbench chain logs --workspace-dir $W --node 1 --lines 50
```

This reads the log **out of the container over SSH** (escalating with sudo if it
has to). Check which case it belongs to before reading it as a fault: a fault
case stops consensus on purpose, so repeated ROUND-CHANGE is normal there.

### What the test decided

Evidence accumulates under `~/.chainbench/sessions/<id>/UTC-<ts>/`.

```bash
S=$(ls -dt ~/.chainbench/sessions/*/ | head -1)
cat $S/UTC-*/tests/*/status.json      # the one-line failure reason — the most useful file
cat $S/UTC-*/tests/*/steps.json       # per-step results and errors
cat $S/UTC-*/session.json             # the verdict per case
```

To fold several sessions into one verdict:

```bash
bin/chainbench report --workspace-dir ~/.chainbench/sessions --all
```

**The flag is named `--workspace-dir` but what it takes here is the sessions
directory.** There is no `--session-dir`.

### Revisiting a failed case

The sweep removes only what passed; on a failure it keeps the workspace and the
container's datadir and logs and prints the paths (`kept for inspection: ...`).
`chain-record.json` holds the composition id, each node's host, port and pid, and
the `docker` and `serverSet` settings. Clean up when you are done — the record
carries the docker settings, so `--docker` does not need to be passed again.

```bash
bin/chainbench chain stop --workspace-dir $W   # rm refuses while nodes run
bin/chainbench chain rm   --workspace-dir $W
```

### Looking into the containers directly

The path for when you do not trust the harness.

```bash
for i in $(seq 1 15); do
  n=$(docker exec chainbench-server$i sh -c 'ps -eo args | grep -c "[/]data/chainbench/bin/"')
  echo "server$i: $n"
done

docker exec chainbench-server1 ls /data/chainbench/logs      # <compositionId>/node1.log
docker exec chainbench-server1 ls /data/chainbench/node      # <compositionId>/node1
docker exec chainbench-server1 ls /data/chainbench/runtime   # <compositionId>/genesis, config
docker port chainbench-server1                               # what it publishes right now
```

**A log only exists on the server its node runs on.** `--all-servers` puts one
node per server, so server1 has `node1.log` and node7's log is on server7.

### Do not run two sweeps at once

The paths can be separated with `TCSWEEP_WS`, but **the server ports collide.**
The allocator gives one slot per server, so both sweeps want 8601 and you get
BLOCKED with `another composition holds it` — and then you misread someone else's
composition as a leftover. That happened once, on 2026-10-01. Check first:

```bash
ps -eo etime,args | grep '[t]csweep'
```

## Injecting dev accounts (accounts.env)

The shared dev accounts are written in `accounts.env` (gitignored, copied from
`accounts.env.sample`) as a list of `name:uid` plus the shared password. Every
time a container starts, `setup-accounts.sh` reads that file and creates the
accounts (it is wired into the compose command). After changing a value,
regenerate with `./gen-env.sh` and recreate with `docker compose -f
build/docker-compose.yml up -d` for it to take effect — `restart` reruns the old
compose command and will not pick up a renamed account (a password-only change is
fine with restart). Neither case needs an image rebuild. Other tools that use the
same account details read this file too.

The harness login comes from the same file: `gen-env.sh` writes the **first
account** into the server set (`build/server-set.yaml`) as `ssh: user/password`,
and makes dataRoot owned by that account. There is one source for the account
details, and it is this file.

Teardown:

```bash
docker compose -f build/docker-compose.yml down
```

## Why the server set carries the containers' real addresses

Nodes have to talk to each other inside the bridge using real addresses — those
are the addresses that end up in genesis and static-nodes. The loopback
substitution happens **only at the moment the harness dials something itself**,
and the switch for it is the `--docker` option. The file can sit there; without
the option nothing happens.
