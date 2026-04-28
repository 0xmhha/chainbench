# Sprint 5c.2 — MCP Remaining 4 Tools + 2 tx_send Modes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Add the four remaining high-level MCP tools (`chainbench_contract_call`,
`chainbench_events_get`, `chainbench_tx_wait`, `chainbench_contract_deploy`) and
the two new `chainbench_tx_send` modes (`set_code`, `fee_delegation`) on top of
Sprint 5c.1's foundation. Sprint 종료 시 EVALUATION_CAPABILITY MCP column 의 ✅
cell 5+ 추가 → MCP coverage 25% → 60%+. 5c.3 (기존 38 tool reroute) 가 후속.

**Architecture:** 5c.1 패턴 그대로 (zod `.strict()` + cross-field check in
handler + `_*Handler` named exports + handler-direct registration). Pre-work
로 chain.ts 를 chain_read.ts (4 read tools) + chain_tx.ts (2 write tools) 로
split. `_buildTxSendWireArgs` 시그니처를 `{wireCommand, wireArgs}` 로 확장 —
fee_delegation mode 가 `node.tx_fee_delegation_send` 로 dispatch.

**Tech Stack:** TypeScript + zod + `@modelcontextprotocol/sdk`. 신규 wire 의존
없음 (chainbench-net 의 4d/4c 명령들 그대로 사용).

Spec: `docs/superpowers/specs/2026-04-28-sprint-5c-2-mcp-remaining-tools.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits prefix: `refactor(mcp):`, `feat(mcp):`, `test(mcp):`,
  `docs+chore(mcp):`.

## File Structure

**Create:**
- `mcp-server/src/tools/chain_read.ts`
- `mcp-server/src/tools/chain_tx.ts`
- `mcp-server/test/chain_read.test.ts`
- `mcp-server/test/chain_tx.test.ts`

**Modify:**
- `mcp-server/src/tools/chain.ts` — 내부를 두 새 파일로 이동, `registerChainTools`
  만 유지
- `mcp-server/test/chain.test.ts` — 삭제 (분할 후)
- `mcp-server/package.json` — version 0.4.0 → 0.5.0
- `mcp-server/src/index.ts` — McpServer version 0.4.0 → 0.5.0
- `docs/EVALUATION_CAPABILITY.md`
- `docs/VISION_AND_ROADMAP.md`
- `docs/NEXT_WORK.md`

---

## Task 0 — chain.ts split + 5c.1 minor carry-over

**Files:**
- Create: `mcp-server/src/tools/chain_read.ts` (account_state 만 우선 이동)
- Create: `mcp-server/src/tools/chain_tx.ts` (tx_send 만 우선 이동)
- Create: `mcp-server/test/chain_read.test.ts`
- Create: `mcp-server/test/chain_tx.test.ts`
- Modify: `mcp-server/src/tools/chain.ts` — `registerChainTools` 만 유지
- Delete: `mcp-server/test/chain.test.ts`

### Step 0.1: Move + standardize

`chain_read.ts` 에 이동:
- 공유 상수: `HEX_ADDRESS`, `HEX_STORAGE_KEY`, `FIELD`
- `AccountStateArgs` (named export)
- `_accountStateHandler` (named export)
- `registerChainReadTools(server)` (NEW; account_state 만 register)

`chain_tx.ts` 에 이동:
- 공유 상수: `HEX_DATA`, `SIGNER_ALIAS`, `MODE` (legacy/1559 만 — 5c.2 후속
  task 가 set_code/fee_delegation 추가)
- `TxSendArgs` (named export)
- `_buildTxSendWireArgs` (named export) — **시그니처 변경**:
  `{wireCommand: "node.tx_send", wireArgs} | {error}`. 모든 기존 분기는
  `wireCommand: "node.tx_send"` 반환.
- `_txSendHandler` (named export) — `built.wireCommand` 를 callWire 첫 인자로
- `registerChainTxTools(server)` (NEW; tx_send 만 register)

`chain.ts` 새 본체:
```typescript
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { registerChainReadTools } from "./chain_read.js";
import { registerChainTxTools } from "./chain_tx.js";

