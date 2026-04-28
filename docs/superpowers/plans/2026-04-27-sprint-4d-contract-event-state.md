# Sprint 4d — Contract / Event / State Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Add contract deploy/call, event log fetch+decode, and account state
assertion commands to the Go `network/` surface. Sprint 4 시리즈의 마지막
sprint — 4d 종료 시점에 evaluation matrix Go column 모든 cell ✅.

**Architecture:** `remote.Client` 에 4 read-only ethclient wrapper 추가
(`CallContract`, `FilterLogs`, `CodeAt`, `StorageAt`). `node.contract_deploy`
는 Sprint 4c 의 fee-mode selector 재사용 (legacy / 1559 / SetCode), `to: nil`
+ optional ABI constructor encoding. `node.contract_call`, `node.events_get`,
`node.account_state` 는 read-only 핸들러 (signer 불필요). `network/internal/
abiutil/` 신설하여 ABI parse/pack/unpack/decode-log 헬퍼 격리. Signer 인터페이스
변경 없음.

**Tech Stack:** Go 1.25, `go-ethereum/ethclient` (CallContract / FilterLogs /
CodeAt / StorageAt), `go-ethereum/accounts/abi`, `go-ethereum/crypto.CreateAddress`.

Spec: `docs/superpowers/specs/2026-04-27-sprint-4d-contract-event-state.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits prefix per sprint convention.

## File Structure

**Create:**
- `network/internal/abiutil/abiutil.go` — ABI parse/pack/unpack/decode-log helpers
- `network/internal/abiutil/abiutil_test.go`
- `tests/unit/tests/node-contract-deploy-call.sh`
- `tests/unit/tests/node-events-get.sh`
- `tests/unit/tests/node-account-state.sh`

**Modify:**
- `network/internal/drivers/remote/client.go` — 4 new methods
- `network/internal/drivers/remote/client_test.go` — 4 new tests
- `network/cmd/chainbench-net/handlers_node_tx.go` — `newHandleNodeContractDeploy`
- `network/cmd/chainbench-net/handlers_node_read.go` — 3 read-only handlers
- `network/cmd/chainbench-net/handlers.go` — 4 dispatcher entries + comment
- `network/cmd/chainbench-net/handlers_test.go` — 4d unit tests
- `network/cmd/chainbench-net/e2e_test.go` — 4 Go E2E tests
- `network/schema/command.json` — 4 new enum entries
- `network/internal/types/command_gen.go` — regenerated
- `docs/EVALUATION_CAPABILITY.md` — flip 4d cells, bump coverage
- `docs/VISION_AND_ROADMAP.md` — Sprint 4d tick
- `docs/NEXT_WORK.md` — P1 done

---

## Task 1 — `remote.Client` extensions + `abiutil` package

**Files:**
- Modify: `network/internal/drivers/remote/client.go`
- Modify: `network/internal/drivers/remote/client_test.go`
- Create: `network/internal/abiutil/abiutil.go`
- Create: `network/internal/abiutil/abiutil_test.go`

### Step 1.1: Client wrappers (RED → GREEN)

Append 4 methods to `client.go`:

```go
func (c *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
    out, err := c.rpc.CallContract(ctx, msg, blockNumber)
    if err != nil {
        return nil, fmt.Errorf("remote.CallContract: %w", err)
    }
    return out, nil
}

func (c *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
    logs, err := c.rpc.FilterLogs(ctx, q)
    if err != nil {
        return nil, fmt.Errorf("remote.FilterLogs: %w", err)
    }
    return logs, nil
}

func (c *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
    code, err := c.rpc.CodeAt(ctx, account, blockNumber)
    if err != nil {
        return nil, fmt.Errorf("remote.CodeAt: %w", err)
    }
    return code, nil
}

