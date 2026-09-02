# chainbench 시스템 방향 — 체인 구성·노드 제어·테스트·증적

> **[제품 목표·확정 방향]** 사용자가 2026-09-02에 확인한 목표를 정리한다.
> 요구와 동작 계약은 `chainbench-requirements-review.md`와 `chainbench-feature-spec.md`가,
> 작업 순서와 상태는 `chainbench-worklist.md`가 이긴다.

## 1. 핵심 명제

chainbench는 사용자가 체인 환경과 테스트를 선언하면 로컬 또는 원격 서버에 실제 다중 노드
체인을 구성하고, 테스트 중 노드를 개별 제어하며, 실행 증적과 실패 자료를 수집하고, 전체
실행 보고서를 생성하는 Go 기반 다체인 테스트 시스템이다. 지원 대상은 stablenet, wbft,
wemix와 `wemix → wbft` consensus upgrade다.

```text
CLI / DSL / MCP
  → 문법·capability 검증
  → 기존 환경 재사용 판단
  → 체인 구성 또는 복구
  → node monitor 준비 판정
  → 단일 테스트 직렬 실행
  → 결과·로그·증적 수집
  → 전체 실행 보고서 생성
```

## 2. 실행 환경

- **Local**: chainbench가 실행되는 장비에서 노드 바이너리와 파일을 직접 관리한다.
- **Remote**: 폐쇄망 서버에 접속해 파일을 확인·배포하고 노드 프로세스를 실행한다.
- **Docker remote simulation**: 개발 환경의 컨테이너를 원격 서버처럼 사용해 주소 변환,
  파일 전송, 원격 명령, 서버별 배치, PID와 프로세스 수명주기를 검증한다.

Docker 주소 치환은 접속 경계에서만 수행한다. 노드 config, enode와 실행 증적에는 실제 체인
네트워크에서 사용할 주소를 보존한다.

## 3. 체인 구성과 표면

일반 구성은 다음 단계로 진행한다.

```text
new → place → keys → genesis → config → build → deploy → init → start
```

CLI는 각 단계를 독립적으로 실행할 수 있어야 한다. DSL은 선언된 환경을 자동 구성하고 MCP도
같은 core 의미와 결과를 제공한다. CLI는 core를 직접 호출하고 MCP는 app을 경유한다.
호출 경로보다 입력 의미, 기본값, 결과와 오류의 동등성이 중요하다.

## 4. 확정 계획과 실행 상태

체인 실행 전에는 모든 결정이 끝난 확정 계획이 필요하다. 이 문서에서는 개념적으로
`ResolvedExecutionPlan`이라 부르지만 현재 코드에 이 이름의 단일 타입을 신설하라는 뜻은 아니다.

확정 계획에는 체인과 consensus, 노드별 역할·identity·server·ports·binary·checksum·config·argv,
genesis 참여 정보, enode·peer 관계, 배포 자료와 기동 phase가 포함된다. 한 실행 revision에서 이미
확정한 사실을 뒤 단계가 다시 선택하지 않는다. binary나 config를 교체하면 새 revision으로 기록한다.

PID, 실제 command, running 상태, health, block, peer와 수집 자료는 변경 가능한 실행 상태로 분리한다.

## 5. 모듈 책임

- `keyring`: nodekey, account, BLS, keystore와 공개 identity의 생성·가져오기·파생·저장·조회
- `resource`: server-set, 접속, host·slot, port band, capacity, 배정, inventory와 환경 접근
- `node`: BP/EN/PN, topology, placement 결과, binary, sync, enode, peer와 canonical record
- `genesis`: 확정된 producer와 account identity를 체인별 genesis와 추가 산출물로 변환
- `nodeconfig`: chain·role 기본값과 환경·테스트·노드 override를 결합해 config를 만들고,
  내부 Command Builder가 배포 경로, binary, key, ports와 peer를 반영해 argv·환경변수 생성
