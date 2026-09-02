# tests/repro — live verification runbook

These scripts drive the **built chainbench CLI against real chain binaries** to
reproduce regression scenarios that cannot run in CI (no chain binary in the
sandbox). They are the live-verification tier for the Go test-bench: the matching
testkit cases under `tests/` are registration- and capability-gated in CI, and
these scripts confirm the actual on-chain behavior in a normal environment.

Run each from the repo root once you have the relevant node binary. Each script
guards its requirements and exits `2` if a binary/tool is missing, so a bare run
prints exactly what it needs.

### Chain-support scenarios

The multi-chain support matrix maps to these scripts:

1. **wemix chain** — `wemix-chain.sh` (pure go-wemix+etcd; tx + contract)
2. **wemix→wbft hardfork** — migrated to Go: `TestUpgradeRunE2E` in `cmd/chainbench/` (croissant handoff + post-fork state/tx/contract)
3. **wbft chain** — migrated to Go: `TestE2E_WbftChain` in `tests/e2e/` (fresh go-wbft from genesis)
4. **stablenet chain** — migrated to Go: `TestE2E_StablenetChain` in `tests/e2e/`
5. **stablenet hardfork** — migrated to Go: `TestE2E_StablenetHardforkSwap` in `tests/e2e/` (binary-swap in place)

> **Migration in progress:** these bash scripts are being ported to Go gated e2e
> tests under `tests/e2e/` (run with `go test -tags e2e`) — no bash/python/web3.
> Ported so far: stablenet chain (4), binary-swap hardfork (5), consensus
> lifecycle, block propagation, endpoint re-sync, proposal expiry, wbft chain (3), wemix→wbft handoff (2). See `tests/e2e/README.md`. The scripts below are
> not yet ported.

## Run everything: `run-all.sh`

`run-all.sh` is the single entry point. It builds the CLI once, runs every script,
and classifies each by exit code — `0` PASS, `2` SKIP (a prerequisite was
missing), anything else FAIL — then prints a summary and exits non-zero **iff**
something actually failed. Missing binaries never fail the run, so a partial
environment reports exactly which scenarios it could and could not exercise.

```sh
# run all the stablenet scenarios a gstable binary can cover:
GSTABLE_BIN=/path/to/gstable tests/repro/run-all.sh

# full sweep (all three chains + hardforks + external attach + funded writes):
GSTABLE_BIN=/path/to/gstable WEMIX_BIN=/path/to/gwemix WBFT_BIN=/path/to/gwbft \
  TEMPLATE=/path/to/template.json FAUCET_PK=0x... EXTERNAL_RPC=http://... \
  POST_FORK_BIN=/path/to/post-fork/gstable \
  tests/repro/run-all.sh

# a subset:
GSTABLE_BIN=/path/to/gstable tests/repro/run-all.sh stablenet-delayed-fork.sh
```

Per-script logs land in `LOGDIR` (default `/tmp/chainbench-repro-logs`); a FAIL
row points at its log. Pass `REBUILD=1` to force a CLI rebuild, or `CHAINBENCH=`
to reuse a prebuilt binary.

## Scripts

| Script | Reproduces | Needs | Notes |
|--------|-----------|-------|-------|
| `wemix-chain.sh` | **pure wemix chain (scenario 1)** — wemix+etcd, tx + contract | `WEMIX_BIN`, `TEMPLATE`, `FAUCET_PK`, etcd/jq/python3 + web3 | boots a go-wemix producer (poa + governance + etcdInit, no croissant), asserts block production, then a value transfer and a returns-42 contract deploy/call |
| `stablenet-delayed-fork.sh` | delayed-Boho fork transition (h-15/16/27/29/35) | `GSTABLE_BIN`, python3 | boots with `--set genesis.overrides.bohoBlock=N`; runs the `delayed-boho`-gated cases + governance writes (`GOV=0` to skip); fails on any skip |
| `stablenet-account-extra.sh` | account-Extra bitmap (h-30/33/34) | `GSTABLE_BIN`, python3 | boots with `--genesis-overlay internal/chains/stablenet/overlays/account-extra.json`; runs the `account-extra`-gated cases; fails on any skip |
| `stablenet-basefee-dynamics.sh` | baseFee increase/stable/decrease (c-03/c-04/c-05) | `GSTABLE_BIN`, `FAUCET_PK`, python3 + web3 | burst load past 20% usage → assert next baseFee rose; a 6-20% block → assert unchanged (best-effort, reported if the band is not hit); idle → assert it fell. Load/timing sensitive (repro-only) |
| `attach-external.sh` | external-chain generic ops (legacy z-layer2 "Layer 2 E2E" RT-Z-02/03/04/05 — an external chain attached over RPC, NOT an Ethereum L2) | `EXTERNAL_RPC` (an already-running chain's RPC; no chain binary). Optional `CHAINBENCH_FUNDED_KEY` for write ops | attaches to the external endpoint and runs the chain-agnostic (rpc-only) read/state cases; with `CHAINBENCH_FUNDED_KEY` set, also runs the write cases (value transfer, fee delegation). Fails on any read skip |

`LOCAL-*.sh` are ad-hoc local captures (gitignored pattern aside) and are not part
of the runbook.

## Secrets

No private key is committed. Scripts that need a funded sender read it from an env
var (`FAUCET_PK`), never a literal. Chain-agnostic write cases read the funded
account key from `CHAINBENCH_FUNDED_KEY` (env only — the `chainbench test` command
never takes it as a flag). The SSH RemoteDriver E2E reads `CHAINBENCH_REMOTE_PASS`
from the environment only.

## Gated Go E2E

`internal/core/driver/remote_e2e_test.go` (build tag `e2e`) drives the SSH RemoteDriver
against a real sshd. `go test ./...` never runs it. Run with the Docker stand-in:

```sh
tests/remote/sshd/run.sh
# or manually:
CHAINBENCH_REMOTE_HOST=127.0.0.1 CHAINBENCH_REMOTE_PORT=2222 \
CHAINBENCH_REMOTE_USER=chainbench CHAINBENCH_REMOTE_PASS=chainbench \
go test -tags e2e -run TestRemoteDriver_E2E -v ./internal/core/driver/
```

## Running scripts individually

Prefer `run-all.sh` (above). To drive one scenario directly — each script
self-builds the CLI to `/tmp` if `CHAINBENCH` is unset:

```sh
go build -o /tmp/chainbench ./cmd/chainbench   # optional; reused via CHAINBENCH

GSTABLE_BIN=/path/to/gstable CHAINBENCH=/tmp/chainbench \
  tests/repro/stablenet-delayed-fork.sh

GSTABLE_BIN=/path/to/gstable FAUCET_PK=0x... CHAINBENCH=/tmp/chainbench \
  tests/repro/stablenet-basefee-dynamics.sh
```
