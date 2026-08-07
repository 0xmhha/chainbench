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
| 2 | **Session(M1)** | 모든 기록의 정본, 기반 | ◐ |
| 3 | **Place(M2)+용량검증** | 배치·포트(고정포트 충돌 제거) | ☑ |
| 4 | **procman 배선+확장(M6 핵심)** | 최우선 안전 갭(검증 없는 Kill→고아 위험) | ☑ |
| 5 | **testspec Parse+Fingerprint** | spec 실행·env 재사용 | ☑ |
| 6 | **keyreg(M3) + assert funcs** | 신원·검증 | ☑ |
| 7 | **Provisioner(M5) → Supervisor(M6)** | 물질화·기동·teardown | ☑ |
| 8 | **Collector RPC-min + Interpreter-min** | 스켈레톤 최소 관측·실행 | ☑ |
| 9 | **★ Engine walking skeleton** | 첫 통합 증명(1체인·local·4노드·tx1) — **실 gstable 라이브 e2e 통과(RunSpec 실행 수직)** | ☑ |
| 10+ | Collector 심화 → remote → 업그레이드 → attach → stablenet → wemix4 이관 → 표면 | 수직 슬라이스 확장 | ☐ |

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
- ◐ **T3.2 Supervisor**(오케스트레이션·teardown·procman배선 단위완료; 실4노드 라이브 헬스게이트는 Phase4) 4노드 wbft 기동·헬스게이트(블록생성·etcd 리더)·teardown(**고아0+datadir삭제**)·**procman을 실제 stop 경로에 배선**(leak=0).
- ◐ **T3.3 Collector**(RPC 스냅샷·WaitLog poll 단위완료; live tail·bp참여·reorg는 후속) 로컬 **live tail**(스캔→tail)+**원격 SSH tail**+RPC 스냅샷+**bp참여·분기(reorg)검출**. attach=RPC-only.
- ☐ **T3.4 Session 저장·재사용** env.json·fingerprint 재사용·records(spec/steps/assert/status).
- **게이트**: 각 컴포넌트 통합테스트 라이브.