func (c *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
    val, err := c.rpc.StorageAt(ctx, account, key, blockNumber)
    if err != nil {
        return nil, fmt.Errorf("remote.StorageAt: %w", err)
    }
    return val, nil
}
```

Tests (one per method, mocking `eth_call` / `eth_getLogs` / `eth_getCode` /
`eth_getStorageAt`). Follow Sprint 4b/4c test patterns.

### Step 1.2: `abiutil` package

`network/internal/abiutil/abiutil.go`:

```go
package abiutil

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
    "math/big"
    "strings"

    "github.com/ethereum/go-ethereum/accounts/abi"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
)

// ParseABI parses a JSON ABI string. Wraps abi.JSON for clearer error context.
func ParseABI(jsonStr string) (abi.ABI, error) {
    parsed, err := abi.JSON(strings.NewReader(jsonStr))
    if err != nil {
        return abi.ABI{}, fmt.Errorf("abiutil.ParseABI: %w", err)
    }
    return parsed, nil
}

// CoerceArgs converts JSON-decoded `[]any` into the Go types each abi.Argument
// expects. Best-effort scalar mapping per spec §4.3 — tuple / nested array /
// fixed-bytesN (N≠32) are out of scope.
func CoerceArgs(inputs abi.Arguments, jsonArgs []any) ([]any, error) {
    if len(inputs) != len(jsonArgs) {
        return nil, fmt.Errorf("abiutil.CoerceArgs: expected %d args, got %d", len(inputs), len(jsonArgs))
    }
    out := make([]any, len(jsonArgs))
    for i, arg := range inputs {
        v, err := coerceOne(arg.Type, jsonArgs[i])
        if err != nil {
            return nil, fmt.Errorf("abiutil.CoerceArgs[%d] (%s): %w", i, arg.Type.String(), err)
        }
        out[i] = v
    }
    return out, nil
}

func coerceOne(t abi.Type, v any) (any, error) {
    switch t.T {
    case abi.UintTy, abi.IntTy:
        return toBigInt(v)
    case abi.AddressTy:
        return toAddress(v)
    case abi.BytesTy:
        return toBytes(v)
    case abi.FixedBytesTy:
        if t.Size == 32 {
            b, err := toBytes(v)
            if err != nil {
                return nil, err
            }
            if len(b) != 32 {
                return nil, fmt.Errorf("bytes32 requires 32 bytes, got %d", len(b))
            }
            var out [32]byte
            copy(out[:], b)
            return out, nil
        }
        return nil, fmt.Errorf("fixed-bytes%d not supported (only bytes32); use raw calldata", t.Size)
    case abi.BoolTy:
        if b, ok := v.(bool); ok {
            return b, nil
        }
        return nil, fmt.Errorf("expected bool, got %T", v)
    case abi.StringTy:
        if s, ok := v.(string); ok {
            return s, nil
        }
        return nil, fmt.Errorf("expected string, got %T", v)
    default:
        return nil, fmt.Errorf("type %s not supported (use raw calldata)", t.String())
    }
}

func toBigInt(v any) (*big.Int, error) {
    switch x := v.(type) {
    case string:
        if strings.HasPrefix(x, "0x") {
            i, ok := new(big.Int).SetString(strings.TrimPrefix(x, "0x"), 16)
            if !ok { return nil, fmt.Errorf("bad hex int: %q", x) }
            return i, nil
        }
        i, ok := new(big.Int).SetString(x, 10)
        if !ok { return nil, fmt.Errorf("bad decimal int: %q", x) }
        return i, nil
    case float64:
        // JSON numbers decode to float64; require integral value.
        i := int64(x)
        if float64(i) != x {
            return nil, fmt.Errorf("non-integral number: %v", x)
        }
        return big.NewInt(i), nil
    case json.Number:
        i, ok := new(big.Int).SetString(string(x), 10)
        if !ok { return nil, fmt.Errorf("bad json.Number: %q", x) }
        return i, nil
    }
    return nil, fmt.Errorf("expected number/string, got %T", v)
}

func toAddress(v any) (common.Address, error) {
    s, ok := v.(string)
    if !ok { return common.Address{}, fmt.Errorf("expected hex string, got %T", v) }
    if !common.IsHexAddress(s) {
        return common.Address{}, fmt.Errorf("not a valid hex address: %q", s)
    }
    return common.HexToAddress(s), nil
}

