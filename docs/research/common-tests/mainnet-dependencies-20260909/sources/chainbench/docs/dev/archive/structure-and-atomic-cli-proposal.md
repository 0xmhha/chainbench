# 코드 구조 AST 검토 · 원자 CLI 모듈 구성 제안

> **[대체됨]** 제안분(internal/app · core/launchopt)은 **구현 완료**. 남은 논의는 [[surface-unification-design]] 로 대체.
> **새 작업의 근거로 쓰지 말 것.** 기록으로만 남긴다.

> 지시 2(AST 기반 구조 분류·검토·제안) · 지시 3(MCP 이전 단계의 원자 CLI 모듈화) 응답.
> 작성: 2026-08-11 · 기준 커밋 `2181191` · 총 55,148 LOC / `internal` 55 패키지 / `cmd/chainbench` 44 파일 4,179 LOC.
> 방법: `go list -json ./internal/...` 로 패키지 import 그래프를 만들고 fan-in/fan-out·계층 위반·
> 소비자 역추적을 계산. 커맨드 표면은 `cobra.Command{Use}` 선언을 grep 으로 전수.

---

## 1. 측정 결과

### 1.1 계층 규율 — **위반 없음**

```
core/* → chains/* | consensus/* | engine | testspec  임포트: 0건
```

design §2 의 "의존 방향 위→아래 단방향"은 **실제로 지켜지고 있다.** 이것은 좋은 소식이고,
아래의 문제들은 계층 위반이 아니라 **중복 스택**의 문제다.

### 1.2 fan-in / fan-out

| 최다 피의존 (fan-in) | | 최다 의존 (fan-out) | |
|---|---:|---|---:|
| `core/node` | 22 | `engine` | 19 |
| `core/registry` | 17 | `mcp` | 19 |
| `core/driver` | 10 | `chainsetup` | 12 |
| `core/rpc` | 9 | `consensus/upgrade` | 9 |
| `core/remote` · `core/obs` | 6 | `core/pipeline/setup` | 9 |

`core/node`·`core/registry` 가 공유 커널로 안정적으로 자리 잡았다(C8 의도대로).
문제는 fan-out 쪽: **`mcp`(19)가 `engine`(19)과 같은 수준으로 core 를 직접 붙잡고 있다.**
표면(H3)은 "얇은 어댑터"여야 한다는 설계와 어긋난다.

### 1.3 핵심 발견 — **오케스트레이션 스택이 3개 병존한다**

| # | 스택 | 상태 저장소 | 진입점 | 소비자 |
|---|---|---|---|---|
| **A. 레거시** | `core/pipeline/{setup,verify,attach,testrun}` + `testkit` + `core/probe` | `core/state` (`state/networks/`, NodeSet JSON) | `setup` `test` `stop` `clean` `status` `node` `hardfork` | `internal/mcp` 대부분 (12파일 중 5) |
| **B. 재설계 엔진** | `engine` + `testspec` + `core/{session,place,keyreg,collector,supervisor,provision}` | `core/session` (`.chainbench/<session>/`) | `run` `validate` | `mcp chainbench_run` 1개 |
| **C. 스텝 CLI** | `chainsetup` + `netcompose` | `netcompose.Workspace` (`<data-dir>/state.json`) | `chain` `net` | (없음 — CLI 전용) |

**세 스택이 같은 개념을 각자 저장한다.**

| 개념 | A | B | C |
|---|---|---|---|
| 노드 목록 | `state.SaveNetwork` → `NodeSet` | `session.Environment.PopulateNodeTable` → `env.json` | `Workspace.State`(아직 노드 테이블 없음) |
| 실행 계획 | `setup.BuildPlan` | `engine.AssemblePlan` | `chainsetup.static` |
| 키 소스 | `core/keys` preset | `core/keys` preset (`keyreg` 는 **미배선**) | `core/keys` preset |
| 진행 상태 | 없음 | `session.json` | `state.json` steps map |

