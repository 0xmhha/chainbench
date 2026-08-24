# chainbench 구현 작업 트래커 — 작업 리스트 · 우선순위 · 폴더 트리 예상도

> **[정본]** **작업 순서·상태의 단일 출처.**
> 이 문서는 *무엇을 만들어야 하는가* 를 정한다. 설계 제안이 여기와 어긋나면 **제안을 고친다.**

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
| 2 | **Session(M1)** | 모든 기록의 정본, 기반 | ☑ |
| 3 | **Place(M2)+용량검증** | 배치·포트(고정포트 충돌 제거) | ☑ |
| 4 | **procman 배선+확장(M6 핵심)** | 최우선 안전 갭(검증 없는 Kill→고아 위험) | ☑ |
| 5 | **testspec Parse+Fingerprint** | spec 실행·env 재사용 | ☑ |
| 6 | **keyreg(M3) + assert funcs** | 신원·검증 | ☑ |
| 7 | **Provisioner(M5) → Supervisor(M6)** | 물질화·기동·teardown | ☑ |
| 8 | **Collector RPC-min + Interpreter-min** | 스켈레톤 최소 관측·실행 | ☑ |
| 9 | **★ Engine walking skeleton** | 첫 통합 증명(1체인·local·4노드·tx1) — **실 gstable 라이브 e2e 통과(RunSpec 실행 수직)** | ☑ |
| 10+ | 수직 슬라이스 확장: remote☑ · attach☑ · 표면(capability/CLI/MCP/-race/백프레셔)☑ · Collector 심화☑ · stablenet☑ · DSL 어휘(바인딩·fault·자산·logs·gas·ws)☑ · read-기반 스위트 이관☑ · 업그레이드(T5.2)☐ · wemix4 이관(T5.5)☐ · 레거시 제거(소비자 이관 후)☐ | 수직 슬라이스 확장 | ◐ |

---

## 1b. 진행 현황 요약 (2026-08-09 기준)

**walking skeleton 완성 · 재설계 엔진(H1)이 CLI·MCP 양쪽에서 도달 가능 · 실행 수직 전체가 CI(mock/attach)+라이브(gstable) 커버.**

**2026-08-09 x-bar 정렬 검토 후속 (이 브랜치):** 문서 정합(T6.5·T6.5b·audit 스냅샷 강등) → **DSL 보충어 어휘 확장**(스텝 값 바인딩 `save`/`$ref`+`read` · fault 액션 5종 · 자산/컨트랙트 액션 3종 · `logs` 이벤트 어세션 · tx fee-cap/nonce 인자 · `gasPrice`/범용 `rpcCall`/`wsSubscribe`) → **supervisor 선언 논항 방출**(T3.2b) → **원격 SSH tail**(`LogReader` seam). 액션 2개·어세션 12개 → **액션 11개·어세션 16개**. 라이브 증명: `TestEngine_Live_NewVocabulary`(실 gstable 4노드). 부수적으로 **라이브에서만 드러난 결함 2건 수정** — `procman` 이 loopback 호스트를 원격으로 오판(단일 노드 정지 불가), 실패 스텝의 사유가 아티팩트에 기록되지 않음.

