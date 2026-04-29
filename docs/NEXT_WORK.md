# Chainbench — 다음 작업 핸드오프 문서

> 작성일: 2026-04-24 (Sprint 4 종료 시점)
> 최종 업데이트: 2026-04-29 (Sprint 5a 완료 — Sprint 5 series 진행 중)
> 이 문서는 **다른 세션에서 맥락 없이 작업을 이어갈 수 있도록** 작성됨.
> 읽는 순서: §1 (프로젝트 컨텍스트) → §2 (현재 상태) → §3 (다음 작업) → §4 (규약) → §5 (주의사항).

---

## 1. 프로젝트 컨텍스트 — 30초 버전

**Chainbench** = go-stablenet(geth 포크, WBFT 합의)용 로컬 블록체인 샌드박스 +
테스트 프레임워크. 최근 방향은 **네트워크 추상화 레이어** 구축 — bash CLI
(`chainbench`) 아래에 Go 바이너리(`chainbench-net`)가 존재하고, MCP 서버도 이
Go 바이너리를 경유하도록 이관 중.

핵심 축:
- **bash CLI** (`chainbench init/start/stop/test/...`) — 기존 진입점, 유지
- **Go network abstraction** (`chainbench-net` at `network/`) — probe / attach /
  read RPC / tx.send 를 wire protocol (NDJSON over stdin/stdout) 로 노출
- **bash client** (`lib/network_client.sh`) — Go 바이너리 호출 wrapper

설계 원칙: 상위 레이어는 **local / remote / (미래의) ssh-remote** 를 동일 명령
표면으로 다루고, 하위에서 provider-per-node 디스패치. 자세한 설계는
`docs/VISION_AND_ROADMAP.md` §5 (특히 §5.17).

### 1.1 디렉토리 레이아웃 (핵심만)

```
chainbench/
├── lib/                              # 기존 bash 라이브러리
│   ├── adapters/{stablenet,wbft,wemix}.sh  # 체인별 어댑터 (python + template)
│   ├── chain_adapter.sh              # bash adapter 로더 (Strategy)
│   ├── network_client.sh             # Go 바이너리 호출 wrapper
│   └── cmd_*.sh                      # 각 CLI 서브커맨드
├── network/                          # Go 모듈 (github.com/0xmhha/chainbench/network)
│   ├── cmd/chainbench-net/           # 바이너리 entrypoint
│   │   ├── main.go, run.go, errors.go
│   │   ├── handlers.go               # 디스패치 + 공유 resolver (160 라인)
│   │   ├── handlers_network.go       # network.load/probe/attach (Sprint 4b split)
│   │   ├── handlers_node_lifecycle.go # node.stop/start/restart/tail_log
│   │   ├── handlers_node_read.go     # block_number/chain_id/balance/gas_price
│   │   │                             # + contract_call/events_get/account_state (Sprint 4d)
│   │   │                             # 844 라인 — 800 soft cap 초과, 5번째 read handler 추가 시 분할 권장
│   │   ├── handlers_node_tx.go       # tx_send/tx_wait/tx_fee_delegation_send/contract_deploy
│   │   │                             # 953 라인 — 동일 P3 debt
│   │   ├── handlers_test.go          # ~4000+ 라인 유닛 테스트
│   │   └── e2e_test.go               # cobra in-process E2E (4d 후 12+ 테스트)
│   ├── internal/
│   │   ├── adapters/                 # Sprint 3c — spec + stablenet + wbft + wemix
│   │   ├── drivers/
│   │   │   ├── local/                # chainbench.sh exec wrapper
│   │   │   └── remote/               # ethclient + auth RoundTripper
│   │   ├── events/                   # 이벤트 버스
│   │   ├── probe/                    # Sprint 3a chain_type 감지
│   │   ├── signer/                   # Sprint 4 서명 경계 (env-only)
│   │   ├── state/                    # pids.json / current-profile.yaml / networks/<name>.json
│   │   ├── types/                    # go-jsonschema 로 생성된 타입들 (command_gen.go 등)
│   │   └── wire/                     # NDJSON encoder/decoder + slog emitter
│   └── schema/                       # JSON Schema — command.json, event.json, network.json
│       └── fixtures/                 # 스키마 유효성 테스트 픽스처
├── templates/                        # genesis.template.json, node.template.toml
├── tests/unit/                       # bash unit tests (30개, 100% 통과 — Sprint 4c 기준)
│   ├── lib/assert.sh
│   ├── run.sh                        # bash 3.2 → 4+ 재-exec 자동 처리
│   └── tests/*.sh                    # 개별 테스트 파일
└── docs/
    ├── VISION_AND_ROADMAP.md         # 전체 비전 + 로드맵 (SSoT)
    ├── SECURITY_KEY_HANDLING.md      # Sprint 4 결과
    ├── NEXT_WORK.md                  # (이 문서)
    └── superpowers/
        ├── specs/YYYY-MM-DD-<topic>.md   # 각 sprint spec
        └── plans/YYYY-MM-DD-<topic>.md   # 각 sprint plan
```

### 1.2 핵심 기술 스택

- Go 1.25 (go.mod 선언), 로컬 toolchain 은 1.23.12 일 수 있음 — 무방
- `github.com/ethereum/go-ethereum v1.17.2` (Sprint 3b.2a 도입)
- `github.com/pelletier/go-toml/v2 v2.3.0` (Sprint 3c 도입, 테스트용)
- bash 3.2+ (macOS 호환; tests/unit/run.sh 가 4+ 재-exec 처리)
- Python 3 (bash 테스트의 JSON-RPC 모킹용)
- jq (bash 클라이언트의 NDJSON 파싱용)

---

## 2. 현재 상태 — 무엇이 완료되었나

### 2.1 Sprint 타임라인

