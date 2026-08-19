# 관심사별 소유 모듈 · 3체인 실행 시뮬레이션

> [[layers]](layers.md) 는 **패키지가 어느 층인지**를 답한다. 이 문서는 그 다음 질문에 답한다 —
> **"이 관심사의 주인이 누구인가."** 그리고 세 체인을 실제로 기동할 때 모듈을 어떻게 타는지 추적한다.
>
> 실측 기준: 2026-08-18. 관련: [[family-bringup-design]](../family-bringup-design.md)

---

## 1. 진단 — 관심사마다 주인이 없다

레이어 문서를 쓰고 나서도 답이 안 되는 질문이 남는다: *"genesis 는 누가 책임지는가?"*
실측하면 이유가 분명하다.

| 관심사 | 만지는 패키지 수 | 실측 |
|---|---:|---|
| **genesis** | **17** | app · chains/{external,stablenet,wbft,wemix} · chainsetup · consensus/{poa,upgrade,wbft} · core/{bringup,driver,preflight,registry} · engine · mcp · netcompose · testspec |
| **key / nodekey** | **17** | chains/wemix/deploy · chainsetup · consensus/upgrade · core/{bringup,config,driver,genesis,hardfork,keyreg,keys,launchopt,session} · engine · keygen · netcompose · testspec · validatorset |
| **노드 생명주기** | **11** | app · chains/wemix/deploy · chainsetup · consensus/upgrade · core/{bringup,driver,hardfork,supervisor} · engine · mcp · netcompose |
| **config.toml 렌더** | 5 | chains/wemix/deploy · consensus/upgrade · core/bringup · engine · netcompose |

**노드를 관리하는 컴포넌트는 있다. 다만 11곳에 흩어져 있고, 그중 무엇이 "주인"인지 코드가 말하지 않는다.**
이것이 레이어를 지켜도 복잡도가 줄지 않는 이유다 — 레이어는 *방향*을 정하지 *책임*을 정하지 않는다.

---

## 2. 관심사 → 소유 모듈

**규칙: 관심사 하나에 소유자 하나. 다른 모든 곳은 소유자를 경유한다.**

체인 구성 5요소(바이너리·genesis·config·nodekey·keystore)는 원래 요구의 축이므로 그대로 쓴다.

| # | 관심사 | 소유 모듈 | 층 | 하는 일 | 하지 않는 일 |
|---|---|---|---|---|---|
| 1 | **바이너리** | `registry.Manifest.Binary` + 표면의 경로 해석 | L1/L6 | 어떤 실행파일인지 선언 | 빌드하지 않는다 |
| 2 | **배치·포트** | `serverset` → `place` → `portplan` | L1 | 인벤토리에서 노드별 host/포트 확정 | 파일을 쓰지 않는다 |
| 3 | **키셋(계정·nodekey)** | `engine.KeySource` | L4 | 키셋 확보(preset 사용 / 신규 생성) | 키를 *등록*하지 않는다 |
| 3b | **키 레지스트리** | `core/keyreg` | L1 | 실행이 쓰는 신원 등록·주소 대조·0600 | 키셋을 만들지 않는다 |
| 4 | **genesis** | `engine.GenesisSource` | L4 | 패밀리에 위임해 genesis(+부산물) 생성 | 파일을 쓰지 않는다 → Sink 로 넘긴다 |
| 4a | ↳ 패밀리 구현 | `consensus/{wbft,poa}` | L2a | wbft: extraData RLP / poa: 바이너리 호출 | 배치·포트를 모른다 |
| 4b | ↳ 공통 변형 | `core/genesis` | L1 | overlay 병합 · config override · fork 순서 검증 | 체인을 모른다 |
| 5 | **노드 config.toml** | `core/nodeconfig` | L1 | 파라미터 → TOML 렌더 (순수) | 어디에 쓸지 모른다 |
| 6 | **실행 argv** | `core/launchopt` | L1 | 다이얼렉트·모듈·오버라이드로 argv 조립 | 실행하지 않는다 |
| 7 | **타깃 물질화** | `core/provision.FileSink` | L1 | **타깃에 파일을 놓는 유일한 통로** (local/SSH) | 무엇을 쓸지 정하지 않는다 |
| 8 | **노드 프로세스** | `core/driver` | L1 | 기동·정지·datadir init (local/SSH) | 순서를 모른다 |
| 9 | **프로세스 추적** | `core/procman` | L1 | PID 테이블 · 검증된 종료 · 고아 보고 | 무엇을 띄울지 모른다 |
| 10 | **기동 순서·복구** | `core/supervisor` | L3 | 페이즈 실행 · 게이트 · 진단 · 재시도 · teardown | 액션의 *의미*를 모른다(주입) |
| 11 | **DSL 구문** | `testspec/spec`(분리 제안) | L1 | 파싱 · 스키마 · v1→v2 마이그레이션 (순수) | 실행하지 않는다 |
| 12 | **DSL 실행** | `testspec` 인터프리터 | L3 | 스텝·어세션 실행 · 값 바인딩 | 구문을 재해석하지 않는다 |
| 13 | **관측** | `core/collector` | L3 | tail · chainstate · bp참여 · reorg | 판정하지 않는다 |
| 14 | **판정·기록** | `core/session` | L3 | **아티팩트 레이아웃의 소유자** · 세션/환경/컴포지션 | 실행하지 않는다 |
| 15 | **네트워크 조립** | `netcompose` / `engine` | L4 | 위를 순서대로 엮는다 | 스스로 파일을 쓰지 않는다 |
| 16 | **유스케이스** | `app` | L5 | 유스케이스 1개 = 함수 1개 | 렌더링하지 않는다 |

