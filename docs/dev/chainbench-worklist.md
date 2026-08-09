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
| 2 | **Session(M1)** | 모든 기록의 정본, 기반 | ☑ |
| 3 | **Place(M2)+용량검증** | 배치·포트(고정포트 충돌 제거) | ☑ |
| 4 | **procman 배선+확장(M6 핵심)** | 최우선 안전 갭(검증 없는 Kill→고아 위험) | ☑ |
| 5 | **testspec Parse+Fingerprint** | spec 실행·env 재사용 | ☑ |
| 6 | **keyreg(M3) + assert funcs** | 신원·검증 | ☑ |
| 7 | **Provisioner(M5) → Supervisor(M6)** | 물질화·기동·teardown | ☑ |
| 8 | **Collector RPC-min + Interpreter-min** | 스켈레톤 최소 관측·실행 | ☑ |
| 9 | **★ Engine walking skeleton** | 첫 통합 증명(1체인·local·4노드·tx1) — **실 gstable 라이브 e2e 통과(RunSpec 실행 수직)** | ☑ |
| 10+ | 수직 슬라이스 확장: remote☑ · attach☑ · 표면(capability/CLI/MCP/-race/백프레셔)☑ · Collector 심화☑ · stablenet☑ · 업그레이드☐ · wemix4 이관☐ | 수직 슬라이스 확장 | ◐ |

---

## 1b. 진행 현황 요약 (2026-08-07 기준)

**walking skeleton 완성 · 재설계 엔진(H1)이 CLI·MCP 양쪽에서 도달 가능 · 실행 수직 전체가 CI(mock/attach)+라이브(gstable) 커버.**