| Sprint | 범위 | 완료일 | 커밋 수 |
|---|---|---|---|
| 2a | wire protocol + LocalDriver + events | 2026-04-22 | (이전) |
| 2b | state.LoadActive + node.stop/start/restart/tail_log | 2026-04-22~23 | (이전) |
| 2c | bash client (`lib/network_client.sh`) + unit tests | 2026-04-23 | (이전) |
| 3a | chain_type probe | 2026-04-23 | 10 |
| 3b | network.attach + state routing (non-local names) | 2026-04-23 | 11 |
| 3b.2a | RemoteDriver + node.block_number + resolveNode (M4 부분) | 2026-04-24 | 9 |
| 3b.2b | Remote RPC auth (API key / JWT via Transport) | 2026-04-24 | 7 |
| 3b.2c | node.chain_id/balance/gas_price + attach 검증 + M4 완전 | 2026-04-24 | 7 |
| 3c | Adapter Go 포팅 (stablenet) + wbft/wemix 스켈레톤 | 2026-04-24 | 7 |
| **4** | **Signer boundary (env) + node.tx_send + 보안 경계 테스트** | **2026-04-24** | **8** |
| **4b** | **keystore signer + EIP-1559 + node.tx_wait** | **2026-04-27** | **9** |
| **4c** | **SignHash + EIP-7702 SetCode + go-stablenet 0x16 fee delegation** | **2026-04-27** | **9** |
| **4d** | **Contract deploy/call + events_get + account_state (Sprint 4 series 종료)** | **2026-04-27** | **10** |
| **5c.1** | **TS wire helper + transformer + 첫 2 high-level MCP tool (account_state + tx_send legacy/1559)** | **2026-04-28** | **7** |
| **5c.2** | **MCP 남은 4 high-level tool + tx_send 4 mode 완비 + chain.ts split** | **2026-04-29** | **9** |
| **5c.3** | **MCP reroute pass 1 — utils 추출 + node.rpc Go 핸들러 + 3 chainbench_node_* reroute (8% of 38) + integration test 도입** | **2026-04-29** | **8** |
| **5a** | **Capability gate — `network.capabilities` Go 핸들러 + `chainbench_network_capabilities` MCP tool + bash `requires_capabilities` frontmatter gating + 2 fault test PoC** | **2026-04-29** | **6** |

각 sprint 는 `docs/superpowers/specs/<date>-<topic>.md` + `docs/superpowers/plans/<date>-<topic>.md` 를 가지고 있음.

### 2.2 현재 커맨드 표면 (chainbench-net wire protocol)

`network/schema/command.json` enum 기준:

**Network 관리:**
- `network.load` — 로컬/원격 네트워크 state 조회 (이름이 "local" 이면 pids.json + profile; 아니면 networks/<name>.json)
- `network.attach` — URL 에 probe → 원격 네트워크 state 저장
- `network.probe` — URL의 chain_type / chain_id 감지

**Node 관리 (로컬 전용 — non-local 네트워크 arg 로 호출 시 NOT_SUPPORTED):**
- `node.stop` / `node.start` / `node.restart` / `node.tail_log`

**Node RPC (로컬+원격 공통):**
- `node.block_number`
- `node.chain_id`
- `node.balance` (address + block_number 옵션)
- `node.gas_price`
- `node.tx_send` (서명 + 방송, signer alias 필수, optional `authorization_list` → EIP-7702 SetCodeTx)
- `node.tx_fee_delegation_send` (Sprint 4c, stablenet only — sender + fee_payer 이중 서명, 0x16 envelope)
- `node.tx_wait` (receipt polling, exponential backoff)
- `node.contract_deploy` (Sprint 4d — bytecode + 옵션 ABI 생성자 args. legacy / 1559 fee-mode)
- `node.contract_call` (Sprint 4d — raw calldata or ABI+method+args. read-only `eth_call`)
- `node.events_get` (Sprint 4d — `eth_getLogs` + 옵션 ABI 기반 log decode)
- `node.account_state` (Sprint 4d — composite reader: balance / nonce / code / storage 선택)
- `node.rpc` (Sprint 5c.3 — generic JSON-RPC passthrough; method regex + params array 검증)

**스키마에 선언됐지만 미구현:**
- `network.capabilities` — Sprint 5 에서 capability gate 와 함께
- `tx.send` — `node.tx_send` 가 대체 (cross-network 급이 필요하면 그때)
- `subscription.open` — WebSocket, 후속

### 2.3 테스트 매트릭스 (Sprint 5c.3 종료 기준)

- Go: 16 packages 전부 green (Sprint 5c.3 에서 `node.rpc` 핸들러 + 5 unit tests 추가)
- Bash: 33/33 테스트 green (5c.3 에서 미증가 — bash 측 변경 없음)
- mcp-server vitest: 90 tests (9 wire + 8 wireResult + 1 chain_read_timeout + 26 chain_read + 29 chain_tx + 13 node + 2 node_start_fallback + 1 mcpResp + 1 integration) — Sprint 5c.3 에서 utils 추출 후 node.test.ts 13 cases + node_start fallback 2 + mcpResp 1 + real-binary integration 1 추가
- Integration test 실행: `npm test` 전 `cd network && go build -o bin/chainbench-net ./cmd/chainbench-net` 필요. binary 부재 시 `it.skip` 으로 자동 스킵
- 주요 커버리지:
  - `signer` 보안 테스트: 유닛 + Go E2E + bash subprocess boundary (Scenario 1~4)
  - `probe` 92.4%, `remote` 97%, `adapters/stablenet` 높은 커버 + 골든 파일 계약
  - 0x4 SetCode + 0x16 fee-delegation: handler 단위 + Go E2E + bash spawn 3-layer 커버
  - Sprint 4d contract / events / state: handler 단위 + Go E2E + bash spawn 3-layer 커버. `abiutil` 단위 테스트로 ABI parse / coerce / pack / unpack / DecodeLog 격리
  - Sprint 5c.1 mcp-server: wire helper (mock binary spawn 9 케이스) + transformer (8 케이스) + chain tools (account_state 4 + tx_send 10 케이스, mock binary spawn)
  - Sprint 5c.3 mcp-server: node tools (rpc / stop / start) wire-call 단위 + binary_path bash fallback + utils mcpResp + real-binary integration (chainbench-net spawn + Python JSON-RPC mock)

