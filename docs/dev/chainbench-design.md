# chainbench 설계 문서 — 구조 · 인터페이스 · 데이터 모델

> **[정본]** 인터페이스·데이터 모델.
> 이 문서는 *무엇을 만들어야 하는가* 를 정한다. 설계 제안이 여기와 어긋나면 **제안을 고친다.**

> 지위: 구현 전 확정 설계(HOW). 근거: `chainbench-requirements-review.md`(요구/결정), `chainbench-refactoring.md`(기존→목표 매핑).
> 원칙: 인터페이스 우선 · 하위호환 병존 · 소유권 단일화 · 취소(ctx) 우선 · 테스트는 항상 직렬(§B-4).
> 표기: Go 시그니처는 계약(contract)이며 필드/에러는 구현 시 미세조정 가능. 기존 타입(`config.Values`, `driver.Driver`, `node.NodeSet`, `registry.ChainPlugin`, `accounts.AccountProvider`, `genesis.MergeOverride`, `procman.Manager`, `obs.Bus`)은 그대로 참조·재사용.

---

## 1. 목표 · 비기능요구 · 설계원칙

**목표(요구 37):** shell 스위트를 golang으로 이관하되, 정의서(JSON)+config로 체인을 구성·검증하고, 확장성·가독성·안정성·복구·명확한 결과추출로 런타임 문제를 조기 발견·디버깅한다.

**비기능요구**
- **확장성**: 신규 체인/신규 테스트를 코드 최소 변경으로 추가.
- **안정성/복구**: 환경 구성 실패를 인식·분류·복구(특히 etcd), hang 금지(모든 대기 timeout).
- **가독성**: 테스트=데이터(정의서), 결과·근거가 아티팩트로 남음.
- **비침습 디버깅**: 노드 성능에 영향 없는 out-of-process 수집(요구 35).

**설계원칙**
1. **인터페이스 우선** — 신규 패키지는 인터페이스를 먼저 확정(feature-spec의 AC와 1:1).
2. **소유권 단일화** — 파일/포트/PID/세션상태는 소유자 1개, 나머지는 채널·불변데이터로 접근(락 표면 최소).
3. **취소 우선** — 모든 장기작업 `context.Context` 종속, 실패 조기전파+자원회수.
4. **하위호환 병존** — 신규 경로를 별도로 세우고 기존 pipeline/e2e는 유지, 점진 이관.
5. **테스트 직렬** — 동시성은 "한 환경 내 N노드" 처리에만, 테스트 실행은 항상 직렬.

---

## 2. 대상 아키텍처 (현 6-layer + 신규)

```
┌ Entrypoints ─ cmd/{chainbench, chainbench-dashboard, chainbench-mcp}
│                     │ (test 정의서 실행 커맨드)
├ Session ──────  core/session      ← .chainbench/<session>/ 정본 소유 [NEW]
│                     │
├ Test 해석 ────  testspec          ← 정의서 파싱·검증·해석(pre→steps→assert→post) [NEW]
│                     │  supervisor  ← 헬스 게이트·etcd 리더게이트·복구 [NEW]
├ Orchestration  pipeline/{setup,verify}  ← 동시화(REFACTOR)   place ← 배치·포트 [NEW/CONSOLIDATE]
│                     │  collector  ← 라이브로그·chainstate 수집 [NEW]   keyreg ← 키 레지스트리 [NEW]
├ Plugins ──────  registry(+capability) · chains/*        [EXTEND]
├ Consensus ────  consensus/{poa,wbft,upgrade}            [KEEP/REFACTOR]
├ Core prim ────  driver · procman(+etcd관측) · genesis · node · rpc · accounts · obs   [KEEP/EXTEND]
└ Surfaces ────  mcp(결과연동) · dashboard(세션소비)      [EXTEND]
```

의존 방향: 위→아래 단방향. 신규(session/testspec/supervisor/place/keyreg/collector)는 core primitive에 의존하되 서로는 인터페이스로만 결합.

---

## 3. 패키지별 인터페이스 (계약)