**완료 (PR):**
- Phase 4 walking skeleton — Engine 오케스트레이션·빌트인 tx/rpc·RunSpec·BuildEnv(AssemblePlan/GenesisSource/orchestration)·최상위 `NewLocalEngine`. **실 gstable 4노드 라이브 e2e**(BuildEnv 기동 + RunSpec 실행 + capstone `Engine.Run`) 통과. (#188~#197)
- launcher 물질화 `provision.Provisioner` 경유(B) (#198)
- SSH key_file 인증 + `kill -0` 종료검증(procman) (D) (#199)
- remote 슬라이스: `RemoteFileSink` + 원격 가능 launcher(Initializer 라우팅) (C) (#200)
- attach 모드: `NewAttachEngine`(RPC-only, mock RPC 로 CI e2e) (#201)
- Phase 6 표면: capability 게이팅(spec `requires`)·`-race` 게이트·CLI `chainbench run` (#202)
- Phase 6 표면: MCP `chainbench_run`·공유 `ReadSessionSummary`·obs 백프레셔(O7) 테스트 (#203, 리뷰중)

**남은 작업:**
- ~~**T6.3 dashboard**~~ ☑ 엔진이 obs 이벤트 emit(`Deps.Emit`/`Bus`) → `chainbench run --dashboard <url>` 가 `dashboard.Forward` 로 chainbenchd 에 스트리밍(라이브). + ☑ **완료 세션 디스크 조회**(F15 AC3): `session.List`/`SessionFilePath`/`ChainstatePaths` + dashboard `/api/sessions`·`/api/sessions/{id}`(session.json)·`/api/sessions/{id}/chainstate`(chainstate.jsonl), `chainbenchd -artifact-root` 로 활성화. attach+mock RPC end-to-end + httptest 로 CI 커버.
- **T5.2 업그레이드 멀티바이너리**(wemix+wbft 핸드오프) — gwemix+etcd 바이너리 필요, 이 환경 라이브 검증 제한.
- ☑ **T5.4 stablenet**(ACL 플러그인·Core 무변경 검증) — stablenet 거버넌스 시나리오가 DSL 엔진에서 실행됨을 CI 검증(예제 `stablenet-governance-read.json` + govbind `proposals(uint256)` calldata·mock RPC e2e `call` 어세션 pass). core 무변경. 실 gstable 라이브(RunSpec/BuildEnv)는 기존 게이트 테스트로 커버.
- **T5.5 wemix4 이관**(레거시 스위트 → DSL) — 대규모.
- **Collector 심화(T3.3)** — ☑ 로컬 live tail(스캔→tail·부분줄 안전)·bp참여 집계(head producer·window prune)·fork/reorg 검출(높이별 hash 불일치)·**엔진 배선**(local/attach 이 `Bus` 설정 시 collection 실행→chainstate·로그를 obs 로 미러, teardown 시 정지)·**chainstate 세션 영속화**(`chainstate/chainstate.jsonl` — F10/F15 jsonl+obs 미러). ☐ 원격 SSH tail(사용자 SSH 환경 필요, T5.1 계열).
- **실 원격 SSH 호스트 라이브 e2e**(RemoteFileSink+RemoteDriver) — SSH 대상 필요(사용자 환경).
- **레거시 경로 정리** — `pipeline/setup`·`pipeline/testrun` 등은 아직 CLI 일부(setup/test/verify)와 공존; 신규 엔진으로 완전 대체는 후속.

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
- ☑ **T1.5 procman EXTEND** `{PID,datadir}`·원격PID·`Alive`. (stop 경로 배선은 T3.2) — F13
- ☑ **T1.6 keyreg** 랜덤/기존/원격다운로드 통합 + **BLSDeriver seam**(외부 bootnode 캡슐화, 부재 시 오류). — F2
- **게이트**: 단위 100% + 동시 모듈 `-race`.

### Phase 2 — Transport
- ◐ **T2.1** Local/Remote Transport를 driver 위에 형식화 + **종료검증(`kill -0`)** + **key_file 인증**. **[D안]** ☑ **key_file 인증**: `remote.Credentials` 에 `PrivateKey`/`Passphrase` 추가, `authMethods`(key 우선+password, 최소 1개 필수, 키자료 미노출), `LoadPrivateKey`(0600 강제·insecure perm 거부). deploy `credentials.go` 가 `key_file`(+ `CHAINBENCH_REMOTE_KEY_FILE`/`_PASSPHRASE` env) 를 `remote.Credentials.PrivateKey` 로 로드 — "future phase 예약" 게이트 제거. 단위검증(authMethods 4케이스·키 미노출·perm 거부·For key_file). ☑ **종료검증(`kill -0`)**: 이미 `procman.Alive`(signal 0)+`StopAll`(SIGTERM→wait→SIGKILL→poll→leak 보고)로 구현, 엔진 teardown 이 이를 경유(라이브 고아0 확인). **남은 것**: driver 위 Transport 타입 형식화(C 원격 슬라이스와 함께).
- ☑ **T2.2 upload-if-absent**(로컬·FileSink; 원격 SSH sink는 remote 슬라이스) `test -f` 존재확인 → 재사용/업로드(현재 항상-업로드 or 항상-읽기).
- **게이트**: 단위(모의) + 통합 1건(로컬 더미 프로세스 검증-종료·고아0).

### Phase 3 — Middle 통합(라이브)
- ☑ **T3.1 Provisioner** datadir+키+genesis+config 물질화(local·remote 동일경로)+upload-if-absent.
- ☑ **T3.2 Supervisor** 오케스트레이션·teardown·procman배선 단위완료 + **실4노드 라이브 헬스게이트(블록전진 gate)로 Phase4 BuildEnv e2e 에서 검증**(고아0). etcd 리더 게이트는 wemix 계열(gwemix/etcd 필요) 후속.
- ◐ **T3.3 Collector** ☑ RPC 스냅샷(height·peers)·WaitLog poll·**로컬 live tail**(offset 증분·스캔→tail·부분줄 미방출·`Deps.OnLine` obs 미러 seam)·**bp참여 집계**(head producer 샘플→`BPParticipation`, `Deps.BPWindow` 로 bounded prune)·**fork/reorg 검출**(높이별 first-seen hash, 노드 간/샘플 간 불일치→`Forked`). 단위+`-race` 검증. ☑ **엔진 배선**(`withCollection` 이 local/attach 의 BuildEnv 를 래핑 — `Bus` 설정 시 env 별 collector 실행, RPC probe(`rpc.Client.HeadBlock`)로 샘플, chainstate 스냅샷·tail 로그를 obs 로 미러, teardown 시 정지; attach+mock RPC 로 chainstate 이벤트 e2e 검증)·**chainstate 세션 영속화**(collection 이 스냅샷을 `chainstate/chainstate.jsonl` 로 기록 — F10/F15 jsonl+obs 미러, 완료 세션 재생용; best-effort). ☐ **원격 SSH tail**(사용자 SSH 환경 필요, T5.1 계열). attach=RPC-only(로컬 로그 없음→tail no-op).
- ☑ **T3.4 Session 저장·재사용** `session.json`·env fingerprint 재사용(엔진 오케스트레이션이 fingerprint 로 env 재사용)·records(spec/steps/assert/status). 엔진 단위·라이브 e2e·attach e2e 로 검증(summary.pass 판정).
- **게이트**: 각 컴포넌트 통합테스트 라이브.

### Phase 4 — Walking Skeleton ★
- ☑ **T4.1 Engine** DI 오케스트레이션(Parse→Applicable skip→fingerprint 기반 env 재사용/BuildEnv→RunSpec→session 저장→종료 시 Teardown) 구현·단위검증 완료. 실제 Place→KeyReg→Genesis→Provision→Supervise→Collect 배선의 **1체인 wbft·local·4노드·tx1** live e2e 는 사용자 환경(체인 바이너리 필요)으로 이월.
- ☑ **T4.2 빌트인 tx/rpc** `NewRegistry(true)` 시드: 액션 `sendTx`(노드서명 전송+영수증 폴링, `wait:false` 조기반환)·`waitBlock`(목표 높이까지 폴링, args: target/on/timeout/pollInterval), 어세션 `chainId`/`blockNumber`/`peerCount`/`balanceAt`/`codeAt`/`nonceAt`/`call`(eth_call 0x-hex 결과)/`txStatus`(영수증 status 0x1/0x0 — F11 negative)(대상노드 RPC 읽기→assert 프리미티브 비교, `compare` 오버라이드). mock RPC(httptest) 단위검증. 컨트랙트 배포·crosstx 등 추가 심화 액션은 후속(이관 필요 시 확장).
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
- ☐ **T6.5 문서 경로 드리프트 정리** T0.0(`pkg/`→`internal/`) 완료 후에도 forward-looking 마이그레이션 문서가 삭제된 `pkg/` 경로를 참조 → 후속 작업 착수 시 혼선. 실제 대상은 모두 `internal/` 하위에 존재(경로만 stale). 대상: `docs/dev/repro-migration-remaining.md`(L48 `pkg/chains/stablenet/overlays/account-extra.json`, L70·73 `pkg/consensus/poa`), `docs/dev/wemix4-migration-plan.md`(L45 `pkg/core/topology`, L47 `pkg/consensus/upgrade`, L48·137 `pkg/chains/wbft/genesis.json`, L50 `pkg/core/rpc`)를 `internal/…` 로 정정. (감사 발견 2026-08-09 — 어느 리스트에도 없던 유일 미추적 항목.)
- ☑ **T6.1 Capabilities** spec `requires: [cap...]` 필드 + 엔진 capability 게이팅: `satisfies`/`applicableWithCaps`(체인 매칭 ∧ 필요 capability ⊆ 타깃 제공). NewLocalEngine 은 `localCapabilities`(manifest.Capabilities + "ws"), NewAttachEngine 은 `["rpc"]` 를 제공집합으로. 미충족 spec 은 skip(fail 아님). 단위(satisfies 4케이스·applicableWithCaps) + attach mock RPC e2e(ws 요구 spec → rpc-only attach 에서 skip). `requires` 는 env 를 바꾸지 않으므로 fingerprint 미포함. · ☑ **T6.2 MCP 결과연동**(F14) `chainbench_run` MCP 도구 — attach 모드로 DSL spec 실행 후 session 판정 반환(CLI `run` 의 MCP 대응). `engine.ReadSessionSummary`(session.json 단일 리더)를 CLI·MCP 가 공유(중복 제거). attach+mock RPC 로 CI 테스트(pass=1·인자 검증). · ☑ **T6.3 dashboard**(F15) — 엔진 오케스트레이션이 obs 이벤트 emit(`Deps.Emit`/`Network`, nil-safe no-op): run started·building environment·environment reused·running spec·spec `<status>`·run complete. `NewLocalEngine`/`NewAttachEngine` 가 `Bus` 옵션으로 배선, `chainbench run --dashboard <url>` 가 `dashboard.Forward` 로 chainbenchd `/api/events` 에 스트리밍(종료 시 flush). 엔진 emit 단위 + attach mock RPC → Forward → dashboard.Server end-to-end CI 테스트. obs.Bus 백프레셔(bounded buffer·drop-on-full)로 관측이 실행을 막지 않음. ☑ **완료 세션 디스크 조회**(F15 AC3): `session.List`/`SessionFilePath`/`ChainstatePaths`(레이아웃 소유자) + dashboard `/api/sessions`·`/api/sessions/{id}`(session.json 판정)·`/api/sessions/{id}/chainstate`(chainstate.jsonl → JSON 배열), id 는 단일 세그먼트 검증(traversal 방지), `chainbenchd -artifact-root` 로 활성화(httptest 검증). · ◐ **T6.4** 백프레셔(O7)·`-race` 게이트(O6)·CLI 정리. ☑ **`-race` 게이트**: CI test 잡을 `go test -race ./...` 로(경합 CI 상시 검출; 풀런 ~71s 로 기존과 동등 — 느린 `verify` 테스트가 CPU 아닌 대기 바운드). ☑ **백프레셔(O7)** obs.Bus 는 bounded buffer(256)·drop-on-full(non-blocking select)·`Dropped` 카운터로 이미 구현 — 회귀 테스트 추가(느린 구독자에서 blocking 없이 drop·정확한 카운트 검증). · ☑ **CLI 엔진 배선**: `chainbench run [spec.json…]` 명령 — `--rpc`(attach: `NewAttachEngine`) 또는 `--binary`(local: `NewLocalEngine`) 로 DSL spec 을 엔진에 실행, `session.json` 요약 출력·실패 시 non-zero exit. attach+mock RPC 로 CI 테스트(pass=1·실패 non-zero·모드 검증). → 재설계 엔진이 CLI 에서 도달 가능.

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