### 지금 이 표를 어기는 곳

| 어기는 곳 | 무엇을 | 왜 문제인가 |
|---|---|---|
| `core/bringup` | genesis 를 직접 `os.WriteFile` | #7 Sink 우회 (레거시, 삭제 예정) |
| `app` | `topology.yaml` 직접 쓰기 | #16 이 파일 경로를 안다 |
| `consensus/upgrade` | 파일 쓰기 + 노드 기동 | L2 가 #7·#8 을 겸한다 |
| `chains/wemix/deploy` | 원격 파일 쓰기 + 기동 | 같은 이유 |
| `testspec` | 파서와 인터프리터가 한 패키지 | #11·#12 가 섞여 순수 파서가 L3 로 끌려 올라간다 |

---

## 3. 빠져 있던 모듈 — DSL 파서

`testspec.Parse` / `ParseV2` 는 존재하지만 **모듈로 서 있지 않다.** 한 패키지가 세 가지를 겸한다:

```
구문(pure)   spec.go · spec_v2.go · migrate.go · schema/
의미(pure)   resolve.go · binding.go · derived.go
실행(I/O)    interpreter.go · run.go · builtins.go · fault.go · logs.go · metric.go · assets.go
```

그 결과 **순수해야 할 파서가 L3 로 끌려 올라간다** — `testspec` 이 `collector`·`session`·`rpc`·`keyreg`·`accounts` 를 import 하기 때문이다.

### 제안: 4분할 — 이름이 무엇인지 말하게 한다

`testspec` 은 **2,998줄**(+`assert` 368)이고, 이름이 *무엇인지*를 말하지 않는다.
DSL 이 있으므로 **언어(구문)와 그것을 실행하는 엔진**이 갈라져야 한다.

| 새 패키지 | 층 | 담는 것 | 줄 | import |
|---|---|---|---:|---|
| `internal/dsl` | **L1** | `Spec` 모델 · `Parse`/`ParseV2` · `migrate` · 스키마 | 689 | 없음(순수)* |
| `internal/dsl/assert` | **L1** | 타입인지 비교 함수 | 368 | 없음(순수) |
| `internal/dsl/bind` | **L1** | 값 바인딩 · `$ref` 해석 | 259 | 없음(순수) |
| `internal/dsl/interp` | **L3** | 인터프리터 · 액션/어세션 · 빌트인 · 실행 | 2,050 | rpc · session · keyreg |

> **`engine` 이라는 이름을 다시 쓰지 않는다.** `internal/dsl/engine` 으로 두면
> `internal/engine`(테스트벤치 엔진)과 이름이 겹쳐, 이 프로젝트가 반복해 온
> "유사하면서 다른 이름"이 하나 더 는다. **`engine` 은 하나뿐이어야 한다.**

### DSL 은 인터프리터이고, 테스트벤치 엔진이 그것을 주도한다

이 구조는 **이미 코드에 있다** — 이름이 그것을 말하지 않았을 뿐이다.

