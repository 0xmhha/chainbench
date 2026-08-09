# chainbench 리팩토링 진행 감사 (2026-08-09) — **시점 스냅샷 (#204 기준)**

> **지위: 특정 시점의 스냅샷이며 진행 상황의 정본이 아니다.** 진행 상황·작업 상태의 **단일 정본은
> [[chainbench-worklist]]**(`docs/dev/chainbench-worklist.md`)이며, 본 문서는 그 시점에 수행한
> 감사 절차와 근거를 보존한다. 본문 §3·§6은 **#204 시점 기준**이고, 그 이후 변경은 **§7 후속 정정**에 기록한다.
>
> 목적: 문서·전 코드를 감사해 (1) 프로젝트 방향, (2) 진행 중 리팩토링과 진척도,
> (3) 완료분의 에러 유무, (4) 미추적 갭의 작업 리스트 반영, (5) 남은 작업을 확정한다.
> 근거: `chainbench-design.md`·`chainbench-refactoring.md`(WP1~6)·`chainbench-worklist.md`(T0~T6)
> · git log(#178~#203) · `go build/vet/test ./...` 실행 결과.

---

## 1. 결론

- **완료된 리팩토링 작업에 코드 에러 없음.** `go build ./...`·`go vet ./...`·`go test ./...`
  전부 통과(EXIT 0), 비-테스트 소스에 `TODO/FIXME/stub/미구현` 마커 0개. 라이브 e2e만
  바이너리(`GSTABLE_BIN` 등) 부재로 CI에서 clean-skip — 의도된 게이팅이며 worklist에 명시됨.
- **미추적 갭 단 1건 발견 → 작업 리스트에 반영 완료.** T0.0(`pkg/`→`internal/`) 마이그레이션
  후에도 forward-looking 문서 2개가 삭제된 `pkg/` 경로를 참조. 어느 리스트에도 없어
  **worklist에 T6.5로 추가**함.
- **현재 방향:** 재설계 엔진(`internal/engine`)으로 레거시 `pipeline/testrun`+`testkit`
  Go-func 테스트 모델을 DSL(`testspec`) 기반으로 교체하는 WP1~6 리팩토링.
  **Phase 0~4(walking skeleton) 완료, Phase 5~6 진행 중.**

권장 행동(**#204 시점** — 1·2는 이후 완료됨, §7 참조):
1. T6.5(문서 경로 정정) 착수 — 후속 마이그레이션 작업 전 혼선 제거.
2. 다음 크리티컬 패스: **T6.3 dashboard(엔진 obs 이벤트 emit)** + **T5.1 remote 실 SSH 라이브 e2e**
   (코드 완료, 사용자 환경 검증만 남음).
3. wemix4 full 이관은 **단계 A(handoff `--genesis-overlay` 주입)** 부터 착수.

---

## 2. 프로젝트 XP (x-bar 분해) — 방향

| 통사 역할 | 대상 | 의미 |
|---|---|---|
| **Head (핵)** | `internal/engine` 오케스트레이터 | `Parse→Applicable skip→env 재사용/BuildEnv→RunSpec→session 저장→Teardown` 조립. 모든 진입점이 이 핵을 호출 |
| **Specifier (지정어)** | `chainbench run --rpc/--binary` CLI · MCP · dashboard | 핵을 활성화하는 표면. `run` 명령이 재설계 엔진에 최초 도달 |
| **Complement (보충어)** | `testspec` DSL(정의서) | 핵이 필수로 요구하는 논항 — 테스트를 Go 함수가 아닌 선언적 spec으로 |
| **Adjunct (부가어)** | remote · attach · upgrade · capabilities | 핵을 바꾸지 않고 합성되는 수직 슬라이스 |

핵심 명제: **한 공유 코어 위에서 multi-chain·multi-node 네트워크를 Docker 없이 기동·검증·테스트.**
새 체인 추가는 manifest + thin plugin의 data-only.

---

## 3. 진행 중 리팩토링 (WP1~6) — 진척도

| Phase | 범위 | 상태 | 근거(git/worklist) |
|---|---|---|---|
| 0 | `pkg/`→`internal/` + 인터페이스 동결 | 완료 | T0.0·T0.1 |
| 1 | testspec·assert·session-path·place·procman·keyreg | 완료 | #178~183 |
| 2 | Transport(key_file 인증·kill 검증·upload-if-absent) | 진행 | #199. 남음: driver 위 Transport 타입 형식화 |
| 3 | Provisioner·Supervisor·Collector | 진행 | #184~186. T3.1·T3.2·T3.4 완료, T3.3 collector 심화(live tail·reorg)만 남음 |
| 4 | ★ Engine walking skeleton | 완료 | #191~202. 실 gstable 4노드 라이브 e2e 통과(BuildEnv·RunSpec·FullRun capstone) |
| 5 | remote·upgrade·attach·stablenet·wemix4 이관 | 부분 | T5.1 RemoteFileSink 완료/실 SSH 라이브 미검증, T5.3 attach 완료, T5.2·5.4·5.5 미착수 |
| 6 | capability·MCP·dashboard·백프레셔·CLI | 부분 | #202~203. T6.1·T6.2 MCP·CLI run·-race·백프레셔 완료, T6.3 dashboard emit만 남음 |

---

## 4. 완료분 무결성 검증 (에러 체크)

| 검사 | 결과 |
|---|---|
| `go build ./...` | 결함 없음 (EXIT 0) |
| `go vet ./...` | 결함 없음 (clean) |
| `go test ./...` | 결함 없음 (전 패키지 PASS; 라이브 e2e는 바이너리 부재로 의도적 SKIP) |
| stub/미구현 마커 | 결함 없음 (`registry.go:89` panic은 invariant guard, stub 아님) |

라이브 SKIP 목록(정상): `TestBuildEnv_Live_Stablenet`, `TestEngine_Live_FullRun`,
`TestPresetGenesisSource_Live_GstableInit`, `TestRunSpec_Live_Stablenet`(모두 `GSTABLE_BIN` 게이트).

---

## 5. 미추적 갭 → 작업 리스트 반영

| 갭 | 기존 추적 여부 | 조치 |
|---|---|---|
| 라이브 e2e 사용자 환경 이월 | 추적됨(worklist T4.1~4.4·T5.1) | 유지 |
| repro 5개 bash 미이관 | 추적됨(repro-migration-remaining.md) | 유지 |
| wemix4 full 이관(useNCP·stateful) | 추적됨(wemix4-migration-plan.md) | 유지 |
| **문서 `pkg/` 경로 드리프트** | **미추적** | **worklist T6.5 추가** |

---

## 6. 남은 작업 리스트 (**#204 시점** — 이후 변경은 §7)

1. **T6.3** dashboard(F15) — 엔진 obs 이벤트 emit → chainbenchd 포워딩(버스·백프레셔는 완료, 엔진 emit만 남음), 크리티컬 패스
2. **T3.3** Collector 심화 — live tail·원격 SSH tail·bp참여·reorg 검출(현재 RPC 스냅샷·WaitLog)
3. **T5.1** remote 실 SSH 호스트 라이브 e2e — 코드 완료, 사용자 환경 검증만
4. **T5.2** 업그레이드 멀티바이너리(wemix+wbft) — 미착수
5. **T5.4** stablenet(ACL 플러그인) → **T5.5** wemix4 DSL 이관 — 미착수
6. **T6.5** 문서 경로 정정 — 신규 추가
7. 레거시 경로 정리(`pipeline/setup`·`testrun` 완전 대체) / repro 잔여 5건 / wemix4 full 이관 — 각 별도 트랙

---

## 7. 후속 정정 (스냅샷 이후 · #205~#224)

본 감사 직후 같은 날 #205~#224가 병합되어 §3·§6의 상당수가 무효화됐다. 정본은 worklist이며,
아래는 스냅샷 독자가 오판하지 않도록 남기는 델타다.

| §6 항목 | #224 시점 실제 | 근거 |
|---|---|---|
| 1. T6.3 dashboard | **완료** — 엔진 obs emit + `run --dashboard` 포워딩 + 완료 세션 디스크 조회(F15 AC3) | #205 · #209 |
| 2. T3.3 Collector 심화 | **거의 완료** — 로컬 live tail·bp참여·fork/reorg·엔진 배선·chainstate jsonl 영속. 원격 SSH tail만 잔여 | #206 · #207 · #208 |
| 4. T5.2 업그레이드 멀티바이너리 | 미착수 유지 (선행: supervisor `LeaderGate`/`ForkSwaps` 실구현) | — |
| 5. T5.4 stablenet | **완료** — 거버넌스 시나리오가 DSL 엔진에서 실행됨(CI) | #215 |
| 5. T5.5 wemix4 이관 | **부분 착수** — `tests/network` 3케이스 전부 + `tests/anzeon` read 계열 포팅 | #218~#224 |
| 6. T6.5 문서 경로 정정 | **완료**, 단 스코프가 2문서 6건 → **7문서 20건**으로 확대 | x-bar 정렬 검토 |

### 7.1 이 스냅샷이 놓친 것 (x-bar 정렬 검토에서 추가 발견)

| 갭 | 성격 | 조치 |
|---|---|---|
| `supervisor.Options`의 `LeaderGate`·`AlignJoinGap`·`ForkSwaps` 를 `supervisor_impl` 이 **읽지 않음**; `FailureMode` 5종 중 `RPCUnready` 만 방출 | **선언되고 방출되지 않는 논항** — F13 AC-1·2·3, F9 AC-3 미충족 | worklist **T3.2b** 신규 |
| DSL 액션이 `sendTx`/`waitBlock` **2개뿐** — design §3.2·F11이 명시한 `stopNode`/`startNode`/`restartNode`/`partition`/`healPartition`/`faucet`/`deployContract`/`registerContract` 부재 | **보충어 어휘 갭** — F8 AC-2, F11 AC-6·7 검증 불가 | worklist **T4.2b·T4.2c** 신규 |
| `component-architecture` §2b 실측 매트릭스의 **판정**이 낡음(place 없음·procman 미배선 등 이미 해소) | 경로가 아니라 판정 드리프트 | worklist **T6.5b** 신규 |

> 본 감사는 "완료분의 무결성"(§4)에는 정확했으나, **인터페이스가 선언한 계약이 구현에서 실제로
> 방출되는지**는 검사 범위 밖이었다. `go build/vet/test` 통과는 그 갭을 잡지 못한다 — 미사용 필드는
> 컴파일 에러가 아니기 때문이다. 향후 감사는 **선언 ↔ 방출 대조**를 절차에 포함해야 한다.
