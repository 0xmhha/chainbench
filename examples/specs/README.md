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
| `contract-deploy-and-register.json` | `faucet` tops up a fresh account, `deployContract` binds the deployed address, `registerContract` calls it, then assertions check the code, the receipt, and the balance (F11 AC-6/7) |
| `cross-call-comparison.json` | step value binding: `read` two on-chain values, then compare later reads against them (`$totalSupply`, `$holderBalance`) — the cross-call comparison a single assertion cannot express |
| `stablenet-governance-read.json` | a stablenet-only governance read: `call` with `proposals(uint256)` calldata (`internal/chains/stablenet/govbind`), asserting the ABI-encoded status word |
| `governance-event-flow.json` | a multi-step flow: save the head, send a governance transaction, then assert on the events it emitted (`logs` with a `fromBlock` bound to the saved head) |
| `network-peers.json` | a multi-node check with `onEach` (`bp1`..`bp4`): every validator reports at least one peer — a DSL port of the legacy `tests/network` peers-connected case |
| `network-health.json` | cross-node `sameBlockHash` (genesis agreement) + `blockAdvance` (head is producing) — DSL ports of the legacy `tests/network` genesis-hash-agreement and block-progression cases |
| `stablenet-system-contracts.json` | `codeAt` (adapter deployed, `NotEqual` "0x") + `call` reads (balanceOf/isAuthorized) shape-checked with `Regexp` — DSL ports of the legacy `tests/anzeon` adapter-code and readable-getter cases |
| `stablenet-gas-policy.json` | `baseFee` within the anzeon min/max bounds — DSL port of the legacy `tests/anzeon` basefee-minimum/maximum cases |
| `stablenet-hardfork.json` | post-Boho artifacts as plain reads: P-256 precompile `call` (valid → success word, corrupted → not), GovMinter `codeAt`, and chainId/blockNumber — DSL port of the legacy `tests/anzeon` hardfork-reads cases |
| `stablenet-estimate-gas.json` | `estimateGas` for a native-coin `transfer` exceeds the 21000 bare-transfer floor — DSL port of the legacy `tests/anzeon` estimate-gas-token-transfer case |
| `fault-node-restart.json` | fault injection: `stopNode` a validator, let the rest produce, `startNode` it again and wait for it to catch up — the DSL form of the legacy quorum/sync-recovery e2e cases |
| `fault-partition-fork.json` | `partition` splits the validators into two groups that can no longer see each other, then `healPartition` restores the mesh in a post-action (F8 AC-2) |
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

## Fault injection

Destructive cases need to stop a node or cut the network, not just read from it:

```jsonc
{"steps": [
   {"stopNode":  {"on": "bp4"}},
   {"waitBlock": {"on": "bp1", "target": 5, "timeout": "60s"}},
   {"startNode": {"on": "bp4"}}],
 "postActions": [{"healPartition": {}}]}
```

- `stopNode` / `startNode` / `restartNode` act on one node, resolved by the same
  selectors assertions use (`bp1`, `en:0`, `bp:any`).
- `partition` takes two or more `groups` of selectors and drops every peer link
  that crosses a group boundary, from both sides. `healPartition` re-adds every
  pair; with no `groups` it heals the whole environment, which is what a
  post-action wants.
- These need capabilities the run actually has. `stopNode` and friends require
  process control, so declare `"requires": ["process"]` — attach mode advertises
  only `rpc` and skips the spec rather than failing it, and a run that somehow
  reaches the action without node control fails with that reason named.

## Funding and contracts

```jsonc
{"steps": [
   {"faucet":           {"to": "0x…aa", "amount": "1000000000000000000"}},
   {"deployContract":   {"bytecode": "0x6080…", "gas": "500000", "save": "contract"}},
   {"registerContract": {"to": "$contract", "data": "0x1234abcd", "save": "registration"}}]}
```

- `faucet` funds an address so a key generated at run time — which is not in the
  genesis alloc, and so cannot pay for its own first transaction — can transact.
  Without `from` it spends from the target node's unlocked coinbase.
- `deployContract` is its own action rather than a `sendTx` because a
  deployment's address exists only in the receipt; it binds that address, so
  later steps reach the contract as `$contract`.
- `registerContract` is the intent-revealing form of a call into a deployed
  contract: `to` is required (a missing one would silently deploy again) and a
  revert always fails the step.

## Events

`logs` queries `eth_getLogs` and compares what you select from the result:

```jsonc
{"assert": "logs",
 "address": "0x…1000", "topics": ["0xddf252ad…"],
 "fromBlock": "$before", "toBlock": "latest",
 "select": "count", "compare": "GreaterOrEqual", "expected": "1"}
```

- `select` defaults to `count` — "did this event fire, and how often". The other
  selectors read one matching log: `data`, `address`, `blockNumber`, `txHash`,
  or `topic0`…`topicN`, with `index` choosing which log (default the first).
- Selecting a field when nothing matched is an error, not an empty string; a
  `count` of zero is a legitimate value and compares normally.
- An empty string in `topics` is a wildcard for that position.
- `logs` is also a `read` source, so an event value can be saved and compared
  against a later one.