---

## 3. 다음 작업 — 우선순위별

> **상위 컨텍스트** (2026-04-29 정렬): chainbench 의 두 모드 (A) coding agent
> evaluation harness, (B) 독립 도구 가운데 **모드 (A) 의 evaluation 표면 강화**
> 가 향후 sprint 의 메인 동력이다. 자세한 매트릭스는 `docs/EVALUATION_CAPABILITY.md`.
>
> Sprint 4 시리즈 (4 → 4b → 4c → 4d) 가 Go `network/` 의 tx + read 매트릭스를
> 완성하고 (2026-04-27), Sprint 5c.1 (2026-04-28) + 5c.2 (2026-04-29) 가 MCP
> high-level tool 6종 (`account_state` / `contract_deploy` / `contract_call` /
> `events_get` / `tx_wait` / `tx_send` 4-mode) 을 모두 노출했다. EVALUATION_CAPABILITY
> MCP column 약 60% 도달. Sprint 5c.3 (2026-04-29) 가 utils 추출 + `node.rpc`
> Go 핸들러 + 3 `chainbench_node_*` tool reroute (3/38 ≈ 8%) + real-binary
> integration test layer 를 도입. Sprint 5a (2026-04-29) 가 capability gate
> 의 wire + MCP + bash 세 layer 통합 — `chainbench_network_capabilities` 노출
> + `requires_capabilities` frontmatter 게이팅. 다음 P1 은 **Sprint 5c.4 —
> lifecycle reroute (chainbench_init/start/stop/restart/status/clean) +
> Go-side wire 핸들러 추가**.

### 🟥 Priority 1 — Sprint 5c.4: lifecycle reroute (chainbench_init/start/stop/restart/status/clean)

**배경**: Sprint 5c.3 (2026-04-29) 가 reroute 첫 패스를 완료 — 3 `chainbench_node_*`
tool 이 wire 경유로 전환되며 utils/ extraction (errorResp + hex 상수 단일 source)
과 real-binary integration test layer 가 자리 잡음. Reroute 진행도 3/38 (~8%).
다음 단계는 **lifecycle 6-tool 묶음** — `chainbench_init/start/stop/restart/status/clean`.
이들은 현재 `runChainbench` 로 bash CLI 를 직접 호출하므로, MCP → wire → Go
일관 표면을 위해서는 (1) Go 측 wire 핸들러 신규 작성 (`network.init`,
`network.start_all`, `network.stop_all`, `network.restart`, `network.status`,
`network.clean` — 일부는 기존 `cmd_init/start/stop` shell 을 LocalDriver 가 wrap)
+ (2) MCP 측 reroute + integration harness 재사용이 필요.

**5c.4 (1순위)**: lifecycle reroute
- Go-side wire handlers 6종 추가 (스키마 enum 갱신 + handler 등록 + handler 단위 테스트)
- 6 lifecycle MCP tool reroute — `chainbench_init/start/stop/restart/status/clean` → callWire
- `chainbench_node_start` 의 `binary_path` fallback 제거 — Go `node.start` 가 binary_path arg 수용 (Sprint 5c.3 P3 row 닫힘)
- Integration test harness 추출 (`test/integration/_harness.ts`) — 5c.3 의 단일 파일 setup 재사용 형태로
- Cleanup-await + port-race 진단 메시지 보강 (5c.3 P3 follow-up)
- `validateRpcMethod` duplicate in remote.ts 통합 (5c.3 P3 follow-up)

**5a/5b/5d (2순위)**: 5c.4 와 독립적으로 진행 가능
- **5a**: capability gate (`network.capabilities` 커맨드 + test 프론트매터 `requires_capabilities`)
- **5b**: SSHRemoteDriver 설계 + 초기 구현 (Q6, S6)
- **5d**: Hybrid 네트워크 예제 (`profiles/hybrid-example.yaml`) + 테스트 시나리오

**Adapter.SupportedTxTypes() promotion**: Sprint 4c 의 `feeDelegationAllowedChains`
hardcoded map 을 Adapter 인터페이스 메서드로 승격 (`docs/ADAPTER_CONTRACT.md` §3 참조).
5번째 chain-specific tx 타입 도입 시 동시 진행.

### 🟨 Priority 2 — Sprint 4d 후속 / 보강 (트리거 조건 시)

Sprint 4d 가 chain-specific cell 은 의도적으로 미터치. 트리거 조건 충족 시
별도 sprint 또는 5 series 와 병행:

- **Account Extra** (go-stablenet 의 `isBlacklisted` / `isAuthorized` 등) — 별도 명령 또는 `chain_state` 합성. Adapter 인터페이스 promotion 과 묶을 가능성.
- **abiutil tuple / nested-array / fixed-bytesN(N≠32) 지원** — 현재 명시적 미지원. 실수요 발생 시 확장 (raw calldata fallback 으로 우회 가능).
- **`abiutil.DecodeLog` anonymous event** — 현재 `topics[0]` 가 event signature 가정. anonymous event (signature 없음) 는 미커버.

### 🟩 Priority 3 — 누적 tech debt (건드릴 때 같이 처리)

누적된 minor 항목들 — 별도 sprint 아니라 관련 코드 건드릴 때 흡수.

**이전 sprint 에서 이월 (원인 스프린트 → 설명)**:

| 원인 | 항목 | 설명 | 트리거 조건 |
|---|---|---|---|
| 2b.3 | APIError.Details 구조화 | 다단계 실패의 phase 별 오류 | 2번째 다단계 명령 등장 시 |
| 2c | jq 3→1 호출 통합 | `_cb_net_parse_result` 에서 jq 3회 → 1회 | 스트리밍/고빈도 케이스 |
| 2c | jq 버전 게이트 | 최신 jq 전용 문법 사용 시 | 최신 jq 기능 도입 시 |
| 3a | isKnownOverride SSoT 중복 | signatures 테이블과 chain-type 열거 중복 | 5번째 chain_type 추가 시 |
| 3b | `created` 플래그 TOCTOU | os.Stat + SaveRemote race window | 멀티테넌트 전환 시 |
| 3b | valid-but-unattached bash wire 커버리지 | Go 는 있고 bash 는 없음 | 3b.2 계열 touch 시 |
| 3b.2a | `Node.Http == ""` 가드 | IPC-only 노드 변형 대비 | IPC 지원 추가 시 |
| 3b.2b | 말포름 URL fall-through in DialWithOptions | 거의 unreachable | remote 관련 대규모 refactor 시 |
| 3b.2c | `tail_log` 의 `Name: "local"` 하드코딩 | ProviderMeta 접근 때문에 M4 에서 제외 | tail_log 수정 시 |
| 3b.2c | `TestHandleNodeBalance_BadAddress` 변형 부족 | too-short / non-hex 케이스 없음 | balance 관련 변경 시 |
| 3b.2c | `types.Auth` typed refactor | loose map → 태그된 union | generator 설정 변경 시 |
| 3c | `cloneMap` shallow | 호출자의 Profile in-place mutate | wire handler 랜딩 시 |
| 3c | 대형 integer alloc override `%v` 포맷 | 과학표기법 위험 (실제 사용 낮음) | |
| 3c | `getInt` 가 문자열-숫자 override 폴백 | 스키마 검증 함께 | 프로파일 검증 도입 시 |
| 3c | adapter fixture `govMasterMinter`/`govCouncil` 없음 | 4개 중 2개 커버 | 다음 adapter touch 시 |
| 3c | malformed JSON template 테스트 없음 | | |
| 4c | `handlers_node_tx.go` 핸들러 closure 길이 | `newHandleNodeTxSend` ~184 줄, `newHandleNodeTxFeeDelegationSend` ~211 줄, 4d 가 추가한 `newHandleNodeContractDeploy` 도 유사 길이 — 50줄 가이드라인 초과. 다음 핸들러 추가 시 phase 별 helper 추출 권장 (`parseTxSendArgs`, `signFeeDelegationEnvelope` 등). 800줄 ceiling 은 아직 여유. | 다음 `handlers_node_tx.go` 핸들러 추가 시 |
| 4c | `feeDelegationAllowedChains` hardcoded 맵 | chain-type 별 tx 타입 지원을 inline 으로 처리. `Adapter.SupportedTxTypes()` 인터페이스 promotion 은 Sprint 5+. | 5번째 chain-specific tx 타입 도입 시 |
| 4c | `resolveNode` 가 chain_type 미반환 | fee-delegation 핸들러가 `state.LoadActive` 두 번 호출 (resolveNode 내부 + chain_type 조회). resolveNode 시그니처 확장은 6개 read 핸들러 모두 영향. | tx_fee_delegation_send 가 hot path 가 될 때 |
| 4d | `handlers_node_read.go` 파일 크기 | Sprint 4d 가 `newHandleNodeContractCall` / `newHandleNodeEventsGet` / `newHandleNodeAccountState` 3개 추가 — 파일이 800줄 ceiling 근접. read handler 도 `handlers_node_tx.go` 와 같이 분할 또는 helper 추출 필요. | 5번째 read handler 추가 시 |
| 4d | `abiutil` tuple / nested-array / fixed-bytesN(N≠32) 미지원 | 의도적 격리 — 실수요 발생 시 확장. 사용자는 raw calldata fallback 으로 우회. | 사용자가 tuple-가진 ABI 인코딩 필요할 때 |
| 5c.1 | mcp-server vitest 통합 테스트 (실 chainbench-net spawn) 부재 | unit 만 있음. real-binary 통합은 5c.3 reroute 시 추가 검토. | reroute 시 |
| 5c.1 | `mcp-server/src/tools/` 의 큰 파일 (test.ts 335, schema.ts 308) | 5c.3 reroute 시 자연스럽게 흡수 | reroute 작업 시 |
| 5c.1 | wire helper 의 subscription / 장기 실행 호출 미지원 | 단일-result 호출만. WS subscription 은 chain log streaming sprint 에서 | WS subscription 도입 시 |
| 5c.1 | `_buildTxSendWireArgs` vs `_accountStateHandler` wire-args 구성 패턴 차이 | 전자는 `Object.entries` 루프, 후자는 explicit if. 5c.2 가 4개 tool 추가하면서 standardize 권장 | 5c.2 tool 추가 시 (해소됨) |
| 5c.2 | `chain_tx.ts` 505 lines — 권장치 200~400 초과 | 5c.3 reroute 시 자연 흡수 (per-tool 파일 분할 검토). 800 ceiling 여유. | reroute 작업 시 |
| 5c.2 | `errorResp` helper duplication (chain_read.ts + chain_tx.ts) | 두 파일에 동일 헬퍼 — 5c.3 P3 utils/mcpResp.ts 추출 | 5c.3 reroute 시 |
| 5c.2 | mcp-server vitest 통합 테스트 (실 chainbench-net spawn) 부재 | 5c.3 reroute 연결 테스트와 함께 검토 | reroute 작업 시 |
| 5c.2 | wire-args round-trip assertion 패턴 미정립 | mock 이 envelope 캡처 못함 — 5c.3 시 mock 확장 검토 | mock 강화 시 |
| 5c.3 | `chainbench_node_start binary_path` falls back to `runChainbench` | wire 측 `node.start` 가 `binary_path` arg 미지원 — 5c.4 가 Go 확장 시 fallback 제거 | 5c.4 lifecycle 작업 시 |
| 5c.3 | integration test cleanup-await 미적용 | `mockProc.kill("SIGTERM")` 후 await 부재 — 단일 test 는 무관, 5c.4 가 6+ integration test 추가 시 cumulative flake 위험 | harness extract 시 |
| 5c.3 | integration test port-race 진단 메시지 부족 | `pickFreePort` 후 Python listen 까지 race window. 충돌 시 stderr drained → 진단 어려움 | 5c.4 시 stderr capture 또는 retry loop |
| 5c.3 | integration test harness 미추출 | 단일 파일에 모든 setup. 5c.4 가 6+ tool 추가 전 `_harness.ts` 분리 권장 | 5c.4 첫 task |
| 5c.3 | `validateRpcMethod` duplicate in remote.ts | node.ts 가 `RPC_METHOD` 로 통합된 후에도 remote.ts 는 별도 helper 유지 | remote.ts reroute 시 |
| 5a | All fault/regression tests not yet frontmatter'd | 5a 는 fault/node-crash + node-recover 만 demo. 점진 부여 (network-partition, p2p-topology, two-down, txpool-leader-change, regression 카테고리들) | remote 환경에서 부적합 test 발견 시 |
| 5a | Runtime capability probe | provider declaration 만 사용. admin RPC 실제 동작 여부 등 runtime probe 후속 | runtime 신뢰도 이슈 발견 시 |
| 5a | MCP capability-aware tool gating | coding agent 가 capability 부재 시 자동 retry/skip 하는 layer 부재 — 사용자가 capability 응답 받아 직접 판단 | LLM ergonomics 강화 시 |
| 5a | Spec/plan terminology drift | spec §4.1 이 `state.LoadNodes` / `state.Node` 표기 — 실제는 `state.LoadActive` + `types.Node`. 후속 reader 혼란 방지를 위해 spec 갱신 가능 | 후속 sprint 5b 시 |