export function registerChainTools(server: McpServer): void {
  registerChainReadTools(server);
  registerChainTxTools(server);
}
```

`HEX_ADDRESS`, `HEX_DATA`, `SIGNER_ALIAS` 등 양쪽에서 쓰는 상수는 둘 다에서
독립 declare (의존성 그래프 단순화). 후속 task 에서 utils 로 빼는 건
선택적.

### Step 0.2: Standardize wire-args construction

`_buildTxSendWireArgs` 의 `Object.entries` 루프를 `_accountStateHandler` 와
같이 explicit per-field assignment 로 변경:

```typescript
const wireArgs: Record<string, unknown> = {
  network: args.network,
  signer: args.signer,
};
if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
if (args.to !== undefined) wireArgs.to = args.to;
if (args.value !== undefined) wireArgs.value = args.value;
if (args.data !== undefined) wireArgs.data = args.data;
if (args.gas !== undefined) wireArgs.gas = args.gas;
if (args.nonce !== undefined) wireArgs.nonce = args.nonce;
if (args.gas_price !== undefined) wireArgs.gas_price = args.gas_price;
if (args.max_fee_per_gas !== undefined) wireArgs.max_fee_per_gas = args.max_fee_per_gas;
if (args.max_priority_fee_per_gas !== undefined) wireArgs.max_priority_fee_per_gas = args.max_priority_fee_per_gas;
return { wireCommand: "node.tx_send", wireArgs };
```

5c.1 의 P3 tech debt row "wire-args 구성 패턴 차이" 를 흡수.

### Step 0.3: Move tests + add carry-over test

`chain_read.test.ts`:
- 5c.1 의 6개 account_state 테스트 그대로 이동 (이름 / assertion 동일)

`chain_tx.test.ts`:
- 5c.1 의 8개 tx_send 테스트 그대로 이동 (단, `_buildTxSendWireArgs` 의 반환
  값은 이제 `{wireCommand, wireArgs}` 이므로 `_Happy_Legacy` / `_Happy_1559`
  의 assertion 에서 `wireCommand === "node.tx_send"` 확인 추가)
- **새로 추가**: `_1559WithGasPrice_Rejected` (5c.1 final review minor #1)

### Step 0.4: Verify + commit

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/mcp-server
npm test          # 9 wire + 8 wireResult + 6 chain_read + 9 chain_tx (8 + 1 new) = 32
npx tsc --noEmit
npm run build
```

```bash
git add mcp-server/src/tools/ mcp-server/test/chain_read.test.ts \
        mcp-server/test/chain_tx.test.ts
git rm mcp-server/test/chain.test.ts
git commit -m "refactor(mcp): split chain.ts into chain_read + chain_tx, standardize args build

Pre-work for Sprint 5c.2 (4 remaining high-level tools + 2 new
tx_send modes). Splits chain.ts so each new tool lands in the
correct file from day one and chain.ts itself stays at the
registerChainTools-orchestrator level.

chain_read.ts hosts the read-only tools (account_state today;
contract_call / events_get / tx_wait land in this sprint).
chain_tx.ts hosts the signer-required tools (tx_send today;
contract_deploy lands in this sprint, plus the set_code and
fee_delegation tx_send modes).

_buildTxSendWireArgs now returns {wireCommand, wireArgs} so the
fee_delegation mode coming up in Task 6 can dispatch to
node.tx_fee_delegation_send instead of node.tx_send. Existing
modes return wireCommand: 'node.tx_send' unchanged.

Also standardizes wire-args construction inside _buildTxSendWireArgs
on the explicit per-field assignment pattern already used by
_accountStateHandler, closing the divergence row from 5c.1's tech
debt list. A new _1559WithGasPrice_Rejected test covers the
symmetric branch the 5c.1 review flagged."
```

---

## Task 1 — `chainbench_contract_call`

**Files:**
- Modify: `mcp-server/src/tools/chain_read.ts` — add ContractCallArgs +
  `_contractCallHandler` + register
- Modify: `mcp-server/test/chain_read.test.ts` — add 7 tests

### Step 1.1: 테스트 먼저 (RED)