### 3.1 `core/session` — 아티팩트 정본 [NEW]
`.chainbench/<session>/` 트리(§D-1)의 유일 소유자. 모든 경로 파생·기록은 여기로 위임(소유권 단일화).
```go
package session

type Session interface {
    ID() string                        // "<UTC-YYYYMMDD-HHMMSS>"
    Root() string                      // ".chainbench/<session>"
    Keys() keyreg.Registry             // 세션 키 레지스트리(§3.5)
    // 재사용: 동일 fingerprint 환경이 이미 있으면 반환(ok=true)
    Environment(fp Fingerprint) (Environment, bool)
    NewEnvironment(fp Fingerprint) (Environment, error)
    Test(seq int, id string) TestRecord
    Save() error                       // session.json
}

type Environment interface {
    ID() string                        // env-id = "env-"+fingerprint[:12] (hex 앞 12자, 짧은 폴더명, §D-2.4·L5)
    Dir() string
    Fingerprint() Fingerprint          // 전체 해시(env.json 기록) — env-id는 그 앞 12hex 축약
    PopulateNodeTable(ns node.NodeSet) // BringUp 결과(ns)로 node table 채움 → Save 전
    Nodes() []node.Node                // node table (endpoint 해석 근거)
    Resolve(selector string) (node.Node, error)   // "node7"|"bp1"|"bp:any"|"en:0"
    ResolveEach(selectors []string) ([]node.Node, error)
    DataPath() string                  // 노드 로그가 쌓이는 실제 경로(§3.6)
    LogPath(nodeName string) string    // environments/<env>/logs/<node>.log
    ChainstateDir() string
    Save() error                       // env.json (설정 fingerprint + node table + dataPath)
}

type TestRecord interface {
    Dir() string
    SetEnvRef(envID string)
    Spec(raw []byte)                   // spec.json
    Step(i int, r StepResult)          // steps.json (누적)
    Assert(r AssertResult)             // assert.json (누적, provenance 포함)
    Status(s TestStatus)               // status.json
    PostAction(r PostResult)           // postaction.json (판정과 독립)
}

// Fingerprint = 정의서 선언값에서 파생(§D-2.4). session은 문자열 타입만 소유하고,
// 산출은 testspec이 담당한다(의존 방향: testspec → session, 순환 없음).
type Fingerprint string   // = hex(sha256(canonical)) 64자. env.json의 "fingerprint" 필드에 전체 기록.

// TestStatus는 typed const enum(status.json의 result). 매직 문자열 금지.
// 주의: 문자열 const는 iota처럼 타입이 전파되지 않으므로 각 줄에 타입을 반복한다.
type TestStatus string
const (
    StatusPass    TestStatus = "pass"
    StatusFail    TestStatus = "fail"
    StatusBlocked TestStatus = "blocked"
    StatusSkip    TestStatus = "skip"
)
```
**폴더명 길이(L5):** `Fingerprint`는 sha256 hex **64자**(전체는 env.json `fingerprint`에만 기록). **폴더명 env-id**는 `"env-"+앞 12hex` = **16자**(예: `env-a1b2c3d4e5f6`, 48-bit → 한 세션 내 소수 env에서 충돌 무시가능). 최장 경로 `.chainbench/UTC-YYYYMMDD-HHMMSS(19)/environments/env-xxxxxxxxxxxx(16)/logs/<node>.log` 는 각 컴포넌트 <255(NAME_MAX)·전체 <4096(PATH_MAX)로 안전. **전체 해시를 폴더명으로 쓰지 않는다**(경로 초과 방지).

### 3.2 `testspec` — 정의서 파싱·해석 [NEW · testkit Go-func 대체]
```go
package testspec

// Spec = 파싱·검증된 테스트 정의서(전 필드는 §4.3 스키마).
type Spec struct { /* SchemaVersion, Id, ApplicableChains, Chain, Topology, Hardforks, Placement,
                      DefaultOn, PreActions, Steps, Assertions, PostActions, Timeouts */ }

func Parse(raw []byte) (Spec, error)               // 필수/옵션 검증 + JSON schema
// Fingerprint는 precedence(flag>config>default) 적용된 **선언값 전체**를 해싱한다:
// binaries+genesis+config+topology+hardforks+placement (§D-2.4). config는 인자(resolved),
// 나머지는 수신자 s(Topology/Hardforks/Placement/Chain)에서 취한다. 체인 미접촉.
func (s Spec) Fingerprint(resolved config.Values) session.Fingerprint  // (§D-2.4)
func (s Spec) Get(dotPath string) (any, bool)      // 닷경로(a.b.c) 리졸버, multiple "," 파서

// Deps: 해석기가 스텝/검증을 수행하려면 반드시 필요한 협력자(생성자 주입).
type Deps struct {
    Keys      keyreg.Registry          // 서명 키(§3.5)
    Accounts  accounts.AccountProvider // tx 서명·전송
    RPC       func(url string) *rpc.Client
    Collector collector.Collector      // log 검증·WaitLog(§3.6)
    Actions   Registry                 // 액션·검증 레지스트리(**주입** — 전역 상태 배제, 테스트 격리)
}
// Registry: 액션/검증 함수 레지스트리. **패키지 전역이 아니라 인스턴스로 주입**(설계원칙 2·소유권 단일화).
// 내장 세트 + 체인별 확장 지점(§9)을 담고, 테스트마다 독립 인스턴스로 격리 가능.
type Registry interface {
    Action(name string) (Action, bool)
    Assertion(name string) (Assertion, bool)
    RegisterAction(name string, a Action)         // ensureChain/ensureStaker/tx/wait/unstake/faucet(K)/
                                                  // deployContract/registerContract(F)/
                                                  // stopNode/startNode/restartNode/partition/healPartition(E)/chainMigrate ...
    RegisterAssertion(name string, a Assertion)   // Equal/NotNil/EqualWith/Len/EqualHashAt ...
}
func NewRegistry(withBuiltins bool) Registry      // 내장 액션·검증 세트로 초기화(전역 init 없음)
// Interpreter: 실행 중인 Environment에 대해 Spec을 원자적 스텝으로 수행. Deps는 생성자에서 주입.
func NewInterpreter(d Deps) Interpreter
type Interpreter interface {
    Run(ctx context.Context, s Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error)
}

// preActions·steps·postActions 는 모두 Action, assertions 는 Assertion. **preActions는 idempotent 가드**
// (RPC로 상태 확인 후 없으면 수행). Do/Check 은 ActionCtx/AssertCtx로 Deps·env·rec에 접근.
type Action    interface{ Do(ctx context.Context, ac *ActionCtx) error }               // atomic(부분성공 없음)
type Assertion interface{ Check(ctx context.Context, ac *AssertCtx) (AssertResult, error) }
type ActionCtx struct { Env session.Environment; Deps *Deps; Rec session.TestRecord; Args map[string]any }
type AssertCtx struct { Env session.Environment; Deps *Deps; On []node.Node; Spec map[string]any }
// 액션·검증 등록은 **주입된 Registry**(위)로 수행(전역 함수 아님). 노드 생명주기·fault 액션
// (stopNode/startNode/restartNode/**partition·healPartition**/chainMigrate)은 destructive/fault 테스트용
// (WBFT-007/008 쿼럼, NODE-005 싱크복구, **F8 AC-2 인위적 분기**=partition으로 유발). partition은
// network_partition(MCP/driver)로 실현. **faucet**(K): keyreg 서명키(op1/acctA)에 gas 자금 공급 —
// genesis alloc으로 미리 funded되지 않은 신규 서명키는 faucet 스텝으로 충전 후 tx. **deployContract/
// registerContract**(F): 바이트코드 배포→주소 캡처→(bp 등록 필요 시) 컨트랙트에 노드정보 등록.
// **타입 안전(Q1):** ActionCtx.Args·AssertCtx.Spec 의 `any`는 DSL 경계값이며, Parse가 스키마 검증 시
// 각 액션/검증이 요구하는 필드로 **조기 타입확정**(예: gas int, expected typed)해 강타입 파생값으로 좁힌다.
```

