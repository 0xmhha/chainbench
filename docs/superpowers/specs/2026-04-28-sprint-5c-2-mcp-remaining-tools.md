# Sprint 5c.2 — MCP Remaining 4 High-Level Tools + 2 tx_send Modes Design Spec

> 2026-04-28 · VISION §6 Sprint 5 — 5c second pass
> Scope: Sprint 5c.1 가 깐 wire helper / transformer / chain.ts 위에서, 남은
> 4 high-level tool (`chainbench_contract_deploy` / `_call` / `_events_get` /
> `_tx_wait`) + `chainbench_tx_send` mode 확장 (`set_code`, `fee_delegation`)
> 을 추가. 본 sprint 종료 시 `EVALUATION_CAPABILITY.md` MCP column 의 ✅ cell
> 5+ 추가 → MCP coverage 25% → 60%+. 5c.3 (기존 38 tool reroute) 가 후속.

## 1. Goal

Sprint 5c.1 이 검증한 패턴 (wire helper + zod `.strict()` + cross-field
검증 + `_*Handler` 명명 + handler-direct registration) 을 그대로 적용하여
4개 신규 high-level tool + 2개 모드를 추가한다. 동시에 기존 chain.ts 를
read / tx 두 파일로 split 하여 5c.3 reroute 작업에 들어가기 전 size 권장치
이내로 정리한다.

본 sprint 의 끝에서 coding agent 는 MCP 만으로 (a) account state read, (b)
event log fetch + decode, (c) contract eth_call (calldata or ABI), (d)
contract creation, (e) signed tx (4 모드 — legacy / 1559 / EIP-7702 SetCode /
go-stablenet 0x16 fee delegation), (f) receipt polling 까지 가능.

## 2. Non-Goals (Deferred)

- **기존 38 tool reroute** — `chainbench_init/start/stop/...` 등은 본 sprint
  무손상. 5c.3 에서 점진 이관.
- **WS subscription / chain log streaming** — wire helper 의 단일-result 호출
  계약 그대로. subscription.open 은 후속 sprint.
- **Adapter `SupportedTxTypes()` interface promotion** — 4c 의 hardcoded
  `feeDelegationAllowedChains` 그대로. 5번째 chain-specific tx 타입 도입 시
  promote.
- **ABI tuple / nested-array / fixed-bytesN(N≠32)** — 4d 의 `abiutil` 한계
  유지. 사용자가 raw `calldata` / `bytecode` 로 우회.
- **Hybrid network / SSH driver / capability gate** — 5a/5b/5d.
- **`chainbench_tx_send` 의 mode 5 이상** — `legacy/1559/set_code/fee_delegation`
  4-way 가 본 sprint 의 final.

## 3. Security Contract

Sprint 4 / 4b / 4c / 4d / 5c.1 의 redaction boundary 그대로 유지. 본 sprint
가 추가하는 surface 의 security 특이점:

- **`chainbench_tx_send` mode = `fee_delegation`** 은 두 alias 를 받음
  (`signer` = 송신자, `fee_payer` = 가스 대납자). 둘 다 alias **이름만**
  받고 raw key material 은 schema 거절. `CHAINBENCH_SIGNER_<sender>_KEY` /
  `CHAINBENCH_SIGNER_<fee_payer>_KEY` (또는 keystore 변형) 모두 host env 에
  존재해야 chainbench-net 이 signer.Load 가능. wire helper 의 `process.env`
  forwarding 으로 자동 통과.
- **`chainbench_tx_send` mode = `set_code`** 의 `authorization_list` 는
  추가 signer alias 들을 포함 (각 entry 의 `signer` 필드). 이들도 alias
  이름만 받음. raw key 의 schema 노출 금지.
- 각 mode 의 dispatch 결과 wire 명령:
  - `legacy` / `1559` / `set_code` → `node.tx_send`
  - `fee_delegation` → `node.tx_fee_delegation_send`

  TS layer 가 mode 보고 wire command 를 선택하므로, mode 가 잘못된 wire 로
  routing 되지 않음을 단위 테스트로 검증.

