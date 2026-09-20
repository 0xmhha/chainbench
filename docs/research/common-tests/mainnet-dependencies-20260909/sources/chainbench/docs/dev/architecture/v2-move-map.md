# 아키텍처 v2 이동표 — 실측 기반

> **[현행 설계]** [[architecture-v2]](architecture-v2.md) 의 함수·파일 이동 계획.
> 실측 2026-08-25: 재편 대상 8개 패키지의 심볼(함수·메서드·타입) 전수를 AST 로
> 뽑았다(테스트 파일 제외). 작업 상태는 [[chainbench-worklist]] §1h 이 정본이다.

## 실측 규모

| 패키지 | 심볼 | 재편에서의 운명 |
|---|---|---|
| `internal/app` | 139 | 워크플로 층으로 슬림화 — 배선은 core 로, 렌더링은 표면으로 |
| `internal/netcompose` | 59 | **해체** — chainsetup·netmap·process 로 분산 (§V5 표) |
| `internal/engine` | 109 | 이분 — 구성 책임은 chainsetup, 테스트 수행은 testengine |
| `internal/core/target` | 12 | `core/machine` 으로 개명 (전량) |
| `internal/core/driver` | 47 | 잔류 + 조회 원시 기능 보강 |
| `internal/serverset` | 48 | netmap 으로 흡수 |
| `internal/core/keyring` | 91 | 이분 — 키 역학 잔류, 저장은 `keyring/store` |
| `internal/core/netmap` | 36 | 잔류 (순수 할당 코어) + wrapper·enode 추가 |

## V1.1 — `core/target` → `core/machine` (12 심볼 전량)

| 현재 | 새 이름 | 비고 |
|---|---|---|
| `target.TargetSpec` | `machine.Spec` | stutter 해소 |
| `target.Target` | `machine.Access` | 능력 손잡이(Files·Driver)라는 실체를 이름에 |
| `target.TargetKind` | `machine.Kind` | |
| `target.ParseTarget` | `machine.Parse` | |
| `target.ServerLookup` | `machine.Lookup` | |
| `Resolve/ResolveWith/ResolveWithMap/resolveOver/mapCredentials/parseHostColonPath/IsRemote` | 동명 유지 | 리시버만 `Spec` |

소비자(2026-08-25 기준): app(keyring.go·serverconf.go·net_steps.go), netcompose,
keyringcmd/keyflags.go, serverset, chainsetup, upgrade, wemix/deploy. 한 PR 원자 개명.

## V2 — netmap wrapper 와 흡수

| 태스크 | 옮기는 것 | 출처 → 목적지 |
|---|---|---|
| V2.1 (신설) | 서버명→손잡이 결합, `--docker` 치환, 치환 보고 | (신규) `netmap` wrapper |
| V2.2 | `RingRef.open`·`RingImportIn.source` 의 배선 | `app/keyring.go` → wrapper 호출 |
| V2.3 | `Workspace.addrMap`·`Workspace.resolveTarget` | `netcompose/workspace.go` → wrapper 호출 |
| V2.4 | serverset 48 심볼 (Load·ByName·Fleet·Placement·Credentials·SetLookup·LocalMap…) | `internal/serverset` → `internal/core/netmap` 하위로 |
| V2.5 | enode·static-nodes 조합 (`Workspace.Config` 내부의 enode 함수, `nodeHost`) | `netcompose/steps_compose.go` → `netmap` (공개키는 입력) |

## V3 — keyring 이분 (91 심볼)

| 파일 | 심볼 | 목적지 | 근거 |
|---|---|---|---|
| `identity.go`·`privatekey.go`·`bls.go` | 15 | **keyring 잔류** | 키 역학(파생·검증) |
| `source.go` | 14 | **keyring 잔류** | 키의 출처(hex·니모닉·파일)는 키 관심사 |
| `generate.go` | 21 | `keyring/store` | 링 폴더 생성·기록 (GenerateAt·ImportRing·writePreset…) |
| `preset.go` | 16 | `keyring/store` | 인덱스 읽기·검증 로드 (LoadPresetAt…) — `Entry.Verify` 파생 부분은 잔류 검토 |
| `ring.go`·`backend.go`·`password.go` | 25 | `keyring/store` | 저장 형식·암호화·비밀번호 |
| (app 에서 이동) 링 위치 해석 | — | `keyring/store` | 플래그>env>기본 우선순위 (V3.2) |

