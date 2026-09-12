# 패키지 전수 트리 — 디렉토리 축

이 문서는 **디렉토리가 지금 어떻게 생겼는지**와 **각 패키지가 무엇을 지원하는지**를 한 자리에서
답한다. 축이 하나뿐인 문서이며, 다른 축은 다른 문서가 답한다:

- **레이어 축**(어느 패키지가 L0~L6 중 어디이고 의존이 어느 방향으로 흐르는지) — [[layers]](layers.md) §2·§3
- **관심사 축**(어떤 관심사의 주인이 누구인지) — [[module-responsibilities]](module-responsibilities.md)
- **통폐합 판단** — [[consolidation-plan]](consolidation-plan.md)

숫자는 **비테스트 줄 수**다. `[L0]`~`[L6]` 은 `layers.md` §2 의 레이어이고, 그 배치는
`internal/arch/layers_test.go` 가 강제한다.

> **디렉토리 깊이는 가시성을 제한하지 않는다.** `internal/core/keyring/derive` 가 `keyring` 아래
> 있다고 해서 `keyring` 만 쓸 수 있는 것이 아니라 모듈 전체가 import 할 수 있다. Go 의 `internal`
> 규칙은 `internal` 이라는 이름의 디렉토리에만 걸리고 저장소에는 그런 디렉토리가 최상위 하나뿐이다.
> 따라서 이 트리는 **주제 분류의 기록이며 규칙이 아니다.** 규칙은 `layers.md` §2 와
> `internal/arch` 의 테스트가 강제한다.

> **트리만 보고 층을 짐작하면 틀린다.** 디렉토리는 주제와 레이어를 섞어 표현하고 있어서, `core`
> 안에 L0(`node`)·L1(`process`)·L3(`session`)이 함께 있고 L3 인 `testhelper`·`validatorset`·`dsl/*`
> 는 `core` 밖에 있다.

## 0. 전수

| 묶음 | 패키지 | 줄 |
|---|---|---|
| `internal/` | 48 | 47,583 |
| `cmd/` | 19 | 5,071 |
| `scripts/inventory/` | 3 | 790 |
| **합계** | **70** | **53,444** |

이 세 숫자는 `internal/arch/packagetree_test.go` 가 `go list ./...` 와 맞춰 본다. `layers.md` §3 의
제목에 있던 개수가 43 에서 멈춰 실제 48 과 갈라져 있었기 때문에 — 개수는 사람이 세면 늦는다 —
양쪽 문서의 숫자를 같은 테스트가 지킨다.

---

## 1. `internal/core` — 22패키지 14,968줄 · 프로젝트 공용 기반