**로드맵 상 아직 불명확한 항목**:
- wbft `GenerateGenesis`/`GenerateToml` 실구현 — wbft 체인 실제 사용 시
- wemix 실구현 — wemix 체인 실제 사용 시
- `chainbench-net network.init` wire handler — bash `chainbench init` 의 Go 포팅 (Adapter 이미 Go 로 있음)
- 401/403 distinct APIError code — 3b.2b follow-up

**Layer 2 테스트 유틸리티 (구 `REMAINING_TASKS.md` 흡수)**:

이전(2026-04 초중반) Layer 2 테스트 유틸리티 sprint 의 의도적 보류 항목.
별도 sprint 가 아닌, 트리거 조건 충족 시 자연스럽게 진행.

| 원인 | 항목 | 설명 | 트리거 조건 |
|---|---|---|---|
| Phase B | `regression/lib/common.sh` 인라인 함수 | `send_raw_tx` / `gov_full_flow` / `assert_receipt_status` 를 Layer 2 로 점진 마이그레이션 (호환성 유지) | 관련 테스트 신규/수정 시 |
| Phase F | 고수준 assertion helper | `common.sh` 에 4개 이미 존재 | 반복 패턴 발견 시 |
| Phase I | rerun-failed + snapshot/restore | ROI 불확실 | 실사용 피드백 후 |
| Phase K | 의존성 그래프 | `depends_on` 데이터 부족 | 프론트매터 축적 후 |
| Phase L | MCP 대화형 세션 (daemon) | 구현 난이도 높음 | 명확한 유즈케이스 확보 시 |
| Phase C-3 | GitHub Actions 워크플로우 | 의도적 보류 | CI 도입 시 |

### ✅ 최근 완료 (참조용)

forward-looking 우선순위에서 분리한 완료 sprint 요약. 자세한 commit chain
및 결과는 §2.1 timeline + 각 sprint 의 spec/plan 파일을 참조.

| Sprint | 완료일 | 핵심 성과 |
|---|---|---|
| 4 | 2026-04-24 | env-only signer + `node.tx_send` (legacy) + 보안 경계 테스트 |
| 4 follow-ups | 2026-04-27 | `CHAINBENCH_NET_LOG` fix · `handlers.go` 5-file split · `SignTx` ctx 주석 |
| 4b | 2026-04-27 | keystore signer · EIP-1559 dynamic fee · `node.tx_wait` receipt polling |
| 4c | 2026-04-27 | `Signer.SignHash` · `SendRawTransaction` · EIP-7702 SetCode (0x4) · go-stablenet fee delegation (0x16) |
| 4d | 2026-04-27 | `abiutil` 패키지 · `remote.Client` read wrapper 4종 · `node.contract_deploy` / `contract_call` / `events_get` / `account_state` (Sprint 4 series 종료) |
| 5c.1 | 2026-04-28 | mcp-server: wire helper · NDJSON → MCP transformer · `chainbench_account_state` · `chainbench_tx_send` (legacy/1559) · vitest layer 도입 |
| 5c.2 | 2026-04-29 | mcp-server: chain.ts split · 4 신규 high-level tool · tx_send mode set_code/fee_delegation · vitest 73 tests |
| 5c.3 | 2026-04-29 | mcp-server: utils 추출 (errorResp + hex) · Go node.rpc 핸들러 · 3 chainbench_node_* reroute · 첫 real-binary integration test |
| 5a | 2026-04-29 | Capability gate — `network.capabilities` Go 핸들러 · `chainbench_network_capabilities` MCP tool · bash `requires_capabilities` frontmatter gating · 2 fault test PoC · vitest 94 / bash 34 |

