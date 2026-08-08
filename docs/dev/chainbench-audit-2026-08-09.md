# chainbench 리팩토링 진행 감사 (2026-08-09)

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

권장 행동:
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

## 6. 남은 작업 리스트 (정리)

1. **T6.3** dashboard(F15) — 엔진 obs 이벤트 emit → chainbenchd 포워딩(버스·백프레셔는 완료, 엔진 emit만 남음), 크리티컬 패스
2. **T3.3** Collector 심화 — live tail·원격 SSH tail·bp참여·reorg 검출(현재 RPC 스냅샷·WaitLog)
3. **T5.1** remote 실 SSH 호스트 라이브 e2e — 코드 완료, 사용자 환경 검증만
4. **T5.2** 업그레이드 멀티바이너리(wemix+wbft) — 미착수
5. **T5.4** stablenet(ACL 플러그인) → **T5.5** wemix4 DSL 이관 — 미착수
6. **T6.5** 문서 경로 정정 — 신규 추가
7. 레거시 경로 정리(`pipeline/setup`·`testrun` 완전 대체) / repro 잔여 5건 / wemix4 full 이관 — 각 별도 트랙