이것이 유지보수성·가독성 저하의 **1차 원인**이다. 새 기능을 어디에 넣어야 하는지 코드가 알려주지
않는다 — 실제로 `netcompose`(가장 최근)는 B(engine)가 아니라 A 의 부품(`core/driver`,
`core/filestore`)에 직접 붙었다.

### 1.4 배선되지 않은 모듈 — `keyreg`

```
keyreg.New(...)  프로덕션 호출 지점: 0
engine/attach.go:79  session.New(cfg.ArtifactRoot, cmd, clock(), nil)
engine/app.go:114    session.New(cfg.ArtifactRoot, cmd, clock(), nil)
                                                              ^^^ keyreg.Registry = nil
```

`keyreg` 는 구현·단위테스트가 끝나 있고(T1.6 ☑) `session`·`testspec` 이 타입으로 받고 있지만
**아무도 생성하지 않는다.** 결과로 배경 1.4·1.5 와 알고리즘 2·3(키를 random 생성할지 기존 것을
쓸지 결정)이 통째로 미구현이며, 실경로는 `keys/preset` 하드코딩(`engine/app.go:50`)이다.

component-architecture §2b 가 경고한 **"테스트 있음 ≠ 프로덕션 배선됨"의 재발**이다. 이번엔
"이름만 있는 것"이 아니라 "생성자만 없는 것"이라 grep 으로도 안 잡힌다.

### 1.5 `cmd/chainbench` 비대

44개 비테스트 파일 4,179 LOC. H3 설계("로직 없는 얇은 어댑터")와 어긋난다. 커맨드 60개가
세 스택에 나뉘어 붙어 있고, 같은 일을 하는 쌍이 존재한다:

```
stop      (A: state.LoadNodeSet + driver.NewLocalDriver 하드코딩)   ↔  net (C) / supervisor.Teardown (B)
status    (A)                                                       ↔  net status (C)
setup     (A: static family 만, governance-etcd 분기 없음)          ↔  chain up (C)
test      (A: testkit Go-func)                                      ↔  run (B: DSL)
```

`cmd/chainbench/stop.go:26` 이 `driver.NewLocalDriver()` 를 하드코딩해 원격 노드를 정지하지 못하는
것(component-arch §2b 기록)도 이 중복의 산물이다 — B 는 이미 원격을 처리한다.

---

## 2. 목표 구조 제안

### 2.1 원리 — 스택 3 → 1, 상태 저장소 3 → 1

```
              ┌─────────────── Surfaces (얇은 어댑터, 로직 0) ───────────────┐
   cmd/chainbench            internal/mcp              internal/dashboard
        │                        │                          │
        └───────────┬────────────┴──────────────────────────┘
                    ▼
            internal/app        ← [NEW] 유스케이스 1개 = 함수 1개 (CLI·MCP 공용)
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
  internal/engine         internal/testspec     (오케스트레이션 · DSL 해석)
        │                       │
        ▼                       ▼
  internal/core/*  (session · place · keyreg · genesis · launchopt · provision ·
                    supervisor · procman · collector · driver · rpc · node · registry)
        │
        ▼
  internal/chains/* + internal/consensus/*      (C6 ACL — 유일한 변화점)
```

**신설은 2개뿐이다.**

| 패키지 | 역할 | 근거 |
|---|---|---|
| `internal/app` [NEW] | 유스케이스 함수 집합. CLI RunE 와 MCP 핸들러가 **같은 함수**를 호출 | 지시 3 의 "CLI ↔ MCP 행위 동일" 요구, chain-cli-execution-plan §4.1 |
| `internal/core/launchopt` [NEW] | 실행옵션 모듈 + Dialect + Builder | [`chain-binary-flag-graph.md`](../chain-binary-flag-graph.md) §3.3 |

**흡수/폐기 (신규 작업 아님 — 소비자 이관 후 삭제):**