## 4. User-Facing Surface

### 4.1 chain.ts split (Task 0)

`mcp-server/src/tools/chain.ts` 를 두 파일로:
- `mcp-server/src/tools/chain_read.ts` — `chainbench_account_state`,
  `chainbench_contract_call`, `chainbench_events_get`, `chainbench_tx_wait`.
  signer 불필요한 read-only tool 들.
- `mcp-server/src/tools/chain_tx.ts` — `chainbench_tx_send` (4 modes),
  `chainbench_contract_deploy`. signer 필요한 write tool 들.
- `mcp-server/src/tools/chain.ts` — `registerChainTools(server)` 만 남기고
  내부에서 read + tx 둘 다 register. external import 표면 유지 (`index.ts`
  변경 없음).

기존 `_accountStateHandler`, `_txSendHandler`, `_buildTxSendWireArgs`,
`AccountStateArgs`, `TxSendArgs`, `HEX_ADDRESS`, `HEX_DATA`, `HEX_STORAGE_KEY`,
`SIGNER_ALIAS`, `MODE`, `FIELD` 모두 새 파일로 이동. test 파일 (`chain.test.ts`)
도 split — `chain_read.test.ts` + `chain_tx.test.ts`.

이 split 은 reviewer 권장 (5c.1 final review §Architectural Soundness 4번)
이며, 4 새 tool 추가 후 file size 가 권장치 (200-400) 를 명확히 초과하지
않도록 사전 분리.

**또한 같은 commit 으로 5c.1 의 minor 두 건 흡수**:
- `_buildTxSendWireArgs` 의 wire-args 구성 패턴을 `_accountStateHandler` 와
  같이 explicit per-field assignment 로 통일 (Object.entries 루프 제거).
- 누락된 테스트 추가: `_1559WithGasPrice_Rejected` (mode=1559 + gas_price
  같이 보내면 거절).

### 4.2 `chainbench_contract_call` (Task 1, read)

```typescript
const ContractCallArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  contract_address: z.string().regex(HEX_ADDRESS),
  // mutually exclusive: calldata XOR (abi + method + args)
  calldata: z.string().regex(HEX_DATA).optional(),
  abi: z.string().optional(),
  method: z.string().optional(),
  args: z.array(z.unknown()).optional(),
  block_number: z.union([z.string(), z.number()]).optional(),
  from: z.string().regex(HEX_ADDRESS).optional(),
}).strict();
```

**Cross-field check** (handler):
- `calldata` 와 `abi`+`method` 는 mutually exclusive
- `calldata` / `abi` 둘 다 미제공 → INVALID_ARGS
- `abi` 제공 시 `method` 도 필수 (`args` 는 optional, 기본 빈 배열)

Wire command: `node.contract_call`. Result: `{result_raw, result_decoded?}`.

### 4.3 `chainbench_events_get` (Task 2, read)

```typescript
const EventsGetArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  address: z.string().regex(HEX_ADDRESS).optional(),
  from_block: z.union([z.string(), z.number()]).optional(),
  to_block: z.union([z.string(), z.number()]).optional(),
  // each topic: string (single hash) | string[] (OR-set) | null (wildcard at position)
  topics: z.array(z.union([
    z.string().regex(HEX_TOPIC),
    z.array(z.string().regex(HEX_TOPIC)),
    z.null(),
  ])).optional(),
  abi: z.string().optional(),
  event: z.string().optional(),
}).strict();
```

`HEX_TOPIC` = `^0x[a-fA-F0-9]{64}$` (32-byte hash, strict).

**Cross-field**: `abi` 만 있고 `event` 없으면 INVALID_ARGS (decode 의도가
불명). `event` 만 있고 `abi` 없으면 INVALID_ARGS.