`describe("contract_call", ...)` 블록에 7 tests (spec §6.1):
- `_Happy_Calldata` — 핸들러 black-box, mock 응답 + result_raw 포함
- `_Happy_ABI` — abi+method+args → wire args 변환
- `_BothCalldataAndABI_Rejected` — handler isError INVALID_ARGS
- `_NeitherCalldataNorABI_Rejected` — handler isError
- `_ABIWithoutMethod_Rejected` — handler isError
- `_BadAddress_RejectedAtBoundary` — `expect(() => parse(...)).toThrow()`
- `_StrictRejectsUnknownKeys` — `.strict()` 작동

### Step 1.2: 구현

```typescript
export const ContractCallArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  contract_address: z.string().regex(HEX_ADDRESS),
  calldata: z.string().regex(HEX_DATA).optional(),
  abi: z.string().optional(),
  method: z.string().optional(),
  args: z.array(z.unknown()).optional(),
  block_number: z.union([z.string(), z.number()]).optional(),
  from: z.string().regex(HEX_ADDRESS).optional(),
}).strict();

export async function _contractCallHandler(args: z.infer<typeof ContractCallArgs>): Promise<FormattedToolResponse> {
  // cross-field
  if (args.calldata && args.abi) {
    return errorResp("calldata and abi are mutually exclusive");
  }
  if (!args.calldata && !args.abi) {
    return errorResp("either calldata or (abi + method + args) is required");
  }
  if (args.abi && !args.method) {
    return errorResp("abi requires method");
  }
  // wire args (explicit assignment)
  const wireArgs: Record<string, unknown> = {
    network: args.network,
    contract_address: args.contract_address,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.calldata !== undefined) wireArgs.calldata = args.calldata;
  if (args.abi !== undefined) wireArgs.abi = args.abi;
  if (args.method !== undefined) wireArgs.method = args.method;
  if (args.args !== undefined) wireArgs.args = args.args;
  if (args.block_number !== undefined) wireArgs.block_number = args.block_number;
  if (args.from !== undefined) wireArgs.from = args.from;
  const result = await callWire("node.contract_call", wireArgs);
  return formatWireResult(result);
}
```

`errorResp()` 는 chain_read.ts 내부 헬퍼 (cross-field error 를 `isError:true`
+ "Error (INVALID_ARGS): ..." 텍스트로 매핑). 동일 패턴이 events_get 에서도 사용.

`registerChainReadTools` 에 `chainbench_contract_call` 추가. Description text:
"Read-only contract call (eth_call). Either provide raw `calldata` (0x-hex
4-byte selector + ABI-encoded args) or provide `abi` + `method` + `args` for
server-side encoding. Returns `result_raw` always; `result_decoded` when ABI
mode used."

### Step 1.3: 검증 + 커밋

```bash
cd mcp-server && npm test  # 32 + 7 = 39
cd mcp-server && npx tsc --noEmit
```

```bash
git add mcp-server/src/tools/chain_read.ts mcp-server/test/chain_read.test.ts
git commit -m "feat(mcp): chainbench_contract_call high-level tool

Wraps node.contract_call with mutually-exclusive calldata vs
abi+method+args modes. Cross-field validation in handler:
calldata XOR (abi + method + args), abi requires method. Address
hex-regex enforced at zod boundary. result_decoded only populated
when ABI mode used (chainbench-net unpacks return data via abiutil).

Tool description points the LLM at both modes — calldata for
already-encoded callers, ABI mode for callers that prefer
server-side encoding. Tuple / nested-array / fixed-bytesN(N!=32)
inputs require raw calldata fallback (same as Sprint 4d's
abiutil limit)."
```

---

## Task 2 — `chainbench_events_get`

**Files:**
- Modify: `mcp-server/src/tools/chain_read.ts`
- Modify: `mcp-server/test/chain_read.test.ts`

### Step 2.1: 테스트 먼저 + impl

7 tests (spec §6.1):
- `_Happy_NoFilters`
- `_Happy_WithAddressAndTopics`
- `_Happy_WithABIDecode`
- `_BadTopicHex_RejectedAtBoundary`
- `_ABIWithoutEvent_Rejected` — handler
- `_EventWithoutABI_Rejected` — handler
- `_StrictRejectsUnknownKeys`

