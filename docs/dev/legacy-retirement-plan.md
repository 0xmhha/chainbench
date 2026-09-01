# 레거시 경로 은퇴 계획 (legacy retirement)

> **[이력]** 레거시 은퇴 계획. 진행은 worklist.
> **현재 상태를 말하지 않는다.** 그때 무엇을 측정·결정했는지의 기록이다.
> 현재 상태는 [[chainbench-worklist]] 와 코드가 정본이다.

> 목적: 재설계 엔진(`internal/engine`) + DSL(`internal/testspec`)이 완성되어 `chainbench run`
> 으로 도달 가능해진 지금, **레거시 Go-func 테스트 경로**(`internal/testkit` +
> `internal/core/pipeline/testrun`)와 그 주변을 안전하게 은퇴시키는 순서·매핑·블로커를 확정한다.
> 근거: `chainbench-refactoring.md`(§4 REPLACE) · `chainbench-worklist.md`(레거시 정리) · 코드 감사.

---

## 1. 레거시 → 신규 매핑

| 레거시 | 역할 | 대체 | 판정 |
|---|---|---|---|
| `internal/testkit` (Case·CaseFunc·registry·T) | Go-func 테스트 케이스 등록·실행 헬퍼 | `internal/testspec`(DSL Parse/Interpreter) + `internal/engine`(RunSpec) | REPLACE |
| `internal/testkit`(Report·Result·Status) | 결과 모델 | **재사용** (신규 경로도 동일 판정 어휘) | KEEP |
| `internal/core/pipeline/testrun` (Run·Options·Report) | 3-phase test 실행기 | `engine.Run` (Parse→env→RunSpec→session) | REPLACE |
| `cmd/chainbench test` | Go-func 케이스 CLI | `cmd/chainbench run` (DSL spec CLI) | REPLACE |
| `internal/mcp`(test 도구: `tools.go`·`remote_tools.go`) | MCP 테스트 실행 | `chainbench_run`(#203, DSL) | REPLACE |
| `internal/core/pipeline/setup`(Provision·Launch·Run) | 순차 셋업 | `engine.BuildEnv`/`AssemblePlan`/`LocalLauncher` | REFACTOR(엔진은 `setup.Plan` 타입만 사용) |
| ~~`internal/core/state`(nodeset.json)~~ | 데이터루트 상태 | `session.Composition`(workspace.json) — **nodeset.json/nodespecs.json 은 P6.2(2026-08-28)에서 삭제** | DONE |
| `internal/core/probe` | RPC 프로브 | `internal/core/collector` | ABSORB |

---

## 2. 은퇴 순서 (의존 역순 — 소비자부터)

레거시 실행기(testrun/testkit)는 **아직 활발히 사용**되므로 소비자를 먼저 신규 경로로 이관해야 제거 가능하다.

1. **suite 이관 (블로커)** — `tests/`(all·anzeon·wbft·api·network·repro·external)의 Go-func 케이스를 DSL spec 으로 포팅. 이것이 완료돼야 testkit/testrun 을 제거할 수 있다.
2. **`chainbench test` 이관** — DSL 스위트가 존재하면 `test` 를 `run` 으로 대체(또는 `test` 를 DSL 실행으로 재구성) 후 레거시 test 명령 제거.
3. **MCP 도구 이관** — `tools.go`/`remote_tools.go` 의 Go-func 실행 도구를 `chainbench_run` 계열로 정리.
4. **testrun 제거** → **testkit(Case/registry) 제거** (Report/Result/Status 는 신규 경로로 이동·잔존).
5. **state → session, probe → collector 흡수** 마무리.

---

## 3. 블로커 (이 환경에서 자동 진행 불가)

| 블로커 | 영향 | 필요 |
|---|---|---|
| **DSL 스위트 부재** | testkit/testrun 제거 불가(실 테스트가 Go-func) | `tests/` 케이스의 DSL 포팅 — 원 시나리오 + 라이브 검증 |
| **체인 바이너리 부재(CI)** | 포팅한 DSL spec 라이브 검증 불가 | gstable/gwemix 등(사용자 환경) |
| **동작 동치성** | `test` → `run` 대체 시 판정·출력 차이 | 매핑 확정 + 회귀 비교 |

---

## 4. 지금 안전하게 할 수 있는 것 (착수분)

- ☑ 죽은 레거시 심볼 제거(`testkit.RunCase` 래퍼).
- ☑ 레거시 패키지 signpost(testkit·testrun package doc — 대체 경로 명시).
- ☑ 본 은퇴 계획 문서화.
- ◐ **suite 이관 착수(소스 있음)**: `tests/` Go-func 케이스는 repo 에 존재하므로 DSL 로 포팅 가능(라이브 검증만 바이너리 필요). `onEach` 다중노드 어세션 수정으로 per-node 케이스 표현 가능 → `tests/network` peers-connected 를 `examples/specs/network-peers.json` 으로 포팅(CI validate).
- ☐ (후속·비파괴) 결과 모델 단일화 검토(testkit.Report ↔ session/engine.Summary 중복 축소).
- ◐ (블로커 해소됨) 나머지 suite 이관 → test/MCP 이관 → testrun/testkit 제거. **DSL 표현력 블로커는 §4.3 대로 해소**됐고, 남은 것은 케이스별 이관 작업량 + 라이브 검증이다.

### 4.4 이관 진척 (2026-08-09)

> **2026-08-28 (P8):** 이관이 끝난 Go 케이스 파일 41개(+유닛테스트)를 삭제했다 — 등록 134 → 56건,
> `tests/api`·`tests/network` 패키지 소멸, 공유 헬퍼는 패키지별 `helpers.go` 로 취합.
> 남은 34건의 미이관 사유는 `tests/specs/README.md` 의 잔여 표가 정본이다. 순서 1(suite 이관)의
> 남은 분량이 그것이고, 2~4(`test` 명령·MCP·testrun·testkit 제거)는 그 뒤에 한 번에 한다.

레거시 등록 케이스는 **134개**(`testkit.Cases()` 실측, 7개 카테고리). 이관분은 `tests/specs/<category>/<name>.json` 에 두고 CI 가 오프라인 검증한다([[tests-specs-readme]] = `tests/specs/README.md`).

| 카테고리 | 레거시 | 이관 | 라이브 |
|---|---:|---:|---|
| api | 11 | **10** | gstable pass=10 |
| consensus | 13 | **8** | gstable pass=8 · gwbft pass=7/skip=1 |
| network | 4 | 3 | 선행 이관분(`examples/specs/network-*`) |
| accounts | 35 | 0 | — |
| gas-policy | 17 | 0 | — |
| hardfork | 8 | 0 | — |
| system-contracts | 46 | 0 | — |

**이관이 레거시의 결함을 드러냈다** — 둘 다 라이브에서만 보이는 것(레거시 유닛테스트는 mock 노드 사용):
- `wbft-extra-info-fields` 가 `istanbul_getWbftExtraInfo` 를 `"latest"` 로 호출 → 체인이 `block -2 not found` 로 거부. 구체 블록번호 필요.
- 같은 케이스가 `ChainCompat:[stablenet,wbft]` 인데 `gasTip` 은 **stablenet 전용** → wbft 응답에 없음. 이관본은 공통 필드와 체인특화 필드를 분리했다.

**이관 불가로 남긴 것**(가짜로 만들지 않음): `ws-subscribe-logs`(구독을 스텝보다 먼저 열어야 함 — 순서 표현 불가) · `block-period-one-second`(저장값 간 산술) · `epoch-transition-carries-epoch-info`(조건부 대기) · `validator-set-count`(spec 이 자기 토폴로지를 참조할 수단 없음).

### 4.1 이관 중 발견한 DSL 표현력 갭 (신규 빌트인 필요)

포팅을 진행하며 확인한, 현재 빌트인으로 표현 불가한 패턴:

| 레거시 케이스 | 필요한 빌트인 | 상태 |
|---|---|---|
| `network` peers-connected | `peerCount` + onEach | ☑ (onEach 수정, `network-peers.json`) |
| `network` block-progression | 헤드 전진 어세션 | ☑ `blockAdvance`(폴링 head 전진, `network-health.json`) |
| `network` genesis-hash-agreement | 노드 간 동일값(no-fork) 어세션 | ☑ `sameBlockHash`(block:"0x0" onEach, `assert.HashesEqual`, `network-health.json`) |

> `tests/network` 3케이스 전부 DSL 로 표현·포팅 완료(CI validate). 다음: `tests/anzeon` 등 체인 특화 케이스 이관 시 필요한 빌트인 식별·추가.

### 4.2 `tests/anzeon`(stablenet) 이관 커버리지

| 분류 | 케이스 | DSL 표현 | 상태 |
|---|---|---|---|
| 시스템 컨트랙트 코드 | native-coin-adapter-code | `codeAt` + `NotEqual "0x"` | ☑ (`stablenet-system-contracts.json`) |
| getter readable | token-balance-readable·account-authorization-readable | `call` + `Regexp`(반환 shape) | ☑ (동 spec) |
| 하드포크 아티팩트(plain read) | p256-precompile-active/rejects-invalid·govminter-v2-code·boho-chain-config-active | `call`(precompile)·`codeAt`·`chainId`/`blockNumber` | ☑ (`stablenet-hardfork.json`) |
| getter 값 대조 | token-metadata(name/symbol) | `call` + `Contains`(심볼 바이트) | ☑ (`stablenet-token-metadata.json`) |
| **교차-call 비교** | totalSupply ≥ balance 등 | `read`+`save` → `$ref` 비교 | ☑ (`stablenet-token-invariants.json`) |
> 결론: **read-기반 anzeon 케이스(시스템 컨트랙트·getter·base fee·하드포크·estimate-gas·token-metadata)는 포팅 완료**(신규 리드 빌트인 `baseFee`/`estimateGas` + 기존 빌트인). **잔여 anzeon 은 read 1-shot 로 표현 불가**한 범주만 남음:

### 4.3 잔여 anzeon — **표현력 블로커 해소됨** (2026-08-09)

§4.3 이 열거했던 6개 범주는 **전부 표현 불가에서 표현 가능으로 바뀌었다**. 각각이 요구하던 "대형 DSL 기능"이 도입됐다:

| 범주 | 요구했던 기능 | 도입 | 상태 |
|---|---|---|---|
| 교차-call 비교 | 스텝 값 바인딩 | **`save` + `$ref`**(design §3.2b) + `read` 액션 | ☑ |
| 거버넌스 다단계 | 스텝 바인딩 + 이벤트 어세션 | 위 + **`logs`**(eth_getLogs, `select` count/data/topicN) | ☑ |
| 가스 파생(tip) | 체인특화 gasTip + 조합 | **`gasPrice`** + 범용 **`rpcCall`**(method+params+dot-path select) | ☑ |
| fee-cap tx | tx 인자 확장 | `sendTx`/`faucet`/`deploy…` 의 **`maxFeePerGas`·`maxPriorityFeePerGas`·`gasPrice`** | ☑ |
| 명시 nonce | 명시-nonce 전송 | `sendTx` 의 **`nonce`** 인자 | ☑ |
| WS 구독 | WS 전송 | **`wsSubscribe`**(`rpc.Subscribe` 경유, `ws` capability) | ☑ |

**설계 원칙(중요):** `rpcCall` 은 체인 어휘를 **core 가 아니라 spec 안에** 둔다 — `istanbul_getWbftExtraInfo`·`wemix_*` 는 정의서가 문자열로 지정하고, 하네스는 체인을 계속 모른다(C6 ACL 유지).

**남은 것은 표현력이 아니라 이관 작업량이다.** 각 anzeon/wbft/api 케이스를 실제 spec 으로 옮기고 라이브 검증하는 일이 남았다. 라이브 근거: `TestEngine_Live_NewVocabulary`(GSTABLE_BIN) 가 실 4노드 stablenet 에서 바인딩·`read`·`faucet`·`logs`·`gasPrice`·`rpcCall`·`wsSubscribe`·`stopNode`/`startNode` 를 한 세션으로 통과.

> 원칙: **소비자 이관 전에는 레거시 제거 금지**(회귀 위험). 각 단계는 비-e2e 통과 + 대표 라이브 확인을 게이트로.

---

## 5. 잔여 미이관 14건 — 정본 (2026-09-01, R5 이관 종료 시점)

등록 레거시 56건 중 **42건이 동명 DSL spec 으로 이관**됐고, 남은 **14건은 전부
이관 불가 부류**다. 아래가 그 14건의 정본이다 — 카테고리, 소스 파일, 사유,
은퇴 시 커버리지 손실. testkit/testrun 이 은퇴하면 이 14건의 Go-func 커버리지가
사라지므로, 각 건의 손실 크기를 여기 못박는다.

세 갈래로 나뉜다. **A) 설계 경계**(DSL 이 의도적으로 표현하지 않음 — 영구),
**B) 외부 블로커**(바이너리·env 부재 — 확보 시 재분류), **C) 라이브 반증**(레거시
전제가 실제 바이너리와 불일치 — 이관해도 무의미).

