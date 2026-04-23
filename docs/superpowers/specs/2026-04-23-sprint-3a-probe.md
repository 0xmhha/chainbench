# Sprint 3a — chain_type probe Design Spec

> 2026-04-23 · Sub-sprint of VISION §6 Sprint 3
> Scope: read-only RPC probe for chain_type/chain_id detection. No RemoteDriver, no signer, no adapter porting.

## 1. Goal

Detect the `chain_type` of an unknown EVM RPC endpoint from a single user-supplied URL.
Provides the foundation for `network attach <url>` (Sprint 3b) to auto-populate `chain_type`
in the network state.

## 2. Non-Goals

- Network-state mutation (attach/save) — that's 3b.
- Authentication (API key/JWT) — that's 3b RemoteDriver.
- Go adapter reimplementation — that's 3c.
- Capability gating — that's Sprint 5.
- Persistent caching of probe results.

## 3. User-Facing Surface

### 3.1 CLI (via chainbench-net wire protocol)

```json
{"command":"network.probe","args":{"rpc_url":"http://127.0.0.1:8545"}}
```

**args:**
- `rpc_url` (string, required) — HTTP(S) JSON-RPC endpoint
- `timeout_ms` (integer, optional, default 5000) — per-probe wall clock timeout
- `override` (string, optional) — force chain_type, skip namespace probes

**Result data on success:**
```json
{
  "chain_type": "stablenet",
  "chain_id":   8283,
  "rpc_url":    "http://127.0.0.1:8545",
  "namespaces": ["istanbul"],
  "overridden": false,
  "warnings":   []
}
```

**Error shapes:**
- `UPSTREAM_ERROR` — RPC unreachable, timeout, non-numeric chainId
- `INVALID_ARGS` — missing/malformed rpc_url, unknown override value

### 3.2 Bash client

```bash
data=$(cb_net_call "network.probe" '{"rpc_url":"http://localhost:8545"}') || exit $?
chain_type=$(jq -r .chain_type <<<"$data")
```

## 4. Detection Algorithm

Table-driven, deterministic:

1. **chain_id probe** — `eth_chainId` required. Failure → UPSTREAM_ERROR (cannot proceed).
2. **Override short-circuit** — if `args.override` set and valid, return immediately with
   `overridden: true`. Do not issue namespace probes.
3. **Namespace probes** — fire each candidate's probe method with `params: []`. Record
   the set of succeeded namespaces (namespace = method prefix before `_`). Probe order:
   - `wemix_getReward` → hints wemix
   - `istanbul_getValidators` → hints stablenet or wbft
4. **Classification** (first match wins):
   - wemix namespace hit → `chain_type: "wemix"`
   - istanbul namespace hit + chain_id ∈ known_stablenet_ids → `chain_type: "stablenet"`
   - istanbul namespace hit (otherwise) → `chain_type: "wbft"`
   - no matches → `chain_type: "ethereum"`
5. **Known id table** (extensible constant):
   - `stablenet`: `{8283}` (placeholder; extend as needed)

Rationale for distinguishing stablenet vs wbft by chain_id: both use `istanbul_*` namespace.
Chain-id is authoritative; namespace alone is ambiguous.

## 5. Package Layout

```
network/internal/probe/
├── probe.go       # Detect(ctx, opts) entry + Options/Result types
├── rpc.go         # minimal JSON-RPC POST helper
├── signatures.go  # table-driven chain signatures + known_ids
└── probe_test.go  # table-driven tests with httptest.Server mock
```

**Package API:**
```go
package probe

type Options struct {
    RPCURL   string
    Timeout  time.Duration   // 0 → default 5s
    Override string          // "" → auto-detect
    Client   *http.Client    // nil → http.DefaultClient
}

type Result struct {
    ChainType  string
    ChainID    int64
    RPCURL     string
    Namespaces []string
    Overridden bool
    Warnings   []string
}

func Detect(ctx context.Context, opts Options) (*Result, error)
```

`Detect` returns `*Result, nil` on success. Returns `nil, *APIError` on typed failure
(same APIError sentinel shape as handlers).

## 6. HTTP / JSON-RPC

Raw `net/http` POST with JSON body `{"jsonrpc":"2.0","id":1,"method":"<name>","params":[]}`.
Rationale: avoids pulling in `go-ethereum` for a read-only probe. Sprint 3b RemoteDriver
will introduce `ethclient`; probe stays light.

Namespace probe "success" = HTTP 200 + JSON response with `result` field present AND
no `error` object. Method-not-found (-32601) returns `error` → counts as unavailable.

## 7. Handler Integration

`cmd/chainbench-net/handlers.go`:
- Add `newHandleNetworkProbe()` closure
- Register `"network.probe"` in `allHandlers()` dispatch
- Args struct: `{RPCURL string `json:"rpc_url"`; TimeoutMs *int `json:"timeout_ms"`; Override string `json:"override"`}`
- Validation: rpc_url required; parse URL (reject non-http(s)); override if set must be in enum

## 8. Schema & Wire

- `command.json` already lists `"network.probe"` — no change.
- `event.json` Result is untyped; data shape is contractual per-command. No schema change.
- No new event types emitted during probe (finite synchronous call).

## 9. Testing Strategy

**Unit (probe_test.go):**
- Table-driven: each entry supplies mock RPC behavior (method → response handler) and
  expected Result.ChainType/Namespaces.
- Cases: stablenet (istanbul + chain_id 8283), wbft (istanbul + different id), wemix,
  ethereum fallback, override short-circuit, timeout, eth_chainId failure, malformed URL.
- Mock uses `httptest.NewServer` dispatching on `request.method` field.

**E2E (cmd/chainbench-net/e2e_test.go):**
- Spawn chainbench-net binary, feed probe command via stdin, assert NDJSON result
  terminator matches expected shape. Uses in-test `httptest.Server` as mock RPC.

**Bash (tests/unit/tests/network-probe.sh):**
- Start python one-shot HTTP server as mock RPC (matches existing test patterns).
- Call `cb_net_call "network.probe" ...` and verify jq-extracted fields.

## 10. Error & Logging Boundaries

- No PII/keys in logs (no auth involved in 3a).
- Log probe start/complete via existing slog in handler; log level INFO with rpc_url,
  chain_type, chain_id. Do not log request/response bodies.
- Timeouts logged as WARN; classification fallback (`ethereum`) logged as INFO with
  observed namespaces.

## 11. Non-Functional

- Total probe wall-clock ≤ timeout_ms; implemented via ctx deadline, not sequential
  timeouts per call (prevents cumulative overrun).
- Probe is stateless — no temp files, no state dir mutation.
- Concurrency: namespace probes fired serially for simplicity. Can parallelize later
  if latency matters (unlikely; N probes × N is small).

## 12. Open Questions (answered)

- **Q: Use go-ethereum ethclient?** → No, Sprint 3a keeps it dep-free. 3b pulls ethclient.
- **Q: Parallelize namespace probes?** → No, serial is fine for ≤3 probes.
- **Q: Cache probe results?** → No, 3a is stateless. 3b may add TTL cache during attach.
- **Q: chain_type "unknown" sentinel?** → No, fall back to "ethereum" (matches network.json enum).

## 13. Out-of-Scope Reminders (session-local, not memory)

Carried from previous sprints, still deferred:
- 2b.3 M3: APIError.Details structured fields
- 2c M3: jq 3→1 call consolidation in `_cb_net_parse_result`
- 2c M4: jq version gate
