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
| `contract-read-and-tx-status.json` | a read-only contract call (`call`) and a receipt-status check on the transaction the step just sent (`sendTx` + `save`, then `txStatus` on `$sent`) |
| `cross-call-comparison.json` | step value binding: `read` two on-chain values, then compare later reads against them (`$totalSupply`, `$holderBalance`) — the cross-call comparison a single assertion cannot express |
| `stablenet-governance-read.json` | a stablenet-only governance read: `call` with `proposals(uint256)` calldata (`internal/chains/stablenet/govbind`), asserting the ABI-encoded status word |
| `network-peers.json` | a multi-node check with `onEach` (`bp1`..`bp4`): every validator reports at least one peer — a DSL port of the legacy `tests/network` peers-connected case |
| `network-health.json` | cross-node `sameBlockHash` (genesis agreement) + `blockAdvance` (head is producing) — DSL ports of the legacy `tests/network` genesis-hash-agreement and block-progression cases |
| `stablenet-system-contracts.json` | `codeAt` (adapter deployed, `NotEqual` "0x") + `call` reads (balanceOf/isAuthorized) shape-checked with `Regexp` — DSL ports of the legacy `tests/anzeon` adapter-code and readable-getter cases |
| `stablenet-gas-policy.json` | `baseFee` within the anzeon min/max bounds — DSL port of the legacy `tests/anzeon` basefee-minimum/maximum cases |
| `stablenet-hardfork.json` | post-Boho artifacts as plain reads: P-256 precompile `call` (valid → success word, corrupted → not), GovMinter `codeAt`, and chainId/blockNumber — DSL port of the legacy `tests/anzeon` hardfork-reads cases |
| `stablenet-estimate-gas.json` | `estimateGas` for a native-coin `transfer` exceeds the 21000 bare-transfer floor — DSL port of the legacy `tests/anzeon` estimate-gas-token-transfer case |
| `stablenet-token-metadata.json` | `call` name()/symbol() returns contain the token symbol bytes (`Contains`) — DSL port of the legacy `tests/anzeon` token-metadata case |

## Step value binding

An assertion compares one read against a literal. To compare two reads against
each other — or to check the receipt of a transaction the spec just sent — save a
value and reference it:

```jsonc
{"steps": [
   {"read":   {"source": "call", "to": "0x…1001", "data": "0x18160ddd", "save": "totalSupply"}},
   {"sendTx": {"from": "0x…", "to": "0x…", "value": "1", "save": "sent"}}],
 "assertions": [
   {"assert": "balanceAt", "address": "0x…", "compare": "LessOrEqual", "expected": "$totalSupply"},
   {"assert": "txStatus",  "hash": "$sent", "expected": "0x1"}]}
```

- `save` on any action binds its result. `sendTx` binds its transaction hash.
- `$name` as a whole string substitutes the value with its type intact;
  `${name}` interpolates it into a longer string (assembling calldata); `$$` is a
  literal `$`.
- A reference nothing saved earlier is an error, and `chainbench validate`
  reports it offline as `UNRESOLVED: ref:<name>` — before any chain starts.
- `read` takes a `source` naming any of the RPC-reading assertions (`call`,
  `balanceAt`, `codeAt`, `nonceAt`, `blockNumber`, `chainId`, `peerCount`,
  `baseFee`, `estimateGas`, `txStatus`), so there is one vocabulary, not two.
