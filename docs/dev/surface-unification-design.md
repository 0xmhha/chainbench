# 표면 통일 리팩토링 — Feature 레지스트리 · cmd 박막화 · 체인 공통/특화 분리

> **[현행 설계]** 표면 통일·명령 표면.
> 지금 향하는 목표. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])이고,
> 작업 순서는 [[chainbench-worklist]] §1g 다.

> 문제: 같은 기능이 CLI·MCP·DSL 세 곳에 각각 구현돼 있고, `cmd/` 가 CLI 가 아니라 오케스트레이터가 돼 있다.
> 목표: **기능을 한 번 등록하면 세 표면이 그것을 렌더링**하게 한다. `cmd/` 는 바인딩과 출력만 남긴다.
>
> 실측: 2026-08-18. 관련: [[layers]](architecture/layers.md) · [[module-responsibilities]](architecture/module-responsibilities.md) ·
> [[cli-steps]](chain-setup/cli-steps.md). 작업 순서는 [[chainbench-worklist]](chainbench-worklist.md) §1g.

---

## 1. 진단 — 숫자

| 사실 | 값 |
|---|---|
| `cmd/chainbench` 비-테스트 코드 | **4,569 줄** / 45 파일 |
| 그중 **`app`(L5)을 거치는** 파일 | 16 |
| **`app` 을 우회해 L1~L4 를 직접 부르는** 파일 | **21** |
| 최악 사례 `upgrade_run.go` | **395 줄**, 9개 패키지 직접 참조 (`consensus/poa`·`consensus/upgrade`·`core/{driver,genesis,keys,launchopt,node,registry,rpc}`) |
| MCP 도구 | 46개 |
| 그중 **손으로 쓴 JSON 스키마** | **34개** |
| DSL 액션·어세션 레지스트리 | `testspec` 안에 **별도로** 존재 |

즉 **기능 목록이 세 벌 있다.** CLI 는 cobra 로, MCP 는 손으로 쓴 스키마로, DSL 은 자체 레지스트리로.
새 기능은 세 번 써야 하고, 실제로 세 곳이 갈라졌다 — `tx send` 는 CLI·MCP·DSL 에 각각 있고,
`faucet`·`verify` 는 CLI·MCP 에만 있다.

이것이 "기능이 산재하고 통일되지 않았다"의 정체다. 레이어는 지켜지는데(상향 의존 0건)
**같은 층 안에서 세 갈래로 갈라져 있다.**

---

## 2. 큰 틀 — 3단계

시스템의 기본 동작은 셋이고, 모든 기능은 이 중 하나에 속한다.

```
① Compose  체인을 구성한다      keys · allocate · genesis · config · launchopts
                                 provision · init · start · stop · restart · rm
② Test     테스트를 수행한다     run · validate · tx · faucet · contract · verify · health · fault
③ Report   결과를 정리한다       status · logs · report · session 조회 · chainstate
```

이 구분이 **레이어와 직교한다** — 각 단계가 L5 유스케이스를 갖고, 그 아래로 L4→L0 을 탄다.
그리고 세 표면 각각이 세 단계를 전부 덮어야 한다:

| | ① Compose | ② Test | ③ Report |
|---|---|---|---|
| **CLI** | `net <step>` 부분 실행 | `run` · `tx` · … | `status` · `report` |
| **DSL** | `env` 선언 | `case` 스텝·어세션 | `hooks` · 세션 판정 |
| **MCP** | `net_*` 도구 | `run`·`tx_*` 도구 | `report`·`status` 도구 |

---

## 3. 핵심 — Feature 레지스트리

### 3.1 지금 왜 추상화가 정당한가

프로젝트 규칙은 "과설계 금지 — 두 번째 사용처가 생길 때 추상화"다.
여기는 **소비자가 이미 셋이고, 셋이 실제로 갈라졌다.** 추측이 아니라 중복 제거다.

### 3.2 형태

```go
// internal/feature (L5)

// Stage is which of the three phases a feature belongs to. It is what a
// surface groups by: CLI command groups, DSL sections, MCP namespaces.
type Stage string

const (
    StageCompose Stage = "compose"
    StageTest    Stage = "test"
    StageReport  Stage = "report"
)

// Descriptor is one feature, described so that any surface can render it
// without knowing the feature. It is the erased form; authors use Register.
type Descriptor struct {
    Name    string   // "net.genesis" · "tx.send" · "report.session"
    Stage   Stage
    Summary string
    // Input returns a fresh zero input. A surface fills it — cobra from flags,
    // MCP from JSON, the DSL from step arguments — and hands it to Invoke.
    Input  func() any
    Invoke func(ctx context.Context, d Deps, in any) (any, error)
}

// Register wraps a typed use case into the registry. Authoring stays typed;
// only the registry is erased, which is the same trade testspec already makes
// for its actions.
func Register[In, Out any](d Descriptor, fn func(context.Context, Deps, In) (Out, error))
```

