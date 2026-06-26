# Refactoring Plan — Clean Code & SSOT

> 작성일: 2026-06-26
> 최종 업데이트: 2026-06-26 (P0 + P1-1 + 사전 버그 2건 완료 — PR #3 머지, commit `63f1d43`)
> 관점: **Clean Code** (함수/파일 크기, 중복, 매직값, 네이밍) + **SSOT** (Single Source of Truth)
> 범위: `lib/` (bash) · `network/` (Go) · `mcp-server/src/` (TypeScript)
> 성격: **리팩토링 추적 문서** — §1/§2 는 발견 목록(증거 포함), §3 은 우선순위+진행상태, §6 은 남은 작업.
> 관련: `docs/NEXT_WORK.md` §3 P3 (기존 tech-debt 추적) · `docs/HARDCODING_AUDIT.md` · `docs/ADAPTER_CONTRACT.md` · `~/.claude/rules/coding-style.md`

---

## 0. 먼저 — 무엇을 리팩토링하지 "않는가" (migration-aware)

이 프로젝트는 **bash → Go network abstraction** 마이그레이션 중이며 (`VISION_AND_ROADMAP.md` §5.12 M0–M9), 일부 중복은 **의도적·과도기적**이다. 아래는 SSOT 관점에서 중복으로 보이지만 **지금 통합하면 안 되는** 항목이다:

- **bash adapter (`lib/adapters/*.sh`) vs Go adapter (`network/internal/adapters/`)** — 동일 genesis/TOML 로직의 이중 구현. M4까지 의도적 공존 (ADAPTER_CONTRACT §4). **통합 대상 아님.**
- **MCP의 `runChainbench`(bash spawn) vs `callWire`(Go 바이너리)** — reroute가 5/38(~13%)만 진행. 두 경로 공존은 로드맵대로다. **경로 자체는 유지**, 단 그 위의 *헬퍼 중복*(아래 SSOT-T1/T2)은 지금 정리 가능.
- **profile YAML의 값** — `profiles/*.yaml`의 chain_id/ports는 정당한 설정 데이터(SSOT 본체)다. 문제는 *코드에 박힌 default* 가 이를 복제하는 것(SSOT-X1).

→ 이 문서의 모든 작업은 **단일 레이어 내부** 또는 **스키마-as-SSOT 강화** 방향으로만 제안한다. 레이어 간 로직 통합은 마이그레이션 로드맵에 위임한다.

---

## 1. SSOT 위반 (우선 처리 권장)

### 1.1 Cross-layer 상수 중복 — `SSOT-X1` 🔴 (구조적)

profile **default 값**이 4개 레이어에 하드코딩으로 산재한다. profile YAML(데이터)이 아니라 *코드/문서의 fallback 값* 이 문제다.

| 값 | 출현 위치 |
|---|---|
| `chain_id 8283` | `lib/adapters/stablenet.sh:45` · `network/internal/adapters/stablenet/stablenet.go:60` · `network/internal/probe/signatures.go:23` · `mcp-server/src/tools/schema.ts:22-23` |
| ports `30301/8501/9501` | `lib/profile.sh:436-438` · `mcp-server/src/tools/schema.ts:65-67` · (+ Go testdata) |
| binary `gstable` | `lib/profile.sh:406` · `lib/cmd_stop.sh:20` · `lib/cmd_node.sh:262` · `mcp-server/src/tools/schema.ts:19` (+ HARDCODING_AUDIT의 9곳) |

- **근본 원인**: `network/schema/network.json`이 스키마 SSOT로 선언됐으나(S8), default 값은 스키마에 명시되지 않아 각 레이어가 자체 fallback을 둔다.
- **방향**: default를 `network/schema/network.json`의 `default` 키로 승격 → Go는 생성 타입에서, bash/TS는 스키마를 읽어 파생. 최소한 `schema.ts`의 default 문서 문자열은 스키마에서 생성하도록.
- **주의**: HARDCODING_AUDIT의 `gstable` 9곳은 이미 M4로 추적 중. 본 항목은 그와 **합쳐서** 진행.

### 1.2 Go: 에러 코드 문자열 중복 — `SSOT-G1` 🔴

`network/cmd/chainbench-net/errors.go` 가 생성 상수(`types.ResultErrorCodeINVALIDARGS` 등, `event_gen.go:147-151`)를 두고도 문자열 리터럴을 **6곳**에서 재사용:
- 생성자 5개: `errors.go:31,35,39,43,47` — `types.ResultErrorCode("INVALID_ARGS")` 형태
- `exitCode()` switch: `errors.go:58-62` — `case "NOT_SUPPORTED"` 등

→ 생성 상수로 교체. 스키마 변경 시 컴파일 타임에 검출되도록.

### 1.3 Go: dispatch ↔ schema 계약 drift — `SSOT-G2` 🔴

`handlers.go` 의 `allHandlers()` 가 `network.init` / `network.start_all` / `network.restart` / `network.clean` 을 등록하지만 `network/schema/command.json` enum(`command_gen.go:24-47`)에는 **없다** (있는 것은 `network.stop_all`, `network.status`). 스키마 검증이 이 명령들을 검증할 수 없다.
- **방향**: 스키마에 4개 명령 추가 + `go generate` 재생성. Sprint 5c.4.2(lifecycle reroute)에서 어차피 이 핸들러들을 손대므로 **그때 동시 처리**.

### 1.4 TS: 응답 헬퍼 중복 — `SSOT-T1` 🟠

- `formatResult()` 가 **5개 파일**에 동일 정의: `config.ts:5` · `remote.ts:21` · `log.ts:5` · `spec.ts:5` · `test.ts:7`
- `textResult()` 가 **2개 파일**에 동일 정의: `network.ts:20` · `consensus.ts:15`
- 이미 `utils/mcpResp.ts`(`errorResp`)가 응답 SSOT로 존재 → `formatResult`/`textResult`를 여기로 이동하고 import 통일. (저위험·고가치, 마이그레이션과 무관)

### 1.5 TS: RPC 메서드 검증 정규식 중복 — `SSOT-T2` 🟠

`remote.ts:29-34` 의 `validateRpcMethod()` 가 `utils/hex.ts` 의 `RPC_METHOD` 정규식을 재구현. `node.ts`는 이미 `RPC_METHOD`로 통합됨. → `remote.ts`도 `hex.ts` 사용. (NEXT_WORK §3 P3 "validateRpcMethod duplicate in remote.ts"로 이미 추적 중 — 본 정리에 흡수)

### 1.6 bash: 타임스탬프 포맷 / 타임아웃 상수 산재 — `SSOT-B1` 🟡

- ISO 타임스탬프 `"%Y-%m-%dT%H:%M:%SZ"` 가 8곳: `cmd_start.sh:258` · `cmd_node.sh:217,388` · `pids_state.sh:183,213` · `formatter.sh:32,55` · `remote_state.sh:138`. → `common.sh`에 `cb_iso_now()` 헬퍼 하나로.
- RPC 타임아웃 상수 네이밍 불일치: `lib/rpc_client.sh:20-21` `CB_RPC_TIMEOUT_*` vs `tests/lib/rpc.sh:26-27` `_CB_RPC_TIMEOUT_*` → 단일 정의 + source.

### 1.7 (참고) Go 내부 매직 상수 — `SSOT-G3` 🟡

read 핸들러가 `10*time.Second`, write 핸들러가 `30*time.Second` 를 인라인 반복(`handlers_node_read.go` 8곳, tx 4곳). `tx_wait`는 이미 명명 상수를 둠 → 동일하게 `const readTimeout/writeTimeout` 패키지 레벨로.

---

## 2. Clean Code 위반

### 2.1 파일/함수 크기 초과 (`~/.claude/rules/coding-style.md`: 파일 200–400 권장/800 상한, 함수 <50)

**Go** (이미 NEXT_WORK §4.6에서 추적 중 — 본 문서는 추출 단위를 구체화):
- `handlers_node_read.go` **929줄** (800 상한 초과) — 긴 closure: `newHandleNodeContractCall`(~137), `newHandleNodeEventsGet`(~138), `newHandleNodeAccountState`(~152)
- `handlers_node_tx.go` **948줄** (800 초과) — `newHandleNodeTxSend`(~185), `newHandleNodeContractDeploy`(~200), `newHandleNodeTxFeeDelegationSend`(~212)
- `handlers_network.go` **522줄** — `newHandleNetworkAttach`(~102)

**bash** (400 권장 초과): `cmd_test.sh`(639) · `cmd_node.sh`(602) · `profile.sh`(521) · `cmd_remote.sh`(434) · `json_helpers.sh`(426)

**TS**: `chain_tx.ts`(488) — `_buildTxSendWireArgs`(~142줄, 4-mode 중첩 검증)

### 2.2 Go: 핸들러 내부 중복 패턴 — `CC-G1` 🔴

가장 ROI 높은 추출 대상 (handler closure가 길어지는 근본 원인):
- **block 라벨 파싱** (`latest/pending/earliest` + hex) 4중 복제: `read.go:172,376,625,788`. `parseBlockArg()`가 이미 있으나 `events_get`만 사용 → balance/contract_call/account_state에 재사용.
- **hex→big.Int 파싱** `new(big.Int).SetString(TrimPrefix(x,"0x"),16)` 20+회 → `parseHexInt`/`parseHexBytes` 헬퍼.
- **fee-mode selector** (legacy/1559 판정) `tx.go:123-133` ↔ `tx.go:349-358` 동일 → `selectFeeMode()`.
- **resolveNode → dialNode → RPC** 스캐폴딩 7+회 → 얇은 wrapper.
- **`json.Unmarshal(args,&req)` + `NewInvalidArgs`** 23회 → 제네릭 헬퍼.

→ 위 헬퍼 추출만으로 두 대형 파일이 800 미만으로 자연 분해됨.

### 2.3 bash: 임베디드 Python / 이중 백엔드 — `CC-B1` 🟠

- `profile.sh:32-275` — **240줄 임베디드 Python YAML 파서**. 단일 파일 절반. 외부 스크립트(`scripts/`)로 추출 권장.
- `json_helpers.sh` — jq/python3 **이중 백엔드**가 read/write/merge마다 로직을 2벌 유지. 단일 백엔드 또는 공통 추상화.
- `import sys, json` heredoc이 코드베이스 전반 60+곳 — 공통 파이썬 유틸로.

### 2.4 TS: wire-args 빌드 패턴 반복 — `CC-T1` 🟡

optional 필드를 `if (x) args.x = x` 식으로 쌓는 패턴이 `chain_read.ts`(4 핸들러)·`chain_tx.ts`·`lifecycle.ts`에 반복 → `buildWireArgs(spec)` 헬퍼로 표준화. (NEXT_WORK §3 P3 "_buildTxSendWireArgs vs _accountStateHandler 패턴 차이"의 연장)

### 2.5 매직값 — `CC-MISC` 🟢

- TS `network.ts:189` BFT 임계 `Math.ceil(2*N/3)` 무설명 · `80`% round-0 임계(`consensus.ts:109,242`) · `2000`ms sleep(`network.ts:166,233`)
- bash 매직 sleep(`cmd_stop.sh:42,48` `0.5/1`, `cmd_init.sh:95` `2`) · `contract.sh:73` base port `8500`(profile은 8501)

---

## 3. 우선순위별 작업 목록

P0 = 저위험·고가치, 마이그레이션과 독립 → 지금 바로. P1 = 중간, 관련 코드 손댈 때. P2 = 큰 구조, 별도 sprint.

| ID | 작업 | 레이어 | 위험 | 노력 | 상태 | 트리거 |
|---|---|---|---|---|---|---|
| **P0-1** SSOT-T1 | `formatResult`/`textResult` → `mcpResp.ts`로 통합 (7곳 제거) | TS | 낮음 | S | ✅ PR #3 | — |
| **P0-2** SSOT-T2 | `remote.ts` `validateRpcMethod` → `hex.ts RPC_METHOD` | TS | 낮음 | S | ✅ PR #3 | — |
| **P0-3** SSOT-G1 | `errors.go` 문자열 → 생성 상수 6곳 | Go | 낮음 | S | ✅ PR #3 | — |
| **P0-4** SSOT-G3 | read/write 타임아웃 명명 상수화 | Go | 낮음 | S | ✅ PR #3 | — |
| **P0-5** SSOT-B1 | `cb_iso_now()` 헬퍼 (8곳 통합) | bash | 낮음 | S | ✅ PR #3 | — |
| **P1-1** CC-G1 | `parseBlockNumberArg`/`selectFeeMode` 추출 + 대형 핸들러 파일 분해 | Go | 중간 | M | ✅ PR #3 | — |
| **P1-2** SSOT-G2 | `command.json`에 init/start_all/restart/clean 추가 + regenerate | Go | 중간 | S | ⬜ 남음 | **Sprint 5c.4.2와 동시** |
| **P1-3** CC-T1 | `buildWireArgs()` 헬퍼로 wire-args 표준화 | TS | 낮음 | M | ⬜ 남음 | chain_*.ts touch 시 |
| **P1-4** SSOT-X1 | profile default를 스키마 `default`로 승격, 레이어가 파생 | 전레이어 | 중간 | M | ⬜ 남음 | M4(adapter binary) 동시 |
| **P2-1** CC-B1 | `profile.sh` Python 파서 추출 · `json_helpers.sh` 백엔드 단일화 | bash | 높음 | L | ⬜ 남음 | 별도 sprint |
| **P2-2** | bash 대형 파일 분할 (cmd_test/cmd_node/cmd_remote) | bash | 중간 | L | ⬜ 남음 | 해당 파일 기능 추가 시 |

`S`≈반나절, `M`≈1–2일, `L`≈sprint 단위.

**P0-5 범위 조정**: 원안의 "RPC 타임아웃 상수 단일화"는 prod(`CHAINBENCH_RPC_TIMEOUT_LOCAL/REMOTE`)와 test(`CHAINBENCH_RPC_TIMEOUT`)의 env 계약·기본값이 다르고 `tests/lib/rpc.sh`가 독립 소싱되어 저위험이 아님 → 아래 §6 N-A 로 분리(잔여).

---

## 4. 권장 진행 순서

1. **P0 묶음 먼저** (P0-1~5): 단일 레이어 내부, 동작 불변, `refactor(scope):` 커밋 5개. 마이그레이션과 충돌 없음. 즉시 안전.
2. **P1-2 는 Sprint 5c.4.2에 끼워서** — 어차피 lifecycle reroute가 그 핸들러를 손댐.
3. **P1-1(Go 헬퍼 추출)** 은 다음 read/tx 핸들러 추가 직전에 — NEXT_WORK §4.6의 "5번째 핸들러 추가 시 분할" 트리거와 일치.
4. **P1-4 / P2** 는 로드맵 sprint 경계에서. 특히 P2-1은 동작 회귀 위험이 커 충분한 e2e 후에.

각 작업은 프로젝트 규약(NEXT_WORK §4.1)대로 `refactor(<scope>):` 단위 커밋, 동작 변경 없음 보장(테스트 green 유지)을 전제로 한다.

---

## 5. 기존 추적과의 관계

본 문서는 `NEXT_WORK.md §3 P3` tech-debt 표와 **중복 추적이 아니라 보완**이다:
- 이미 추적 중(handlers 파일 크기, validateRpcMethod, feeDelegationAllowedChains, gstable 하드코딩)은 본 문서에서 **추출 단위·우선순위**를 구체화.
- 신규로 부각된 것: SSOT-G1(에러 코드 문자열), SSOT-G2(dispatch↔schema drift), SSOT-T1(formatResult/textResult 7중복), SSOT-X1(cross-layer default 승격), SSOT-B1(타임스탬프 8중복).

---

## 6. 진행 현황 + 남은 작업 (2026-06-26)

### 6.1 완료 — PR #3 (squash merge, commit `63f1d43`)

- **SSOT (P0-1~5)**: errors.go 생성상수 · 타임아웃 명명상수 · `cb_iso_now()` · MCP `formatExecResult`/`textResult`/`RPC_METHOD` 통합.
- **Clean-code (P1-1)**: `parseBlockNumberArg`/`selectFeeMode` 추출 + 핸들러 파일 분해 — `handlers_node_read.go` 929→655, `handlers_node_tx.go` 948→680 (+ `handlers_node_read_events.go`, `handlers_node_tx_fee_delegation.go`).
- **사전 버그 2건** (리팩토링 중 발견, 본 계획 범위 밖이었으나 동반 수정):
  - `fix(lib)`: BSD/macOS 비호환 mktemp 템플릿 (`-XXXXXX.json` → `-XXXXXX`) — `profile.sh` + `logs/timeline.sh` + `logs/anomaly.sh`.
  - `fix(report)`: 결과 0건일 때 `report --format json` 이 빈 stdout → zero-count JSON/markdown 으로 수정.
- **검증**: Go 15 pkg green · TS vitest 123 pass · bash 36/38 (남은 2건은 `cast` 미설치 환경분).

### 6.2 남은 작업 — 우선순위순

**즉시:**

| ID | 작업 | 비고 |
|---|---|---|
| **N2** | foundry(`cast`) 프로비저닝을 coding-agent `doctor`/`setup` 에 추가 | `doctor`=설치 진단(read-only), `setup`=미설치 시 `foundryup` 안내/설치. 완료 시 bash `lib-contract`/`lib-event` 포함 38/38. **코드 버그 아님 — 환경 의존** |

**P1 (관련 코드 손댈 때):**

| ID | 작업 | 트리거 |
|---|---|---|
| **P1-2** SSOT-G2 | `command.json` 에 `network.init/start_all/restart/clean` 추가 + `go generate` (dispatch↔schema drift) | Sprint 5c.4.2 (lifecycle reroute) 와 동시 |
| **P1-3** CC-T1 | TS `buildWireArgs()` 헬퍼로 wire-args 빌드 패턴 표준화 | chain_read/chain_tx/lifecycle.ts touch 시 |
| **P1-4** SSOT-X1 | profile default(chain_id/ports/binary)를 스키마 `default` 로 승격 → 레이어가 파생 | M4 (adapter binary) 와 동시 |

**P2 (별도 sprint, 큰 구조):**

| ID | 작업 | 위험 |
|---|---|---|
| **P2-1** CC-B1 | `profile.sh` 임베디드 Python(240줄) 추출 · `json_helpers.sh` jq/python 이중백엔드 단일화 | 높음 — 충분한 e2e 후 |
| **P2-2** | bash 대형 파일 분할 (`cmd_test.sh` 639 · `cmd_node.sh` 602 · `cmd_remote.sh` 434) | 중간 |

**기타 (잔여 항목):**

| ID | 작업 | 비고 |
|---|---|---|
| **N-A** | RPC 타임아웃 상수 네이밍 단일화 (`CB_RPC_TIMEOUT_*` vs `_CB_RPC_TIMEOUT_*` vs `_CB_REMOTE_RPC_TIMEOUT`) | P0-5 에서 분리 — env 계약 차이로 저위험 아님. 회귀 테스트 영향 검토 후 |
| **N-B** | §2.5 매직값 상수화 (BFT 임계 2/3, round-0 80%, 2000ms sleep 등) | 해당 파일 touch 시 흡수 |

### 6.3 추천 진행 순서

`N2 (foundry → 38/38)` → `P1-2 + Sprint 5c.4.2 묶음` → `P1-3 / P1-4` → `P2`.
