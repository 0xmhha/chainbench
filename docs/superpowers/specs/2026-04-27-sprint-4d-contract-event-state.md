# Sprint 4d — Contract / Event / State Design Spec

> 2026-04-27 · VISION §6 Sprint 4 series — Sprint 4d (Sprint 4 시리즈 종료)
> Scope: 마지막 evaluation matrix Go column cells — contract deploy, contract
> call, event log fetch+decode, account state assert. 4d 종료 시점에
> `docs/EVALUATION_CAPABILITY.md` §2 / §4 의 Go column 모든 cell 이 ✅.

## 1. Goal

Sprint 4 시리즈의 마지막 sprint. coding agent 가 chainbench-net 으로 evaluation
시나리오를 짤 때 필요한 contract / event / state 검증 능력을 Go `network/` 에
이식한다. Signer 인터페이스 변경 없음 — 4c 까지 다듬은 surface 위에서 ethclient
의 read-only API + ABI 헬퍼를 추가한다.

## 2. Non-Goals (Deferred)

- **WS subscription** — `eth_subscribe` 기반 실시간 로그/블록 알림. Sprint 5+.
- **Multi-call / batch RPC** — 단일 RPC 호출만. 여러 read 를 한 wire command 에
  묶는 것은 후속.
- **Storage proof / state proof** — `eth_getProof` 미지원. 단순 `eth_getStorageAt`만.
- **`debug_traceTransaction` / state diff** — 디버깅 RPC 표면. 별도 sprint.
- **MCP exposure** — Sprint 5c.
- **Sophisticated ABI type coercion** — JSON args → ABI types 매핑은 best-effort
  scalar 만 (uint/int → big.Int from decimal string, address → common.Address from
  hex, bytes → []byte from hex, bool, string). Tuple / fixed-bytes / nested
  arrays 미지원 — caller 가 raw calldata 를 보내면 됨.

## 3. Security Contract

Sprint 4 / 4b / 4c 의 redaction boundary 그대로 유지. 4d 가 추가하는 surface 는
모두 read-only (`eth_call`, `eth_getLogs`, `eth_getCode`, `eth_getStorageAt`) +
한 종류의 write (`contract_deploy` — sender 의 SignTx 로 처리, fee delegation/
1559/SetCode 와 동일 fee-mode 선택자 흐름). signer 변경 없음.

## 4. User-Facing Surface

### 4.1 `remote.Client` 확장 (Task 1)

```go
func (c *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
func (c *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
func (c *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
func (c *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error)
```

Thin wrappers. 같은 `remote.<Method>` 에러 wrap 컨벤션.

### 4.2 `node.contract_deploy` (Task 2)

```json
{
  "command": "node.contract_deploy",
  "args": {
    "network":   "sepolia",
    "node_id":   "node1",
    "signer":    "alice",
    "bytecode":  "0x6080604052...",
    "abi":       "[{...}]",
    "constructor_args": [42, "0xabcd..."],
    "value":     "0x0",
    "gas":       3000000,
    "max_fee_per_gas":          "0x59682f00",
    "max_priority_fee_per_gas": "0x3b9aca00",
    "nonce":     7
  }
}
```

**Required**: `signer`, `bytecode`. `network`, `node_id`, fee/gas/nonce 동일.

**Optional**:
- `abi` (string, JSON ABI) + `constructor_args` (array of JSON values) — 둘 다
  제공되면 server 가 `abi.Pack("", args...)` 으로 constructor args 인코딩 후
  bytecode 에 append.
- 둘 다 미제공 → bytecode 그대로 (caller 가 이미 args 를 인코딩한 상태)
- `abi` 만 제공 + `constructor_args` 미제공 → `abi.Pack("")` (no args) — bytecode 그대로

**Selection rules**: Sprint 4c 의 fee-mode selector 재사용 (legacy / 1559 /
SetCode 3-way). `to: nil` 만 다름 (contract creation).