**입력 struct 의 태그가 세 바인딩을 전부 만든다.**

```go
type NetGenesisIn struct {
    DataDir string `cb:"data-dir,required" help:"workspace directory"`
    ChainID int64  `cb:"chain-id"          help:"override the manifest chain id"`
    Set     []string `cb:"set"             help:"genesis config override key=value"`
}
```

| 표면 | 태그로 만드는 것 |
|---|---|
| CLI | cobra 플래그 (`--data-dir` 필수, `--chain-id`, `--set` 반복) |
| MCP | JSON 스키마 (`required:["data_dir"]`, 타입·설명) — **손으로 쓰던 34개가 사라진다** |
| DSL | 스텝 인자 이름·검증 (`chainbench validate` 가 오프라인으로 잡는다) |

### 3.3 이것이 해결하는 것

| 지금 | 이후 |
|---|---|
| 기능 추가 = 3곳 작성 | 1곳 등록 |
| MCP 스키마 34개 손작성 | 태그에서 파생 |
| CLI 에만 있는 기능(`faucet`·`verify` 의 DSL 부재) | 등록되면 세 표면에 동시 노출 |
| "이 기능이 어느 표면에 있나?" 를 코드로 답할 수 없음 | 레지스트리 순회로 답함 (`capabilities` 명령이 이미 하려던 것) |

### 3.4 하지 않는 것

- **자동 CLI 생성기를 만들지 않는다.** cobra 명령은 계속 손으로 쓰되, **플래그 바인딩만** 태그에서 만든다.
  명령 이름·계층·도움말 문구는 사람이 정하는 편이 낫다.
- **DSL 문법을 레지스트리에서 생성하지 않는다.** DSL 은 자체 문법(v2 스키마)이 정본이고,
  레지스트리는 *이름 해결*만 제공한다(`testspec.Unresolved` 가 이미 하는 일).

---

## 4. 명령 표면 설계

### 4.1 지금 — 72개 엔드포인트, 겹침 심함

최상위 **26** + 서브커맨드 **46**. 같은 일을 하는 명령이 여럿이다.

| 하는 일 | 오늘 이걸로도 되고 | 저걸로도 된다 |
|---|---|---|
| 네트워크 기동 | `setup --launch` | `net up` · `chain up` · `remote deploy` |
| 정지 | `stop` | `net stop` · `chain down` |
| 상태 | `status` | `net status` · `chain status` |
| 노드 재기동 | `node start/stop` | `net restart` |
| 헬스 | `verify` | `net health` |
| 로그 | `log` | `net logs` |
| 정리 | `clean` | `net rm` |

**이것이 "기능이 산재한다"의 사용자 쪽 얼굴이다.** 어느 것을 써야 하는지 코드도 문서도 답하지 못한다.

### 4.2 목표 — 명사를 모델에 맞춘다

청사진 모델의 명사가 곧 명령이다: **네트워크 · 테스트 · 리포트**, 그리고 그것들이 참조하는
**체인 · 서버 · 키 · 스펙**.

```
① Compose ─ net       청사진 → 확정 → 산출 → 기동
② Test    ─ test/tx/faucet/contract
③ Report  ─ report
참조      ─ chain · server · key
```

#### ① `net` — 네트워크 (청사진 파이프라인이 그대로 명령이 된다)

| 명령 | 하는 일 | I/O |
|---|---|---|
| `net init` | 청사진 스캐폴드 생성 | 파일 1개 |
| `net resolve` | 청사진 → `ResolvedNetwork`. **값과 출처를 보여준다** | **없음** (순수) |
| `net build` | 산출물 생성(genesis·config·argv) → 타깃 | 파일만 |
| `net up` | resolve + build + start | 프로세스 |
| `net start` · `stop` · `restart` | 프로세스 수명주기 | 프로세스 |
| `net rm` | 데이터 플레인 제거 | 파일 |
| `net status` · `health` · `logs` | 관측 | 읽기 |
| `net hardfork` | 실행 중 네트워크의 포크 — 바이너리 스왑(type-2)과 체인 핸드오프(type-1)를 **한 명령**이 덮는다 | 프로세스 |

**`net resolve` 가 핵심이다.** 아무것도 만들지 않고 "이 청사진이면 이런 네트워크가 된다"를
출처와 함께 보여준다 — 지금은 이걸 볼 방법이 없어서 띄워 봐야 안다.

**옛 9스텝은 명령이 아니라 필터가 된다.** `net build --only genesis` 처럼.
9스텝은 상태가 점증하던 모델의 산물이었고, 청사진에서는 resolve·build 두 단계로 접힌다.