### 3.2b 스텝 값 바인딩 (step value binding) — DSL 문법 확정
정의서는 원래 **1-shot read** 만 표현할 수 있었다(각 어세션이 값을 읽고 리터럴 기대값과 비교). 그래서
"read A 값을 저장했다가 B와 비교"(교차-call 비교·거버넌스 다단계·방금 보낸 tx의 영수증 확인)는
표현 불가였다. 이를 최소 문법으로 연다.

```jsonc
{"steps":[
   {"read":{"source":"call","to":"0x..1001","data":"0x18160ddd","save":"totalSupply"}},
   {"sendTx":{"from":"0x..","to":"0x..","value":"1","save":"sent"}}],
 "assertions":[
   {"assert":"balanceAt","address":"0x..","compare":"LessOrEqual","expected":"$totalSupply"},
   {"assert":"txStatus","hash":"$sent","expected":"0x1"}]}
```

**규칙(3개뿐)**
1. **`save`** — 어떤 액션이든 `"save":"<name>"` 을 선언하면 그 액션의 결과가 `<name>` 으로 바인딩된다.
   결과는 `ActionCtx.Value`(액션이 설정)이며, **설정하지 않은 액션은 tx hash 가 대신 바인딩**된다
   — 그래서 `sendTx` 는 바인딩을 전혀 모르면서도 참조 가능해진다.
2. **`$name`** — 문자열 **전체**가 참조면 **타입을 보존**한 채 값으로 치환된다(숫자는 숫자로 남아
   비교자가 그대로 쓴다). **`${name}`** 은 긴 문자열 안에 텍스트로 보간된다(calldata 조립).
   **`$$`** 는 리터럴 `$`.
3. **미바인딩 참조는 오류** — 조용한 빈 문자열이 아니다. 오타가 `""` 와 비교되어 통과하는 일이 없다.

**스코프**: 바인딩은 **한 Spec 실행(=한 테스트) 단위**로, `Interpreter.Run` 호출 안에서만 산다.
테스트 간 누수 없음 · 인터프리터는 무상태 유지(설계원칙 2).

**치환 시점**: 각 액션/어세션 **디스패치 직전**. 파싱된 Spec 은 불변으로 두고 사본에 치환하므로,
같은 Spec 을 두 번 실행해도 동일하게 동작한다.

**`read` 액션**: `{"read":{"source":"<어세션 이름>", ...}}` 은 어세션의 리더를 **그대로 재사용**해
값을 읽고 `save` 한다(`call`/`balanceAt`/`codeAt`/`nonceAt`/`blockNumber`/`chainId`/`peerCount`/
`baseFee`/`estimateGas`/`txStatus`). 어휘를 두 벌 만들지 않기 위해 리더는 단일 소스다.

**오프라인 검증**: `testspec.Unresolved` 가 액션·어세션 이름과 함께 **참조도 검사**한다 —
실행 순서(pre→steps→assert→post)를 따라가며 **더 앞에서 `save` 되지 않은 `$ref` 는
`ref:<name>` 으로 보고**한다. `chainbench validate` 가 이를 실행 전에 잡는다.

