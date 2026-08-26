# chainbench 리팩토링 갭 분석

> 대상: `background`/`algorithm`/`key point`에 기술된 목표 아키텍처 대비 현재 코드.
> 산출물 형태: 분석 문서(코드 변경 없음). 근거는 `path:line`으로 표기.
> 작성 기준일: 2026-08-11.

## 한 줄 결론

목표 아키텍처의 **빌딩블록은 대부분 이미 존재**한다(genesis-builder, key-manager, deploy skip-if-exists, procman, 세션 지문 재사용, DSL의 pre/post hook·genesis overlay, RemoteDriver, `place.RemotePerHost`). 실제 갭은 **DSL→엔진→원격 실행의 배선**과 **`server-set.yaml`(serverset)을 노드 구동 경로에 연결**, 그리고 **local/remote를 하나의 "경로(path)"로 다루는 1급 추상**과 **원격 프로세스 종료 검증**에 집중된다.

## 1. 요구사항 ↔ 기존 모듈 매핑

| background/algorithm 요구 | 기존 구현 (근거) | 상태 |
|---|---|---|
| 1.2 base genesis(template)→override 빌드 | `internal/core/genesis/builder.go` (Mode: Existing/Build/TemplateOverride/UpgradeInherit) | ✅ 충족 |
| 1.4 node key / 1.5 keystore (random·import, genesis 반영) | `internal/keymat/source.go`, `internal/core/keys`, `internal/core/keyreg`, `internal/keygen` | ✅ 충족 |
| algo 4/5 genesis overlay(테스트 추가정보) | DSL `ChainSpec.GenesisOverlay` (`internal/testspec/spec.go:21`) + builder | ✅ 충족 |
| 1.3 체인·테스트별 config | DSL `ChainSpec.Config` (`spec.go:20`), `internal/core/config` | ✅ 충족 |
| algo 6 deploy skip-if-exists | `internal/core/filestore/provision.go:46-57` (`Exists`→`Skipped`, "reused rather than") | ✅ 충족 |
| algo 7 command 빌드(http/ws/metric/chainId 등) | `internal/core/nodeconfig`, `internal/consensus/upgrade/launch.go` | ✅ 충족 |
| algo 8 동일 구성 재사용 skip | 세션 지문 재사용 `internal/core/session/session.go:33-36`, `Spec.Fingerprint` (`spec.go`) | ✅ 충족 |
| algo 9 health 체크(최신 블록 등) | `internal/engine/health.go`, `internal/core/probe` | ✅ 충족 |
| algo 10/12 pre/post-test hook | DSL `PreActions`/`PostActions` (`spec.go:36,39`), `internal/testspec/fault.go`·`interpreter.go` | ✅ 충족 |
| algo 11/13 test 수행·결과정리 | `internal/engine/collect.go`·`summary.go`, `internal/testspec/run.go`·`report.go` | ✅ 충족 |
| 3/13 결과 검증 log·rpc·metric | `internal/core/collector`,`logs`,`rpc`,`obs`,`probe` | ✅ 충족 |
| algo 15 local 프로세스 pid 종료·검증 | `internal/core/procman/procman.go:164-190` (SIGTERM→grace→SIGKILL→`Alive` 검증) | ✅ 충족(local) |
| key-point 2 로컬/원격 드라이버 | `internal/core/driver`(Local/Remote), `RemoteDriver` (`driver/remote.go:38`) | ✅ 충족 |
| key-point 2 원격 배치 모드 | `internal/core/place` `RemotePerHost` (`place/place.go:16`) | ✅ 충족(모드 존재) |
| **key-point 2 DSL의 target-machine(local/remote) 선택** | `Spec.Placement string`(`spec.go:34`) 존재하나 → `place.Mode` 매핑 부재 | ⚠️ **갭** |
| **key-point 4 `server-set.yaml`을 노드 구동에 사용** | `serverset`는 `keys import`만 소비(엔진 미연결) | ⚠️ **갭** |
| **key-point 2 통합 "경로"(local dir / `user@host:dir`) 1급 추상** | ad-hoc 파싱만(`remote.CredentialsFromEnv`), plan `Network:"local"` 하드코딩(`engine/plan.go:71`) | ⚠️ **갭** |
| **algo 15 원격 프로세스 종료·검증** | `RemoteDriver.Stop`은 `kill PID || true`로 검증 없음(`driver/remote.go:126-134`) | ⚠️ **갭** |
| **key-point 2 local+remote 프로세스 통합 관리** | `procman.Manager`는 `syscall.Kill`(로컬 시그널) 기반; 원격 PID는 별도 transport 필요(주석 `procman.go:54`) | ⚠️ **갭** |

## 2. 갭 상세 (심각도·위치·수정 방향)

