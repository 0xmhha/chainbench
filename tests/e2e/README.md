# tests/e2e — live end-to-end verification (Go, gated)

These are the **live-verification tier**: Go tests that boot a network from a
**real chain binary** and drive it end to end. They replace the former
bash+python `tests/repro/*.sh` scripts — network orchestration goes through the
`chainbench` CLI (via `os/exec`), and every assertion is pure Go
(`pkg/core/rpc` + `pkg/accounts`). No bash, no python, no web3.

## Gating

Every file is behind the `//go:build e2e` tag, so `go test ./...` (CI) never
compiles or runs them. Each test also `t.Skip`s when its chain binary is absent,
so a partial toolchain runs only what it can.

## Running

```sh
# one scenario
GSTABLE_BIN=/path/to/gstable go test -tags e2e -run TestE2E_StablenetChain -v ./tests/e2e/

# everything the environment can run (missing binaries -> SKIP, never FAIL)
GSTABLE_BIN=/path/to/gstable WBFT_BIN=/path/to/gwbft WEMIX_BIN=/path/to/gwemix \
  go test -tags e2e -v ./tests/e2e/
```

## Binaries & keys

- Chain binaries come from env: `GSTABLE_BIN`, `WBFT_BIN`, `WEMIX_BIN` (or the
  binary on `PATH`). Absent → the test skips.
- The funded sender is node 1's key from `keys/preset` — a committed **TEST
  fixture** (public, local-only). It is loaded at runtime, never a literal, and
  is only ever used to fund local ephemeral test networks.

## Scenarios (chain-support matrix)

| # | Scenario | Test | Binary |
|---|----------|------|--------|
| 3 | fresh wbft chain (block/tx/contract) | `TestE2E_WbftChain` | `WBFT_BIN` (go-wbft's `gwemix`) |
| 4 | stablenet chain (block/tx/contract) | `TestE2E_StablenetChain` | `GSTABLE_BIN` |
| 5 | stablenet binary-swap hardfork | `TestE2E_StablenetHardforkSwap` | `GSTABLE_BIN` (+ `POST_FORK_BIN`) |
| — | WBFT consensus lifecycle (b-08/09/10, a1-04) | `TestE2E_StablenetConsensusLifecycle` | `GSTABLE_BIN` |
| — | near-head block propagation (a1-07) | `TestE2E_StablenetBlockPropagation` | `GSTABLE_BIN` |
| — | endpoint re-sync (a1-02/03/06) | `TestE2E_StablenetSyncGap` | `GSTABLE_BIN` (`SYNCMODE=snap GAP=150` for snap) |
| — | proposal expiry → Expired (f3-06) | `TestE2E_StablenetProposalExpiry` | `GSTABLE_BIN` (short-expiry overlay) |

> **Scenario 2 (wemix→wbft handoff)** lives in `cmd/chainbench/upgrade_run_e2e_test.go`
> (`TestUpgradeRunE2E`) — it drives `chainbench upgrade run` and asserts the
> croissant handoff plus post-fork state/tx/contract on the wbft successor. Run it
> with `CHAINBENCH_E2E_FROM_BIN`/`_TO_BIN`/`_TEMPLATE` and etcd on PATH.
>
> Migration in progress: the remaining scenarios (1 wemix pure, basefee, and the
> account-extra / delayed-fork overlay cases — the last two need a gstable build
> matching the fixtures) are being ported from `tests/repro/*.sh`.

## Harness

`harness_test.go` provides the shared helpers: `boot` (setup --launch), the
`network` type, `waitAdvancing`/`waitCross` (WBFT warmup-tolerant polling),
`sendValue`, `deployReturns42`/`waitCallReturns42`, `balance`, and
`presetFundedKey`.