| 대상 | 흡수처 | 선행 조건 |
|---|---|---|
| `core/state` | `core/session` | `mcp` 5파일 + `cmd` 7파일 이관 |
| `core/pipeline/testrun` · `testkit` | `engine` + `testspec` | `cmd/test.go`, `mcp/tools.go` 이관 |
| `core/probe` | `core/collector` | `mcp/network_tools.go` 이관 |
| `core/keys` | `core/keyreg` (preset = `Source` 한 종류로 편입) | §1.4 배선 선행 |
| `core/pipeline/setup` | `engine`(plan) + `core/filestore`(물질화) | 13개 소비자 |
| `netcompose.Workspace` | `core/session` 의 "long-lived environment" 모드 | 아래 §3.2 |
| `chainsetup` | `app` 유스케이스 + `engine` | 케이스 지식은 데이터(profiles/)로 |

### 2.2 `internal/app` 의 형태

핵심은 **cobra 타입도 MCP 타입도 모르는 순수 유스케이스**다.

```go
package app

// Deps are the collaborators every use case shares, injected once at the
// surface boundary. No package-level state.
type Deps struct {
    Session  session.Store
    Registry registry.Lookup
    Driver   driver.Driver
    Keys     keyreg.Registry
    Bus      *obs.Bus
    Clock    func() time.Time
}

// One use case = one function. Input/Output are plain structs that both cobra
// flag binding and MCP JSON-schema binding can target.
func NewNetwork  (ctx context.Context, d Deps, in NewNetworkIn)  (NewNetworkOut,  error)
func GenerateKeys(ctx context.Context, d Deps, in KeysIn)        (KeysOut,        error)
func AllocatePorts(ctx context.Context, d Deps, in AllocateIn)   (AllocateOut,    error)
func BuildGenesis(ctx context.Context, d Deps, in GenesisIn)     (GenesisOut,     error)
func RenderConfig(ctx context.Context, d Deps, in ConfigIn)      (ConfigOut,      error)
func Provision   (ctx context.Context, d Deps, in ProvisionIn)   (ProvisionOut,   error)
func InitDataDir (ctx context.Context, d Deps, in InitIn)        (InitOut,        error)
func StartNodes  (ctx context.Context, d Deps, in StartIn)       (StartOut,       error)
func StopNodes   (ctx context.Context, d Deps, in StopIn)        (StopOut,        error)
func TailLogs    (ctx context.Context, d Deps, in LogsIn)        (LogsOut,        error)
func RunSpecs    (ctx context.Context, d Deps, in RunIn)         (RunOut,         error)
// ...
```

- `cmd/chainbench/<verb>.go` → 플래그 바인딩 + `app.X()` 호출 + 출력 포매팅. **평균 40줄 목표.**
- `internal/mcp/<verb>.go` → JSON 스키마 + 같은 `app.X()` 호출. 별도 로직 금지.
- 이 경계가 서면 `cmd/chainbench` 4,179 LOC 는 대략 1/3 로 줄고, MCP 의 fan-out 19 는 1(`app`)이 된다.

### 2.3 목표 폴더 트리 (델타만)

```
internal/
  app/                 [NEW]  유스케이스 — CLI·MCP 공용 진입점
  engine/              [KEEP] 오케스트레이션
  testspec/            [KEEP] + schema/v2.schema.json  [NEW] 문법 정본
  core/
    launchopt/         [NEW]  Dialect + 옵션 모듈 10 + Builder
    session/           [EXT]  + long-lived workspace 모드(netcompose 흡수)
    keyreg/            [EXT]  + preset Source, **프로덕션 배선**
    genesis/ place/ provision/ supervisor/ procman/ collector/ driver/
    rpc/ node/ registry/ config/ nodeconfig/ obs/ remote/ ...   [KEEP]
    state/             [→session]
    keys/              [→keyreg]
    probe/             [→collector]
    pipeline/          [→engine + provision]  (setup/verify/attach/testrun 전부)
  chainsetup/          [→app + profiles/ 데이터]
  netcompose/          [→core/session]
  testkit/             [→engine + testspec]
  chains/ consensus/   [KEEP] C6 ACL
  mcp/ dashboard/      [EXT]  app 만 의존하도록 축소
```

---

## 3. 지시 3 — 원자 CLI 모듈 구성

### 3.1 현황 판정

