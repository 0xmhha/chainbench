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
| `internal/core/state`(nodeset.json) | 데이터루트 상태 | `internal/core/session`(env.json) | REFACTOR |
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
