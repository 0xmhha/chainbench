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

# run against a running node
chainbench run --chain stablenet --rpc http://127.0.0.1:8600 examples/specs/smoke-rpc-reads.json

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
