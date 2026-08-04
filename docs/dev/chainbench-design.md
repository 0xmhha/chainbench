# chainbench 설계 문서 — 구조 · 인터페이스 · 데이터 모델

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
┌ Entrypoints ─ cmd/{chainbench, chainbenchd, chainbench-mcp}
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
    ID() string                        // env-id (= fingerprint 축약)
    Dir() string
    Fingerprint() Fingerprint
    PopulateNodeTable(ns node.NodeSet) // BringUp 결과(ns)로 node table 채움 → Save 전
    Nodes() []node.Node                // node table (endpoint 해석 근거)
    Resolve(selector string) (node.Node, error)   // "bp1"|"bp:any"|"en:0"
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
type Fingerprint string   // testspec.Spec.Fingerprint() session.Fingerprint 로 계산
```

### 3.2 `testspec` — 정의서 파싱·해석 [NEW · testkit Go-func 대체]
```go
package testspec

// Spec = 파싱·검증된 테스트 정의서(전 필드는 §4.3 스키마).
type Spec struct { /* Id, ApplicableChains, Chain, Topology, Hardforks, Placement,
                      DefaultOn, PreActions, Steps, Assertions, PostActions, Timeouts */ }

func Parse(raw []byte) (Spec, error)               // 필수/옵션 검증 + JSON schema
// Fingerprint는 precedence(flag>config>default)로 **resolved된 선언 config**를 해싱한다(§3.1 주). 체인 미접촉.
func (s Spec) Fingerprint(resolved config.Values) session.Fingerprint  // (§D-2.4)
func (s Spec) Get(dotPath string) (any, bool)      // 닷경로(a.b.c) 리졸버, multiple "," 파서

// Deps: 해석기가 스텝/검증을 수행하려면 반드시 필요한 협력자(생성자 주입).
type Deps struct {
    Keys      keyreg.Registry          // 서명 키(§3.5)
    Accounts  accounts.AccountProvider // tx 서명·전송
    RPC       func(url string) *rpc.Client
    Collector collector.Collector      // log 검증·WaitLog(§3.6)
    Funcs     map[string]AssertFunc    // source:"func" 검증 함수 레지스트리
}
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
func RegisterAction(name string, a Action)          // ensureChain/ensureStaker/tx/wait/unstake/
                                                    // stopNode/startNode/restartNode/chainMigrate ...
                                                    // (노드 생명주기 액션은 destructive/fault 테스트용: WBFT-007/008, NODE-005)
func RegisterAssertion(name string, a Assertion)    // Equal/NotNil/EqualWith/Len/EqualHashAt ...
```

### 3.3 `supervisor` — 헬스 게이트·복구 [NEW · runGovHandoff 재시도 대체]
```go
package supervisor

type Supervisor interface {
    // 노드 기동 + 헬스 게이트(etcd 리더 준비, 포크 도달 등) + 백오프 복구.
    BringUp(ctx context.Context, plan setup.Plan, opts Options) (node.NodeSet, Diagnosis, error)
}
type Options struct {
    LeaderGate   bool          // producer etcd 리더 준비 폴링(C-etcd)
    StartGap     time.Duration // etcd 조인 슬롯(gap 7/11/17/23s)에 맞춘 시작 정렬
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
    Mode        FailureMode   // EtcdJoinFailed | EtcdStale | ForkNotCrossed | QuorumLost | RPCUnready
    Detail      string        // 실제 로그 라인(원인 은닉 금지)
    ProducerLog string        // 최종 실패 시 producer 로그 tail (세션 보존)
}
```
> etcd: `etcdInit` 후 **리더 준비 확인 게이트** + **stale etcd 정리** + **gap 정렬**로 "바로 연결"(C-etcd §요약). 실패는 분류·기록.
> 하드포크는 2종(§D-2.8): **type-1 체인 업그레이드**는 노드별 바이너리 집합(plan.Nodes[].Binary)으로, **type-2 동일체인**은 `ForkSwaps`로 fork 블록 전 바이너리 교체를 supervisor가 오케스트레이션.

### 3.4 `core/place` — 통합 배치·포트 [NEW · portplan+topology CONSOLIDATE]
```go
package place

type Mode int
const ( LocalStepped Mode = iota; LocalOSAssigned; RemotePerHost )

type NodeReq struct { Name string; Role node.Role; Sync string; Binary string }
type NodePlacement struct {
    Name string; Host string           // local=127.0.0.1 / remote=서버IP
    Ports node.Endpoints               // p2p/etcd(=p2p+1)/http/ws/auth
    DataPath string
}
type Allocator interface {
    Allocate(reqs []NodeReq, mode Mode) ([]NodePlacement, error)
}
```
- `LocalStepped`: index 기반 결정적 스텝. `LocalOSAssigned`: `:0` 바인드 후 회수(고정포트 이중바인드 근절, E-3-1). `RemotePerHost`: 동일 포트 + 서버별 IP(요구 16).

### 3.5 `core/keyreg` — 키 레지스트리 [NEW · keys+deploy/keys CONSOLIDATE]
```go
package keyreg

type Key struct { Name, Address string; Private []byte; BLS, PoP []byte }
type Source int
const ( Random Source = iota; LocalFile; RemoteDownload )

type Registry interface {
    Ensure(name string, src Source, ref string) (Key, error)  // 생성/복사/다운로드 → 세션 keys/에 저장
    Get(name string) (Key, bool)
    UploadTo(ctx context.Context, fp driver.FileProvisioner, names []string, remotePath string) error // 랜덤키→remote
}
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
{ "envId":"<fp>", "fingerprint":"<fp>",
  "meta":{"chain":"wbft","binaries":{"producer":"go-wemix@<build>","default":"go-wbft@<build>"}}, // (36)
  "dataPath":"/data/.../<env>",
  "nodes":[ {"name":"bp1","role":"bp","sync":"archive","binary":"go-wbft","buildVersion":"...",
             "host":"127.0.0.1","rpc":"http://...:40010","ws":"...","ports":{"p2p":30010,"etcd":30011}} ] }
```
### 4.3 `spec.json` (TestSpec DSL — 정본 §D-2)
필수: `id, chain(name+binary|binaries), assertions`. 옵션: 그 외.
```jsonc
{ "id":"GOV-005", "applicableChains":"wbft",
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
          ns, diag = supervisor.BringUp(ctx, plan, {LeaderGate,StartGap})  # 2) etcd 게이트·복구
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
- **패턴**: `errgroup.Group`(팬아웃, 최초에러 시 ctx 취소) + `semaphore.Weighted(min(cores-2,N))`(자원 상한·성능 35) + `context`(취소 전파) + **index 슬라이스/채널 팬인**(락 없는 수집).
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
- **fingerprint 대상 = resolved config**: precedence(flag>config>default, §B-3) 적용 *후*의 선언값을 해싱해야 재사용 판정이 정확(같은 config 파일+다른 flag=다른 env). §3.1·§3.2 반영.
- **MCP/대시보드 seam**: MCP(요구 31)는 세션 아티팩트(status/assert.json)를 읽어 결과 응답; 대시보드(요구 33·34)는 collector의 chainstate + session을 소비. 인터페이스는 F14·F15에서 확정(design은 소비 지점만 명시).
- 위 항목은 feature-spec(F3·F5·F6·F10·F12·F14·F15)의 AC에서 확정.
