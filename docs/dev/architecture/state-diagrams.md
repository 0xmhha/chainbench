# 상태 다이어그램

> 2026-08-11 코드 실측. 자매 문서: [아키텍처](software-architecture.md) · [컴포넌트](component-diagram.md) · [시퀀스](sequence-diagrams.md)
> 대상 Aggregate: `TestRun`(C1) · `Environment`(C2) · `NodeProcess`(C3) · `Session`(C5) + 워크스페이스 스텝.

---

## 1. 테스트 하나의 생애 — `TestRun` (C1)

```mermaid
stateDiagram-v2
    [*] --> Parsing

    Parsing --> Blocked: 파싱 실패 / schemaVersion 미지원
    Parsing --> Gating: Spec 확정

    Gating --> Skipped: 체인 미적용 (applicableChains)
    Gating --> Skipped: capability 미충족 (requires ⊄ 제공집합)
    Gating --> Resolving: 적용 대상

    Resolving --> EnvReady: fingerprint 일치하는 env 재사용
    Resolving --> Building: 신규 env
    Building --> EnvReady: BringUp OK
    Building --> Blocked: Diagnosis.OK=false

    EnvReady --> PreActions
    PreActions --> Blocked: pre 실패 (테스트 미수행)
    PreActions --> Steps

    Steps --> Steps: 다음 스텝 (원자 · 부분성공 없음)
    Steps --> Failed: 스텝 실패 (revert · 미바인딩 $ref · 미지 액션)
    Steps --> Asserting

    Asserting --> Passed: 전 어세션 통과
    Asserting --> Failed: 어세션 1건 이상 실패

    Passed --> PostActions
    Failed --> PostActions
    PostActions --> [*]: 판정과 독립 (post 실패가 결과를 바꾸지 않는다)

    Skipped --> [*]
    Blocked --> [*]

    note right of Blocked
        blocked = 인프라 문제 (테스트가 돌지 못함)
        failed  = 테스트가 돌았고 기대와 달랐음
        CI exit: pass=0 · fail=1 · blocked=2
    end note
```

| 상태 | 의미 | 기록 |
|---|---|---|
| `skip` | 이 체인/능력에 해당 없음 — 실패가 아니다 | `status.json` |
| `blocked` | 환경 구성 또는 preAction 실패로 **테스트가 수행되지 않음** | + `Diagnosis` |
| `fail` | 스텝 또는 어세션이 기대와 다름 | + `steps.json` · `assert.json`(provenance) |
| `pass` | 전 어세션 통과 | + provenance |

---

## 2. 환경 — `Environment` (C2)

fingerprint 가 재사용의 유일한 판정 기준이다.

```mermaid
stateDiagram-v2
    [*] --> Declared: spec 의 선언값 + resolved config

    Declared --> Fingerprinted: sha256(binaries+genesis+config+topology+hardforks+placement)

    Fingerprinted --> Reused: 같은 fp 의 env 가 세션에 있음
    Fingerprinted --> Allocating: 없음

    Allocating --> Rejected: 용량 위반 (validators 4 미만 · 호스트×슬롯 초과 · 포트대역 초과)
    Allocating --> KeysReady: 배치·포트 확정

    KeysReady --> GenesisBuilt: KeySet → validators·BLS·extraData·alloc
    GenesisBuilt --> Provisioned: 물질화 (있으면 skip)
    Provisioned --> Launching
    Launching --> Gating: 전 노드 init + launch
    Gating --> Healthy: LeaderGate → HealthGate 통과
    Gating --> Diagnosed: 게이트 실패

    Diagnosed --> Launching: 재시도 (RemoveDataDir 로 stale 정리)
    Diagnosed --> Rejected: 재시도 소진

    Healthy --> Reused: env.json 저장 (node table + dataPath)
    Reused --> Collecting: collector 가동
    Collecting --> Collecting: 같은 fp 의 다음 테스트가 재사용
    Collecting --> TornDown: 세션 종료
    TornDown --> [*]
    Rejected --> [*]

    note right of Fingerprinted
        placement(local↔remote)도 fp 에 포함된다 —
        local 로 세운 env 를 remote 선언이
        재사용하면 포트/호스트가 어긋난다
    end note
```

**env-id** = `"env-" + fingerprint[:12]`. 전체 64-hex 는 `env.json` 의 `fingerprint` 에만 기록한다
(경로 길이 상한 보호).

---

## 3. 노드 프로세스 — `NodeProcess` (C3)

불변식: **고아 0**.

```mermaid
stateDiagram-v2
    [*] --> Armed: armSpecs — config 렌더 · 신원 플래그 · 바이너리 확정
    Armed --> Materialized: genesis · config 물질화 (upload-if-absent)
    Materialized --> Initialized: 노드바이너리 init --datadir
    Initialized --> Running: Launch → PID
    Running --> Tracked: procman.Track{PID, Label, DataDir, Host}

    Tracked --> Stopping: StopOne (fault 스텝) / StopAll (teardown)
    Tracked --> Crashed: 프로세스 소멸 (Alive=false)

    Stopping --> Terminated: SIGTERM → grace 대기
    Stopping --> Killed: grace 초과 → SIGKILL
    Killed --> Terminated: 폴링으로 소멸 확인
    Killed --> Leaked: 폴링 후에도 생존

    Terminated --> DataRemoved: RemoveDataDirs (종료와 별개 연산)
    Terminated --> [*]
    DataRemoved --> [*]

    Crashed --> Tracked: restartNode (arming 재사용)
    Leaked --> [*]: leak 보고 (조용히 넘기지 않는다)

    note right of DataRemoved
        내장 etcd 는 프로세스 종료로 함께 죽는다.
        문제는 같은 datadir 재기동 시 남은 클러스터 상태 →
        해법은 삭제이며, 종료와는 별개 기능이다
    end note
```