**재조정 검토 (#225 머지 후):** 병합 상태 green 확인(`go build/vet ./...`·`golangci-lint`·`go test ./...` 52 pkg 전부 통과). 진행 재정리: **T6.5·T6.5b 완료**(문서 pkg→internal 정리·§2b/§3/§5 실측 갱신), **T4.2 인벤토리 현행화**(액션 11·어세션 16). **표현력 블로커는 전부 해소** — 이제 남은 건 **작업량·환경 의존** 뿐: (1) 케이스별 스위트 이관(anzeon 거버넌스·wbft·api·repro — 소스 있음, 라이브만 바이너리), (2) **T5.2 업그레이드 멀티바이너리**(gwemix+etcd 필요), (3) **T5.5 wemix4 이관**(대규모), (4) **레거시 제거**(testkit/testrun — `test`/MCP/upgrade 소비자 이관 선행), (5) **실 SSH/바이너리 라이브 검증**(사용자 환경). 표현력이 아닌 이 항목들이 다음 우선순위 결정 대상.

**완료 (PR):**
- Phase 4 walking skeleton — Engine 오케스트레이션·빌트인 tx/rpc·RunSpec·BuildEnv(AssemblePlan/GenesisSource/orchestration)·최상위 `NewLocalEngine`. **실 gstable 4노드 라이브 e2e**(BuildEnv 기동 + RunSpec 실행 + capstone `Engine.Run`) 통과. (#188~#197)
- launcher 물질화 `provision.Provisioner` 경유(B) (#198)
- SSH key_file 인증 + `kill -0` 종료검증(procman) (D) (#199)
- remote 슬라이스: `RemoteFileSink` + 원격 가능 launcher(Initializer 라우팅) (C) (#200)
- attach 모드: `NewAttachEngine`(RPC-only, mock RPC 로 CI e2e) (#201)
- Phase 6 표면: capability 게이팅(spec `requires`)·`-race` 게이트·CLI `chainbench run` (#202)
- Phase 6 표면: MCP `chainbench_run`·공유 `ReadSessionSummary`·obs 백프레셔(O7) 테스트 (#203, 리뷰중)

**남은 작업:**
- ~~**T6.3 dashboard**~~ ☑ 엔진이 obs 이벤트 emit(`Deps.Emit`/`Bus`) → `chainbench run --dashboard <url>` 가 `dashboard.Forward` 로 chainbench-dashboard 에 스트리밍(라이브). + ☑ **완료 세션 디스크 조회**(F15 AC3): `session.List`/`SessionFilePath`/`ChainstatePaths` + dashboard `/api/sessions`·`/api/sessions/{id}`(session.json)·`/api/sessions/{id}/chainstate`(chainstate.jsonl), `chainbench-dashboard -artifact-root` 로 활성화. attach+mock RPC end-to-end + httptest 로 CI 커버.
- **T5.2 업그레이드 멀티바이너리**(wemix+wbft 핸드오프) — gwemix+etcd 바이너리 필요, 이 환경 라이브 검증 제한.
- ☑ **T5.4 stablenet**(ACL 플러그인·Core 무변경 검증) — stablenet 거버넌스 시나리오가 DSL 엔진에서 실행됨을 CI 검증(예제 `stablenet-governance-read.json` + govbind `proposals(uint256)` calldata·mock RPC e2e `call` 어세션 pass). core 무변경. 실 gstable 라이브(RunSpec/BuildEnv)는 기존 게이트 테스트로 커버.
- **T5.5 wemix4 이관**(레거시 스위트 → DSL) — 대규모.
- **Collector 심화(T3.3)** — ☑ 로컬 live tail(스캔→tail·부분줄 안전)·bp참여 집계(head producer·window prune)·fork/reorg 검출(높이별 hash 불일치)·**엔진 배선**(local/attach 이 `Bus` 설정 시 collection 실행→chainstate·로그를 obs 로 미러, teardown 시 정지)·**chainstate 세션 영속화**(`chainstate/chainstate.jsonl` — F10/F15 jsonl+obs 미러). ☐ 원격 SSH tail(사용자 SSH 환경 필요, T5.1 계열).
- **실 원격 SSH 호스트 라이브 e2e**(RemoteFileSink+RemoteDriver) — SSH 대상 필요(사용자 환경).
- ◐ **레거시 경로 정리** — 착수: 죽은 심볼 제거(`testkit.RunCase`)·레거시 패키지 signpost(testkit·testrun→engine+testspec)·**은퇴 계획 문서**([[legacy-retirement-plan]] = `docs/dev/legacy-retirement-plan.md`: 매핑·순서·블로커·DSL 표현력 갭). **suite 이관 착수**: `tests/` Go-func 케이스는 repo 에 있어 DSL 포팅 가능(라이브만 바이너리 필요) — `onEach` 다중노드 어세션 수정 + 신규 빌트인(`blockAdvance` 헤드 전진·`sameBlockHash` 노드간 no-fork, `rpc.BlockByNumber` 기반) 추가로 **`tests/network` 3케이스 전부 DSL 포팅 완료**(`network-peers.json`·`network-health.json`). **`tests/anzeon` 착수**: read-shape 계열(adapter-code `codeAt NotEqual`, balanceOf/isAuthorized `call Regexp`)→`stablenet-system-contracts.json`, base fee 경계(`baseFee` 신규 빌트인, `rpc.BlockByNumber` baseFeePerGas)→`stablenet-gas-policy.json`, 하드포크 아티팩트(P-256 precompile `call`·GovMinter `codeAt`·chainId/blockNumber)→`stablenet-hardfork.json`, 가스 추정(`estimateGas`=`rpc.EstimateGas`)→`stablenet-estimate-gas.json`, token-metadata(`call`+`Contains` 심볼)→`stablenet-token-metadata.json` 포팅. **read-기반 anzeon 이관 완료**(시스템컨트랙트·getter·base fee·하드포크·estimate-gas·token-metadata). **잔여 anzeon 6범주(교차-call 비교·거버넌스 다단계·gasTip 조합·fee-cap tx·명시 nonce·WS)의 표현력 블로커는 2026-08-09 전부 해소** — `save`/`$ref` 바인딩+`read`(T4.2a)·`logs`(eth_getLogs)·`gasPrice`+범용 `rpcCall`·tx fee-cap/nonce 인자·`wsSubscribe`. 예제: `stablenet-token-invariants`·`governance-event-flow`·`gas-policy-derived`·`stablenet-fee-boundary`·`stablenet-nonce-ordering`·`ws-subscribe-heads`. **라이브 증명**: `TestEngine_Live_NewVocabulary`(실 gstable 4노드, GSTABLE_BIN). 남은 건 표현력이 아니라 **케이스별 이관 작업량**([[legacy-retirement-plan]] §4.3). `test`/MCP/upgrade 가 아직 레거시 사용 → 소비자 이관 전 제거 금지.

---

## 1c. 재계획 (2026-08-11) — 배경요구 재대조 후 잔여 작업

> 근거: [[dsl-v2-proposal]] · [[chain-binary-flag-graph]] · `archive/structure-and-atomic-cli-proposal`
> (2026-08-11 검토 3종 — 셋째는 제안분이 구현되어 [[archive/README|archive]] 로 이동했다).
> 이 절은 §2 의 Phase 목록을 대체하지 않고, **배경 요구(체인 구성 5요소 · 실행옵션 · 3-검증원)** 대조에서
> 새로 드러난 갭과 그 순서를 얹는다. §2 의 미완 항목(T5.2·T5.5·레거시 제거)은 그대로 유효하다.

### 완료

- ☑ **T7.1 keyreg 프로덕션 배선** — `keyreg.New` 의 프로덕션 호출 지점이 0 이던 문제(배경 1.4·1.5 / 알고리즘 2·3 미구현)를 해소.
  - `session.NewWithKeys(baseDir, cmd, at, keyreg.Deps)` 신설 — **레지스트리 경로를 session 이 소유**(design §3.1). `New(..., nil)` 은 테스트용으로 병존.
  - `keyreg.Literal` 소스 신설 — 호출자가 이미 들고 있는 키 자료(preset 노드 신원)가 재-읽기 없이 레지스트리로 들어오는 경로.
  - `keyreg.EnsureOpts.ExpectAddress` 신설 — 등록 시 **개인키에서 주소를 재유도해 선언값과 대조**. C2 불변식("genesis 등록신원 = 실제 키 일치")을 등록 시점에 강제한다. 불일치 시 키는 저장되지 않는다(디스크·메모리 모두).
  - `engine.KeySource` seam — `PresetKeySource`(기존 세트 사용) / `GeneratedKeySource`(`keygen.GeneratePreset` 로 신규 생성, idempotent 재사용). `Dir()` 이 구성 시점에 알려지므로 launcher·genesis source 배선은 그대로.
  - `engine.RegisterIdentities` — 세션 레지스트리에 `node1..nodeN` 등록. `sess.Keys()` 의 **첫 실소비자**.
  - `Deps.NewSession` 이 ctx 를 받는다 — 키 생성이 외부 `bootnode` 프로세스를 부르므로 취소·timeout 이 전파돼야 한다.
  - CLI: `run --keys-source preset|generate --bootnode <path>`.
  - 검증: keyreg 단위 3건(Literal·ExpectAddress 4케이스·거부 시 미저장) · session 2건 · engine 6건 · **e2e 2건**(바이너리 없이 `Run` → `<session>/keys/node<i>/address` 생성 확인, 키셋보다 큰 토폴로지 거부) · CLI 6케이스. `go test -race ./...` green.
  - ~~한계: `keygen` 이 extraData 를 0-placeholder 로 씀~~ → **T7.2 에서 해소**: `keygen.WBFTExtraData` 가 생성 검증자셋에서 extra-data RLP 를 계산한다.

### 잔여 (우선순위 순)

| # | 작업 | 왜 이 순서인가 | 상태 |
|---|---|---|---|
| **T7.2** | **wbft extraData RLP 산출** — `keygen.WBFTExtraData`: WBFTExtra(10필드) RLP 를 자체 최소 인코더로 계산(geth 의존성 없음). 게이트: 배포된 preset 의 extra-data 를 자기 메타데이터에서 바이트 동일 재현. GasTip=InitialGasTip·Diligence=DefaultDiligence 는 체인 소스 대조 확인 | T7.1 이 연 "랜덤 키셋" 경로의 유일한 잔여 블로커였음 | ☑ |
| **T7.3** | **`internal/core/launchopt`** — Dialect 2장(geth114 / geth110-wemix) + 관심사 모듈 10 + Builder(cross-module 검증) | 배경 2·알고리즘 7 미충족. 현재 launch args 가 5곳 분산 | ☑ |
| **T7.4** | **launchopt 전환** — `armSpecs`(engine)·`upgrade.LaunchArgs`·`ExtraArgs` 클로저 2곳(chainsetup·cmd)을 Builder 로 흡수 + CLI `--chain-id/--network-id/--launch-opt` 커스텀 심. 게이트는 flag-pair 동등성(레거시 argv 가 관심사를 2회 교차 배치해 바이트 동일은 구조적으로 불가 — `architecture/code-graph.md` §4). 잔여: legacy stack A 의 `nodeconfig.LaunchArgs` 호출 2곳은 T7.11 에서 스택과 함께 이관 | 5곳 → 1곳(레거시 스택 제외) | ☑ |
| **T7.5** | **`internal/app` 유스케이스 층** — 유스케이스 1개=함수 1개, cobra·MCP 타입 무지. NetNew/NetStatus + net 스텝 전체가 이 층 경유 | fan-out 축소는 소비자 이관에 비례(레거시 소비자는 T7.11 잔여) | ☑ |
| **T7.6** | **`net` 원자 스텝** — keys/allocate/genesis/config/launchopts/provision/init/start/stop/restart/rm/logs/health. 각 스텝 = app 함수 1 + CLI 서브커맨드 1 + MCP 도구 1. keys 는 engine.KeySource, argv 는 engine.NodeLaunchArgs(단일 조립 지점) 재사용 | 로컬 타깃 완성; 원격 rm/logs 는 명시적 미지원 오류 | ☑ |
| **T7.7** | **`netcompose.Workspace` → `core/session` 흡수** — `session.Composition`(장수명 환경 모드)이 디렉토리·manifest·스텝 스탬프를 소유, Workspace 는 도메인 상태만 | 잔여 저장소 `core/state` 는 T7.11 에서 스택과 함께 | ☑ |
| **T7.8** | **DSL v2** — env/case 분리, do/expect 통일 문장형(v1 은 같은 시퀀스로 desugar — 실행 경로 1개), strict 파싱, keys/launch/genesis(set·overlay) 선언, schema/v2.schema.json 정본, `migrate-spec`(라운드트립 게이트), hooks.onFail | 미배선 선언은 이름 붙여 거부: genesis existing/build/inherit·role-scoped launch·override hook(G5) | ☑ |
| **T7.9** | **metric 검증원** — portplan 이 metrics 포트(HTTP+3, rpcStep≥4) 할당, collector.ScrapeMetrics(Prometheus 텍스트), `expect:"metric"` 어세션(기본 GreaterOrEqual). metrics 포트 없는 노드는 명시적 실패 | 3-검증원(log·rpc·metric) 완성 | ☑ |
| **T7.10** | **단일 경로 문법** — `netcompose.ParseTarget`: `/local/path` · `user@host:/path` · `ssh://user@host:port/path`. `net new --target` + MCP `target` 인자; 레거시 4-플래그는 유지하되 혼용 거부 | setup 명령의 4-플래그는 T7.11 에서 스택과 함께 | ☑ |
| **T7.11** | **레거시 스택 A 제거** — 진행: `core/probe`→`core/collector`(Detect) · `Plan`→`core/driver` · `pipeline/verify`→**`core/health`**(app.VerifyNetwork 경유) · `pipeline/attach`→**`core/node.AttachedSet`** 흡수 완료. pipeline 3/5 소멸(verify·attach 제거, Plan 이전). **표면 이관 완료**(§1d) · **패키지 이동 완료**(§1e: `pipeline/setup`→`core/bringup`) · **netcompose 대체 진행 중**(§1f: b-1~b-4 완료, b-5·b-6 은 라이브 검증 선행). **잔여**: `pipeline/testrun`+`testkit`(cmd test + mcp, **케이스 이관 91건 선행**) | 표면은 app 1곳으로 수렴 — 남은 건 라이브 검증 후 전환·삭제, 그리고 케이스 이관(작업량) | ◐ |
| — | **T5.2 업그레이드 멀티바이너리** · **T5.5 wemix4 이관** · **실 SSH 라이브 e2e** | §2 기존 항목, 환경 의존 | ☐ |

---

## 1d. T7.11a — 레거시 스택 A 표면 이관 (2026-08-18 완료)

> 목표: 레거시 패키지를 **삭제하는 것**이 아니라, 삭제를 기계적으로 만드는 것.
> 착수 전 실측: `pipeline/setup` 소비자 7파일 · `core/state` 소비자 9파일이 각자
> `state.Load → setup.X → state.Save` 를 반복하고 있었다. 이 중복이 삭제 불가의 실제 원인이었다.

| # | 작업 | 결과 |
|---|---|---|
| **T7.11a-1** | 네트워크 수명주기 유스케이스 | `driver.StopNode/RelaunchNode/StopNodeSet` 흡수(`Plan`→driver 와 동일 근거) · `app.NetworkStatus/NetworkStop/NodeStop/NodeStart/NetworkRemove` + `app.GCSessions` 신설 · `Deps.Driver` seam(프로세스 없이 테스트, 원격 라우팅) · cmd status/stop/node/clean + stop MCP 도구 이관 |
| **T7.11a-2** | 네트워크 기동 유스케이스 | `app.NetworkPlan/NetworkProvision/NetworkLaunch`(단일 `NetworkSpecIn`) + `app.ResolveChain` 신설 · cmd setup 258→175줄(플래그 바인딩·렌더링만) · start MCP 도구가 같은 함수 경유 |
| **T7.11b** | 체인 업그레이드 | `app.HardforkPlan/HardforkExecute` 신설(계획/실행 분리) · cmd hardfork 이관 |
| **T7.11c** | MCP 잔여 표면 | mcp status·setup_plan·resolveNodeSet + cmd test 의 노드셋 해석을 app 경유로 |

**소비자 수렴**: `pipeline/setup` 7파일 → **app + 라이브테스트 1건**, `core/state` 9파일 → **app 전용**.
`cmd/chainbench` 는 두 레거시 패키지를 더 이상 import 하지 않는다.

**이관이 드러낸 결함 2건**(둘 다 테스트가 없어 보이지 않던 것):
- start MCP 도구가 `nodespecs.json` 을 저장하지 않아, MCP 로 띄운 네트워크에서는 `node start` 가 동작하지 않았다 → `NetworkLaunch` 경유로 해소.
- `--endpoints 0` 이 "미지정"과 구분되지 않았다(둘 다 0) → `*int` 로 교정하고 테스트 추가.

**동작 변경 1건(의도)**: 토폴로지 파일의 `chain` 이 `--chain` 을 조용히 덮어쓰던 것을,
사용자가 명시한 경우에는 덮어쓰지 않도록 바꿨다(`ChainExplicit`).

**검증**: `go build/vet ./...` · `go test ./...` · 영향 패키지 `-race` 전부 통과. 신규 단위 테스트 27건.

---

## 1e. (a) 패키지 이동 — 완료

`core/pipeline/setup` → **`core/bringup`**(패키지명 `setup`→`bringup`), 죽은 `Plan = driver.Plan` 별칭 제거,
에러 접두사 정리, `core/state` package doc 에 단일 소유자·수명 명시. **순수 이동 — 동작·시그니처 변경 없음.**
`core/pipeline/` 에는 `testrun` 만 남았고, 그것은 레거시 스택 B 와 함께 사라진다.

이동한 이유: 3-phase pipeline 프레이밍은 이미 소멸했는데(verify→`core/health`, attach→`node.AttachedSet`,
`Plan`→`core/driver`) 이름만 남아 존재하지 않는 phase 를 암시했고, `cmd setup`·`internal/chainsetup` 과 3중 충돌했다.

---

## 1f. (b) netcompose 가 레거시 setup 스택을 대체 — 진행 중

착수 전 실측한 두 스택의 격차. netcompose 는 launch argv(`engine.NodeLaunchArgs`)·포트 할당(`place`)·
키 소스(`engine.KeySource`)를 이미 엔진과 공유하고 있었으나, **네트워크를 기술하는 방법**에 구멍이 있었다.

| # | 작업 | 내용 | 상태 |
|---|---|---|---|
| **b-1** | 구성 패리티(결함) | **syncMode 미렌더 수정** — `Config` 이 `nodeconfig.Params.SyncMode` 를 채우지 않아 모든 노드가 `full` 이었다(steps 로 구성한 snap-sync 테스트가 조용히 full sync 를 돌고 있었음). `--endpoint-syncmode` 신설, validator 는 항상 full. + genesis `--set` 오버라이드·`--overlay` 딥머지(양쪽 fork 순서 재검증) + capabilities 파생(manifest+ws+`delayed-<fork>`+overlay) | ☑ |
| **b-2** | 토폴로지·외부 매니페스트 | `allocate --topology`(노드별 role/sync_mode/bootnode; validator 수는 **요청값이 아니라 해석된 배치**에서 셈 — genesis 가 이 값으로 검증자셋을 만든다) · `new --manifest/--genesis-template`(워크스페이스에 기록 → 이후 모든 스텝이 같은 플러그인 해석) · 체인 해석을 `chains/external.ResolveChain` 1곳으로 | ☑ |
| **b-3** | NodeSet 브릿지 | `Workspace.NodeSet()`/`RPCHost()` + `app.NetworkStatus`/`NetworkStop` 이 **디렉토리의 상태 매니페스트로 스택을 판별**해 양쪽을 읽는다 → `status`/`stop`·MCP 도구가 워크스페이스에서 그대로 동작. 부수 수정: health 스텝이 target 무관하게 `127.0.0.1` 을 찌르던 원격 버그 | ☑ |
| **b-4** | `net up` 매크로 | 9개 스텝을 순서대로 실행하는 유스케이스 1개 + CLI. `--stage provision\|start`. 실패 시에도 성공한 스텝을 출력(워크스페이스는 그 지점부터 손으로 재개 가능). **`--stage=provision` 로 end-to-end 실증**(genesis·config 3개·argv·노드 테이블) | ☑ |
| **b-5** | `setup` → `net up` 전환 | `setup --launch/--provision` 내부를 `NetUp` 으로 교체. **전제가 바뀌었다**: b-6 이 먼저 일어나 `setup` 은 이미 `engine.LocalSetup` 위에 있다. 남은 것은 두 경로(`setup` / `net up`)를 하나로 합칠지의 판단이며, 그 자체가 S 계열(표면 통일)의 문제다 | ◐ **재검토 필요** |
| **b-6** | `core/bringup`·`core/state` 삭제 | ☑ **완료** — #241 병합 시 `engine.LocalSetup`·`session.SaveLocalNodeSet` 로 수렴하며 두 패키지가 소멸했다(§1f-x) | ☑ |

### 1f-x. 병합에서 달라진 것 (2026-08-21)

이 절이 계획한 것과 **실제로 병합된 것이 다르다.** 브랜치가 열려 있는 동안 main 에
#239·#240 이 들어와 `core/pipeline/setup` 과 `core/state` 를 먼저 지웠고, 그 자리를
`engine.LocalSetup`·`engine.BuildLocalPlan`·`session.SaveLocalNodeSet` 이 채웠다.

이 브랜치는 같은 결론에 다른 경로로 도달해 있었으므로(§1e 는 `pipeline/setup` 을
`core/bringup` 으로 **옮겼을 뿐**이고, 실측 파일 유사도 80~99%), 병합 시 셋 중 둘은
main 쪽을 채택했다.

| 쟁점 | 채택 | 근거 |
|---|---|---|
| nodeset 영속 | `session.SaveLocalNodeSet` (main) | [[layers]] 가 `core/state` 를 ❌ 레거시로, `core/session` 을 ✅ 소유자로 판정 |
| plan/provision/launch | `engine.LocalSetup` (main) | `core/bringup` 이 genesis 를 `os.WriteFile` 로 직접 써 파일 통로를 우회 |
| `StopNode`·`RelaunchNode`·`StopNodeSet` | **`core/driver` (이 브랜치)** | `driver.Driver`·`node.NodeSet` 만 쓴다. `engine` 에 두면 노드 하나 멈추는 데 내부 24패키지 import(vs 2) |

조용히 사라질 뻔한 것 하나를 옮겼다: `bringup.Run` 은 setup 진행 이벤트를 발행했으나
`engine.LocalSetup` 은 하지 않았다. 끝날 때까지 아무것도 보고하지 않는 기동은 멈춘 것과
구분되지 않으므로 `LocalSetup` 에 `Bus` 를 더했다.

### 라이브 검증 결과 (2026-08-18, 실제 바이너리)

| 체인 | `net up` | 블록 전진 | `run` api 9건 | 고아 |
|---|---|---|---|---|
| stablenet | 성공 | 97 → 110 → 122 | 9/9 | 0 |
| wbft | 성공 | 24 → 36 → 48 | 9/9 | 0 |
| **wemix** | **실패 — 블록 0** | 정지 | — | 0 |

wemix 실패의 원인은 `net up` 이 아니라 **`poa.BuildGenesis` 가 템플릿 치환을 하기 때문**이다 —
`alloc:{}`·`minerNodeId:"0x0"` 인 죽은 genesis 가 나오고 노드가 ethash 로 돈다. 같은 4노드를
**실제 절차대로 손으로** 띄우면 정상이다(거버넌스 컨트랙트 5개 · etcd 4멤버 · 블록 238→268 ·
4노드 전부 sealing, 20블록에서 5/6/5/4). 즉 **F4(`GenesisArtifacts`)·F5(poa 액션 배선)가 이 결함의
수정분**이고, b-5 의 wemix 몫은 그것을 기다린다.

wbft 계열은 게이트를 통과했으므로 b-5 를 stablenet/wbft 한정으로 먼저 끊을 수 있다.

### b-5 를 지금 하지 않은 이유

`--stage=start` 는 실제 프로세스를 띄우므로 **체인 바이너리 없이는 검증할 수 없다.** 그리고 전환은
관측 가능한 변화를 동반한다 — **포트 대역이 달라진다**(bringup: `ports.base_*` = http 8501·p2p 30301 /
netcompose: `place` 할당 = http 8600·p2p 31000, step 10). 기존 절차 문서·라이브 스크립트가 이 번호에
의존한다면 함께 갱신해야 한다.

**검증 절차(바이너리 보유 환경)**:
```sh
chainbench net up --data-dir /tmp/n1 --chain stablenet --binary $GSTABLE_BIN \
  --keys keys/preset --validators 4
chainbench net health --data-dir /tmp/n1     # 블록 전진 확인
chainbench run --data-dir /tmp/n1 tests/specs/api/*.json
chainbench net stop --data-dir /tmp/n1       # 고아 0 확인
```
이것이 통과하면 b-5(내부 교체)는 작은 변경이고, b-6(삭제)이 뒤따른다.

---

> **여전히 미배선인 선언 1건**: `testspec.Deps.Keys` 는 타입으로만 존재하고 소비자가 없다.
> 현재 `sendTx` 는 노드측 unlocked 계정으로 서명하므로(`eth_sendTransaction`) 로컬 서명키를 쓰지 않는다.
> **소비자가 생길 때(로컬 서명 tx 액션) 배선한다** — 소비자 없이 값을 넣으면 T3.2b 가 고친 "선언만 하고 방출 안 함"을 되풀이하게 된다.

---

## 1g. 아키텍처 정합 · 패밀리 기동 (사전 작업 완료 · K0 부터 착수)

> 설계 근거: [[layers]](architecture/layers.md) · [[module-responsibilities]](architecture/module-responsibilities.md) ·
> [[family-bringup-design]](family-bringup-design.md).
> **작업 상태는 이 절에서만 관리한다** — 설계 문서는 *무엇을 왜*, 이 절은 *언제 어디까지*.

세 갈래가 있고 서로 독립이다. A(규칙) → B(파서) → F(패밀리 기동) 순으로 착수하되,
B 는 F 와 병행 가능하다.

### 완료 — 착수 전 사전 작업 (2026-08-18~19)

코드 갈래에 들어가기 전에 **가정을 사실로 바꾸는 일**만 먼저 했다. 아래 셋이 없으면
K0·S0 가 추측 위에 서게 된다.

| # | 작업 | 무엇이 확정됐나 | 상태 |
|---|---|---|---|
| **P-1** | BLS 파생을 순수 Go 로 실증 | `kilic/bls12-381` + stdlib `crypto/hkdf` 로 **preset node1~5 의 BLS 공개키·PoP 이 바이트 동일**하게 재현됨, `CGO_ENABLED=0` 에서. → **K1(`--bootnode` 제거)이 가능하다는 것이 증명됨.** 함정 둘: PoP 의 DST(`..._POP_`) 누락 시 형식은 멀쩡한데 검증 실패 · `blst_keygen` v4 는 salt 를 루프 **이전에** 한 번 해시 | ☑ |
| **P-2** | `netcompose/target.go` → `core/target` 이동 | L4 에 있던 타깃 해석이 L1 로 내려감(`driver`/`provision`/`remote` 만 import). S 계열의 선행 조건 | ☑ |
| **P-3** | **문서 통치 구조** — 전 문서에 등급(정본/현행 설계/이력/대체됨) 표기 + `docs/README.md` 에 권위 순서 + 대체된 2건 `dev/archive/` 이동 | 문서끼리 어긋날 때 **무엇이 이기는지**가 정해짐. 이걸 한 이유는 아래 참조 | ☑ |

**P-3 을 한 이유**는 낡은 문서가 아니라 **등급이 없는 문서**였다. 23개 `dev/` 문서 중 자기 지위를
밝힌 것이 4개뿐이어서, 정본인 [[chainbench-requirements-review]] §D-2.8 을 제안 문서로 오인해
건너뛰었고 그 결과 하드포크 분류를 반대로 잡았다(A7b 가 그 정정분이다). 문서를 지웠다면
**정답이 든 문서가 사라졌을 것**이므로, 삭제가 아니라 등급 표기로 해결했다.

부수 정리: 저장소의 키가 전부 테스트 픽스처임을 루트 `README.md`·`tests/README.md` 에 명시하고
(스캐너 검출 32건이 정상 상태임과 **진짜 유출을 가려내는 기준**을 함께 적음), 파일명 오탐용
`.precommit-allow` 를 도입했다(면제는 파일명 검사에만, 내용 스캔은 그대로).

### A — 규칙을 코드로 (설계가 썩지 않게)

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **A1** | 레이어 검사 테스트 (`internal/arch`) — [[layers]] §3 표를 **파싱**한다(코드에 복제하지 않음) | ☑ 상향 의존 **0건** · 미배치 패키지 거부 · 유령 항목 거부 · 네 실패 경로를 실제로 확인 |
| **A2** | 상태 쓰기 허용목록 테스트 (`internal/arch`) — [[layers]] §5 표가 정본 | ☑ ❌ 4곳(app·chainsetup·wemix/deploy·upgrade)은 예외로 명시 · 신규 위반 차단 · **stale 예외도 차단** · `os.Create` 를 세면서 미기재였던 `engine` 을 찾아냄 |
| **A3** | `app` 의 `topology.yaml` 쓰기를 `provision.FileStore` 경유로 | ☑ **정리가 아니라 결함 수정이었다** — 원격 프로비전이 genesis·config 를 조작자 머신에 쓰고 있었다(신원만 원격으로 갔다). `Deps.Files` seam 추가 · 드라이버가 파일을 보낼 수 있으면 그것이 기본 저장소 · 회귀 테스트 3건 |
| **A4** | `chains/wemix/deploy` 를 `FileStore` 경유로 | ☑ `pullKeystores` 가 읽기·쓰기 양쪽 모두 store 경유. 직접 파일 쓰기 0건 |
| **A4b** | `chainsetup`·`consensus/upgrade` 를 `FileStore` 경유로 | ☑ **완료 2026-08-23.** 실측은 8+2 가 아니라 **11+2 = 13곳**. `Options`/`HandoffOptions`/`upgrade.LaunchOptions` 에 `Files` seam(nil=로컬). **직접 쓰기 0건**이 되어 [[layers]] §5 의 ❌ 가 사라졌고, **A2 가 stale 예외를 잡아** 표에서 지우게 했다(가드가 의도대로 동작). `os.MkdirAll` 대부분이 함께 사라진 게 핵심 — `Write` 가 부모를 만들므로 디렉토리를 미리 만들던 코드는 **경로를 아는 코드**였고 그게 위반의 실체였다. 라이브: `chain up --case wemix` 15/15 · 엔진 게이트 통과 | ☑ |
| **A5** | ~~`core/bringup`·`core/state`~~ 삭제 완료 · `testkit`·`core/pipeline/testrun` 잔여 | 앞 둘은 #241 에서 소멸(`engine.LocalSetup`·`session.SaveLocalNodeSet` 로 수렴). 뒤 둘은 **케이스 이관 선행** | ◐ |
| **A6** | `netreg`·`obs` 파일 싱크를 `session` 으로 흡수 검토 | 컨트롤 플레인 단일화 | ☐ |
| **A7b** | **`hardfork` 와 `upgrade` 통합** — hardfork 가 상위 범주, upgrade 는 type-1 핸드오프. 선언은 `Hardfork{AtBlock, BinaryAfter, ProducersAfter}` 하나, **메커니즘(스왑/핸드오프)은 파생** | 세 사례(같은체인 스왑 · wemix→wbft 핸드오프 · genesis 전용 포크)가 한 선언에서 갈림 · 명령 둘 → 하나 | ☐ |
| **A7** | **이름 겹침 검출 테스트** — exported 식별자가 2개 이상 패키지에 같은 이름이면 보고(관용 허용목록 명시) | 실측(08-21): `Identity`×2 · `Plan`×4 · `Config`×3 · `Step`×4. `Node`×3 은 K3 에서 ×2 로 줄었다 | ☐ |

**이름은 각 항목이 자기 범위에서 함께 고친다**([[layers]] §5b). 개명만 하는 커밋은 리뷰가 어렵고
동작 변경과 섞이면 더 어렵다. 규칙: **한 개념 = 한 이름, 다른 개념 = 다른 이름, 식별자는 명명된 타입.**

**A1·A2 를 먼저 하는 이유**: 이후 모든 작업(B·F 포함)이 규칙 위반을 자동으로 잡힌다.
설계를 지키는 일을 사람의 주의력에 맡기지 않는다.

### B — DSL 파서를 모듈로

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **B1** | **`testspec`(2,998줄) 4분할** — `dsl`(L1 구문 689) · `dsl/assert`(L1 368) · `dsl/bind`(L1 259) · `dsl/interp`(L3 2,050) | `chainbench validate` 가 rpc/session 을 링크하지 않음 · 파서 fuzz | ☐ |
| **B2** | `Spec.Fingerprint()` 가 `string` 반환 (현재 `session.Fingerprint`) | **구문이 L3 에 묶인 유일한 이유**가 이 타입 하나다 — B1 의 선행 조건 | ☐ |

현재 `testspec` 이 `collector`·`session`·`rpc`·`keyreg`·`accounts` 를 import 해서
**순수해야 할 파서가 L3 로 끌려 올라가 있다**([[module-responsibilities]] §3).

**이름 주의**: 인터프리터를 `dsl/engine` 으로 두면 `internal/engine`(테스트벤치 엔진)과 겹친다.
**`engine` 은 하나뿐이어야 한다** — 주도하는 쪽이다. 인터프리터는 `dsl/interp`.
이 구조(엔진이 파싱→환경준비→인터프리터 순차실행→기록을 주도)는 **이미 코드에 있다**
(`engine.Run` + `engine/wire.go`의 `NewRunSpec`).

**레지스트리는 import 가 아니라 주입이다.** 엔진(L3)이 기능 레지스트리(L5)를 import 하면 상향이지만,
`interpreter.go` 가 이미 `Registry`·`Action`·`Assertion`·`Deps` 를 **스스로 정의**하고 L5 가 구현체를
넘긴다 — 값 전달이라 층을 거스르지 않는다. 코드 변경이 아니라 문서 문구 문제였다.

### F — 패밀리별 기동 (wemix 를 Go 로)

| # | 작업 | 게이트 | 리스크 | 상태 |
|---|---|---|---|---|
| **F1** | `PortReservation` — 패밀리별 포트 대역. `serverset` 전역 `p2pStep>=2` 제거 | ☑ **완료 2026-08-22.** poa {3,3}·wbft {2,3} · poa 는 step 2 를 거부 · wemix 노드가 **etcd client(p2p+2)까지 예약**(실측: workspace `etcdClient: 31002`) · stablenet 포트 불변 | 낮음 | ☑ |
| **F2** | `--networkid` 방출 | ☑ **완료 2026-08-22.** 다이얼렉트가 아니라 **매니페스트 사실**이라 모든 체인에 방출(argv 실측 `--networkid 8283`) · `--network-id` 오버라이드는 상위 레이어라 그대로 이김 · stablenet 4노드 라이브 블록 전진 | 낮음 | ☑ |
| **F3** | `BringUpPhases` + `Deps.Launch(nodes)` + `Deps.Action` | ☑ **완료 2026-08-22.** wbft 는 1페이즈(노드 목록 없음 = 전체) — stablenet 4노드 라이브 불변 · poa 는 boot(deploy-governance·etcd-init·verify-etcd) → rest · **미배선 액션은 오류**(LeaderGate·SwapBinary 와 동일 계약) · 액션 실패 시 다음 페이즈 미기동 | **중** | ☑ |
| **F4** | `GenesisArtifacts` + `WemixGenesisSource`(`poa.PrepareTemplate`→`GenerateGenesis`) | ☑ **완료 2026-08-22.** wbft `Extra` nil · poa config `Validate()` 통과 · `poa.Family.BuildGenesis` 는 이제 **거부**(치환 템플릿을 genesis 로 돌려주던 자리) · **라이브(실 gwemix)**: alloc 4계정·extraData 실값·chainId 8285(매니페스트)·`gwemix init` 수용 | 중 | ☑ |
| **F5** | poa 액션 배선 + 유스케이스 수렴(3곳→1곳) | ☑ **라이브 통과 2026-08-22** — wemix 4노드가 **일반 엔진 경로**로 뜬다: 바이너리 생성 genesis → boot 페이즈(프로듀서 단독) → deploy-governance(컨트랙트 5종) → etcd-init → **verify-etcd 가 클러스터 확인** → rest 페이즈 → 블록 전진. `TestWemix_Live_BringUp`(GWEMIX_BIN 게이트). **스텝 경로도 페이즈를 태운다**(2026-08-23): `net up`/`net start` 가 패밀리 페이즈 순서로 기동하고 사이에 부트스트랩 액션을 실행한다 — wemix 가 엔진·스텝 양쪽에서 뜬다. **F5b 완료 — 다중 프로듀서 sealing 로테이션**(2026-08-23): `net up --validators 4` 46초, 클러스터가 넷을 다 담고(`node1..node4`) **최근 25블록 봉인자 4명(8/6/6/5)**. 조인은 rest 페이즈의 `etcd-join` 액션이 수행한다.

가설(순서)은 **틀렸다**. go-wemix 소스가 말하는 실제 계약은 둘이다. ① **`admin.etcdJoin(name)` 의 인자는 조인할 노드가 아니라 물어볼 상대**다 — 조이너가 eth 와이어(`EtcdAddMemberMsg 0x14`)로 피어에게 요청하면 피어가 클러스터 문자열로 답하고 조이너가 그걸로 자기 서버를 띄운다(`wemix/etcdutil.go:1289`, `eth/protocols/eth/wemix_handlers.go:77`). 자기 이름을 넘기면 **에러 없이 아무 일도 안 일어난다** — 앞선 세션이 여기서 막혔다. ② 조이너는 거버넌스 멤버 목록을 **체인에서 읽어** 알게 되므로, 방금 뜬 노드는 아무도 몰라 `not found` 로 거절한다. 그래서 `admin.wemixInfo.nodes` 에 상대가 보일 때까지 기다린 뒤 조인한다. 또 조인이 **null 을 반환하고도 클러스터에 안 들어가는 경우**(4대 중 1대)가 실측돼, 반환값이 아니라 **클러스터를 증거로** 재시도한다. | 중 | ☑ |
| **F6** | `chainsetup/wemix.go` 의 `NotImplemented` 제거 | ☑ **라이브 통과 2026-08-23** — `chain up --case wemix --validators 4` 가 **15스텝 전부 OK**(deploy-governance 12.3s · etcd-join 37.6s · head 22), 25블록 봉인자 4명. wemix 케이스는 `Supported`. 절차 자체도 정정했다: **선언된 순서가 틀려 있었다**(genesis 가 allocate 앞 — 거버넌스 멤버는 배치에서 나오는 ip/port 를 담는다), 그리고 `launch-rest`·`etcd-join` 이 아예 빠져 있었다. 러너는 절차 사본을 갖지 않고 netmap·WemixGenesisSource·패밀리 페이즈·WemixBootstrap 을 그대로 조립한다. 스텝 id 와 패밀리 액션 이름이 어긋나면 테스트가 막는다. | 낮음 | ☑ |

**F1 은 버그 수정이다** — 현재 `p2pStep>=2` 는 wemix 에서 틀렸다(etcd 가 p2p+1 peer·p2p+2 client 둘을 쓴다).

**F3 이 유일한 실질 리스크다.** `Deps.Launch` 시그니처 변경이 engine·netcompose·chainsetup 에
파급된다. `nodes=nil → 전체` 규약이면 이관은 기계적이고, wbft 계열은 argv·순서가 바이트 동일하게
유지되므로 stablenet/wbft 회귀를 §1f b-5 절차로 즉시 확인할 수 있다.

### K — keyring (첫 착수 대상)

> 근거: [[keyring-design]](keyring-design.md).
> **키는 세 체인이 동일하다**(실증: 같은 nodekey → 세 체인 동일한 address·pubkey, BLS 도 Go 파생이
> `bootnode` 출력과 바이트 동일). 그래서 여기부터 정리하면 위쪽이 단순해진다.
> 실측 문제: 키 관심사가 **5패키지 1,236줄**에 흩어져 있고(읽기 `keys`/쓰기 `keygen`/저장 `keymat`/
> 런타임 `keyreg`), preset 이 **신원·네트워크 결정·파생 산출물 셋을 섞어** 담아 preset 을 전제로 만든다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **K0** | `core/keyring` 신설 — nodekey 생성 · 신원 파생(주소·devp2p 공개키·BLS·PoP, 전부 in-process) | 배포 preset 의 node1..5 를 nodekey 만으로 **바이트 동일 재현**(골든) · `CGO_ENABLED=0` · fuzz | ☑ |
| **K1** | `--bootnode` 제거 — BLS 를 자체 파생 | **`PATH` 를 비운 채 4노드 키셋 생성 성공** · 세 계층(keygen·engine·CLI)에 각각 게이트 테스트 | ☑ |
| **K2** | `keygen.WBFTExtraData` → `consensus/wbft.ExtraData` | 골든 유지 · **`BuildGenesis` 가 비어 있으면 파생** · `Take` 의 stale extraData 결함 수정 | ☑ |
| **K3** | `core/keys`·`keygen`·`keymat`·`core/keyreg` 흡수 | **4패키지 → 1**(1,565줄 → `core/keyring` 1,497줄) · **신원 타입 5 → 1** · `Nodekey` → `PrivateKey`(역할이 아니라 실체로 명명) | ☑ |
| **K4** | `keyring` 명령 — new/add/list/show/import/export | `--keyring` 출처 표기(플래그/env/기본) · `--with-bls` 선택 · `export` 는 `--yes` 필수 · `list --verify` · `add` 는 검증자 승격 안 함 · 기존 3개 그룹 유지(deprecated) | ☑ |
| **K5** | preset 분해 — 신원과 네트워크 결정을 타입으로 분리 | `Preset{Nodes, Network}` · `NetworkFor(n)` 이 선언 유무를 흡수 · **`keyring new --validators 0` = 신원만** · **라이브: 신원만 있는 링으로 stablenet 4노드 블록 생성 + api 9/9** · 기존 preset 읽기 호환 | ☑ |
| **K6** | `provision.FileSink` → `FileStore` (읽기 추가) | **자체 SSH 파일 I/O 9곳 → 0** · 와이어 형식 정의 1곳 · `keyring.FileSource` 가 로컬·원격 겸용 | ☑ |
| **K8** | **표면 통일** — 유스케이스를 `internal/app` 으로, CLI·MCP 는 노출 수단으로 | CLI 로 만든 링을 MCP 가 읽음(실증) · MCP 도구 5개(`new`/`add`/`list`/`show`/`import`) · **`export` 는 의도적 부재**(비밀이 에이전트 기록에 남지 않도록, 부재를 테스트로 고정) · `GenerateOpts.Validators` 를 `*int` 로 바꿔 "미설정"과 "없음"을 타입으로 구분 | ☑ |
| **K7** | `--from` 단일 경로 문법 + `srv://<인벤토리이름>/path` | **네 표기가 한 코드로**(로컬·srv·host:path·ssh://) · **명령줄에 IP 없음** · 플래그 4개 → 1개(구 플래그는 deprecated 유지) | ☑ |

**의존성 추가**(K0 에서 완료): `github.com/kilic/bls12-381 v0.1.0` — **순수 Go** BLS12-381.
당초 적었던 `supranational/blst` 는 **CGO 라 `CGO_ENABLED=0` 빌드를 깬다**. kilic 은 go-wbft 의
`go.sum` 과 **모듈 해시가 동일**하다. `decred/…/secp256k1/v4` 는 이미 간접 의존성이었고 직접으로 승격.

**K6 이 keyring 을 넘어선다**: `FileSink` 에 읽기가 없어서 `keymat` 이 자체 SSH 읽기를 따로 만들었다 —
추상화가 한쪽 방향만 있으면 반대 방향은 옆에 새로 생긴다. 넓히면 청사진 읽기·genesis 확인·
산출물 검증이 전부 같은 통로를 쓴다.

**DST 주의**(K0): PoP 서명의 DST 를 빼면 **형식은 멀쩡한데 검증 실패하는 PoP** 이 나온다.
실제로 첫 파생 시도에서 그렇게 됐다 — 골든 테스트로 고정한다.

### N — 네트워크 청사진 (구성 정보를 하나의 선언으로)

> 근거: [[network-blueprint-design]](network-blueprint-design.md).
> 실측 문제: 구성 정보가 **4조각**(`topology.yaml`·`serverset`·`keys/preset`·`poa.Config`)으로 흩어져
> 어느 것도 전체를 말하지 못한다. **preset 이 선택이 아니라 전제**이고(`keys.LoadPreset` 필수),
> 바이너리 경로가 **20개 파일**에 분산돼 있다. 노드별 nodekey·계정·포트·서버를 지정할 수단이 없다.

> **원칙: raw 가 먼저, preset 은 나중.** 손으로 쓴 값만으로 네트워크가 서는 것을 먼저 만들고,
> 그 위에 preset 을 **생성기**로 얹는다. 반대로 하면 preset 이 다시 전제가 된다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **N0** | 역할을 `bp·en·pn` 3종으로 정리 — **NM1 로 흡수**([[netmap-design]] §4) | `validator→bp`·`endpoint→en` 이관 · `node.RoleEN`/`RoleEndpoint` 중복 해소 · 기존 토폴로지 호환 | ◐ NM1·NM1c 완료. 남은 것은 **방출 전환 = NM6**([[netmap-design]] §4) — 조립이 기록하는 역할 값을 `bp/en` 으로 바꾸고 `LegacySpelling` 을 지우는 일이다. NM3 에서 분리했다(`netmap.Is` 로 두 철자가 다 안전해져 급하지 않고, argv 가 바뀌어 라이브 재검증이 따로 필요하다) |
| **N0b** | **피어링 그래프를 역할에서 파생** — 현재는 풀메시 고정. `bp ↔ pn ↔ en`(en 은 bp 를 직접 모른다) | ☑ **NM2 완료** — mesh 골든 동일 · proxied 는 bp/en 의 목록에 pn 만 · poa+pn 은 `SupportsRole` 로 거부. 조립 4곳의 **전환**은 NM3 | ☑ |
| **N1** | `blueprint` 선언 스키마 + 파서 (L1 순수) | 부분 청사진 round-trip · 미지 필드 거부 · fuzz | ☐ |
| **N2** | `Resolve` — 출처 사슬(명시>인벤토리>키셋>플러그인>패밀리>내장) + `Sources` 기록 | 같은 청사진 → 항상 같은 `ResolvedNetwork`(결정성) | ☐ |
| **N3** | **raw 경로 완성** — 노드별 nodekey·계정·포트·서버·바이너리 오버라이드 선언 | **preset 없이** 손으로 쓴 청사진만으로 3체인 4노드 기동(라이브) | ☐ |
| **N4** | `Materialize` — 노드별 산출물 묶음 → Sink | 로컬/원격 분기 없음 | ☐ |
| **N5** | **그 다음** preset 지원 — `net blueprint --from-preset` 이 청사진을 **생성** | preset 산출 청사진이 N3 경로와 동일 결과 | ☐ |
| **N6** | `topology.yaml` 흡수 (이관 기간 병존, 혼용 거부) | | ☐ |
| **N7** | **`core/netmap`** — ☑ **완료 2026-08-22** (NM1·NM1c·NM1b·NM2·NM3·NM4·NM5). 배치의 단일 소유자가 섰다: 자원 풀·결정적 할당·양방향 대장·라벨·피어링·경로. `place` 할당기 소멸 | 실측(착수 전): 배치 타입 8개 · 포트 표현 3벌(etcd 소실) · 역할 어휘 4벌 · static-nodes 조립 4벌 전부 풀메시 · `"node%d"` 라벨 파생 32곳. **종료 시**: 포트 1벌(etcd 보존) · 역할 폴딩 1곳 · 피어링 1곳(mesh/proxied) · 경로 파생 1곳 | ☑ |
| **N8** | ~~`serverset` 가용 자원 풀~~ → **이미 있음**(`slots`·포트 밴드, 2026-08-18). 명시적 범위 풀은 근거 생기면 | — | ☑ |
| **N9** | **해석 순서 강제** — ① keyring → ② netmap → ③ enode → ④ genesis ⑤ config | ③ 이전에 ④⑤ 를 만들 수 없다(컴파일 타임 또는 명시적 오류) | ☐ |
| **N10** | **계정 라벨** — `account1` ↔ 주소·개인키. **faucet 누락은 오류** | 테스트 정의에 주소가 등장하지 않음 · 잔액 0 계정의 가스 자금원 보장 | ☐ |
| **N11** | **다중 config** — 이름으로 참조, 노드별 지정. `restartNode` 액션은 **이미 있고** `config:` 인자만 추가 | 일부 노드만 다른 config 로 재기동 | ☐ |
| **N12** | **deploy skip 을 내용 해시로** (현재는 존재 여부만) | 같은 경로에 **다른 내용**이면 skip 하지 않음 | ☐ |
| **N13** | **skip 사유 기록** — `TestRecord.Status(s)` 에 사유가 없어 "왜 skip 됐는지"가 아티팩트에 안 남는다 | ☑ **F5 와 함께 완료 2026-08-22** — `TestRecord.Reason(why)` + `statusDoc.Reason`. 적용 4곳: 파싱 실패·미적용(`does not apply to this target (chain or required capabilities)`)·blocked 2종. 이유 없는 blocked 하나가 `chain.binary` 누락을 찾는 데 한 세션을 썼다 | ☑ |
| **N14** | **`capability` 이름 충돌 정리** — `engine/capability`(DSL 게이팅)와 `core/capability`(표면 카탈로그)가 무관한데 같은 이름 | S 계열의 `feature` 레지스트리와 함께 정리 | ☐ |

**N7~N14 는 2026-08-19 요구 재도출분**([[network-blueprint-design]] §6). 세 체인을 실제로 구성한
기록에서 다시 뽑았고, 다섯 요구(로컬 포트·원격 IP풀·라벨 지정·enode 순서·계정 라벨)가 전부
**같은 것 하나**를 가리켜 `netmap` 으로 모았다.

**키 파생 주의**(N3): wbft 계열은 nodekey 하나에서 계정·BLS 가 파생되므로 `bootnode` 바이너리가
필수다. poa 는 계정이 nodekey 와 독립이라 `account:` 를 따로 선언해야 한다.
**BLS 는 선언 필드가 아니다** — 선언하면 nodekey 와 어긋날 수 있고, 그 불일치는 합의에서 터진다.

**`pn` 주의**(N0b): 세 체인에 proxy 모드 플래그가 **없다**(실측). `pn` 은 argv 가 아니라
**static-nodes 그래프**로 표현된다 — `bp ↔ pn ↔ en`, en 은 bp 를 직접 알지 못한다.
현재 구현은 풀메시라 pn 을 두어도 효과가 없다.
**poa(wemix)는 pn 을 쓰지 않는다** — etcd 가 그 자리다. 선언하면 조용히 무시하지 말고 **오류**로
거부한다(`Family.SupportsRole`).

**N 과 F 의 관계**: F4(`GenesisArtifacts`)·F3(`BringUpPhases`)는 `ResolvedNetwork` 를 입력으로 받는다.
**N1·N2 를 먼저** 해야 F 가 조각을 다시 모으지 않는다.

### S — 표면 통일 (CLI/MCP/DSL 을 한 레지스트리로)

> 근거: [[surface-unification-design]](surface-unification-design.md).
> 실측 문제: `cmd/chainbench` 4,569줄 중 **21파일이 `app` 을 우회**해 L1~L4 를 직접 부른다.
> MCP 도구 46개 중 **34개가 JSON 스키마 손작성**. DSL 은 또 다른 레지스트리를 갖는다 —
> **기능 목록이 세 벌**이고 실제로 갈라졌다(`faucet`·`verify` 는 DSL 에 없다).

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **S0** | **`internal/feature`**(별도 패키지, `Deps` 소유) 레지스트리 골격 · 입력 태그→cobra 플래그/JSON 스키마 바인딩 · **`ReadOnly` 속성**(선언식 — 상태 불변 + 출력에 비밀 없음; `keyring export` 는 자격 없음) | 기존 동작 무변경 · 미등록 기능 카운트 테스트 · ReadOnly 선언이 스키마에 노출 | ☐ |
| **S1** | ① Compose 이관 — `net.*` 9스텝 등록(이미 `app` 경유라 등록만) | `net up` 3체인 회귀 | ☐ |
| **S2** | MCP `net_*` 를 레지스트리 소비로 전환 | 손작성 스키마 감소분 측정 | ☐ |
| **S3** | ② Test 이관 — `tx`·`faucet`·`contract`·`verify` | CLI/MCP/DSL 동시 노출 확인. **선례: keyring(K8)** 이 같은 형태로 끝났다 — 유스케이스는 `app`, 표면은 바인딩과 렌더링만 | ☐ |
| **S4** | ③ Report 이관 — `status`·`report`·`logs` | | ☐ |
| **S5** | `cmd/` 규칙 위반 21파일 정리(`upgrade_run.go` 395줄부터) | `cmd/` 가 `app` 만 import · 4,569→~1,800줄 | ☐ |
| **S6** | `cmd` import 화이트리스트 테스트 | 재발 차단 | ☐ |
| **S7** | **`query` 조회 투영** — `ReadOnly` 기능들을 최상위 `query <명사> <동사>` 로 **자동 생성**(손 트리 금지, [[surface-unification-design]] §4.4, 확정 2026-08-25). 정본 철자는 명사 그룹, `query` 는 같은 등록의 두 번째 렌더링. MCP 는 같은 속성으로 조회 전용 도구 목록을 얻는다 | `query keyring list` == `keyring list` (같은 등록 실증) · 비-ReadOnly 기능이 query 에 나타나면 테스트 실패 | ☐ |

**`internal/feature` 를 별도 패키지로 두는 이유**: `app/feature` 로 하면 `Invoke(ctx, Deps, in)` 의
`Deps` 가 `app` 에 있어 `app → app/feature → app` **참조 순환**이 된다. 순환은 발생해서는 안 되며,
발생했다는 것 자체가 설계 미흡의 증거다. `feature` 가 `Deps` 를 소유하고 `app` 이 그것을 import 한다.

**F 계열과의 순서**: 독립이지만 `net start` 를 둘 다 건드린다 — **F3(페이즈 구조)을 먼저** 하고
S1 에서 등록해야 두 번 등록하지 않는다.

### D — 대시보드 디버깅 지원 (metric 시각화부터)

> 근거: [[dashboard-metrics-design]](dashboard-metrics-design.md). 결정(2026-08-24):
> **Prometheus·Grafana 서버 없이 자체 동작** — 노드의 `/debug/metrics/prometheus` 를
> 대시보드가 직접 긁는다. 파서(collector)·대상 목록(netmap)·SSE 배관·SPA 골격은 재사용.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **D1** | metric 인포그래픽 — 주기 스크레이프 루프 + 메모리 링버퍼 + SPA 차트(블록 높이·피어·txpool 노드별 비교) | 실행 중인 네트워크에서 노드별 차트가 실데이터로 갱신 · 외부 프로세스 의존 0 · 레퍼런스는 설계 문서 §4 (statsviz·Gatus·expfmt 등) | 미착수 |
| D2 | 완료 세션 화면 — 있는 `/api/sessions`·chainstate API 에 화면 소비자를 만든다 | 완료 세션의 판정·체인 상태 이력이 브라우저에서 열람 | 미착수 |
| D3 | 로그 연계 — 차트·이벤트 시점에서 해당 노드 로그로 점프 | collector tail 배관 재사용 | 미착수 |

### A(환경) — 지금 이 환경에서 무엇이 돌고 있는가

> 근거: 13개 체인 구성 요구 중 ⑩⑪⑫⑬ 는 한 문제다 — **아무도 환경의 현재 상태를 모른다.**
> 실측(2026-08-24~25): `os/signal` import 0건 · 파일 락 0건 · `preflight` 는 선언 정합성만
> 검사하고 살아있는 포트는 안 본다 · `procman` 은 자기가 띄운 것만 안다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **A1** | 인터럽트 핸들러 + 노드 수명 분리 + 실패 경로에서도 기록 | ☑ **완료 2026-08-25.** `net up --chain wemix` 를 30초에 인터럽트 → 종료 130 · 노드 4개 · **PID 4개 기록** · `net stop` 이 0으로 정리. 드러난 결함 2건: 노드를 `exec.CommandContext(요청 ctx)` 로 띄워 CLI 호출에 종속(취소되자 4개 중 3개만 남음) · 워크스페이스를 성공 시에만 저장해 실패한 스텝이 PID까지 버림 | ☑ |
| **A2** | 워크스페이스 락 — 실행 단위, 4상태(free/live/stale/foreign) | ☑ **완료 2026-08-25.** 46초 wemix 기동 중 t=20s 동시 실행이 pid·host·시각·명령줄과 함께 거부됨. **락은 호출이 아니라 실행 단위** — `net up` 이 9스텝을 부르며 각각 또 잡으므로 재진입 허용, 가장 바깥만 해제(첫 구현은 안쪽 release 가 바깥 락을 지워 동시 실행이 그대로 들어왔다) | ☑ |
| **A3** | 기동 전 포트 점유 조회 (`core/occupancy`) | ☑ **완료 2026-08-25.** `init` 직전(타깃에 쓰기 전) 전 노드 포트를 조회해 거부하고, **우리 것인지 남의 것인지** 분류한다. **dial 로는 못 잡는다**: 노드가 와일드카드 소켓에 바인드하면 루프백 dial 이 거부된다(lsof `*:8600` 점유 중 dial 실패 실측). bind 도 한 형태로는 부족 — 실측: 8600(와일드카드 점유)은 `127.0.0.1` bind 성공/`:` bind 실패, 8603(루프백 점유)은 정반대. **둘 다 성공해야 비어 있는 것**으로 판정하니 20개 리슨 소켓을 20개로 정확히 잡는다 | ☑ |
| A4 | 원격/fleet 점유 조회 — 모든 서버 대상(요구 ⑩의 "모든 서버") | dial 경로는 있으나 fleet 다중 호스트는 R5 선행 | ☐ |

### G — genesis 정리 (리팩토링 순서 ③)

> 순서: keyring ☑ → netmap ☑ → **genesis** → config. 실측(2026-08-25): genesis 를
> 만드는 진입점이 흩어져 있고, 그 결과 경로마다 기능이 달랐다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **G1** | **조립 지점 단일화** — `engine.BuildGenesis`(소스 선택 + 커스터마이즈)를 모든 경로가 부른다 | ☑ **완료 2026-08-25.** 소스 선택이 **5곳 → 1곳**(엔진·스텝 두 경로가 `poa.FamilyID` 로 분기 + 세 곳이 하드코딩). 커스터마이즈를 소스 **밖으로** 빼서 `netcompose.customizeGenesis`(=`genesis.BuildNetwork` 의 옵션 처리를 줄 단위로 재구현한 사본) 소멸. 라이브: stablenet 4노드 · `chain up --case wemix` 15/15 | ☑ |
| **G1a** | 닫힌 격차 — **wemix + overlay** | ☑ genesis overlay 가 스텝 경로에선 반영되고 엔진 경로에선 **말없이 버려지던** 분기. 커스터마이즈가 소스 안(한 패밀리만 쓰는 곳)에 있었던 탓. 이제 패밀리가 만든 base 위에 동일하게 적용된다 | ☑ |
| G2 | `chainsetup/handoff_driver` 의 자체 genesis 조립(`BaseGenesis` + `MergeOverride`)도 같은 조립으로 | 핸드오프 경로는 `BringUpPhases` 도 안 쓴다 — 함께 처리 | ☐ |
| G3 | genesis 산출물의 by-product(`Extra`) 배치 규칙 정리 — 지금은 소비자마다 따로 쓴다 | | ☐ |

### R — 로컬 docker 를 원격 서버처럼 (원격 경로 검증)

> 근거: [[docker-remote-design]](docker-remote-design.md). 실 원격 서버 없이 Rancher 의
> ubuntu 컨테이너를 가상 서버로 쓴다. 인벤토리는 실주소를 유지하고, 하네스가 접속하는
> 최하위 4곳(dialSSH + RPC 조립 3곳)에서만 `AddrMap` 이 loopback 퍼블리시 포트로 치환.
> 선례: `~/Work/github/packages/wemix-bp-test` 의 LocalMap (동작 검증 완료).

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **R1** | `AddrMap` seam — **`--docker` 옵션이 전원**(파일 존재는 활성화 아님, §3.2a) + 매핑 파일(gitignore) 로드 + 접속 경계 주입 + 적용 보고 | ☑ **완료 2026-08-24.** `remote.AddrMap` 을 `target.resolveOver`(SSH 두 형태의 단일 수렴점)와 netcompose 의 `resolveTarget`/헬스 프로브에 주입. 옵션 없으면 파일 있어도 항등(라이브: 실주소 다이얼 후 timeout 실증) · 옵션+파일 부재는 오류(패키지·CLI 회귀) · `net` 은 `State.Docker` 영속 · 적용 내역 보고("docker: dialing 172.30.0.11:22 as 127.0.0.1:2201") · CLI/MCP 동일 옵션 | ☑ |
| R2 | docker 가상 서버 생성 스크립트 — compose + 인벤토리 v2 + localmap 자동 생성 | ☑ **완료 2026-08-24** (`env/docker/gen-env.sh`). 15대 기동, 생성된 인벤토리를 `net pool --fleet` 이 15×1=15 로 읽음. 퍼블리시 포트는 127.0.0.1 바인딩. server15 는 pn 예정(역할은 할당이 정하므로 서버 계층 구분 없음). **같은 날 접근 모델 교체(DR-b 해소)**: 실서버가 id+password 이고 sudo 가 그 비밀번호를 요구한다는 운영 사실에 맞춰, 키 로그인을 없애고 사용자 `chainbench`+비밀번호(+비밀번호 요구 sudo)로 재구성. `remote.ExecWithInput` + `driver.SSHSudoRunner`(sudo -S -k, 비밀번호는 stdin) 신설 — NM-e 의 "운반만 되던 sudo" 가 소비 가능해짐. 라이브: password 로그인·sudo whoami=root·root 전용 쓰기·keyring 원격 2종 전부 통과 | ☑ |
| R3 | keyring 원격 경로 라이브 — `import --from srv://` | ☑ **완료 2026-08-24.** server1 의 nodekey 를 `srv://server1/...` + `--docker` 로 가져와 주소·공개키 파생, 번역 보고 출력. 서버로의 쓰기 방향은 R4 의 provision 이 담당(keyring 에 서버 쓰기 verb 없음 — 의도). **후속(같은 날): 검증을 상설 테스트로** — 게이트된 라이브 스위트(`Live_Keyring*`, `CHAINBENCH_DOCKER_FLEET=<build>` + 함대만 있으면 한 명령) 가 raw key·암호화 keystore(+password) 원격 가져오기와 주소 왕복 동일성을 검증. 이 스위트가 **실결함 1건을 즉시 잡음**: `user@host:path` 형태는 포트 미지정(0)이 매핑 뒤에 22 로 기본화되어 변환표를 지나침 → 매핑 전 기본화(`mapCredentials`, 함대 불필요 단위 회귀 동반). **니모닉 가져오기를 CLI/MCP 에 노출**(core 에만 있던 갭): 골든 벡터(dev mnemonic → 0xf39F…) + 출처 배타성 테스트. 실측: `Ring.Install` 은 소비자 0 (배송은 provision 소관 — 정리 후보) | ☑ |
| R4 | 원격 조립·기동 라이브 (단일 서버) | ☑ **완료 2026-08-24.** `net up --server server1 --docker` 로 stablenet 4검증자가 **docker 서버 위에서** 15스텝 완주, 블록 16→24 전진(매핑 포트로 프로브), stop 후 고아 0. **P2 실증**: genesis·static-nodes·workspace 전부 실주소(172.30.0.11), loopback 0건(metrics 자기 바인드 기본값 제외). **원격 경로의 선재 결함 4건을 이 과정에서 발견·수정**: ① init 이 타깃 경로를 로컬 `os.ReadFile` 로 읽음 → Files seam 경유 ② netcompose 에 신원 배송 부재 + config/argv 가 로컬 키 경로를 타깃에 구움 → `keysBase()` + provision 의 `shipIdentities`(engine 방식 이식) ③ **원격 launch 셸 문법** — `mkdir && nohup CMD &` 는 리스트 전체가 백그라운드 서브셸이 되어 세션 파이프를 문 채 노드를 기다림(노드가 즉사할 때만 우연히 통과) → `|| exit 1; nohup … &` + 문법 회귀 테스트 ④ 헬스 프로브가 fleet 에서 노드별 주소 대신 target 주소를 물음 → 노드 기록 주소로 | ☑ |
| R5 | **fleet 다중 호스트 기동** — netcompose 의 target 이 워크스페이스당 1개라 `fleetTarget` 이 `Hosts[0]` 만 취함(실측). 노드별 target 해석(provision·init·start·stop 이 각 노드의 호스트로) | 15대 fleet 에서 `net up --fleet` 완주 · N4(Materialize)와 설계 겹침 — 함께 판단 | 미착수 |

### 확정된 결정 (2026-08-22)

| # | 결정 | 근거 |
|---|---|---|
| D1 | **노드는 이름 둘을 갖는다** — 신원 `node7`(=`Index`, 저장·경로·keyring), 별칭 `en2`(역할 내 서수, 정의서 주소지정). **개명 없음** | 영속 식별자는 이미 라벨이 아니라 `Index int` 라 표기·주소지정 문제였다. spec 은 여러 토폴로지에서 도는데 `node7` 은 넷마다 다른 노드를 가리킨다 ([[netmap-design]] §2.5a) |
| D2 | **`--peering proxied` 는 필수** | 메인넷이 bp–pn–en 이고 **트랜잭션이 en 을 거쳐 전파**된다. mesh 만으로는 실제 전파 경로를 태우지 못한다 |
| D3 | **인벤토리 v2 단독** (v1 호환 없음) | 실제 파일은 gitignore 라 배포본이 확인되지 않는다 — 지금이 전환 비용 최저. 호환 로더는 다섯 번째 폴딩표가 된다 |

**NM5 완료 (2026-08-22) — netmap 본 트랙 종료.** 잔여는 NM6(철자 방출 전환, 2026-08-24 에
NM3 에서 분리 — N0 행과 [[netmap-design]] §4) 하나다. 라벨이 파생값에서 **데이터**가 됐다:
워크스페이스가 `label` 을 저장하고(구 워크스페이스는 index 폴백), `netmap.Layout` 이
datadir·config·log 경로를 그 라벨에서 파생한다 — `fmt.Sprintf("node%d")` 로 흩어져 있던 6곳이
한 함수가 됐다. `Request.Label` 로 운영자가 지은 이름이 보존되고, `net map --addr 127.0.0.1:31021`
로 **로그 한 줄의 주소를 노드로 되짚는다**(파생 etcd 포트 포함).

**`place` 의 할당기를 삭제했다** — `Allocator`·`NodePlacement`·`Mode`·`Capacity` 모두 소비자 0.
`NodeReq`(역할·sync·binary)만 남고, 그것은 배치가 아니라 기동 입력이다. N7 이 겨눴던
"`NodePlacement.Name` 이 버려진다"는 **필드 자체를 지워서** 닫혔다 — 5곳에서 4가지 철자로
만들어지고 아무도 읽지 않던 값이었다. **N7 완료.**

**NM4 완료 (2026-08-22)** — 조회 표면. `net map` 은 네 방향으로 답한다(노드 번호·라벨(신원 또는
별칭)·호스트·포트) — **포트로 노드를 되찾는 것**이 원래 동기였고, 이제 파생 etcd 포트로도 된다.
`net pool` 은 "왜 15개가 거부됐는가"를 명령 하나로 답한다(호스트×슬롯=용량, 사용/여유, 출처).
유스케이스는 `app` 에 하나씩, CLI·MCP 는 바인딩만(K8 선례). **자격증명은 어느 쪽에도 없고**,
`NetPoolOut` 에 그런 필드가 없다는 것을 리플렉션 테스트로 고정했다 — 유출은 조용하기 때문이다.

**NM4 를 하다 같은 결함을 한 번 더 잡았다**: `net map` 이 etcd 를 `-` 로 찍길래 보니
워크스페이스→netmap 변환이 또 그 필드를 빠뜨리고 있었다(손복사 4곳 중 하나). 필드별 복사를
없애고 `NodeState` 가 `node.Endpoints` 를 **임베드**하도록 바꿨다 — JSON 키는 그대로 인라인되어
영속 형식이 변하지 않으면서, 복사가 사라지니 빠뜨릴 필드도 없다.

**NM3 부분 완료 (2026-08-22)** — static-nodes 조립이 세 벌에서 한 곳(`netmap.Peering`)으로
모였다. engine·netcompose 가 이를 경유하고 `--peering` 이 CLI·MCP 에 노출된다. 착수 전
**전수 조사**로 역할 철자를 하나만 비교하던 9곳을 `netmap.Is` 로 접었다(NM3 part 1) — 그 목록에
genesis 검증자 수·`--unlock` 무장·BFT 최소치가 들어 있어, 정규 철자를 방출하는 순간 전부
오동작했을 자리다.

**라이브가 설계를 정정했다**: 초안의 proxied(= bp 가 pn 만 다이얼)로 5노드를 띄우자 **블록이
0에서 멈췄다.** 모든 bp 가 `ROUND-CHANGE` 를 자기 것만 세며 반복했고(`currentRoundChanges.count=1`),
pn 로그의 WBFT 라인은 2줄뿐이었다 — **pn 은 검증자가 아니라 합의 트래픽을 중계하지 않는다.**
확정형은 **bp↔bp 직결 + bp↔pn + pn↔en**(en 은 여전히 bp 를 모른다). 정정 후 재검증:
stablenet mesh 4노드 api 9/9 · wbft mesh 4노드 블록 54 · stablenet proxied 5노드 블록 전진 +
피어 4 + api 9/9, 세 경우 모두 고아 0.
**할당 경로도 전환됐다**: engine·netcompose·chainsetup 이 `netmap.Assign` 을 쓰고
`place.Allocator` 의 프로덕션 호출은 **0**이다. 전환하며 실측한 두 가지: `LocalOSAssigned` 는
소비자가 없었고(격자로 흡수되지 않는 별개 전략), `MinValidators` 는 전 호출지에서 1이었다 —
"프로듀서 최소 하나"는 `Assign` 이 직접 거부한다. 라이브 재확인: 포트 동일(8600·31000 대역),
api 9/9, 고아 0.
**포트 표현도 하나가 됐다** — 3벌 → 1벌. 설계는 `node.Endpoints` 를 `netmap.Ports` 로 *대체*
한다고 했으나 그 방향은 **상향 의존**이다(`node` 는 L0, `netmap` 은 L1). 어휘를 아래에 두는 것이
유일한 무순환 해법이라, `node.Endpoints` 가 `Etcd` 를 갖고 `portplan.Ports`·`netmap.Ports` 가
그 별칭이 됐다. 결과: **etcd 포트가 런타임·워크스페이스까지 살아남는다**(`"etcd": 31001` 실측).
그 포트는 `p2pStep>=2` 규칙이 존재하는 이유인데, 규칙이 지키는 값을 정작 아무도 되읽을 수
없던 상태였다. **NM3 완료** — 단, 원래 범위에 있던 철자 방출 전환은 여기서 하지 않았고,
**NM6 으로 분리**해 남겼다(2026-08-24 검토에서 확정. N0 행이 추적한다).

**NM2 완료 (2026-08-22)** — 피어링이 역할에서 파생된다. `mesh` 는 현행과 바이트 동일(골든:
`armSpecs` 가 렌더한 config 의 enode 목록 == `netmap.Mesh`, **self 항목 포함까지**; self 를 뺀
변형으로 실패를 확인), `proxied` 는 bp↔pn↔en 이라 **en 의 목록에 bp 가 없다**. pn 없는 proxied 와
poa+pn 은 거부한다 — `ConsensusFamily.SupportsRole` 신설(구현 2곳뿐이라 값싼 seam).
**동시에 고친 잠복 결함**: 두 패밀리의 `StartFlags` 가 `--mine` 을 `RoleValidator` 철자에만
걸고 있었다. NM3 이 정규 철자를 방출하면 **프로듀서가 --mine 없이 떠서 체인이 멈춘다** —
NM1c 가 셀렉터에서 찾은 것과 같은 부류이며, 이번엔 블록 생성 자체를 좌우한다. 이제 두 철자
모두 `netmap.NormalizeRole` 로 접는다. N0b 가 닫혔다.

**NM1b 완료 (2026-08-22)** — `netmap.Pool`/`Assign` 이 자원 격자(hosts × slots)를 결정적으로
할당하고, 인벤토리가 v2(pool) 단독이 됐다. `place` 의 두 결정적 모드는 이 격자의 특수해임을
**등가 테스트로 고정**했다(라벨·호스트·전 포트 비교) — 없으면 NM3 이 리팩터를 자처하며 모든
넷의 포트를 옮긴다. v2 는 호스트별 개별 설정을 잃었다(풀은 균질); 근거가 생기면 `hosts[]`
오버라이드로 얹는다. `--peering`·`net map`·`net pool` 은 NM2·NM4.

**NM1c 완료 (2026-08-22)** — D1 의 코드 반영이자 NM3 의 선행 조건. 셀렉터의 역할 폴딩표가
`netmap.NormalizeRole` 을 경유하고, 신원 라벨(`node7`)도 셀렉터로 해석된다.
**고친 결함**: `session.rolesForToken` 이 `"bp"` 를 `RoleValidator` 에만 매핑해, 정규 역할
`bp` 를 가진 노드가 자기 셀렉터에 매칭되지 않았다. 두 철자가 섞인 넷에서는 **`bp1` 이 두 번째
노드로 조용히 해석**된다(옛 표로 실측). NM3 이 정규 철자를 방출하기 시작하면 터질 자리였다.

### 결정이 필요한 열린 질문

[[family-bringup-design]] §9 참조. 요약: (1) `Phase.Actions` 를 문자열로 둘지 타입 상수로 둘지,
(2) 부트노드 선정 기준(현 `poa.BootRole` 은 4검증자면 전부 참이라 기준이 못 된다 → 토폴로지
`bootnode: true` 권장), (3) wemix `config.json` 의 `env` 정책값을 어디서 받을지.

---

## 2. 전체 작업 리스트 (Phase · Task)

### Phase 0 — 레이아웃 정리 + 인터페이스 동결
- ☑ **T0.0 `pkg/` → `internal/` 마이그레이션** 기계적 rename: 디렉토리 이동 + import 경로 `github.com/0xmhha/chainbench/pkg/…` → `…/internal/…`(cmd/ 포함 전 소스). **게이트**: `go build ./...` + `go test ./...` 통과, 동작 변화 0. (외부 importer 없음 → 파괴 없음.) 공개 API가 필요하면 그 slice만 `pkg/`에 남김.
- ☑ **T0.1** Transport·Allocator(+Capacity)·KeyRegistry(+BLSDeriver)·GenesisBuilder·Provisioner·Supervisor·Collector·Session·Interpreter·Capabilities를 **컴파일되는 Go stub**으로(design §3과 1:1, `internal/` 하위). enum은 typed const(FailureMode/TestStatus/Mode/Source). **게이트**: 컴파일 + 각 인터페이스↔F(AC) 매핑 표.

### Phase 1 — Low atomic (TDD 먼저, 병렬 가능)
- ☑ **T1.1 testspec** Parse+필수검증(schemaVersion 등)·**Fingerprint(canonical·키정렬→결정성)**·`Get(dotPath)`·`,`파서. 순수. — F3·F7
- ☑ **T1.2 assert funcs** 타입인지 비교(Equal/Len/EqualHashAt/EqualCI/InDelta …). 순수. — F6
- ☑ **T1.3 session-path** 결정적 경로·env-id(`env-`+12hex)·레이아웃. — F1
- ☑ **T1.4 place** portplan(로컬 스텝/OS)+원격 동일포트 **통합 Allocator** + **용량검증(min≥4·max=서버×포트)**. — F12
- ☑ **T1.5 procman EXTEND** `{PID,datadir}`·원격PID·`Alive`. (stop 경로 배선은 T3.2) + **`StopOne`**(단일 노드 정지 — fault 스텝용, `StopAll` 은 네트워크 전체를 내려 쿼럼 테스트에 못 씀) + **loopback=local 판정 수정**(라이브 검증에서 발견: 로컬 런처가 `Host:"127.0.0.1"` 을 기록해 `IsRemote()` 가 참이 되어 시그널 대상에서 제외되고 있었음 — teardown 이 동작한 건 `Teardown` 이 빈 Host 로 **중복 track** 하던 우연 덕분) — F13
- ☑ **T1.6 keyreg** 랜덤/기존/원격다운로드 통합 + **BLSDeriver seam**(외부 bootnode 캡슐화, 부재 시 오류). — F2
- **게이트**: 단위 100% + 동시 모듈 `-race`.

### Phase 2 — Transport
- ◐ **T2.1** Local/Remote Transport를 driver 위에 형식화 + **종료검증(`kill -0`)** + **key_file 인증**. **[D안]** ☑ **key_file 인증**: `remote.Credentials` 에 `PrivateKey`/`Passphrase` 추가, `authMethods`(key 우선+password, 최소 1개 필수, 키자료 미노출), `LoadPrivateKey`(0600 강제·insecure perm 거부). deploy `credentials.go` 가 `key_file`(+ `CHAINBENCH_REMOTE_KEY_FILE`/`_PASSPHRASE` env) 를 `remote.Credentials.PrivateKey` 로 로드 — "future phase 예약" 게이트 제거. 단위검증(authMethods 4케이스·키 미노출·perm 거부·For key_file). ☑ **종료검증(`kill -0`)**: 이미 `procman.Alive`(signal 0)+`StopAll`(SIGTERM→wait→SIGKILL→poll→leak 보고)로 구현, 엔진 teardown 이 이를 경유(라이브 고아0 확인). **남은 것**: driver 위 Transport 타입 형식화(C 원격 슬라이스와 함께).
- ☑ **T2.2 upload-if-absent**(로컬·FileSink; 원격 SSH sink는 remote 슬라이스) `test -f` 존재확인 → 재사용/업로드(현재 항상-업로드 or 항상-읽기).
- **게이트**: 단위(모의) + 통합 1건(로컬 더미 프로세스 검증-종료·고아0).

### Phase 3 — Middle 통합(라이브)
- ☑ **T3.1 Provisioner** datadir+키+genesis+config 물질화(local·remote 동일경로)+upload-if-absent.
- ☑ **T3.2 Supervisor** 오케스트레이션·teardown·procman배선 단위완료 + **실4노드 라이브 헬스게이트(블록전진 gate)로 Phase4 BuildEnv e2e 에서 검증**(고아0). etcd 리더 게이트는 wemix 계열(gwemix/etcd 필요) 후속.
- ◐ **T3.3 Collector** ☑ RPC 스냅샷(height·peers)·WaitLog poll·**로컬 live tail**(offset 증분·스캔→tail·부분줄 미방출·`Deps.OnLine` obs 미러 seam)·**bp참여 집계**(head producer 샘플→`BPParticipation`, `Deps.BPWindow` 로 bounded prune)·**fork/reorg 검출**(높이별 first-seen hash, 노드 간/샘플 간 불일치→`Forked`). 단위+`-race` 검증. ☑ **엔진 배선**(`withCollection` 이 local/attach 의 BuildEnv 를 래핑 — `Bus` 설정 시 env 별 collector 실행, RPC probe(`rpc.Client.HeadBlock`)로 샘플, chainstate 스냅샷·tail 로그를 obs 로 미러, teardown 시 정지; attach+mock RPC 로 chainstate 이벤트 e2e 검증)·**chainstate 세션 영속화**(collection 이 스냅샷을 `chainstate/chainstate.jsonl` 로 기록 — F10/F15 jsonl+obs 미러, 완료 세션 재생용; best-effort). ☑ **원격 SSH tail**: tail 루프가 **`collector.LogReader` seam**(`ReadFrom(ctx,path,offset)`)을 거치도록 리팩터 → 로컬은 `LocalLogReader`(파일), 원격은 `driver.RemoteLogReader`(SSH `tail -c +N`, **1-based 바이트 오프셋** — `tail -n` 은 줄 단위라 collector 가 추적하는 정확한 바이트 위치를 잃어 줄 중복/분할이 남). 파일 부재는 오류 아님(첫 줄 쓰기 전 tail 시작 허용), 전송 실패만 오류. 단위검증 5건. **실 SSH 호스트 라이브 e2e 는 사용자 환경 필요**(T5.1·C2). attach=RPC-only(로컬 로그 없음→tail no-op).
- ☑ **T3.4 Session 저장·재사용** `session.json`·env fingerprint 재사용(엔진 오케스트레이션이 fingerprint 로 env 재사용)·records(spec/steps/assert/status). 엔진 단위·라이브 e2e·attach e2e 로 검증(summary.pass 판정).
- ☑ **T3.2b supervisor 선언 논항 방출** (x-bar 정렬 검토 2026-08-09) 선언만 되어 있던 `Options` 3개를 실제로 읽는다.
  - **`LeaderGate`**: `Deps.LeaderGate(ctx, ns, window)` seam 신설 — **HealthGate 보다 먼저** 실행(클러스터에 리더가 없으면 노드는 healthy 일 수 없다). "리더 준비"의 판정은 체인특화(go-wemix 내장 etcd)라 주입식이고, **언제 돌릴지·얼마나 기다릴지·실패를 어떻게 분류할지는 supervisor 가 소유**. **요청했는데 미배선이면 조용히 통과가 아니라 오류**(F13 AC-1).
  - **`AlignJoinGap`**: `JoinGap(N)`(C-etcd 표: ≤11→7s·≤23→11s·≤41→17s·else 23s)·`JoinWindow(N)=(N+1)*gap` 신설 — 리더 게이트 데드라인을 **클러스터 크기에서 파생**. 고정 타임아웃은 *아직 자기 조인 슬롯이 오지 않은* 노드를 조인 실패로 오판한다(L7).
  - **`ForkSwaps`**: `Deps.SwapBinary` seam 으로 type-2 스왑 수행. **선언했는데 미배선이면 오류** — 조용히 건너뛰면 체인이 잘못된 바이너리로 포크를 넘어 훨씬 덜 명확한 곳에서 실패한다(F9 AC-3).
  - **`FailureMode` 분류**: `Classify(err)` 신설(etcd stale/join·quorum·fork·rpc 시그니처) — **launch 실패도 분류**(이전엔 전부 `RPCUnready`), HealthGate 가 Mode 를 안 채우면 에러 텍스트에서 파생, **Gate 가 스스로 분류했으면 보존**. 매칭 없으면 `UnknownFailure` — 그럴듯한 오분류는 읽는 사람을 엉뚱한 로그로 보내므로 정직한 "모름"이 낫다(F13 AC-2).
  - **enum 0 값 교정**: `EtcdJoinFailed` 가 iota 0 이라 **빈 `Diagnosis` 가 "EtcdJoinFailed" 로 읽히던** 함정 제거 → `UnknownFailure` 를 0 으로.
  - 재시도 시 `RemoveDataDir:true` 로 stale etcd 상태 정리(F13 AC-3)는 기존 동작 유지·주석 명시. 단위검증 12건.
  - **잔여**: 실제 etcd 리더 게이트 구현체(wemix 계열)와 `SwapBinary` 구현체 배선은 **T5.2**(gwemix 라이브 필요).
- **게이트**: 각 컴포넌트 통합테스트 라이브.

### Phase 4 — Walking Skeleton ★
- ☑ **T4.1 Engine** DI 오케스트레이션(Parse→Applicable skip→fingerprint 기반 env 재사용/BuildEnv→RunSpec→session 저장→종료 시 Teardown) 구현·단위검증 완료. 실제 Place→KeyReg→Genesis→Provision→Supervise→Collect 배선의 **1체인 wbft·local·4노드·tx1** live e2e 는 사용자 환경(체인 바이너리 필요)으로 이월.
- ☑ **T4.2 빌트인 tx/rpc + 어휘 확장(#225)** `NewRegistry(true)` 시드 — **액션 11**: `sendTx`(fee-cap/nonce 인자 포함)·`waitBlock`·`read`(임의 RPC-read 소스 저장)·`deployContract`·`registerContract`·`faucet`·`stopNode`·`startNode`·`restartNode`·`partition`·`healPartition`. **어세션 16**: `chainId`·`blockNumber`·`peerCount`·`balanceAt`·`codeAt`·`nonceAt`·`call`·`txStatus`(F11)·`blockAdvance`·`sameBlockHash`·`baseFee`·`estimateGas`·`gasPrice`·`rpcCall`(체인 어휘를 spec 문자열로, core 무지 유지)·`logs`(eth_getLogs select)·`wsSubscribe`. **스텝 값 바인딩**(`save`/`$ref`/`${..}`). mock RPC 단위검증 + 라이브 `TestEngine_Live_NewVocabulary`(실 gstable 4노드). → 컨트랙트 배포·이벤트·fault·WS·교차-call 전부 표현 가능.
- ☑ **T4.2b 노드 생명주기·fault 액션** `stopNode`/`startNode`/`restartNode` 는 **`testspec.NodeControl` seam**(`Deps.Nodes`) 경유 — 로컬 엔진은 `engine.NodeController`(런처 앞단에서 노드별 arming·PID 기억, supervisor 와 **procman 공유**해 중도 재기동 노드도 teardown 에 포함) 를 주입하고, **attach 는 nil** 이라 액션이 "이 실행은 노드 프로세스를 소유하지 않는다"고 명확히 실패한다. `procman.StopOne`(SIGTERM→grace→SIGKILL→검증) 신규 — `StopAll` 은 네트워크 전체를 내려 쿼럼 테스트에 못 씀. `partition`/`healPartition` 은 `admin_removePeer`/`admin_addPeer`(+`admin_nodeInfo` 로 enode 획득) 로 **그룹 경계를 넘는 링크를 양방향 절단**. 예제 `fault-node-restart.json`·`fault-partition-fork.json`(`requires:["process"]` → attach 에서 skip). 단위검증(선택·미배선 오류·8회 removePeer·풀메시 heal·재기동 arming 재사용).
- ☑ **T4.2c 자산·컨트랙트 액션** `faucet`(수신자에 wei 공급 — 런타임 생성 서명키는 genesis alloc 에 없어 첫 tx 가스도 못 냄; `from` 미지정 시 **대상 노드의 unlocked coinbase** 를 자금원으로 사용)·`deployContract`(`to` 없는 tx → 영수증의 `contractAddress` 를 `ac.Value` 로 바인딩 — **주소가 영수증에만 있으므로 `sendTx` 로는 표현 불가**, 이것이 별도 액션인 이유)·`registerContract`(배포 주소로의 등록 호출 — `to` 필수·revert 는 항상 실패인 **의도-명시형 sendTx**). 예제 `contract-deploy-and-register.json`(faucet→deploy→`$contract` 로 register→codeAt/txStatus/balanceAt 검증). 단위검증(coinbase 자금원·value hex·필수인자·주소 없는 영수증 오류·배포엔 `to` 미포함).
- ☑ **T4.3 RunSpec 배선** `engine.NewRunSpec(testspec.Deps)` — 인터프리터를 `Deps.RunSpec` 에 바인딩하는 조립 seam. spec→interpreter→빌트인 어세션→RPC→기록 상태 **실행 수직**을 mock RPC 로 통합검증(pass/fail). BuildEnv 배선(Place→KeyReg→Genesis→Provision→Supervise; 레거시 `pipeline/setup.Plan` 매핑 필요)은 후속.
- ☑ **T4.3b tx 스텝 시맨틱(F11)** sendTx 가 영수증 status 를 검사 — 기본 revert(0x0)=스텝 실패, `expectRevert:true`(또는 `expect:"revert"`)=원자적 negative(성공하면 스텝 실패). 스텝 provenance(tx hash·receipt)를 `ActionCtx` 출력→`StepResult.Hash/Receipt` 로 기록(미사용 필드 활용). mock RPC 단위검증(revert 실패·expectRevert 양방향·provenance 기록·미지 스텝 기록후 실패). → wemix4 negative(GOV 권한거부 등) 이관 기반.
- ☑ **T4.4 BuildEnv 배선 [A안]** `engine.AssemblePlan(plugin, []PlacedNode, genesis, dataRoot, caps)` — place 할당 포트로 `setup.Plan` 을 조립하는 순수함수(레거시 `setup.BuildPlan` 의 `node.Offset` 고정포트 경로 대체). LocalOSAssigned·remote 모드가 이 함수 변경 없이 합성됨. 실 wbft 플러그인으로 단위검증(할당포트 반영·바이너리 폴백·datadir 기본값). **후속 슬라이스**: (T4.4b) keyreg→genesis 피딩(검증자 주소·ExtraData RLP), (T4.4c) provision+supervisor.BringUp 오케스트레이션(launch seam 주입).
- ☑ **T4.4b GenesisSource seam** `engine.GenesisSource`(plugin·검증자수→genesis 바이트) + `PresetGenesisSource`(preset `metadata.json` 기반). **핵심 발견**: wbft 계열 ExtraData(RLP 검증자셋)는 코드가 계산하지 않고 preset 에 baked — 패밀리는 템플릿 치환만 함. 따라서 랜덤 keyreg 키만으로는 유효 genesis 불가 → preset 이 검증자셋·ExtraData 소스, keyreg 는 노드 신원/계정 키. 실 wbft 패밀리+최소 템플릿으로 단위검증(치환·Take 검증자 제한·preset 부재 에러). BuildEnv 조립(T4.4c)이 이 seam 을 호출.
- ☑ **T4.4c BuildEnv 오케스트레이션** `engine.NewBuildEnv(BuildDeps)` — allocate(place)→genesis(GenesisSource)→AssemblePlan→provision(주입 seam)→`supervisor.BringUp`(launch/health seam 주입)→NodeSet+Teardown 반환. `Deps.BuildEnv` production wiring 완성. fake allocator/genesis/provision + 실 supervisor(fake launch/health)로 파이프라인 스레딩 단위검증(4노드 조립·genesis 검증자수·plan 포트·teardown·에러전파). **남은 것**: (1) 실 `Provision` 파일생성(config.toml·keystore·nodekey; 레거시 `setup.provision` 참조), (2) 실 4노드 라이브 e2e(바이너리 필요, 사용자 환경). → **코드 수준 walking skeleton 조립 완료**(BuildEnv+RunSpec 모두 `Deps` 배선).
- ☑ **T4.4d GenesisSource 라이브검증** `TestPresetGenesisSource_Live_GstableInit`(GSTABLE_BIN 게이트) — `PresetGenesisSource`+실 stablenet 플러그인으로 `keys/preset` 에서 genesis 생성 → **실 gstable v1.1.0 `init` 통과**("Successfully wrote genesis state", `<datadir>/gstable/chaindata` 생성). 이월했던 "랜덤키만으로 유효 genesis 불가·preset 필요" 가정을 실 바이너리로 확정. CI 는 바이너리 부재로 SKIP(green 유지). **실 4노드 라이브 기동(Provision+driver Launch+HealthGate)은 T4.4e 후속**.
- ☑ **T4.4e RunSpec 라이브 e2e (walking skeleton 증명)** `TestRunSpec_Live_Stablenet`(GSTABLE_BIN 게이트) — 실 gstable 로 4노드 stablenet 기동(`setup.Launch` fixture) → `engine.NewRunSpec`(인터프리터+빌트인) 으로 spec 실행: **sendTx(노드서명+영수증)·chainId==8283·blockNumber≥1 어세션 전부 실 RPC 대상 → status pass** → teardown 고아0. 로컬 39.7s 통과. **발견**: (1) geth IPC 유닉스소켓 경로 <104자 제약 → dataRoot 는 `/tmp/cblXXX`(t.TempDir 불가), (2) wbft 블록생성 웜업 ~35s → 헬스게이트 넉넉히 폴링. → **DSL 이 실 체인에서 실행·검증됨(실행 수직 라이브 완성)**. 남은 것: BuildEnv 자체의 실 launcher(place 포트 기동)는 T4.4f 선택.
- ☑ **T4.4f BuildEnv 실 launcher (기동 수직 라이브)** `engine.LocalLauncher`(preset 기반 arming: config 렌더·신원설치·datadir init·기동) + `armSpecs`(순수, 단위검증: validator --unlock/--nodekey, static-node enode 가 plan p2p 포트 사용, endpoint 는 unlock 없음). `TestBuildEnv_Live_Stablenet`(GSTABLE_BIN 게이트): 실 allocator+PresetGenesisSource+LocalLauncher+블록전진 헬스게이트로 `NewBuildEnv` → **실 4노드 stablenet 을 allocator 할당 포트(node1 :8600)로 기동·헬스통과·teardown 고아0**. 로컬 통과. → **Engine 기동 수직(BuildEnv)도 실 체인 라이브 증명. RunSpec(#195)+BuildEnv 로 Engine.Run 전 구간 라이브 커버.** (짧은 session root 로 IPC 소켓 경로 <104자 유지.)
- ☑ **T4.5 Engine 최상위 배선 (capstone)** `engine.NewLocalEngine(LocalConfig)` — allocator·PresetGenesisSource·LocalLauncher·`NewBlockAdvanceGate`·인터프리터·session 을 하나의 `engine.Deps` 로 조립하는 실행 가능한 진입점(CLI/MCP 가 호출). `NewBlockAdvanceGate`(head≥target 폴링, 로컬 non-etcd 라이브니스), `applicableTo`/`validatorReqs` 헬퍼. 단위검증(applicable 매칭·구성검증·미지체인 에러). **`TestEngine_Live_FullRun`(GSTABLE_BIN 게이트) capstone**: `NewLocalEngine`→`Engine.Run([spec])` 한 번으로 **실 gstable 4노드 기동→spec 실행(chainId/blockNumber)→teardown→session.json 저장, summary.pass=1** 검증. 로컬 4.5s 통과·고아0. → **walking skeleton 을 실행 가능한 단일 진입점으로 종료(Engine.Run 전체가 실 체인에서 동작).**
- ☑ **T4.6 launcher 파일 물질화를 provision.Provisioner 경유 [B안]** `LocalLauncher` 가 genesis·per-node config 를 `provision.Provisioner`(`FileSink`) 로 물질화 — 기존 ad-hoc `os.WriteFile(genesis)`+`driver.Provision(config)` 제거. **upload-if-absent**(기존 파일 재사용) + **원격 `RemoteFileSink` 로 교체할 seam**(슬라이스 C 대비) 확보. 순수 `materialize` 헬퍼로 분리해 recording FileSink 로 단위검증(genesis+config 기록·재사용 스킵). 리팩터 후 실 gstable 라이브 2종(BuildEnv/FullRun) 재통과·고아0. 프로덕션 engine 은 이제 레거시 `setup.Provision/Launch` 미호출(`setup.Plan` 타입만 사용).

### Phase 5 — 수직 슬라이스 (매번 통합 유지)
- ◐ **T5.1 remote [C안]** ☑ `driver.RemoteFileSink`(`provision.FileSink` 구현: SSH `test -f` 존재확인 + `ProvisionFile` base64 전송) — B의 `LocalLauncher.Sink` seam 에 그대로 주입 가능(upload-if-absent). ☑ launcher init 을 `driver.Initializer` capability 경유로 라우팅(local/remote 드라이버 공통) → `LocalLauncher{Driver:RemoteDriver, Sink:RemoteFileSink}` 로 **원격 기동 가능**. 단위검증: RemoteFileSink(exist/absent/transport err·base64 write·`provision.FileSink` 만족), launcher 전체 합성(materialize→init(Initializer)→launch, fake driver/sink). 로컬 live 2종 재통과(init 라우팅 변경 후·고아0). **남은 것**: 실 원격 SSH 호스트 대상 라이브 e2e(사용자 환경·SSH 필요). · ☐ **T5.2 업그레이드 멀티바이너리**(wemix+wbft) · ☑ **T5.3 attach** `engine.NewAttachEngine(AttachConfig{Chain,RPCURLs,ArtifactRoot})` + `NewAttachBuildEnv`(attach.Build 로 RPC 엔드포인트에서 NodeSet 구성, **기동/teardown 없음** — attach 는 노드를 만들지 않음). 바이너리·preset 불필요 → **mock RPC 로 Engine.Run 전체 e2e 가 CI 에서 실행**(chainId/blockNumber 어세션 pass·미적용 spec skip·구성검증). walking skeleton 실행수직을 바이너리 없이 CI 커버하는 첫 통합. · ☑ **T5.4 stablenet**(ACL 플러그인·Core 무변경) — 거버넌스 read 시나리오를 DSL 로 표현·엔진 실행 CI 검증(예제 spec + govbind calldata mock RPC e2e). · ☐ **T5.5 wemix4 이관**(DSL).

### Phase 6 — 표면·마감
- ☑ **T6.7 spec 오프라인 검증 + 예제** `chainbench validate [spec…]` — 실행 없이 (1) 파싱 검증(OK/INVALID), (2) **이름 해결**(`testspec.Unresolved`: 스텝 액션·어세션 이름을 빌트인 registry 와 대조 → 미등록 시 `UNRESOLVED: action:…/assert:…`, 오타를 런타임 전 포착), 무효/미해결 시 exit 1. `--chain` 은 manifest capability·applicableChains 대조로 실행/스킵(OK·SKIP(chain not applicable)·SKIP(needs caps))을 정보 표시(파싱/해결 오류만 실패). `examples/specs/*.json`(RPC-read·tx/waitBlock 스텝·expectRevert negative·call/txStatus) + CI 가드 테스트(`validate --chain stablenet` 전부 OK)로 DSL 문서-파서 드리프트 방지. → 이관 작성자 빠른 피드백.
- ◐ **T6.6 F16 세션 운영·보안 규약** ☑ **schemaVersion 거부(O2)**: `testspec.validate` 가 미지원 버전 명시 거부(현재 지원 `"1"`). ☑ **CI exit code(O5)**: `chainbench run` 이 세션 판정을 exit code 로 매핑 — 전건 pass=0·fail=1·blocked/인프라=2(`exitError`+`main` 배선). ☑ **세션 GC(O4)**: `chainbench clean --artifact-root --older-than <Nd|Nw|dur>/--keep-last N` — 완료 세션(session.json 보유)만 대상 → 실행중 세션 보존, `session.SessionDir`/`List` 재사용. ☑ **키파일 0600(O3)**: keyreg 가 이미 0600/0700 생성(기존). 단위검증(버전 거부·exit 0/1/2·older-than/keep-last/정책필수·실행중 보존).
- ☑ **T6.5 문서 경로 드리프트 정리** T0.0(`pkg/`→`internal/`) 완료 후에도 문서가 삭제된 `pkg/` 경로를 참조 → 후속 작업 착수 시 혼선. 실제 대상은 모두 `internal/` 하위에 존재(경로만 stale). **최초 스코프는 2문서 6건이었으나 x-bar 정렬 검토(2026-08-09)에서 실제 표면이 7문서 20건임이 드러나 확대**: `repro-migration-remaining.md`(3) · `wemix4-migration-plan.md`(5) · `wemix4-port-tracker.md`(4: `pkg/core/pipeline/testrun`·`pkg/core/procman`×2·`pkg/core/topology`) · **`topology.md`(1: `pkg/core/topology` — 사용자 대상 참조 문서라 최우선)** · `chainbench-requirements-review.md`(2: `pkg/mcp`·`pkg/consensus/poa`) · `chainbench-refactoring.md`(3: `pkg/mcp`·`pkg/dashboard`·`pkg/testkit`) · `chainbench-component-architecture.md`(2: `pkg/core/place`) 전부 `internal/…` 로 정정. `worklist`/`audit` 의 `pkg/` 언급은 마이그레이션 자체를 서술하므로 유지. **경로가 아니라 판정이 낡은 건은 T6.5b**(§2b 실측 매트릭스).
- ☑ **T6.5b component-architecture §2b 실측 매트릭스 갱신** §2b 판정이 낡았던 것(`place` 없음·`procman` 미배선·용량검증·upload-if-absent·로컬 tail·bp참여/reorg 없음)을 **현재(#225) 열로 전부 해소 표기**(#225). 후속 정합(이 검토): §3 C-테이블에 남아있던 stale 델타(C1 "어휘 △"·C3 "게이트 △"·C4 "원격 tail ✗")를 #225 반영으로 갱신 + §2b 컬럼 헤더 `#224`→`#225`.
- ☑ **T6.1 Capabilities** spec `requires: [cap...]` 필드 + 엔진 capability 게이팅: `satisfies`/`applicableWithCaps`(체인 매칭 ∧ 필요 capability ⊆ 타깃 제공). NewLocalEngine 은 `localCapabilities`(manifest.Capabilities + "ws"), NewAttachEngine 은 `["rpc"]` 를 제공집합으로. 미충족 spec 은 skip(fail 아님). 단위(satisfies 4케이스·applicableWithCaps) + attach mock RPC e2e(ws 요구 spec → rpc-only attach 에서 skip). `requires` 는 env 를 바꾸지 않으므로 fingerprint 미포함. · ☑ **T6.2 MCP 결과연동**(F14) `chainbench_run` MCP 도구 — attach 모드로 DSL spec 실행 후 session 판정 반환(CLI `run` 의 MCP 대응). `engine.ReadSessionSummary`(session.json 단일 리더)를 CLI·MCP 가 공유(중복 제거). attach+mock RPC 로 CI 테스트(pass=1·인자 검증). · ☑ **T6.3 dashboard**(F15) — 엔진 오케스트레이션이 obs 이벤트 emit(`Deps.Emit`/`Network`, nil-safe no-op): run started·building environment·environment reused·running spec·spec `<status>`·run complete. `NewLocalEngine`/`NewAttachEngine` 가 `Bus` 옵션으로 배선, `chainbench run --dashboard <url>` 가 `dashboard.Forward` 로 chainbench-dashboard `/api/events` 에 스트리밍(종료 시 flush). 엔진 emit 단위 + attach mock RPC → Forward → dashboard.Server end-to-end CI 테스트. obs.Bus 백프레셔(bounded buffer·drop-on-full)로 관측이 실행을 막지 않음. ☑ **완료 세션 디스크 조회**(F15 AC3): `session.List`/`SessionFilePath`/`ChainstatePaths`(레이아웃 소유자) + dashboard `/api/sessions`·`/api/sessions/{id}`(session.json 판정)·`/api/sessions/{id}/chainstate`(chainstate.jsonl → JSON 배열), id 는 단일 세그먼트 검증(traversal 방지), `chainbench-dashboard -artifact-root` 로 활성화(httptest 검증). · ◐ **T6.4** 백프레셔(O7)·`-race` 게이트(O6)·CLI 정리. ☑ **`-race` 게이트**: CI test 잡을 `go test -race ./...` 로(경합 CI 상시 검출; 풀런 ~71s 로 기존과 동등 — 느린 `verify` 테스트가 CPU 아닌 대기 바운드). ☑ **백프레셔(O7)** obs.Bus 는 bounded buffer(256)·drop-on-full(non-blocking select)·`Dropped` 카운터로 이미 구현 — 회귀 테스트 추가(느린 구독자에서 blocking 없이 drop·정확한 카운트 검증). · ☑ **CLI 엔진 배선**: `chainbench run [spec.json…]` 명령 — `--rpc`(attach: `NewAttachEngine`) 또는 `--binary`(local: `NewLocalEngine`) 로 DSL spec 을 엔진에 실행, `session.json` 요약 출력·실패 시 non-zero exit. attach+mock RPC 로 CI 테스트(pass=1·실패 non-zero·모드 검증). → 재설계 엔진이 CLI 에서 도달 가능. ☑ **기계 판독 출력**: `run --json`(세션 판정: session 경로+tests+pass/fail/blocked/skip)·`validate --json`(per-spec spec/id/ok/result 배열) — exit code(F16-O5)와 함께 무인 CI 게이팅 완성. 단위검증(JSON 파싱·필드).

---

## 3. 폴더 트리 예상도 (구현 후 · `internal/` 레이아웃)

> **레이아웃 결정**: chainbench는 애플리케이션이고 `pkg/`가 자기 `cmd/`에만 쓰임(외부 importer 0) → **`internal/`이 정석**(컴파일러 강제 캡슐화). `pkg/`는 관례일 뿐이라 폐기. 진짜 공개 Go API가 생기면 그 slice만 `pkg/`로 승격.
> `[NEW]` 신규 · `[EXT]` 확장 · `[KEEP]` 유지·재사용 · `[REF]` 재구성 · `[REPL]` 교체 · `[→x]` x로 흡수/이동. C#=DDD 컨텍스트(§1b).

```
cmd/                          # 바이너리 진입점(유일하게 internal/ 밖에서 internal/ import)
  chainbench/          [EXT]   · engine 호출
  chainbench-mcp/      [KEEP]
  chainbench-dashboard/         [KEEP]
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
- **작업 순서·상태의 단일 출처는 이 문서다.** 설계 문서([[layers]]·[[module-responsibilities]]·[[family-bringup-design]]·[[dsl-v2-proposal]] 등)는 *무엇을 왜* 만 담고, 순서표를 복제하지 않는다 — 복제하면 갈라진다(2026-08-18 실측: 착수 순서가 3문서에 중복돼 있었다).
- **TDD**: Low는 RED→GREEN 단위테스트 먼저. 동시성은 `-race`. 통합은 라이브(실 노드).
- **Go 품질**: [[go-code-quality-guidelines]](const화·DI·typed enum·ctx 전파·docs.go 등) 준수.
- **PR 단위**: Task 1개 ≈ PR 1개. 커밋은 [[commit-message-rules]]·[[pr-body-no-emoji-no-claude]]·[[git-add-explicit]].
- **하위호환 병존**: 신규 경로를 별도로 세우고 기존 `setup/upgrade/test`는 유지 → 회귀 없이 점진 이관.
- **게이트**: 각 Task는 비-e2e 통과 + 해당 F의 AC(대표 1건 라이브) 충족 후 다음.
