# 레거시 경로 은퇴 계획 (legacy retirement)

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
- ☐ (후속·비파괴) 결과 모델 단일화 검토(testkit.Report ↔ session/engine.Summary 중복 축소).
- ☐ (후속·블로커 해소 후) suite 이관 → test/MCP 이관 → testrun/testkit 제거.

> 원칙: **소비자 이관 전에는 레거시 제거 금지**(회귀 위험). 각 단계는 비-e2e 통과 + 대표 라이브 확인을 게이트로.