func toBytes(v any) ([]byte, error) {
    s, ok := v.(string)
    if !ok { return nil, fmt.Errorf("expected hex string, got %T", v) }
    s = strings.TrimPrefix(s, "0x")
    if s == "" { return []byte{}, nil }
    return hex.DecodeString(s)
}

// PackMethodCall ABI-encodes a method call: 4-byte selector + packed args.
func PackMethodCall(parsed abi.ABI, methodName string, jsonArgs []any) ([]byte, error) {
    method, ok := parsed.Methods[methodName]
    if !ok {
        return nil, fmt.Errorf("abiutil.PackMethodCall: method %q not in ABI", methodName)
    }
    coerced, err := CoerceArgs(method.Inputs, jsonArgs)
    if err != nil {
        return nil, err
    }
    return parsed.Pack(methodName, coerced...)
}

// UnpackMethodResult decodes the return data of a method call into JSON-friendly values.
func UnpackMethodResult(parsed abi.ABI, methodName string, data []byte) ([]any, error) {
    return parsed.Unpack(methodName, data)
}

// PackConstructor encodes constructor args (no selector prefix). Empty args → empty bytes.
func PackConstructor(parsed abi.ABI, jsonArgs []any) ([]byte, error) {
    if len(jsonArgs) == 0 {
        return parsed.Pack("")
    }
    coerced, err := CoerceArgs(parsed.Constructor.Inputs, jsonArgs)
    if err != nil {
        return nil, err
    }
    return parsed.Pack("", coerced...)
}

// DecodeLog parses log topics+data via an event ABI definition. Returns a
// JSON-friendly map keyed by argument name.
func DecodeLog(parsedABI abi.ABI, eventName string, log types.Log) (map[string]any, error) {
    event, ok := parsedABI.Events[eventName]
    if !ok {
        return nil, fmt.Errorf("abiutil.DecodeLog: event %q not in ABI", eventName)
    }
    out := map[string]any{}
    if err := parsedABI.UnpackIntoMap(out, eventName, log.Data); err != nil {
        return nil, fmt.Errorf("abiutil.DecodeLog data: %w", err)
    }
    // Indexed args come from topics[1..]
    var indexed abi.Arguments
    for _, a := range event.Inputs {
        if a.Indexed { indexed = append(indexed, a) }
    }
    if len(log.Topics) >= 1+len(indexed) {
        if err := abi.ParseTopicsIntoMap(out, indexed, log.Topics[1:]); err != nil {
            return nil, fmt.Errorf("abiutil.DecodeLog topics: %w", err)
        }
    }
    return out, nil
}
```

Tests (`abiutil_test.go`) — coerce per type, pack constructor with/without args,
pack method call, unpack result, decode log with one indexed + one non-indexed arg.

### Step 1.3: Verify + commit

```bash
cd network && go test ./internal/drivers/remote/... ./internal/abiutil/... -count=1 -v
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./... && gofmt -l network/
```

```bash
git add network/internal/drivers/remote/ network/internal/abiutil/
git commit -m "feat(remote+abiutil): add Client read wrappers + abiutil package

remote.Client gains four read-only wrappers (CallContract, FilterLogs,
CodeAt, StorageAt) needed for Sprint 4d's contract / event / state
commands. Same remote.<Method> error-wrap convention.

network/internal/abiutil is a new package isolating ABI parse / arg
coercion / method pack / result unpack / event decode. JSON-friendly
arg types (decimal/hex string -> big.Int, hex string -> address,
hex string -> []byte, bool, string). Tuple / nested-array /
fixed-bytesN (N!=32) explicitly unsupported — caller can fall back
to raw calldata."
```

---

## Task 2 — `node.contract_deploy`

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_tx.go`
- Modify: `network/cmd/chainbench-net/handlers.go` (registration)
- Modify: `network/cmd/chainbench-net/handlers_test.go`
- Modify: `network/schema/command.json`
- Modify: `network/internal/types/command_gen.go` (regenerated)

### Step 2.1: schema + regen