- `process`: materialize, init, launch, PID·command, stop/restart, timeout, cleanup과 교체 후 재실행
- `chainsetup`: 구성 단계, checkpoint와 resume. workspace 영속은 `session.Composition`에 위탁
- `consensus/upgrade`: consensus 전환과 handoff. 외부 최상위 모듈로 분리하지 않음
- `nodemonitor`: inspector·preflight·health·collector의 사실을 조합해 테스트 실행 허가 판정
- `testengine`: 환경 준비부터 테스트, verdict 생산과 teardown까지의 생명주기
- `session`: 환경·테스트 증적과 실행 상태 영속
- report builder: testengine verdict와 session 증적을 모아 실행 전체 report 생성

`process`는 gstable, gwbft, gwemix 같은 노드 프로그램이 아니라 이를 관리하는 chainbench 모듈이다.

## 6. 자료 재사용과 배포

서버나 로컬에 자료가 있으면 존재 여부, 크기, checksum, 버전과 실행 권한을 확인한다. 동일하면
재사용하고, 없으면 upload/download하며, 다르면 내용 해시 경로나 명시적 교체를 사용한다. 이름이나
존재 여부만으로 동일성을 판단하지 않는다. 대상은 binary, genesis, config, key material, contract
artifact와 테스트 입력이다. private key와 password 원문은 출력과 report에 기록하지 않는다.

## 7. 노드 제어, config와 command

각 노드는 개별 및 전체 start, stop, restart와 상태 확인이 가능해야 한다. 테스트 중 노드별 binary와
config를 교체할 수 있고, 한 네트워크의 노드가 서로 다른 binary를 사용할 수 있어야 한다.

Command builder는 binary, argv, 환경변수, config·datadir·genesis·key 경로, ports, peer, 대상 host,
working directory와 log 경로를 확정한다. 실제 command는 민감 정보를 제거한 뒤 기록한다.

테스트별 config fixture는 `config-<test-purpose>...` 형식으로 이름 짓는다. 원본 fixture, override,
최종 config, 적용 노드·시각과 checksum을 보존한다.

## 8. 컨트랙트 테스트

컨트랙트 배포 결과 주소는 `save/$ref`로 뒤 단계에 전달할 수 있어야 한다. 약속된 deployer와 nonce로
결정적 주소를 만들 수도 있어야 한다. deployer, nonce, 예상·실제 주소, transaction hash, receipt,
block number, artifact와 bytecode checksum을 증적으로 남긴다.

## 9. DSL 사전 검사와 환경 재사용

DSL은 schema/syntax와 semantic/capability를 나눠 검사한다. 후자는 선택한 체인과 바이너리가 role,
flag, RPC, metric, action, assertion과 upgrade 조합을 지원하는지 확인한다. 지원되지 않는 테스트는
자원 배정, 파일 배포와 프로세스 실행 전에 중단한다.

기존 환경은 다음 조건이 모두 맞을 때 재사용한다.

```text
선언 fingerprint 일치
+ 배포 자료와 checksum 일치
+ node와 peer health 정상
= 기존 환경 재사용
```

fingerprint의 확정 입력은 `binaries-set + genesis + config + topology + hardforks + placement`다.
key, ports와 역할은 이 여섯 요소의 canonical 내용에 포함될 때만 fingerprint에 반영한다. runtime health와
배포 파일 checksum은 fingerprint에 섞지 않고 별도 재사용 조건으로 검사한다.

## 10. Node Monitor

목표 `nodemonitor` 모듈은 PID와 command, RPC, block 진행, consensus 참여, peer 연결, topology, chain ID와 genesis,
sync, fork/divergence, metrics와 logs를 관측해 테스트 실행 허가를 결정한다.

```text
READY       → 테스트 시작
WAITABLE    → MaxNodeMonitorTimeout까지 대기
RESTARTABLE → 노드 재시작 후 재검사, 실패하면 종료
FATAL       → 강제 조치가 필요하므로 즉시 종료
```

데이터 삭제, 강제 rewind, genesis 교체처럼 파괴적 조치가 필요한 상태는 자동 복구하지 않는다.
환경 재사용 전, 신규 구성 후, restart·binary/config 변경·partition 복구 후와 각 테스트 전에 적용한다.

## 11. 테스트 실행, 산출물과 보고

