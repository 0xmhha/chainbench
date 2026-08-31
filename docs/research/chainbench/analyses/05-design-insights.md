# 설계 심층 분석과 리팩토링 제안

## 1. 네트워크 구성의 단일 소유자

### chainbench는 어떻게 했나

workspace 경로는 `chainsetup.NetUp`의 9단계로 구성되지만 (`internal/chainsetup/verbs_up.go:85`), local test engine도 할당·genesis·plan·provision·bring-up을 직접 조립한다 (`internal/testengine/buildenv.go:71`). `app.RunSuite`는 workspace를 구성한 뒤 attach engine을 사용하므로 세 번째 조립 형태까지 생긴다 (`internal/app/workflow.go:140`).

### 가져갈 교훈

1. `chainsetup`을 `Plan → Provision → Start → Stop/Resume`의 canonical lifecycle로 정한다.
2. local/remote/handoff 차이는 별도 파이프라인이 아니라 plan 또는 strategy로 주입한다.
3. `testengine.NewLocalEngine`과 `NewBuildEnv`는 canonical provisioner로 교체한 뒤 제거한다.

## 2. DSL 언어와 실행기의 분리

### chainbench는 어떻게 했나

`testspec.Parse`는 v1/v2 문법을 하나의 `Spec`으로 수렴시킨다 (`internal/testspec/spec.go:66`). 그러나 같은 패키지의 interpreter 계약은 accounts, key store, RPC, collector, node, session 타입을 안다 (`internal/testspec/interpreter.go:6`). 구체 구현은 registry와 `Deps`로 외부에서 주입되므로 전역 결합은 피했지만, 패키지 수준에서는 순수 문법만 가져가는 소비자도 런타임 의존 경계와 함께 놓인다.

### 가져갈 교훈

1. `internal/testspec`에는 AST, schema, parse, fingerprint만 둔다.
2. 마이그레이션은 `internal/testspec/migrate`, 실행은 `internal/testengine/interpreter` 또는 `internal/dslinterp`로 옮긴다.
3. registry descriptor와 offline validation을 공유해 새 액션 등록 누락을 CLI `validate`에서 잡는다.

## 3. 표면과 유스케이스 경계

### chainbench는 어떻게 했나

workspace 모드의 CLI는 `app.RunSuite`를 호출하지만 attach/local 모드는 CLI가 `testengine`과 resource resolution을 직접 조립한다 (`cmd/chainbench/run.go:60`, `cmd/chainbench/run.go:192`). 반대로 `app.Net*`에는 `chainsetup`을 그대로 전달하는 함수가 남아 있다 (`internal/app/net.go:56`).

### 가져갈 교훈

1. `app.Run` 하나가 attach/local/workspace/handoff 모드 선택을 소유한다.
2. CLI와 MCP는 같은 input/output DTO를 각 표면 형식으로 변환하는 일만 맡는다.
3. app의 타입 별칭과 1줄 forwarding wrapper는 소비자 전환 후 삭제한다.

## 4. 상태와 cleanup의 명확성

### chainbench는 어떻게 했나

PID는 process ledger가 정본이지만 workspace node record에도 복제된다 (`internal/chainsetup/workspace.go:103`, `internal/chainsetup/workspace.go:307`). 두 파일은 순차 저장돼 하나의 원자적 transaction이 아니다 (`internal/chainsetup/workspace.go:297`). 엔진 teardown은 실행 context를 그대로 쓰고 오류를 버린다 (`internal/testengine/engine_impl.go:74`).

이중 표현은 서로 다른 수명과 조회 목적을 가진 의도된 projection이므로 곧바로 충돌이라고 단정할 수는 없다. 다만 workspace와 process JSON은 제자리 덮어쓰기를 사용하고 (`internal/core/session/composition.go:75`, `internal/core/process/ledger.go:56`), lock 획득도 배타 생성이 아니라 확인 뒤 `WriteFile`하는 방식이라 동시 실행과 프로세스 중단에 대한 안전성 보강이 필요하다 (`internal/core/session/lock.go:83`, `internal/core/session/lock.go:107`).

### 가져갈 교훈

1. topology record에서 runtime PID를 제거하고 읽을 때 ledger와 합성한다.
2. cleanup은 제한 시간이 있는 별도 context로 실행하고 본 오류와 `errors.Join`한다.
3. 부분 build 실패에도 cleanup handle을 돌려줄 수 있는 `BuildResult` 계약을 사용한다.
4. session 파일의 read/write DTO와 파일명을 `session` 패키지 하나가 소유하게 한다.
5. control-plane JSON은 임시 파일·동기화·rename primitive로 통일하고 lock은 원자적으로 획득한다.