Add `"node.contract_deploy"` to `command.json` (alphabetical). `go generate ./...`.

### Step 2.2: failing tests first

`TestHandleNodeContractDeploy_Happy_Bytecode`:
- mock captures `eth_sendRawTransaction`
- args: `bytecode: "0x6080..."`, no `abi`
- assert: `data.tx_hash` 0x + 64 hex; `data.contract_address` 0x + 40 hex;
  `crypto.CreateAddress(senderAddr, 0)` 와 일치

`TestHandleNodeContractDeploy_Happy_WithABI`:
- args include `abi: "[{type: constructor, inputs: [{type: uint256}]}]"` and
  `constructor_args: [42]`
- mock captures broadcast bytes; decode raw tx, extract `data` field, verify
  it ends with `bytecode || abi.encode(42)` (64-byte big-endian for uint256)

`TestHandleNodeContractDeploy_MissingBytecode` → INVALID_ARGS  
`TestHandleNodeContractDeploy_BadABI` → INVALID_ARGS  
`TestHandleNodeContractDeploy_ABIArgsMismatch` (1 arg given for 2-input constructor) → INVALID_ARGS

### Step 2.3: implement

`newHandleNodeContractDeploy(stateDir)` 패턴은 `newHandleNodeTxSend` 와 거의 동일:
- args parse / hex 검증 / fee-mode selector 재사용 (legacy / 1559 / SetCode 안 함 — deploy 에 SetCode 는 의미 없음, 단순히 legacy + 1559 만 지원)
- if `abi` provided: `abiutil.ParseABI(abi)` → `abiutil.PackConstructor(parsed, args)` → `data = bytecode || encoded`
- else: `data = bytecode`
- to: nil (contract creation)
- compute `contract_address = crypto.CreateAddress(s.Address(), nonce)`
- sign + broadcast (재사용 가능하면 helper 추출)

selector 는 SetCode 제외 — `deploy + authorization_list` 조합은 본 sprint 미지원.

### Step 2.4: register + verify + commit

Add to `allHandlers`:
```go
"node.contract_deploy": newHandleNodeContractDeploy(stateDir),
```

```bash
cd network && go test ./cmd/chainbench-net/... -count=1 -v -run 'TestHandleNodeContractDeploy'
cd network && go test ./... -count=1 -timeout=60s
git add network/schema/command.json network/internal/types/command_gen.go \
        network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_node_tx.go \
        network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): node.contract_deploy with optional ABI constructor

New command that signs + broadcasts a contract-creation tx (to: nil)
via the existing fee-mode selector (legacy + 1559; SetCode is not a
meaningful pairing with deployment). When the caller provides abi +
constructor_args, the handler parses the ABI, coerces JSON args via
abiutil.CoerceArgs, packs the constructor input, and appends to the
bytecode. When abi is omitted, bytecode is broadcast as-is so callers
that pre-encoded their args still work.

Result returns tx_hash plus the locally-computed contract_address
(crypto.CreateAddress over sender + nonce). Caller uses node.tx_wait
to confirm the tx and then verifies the address has code.

Tests: bytecode-only happy, abi+args happy with packed-data assertion,
missing bytecode, malformed ABI, arg-count mismatch — all INVALID_ARGS
at the boundary before any RPC."
```

---