| ID | 심각도 | 위치(근거) | 문제 | 수정 방향 |
|---|---|---|---|---|
| G1 | High | `internal/engine/app.go:100`, `internal/chainsetup/static.go:124` | 엔진/정적 셋업이 `place.LocalStepped`를 **하드코딩** → DSL이 remote를 요구해도 로컬로만 구동 | `Spec.Placement`(및 신규 `targetMachine`) → `place.Mode` 리졸버 도입, 엔진 `BuildEnv.Mode`/`Capacity`를 스펙에서 주입 |
| G2 | High | `serverset` grep 결과 소비처가 `cmd/chainbench/keyflags.go`뿐 | `server-set.yaml`이 **노드 구동/배포 경로와 미연결**(keys import 전용) | `serverset.Server.Credentials`→`driver.SSHRunner`→`RemoteDriver`를 엔진 셋업에서 사용; `--server`/스펙 target을 launcher까지 전달 |
| G3 | Med | `cmd/chainbench/resolve.go:31-50` | 원격 setup이 **단일 host/env(`CHAINBENCH_REMOTE_PASS`)** 기반이라 인벤토리(다중 서버)와 분리됨 | resolve 경로를 `serverset` 인벤토리 기반으로 통합(인덱스/`user@host:dir`로 다중 노드 배치) |
| G4 | Med | `internal/engine/plan.go:71` (`Network:"local"`), `place.NodePlacement.DataPath string` | local/remote를 **하나의 경로 타입으로 다루지 않음**(문자열·하드코딩) | 1급 `Location`/`Path` 타입 도입(local dir vs `user@host:dir`)해 plan/provision/driver를 관통시킴 (key-point 2) |
| G5 | Med | `internal/core/driver/remote.go:126-134` | `RemoteDriver.Stop`이 kill 후 **종료 확인 없음** → algo 15의 "정상 종료 검증" 미충족 | kill 후 `kill -0`/재조회로 종료 검증, 미종료 시 SIGKILL 에스컬레이션(로컬 `StopAll` 패턴 대칭) |
| G6 | Med | `internal/core/procman/procman.go:54,164` | procman이 **로컬 시그널 전용** → 원격 PID를 동일 관리면에서 추적·종료 못함 | procman에 transport 추상(로컬 signal / 원격 exec)을 주입해 local+remote PID를 단일 레지스트리로 관리(key-point 2) |
| G7 | Low | `internal/testspec/spec.go:24-40` | DSL 스키마에 **명시적 `targetMachine`(local/remote)·서버 바인딩 필드 부재** | `Spec`에 `targetMachine`(기본 local) + 원격 서버 선택(인덱스/alias) 필드 추가, `Placement`와 정합 |

## 3. 사실 / 의견

| 구분 | 내용 | 확신도 |
|---|---|---|
| Fact | 엔진·정적 셋업이 `place.LocalStepped`를 하드코딩(app.go:100, static.go:124) | None |
| Fact | `serverset`(server-set)는 `keys import` 경로에서만 소비됨(grep 결과 그 외 없음) | None |
| Fact | `provision.Provision`은 존재 파일 skip(재사용) 구현(provision.go:46-57) | None |
| Fact | 로컬 `procman.StopAll`은 SIGTERM→SIGKILL→`Alive` 생존 검증 수행(procman.go:164-190) | None |
| Fact | `RemoteDriver.Stop`은 `kill PID 2>/dev/null || true`만 수행, 검증 없음(remote.go:126-134) | None |
| Fact | DSL은 `PreActions`/`PostActions`(hook), `GenesisOverlay`, `Placement`, `Fingerprint`를 이미 보유(spec.go) | None |
| Opinion(High) | 핵심 리팩토링은 "신규 기능"보다 **기존 부품의 배선(DSL→엔진→serverset→RemoteDriver)** 이다 | — |
| Opinion(High) | G1·G2가 최우선. 이 둘만 해결해도 DSL 기반 원격 구동의 종단 경로가 열린다 | — |
| Opinion(Mid) | G4(통합 경로 타입)는 파급이 커서 G1·G2 이후 별도 단계로 두는 것이 안전 | — |
| Opinion(Mid) | G5·G6는 원격 안정성/정합성 확보에 필요하나 종단 경로가 열린 뒤 처리 가능 | — |

## 4. 권장 우선순위

1. **G1 + G2 (High)** — 스펙 placement/target → `place.Mode` 리졸버 + `serverset` 자격증명을 엔진 launcher까지 연결. (원격 종단 경로 확보)
2. **G7 (Low, G1과 함께)** — DSL에 `targetMachine`/서버 선택 필드 추가.
3. **G5 (Med)** — 원격 종료 검증.
4. **G4 + G6 (Med)** — 통합 경로 타입 + procman 원격 관리(구조적 파급 큼, 별도 단계).
5. **G3 (Med)** — 단일 host resolve 경로를 인벤토리 기반으로 통합/정리.

> 비고: 본 문서는 검토 산출물이며 코드는 변경하지 않았다. 실제 리팩토링 착수 시 각 갭을 개별 태스크로 분해하고 회귀 테스트를 동반할 것.