### 3.3 `supervisor` — 헬스 게이트·복구 [NEW · runGovHandoff 재시도 대체]
```go
package supervisor

type Supervisor interface {
    // **기동 소유자(L6):** setup은 Plan 산출 + 동시 provision/launch **프리미티브**만 제공하고,
    // 실제 노드 기동·헬스 게이트(etcd 리더 준비, 포크 도달)·백오프 복구는 supervisor가 오케스트레이션한다.
    BringUp(ctx context.Context, plan setup.Plan, opts Options) (node.NodeSet, Diagnosis, error)
    // Teardown: 프로세스 종료(SIGTERM→SIGKILL, procman)로 **내장 etcd 포함 노드 전체 종료**.
    // etcd는 별도 프로세스가 아니라 노드 내장이므로 프로세스 종료 시 함께 종료된다(S2).
    Teardown(ctx context.Context, ns node.NodeSet, opts TeardownOpts) error
}
type TeardownOpts struct {
    RemoveDataDir bool          // 종료와 **별개 기능**(S2): 재-셋업/디스크관리 시 datadir 삭제.
                                // procman이 노드별 {PID, datadir}를 추적해야 정확한 삭제 가능(§procman EXTEND).
    Grace         time.Duration // SIGTERM→SIGKILL 유예
}
type Options struct {
    LeaderGate   bool          // producer etcd 리더 준비 폴링(C-etcd)
    AlignJoinGap bool          // etcd 조인 슬롯에 시작 정렬(L7). gap은 supervisor가 **클러스터 크기 N에서 파생**
                               // (C-etcd: sz≤11→7s, ≤23→11s, ≤41→17s, else 23s) — 호출자가 고정값으로 넘기지 않는다.
    MaxAttempts  int
    Backoff      func(attempt int) time.Duration
    // 하드포크 type-2(동일체인, pre≠post 바이너리, §D-2.8): fork 블록 도달 *전에*
    // 해당 노드 바이너리를 fork-aware로 교체하도록 스케줄. type-1(체인 업그레이드=handoff)은
    // producer/successor가 처음부터 서로 다른 바이너리이므로 plan.Nodes[].Binary로 해결.
    ForkSwaps    []ForkSwap    // {nodeSelector, atBefore(fork), toBinary}
}
type ForkSwap struct { Node string; Fork string; ToBinary string }
type Diagnosis struct {
    OK          bool
    Mode        FailureMode   // 분류된 실패모드(아래 typed const)
    Detail      string        // 실제 로그 라인(원인 은닉 금지)
    ProducerLog string        // 최종 실패 시 producer 로그 tail (세션 보존)
}
// FailureMode는 typed const enum(매직 문자열 금지). 라벨은 String()으로.
type FailureMode int
const ( EtcdJoinFailed FailureMode = iota; EtcdStale; ForkNotCrossed; QuorumLost; RPCUnready )
```
> etcd: `etcdInit` 후 **리더 준비 확인 게이트** + **재기동 시 datadir 정리** + **gap 정렬**로 "바로 연결"(C-etcd §요약). 실패는 분류·기록.
> **stale etcd의 실체(S2):** etcd는 노드 내장이므로 프로세스 종료 시 함께 종료된다 — "살아있는 etcd 정리"는 불필요. 문제는 **같은 datadir로 재기동** 시 남은 클러스터 상태(`cannot fetch cluster info`)이며, 해법은 **재-셋업 전 datadir 삭제**(`Teardown{RemoveDataDir:true}`, 종료와 별개 기능).
> 하드포크는 2종(§D-2.8): **type-1 체인 업그레이드**는 노드별 바이너리 집합(plan.Nodes[].Binary)으로, **type-2 동일체인**은 `ForkSwaps`로 fork 블록 전 바이너리 교체를 supervisor가 오케스트레이션.