```
internal/core/
├── home            52  [L0] 약속된 위치의 소유자 ~/.chainbench. 경로를 안 대면 키셋·세션·구성이 다 여기로 (요구 7)
├── node         1,113  [L0] 노드에 대해 아는 것 전부 — Node·NodeSet·Role·Endpoints·Label·Placement·Map·
│                            Peering·Layout·Enode + 노드 레이아웃 선언(Topology·Entry·Load).
│                            최다 피참조. 내부 import 0
├── wait            43  [L0] 취소 가능한 유일한 멈춤 — Sleep(ctx, d). 내부 import 0
├── rpc            511  [L1] JSON-RPC over HTTP 최소 클라이언트 (verify·test 단계용)
├── remote         552  [L1] 원격 접근 — API key/JWT 전송, SSH 터널, host-key 정책. rpc.DialWithClient 용 *http.Client
├── process      1,440  [L1] 프로세스 기동/정지/provision(Initializer·LogReader)·PID 추적·검증된 종료(run ledger)
│                            + 기동 정책(Direct: arm·materialize·init·launch / Launcher: 헬스 게이트·진단·재시도·teardown)
├── inspector      241  [L1] 요청 시 실사 — 포트 점유(로컬 bind 두 형태, 원격 probe)·경로 존재·호스트 도달.
│                            사실만 답하고 판단하지 않는다
├── filestore      297  [L1] FileSink — 타깃에 파일을 놓는 유일한 통로 (data dir·config·genesis·key)
├── nodeconfig   1,411  [L1] 노드 하나의 설정 — config.toml 렌더 · launch argv 조립(Argv) ·
│                            dot-path 설정값 3단 해석(Values·Merge·Resolve·Flatten·Defaults; 코드 기본값 < 파일 < 플래그/env)
├── genesis        550  [L1] genesis.json 빌더 — SourceFor(패밀리가 SourceProvider 를 선언하면 그것, 아니면 프리셋 치환)
│                            · Compose(소스 + 오버라이드 + 오버레이 + fork 검증)
├── blueprint    1,248  [L1] 네트워크 선언 1문서 — 파싱·왕복·문서 내부 검증. 미지 필드 거부. 해석하지 않는다
│                            (빠진 값 채우기는 한 층 위 Resolve 몫)
├── keyring        508  [L1] 키 모델 — Entry·Preset·Network·Label·출처(hex·니모닉·파일)·비밀번호 입력
│   ├── derive     345  [L1] 키 파생 — secp256k1 키·주소·devp2p 공개키·BLS·PoP (in-process 순수 계산)
│   ├── store    1,422  [L1] 키셋 저장·읽기 — 디스크 레이아웃·metadata 색인·keystore/raw 백엔드 · 키 출처(KeySource)
│   └── operation  666  [L1] 키셋에 가하는 동사 — new·add·list·show·export·import·세트 복제.
│                            서버 접근은 자기가 선언한 Opener 로 주입받는다
├── registry       759  [L1] ChainPlugin/ConsensusFamily 인터페이스 + 레지스트리, 그리고 그 플러그인이 선언하는 것 —
│                            capability 카탈로그·핸들러(LoadCatalog·RegisterHandler·GetByAddress)·검증자 조회(Validators)
│                            · 인자 디코딩(ArgString·ArgInt·ArgBigInt·ArgStrings·ArgBool)
├── preflight      297  [L1] 현재 vs 목표 비교 — 타깃에 조립된 체인(Have)과 다음 테스트가 원하는 체인(Want)을 견줘
│                            reuse / rebuild-nodes N / rebuild-all / compose 를 답한다. 판단만 하고 보지 않는다
├── session      1,298  [L3] 아티팩트 레이아웃의 소유자 .chainbench/<session>/ — 세션·환경·컴포지션 +
│                            이름 붙인 네트워크 레지스트리(SaveNetwork·LoadNetwork·ListNetworks·RemoveNetwork)
├── collector    1,440  [L3] live tail·chainstate·bp 참여·reorg + 이벤트(Bus·Event·Kind·Phase)
│                            + 로그 검색·타임라인(Search·Timeline) + RPC 로부터의 체인 종류·능력 감지
├── health         425  [L3] "이 네트워크가 블록을 만드나"(요구 9) — NodeSet 전수 RPC 샘플 → chain id·height·peers·sync
│                            + 1차 노드 height 전진으로 producing 판정
├── report         214  [L3] 실행 전체 report 의 집계자 — 세션이 영속한 테스트별 verdict(status.json)와 증적 경로를
│                            모아 report.json(Build·Generate·Write·Read). 판정을 다시 하지 않는다
└── hardfork       136  [L3] 바이너리 교체(swap) 업그레이드 계획/실행 — 같은 데이터디렉토리를 멈췄다 fork 를 켠 새
                             바이너리로 재기동(합의 엔진 불변). consensus/upgrade 의 핸드오프와 의도적으로 별개
```

`registry` 가 L1 인 것이 이 층 구조의 핵심이다. **인터페이스는 아래, 구현은 위(L2)** 에 있어서
L3/L4 가 체인을 모른 채 `ChainPlugin` 만 쓸 수 있다.

---

## 2. 체인·합의 정의 — 10패키지 4,645줄