### Phase 4 — Walking Skeleton ★
- ◐ **T4.1 Engine** DI 오케스트레이션(Parse→Applicable skip→fingerprint 기반 env 재사용/BuildEnv→RunSpec→session 저장→종료 시 Teardown) 구현·단위검증 완료. 실제 Place→KeyReg→Genesis→Provision→Supervise→Collect 배선의 **1체인 wbft·local·4노드·tx1** live e2e 는 사용자 환경(체인 바이너리 필요)으로 이월.
- ◐ **T4.2 빌트인 tx/rpc** `NewRegistry(true)` 시드: 액션 `sendTx`(노드서명 전송+영수증 폴링, `wait:false` 조기반환), 어세션 `chainId`/`blockNumber`/`peerCount`/`balanceAt`/`codeAt`(대상노드 RPC 읽기→assert 프리미티브 비교, `compare` 오버라이드). mock RPC(httptest) 단위검증 완료. 컨트랙트 배포·crosstx 등 심화 액션은 후속.
- ◐ **T4.3 RunSpec 배선** `engine.NewRunSpec(testspec.Deps)` — 인터프리터를 `Deps.RunSpec` 에 바인딩하는 조립 seam. spec→interpreter→빌트인 어세션→RPC→기록 상태 **실행 수직**을 mock RPC 로 통합검증(pass/fail). BuildEnv 배선(Place→KeyReg→Genesis→Provision→Supervise; 레거시 `pipeline/setup.Plan` 매핑 필요)은 후속.
- ◐ **T4.4 BuildEnv 배선 [A안]** `engine.AssemblePlan(plugin, []PlacedNode, genesis, dataRoot, caps)` — place 할당 포트로 `setup.Plan` 을 조립하는 순수함수(레거시 `setup.BuildPlan` 의 `node.Offset` 고정포트 경로 대체). LocalOSAssigned·remote 모드가 이 함수 변경 없이 합성됨. 실 wbft 플러그인으로 단위검증(할당포트 반영·바이너리 폴백·datadir 기본값). **후속 슬라이스**: (T4.4b) keyreg→genesis 피딩(검증자 주소·ExtraData RLP), (T4.4c) provision+supervisor.BringUp 오케스트레이션(launch seam 주입).
- ◐ **T4.4b GenesisSource seam** `engine.GenesisSource`(plugin·검증자수→genesis 바이트) + `PresetGenesisSource`(preset `metadata.json` 기반). **핵심 발견**: wbft 계열 ExtraData(RLP 검증자셋)는 코드가 계산하지 않고 preset 에 baked — 패밀리는 템플릿 치환만 함. 따라서 랜덤 keyreg 키만으로는 유효 genesis 불가 → preset 이 검증자셋·ExtraData 소스, keyreg 는 노드 신원/계정 키. 실 wbft 패밀리+최소 템플릿으로 단위검증(치환·Take 검증자 제한·preset 부재 에러). BuildEnv 조립(T4.4c)이 이 seam 을 호출.
- ◐ **T4.4c BuildEnv 오케스트레이션** `engine.NewBuildEnv(BuildDeps)` — allocate(place)→genesis(GenesisSource)→AssemblePlan→provision(주입 seam)→`supervisor.BringUp`(launch/health seam 주입)→NodeSet+Teardown 반환. `Deps.BuildEnv` production wiring 완성. fake allocator/genesis/provision + 실 supervisor(fake launch/health)로 파이프라인 스레딩 단위검증(4노드 조립·genesis 검증자수·plan 포트·teardown·에러전파). **남은 것**: (1) 실 `Provision` 파일생성(config.toml·keystore·nodekey; 레거시 `setup.provision` 참조), (2) 실 4노드 라이브 e2e(바이너리 필요, 사용자 환경). → **코드 수준 walking skeleton 조립 완료**(BuildEnv+RunSpec 모두 `Deps` 배선).
- ◐ **T4.4d GenesisSource 라이브검증** `TestPresetGenesisSource_Live_GstableInit`(GSTABLE_BIN 게이트) — `PresetGenesisSource`+실 stablenet 플러그인으로 `keys/preset` 에서 genesis 생성 → **실 gstable v1.1.0 `init` 통과**("Successfully wrote genesis state", `<datadir>/gstable/chaindata` 생성). 이월했던 "랜덤키만으로 유효 genesis 불가·preset 필요" 가정을 실 바이너리로 확정. CI 는 바이너리 부재로 SKIP(green 유지). **실 4노드 라이브 기동(Provision+driver Launch+HealthGate)은 T4.4e 후속**.
- ☑ **T4.4e RunSpec 라이브 e2e (walking skeleton 증명)** `TestRunSpec_Live_Stablenet`(GSTABLE_BIN 게이트) — 실 gstable 로 4노드 stablenet 기동(`setup.Launch` fixture) → `engine.NewRunSpec`(인터프리터+빌트인) 으로 spec 실행: **sendTx(노드서명+영수증)·chainId==8283·blockNumber≥1 어세션 전부 실 RPC 대상 → status pass** → teardown 고아0. 로컬 39.7s 통과. **발견**: (1) geth IPC 유닉스소켓 경로 <104자 제약 → dataRoot 는 `/tmp/cblXXX`(t.TempDir 불가), (2) wbft 블록생성 웜업 ~35s → 헬스게이트 넉넉히 폴링. → **DSL 이 실 체인에서 실행·검증됨(실행 수직 라이브 완성)**. 남은 것: BuildEnv 자체의 실 launcher(place 포트 기동)는 T4.4f 선택.
- ☑ **T4.4f BuildEnv 실 launcher (기동 수직 라이브)** `engine.LocalLauncher`(preset 기반 arming: config 렌더·신원설치·datadir init·기동) + `armSpecs`(순수, 단위검증: validator --unlock/--nodekey, static-node enode 가 plan p2p 포트 사용, endpoint 는 unlock 없음). `TestBuildEnv_Live_Stablenet`(GSTABLE_BIN 게이트): 실 allocator+PresetGenesisSource+LocalLauncher+블록전진 헬스게이트로 `NewBuildEnv` → **실 4노드 stablenet 을 allocator 할당 포트(node1 :8600)로 기동·헬스통과·teardown 고아0**. 로컬 통과. → **Engine 기동 수직(BuildEnv)도 실 체인 라이브 증명. RunSpec(#195)+BuildEnv 로 Engine.Run 전 구간 라이브 커버.** (짧은 session root 로 IPC 소켓 경로 <104자 유지.)
- ☑ **T4.5 Engine 최상위 배선 (capstone)** `engine.NewLocalEngine(LocalConfig)` — allocator·PresetGenesisSource·LocalLauncher·`NewBlockAdvanceGate`·인터프리터·session 을 하나의 `engine.Deps` 로 조립하는 실행 가능한 진입점(CLI/MCP 가 호출). `NewBlockAdvanceGate`(head≥target 폴링, 로컬 non-etcd 라이브니스), `applicableTo`/`validatorReqs` 헬퍼. 단위검증(applicable 매칭·구성검증·미지체인 에러). **`TestEngine_Live_FullRun`(GSTABLE_BIN 게이트) capstone**: `NewLocalEngine`→`Engine.Run([spec])` 한 번으로 **실 gstable 4노드 기동→spec 실행(chainId/blockNumber)→teardown→session.json 저장, summary.pass=1** 검증. 로컬 4.5s 통과·고아0. → **walking skeleton 을 실행 가능한 단일 진입점으로 종료(Engine.Run 전체가 실 체인에서 동작).**
- ☑ **T4.6 launcher 파일 물질화를 provision.Provisioner 경유 [B안]** `LocalLauncher` 가 genesis·per-node config 를 `provision.Provisioner`(`FileSink`) 로 물질화 — 기존 ad-hoc `os.WriteFile(genesis)`+`driver.Provision(config)` 제거. **upload-if-absent**(기존 파일 재사용) + **원격 `RemoteFileSink` 로 교체할 seam**(슬라이스 C 대비) 확보. 순수 `materialize` 헬퍼로 분리해 recording FileSink 로 단위검증(genesis+config 기록·재사용 스킵). 리팩터 후 실 gstable 라이브 2종(BuildEnv/FullRun) 재통과·고아0. 프로덕션 engine 은 이제 레거시 `setup.Provision/Launch` 미호출(`setup.Plan` 타입만 사용).

### Phase 5 — 수직 슬라이스 (매번 통합 유지)
- ◐ **T5.1 remote [C안]** ☑ `driver.RemoteFileSink`(`provision.FileSink` 구현: SSH `test -f` 존재확인 + `ProvisionFile` base64 전송) — B의 `LocalLauncher.Sink` seam 에 그대로 주입 가능(upload-if-absent). ☑ launcher init 을 `driver.Initializer` capability 경유로 라우팅(local/remote 드라이버 공통) → `LocalLauncher{Driver:RemoteDriver, Sink:RemoteFileSink}` 로 **원격 기동 가능**. 단위검증: RemoteFileSink(exist/absent/transport err·base64 write·`provision.FileSink` 만족), launcher 전체 합성(materialize→init(Initializer)→launch, fake driver/sink). 로컬 live 2종 재통과(init 라우팅 변경 후·고아0). **남은 것**: 실 원격 SSH 호스트 대상 라이브 e2e(사용자 환경·SSH 필요). · ☐ **T5.2 업그레이드 멀티바이너리**(wemix+wbft) · ☑ **T5.3 attach** `engine.NewAttachEngine(AttachConfig{Chain,RPCURLs,ArtifactRoot})` + `NewAttachBuildEnv`(attach.Build 로 RPC 엔드포인트에서 NodeSet 구성, **기동/teardown 없음** — attach 는 노드를 만들지 않음). 바이너리·preset 불필요 → **mock RPC 로 Engine.Run 전체 e2e 가 CI 에서 실행**(chainId/blockNumber 어세션 pass·미적용 spec skip·구성검증). walking skeleton 실행수직을 바이너리 없이 CI 커버하는 첫 통합. · ☐ **T5.4 stablenet**(ACL 플러그인만·Core 무변경) · ☐ **T5.5 wemix4 이관**(DSL).

### Phase 6 — 표면·마감
- ☐ **T6.1 Capabilities**(registry/chains) · ☐ **T6.2 MCP 결과연동**(F14) · ☐ **T6.3 dashboard**(F15) · ☐ **T6.4** 백프레셔(O7)·`-race` 게이트(O6)·CLI 정리.

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