```typescript
const HEX_TOPIC = /^0x[a-fA-F0-9]{64}$/;

export const EventsGetArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  address: z.string().regex(HEX_ADDRESS).optional(),
  from_block: z.union([z.string(), z.number()]).optional(),
  to_block: z.union([z.string(), z.number()]).optional(),
  topics: z.array(z.union([
    z.string().regex(HEX_TOPIC),
    z.array(z.string().regex(HEX_TOPIC)),
    z.null(),
  ])).optional(),
  abi: z.string().optional(),
  event: z.string().optional(),
}).strict();

export async function _eventsGetHandler(args): Promise<FormattedToolResponse> {
  if (args.abi && !args.event) return errorResp("abi requires event");
  if (args.event && !args.abi) return errorResp("event requires abi");
  // wire args explicit assignment
  ...
  const result = await callWire("node.events_get", wireArgs);
  return formatWireResult(result);
}
```

Description: "Fetch event logs (eth_getLogs) with optional filtering. When
`abi` + `event` provided, each log's topics + data are decoded server-side and
included as `decoded` field on each log entry."

### Step 2.2: 검증 + 커밋

```bash
cd mcp-server && npm test  # 39 + 7 = 46
```

```bash
git commit -m "feat(mcp): chainbench_events_get high-level tool"
```

---

## Task 3 — `chainbench_tx_wait`

**Files:**
- Modify: `mcp-server/src/tools/chain_read.ts`
- Modify: `mcp-server/test/chain_read.test.ts`

### Step 3.1: 테스트 + impl

6 tests (spec §6.1):
- `_Happy_DefaultTimeout`
- `_Happy_CustomTimeout`
- `_BadTxHash_RejectedAtBoundary`
- `_NegativeTimeout_RejectedAtBoundary`
- `_StrictRejectsUnknownKeys`
- `_WireFailure_PassedThrough`

```typescript
const HEX_TX_HASH = /^0x[a-fA-F0-9]{64}$/;

export const TxWaitArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  tx_hash: z.string().regex(HEX_TX_HASH),
  timeout_ms: z.number().int().positive().optional(),
}).strict();

export async function _txWaitHandler(args): Promise<FormattedToolResponse> {
  // wire helper timeout = caller's timeout_ms + 5000ms (poll grace), or
  // 35000ms if caller didn't set timeout_ms (chainbench-net default ~30000 + grace).
  const timeoutMs = (args.timeout_ms ?? 30000) + 5000;
  const wireArgs: Record<string, unknown> = {
    network: args.network,
    tx_hash: args.tx_hash,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.timeout_ms !== undefined) wireArgs.timeout_ms = args.timeout_ms;
  const result = await callWire("node.tx_wait", wireArgs, { timeoutMs });
  return formatWireResult(result);
}
```

Description: "Poll for a transaction receipt with exponential backoff. Returns
the receipt when found (status, block_number, gas_used, contract_address etc.)
or `{status: 'pending', tx_hash}` if the tx is still unconfirmed at timeout."

### Step 3.2: 검증 + 커밋

```bash
cd mcp-server && npm test  # 46 + 6 = 52
```

```bash
git commit -m "feat(mcp): chainbench_tx_wait high-level tool"
```

---

## Task 4 — `chainbench_contract_deploy`

**Files:**
- Modify: `mcp-server/src/tools/chain_tx.ts`
- Modify: `mcp-server/test/chain_tx.test.ts`

### Step 4.1: 테스트 + impl

6 tests (spec §6.2):
- `_Happy_LegacyBytecode`
- `_Happy_1559WithABI`
- `_BadBytecode_RejectedAtBoundary`
- `_LegacyWithMaxFee_Rejected`
- `_1559WithoutMaxFee_Rejected`
- `_StrictRejectsUnknownKeys`