```
internal/consensus/             합의 패밀리 [L2a] — 체인 id 를 모른다
├── wbft            576  wbft genesis(extraData RLP) · start flags. stablenet 과 wbft 체인이 공유
├── poa           1,499  wemix config · genesis 생성 · 거버넌스/etcd 부트스트랩 프리미티브와 그 실행자
│                        (Bootstrap: 패밀리가 선언한 액션을 한 타깃에서 / Info·WaitEtcdCluster: 클러스터가 실제로 섰는지)
└── upgrade       1,949  체인 핸드오프 — 계획(BuildPlan)·기동(Launch)·메시(WireMesh) + 핸드오프 본문 1개
                         (Handoff: config → base genesis → plan → overlay → launch → mesh → governance →
                          etcd → verify → fork 대기)

internal/chains/                체인 어댑터 [L2b] — 자기 체인만 안다
├── all              16  등록 집합 — blank import 로 내장 플러그인과 그 capability 전체를 등록. 유일한 plug-in seam
├── common           72  모든 체인이 공유하는 capability 구현("common" 프로젝트)
├── external         89  프로젝트가 준 매니페스트 파일로 플러그인 로드 — 코드 변경 없이 미내장 체인을 벤치
│                        (기존 합의 패밀리 위에서)
├── stablenet       175  stablenet 특화 capability (거버넌스 시스템 컨트랙트)
│   └── govbind     162  GovBase 바인딩 — propose→approve→execute calldata 빌더 · MintProof 인코더 ·
│                        디코더 2개(ProposalCreated 로그의 proposalId, proposals() 의 status)
├── wbft             35  go-wbft 플러그인 등록 — wbft 패밀리 + wbft accounts 프로토콜 + 매니페스트·croissant 템플릿(embed)
└── wemix            72  wemix 특화 capability (poa/etcd 부트스트랩)

internal/accounts    982  [L1] tx 서명 — 외부 accounts SDK 경계. EncodeABI·EncodeCall·EncodeCallArgs·Selector·
                          EventTopic·FindLog·CreateAddress·Word/WordAt/WordToBig·ForChain·GenerateKey
internal/validatorset 85  [L3] 체인의 합의 신원 제시 — 키셋에서 검증자셋과 관련 역할을 Load
```

---

## 3. 자원 · 테스트 · 표면 — 16패키지 27,970줄