**Result**:
```json
{
  "tx_hash": "0x...",
  "contract_address": "0x..."
}
```
`contract_address` = `crypto.CreateAddress(senderAddr, nonce)` — 로컬 계산.
Caller 는 broadcast 후 `node.tx_wait` 으로 receipt 폴링 후 실제 배포 확인.

**Error matrix**:
- INVALID_ARGS — missing bytecode/signer/to never present, malformed hex,
  unknown alias, ABI parse fail, ABI args mismatch
- UPSTREAM_ERROR — dial / chainID / nonce / gas / broadcast 실패
- INTERNAL — SignTx 실패 (invariant)

### 4.3 `node.contract_call` (Task 3)

```json
{
  "command": "node.contract_call",
  "args": {
    "network":          "sepolia",
    "node_id":          "node1",
    "contract_address": "0x...",
    "abi":              "[{...}]",
    "method":           "balanceOf",
    "args":             ["0xabcd..."],
    "block_number":     "latest",
    "from":             "0x..."
  }
}
```

대안 (raw calldata):
```json
{
  "args": {
    "contract_address": "0x...",
    "calldata":         "0x70a08231000000000000000000000000abcd..."
  }
}
```

**Selection rule**: `calldata` 와 `abi`+`method` 는 mutually exclusive. 둘 다
제공 → INVALID_ARGS. 둘 다 미제공 → INVALID_ARGS.

**ABI args 타입 매핑** (best-effort):
| Solidity type | JSON form | Go type |
|---|---|---|
| `uint*` / `int*` | decimal string OR number | `*big.Int` |
| `address` | `"0x" + 40 hex` | `common.Address` |
| `bytes` / `bytes32` | `"0x" + hex` | `[]byte` (32-byte fixed-size 는 자동 패딩) |
| `bool` | `true` / `false` | `bool` |
| `string` | string | `string` |

Tuple / nested array / fixed-bytesN (N≠32) → caller 가 raw `calldata` 사용.

**Result**:
```json
{
  "result_raw": "0x...",
  "result_decoded": [{"type": "uint256", "value": "1000000000000000000"}]
}
```
`result_decoded` 는 `abi`+`method` 경로에서만 채워짐 (raw calldata 경로는
`result_raw` 만).

**Error matrix**: INVALID_ARGS / UPSTREAM_ERROR / INTERNAL (decode 실패는
INTERNAL).

### 4.4 `node.events_get` (Task 4)

```json
{
  "command": "node.events_get",
  "args": {
    "network":    "sepolia",
    "node_id":    "node1",
    "address":    "0x...",
    "from_block": "0x10",
    "to_block":   "latest",
    "topics":     ["0xddf2...", null, "0x000...abcd"],
    "abi":        "[{...}]",
    "event":      "Transfer"
  }
}
```

`address` / `topics` 는 optional (필터링용). `abi` + `event` 는 server-side
decode 위해 optional. 둘 다 제공되면 각 log 의 topics+data 를 ABI 기반으로 decode.

**Result**:
```json
{
  "logs": [
    {
      "block_number":  123,
      "block_hash":    "0x...",
      "tx_hash":       "0x...",
      "tx_index":      0,
      "log_index":     0,
      "address":       "0x...",
      "topics":        ["0x...", "0x..."],
      "data":          "0x...",
      "removed":       false,
      "decoded": {                       // optional, only if abi+event provided
        "event": "Transfer",
        "args":  {"from": "0x...", "to": "0x...", "value": "1000"}
      }
    }
  ]
}
```

**Error matrix**: INVALID_ARGS / UPSTREAM_ERROR / INTERNAL.

### 4.5 `node.account_state` (Task 5)

```json
{
  "command": "node.account_state",
  "args": {
    "network":      "sepolia",
    "node_id":      "node1",
    "address":      "0x...",
    "fields":       ["balance", "nonce", "code", "storage"],
    "storage_key":  "0x0000000000000000000000000000000000000000000000000000000000000000",
    "block_number": "latest"
  }
}
```