## V4 — process 대장

| 태스크 | 내용 | 출처 |
|---|---|---|
| V4.1 | driver 조회 보강: 바이너리/pid 실행 여부·포트·명령 수행(결과 회수) | 신규 (PortProber·ExecWithInput 은 기존) |
| V4.2 | `process.Manager` 대장: 머신·바이너리·명령·pid 기록과 조회 | 신규 |
| V4.3 | `Workspace.checkVacant`·`scanOccupancy`·`recordedLeftovers` (99줄) + State 의 pid 기록 | `netcompose/steps_lifecycle.go` → `core/process` |

## V5 — netcompose 해체 (59 심볼 전수)

| 심볼 (steps_compose.go / steps_lifecycle.go / workspace.go / new.go) | 목적지 |
|---|---|
| `Workspace.New`·`Retarget`·`Keys`·`Allocate`·`Genesis`·`Config`·`LaunchOpts`·`Provision`·`Init`·`Start`·`Stop`·`Restart`·`Rm`·`Logs`·`Health` + Opts 타입들 | `chainsetup` (순차 진행·단계 표면) |
| `Workspace.genesisArtifacts`·`networkCapabilities`·`shipIdentities`·`driverSpec`·`syncModeFor` | 단계 내용 → 기능 모듈 (genesis·provision·driver 경계로) |
| `Workspace.Netmap`·`netmapRequests`·`AllocateOpts.placements`·`nodeHost` | `netmap` (할당·조회) |
| `Workspace.addrMap`·`resolveTarget`·`keysBase` | `netmap` wrapper (V2.3) |
| `checkVacant`·`scanOccupancy`·`recordedLeftovers`·`startPhase`·`runPhaseActions`·`phaseHasNode`·`phaseActionNode`·`owner` | `core/process` + chainsetup 사전 점검 (V4.3·V5.3) |
| `State`·`NodeState`·`Step`·`Workspace`(Open·Save·Lock·State·Dir·SetEnv·markStep·NodeSet·RPCHost·NodeLabel·binary·plugin) | `chainsetup` 워크스페이스 (실행 기록 폴더와 통합, V5.2) |
| `ParseOverrides` | `chainsetup` (입력 해석) |

## V6 — engine 이분 (109 심볼, 파일 단위)

| 파일 | 심볼 | 목적지 | 근거 |
|---|---|---|---|
| `launcher.go`·`locallaunch.go`·`localplan.go`·`nodecontrol.go` | 33 | `chainsetup`/`process` | 기동·프로세스 제어는 구성 책임 |
| `buildenv.go`·`genesis.go`·`wemixgenesis.go`·`wemixbootstrap.go`·`keysource.go` | 39 | `chainsetup` + 기능 모듈 | 환경 구축·genesis·키 소싱은 셋업 |
| `collect.go`·`summary.go`·`attach.go`·`health.go`·`capability.go`·`chainstate_sink.go`·`plan.go` | 26 | `testengine` | 테스트 수행·수집·관측 |
| `engine.go`·`engine_impl.go`·`app.go`·`wire.go` | 11 | `testengine` 골격 + app 워크플로 | 진입·배선. V6.1 착수 시 심볼 단위 재판정 |

## app 슬림화 후 남는 것 (V3.3·V6.2)

| 현재 app 파일 | 재편 후 |
|---|---|
| `keyring.go` (26) | MCP 용 얇은 함수만 (배선→core, 렌더링→표면) |
| `net_steps.go`(32)·`net_up.go`(5)·`net.go`(6) | chainsetup 호출 껍데기로 축소, CLI 는 직접 호출로 전환 |
| `netmap_query.go` (10) | netmap 호출 껍데기 |
| `serverconf.go` (5) | netmap 으로 이동 (서버 선택은 서버 정보 관심사) |
| `setup.go`(13)·`network.go`(20)·`app.go`(10) 등 | V6.2 워크플로(DSL→셋업→테스트→레포트)로 재편 |
