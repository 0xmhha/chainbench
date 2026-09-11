# chainbench 구현 작업 트래커 — 작업 리스트 · 우선순위 · 폴더 트리 예상도

> **[정본]** **작업 순서·상태의 단일 출처.**
> 이 문서는 *무엇을 만들어야 하는가* 를 정한다. 설계 제안이 여기와 어긋나면 **제안을 고친다.**

> 근거: `chainbench-component-architecture.md`(§2b 실측·§3 컴포넌트·§5 Phase·§1b DDD) · `chainbench-design.md`(§3 인터페이스) ·
> `chainbench-feature-spec.md`(F1~F16 AC) · `chainbench-refactoring.md`(WP1~6).
> 원칙: **Low는 TDD 먼저 → walking skeleton으로 조기 통합 → 수직 슬라이스 확장**(big-bang 금지). 코드는 [[go-code-quality-guidelines]] 준수.
> 상태 표기: ☐ 미착수 · ◐ 진행 · ☑ 완료.

> **이어서 할 일은 §1n "다음 사람에게" 부터 읽는다** (2026-09-08). 남은 것이 무엇이고 무엇이
> 필요한지, 그리고 라이브를 쫓을 때 이번에 비싸게 배운 측정 함정 여섯이 거기 있다.

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

**2026-08-09 x-bar 정렬 검토 후속 (이 브랜치):** 문서 정합(T6.5·T6.5b·audit 스냅샷 강등) → **DSL 보충어 어휘 확장**(스텝 값 바인딩 `save`/`$ref`+`read` · fault 액션 5종 · 자산/컨트랙트 액션 3종 · `logs` 이벤트 어세션 · tx fee-cap/nonce 인자 · `gasPrice`/범용 `rpcCall`/`wsSubscribe`) → **supervisor 선언 논항 방출**(T3.2b) → **원격 SSH tail**(`LogReader` boundary). 액션 2개·어세션 12개 → **액션 11개·어세션 16개**. 라이브 증명: `TestEngine_Live_NewVocabulary`(실 gstable 4노드). 부수적으로 **라이브에서만 드러난 결함 2건 수정** — `procman` 이 loopback 호스트를 원격으로 오판(단일 노드 정지 불가), 실패 스텝의 사유가 아티팩트에 기록되지 않음.

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
- ◐ **레거시 경로 정리** — 착수: 죽은 심볼 제거(`testkit.RunCase`)·레거시 패키지 signpost(testkit·testrun→engine+testspec)·**은퇴 계획 문서**([[legacy-retirement-plan]] = `docs/dev/archive/legacy-retirement-plan.md`: 매핑·순서·블로커·DSL 표현력 갭). **suite 이관 착수**: `tests/` Go-func 케이스는 repo 에 있어 DSL 포팅 가능(라이브만 바이너리 필요) — `onEach` 다중노드 어세션 수정 + 신규 빌트인(`blockAdvance` 헤드 전진·`sameBlockHash` 노드간 no-fork, `rpc.BlockByNumber` 기반) 추가로 **`tests/network` 3케이스 전부 DSL 포팅 완료**(`network-peers.json`·`network-health.json`). **`tests/anzeon` 착수**: read-shape 계열(adapter-code `codeAt NotEqual`, balanceOf/isAuthorized `call Regexp`)→`stablenet-system-contracts.json`, base fee 경계(`baseFee` 신규 빌트인, `rpc.BlockByNumber` baseFeePerGas)→`stablenet-gas-policy.json`, 하드포크 아티팩트(P-256 precompile `call`·GovMinter `codeAt`·chainId/blockNumber)→`stablenet-hardfork.json`, 가스 추정(`estimateGas`=`rpc.EstimateGas`)→`stablenet-estimate-gas.json`, token-metadata(`call`+`Contains` 심볼)→`stablenet-token-metadata.json` 포팅. **read-기반 anzeon 이관 완료**(시스템컨트랙트·getter·base fee·하드포크·estimate-gas·token-metadata). **잔여 anzeon 6범주(교차-call 비교·거버넌스 다단계·gasTip 조합·fee-cap tx·명시 nonce·WS)의 표현력 블로커는 2026-08-09 전부 해소** — `save`/`$ref` 바인딩+`read`(T4.2a)·`logs`(eth_getLogs)·`gasPrice`+범용 `rpcCall`·tx fee-cap/nonce 인자·`wsSubscribe`. 예제: `stablenet-token-invariants`·`governance-event-flow`·`gas-policy-derived`·`stablenet-fee-boundary`·`stablenet-nonce-ordering`·`ws-subscribe-heads`. **라이브 증명**: `TestEngine_Live_NewVocabulary`(실 gstable 4노드, GSTABLE_BIN). 남은 건 표현력이 아니라 **케이스별 이관 작업량**([[legacy-retirement-plan]] §4.3). `test`/MCP/upgrade 가 아직 레거시 사용 → 소비자 이관 전 제거 금지.

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
  - `engine.KeySource` boundary — `PresetKeySource`(기존 세트 사용) / `GeneratedKeySource`(`keygen.GeneratePreset` 로 신규 생성, idempotent 재사용). `Dir()` 이 구성 시점에 알려지므로 launcher·genesis source 배선은 그대로.
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
| **T7.11** | **레거시 스택 A 제거** — 진행: `core/probe`→`core/collector`(Detect) · `Plan`→`core/driver` · `pipeline/verify`→**`core/health`**(app.VerifyNetwork 경유) · `pipeline/attach`→**`core/node.AttachedSet`** 흡수 완료. pipeline 3/5 소멸(verify·attach 제거, Plan 이전). **표면 이관 완료**(§1d) · **패키지 이동 완료**(§1e: `pipeline/setup`→`core/bringup`) · **netcompose 대체 진행 중**(§1f: b-1~b-4 완료, b-5·b-6 은 라이브 검증 선행). **잔여**: `pipeline/testrun`+`testkit`(cmd test + mcp, **케이스 이관 91건 선행**) | 표면은 app 1곳으로 수렴 — 남은 건 라이브 검증 후 전환·삭제, 그리고 케이스 이관(작업량) | ☑ **완료 확인 2026-09-07.** 대상 셋이 전부 없다 — `core/probe`·`core/driver`·`internal/pipeline` 어느 것도 트리에 없고, 흡수처(`core/collector`·`core/health`·`core/node`)만 남았다. 게이트로 재고 닫는다 |
| **T7.12** | **`overlays/account-extra.json` params 형식 교정** — `internal/chains/stablenet/overlays/account-extra.json` 의 `govCouncil.params.authorizedAddresses`·`blacklistedAddresses` 가 JSON 배열인데, genesis 의 `SystemContract.params` 는 `map[string]string` 이라 `gstable init` 이 `cannot unmarshal array ... of type string` 으로 거부한다. 콤마로 이어붙인 문자열로 고쳐야 `setup --genesis-overlay` 기동이 성공한다. 다른 소비자(레거시 testkit setup 경로)도 이 오버레이를 쓰는지 확인 후 일괄 교정. **게이트**: 이 오버레이로 `tests/repro/stablenet-account-extra.sh` 가 스킵 없이 통과 | 라이브 검증 때 스크래치 사본으로만 우회했고 원본은 그대로다 — 오버레이 경로가 실제로는 깨져 있다 ([[remaining-work]] §1.1) | ☑ **완료 확인 2026-09-07.** 오버레이에 `params` 키가 아예 없고, **왜 넣으면 안 되는지**가 파일 주석에 남아 있다(base 템플릿의 `govCouncil.params` 는 평평한 string-map 이라 배열을 넣으면 `gstable init` 이 거부한다). 이 오버레이를 쓰는 스펙 4건 전부 `validate` 통과 |
| — | **T5.2 업그레이드 멀티바이너리** · **T5.5 wemix4 이관** · **실 SSH 라이브 e2e** | §2 기존 항목, 환경 의존 | ☐ |

---

## 1d. T7.11a — 레거시 스택 A 표면 이관 (2026-08-18 완료)

> 목표: 레거시 패키지를 **삭제하는 것**이 아니라, 삭제를 기계적으로 만드는 것.
> 착수 전 실측: `pipeline/setup` 소비자 7파일 · `core/state` 소비자 9파일이 각자
> `state.Load → setup.X → state.Save` 를 반복하고 있었다. 이 중복이 삭제 불가의 실제 원인이었다.

| # | 작업 | 결과 |
|---|---|---|
| **T7.11a-1** | 네트워크 수명주기 유스케이스 | `driver.StopNode/RelaunchNode/StopNodeSet` 흡수(`Plan`→driver 와 동일 근거) · `app.NetworkStatus/NetworkStop/NodeStop/NodeStart/NetworkRemove` + `app.GCSessions` 신설 · `Deps.Driver` boundary(프로세스 없이 테스트, 원격 라우팅) · cmd status/stop/node/clean + stop MCP 도구 이관 |
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
| **A3** | `app` 의 `topology.yaml` 쓰기를 `filestore.Store` 경유로 | ☑ **정리가 아니라 결함 수정이었다** — 원격 프로비전이 genesis·config 를 조작자 머신에 쓰고 있었다(신원만 원격으로 갔다). `Deps.Files` boundary 추가 · 드라이버가 파일을 보낼 수 있으면 그것이 기본 저장소 · 회귀 테스트 3건 |
| **A4** | `chains/wemix/deploy` 를 `FileStore` 경유로 | ☑ `pullKeystores` 가 읽기·쓰기 양쪽 모두 store 경유. 직접 파일 쓰기 0건 |
| **A4b** | `chainsetup`·`consensus/upgrade` 를 `FileStore` 경유로 | ☑ **완료 2026-08-23.** 실측은 8+2 가 아니라 **11+2 = 13곳**. `Options`/`HandoffOptions`/`upgrade.LaunchOptions` 에 `Files` boundary(nil=로컬). **직접 쓰기 0건**이 되어 [[layers]] §5 의 ❌ 가 사라졌고, **A2 가 stale 예외를 잡아** 표에서 지우게 했다(가드가 의도대로 동작). `os.MkdirAll` 대부분이 함께 사라진 게 핵심 — `Write` 가 부모를 만들므로 디렉토리를 미리 만들던 코드는 **경로를 아는 코드**였고 그게 위반의 실체였다. 라이브: `chain up --case wemix` 15/15 · 엔진 게이트 통과 | ☑ |
| **A5** | ~~`core/bringup`·`core/state`·`testkit`·`core/pipeline/testrun`~~ **전부 소멸 2026-09-06** | — | ☑ 앞 둘은 #241 에서(`engine.LocalSetup`·`session.SaveLocalNodeSet` 로 수렴). 뒤 둘도 이제 없다 — 케이스 이관(P6.4)과 R 트랙 재편이 가져갔다 |
| **A6** | ~~`netreg`·`obs` 를 `session` 으로 흡수~~ **완료 2026-09-06 확인** | 컨트롤 플레인 단일화 | ☑ R1 에서 끝났다. `internal/netreg` 도 `core/obs` 도 없고, 네트워크 레지스트리는 `core/session/netreg.go`(`SaveNetwork`·`ListNetworks`), 이벤트 버스는 `core/collector/bus.go` 다 |
| **A7b** | ~~**`hardfork` 와 `upgrade` 통합**~~ **통합하지 않기로 재확인 (2026-09-07)** | 세 사례가 한 선언에서 갈리는지 | ☒ **결정 유지, 근거는 다시 세웠다.** 2026-08-31 의 근거는 "의도적으로 다른 모델"과 `Plan`·`BuildPlan` 이름 충돌이었는데, **이름 충돌은 근거가 아니다** — A7 이 보였듯 이름은 여러 이유로 겹친다. 입력을 세어 다시 판단했다.

| 사례 | 지금 어떻게 표현되나 | 입력 |
|---|---|---|
| 같은 체인 바이너리 스왑 | `chain hardfork` | 4개(워크스페이스·대상 체인·대상 바이너리·블록). **이미 조립된 망**에 작용한다. 제네시스는 그대로, 데이터 디렉터리도 그대로, 노드만 새 실행 파일로 돌아온다 |
| wemix→wbft 핸드오프 | `upgrade run` | 11개(역할·network id·포크 블록·from 제네시스 바이트·to 제네시스 입력·거버넌스 멤버 주소·노드 공개키·포트 4종). **제네시스를 만들고 망을 새로 조립**하며 두 바이너리를 동시에 돌린다 |
| genesis 전용 포크 | `hardforks: {"boho": 10}` 또는 `--set bohoBlock=10` | **계획이 아니다.** 제네시스 config 필드 하나이고, 스왑도 계획도 없다 |

제안된 선언 `Hardfork{AtBlock, BinaryAfter, ProducersAfter}` 를 대 보면 첫째는 들어맞지만, **셋째는 애초에 이 모양의 선언이 아니고**(제네시스가 이미 말하는 것을 두 번째 철자로 다시 말하게 되는데, 그게 A7 이 막으려는 것이다), **둘째는 자기만 쓰는 필드 여덟 개가 더 필요하다.** 그것들을 공용 선언에 넣으면 모든 하드포크 선언이 자기와 무관한 여덟 칸을 이고 다니고, "메커니즘은 파생된다"는 규칙은 결국 **그 필드들의 존재로 메커니즘을 판별**하게 된다. 메커니즘의 입력에서 메커니즘을 파생하는 것은 파생이 아니다.

