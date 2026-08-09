# Example DSL test specs

Illustrative `chainbench` DSL test specs — a declarative alternative to Go-func
test cases. They demonstrate the built-in assertions and actions and are
validated in CI (`go test ./cmd/chainbench` runs `chainbench validate` over this
directory), so they always parse.

The addresses are placeholders; point a spec's addresses/data at your own network
before running it.

Run a spec against a live network (attach) or a locally built one:

```sh
# offline check (no chain needed)
chainbench validate examples/specs/*.json
chainbench validate --chain stablenet examples/specs/smoke-rpc-reads.json
chainbench validate --json examples/specs/*.json   # machine-readable

# run against a running node
chainbench run --chain stablenet --rpc http://127.0.0.1:8600 examples/specs/smoke-rpc-reads.json
chainbench run --json --chain stablenet --rpc http://127.0.0.1:8600 examples/specs/smoke-rpc-reads.json  # JSON verdict for CI

# run against a locally built network
chainbench run --chain stablenet --binary ./gstable examples/specs/smoke-rpc-reads.json
```

| Spec | Shows |
|------|-------|
| `smoke-rpc-reads.json` | RPC-read assertions (`chainId`, `blockNumber`, `peerCount`) with a `compare` override |
| `tx-transfer-and-balance.json` | `sendTx` and `waitBlock` steps, then `balanceAt`/`nonceAt` assertions |
| `negative-revert.json` | a negative tx step (`expectRevert`) — the transaction must revert (F11) |
| `contract-read-and-tx-status.json` | a read-only contract call (`call`) and a receipt-status check (`txStatus`) |
| `stablenet-governance-read.json` | a stablenet-only governance read: `call` with `proposals(uint256)` calldata (`internal/chains/stablenet/govbind`), asserting the ABI-encoded status word |
| `network-peers.json` | a multi-node check with `onEach` (`bp1`..`bp4`): every validator reports at least one peer — a DSL port of the legacy `tests/network` peers-connected case |
| `network-health.json` | cross-node `sameBlockHash` (genesis agreement) + `blockAdvance` (head is producing) — DSL ports of the legacy `tests/network` genesis-hash-agreement and block-progression cases |
| `stablenet-system-contracts.json` | `codeAt` (adapter deployed, `NotEqual` "0x") + `call` reads (balanceOf/isAuthorized) shape-checked with `Regexp` — DSL ports of the legacy `tests/anzeon` adapter-code and readable-getter cases |
| `stablenet-gas-policy.json` | `baseFee` within the anzeon min/max bounds — DSL port of the legacy `tests/anzeon` basefee-minimum/maximum cases |
| `stablenet-hardfork.json` | post-Boho artifacts as plain reads: P-256 precompile `call` (valid → success word, corrupted → not), GovMinter `codeAt`, and chainId/blockNumber — DSL port of the legacy `tests/anzeon` hardfork-reads cases |
| `stablenet-estimate-gas.json` | `estimateGas` for a native-coin `transfer` exceeds the 21000 bare-transfer floor — DSL port of the legacy `tests/anzeon` estimate-gas-token-transfer case |
| `stablenet-token-metadata.json` | `call` name()/symbol() returns contain the token symbol bytes (`Contains`) — DSL port of the legacy `tests/anzeon` token-metadata case |