**Required**: `address`. `fields` 미제공 → `["balance", "nonce", "code"]` 기본
(storage 는 명시 요청만 — `storage_key` 도 같이 필수).

**Result** (요청된 field 만):
```json
{
  "address": "0x...",
  "block":   "latest",
  "balance": "0x...",
  "nonce":   42,
  "code":    "0x6080...",
  "storage": "0x000000..."
}
```

**Error matrix**: INVALID_ARGS (storage 요청 시 `storage_key` 필수).

## 5. Package Layout

**Modify:**
- `network/internal/drivers/remote/client.go` — 4 신규 메서드.
- `network/internal/drivers/remote/client_test.go` — 4 신규 테스트.
- `network/cmd/chainbench-net/handlers_node_tx.go` — `newHandleNodeContractDeploy`
  (Task 4c 의 fee-mode selector 재사용).
- `network/cmd/chainbench-net/handlers.go` — register 4 신규 명령 + 파일 layout 주석.
- `network/cmd/chainbench-net/handlers_test.go` — 4 신규 핸들러 unit tests.
- `network/cmd/chainbench-net/handlers_node_read.go` — `newHandleNodeContractCall`,
  `newHandleNodeEventsGet`, `newHandleNodeAccountState` (read-only).
- `network/cmd/chainbench-net/e2e_test.go` — Go E2E.
- `network/schema/command.json` — 4 신규 enum.
- `network/internal/types/command_gen.go` — 재생성.
- `tests/unit/tests/node-contract-deploy-call.sh` (NEW)
- `tests/unit/tests/node-events-get.sh` (NEW)
- `tests/unit/tests/node-account-state.sh` (NEW)
- `docs/EVALUATION_CAPABILITY.md` — cells flip + Sprint 4d row + coverage bump
  (~45% → ~60%+ 추정).
- `docs/VISION_AND_ROADMAP.md` — Sprint 4d 체크.
- `docs/NEXT_WORK.md` — P1 (Sprint 4d) 완료 표시 + §3.5 추가.

## 6. ABI helper (Task 2/3)

`network/internal/abiutil/abiutil.go` (NEW):
- `ParseABI(jsonStr string) (abi.ABI, error)`
- `PackArgs(method abi.Method, jsonArgs []any) ([]byte, error)` — JSON args →
  Go types coercion 후 method.Inputs.Pack
- `UnpackResult(method abi.Method, data []byte) ([]any, error)` — JSON-friendly
  values
- `DecodeLog(eventABI abi.Event, log types.Log) (map[string]any, error)` —
  topics+data → JSON-friendly map

타입 coercion 한계는 §4.3 에 명시. 미지원 타입은 INVALID_ARGS 로 surface.

## 7. Tests

### 7.1 remote.Client unit (Task 1)
- `TestClient_CallContract_Happy` — mock 이 0x05 반환, 1-byte slice 검증.
- `TestClient_FilterLogs_Happy` — mock 이 1개 log 반환, 슬라이스 길이 1 검증.
- `TestClient_CodeAt_Happy` — mock 이 bytecode 반환.
- `TestClient_StorageAt_Happy` — mock 이 32-byte slot 반환.
- (각 `_Reject` 변형 — RPC 에러 wrapping 확인)

### 7.2 contract_deploy handler (Task 2)
- `TestHandleNodeContractDeploy_Happy_Bytecode` — abi 미제공, raw bytecode
  broadcast.
- `TestHandleNodeContractDeploy_Happy_WithABI` — abi+args 제공, server 가 args 인코딩.
- `TestHandleNodeContractDeploy_MissingBytecode` → INVALID_ARGS.
- `TestHandleNodeContractDeploy_BadABI` → INVALID_ARGS.
- `TestHandleNodeContractDeploy_ABIArgsMismatch` (잘못된 arg 개수/타입) → INVALID_ARGS.
- (fee-mode selector 는 Task 4c 의 selector 와 동형 — 별도 회귀 테스트 불필요)