```
internal/resource  3,080  [L1] 네트워크가 무엇으로 조립되는가 — 풀(호스트 × 포트 슬롯)·배정(Assign)·
                          포트 밴드 산술(Plan·PlanBands·ValidatePorts)·서버 세트(호스트·밴드·자격·호스트키·docker 치환)·
                          여는 유일 통로(Opener)·세트를 풀로 해석(Pool·PoolFor)·인벤토리·baseline 드리프트 검사·
                          워크스페이스 설정·머신 지정(Spec·Access)·devp2p network id 해석(Resolve·Flag·ValidateUniform)

internal/dsl/             [L3] 테스트 정의 언어 (DDD C1, 핵심 도메인)
├── (dsl)      1,192  v1·v2 문법·파싱·검증·statement 파생(Parse·SequenceOf·ActionName·ArgsOf) + JSON 스키마.
│                     순수 — 실행 인프라(rpc·session·collector)를 import 하지 않는다
├── assert       399  타입 인식 비교 프리미티브 — 해석기가 어세션을 검사할 때 쓰는 비교기(Equal·InDelta 등)
└── interp       950  실행 계약(Action·Assertion·Registry·Reader·Deps·ActionCtx·AssertCtx·NodeControl)
                      + 해석기(NewInterpreter·Run) + $ref/save 바인딩 + Fingerprint(환경 재사용 키)
                      + Unresolved(오프라인 이름 검증) + caseTimeout. 계약이 여기 사는 것이 핵심 —
                      testhelper(L3)가 구현하므로 testengine(L4)으로 올릴 수 없다

internal/testhelper 3,717 [L3] DSL 내장 어휘 — 액션(sendTx·waitBlock·read·fault·assets·deployContract·
                          registerContract·newAccount·faucet·partition/heal·start/stop/restart/swapNode·ws open/subscribe)
                          과 어세션·리더의 구현 및 등록(Register·Registry) + 계정 해석(ResolveAccount)

internal/testengine 2,892 [L4] 테스트 엔진 — RunSuite 가 4단계를 소유: ① DSL 이 선언한 체인을 chainsetup 으로 구성
                          ② pre-test hook ③ test ④ post-test hook(②~④는 해석기가 spec 에서 수행).
                          + attach 경로(AttachWorkspaceRun·NewAttachEngine) · Precheck · ValidateSpecs ·
                          overlay 작성 · 노드 게이트 연결(factsFromReport) · 세션 요약

internal/chainsetup 6,585 [L4] 체인 셋업 오케스트레이터 — 선언을 이름 붙인 스텝 열로 바꿔 실행하고
                          워크스페이스에 무엇을 했는지 기록한다. NetNew·NetKeys·NetGenesis·NetConfig·NetAllocate·
                          NetProvision·NetStart·NetUp·NetResume·NetRestart·NetStop·NetRm·NetStatus·NetHealth·
                          NetLogs·NetEnodes·NetEndpoints·NetLaunchOpts·NetBaseline{Check,Approve}·
                          NetVerifyValidators·NodeStart/Stop/Swap·Hardfork{Plan,Execute}·
                          재사용 판단(PlanReuse·ReconcileReuse·GenesisDeclared·WantOf)·실행 중 프로세스 실사

internal/nodemonitor  394 [L4] 테스트 실행 허가 판정 + 제한 복구(E6) — health·collector·inspector·process/inspect·
                          preflight 가 낸 사실을 조합해 노드별 READY/WAITABLE/RESTARTABLE/FATAL 판정(Classify),
                          WAITABLE 은 예산까지 대기 · RESTARTABLE 은 상한까지 재시작 · FATAL 은 파괴적 조치 없이 종료(Gate).
                          관측과 재시작은 재구현하지 않고 seam(Observer·Restarter)으로 주입받는다

internal/app       2,943  [L5] 유스케이스 1개 = 함수 1개. cobra·MCP 타입을 모른다. Net*(20여) · Keyring*(8) ·
                          Tx/Contract(TxSend·TxWait·ContractDeploy·ContractCall) · Faucet · Report · Log* ·
                          Network*(attach/detach/registry) · Upgrade{Run,Genesis} · Hardfork{Plan,Execute} ·
                          RunSuite(s) · Verify* · Capabilit* · Resolve*(binary·chain·key·nodes·server) · GCSessions
internal/feature     502  [L5] 기능 등록의 한 자리 — Descriptor·Register[In,Out]·Stage·ReadOnly, 그리고 입력 struct
                          태그 하나가 만드는 두 바인딩(Flags → cobra 플래그, Schema → MCP JSON 스키마).
                          명령을 생성하지는 않는다 — 이름·계층·도움말은 사람이 정한다

internal/mcp       2,874  [L6] MCP 표면(요구 14) — 도구 스키마 바인딩과 렌더링.
                          chain·network·keyring·capability·run·log·tx·consensus 도구군 + read-only 선언
internal/dashboard   318  [L6] 대시보드 HTTP 백엔드(요구 19) — SSE 스트림 · runs/sessions API · SPA 자산

internal/arch      1,031  (층 없음) layers.md · chainbench-system-direction.md 의 규칙을 강제하는 테스트.
                          제품 코드 0, import 0 — 문서를 읽고 측정과 맞춰 본다
internal/testsupport  26  [L0] 교차 패키지 테스트 게이트 — ServersBuildDir·EnvDockerServers 스킵 헬퍼.
                          패키지-로컬 _test.go 로는 교차 참조가 안 돼 정규 패키지로 둔다. 내부 import 0
```

---

## 4. `cmd/` — 19패키지 5,071줄 · [L6] 표면

`layers.md` §3 의 배치 검사는 `internal/` 만 대상으로 한다 — `cmd` 는 정의상 최상위이고 무엇이든
import 할 수 있다.