`chain-cli-execution-plan.md §4.1` 이 정의한 원자 명령 표면은 **이미 착수되어 있다**:
`internal/netcompose`(Workspace + TargetSpec) + `net new` / `net status`. 골격은 옳다.

다만 두 가지를 지금 고쳐야 나중에 되돌리는 일이 없다.

1. **`netcompose.Workspace` 는 `core/session` 과 같은 것을 다르게 저장한다**(§1.3 표).
   지금은 필드가 6개뿐이라 통합 비용이 거의 0 이지만, 노드 테이블·포트맵·PID 가 들어가는 순간
   `session.Environment` 와 완전 중복된다.
2. **`net` 스텝이 `app` 이 아니라 core 를 직접 부른다** → 같은 스텝의 MCP 미러(P3)를 만들 때
   로직이 복제된다.

### 3.2 제안 — 스텝 = 유스케이스 = MCP 도구 (1:1:1)

```
chainbench net <step>   ──┐
                          ├──►  app.<Step>(ctx, deps, in) ──► core/*
mcp  net_<step>         ──┘
```

**워크스페이스는 `session` 이 소유한다.** `session.Session` 에 "명시적으로 열고 닫는 장수명
환경" 모드를 추가하고, `netcompose.Workspace` 는 그 위의 얇은 뷰로 남기거나 제거한다.
근거: 노드 테이블·env fingerprint·로그 경로·PID 는 이미 `session.Environment` 의 계약이며
(design §3.1), 두 벌로 갈라두면 `net start` 로 띄운 노드를 `run` 이 못 보는 상태가 된다.

### 3.3 스텝 카탈로그 (배경/알고리즘 매핑)

| 알고리즘 단계 | 스텝 명령 | 유스케이스 | 단계 검증(무엇을 보고 성공이라 하나) |
|---|---|---|---|
| 1 | `net new --chain --target` | `NewNetwork` | 워크스페이스 생성·target 해석 |
| 2 | `net keys nodekeys --source random\|preset\|import\|remote` | `GenerateKeys` | 노드키 개수·enode 출력 |
| 3 | `net keys accounts --source … --balance` | `GenerateKeys` | 주소·잔액 테이블 |
| — | `net allocate --mode os\|stepped --hosts` | `AllocatePorts` | 포트맵 + **용량검증(min≥4, max)** |
| 4·5 | `net genesis --mode existing\|build\|template\|inherit --base --set --overlay` | `BuildGenesis` | genesis 해시·validator 셋·chainId |
| — | `net config --sync-mode --set` | `RenderConfig` | 노드별 TOML |
| — | `net launchopts [--set key=val]` **[NEW]** | `BuildLaunchArgs` | **조립된 실행 command 출력(실행 없이)** |
| 6 | `net provision` | `Provision` | 물질화 파일 목록 + upload-if-absent 결과 |
| 6 | `net init` | `InitDataDir` | `<datadir>/chaindata` 존재 |
| 7·8 | `net start [--node\|--all]` | `StartNodes` | PID · procman 등록 |
| 9 | `net health [--min-height]` | `Health` | head 전진 · peer 수 · (etcd 리더) |
| 10–12 | `net run <case.json>` | `RunSpecs` | session 판정 |
| 13 | `net report` | `Report` | status/assert 요약 |
| 15 | `net stop [--grace] [--rm-data]` | `StopNodes` | **PID 기반 종료 + 고아 0 검증** |
| — | `net logs --node --follow --since` | `TailLogs` | 최근 라인 |
| — | `net status` | `Status` | 각 스텝 done/detail |
| poa | `net deploy-governance` · `net etcd-init` · `net verify-etcd` | 동명 | `admin.wemixInfo.etcd.cluster` 비어있지 않음 |

**`net launchopts` 를 새로 넣은 이유**: 배경 2 · 알고리즘 7 이 요구하는 "실행 command 빌드"를
**실행과 분리해 눈으로 확인**할 수 있어야 원자 CLI 로서 의미가 있다. builder 가 조립한 최종
argv 를 그대로 출력하고, `net start` 는 같은 결과를 실행한다.

### 3.4 원자성 규율 (지금 정하지 않으면 나중에 못 고치는 것)