### 7.3 contract_call handler (Task 3)
- `TestHandleNodeContractCall_Happy_Calldata` — raw calldata, 결과 hex 반환.
- `TestHandleNodeContractCall_Happy_ABI` — abi+method+args, server 인코딩 + decode.
- `TestHandleNodeContractCall_BothCalldataAndABI` → INVALID_ARGS.
- `TestHandleNodeContractCall_NeitherCalldataNorABI` → INVALID_ARGS.
- `TestHandleNodeContractCall_BadAddress` → INVALID_ARGS.

### 7.4 events_get handler (Task 4)
- `TestHandleNodeEventsGet_Happy_NoDecode` — abi 미제공, raw log 반환.
- `TestHandleNodeEventsGet_Happy_WithDecode` — abi+event, decode 결과 검증.
- `TestHandleNodeEventsGet_BadAddress` → INVALID_ARGS.
- `TestHandleNodeEventsGet_BadTopicHex` → INVALID_ARGS.

### 7.5 account_state handler (Task 5)
- `TestHandleNodeAccountState_Default` — fields 미제공, balance+nonce+code 반환.
- `TestHandleNodeAccountState_StorageOnly` — `fields: ["storage"]` + `storage_key`.
- `TestHandleNodeAccountState_StorageWithoutKey` → INVALID_ARGS.
- `TestHandleNodeAccountState_BadAddress` → INVALID_ARGS.

### 7.6 dispatcher
- `TestAllHandlers_IncludesNew4dCommands` — `node.contract_deploy`, `node.contract_call`,
  `node.events_get`, `node.account_state` 모두 등록 확인.

### 7.7 Go E2E
- `TestE2E_NodeContractDeploy_AgainstAttachedRemote` — attach + deploy with
  bytecode (no abi); broadcast 검증 + contract_address 형식 검증.
- `TestE2E_NodeContractCall_AgainstAttachedRemote` — attach + call (raw calldata).
- `TestE2E_NodeEventsGet_AgainstAttachedRemote` — attach + events_get with
  filter (no decode).
- `TestE2E_NodeAccountState_AgainstAttachedRemote` — attach + state (default
  fields).

### 7.8 Bash spawn tests (Task 6)
- `node-contract-deploy-call.sh` — deploy bytecode + call eth_call sequence.
- `node-events-get.sh` — eth_getLogs roundtrip.
- `node-account-state.sh` — balance/nonce/code 합성 응답.

## 8. Schema

`command.json` enum 에 4 항목 추가 (알파벳 순):
- `node.account_state`
- `node.contract_call`
- `node.contract_deploy`
- `node.events_get`

`go generate ./...` 으로 `command_gen.go` 재생성.

## 9. Error Classification

Sprint 4 / 4b / 4c 와 동일 패턴:
- INVALID_ARGS — 입력 구조/값 문제 (RPC 호출 전)
- UPSTREAM_ERROR — RPC 실패 (`eth_call` / `eth_getLogs` / etc.)
- INTERNAL — ABI parse/pack/unpack 실패 (invariant — 입력은 boundary 에서 검증됨)

## 10. Out-of-Scope Reminders

**Deferred to Sprint 5 / 5+**:
- WebSocket subscription
- `debug_traceTransaction`
- ABI tuple / nested arrays / fixed-bytesN (≠32)
- Storage proof
- MCP exposure
- Adapter.SupportedTxTypes() promotion (Sprint 4c hardcoded allowlist 그대로)

**Sprint 4 시리즈 완료 후 이행**:
- bash CLI `gstable` hardcoding (M4)
- wbft / wemix `GenerateGenesis`/`GenerateToml` 실구현