#### ② `test` · `tx` · `faucet` · `contract`

| 명령 | 하는 일 |
|---|---|
| `test run` | DSL spec 실행 |
| `test validate` | spec 오프라인 검증 |
| `test migrate` | v1 → v2 변환 |
| `tx send` · `tx wait` | 단건 tx |
| `faucet` | 자금 공급 |
| `contract deploy` · `call` | 컨트랙트 |

`tx`·`faucet`·`contract` 는 **DSL 액션의 CLI 대응**이다(`sendTx`·`faucet`·`deployContract`).
같은 기능 레지스트리에서 나오므로 세 표면이 자동으로 같아진다(§3).

#### ③ `report`

`report show`(세션 판정) · `report list` · `report gc`

#### 참조 명령

`chain list|info` (플러그인·capability) · `server list|check` (인벤토리 검증) ·
`key new|import|show|set` (키 자료 — 현재 `keys`·`validator`·`account` 셋으로 갈라진 것을 통합)

### 4.3 사라지는 것

| 삭제 | 어디로 |
|---|---|
| `setup` | `net` |
| `chain up/down/status` | `net` (`cases`·`steps` 는 문서로) |
| `remote deploy/bootstrap` | `net` (타깃이 remote 면 같은 경로) |
| `status`·`stop`·`node`·`verify`·`log`·`clean` | `net status/stop/restart/health/logs/rm` |
| `test`(레거시 Go-func) | `test run` (케이스 이관 후) |
| `chains`·`capabilities` | `chain list/info` |
| `consensus` | `net status --consensus` |
| `keys`·`validator`·`account` | `key` |
| `migrate-spec` | `test migrate` |

**26 → 9 최상위, 72 → 약 32 엔드포인트.**

> 이행 중에는 옛 명령을 **deprecated 별칭**으로 남기고 안내를 출력한다. 한 번에 끊지 않는다.

---

## 4b. `cmd/` 의 목표 형태

```
cmd/chainbench/
  main.go          진입점
  root.go          전역 플래그 · 명령 등록
  net.go test.go report.go chain.go server.go key.go
                   명사당 1파일 — 명령 정의 + 플래그 바인딩 + 출력 렌더
```

**규칙 3개**

1. `cmd/` 는 **`internal/app` 만 import 한다.** L1~L4 직접 참조 금지.
2. `cmd/` 에 **조건 분기 로직이 없다.** 있다면 유스케이스가 빠진 것이다.
3. 파일 하나가 **200줄을 넘으면** 유스케이스가 cmd 에 남아 있다는 신호다.

현재 위반: 21파일이 규칙 1 위반, `upgrade_run.go`(395)·`run.go`(327)·`keyflags.go`(311)·
`chain.go`(271)·`remote.go`(263)·`net_steps.go`(249) 가 규칙 3 위반.

목표: **4,569줄 → 대략 1,800줄** (바인딩+렌더만 남는 분량).

---

## 4c. 모듈 — 어떤 기능을 누가 지원하는가

### 신설 (5개)

| 모듈 | 층 | 담당 | 왜 신설인가 |
|---|---|---|---|
| `core/blueprint` | L1 | 선언 스키마·파싱·구조 검증 (순수) | 지금은 선언 형식 자체가 없다 |
| `core/peering` | L1 | 연결 그래프 계산 — mesh / proxied (순수) | 지금은 풀메시가 코드에 박혀 있다 |
| `core/resolve` | L3 | 청사진+인벤토리+플러그인+키셋 → `ResolvedNetwork` (+출처) | 지금은 스텝마다 흩어져 재조립된다 |
| `core/materialize` | L3 | `ResolvedNetwork` → 산출물 → `Sink` | 지금은 스텝마다 다르게 쓴다 |
| `internal/feature` | L5 | 기능 레지스트리 (§3). `Deps` 를 소유한다 — `app/feature` 로 두면 `app` 과 참조 순환 | 표면 3개가 각자 목록을 갖는다 |

### 변경 (6개)

| 모듈 | 무엇을 더하나 |
|---|---|
| `core/registry` | `BringUpPhases` · `PortReservation` · `SupportsRole` · `SupportsPeeringGraph` |
| `core/portplan` | 패밀리 `Reservation` 기반 검증 (전역 `p2pStep>=2` 제거) |
| `core/supervisor` | `Launch(plan, nodes)` · `Deps.Action` |
| `consensus/poa` | 액션 배선 · `PrepareTemplate` (현 `BuildGenesis` 재배치) |
| `consensus/wbft` | `BringUpPhases` 1페이즈 반환 |
| `testspec` | 구문(L1) / 실행(L3) 분리 |

### 흡수·삭제 (6개)