## Task 3 — `node.contract_call`

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_read.go`
- Modify: `network/cmd/chainbench-net/handlers.go` (registration)
- Modify: `network/cmd/chainbench-net/handlers_test.go`
- Modify: `network/schema/command.json`
- Modify: `network/internal/types/command_gen.go` (regenerated)

### Step 3.1: schema + tests

Add `"node.contract_call"` to enum. Regen.

Tests: `_Happy_Calldata`, `_Happy_ABI`, `_BothCalldataAndABI`, `_NeitherCalldataNorABI`,
`_BadAddress`.

### Step 3.2: implement

```go
func newHandleNodeContractCall(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network         string `json:"network"`
            NodeID          string `json:"node_id"`
            ContractAddress string `json:"contract_address"`
            Calldata        string `json:"calldata"`
            ABI             string `json:"abi"`
            Method          string `json:"method"`
            Args            []any  `json:"args"`
            BlockNumber     json.RawMessage `json:"block_number"`
            From            string `json:"from"`
        }
        // parse, validate
        if req.ContractAddress == "" || !common.IsHexAddress(req.ContractAddress) {
            return nil, NewInvalidArgs(...)
        }
        if req.Calldata != "" && req.ABI != "" {
            return nil, NewInvalidArgs("calldata and abi are mutually exclusive")
        }
        if req.Calldata == "" && req.ABI == "" {
            return nil, NewInvalidArgs("either calldata or abi+method+args required")
        }
        // build calldata
        var calldata []byte
        var parsed abi.ABI
        var method string = req.Method
        if req.Calldata != "" {
            // decode hex
            calldata = ...
        } else {
            parsed, err = abiutil.ParseABI(req.ABI)
            ...
            calldata, err = abiutil.PackMethodCall(parsed, req.Method, req.Args)
            ...
        }
        // resolveNode + dialNode + CallContract
        msg := ethereum.CallMsg{To: &contractAddr, Data: calldata}
        if req.From != "" { msg.From = common.HexToAddress(req.From) }
        result, err := client.CallContract(ctx, msg, blockNum)
        ...
        out := map[string]any{
            "result_raw": "0x" + hex.EncodeToString(result),
        }
        if req.ABI != "" {
            decoded, derr := abiutil.UnpackMethodResult(parsed, method, result)
            if derr != nil {
                return nil, NewInternal("unpack result", derr)
            }
            out["result_decoded"] = decoded
        }
        return out, nil
    }
}
```

### Step 3.3: register + commit

```
git commit -m "feat(network-net): node.contract_call with calldata or ABI mode"
```

---

## Task 4 — `node.events_get`

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_read.go`
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`
- Modify: `network/schema/command.json`
- Modify: `network/internal/types/command_gen.go`

### Step 4.1: schema + tests + impl

Tests: `_Happy_NoDecode`, `_Happy_WithDecode`, `_BadAddress`, `_BadTopicHex`.

```go
func newHandleNodeEventsGet(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network    string          `json:"network"`
            NodeID     string          `json:"node_id"`
            Address    string          `json:"address"`
            FromBlock  json.RawMessage `json:"from_block"`
            ToBlock    json.RawMessage `json:"to_block"`
            Topics     []any           `json:"topics"`     // each elem: string OR []string OR null
            ABI        string          `json:"abi"`
            Event      string          `json:"event"`
        }
        // build ethereum.FilterQuery
        // parse from/to block (int OR "latest"/"earliest")
        // parse topics: nil → wildcard at that position; string → single hash; []string → OR-set
        // call FilterLogs
        // optionally decode each log via abiutil.DecodeLog
        // return {"logs": [...]}
    }
}
```

### Step 4.2: register + commit

---

## Task 5 — `node.account_state`

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_read.go`
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`
- Modify: `network/schema/command.json`
- Modify: `network/internal/types/command_gen.go`

### Step 5.1: schema + tests + impl

Tests: `_Default` (fields omitted → balance/nonce/code), `_StorageOnly` (with key),
`_StorageWithoutKey` → INVALID_ARGS, `_BadAddress`.

```go
func newHandleNodeAccountState(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network     string          `json:"network"`
            NodeID      string          `json:"node_id"`
            Address     string          `json:"address"`
            Fields      []string        `json:"fields"`
            StorageKey  string          `json:"storage_key"`
            BlockNumber json.RawMessage `json:"block_number"`
        }
        // default fields = ["balance", "nonce", "code"]
        // for each requested field, call appropriate Client method
        // out["balance"] = "0x" + bal.Text(16)
        // out["nonce"]   = uint64
        // out["code"]    = "0x" + hex.Encode(code)
        // out["storage"] = "0x" + hex.Encode(slot)
        return out, nil
    }
}
```

### Step 5.2: register + commit + dispatcher test

After all 4 commands registered:
```go
func TestAllHandlers_IncludesNew4dCommands(t *testing.T) {
    h := allHandlers("x", "y")
    for _, name := range []string{"node.contract_deploy", "node.contract_call", "node.events_get", "node.account_state"} {
        if _, ok := h[name]; !ok {
            t.Errorf("allHandlers missing %s", name)
        }
    }
}
```

---

## Task 6 — bash + Go E2E + docs

**Files:**
- Create: `tests/unit/tests/node-contract-deploy-call.sh` — deploy bytecode then call back
- Create: `tests/unit/tests/node-events-get.sh` — eth_getLogs roundtrip
- Create: `tests/unit/tests/node-account-state.sh` — balance/nonce/code default
- Modify: `network/cmd/chainbench-net/e2e_test.go` — 4 cobra in-process E2E tests
- Modify: `docs/EVALUATION_CAPABILITY.md` — flip ✅, bump coverage to ~60%
- Modify: `docs/VISION_AND_ROADMAP.md` — Sprint 4d block ticked
- Modify: `docs/NEXT_WORK.md` — P1 done, §3.5 row added

### Step 6.1: bash tests

Each follows the `node-tx-wait.sh` template (mock + ephemeral port + trap +
readiness poll + bash 3.2 safe).

`node-contract-deploy-call.sh`: deploy bytecode + call eth_call (sequential —
deploy returns tx_hash + contract_address, then call uses contract_address with
calldata).

### Step 6.2: Go E2E

Mirror `TestE2E_NodeTxSend_DynamicFee_AgainstAttachedRemote` shape. 4 tests
total — one per new command.

### Step 6.3: Docs

EVALUATION_CAPABILITY:
- §2: `Contract deploy`, `Contract call (eth_call+ABI)` → ✅ Sprint 4d
- §2: `Event log fetch (eth_getLogs)`, `Event log decode` → ✅ Sprint 4d
- §2: `Account state assert` → ✅ Sprint 4d (fully — was partial balance only)
- §4: `Tx receipt event log` → ✅ Sprint 4d (events_get covers this)
- §5: Sprint 4d row → ✅ 완료 (2026-04-27)
- §6: Go coverage ~45% → ~60%

VISION_AND_ROADMAP §6: Sprint 4d block ticked + 완료 line.

NEXT_WORK §3 P1 → 완료. §3.5 4-row table 에 4d 추가. §2.1 timeline 에 4d row 추가.

### Step 6.4: Verify + commit

```bash
cd network && go test ./... -count=1 -timeout=120s
bash /Users/wm-it-22-00661/Work/github/tools/chainbench/tests/unit/run.sh
go -C network vet ./... && gofmt -l network/
```

Bash count: 30 → 33 (added 3 new bash tests).

```bash
git add tests/unit/tests/ network/cmd/chainbench-net/e2e_test.go docs/
git commit -m "test+docs(sprint-4d): contract / event / state E2E + roadmap

3 new bash spawn tests:
  - node-contract-deploy-call.sh: deploy bytecode + eth_call sequence
  - node-events-get.sh: eth_getLogs roundtrip with topic filter
  - node-account-state.sh: balance/nonce/code default fields

4 new Go E2E tests covering contract_deploy, contract_call,
events_get, account_state.

Roadmap docs:
  - EVALUATION_CAPABILITY ticks all 4d cells; Go coverage 45% -> 60%.
  - VISION_AND_ROADMAP Sprint 4d block ticked.
  - NEXT_WORK P1 marked complete; §3.5 absorbs the 4d sprint.

Sprint 4 series concludes: Go network/ tx + read matrix is now
complete. Sprint 5 (MCP exposure) becomes next P1."
```

---

## Final report (after all tasks)

Commit range, package counts (Go 15 + abiutil = 16), bash counts (30 → 33),
capability matrix delta (5 cells flipped to ✅), confirmed deferrals (Sprint
5c MCP, 5b SSH, 5a capability gate, 5d hybrid, M4 hardcoding, wbft/wemix
realization, Adapter.SupportedTxTypes promotion).