Sprint 4 시리즈 종료 (4 → 4b → 4c → 4d, 2026-04-24 ~ 2026-04-27): Go
`network/` tx + read 매트릭스 100% 도달 — `docs/EVALUATION_CAPABILITY.md`
§2 / §4 Go column 모든 cell ✅. Sprint 5c.1 (2026-04-28) 이 MCP 노출의
첫 두 high-level tool 을 추가, Sprint 5c.2 (2026-04-29) 가 남은 4 tool +
tx_send mode 2종 (set_code / fee_delegation) 을 추가하여 EVALUATION_CAPABILITY
MCP column 약 60% 도달. Sprint 5c.3 (2026-04-29) 이 reroute 첫 패스 — utils
추출 + Go `node.rpc` 핸들러 + 3 `chainbench_node_*` tool reroute (3/38 ≈ 8%) +
real-binary integration test layer 도입. Sprint 5a (2026-04-29) 가 capability
gate 의 wire + MCP + bash 세 layer 통합 — fault test 가 remote 환경에서 자동
skip 가능. 다음 P1 은 §3 = Sprint 5c.4 (lifecycle reroute).

---

## 4. 프로젝트 규약 (10+ sprint 에서 확립됨)

### 4.1 커밋 discipline (절대 준수)

```
English 메시지
NO Co-Authored-By trailer
NO "Generated with Claude Code" attribution
NO emoji
Task 단위 커밋 (플랜의 task 1개 = 커밋 1~2개)
```

변경 종류별 커밋 prefix:
- `feat(<scope>):` — 새 기능
- `fix(<scope>):` — 버그 / 리뷰 지적 수정
- `test(<scope>):` — 테스트 추가/수정
- `refactor(<scope>):` — 동작 변경 없는 구조 개선
- `docs:` — 문서
- `style(<scope>):` — 포매팅, 린터 자동수정
- `chore(<scope>):` — go mod tidy, 설정 변경

### 4.2 워크플로우

새 sprint 진행 시 표준 플로우:

1. **spec 작성**: `docs/superpowers/specs/YYYY-MM-DD-<topic>.md`
   - Goal / Non-goals / User-facing surface / Architecture / Tests / Out-of-scope
2. **plan 작성**: `docs/superpowers/plans/YYYY-MM-DD-<topic>.md`
   - Task 분해 + 각 task 의 step-by-step
   - Step 은 bite-sized: 테스트 작성 → RED 확인 → 구현 → GREEN → 커밋
3. **spec+plan 커밋**: `docs: add Sprint X spec + plan for <topic>`
4. **subagent-driven-development**: task 마다 fresh subagent → 코드 리뷰 → 다음 task
5. **Final review**: 전체 sprint 를 security / scope / test quality 관점으로
6. **Roadmap update**: `docs/VISION_AND_ROADMAP.md` 의 Sprint 체크박스 갱신

### 4.3 테스트 레이어링

| 레이어 | 위치 | 용도 |
|---|---|---|
| Unit (Go) | `network/internal/**/<pkg>_test.go` | 패키지별 동작 |
| Handler | `network/cmd/chainbench-net/handlers_test.go` | 핸들러 인자 검증 + 오류 분류 |
| Go E2E | `network/cmd/chainbench-net/e2e_test.go` | cobra 인프로세스 — wire 스키마 검증 |
| Bash unit | `tests/unit/tests/*.sh` | subprocess 경계 — python mock 사용 |

Go E2E 템플릿 패턴 (반복 사용):
```go
rpcSrv := httptest.NewServer(http.HandlerFunc(func(w, r) {
    var req struct{ Method string; ID json.RawMessage }
    _ = json.NewDecoder(r.Body).Decode(&req)
    switch req.Method {
    case "eth_chainId":        fmt.Fprintf(w, ...)
    case "istanbul_getValidators", "wemix_getReward": /* -32601 */
    // 등
    }
}))
defer rpcSrv.Close()

t.Setenv("CHAINBENCH_STATE_DIR", t.TempDir())
root := newRootCmd()
root.SetIn(strings.NewReader(`{"command":"...","args":{...}}`))
root.SetOut(&stdout); root.SetErr(&stderr); root.SetArgs([]string{"run"})
if err := root.Execute(); err != nil { ... }

// NDJSON 스캔 + schema.ValidateBytes("event", line) 검증
// ResultMessage 터미네이터 찾아 파싱 후 필드 assert
```

Bash test 템플릿 (반복 사용):
```bash
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"
CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

TMPDIR_ROOT="$(mktemp -d)"
BINARY="${TMPDIR_ROOT}/chainbench-net-test"
( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net )
export CHAINBENCH_NET_BIN="${BINARY}"

PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
# ... JSON-RPC mock ...
PYEOF
MOCK_PID=$!
cleanup() { kill "${MOCK_PID}" 2>/dev/null || true; wait "${MOCK_PID}" 2>/dev/null || true; rm -rf "${TMPDIR_ROOT}"; }
trap cleanup EXIT INT TERM HUP

# Readiness poll (fail-loud)
mock_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then mock_ready=1; break; fi
  sleep 0.1
done
[[ "${mock_ready}" -eq 1 ]] || { echo "FATAL: mock not listening" >&2; cat "${MOCK_LOG}" >&2; exit 1; }

source "${CHAINBENCH_DIR}/lib/network_client.sh"

describe "..."
data="$(cb_net_call "..." '{...}')"
assert_eq "$(jq -r .field <<<"$data")" "expected" "label"

unit_summary
```

### 4.4 에러 분류 매트릭스

`network/cmd/chainbench-net/errors.go` 의 APIError 코드:

| 코드 | 사용처 |
|---|---|
| `INVALID_ARGS` | 호출자 입력의 구조적 문제 (null field, bad format, 알 수 없는 enum 값) |
| `UPSTREAM_ERROR` | 의존성 실패 (state 파일 없음, RPC 엔드포인트 실패, env 설정 오류) |
| `NOT_SUPPORTED` | 연산이 해당 환경에서 의미 없음 (로컬 전용 명령의 원격 호출) |
| `INTERNAL` | invariant 깨짐 (pre-check 통과했는데도 마샬 실패 등) |
| `PROTOCOL_ERROR` | wire 프로토콜 파싱 실패 (dispatcher 레벨) |

### 4.5 Redaction 패턴 (보안 경계)

