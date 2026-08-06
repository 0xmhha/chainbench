# chainbench 구현 작업 트래커 — 작업 리스트 · 우선순위 · 폴더 트리 예상도

> 근거: `chainbench-component-architecture.md`(§2b 실측·§3 컴포넌트·§5 Phase·§1b DDD) · `chainbench-design.md`(§3 인터페이스) ·
> `chainbench-feature-spec.md`(F1~F16 AC) · `chainbench-refactoring.md`(WP1~6).
> 원칙: **Low는 TDD 먼저 → walking skeleton으로 조기 통합 → 수직 슬라이스 확장**(big-bang 금지). 코드는 [[go-code-quality-guidelines]] 준수.
> 상태 표기: ☐ 미착수 · ◐ 진행 · ☑ 완료.

---

## 1. 개발 우선순위 (walking skeleton 최단 도달 → 심화)

크리티컬 패스 = 스켈레톤에 필요한 각 컴포넌트의 **얇은 버전**부터.

| 순위 | 작업 | 이유 | 상태 |
|------|------|------|------|
| 0 | **`pkg/` → `internal/` 마이그레이션** | 앱은 internal이 정석(외부 importer 0, 컴파일러 강제 캡슐화); 신규 패키지 전 정리 | ☑ |
| 1 | **P0 인터페이스 동결** | 전 작업 언블록·병렬 TDD 개방 | ☑ |
| 2 | **Session(M1)** | 모든 기록의 정본, 기반 | ◐ |
| 3 | **Place(M2)+용량검증** | 배치·포트(고정포트 충돌 제거) | ☑ |
| 4 | **procman 배선+확장(M6 핵심)** | 최우선 안전 갭(검증 없는 Kill→고아 위험) | ◐ |
| 5 | **testspec Parse+Fingerprint** | spec 실행·env 재사용 | ☐ |
| 6 | **keyreg(M3) + assert funcs** | 신원·검증 | ☐ |
| 7 | **Provisioner(M5) → Supervisor(M6)** | 물질화·기동·teardown | ☐ |
| 8 | **Collector RPC-min + Interpreter-min** | 스켈레톤 최소 관측·실행 | ☐ |
| 9 | **★ Engine walking skeleton** | 첫 통합 증명(1체인·local·4노드·tx1) | ☐ |
| 10+ | Collector 심화 → remote → 업그레이드 → attach → stablenet → wemix4 이관 → 표면 | 수직 슬라이스 확장 | ☐ |

---

## 2. 전체 작업 리스트 (Phase · Task)

### Phase 0 — 레이아웃 정리 + 인터페이스 동결
- ☑ **T0.0 `pkg/` → `internal/` 마이그레이션** 기계적 rename: 디렉토리 이동 + import 경로 `github.com/0xmhha/chainbench/pkg/…` → `…/internal/…`(cmd/ 포함 전 소스). **게이트**: `go build ./...` + `go test ./...` 통과, 동작 변화 0. (외부 importer 없음 → 파괴 없음.) 공개 API가 필요하면 그 slice만 `pkg/`에 남김.
- ☑ **T0.1** Transport·Allocator(+Capacity)·KeyRegistry(+BLSDeriver)·GenesisBuilder·Provisioner·Supervisor·Collector·Session·Interpreter·Capabilities를 **컴파일되는 Go stub**으로(design §3과 1:1, `internal/` 하위). enum은 typed const(FailureMode/TestStatus/Mode/Source). **게이트**: 컴파일 + 각 인터페이스↔F(AC) 매핑 표.