```typescript
const DEPLOY_MODE = z.enum(["legacy", "1559"]);

export const ContractDeployArgs = z.object({
  network: z.string().min(1),
  node_id: z.string().optional(),
  signer: z.string().regex(SIGNER_ALIAS),
  mode: DEPLOY_MODE,
  bytecode: z.string().regex(HEX_DATA),
  abi: z.string().optional(),
  constructor_args: z.array(z.unknown()).optional(),
  value: z.string().optional(),
  gas: z.union([z.string(), z.number()]).optional(),
  nonce: z.union([z.string(), z.number()]).optional(),
  gas_price: z.string().optional(),
  max_fee_per_gas: z.string().optional(),
  max_priority_fee_per_gas: z.string().optional(),
}).strict();

export function _buildContractDeployWireArgs(args): {wireArgs} | {error} {
  // mode-fee 배타 (tx_send 와 동일 로직 — _buildTxSendWireArgs 와 별개 함수
  // 로 두고 두 함수 모두 같은 패턴 따라가는지 review 가 검증)
  ...
}

export async function _contractDeployHandler(args): Promise<FormattedToolResponse> {
  const built = _buildContractDeployWireArgs(args);
  if ("error" in built) return errorResp(built.error);
  const result = await callWire("node.contract_deploy", built.wireArgs);
  return formatWireResult(result);
}
```

### Step 4.2: 검증 + 커밋

```bash
cd mcp-server && npm test  # 52 + 6 = 58
```

```bash
git commit -m "feat(mcp): chainbench_contract_deploy high-level tool"
```

---

## Task 5 — `chainbench_tx_send` mode `set_code`

**Files:**
- Modify: `mcp-server/src/tools/chain_tx.ts`
- Modify: `mcp-server/test/chain_tx.test.ts`

### Step 5.1: 테스트 + impl

`MODE` enum 확장: `legacy | 1559 | set_code | fee_delegation` (둘 다 동시
landing — schema 만 미리 확장해두고 fee_delegation 의 cross-field 는 Task 6
에서. 본 task 의 schema 는 set_code 만 활성화).

이번 task 는 set_code 활성화. 추가 필드:
```typescript
const HEX_HEX = /^0x[a-fA-F0-9]+$/; // generic hex (chain_id, nonce in auth list)

const AuthorizationEntry = z.object({
  chain_id: z.string().regex(HEX_HEX),
  address: z.string().regex(HEX_ADDRESS),
  nonce: z.string().regex(HEX_HEX),
  signer: z.string().regex(SIGNER_ALIAS),
}).strict();

// TxSendArgs 에 추가:
authorization_list: z.array(AuthorizationEntry).optional(),
fee_payer: z.string().regex(SIGNER_ALIAS).optional(),  // Task 6 가 활성화
```

`_buildTxSendWireArgs` 의 set_code 분기:
```typescript
if (args.mode === "set_code") {
  if (!args.authorization_list || args.authorization_list.length === 0) {
    return { error: "mode 'set_code' requires non-empty authorization_list" };
  }
  if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
    return { error: "mode 'set_code' requires both max_fee_per_gas and max_priority_fee_per_gas (1559 envelope)" };
  }
  if (args.gas_price) return { error: "mode 'set_code' rejects gas_price" };
  if (args.fee_payer) return { error: "mode 'set_code' rejects fee_payer (use mode 'fee_delegation' instead)" };
  // wire args: 1559 fields + authorization_list passthrough
  ...
  return { wireCommand: "node.tx_send", wireArgs };
}
```

또한 기존 legacy/1559 분기에 `if (args.authorization_list)` 거절 추가. 6 tests
(spec §6.2):
- `_Happy_SetCode`
- `_SetCodeWithoutAuthList_Rejected`
- `_SetCodeWithoutMaxFee_Rejected`
- `_SetCodeWithGasPrice_Rejected`
- `_AuthListInLegacyMode_Rejected`
- `_BadAuthEntry_RejectedAtBoundary` — entry 의 잘못된 hex → zod throw

### Step 5.2: 검증 + 커밋

```bash
cd mcp-server && npm test  # 58 + 6 = 64
```

```bash
git commit -m "feat(mcp): chainbench_tx_send mode set_code (EIP-7702)"
```

---

## Task 6 — `chainbench_tx_send` mode `fee_delegation`

**Files:**
- Modify: `mcp-server/src/tools/chain_tx.ts`
- Modify: `mcp-server/test/chain_tx.test.ts`

### Step 6.1: 테스트 + impl

`fee_payer` 필드는 Task 5 에서 schema 에 이미 추가됨. 본 task 는 cross-field
+ wire dispatch 활성화.