Wire command: `node.events_get`. Result: `{logs: [...]}`.

### 4.4 `chainbench_tx_wait` (Task 3, read — receipt polling)

```typescript
const TxWaitArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  tx_hash: z.string().regex(HEX_TX_HASH),
  timeout_ms: z.number().int().positive().optional(),
}).strict();
```

`HEX_TX_HASH` = `^0x[a-fA-F0-9]{64}$` (32-byte hex).

`timeout_ms` 미지정 시 chainbench-net 의 default (현재 60000ms — chainbench-net
`defaultTxWaitMs`, `handlers_node_tx.go:583`) 사용. wire helper 의 호출 timeout
은 `timeout_ms + 5000ms` (poll grace) 로 자동 설정 — caller 가 명시한 timeout
보다 wire timeout 이 먼저 발동하지 않도록. MCP 핸들러는 caller 가 `timeout_ms`
를 생략한 경우 동일한 60000ms fallback 을 적용해 wire timeout 이 65000ms 가
되도록 한다 (서버 측 default 와 정합).

Wire command: `node.tx_wait`. Result: receipt JSON (status / block_number /
gas_used / logs_count / contract_address / effective_gas_price 등) 또는
pending 상태 (timeout 전 미확정).

### 4.5 `chainbench_contract_deploy` (Task 4, write)

```typescript
const ContractDeployArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  signer: z.string().regex(SIGNER_ALIAS),
  mode: z.enum(["legacy", "1559"]),  // SetCode/fee_delegation 은 deploy 와 의미 없음
  bytecode: z.string().regex(HEX_DATA),
  abi: z.string().optional(),
  constructor_args: z.array(z.unknown()).optional(),
  value: z.string().optional(),
  gas: z.union([z.string(), z.number()]).optional(),
  nonce: z.union([z.string(), z.number()]).optional(),
  // legacy-only:
  gas_price: z.string().optional(),
  // 1559-only:
  max_fee_per_gas: z.string().optional(),
  max_priority_fee_per_gas: z.string().optional(),
}).strict();
```

**Cross-field**: `tx_send` 와 동일한 mode-fee 배타. `abi` 제공 시
`constructor_args` 는 optional (미제공이면 빈 배열로 처리).

Wire command: `node.contract_deploy`. Result: `{tx_hash, contract_address}`.
Caller 는 `chainbench_tx_wait` 으로 receipt 폴링 후 실제 배포 확인.

### 4.6 `chainbench_tx_send` mode `set_code` (Task 5, write)

기존 `TxSendArgs` 의 mode enum 을 `legacy | 1559 | set_code | fee_delegation`
으로 확장. `set_code` 모드 추가 schema 필드:

```typescript
const AuthorizationEntry = z.object({
  chain_id: z.string().regex(HEX_DATA),  // hex
  address: z.string().regex(HEX_ADDRESS),
  nonce: z.string().regex(HEX_DATA),     // hex
  signer: z.string().regex(SIGNER_ALIAS),
}).strict();

// TxSendArgs 에 추가:
authorization_list: z.array(AuthorizationEntry).optional(),
```

**Cross-field for `mode: set_code`**:
- `authorization_list` 필수 (1+ entry)
- 1559 fee fields 둘 다 필수 (set_code 는 1559 envelope 위에서만 동작)
- `gas_price` 거절
- 다른 mode 에서 `authorization_list` 제공 시 INVALID_ARGS

Wire command: `node.tx_send` (chainbench-net 이 authorization_list 보고 자동
SetCodeTx 로 라우팅). Result: `{tx_hash}` (Sprint 4c 와 동일).

### 4.7 `chainbench_tx_send` mode `fee_delegation` (Task 6, write)

새 mode. **Wire command 가 다름**: `node.tx_fee_delegation_send`.

추가 schema 필드:

```typescript
// TxSendArgs 에 추가:
fee_payer: z.string().regex(SIGNER_ALIAS).optional(),
```