```
cmd/chainbench           270  main. 사용자용 CLI(요구 15) 루트 조립
├── surface               81  모든 명령군이 자기에 대해 말해야 하는 것 — 명령 트리의 두 번째 렌더링이
│                             손으로 유지하는 목록이 아니라 선언 하나를 읽게 한다
├── exitcode              33  종료 상태를 결정한 명령에서 그것을 적용하는 main 까지 운반
├── chaincmd             925  체인을 COMPOSE 하고 구성된 것을 읽기 — new·build·config·up·resume·blueprint
│                             + 읽는 동사(show·status·health·logs·enode)
├── lifecyclecmd         432  up 이후의 네트워크 — stop·ps·clean(실행이 남긴 것 제거)·
│                             여전히 하나의 건강한 체인인지 판정(verify·consensus·baseline)
├── nodecmd              117  네트워크의 노드 1개 — 개별 start/stop, RPC 대화
├── suitecmd             507  테스트 스펙 실행 — run(스펙이 선언한 네트워크를 구성 또는 attach 후 실행)·
│                             validate(실행 없이 검사)·migrate-spec(v1 → v2)
├── testcmd               65  디렉토리의 DSL 테스트 케이스 목록 — run 전에 무엇이 있는지 발견
├── txcmd                212  체인에 일을 맡기고 결과를 기다리기 — send·wait·deploy·call
├── accountcmd           177  노드나 체인이 아니라 ACCOUNT 에 가하는 것 — 상태 읽기, 자금 공급
├── keyringcmd         1,073  키 재료 CLI — new·add·list·show·export·import·derive
├── upgradecmd           229  체인을 다른 바이너리·다른 fork 로 옮기기 — upgrade run(wemix→wbft 핸드오프)·
│                             hardfork(구성된 체인에 fork 적용) + 둘이 함께 쓰는 genesis 파생
├── reportcmd            137  실행을 되읽기 — report(세션이 기록한 verdict 와 증적)·log(수집한 노드 로그)
├── networkcmd           215  이름 붙인 네트워크 레지스트리 — MCP chainbench_network_*·chainbench_remote_rpc 의 CLI 거울
├── resourcecmd          237  네트워크가 무엇으로 구성될 수 있는지를 질문으로 — 서버·포트 슬롯·용량
├── filecmd              123  서버 데이터 플레인과 파일 주고받기
└── catalogcmd           145  "이 빌드가 무엇을 아는가" — 구성 가능한 체인, 등록된 capability. 네트워크를 건드리지 않는다
cmd/chainbench-mcp        56  main. MCP 표면(요구 14)을 stdio 로 — stdin 의 줄 단위 JSON-RPC 를 읽어 mcp 도구로 디스패치
cmd/chainbench-dashboard  37  main. 대시보드 데몬(요구 19) — obs 이벤트 버스와 run 저장소를 HTTP+SSE 로 호스팅
```

---

## 5. `scripts/inventory/` — 3패키지 790줄

측정 도구다. 제품 바이너리가 아니고, 이 문서와 `code-graph.md`·`consolidation-plan.md` 의 숫자가
여기서 나온다.

```
scripts/inventory/
├── code-graph        449  chainbench 모듈의 AST 그래프 — 패키지=노드, import=엣지, 그 엣지가 실제로
│                          참조하는 exported 심볼. 빌드 불필요
├── chain-flag-graph  282  go-ethereum 파생 체인 저장소의 명령/플래그 그래프를 cmd/ 트리 파싱으로 추출
└── surface-graph      59  각 표면(CLI·MCP·DSL)이 등록한 것과 그중 app 층을 넘는 것 — U 트랙 진척을
                           추정이 아니라 코드에서 읽는다
```

---

## 6. 이 트리에서 눈에 보이는 것

**가장 큰 덩어리에 구조가 가장 없다.** `chainsetup` 6,585줄이 한 패키지 22파일이고, 축은 파일명
접두사로만 암시된다(`steps_*` 단계 / `verbs_*` 진입점 / `workspace.go`·`reuse.go` 상태). 위 트리에서
한 줄로 압축된 자리가 실제로는 전체에서 가장 읽기 어려운 자리다. `testhelper`(3,717) · `resource`(3,080) ·
`app`(2,943) · `testengine`(2,892) · `mcp`(2,874) 도 같은 모양이다.

**깊이가 있는 곳은 `core` 와 `dsl` 뿐이다.** `core` 22패키지 depth 3, `dsl` 3패키지, `consensus` 3,
`chains` 7(depth 3). 나머지 최상위는 자식이 없다. 즉 트리를 더 읽기 쉽게 만드는 여지는 "최상위를
줄이는" 쪽보다 "큰 최상위에 자식을 만드는" 쪽에 있다 — 판단은 [[consolidation-plan]](consolidation-plan.md) §5 에서 한다.

**층과 디렉토리가 어긋나는 자리가 셋 있다.** `core` 안의 L3 다섯 개(`session`·`collector`·`health`·
`report`·`hardfork`)는 이름이 core 인데 정책이고, `core` 밖의 L3 넷(`dsl`·`dsl/assert`·`dsl/interp` 계약
구현체인 `testhelper`, 그리고 `validatorset`)은 정책인데 core 밖이며, `resource`(L1)는 프리미티브인데
최상위다. 셋 다 의도된 것이고 각 이유는 `layers.md` §3 에 있지만, 경로만 보고는 알 수 없다.
