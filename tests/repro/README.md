# tests/repro — live verification runbook

These scripts drive the **built chainbench CLI against real chain binaries** to
reproduce regression scenarios that cannot run in CI (no chain binary in the
sandbox). They are the live-verification tier for the Go test-bench: the matching
testkit cases under `tests/` are registration- and capability-gated in CI, and
these scripts confirm the actual on-chain behavior in a normal environment.

Run each from the repo root once you have the relevant node binary. Each script
guards its requirements and exits `2` if a binary/tool is missing, so a bare run
prints exactly what it needs.

## Scripts

| Script | Reproduces | Needs | Notes |
|--------|-----------|-------|-------|
| `stablenet-delayed-fork.sh` | delayed-Boho fork transition (h-15/16/27/29/35) | `GSTABLE_BIN`, python3 | boots with `--set genesis.overrides.bohoBlock=N`; runs the `delayed-boho`-gated cases + governance writes (`GOV=0` to skip); fails on any skip |
| `stablenet-account-extra.sh` | account-Extra bitmap (h-30/33/34) | `GSTABLE_BIN`, python3 | boots with `--genesis-overlay manifests/overlays/stablenet-account-extra.json`; runs the `account-extra`-gated cases; fails on any skip |
| `stablenet-sync-gap.sh` | endpoint re-sync (a1-02 full, a1-06 downloader, a1-03 snap) | `GSTABLE_BIN`, python3 | `node stop --index` → open a ≥`GAP` block gap → `node start --index` → assert re-sync (head within 2, matching hash + stateRoot, state access, `eth_syncing=false`). Snap: `SYNCMODE=snap GAP=150` |
| `stablenet-basefee-dynamics.sh` | baseFee increase/stable/decrease (c-03/c-04/c-05) | `GSTABLE_BIN`, `FAUCET_PK`, python3 + web3 | burst load past 20% usage → assert next baseFee rose; a 6-20% block → assert unchanged (best-effort, reported if the band is not hit); idle → assert it fell. Load/timing sensitive (repro-only) |
| `wemix-wbft-handoff.sh` | go-wemix→go-wbft croissant handoff (C1–C3) | `WEMIX_BIN`, `WBFT_BIN`, `TEMPLATE`, etcd/jq/python3/curl | passes iff head crosses croissant AND a go-wbft validator mined a post-croissant block |

`LOCAL-*.sh` are ad-hoc local captures (gitignored pattern aside) and are not part
of the runbook.

## Secrets

No private key is committed. Scripts that need a funded sender read it from an env
var (`FAUCET_PK`), never a literal. The SSH RemoteDriver E2E reads
`CHAINBENCH_REMOTE_PASS` from the environment only.

## Gated Go E2E

`pkg/core/driver/remote_e2e_test.go` (build tag `e2e`) drives the SSH RemoteDriver
against a real sshd. `go test ./...` never runs it. Run with the Docker stand-in:

```sh
tests/remote/sshd/run.sh
# or manually:
CHAINBENCH_REMOTE_HOST=127.0.0.1 CHAINBENCH_REMOTE_PORT=2222 \
CHAINBENCH_REMOTE_USER=chainbench CHAINBENCH_REMOTE_PASS=chainbench \
go test -tags e2e -run TestRemoteDriver_E2E -v ./pkg/core/driver/
```

## Typical run

```sh
# build once; each script also self-builds to /tmp if needed
go build -o /tmp/chainbench ./cmd/chainbench

GSTABLE_BIN=/path/to/gstable CHAINBENCH=/tmp/chainbench \
  tests/repro/stablenet-delayed-fork.sh

GSTABLE_BIN=/path/to/gstable CHAINBENCH=/tmp/chainbench \
  tests/repro/stablenet-account-extra.sh

GSTABLE_BIN=/path/to/gstable CHAINBENCH=/tmp/chainbench \
  tests/repro/stablenet-sync-gap.sh

GSTABLE_BIN=/path/to/gstable FAUCET_PK=0x... CHAINBENCH=/tmp/chainbench \
  tests/repro/stablenet-basefee-dynamics.sh
```