`signer.sealed` 에서 확립:
- 필드 unexported (`key ecdsa.PrivateKey` 소문자 시작)
- 접근자 없음 (Address() 만 공개)
- `LogValue() slog.Value` → `"***"`
- `String() string` → `"<signer:***>"`
- `GoString() string` → `"<signer:***>"` (⚠️ `%#v` 가드용 — 빠지면 reflection 노출됨)
- 에러 메시지에 key material 포함 금지 — alias 와 env var 이름만 참조

민감한 값을 다루는 다른 boundary 가 생기면 이 패턴 그대로 적용.

### 4.6 파일 / 함수 크기 규칙

`~/.claude/rules/coding-style.md` (사용자 전역 규칙):
- 파일: 200~400 권장, 800 상한
- 함수: <50 라인 권장
- 중첩: ≤3 레벨 권장, 4 상한

**현재 초과 중 (Sprint 4d 종료 시점)** — §3 P3 tech debt 표 참조:
- `handlers_node_read.go` 844 라인 — 5번째 read handler 추가 시 분할
- `handlers_node_tx.go` 953 라인 — 다음 핸들러 추가 시 분할
- 핸들러 closure 길이 ~120-200 라인 — phase 별 helper 추출 권장 (Sprint 4 시리즈 전체 패턴)

이전 정리: Sprint 4b 에서 `handlers.go` 1046 → 5 파일로 분할 (handlers.go / handlers_network.go / handlers_node_lifecycle.go / handlers_node_read.go / handlers_node_tx.go).

**Sprint 5c.1 신규 mcp-server 파일 (2026-04-28)** — 모두 권장치 (200~400) 이내:
- `mcp-server/src/utils/wire.ts` 173 lines (Task 1) — spawn + NDJSON 스트림 파서
- `mcp-server/src/utils/wireResult.ts` 65 lines (Task 2) — NDJSON → MCP response transformer
- `mcp-server/src/tools/chain.ts` 249 lines (Task 3+4) — `chainbench_account_state` + `chainbench_tx_send` 등록 + handler

기존 큰 파일 (`tools/test.ts` 335, `tools/schema.ts` 308) 은 권장치 이내 (200~400) 이지만 5c.3 reroute 시 자연스럽게 흡수 후 재평가.

**Sprint 5c.2 mcp-server 파일 변화 (2026-04-29)** — chain.ts 가 read/tx 로 분리됨:
- `mcp-server/src/tools/chain_tx.ts` 505 lines (Task 4+5+6 누적) — 권장치 초과, 800 ceiling 여유 (5c.3 reroute 시 per-tool 분할 검토)
- `mcp-server/src/tools/chain_read.ts` 398 lines (Task 0+1+2+3 누적) — 권장치 이내
- `mcp-server/src/tools/chain.ts` 18 lines (Task 0 split 후 orchestrator only)

**Sprint 5c.3 mcp-server 파일 변화 (2026-04-29)** — utils 추출 + node.ts reroute + integration test layer:
- `mcp-server/src/tools/node.ts` 165 lines (Task 0+2+3+fix 누적) — 권장치 이내
- `mcp-server/src/utils/mcpResp.ts` 24 lines (Task 0) — errorResp + FormattedToolResponse SSoT
- `mcp-server/src/utils/hex.ts` 22 lines (Task 0) — HEX_*, SIGNER_ALIAS, RPC_METHOD 상수 SSoT
- `mcp-server/src/tools/chain_read.ts` 374 lines (Task 0 refactor 후) — 권장치 이내
- `mcp-server/src/tools/chain_tx.ts` 488 lines (Task 0 refactor 후) — 권장치 초과, 5c.4+ reroute 시 per-tool 분할 검토
- `mcp-server/test/integration/node_rpc.integration.test.ts` 238 lines (Task 4) — 신규 integration test layer (real-binary spawn)

---

## 5. 주의사항 (trap & gotcha)

### 5.1 Stale LSP diagnostics

**현상**: 서브에이전트가 커밋한 직후 LSP 가 아직 인덱싱 안해서
`undefined: SomeNewFunc` 에러를 report. 실제는 녹색.

**확인 방법**: `go test ./path/... -count=1` 실행해서 통과하면 무시.

**언제 빈번한가**: 새 심볼(함수, 타입) 도입 직후. 기존 심볼 수정은 드묾.

### 5.2 `types.Auth` 는 tagged union 이 아님

`network/schema/network.json` 의 `Auth` 는 oneOf/discriminator 구조지만
go-jsonschema 가 tagged union 을 생성하지 않고 `map[string]interface{}` 로 fallback.

영향:
- `AuthFromNode` 는 `auth["type"].(string)` 로 분기
- 태그된 union 으로 리팩터하려면 generator 설정 변경 또는 hand-written
- Sprint 4b 이후 고려 (현재는 현실적 타협)

### 5.3 go-stablenet ethclient 호환

go-stablenet fork 의 module path 도 `github.com/ethereum/go-ethereum` 이므로
한 바이너리에 둘 다 import 불가.

**결정 (Sprint 3b.2a 시)**: 업스트림 `github.com/ethereum/go-ethereum` 사용. go-stablenet 노드는 표준 eth_* JSON-RPC 제공하므로 호환. 체인 특화 RPC (`istanbul_*` 등) 는 raw `rpc.Client.Call` 로 (Sprint 3a probe 패턴).

### 5.4 bash 3.2 vs 4+

macOS default bash 는 3.2. 일부 테스트가 4+ 기능 사용.

**기존 처리**: `tests/unit/run.sh` 가 `BASH_VERSINFO[0] < 4` 이면
`/opt/homebrew/bin/bash` 로 re-exec (Sprint 2b 에서 추가).

새 bash 테스트 작성 시 가급적 bash 3.2 safe 하게 (associative array 금지, `&>` 금지, `${var^^}` 금지).

### 5.5 commit-hook 관련

`.git/index.lock` 이 orphan 으로 남는 경우가 간혹 발생. 해결:
```bash
lsof .git/index.lock 2>&1 | head -3   # 프로세스 확인
# 비어있으면 orphan 이므로 안전하게 제거
rm -f .git/index.lock
```