**Cross-field for `mode: fee_delegation`**:
- `fee_payer` 필수
- `to` 필수 (fee delegation 은 contract creation 미지원)
- 1559 fee fields 둘 다 필수
- `gas`, `nonce` 둘 다 필수 (chainbench-net 이 fee delegation 은 auto-fill 안 함)
- `gas_price` 거절
- `authorization_list` 거절 (fee delegation + set_code 조합 미지원)
- 다른 mode 에서 `fee_payer` 제공 시 INVALID_ARGS

Wire command 선택 로직 (`_buildTxSendWireArgs` 시그니처 변경):

```typescript
export function _buildTxSendWireArgs(args: TxSendArgsT):
  | { wireCommand: "node.tx_send" | "node.tx_fee_delegation_send"; wireArgs: Record<string, unknown> }
  | { error: string };
```

Handler (`_txSendHandler`) 가 `built.wireCommand` 를 callWire 첫 인자로 사용.

## 5. Package Layout

**Create:**
- `mcp-server/src/tools/chain_read.ts` — 4 read tool registration + handlers
- `mcp-server/src/tools/chain_tx.ts` — 2 write tool registration + handlers
- `mcp-server/test/chain_read.test.ts` — 분할 후 read tool 테스트
- `mcp-server/test/chain_tx.test.ts` — 분할 후 tx tool 테스트

**Modify:**
- `mcp-server/src/tools/chain.ts` — 내부 구현 모두 두 파일로 이동, 본 파일은
  `registerChainTools(server)` 만 유지 (read + tx 두 register 호출)
- `mcp-server/test/chain.test.ts` — Task 0 종료 시 삭제 (두 split test 파일이 대체)
- `mcp-server/package.json` — 0.4.0 → 0.5.0
- `mcp-server/src/index.ts` — McpServer version 0.4.0 → 0.5.0
- `docs/EVALUATION_CAPABILITY.md` — MCP column cell 5+ flip + §6 coverage 갱신
- `docs/VISION_AND_ROADMAP.md` — Sprint 5c.2 박스 ✅ + §5.17.7 갱신
- `docs/NEXT_WORK.md` — §1/§2.1/§3/§4.6 갱신

## 6. Tests

총 vitest 추가 테스트 ~25-30개 (5c.1 종료 = 31 → 5c.2 종료 ~60+).

### 6.1 chain_read.test.ts (Task 1+2+3)

**Account state** — 5c.1 의 6 테스트 그대로 이전.

**Contract call**:
- `_Happy_Calldata` — calldata 만 → wire args 에 calldata, no abi/method
- `_Happy_ABI` — abi+method+args → wire args 변환 (abi pass-through string)
- `_BothCalldataAndABI_Rejected` — handler isError INVALID_ARGS
- `_NeitherCalldataNorABI_Rejected` — handler isError
- `_ABIWithoutMethod_Rejected`
- `_BadAddress_RejectedAtBoundary` — zod throw
- `_StrictRejectsUnknownKeys` — `.strict()` 작동

**Events get**:
- `_Happy_NoFilters` — minimal args (network 만)
- `_Happy_WithAddressAndTopics`
- `_Happy_WithABIDecode`
- `_BadTopicHex_RejectedAtBoundary`
- `_ABIWithoutEvent_Rejected` — handler
- `_EventWithoutABI_Rejected`
- `_StrictRejectsUnknownKeys`

**Tx wait**:
- `_Happy_DefaultTimeout`
- `_Happy_CustomTimeout`
- `_BadTxHash_RejectedAtBoundary`
- `_NegativeTimeout_RejectedAtBoundary` — zod `.positive()`
- `_StrictRejectsUnknownKeys`
- `_WireFailure_PassedThrough`

### 6.2 chain_tx.test.ts (Task 4+5+6)

**Tx send** — 5c.1 의 8 테스트 그대로 이전 + standardize 패턴 적용 후.

