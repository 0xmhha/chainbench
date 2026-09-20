# 리팩토링 후속 작업 인수인계

> **[현행 설계 보조]** 2026-09-02 AST·문서 X-bar 대조와 사용자 확인을 반영했다.
> 제품 방향은 `chainbench-system-direction.md`, 작업 순서와 상태는 `chainbench-worklist.md` §1k가 이긴다.

## 1. 현재 판단

현재 코드는 역할·배치·노드별 binary, 단계형 chainsetup, DSL 실행과 workspace 복구까지 크게
정리됐다. 후속 작업은 다음 원칙을 완성해야 한다.

1. 배치와 key에서 확정한 producer identity를 genesis가 다시 선택하지 않는다.
2. local, remote와 Docker simulation에서 자료·command·PID·node control이 같은 계약을 사용한다.
3. 테스트 전 정상성 판정부터 실패 자료 수집과 전체 report까지 하나의 실행 기록으로 연결한다.

## 2. 확정된 소유권과 폐기한 제안

- `resource`: 서버, 접속, slot, ports, capacity, inventory와 환경 접근
- `node`: BP/EN/PN, topology, placement, binary, sync, enode와 node record
- `process`: materialize, init, launch, PID·command, stop/restart와 기동 정책
- `nodeconfig`: config와 Command Builder
- `chainsetup`: 구성 단계, checkpoint와 resume. 영속은 `session.Composition`에 위탁
- `consensus/upgrade`: consensus 전환과 handoff. 외부 모듈로 분리하지 않는다.
- `nodemonitor`: 실행 허가와 제한 복구 판정. 기존 atomic 관측 모듈을 조합
- `testengine`: 테스트 생명주기와 verdict 생산
- `session`: 환경·테스트 증적 영속
- report builder: 전체 실행 결과 집계
- CLI는 core를 직접 호출하고 MCP는 app을 경유한다.

다음 과거 제안은 폐기한다.

- key, binary와 enode를 `resource.Plan`이 소유
- 별도 mutating `resource claim/release` CLI를 필수화
- process를 단순 exec wrapper로 축소
- 모든 mixed-binary 구성을 chainsetup 우회로 판정
- upgrade를 consensus 밖으로 이동
- `dsl/interp`를 단순히 testengine으로 이동
- CLI와 MCP의 호출 경로 자체를 통일

## 3. E1: resolved producer와 genesis 일치

`genesis.Request`는 확정 producer 대신 validator 개수와 placement를 받는다
(`internal/core/genesis/source.go:30-45`). WBFT preset source는 첫 N개 identity를 다시 고른다
(`internal/core/genesis/source.go:79-96`). topology에서 BP가 첫 N개 노드가 아니면 실행 BP와
genesis validator가 달라질 수 있다.

작업:

1. `EN, BP, PN, BP` 순서의 실패 테스트를 먼저 만든다.
2. topology, placement와 key identity를 결합한 resolved producer 입력을 정의한다.
3. WBFT가 address와 BLS identity를 resolved producer 순서대로 사용하게 한다.
4. POA와 WBFT가 같은 producer 입력 계약을 사용하되 family별 필드만 소비하게 한다.
5. 누락, 중복, role 불일치를 genesis write 전에 거부한다.

완료 조건:

- genesis validator와 실제 BP identity가 일치한다.
- 동일 입력은 동일 genesis를 만든다.
- count로 첫 N개를 다시 선택하는 구성 경로가 사라진다.
- genesis readback으로 address, BLS와 producer 관계를 검증한다.

## 4. E0A: 산출물 계약 선행 고정

E2 이후 작업이 각각 임시 저장 형식을 만들지 않도록 session artifact 구조를 먼저 고정한다.

```text
~/.chainbench/<timestamp>/
├─ session.json
├─ environments/<env-id>/
├─ tests/<NNN>_<test-name>/
└─ report
```

environment는 재사용 가능한 genesis·key reference·config·command·deployment의 정본이다. 테스트 폴더는
`env-ref`와 함께 해당 테스트가 실제 사용한 genesis, key 정보, config, command를 snapshot 또는
content-addressed reference로 보존한다. 이로써 환경 자료를 무조건 복제하지 않으면서 테스트별 검토도 가능하다.

`testengine`은 verdict를 생산하고 `session`은 증적을 저장하며 report builder는 전체 결과를 집계한다.

## 5. E2: 자료 재사용 무결성

local, remote와 Docker simulation에서 binary, genesis, config, key material과 contract artifact를
동일한 규칙으로 검사하고 재사용한다.

- 존재 여부만 보는 경로를 조사한다.
- 크기와 checksum을 포함한 동일 파일 판정을 정의한다.
- 동일하면 재사용하고, 없으면 upload/download하며, 다르면 내용 해시 경로나 명시적 교체를 사용한다.
- 재사용·전송·교체 결과와 checksum을 deployment 증적에 기록한다.
- secret 원문이 command, log와 report에 노출되지 않도록 검사한다.

완료 조건은 같은 내용 재전송 0, 같은 이름의 다른 내용 오재사용 0, local/SSH/Docker 판정 동등,
중간 실패 산출물의 정상 파일 오인 0이다.

## 6. E3·E4: config, command, process와 노드별 제어

`process`는 노드 바이너리가 아니라 chainbench 내부 프로세스 관리 모듈이다. 기존 Direct,
Launcher와 chainsetup start 경로가 command 생성과 기동 정책을 중복 소유하는지 먼저 측정한다.