### 5.6 go generate 실행

`network/internal/types/doc.go` 가 `//go:generate` 지시어를 가짐.
스키마 (`network/schema/*.json`) 수정 시:
```bash
cd network && go generate ./...
```
→ `command_gen.go` 등 재생성. 생성 결과는 반드시 커밋.

### 5.7 go-ethereum 버전 이슈

v1.17.2 는 무거운 transitive deps (zkvm_runtime, gnark-crypto, blst, kzg) 를 끌고 옴 — 정상이며 `go mod tidy` 로 관리. CGO 요구 없음 (pure-Go fallback 있음).

### 5.8 세션 격리 주의

**`memory/` 디렉토리에 프로젝트별 follow-up 저장 금지** — 여러 프로젝트가
memory 충돌 발생. 세션 간 이어지는 정보는 **이 문서 (`docs/NEXT_WORK.md`)
또는 spec/plan 에 저장**.

---

## 6. 새 세션에서 작업 재개 가이드

### 6.1 첫 단계 (모든 새 세션에서)

```bash
# 1. 현재 상태 확인
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git log --oneline -15
git status

# 2. 테스트 전부 녹색인지 확인
go -C network test ./... -count=1 -timeout=60s
bash tests/unit/run.sh
```

### 6.2 로드맵 확인

```bash
# 현재 진행 상태
cat docs/VISION_AND_ROADMAP.md | grep -E "^\s*- \[" | head -40
```

### 6.3 다음 작업 결정

이 문서의 §3 우선순위별로 선택. 명시적 사용자 지시 없으면 P1/P2 순으로.

### 6.4 Sprint 진행 플로우

사용자가 "Sprint X 진행" 하면:

1. VISION_AND_ROADMAP.md 해당 Sprint 섹션 + 관련 §5 참조
2. spec 작성 (`docs/superpowers/specs/YYYY-MM-DD-<topic>.md`)
3. plan 작성 (`docs/superpowers/plans/...`)
4. spec+plan 커밋 후 사용자 확인 (선택)
5. subagent-driven-development 로 실행:
   - task 마다 fresh `general-purpose` subagent → 구현
   - task 완료 후 `superpowers:code-reviewer` subagent → 리뷰
   - Important 이상 지적은 즉시 fix (별도 commit)
   - Minor 는 session-local 추적 (이 문서에 기록)
6. 전체 완료 시 final review
7. 로드맵 체크박스 갱신 + 이 문서의 Follow-up 섹션 업데이트

### 6.5 인용 키 파일

**반드시 읽기 (작업 시작 전)**:
- `docs/VISION_AND_ROADMAP.md` — 전체 방향 (SSoT)
- `docs/NEXT_WORK.md` — 이 문서

**sprint 별 상세**:
- `docs/superpowers/specs/2026-04-27-sprint-4b-keystore-1559-tx-wait.md` (최근)
- `docs/superpowers/plans/2026-04-27-sprint-4b-keystore-1559-tx-wait.md` (최근)
- `docs/superpowers/specs/2026-04-24-sprint-4-signer-tx-send.md` (직전)
- `docs/superpowers/plans/2026-04-24-sprint-4-signer-tx-send.md` (직전)
- 역사적 맥락 필요하면: `docs/superpowers/specs/2026-04-23-*.md` 전체

**코드 규약**:
- `~/.claude/rules/coding-style.md`
- `~/.claude/rules/security.md` (signer 등 보안 경계)

**보안**:
- `docs/SECURITY_KEY_HANDLING.md`

**기존 모듈별 진입점** (3분 읽기로 모듈 이해 가능):
- `network/internal/signer/signer.go` — 서명 경계 참조 구현
- `network/cmd/chainbench-net/handlers.go` — 핸들러 패턴 (resolveNode, dialNode, 에러 분류)
- `network/internal/drivers/remote/client.go` — ethclient wrapping 패턴
- `lib/network_client.sh` — bash → Go 호출 브리지

### 6.6 자주 쓰는 커맨드 모음

```bash
# Go 테스트 + 린트
go -C network test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/

# 특정 패키지만
go -C network test ./internal/signer/... -v -count=1

# 커버리지
go -C network test ./internal/probe/... -count=1 -cover

# 스키마 재생성 (스키마 변경 시 필수)
cd network && go generate ./...

# Bash 테스트
bash tests/unit/run.sh
bash tests/unit/tests/security-key-boundary.sh   # 단일 테스트

# 빌드
cd network && go build -o bin/chainbench-net ./cmd/chainbench-net

# Lint 오류 한 방 확인 (go vet + gofmt 이후)
go -C network vet ./... && gofmt -l network/ && echo "CLEAN"
```

---

## 7. 맺음

Sprint 2a ~ Sprint 4 총 10개 서브스프린트 누적 결과:
- Go 15 packages, bash 27 tests 전부 green
- Network abstraction 이 wire protocol 을 통해 MCP / bash CLI / 미래의 다른
  클라이언트 모두에게 동일한 표면 노출
- remote RPC 읽기/쓰기 전부 가능 (auth 포함, 서명 포함)
- 보안 경계 (key redaction) 확립 + 자동화된 회귀 방지
- 체인 어댑터 Go 포팅 첫 단계 (stablenet complete, wbft/wemix skeleton)

**다음 자연스러운 확장**: Sprint 4b (keystore + EIP-1559 + tx.wait) 또는
Sprint 5 (capability + SSH + MCP 이관). 사용자 선호에 따라 진행.

---

**이 문서가 최신 상태인지 확인하는 방법**: `git log -- docs/NEXT_WORK.md` 의
마지막 커밋 시점이 HEAD 와 근접한지. Sprint 완료마다 본 문서의 §2 (완료)
/ §3 (다음 작업) / §4.6 (tech debt 표) 업데이트 필수.