### 3.4 `core/place` — 통합 배치·포트 [NEW · portplan+topology CONSOLIDATE]
```go
package place

type Mode int
const ( LocalStepped Mode = iota; LocalOSAssigned; RemotePerHost )

type NodeReq struct { Name string; Role node.Role; Sync string; Binary string }
type NodePlacement struct {
    Name string; Host string           // local=127.0.0.1 / remote=서버IP
    Ports portplan.Ports               // P2P·**Etcd(=P2P+1, wemix 내장 파생·예약)**·HTTP·WS(=HTTP+1)·Auth(=HTTP+2)
    DataPath string
}
type Allocator interface {
    // Allocate는 배치 전 **용량 사전검증(fail-fast)** 후 배치한다:
    //  - min: BFT 블록생성 최소(validators ≥ 4) 미달 → 오류
    //  - max: local=가용 포트 대역, remote=Σ(서버 슬롯, 서버당 3~4노드) 초과 → 오류
    Allocate(reqs []NodeReq, mode Mode, cap Capacity) ([]NodePlacement, error)
}
type Capacity struct { MinValidators int; Hosts int; SlotsPerHost int; PortBandSize int } // C2 Environment 불변식
```
- **용량 검증(C·요구 5)**: 노드 수가 **min(≥4 BFT)·max(서버×슬롯 or 포트대역)** 를 벗어나면 배치 이전에 실패. 원격 자원 낭비 방지(§0-2-3). `topology`가 개수, `place`가 물리 용량 담당.
- **포트 타입(L1):** etcd는 wemix **바이너리 내장**이고 그 포트는 바이너리가 `P2P+1`로 파생하므로, launch flag가 아니라 **예약 대상**이다. 이 예약은 기존 `portplan.Ports`(P2P·Etcd·HTTP·WS·Auth, p2pStep≥2·rpcStep≥3로 **P2P 밴드와 RPC 밴드 분리** → 충돌 없음)에 이미 존재하므로 place는 이를 재사용(consolidate)한다. `node.Endpoints`(P2P/HTTP/WS/Auth/**Metrics**, etcd 필드 없음)는 별개 타입 — node table 직렬화용이며 etcd 예약은 `portplan.Ports`가 소유. (place가 두 타입의 필드차이(Metrics↔Etcd)를 흡수.)
- `LocalStepped`: index 기반 결정적 스텝. `LocalOSAssigned`: `:0` 바인드 후 회수(고정포트 이중바인드 근절, E-3-1). `RemotePerHost`: 동일 포트 + 서버별 IP(요구 16).

### 3.5 `core/keyreg` — 키 레지스트리 [NEW · keys+deploy/keys CONSOLIDATE]
```go
package keyreg

type Key struct { Name, Address string; Private []byte; BLS, PoP []byte }
type Source int
const ( Random Source = iota; LocalFile; RemoteDownload )

// BLSDeriver: BLS 공개키·PoP 생성 seam. chainbench에 네이티브 BLS 크립토가 없어
// **외부 `bootnode` 바이너리에 위임**(§2b·D). 주입식(부재 시 명확 오류) — Random 소스로
// validator 키를 만들 때 이 Deriver로 BLS/PoP를 채운다(genesis bp-신원 등록에 필요).
type BLSDeriver interface { Derive(ctx context.Context, private []byte) (bls, pop []byte, err error) } // 외부 bootnode 프로세스 실행 → ctx로 timeout

type Registry interface {
    // opts로 BLSDeriver 주입. Random+validator면 BLS/PoP 생성, 아니면 생략.
    // ctx: RemoteDownload는 SSH 네트워크 I/O이므로 취소·timeout 전파(첫 인자).
    Ensure(ctx context.Context, name string, src Source, ref string, opts EnsureOpts) (Key, error) // 생성/복사/다운로드 → 세션 keys/
    Get(name string) (Key, bool)                          // 인메모리 조회 — ctx 불필요
    UploadTo(ctx context.Context, fp driver.FileProvisioner, names []string, remotePath string) error // 랜덤키→remote
}
type EnsureOpts struct { NeedBLS bool; BLS BLSDeriver } // NeedBLS면 BLS 필수 — Deriver 없으면 오류(누출 방지)
```
- 모든 키(노드키·서명키)를 `keys/<name>/`에 이름매핑 → op1/bp1 매핑(§D-2.2). remote 기존키는 다운로드, 랜덤키 remote 사용 시 업로드(요구 17).

### 3.6 `core/collector` — 라이브 로그·chainstate 수집 [NEW · probe+obs 확장]
```go
package collector

type Collector interface {
    Start(ctx context.Context, env session.Environment) error   // 노드별 tail + chainstate goroutine 기동
    WaitLog(ctx context.Context, nodeName, pattern string, timeout time.Duration) (LogMatch, error)
    Snapshot() Chainstate                                       // blocks/bp참여/sync/peers/forks(§D-2.5)
    Stop() error
}
type LogMatch struct { File string; Lines [2]int; ByteOffset int64; Text string }  // provenance(§D-2.6)
```
- **append-only tail**(일회성 복사 아님) → 누락 방지. `WaitLog`로 완결성/가독성(§D-2.7). out-of-process·버퍼·레이트리밋(35). remote는 SSH tail.
- **백프레셔 정책(O7):** 노드 무영향(35)이 최우선이므로 수집기는 **노드를 절대 블로킹하지 않는다**. 두 스트림을 분리 — **① 로그 tail**은 손실 불가(검증 근거)이므로 버퍼 초과 시 **디스크로 스풀**(드롭 금지, "누락 0" 유지). **② chainstate 폴링**은 파생 지표이므로 버퍼 초과 시 **최신값으로 병합/드롭** 허용(카운터로 드롭 수 기록). 어느 경우도 노드 프로세스는 대기하지 않는다.

### 3.7 확장: `registry.ChainPlugin` capability [EXTEND]
```go
// 기존 ChainPlugin에 선택적 capability 인터페이스를 type-assert로 부가(하위호환).
type Capabilities interface {
    NodeComposition() NodeComposition   // 체인별 N노드 구성법(요구 2): etcd/ncp/staking 등 초기화 훅
    SupportedForks() []string           // 이 체인이 아는 하드포크 이름(요구 26)
    SupportedAssertions() []string      // "istanbul"|"wemix" 등 검증 네임스페이스
    TestCapabilities() []string         // 테스트가 require 할 수 있는 기능 태그(요구 3)
}
```
- 해석기는 `Spec.ApplicableChains`/기능요구와 대조해 **미적용은 SKIP**(요구 3, D-3).

### 3.8 genesis 소싱 — 별도 모듈, 4가지 모드 [KEEP · `core/genesis`] (요구 7,17,25 · L3)
genesis 합성은 **독립 모듈**(`core/genesis`)이 소유하며, 정의서는 `chain.genesis`/`genesisOverlay`로 모드를 선택한다. 4가지:
| 모드 | 의미 | 기존 함수 |
|------|------|-----------|
| **① existing** | 이미 존재하는 genesis 파일을 그대로 사용 | 파일 bytes 로드(빌드 없음) |
| **② build** | 파라미터로 직접 생성 | `genesis.Build(plugin, Inputs)` → 내부에서 `plugin.Family().BuildGenesis(template, params)` |
| **③ template+override** | 템플릿을 얹어 수정 | `Build`(템플릿) + `MergeOverride`/`ApplyConfigOverrides`/`SetConfigSection` |
| **④ upgrade-inherit** | wemix 체인이 쓰던 genesis를 그대로 받아 업그레이드용 항목만 추가 | `MergeOverride(wemixGenesis, upgradeOverlay)` (=`--genesis-overlay`) |
> 주의: `BuildGenesis`는 **`ConsensusFamily`의 메서드**(`registry.go:46`)이고, genesis 패키지의 진입점은 **`genesis.Build`**(`genesis.go:38`)다 — 이름 혼동 금지. 오버레이는 항상 새 바이트 반환(원본 불변, E-3-3).

### 3.9 역할 용어집 (요구 4 · L2)
**단일 도메인 어휘 `bp`/`en`/`boot`를 정의서와 아티팩트(env.json) 전반에서 일관 사용**한다. 코드 `node.Role` enum("validator"/"endpoint"/"boot")은 **내부 식별자**이며, session이 env.json 기록 시 도메인 용어로 직렬화한다(단일 소유자에서 1회 변환).
| 도메인 용어 (정의서·env.json) | 코드 enum (node.Role, 내부) | 의미 |
|-------------------------------|-----------------------------|------|
| **`bp`** | `RoleValidator`("validator") | **block producer** — 블록 생성·검증(staking 선정). BFT 코드가 "validator"라 부를 뿐 동일 역할 |
| **`en`** | `RoleEndpoint`("endpoint") | **endpoint** — 비생성 RPC/싱크 노드 |
| **`boot`** | `RoleBoot`("boot") | bootnode — 부트스트랩(wemix governance 배포) |
> **왜 `bp`/`en` 단축어로 통일하는가:** wemix4 도메인의 표준 명칭이고 대칭적(2글자)이라 정의서 가독성이 높다. `bp`↔`validator`는 약어↔확장이 **아니라** 도메인 어휘와 BFT-코드 어휘의 **동의어**다 — 코드값 "validator"에 맞추면 `en`↔`endpoint`(약어↔확장)와 짝이 안 맞아 비대칭으로 보였다. 해결: **표층(정의서·아티팩트)은 `bp`/`en`으로 통일, enum은 내부에만.**
> 입력 관용: topology 파서는 `"en"`·`"endpoint"` 둘 다 받아 `RoleEndpoint`로 정규화(topology.go:60-61). `node.RoleEN`("en")은 **dead code** → 제거(refactoring §2).
> **구현 주의:** `node.Node.Role`의 json 태그는 enum값("validator"/"endpoint")을 낸다. 따라서 env.json은 `node.Node`를 **그대로 marshal하지 않고**, session이 `RoleValidator→"bp"`·`RoleEndpoint→"en"`·`RoleBoot→"boot"`로 매핑한 **전용 레코드**로 기록한다(위 "1회 변환"의 실체).

---

## 4. 데이터 모델 (아티팩트 스키마)

### 4.1 `session.json`
```jsonc
{ "id":"UTC-20260804-...", "command":"chainbench test --suite gov",
  "startedAt":"...", "tests":[{"seq":1,"id":"GOV-003","env":"<env-id>","status":"pass"}],
  "summary":{"pass":10,"fail":1,"blocked":0} }
```
### 4.2 `env.json`
```jsonc
{ "envId":"env-a1b2c3d4e5f6",              // 폴더명 = "env-"+fingerprint[:12] (L5)
  "fingerprint":"a1b2c3d4e5f6...<64hex>",  // 전체 sha256 (재사용 판정 근거)
  "meta":{"chain":"wbft","binaries":{"producer":"go-wemix@<build>","default":"go-wbft@<build>"}}, // (36)
  "dataPath":"/data/.../<env>",
  "nodes":[ {"name":"bp1","role":"bp","sync":"archive","binary":"go-wbft","buildVersion":"...", // 도메인 용어 bp/en(§3.9)
             "host":"127.0.0.1","rpc":"http://...:40010","ws":"...",
             "ports":{"p2p":30010,"http":40010,"ws":40011,"auth":40012,"metrics":0}} ] } // = node.Endpoints. etcd=p2p+1(30011)는 파생·예약(portplan)이며 별도 저장 안 함
```
### 4.3 `spec.json` (TestSpec DSL — 정본 §D-2)
필수: `schemaVersion, id, chain(name+binary|binaries), assertions`(F16-O2). 옵션: 그 외.
```jsonc
{ "schemaVersion":"1", "id":"GOV-005", "applicableChains":"wbft",
  "chain":{"name":"wbft","binary":"go-wbft","config":"...","genesisOverlay":{...}},
  "topology":{"bp":7,"en":5,"sync":{"bp1":"archive","default":"full"},"bootnode":15},
  "hardforks":{"croissant":100,"brioche":50},
  "placement":"local", "remote":{"cluster":"cluster.yaml"},
  "defaultOn":"bp:any", "timeouts":{"test":"10m","waitReceipt":"30s"},
  "preActions":[{"ensureChain":true},{"ensureStaker":{"name":"A"}}],
  "steps":[{"tx":{"on":"bp1","signer":"op1","call":"registerStaker(...)","args":[...],"gas":"auto",
                  "waitFor":"receipt","expectStatus":"0x1"}},          // 기본 성공; negative는 "expectRevert":true (F11)
           {"tx":{"on":"bp1","signer":"op2","call":"unstake(1)","expectRevert":true}}],
  "assertions":[{"on":"bp1","source":"rpc","method":"istanbul_getValidators","assert":"Len","expected":7},
                {"onEach":["bp1","en1"],"source":"rpc","method":"eth_getBlockByNumber","assert":"EqualHashAt","at":"<h>"}],
  "postActions":[{"unstake":{...}}] }
```
### 4.4 `steps.json` / `assert.json` / `status.json`
```jsonc
// steps.json
[{ "i":0,"type":"tx","on":"bp1","signer":"op1","nonce":12,"gas":"auto","hash":"0x..","receipt":{"status":"0x1"} }]
// assert.json (provenance §D-2.6)
[{ "id":"validators==7","on":"bp1","source":"rpc","method":"istanbul_getValidators","raw":"[...]",
   "assert":"Len","actual":7,"expected":7,"pass":true },
 { "id":"reward-log","source":"log","logFile":".../logs/bp1.log","lines":[1423,1423],"byteOffset":92417,
   "extracted":"block reward ...","assert":"NotNil","pass":true }]
// status.json
{ "id":"GOV-005","result":"pass|fail|blocked","durationMs":38210,"assertPass":3,"assertFail":0 }
```

---

## 5. 실행 흐름 (직렬 테스트, 환경 재사용)

```
커맨드(여러 테스트, 직렬)
  └ session.New()                              # .chainbench/<session>/
     for each test (직렬):
       spec = testspec.Parse(...)
       if !applicable(spec, chain): status=SKIP; continue          # 요구 3
       resolved = config.Resolve(file, flags)                      # flag>config>default (§3.1 주)
       fp = spec.Fingerprint(resolved)                             # resolved 선언값 파생(§D-2.4, testspec→session)
       env, ok = session.Environment(fp)
       if !ok:                                                     # 재구성 (없을 때만)
          env  = session.NewEnvironment(fp)                        # 1) 빈 환경 폴더 생성
          plan = setup.BuildPlan(resolved, plugin, env.Dir())      #    실제 3-arg(cfg,plugin,dataRoot); place.Allocate로 포트 결정(§3.4)
          ns, diag = supervisor.BringUp(ctx, plan, {LeaderGate,AlignJoinGap})  # 2) etcd 게이트·복구(기동 소유·L6)
          if !diag.OK: record(diag); status=BLOCKED; continue
          env.PopulateNodeTable(ns); env.Save()                    # 3) ns→node table 기록(env.json)
          collector.Start(ctx, env)                                # 4) 라이브 수집 시작(dataPath tail)
       # (ok=재사용: collector는 최초 생성 시 이미 가동 중)
       # preActions (idempotent 가드) → 실패 시 BLOCKED (테스트 미수행)
       if !runPre(spec.PreActions, env): status=BLOCKED; continue  # 요구 27·i
       runSteps(spec.Steps, env)                                   # atomic tx/wait
       result = runAssertions(spec.Assertions, env, collector)     # rpc/func/log + provenance
       runPost(spec.PostActions, env)                              # 실패해도 판정 독립(§D-2.9)
       rec.Status(result)
  session.Save()
```

---

## 6. 동시성 모델 (요구 E)

- **범위**: "한 환경 내 N노드" 처리만 동시. 테스트 실행은 직렬(§B-4).
- **패턴**: `errgroup.Group`(팬아웃, 최초에러 시 ctx 취소) + `semaphore.Weighted(max(1, min(cores-2,N)))`(자원 상한·성능 35 — **1~2코어에서 `cores-2≤0` 언더플로우/데드락 방지 클램프, S1**) + `context`(취소 전파) + **index 슬라이스/채널 팬인**(락 없는 수집).
- **소유권/락**:
  - `procman.Manager` PID맵 → `sync.Mutex`(기존). remote PID·etcd관측 확장 시 동일.
  - `session` 상태쓰기 → 단일 writer(or flock).
  - `collector` 로그 → 노드별 tail goroutine → **채널→단일 writer**(파일 레이스 제거).
  - `obs.Bus` 구독자 → **현재 `sync.Mutex`**(bus.go:17). 다독 패턴이면 RWMutex 승격 검토(필수 아님).
- **레이스 근절**: 결정적 index 포트 or OS(`:0`)(place), 공유 genesis 원본 불변, ctx 미취소 고아 노드 → errgroup+procman.StopAll.

---

## 7. remote 전략 (요구 9·10·11·16)

- 공통 절차(접근·upload·download·동일포트)를 **core로 승격**: `driver.RemoteDriver`(+`FileProvisioner`, 기존) + `core/remote`(ssh/auth, 기존) + `place.RemotePerHost` + `keyreg.UploadTo/RemoteDownload`.
- 체인 특화(wemix etcd/gov)는 `chains/wemix/deploy`에 잔류. 테스트 정의서는 **placement-무관**(local/remote 동일 spec, `placement` 필드만 상이).
- **SSH 접속 자격증명(L6b · 보안):** remote 접속에 필요한 **ssh 포트·서버 IP 목록·user·password(또는 keyPath)** 는 **정의서에 넣지 않고**, git에 **절대 커밋하지 않는** 별도 파일 `server-set.yaml`을 런타임에 읽어 사용한다. 정의서는 `remote.cluster`로 이 파일을 **참조만** 한다. 파일은 `.gitignore` 처리하고 `server-set.sample.yaml`(더미값)만 추적.
```yaml
# server-set.yaml (gitignore 대상 — 실값 커밋 금지)
sshPort: 22
user: deploy
password: "<secret>"          # 또는 keyPath: ~/.ssh/id_ed25519
hosts: [10.0.0.11, 10.0.0.12, 10.0.0.13]   # 1서버=1노드 (동일포트+다른IP, 요구 16)
```

---

## 8. 마이그레이션 (하위호환 병존)

1. `session` → 2. `place`(고정포트 근절) → 3. `testspec`+testrun 재작성 → 4. 동시화+`supervisor` → 5. `keyreg`+`collector`+fingerprint 재사용 → 6. MCP/대시보드 연동. (로드맵 §F)
- 각 단계: 신규 경로를 별도 커맨드/플래그로, 기존 유지 → 비-e2e 스위트 통과 + 대표 e2e 1건 라이브 후 다음.

---

## 9. 미결 · 대안 (검토 중 도출)
- **place 포트 모드 기본값**: LocalOSAssigned(안전) vs LocalStepped(재현성). → 기본 OS할당, `--ports fixed`로 스텝 선택.
- **collector chainstate 저장형식**: 파일(jsonl) vs obs 스트림. → jsonl 파일 + obs 미러(대시보드).
- **testspec Action/Assertion 레지스트리 범위**: 내장 세트 + 체인별 확장 지점.
- **`applicableChains` ↔ `chain.name` 관계**: `chain.name`은 이 테스트가 도는 **대상 체인**, `applicableChains`는 **호환 체인 집합**(예: 같은 합의계열 wbft·stablenet). 스위트 실행이 대상 체인을 `applicableChains` 내에서 바꿔 재사용할 수 있는지(체인-스윕) 여부를 F3에서 확정. 미적용이면 SKIP(요구 3).
- **fingerprint 대상 = 선언값 전체**(`binaries+genesis+config+topology+hardforks+placement`, §D-2.4): precedence(flag>config>default, §B-3) 적용 *후*의 선언값을 해싱해야 재사용 판정이 정확(같은 config 파일+다른 flag=다른 env). **placement(local↔remote)도 포함**(O1) — local로 세운 env를 remote 선언이 오재사용하면 포트/호스트가 어긋난다. env-id 폴더명은 해시 앞 12hex 축약(§3.1·L5). §3.1·§3.2·§D-2.4 반영.
- **MCP/대시보드 seam**: MCP(요구 31)는 세션 아티팩트(status/assert.json)를 읽어 결과 응답; 대시보드(요구 33·34)는 collector의 chainstate + session을 소비. 인터페이스는 F14·F15에서 확정(design은 소비 지점만 명시).
- 위 항목은 feature-spec(F3·F5·F6·F10·F12·F14·F15)의 AC에서 확정.

### 9.1 검토 반영(2차) — 확정 항목
| 항목 | 결정 | 앵커 |
|------|------|------|
| genesis 소싱 | 별도 모듈 4모드(existing/build/template+override/upgrade-inherit); 진입점 `genesis.Build`(≠Family.BuildGenesis) | §3.8 (L3) |
| 역할 용어 | 표층은 도메인 어휘 `bp`/`en`/`boot` 일관 사용, 코드 enum(validator/endpoint)은 내부; `RoleEN` dead code | §3.9 (L2) |
| etcd 포트 | wemix 내장, `P2P+1` 예약(`portplan.Ports`), RPC밴드와 분리 | §3.4 (L1) |
| fingerprint 길이 | 전체 sha256은 env.json에만, 폴더는 `env-`+12hex | §3.1 (L5) |
| 기동 소유권 | supervisor.BringUp이 소유(setup=plan+프리미티브) | §3.3 (L6) |
| etcd 조인 gap | supervisor가 N에서 파생(고정값 아님) | §3.3 (L7) |
| stale etcd | 내장 etcd는 프로세스 종료로 함께 종료; 문제는 datadir → `Teardown{RemoveDataDir}` 별도 기능 | §3.3 (S2) |
| SSH 자격증명 | `server-set.yaml`(gitignore) 런타임 로드 | §7 (L6b) |
| 액션·검증 레지스트리 | 전역 아님 — `Deps.Actions`로 인스턴스 주입 | §3.2 (P1) |
| 세마포어 상한 | `max(1, min(cores-2,N))` 클램프 | §6 (S1) |