### Phase 1 — Low atomic (TDD 먼저, 병렬 가능)
- ☐ **T1.1 testspec** Parse+필수검증(schemaVersion 등)·**Fingerprint(canonical·키정렬→결정성)**·`Get(dotPath)`·`,`파서. 순수. — F3·F7
- ☐ **T1.2 assert funcs** 타입인지 비교(Equal/Len/EqualHashAt/EqualCI/InDelta …). 순수. — F6
- ☑ **T1.3 session-path** 결정적 경로·env-id(`env-`+12hex)·레이아웃. — F1
- ☑ **T1.4 place** portplan(로컬 스텝/OS)+원격 동일포트 **통합 Allocator** + **용량검증(min≥4·max=서버×포트)**. — F12
- ☑ **T1.5 procman EXTEND** `{PID,datadir}`·원격PID·`Alive`. (stop 경로 배선은 T3.2) — F13
- ☐ **T1.6 keyreg** 랜덤/기존/원격다운로드 통합 + **BLSDeriver seam**(외부 bootnode 캡슐화, 부재 시 오류). — F2
- **게이트**: 단위 100% + 동시 모듈 `-race`.

### Phase 2 — Transport
- ☐ **T2.1** Local/Remote Transport를 driver 위에 형식화 + **종료검증(`kill -0`)**(현재 Kill만) + **key_file 인증**(현재 password only).
- ☐ **T2.2 upload-if-absent** `test -f` 존재확인 → 재사용/업로드(현재 항상-업로드 or 항상-읽기).
- **게이트**: 단위(모의) + 통합 1건(로컬 더미 프로세스 검증-종료·고아0).

### Phase 3 — Middle 통합(라이브)
- ☐ **T3.1 Provisioner** datadir+키+genesis+config 물질화(local·remote 동일경로)+upload-if-absent.
- ☐ **T3.2 Supervisor** 4노드 wbft 기동·헬스게이트(블록생성·etcd 리더)·teardown(**고아0+datadir삭제**)·**procman을 실제 stop 경로에 배선**(leak=0).
- ☐ **T3.3 Collector** 로컬 **live tail**(스캔→tail)+**원격 SSH tail**+RPC 스냅샷+**bp참여·분기(reorg)검출**. attach=RPC-only.
- ☐ **T3.4 Session 저장·재사용** env.json·fingerprint 재사용·records(spec/steps/assert/status).
- **게이트**: 각 컴포넌트 통합테스트 라이브.

### Phase 4 — Walking Skeleton ★
- ☐ **T4.1 Engine** Place→KeyReg→Genesis→Provision→Supervise→Collect→Interpreter 조립 → **1체인 wbft·local·4노드·tx1** e2e·세션 아티팩트 검증.

### Phase 5 — 수직 슬라이스 (매번 통합 유지)
- ☐ **T5.1 remote**(Transport 교체만) · ☐ **T5.2 업그레이드 멀티바이너리**(wemix+wbft) · ☐ **T5.3 attach** · ☐ **T5.4 stablenet**(ACL 플러그인만·Core 무변경) · ☐ **T5.5 wemix4 이관**(DSL).

### Phase 6 — 표면·마감
- ☐ **T6.1 Capabilities**(registry/chains) · ☐ **T6.2 MCP 결과연동**(F14) · ☐ **T6.3 dashboard**(F15) · ☐ **T6.4** 백프레셔(O7)·`-race` 게이트(O6)·CLI 정리.

---

## 3. 폴더 트리 예상도 (구현 후 · `internal/` 레이아웃)

> **레이아웃 결정**: chainbench는 애플리케이션이고 `pkg/`가 자기 `cmd/`에만 쓰임(외부 importer 0) → **`internal/`이 정석**(컴파일러 강제 캡슐화). `pkg/`는 관례일 뿐이라 폐기. 진짜 공개 Go API가 생기면 그 slice만 `pkg/`로 승격.
> `[NEW]` 신규 · `[EXT]` 확장 · `[KEEP]` 유지·재사용 · `[REF]` 재구성 · `[REPL]` 교체 · `[→x]` x로 흡수/이동. C#=DDD 컨텍스트(§1b).