한 시점에는 단일 테스트 하나만 수행한다. 한 명령으로 여러 테스트를 요청할 수 있지만 내부에서는
직렬 실행한다. 각 테스트는 PASS, FAIL, BLOCKED, SKIP, infrastructure failure와 미실행을 구분한다.
실패한 테스트는 관련 node logs, process, RPC, peer와 block 관측을 수집한다.

```text
~/.chainbench/<실행날짜-시간>/
├─ session.json
├─ environments/<env-id>/       # 재사용 환경의 정본
│  ├─ env.json
│  ├─ genesis/
│  ├─ keys/                     # 공개 identity와 reference
│  ├─ config/
│  ├─ commands/
│  └─ deployment/
├─ tests/<NNN>_<테스트명>/       # 사용자가 요구한 테스트별 폴더
│  ├─ env-ref
│  ├─ DSL snapshot
│  ├─ genesis/ keys/ config/    # 해당 테스트가 실제 사용한 snapshot 또는 content-addressed reference
│  ├─ commands/ deployment/
│  ├─ logs/ observations/
│  ├─ steps/ assertions/ postactions/
│  └─ result
└─ report
```

모든 테스트가 끝난 뒤 timestamp root에 최종 report를 생성한다. report는 테스트별 verdict와 증적
경로를 연결한다. 중간 실패 후 다음 테스트를 계속할지는 suite 정책으로 명시한다.

## 12. 요구 추적성

| ID | 사용자 요구 | 본문 | 작업 |
|---|---|---|---|
| R01 | local 또는 remote 체인 구성 | §2 | E9 |
| R02 | Docker로 폐쇄망 remote 환경 모사 | §2 | E9 |
| R03 | 서버 자료 확인·재사용·upload/download·동일성 검사 | §6 | E2 |
| R04 | local/remote 노드 PID와 실행 command 관리 | §4·§5·§7 | E4 |
| R05 | 테스트 노드 개별 제어 | §7 | E4·E7 |
| R06 | Command Builder와 실제 설정·배포 정보 반영 및 기록 | §5·§7 | E3·E4 |
| R07 | 테스트 중 stop/start, binary와 config 교체 | §7 | E4·E7 |
| R08 | 노드별 서로 다른 binary | §4·§7 | E3·E7 |
| R09 | Config Builder와 override | §5·§7 | E3 |
| R10 | `config-<test-purpose>` fixture naming | §7 | E3 |
| R11 | contract 배포, 동적·결정적 주소 | §8 | E7 |
| R12 | 실패 시 log와 디버깅 자료 수집 | §11 | E6·E8 |
| R13 | 전체 테스트 종료 후 최종 report | §11 | E8 |
| R14 | 단일 테스트 직렬 실행과 한 명령의 다중 테스트 | §11 | E7·E8 |
| R15 | CLI 단계·DSL 자동 구성·MCP 도구 | §3 | E9 |
| R16 | DSL syntax 및 chain/binary capability 사전 검사 | §9 | E5 |
| R17 | 동일 환경 재사용 | §9 | E5·E6 |
| R18 | Node Monitor, timeout, 제한 재시작과 즉시 종료 | §10 | E6 |
| R19 | timestamp/test별 산출물과 root report | §11 | E0A·E8 |

## 13. 현재 상태와 목표 구분

현재 코드에는 단계형 chainsetup, node record, per-node binary, process ledger, workspace resume, DSL schema
drift 검사와 일부 collector/session 기능이 있다. checksum 기반 전체 자료 재사용, 위 상태 모델의 Node Monitor,
19번의 완전한 산출물 구조와 전체 report는 목표이며 구현 완료로 간주하지 않는다. 정확한 상태는 worklist §1k에서 관리한다.

## 14. 완료 판단 원칙

- 하위 atomic 모듈은 한 종류의 사실만 결정한다.
- 앞 단계가 확정한 사실을 다음 단계가 다시 선택하지 않는다.
- local, remote와 Docker simulation이 같은 core 계약을 사용한다.
- CLI, DSL과 MCP는 같은 core 의미로 수렴한다.
- 사전 검사는 외부 상태 변경 전에 실패한다.
- 실행한 binary, config, command와 관측 결과를 재검토할 수 있다.
- 여러 테스트를 실행해도 한 번에 하나씩 수행하며 마지막에 전체 report가 남는다.