```go
// internal/engine — 주도
func (e *engine) Run(ctx, specs [][]byte) {
    sess := NewSession(...)                          // 세션 개시
    for each raw {
        spec := dsl.Parse(raw)                       // ① 파싱          (L1)
        env  := sess.Environment(fp) or BuildEnv(…)  // ② 환경 준비/재사용
        st   := e.deps.RunSpec(ctx, spec, env, rec)  // ③ 순차 실행     (L3 인터프리터)
        rec.Status(st)                               // ④ 기록
    }
    sess.Save()                                      // 판정
}

// internal/engine/wire.go — 인터프리터를 엔진에 묶는 seam (현존)
func NewRunSpec(deps Deps) RunSpecFunc {
    interp := NewInterpreter(deps)
    return func(...) { return interp.Run(ctx, spec, env, rec) }
}
```

```
engine(L4) ──주도──▶ dsl/interp(L3) ──사용──▶ dsl(L1) · dsl/bind(L1) · dsl/assert(L1)
   └─ 파싱은 engine 이 직접 한다 (L4→L1 은 하향이라 층 위반이 아니다)
```

\* **구문을 순수하게 만드는 데 필요한 변경은 하나뿐이다**: `Spec.Fingerprint()` 가
`session.Fingerprint`(L3) 대신 `string` 을 반환하고 호출자가 변환한다.
지금 구문이 L3 에 묶인 유일한 이유가 **그 타입 하나**다 (실측).

> **분류 정정**: 앞서 `derived.go`(251)를 "의미(순수)"로 적었으나 `rpc`·`session` 을 6곳에서
> 쓴다 — **실행**이다.

### 레지스트리는 import 가 아니라 주입이다

엔진(L3)이 기능 레지스트리(L5)를 import 하면 상향이라 불가능하다.
그런데 **이미 올바른 모양이다** — `interpreter.go` 가 `Registry`·`Action`·`Assertion`·`Deps` 를
**스스로 정의**하고, L5 가 구현체를 만들어 `Deps` 로 넘긴다. 값 전달이므로 층을 거스르지 않는다.

따라서 "세 표면이 레지스트리를 읽는다"가 아니라 **"CLI·MCP 는 읽고, 엔진은 주입받는다"**이다.

**효과 3가지**
1. `chainbench validate`(오프라인 검증)가 실행 스택 전체를 링크하지 않는다.
2. 파서를 fuzz 하기 쉬워진다([[go-code-quality-guidelines]] "파서는 fuzz").
3. **DSL 문법이 실행과 독립적으로 버전을 갖는다** — v2 스키마가 정본이라는 T7.8 결정이 구조로 드러난다.

---

## 4. 3체인 실행 시뮬레이션

`chainbench net up --chain <X> --binary <B> --server local` 한 줄이 모듈을 어떻게 타는지.

### 4.1 공통 골격 (세 체인 동일)

```
L6  cmd/chainbench net up                       플래그 바인딩만
L5  app.NetUp                                   9개 스텝을 순서대로
     │
     ├─ NetNew        → netcompose.New          → external.ResolveChain ─→ registry.ChainPlugin
     ├─ NetAllocate   → app.ResolveServer       → serverset.Placement   (#2 배치·포트)
     │                  netcompose.Allocate     → place.Allocate → portplan.Plan
     ├─ NetKeys       → netcompose.Keys         → engine.KeySource      (#3 키셋)
     │                                            └ keys.LoadPreset | keygen.GeneratePreset
     ├─ NetGenesis    → netcompose.Genesis      → engine.GenesisSource  (#4 genesis)
     │                                            └ ★ 패밀리 분기 ★
     │                                          → core/genesis (overlay·override·fork검증)
     │                                          → provision.FileSink.Write            (#7)
     ├─ NetConfig     → netcompose.Config       → nodeconfig.Generate   (#5) → Sink   (#7)
     ├─ NetLaunchOpts → netcompose.LaunchOpts   → engine.NodeLaunchArgs → launchopt   (#6)
     ├─ NetProvision  → netcompose.Provision    → Sink.Exists (upload-if-absent)      (#7)
     ├─ NetInit       → netcompose.Init         → driver.Initializer.InitDatadir      (#8)
     └─ NetStart      → netcompose.Start        → ★ 패밀리 분기 ★                     (#10)
                                                → driver.Launch (#8) → procman (#9)
L3  core/session 이 전 과정의 스텝 스탬프를 기록                                       (#14)
```

**분기점은 정확히 2개다** — genesis 생성과 기동 순서. 나머지 7스텝은 세 체인이 같은 코드를 탄다.

### 4.2 go-stablenet · go-wbft (family `wbft`)