1. **각 스텝은 idempotent.** 재실행이 안전해야 알고리즘 6("이미 있으면 skip")과 8("동일 환경이면
   기동 skip")이 CLI 수준에서 성립한다. 판정 기준은 fingerprint — `net` 워크스페이스도
   `session` 과 같은 fingerprint 를 계산해 재사용/재구성을 결정한다.
2. **스텝은 자기 선행조건을 확인하고 명확히 실패한다.** `net start` 전에 `net init` 이 안 됐으면
   "chaindata 없음 — `net init` 을 먼저 실행" 이라고 말한다. 조용한 성공 금지.
3. **출력은 사람용 + `--json`.** `run --json`/`validate --json` 이 이미 있는 규약을 전 스텝에 확대.
4. **경로는 하나의 문자열.** local `/path`, remote `user@host:/path`. `--remote-host/--remote-user/
   --remote-port/--target-dir` 4개 플래그(`cmd/chainbench/net.go:52-55`)를 `--target` 하나로 접는다
   (자격증명은 여전히 `remote-server-config.yaml` — spec/CLI 에 비밀 금지).
5. **CLI 커맨드 추가 = MCP 도구 추가**를 같은 PR 에서. 표면이 갈라지면 다시 못 합친다.

### 3.5 기존 60개 커맨드 정리

| 분류 | 커맨드 | 조치 |
|---|---|---|
| 원자 스텝(유지·확장) | `net *`, `keys`, `account *`, `validator *`, `tx *`, `contract *`, `faucet`, `log`, `node *` | `app` 경유로 재배선 |
| 오케스트레이터(데모로 격하) | `chain up`, `setup`, `upgrade run` | 원자 스텝의 **조합 예제**로 유지, 자체 로직 제거 |
| 중복(폐기 대상) | `stop`, `clean`, `status`, `test` | `net stop`/`net clean`/`net status`/`run` 으로 흡수. `test`(testkit Go-func)는 DSL 이관 완료 후 |
| 진단(유지) | `chains`, `capabilities`, `consensus`, `hardfork`, `verify`, `report`, `validate`, `resolve` | 그대로 |

---

## 4. 실행 순서 권고

기존 `chain-cli-execution-plan.md §5` 의 P2–P6 를 대체하지 않고 **선행 2단계를 앞에 끼운다.**

| 순서 | 작업 | 게이트 |
|---|---|---|
| **P1′** | `internal/app` 신설 + `net new`/`net status` 를 app 경유로 이전 | 동작 변화 0, `go test -race ./...` green |
| **P2′** | **`keyreg` 프로덕션 배선** (§1.4) — `session.New(…, keyreg.New(…))`, preset 을 `Source` 로 편입, `net keys --source` 노출 | 랜덤 노드키로 4노드 기동 라이브 1건 |
| P2 | static 원자 스텝 잔여(`allocate`/`genesis`/`config`/`provision`/`init`/`start`/`stop`/`logs`) | gstable 라이브 단계별 검증 |
| P2.5 | `core/launchopt` + `net launchopts` | 골든 테스트로 기존 argv 와 바이트 동일 |
| P3 | MCP 미러(=app 재사용) | CLI↔MCP 동일 출력 테스트 |
| P4 | governance-etcd 스텝(gwemix) | `admin.wemixInfo.etcd.cluster` 검증 |
| P5 | 원격(`--target user@host:/path`) | 실 SSH 라이브 |
| P6 | DSL v2 + 스위트 이관 | [`dsl-v2-proposal.md`](../dsl-v2-proposal.md) §4 |
| P7 | 레거시 스택 A 제거(`state`/`pipeline`/`testkit`/`probe`) | 소비자 0 확인 후 |

**P2′ 를 앞으로 당긴 이유**: 원자 스텝의 2·3번(키 소싱)이 곧 알고리즘의 2·3번이다. 이것 없이
`net genesis` 부터 만들면 preset 하드코딩 위에 스텝을 쌓게 되고, 나중에 키 소스를 열 때
genesis·provision·start 를 전부 다시 건드려야 한다.