**Tx send mode set_code**:
- `_Happy_SetCode` — `_buildTxSendWireArgs({mode:"set_code", max_fee_*, authorization_list:[...]})` → `{wireCommand: "node.tx_send", wireArgs: {... + authorization_list}}`
- `_SetCodeWithoutAuthList_Rejected` — error
- `_SetCodeWithoutMaxFee_Rejected`
- `_SetCodeWithGasPrice_Rejected`
- `_AuthListInLegacyMode_Rejected` — non-set_code mode 에서 authorization_list 제공 시 거절
- `_BadAuthEntry_RejectedAtBoundary` — entry 의 잘못된 hex / 누락 필드 → zod throw

**Tx send mode fee_delegation**:
- `_Happy_FeeDelegation` — `{wireCommand: "node.tx_fee_delegation_send", wireArgs: {sender + fee_payer + 1559 + gas + nonce}}`
- `_FeeDelegationWithoutFeePayer_Rejected`
- `_FeeDelegationWithoutTo_Rejected`
- `_FeeDelegationWithGasPrice_Rejected`
- `_FeeDelegationWithoutGasOrNonce_Rejected`
- `_FeeDelegationWithAuthList_Rejected`
- `_FeePayerInLegacyMode_Rejected` — non-fee_delegation mode 에서 fee_payer 제공 시 거절

**Contract deploy**:
- `_Happy_LegacyBytecode`
- `_Happy_1559WithABI` — abi+constructor_args → wire 에 그대로 통과 (chainbench-net 이 인코딩)
- `_BadBytecode_RejectedAtBoundary`
- `_LegacyWithMaxFee_Rejected`
- `_1559WithoutMaxFee_Rejected`
- `_StrictRejectsUnknownKeys`

### 6.3 dispatcher / integration

`registerChainTools` 가 6개 tool 모두 register 하는지 확인하는 단일
integration 테스트는 추가 없음 — 각 tool 의 handler-direct 테스트가 이미
register 경로를 검증.

## 7. Schema

`network/schema/command.json` 변경 없음. Sprint 4c/4d 가 추가한 enum 그대로
사용.

## 8. Error Classification

5c.1 매트릭스 그대로:

| 단계 | 코드 | 사용처 |
|---|---|---|
| Zod `.strict()` | (zod throw) | 잘못된 hex / 잘못된 enum / unknown key / 잘못된 alias |
| Tool wrapper handler | INVALID_ARGS | mode + 필드 조합 cross-field 위반 |
| chainbench-net | INVALID_ARGS / UPSTREAM_ERROR / NOT_SUPPORTED / INTERNAL | Go boundary |

## 9. Out-of-Scope Reminders

**5c.3 으로 이행**:
- 기존 38 tool 의 wire 경유 reroute (`runChainbench` → `callWire`)
- `lib/cmd_init/start/stop/...` 의 Go 포팅 (network.init wire handler 등)
- mcp-server vitest 통합 테스트 (실 chainbench-net spawn)

**Sprint 5 series 외**:
- WS subscription / chain log streaming
- Capability gate (5a) / SSH driver (5b) / Hybrid 예제 (5d)
- Adapter.SupportedTxTypes() promotion

## 10. Migration / Backwards Compat

- 기존 `chainbench_account_state` / `chainbench_tx_send` (legacy/1559) 의
  invocation 형식 변경 없음. mode enum 만 확장 — `legacy/1559` 호출은 그대로.
- chain.ts split 은 외부 import 변경 없음 (`registerChainTools` 그대로 export).
- 패키지 버전 0.4.0 → 0.5.0 (4 신규 tool + 2 신규 mode 노출이므로 minor bump).
- `_buildTxSendWireArgs` 시그니처 변경 (`{wireArgs}` → `{wireCommand, wireArgs}`)
  은 named-export internal API — 외부 consumer 없음 (테스트 만이 사용).