```
[genesis]  engine.PresetGenesisSource
             → genesis.Build(plugin, Inputs{Validators, BLSKeys, ExtraData, Alloc})
             → plugin.Family().BuildGenesis  ─→ consensus/wbft.BuildGenesis
                                                 (검증자셋 → extraData RLP 치환)
             → GenesisArtifacts{Genesis, Extra: nil}

[기동]     registry.Family().BringUpPhases(nodes)
             → [{Name:"all", Nodes:[1..N], Actions:nil}]        ← 한 페이즈
           supervisor.BringUp
             → Launch(plan, nil)   ← nil = 전체 동시
             → HealthGate(블록 전진)
```

라이브 확인(2026-08-18): stablenet 97→110→122, wbft 24→36→48, 4노드 동기, api 9/9.

### 4.3 go-wemix (family `poa`)

```
[genesis]  engine.WemixGenesisSource                             ← F4 신규
             → poa.PrepareTemplate(template, chainID, coinbase)  ← 현 BuildGenesis 재배치
             → poa.Config{
                 Members:  preset 노드마다 {Addr: address,
                                            ID: "0x"+publicKey,   ← 이미 idv5
                                            IP/Port: place 배치값,
                                            Bootnode: topology}
                 Accounts: 계정 + maintenance
                 Env:      정책값 }
             → poa.Config.Validate()      (부트노드 정확히 1개 · idv5 128hex)
             → poa.GenerateGenesis(Runner, binary, config.json, template)   ← 바이너리가 생성
             → GenesisArtifacts{Genesis, Extra:{"wemix-config.json": ...}}
             → Sink.Write ×2                                     (#7)

[기동]     registry.Family().BringUpPhases(nodes)
             → [{"boot", [1], ["deploy-governance","init-etcd"]},
                {"rest", [2..N], nil}]                           ← 두 페이즈
           supervisor.BringUp
             phase "boot":
               Launch(plan, [1])                                 (#8)
               HealthGate(IPC 준비)
               Action("deploy-governance", node1) ─주입→ poa.DeployGovernance(Runner,…)
               Action("init-etcd",         node1) ─주입→ poa.EtcdInit(Runner,…)
             phase "rest":
               Launch(plan, [2..N])                              (#8)
               LeaderGate(etcd 리더, window = JoinWindow(N))
               HealthGate(블록 전진)
```

라이브 확인(수동 절차, 2026-08-18): 거버넌스 5컨트랙트 배포 → etcd 4멤버 → 238→253→268,
**4노드 sealing 로테이션(5/6/5/4)**.

### 4.4 세 체인의 차이가 사는 곳

| 무엇 | 어디 | 상위가 아는가 |
|---|---|---|
| extraData RLP vs config.json members | `consensus/{wbft,poa}` (L2a) | ❌ |
| 1페이즈 vs 2페이즈 | `Family.BringUpPhases` 반환값 (데이터) | 순서만 안다 |
| deploy-governance / init-etcd | `Action(name)` 로 주입 | **이름만** 안다 |
| p2p 대역 2칸 vs 3칸(etcd peer+client) | `Family.PortReservation()` | 숫자만 안다 |
| `--networkid` 방출 | `launchopt` 다이얼렉트 | ❌ |

**L3 supervisor 는 끝까지 wemix 를 모른다.** `Action("deploy-governance")` 를 언제 부르고,
얼마나 기다리고, 실패를 어떻게 분류할지만 안다. 이것이 `LeaderGate` 가 이미 쓰는 계약과 같다.

---

## 5. [[layers]] 보정 사항

이 문서를 쓰며 드러난, 레이어 문서가 놓친 것.

| # | 보정 |
|---|---|
| B1 | **`testspec`(2,998줄)을 `dsl`(L1 구문) · `dsl/assert`(L1) · `dsl/bind`(L1) · `dsl/interp`(L3)로 4분할.** 이름이 무엇인지 말하지 않는 것도 문제다 |
| B2 | **노드 생명주기에 명시적 소유자를 세운다** — `driver`(#8) 실행 / `procman`(#9) 추적 / `supervisor`(#10) 순서. 지금은 11곳이 각자 부른다 |
| B3 | 레이어표에 **관심사 열**을 붙인다. 패키지 이름만으로는 "genesis 담당"이 안 보인다 |

---

## 6. 착수 순서

> **작업 순서와 상태는 [[chainbench-worklist]](../chainbench-worklist.md) §1g 에서 관리한다.**
> 이 문서는 *무엇을 왜* 를 정하고, 트래커는 *언제 어디까지* 를 기록한다.
> 순서를 여기 두면 두 곳이 갈라진다.

> 이 문서가 트래커에 더한 것: **B1(`testspec` 4분할)** — F 계열과 독립이라 병행 가능하다.