- chain·role·environment·test·node config override 우선순위를 한 builder에서 고정한다.
- fixture naming을 `config-<test-purpose>...` 형식으로 정의한다.
- 원본 fixture, override, 최종 config, 적용 노드·시각과 checksum을 기록한다.
- `nodeconfig`의 Command Builder가 binary, config, key, ports, peer와 배포 경로를 argv에 반영한다.
- PID와 실제 command를 노드·서버별로 기록한다.
- 개별·전체 start/stop/restart를 같은 ledger에 연결한다.
- binary/config 교체를 새 실행 revision으로 기록한다.
- 부분 기동 실패, timeout, cancellation과 orphan cleanup을 검증한다.

완료 조건은 노드별 PID→binary/checksum/config/argv 추적, override 격리, config parser readback,
개별 조작의 다른 노드 무손상, 재시작 후 ledger 일치와 local/remote/Docker 동등성이다.

## 7. E5: DSL syntax와 capability 사전 검사

- JSON Schema와 Go parser의 필드·타입 drift를 기계적으로 검사한다.
- syntax 검사와 semantic/capability 검사를 분리한다.
- chain/binary별 role, flag, RPC, metric, action, assertion과 upgrade 지원을 registry에서 조회한다.
- 지원하지 않는 테스트를 resource 배정과 파일 write 전에 종료한다.
- CLI와 MCP가 같은 판정 결과를 제공한다.
- endpoint selector(`node7`, `bp1`, role/index, `on`, `onEach`, `defaultOn`)를 검사한다.
- waitReceipt/waitBlocks/waitEpoch/waitSeconds/waitFork의 timeout 누락을 검사한다.
- PN 없는 proxied 구성과 wemix POA의 PN을 거부한다.
- applicableChains 불일치는 오류가 아니라 SKIP으로 판정한다.

## 8. E6: Node Monitor 실행 허가

목표 `nodemonitor` 모듈은 inspector, preflight, health와 collector를 복제하지 않고 조합한다.
inspector는 사실 실사, preflight는 have/want 비교, collector는 관측 보존을 유지하고 nodemonitor가
READY/WAITABLE/RESTARTABLE/FATAL 판정과 제한 복구를 소유한다.

```text
READY       → 테스트 시작
WAITABLE    → MaxNodeMonitorTimeout까지 대기
RESTARTABLE → 노드 재시작 후 재검사
FATAL       → 자동 강제 조치 없이 즉시 종료
```

관측 범위는 PID·command, RPC·block 진행, consensus 참여·sync, peer·topology, chain ID·genesis,
fork/divergence, metrics와 logs다. 환경 재사용 전과 각 테스트 전에 통과해야 한다. 데이터 삭제,
rewind와 genesis 교체가 필요한 상태는 FATAL로 종료한다. 모든 판정과 복구 시도를 증적에 남긴다.

## 9. E7: 동적 테스트와 컨트랙트

- 여러 테스트 요청도 한 번에 하나씩 직렬 실행한다.
- 노드별 stop/start/restart, partition/heal, binary/config 교체를 DSL action과 연결한다.
- 컨트랙트 동적 주소는 `save/$ref`로 전달한다.
- 고정 deployer와 nonce로 결정적 주소를 만드는 시나리오를 검증한다.
- deployer, nonce, 예상·실제 주소, tx, receipt와 artifact checksum을 남긴다.
- PASS, FAIL, BLOCKED, SKIP, infrastructure failure와 미실행을 구분한다.
- 기대한 revert는 성공으로, pre 실패는 BLOCKED로, post 실패는 본 테스트 verdict와 분리해 기록한다.
- nonce 순차, gas 자동/static/null과 named signing key/faucet을 검증한다.

## 10. E8: 산출물과 최종 report

```text
~/.chainbench/<실행날짜-시간>/
├─ session.json
├─ environments/<env-id>/
├─ tests/<NNN>_<테스트명>/
│  ├─ env-ref
│  ├─ DSL·genesis·keys·config·commands·deployment
│  ├─ logs·observations·steps·assertions·postactions
│  └─ result
└─ report
```

실패 시 관련 node log, PID·command, RPC, peer와 block 관측을 수집한다. 모든 테스트 종료 후 root에
최종 report를 만들고 테스트별 verdict와 증적 경로를 연결한다. private key와 password 원문은 없다.

append-only log tail, remote reconnect, chainstate JSONL과 observation mirror를 검증한다. assertion은
RPC·함수·로그의 provenance로 원본까지 역추적할 수 있어야 한다.

## 11. E9: 표면과 실행 환경 동등성

- CLI 단계 실행, DSL 자동 구성, MCP tool이 같은 core 의미로 수렴하는지 확인한다.
- MCP가 topology, 노드별 binary, sync, server-set과 genesis override를 표현하는지 점검한다.
- local, remote와 Docker simulation에서 같은 시나리오를 검증한다.
- 일반 per-node mixed-binary와 `consensus/upgrade` handoff를 구분한다.

## 12. 실행 순서

```text
E0 현행 재측정 → E0A artifact/session 계약
 ├─ E1 genesis identity
 ├─ E2 material reuse
 ├─ E3 config/command evidence
 └─ E5 DSL validation

E2 + E3 → E4 process/node control
E1 + E2 + E3 + E4 + E5 → E6 node monitor
E4 + E5 + E6 → E7 dynamic tests/contracts
E0A + E3 + E6 + E7 → E8 final aggregation/report
E1~E8 → E9 surface/environment parity
```

실제 상태와 일정은 `chainbench-worklist.md` §1k에서만 갱신한다.