```
cmd/                          # 바이너리 진입점(유일하게 internal/ 밖에서 internal/ import)
  chainbench/          [EXT]   · engine 호출
  chainbench-mcp/      [KEEP]
  chainbenchd/         [KEEP]
internal/                     # 전 구현 패키지(외부 import 컴파일러 차단)
  engine/              [NEW]   H1 · 오케스트레이터(Parse→…→Teardown 조립)
  testspec/            [NEW]   C1 · DSL Parse·Fingerprint·Interpreter
    assert/            [NEW]   C1 · 타입인지 검증 함수
  accounts/            [KEEP]  C8 · tx 서명(외부 SDK github.com/0xmhha/accounts 래핑)
  core/
    session/           [NEW]   C5 · .chainbench 정본·env 재사용·기록
    place/             [NEW]   C2 · 배치·포트 통합 + 용량검증(portplan/topology 내부 재사용)
    keyreg/            [NEW]   C2 · 키 레지스트리 + BLSDeriver seam
    collector/         [NEW]   C4 · live tail·원격tail·chainstate·bp참여·분기
    supervisor/        [NEW]   C3 · 기동·헬스게이트·teardown·복구
    driver/            [EXT]   C7 · Transport seam(+kill검증·key_file·upload-if-absent)
    procman/           [EXT]   C3 · {PID,datadir}·원격PID·stop 경로 배선
    genesis/           [EXT]   C2 · GenesisBuilder 4모드
    registry/          [EXT]   C6 · ChainPlugin + Capabilities
    capability/        [EXT]   C6
    obs/               [EXT]   C4 · collector 전송로
    logs/              [EXT]   C4 · 스캔→live tail
    probe/             [→collector] C4
    keys/              [→keyreg]    C2
    state/             [→session]   C5
    config/ node/ rpc/ nodeconfig/ portplan/ topology/ remote/ hardfork/ preflight/ netid/ consensus/  [KEEP]  C8/C2/C7
    pipeline/
      setup/           [REF]   C2 · Provisioner(동시화·Transport 통일)
      verify/          [REF]   C4 · 동시화
      attach/          [KEEP]  · rpc-url only(RPC 강등)
      testrun/         [REPL]  → engine+testspec
  consensus/
    poa/ wbft/ upgrade/    [KEEP]  · 합의·핸드오프
  chains/
    wemix/ wbft/ stablenet/ ...  [EXT]  C6 · plugin + Capabilities + BLSDeriver impl
      wemix/deploy/       [REF]  · 원격 공통절차 core 승격, wemix 특화만 잔류
  mcp/                 [EXT]   H3 · 결과연동
  dashboard/           [EXT]   H3 · 세션 소비
  testkit/             [REPL]  · Go-func→DSL (Report 결과모델은 재사용)
루트:
  remote-server-config.sample.yaml  [KEEP]  · 실파일은 gitignore(SSH 자격증명)
  .chainbench/<session>/            (런타임 산출·gitignore) · session이 소유
```

**T0.0 마이그레이션**: 위 `pkg/*` 전체를 `internal/*`로 이동 + import 경로 rename(cmd/ 포함). 구조는 보존(디렉토리 그대로, 경로 prefix만 pkg→internal).
**신규 디렉토리(7)**: `internal/engine`, `internal/testspec`(+`/assert`), `internal/core/{session,place,keyreg,collector,supervisor}`.
**흡수/이동**: `core/keys`→keyreg, `core/probe`→collector, `core/state`→session.
**교체**: `pipeline/testrun`·`testkit`(Go-func) → engine+testspec DSL(결과모델 재사용).

---

## 4. 진행 규칙
- **TDD**: Low는 RED→GREEN 단위테스트 먼저. 동시성은 `-race`. 통합은 라이브(실 노드).
- **Go 품질**: [[go-code-quality-guidelines]](const화·DI·typed enum·ctx 전파·docs.go 등) 준수.
- **PR 단위**: Task 1개 ≈ PR 1개. 커밋은 [[commit-message-rules]]·[[pr-body-no-emoji-no-claude]]·[[git-add-explicit]].
- **하위호환 병존**: 신규 경로를 별도로 세우고 기존 `setup/upgrade/test`는 유지 → 회귀 없이 점진 이관.
- **게이트**: 각 Task는 비-e2e 통과 + 해당 F의 AC(대표 1건 라이브) 충족 후 다음.