- `StopOne` 과 `StopAll` 이 분리된 이유: `StopAll` 은 네트워크 전체를 내려 쿼럼 테스트에 쓸 수 없다.
- `Host` 가 loopback 이면 **local** 로 판정한다(원격 오판 시 단일 노드 정지가 불가능해진다 — 라이브에서만
  드러났던 결함).

---

## 4. 세션과 키 레지스트리 — `Session` (C5)

```mermaid
stateDiagram-v2
    state "세션 생명주기" as SL {
        [*] --> KeyMaterialization
        KeyMaterialization --> Aborted: KeySource.Ensure 실패 (bootnode 부재 · 키셋이 토폴로지보다 작음)
        KeyMaterialization --> TreeCreated: KeySet 확보

        TreeCreated --> RegistryBound: 세션 트리 생성 — keys · environments · tests
        RegistryBound --> IdentitiesRegistered: keyreg.New — 세션 keys 디렉토리

        IdentitiesRegistered --> Aborted: 주소 재유도 ≠ 선언값 (C2 위반)
        IdentitiesRegistered --> Open: node1..nodeN 등록 (0600)

        Open --> Open: env 생성/재사용 · 테스트 기록 · 런타임 키 추가
        Open --> Saved: session.json 기록
        Saved --> [*]
        Aborted --> [*]
    }
```

키 하나의 상태:

```mermaid
stateDiagram-v2
    [*] --> Requested: Ensure(name, source, ref, opts)
    Requested --> Cached: 같은 이름이 이미 있음 (idempotent)
    Requested --> Materializing

    Materializing --> Generated: Random — crypto/rand
    Materializing --> Read: LocalFile — 경로에서 읽기
    Materializing --> Fetched: RemoteDownload — SSH
    Materializing --> Held: Literal — 호출자가 들고 있던 hex

    Generated --> AddressDerived
    Read --> AddressDerived
    Fetched --> AddressDerived
    Held --> AddressDerived

    AddressDerived --> Rejected: ExpectAddress 불일치
    AddressDerived --> BLSDerived: NeedBLS (외부 bootnode)
    AddressDerived --> Persisted: BLS 불필요
    BLSDerived --> Rejected: BLSDeriver 미주입
    BLSDerived --> Persisted

    Persisted --> Cached: keys 아래 이름별 private · address (+bls · pop) — private 0600 · dir 0700
    Cached --> [*]
    Rejected --> [*]

    note right of Rejected
        거부된 키는 디스크에도 메모리에도 남지 않는다 —
        남으면 다음 실행이 레지스트리가 거절한
        자료를 주워 쓴다
    end note
```

---

## 5. 워크스페이스 스텝 — `chainbench net`

각 스텝은 독립적으로 실행·재실행 가능하고, 워크스페이스가 진행을 누적한다.

```mermaid
stateDiagram-v2
    [*] --> Empty
    Empty --> Created: net new (chain · target)
    Created --> Keyed: net keys
    Keyed --> Allocated: net allocate
    Allocated --> Genesised: net genesis
    Genesised --> Configured: net config
    Configured --> Composed: net launchopts (실행 command 확인)
    Composed --> Provisioned: net provision
    Provisioned --> Initialized: net init
    Initialized --> Started: net start
    Started --> Healthy: net health
    Healthy --> Tested: net run CASE
    Tested --> Started: 다음 케이스
    Started --> Stopped: net stop
    Healthy --> Stopped: net stop
    Stopped --> Started: net start (재기동)
    Stopped --> Empty: net rm --data

    Created --> Created: 재실행 = idempotent
    Keyed --> Keyed: 재실행 = 기존 세트 재사용

    note right of Composed
        각 상태는 net status 로 조회된다.
        선행조건 미충족 스텝은 조용히 성공하지 않고
        "무엇을 먼저 실행하라"를 말한다
    end note
```

---

## 6. 상태 저장소 대응표 (통합 전/후)

| 개념 | 현재 A(레거시) | 현재 B(엔진) | 현재 C(스텝) | 통합 후 |
|---|---|---|---|---|
| 노드 목록 | `state.SaveNetwork` | `env.json` node table | (미보유) | `session.Environment` |
| 진행 상태 | 없음 | `session.json` | `state.json` steps | `session.json` + steps |
| 키 자료 | preset 경로 | **`<session>/keys/`** | preset 경로 | `<session>/keys/` |
| PID | 없음 | `procman` + env | `state.json` | `procman` + `session` 영속 |

통합 순서는 [`../chainbench-worklist.md`](../chainbench-worklist.md) §1c T7.5·T7.7·T7.11.