대신 A7 이 지목한 이름을 고쳤다: `hardfork.Plan`→`SwapPlan`, `hardfork.BuildPlan`→`PlanSwap`. 둘이 다른 것이라면 같은 낱말을 쓰지 말아야 한다. `BuildPlan` 겹침은 사라졌고 `Plan` 은 넷에서 셋으로 줄었다. 비테스트 수정 4곳 |
| **A8** | **외부 소비자 0 인 공개 심볼 정리** | 원인별로 갈라 세고, 진짜 도달 불가만 없앤다 | ☑ **완료 2026-09-07 — 세 번 보고한 숫자가 모두 틀린 것을 말하고 있었다.** 230 → 896 → 928 은 "이만큼 죽었다"로 읽혔지만, 실제로는 "다른 패키지가 셀렉터로 이름을 대지 않는다"였고 거기엔 원인이 다섯 있다. ① **메서드**(928 중 424): 수신자가 값이라 이 방식으로는 **아예 측정 불가**다. ② **공개 시그니처에 든 타입**: 호출자가 값만 받고 타입 이름을 안 적는다(`app` 의 반환 타입들). ③ **형제가 쓰이는 상수 세트의 한 멤버**: govbind 의 제안 상태 일곱은 컨트랙트 getter 가 돌려주는 값의 대응표라, 아직 아무도 안 쓴 것을 지우면 표가 반쪽이 된다. ④ **자기 패키지만 쓰는 것**(277). 여기서 제안된 처방인 "대부분 unexport"는 **틀렸다** — `nodeconfig` 의 `Key*` 111개는 knob 의 공개 어휘이고 일부는 밖에서도 쓴다. ⑤ **정말 아무도 안 부르는 것: 1,226개 중 3개**. 지웠다: 컨텍스트 없는 `store.Import` 래퍼(A7 의 겹침도 함께 사라졌다) · 구현체도 소비자도 없이 `Driver` 를 다시 설명하던 `process.Transport` · 표면이 자기 목록을 안 들게 하려고 만들었는데 그런 표면이 없던 `remote.EnvNames`. 계산은 `internal/arch/reach.go` 한 곳에 있고 도구(`code-graph -dead`)와 테스트가 그것을 함께 쓴다(#349 의 교훈). 테스트가 지키는 것은 개수가 아니라 **불변식**이다 — 공개인데 아무도 이름을 대지 않는 것이 0. 개수에 예산을 걸면 knob 하나 추가할 때마다 실패하고, 옳은 일을 벌하는 래칫은 사람들이 끄는 법을 배운다. **변이가 내 결함을 잡았다**: 첫 판은 자기 선언이 자기 이름을 한 번 쓴다는 것을 안 빼서 `nowhere` 가 구조적으로 0 이었고, 안 부르는 함수를 넣어도 통과했다 |
| **A7** | **이름 겹침 검출 테스트** — 공개 패키지 레벨 선언이 두 패키지 이상에 같은 이름으로 있으면 보고한다 | ☑ **완료 2026-09-07.** `internal/arch/naming.go` 가 세고 `naming_test.go` 가 래칫을 건다. **메서드는 세지 않는다** — 수신자가 이름공간이라 `Stop`·`Save` 같은 것 577개가 목록을 채우고 `RoleValidator` 를 덮었다. 규칙으로 설명되는 것 101개(app 파사드는 **참조까지 확인한다**. 그래야 뜻이 다른 `app.Network` 대 `keyring.Network` 를 파사드로 오인하지 않는다), 관용 19개, 빚 33개이며 관용과 빚은 둘 다 줄기만 하고 유령 항목도 거부한다. 변이 둘(새 겹침 추가 · 참조 검사 제거)로 확인했다. **첫 수확이 `RoleValidator`×3** 이라 그 자리에서 지웠다: `dsl.RoleValidator`→`BinaryAfter`(포크 뒤 바이너리), `validatorset.RoleValidator`→`AccountValidator`(계정의 기능). 08-25 이 지목했던 `Runner`×3 과 `Ports` 는 빚 목록에 이유와 함께 남았다 | ☑ |

**이름은 각 항목이 자기 범위에서 함께 고친다**([[layers]] §5b). 개명만 하는 커밋은 리뷰가 어렵고
동작 변경과 섞이면 더 어렵다. 규칙: **한 개념 = 한 이름, 다른 개념 = 다른 이름, 식별자는 명명된 타입.**

**A1·A2 를 먼저 하는 이유**: 이후 모든 작업(B·F 포함)이 규칙 위반을 자동으로 잡힌다.
설계를 지키는 일을 사람의 주의력에 맡기지 않는다.

### B — DSL 파서를 모듈로

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **B1** | **`testspec` 분할** — 문법과 런타임을 가른다 | `validate` 는 다이얼도 쓰기도 하지 않는다 · 파서 fuzz | ☑ **완료 2026-09-08 — 게이트를 다시 썼다.** 분할은 끝났고(#311·#326), fuzz 도 끝났다(#357, 550만 실행에 v1 문법 결함 셋 발견).\n\n**옛 게이트 "`validate` 가 rpc/session 을 링크하지 않는다"는 통과할 수 없고, 이제 잴 값어치도 없다.** U 트랙이 모든 표면을 `app` 으로 모았고 `app` 의 fanOut 이 22 라, 한 바이너리에서 어느 명령을 켜도 전부 링크된다. 바이너리를 쪼개면 문자는 만족하지만 그 통합을 되돌린다.\n\n**실측이 진단도 뒤집었다**: `validate.go` 는 rpc·session 을 **한 번도 참조하지 않는다**(쓰는 것은 dsl·dsl/interp·testhelper·core/node·core/registry). 링크는 둘에서 온다 — ① run 경로와 같은 패키지에 산다, ② `validate` 가 이름 해석에 부르는 `testhelper.Registry()` 가 **이름 목록이 아니라 액션 구현 자체**라 rpc·session·keyring·accounts 를 끌고 온다. 그래서 파일을 별도 패키지로 옮겨도 ②가 남는다.\n\n**게이트는 행동으로 바꿨다** — 지킬 것은 링크 크기가 아니라 "아무것도 없을 때 돌리는 명령"이라는 계약이다. `TestValidate_TouchesNothing` 은 커밋된 스펙 122개를 빈 작업 디렉터리에서 검증하고 **아무 파일도 안 생기는지**와 25ms 안에 끝나는지를 본다(다이얼했다면 시한까지 기다렸을 것이다). `TestValidate_ReachesForNothingLive` 는 `validate.go` 가 rpc·session·process·filestore·remote·chainsetup 을 import 하지 않는지 본다 — 라이브 검사가 들어오면 여기서 먼저 걸린다. 변이 둘로 확인.\n\n**어휘에서 이름을 떼어내는 안(②의 근본 해결)은 하지 않는다.** 액션 45개의 등록 방식을 바꾸는 일인데 얻는 것이 오프라인 검증의 링크 크기뿐이다. 문법만 쓰는 다른 소비자가 생기면 그때 근거가 선다 |
| **B2** | `Spec.Fingerprint()` 가 `string` 반환 (현재 `session.Fingerprint`) | **구문이 L3 에 묶인 유일한 이유**가 이 타입 하나다 — B1 의 선행 조건 | ☑ **완료 2026-09-07 (#359) — 다만 이 행의 전제가 틀렸다.** "구문이 L3 에 묶인 유일한 이유가 이 타입 하나"가 아니었다: `dsl` 자체는 fanOut 0 이고, 묶인 것은 `dsl/interp` 이며 그것이 `session` 을 쓰는 곳은 세 파일이다(`fingerprint.go` 1개 · `interpreter.go` 4개 · `run.go` 9개). `Fingerprint` 를 `string` 으로 되돌려도 첫 파일 하나만 풀린다. 실제로 한 일은 **요구를 좁힌 것**이다 — interp 는 `Environment` 11개 중 4개, `TestRecord` 12개 중 4개만 쓰므로 `NodeTable`·`Recorder` 로 선언했다. 완전 분리(결과 타입 이관)는 서로 맞아야 하는 구조체를 둘 만들어 아티팩트에서 증거가 조용히 빠지는 길이라 **하지 않기로 했고 근거를 남겼다** |

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
| **K9** | **원격 링** (사용자 결정 2026-08-25) — `--keyring-dir` 이 target 문법(`srv://…`)을 수용, 링의 생성·조회·가져오기가 **파일 인터페이스 경유로 서버 위에서** 동작(`GenerateAt/ExtendAt/ImportAt/LoadPresetAt`, 로컬 래퍼 유지로 기존 18개 호출처 무변경). `--server-set`·`--docker` 는 전 동사 공통 플래그로 승격. **수정된 선재 결함**: 원격형 경로를 로컬 폴더 *이름*으로 취급해 운영자 머신에 조용히 생성하던 오배치 | 라이브(함대): 서버 위 생성·add 비승격·`list --verify` 원격 통과·중복 생성 원격 존재 검사로 거부·`--docker` 부재 시 실주소 timeout · 게이트 스위트 `Live_…CreatesARingOnAServer` 상설화 | ☑ |
| **K10** | **링 통째 가져오기 + 무결성 게이트** (사용자 결정 2026-08-25) — `import --from-ring <target>` 이 원격/로컬 링 전체를 한 명령으로 복제: 라벨·순번·validator 선언(BLS 목록·alloc 포함) 그대로, 항목마다 키에서 재파생해 원본 인덱스와 대조(`Entry.Verify`) 후 하나라도 다르면 전체 거부, 목적지에 링이 있으면 거부, 비밀번호는 원본 유지 또는 `--password` 재암호화. 단건 import 는 `--expect-address` 로 같은 대조를 호출자가 걸 수 있다. MCP `keyring_import` 에 `fromRing`/`expectAddress` 동일 노출 | 단위(복제가 선언 보존·변조 원본 거부·목적지 점유 거부) + CLI(선언 동반·플래그 혼용 거부·주소 불일치 거부) + 라이브 `Live_…ClonesARingFromAServer`(서버 링 → 로컬, 치환 보고·양측 주소 일치·로컬 재검증) | ☑ |
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
| **N0** | 역할을 `bp·en·pn` 3종으로 정리 — **NM1 로 흡수**([[netmap-design]] §4) | `validator→bp`·`endpoint→en` 이관 · `node.RoleEN`/`RoleEndpoint` 중복 해소 · 기존 토폴로지 호환 | ☑ **NM6 완료 2026-09-07.** 조립·핸드오프·attach 가 모두 `bp`/`en` 을 기록하고 `LegacySpelling` 은 없다. 옛 철자는 **읽는 자리에서 접는다** — `Node.UnmarshalJSON` 과 `Record.UnmarshalJSON` 둘 다에 붙였는데, 워크스페이스가 표를 두 벌(`Record` 는 workspace.json, `Node` 는 단계 사이 전달) 들고 있어 한쪽만 접으면 `chain status` 는 "validator", 나머지는 "bp" 를 말하게 된다. 선택자는 집합 대신 `node.Is` 로 접어 코드가 직접 넘긴 표까지 맞춘다. 알아채지 못한 채 굳을 뻔한 것을 테스트 다섯 개가 잡았다(라이브 재검증은 아직) |
| **N0b** | **피어링 그래프를 역할에서 파생** — 현재는 풀메시 고정. `bp ↔ pn ↔ en`(en 은 bp 를 직접 모른다) | ☑ **NM2 완료** — mesh 골든 동일 · proxied 는 bp/en 의 목록에 pn 만 · poa+pn 은 `SupportsRole` 로 거부. 조립 4곳의 **전환**은 NM3 | ☑ |
| **N1** | `blueprint` 선언 스키마 + 파서 (L1 순수) | 부분 청사진 round-trip · 미지 필드 거부 · fuzz | ☑ **완료 2026-09-07.** `internal/core/blueprint` — `Parse`·`Marshal`·`Validate`. **해석은 하지 않는다**: 빠진 값을 채우는 일은 N2 몫이고, 둘을 갈라 두어야 부분 선언이 왕복한다. 미지 필드는 `KnownFields` 로 거부하는데, 선언에서는 오타가 증상을 남기지 않고 다른 곳에서 온 값으로 조용히 대체되기 때문이다. 검증은 **문서가 혼자 판단할 수 있는 것만** 한다(이름 중복·역할·참조가 선언된 노드를 가리키는지·모순되는 두 지정). 체인 등록 여부나 파일 존재는 확인하지 않는다 — 그것을 여기서 하면 같은 질문이 두 곳에 생긴다. **퍼저가 진짜 결함을 찾았다**: `nodes: []` 와 `nodes` 생략이 서로 다른 값이 되어 생성기 출력이 자기 입력과 같지 않았다(`normalize` 로 접었고 그 입력을 코퍼스에 남겼다). 금액은 텍스트로 든다 — `1500000000000000000000000` 을 숫자로 읽으면 아래 자리가 사라지고 제네시스가 다른 잔액을 만든다. 포트는 `node.Endpoints` 를 그대로 쓴다(NM7 이 끝낸 "포트 표현 3벌"이 선언 형식에서 다시 시작되지 않게). 왕복 9종 · 미지 필드 6종 · 문서 내부 규칙 22종 · fuzz 326만 실행, 변이 셋으로 확인 |
| **N2** | `Resolve` — 출처 사슬 + `Sources` 기록 | 같은 청사진 → 항상 같은 `ResolvedNetwork`(결정성) | ☑ **완료 2026-09-07.** `Resolve(bp, Inputs) ResolvedNetwork`. 파일도 소켓도 열지 않고 배치·키셋·체인 사실을 **주입받는다** — 그래야 같은 입력이 같은 스냅샷을 낸다. **포트는 필드 단위로 병합한다**: `ports: {p2p: 8589}` 는 p2p 만 고정하고 나머지는 배치가 정한다. 통째로 골랐다면 포트 하나를 적는 일이 나머지 여섯을 버리는 일이 됐을 것이다. **검사를 두 곳으로 갈랐다** — 문서 혼자 판단할 수 있는 것은 `Validate`, 노드 표가 있어야 아는 것은 `Resolve`. 이 구분이 결함 둘을 드러냈다: ① 이름을 안 붙인 문서에서 `validators: [bp1]` 을 `Validate` 가 거부했다(역할 라벨은 파생되므로 정당한 참조다), ② 바이너리 오버라이드가 **선언된** 이름으로만 맞춰 이름 없는 노드는 영영 못 바꾸면서 아무 말도 안 했다. 이제 확정된 이름으로 맞추고, 가리키는 노드가 없으면 거절한다. 검증자 목록은 노드 순서로 낸다 — 제네시스가 목록으로 기록하므로 순서가 흔들리면 문서 하나에서 제네시스 둘이 나온다. 결정성 20회 반복 · 출처 9종 · 거절 5종, 변이 셋으로 확인 |
| **N3** | **raw 경로 완성** — 노드별 nodekey·계정·포트·서버·바이너리 오버라이드 선언 | **preset 없이** 손으로 쓴 청사진만으로 3체인 4노드 기동(라이브) | ☑ **완료 2026-09-07 — 라이브 게이트 통과.** `chain up --blueprint network.yaml` 로 stablenet·wbft·wemix 각 4노드(bp 3 + en 1)가 preset 디렉터리 없이 서서 블록을 만들었다(높이 15/15/59, 노드끼리 일치). 청사진이 키를 들면 `store.DeclaredKeys` 가 그 링을 재료화하고, 조립은 `KeySource` 경계를 그대로 쓰므로 새 경로를 배울 것이 없다. **첫 시도는 조용히 실패했다** — 모든 스텝이 성공을 보고했는데 봉인 노드가 전부 `Failed to read password file` 로 죽었다. 링 기록 절차를 일부만 되풀이해 공용 `password` 파일을 빠뜨린 탓이고, 이 트랙이 없애려는 갈라짐을 내가 만든 셈이다. `ImportRing`(이미 있던 단일 writer)을 쓰게 고쳤고 덤으로 키가 색인이 주장하는 신원을 재파생하는지까지 검증한다. **못 지키는 선언은 이름을 대고 거절한다**: 노드별 `server:` 는 아직 배정기가 못 받으므로 무시하지 않고 거부한다. 남은 빚 둘: 노드별 서버 배치, 그리고 BLS 를 패밀리에 묻지 않고 늘 파생하는 것(생성 경로도 같다) |
| **N4** | `Materialize` — 노드별 산출물 묶음 → Sink | 로컬/원격 분기 없음 | ☑ **완료 2026-09-07 — 재구조화는 A3·A4b 가 이미 끝냈고, 남은 일은 그것을 못 박는 것이었다.** 실측: `chainsetup` 에 직접 파일 쓰기 **0곳**, 18곳이 전부 `Files` 경계를 지난다. `IsRemote()` 를 묻는 자리는 10곳인데 **9곳은 "대상이 어디냐"는 사실**이지 두 번째 코드 경로가 아니다(원격 노드의 키는 대상 데이터 루트 밑에 있고, 로컬은 키 세트 그 자체다 — 조작자 쪽 경로를 원격 config 에 박은 것이 노드가 못 보는 기계에서 nodekey 를 찾게 만든 원인이었다). `internal/arch` 에 래칫을 두어 **각 자리가 왜 사실인지를 목록에 적게** 했다. 줄기만 하고, 사라진 항목도 거부한다. 변이 둘로 확인. **진짜 결함은 하나**: `rm` 이 원격 데이터 플레인을 거부한다. `filestore.Store` 에 삭제가 없어서(확인·읽기·쓰기·체크섬뿐) 경계를 통과할 수 없다. 파괴적 원격 작업이라 검증할 기계가 있는 **R6 과 함께** 간다. 그때까지는 절반만 지우거나 지웠다고 거짓말하지 않고 소리 내어 거절한다 |
| **N5** | preset 을 청사진 **생성기**로 — `chain blueprint --from-preset` | preset 산출 청사진이 N3 경로와 동일 결과 | ☑ **완료 2026-09-07.** 뒤집힘이 성립한다: `preset → (내부 조립) → 네트워크` 가 `preset → 청사진 → 네트워크` 가 됐고, 가운데를 열어 보고 고칠 수 있다. **키는 경로로 가리키고 문서에 박지 않는다** — 청사진은 읽고 나누고 커밋하라고 만든 것이라, 비밀키를 넣는 생성기는 기본값을 위험하게 만든다. 테스트가 문서에 키가 섞였는지 본다. 동일성은 저장소 preset 으로 확인했다: 찍은 문서를 다시 읽어 해석하면 주소·devp2p 키·BLS 키·검증자 순서가 preset 이 든 것과 전부 같다. **라이브**: 찍은 문서로 wbft 5노드(bp 4 + en 1)가 서서 높이 15, 워크스페이스의 검증자 넷이 preset metadata 와 순서까지 일치. `chain blueprint` 는 명령 표면의 네 번째 갈래(저작)로 등록했다 — 대상을 건드리지 않으므로 `stop`·`rm` 과 같은 칸에 두면 도는 체인에 뭔가 한다고 말하는 셈이다. 변이 둘(키 인라인 · 노드 순서 뒤집기)로 확인 |
| **N6** | `topology.yaml` 흡수 (이관 기간 병존, 혼용 거부) | 두 형식을 다 읽되 한 조립에 섞으면 거부 | ☑ **완료 2026-09-07.** 청사진은 토폴로지의 진부분집합이라(토폴로지가 말하는 것 전부 + 키·포트·서버·계정·alloc·거버넌스) `chain blueprint --from-topology` 로 손 번역 없이 넓은 형식으로 옮긴다. 그래서 흡수이지 손실 있는 이관이 아니고, 플래그 데이 없이 둘을 병존시킬 수 있다. **혼용은 거부한다** — 한 조립에 청사진과 토폴로지를 같이 주면 둘 중 한 저자는 자기 것이 아닌 네트워크를 읽게 된다. **옮길 자리가 없는 것은 버리지 않고 보고한다**: bootnode 표시와 토폴로지식 바이너리 이름(청사진은 경로를 직접 쓴다). 조용히 잃으면 온 파일과 다른 네트워크를 만드는 문서가 되고, 그게 이 트랙이 없애려는 실패다. 레거시 철자는 변환에서 정본으로 접힌다(NM6) |
| **N7** | **`core/netmap`** — ☑ **완료 2026-08-22** (NM1·NM1c·NM1b·NM2·NM3·NM4·NM5). 배치의 단일 소유자가 섰다: 자원 풀·결정적 할당·양방향 대장·라벨·피어링·경로. `place` 할당기 소멸 | 실측(착수 전): 배치 타입 8개 · 포트 표현 3벌(etcd 소실) · 역할 어휘 4벌 · static-nodes 조립 4벌 전부 풀메시 · `"node%d"` 라벨 파생 32곳. **종료 시**: 포트 1벌(etcd 보존) · 역할 폴딩 1곳 · 피어링 1곳(mesh/proxied) · 경로 파생 1곳 | ☑ |
| **N8** | ~~`serverset` 가용 자원 풀~~ → **이미 있음**(`slots`·포트 밴드, 2026-08-18). 명시적 범위 풀은 근거 생기면 | — | ☑ |
| **N9** | **해석 순서 강제** | 선행 단계가 안 돌았으면 명시적으로 거부한다 | ☑ **완료 2026-09-07.** `composeNeeds` 한 곳에 선언하고 모든 단계가 `require` 를 거친다. **적어 보니 두 군데가 틀려 있었다**: 순서는 keyring→netmap 이 아니라 `place → keys` 다(`keys` 가 노드 수를 배치에서 받는다), 그리고 `genesis` 는 키셋을 요구하지 않는다(템플릿을 치환하는 외부 체인은 키를 안 읽고, 요구하게 했더니 되던 조립이 깨졌다). 선언을 훑는 표 테스트 + 순환·도달 검사 + 유스케이스 경유 1건, 변이 세 개로 확인 |
| **N10** | **계정 라벨** — 라벨 ↔ 주소·개인키. **faucet 은 예약 라벨** | ☑ **완료 2026-09-05.** 실측이 문제를 말한다: 스펙 94파일에 주소 505회, 그중 **키에서 파생돼 키셋이 바뀌면 틀어지는 것이 143회**(`node1` 하나가 93회). 라벨은 주소만이 아니라 **서명 방법**까지 답한다 — 노드 계정은 그 노드가, 하네스만 가진 계정은 여기서 서명해 raw 로 보낸다. `0x` 접두사면 주소, 아니면 라벨. 미지의 라벨은 **아는 라벨을 나열하며 실패**(zero address 로 흘러가지 않는다). 주소 자리(`address`·`from`·`to`·`deployer`·`funder`)는 전부 라벨을 받는다. **2차**: `accounts:` 선언으로 dev 계정을 **genesis 밖에서** 만들고 faucet 으로 채운다 — 계정 추가가 genesis 수정을 뜻하지 않게 된 것이 이 단계의 요점. 잔액 없이 선언하면 0으로 남는다(가스 부족 경로 테스트용). 라이브: 선언한 dev1 이 **자기 키로 서명**하고 dev2 잔액이 정확히 1 wei(0에서 시작해 그 한 번만 받음). 잔여: 스펙 143곳 이관(기계적) | ☑ |
| **N11** | **다중 config** — 노드별 지정 | 일부 노드만 다른 config 로 재기동한다 | ☑ **완료 2026-09-08.** 게이트는 `swapNode` 가 이미 만족했고(`binary` 없이 `config` 만 줘도 액션·유스케이스·워크스페이스 세 층이 받는다), 빠져 있던 통합 테스트를 채웠다. 걸림돌이던 "키셋을 갖춘 기동된 망"은 실제로 조립해서 만들었다 — preset 키·배치·제네시스·config·argv 기록까지 진짜로 하고 드라이버만 스텁이다. 검증: node2 의 config 가 `SyncMode = "snap"` 으로 다시 렌더되고, **node1 은 그대로이며**, 오버라이드가 `node2` 범위에만 기록돼 나중 재기동이 그것을 쓴다. `restartNode` 에 `config:` 를 또 붙이지 않는다 — 한 동작에 철자가 둘이 된다. 변이 둘(config 만은 거부 · 오버라이드를 전 노드 범위로)로 확인. 부수 확인: config 만 바꾸는 스왑도 **바이너리 기록은 필요하다**(재기동에 실행 파일이 있어야 한다) |
| **N12** | **deploy skip 을 내용 해시로** | 같은 경로에 **다른 내용**이면 skip 하지 않음 | ☑ **2026-09-06.** deploy 가 genesis 와 config 를 **존재 여부만** 보고 "reused, not rewritten" 이라고 보고했다. 누가 편집한 genesis 도, 이전 구성이 남긴 config 도 똑같이 "있음"이라 노드가 그걸로 뜬다.

genesis·config 를 쓸 때 해시를 `State.LaunchInputs` 에 기록하고, deploy 가 대상의 내용과 대조한다. 다르면 파일 이름과 두 해시를 대고 어느 스텝을 다시 돌릴지 말한다. 되쓰지 않고 거절하는 이유는 deploy 의 일이 만드는 것이 아니라 확인하는 것이기 때문이다.

**같은 스텝 안에서 신원 파일은 이미 제대로 하고 있었다** — `shipIdentities` 가 `Checksum` 과 `filestore.Hash` 로 내용을 견준다. genesis·config 만 빠져 있었다.

테스트 3건(편집된 genesis 거절 · 교체된 config 거절 · 자기가 쓴 것은 통과). 변이 둘로 확인했다: 비교를 끄거나 해시 기록을 빼면 편집된 genesis 가 "reused" 로 통과한다 |
| **N13** | **skip 사유 기록** — `TestRecord.Status(s)` 에 사유가 없어 "왜 skip 됐는지"가 아티팩트에 안 남는다 | ☑ **F5 와 함께 완료 2026-08-22** — `TestRecord.Reason(why)` + `statusDoc.Reason`. 적용 4곳: 파싱 실패·미적용(`does not apply to this target (chain or required capabilities)`)·blocked 2종. 이유 없는 blocked 하나가 `chain.binary` 누락을 찾는 데 한 세션을 썼다 | ☑ |
| **N14** | ~~**`capability` 이름 충돌 정리**~~ **대상 소멸 2026-09-06** | — | ☒ `engine/capability` 도 `core/capability` 도 없다. 이름이 겹치는 패키지는 `wbft` 하나뿐이고(체인 플러그인과 합의 패밀리) 그건 별개다. 충돌은 R 트랙의 모듈 재편 과정에서 사라졌다 |

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
> **2026-09-05: 표면 경로 부분은 §1l 의 U 트랙이 가져갔다.** S5·S6 은 폐기했다. 아래 줄수는
> 08-18 시점 기록이며, 지금은 CLI 가 네 패키지로 나뉘어 합계 4,176줄이다(`cmd/chainbench`
> 1,981 · `keyringcmd` 1,119 · `chaincmd` 828 · `resourcecmd` 248, 비테스트 기준). 코드가 옮겨
> 갔을 뿐 줄지 않았다.
> 실측 문제: `cmd/chainbench` 4,569줄 중 **21파일이 `app` 을 우회**해 L1~L4 를 직접 부른다.
> MCP 도구 46개 중 **34개가 JSON 스키마 손작성**. DSL 은 또 다른 레지스트리를 갖는다 —
> **기능 목록이 세 벌**이고 실제로 갈라졌다(`faucet`·`verify` 는 DSL 에 없다).

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **S0** | **`internal/feature`**(별도 패키지, `Deps` 소유) 레지스트리 골격 · 입력 태그→cobra 플래그/JSON 스키마 바인딩 · **`ReadOnly` 속성**(선언식 — 상태 불변 + 출력에 비밀 없음; `keyring export` 는 자격 없음) | 기존 동작 무변경 · 미등록 기능 카운트 테스트 · ReadOnly 선언이 스키마에 노출 | ☑ **완료 2026-09-08.** `internal/feature`(L5): `Registration`·`Register[In,Out]`·`Stage`·`ReadOnly`, 그리고 **입력 struct 태그 하나가 두 바인딩을 만든다** — `Flags` 가 cobra 플래그를, `Schema` 가 MCP JSON 스키마를.

**게이트 셋 다 실측으로 닫았다.** ① *기존 동작 무변경*: `NetGenesisIn` 에 태그를 달았고(태그는 무해하다), 파생한 플래그 4개가 `chain genesis` 가 손으로 선언한 것과 이름·타입·도움말까지 일치한다. 그래서 S1 의 이관은 표면을 바꾸는 일이 아니라 **손으로 쓴 절반을 지우는 일**이 된다. ② *미등록 기능 카운트*: app 의 유스케이스 80개 중 0개 등록. 큰 숫자이고 그게 계획이다 — 래칫이 막는 것은 이 숫자가 **오르는 것**이다. ③ *ReadOnly 가 스키마에 노출*: `readOnlyHint` 로 나간다.

**이름 둘을 A7 이 커밋 전에 잡았다**: 설계가 `Descriptor` 라 불렀는데 그 이름이 이미 두 뜻(`core/registry` 의 체인 플러그인, `app` 의 재수출)으로 있어 `Registration` 으로, `All` 은 `core/registry` 것과 겹쳐 `Registered` 로 바꿨다.

**명령은 생성하지 않는다**(§3.4). 이름·계층·도움말 문구는 사람이 정하고, 태그에서 만드는 것은 플래그 바인딩뿐이다. 변이 셋으로 확인 |
| **S1** | ① Compose 이관 — `net.*` 9스텝 등록(이미 `app` 경유라 등록만) | `net up` 3체인 회귀 | ☑ **완료 2026-09-08.** compose 스테이지 **13개** 등록(9스텝 + 조회 넷). 미등록 카운트 80 → **67**, 래칫이 내려간 것을 알아채고 상수를 낮추라고 실패로 말했다. **태그 대조가 갈라짐 하나를 잡았다**: `chain.keys --keys-source` 가 태그엔 기본값 없음, 명령엔 `preset` 이었다. 명령을 파생으로 바꿀 때 몰랐으면 `chain keys` 의 기본 동작이 조용히 바뀐다. 그래서 **기본값 태그를 이번에 만들었다** — 영값에서 유추할 수 없다(`--validators` 는 4가 기본, `--endpoints` 는 0이 기본인데 유추하면 검증자 0인 네트워크가 된다). 파생 플래그 24개가 이름·타입·도움말·기본값까지 일치한다. 대조는 **한쪽 방향**이다 — 태그가 파생하는 것은 명령에 다 있어야 하고 명령은 더 가져도 된다(`--server` 류는 공유 바인더 몫이라 어느 한 입력의 것이 아니다). **라이브 3체인 회귀 통과**: 등록 전 커밋을 별도 worktree 에 빌드해 같은 절차로 나란히 돌렸다(stablenet 20/24 · wemix 58/58 · wbft 24/0). wbft 만 갈려서 before↔after 를 번갈아 4판씩 더 돌렸더니 **before 도 4판 중 1판이 0** 이고 after 는 2판이 정상 — 등록과 무관한 wbft 자체의 불안정이고, 이번 주에 규명한 "검증자가 늦게 합류하면 망이 안 선다"와 같은 자리다(`chain up` 은 아직 `health.Participants` 를 안 쓴다). **측정 실수 셋을 기록해 둔다**: 12초 단일 표본이 head 0 을 회귀처럼 보이게 했고(wbft 는 1에 머물다 가속한다), `pkill -f 'gstable|gwemix'` 가 그 문자열을 본문에 담은 러너 자신을 죽였고, 도는 스크립트를 편집해 bash 가 아직 안 읽은 줄이 어긋났으며 첫 러너가 살아 있는 채 둘째를 띄워 포트 8600 을 두고 다퉜다 — 같은 바이너리·같은 체인이 24 와 0 을 동시에 낸 것이 그 증거다 |
| **S2** | MCP `net_*` 를 레지스트리 소비로 전환 | 손작성 스키마 감소분 측정 | ◐ **조회 전용 목록은 완료(2026-09-08), 스키마 감축은 근거가 약하다.** §4.4 규칙 3 을 구현했다 — 도구가 `ReadOnly` 로 선언하고 `tools/list` 가 MCP 자신의 철자인 `annotations.readOnlyHint` 로 실어 나른다(이 서버는 2024-11-05 를 선언하고 그 필드는 2025-03-26 에 생겼으므로, 모르는 클라이언트는 무시하고 아는 쪽은 정답을 얻는다 — chainbench 만 아는 이름을 짓는 것보다 낫다). 54개 중 **28개**가 선언했다.

**CLI 와 MCP 가 어긋나지 않도록 짝을 못박았다.** 공유 레지스트리가 아직 없어 같은 사실을 두 번 선언하는데(cobra 주석 / 구조체 필드), 그게 바로 갈라지는 모양이라 18쌍을 테스트가 붙들고 있다. 한쪽만 표시를 떼면 실패한다(변이 확인).

**손작성 스키마 감축은 지금 하지 않는다.** 실측: 되풀이되는 속성은 `rpc` 16 · `workspaceDir` 11 · `chain` 11 이고 전부 한 줄짜리 선언이다. **설명이 갈라졌는지 세어 보니 갈라지지 않았다** — `rpc` 15곳이 같은 표현이고 다른 둘은 실제로 다른 인자다(배열 허용, attach 용). 즉 이 감축은 교정이 아니라 예방이고, 54개 도구를 기계적으로 훑는 diff 를 그 근거로 사기에는 약하다. 갈라짐이 실제로 생기면 그때가 착수 시점이고, 그 신호는 이 줄에 적힌 숫자를 다시 세면 나온다 |
| **S3** | ② Test 이관 — `tx`·`faucet`·`contract`·`verify` | CLI/MCP 동시 노출 | ☑ **U4 가 했다(2026-09-05).** 넷 다 `app.TxSend`·`Faucet`·`ContractDeploy`·`VerifyNetwork` 를 CLI 와 MCP 가 함께 부른다(실측 확인 2026-09-06). **DSL 은 뺀다** — 액션은 표면이 아니라 L3 어휘이고 app 을 부르면 import 순환이다(§1l U7 참조) |
| **S4** | ③ Report 이관 — `status`·`report`·`logs` | 등록 + 태그가 배포된 표면과 일치 | ☑ **완료 2026-09-08.** 넷 등록(report.session·report.log·report.timeline·network.status), 모두 조회 전용. 미등록 카운트 67 → **63**.\n\n**등록하려니 세 함수의 모양이 맞지 않았다** — `LogSearch(_ Deps, dir string, in)` 처럼 디렉터리를 별도 인자로 받았는데, 그건 사실 입력이다. `(ctx, Deps, In)` 으로 바꾸면서 필드를 두 번 선언하지 않으려고 **바인더가 임베디드 구조체를 읽게** 했다. 그래서 `app.LogSearchIn` 은 `Dir` + `collector.SearchOpts` 이고, 검색 조건은 여전히 한 곳에만 선언돼 있다.\n\n**표면 래칫이 곧바로 잡았다**: 호출부를 고치다 `reportcmd/log` 가 `collector` 를 직접 참조하게 됐다. app 이 `LogSearchFilter` 로 어휘를 내주게 고쳤고, 그 이름은 A7 이 다시 잡았다 — `LogFilter` 는 `core/rpc` 에서 eth_getLogs 필터를 뜻한다. 파생 플래그 7개가 명령과 일치하고, 변이로 확인했다 |
| **S5** | ~~`cmd/` 규칙 위반 파일 정리~~ **폐기 2026-09-05** | 게이트가 "`cmd/` 가 `app` 만 import" 였는데, 지키려던 성질은 import 목록이 아니라 표면 사이의 동등성이다. 줄수 목표(~1,800)는 **달성하지 못했다**: CLI 네 패키지 합계가 4,569 → 4,176 으로 거의 그대로고, 줄어든 것처럼 보였던 것은 코드가 `chaincmd`·`keyringcmd`·`resourcecmd` 로 옮겨 갔기 때문이다. 폐기 사유는 목표 달성이 아니라 규칙 교체다. 남은 실체는 §1l 의 U 트랙이 가져간다 | ☒ |
| **S6** | ~~`cmd` import 화이트리스트 테스트~~ **폐기 2026-09-05** | 대리 지표 대신 기능별 동등성 테스트로 대체한다(U0·U2~U7). 라체트는 "app 을 거치지 않는 항목 수"로 세운다 | ☒ |
| **S7** | **`query` 조회 투영** — `ReadOnly` 기능들을 최상위 `query <명사> <동사>` 로 **자동 생성**(손 트리 금지, [[surface-unification-design]] §4.4, 확정 2026-08-25). 정본 철자는 명사 그룹, `query` 는 같은 등록의 두 번째 렌더링. MCP 는 같은 속성으로 조회 전용 도구 목록을 얻는다 | `query keyring list` == `keyring list` (같은 등록 실증) · 비-ReadOnly 기능이 query 에 나타나면 테스트 실패 | ☑ **완료 2026-09-07.** `query` 는 손으로 만들지 않고 **명령 트리에서 생성한다**. 각 명령이 `surface.ReadOnly` 로 스스로 선언하고(추론이 아니다 — `node rpc` 는 메서드가 인자라 코드로 알 수 없고, `keyring show` 는 안전한데 `keyring export` 는 비밀을 찍으니 둘 다 "찍기만 한다"로는 갈리지 않는다), 투영이 그것을 읽는다. 19개가 선언했고 `query` 가 정확히 그것만 담는다.

**같은 등록임을 세 가지로 확인한다**: 설명이 같고, `RunE` 가 같은 함수 값이며, **플래그 변수가 공유된다**(투영에 `--json` 을 세우면 정본 명령이 그것을 본다). 셋째가 잡는 위험은 "생성자를 다시 불러 만든 두 번째 등록"이다 — 도움말은 멀쩡해 보이지만 변수가 따로라 갈라진다.

**주석 하나를 실측으로 고쳤다**: "복사로는 변수 공유를 얻을 수 없다"고 적었는데 틀렸다. `pflag.Flag` 의 `Value` 가 인터페이스라 구조체를 복사해도 변수는 공유된다. 탐침으로 확인하고 주석을 사실대로 바꿨다.

변이 셋으로 확인: 쓰기 명령(`chain genesis`)에 선언을 붙이면 잡히고, 플래그를 새로 선언하면 "두 플래그 집합"으로 걸리고, 투영에서 하나를 빼면 선언과 안 맞는다고 실패한다 |
| **S8** | **netmap 조회의 자기 그룹 독립** (사용자 결정 2026-08-25) — 배치 조회를 `net` 에서 분리해 모듈 이름 그대로의 최상위 그룹 `netmap`(show·pool·plan)으로. 동기: 조회가 조합 그룹에 묶여 있으면 netmap 모듈만 고쳤을 때 그 부분만 테스트할 수 없다. `plan` 신설 = 할당기를 질문으로 실행(인벤토리+형태→배치표, 워크스페이스 없음·무기록) → 배치 변경을 조합 없이 검증하는 통로. MCP 도 동일 이동(`chainbench_netmap_show/pool/plan`). `net` 은 상태를 바꾸는 조합 단계만 소유 | CLI 테스트 7건(plan 결정성·무기록·producer 0 거부·인벤토리 반영·자격 비노출, show 양방향·워크스페이스 요구, pool used 집계) — netmap 만 대상으로 실행 가능 | ☑ |
| **S10** | **용도별 포트 대역 + 실서버 방화벽 재현** (사용자 요구 2026-08-26) — ① 서버 세트 `ports` 에 `ws`/`auth`/`metrics` 대역 선언 추가(선언 시 rpc 파생 꺼짐, `portplan.PlanBands`). 로더의 p2pStep 최소는 1로(패밀리 요구는 allocate 의 portplan 이 검사 — wemix 만 넓은 예약, wbft 는 관성이던 예약 2→1 정직화). ② docker 서버들: `firewall.sh` 가 Wemix3.5 테스트 서버 허용 목록(TCP 10022·8501-8504·8601-8604·8701-8704·6060·3000·3001·9100·9090·30301-30304·1099·5901·5044·9200, UDP 30303)만 열고 기본 DROP. ③ sshd 10022 + 서버 세트 `ssh.port: 10022`, gen-env 가 실서버 규격 대역을 서버 세트에 기록 | 단위: 용도별 대역 로드·검증·tight-step 판정. 라이브: 허용/차단 포트 프로브(DROP·refused 구분), 새 대역으로 5대 분산 기동→블록 24→고아 0, keyring·driver·워크플로 라이브 스위트 10022 경유 통과 | ☑ |
| **S11** | **machine.Kind 제거 — 구분은 저장하지 않고 파생** (사용자 결정 2026-08-26) — Kind(local/remote/server)는 축이 지리처럼 읽히지만 실체는 "지정 방식"이었고, 저장된 kind 는 주소와 어긋날 수 있었다. Spec 은 필드만 갖고(서버명·호스트·경로), 지역성은 파생한다: `Server` 있음 → 세트 항목, 루프백/빈 호스트(로그인 미지정) → 이 머신, 그 외 → 직접 지정(환경 인증). 루프백+User/Port 명시는 의도적 SSH dial(터널·퍼블리시 포트). 표시 문구는 `Spec.Describe()` 한 곳으로 — net status·MCP·실행 기록의 자체 분기 소멸, 래칫 3항목 축소 | 단위: 파생 규칙·Validate·Describe. 라이브: fleet 기동·status `server server1:…` 표기·workspace.json 에 kind 없음·실행 기록 where 일치·keyring 스위트 통과 | ☑ |
| **S12** | **호스트키 정책을 파일로** (사용자 정책 2026-08-26: 환경변수 셋업 없이 config 로드) — `CHAINBENCH_SSH_KNOWN_HOSTS`·`CHAINBENCH_SSH_INSECURE_HOST_KEY` 제거. 서버 세트 `ssh:` 에 `known_hosts_file`/`insecure_host_key`(정확히 하나) 선언, 경로는 다른 시크릿 파일과 같은 규칙(세트 파일 기준 상대·`~` 확장). `remote.HostKeyPolicy` 가 데이터, `Spec.ResolveWithPolicy` 가 전달, 직접 표기(user@host)는 세트의 set-level 정책을 적용. docker 생성 세트는 `insecure_host_key: true` 를 스스로 선언 | 단위: 정책 콜백·모순 선언 거부·파일 경로 해석. 라이브: **환경변수 0개**로 keyring 원격 생성·검증, 5대 fleet 기동·정지(고아 0), keyring·driver·sudo 라이브 스위트 통과 | ☑ |
| **S9** | **서버 세트 단일화** (사용자 결정 2026-08-25) — ① 용어: 서버 목록 파일을 "서버 세트"로 통일(파일 `server-set.yaml`, 플래그 `--server-set`, 코드 `serverset` 패키지와 일치; "inventory" 는 Ansible 미경험자에게 낯설고, "server config" 는 노드 config 와 충돌). ② 자격 단일화: 이름 붙은 서버의 SSH 자격은 **서버 세트 파일이 유일한 출처** — 환경변수 참조 제거(남은 export 가 접속을 조용히 바꾸는 사고 차단). 비밀을 파일 밖에 두려면 `ssh.password_file`/`key_passphrase_file` 로 한 줄짜리 0600 파일 참조(시크릿 매니저가 내려주는 형태). `CHAINBENCH_REMOTE_*` 는 참조할 파일이 없는 직접 표기(`user@host:/path`) 전용으로 축소 | 단위: 환경변수를 켜도 파일 값이 이기는 것·password_file 판독(개행 절사)·password/password_file 동시 지정 거부·빈 시크릿 파일 거부. 라이브: docker 서버들에서 keyring 전체 스위트 재통과 | ☑ |

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
| A4 | 원격/fleet 점유 조회 — 모든 서버 대상(요구 ⑩의 "모든 서버") | dial 경로는 있으나 fleet 다중 호스트는 R5 선행(☑ 2026-08-26 해소) | ☐ |

### H — 약속된 위치 (요구 ⑦)

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **H1** | **`core/home`(L0) 신설 — `~/.chainbench` 의 소유자** | ☑ **완료 2026-09-05.** "기본 위치" 가 세 답이었다: 키 세트 `keys/default`, 세션 `chainbench-out` (**둘 다 cwd 상대**), 구성만 `~/.chainbench/<stamp>`. 실측: 다른 디렉토리에서 `keyring list` 가 **자기가 만든 링을 못 보고**, `run` 이 세션을 그 자리에 흩뿌렸다. 셋을 한 답으로 모았고 우선순위(명시 > `CHAINBENCH_KEYRING` > 약속된 위치)는 그대로. 라이브: `/tmp` 에서 만든 링을 다른 곳에서 조회, 경로 없이 구성하면 `~/.chainbench/<stamp>/chainsetup` 에 모인다. 아키텍처 가드가 신규 패키지를 잡아 §3 에 배치될 때까지 막았다 | ☑ |

### G — genesis 정리 (리팩토링 순서 ③)

> 순서: keyring ☑ → netmap ☑ → **genesis** → config. 실측(2026-08-25): genesis 를
> 만드는 진입점이 흩어져 있고, 그 결과 경로마다 기능이 달랐다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **G1** | **조립 지점 단일화** — `engine.BuildGenesis`(소스 선택 + 커스터마이즈)를 모든 경로가 부른다 | ☑ **완료 2026-08-25.** 소스 선택이 **5곳 → 1곳**(엔진·스텝 두 경로가 `poa.FamilyID` 로 분기 + 세 곳이 하드코딩). 커스터마이즈를 소스 **밖으로** 빼서 `netcompose.customizeGenesis`(=`genesis.BuildNetwork` 의 옵션 처리를 줄 단위로 재구현한 사본) 소멸. 라이브: stablenet 4노드 · `chain up --case wemix` 15/15 | ☑ |
| **G1a** | 닫힌 격차 — **wemix + overlay** | ☑ genesis overlay 가 스텝 경로에선 반영되고 엔진 경로에선 **말없이 버려지던** 분기. 커스터마이즈가 소스 안(한 패밀리만 쓰는 곳)에 있었던 탓. 이제 패밀리가 만든 base 위에 동일하게 적용된다 | ☑ |
| **G2** | 핸드오프 경로를 패밀리 선언으로 · `chains/wemix/deploy` 폐기 | ◐ **2026-09-05.** **`remote` 명령군과 `deploy` 패키지(995줄) 폐기** — 네 하위명령 모두 core 경유 대체가 있다(`chain up --server`·poa 페이즈 액션·`upgrade`·`keyring import --from srv://`). **핸드오프가 패밀리 순서를 따른다**: 전원 기동 후 부트스트랩하던 것을 `BringUpPhases` 로 물어 프로듀서 단독 → 거버넌스·etcd → 나머지 → 메시 순으로. 거버넌스 배포 전 `WaitProducing` 도 컴포지션 경로와 맞췄다. **간헐 실패의 원인을 잡았다(2026-09-05).** 앞서 신원 대조 문제로 적었는데, 측정해 보니 아니었다. 실패한 판에서 프로듀서에게 직접 물으니 `self.name="producer"`, `self.addr=0xf959…` 로 자신을 제대로 찾고 있었다. `self.miner=false` 는 원인이 아니라 클러스터가 비어 있어서 생긴 결과였다. 진짜 원인은 두 가지다. 첫째, 노드는 거버넌스 계약을 읽고 나서야 자기가 어느 멤버인지 알고, 그 전에는 `admin.etcdInit()` 이 `ErrNotRunning` 으로 거절한다. 둘째, 그 거절을 콘솔이 글자로 찍을 뿐 프로세스는 성공으로 끝나는데 `poa.EtcdInit` 이 출력을 안 읽었다. 형제 함수 `EtcdJoin` 은 F5b 때 같은 대가를 치르고 이미 읽고 있었다. 그래서 클러스터가 만들어지지 않았는데도 아무 데서도 오류가 나지 않았다. 고친 것은 `poa.WaitSelf`(자신을 알아볼 때까지 대기, 두 경로 모두 배선)와 `EtcdInit` 의 출력 검사다. **덤으로 요구사항 ⑩의 미배선을 찾았다** — 컴포지션 경로는 A3 이후 포트를 검사하는데 핸드오프는 안 했다. 남은 노드가 프로듀서 p2p 포트를 잡고 있자 "IPC 가 30초 안에 안 생겼다"로만 보고하고 포트도 범인도 말하지 않았다(3연속 실패). `Handoff.checkVacant` 로 배선하고, 어느 노드를 띄우는지 판정하는 규칙을 `selected` 하나로 합쳤다(검사와 기동이 어긋나면 안 되므로). **확인**: 최종 트리로 `TestUpgradeRunE2E` 연속 통과 | ☑ |
| G3 | genesis 산출물의 by-product(`Extra`) 배치 규칙 정리 — 지금은 소비자마다 따로 쓴다 | | ☐ |

### R — 로컬 docker 를 원격 서버처럼 (원격 경로 검증)

> 근거: [[docker-remote-design]](docker-remote-design.md). 실 원격 서버 없이 Rancher 의
> ubuntu 컨테이너를 가상 서버로 쓴다. 인벤토리는 실주소를 유지하고, 하네스가 접속하는
> 최하위 4곳(dialSSH + RPC 조립 3곳)에서만 `AddrMap` 이 loopback 퍼블리시 포트로 치환.
> 선례: `~/Work/github/packages/wemix-bp-test` 의 LocalMap (동작 검증 완료).

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **R1** | `AddrMap` boundary — **`--docker` 옵션이 전원**(파일 존재는 활성화 아님, §3.2a) + 매핑 파일(gitignore) 로드 + 접속 경계 주입 + 적용 보고 | ☑ **완료 2026-08-24.** `remote.AddrMap` 을 `target.resolveOver`(SSH 두 형태의 단일 수렴점)와 netcompose 의 `resolveTarget`/헬스 프로브에 주입. 옵션 없으면 파일 있어도 항등(라이브: 실주소 다이얼 후 timeout 실증) · 옵션+파일 부재는 오류(패키지·CLI 회귀) · `net` 은 `State.Docker` 영속 · 적용 내역 보고("docker: dialing 172.30.0.11:22 as 127.0.0.1:2201") · CLI/MCP 동일 옵션 | ☑ |
| R2 | docker 가상 서버 생성 스크립트 — compose + 인벤토리 v2 + localmap 자동 생성 | ☑ **완료 2026-08-24** (`env/docker/gen-env.sh`). 15대 기동, 생성된 인벤토리를 `net pool --fleet` 이 15×1=15 로 읽음. 퍼블리시 포트는 127.0.0.1 바인딩. server15 는 pn 예정(역할은 할당이 정하므로 서버 계층 구분 없음). **같은 날 접근 모델 교체(DR-b 해소)**: 실서버가 id+password 이고 sudo 가 그 비밀번호를 요구한다는 운영 사실에 맞춰, 키 로그인을 없애고 사용자 `chainbench`+비밀번호(+비밀번호 요구 sudo)로 재구성. `remote.ExecWithInput` + `driver.SSHSudoRunner`(sudo -S -k, 비밀번호는 stdin) 신설 — NM-e 의 "운반만 되던 sudo" 가 소비 가능해짐. 라이브: password 로그인·sudo whoami=root·root 전용 쓰기·keyring 원격 2종 전부 통과 | ☑ |
| R3 | keyring 원격 경로 라이브 — `import --from srv://` | ☑ **완료 2026-08-24.** server1 의 nodekey 를 `srv://server1/...` + `--docker` 로 가져와 주소·공개키 파생, 번역 보고 출력. 서버로의 쓰기 방향은 R4 의 provision 이 담당(keyring 에 서버 쓰기 verb 없음 — 의도). **후속(같은 날): 검증을 상설 테스트로** — 게이트된 라이브 스위트(`Live_Keyring*`, `CHAINBENCH_DOCKER_SERVERS=<build>` + 함대만 있으면 한 명령) 가 raw key·암호화 keystore(+password) 원격 가져오기와 주소 왕복 동일성을 검증. 이 스위트가 **실결함 1건을 즉시 잡음**: `user@host:path` 형태는 포트 미지정(0)이 매핑 뒤에 22 로 기본화되어 변환표를 지나침 → 매핑 전 기본화(`mapCredentials`, 함대 불필요 단위 회귀 동반). **니모닉 가져오기를 CLI/MCP 에 노출**(core 에만 있던 갭): 골든 벡터(dev mnemonic → 0xf39F…) + 출처 배타성 테스트. 실측: `Ring.Install` 은 소비자 0 (배송은 provision 소관 — 정리 후보) | ☑ |
| R4 | 원격 조립·기동 라이브 (단일 서버) | ☑ **완료 2026-08-24.** `net up --server server1 --docker` 로 stablenet 4검증자가 **docker 서버 위에서** 15스텝 완주, 블록 16→24 전진(매핑 포트로 프로브), stop 후 고아 0. **P2 실증**: genesis·static-nodes·workspace 전부 실주소(172.30.0.11), loopback 0건(metrics 자기 바인드 기본값 제외). **원격 경로의 선재 결함 4건을 이 과정에서 발견·수정**: ① init 이 타깃 경로를 로컬 `os.ReadFile` 로 읽음 → Files boundary 경유 ② netcompose 에 신원 배송 부재 + config/argv 가 로컬 키 경로를 타깃에 구움 → `keysBase()` + provision 의 `shipIdentities`(engine 방식 이식) ③ **원격 launch 셸 문법** — `mkdir && nohup CMD &` 는 리스트 전체가 백그라운드 서브셸이 되어 세션 파이프를 문 채 노드를 기다림(노드가 즉사할 때만 우연히 통과) → `|| exit 1; nohup … &` + 문법 회귀 테스트 ④ 헬스 프로브가 fleet 에서 노드별 주소 대신 target 주소를 물음 → 노드 기록 주소로 | ☑ |
| R5 | **fleet 다중 호스트 기동** — 노드별 머신 해석 완성: allocate 가 노드마다 서버 세트 항목명을 기록하고(`NodeState.Server`), genesis·config·provision(신원 배송)·init·start·stop·restart·logs·사전 점검(포트 프로브·바이너리 검사)이 전부 **그 노드의 머신**으로 간다(`machineFor`/`eachMachine`, 명령당 머신별 1회 dial 캐시) | ☑ 라이브(2026-08-26): 5대 서버에 4 검증자+1 endpoint 분산 기동 → 서로 다른 머신끼리 합의해 블록 26 봉인 → 대장에 5머신 5기록 → stop → 5대 전부 고아 0. 단위: fleet allocate 가 서버명·호스트 분산을 기록 | ☑ |

| **R6** | **poa(wemix) 원격 실행 + 직렬 브링업** — poa 는 genesis 를 바이너리 실행으로 만들고 기동 뒤 거버넌스·etcd 부트스트랩도 바이너리를 실행하는데, 둘 다 하네스 로컬 `os/exec` 였다. `resource.Access`(Files+Driver.Commander)로 genesis 생성과 부트스트랩을 **타깃에서** 실행(노드별 머신 해석). 부트 노드를 최상위 producer 로 하고, 나머지 producer 를 내림차순으로 **한 대씩 시작+join**(직렬)하여 joiner 의 fork 경합(late-join 노드가 mid-reorg 구간을 "unauthorized block"으로 거부)을 제거 | ☑ 라이브(2026-09-02): 15대 docker 서버셋에 14 bp + 1 en 웹믹스가 원격에서 genesis 생성·거버넌스·etcd 부트스트랩 완주, etcd 14멤버 형성, sealer 로테이션 확인. `run` 에 `--docker` 추가. 단, **아래 잔여 버그로 재현 신뢰성은 불완전** | ◐ |

**R6 잔여 버그 (2026-09-02, 추후 디버깅) — go-wemix etcd 붕괴.** 직렬 브링업으로 fork 경합은
사라졌으나, 14 bp 규모의 느린 브리지에서 **부트 노드가 형성한 etcd 클러스터가 가끔 붕괴**한다.
`etcdInit` 으로 형성(VerifyEtcd 통과)된 뒤, 노드의 배경 etcd 관리가 거버넌스 멤버 목록(14개)을
보고 그 "클러스터"에 join 하려다 실패하며 형성된 클러스터를 잃는다:

```
etcd join failed name=node14 error="not found"
etcd failed to start: cannot fetch cluster info from peer urls
```

증상: 부트 노드 `admin.wemixInfo.etcd.cluster = undefined`, 부트가 거버넌스 블록에서 정지,
후속 join 이 전부 "not found". 재현 2회 중 1회 완전 성공(14멤버·로테이션), 1회 부트 etcd 붕괴.
chainbench 오케스트레이션(genesis·부트스트랩·직렬 순서)은 정상이고, **go-wemix 바이너리의 etcd
형성·안정성** 영역이다. 조사 방향: (a) 부트 노드가 거버넌스 멤버 전체를 아는 상태로 단일 클러스터를
안정 유지하는 조건(초기 클러스터를 자기 자신만으로 고정, 나머지는 add-member 순서), (b) etcd 붕괴
감지 후 재형성(`etcdInit` 재발행)을 브링업에 넣을지 — 단 형성 중 재발행이 오히려 방해한 선례 있음.
키·genesis 는 원인이 아님을 실측 확인(genesis 해시 전 노드 동일, 배포된 키 경로를 command 가 사용).

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
poa+pn 은 거부한다 — `ConsensusFamily.SupportsRole` 신설(구현 2곳뿐이라 값싼 boundary).
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

## 1h. 아키텍처 v2 — 모듈 재편 (사용자 결정 2026-08-25)

> 결정 요지. **CLI 는 core 를 직접 호출**한다(기능 동작 확인 용도). **MCP 는 app 을
> 경유**하며, app 은 워크플로 층이다(DSL 파싱 → chainsetup → testengine → 수집 → 레포트).
> **netmap** 이 서버 정보 관리·자원 분배(ip·port)·enode 생성(공개키는 입력)·low level
> 접근 wrapper(유일 통로)를 소유한다. low level(machine·process·FileStore·driver·SSH)은
> 필요 정보를 전부 파라미터로 받는 순수 기능으로 남는다. 경계마다 **소비자 측 작은
> interface** 로만 노출한다. 상세 설계는 V0.1 에서 architecture 문서로 기록한다.

**모듈 네이밍 규칙** (새 모듈·개명 시 이 표로 판정):

1. 소문자 한 덩어리 — 하이픈·언더스코어·대문자 금지, 폴더명 = 패키지명
2. 소유하는 명사로 짓는다 — 동작이 역할이면 "대상+동작명사" 합성 최대 2단어 (`chainsetup`)
3. 덤핑 단어 금지 — util·common·helpers·shared·misc·base·manager (관리자는 타입으로: `process.Manager`)
4. stutter 금지 — `machine.Spec` ○ / `machine.Machine` ✗ (현 `target.Target` 이 위반 사례)
5. 단수형 — 집합이 주제일 때만 집합 명사 (`serverset` ○, 현 `accounts` 위반)
6. 계층은 경로가 말한다 — 이름에 core·app 접두어 금지
7. 약어는 업계 표준만 — rpc·ssh·mcp·dsl ○ / `netreg` ✗

**빌드 안전 순서 원칙**: 모든 이동은 신설(추가만) → 소비자 전환 → 구코드 제거의
세 단계로 하고, 태스크 하나 = PR 하나 = 빌드·테스트·lint 통과 상태다. 개명은 한 PR
안에서 원자적으로 끝낸다(별칭 잔류 금지). 선행 열이 비면 착수 가능.

| # | 작업 | 선행 | 게이트 | 상태 |
|---|---|---|---|---|
| **V0.1** | 아키텍처 v2 결정 기록 — 레이어 그림·모듈 책임·CLI/MCP 비대칭·네이밍 규칙을 architecture 문서로 | — | 문서 등급 표기(현행 설계) + docs/README 권위 순서 반영 | ☑ |
| **V0.2** | AST 전수 측정 — app·netcompose·engine·target·driver 함수별 이동표(현 위치 → 목표 칸) | — | 이동표가 V1~V6 각 태스크의 대상 파일을 명시 | ☑ (8패키지 541심볼, [[v2-move-map]](archive/v2-move-map.md)) |
| **V1.1** | `core/target` → `core/machine` 개명 — `machine.Spec`/`machine.Access`(stutter 해소), 소비자 일괄 전환 | V0.2 | 한 PR 원자 개명 · 전 소비자 컴파일 · 기존 테스트 무변경 통과 | ☑ |
| **V1.2** | 무분기 감사 — machine 소비자의 local/remote 분기 전수 검사, 분기는 machine 내부로 | V1.1 | 제거: app keyring 로컬 지름길(해석기로 단일화)·netcompose 구조 검증(`Spec.Validate` 신설로 이동). 잔여 분기는 래칫 테스트가 유예 목록으로 고정(V2.2·V2.3·V5 에서 소멸, 축소만 허용) — `internal/arch` TestMachineConsumersDoNotBranchOnKind | ☑ |
| **V2.1** | netmap 접근 wrapper 신설 — 서버 이름 → 능력 손잡이(FileStore·Driver), `--docker` 치환·치환 보고·자격 결합 내장. 추가만, 기존 코드 무변경 | V1.1 | 단위: 치환·보고·자격이 wrapper 한 곳에서 재현 | ☑ (`netmap.Opener`) |
| **V2.2** | keyring 소비 전환 — `RingRef.open` 의 개별 배선을 wrapper 호출로 교체 | V2.1 | keyring 라이브 스위트(원격 링·복제) 통과 | ☑ (keyflags 분기도 소멸, 래칫 축소) |
| **V2.3** | netcompose 소비 전환 — `resolveTarget` 을 wrapper 로 교체(서버 세트 nil 전달 결함 구조적 해소) | V2.1 | 원격 net 단계가 env 변수 없이 서버 세트만으로 동작(라이브) | ☑ (health 프로브도 `Opener.AddrMap` 경유) |
| **V2.4** | serverset 흡수 — netmap 이 서버 정보 관리를 소유(패키지 이동 또는 내부화) | V2.2, V2.3 | 외부에서 serverset 직접 import 0건 — `internal/netmap/internal/serverset` 로 컴파일러가 강제. `ResolveServer` 도 app 에서 모듈로 이동(app 은 별칭만) | ☑ |
| **V2.5** | enode 생성 이관 — netcompose 의 enode·static-nodes 조합을 netmap 으로(공개키는 입력 파라미터) | V2.3 | 기존 enode 골든 값 바이트 동일 (`netmap.Enode`·`PeerList`, 골든 테스트 이동) | ☑ |
| **V3.1** | `keyring/store` 분리 — 링 저장·읽기(레이아웃·metadata·암호화 파일)를 하위 패키지로, 키 역학은 keyring 에 잔류 | V0.2 | 한 PR 내 소비자 전환 · 전 테스트 통과 | ☑ (17개 소비 파일 전환, 저장 테스트 동반 이동) |
| **V3.2** | 링 위치 해석 이동 — `--keyring-dir` 우선순위(플래그>env>기본)를 store 로 | V3.1 | 위치 보고(`keyring: <dir> (<source>)`) 동작 유지 | ☑ (`store.Locate`) |
| **V3.3** | app keyring 슬림화 + CLI 직접 호출 — keyringcmd 가 core(store·netmap wrapper)를 직접 호출, app 은 MCP 용 얇은 함수만 | V2.2, V3.2 | CLI 전 명령 동작 동일(테스트) · app keyring 은 호출만 | ☑ (동사 조립은 `internal/keyring` 모듈로 — CLI 직접 호출, app 은 별칭+위임 27줄) |
| **V4.1** | driver 조회 보강 — 머신에서 바이너리/pid 실행 여부·포트 사용·명령 수행(결과 회수) 원시 기능 채움 | V1.1 | 단위 + docker 라이브(양면 프로브 유지) | ☑ (`ProcessInspector`·`Commander` 양쪽 드라이버. 원격 pid 는 `/proc` — 비특권 로그인에서 kill -0 의 EPERM 이 부재로 읽히는 함정 회피) |
| **V4.2** | `core/process` 신설 — `process.Manager` 실행 대장: 어떤 머신·어떤 바이너리·어떤 명령·pid | V4.1 | 단위: 기록·조회·이중 기동 감지 | ☑ (기존 procman 흡수·개명. 영속 `Ledger` 신설: Record 가 이중 기동을 두 pid 명시로 거부, 재열람 왕복 고정) |
| **V4.3** | pid 기록 전환 — netcompose 워크스페이스의 pid 관리를 process 대장으로 | V4.2 | start→stop→고아 0 라이브 재검증 | ☑ (`process.json` 이 정본, NodeState.PID 는 열람 뷰로 동기화. 라이브: 4노드 기동→블록 26→stop→서버 고아 0·대장 0건) |
| **V5.1** | chainsetup 수렴 — netcompose 의 순차 진행을 chainsetup 으로 이동, 단계 내용은 기능 모듈로 분리. 역할 정의: 체인을 구성해 블록 생성 상태까지 | V2.3, V4.3 | 기존 net up 전 단계 라이브 통과 · netcompose 잔여 코드 0 | ☑ (패키지 해체 완료 — 레거시 사례 러너는 T7.11 은퇴까지 동거, Step→CaseStep. 단계 내용의 기능 모듈 분리는 이미 경계에 있는 것 유지) |
| **V5.2** | 실행 기록 폴더 — 지정 폴더 아래 실행마다 폴더: 체인 id·입력 사본·배치표·genesis·실행 명령. **서버 세트 ssh 절 제외**(테스트로 고정) | V5.1 | 기록에서 자격증명 grep 0건 테스트 | ☑ (`runs/<stamp>/` — manifest·genesis 회수·launch-commands. 카나리 비밀번호 테스트로 값 비노출 고정; "password" 단어는 argv 의 파일 경로로 정당) |
| **V5.3** | 사전 점검 배선 — 구성 전 process 대장으로 기동 중 노드 검사, 케이스별 함수 분리·조립(전체 셋업·부분 재시작·점검만) | V5.1 | 이미 도는 노드 위 재구성 거부 라이브 | ☑ (포트 충돌은 init 의 기존 점검, 포트가 비어도 같은 바이너리가 돌면 start 가 pid 지목 거부 — `checkUnmanaged`+`Preflight`(점검만 진입점). 라이브 양쪽 재현) |
| **V5.4** | CLI `netcmd` 추출 — net 그룹 6파일을 패키지로, chainsetup 직접 호출 (serverFlags 중복 해소 포함) | V5.1 | keyringcmd 패턴 준수 · 도움말 무손실 | ☑ (선행으로 net·network·hardfork 동사 23개를 chainsetup 모듈로 이동(app 은 위임만), 테스트 동반 이동. 서버 선택 플래그는 `cmd/chainbench/internal/serverflag` 하나로 — netcmd·netmapcmd·run 공용) |
| **V6.1** | engine → `testengine` — 구성 책임 제거, "구성된 체인 위에서 테스트만 일관 수행" 으로 축소·개명 | V5.1 | 기존 테스트 스위트 결과 동일 | ☑ (구성 파일 10개(빌드환경·genesis·keysource·launcher·plan·nodecontrol·wemix 계열)와 테스트가 chainsetup 으로, 러너는 setup_bridge 한 파일로 위탁 — 의존 방향 chainsetup→engine 이 testengine→chainsetup 으로 역전, 순환 0) |
| **V6.2** | app 워크플로 — DSL 파싱 → chainsetup → testengine → 수집 → 레포트를 app 이 한 흐름으로 제공 | V6.1 | e2e: DSL 입력 하나로 셋업+테스트+레포트 산출 | ☑ (`app.RunSuite`: testspec.ReadFiles(신설, CLI 도 공용) → NetUp → 봉인 대기(WaitBlocks) → attach 실행 → 수집 → 자동 해체. 라이브 e2e: DSL 1건 → server1 4노드 → 1 pass → 고아 0) |
| **V6.3** | MCP 전환 — MCP 도구가 app 워크플로·얇은 app 함수만 경유(CLI 는 core 직접 유지) | V6.2 | run 도구가 app 경유로 전환(`AttachRun`/`SessionSummary`). 잔여 직결 import 14종은 래칫 테스트가 축소 전용 목록으로 고정(각 항목이 소멸 후속을 명시) — `internal/arch` TestMCPGoesThroughApp | ☑ (전면 0건은 후속 축소로) |
| **V7** | 기회 개명 백로그 | 해당 트랙 | 네이밍 규칙 표 판정 통과 | ☑ **완료 2026-09-07 — 둘의 판정이 갈렸다.** `netreg`(규칙 7)은 **모듈이 이미 없다** — R1 이 `core/session` 으로 흡수했고, 남은 것은 파일 이름 셋이었다. 규칙 표는 모듈을 판정하지만 약어는 읽는 사람이 마주치는 모든 이름에 해당하므로 `networks.go` 로 바꿨다(소비자 0, 기계적).

`accounts`(규칙 5)는 **개명하지 않는다.** 이 패키지는 상류 SDK `github.com/0xmhha/accounts` 위의 경계이고, 이름이 그 SDK 를 가리켜서 찾기 쉽다. 규칙 5 의 목적은 복수형이 "실은 하나"라는 사실을 감추는 것을 막는 일인데, 여기서 주제는 계정 하나도 계정 집합도 아니라 **SDK 경계**다. 단수 `account` 로 바꾸면 그 대응이 끊기고(읽는 사람이 "accounts SDK 를 어떻게 쓰나"를 찾을 곳이 사라진다) 소비자 27곳이 바뀐다. 규칙이 막으려는 해악이 없는 자리에서 규칙의 글자만 맞추는 거래다 |

각 태스크 마무리마다: 해당 경계에 소비자 측 interface 수립 · 네이밍 규칙 판정 ·
빌드·테스트·lint·(원격이면) docker 라이브 게이트.

## 1i. 모듈 재편 — 관심사 단위 (사용자 결정 2026-08-27)

> 설계·근거·이동표는 [[module-plan]](architecture/module-plan.md). 실측은
> `go run ./scripts/inventory/code-graph -symbols .` 이며, **단계마다 재측정으로 열고
> 재측정으로 닫는다**(같은 문서 §1).
>
> 부서 셋으로 나눈다. **자원(`resource`)이 정하고, 노드(`node`)가 기록하고,
> 프로세스(`process`)가 돌린다.** 그 위에 빌더 셋(genesis · nodeconfig · dsl),
> 그 위에 표면(cmd · mcp), 그 위에 오케스트레이션(chainsetup · testengine).
>
> **표는 실행 순서다**(위에서 아래로). P5(표면)가 P2 앞에 오는 것은 2026-08-28
> 결정 — P1 이 `netmap` 모듈을 없앤 뒤에도 표면이 그 이름을 부르고 있어, 이름과
> 경계를 먼저 맞춘다. P5 는 기록 구조를 건드리지 않는다(그건 P2 의 것).
>
> 이름은 **부서가 소유한 대상 명사**로 짓는다. 은유(`netmap`)·약어(`netreg`·
> `portplan`)·산출물(`place`)·동작(`alloc`)은 간판이 되지 못한다. `alloc` 은 이
> 저장소에서 이미 genesis 계정 배분을 뜻하므로 쓸 수 없다(실측 13건).

| # | 작업 | 선행 | 게이트 | 상태 |
|---|---|---|---|---|
| **P1.1** | **어휘·지도·경로·enode → `core/node`** — label(35)·role(36)·map(114)·peering(159)·layout(38)·enode(32) 약 380줄 이동. `Ports` 별칭 제거하고 `node.Endpoints` 로 통일 | — | `node` out-edge **0 유지**(L0) · 어휘만 쓰던 6곳(poa·wbft·nodeconfig·session·topology·testspec)의 `core/netmap` import 소멸 · `core/netmap` 에 pool.go 만 잔류 · 기존 테스트 무변경 통과 | ☑ **완료 2026-08-27** — `node` out-edge 0 · 어휘 6곳 import 소멸(엣지 268→260) · `core/netmap` 은 2파일 191줄만 잔류 · `NodeLabel`→`node.Label` stutter 해소 · 61패키지 테스트 통과 |
| **P1.2** | **`serverset` 승격 + `Opener` 합류 → `internal/resource`** — `netmap/internal/serverset`(1,028)와 `netmap` 표면(229)을 한 패키지로. 봉인 목적은 wrapper 가 같은 패키지에 들어오면서 유지. **서버 쪽 `Placement` 삭제**: `Source`·`DataRoot`·`Remote` 세 필드가 전부 같은 반환값(`Pool.Source`·`Target.DataRoot`·`Spec.IsRemote()`)에 이미 있는 사본이라, `ResolveServer` 가 `Pool` 과 `machine.Spec` 을 따로 돌려준다. **`fleet` 낱말 제거**(제품 용어는 server set 하나): `--fleet`→`--all-servers` · MCP `"fleet"`→`"all_servers"` · `ServerRef.Fleet`→`.All` · `fleetTarget`→`setTarget` · `CHAINBENCH_DOCKER_SERVERS`→`CHAINBENCH_DOCKER_SERVERS` · `ServersBuildDir`→`ServersBuildDir` · 주석의 은유까지. **`Config.Fleet()` 삭제 → `Config.Pool()` 로 통합**(둘이 같은 일을 하고 `Pool()` 은 프로덕션 호출자 0). 슬롯 나눗셈(`slots/len(hosts)`)은 제거 — **착수 후 실측 결과 도달 불가능한 죽은 산술**이었다(v2 는 풀 하나에 slots 하나를 선언하고 `expand()` 가 모든 호스트에 복사하므로 같은 값 N개를 N으로 나눈 값이다). 호스트별 슬롯은 형식 변경이라 별도 결정으로 분리 | P1.1 | `serverset → core/netmap` 엣지 소멸(동일 패키지) · 서버 쪽 `Placement` 심볼 0 · **`fleet` 문자열 0**(주석·테스트·스크립트 포함) · `--server-set` 과 `--docker` 라이브 경로 동작 동일 · keyring 원격 스위트(링 생성·복제) 통과 | ☑ **완료 2026-08-27** — `internal/netmap` 소멸(패키지 75→74, 엣지 260→258) · 서버측 `Placement` 심볼 0(`ResolveServerOut{Pool, Target, HasTarget}`) · `fleet` 문자열 0(코드·스크립트·샘플) · `resource` 5파일 1,176줄, 소비자 6 · 계층 위반 0 · 61패키지 통과 |
| **P1.3** | **풀·배정·포트밴드 → `resource`, `place` 흡수** — pool(145)+portplan(184) 이동, `place.NodeReq` → `node.LaunchReq`(착수 후 정정: 기동 요청이라 배정 요청과 다르다), `Ports` 별칭 2개 제거하고 `node.Endpoints` 로 통일 | P1.2 | 패키지 **4개 소멸**(`core/netmap`·`netmap`·`core/portplan`·`core/place`) · `resource → node` 단방향 · 계층 위반 0·  `net allocate`·`netmap plan` 산출 바이트 동일 | ☑ **완료 2026-08-28** — 패키지 4개 소멸(75→71, 엣지 268→246) · `resource → node`·machine·remote 만 · `node` out-edge 0 · `Bands` 중복 2→1(`plan()` 소멸) · `Reservation` 은 노드 사실이라 `core/node` 로(패밀리·registry 가 resource 를 import 하지 않음) · 골든 동일 · 61패키지·-race·lint 0 |
| **P1.4** | **슬롯 상한 검사** (결정 2026-08-28) — `Pool.Validate()` 가 선언한 슬롯 수만큼 `PlanBands` 를 실제로 돌려 `ValidatePorts` 로 충돌 검사. "가용 포트 수를 넘는 slots" 를 선언 시점에 거부(지금은 밴드에 끝이 없어 `slots: 1000` 도 통과, `pool.go` 주석은 하지 않는 검사를 한다고 적혀 있음). 형식 변경 없음. *(옛 P1.4? "호스트별 슬롯" 은 폐기 — 밴드가 세트 공통이라 포트가 허용하는 노드 수는 모든 호스트에서 같다. 개념이 성립하지 않음, 사용자 확인 2026-08-28)* | P1.3 | 초과 slots 선언이 `Validate` 에서 거부됨 · 기존 유효 세트는 전부 통과 | ☑ **완료 2026-08-28** — `Pool.Validate()` 가 선언 슬롯 전부를 `PlanBands`→`ValidatePorts` 로 검사. p2p 8500/10 + rpc 8600 에서 slots 11 이 "11 slot(s) exceed … port 8600" 으로 거부, 10 은 통과(테스트) |
| **P1.5** | **`resource.Inventory` 신설** (결정 2026-08-28) — 서버 세트 소유 모듈이 가용/할당을 관리: 메모리 인스턴스(`Open`·`Adopt`·`Take`·`Release`·`Usage`·`Full`), ip·port 가 필요한 곳은 여기서 할당받는다. **메모리가 정본**(파일 영속·복구는 최종 항목 F1). **반납 = `rm`**(`stop` 은 pid 만 제거, 자원은 노드 것으로 유지 — pid 는 재기동 시 갈린다). `Full` 오류는 누가 쥐고 있는지까지 출력. `net pool` 의 `Used` 가 워크스페이스 하나만 세던 것이 이것으로 고쳐짐 | P1.4 | `Usage` 가 같은 세트의 모든 네트워크를 계수 · `Take` 초과 시 `ErrFull` 에 보유자 목록 · 단위 테스트 | ☑ **완료 2026-08-28** — `resource.Inventory`(`NewInventory`·`Adopt`·`Take`·`Release`·`Usage`·`Full`·`ErrFull`) 메모리 인스턴스. `Adopt` 은 워크스페이스 기록(`chainsetup.Allocations`)에서 파생, host+p2p 포트로 슬롯 역산, 세트 밖·밴드 밖 기록은 무시, 중복 claim 은 먼저 것 유지. `app.NetPool` 이 명명된 워크스페이스 + `~/.chainbench` 아래 전부(`chainsetup.Discover`)를 채택해 `Used/Free/ByNetwork` 답변, `resource pool` 이 보유자 출력. 기본 워크스페이스 경로 소유를 `chainsetup.DefaultWorkspaceDir` 로 이동. **라이브 확인: 같은 세트의 두 조립이 같은 포트를 받아 인벤토리가 첫 것만 셈 → P2.x 의 근거** |
| **P5** | **표면 재정리 (P2 앞으로 당김, 결정 2026-08-28)** — ① `netmapcmd` 해체: `netmap` 모듈이 P1 에서 사라졌는데 표면이 그 이름을 부르고 있다. `netmap pool`(자원)·`plan`(배정 미리보기)은 resource 의 그룹으로, `show`(배치 조회)는 node 쪽으로, keyring 방식(모듈=그룹=MCP 묶음) 적용. 구체 모양은 착수 시 제안. ② **플래그 분리**(사용자 결정 2026-08-28): `--workspace-dir` = 셋업 정보를 생성하는 경로(+ 기본 경로 규칙, `~/.chainbench/<날짜시간>/<테스트명>/chainsetup` 방향) / `--data-dir` = 노드가 블록 데이터를 쌓는 디렉터리(geth 계열 `--datadir` 과 같은 뜻으로 통일). ③ CLI 는 모듈 직접·MCP 는 app 경유, 래칫 14종 축소. **P2 와의 선긋기: 이름과 경계만, 기록 구조는 건드리지 않는다** | P1.5 | 표면에서 `netmap` 문자열 0(재편으로 정한 이름만) · `--data-dir` 이 워크스페이스를 뜻하는 곳 0 · 래칫 14→한 자릿수 | ☑ **완료 2026-08-28** — `netmapcmd` 소멸: `resource pool·plan`(신설 `resourcecmd`) / `net show`(netcmd 로) / MCP `chainbench_resource_pool`·`_resource_plan`·`_net_show` · 공용 렌더러 `internal/mapview`(plan 과 show 가 같은 표) · 현행 경로 9곳 `--data-dir`→`--workspace-dir`(MCP 인자 `dataDir`→`workspaceDir`), 레거시 14곳은 T7.11 은퇴까지 유지 · `net new/up` 기본 경로 `~/.chainbench/<타임스탬프>/chainsetup`(생략 시 첫 줄에 경로 보고) · 표면에서 `netmap` 문자열 0 · 61패키지·race·lint 0 |
| **P2** | 노드 사실 레코드 — "노드 하나" 타입 10 → 3, 경로 계산 4곳 → 1 | P5 | 심볼 인벤토리로 계수 확인. `workspace.json` 의 생성 주체·이유와 워크스페이스 삼중 정의(chainsetup.Workspace·session.Composition·Environment)의 소유 확정 포함 | ☑ **완료 2026-08-28** — `node.Record` 신설(옛 `chainsetup.NodeState`, JSON 계약 유지) · 노드 타입 10→3(Record·driver.NodeSpec·node.Node), 나머지는 정확한 이름으로(`collector.Sample`·`chainsetup.Probe`·`topology.Entry`) · `node.Layout` 에 Nodekey/Keystore/StaticNodes/IPC 경로 추가, 데이터플레인 손조립 0 · 워크스페이스 삼중 정의 해소: `workspace.json` 의 주체는 `session.Composition`, `chainsetup.Workspace` 는 그 위의 도메인 상태, `session.Environment` 는 다른 수명(아티팩트) · `node` out-edge 0 유지 · 61패키지·race·lint 0 |
| **P2.x** | **배정이 Inventory 를 소비** — `Assign` 이 `Inventory` 에서 빈 슬롯을 받아 시작. 같은 세트로 올린 두 번째 네트워크가 첫 네트워크의 포트를 다시 받지 않게 됨(지금은 둘 다 슬롯 1부터 시작해 충돌) | P2 | 같은 세트 위 2개 네트워크 동시 기동 라이브 | ☑ **완료 2026-08-28** — `Inventory.Assign(reqs, network)` 이 빈 슬롯을 `Take` 해 배치(`place`). `chainsetup.Inventory(pool, self, named…)` 한 곳에서 인벤토리를 조립(기본 루트 전부 + 명명, 자기 자신 제외)하고 allocate·plan·pool 이 같은 claim 집합을 읽는다. `resource.Assign(pool)` 은 빈 인벤토리 위의 계획으로 남아 골든 바이트 동일. **라이브**: 같은 세트 두 조립이 31000/31010 과 31020/31030 을 받고, 반쯤 찬 세트의 `resource plan` 이 31040 부터 시작 |
| **P3** | 프로세스 — 실행은 `driver`, 정책은 `process`. 기동 진입점 8 → **3**(driver 실행 · `launcher` 기동 정책 · `process` 종료 정책 — 사용자 결정 2026-08-28: 옛 `supervisor` 는 sudo 역할로 읽혀 `launcher` 로). **`occupancy` → `inspector` 개명·확장**(사용자 결정 2026-08-28): ip 가용·port 가용·경로 유효성(data root·datadir·genesis·keystore·nodekey·config·log·binary)을 요청 시에만 실사해 사실만 답한다. 타입은 stutter 회피(`inspector.Report` 등), `driver.ProcessInspector` 어휘도 이때 정리 | P2 | 진입점 계수 · `chainsetup` 714줄 감소 | ☑ **P3 완료 2026-08-28** · P3.1 — `core/launcher` = `supervisor`+`chainsetup.LocalLauncher`(→`Direct`)+`driver/lifecycle.go`, `supervisor` 낱말 0, `driver.NodeOf` 로 Node 조립 4→1(hardfork 의 loopback 리터럴 소멸), `workspace.go` 의 etcd 포트 누락 복사 수정, `chainsetup` 6,593→6,256. **P3.2 완료 2026-08-28** — `NodeController`→`launcher.Controller`(pid 맵 삭제: arming 만 기억, pid 는 `session.Environment` 노드표 한 곳 — `NodeControl.Stop/Start` 가 갱신된 노드를 돌려주고 fault 액션이 `Env.UpdateNode` 로 써넣음), `chainsetup` 6,256→6,123. `Workspace.Start/Stop/Restart` 는 노드별 머신을 거쳐 driver 를 부르고 `Record.PID` 에 적으므로 그대로(조립 모드의 한 기록). `LocalSetup` 은 레거시 `setup --launch`/MCP `_start` 경로라 T7.11 과 함께. **P3.3 완료 2026-08-28** — `core/occupancy`→`core/inspector`: `Ports`(옛 Scan) · `Paths`(file seam 으로 타깃에서 존재 확인) · `Hosts`(도달) 세 질문, 요청 시에만, 사실만. `net start` 가 기동 전 binary·genesis·datadir·config 존재를 각 노드의 머신에서 확인해 빠진 것을 이름으로 보고. `occupancy` 문자열 0 |
| **P4** | 빌더 셋 — genesis 생성 지점 5 → 1, config 렌더 2 → 1, dsl 파서가 액션을 import 하지 않음 | P2, P3 | 계수 + import 방향 | ☑ **P4 완료 2026-08-28** · P4.1 genesis 완료 2026-08-28** — `core/genesis` 가 소스 선택(`SourceFor`, 패밀리 id 분기 0: `SourceProvider` 타입 단언)·`Compose`·프리셋 소스 소유, wemix 소스는 `consensus/poa.GenesisSource`(Family 가 capability 구현), 호출자 5곳 전부 `Compose`/주입 `Source` 경유, `chainsetup` 직접 파일 쓰기 0(layers §5 에서 제외), 5,828줄. **P4.2 config 완료 2026-08-28** — `nodeconfig.Spec` 단일 입력, `TOML`/`Argv` 두 렌더러, Spec 조립은 `launcher.NodeConfig` 한 곳, compose 의 config·launchopts·start 가 `peerPlan` 으로 같은 입력을 모음, argv 조립 3→1(`upgrade.LaunchArgs`·deploy 평평한 `LaunchArgs` 도 `Argv` 경유), `driverSpec` 의 `SyncMode` 누락 수정. `launchopt` 는 소유로 편입(호출자는 `nodeconfig.Argv` 뿐), 디렉터리 유지. **P4.3 dsl 완료 2026-08-28** — `testspec`(문법·해석기 1,432줄)과 `internal/testhelper`(액션·어세션·리더 2,129줄) 분리. `Registry` 에 `Reader` 추가로 문법이 액션 파일을 부르던 고리(`readerFor`) 제거, `NewRegistry()` 빈 레지스트리 + `testhelper.Register`. `testspec→testhelper` import 0 |
| **P4.x** | **`preflight` 재정의** (결정 2026-08-28) — 계획 자기모순 검사에서 **현재 vs 목표 비교**로: 타깃의 현 체인 구성을 분석하고, 정상 동작 여부를 inspector 로 확인하고, 다음 테스트가 요구하는 구성과 비교해 "그대로 사용 / N번 서버만 재구성 / 전체 재구성" 을 답한다(연속 테스트의 재구성 비용 제거 — 설정 파일 비교만으로는 부족: 파일이 같아도 노드가 비정상일 수 있다). 기존 계획 검사는 각 빌더로(포트→resource, genesis 포크→genesis 빌더, netid→config 빌더) | P4, P3 | 동일 구성 연속 테스트에서 재구성 스킵 라이브 · 부분 변경 시 해당 노드만 재구성 | ☑ **완료 2026-08-28** — `core/preflight` = `Have`/`Want`/`Compare`/`Check`/`Decision`(reuse·rebuild-nodes·rebuild-all·compose, 이유 포함), 의존 `node` 뿐 · `chainsetup.Workspace.Have/Compare` + liveness(pid 는 노드의 머신, RPC head 는 노드 주소) · `app.RunSuite` 가 NetUp 전에 물어 reuse 는 건너뛰고 rebuild-nodes 는 `NetRestart` 만 · 옛 계획 검사는 `upgrade.NetworkPlan.validate` 로(빌더 함수 호출) · 표 테스트 9 + liveness 3 |
| **P6** | `chainsetup`·`testengine` — 남는 것은 순서뿐. 6,593 → 2,000줄 이하 | P5 | `setup_bridge.go` 소멸 · `testengine→chainsetup` 엣지 소멸 | ☒ **닫는다 2026-09-08 — 줄 수 목표는 근거를 잃었고, 남은 게이트는 좁히기로 달성되지 않는다.**\n\n**실측**: `chainsetup` 4,652줄 · `testengine` 2,349줄, 합 7,001. 그중 **주석이 1,547줄(22%)** 이고 공백 451, 실제 코드는 5,003 이다. 함수·메서드가 213개라 평균 20줄 남짓 — 비대한 함수가 몰린 구조가 아니다. 가장 큰 것이 `netUpFrom` 133 · `RunSuite` 132 · `Run` 109 · `Allocate` 99 다.\n\n**목표 2,000 은 P5 시점의 6,593 에서 잡은 것인데, 그 뒤 이 두 패키지가 하는 일이 늘었다**(청사진 N1~N6, N9 순서 선언, NM6, 원격 경로). 줄이 는 것은 기능이 는 것이지 부풀어서가 아니다. 그리고 줄 수를 목표로 삼으면 **가장 먼저 지워지는 것이 그 22%의 주석**인데, 이 세션에서 판단을 바꾼 것이 바로 그 주석들이었다(`derive quorum` 이 노드 표를 안 읽는 이유, `genesis` 가 키셋을 요구하지 않는 이유).\n\n**게이트 둘 중 하나는 충족**이다 — `setup_bridge.go` 는 없다. 남은 `testengine→chainsetup` 엣지는 비테스트 4파일이고, **층 위반이 아니다**(둘 다 L4). 노드 수명(start/stop/restart/swap)을 인터페이스로 좁혀 보면 **4파일 중 1개**(`nodegate.go`)만 import 가 사라지고 `suite.go` 에는 `NetUp`·`Open`·`WantOf`·`NetworkStatus`·`NetStop`·`NetEndpoints` 등 11개가 남는다 — **패키지 엣지는 그대로다.** 인터페이스가 12개 메서드라면 그것은 요구를 좁힌 것이 아니라 chainsetup 을 다른 이름으로 부르는 것이다.\n\n**엣지를 실제로 없애는 길은 하나뿐이다**: 조립을 testengine 밖으로 내보내 엔진이 망을 *받게* 하는 것. 그것은 좁히기가 아니라 재설계이고, 근거가 생기면 그때 별도 항목으로 세운다. 지금은 열어 둘 이유가 없다 |
| **P7** | DSL 케이스 4종 — go-wemix · wemix→wbft · wbft 단독 · stablenet | P6 | 러너에 `if chain ==` 0건 | ☑ **완료 2026-08-28** — 당시 `tests/cases/env/` 선언 4개 + 케이스 4개(지금은 `tests/tc/`, env 는 정의서 인라인) · 문법 `env.upgrade`(schema·strict·lowering, `binaries` 는 producer/validator 역할) · 실행기 `app.RunSuite` 가 선언의 모양으로 조립기 선택(`upgrade` → `upgrade.Handoff`, 아니면 `NetUp`) · 표면 `run --workspace-dir` · `validate` 가 env 를 풀고 env 파일을 선언으로 검증 · 러너 `if chain ==` 0건 · **라이브 4/4 완결 2026-08-31**(gstable·go-wbft·go-wemix 빌드; 핸드오프는 후계 검증자의 블록 21 봉인까지) — 그 과정에서 gwemix 0.10.x 의 `etcd.members` 모양 변화가 깨뜨린 verify 파싱을 `EtcdState` 필드 제거로 해소 |
| **P8** | `test-helper` — 액션 1,541줄 + testkit + tests 공통부 취합 | P7 | 파서가 액션을 모르고 액션이 문법을 모른다 | ☑ **완료 2026-09-07 (#357).** 남은 것은 문법 갭이 아니라 **낡은 기록**이었다 — 막혔다던 19건 중 15건에 이미 스펙이 있었고 122개 전부 `validate` 를 통과한다. 문서를 고치고 `TestSpecDoc_BlockedCasesHaveNoSpec` 이 그 주장을 검사하게 했다. 진짜 남은 8건은 이유가 유효하다(SDK 가드 2 · 조작자 공급 키 2 · 다른 빌드가 필요한 4) |
| **F1(최종)** | **파일 영속·복구 시스템** (사용자 결정 2026-08-28: 모든 작업의 맨 마지막) — chainbench 프로세스 장애로 중단됐을 때 재실행하여 이전 진행 상황을 복구하고 서버 상태를 재확인. `Inventory` 등 메모리 정본의 파일 저장이 이때 들어온다. 그 전까지는 기존 기록에서 `Adopt` 으로 파생(사본 금지 원칙) | P8 | **설계안 2026-08-28**: `docs/dev/architecture/f1-recovery.md` — 요청 기록(`workspace.json.request`) · `net resume`(잠금 인수 → 생사 대조 → 첫 미완 단계부터 → 재확인) · 세트 잠금(인벤토리 파일 없음) · 주인 없는 프로세스 입양. §4 는 제안대로 결정 → ☑ **완료 2026-08-28**: `State.Request` 기록 · `net resume`(reconcile → 첫 미완 단계부터 → 죽은 노드 재기동) · `session.AcquireLock` + 세트 잠금(`~/.chainbench/<set>.lock`) · 우리 argv 프로세스 입양 · 단위 6건 + gstable 라이브(kill -9 → resume) | ☑ |

### 1j. 사용자 주도 통폐합 (2026-08-31 확정 — 정본: `docs/dev/architecture/consolidation-plan.md`)

§1i 의 표는 F1 까지 전부 끝났다. 이후는 **사용자가 주도**한다. 확정된 방향:
모듈을 관심사 단위로 통폐합(internal 55 → 약 20, R1~R5), 그다음 표면 재정리
(`net` 폐기 → `chain` 그룹, 구성 6단계 사전, CLI↔DSL 1:1). testengine 은
① chainsetup 수행 ② pre hook ③ test(interpreter 내장) ④ post hook 의 4단계가
되고, testengine → chainsetup 의존은 의도적으로 부활한다(P6.1 게이트 대체).
아래의 예전 후보 목록은 이 계획 안으로 흡수됐다(34건 이관 = R5, verbs 질문 =
표면 재정리에서 함께).

**R1 완료 (2026-08-31, PR #325 — 9 relocation, internal 46).** 소형 흡수 8건 계획 중,
측정된 층 그래프와 대조해 arch-안전한 것만 실행했다: topology→node · launchopt·config→nodeconfig ·
netid→resource · consensus·capability→registry · obs·logs→collector · netreg→session. 상세·근거는
정본(`consolidation-plan.md` §R1 실행 결과). **다음은 R2(DSL 분리).**

**R1 에서 갈라져 나온 후속 작업 (별도 트랙):**

- ☐ **validatorset 홈 결정** — `core/node`(L0)로 넣으려던 계획은 층 위반(validatorset 이
  `chains/all`·`registry` import). 지금은 제자리(독립 L3)에 둔다. 로스터 계산(키+registry→검증자)의
  올바른 소유 모듈을 후속에서 정한다. 소비자는 `cmd` 하나뿐.
- ☐ **health 를 inspector 조합 레이어로 재배선** — health→inspector 흡수는 하지 않기로 결정.
  방향: `inspector` 는 atomic 실사 프리미티브(L1, "판단 없음")로 두고, `health`(블록 전진 *판정*)는
  그 atomic 들을 **조합하는 inspector 위 레이어**로 제공한다(현재 health 는 obs/rpc 를 직접 쓴다 →
  inspector 프리미티브를 쓰도록 재배선). atomic ↔ 조합의 층 분리.
- ☑ **hardfork 는 통폐합 대상 아님**(결정으로 닫힘) — `hardfork`(바이너리 swap)와 `consensus/upgrade`(합의-패밀리
  handoff)는 의도적으로 다른 모델이라 별개로 유지. R1 표의 `hardfork→genesis` 항목은 폐기.

이전 기록 (2026-08-31 이전 후보):

1. **레거시 34건 이관 + 스택 은퇴** — `tests/specs/README.md` 잔여 표의 문법 갭을
   묶음별로(자산·부정기대·fee-delegation 0x16·EIP-7702·비동기 제출·토폴로지 파생
   quorum·delayed-boho 크로스오버) 확장하고, 끝나면 `testkit`·`testrun`·`test` 명령·
   MCP `chainbench_test` 를 한 번에 은퇴(T7.11 완결). 이제 네 갈래가 전부 라이브로
   돌므로 이관분의 라이브 확인도 곧바로 된다.
2. **열린 질문 7** — `verbs_*.go`(1,384줄)의 자리. `app` 이전이면 `chainsetup` 이
   2,000줄 아래로 가지만 "CLI 는 core 직접" 원칙과 충돌 — 사용자 결정.
3. **열린 질문 2·5** — `session`(1,594줄) 경계 재측정, `netreg`(161줄, 소비자 mcp
   하나) 개명 또는 흡수. 소품.


## 1k. 체인 실행·테스트 증적 완결 (사용자 확인 2026-09-02)

> 제품 방향은 [[chainbench-system-direction]](chainbench-system-direction.md), 현재 코드와의 차이와
> 완료 조건은 [[refactoring-follow-up-handoff-2026-09-02]](refactoring-follow-up-handoff-2026-09-02.md)를
> 따른다. 이 절만 작업 순서와 상태를 소유한다. 기존 완료 기능을 다시 만들지 말고 각 항목의 첫 단계에서
> 코드·테스트·CLI를 재측정해 남은 차이만 구현한다.

| # | 작업 | 선행 | 핵심 게이트 | 상태 |
|---|---|---|---|---|
| **E0** | **현행 재측정과 계약 고정** — key/resource/node/genesis/config/process/monitor/test/report의 producer·consumer, CLI·DSL·MCP 입력, local/remote/Docker 경로를 AST와 테스트로 재확인 | — | 중복 owner·직접 import·기존 완료 기능 목록, 수정 대상과 비대상 파일 확정 | ☑ |
| **E0A** | **session artifact 계약 선행** — `environments/<env-id>` 정본 + `tests/<NNN>_<name>`별 env-ref·사용자료 snapshot/reference + root report의 2축 schema와 owner 고정 | E0 | testengine=verdict, session=영속, report builder=집계 · secret 정책 · 원자 write/readback | ☑ |
| **E1** | **resolved producer → genesis** — validator count 기반 첫 N개 재선택을 제거하고 실제 BP identity/address/BLS를 사용 | E0A | `EN,BP,PN,BP` topology genesis readback 일치 · 누락/중복 no-write · 결정성 | ☑ |
| **E2** | **자료 재사용 무결성** — asset별 owner를 유지하며 local/SSH/Docker의 binary·genesis·config·key·contract를 checksum으로 비교하고 결과 기록 | E0A | 같은 내용 재전송 0 · 같은 이름/다른 내용 오재사용 0 · secret 출력 0 | ☑ |
| **E3** | **config·command 증적** — `nodeconfig` Command Builder, `config-<test-purpose>` fixture, override 우선순위, config readback, 노드별 argv·binary·checksum 기록 | E0A | node override 격리 · config/argv 동일 사실 · 변경 전후 revision 보존 | ☑ |
| **E4** | **process와 개별 node control 정합** — Direct/Launcher/chainsetup start 중복을 측정하고 PID·실행 command·start/stop/restart·binary/config 교체를 하나의 ledger에 연결 | E2, E3 | 개별/전체 제어 · 실제 PID/command 일치 · 부분 실패 cleanup · orphan 0 · 세 환경 동등 | ☑ |
| **E5** | **DSL syntax + semantic/capability 사전 검사** — schema/parser drift, selector/wait timeout, PN 제약과 chain·binary별 role/flag/RPC/metric/action/assertion/upgrade 지원을 모든 write 전에 검증 | E0A | unsupported는 no-write · applicableChains는 SKIP · CLI/MCP 판정 동등 | ☑ |
| **E6** | **`nodemonitor` 실행 허가** — inspector/preflight/health/collector를 복제하지 않고 READY/WAITABLE/RESTARTABLE/FATAL로 조합, `MaxNodeMonitorTimeout` 적용 | E1~E5 | 재사용 전·각 테스트 전 gate · 제한 재시작 · 파괴적 자동 복구 0 · 판정 증적 | ☑ **stage a**(`internal/nodemonitor` L4): `Verdict`·`Classify`(순수, worst-first)·`Gate`(관측→분류→제한복구 루프, `Observer`/`Restarter`/`Clock`/`EvidenceSink` 주입). `process.FailureMode` 재사용, 파괴 상태(chainId 불일치·fork·EtcdStale·QuorumLost)=FATAL 자동복구 0. **stage b**(testengine 배선): `health.Run`+recorded pid→Facts(순수 `factsFromReport`), `chainsetup.NetRestart` 재시작, `composeWorkspace`가 compose/reuse 후 1회·`PreSpec` 훅이 각 테스트 전 게이트. 관측·재시작은 재구현 없이 기존 함수 재사용(사용자 결정). 고아 `process.BringUp` 상한재시도는 모양이 달라 미재사용→1k-Z2 이월. **라이브 검증(GSTABLE_BIN) 이월** — CI는 순수 매핑·게이트·엔진루프 단위테스트로 green | ☑ |
| **E7** | **동적 테스트와 contract** — 노드별 제어, partition/heal, binary/config 교체, 동적 `save/$ref` 주소와 deployer+nonce 결정적 주소 | E4, E5, E6 | per-node mixed binary · contract tx/receipt/address/checksum · 상태별 verdict | ☑ 기존(fault·partition·save/$ref·deploy·call)은 재구현 없이 조합. **신규**: ① `createAddress` 리더 — `accounts.CreateAddress`(SDK `tx.CreateAddress` 래핑, canonical 벡터 검증), nonce 생략 시 현재 nonce → deployContract 예상=실제 assert. ② `swapNode` action — 노드별 binary/config 교체(`chainsetup.NodeSwap`, 같은 datadir/genesis, **E4 recordSwap로 revision 보존**, config는 `writeNodeConfig` 공유·config-`<purpose>` fixture provenance). `interp.NodeSwapper` 선택 인터페이스(코어 미변경). ③ `contractChecksum` 리더 — bytecode/runtime code sha256(`filestore.Hash` 재사용). 기대 revert=PASS 등 verdict는 기존(`checkTxOutcome`). **swapNode config launch·라이브 시나리오는 사용자 환경 이월**, CI는 매핑·플러밍·revision·config render 단위테스트로 green | ☑ |
| **E8** | **최종 증적 집계와 report** — E0A schema에 append-only logs, chainstate JSONL, assertion provenance, 테스트별 result를 채우고 root report 생성 | E0A, E3, E6, E7 | 실패 자료 수집 · remote reconnect · 전체 종료 후 report · secret 원문 0 | ☑ 기존(session schema·`AssertResult.Provenance`·report.Build/Summary·chainstate_sink)은 확장. **E8-1** `session.Scrub` — 증적 write seam(`writeJSON`+`Spec`)에서 키·비밀번호 마스킹, hash/주소는 보존. workspace.json(기능)은 제외. **E8-2** FAIL/BLOCKED 시 엔진 `OnFail` 훅이 `collectFailureData`로 health.Run(RPC/peer/block)·process 레저(pid/command)·노드 로그 tail을 테스트별 `observations/`에 수집(`TestRecord.Observation`, 스크럽 적용). **E8-3** report가 observations/*를 링크, RunSuite가 bus로 chainstate.jsonl을 headless 경로에서도 기록. **E8-4** `collector.ReconnectingLogReader`(백오프 재시도, 항상 적용) + `resource.Access.Runner` 노출 + `chainsetup.NetRunner` + RunSuite가 remote면 `RemoteLogReader` 주입. **원격 SSH read·chainstate 라이브는 사용자 환경 이월**, CI는 스크럽·OnFail·tail·reconnect·report 링크 단위테스트로 green | ☑ |
| **E9** | **표면·환경 동등성** — CLI 단계, DSL 자동 구성, MCP 도구와 local/remote/Docker simulation을 같은 시나리오로 검증 | E1~E8 | 의미·기본값·오류·결과 동등 · 일반 mixed-binary와 `consensus/upgrade` handoff 구분 | ☑ **E9-1** MCP↔CLI GAP 해소: `chainbench_run`이 rpc 없으면 compose(`app.RunSuite`, `RunSuiteIn.SpecContent`로 인라인 스펙)·있으면 attach. `chainbench_hardfork`→`app.HardforkPlan/Execute`(execute 기본 false). `chainbench_upgrade`→신설 `app.UpgradeRun`(handoff orchestration을 `upgrade.Handoff.Run`으로 추출, CLI `upgrade run`과 공유). **E9-2** 구조적 parity: 두 표면이 미설정 시 zero-value를 넘겨 `compositionOf`가 정본 기본값 단일 소스(validators 4·keys/preset 고정), run 모드 선택 검증. mixed-binary vs handoff 구분·컴포즈 기본값/선택/가드는 기존 `compose_internal_test`가 커버. 세 표면은 arch(`TestMCPGoesThroughApp`)가 app 경유로 고정. **행위적 CLI-vs-MCP diff는 cmd=main이라 불가**, local/remote/Docker 실제 실행은 라이브 이월(환경 분기 단일 지점 arch 고정) | ☑ |

E1·E2·E3·E5는 E0A에서 저장 계약과 수정 파일을 분리한 뒤 병렬 진행할 수 있다. E4는 process와 chainsetup,
E6는 inspector/preflight/health/collector, E7은 DSL/testhelper/upgrade, E8은 session/collector/report
영역과 충돌하므로 해당 owner 작업과 동시에 수정하지 않는다.

각 작업은 코드 변경 전에 입력·출력 타입, owner, 허용 import, 외부 상태 변경 시점과 rollback을 검토받는다.
구현 후에는 단위·거부·readback 테스트, CLI JSON 예시, local 검증과 가능한 remote/Docker 라이브 결과,
남은 wrapper·alias·직접 import를 보고한다.

### 1k-Z. 레거시 네이밍 정리 (추후, E-series 이후 · 비차단) — ☑ 완료

**완료(2026-09-03):** R1에서 흡수된 파일이 달고 있던 leftover `// Package <옛>` doc 주석
6곳(netreg→session, probe/logs→collector, config/launchopt→nodeconfig, event.go의 중복
`Package collector`)을 일반 섹션 주석으로 바꿔 패키지마다 정본 doc 하나만 남겼다. 폐기된
옛-패키지 에러 접두 정리: netreg `state:`→`session:`, run store `obs:`→`collector:`. 살아있는
하위 개념 접두 `logs:`(로그 검색)·`launchopt:`(런치 옵션)는 유지(폐기 패키지명이 아니라 실제
개념). 테스트가 이 문자열을 검사하지 않고 errors.Is 무영향 → 동작·공개 계약 변화 0. 남은 옛-이름
언급은 의도적 provenance 주석("Formerly the standalone X package…")뿐. build·test·arch·lint 통과.

아래는 착수 전 기록:

모듈 통폐합(R1) 과정에서 흡수된 파일·식별자가 **옛 이름을 그대로 달고 있다.** 동작에는
문제 없으나 "어디를 봐야 하는가"를 흐린다. E-series를 막지 않는 **순수 개명 리팩토링**으로
뒤에 한 번에 정리한다(파일·심볼 rename + 주석의 "옛 core/X" 표기 정돈, 동작 변화 0).

정리 대상(측정으로 확정):

- `internal/core/session/netreg.go` — `core/netreg` 가 session 으로 합쳐졌는데 파일명·에러
  접두("state: …")·주석이 옛 `core/state` 어휘를 유지(E0A 에서 관찰). session 어휘로 통일.
- `core/collector` 안의 "옛 `core/obs`"·"옛 `core/logs`", `core/nodeconfig` 안의 "옛
  `core/launchopt`"·"옛 `core/config`" 등 §3 표가 나열한 흡수 이력 표기 — 코드 심볼은 이미
  새 위치이나 파일/주석이 옛 이름을 인용하는 곳을 정돈.
- 판정: **파일 이동/개명만으로 완료로 보지 않는다** — 에러 접두·doc 주석·테스트 이름까지
  새 owner 어휘로 맞춘 뒤 완료.

게이트: `go build`·`go test`·arch 테스트 통과, 동작·공개 계약 변화 0, 남은 옛-이름 인용 목록 보고.

### 1k-Z2. E4 에서 미룬 항목 (추후, 비차단)

E4(launch+record 통일 · 스왑 revision 보존)에서 근거를 대고 미룬 둘.

- **핸드오프 레저 기록** — `consensus/upgrade` 핸드오프는 workspace 가 없어(compose.go
  핸드오프 경로) 영속 레저(process.json)를 읽는 소비자가 없다. teardown 은 레저가 아니라
  반환 NodeSet 으로 `StopNodeSet` 을 부른다. `LaunchAndRecord` 는 nil 레저를 받아 이미
  통과한다. **핸드오프가 workspace 를 얻는 작업과 함께** LaunchOptions 에 선택적 Ledger 를
  넣고 teardown 에서 Clear 한다. 그 전까지는 SIGKILL 엣지의 orphan 복구만을 위해 배선하지
  않는다.
- ☑ **죽은 process 오케스트레이션 은퇴 (완료)** — `Launcher`(bringup.go: impl/NewLauncher/
  BringUp/gate/Teardown/Deps/Result)·`Controller`(controller.go)·`Manager`(process.go)·
  `Direct`(direct.go)·`PlanOf`(launcher_plan.go)·`gate.go`(JoinGap/JoinWindow/Classify)·
  `StopNode`/`RelaunchNode`·`Proc.IsRemote` 를 그 테스트와 함께 제거(~3200줄). 생존 프리미티브만
  남김: `FailureMode`(→failuremode.go, nodemonitor 소비)·`NodeConfig`(→nodeconfig.go, chainsetup
  소비)·`StopNodeSet`(핸드오프 teardown)·`Proc`/`Alive`(레저·inspector)·`Plan`(데이터 캐리어).
  no-branch-on-kind 래칫에서 process 항목 제거. build·vet·test·lint 통과, 동작 변화 0.

## 1l. 표면 단일 진입 (사용자 결정 2026-09-05)

> **결정**: CLI·MCP·DSL 은 모두 `app` 을 통해 기능에 닿는다. 세 표면이 같은 층을 지나야
> 같은 경험을 준다. `app` 이 커지더라도 규칙이 하나인 편이 낫다는 판단이다.
> 이 결정은 [[architecture-v2]](architecture/architecture-v2.md) §2 의 비대칭 규칙("CLI 는 core
> 직접, MCP 는 app 경유", 2026-08-25)을 대체한다. S 절의 **S5·S6 은 폐기**한다. 게이트가
> "cmd 는 app 만 import" 였는데, 지키려던 성질은 import 목록이 아니라 표면 사이의 동등성이다.
>
> **근거가 된 실측(2026-09-05, AST 전수)**: 세 표면의 등록 지점 **157개**(CLI 58 · MCP 54 ·
> DSL 45 = 액션 18 + 어서션 27)를 파싱해 실제로 닿는 패키지를 따라갔다. app 을 지나지 않고
> 그 아래에 닿는 항목이 **109개**다(CLI 40 · MCP 24 · DSL 45). app 을 한 번도 부르지 않는
> 항목만 세면 102개다.
>
> 표면 사이 비교는 이름이 겹쳐 자동으로 묶으면 틀린다(`new` 는 `chain` 밑에도 `keyring`
> 밑에도 있고, `run` 은 스위트 실행과 `upgrade run` 둘 다이다). 그래서 **손으로 확인한 32쌍**만
> 쓴다. 그중 **22쌍이 서로 다른 경로로 간다.** 양쪽이 모두 app 을 지나는 것은 `chain show`·
> `hardfork`·`resource plan`·`resource pool` **4쌍뿐이고**, 나머지 6쌍은 양쪽이 똑같이 app 을
> 건너뛴다(경로는 같지만 구현은 각자다).
>
> 갈라진 예를 들면, `report` 는 CLI 가 `core/report`+`core/session` 을 직접 부르는데 MCP 는 app
> 으로 간다. `build`·`config`·`health`·`logs`·`resume`·`status`·`new` 는 CLI 가 `chainsetup` 을
> 직접 부르고 MCP `chain_*` 는 전부 app 이다. keyring 다섯 동사는 CLI 가
> `core/keyring/operation` 을 부르고 MCP 는 app 을 부르는데, app 이 그 모듈을 얇게 감싼 것이라
> 구현은 같고 경로만 다르다. `send`·`deploy`·`faucet` 은 CLI·MCP·DSL 세 곳이 각각 조립한다.
>
> **선례**: Helm 은 `pkg/action` 에 동사마다 액션을 하나씩 두고 CLI 와 SDK 가 같은 것을 부른다
> (`pkg/action/doc.go`: "Actions approximately match the command line invocations that the Helm
> client uses"). CLI 는 `cmd/helm` 이 아니라 `pkg/cmd` 에 있고, `cmd/helm/helm.go` 는 50줄짜리
> `package main` 으로 `helmcmd.NewRootCmd` 만 부른다. U1 이 따르려는 모양이 이것이다.
>
> 반대 방향의 증거도 같은 저장소에 있다. 차트 의존성을 갱신하는 `downloader.Manager` 는
> `pkg/cmd` 의 다섯 파일(`install`·`upgrade`·`package`·`dependency_update`·`dependency_build`)에
> 나오고 **`pkg/action` 에는 한 번도 나오지 않는다.** 옵션 `DependencyUpdate` 는 액션이
> 선언하는데 그 동작은 CLI 가 구현한다. `action.Install` 을 직접 쓰는 소비자가 그 옵션을 켜도
> 아무 일도 일어나지 않는다는 뜻이다. **표면이 층 위에 앉는 것을 허용하면 갈라진다.**
>
> Docker 는 CLI 가 엔진에 닿는 통로를 API 클라이언트 하나로 두었다(`cli/command/**` 의 각
> 동사가 `dockerCLI.Client()` 로 시작한다). 데몬이 별도 프로세스라 우회할 방법 자체가 없다.
>
> **규칙**: 기능 하나에 `app` 진입점 하나를 둔다. `core` 모듈은 기구를 갖고, `app` 은 동사를
> 갖고, 표면은 바인딩과 렌더링만 한다. 표면은 `app` 을 건너뛰지 않는다.
> **게이트는 import 규칙이 아니라 동등성 테스트다.** 같은 입력에 두 표면이 같은 결과를 내는
> 것을 기능마다 증명한다. 그러려면 CLI 가 import 가능해야 하므로 U1 이 앞에 온다.

| # | 작업 | 선행 | 핵심 게이트 | 상태 |
|---|---|---|---|---|
| **U0** | **규칙 확정과 측정 고정** | — | 도구가 157 항목·우회 109 를 재현 · 라체트가 표면별 상한을 고정 · 동등성 테스트 본보기 1건 · 문서 사이 모순 0 | ☑ **2026-09-05.** 문서 쪽은 #349 에서 끝냈다(`architecture-v2` §2 개정, S5·S6 폐기, 표면 통일 설계에 대체 표시). 코드 쪽은 이번에 붙였다. 인벤토리 걷기를 `internal/arch/surface.go` 로 옮겨 **도구와 테스트가 같은 코드를 센다**(두 벌이면 숫자가 갈라진다). `TestSurfacesReachThroughApp` 이 표면별 예산(CLI 40 · MCP 24 · DSL 18 · DSLa 27)을 **양방향으로** 잡는다. 넘으면 빚이 늘었다고 막고, 밑돌면 작업하고도 천장을 안 내렸다고 막는다. 실제로 41 로 올리고 39 로 내려 둘 다 실패하는 것을 확인했다. **동등성 테스트 본보기**는 `cmd/chainbench/resourcecmd/parity_test.go` 의 `TestParity_ResourcePool` 이다. `resource pool` 은 CLI 와 MCP 가 이미 둘 다 `app.NetPool` 을 부르는 네 쌍 중 하나라 지금 통과한다. 렌더링이 아니라 **답**을 비교한다(양쪽 JSON 을 값으로 디코드). MCP 쪽에 `res.Slots++` 를 심어 실제로 어긋남을 잡는 것을 확인하고 되돌렸다 |
| **U1** | **CLI 를 import 가능한 패키지로** (U0 의 본보기가 `resourcecmd` 에서 돌아가는 것은 그 패키지가 이미 import 가능하기 때문이다. `package main` 에 남은 32개는 같은 테스트를 쓸 수 없다.) — `package main` 안에 cobra 명령 생성자가 **32개** 있어 그 명령들은 테스트에서 부를 수 없다. 표면 패키지로 옮기고 `package main` 은 배선만 남긴다(Helm 이 `cmd/helm` 을 `pkg/cmd` 로 옮긴 것과 같은 이유). 동등성 테스트가 가능해지는 전제다 | U0 | `package main` 의 명령 생성자 32 → 0 · 옮긴 그룹마다 표면 테스트 1건 이상 · E9 가 "cmd=main 이라 불가"로 이월한 CLI-vs-MCP diff 가 가능해짐 | ☑ **32 → 1 (2026-09-05).** `package main` 에는 이제 루트 하나만 남았다(149줄, `main.go`·`root.go`·`interrupt.go`). Helm 이 `cmd/helm/helm.go` 를 50줄 배선으로 두고 명령을 `pkg/cmd` 에 옮긴 것과 같은 모양이다. 표면 패키지는 여덟 개가 늘었다. `txcmd`(tx·contract) · `accountcmd`(account·faucet) · `catalogcmd`(chains·capabilities) · `nodecmd` · `lifecyclecmd`(status·stop·clean·verify·consensus) · `suitecmd`(run·validate·migrate-spec) · `reportcmd`(report·log) · `upgradecmd`(upgrade·genesis·run·hardfork). 그룹이 곧 패키지라는 기존 관례(`keyringcmd`·`chaincmd`·`resourcecmd`)를 그대로 따랐다.

**이동은 순수하다**: 라체트 109 불변, 명령 트리 불변.

**공유 심볼 둘을 정리했다.** `exitError` 는 `suitecmd` 가 만들고 `main` 이 읽으므로 표면과 프로세스 사이의 계약이다. `cmd/chainbench/exitcode` 패키지로 뺐다. `obsBus` 는 전역 `dashboardURL` 을 읽고 있었는데, `--dashboard` 는 루트의 persistent 플래그라 하위 명령이 상속한다. 명령에서 직접 읽게 바꿔 표면 패키지에서 공유 가변 상태를 없앴다.

**앞서는 쓸 수 없던 테스트 30건**이 붙었다. 그게 이 항목의 요점이다. 동등성 4건(`account state`·`contract call`·`chains`·`node rpc`)은 표면이 **무엇을 인쇄했는지가 아니라 체인에 무엇을 물었는지**를 비교한다. CLI 는 사람에게, MCP 는 프로그램에게 쓰므로 글자를 맞추면 서식을 고정하게 된다. 가짜 노드가 RPC 호출을 기록하고 두 목록을 견준다. 변이를 심어 실제로 잡는 것을 확인했다. 나머지는 거절 경로와 종료 코드 계약이다.

**e2e 도 실제 마운트 지점을 거치게 바꿨다.** 잎사귀 생성자를 직접 부르면 그 명령이 루트에 제대로 붙어 있는지는 시험하지 못한다. `newRootCmd()` 에 `upgrade run` 을 붙여 부른다.

덤으로 `resolve.go` 의 `remoteDriver` 를 걷어냈다. `remote` 명령군 폐기(#346) 이후 자기 테스트 말고 부르는 데가 없었다. |
| **U2** | **조합 계열 이관** — `chaincmd` 의 명령 19개가 `chainsetup` 을 직접 불렀다 | U1 | 기능별 CLI==MCP 동등성 테스트 · 라체트 감소 · `net up` 3체인 회귀 | ☑ **2026-09-05. CLI 40 → 31.** `chaincmd` 가 `chainsetup`·`resource`·`core/node` 를 직접 부르던 것을 전부 `app` 경유로 돌렸다. app 이 이미 대부분을 얇게 감싸고 있어서(`NetStatus` 는 `chainsetupmod.NetStatus` 로 넘기는 한 줄) 빠진 진입점만 채웠다. `NetEnodes` · `DefaultWorkspaceDir` · `State` · `NodeSet` · `TargetSpec`/`ParseTarget`.

**기본 워크스페이스를 app 으로 올린 것이 이 항목의 알맹이다.** 두 표면이 각자 계산하면 한쪽이 다른 쪽이 못 찾는 곳에 구성한다. `app.DefaultWorkspaceDir` 하나만 쓰게 했다.

**동등성 테스트 3건.** 인쇄한 글자가 아니라 **디스크에 남은 워크스페이스**를 비교한다. 구성은 디스크에 대한 부수효과이고 그 상태가 곧 답이다. 워크스페이스 경로와 시각만 정규화하고 나머지는 그대로 견준다.

**여기서 제 테스트 결함을 둘 잡았다.** 처음에는 디렉터리를 훑어 "JSON 파일"을 집었는데 알파벳순으로 `process.json`(양쪽 다 `{"procs": []}`)이 걸려서, 빈 문서 둘을 비교하며 통과하고 있었다. `workspace.json` 을 이름으로 지정하고 가드를 실질화했다. status 쪽은 "stablenet" 부분 문자열을 찾았는데 그 낱말이 step 의 `detail` 안에도 있어서, 도구가 chain 필드를 아예 안 내보내도 통과했다. 도구의 JSON 을 파싱해 구조로 견주게 고쳤다. 셋 다 변이를 심어 실제로 잡는 것을 확인했다.

`internal/app/net.go` 머리말의 "CLI 는 모듈을 직접 부르고 여기를 지나지 않는다" 도 고쳤다. U0 에서 뒤집힌 규칙의 잔재였다 |
| **U3** | **keyring 계열 이관** | U1 | 동등성 테스트 · keyring 라이브 스위트 통과 · `operation` 단위 테스트는 app 을 지나지 않음 | ☑ **2026-09-05. CLI 31 → 24.** 표면이 `core/keyring/operation`·`keyring`·`store`·`derive`·`resource` 를 직접 부르던 것을 app 경유로 돌렸다.

**중복 구현을 하나 지운 것이 알맹이다.** `keyflags.go` 가 플래그로 `keyring.Source` 를 조립하고 있었는데, `operation.ImportIn.source` 에 같은 읽기가 이미 있었다. "정확히 하나의 출처"나 "니모닉 없는 hd 옵션"이 무슨 뜻인지를 두 곳이 각자 정하고 있었다는 뜻이다. 모듈 쪽을 `operation.KeyRef`+`ResolveKey` 로 승격하고 표면은 **설명만** 하게 했다. `SaveKey`·`GenerateKey` 도 같은 이유로 모듈로 올렸다. 비밀번호는 값이 아니라 씨앗 함수로 넘긴다. `--password-once` 는 물어보는데, 쓰지도 않을 비밀번호를 미리 묻는 건 사용자를 대하는 좋은 방식이 아니다.

**단위 테스트가 진짜 결함을 잡았다.** CLI 의 `--hd-coin-type` 기본값이 60 이라, 모듈의 "니모닉 없이 hd 옵션을 줬다" 거절에 항상 걸렸다. 플래그 기본값은 사용자가 지정한 것이 아니므로 0 으로 바꾸고 뜻은 도움말에 적었다.

**표면 사이 기능 격차도 찾았다.** MCP `keyring_import` 에 `privateKey` 가 없었다(CLI 에는 `--private-key` 가 있다). app 이 이미 그 필드를 갖고 있어 스키마 한 줄로 닫았다.

동등성 3건(list·show·import). 셋 다 변이로 확인했다 |
| **U4** | **테스트 동사 계열 신설** | U1 | CLI·MCP 동등성 · `mcpImportAllowed` 에서 `accounts`·`core/rpc` 제거 | ☑ **2026-09-05. CLI 24 → 17, MCP 24 → 18.** `send`·`deploy`·`call`·`faucet`·`wait`·`state` 의 유스케이스가 어느 모듈에도 없이 **CLI·MCP·DSL 세 곳에 각각** 쓰여 있었다. `internal/app/chainops.go` 로 모아 CLI 와 MCP 를 그리로 돌렸다. `HexBytes`·`Wei` 도 여기 있다. 한쪽이 `0x` 접두사를 요구하고 다른 쪽이 안 하는 차이는 아무도 테스트할 생각을 못 한다.

`accountcmd/provider.go` 는 소멸했다(`app.ChainRef.provider` 가 같은 일을 한다). MCP 의 `openWalletFromArgs`·`hexBytes`·`weiArg` 도 소멸했다. **라체트에서 `internal/accounts` 가 빠졌다** — "계정 동사 이관과 함께 사라진다"고 예고돼 있던 항목이고, 실제로 그렇게 됐다.

동등성 4건인데 **서명된 원본 트랜잭션을 바이트로 비교**한다. nonce·gas·value·calldata·서명을 한 번에 덮으므로, 여기서 일치하면 두 표면이 `--value` 나 `--data` 를 다르게 읽고 있을 수 없다. 넷 다 변이로 확인했다.

**잔여 없음(정정 2026-09-06).** "DSL 도 옮긴다"고 적었는데 U7 에서 불가로 판정됐다 — `testhelper` 는 L3 이고 app 은 L5 이며, import 순환이다. DSL 액션은 표면이 아니라 언어의 어휘다 |
| **U5** | **실행·보고 계열 이관** | U2 | 동등성 테스트 · `run.go` 에서 흐름 조립 소멸 · 라이브 스위트 회귀 | ☑ **2026-09-05. CLI 17 → 12, MCP 18 → 16.** `run`·`report`·`validate`·`log`·`verify` 를 app 경유로.

**`app.Report` 가 산문을 돌려주고 있었다.** MCP 는 그걸 그대로 냈고 CLI 는 같은 읽기를 다시 해서 표로 그렸다. 렌더링하는 층은 표면이 우회할 수밖에 없는 층이다. 이제 보고서를 돌려주고 표면이 각자 그린다.

**`VerifyNetworkIn` 주석에 "노드 집합 해석은 표면의 몫"이라고 적혀 있었고, 실제로 두 표면이 각자 했다.** 워크스페이스가 어느 엔드포인트를 뜻하는지, 붙인 집합을 뭐라 부르는지를 두 번 정하고 있었다. `app.ResolveNodes` 하나로 합쳤다.

**레이어 검사가 제 실수를 잡았다.** 대시보드 이벤트 배선을 app 에 넣었더니 `L5 app → L6 dashboard` 로 걸렸다. 표면이 다른 표면을 부르는 것은 우회가 아니므로, 배선은 `dashboard.Stream` 으로 표면 층에 두고 라체트도 L6 끼리의 호출은 세지 않게 고쳤다. 안 그러면 옳은 일을 하고 숫자가 나빠진다.

동등성 2건(report·log). 둘 다 변이로 확인했다.

**잔여**: `verify` 는 `dashboard` 만 부른다(같은 층이라 규칙 위반이 아니다) |
| **U6** | **조회 계열 이관** | U2 | 동등성 테스트 · ReadOnly 선언이 세 표면에 동일 노출 | ☑ **2026-09-05. CLI 12 → 0, MCP 15 → 0.** **두 표면 112개 등록이 전부 app 을 지난다.**

CLI 쪽은 `chains`·`capabilities`·`consensus`·`node rpc`·`migrate-spec`·`validator`·`upgrade` 를, MCP 쪽은 `network_*`·`remote_rpc`·`consensus_*`·`txpool`·`node_rpc`·`log`·`chain_new` 을 옮겼다. `app.Chains`·`Capabilities`·`Validators`·`NodeCall`·`ReadNode`·`MigrateSpec`·`DeriveIdentity`·`ValidatorSetOf`·`GenerateSet`·`ResolveBinary`·`UpgradeGenesis`·`AttachNetwork`·`Network(s)`·`DetachNetwork`·`DetectNetwork`·`PeersOf`·`CallOnNode`·`PeersOfNode`·`NodeAt`·`RegisterCapability` 를 신설했다.

**중복 둘이 죽었다.** `resolveBinary` 가 `upgradecmd` 와 app 에 각각 있어서 한쪽이 받는 이름을 다른 쪽이 거절할 수 있었다. 그리고 validator 키에서 무엇을 파생할지(`WithBLS` 인지 `AccountOnly` 인지)를 표면이 정하고 있었는데, 그건 합의 패밀리의 규칙이다.

**저장된 망의 노드에 닿는 법도 하나로 모았다.** 붙여 둔 노드는 SSH 터널이나 docker 매핑 뒤에 있을 수 있고, 거기 닿는 길은 노드 기록의 auth 서술자다. `CallOnNode`·`PeersOfNode` 는 URL 이 아니라 노드를 받는다.

**제가 낸 회귀를 테스트가 잡았다.** `ReadNode` 를 만들면서 네 읽기를 전부 필수로 했는데, 원래는 head 만 필수고 나머지는 최선 노력이었다. net 네임스페이스를 끈 노드가 오류가 돼 버렸다. 계약을 되돌리고 왜 그런지 적었다 |
| **U7** | ~~**DSL 흡수**~~ **전제가 틀렸다 — 폐기 2026-09-05** | U4 | — | ☒ **DSL 액션은 표면이 아니다.** 계획할 때 "액션 18개와 어세션 27개가 app 을 지나지 않는다"고 셌는데, `testhelper` 는 [[layers]] §3 에서 **L3 도메인 서비스**다. `dsl/interp` 의 `Action`/`Assertion` 계약을 구현하는 쪽이고, app 은 L5 다. 아래층이 위층을 부를 수 없다.

**컴파일러로 확인했다.** `testhelper` 에 `app` import 를 넣어 보니 레이어 위반일 뿐 아니라 **import 순환**이다: `app → testengine → … → testhelper`.

**중복의 성격도 달랐다.** U3·U4 에서 찾은 것은 같은 읽기가 두 벌 있는 것이었다. DSL 의 `sendTx` 와 `faucet` 은 라벨을 주소로 풀고, 노드 서명과 로컬 서명을 가르고, 수수료 인자를 적용한다. `app.TxSend`/`Faucet` 보다 하는 일이 많고 메커니즘이 다르다. 같은 구현의 복사본이 아니라 언어의 어휘다.

DSL 에게 표면은 `run` 이고, CLI 와 MCP 두 철자 모두 이미 app 을 지난다.

**라체트도 고쳤다.** 45개를 빚으로 세면 구조상 0 에 닿을 수 없고, 내려갈 수 없는 천장은 다음 사람에게 무시하는 법을 가르친다. 이제 표면만 센다(0/112). 어휘는 세지 않고 보고만 한다 |
| **U8** | **규칙 대칭 마감** | U2~U6 | 우회 항목 0 · 문서와 테스트가 한 규칙만 말함 | ☑ **2026-09-05.** `mcpImportAllowed` 의 마지막 여섯 항목(`core/rpc`·`core/collector`·`resource`·`core/node`·`core/session`·`core/remote`)이 각자 예고한 이관과 함께 사라져 목록이 비었다. 목록은 줄기만 하고 사라진 항목이 남아 있으면 테스트가 실패하므로, **빈 map 은 규칙이 안 걸린 것이 아니라 지켜지고 있다는 뜻이다.** 비대칭 라체트는 이것으로 폐기했고, 표면 규칙은 `TestSurfacesReachThroughApp` 하나가 말한다 |

**측정 기준선(2026-09-05 착수 시점)**: 등록 항목 157 = CLI 58 + MCP 54 + DSL 45(액션 18 + 어서션 27).
app 아래에 닿는 항목 109 = CLI 40 + MCP 24 + DSL 45.
손으로 확인한 32쌍 중 경로 불일치 22, 양쪽 다 app 인 것 4.

**U0~U8 이후(같은 날)**: 표면 등록 **0 / 112** 가 app 을 우회한다(CLI 0 · MCP 0).
DSL 어휘 45개는 L3 이라 세지 않는다 — U7 행에 왜 그런지 적었다.
동등성 테스트 17건이 붙었고 전부 변이로 확인했다.
`go run ./scripts/inventory/surface-graph .` 으로 재현한다.

### 라이브 스위트에서 남은 관찰 (2026-09-05)

`tests/e2e` 를 되살린 뒤 실제 바이너리로 돌린 결과다. 17개 중 **16 통과**, 1 skip, 그리고 간헐 실패 1건.

**`TestE2E_WbftQuorum6of6Halts2` 의 간헐 실패를 쫓다가 별개의 결함을 찾았다.**

실패한 판의 로그를 보니 재기동한 노드에 `Unclean shutdown detected` 와 `Switch sync mode from full sync to snap sync: snap sync incomplete` 가 찍혀 있었다. 노드는 높이 0 에 갇히고 피어 하나만 붙은 채 자기 ROUND-CHANGE 만 듣고 있었다.

**여기서 두 번 틀렸다.** "불결한 종료가 원인"이라고 했는데 통과한 판에도 똑같이 있었다. "snap sync 전환이 원인"으로 고쳤는데 그것도 통과한 판에 있었다. 실패한 판 하나만 보고 원인을 말하면 안 된다는 것을 배웠다. **통과한 판에도 그 표시가 있는지 먼저 봐야 한다.**

그 과정에서 **`process.Stop` 이 문서와 다르게 동작하는 것**을 찾았다. 바로 위 주석과 `doc.go` 가 "SIGTERM, grace, SIGKILL" 이라고 말하는데 코드는 `proc.Kill()`, 즉 유예 없는 SIGKILL 이었다. 데이터베이스를 가진 프로세스에게는 닫을 기회를 줘야 한다. 원격 드라이버는 또 달라서, 같은 `node stop` 이 로컬에서는 즉사이고 원격에서는 신호만 던지고 반환이었다.

고친 뒤 10판: **전부 통과, `unclean=0`, `snap=0`, 재기동 노드가 매번 블록을 받아 따라잡았다**(수정 전 4판은 `unclean` 4/4, `snap` 2/4). 개입으로 인과 사슬의 앞 고리를 끊은 것은 확인했지만, **이 수정이 그 실패를 고쳤다고는 주장하지 않는다** — 수정 전에도 4판 연속 통과한 적이 있어 실패율을 정확히 모른다.

**`TestE2E_StablenetProposalExpiry` 는 되살렸다(2026-09-06).** 이미 구성된 워크스페이스에 capability 로 게이트된 스펙을 돌릴 방법이 없었다. `chainbench test` 는 폐기됐고, `run --workspace-dir` 는 스펙 선언대로 **다시 구성**하고, `run --rpc` 는 붙기는 하지만 워크스페이스를 안 읽어서 게이트가 늘 닫혀 있었다.

`run --workspace-dir <dir> --attach` 를 만들었다. 워크스페이스가 엔드포인트와 **그 구성이 광고한 capability** 를 함께 준다. 둘을 한 번에 읽는 것이 핵심이다. 엔드포인트만 넘기면 게이트가 대조할 것이 없어 스펙이 skip 된다.

MCP `chainbench_run` 에도 `attach` 인자로 같이 넣었다. 한쪽에만 있으면 U 트랙이 없앤 격차가 다시 생긴다.

`AttachRunIn.Caps` 는 원래부터 있었는데 아무도 채우지 않았다. 배선만 없던 것이지 설계가 없던 것은 아니다.

**변이로 확인했다**: `in.Caps` 를 비우면 정확히 예전 증상(skip)으로 돌아간다.

**다만 시나리오는 아직 통과하지 않는다(정정 2026-09-06).** 처음에 1판만 돌리고 "통과"라고 적었는데 성급했다. 3판씩 두 번 재니 **3판 중 2판 실패**다. 고쳐진 것은 **게이트**이고, 실패는 그 안쪽의 별개 문제다 — `sendTx` 가 en1 에 보낸 트랜잭션의 영수증이 시간 초과한다(`context deadline exceeded`). N12 적용 전후로 실패 비율이 같아(각 2/3) N12 탓도 아니다.

skip 이 fail 로 바뀐 것은 후퇴가 아니다. skip 은 아무것도 검증하지 않던 상태이고, fail 은 시나리오가 실제로 무언가를 재고 있다는 뜻이다. 원인은 미상이며 wbft 정족수 회복 건과는 별개다.

### 노드 주소 체계와 라우팅 (2026-09-06)

**두 체계가 같은 집합 위에 겹쳐 있다.** `node1` 은 **순서**(`Index`)로, `en1` 은 **역할**로 정해진다. 역할 어휘는 `bp`(생산자)·`en`(RPC 엔드포인트)·`pn`(중계 계층)이고, 레거시 철자 `validator`/`endpoint` 는 `NormalizeRole` 이 접는다. `parseSelector` → `rolesForToken` → `NormalizeRole` 사슬은 정연하다. `pn` 은 topology 파일로만 생기고 카운트 경로(`--validators/--endpoints`)로는 안 생기는데, `peering.go` 가 그 경우를 막으므로 일관된다.

**어긋난 곳은 배정과 사용 사이였다.** 두 규칙이 각각 옳은데 아무도 둘을 맞춰 보지 않았다.

- `process.NodeConfig` 는 **`bp` 에만** `--unlock` 을 건다. `en`·`pn` 은 어떤 계정으로도 서명하지 못한다.
- `ResolveAccount` 는 **`node<N>` 에 키를 주지 않는다.** 그 노드가 서명한다는 뜻이다.

합치면 **`from: nodeN` 은 `on` 이 같은 노드일 때만 성립한다.** 그런데 `on` 과 `from` 은 독립 선택자이고, `malformedSelectors` 는 `on` 의 문법만 본다.

v1 스펙 45개가 `on: enN, from: nodeN` 으로 쓰여 있었고, 접속 표가 역할을 다 버려 `en1` 이 우연히 node1 로 풀리는 동안만 굴러갔다. 고친 것은 셋이다. ① 서명이 필요한 100곳의 `on` 을 송신 노드로 맞췄다(로컬 서명해 raw 로 보내는 30곳은 어느 노드든 받으므로 `en1` 을 유지). ② `misdirectedSends` 가 오프라인에서 거절한다. ③ 워크스페이스에 붙을 때 역할을 보존해 `en1` 이 실제로 엔드포인트를 가리킨다.

**검증 경로가 둘로 갈라져 있던 것도 찾았다.** `Precheck` 와 `validateRaw` 가 각자 검사해서, `Precheck` 에만 넣은 검사를 `validate` 명령이 못 잡았다. 지금은 둘 다 지난다.

**측정**: `proposal-expiry` 가 3판 중 1판 통과에서 **5판 중 4판 통과**로.

### 미해결 — 체인이 제때 처리하지 않는다 (2026-09-06)

둘 다 원인 미상이고 모양이 닮았다. 함께 보는 편이 나을 수 있다.

**`TestE2E_WbftQuorum6of6Halts2`** — 6노드 중 2개를 멈춰 정족수를 깨는 것은 매번 되고, 되살린 뒤 2분 안에 생산이 재개되지 않는다(5판 중 2판). 쫓다가 `process.Stop` 이 SIGKILL 이던 별개 결함을 찾아 고쳤고 그 뒤 10판 통과했지만, 수정 전에도 4판 연속 통과한 적이 있어 **인과는 증명하지 못했다.**

**`TestE2E_StablenetProposalExpiry`** — 올바른 노드(`node1`)에 보낸 트랜잭션의 영수증이 안 온다(5판 중 1판). 라우팅을 고친 뒤 빈도가 크게 떨어졌으므로 상당 부분은 스펙 문제였고, 남은 것이 원래부터 있던 별개 문제인지 같은 뿌리인지는 모른다.

**쫓을 때의 교훈**: 실패한 판 하나만 보고 원인을 말하면 안 된다. 이 두 건을 쫓으며 가설 세 개가 무너졌는데, 셋 다 **통과한 판에도 같은 표시가 있었다.** 통과·실패를 나란히 놓고 갈리는 것을 찾아야 한다.

### U 트랙에서 배운 것 (2026-09-05)

계획을 세울 때는 몰랐고 하면서 알게 된 것들이다. 다음에 같은 종류의 일을 할 때 쓰라고 적는다.

**중복은 두 종류였고 대응이 다르다.** CLI 와 MCP 사이의 것은 같은 읽기를 두 벌 쓴 것이라 하나로 합치면 됐다(U3 의 키 참조, U4 의 온체인 동사). DSL 과의 것은 그렇지 않았다. 이름이 같고 의도가 같아도 메커니즘이 다르면 합칠 대상이 아니다. 세기 전에 열어 봐야 한다.

**레이어가 계획을 이긴다.** U7 은 "DSL 도 app 을 지나게 한다"였는데, `testhelper` 가 L3 이라 불가능했다. 문서를 읽고 안 것이 아니라 import 를 넣고 컴파일러에게 물어서 알았다. 계획이 레이어를 건드릴 때는 그렇게 확인하는 편이 빠르다.

**지표가 못 내려가면 지표를 의심한다.** DSL 45개를 빚으로 세는 한 0 에 닿을 수 없었다. 내려갈 수 없는 천장은 다음 사람에게 무시하는 법을 가르친다.

**통과하는 테스트는 변이로 확인해야 한다.** 17건 중 두 건이 처음에는 아무것도 증명하지 못했다. 하나는 디렉터리를 훑다 빈 `process.json` 두 개를 비교하고 있었고, 하나는 step 의 detail 안에 우연히 들어 있는 낱말을 찾고 있었다. 둘 다 초록색이었다.

**컴파일되는 것과 도는 것은 다르다.** `tests/e2e` 는 8월 말부터 죽어 있었는데 CI 의 e2e 태그 vet 은 못 잡았다. 명령 이름이 문자열이기 때문이다. 테스트가 **참조하는 것**의 썩음은 컴파일러가 잡지만, 테스트가 **요구하는 것**의 썩음은 실제로 돌려 봐야 안다.

**라이브는 한 번에 하나만 돌린다.** 전체 스위트가 도는 중에 다른 실행을 시작했다가 그쪽 `pkill` 이 앞 실행의 노드를 죽였다.

## 1m. 남은 작업 — 순서 (2026-09-07 그래프 재분석)

> 2026-09-06 에 게이트로 전수를 세어 두었다. 하루 뒤 AST 그래프(패키지 64개, 간선 202개, 위반 0)로
> 각 항목이 어느 패키지 어느 파일에 떨어지는지 다시 재고 순서를 붙였다.
> **세는 일과 순서를 정하는 일은 달랐다.** 셀 때 "작고 독립적"으로 분류한 것 중 둘이 그렇지 않았고,
> 하나는 지금 코드에서 통과할 수 없는 게이트였다.

### 재측정이 뒤집은 전제 셋

**B2 는 적힌 만큼 풀지 못한다.** "구문이 L3 에 묶인 유일한 이유가 이 타입 하나"라고 적었는데,
`internal/dsl/interp` 는 세 파일에서 `session` 을 쓴다.

| 파일 | 쓰는 이름 |
|---|---|
| `fingerprint.go` | `Fingerprint` 하나만 쓴다 |
| `interpreter.go` | `AssertResult`·`Environment`·`TestRecord`·`TestStatus` 를 쓴다 |
| `run.go` | 위에 더해 `PostResult`·`StatusBlocked/Fail/Pass`·`StepResult` 까지 아홉 개를 쓴다 |

`Fingerprint` 가 `string` 을 반환하게 바꾸면 `fingerprint.go` 하나가 풀리고 패키지 의존은 그대로다.
덧붙이면 문법 패키지 `internal/dsl` 자체는 fanOut 이 0 이라 이미 깨끗하다. 묶여 있는 것은 구문이 아니라
해석기이고, 그것을 떼는 일은 `run.go` 의 아홉 이름을 옮기는 큰 일이다. **B2 는 작은 항목이 아니다.**

**B1-a 의 게이트는 U 트랙이 통과 불가능하게 만들었다.** `suitecmd` 가 지금 참조하는 내부 패키지는
`app`·`dashboard`·`core/home`·`chains/all` 뿐이고 `validate.go` 는 `app.Validate` 한 줄만 부른다.
그런데 `app` 의 fanOut 이 22 라서 링크가 `suitecmd → app → testengine → core/rpc·core/session` 으로
전이된다. 바이너리가 하나인 이상 어느 명령을 켜도 전부 링크된다. 즉 "링크하지 않게"는 바이너리를
쪼개지 않는 한 통과할 수 없다. **게이트를 호출 수준으로 다시 쓰는 결정이 착수보다 먼저다.**

**A7 은 `wbft` 하나가 아니다.** `RoleValidator` 라는 같은 이름이 뜻이 다른 세 곳에 있다.

| 어디 | 값 | 무엇을 가리키나 |
|---|---|---|
| `core/node/node.go:45` | `"validator"` | 블록을 만드는 **노드 역할**의 옛 표기다 |
| `dsl/spec_v2.go:95` | `"validator"` | 포크 뒤를 이어받는 **바이너리**다 |
| `validatorset/validatorset.go:20` | `"validator"` | **계정**의 기능이다 |

2026-09-06 에 고친 `node1` 대 `en1` 혼동과 같은 뿌리다. 이름 겹침 검출을 먼저 붙이면 N0/NM6 이
무엇을 지워야 하는지가 목록으로 나온다. **A7 은 N0/NM6 의 선행 조건이다.**

### 그래프가 찾아낸 합류점

`internal/chainsetup/steps_compose.go` 한 파일(841줄, 이 저장소에서 가장 큰 비테스트 파일)에
네 항목이 겹친다.

- **N9** 가 강제하려는 해석 순서가 이 파일의 함수 순서 그 자체다.
  `Keys`(64행) → `Allocate`(193행) → `Genesis`(314행) → `Config`(396행) → `Provision`(520행).
- **N0/NM6** 의 레거시 역할 방출이 `placements()` 안 167·171 행이다.
- **N1~N6** 청사진이 건드릴 `placements`(144행)·`Netmap`(672행)·`peerPlan`(794행)이 여기 있다.
- **P6** 이 줄이려는 6,781줄 가운데 가장 큰 단일 파일이 이것이고 `steps_lifecycle.go`(918줄)가 다음이다.

따로 착수하면 같은 파일을 네 번 연다. 그리고 **P6 은 독립 작업이 아니라 위 셋의 결과다.** 줄 수를
목표로 삼아 따로 착수하면 U 트랙 때처럼 코드가 옮겨 다니기만 하고 줄지 않는다(그때 4,176 대 4,569 로
확인했다). 같은 이유로 **A8 은 N1~N6 뒤에 와야 한다.** 청사진을 바꾸면 죽은 심볼 목록이 다시 그려지므로
지금 세어 둔 896건은 착수 근거가 아니라 착수 뒤에 다시 잴 대상이다.

### 순서

**0단계 ☑ 2026-09-07.** B1-a 게이트를 다시 쓰고, S 트랙을 남은 값어치로 줄여 적고, A8 을 2단계 뒤로 옮겼다.
이 절이 그 결과다.

**1단계 ☑ 2026-09-07. A7 이름 겹침 검출.** 규칙으로 설명되는 것 101개, 관용 19개, 빚 33개.
첫 수확이 `RoleValidator`×3 이었고, 그것이 2단계의 입구가 되었다.

**2단계 ◐.** `steps_compose.go` 세 덩어리 가운데 둘이 끝났다.

| 무엇 | 상태 |
|---|---|
| **N0/NM6** 역할 방출을 정본으로 | ☑ 방출은 `bp`/`en` 이고 `LegacySpelling` 은 없다. 옛 철자는 `Node`·`Record` 두 경계에서 읽을 때 접는다 |
| **N9** 해석 순서 강제 | ☑ `composeNeeds` 에 선언하고 모든 단계가 `require` 를 거친다 |

**라이브 재검증 2026-09-07: 19/19 통과**(763초, 실패도 스킵도 없음). 역할 방출이 바뀌면 argv 와 지속 상태가
달라지는데, 두 체인에서 조립·기동·합의·하드포크 스왑·정족수 붕괴와 회복·스냅 싱크가 모두 지나갔다.
다만 **불안정 두 건의 상태는 이 한 판으로 알 수 없다** — `WbftQuorum6of6Halts2`(원래 5판 중 2판 실패)와
`StablenetProposalExpiry`(5판 중 1판 실패)가 통과했지만 한 판 통과는 예상 범위 안이다.
| **N1** 선언 스키마와 파서 | ☑ `internal/core/blueprint`. `steps_compose.go` 를 건드리지 않는 새 패키지라 앞의 둘과 파일이 겹치지 않았다 |
| **N2~N6** 해석·raw 경로·물질화·preset·topology 흡수 | ☐ 여기부터가 `steps_compose.go` 를 바꾼다. 다섯이 한 덩어리라 절반만 넣으면 중간 상태로 남는다 |

**3단계 ☐.** A8 을 다시 재고, 이어서 A7b 를 결정한다. N1~N6 뒤다.

**병행할 수 있는 것** — 위와 파일이 겹치지 않는다.

| # | 무엇 | 지금 상태 | 게이트 |
|---|---|---|---|
| **N11** | 다중 config | ☑ **완료 2026-09-08.** 게이트는 `swapNode` 가 이미 만족했고, 빠져 있던 통합 테스트를 채웠다(진짜로 조립하고 드라이버만 스텁) | 일부 노드만 다른 config 로 재기동한다 |
| **V7** | 기회 개명 백로그 | ☑ `netreg` 는 모듈이 이미 없어 파일 이름만 `networks.go` 로 바꿨고, `accounts` 는 상류 SDK 를 가리키는 이름이라 두기로 했다 | 네이밍 규칙 표를 통과한다 |
| **B1**-b | 파서 fuzz | ☑ **완료 2026-09-07 (#357).** `FuzzParse`·`FuzzMigrateV1`·`FuzzInlineEnv` 셋. 씨앗은 저장소의 스펙 122개 — 무작위 바이트는 거부 경로만 훑고, 진짜 스펙이라야 변이가 수용 경로에 닿는다 | 죽지 않는 것에 더해, **통과한 것은 해석기가 쓸 수 있어야 한다**(id·chain·schemaVersion). **fuzz 가 v1 문법의 결함을 찾았다** — 아래 |
| **P8** | 미이관 테스트 케이스 | 문법 갭을 메우거나 이관하지 않을 이유를 적는다 | ☑ **완료 2026-09-07 — 갭이 아니라 기록이 문제였다.** `tests/specs/README.md` 가 갭에 막혔다고 적은 19건 중 **15건에 이미 스펙이 있고 전부 검증을 통과한다**. 없다고 적힌 프리미티브가 그동안 다 생겼기 때문이다: 로그를 유발하기 전에 구독을 여는 `wsOpen`, 손상된 이중서명을 조립하는 `sendRawTampered`, EIP-7702 의 `sendSetCode`, 로컬 키를 만드는 `newAccount`, "오류는 나도 되지만 not-found 는 아니다"를 묻는 `methodPresent`, revert 를 기대하는 `callError`, ceil(2n/3) 의 `derive op:"quorum"`. **불완전한 문서보다 나쁜 상태였다** — 그걸 보고 계획하면 끝난 일을 다시 하고, 커버리지를 감사하면 실제보다 얇다고 믿는다. 문서를 실제 상태로 고치고 `TestSpecDoc_BlockedCasesHaveNoSpec` 이 그 주장을 검사하게 했다(layers.md 처럼 **문서를 파싱하고 복제하지 않는다**). 변이 둘로 확인. **진짜 남은 8건**: SDK 클라이언트 가드 2건(표현 대상이 아니다) · 조작자 공급 키가 필요한 2건(`newAccount` 는 키를 만들 뿐 받지 못한다) · 바이너리가 기능을 안 담은 P256 3건과 genesis 빌더가 처음부터 최종 코드를 굽는 govminter 1건(라이브 반증, 다른 빌드가 필요하다). 기준선도 바뀌었다 — 레거시 등록부 `testkit.Cases()` 가 2026-09-06 에 사라져(A5) 134/56 같은 숫자는 다시 뽑을 수 없다. 기준은 커밋된 스펙 **122개**이고 전부 `validate` 를 통과한다 |
| **B2** | 해석기를 `session` 에서 떼기 | 요구를 좁힌다. **완전 분리는 하지 않는다** | ☑ **완료 2026-09-07 — 재보고 후 범위를 바꿨다.** 실측하니 결합이 타입 목록보다 훨씬 좁았다: interp 는 `Environment` 11개 중 **4개**(`Nodes`·`Resolve`·`ResolveEach`·`UpdateNode`), `TestRecord` 12개 중 **4개**(`Step`·`Assert`·`PostAction`·`Status`)만 쓴다. 그래서 `interp.NodeTable` 과 `interp.Recorder` 로 요구를 선언했다(`operation.Opener` 와 같은 수법). 세션 쪽은 구조적으로 만족하므로 엔진은 바뀐 것이 없다.

**완전 분리는 안 하기로 했고 그 이유를 남긴다.** `StepResult` 형제들을 interp 로 옮기면 서로 맞아야 하는 구조체가 둘이 되고, 한쪽에 필드를 더하고 다른 쪽을 잊으면 **아티팩트에서 증거가 조용히 빠진다.** 그건 올해 내내 없애 온 실패다(포트 표현 3벌, 링 writer 2벌, 낡은 이관 기록). import 하나 줄이자고 되풀이되는 위험을 사는 것은 나쁜 거래다. 게다가 애초의 동기였던 링크 분리는 U 트랙 이후 성립하지 않는다 — 바이너리가 하나라 `app` 을 통해 전부 링크된다.

**좁히기가 능력이 됐다는 증거**: 해석기가 세션 없이 돈다. 전에는 interp·testhelper 의 테스트가 임시 디렉터리에 진짜 세션을 세웠는데, 11개 구현보다 그게 쌌기 때문이다. 이제 4개면 된다. 변이 둘로 확인 |

**S 트랙 잔여** — U 트랙이 "기능 목록이 세 벌"이라는 08-18 진단을 이미 무너뜨렸다. CLI 와 MCP 는 한
진입점을 쓴다. 남은 값어치는 **MCP 손작성 스키마 감축**과 **`query` 투영(S7)** 둘뿐이다.
S0·S1·S2·S4 는 여기서 닫는다.

### 이 기기에서 못 하는 것 — 로컬 docker 함대가 필요하다

> **정정 2026-09-07.** 이 절은 "다른 머신이 필요하다"고 적혀 있었는데 **근거 없이 쓴 것이고 사실과 반대다.**
> R 트랙의 전제 자체가 "실 원격 서버 없이 컨테이너를 가상 서버로 쓴다"이고([[docker-remote-design]]),
> R1~R5 는 전부 끝났으며 **R6 도 15대 docker 서버셋에서 라이브로 완주했다**(2026-09-02).
> 필요한 것은 다른 기계가 아니라 이 기계 위의 docker 데몬이고, 지금 그것이 떠 있지 않다.
> 함대는 `env/docker/gen-env.sh` 가 만들고 산출물(인벤토리·localmap)은 gitignore 대상이다.

| # | 무엇 | 남은 것 |
|---|---|---|
| **R6** 잔여 | poa 원격 브링업 | **chainbench 결함이 아니다.** `etcdInit` 이 형성한 클러스터를 노드의 배경 etcd 관리가 거버넌스 멤버 14개를 보고 다시 join 하려다 잃는다. 재현 2회 중 1회. go-wemix 바이너리 쪽이다 |
| **G2** 잔여 | 핸드오프의 원격 경로 | R6 과 같은 함대에서 본다 |
| `chain rm` 원격 | N4 가 남긴 유일한 진짜 분기 | `filestore.Store` 에 삭제가 없다. 파괴적 원격 작업이라 검증할 함대가 있을 때 함께 |

### 미해결 조사

| 무엇 | 실측 | 아는 것 |
|---|---|---|
| **`TestE2E_WbftQuorum6of6Halts2`** | **2026-09-07: 18판 무실패** | **재현되지 않는다. 고쳤다는 말이 아니다.** 실제 테스트 10판 + 손 재현 8판. "head 2 에 멈춤"이라는 기록에서 **이른 정지가 방아쇠**라는 가설을 세워 head 2·4·20 에서 시험했는데 여덟 판 전부 3초 만에 회복했고 여섯 노드가 피어 5를 보고했다 — 가설은 무너졌고 대신할 원인은 없다. 그 사이 들어간 변경(`process.Stop`·NM6)이 이것과 연결된다는 것도 보인 적이 없다. 재현되면 이제 진단이 남는다 |
| ~~**`TestE2E_StablenetProposalExpiry`**~~ | ☑ **규명·수정 2026-09-07 (#358)** | 실패 4판과 통과 5판이 **`node1 sealed=0`** 하나로 갈렸다. chainbench 의 결함은 준비 판정이었다 — `detectProducing` 이 주 노드 하나에게만 물어서, 4검증자 BFT 망이 셋만 봉인해도 "생산 중"이었다. `health.Participants` 로 봉인자를 세고 하네스가 **모든 생산자의 참여**를 기다린다. 검사 없이 16판 중 5실패 → 검사와 함께 16판 중 0실패, 빼면 5판 안에 복귀 |
| **`G2` 잔여** | — | 원격 경로라 함대와 함께 본다 |

### 이 순서의 단점

2단계가 한 덩어리라 PR 이 커지고, 중간에 멈추면 `steps_compose.go` 가 절반만 바뀐 채 남는다.
쪼개려면 N0/NM6 만 먼저 잘라낼 수 있는데 그러면 N9·N1~N6 때 같은 파일을 한 번 더 연다.
그리고 0단계는 결과물이 문서뿐이라 진척이 눈에 보이지 않는다.

**쫓을 때의 교훈**: 실패한 판 하나만 보고 원인을 말하면 안 된다. 위 두 건에서 가설 셋이 무너졌는데
**셋 다 통과한 판에도 같은 표시가 있었다.** 통과와 실패를 나란히 놓고 갈리는 것을 찾는 단계가 원인과
우연을 가른다.

## 1n. 다음 사람에게 (2026-09-08)

> 이 기기(docker 데몬 미기동)에서 할 수 있는 일은 끝났다. 남은 것은 셋인데 **하나만 실제 작업**이고
> 둘은 착수 근거가 아직 없다.

### 지금 트리 상태

`main` + PR #361(커밋 12개). 53개 패키지 통과, `gofmt`·`go vet`·`golangci-lint`·`go build -tags e2e` 통과.
라이브 스위트는 2026-09-07 에 19/19 통과했다.

이번에 닫힌 것: S0·S1·S2(조회 목록)·S4·S7 · N0/NM6 · N9 · N1~N6 · A7 · A7b · A8 · B1 · B1-b · N11 · P6 · P8 · V7.

### 남은 것 1 — 로컬 docker 함대가 필요하다 (유일한 실제 작업)

**다른 기계가 아니라 docker 데몬이 필요하다.** R 트랙의 전제가 "실 원격 서버 없이 컨테이너를 가상
서버로 쓴다"이고 R1~R5 는 끝났으며 R6 도 15대 서버셋에서 라이브 완주했다([[docker-remote-design]]).

착수 순서:

1. docker 를 띄우고 `env/docker/gen-env.sh` 로 15대 함대를 만든다. 산출물(인벤토리·localmap)은
   gitignore 대상이라 저장소에 없다.
2. **R6 잔여** — `etcdInit` 이 형성한 클러스터를 노드의 배경 etcd 관리가 거버넌스 멤버 14개를 보고
   다시 join 하려다 잃는다(재현 2회 중 1회). **chainbench 오케스트레이션은 정상이고 go-wemix
   바이너리 쪽이다.** 체인팀과 볼 일이지 여기서 고칠 것이 아닐 수 있다.
3. **G2 잔여** — 핸드오프의 원격 경로. 같은 함대에서 본다.
4. **`chain rm` 원격** — `filestore.Store` 에 삭제가 없어(확인·읽기·쓰기·체크섬뿐) 경계를 통과하지
   못한다. 파괴적 원격 작업이므로 검증할 함대가 있을 때 인터페이스에 `Remove` 를 더한다. 지금은
   절반만 지우거나 지웠다고 거짓말하지 않고 소리 내어 거절한다.

### 남은 것 2 — 근거가 생기면 (지금은 하지 않는다)

| 무엇 | 무엇이 근거가 되나 |
|---|---|
| **S2 스키마 감축** | 지금 되풀이는 `rpc` 16 · `workspaceDir` 11 · `chain` 11 인데 **설명이 갈라지지 않았다**(16곳 중 15곳이 같은 표현). 예방이지 교정이 아니다. 저 숫자를 다시 세어 갈라짐이 보이면 그때다 |
| **어휘에서 이름 떼기** | `validate` 가 이름 해석에 부르는 레지스트리가 액션 구현 자체라 rpc·session 을 끌고 온다. 액션 45개의 등록 방식을 바꾸는 일이고, 문법만 쓰는 **다른 소비자**가 생기면 근거가 선다 |
| **testengine→chainsetup 엣지** | 좁히기로는 안 된다(4파일 중 1개만 사라진다, P6 참조). 조립을 엔진 밖으로 내보내 엔진이 망을 *받게* 하는 재설계라야 하고, 그럴 이유가 생기면 별도 항목으로 세운다 |
| **나머지 기능 등록** | `internal/feature` 에 **24개**가 등록됐고(2026-09-11 재측정, 문서의 17은 낡았다) 래칫이 63을 천장으로 잡고 있다. 표면을 파생 플래그로 바꾸는 일은 `TestComposeFeatures_TagsMatchTheCommands` 가 지켜 주므로 **손으로 쓴 절반을 지우는 일**이 된다 |

### 남은 것 3 — 재현되면 본다

`TestE2E_WbftQuorum6of6Halts2` 는 2026-09-07 에 18판 무실패였고 **재현되지 않는다**(고쳤다는 뜻이
아니다). 이제 대기가 포기할 때 모든 노드의 head 와 피어 수를 남기므로, 다시 나오면 그것부터 본다.

### 이번에 비싸게 배운 것 — 측정을 먼저 의심하라

라이브를 쫓을 때 **틀린 답을 그럴듯하게 만든 것이 코드가 아니라 측정이었다.**

1. **한 번만 재고 판단하지 않는다.** 12초 단일 표본이 head 0 을 회귀처럼 보이게 했다. wbft 는 1에
   머물다 가속하므로 25초까지 재면 24다.
2. **`pkill -f` 는 자기 자신을 맞춘다.** 패턴이 스크립트 본문에 들어 있으면 러너가 죽는다.
   `pkill -x <실행파일>` 로 정확히 맞춘다.
3. **도는 스크립트를 편집하지 않는다.** bash 는 바이트 오프셋으로 이어 읽으므로 아직 안 읽은 줄이
   어긋난다. 복사본을 만들어 돌린다.
4. **라이브를 둘 이상 동시에 돌리지 않는다.** 포트 8600 을 두고 다투면 같은 바이너리·같은 체인이
   24와 0을 함께 낸다. 새로 띄우기 전에 `pgrep` 으로 0을 확인한다.
5. **통과와 실패를 나란히 놓는다.** 실패한 판만 보면 가설이 선다. 제안 만료 건은 통과 5판과 실패
   4판을 견줘 `node1 sealed=0` 하나로 갈렸다.
6. **`golangci-lint` 를 검증 절차에 넣는다.** `gofmt`·`vet`·`build`·`test` 만 돌리다 CI 에서 `unused`
   에 걸렸다. 로컬에 있는데 안 돌린 것이다.


## 1o. 배선 감사 — 반영 대기 (WA 트랙, 2026-09-08)

제품의 두 목적이 실제로 배선됐는지 코드 전체를 AST 그래프로 훑어 확인했다. 6개 계층
(DSL 문법·인터프리터 / compose(DSL→체인) / run·record·report / MCP 표면 / CLI 표면·parity /
tests/tc 코퍼스)을 병렬로 감사했다. 두 목적은 이렇다.

1. tests/tc 의 DSL 을 파싱해 체인을 구성하고, 지정 테스트를 수행하고, 결과 리포트를 쓴다.
2. 체인을 구성한 뒤 동적으로 원하는 테스트를 수행한다.

둘 다 MCP·CLI 양쪽에서 되어야 한다. 도달성 요약:

| | CLI | MCP |
|---|---|---|
| 목적1 spec→구성→실행→리포트 | 됨(`chainbench run`) | 구멍: 세션 경로 미출력→리포트 회수 불가, spec 경로 실행 불가, 실패를 tool error 로 안 냄 |
| 목적2 구성 후 동적 실행 | 구멍: 단일 테스트 목록·실행 없음, 네트워크 attach 없음 | 구멍: 한 번에 구성+유지 안 됨, 노드 stop/start 없음, run 은 항상 teardown |

happy path 는 CLI 에서만 온전하다. 아래는 심각도 순 작업리스트다. 모두 정적 분석이며 라이브
실행으로 확인한 것은 없다(특히 WA1 은 이름·grep 증거다).

> **상태(2026-09-10):** 이 목록은 **PR #363 으로 반영을 마쳤다.** 아래 체크박스는 감사 당시
> 상태 그대로이고 갱신되지 않았으니, 미완으로 읽지 않는다. 표본 확인: WA2(`internal/mcp/test_tools.go`·
> `cmd/chainbench/testcmd/test.go`), WA4(`internal/mcp/node_tools.go`), WA8(`spec_v2.go:549,565` 의
> expect 검증), WA16(주석에 적힌 대로 testhelper 비교자 경유), WA26(`SPECS.md` 의
> `internal/testspec` 참조 0건). **남은 것은 §1p 에 따로 적혀 있다** — metric 수집(C),
> 체인별 거버넌스 케이스(B 잔여), MCP 플러그인 재배포(D), 커버리지(WA25).

### A. 정확성·목적 차단 (먼저 반영)

- [ ] **WA1** [치명] 라이브 MCP 플러그인이 이 소스로 빌드된 게 아니다. 라이브는 `net_*`·`chainbench_test`·`test_list`·`setup_plan` 이름인데 이 브랜치는 `chain_*` 로 등록하고 test/test_list/setup_plan 이 없다. 증거: `internal/mcp/tools.go:16-76`. 방향: 라이브 바이너리를 이 소스로 재빌드하거나 이름 매핑을 맞춘다.
- [ ] **WA2** [목적1·2 차단] 단일 테스트 목록·실행 기능이 CLI·MCP 양쪽에 없다. testengine 에 RunOne/ListTests 카탈로그 API 가 없다. 증거: `internal/testengine/` 전역 grep 무결과, `cmd/chainbench/` 에 test 그룹 없음. 방향: testengine 에 목록·단건 실행 API 를 만들고 CLI·MCP 에 붙인다.
- [ ] **WA3** [목적2 차단·MCP] 한 번에 구성+유지가 MCP 로 안 된다. `NetUp` 이 CLI 전용(`cmd/chainbench/chaincmd/up.go:49`), `chainbench_run` 은 항상 teardown 하고 `KeepUp` 을 MCP 로 노출 안 한다(`internal/mcp/run_tool.go:108-115`, `internal/testengine/suite.go:75-76`). run 이 `Server`·`Docker`·`WaitBlocks` 도 드롭한다. 방향: NetUp 을 MCP 로 노출하거나 run 에 KeepUp·Server·Docker 를 배선한다.
- [ ] **WA4** [목적2·MCP] 노드 단위 stop/start(장애 주입)가 MCP 에 없다. `app.NodeStop`/`NodeStart` 가 CLI 전용(`internal/app/net.go:147,152`, `cmd/chainbench/nodecmd/node.go:32,55`). 방향: MCP 도구를 더한다.
- [ ] **WA5** [목적2·CLI] 네트워크 레지스트리(attach/detach/list/info/peers/topology, remote_rpc)가 CLI 에 없다. MCP 전용(`internal/mcp/network_tools.go`, `remote_tools.go`). 방향: CLI `network`·`remote` 그룹을 더한다.
- [ ] **WA6** [목적1·MCP] MCP run 이 세션 경로를 출력 안 해 리포트를 회수 못 한다(dataDir 없으면 임시 dir). spec 을 경로로 못 넘겨 env 참조 해석이 우회된다(`ReadSpecFiles` CLI 전용). 증거: `internal/mcp/run_tool.go:104,141-153`, `internal/app/workflow.go:115-120`. 방향: run 출력에 세션 root 를 싣고, spec 경로 입력을 받는다.
- [ ] **WA7** [목적1·MCP] MCP run 이 테스트 실패를 tool error 로 안 낸다(항상 nil). CLI 는 exit 1/2 를 낸다. 증거: `internal/mcp/run_tool.go:141-153`, `cmd/chainbench/suitecmd/run.go:237-245`. 방향: `Summary.Failed()` 이면 tool error 를 낸다.
- [ ] **WA8** [정확성] `do` 의 `expect`(revert/reject/fail) 오타가 검증 안 돼, 오타 나면 "성공해야 함"으로 조용히 떨어진다. 부정 케이스가 거짓 통과한다. 증거: `internal/testhelper/builtins.go:432,446`, `fault.go:63`. 방향: expect 어휘를 오프라인 검증에 넣는다.
- [ ] **WA9** [정확성] node-table `role:pn` 이 아직 mesh 로 배선된다. proxied 자동 선택이 count-form 에만 있다(`internal/testengine/compose.go:175-177`). node-table 형식은 `inlineTopologyOf` 경로라 `proxies=0` 으로 mesh 가 된다. en 이 bp 를 직접 본다. 방향: node-table 에 pn 이 있으면 proxied 를 선택하고, `Peering.Validate` 가 mesh+pn 을 거절하게 한다.
- [ ] **WA10** [정확성·증적] attach 경로(`app.AttachRun`)가 readiness 게이트·실패증적·Precheck 를 배선 안 한다. MCP `run --attach` 로 돌면 E6·E8 이 무력화되고 assertion 오류 메시지가 사라진다. 증거: `internal/app/workflow.go:104-108,112` vs `internal/testengine/suite.go:261,330-338`. 방향: AttachConfig 에 PreSpec·OnFail·Control·LogReader·Precheck 를 배선한다.
- [ ] **WA11** [증적] artifacts.json 매니페스트를 쓰는 프로덕션 코드가 없다. `Recorder` 인터페이스에 Artifacts 통로가 없어 리포트 추적성 필드가 항상 빈다. 증거: `internal/dsl/interp/requirements.go:46-55`, `internal/core/session/record_impl.go:62-67`, `internal/core/report/report.go:17-24`. 방향: Recorder 에 Artifacts 를 노출하고 compose 산출물을 기록한다.
- [ ] **WA12** [증적·경미] pre-action 실패·FAIL 이 status.json 에 reason 을 안 남긴다. 증거: `internal/dsl/interp/run.go:28-33,50,61-72`. 방향: `rec.Reason(...)` 를 부른다.

### B. 문법 — 선언됐으나 죽은 배선

- [ ] **WA13** [문법] `waitFor` 의 source 가 오프라인 미검증(read 는 검증). 오타가 라이브에서야 실패한다(스펙 20건 사용). 증거: `internal/dsl/interp/resolve.go:41`, `internal/testhelper/read.go:234-237`. 방향: waitFor source 도 오프라인 검증에 넣는다.
- [ ] **WA14** [문법] 케이스 상위 `on`(DefaultOn)이 라우팅에 미적용. 스텝에 `on` 이 없으면 조용히 nodes[0] 로 간다. 증거: `internal/dsl/spec_v2.go:309`, `internal/dsl/interp/run.go:205-224`. 방향: resolveOn 이 DefaultOn 을 기본값으로 읽게 한다.
- [ ] **WA15** [문법] 상위 `timeouts` 맵이 아무 데서도 안 읽힌다. 증거: `internal/dsl/spec.go:36`, 소비처는 `migrate.go:91` 뿐. 방향: 인터프리터가 액션 타임아웃 기본값으로 소비하게 한다.
- [ ] **WA16** [문법] `InDelta` 비교자가 DSL 에서 도달 불가(디스패치 맵에 없음). 증거: `internal/dsl/assert/assert.go:97,202-219`. 방향: funcs 맵에 등록하거나 인터프리터에서 호출한다.
- [ ] **WA17** [문법] 임베드 `SchemaV2` 가 검증에 미사용이고 파서와 이미 드리프트(`onEach` 스키마는 string, 코드는 array). 증거: `internal/dsl/spec_v2.go:18`, `internal/dsl/schema/v2.schema.json:290,331` vs `run.go:212`. 방향: 스키마를 실제 검증에 쓰거나 파서와 일치시킨다.
- [ ] **WA18** [문법·경미] `onEach` 가 do-스텝 스키마에 있으나 액션 라우팅이 소비 안 한다(어서션만 소비). 증거: `v2.schema.json:290`, `builtins.go:543`. 방향: do-스텝 onEach 를 소비하거나 스키마에서 뺀다.

### C. compose — DSL→체인 배선 누락

- [ ] **WA19** [compose] `env.target` 이 배치를 안 하고 fingerprint 만 바꾼다(오배선). 증거: `spec_v2.go:308`, `compose.go:158-170`(Target 미설정), `verbs_up.go:45`. 방향: compose 가 Target 을 NetUpIn 에 배선하거나, env.target 을 문법에서 뺀다.
- [ ] **WA20** [compose] upgrade(handoff) env 가 같은 env 의 hardforks/topology/launch/config 를 무시한다. 증거: `compose.go:100-116`. 방향: 핸드오프 경로가 이 필드들을 이어받게 한다.
- [ ] **WA21** [compose·역방향] DSL 로 표현 못 하는 chainsetup 기능: manifest/template(`verbs_up.go:42-43`), blueprint(`verbs_up.go:59`), keys-validator-subset(`verbs_steps.go:89-90`), chainID/networkID 고정, deploy-only stage(`verbs_up.go:27`). 방향: 필요한 것을 EnvV2 문법에 더한다.
- [ ] **WA22** [compose·게이팅] `env.capabilities` 가 케이스에 `requires` 가 있으면 통째로 버려진다. 증거: `spec_v2.go:312-314`. 방향: 두 목록을 병합한다.

### D. 커버리지·문서

- [ ] **WA23** [커버리지] compose→run→report 전 과정 테스트가 전부 live-gated 라 CI 가 건너뛴다. 증거: `internal/testengine/*_live_test.go`. 방향: 바이너리 없이 도는 CI 통합 테스트를 하나 만든다.
- [ ] **WA24** [죽은 능력] 스펙이 안 쓰는 등록물: 액션 `faucet`·`registerContract`, 어서션 `metric`·`createAddress`·`contractChecksum`, `hooks.post/onFail`, `defaultOn`, `placement`, `boot` role. 방향: 스펙으로 검증하거나 등록을 뺀다.
- ◐ **WA25** [커버리지] go-wemix(5건)·go-wbft(6건) 얕음. **proxied 라우팅 스펙은 생겼다 (2026-09-11)** — `tests/tc/go-wbft/network/01-wbft-proxied-routing.json`. peer 수가 그 그래프의 서명이다: en1=1 · pn1=3 · bp1=bp2=2, 라이브 실측이 정확히 일치했다. **판별력도 확인**했다 — 같은 함대에서 pn 없는 4노드 mesh 를 올리면 en1 이 3 을 보므로 `en1 == 1` 은 mesh 를 실제로 구분한다(공허한 검사가 아니다). 남은 것은 두 체인의 tx·fault·거버넌스 케이스 확충이다. 참고: **poa(go-wemix) 는 pn 을 거부**하므로(패밀리에 프록시 계층이 없다) proxied 라우팅 검증은 wbft 계열에만 성립한다.
- [ ] **WA26** [문서] SPECS.md 가 없어진 `internal/testspec` 를 7곳 참조(드리프트). 증거: `tests/tc/SPECS.md:123,136,155,159,336,380`. 방향: `internal/testhelper`/`internal/testengine` 로 갱신한다. (2026-09-08 추가된 `docs/guide/dsl-authoring.md` 로 일부 해소 가능.)


## 1p. WA 트랙 이후 — fleet 검증 결과와 남은 작업 (2026-09-08)

WA 트랙은 PR #363 으로 main 에 머지됐다(WA1~WA26 중 코드/문법/표면 항목 반영). 이어
docker 함대(15대, stablenet/wbft/wemix)에서 커버리지 스펙을 라이브로 돌려 다음을 확인했다.

**라이브 통과 확인**: createAddress·contractChecksum, faucet(amount 를 십진 wei 로 수정),
proxied pn 라우팅(keys preset 로 변경), registerContract, go-wbft tx·fault, go-wemix fault.
전부 preset 키만 쓴다 — 생성 키를 저장소에 남기지 않는다.

### 남은 작업

- ◐ **C. metric 수집 인프라 (신규 — WA24 metric 이 드러냄)**. **코드 경로는 열렸다
  (2026-09-11), 라이브 검증만 남았다.** 진단이 셋 중 하나 틀렸다 — ②수집 경로는 이미
  있었다(`collector.ScrapeMetrics`). 실제로 막던 것과 처리:
  - **바인드 주소**: `metricsHost` 기본값이 `127.0.0.1` 이라 노드 자기 기계에서만 닿았다.
    HTTP 와 같은 `0.0.0.0` 으로 맞췄다. 좁히려면 `metricsHost` 노브를 노드별로 준다.
  - **주소 번역**: 어서션이 `Host`+`Ports.Metrics` 로 직접 조립해 dial 했다. `node.Node` 에
    `MetricsURL` 을 더해 `RPCURL` 과 **같은 opener 를 지나게** 했다 — 번역은 이제 구성이
    기록하고 어서션은 읽기만 한다.
  - **config 없는 기동**: `Argv` 에 Metrics 모듈이 없어 핸드오프 재기동 노드는 metrics 가
    아예 없었다(같은 망이 포크 전에는 답하고 후에는 답하지 않았다). 포트가 배정된 노드면
    커맨드라인으로도 말하게 했다.
  - 제거했던 `03-metric-head-block` 스펙을 **되살렸고 라이브로 통과시켰다**(2026-09-11,
    15대 함대, `chain_head_block=3`).
  - **라이브가 유닛 검증이 놓친 것 둘을 더 찾았다.** ① `env/docker/gen-env.sh` 가 metrics
    포트 6060 을 퍼블리시하지도 localmap 에 넣지도 않았다 — `AddrMap` 은 매핑 없는 포트를
    **그대로 두므로** 호스트는 loopback 으로 바뀌고 포트는 6060 으로 남아 아무것도 답하지
    않았다. ② 기록된 `MetricsURL` 을 `HTTPEndpoint`(주소만 해석)로 만들어
    **`/debug/metrics/prometheus` 경로가 빠졌다** — 살아 있는 metrics 서버가 루트에서 404 를
    냈다. 유닛 테스트는 둘 다 잡지 못했다(전자는 함대 정의, 후자는 경로를 검사하지 않음).
- [x] **B — 부정 경로**: `go-stablenet/tx/01-negative-tx-revert` (revert 하는 런타임 배포
  후 `expect:revert`) fleet 검증 완료.
- [ ] **B 잔여 — 거버넌스(체인 특화, fleet)**. go-wbft·go-wemix 거버넌스는 stablenet 을
  그대로 옮길 수 없다 — stablenet 의 GovValidator 흐름(0x…1001, proposeAddMember)을 go-wbft 로
  적응해 fleet 에서 돌리니 **2단계 proposeAddMember 가 revert** 했다(node1 이 gwbft 의 gov
  멤버가 아니거나 시스템 컨트랙트 셋업이 다름). 각 체인의 실제 거버넌스(컨트랙트 주소·선택자·
  멤버·정족수)를 확인한 뒤 작성해야 한다. go-wemix(poa)는 etcd/거버넌스 배포 경로라 더 다르다.
  registerContract 도 실제 메서드/ABI 를 쓰는 케이스는 여기 포함(현재는 임의 호출 컨트랙트로 최소 검증).
  negative-tx 의 reject(제출 거부) 변형도 남아 있다(신뢰할 수 있는 재현 방법 확정 필요).
- [ ] **D. WA1 — 라이브 MCP 플러그인 재배포**. 배포본이 `net_*` 이름의 stale 빌드다. 저장소는
  `chain_*` 로 개명됐으니 재빌드·재배포만 하면 맞는다. 코드가 아니라 배포 작업이다.
- [x] **E. WA21 — DSL 문법 확장(D4 로 미룬 것)**. env 에 `blueprint`(레이아웃+키 한 문서)와
  `keys.nodekeys.validators`(생성 키 중 N개만 validator)를 더해 compose 로 배선했다(단위 테스트).
  **deploy-only stage 는 넣지 않았다** — 테스트 케이스는 어서션을 돌리므로 항상 망을 기동해야
  한다(UpStart). 배포만 하는 compose 는 테스트 env 의미가 없다. blueprint/validator-subset 를
  실제로 쓰는 라이브 스펙은 다른 커버리지처럼 fleet 에서 작성·검증하면 된다.
- [ ] **WA10 잔여 없음** — attach 경로 게이트·증적(E6·E8)까지 반영됨(PR #363).

기존 §1n 의 R6(go-wemix etcd collapse — 체인팀), G2(핸드오프 원격), 원격 `chain rm`
(filestore Remove 부재)은 WA 와 무관하게 그대로 남아 있다.


## 1q. workspace-config 트랙 이후 — 남은 두 항목 (2026-09-10)

workspace-config(W1~W6, PR #369)와 그 후속(런타임 validator 검사·정의서 순차 실행·
승인 기준·다이얼 주소 레이어 정리, PR #370)은 반영됐다. 인계 문서
`docs/research/chainbench/analyses/10-prepared-inputs-server-ref-handoff.md` 에서 나온
항목 중 **둘은 아직 코드가 없고, 지금까지 이 작업 리스트에 적혀 있지 않았다.** 연구
문서에만 남아 있으면 사라지므로 여기 옮긴다.

- [ ] **L1. 노드별 시도(attempt) 로그 경로**. 지금 노드 로그는 노드당 한 파일
  (`logs/<compId>/<label>.log`)이라 **재기동하면 이전 실행의 로그가 덮인다**. 실패한
  시도의 증적이 다음 시도로 지워지는 것이 문제다 — 특히 reuse-if-matching 이 한 노드를
  여러 번 재작업할 때. 인계 문서와 `workspace-config.sample.yaml` 의 주석은
  `logs/<compId>/<nodeId>/<attemptId>.log` 를 예고하지만 `core/node/layout.go` 에
  attemptId 개념이 없다. **필요한 것**: layout 에 attempt 축 추가, 기동마다 증가시키는
  소유자 결정(런 원장이 유력), 실패 수집(`collectFailureData`)이 해당 시도의 로그를
  고르도록 배선. **게이트**: 한 노드를 두 번 재기동한 뒤 두 시도의 로그가 모두 남아 있고,
  실패한 테스트의 관측 폴더가 그 시도의 것을 담는다.

- [ ] **L2. 준비된 키의 대상 검증(공개 신원만 반환)**. 인계 문서 §7 은 "준비된 키의 검증은
  가능한 한 대상에서 수행하고 공개 신원만 반환한다"고 정했다. 현재 srv:// keyring 은
  **키 묶음 전체를 로컬로 내려받아** 쓴다(`materializeKeyring`, PR #370). 이는 사용자가
  명시적으로 승인한 경로이고 런타임 서명에 로컬 경로가 필요해서 정당하지만, §7 이 말한
  "대상에서 검증" 은 아직 없다. **필요한 것**: 대상 서버에서 키를 열어 주소·공개키만
  돌려주는 경로(다운로드 없이), 그리고 그 결과로 genesis validator 대조. 다운로드가 필요한
  경우(서명)와 검증만 필요한 경우(대조)를 구분하는 것이 요점이다. **게이트**: 원격 키 묶음을
  로컬에 내려받지 않고 공개 신원 목록을 얻어 genesis 와 대조한다.

두 항목 모두 지금 동작을 막지 않는다 — L1 은 증적 보존, L2 는 키 취급 범위를 좁히는
일이다. 착수 전에 인계 문서 §7·§8 을 함께 읽을 것.


## 1r. 모니터링 이슈 재검토와 수정 계획 (2026-09-10)

모니터링 세션이 제기한 16건(MON-001~016,
`docs/research/chainbench/analyses/11-monitoring-open-issues.md`)을 HEAD `2cc82692` 기준으로
다시 판정했다. 판정 근거·추가 발견·검증 계획은 정본
[`monitoring-issue-review-2026-09-10.md`](monitoring-issue-review-2026-09-10.md)에 있다.

결과: **유효 14건, 해결·검증 완료 2건(MON-004·MON-006), 오탐 0건.** 기준 시점의 전체
테스트는 통과 상태였으므로, 남은 14건은 모두 기존 테스트가 잡지 못하는 결함이다.
문서에 없던 추가 발견 5건(N1~N5)도 정본에 적었다.

작업은 작은 모듈에서 상위 조합, 표면 순서로 진행한다. 각 항목의 완료 기준은 정본의
6절, 검증 방법은 7절을 따른다.

- ☑ **MR-A. 프리미티브·합의 패밀리** (`c6b8b091`)
  - A1 MON-012 `internal/consensus/wbft` RLP 길이 범위 검사 + fuzz
  - A2 MON-014 `internal/consensus/poa` 멤버 수 uint256 검증·상한
  - A3 MON-003 `internal/dsl` null·비객체 env 거부
  - A4 MON-011·N1 `internal/core/filestore` 로컬 Write가 mode 강제(심볼릭 링크 포함).
    비밀 쓰기 여러 곳의 공통 원인이므로 호출부마다 고치지 않고 스토어에서 고친다.
- ☑ **MR-B. 키와 비밀** (`2246f14e`) — 결정 ①② 승인 반영
  - B1 MON-001 인라인 개인 키가 `State.Nodes[].Key`·`State.Request`에 남지 않게
  - B2 MON-002 명시 키와 기존 keyring 신원 불일치를 자원 변경 전에 거부(재사용 자체는 유지)
- ☑ **MR-C. 구성 오케스트레이션** (`internal/chainsetup`) — C1 `e6641505`, C2~C4 `72e7ca63`, C5 `d370597d`
  - C1 MON-009 후보 생성과 실행 경로 반영 분리 — 거부 시 파일·해시·PID 보존
  - C2 MON-010 발견 결과에 서버·전체 datadir 보존(구성 간 label 충돌 제거)
  - C3 MON-008·N2 `recordRun`이 실제 GenesisPath 사용 + 노드 config 수집
  - C4 MON-007 existing genesis와 변경 옵션 충돌 거부(capability 산출 포함)
  - C5 MON-016 baseline 관측 시점·누락 처리 정정
  - C1·C5는 "대상의 현재 파일을 읽어 해시" 기능을 공유하므로 공용 헬퍼 하나로 만든다.
- ☑ **MR-D. 표면·환경**
  - ☑ D1 MON-015·N3 `--json` stdout 계약과 순차 실행 종료 코드 (`edb05eb8`)
  - ☑ D2 MON-005·N5 `env/docker/gen-env.sh`와 README를 새 server-set 계약에 맞춤 (`d210ac93`).
    생성기가 workspace-config 와 server-set-wemix 까지 찍고, 생성물을 실제 파서로 읽는
    테스트를 붙였다. 로컬 `build/` 는 손으로 고쳐 놓은 상태였고 생성기만 옛 형식을
    쓰고 있었다 — 새 체크아웃에서만 깨지는 모양이었다.
  - ☑ D3 MON-013 샘플 안내를 실제 소비 범위와 일치 (`21d47abe`).
    `binaryAliases` 는 파싱만 되고 읽는 곳이 없으며, 객체형 참조는 아예 로드에
    실패한다. 두 경우를 갈라 적고 테스트로 고정했다.
  - ☑ D4 N4 data-root 충돌 규칙을 `internal/resource` 한 곳으로 (`35735508`)

- ☑ **Docker 라이브 검증** (컨테이너 15대) — 7절 다섯 시나리오 모두 통과. 검증 중
  포트 충돌 메시지가 다른 워크스페이스의 노드를 자기 것으로 부르는 결함을 새로 찾아
  같이 고쳤다(`9d0109e3`). 상세는 정본 9절.

6절 결정 6건은 승인 완료(정본 8절). 남은 것은 PR 하나.


## 1s. 남은 작업 한눈에 (2026-09-10)

§1n 부터 §1r 까지 트랙마다 흩어져 있던 미완 항목을 한 곳에 모았다. 각 항목의 근거와
배경은 원래 절에 그대로 두고, 여기서는 **무엇이 남았고 왜 남았는지**만 적는다. 표시가
낡아 미완으로 보이던 것들(§1o 의 WA 체크박스, `hardfork` 결정)은 이번에 정리했다.

착수 순서를 정한 목록이 아니다. 성격이 다른 다섯 갈래이고, 갈래끼리는 서로 막지 않는다.

### A. 지금 진행 중

- ~~**모니터링 트랙(§1r)**~~ — **머지 완료 (`9b6b0930`).** 재검토를 두 번 거쳤다.
  1차에서 **MON-001 이 미해결이었음을 확인하고 고쳤다**(인라인 키가 상태 파일·stderr·
  `--json` stdout 세 곳으로 샜다). 2차에서 **MON-017 을 새로 확인해 고쳤다** — 노드별
  바이너리 이름이 다르면 실행 중인 노드를 발견하지 못했고, Docker 에서 전후를 대조해
  확인했다. **MON-015 의 단일 실행 분기**도 복수 실행과 답이 달라 맞췄다. MON-007·010 은
  동작은 맞았고 근거를 넓혔다. 3차에서 **MON-018** — 그 키 노출 회귀 테스트가 문법 오류로
  끝나 검사에 닿은 적이 없었음을 확인하고 바로잡았다(프로덕션 결함은 없었다). 상세와 검증
  수준 구분은 정본의 재검토 절들.

### B. fleet 커버리지와 관측 (§1p)

라이브 함대가 있어야 진행되는 갈래다. **함대는 2026-09-11 에 섰다** — 15대 docker,
세 체인 패밀리 15노드 스모크가 모두 통과했다(stablenet · wbft · wemix(poa)). 체인
바이너리는 세 저장소를 `golang:1.23-bookworm` 컨테이너에서 linux/arm64 로 빌드했다
(호스트가 macOS 라 `CGO_ENABLED=0` 크로스컴파일은 blst 에서 실패한다).

- [x] **C. metric 수집 인프라** — **라이브 검증 완료 (2026-09-11).** 15대 docker 함대에서
  `03-metric-head-block` 통과(`chain_head_block=3`). 상세는 §1p C. 라이브가 코드 검증이
  놓친 결함 둘을 더 찾았다 — `gen-env.sh` 가 metrics 포트를 퍼블리시·매핑하지 않았고,
  기록된 `MetricsURL` 에 Prometheus 경로가 빠져 있었다(둘 다 수정).
- [ ] **B 잔여 — 체인별 거버넌스 케이스.** stablenet 의 GovValidator 흐름을 go-wbft 로
  그대로 옮기면 2단계 `proposeAddMember` 가 revert 한다. 체인마다 컨트랙트 주소·선택자·
  멤버·정족수를 확인한 뒤 써야 한다. go-wemix(poa)는 etcd·거버넌스 배포 경로라 더 다르다.
  실제 ABI 를 쓰는 `registerContract` 케이스와 negative-tx 의 reject 변형도 여기 묶인다.
- [ ] **WA25. go-wemix·go-wbft 커버리지가 얕다.** 각각 5건·6건뿐이고, pn/proxied 라우팅을
  검증하는 스펙이 없다.
- [ ] **D. 라이브 MCP 플러그인 재배포.** 배포본이 `net_*` 이름의 낡은 빌드다. 저장소는
  `chain_*` 로 개명됐으니 재빌드·재배포만 하면 맞는다. **코드 작업이 아니라 배포 작업이다.**

### C. 키 취급과 증적 (§1q)

둘 다 지금 동작을 막지 않는다. 하나는 증적 보존, 하나는 키가 로컬에 내려오는 범위를
좁히는 일이다.

- [ ] **L1. 노드별 시도(attempt) 로그 축.** 노드 로그가 노드당 한 파일이라 재기동하면
  이전 시도의 로그가 덮인다. reuse-if-matching 이 한 노드를 여러 번 재작업할 때 특히
  문제다. `core/node/layout.go` 에 attempt 축이 없다.
- [ ] **L2. 대상에서 키 검증(공개 신원만 반환).** 지금 `srv://` keyring 은 묶음 전체를
  로컬로 내려받는다. 서명에 로컬 경로가 필요해 정당한 경로지만, 대조만 필요한 경우까지
  내려받을 이유는 없다. 다운로드가 필요한 경우와 검증만 필요한 경우를 가르는 것이 요점이다.

### D. 모니터링 트랙에서 파생된 후속 (§1r 8절)

이번 PR 의 범위 밖으로 명시하고 미룬 것들이다.

- [ ] **후보·반영 완전 분리.** 부분 재사용에서 승인된 노드만 정지·반영·재기동한다.
  지금은 판정을 쓰기 앞으로 옮기는 데까지 했다(MON-009).
- [ ] **`binaryAliases` 와 객체형 참조를 실제로 소비.** 전자는 **리졸버까지는 생겼다** —
  `WorkspaceConfig.BinaryPath`(`resource/workspaceconfig.go:327`)가 별칭을 적용한다. 다만
  **호출자가 테스트 둘뿐이라 프로덕션 경로에는 아직 배선되지 않았다**(2026-09-11 확인). 후자
  후자(`{server,ref}`·`serverIndex`·`localPath`)는 아직 파싱조차 안 된다. 샘플과 안내
  문서가 "미구현" 이라고 적어 두었고, 구현하면 `TestWorkspaceConfig_SampleCommentsMatchWhatParses`
  가 실패하며 그 주석을 걷으라고 알린다.
- [ ] **여러 정의서 실행의 통합 report.** 지금은 실행마다 따로 남는다.

### D2. 코드 건강도 검토에서 나온 것 — **다섯 항목 완료 (2026-09-11)**

AST 로 다시 측정했다. 구조는 깨끗하다 — 층 위반 0, 래칫 통과, 린터 0건. 중복과 문서
두 갈래만 남았고, 중복은 **표면·상위 계층이 프리미티브를 각자 다시 만든** 한 가지
경향이었다. 근거와 위치는 정본
[`architecture/code-health-review-2026-09-10.md`](architecture/code-health-review-2026-09-10.md).

- [x] `shellQuote` 4곳 통합 → `remote.ShellQuote` 하나. 셸을 실제로 태워 왕복시키는
      테스트를 붙였다(따옴표·개행·`$(id)`·`;id;` 18케이스). 네 사본 중 어느 것도
      테스트가 없었다.
- [x] `ArgString`/`ArgInt` 사본 제거. **단순 치환이 불가능했다** — `internal/mcp` 는
      `core/*` 를 직접 import 할 수 없고(`arch.TestMCPGoesThroughApp`, shrink-only),
      그래서 그 사본은 **규칙이 만들어낸 결과**였다. `app` 이 중계하도록 고쳤다
      (`app/args.go`). 구현은 `core/registry` 한 곳, 짧은 이름은 183 호출부에 그대로.
      `ArgStrings`/`ArgBool` 도 같이 올렸다.
- [x] `deps(cmd)` → `surface.Deps`. 실제로는 **12곳이었고 세 갈래로 갈라져 있었다** —
      7곳 Logf 만, 4곳 +Env, 1곳 +Command. 즉 어떤 명령 트리로 들어왔느냐에 따라
      워크스페이스 잠금이 `(command not recorded)` 로 남거나 실행을 이름으로 남겼다.
      합집합으로 통일했다(Env 는 app 의 nil 폴백과 동일, Command 는 잠금이 늘 가져야
      할 출처).
- [x] package doc — `internal` 미문서 **8개를 0개로**. 그 과정에서 두 건은 이름이
      틀린 문서였다: `core/keyring/operation` 이 "Package keyring", `core/filestore` 가
      "Package provision" 으로 시작했다(이관 후 남은 잔재, godoc 에 틀린 이름이 뜬다).
- [x] `netUpFrom` 191줄 → **95줄**. `planUp`(38, 검증·기본값)과 `upSteps`(69, 스텝
      테이블)로 갈랐다. 잠금 획득과 실행 루프만 본문에 남는다.

**이 작업이 새로 찾은 것**

- [x] ~~**`core/keyring/derive` 에 테스트가 하나도 없다.**~~ **해소 (2026-09-11).**
      `keys/preset` 을 정답 벡터로 삼아 5노드의 주소·devp2p 공개키·BLS 공개키·PoP 를 전부
      재파생해 바이트 단위로 대조한다. **테스트가 실패할 수 있음을 뮤테이션으로 확인**했다 —
      version-3 salt(주석이 지목한 그 미묘한 지점)로 바꾸면 preset 대조가 깨진다.
      `Derive` 의 doc 이 이미 "픽스처와 바이트 단위로 대조된다"고 적어 두었는데 그 대조가
      실제로는 없었다. 원래 진단: BLS 파생이 blst 의 `blst_keygen` 과 어긋나면 *형태는
      맞지만 wbft 노드가 거부하는* 키가 나온다 — 합의 문제처럼 보이는 키 문제다.

### E. 오래 남아 있는 잔여 (§1n 및 그 이전)

- [x] **원격 `chain rm`** — **완료·라이브 검증 (2026-09-11).** `filestore.Store` 에
  `Remove` 를 더하고 `Workspace.Rm` 이 target 의 store 를 지나게 했다. 그래서 이제 원격이
  분기가 아니라 같은 경로다 — `internal/arch` 의 target-분기 허용 목록에서 `Rm` 항목이
  **사라졌다**(래칫이 줄었다). 파괴적 연산이라 가드를 두 겹으로 뒀다:
  `filestore.CheckRemovable`(빈 경로·상대 경로·`/`·1세그먼트·glob·`$`/`~` 거부, 두 Store
  구현 모두가 적용)과 `filestore.CheckWithin`(호출자가 아는 target dataRoot 안으로 구속,
  root 자신도 거부). 라이브: stablenet 3노드 → stop → rm → **수동 정리 없이** wbft 3노드가
  같은 서버에 올라가 블록 8까지 진행. 자기 구성만 지우고 다른 구성과 `bin/` 은 남는다.
  `env/docker/README.md` 의 수동 정리 절차를 이 명령으로 교체했다.
  - 남은 흠(경미): 삭제 후 `runtime/<id>/`·`configs/` **빈 디렉터리가 남는다.** 기록된
    경로만 지우기 때문이고(그게 안전한 쪽이다 — 구성 id 가 비면 `runtime/` 전체가 대상이
    될 수 있다), 아무것도 막지 않는다.
- [x] **G2. 핸드오프 원격 — 완료·라이브 검증 (2026-09-11).** 원격 대상에서 핸드오프가
  끝까지 돈다: `handoff confirmed: head 21; block 21 sealed by 0x8eb79036… (successor)`.
  `upgrade run --server <name> --server-set … --workspace-config … --docker`.
  - **포트는 이제 server set 에서 나온다.** 이전에는 profile 의 `base_rpc: 40010` 을 써서
    함대가 퍼블리시하지 않는 포트로 dial 했다. 서버를 지정하면 그 서버의 slot 을 받는다
    (5노드 → slot 1~5, http 8601~8605, p2p step 3 으로 30301/30304/…). profile 의
    `ports:` 는 단일 호스트 로컬 실행의 폴백으로 남는다.
  - **라이브가 찾은 결함 셋** — 전부 "대상 인식 경계가 이미 있는데 로컬 가정이 남아 있던" 자리:
    1. `upgrade/launch.go` 가 HTTP 를 **`127.0.0.1` 에 하드코딩**했다. 컨테이너 안에서는 그것이
       컨테이너 자신의 loopback 이라 퍼블리시 포트가 아무것도 못 만난다 — 노드는 돌고 있는데
       바깥에서 "not ready" 로 보인다. 대상에 배치된 노드는 `0.0.0.0` 에 바인드한다.
    2. readiness·mesh dial 이 노드 **자기 주소**로 나갔다. `NodeSpec.RPCURL`(opener 가 번역)을
       더해 `process.NodeOf` 가 그것을 우선한다 — `MetricsURL` 과 같은 형태다.
    3. `env/docker/firewall.sh` 가 포트 대역을 **4 slot 폭으로 하드코딩**했다(8601:8604).
       `gen-env.sh` 의 `SLOTS` 를 올려도 방화벽이 안 따라와, slot 5 노드가 돌면서 DROP 됐다.
       이제 두 파일이 같은 knob 에서 파생된다.
  - `resource.ValidatePortsPerHost` 를 더했다. 포트는 **머신 안에서만** 경쟁하므로, 서버 15대에
    퍼진 망은 node1·node2 가 다른 호스트에 같은 포트를 갖는 것이 정상이다. 기존
    `ValidatePorts` 는 그것을 충돌로 봤다.
  - **남은 것: 서버 여러 대에 걸친 핸드오프.** 배치는 되지만(`--all-servers`) 실행이 아직
    단일 머신을 지난다 — `LaunchOptions` 가 `Files` 하나, `Launch` 가 `Driver` 하나를 받고,
    컴포지션 경로는 노드마다 머신을 푼다(`chainsetup.machineFor`). 그 배선이 없는 채로 돌면
    전 노드가 한 서버에 뜨면서 겉보기엔 성공하므로, **호스트가 둘 이상인 배치는 이유를 말하고
    거절한다**(`MultiMachine`). 요청받은 15+15(서버당 wemix 1 + wbft 1)는 이 배선 다음이다.
  - 로컬 경로 회귀 없음: 로컬 핸드오프도 `handoff confirmed: head 22` 로 완주.
- [ ] **R6. go-wemix boot-etcd collapse.** 키·genesis 문제가 아님을 확인하고 넘겼다.
  **체인팀 몫이다.**
- [ ] **validatorset 홈 결정.** `core/node`(L0)로 넣으려던 계획은 층 위반이라 제자리에
  뒀다. 로스터 계산의 올바른 소유 모듈을 정해야 한다. 소비자는 `cmd` 하나뿐이라 급하지 않다.
- [ ] **health 를 inspector 조합 레이어로 재배선.** `inspector` 는 판단 없는 atomic
  프리미티브로 두고, 블록 전진을 *판정* 하는 `health` 가 그 위에서 조합하게 한다. 지금
  `health` 는 obs/rpc 를 직접 쓴다.
- ◐ **Phase 2~6 의 부분 완료 항목** — T2.1(driver 위 Transport 타입 형식화), T3.3, T5.1,
  T6.6. 각 절에 무엇이 끝났고 무엇이 남았는지 적혀 있다.

### 이번에 바로잡은 표시

- §1o 의 WA1~WA26 체크박스는 PR #363 으로 반영을 마쳤는데 미완으로 남아 있었다. 절
  머리에 상태와 표본 확인 근거를 적었다.
- `hardfork 는 통폐합 대상 아님` 은 할 일이 아니라 내린 결정이라 ☑ 로 바꿨다.


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
- ☑ **T1.6 keyreg** 랜덤/기존/원격다운로드 통합 + **BLSDeriver boundary**(외부 bootnode 캡슐화, 부재 시 오류). — F2
- **게이트**: 단위 100% + 동시 모듈 `-race`.

### Phase 2 — Transport
- ◐ **T2.1** Local/Remote Transport를 driver 위에 형식화 + **종료검증(`kill -0`)** + **key_file 인증**. **[D안]** ☑ **key_file 인증**: `remote.Credentials` 에 `PrivateKey`/`Passphrase` 추가, `authMethods`(key 우선+password, 최소 1개 필수, 키자료 미노출), `LoadPrivateKey`(0600 강제·insecure perm 거부). deploy `credentials.go` 가 `key_file`(+ `CHAINBENCH_REMOTE_KEY_FILE`/`_PASSPHRASE` env) 를 `remote.Credentials.PrivateKey` 로 로드 — "future phase 예약" 게이트 제거. 단위검증(authMethods 4케이스·키 미노출·perm 거부·For key_file). ☑ **종료검증(`kill -0`)**: 이미 `procman.Alive`(signal 0)+`StopAll`(SIGTERM→wait→SIGKILL→poll→leak 보고)로 구현, 엔진 teardown 이 이를 경유(라이브 고아0 확인). **남은 것**: driver 위 Transport 타입 형식화(C 원격 슬라이스와 함께).
- ☑ **T2.2 upload-if-absent**(로컬·FileSink; 원격 SSH sink는 remote 슬라이스) `test -f` 존재확인 → 재사용/업로드(현재 항상-업로드 or 항상-읽기).
- **게이트**: 단위(모의) + 통합 1건(로컬 더미 프로세스 검증-종료·고아0).

### Phase 3 — Middle 통합(라이브)
- ☑ **T3.1 Provisioner** datadir+키+genesis+config 물질화(local·remote 동일경로)+upload-if-absent.
- ☑ **T3.2 Supervisor** 오케스트레이션·teardown·procman배선 단위완료 + **실4노드 라이브 헬스게이트(블록전진 gate)로 Phase4 BuildEnv e2e 에서 검증**(고아0). etcd 리더 게이트는 wemix 계열(gwemix/etcd 필요) 후속.
- ◐ **T3.3 Collector** ☑ RPC 스냅샷(height·peers)·WaitLog poll·**로컬 live tail**(offset 증분·스캔→tail·부분줄 미방출·`Deps.OnLine` obs 미러 boundary)·**bp참여 집계**(head producer 샘플→`BPParticipation`, `Deps.BPWindow` 로 bounded prune)·**fork/reorg 검출**(높이별 first-seen hash, 노드 간/샘플 간 불일치→`Forked`). 단위+`-race` 검증. ☑ **엔진 배선**(`withCollection` 이 local/attach 의 BuildEnv 를 래핑 — `Bus` 설정 시 env 별 collector 실행, RPC probe(`rpc.Client.HeadBlock`)로 샘플, chainstate 스냅샷·tail 로그를 obs 로 미러, teardown 시 정지; attach+mock RPC 로 chainstate 이벤트 e2e 검증)·**chainstate 세션 영속화**(collection 이 스냅샷을 `chainstate/chainstate.jsonl` 로 기록 — F10/F15 jsonl+obs 미러, 완료 세션 재생용; best-effort). ☑ **원격 SSH tail**: tail 루프가 **`collector.LogReader` boundary**(`ReadFrom(ctx,path,offset)`)을 거치도록 리팩터 → 로컬은 `LocalLogReader`(파일), 원격은 `driver.RemoteLogReader`(SSH `tail -c +N`, **1-based 바이트 오프셋** — `tail -n` 은 줄 단위라 collector 가 추적하는 정확한 바이트 위치를 잃어 줄 중복/분할이 남). 파일 부재는 오류 아님(첫 줄 쓰기 전 tail 시작 허용), 전송 실패만 오류. 단위검증 5건. **실 SSH 호스트 라이브 e2e 는 사용자 환경 필요**(T5.1·C2). attach=RPC-only(로컬 로그 없음→tail no-op).
- ☑ **T3.4 Session 저장·재사용** `session.json`·env fingerprint 재사용(엔진 오케스트레이션이 fingerprint 로 env 재사용)·records(spec/steps/assert/status). 엔진 단위·라이브 e2e·attach e2e 로 검증(summary.pass 판정).
- ☑ **T3.2b supervisor 선언 논항 방출** (x-bar 정렬 검토 2026-08-09) 선언만 되어 있던 `Options` 3개를 실제로 읽는다.
  - **`LeaderGate`**: `Deps.LeaderGate(ctx, ns, window)` boundary 신설 — **HealthGate 보다 먼저** 실행(클러스터에 리더가 없으면 노드는 healthy 일 수 없다). "리더 준비"의 판정은 체인특화(go-wemix 내장 etcd)라 주입식이고, **언제 돌릴지·얼마나 기다릴지·실패를 어떻게 분류할지는 supervisor 가 소유**. **요청했는데 미배선이면 조용히 통과가 아니라 오류**(F13 AC-1).
  - **`AlignJoinGap`**: `JoinGap(N)`(C-etcd 표: ≤11→7s·≤23→11s·≤41→17s·else 23s)·`JoinWindow(N)=(N+1)*gap` 신설 — 리더 게이트 데드라인을 **클러스터 크기에서 파생**. 고정 타임아웃은 *아직 자기 조인 슬롯이 오지 않은* 노드를 조인 실패로 오판한다(L7).
  - **`ForkSwaps`**: `Deps.SwapBinary` boundary 으로 type-2 스왑 수행. **선언했는데 미배선이면 오류** — 조용히 건너뛰면 체인이 잘못된 바이너리로 포크를 넘어 훨씬 덜 명확한 곳에서 실패한다(F9 AC-3).
  - **`FailureMode` 분류**: `Classify(err)` 신설(etcd stale/join·quorum·fork·rpc 시그니처) — **launch 실패도 분류**(이전엔 전부 `RPCUnready`), HealthGate 가 Mode 를 안 채우면 에러 텍스트에서 파생, **Gate 가 스스로 분류했으면 보존**. 매칭 없으면 `UnknownFailure` — 그럴듯한 오분류는 읽는 사람을 엉뚱한 로그로 보내므로 정직한 "모름"이 낫다(F13 AC-2).
  - **enum 0 값 교정**: `EtcdJoinFailed` 가 iota 0 이라 **빈 `Diagnosis` 가 "EtcdJoinFailed" 로 읽히던** 함정 제거 → `UnknownFailure` 를 0 으로.
  - 재시도 시 `RemoveDataDir:true` 로 stale etcd 상태 정리(F13 AC-3)는 기존 동작 유지·주석 명시. 단위검증 12건.
  - **잔여**: 실제 etcd 리더 게이트 구현체(wemix 계열)와 `SwapBinary` 구현체 배선은 **T5.2**(gwemix 라이브 필요).
- **게이트**: 각 컴포넌트 통합테스트 라이브.

### Phase 4 — Walking Skeleton ★
- ☑ **T4.1 Engine** DI 오케스트레이션(Parse→Applicable skip→fingerprint 기반 env 재사용/BuildEnv→RunSpec→session 저장→종료 시 Teardown) 구현·단위검증 완료. 실제 Place→KeyReg→Genesis→Provision→Supervise→Collect 배선의 **1체인 wbft·local·4노드·tx1** live e2e 는 사용자 환경(체인 바이너리 필요)으로 이월.
- ☑ **T4.2 빌트인 tx/rpc + 어휘 확장(#225)** `NewRegistry(true)` 시드 — **액션 11**: `sendTx`(fee-cap/nonce 인자 포함)·`waitBlock`·`read`(임의 RPC-read 소스 저장)·`deployContract`·`registerContract`·`faucet`·`stopNode`·`startNode`·`restartNode`·`partition`·`healPartition`. **어세션 16**: `chainId`·`blockNumber`·`peerCount`·`balanceAt`·`codeAt`·`nonceAt`·`call`·`txStatus`(F11)·`blockAdvance`·`sameBlockHash`·`baseFee`·`estimateGas`·`gasPrice`·`rpcCall`(체인 어휘를 spec 문자열로, core 무지 유지)·`logs`(eth_getLogs select)·`wsSubscribe`. **스텝 값 바인딩**(`save`/`$ref`/`${..}`). mock RPC 단위검증 + 라이브 `TestEngine_Live_NewVocabulary`(실 gstable 4노드). → 컨트랙트 배포·이벤트·fault·WS·교차-call 전부 표현 가능.
- ☑ **T4.2b 노드 생명주기·fault 액션** `stopNode`/`startNode`/`restartNode` 는 **`testspec.NodeControl` boundary**(`Deps.Nodes`) 경유 — 로컬 엔진은 `engine.NodeController`(런처 앞단에서 노드별 arming·PID 기억, supervisor 와 **procman 공유**해 중도 재기동 노드도 teardown 에 포함) 를 주입하고, **attach 는 nil** 이라 액션이 "이 실행은 노드 프로세스를 소유하지 않는다"고 명확히 실패한다. `procman.StopOne`(SIGTERM→grace→SIGKILL→검증) 신규 — `StopAll` 은 네트워크 전체를 내려 쿼럼 테스트에 못 씀. `partition`/`healPartition` 은 `admin_removePeer`/`admin_addPeer`(+`admin_nodeInfo` 로 enode 획득) 로 **그룹 경계를 넘는 링크를 양방향 절단**. 예제 `fault-node-restart.json`·`fault-partition-fork.json`(`requires:["process"]` → attach 에서 skip). 단위검증(선택·미배선 오류·8회 removePeer·풀메시 heal·재기동 arming 재사용).
- ☑ **T4.2c 자산·컨트랙트 액션** `faucet`(수신자에 wei 공급 — 런타임 생성 서명키는 genesis alloc 에 없어 첫 tx 가스도 못 냄; `from` 미지정 시 **대상 노드의 unlocked coinbase** 를 자금원으로 사용)·`deployContract`(`to` 없는 tx → 영수증의 `contractAddress` 를 `ac.Value` 로 바인딩 — **주소가 영수증에만 있으므로 `sendTx` 로는 표현 불가**, 이것이 별도 액션인 이유)·`registerContract`(배포 주소로의 등록 호출 — `to` 필수·revert 는 항상 실패인 **의도-명시형 sendTx**). 예제 `contract-deploy-and-register.json`(faucet→deploy→`$contract` 로 register→codeAt/txStatus/balanceAt 검증). 단위검증(coinbase 자금원·value hex·필수인자·주소 없는 영수증 오류·배포엔 `to` 미포함).
- ☑ **T4.3 RunSpec 배선** `engine.NewRunSpec(testspec.Deps)` — 인터프리터를 `Deps.RunSpec` 에 바인딩하는 조립 boundary. spec→interpreter→빌트인 어세션→RPC→기록 상태 **실행 수직**을 mock RPC 로 통합검증(pass/fail). BuildEnv 배선(Place→KeyReg→Genesis→Provision→Supervise; 레거시 `pipeline/setup.Plan` 매핑 필요)은 후속.
- ☑ **T4.3b tx 스텝 시맨틱(F11)** sendTx 가 영수증 status 를 검사 — 기본 revert(0x0)=스텝 실패, `expectRevert:true`(또는 `expect:"revert"`)=원자적 negative(성공하면 스텝 실패). 스텝 provenance(tx hash·receipt)를 `ActionCtx` 출력→`StepResult.Hash/Receipt` 로 기록(미사용 필드 활용). mock RPC 단위검증(revert 실패·expectRevert 양방향·provenance 기록·미지 스텝 기록후 실패). → wemix4 negative(GOV 권한거부 등) 이관 기반.
- ☑ **T4.4 BuildEnv 배선 [A안]** `engine.AssemblePlan(plugin, []PlacedNode, genesis, dataRoot, caps)` — place 할당 포트로 `setup.Plan` 을 조립하는 순수함수(레거시 `setup.BuildPlan` 의 `node.Offset` 고정포트 경로 대체). LocalOSAssigned·remote 모드가 이 함수 변경 없이 합성됨. 실 wbft 플러그인으로 단위검증(할당포트 반영·바이너리 폴백·datadir 기본값). **후속 슬라이스**: (T4.4b) keyreg→genesis 피딩(검증자 주소·ExtraData RLP), (T4.4c) provision+supervisor.BringUp 오케스트레이션(launch boundary 주입).
- ☑ **T4.4b GenesisSource boundary** `engine.GenesisSource`(plugin·검증자수→genesis 바이트) + `PresetGenesisSource`(preset `metadata.json` 기반). **핵심 발견**: wbft 계열 ExtraData(RLP 검증자셋)는 코드가 계산하지 않고 preset 에 baked — 패밀리는 템플릿 치환만 함. 따라서 랜덤 keyreg 키만으로는 유효 genesis 불가 → preset 이 검증자셋·ExtraData 소스, keyreg 는 노드 신원/계정 키. 실 wbft 패밀리+최소 템플릿으로 단위검증(치환·Take 검증자 제한·preset 부재 에러). BuildEnv 조립(T4.4c)이 이 boundary 을 호출.
- ☑ **T4.4c BuildEnv 오케스트레이션** `engine.NewBuildEnv(BuildDeps)` — allocate(place)→genesis(GenesisSource)→AssemblePlan→provision(주입 boundary)→`supervisor.BringUp`(launch/health boundary 주입)→NodeSet+Teardown 반환. `Deps.BuildEnv` production wiring 완성. fake allocator/genesis/provision + 실 supervisor(fake launch/health)로 파이프라인 스레딩 단위검증(4노드 조립·genesis 검증자수·plan 포트·teardown·에러전파). **남은 것**: (1) 실 `Provision` 파일생성(config.toml·keystore·nodekey; 레거시 `setup.provision` 참조), (2) 실 4노드 라이브 e2e(바이너리 필요, 사용자 환경). → **코드 수준 walking skeleton 조립 완료**(BuildEnv+RunSpec 모두 `Deps` 배선).
- ☑ **T4.4d GenesisSource 라이브검증** `TestPresetGenesisSource_Live_GstableInit`(GSTABLE_BIN 게이트) — `PresetGenesisSource`+실 stablenet 플러그인으로 `keys/preset` 에서 genesis 생성 → **실 gstable v1.1.0 `init` 통과**("Successfully wrote genesis state", `<datadir>/gstable/chaindata` 생성). 이월했던 "랜덤키만으로 유효 genesis 불가·preset 필요" 가정을 실 바이너리로 확정. CI 는 바이너리 부재로 SKIP(green 유지). **실 4노드 라이브 기동(Provision+driver Launch+HealthGate)은 T4.4e 후속**.
- ☑ **T4.4e RunSpec 라이브 e2e (walking skeleton 증명)** `TestRunSpec_Live_Stablenet`(GSTABLE_BIN 게이트) — 실 gstable 로 4노드 stablenet 기동(`setup.Launch` fixture) → `engine.NewRunSpec`(인터프리터+빌트인) 으로 spec 실행: **sendTx(노드서명+영수증)·chainId==8283·blockNumber≥1 어세션 전부 실 RPC 대상 → status pass** → teardown 고아0. 로컬 39.7s 통과. **발견**: (1) geth IPC 유닉스소켓 경로 <104자 제약 → dataRoot 는 `/tmp/cblXXX`(t.TempDir 불가), (2) wbft 블록생성 웜업 ~35s → 헬스게이트 넉넉히 폴링. → **DSL 이 실 체인에서 실행·검증됨(실행 수직 라이브 완성)**. 남은 것: BuildEnv 자체의 실 launcher(place 포트 기동)는 T4.4f 선택.
- ☑ **T4.4f BuildEnv 실 launcher (기동 수직 라이브)** `engine.LocalLauncher`(preset 기반 arming: config 렌더·신원설치·datadir init·기동) + `armSpecs`(순수, 단위검증: validator --unlock/--nodekey, static-node enode 가 plan p2p 포트 사용, endpoint 는 unlock 없음). `TestBuildEnv_Live_Stablenet`(GSTABLE_BIN 게이트): 실 allocator+PresetGenesisSource+LocalLauncher+블록전진 헬스게이트로 `NewBuildEnv` → **실 4노드 stablenet 을 allocator 할당 포트(node1 :8600)로 기동·헬스통과·teardown 고아0**. 로컬 통과. → **Engine 기동 수직(BuildEnv)도 실 체인 라이브 증명. RunSpec(#195)+BuildEnv 로 Engine.Run 전 구간 라이브 커버.** (짧은 session root 로 IPC 소켓 경로 <104자 유지.)
- ☑ **T4.5 Engine 최상위 배선 (capstone)** `engine.NewLocalEngine(LocalConfig)` — allocator·PresetGenesisSource·LocalLauncher·`NewBlockAdvanceGate`·인터프리터·session 을 하나의 `engine.Deps` 로 조립하는 실행 가능한 진입점(CLI/MCP 가 호출). `NewBlockAdvanceGate`(head≥target 폴링, 로컬 non-etcd 라이브니스), `applicableTo`/`validatorReqs` 헬퍼. 단위검증(applicable 매칭·구성검증·미지체인 에러). **`TestEngine_Live_FullRun`(GSTABLE_BIN 게이트) capstone**: `NewLocalEngine`→`Engine.Run([spec])` 한 번으로 **실 gstable 4노드 기동→spec 실행(chainId/blockNumber)→teardown→session.json 저장, summary.pass=1** 검증. 로컬 4.5s 통과·고아0. → **walking skeleton 을 실행 가능한 단일 진입점으로 종료(Engine.Run 전체가 실 체인에서 동작).**
- ☑ **T4.6 launcher 파일 물질화를 provision.Provisioner 경유 [B안]** `LocalLauncher` 가 genesis·per-node config 를 `provision.Provisioner`(`FileSink`) 로 물질화 — 기존 ad-hoc `os.WriteFile(genesis)`+`driver.Provision(config)` 제거. **upload-if-absent**(기존 파일 재사용) + **원격 `RemoteFileSink` 로 교체할 boundary**(슬라이스 C 대비) 확보. 순수 `materialize` 헬퍼로 분리해 recording FileSink 로 단위검증(genesis+config 기록·재사용 스킵). 리팩터 후 실 gstable 라이브 2종(BuildEnv/FullRun) 재통과·고아0. 프로덕션 engine 은 이제 레거시 `setup.Provision/Launch` 미호출(`setup.Plan` 타입만 사용).

### Phase 5 — 수직 슬라이스 (매번 통합 유지)
- ◐ **T5.1 remote [C안]** ☑ `driver.RemoteFileSink`(`provision.FileSink` 구현: SSH `test -f` 존재확인 + `ProvisionFile` base64 전송) — B의 `LocalLauncher.Sink` boundary 에 그대로 주입 가능(upload-if-absent). ☑ launcher init 을 `driver.Initializer` capability 경유로 라우팅(local/remote 드라이버 공통) → `LocalLauncher{Driver:RemoteDriver, Sink:RemoteFileSink}` 로 **원격 기동 가능**. 단위검증: RemoteFileSink(exist/absent/transport err·base64 write·`provision.FileSink` 만족), launcher 전체 합성(materialize→init(Initializer)→launch, fake driver/sink). 로컬 live 2종 재통과(init 라우팅 변경 후·고아0). **남은 것**: 실 원격 SSH 호스트 대상 라이브 e2e(사용자 환경·SSH 필요). · ☐ **T5.2 업그레이드 멀티바이너리**(wemix+wbft) · ☑ **T5.3 attach** `engine.NewAttachEngine(AttachConfig{Chain,RPCURLs,ArtifactRoot})` + `NewAttachBuildEnv`(attach.Build 로 RPC 엔드포인트에서 NodeSet 구성, **기동/teardown 없음** — attach 는 노드를 만들지 않음). 바이너리·preset 불필요 → **mock RPC 로 Engine.Run 전체 e2e 가 CI 에서 실행**(chainId/blockNumber 어세션 pass·미적용 spec skip·구성검증). walking skeleton 실행수직을 바이너리 없이 CI 커버하는 첫 통합. · ☑ **T5.4 stablenet**(ACL 플러그인·Core 무변경) — 거버넌스 read 시나리오를 DSL 로 표현·엔진 실행 CI 검증(예제 spec + govbind calldata mock RPC e2e). · ☐ **T5.5 wemix4 이관**(DSL).

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
    keyreg/            [NEW]   C2 · 키 레지스트리 + BLSDeriver boundary
    collector/         [NEW]   C4 · live tail·원격tail·chainstate·bp참여·분기
    supervisor/        [NEW]   C3 · 기동·헬스게이트·teardown·복구
    driver/            [EXT]   C7 · Transport boundary(+kill검증·key_file·upload-if-absent)
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
  server-set.sample.yaml  [KEEP]  · 실파일은 gitignore(SSH 자격증명)
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
