# tests/e2e — live end-to-end verification (Go, gated)

These are the **live-verification tier**: Go tests that boot a network from a
**real chain binary** and drive it end to end. They replace the former
bash+python `tests/repro/*.sh` scripts — network orchestration goes through the
`chainbench` CLI (via `os/exec`), and every assertion is pure Go
(`internal/core/rpc` + `internal/accounts`). No bash, no python, no web3.

## Gating

Every file is behind the `//go:build e2e` tag, so `go test ./...` (CI) never
compiles or runs them. Each test also `t.Skip`s when its chain binary is absent,
so a partial toolchain runs only what it can.

## Running

```sh
# one scenario
GSTABLE_BIN=/path/to/gstable go test -tags e2e -run TestE2E_StablenetChain -v ./tests/e2e/

# everything the environment can run (missing binaries -> SKIP, never FAIL)
GSTABLE_BIN=/path/to/gstable WBFT_BIN=/path/to/go-wbft/build/bin/gwemix \
  go test -tags e2e -v ./tests/e2e/
```

## Binaries & keys

- Chain binaries come from env: `GSTABLE_BIN` (or `gstable` on `PATH`) and
  `WBFT_BIN` (or `gwbft` on `PATH`). Absent → the test skips. The hardfork swap
  also reads `PRE_FORK_BIN` and `POST_FORK_BIN`.
- The `chainbench` CLI the harness drives comes from `CHAINBENCH`; unset, the
  harness builds `./cmd/chainbench` into a temp dir.
- The funded sender is node 1's key from `presets/keys` — a committed **TEST
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
| — | fault under and over 1/3, quorum with 3 and 6 validators (WBFT-007/008/011) | `TestE2E_WbftFaultTolerance`, `TestE2E_WbftFaultHalt`, `TestE2E_WbftQuorumAllRequired`, `TestE2E_WbftQuorum6of6Tolerates1`, `TestE2E_WbftQuorum6of6Halts2` | `WBFT_BIN` |
| — | round-robin proposer, view change (WBFT-003) | `TestE2E_WbftRoundRobinProposer`, `TestE2E_WbftViewChange` | `WBFT_BIN` |
| — | snap sync over a large gap | `TestE2E_WbftSnapSync` | `WBFT_BIN` (`GAP=90`) |
| — | `admin_peers`, `istanbul_isValidator` (RPC-019/023) | `TestE2E_WbftAdminPeers`, `TestE2E_WbftIsValidator` | `WBFT_BIN` |
| — | genesis govNCP whitespace accepted, empty list refused (NODE-006/007) | `TestE2E_WbftGenesisNCPWhitespace`, `TestE2E_WbftGenesisEmptyNCP` | `WBFT_BIN` |
| — | helper checks with no chain binary: the empty-NCP refusal matcher, and an `admin_peers` RPC failure not read as zero peers | `TestEmptyNCPRefusalRejectsUnrelatedFailures`, `TestAdminPeerCountDoesNotHideRPCFailure` | none |

`ncp_init_runtime_test.go` holds the runtime-directory helpers the genesis NCP
tests use; they reuse the storage policy in `tests/webui/runtime_storage.py`.

> **Scenario 2 (wemix→wbft handoff)** lives in the DSL: `tests/tc/go-wemix/hardfork/`
> asserts the croissant handoff and that state written before the fork survives it,
> composed from the chain preset `presets/chain/wemix-to-wbft.json`. The Go e2e that used to hold
> it drove `chainbench upgrade run`, a command the CLI no longer has, and was removed.

## Harness

`harness_test.go` provides the shared helpers: `boot` (`chainbench chain up` into a workspace), the
`network` type, `waitAdvancing`/`waitCross` (WBFT warmup-tolerant polling),
`sendValue`, `deployReturns42`/`waitCallReturns42`, `balance`, and
`presetFundedKey`.