## 5. 이름과 가독성

### chainbench는 어떻게 했나

같은 영역에 `resource`, `resourcecmd`, `NetMap/NetPool/NetPlan` 어휘가 공존한다. `testengine/app.go`는 application 계층이 아니라 local engine 조립을 담고, 폐기된 `Supervisor` 용어가 `BuildDeps`에 남아 있다 (`internal/testengine/app.go:92`, `internal/testengine/buildenv.go:41`).

### 가져갈 교훈

1. 패키지명은 소유 대상을, 함수명은 동작과 대상을 함께 말하게 한다.
2. 임시 이름 변경은 `app.go`→`local_engine.go`, `Supervisor`→`Launcher`부터 시작한다.
3. `Net*`/`Network*` 전달 API를 제거하고 `InspectNetwork`, `RemoveNetwork`처럼 범위를 드러낸다.

## 6. CLI를 통한 core 동작 검증

### 현재 기반

root command는 명령을 한곳에서 등록하고 (`cmd/chainbench/root.go:28`), `validate`는 spec 파싱과 이름 검증을 오프라인으로 실행할 수 있다 (`cmd/chainbench/validate.go:53`). `netcmd`와 `resourcecmd`에는 Cobra 실행 helper를 이용한 명령 테스트가 있다 (`cmd/chainbench/netcmd/net_test.go:20`, `cmd/chainbench/resourcecmd/resource_test.go:23`).

### 목표 검증 매트릭스

| core 계약 | CLI 명령 | 검증 방식 |
|---|---|---|
| DSL parse/resolve | `chainbench validate` | 파일·stdin 입력, JSON 오류 code golden |
| resource allocation | `chainbench resource plan --json` | 결정적 placement golden |
| composition plan | `chainbench net up --dry-run --json` 신설 | 외부 프로세스 없이 단계·argv·파일 계획 검증 |
| lifecycle recovery | `chainbench net resume --json` | fake driver로 중단 지점 재개 검증 |
| test execution | `chainbench run --rpc ... --json` | httptest RPC로 session manifest 검증 |
| cleanup | `chainbench net stop/rm --json` | fake ledger/driver로 멱등성과 오류 합성 검증 |

명령 테스트는 Cobra 객체만 확인하는 수준을 넘어, 같은 application service에 fake port를 주입해 core 계약의 입력·출력·exit code를 고정해야 한다. 실제 binary smoke test는 소수의 대표 경로만 두고 나머지는 in-process command test로 유지한다.

## 종합: 적용 우선순위

초기 분석은 상위 구성 경로를 먼저 합치도록 제안했다. 이 순서는 최근 리팩토링의 문제를 반복한다. 하위 판단이 여러 곳에 남은 상태에서 `chainsetup`이나 `testengine`을 먼저 합치면 중복 구현을 새 패키지로 옮길 뿐이다.

수정한 우선순위는 다음과 같다.

1. **P0 — 기준선과 계약**: AST 그래프, 공개 심벌, CLI, workspace와 session 산출물을 고정한다.
2. **P1 — resource**: 서버 셋, 포트, 풀, 인벤토리, 배정, 접근, enode 조합을 원자 하위 경계로 나눈다.
3. **P2 — node**: 역할, label, placement, topology를 하나의 노드 사실 모델로 수렴한다.
4. **P3 — genesis**: validator 개수가 아니라 확정된 producer identity를 입력으로 받는다 (`internal/core/genesis/source.go:29`).
5. **P4 — nodeconfig**: 같은 Spec에서 TOML과 argv를 렌더한다.
6. **P5 — process**: 불변 launch manifest만 받아 materialize, launch, stop, cleanup을 수행한다.
7. **P6 — dsl/testhelper/collector**: 문법, 실행기, 어휘, 관측 경계를 정리한다.
8. **P7 — chainsetup**: 앞 단계의 operation만 순서대로 조합하고 workspace를 기록한다.
9. **P8 — testengine**: 구성된 환경에서 테스트를 실행하는 역할만 남긴다.
10. **P9 — app/CLI/MCP**: 같은 유스케이스를 호출하고 입력 변환과 렌더링만 맡긴다.
11. **P10 — 은퇴**: 호출자와 import가 0인 레거시만 삭제한다.

단계별 상세 범위와 승인 게이트는 `06-refactoring-plan.md`에 기록한다.