| # | 카테고리 | 케이스 | 소스(`tests/`) | 갈래 | 사유 · 손실 |
|---|---|---|---|---|---|
| 1 | consensus | `validator-set-count` | `wbft/consensus/validators.go` | A | spec 이 **자기 토폴로지**(validator 노드 수)를 참조해 비교. DSL 은 spec 이 자기 규모를 모르게 설계(이식성). 손실: "검증자셋이 실행 노드 전부를 덮는가" 불변식. `istanbul_getValidators` 길이 ≥1 은 다른 spec 이 커버 |
| 2 | consensus | `prev-seals-quorum` | `wbft/consensus/seals_quorum.go` | A | 위 + **토폴로지 파생 quorum**(ceil 2N/3) 산술. 손실: prev-seal 의 sealer 수 ≥ quorum. seal **존재**는 `wbft-seals-quorum`(이관됨)이 커버 |
| 3 | consensus | `epoch-transition-carries-epoch-info` | `wbft/consensus/epoch.go` | A | 에폭 경계까지 **조건부 대기** 후 그 블록 조회. DSL 에 "N 의 배수 블록까지 대기" 표현이 없다. 손실: 에폭 전이 블록의 epoch 정보 |
| 4 | api | `ws-subscribe-logs` | `anzeon/ws_subscribe.go` | A | **구독을 유발 tx 보다 먼저** 열어야 하는데 어세션은 스텝 뒤에 돈다. 순서 표현 불가. 손실: 로그 구독 스트림(heads 구독 `ws-subscribe-new-heads` 는 이관됨) |
| 5 | accounts | `zero-address-transfer-blocked` | `wbft/accounts/value_guard.go` | A | accounts **SDK 클라이언트측 정적 가드**(제출 전 거부)를 검사. DSL sendTx 는 노드로 직행해 이 가드를 안 태운다 — 의미가 다르다. 손실: SDK 가드(노드 거부 아님) |
| 6 | accounts | `precompile-transfer-blocked` | `wbft/accounts/value_guard.go` | A | 위와 동일(precompile 주소로의 전송을 SDK 가 막음) |
| 7 | accounts | `external-value-transfer` | `external/write.go` | B | **operator 공급 키**(`CHAINBENCH_FUNDED_KEY`)로 외부(chainbench 미구성) 체인에 전송. suite 는 자기 넷만 구성. 손실: 외부 체인 스모크 |
| 8 | accounts | `external-fee-delegated-transfer` | `external/write.go` | B | 위 + 0x16. 손실: 외부 체인 fee-delegation 스모크 |
| 9 | accounts | `set-code-delegation` | `wbft/accounts/set_code_delegation.go` | B | **EIP-7702(0x04)** set-code — authorizationList·authority 서명 프리미티브 미구현. 손실: 0x04 경로. (프리미티브를 만들면 이관 가능 — 후속) |
| 10 | gas-policy | `tipcap-underpriced-rejected` | `anzeon/tx_rejections.go` | C | **라이브 반증**(결함 표): 이 넷에서 재현 불안정 → 이관 보류. 손실: tipCap 미달 거부(다른 제출거부 4건이 인접 커버) |
| 11 | hardfork | `p256-precompile-active` | `anzeon/hardfork_reads.go` | C | **라이브 반증**: 이 gstable 빌드에 P256VERIFY(0x100) 미탑재 — valid 벡터가 `"0x"` 반환. 손실: P256 활성(바이너리 확보 시 재분류) |
| 12 | hardfork | `p256-rejects-invalid` | `anzeon/hardfork_reads.go` | C | 위와 동일 — precompile 부재로 "거부"가 우연히 성립할 뿐 검증 의미 없음 |
| 13 | hardfork | `p256-inactive-before-boho` | `anzeon/fork_transition.go` | C | 위 P256 부재 + delayed-boho. 포크 후에도 "0x" 라 "활성" 절반 성립 불가 |
| 14 | hardfork | `govminter-code-changes-at-boho` | `anzeon/fork_transition.go` | C | **라이브 반증**: chainbench genesis 빌더가 bohoBlock=10 이어도 GovMinter 코드를 처음부터 최종본으로 굽는다(블록 1·latest getCode md5 동일, head 0x20 실측). "v1→v2 스왑" 신호 자체가 없다 |

**갈래별 은퇴 판단:**
- **A(6건)**: 영구 미이관. DSL 의 의도적 경계이거나 대상이 SDK/순서라 표현 불가.
  은퇴해도 재구현 대상 아님. 인접 spec 이 핵심 불변식을 부분 커버(2·4).
- **B(3건)**: 프리미티브·바이너리·env 확보 시 이관 가능. 9(0x04)는 프리미티브만
  만들면 되므로 후속 1순위. 7·8 은 외부 체인 전용이라 CI 스모크에서 빠져도 무방.
- **C(4건)**: 레거시 전제가 **실제 바이너리와 불일치**. 이관해도 통과가 무의미하거나
  라이브에서 fail. Boho/P256 충실 바이너리·genesis 소스 확보 후 재평가.

**결론:** 14건 중 **재구현 가치가 있는 것은 B 의 9(set-code 0x04) 하나**뿐이고,
나머지 13건은 설계 경계·외부 블로커·라이브 반증이라 은퇴로 잃는 실효 커버리지가 작다.
testkit·testrun·`chainbench test`·MCP `chainbench_test` 은퇴를 진행해도 되며, 9 는
후속 프리미티브(0x04)로 이관, C(4건)는 바이너리 확보 시 재평가한다.