| 모듈 | 어디로 |
|---|---|
| `netcompose` | resolve·materialize 로 분해, 얇은 오케스트레이터만 잔류 |
| `chainsetup` | `app` 유스케이스로 |
| `chains/wemix/deploy` | 타깃이 remote 인 **일반 경로**로 흡수 (전용 경로 소멸) |
| `core/bringup` · `core/state` | 삭제 (§1f b-6) |
| `testkit` · `core/pipeline/testrun` | 삭제 (케이스 이관 후) |

### 기능 → 모듈 한눈

```
청사진 선언       blueprint(L1)
  ↓ 해석          resolve(L3) ← serverset·place·portplan·peering·keys·registry
  ↓ 산출          materialize(L3) → nodeconfig·launchopt·genesis·(family)
  ↓ 배치          provision.FileSink(L1)  [local | ssh]
  ↓ 기동          supervisor(L3) → driver(L1) → procman(L1)
  ↓ 관측          collector·health(L3) → session(L3)
  ↓ 실행          testspec(L1 구문 / L3 실행)
  ↓ 판정          session(L3)
전 구간 노출      feature(L5) → cmd · mcp 는 읽고, dsl/interp 는 주입받는다
```

---

## 5. 3체인 구성 요소 — 공통과 특화

> **이 절은 [[network-blueprint-design]](network-blueprint-design.md) 로 옮겼다.**
> 여기 있던 8요소·9스텝 표는 **누락이 많았다** — 노드별 nodekey/계정 지정, BP 수와 역할 배정,
> alloc 잔액, 검증자 집합 선언, 바이너리의 노드별 오버라이드, 연결 구성(static-nodes vs
> 거버넌스 member + etcd)이 빠져 있었다. 전수 목록(N1~N14 · P1~P13 · C1~C4)과
> 선언→해석→물질화 구조는 그 문서에 있다.
>
> 요약만 남기면: **특화는 6개 요소이며 전부 poa(wemix) 한 패밀리에 속한다.**
> 나머지는 세 체인이 같은 코드를 탄다.

---

## 6. 이행 — 한 번에 하나씩

전면 재작성이 아니라 **기능 단위 이관**이다. 각 단계가 독립적으로 green 이어야 한다.

### 6.1 순서

| 단계 | 내용 | 게이트 |
|---|---|---|
| **S0** | `feature` 레지스트리 골격 + 태그→cobra/JSON 바인딩 + **레지스트리에 없는 기능을 세는 테스트** | 기존 동작 무변경 |
| **S1** | ① Compose 계열부터 이관 — `net.*` 9스텝(이미 `app` 경유) | `net up` 3체인 회귀 |
| **S2** | MCP `net_*` 를 레지스트리 소비로 전환 | 손작성 스키마 감소분 측정 |
| **S3** | ② Test 계열 — `tx`·`faucet`·`contract`·`verify` | CLI/MCP/DSL 동시 노출 확인 |
| **S4** | ③ Report 계열 — `status`·`report`·`logs` | |
| **S5** | 규칙 1 위반 21파일 정리 (`upgrade_run.go` 부터) | `cmd/` 가 `app` 만 import |
| **S6** | 규칙 위반 검사 테스트 (`cmd` import 화이트리스트) | 재발 차단 |

**S1 을 Compose 부터 하는 이유**: 9스텝이 이미 `app` 을 경유해서 **이관이 아니라 등록만** 하면 된다.
가장 싼 것으로 레지스트리를 검증하고, 비싼 것(S5)으로 간다.

### 6.2 F 계열(패밀리 기동)과의 관계

독립이다. F1~F6 은 **`net genesis`/`net start` 의 내부**를 바꾸고,
S0~S6 은 **그 기능이 표면에 노출되는 방식**을 바꾼다. 병행 가능하되,
`net start` 를 동시에 건드리는 S1 과 F3 은 순서를 정해야 한다 — **F3 을 먼저** 하는 편이 낫다
(페이즈 구조가 확정된 뒤 등록해야 다시 등록하지 않는다).

---

## 7. 이 설계가 답하지 않는 것

- **`chain` 과 `remote` 표면을 어떻게 할 것인가.** 셋 중 `net` 이 목표 표면이지만,
  `remote deploy`/`remote bootstrap` 은 오늘 wemix 를 띄우는 **유일한** 경로다.
  F5 로 `net` 이 그것을 덮은 뒤에야 정리할 수 있다.
- **DSL v2 의 `env` 선언이 Compose 기능과 1:1 인가.** [[dsl-v2-proposal]] 의 `env` 는 선언형이고
  레지스트리는 명령형이다. 대응 규칙을 정해야 한다.
- **`Deps` 가 커지는 문제.** 지금 4필드(Clock·Env·Driver + 암묵)인데 기능이 늘면 커진다.
  기능별 Deps 분리 vs 단일 Deps 유지는 S3 쯤에서 판단한다.