`_buildTxSendWireArgs` 의 fee_delegation 분기:
```typescript
if (args.mode === "fee_delegation") {
  if (!args.fee_payer) return { error: "mode 'fee_delegation' requires fee_payer alias" };
  if (!args.to) return { error: "mode 'fee_delegation' requires to (no contract creation)" };
  if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
    return { error: "mode 'fee_delegation' requires both max_fee_per_gas and max_priority_fee_per_gas" };
  }
  if (args.gas === undefined) return { error: "mode 'fee_delegation' requires gas (no auto-fill)" };
  if (args.nonce === undefined) return { error: "mode 'fee_delegation' requires nonce (no auto-fill)" };
  if (args.gas_price) return { error: "mode 'fee_delegation' rejects gas_price" };
  if (args.authorization_list) return { error: "mode 'fee_delegation' rejects authorization_list (no set_code combo)" };
  // wire args: sender + fee_payer + 1559 + gas + nonce + to/value/data
  const wireArgs = { ... };
  return { wireCommand: "node.tx_fee_delegation_send", wireArgs };
}
```

기존 legacy/1559/set_code 분기에 `if (args.fee_payer)` 거절 추가.

7 tests (spec §6.2):
- `_Happy_FeeDelegation` — assertion 에 `wireCommand === "node.tx_fee_delegation_send"`
- `_FeeDelegationWithoutFeePayer_Rejected`
- `_FeeDelegationWithoutTo_Rejected`
- `_FeeDelegationWithGasPrice_Rejected`
- `_FeeDelegationWithoutGasOrNonce_Rejected`
- `_FeeDelegationWithAuthList_Rejected`
- `_FeePayerInLegacyMode_Rejected`

### Step 6.2: 검증 + 커밋

```bash
cd mcp-server && npm test  # 64 + 7 = 71
```

```bash
git commit -m "feat(mcp): chainbench_tx_send mode fee_delegation (go-stablenet 0x16)

Last mode of chainbench_tx_send. Unlike legacy/1559/set_code
(which all dispatch to node.tx_send), fee_delegation dispatches
to node.tx_fee_delegation_send via the wireCommand field that
Task 0 introduced on _buildTxSendWireArgs's return shape.

Cross-field validation enforces fee_delegation's stablenet-specific
constraints: fee_payer alias is required, to is required (no
contract creation), 1559 fee fields are mandatory, gas + nonce
must be explicit (chainbench-net does not auto-fill those for
this tx type), gas_price is rejected, and authorization_list is
rejected (set_code combo unsupported).

Other modes also gain a guard: if fee_payer is set in any
non-fee_delegation mode, the request is rejected at the boundary
to keep the discriminator clean.

Note: this surface assumes the underlying network is go-stablenet
or another adapter on the chainbench-net allowlist for tx type
0x16. The MCP layer does NOT pre-validate adapter compatibility;
chainbench-net surfaces NOT_SUPPORTED if the chain's adapter
declines fee delegation, and the formatter passes that through
as an isError response."
```

---

## Task 7 — Docs + version bump 0.5.0

**Files:**
- Modify: `mcp-server/package.json` — 0.4.0 → 0.5.0
- Modify: `mcp-server/src/index.ts` — McpServer version 0.5.0
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`

### Step 7.1: EVALUATION_CAPABILITY.md

§2 (Tx 매트릭스) MCP column flips:
- `Fee Delegation (0x16)`: ❌ → ✅ Sprint 5c.2 (chainbench_tx_send mode='fee_delegation')
- `EIP-7702 SetCode (0x4)`: ❌ → ✅ Sprint 5c.2 (chainbench_tx_send mode='set_code')
- `Contract deploy`: ❌ → ✅ Sprint 5c.2 (chainbench_contract_deploy)
- `Contract call (eth_call+ABI)`: ❌ → ✅ Sprint 5c.2 (chainbench_contract_call)
- `Event log fetch (eth_getLogs)`: ❌ → ✅ Sprint 5c.2 (chainbench_events_get)
- `Event log decode`: ❌ → ✅ Sprint 5c.2 (chainbench_events_get with abi+event)
- `Receipt polling (status+logs)`: ❌ → ✅ Sprint 5c.2 (chainbench_tx_wait)

§4 (검증 매트릭스) MCP column:
- `Tx receipt status`: ❌ → ✅ Sprint 5c.2
- `Tx receipt event log`: ❌ → ✅ Sprint 5c.2

§5 (Sprint 별 도달 목표) — 5c.2 row 추가 (✅ 완료 2026-04-28).

§6: MCP coverage 25% → ~60% (8 cell 추가).

### Step 7.2: VISION_AND_ROADMAP.md

§6 Sprint 5 분할 갱신:
- `Sprint 5c.1` 완료 그대로
- `**Sprint 5c.2**` 완료 박스 + 본 sprint 가 추가한 4 tool + 2 mode 명시
- `Sprint 5c.3` (기존 38 tool reroute) 가 다음 P1
- 5a / 5b / 5d 그대로

§5.17.7 갱신 — 5c.2 단락 추가.

### Step 7.3: NEXT_WORK.md

§1 헤더 최종 업데이트.
§2.1 timeline 표 — 5c.2 row 추가.
§2.3 테스트 매트릭스 — vitest 71 tests 갱신.
§3 P1 narrative — 다음 P1 = 5c.3.
§3.5 (또는 최근 완료 표) — 5c.2 entry 추가.
§3 P3 tech debt 표 — 다음 row 추가:
- `5c.2: chain_tx.ts size` — 현재 약 ? lines, 5c.3 reroute 가 lifecycle/test
  tool 들을 chain_tx.ts 와 직교 위치로 추가 시 재검토
- `5c.2: errorResp 헬퍼 chain_read.ts ↔ chain_tx.ts 중복` — 두 파일에 동일
  헬퍼 — 5c.3 에서 utils/mcpResp.ts 로 추출 검토
§4.6 file size 표 — 신규 두 파일 사이즈 명시.

### Step 7.4: 검증 + 커밋

```bash
cd mcp-server && npm test                                         # 71 / 71
cd mcp-server && npx tsc --noEmit && npm run build
go -C network test ./... -count=1 -timeout=60s                    # regression
bash tests/unit/run.sh                                             # regression
```

```bash
git add mcp-server/package.json mcp-server/src/index.ts \
        docs/EVALUATION_CAPABILITY.md docs/VISION_AND_ROADMAP.md docs/NEXT_WORK.md
git commit -m "docs+chore(sprint-5c-2): roadmap + capability matrix + version 0.5.0

EVALUATION_CAPABILITY MCP column gains 8 ✅ cells:
  §2: Fee Delegation (0x16), EIP-7702 SetCode, Contract deploy,
      Contract call, Event log fetch, Event log decode, Receipt
      polling — Sprint 5c.2
  §4: Tx receipt status, Tx receipt event log — Sprint 5c.2
  §6: MCP coverage 25% -> ~60%

VISION_AND_ROADMAP Sprint 5c.2 marked complete; next P1 is 5c.3
(reroute existing 38 tools through the wire helper). 5a/5b/5d
unchanged.

NEXT_WORK timeline gains the 5c.2 row; tech debt picks up the
chain_tx.ts size watch and the errorResp helper duplication
between chain_read.ts and chain_tx.ts (utils/mcpResp.ts
candidate at 5c.3 time).

mcp-server version 0.4.0 -> 0.5.0 (4 new high-level tools, 2
new tx_send modes)."
```

---

## Final report (after all tasks)

Commit chain (8 expected: split + 4 tool features + 2 mode features + docs +
spec/plan = 9 with the spec/plan commit). vitest 31 → 71 (40 new). EVALUATION
MCP cells 25% → 60%. Confirmed deferrals: 5c.3 (reroute), 5a/5b/5d
(capability/SSH/hybrid).

가능한 follow-up (P3 후보):
- chain_tx.ts size 모니터링 (~600 lines 추정 — 권장치 200-400 초과 가능성)
- errorResp 헬퍼 두 파일 중복 — 5c.3 시 utils/mcpResp.ts 추출
- mcp-server vitest real-binary 통합 테스트 — 5c.3 시 reroute 연결 테스트와 함께
