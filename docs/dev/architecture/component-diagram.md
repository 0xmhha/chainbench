# 컴포넌트 다이어그램

> **[이력]** 2026-08-11 시점.
> **현재 상태를 말하지 않는다.** 그때 무엇을 측정·결정했는지의 기록이다.
> 현재 상태는 [[chainbench-worklist]] 와 코드가 정본이다.

> 2026-08-11 코드 실측. 자매 문서: [아키텍처](software-architecture.md) · [시퀀스](sequence-diagrams.md) · [상태](state-diagrams.md)
> 표기: `✅` 배선 완료 · `◐` 부분 · `☐` 미구현 · `→x` x 로 흡수 예정

---

## 1. 전체 컴포넌트 맵

```mermaid
graph TB
  classDef surface fill:#e8f0fe,stroke:#4a72c4,color:#123
  classDef orch fill:#e6f4ea,stroke:#3f8a54,color:#123
  classDef mid fill:#fef7e0,stroke:#c99a1e,color:#123
  classDef low fill:#f5f5f5,stroke:#999,color:#123
  classDef acl fill:#fce8e6,stroke:#c5392e,color:#123
  classDef legacy fill:#eeeeee,stroke:#bbb,color:#666,stroke-dasharray:4 3

  subgraph SURF["Surfaces — 얇은 어댑터"]
    CLI["cmd/chainbench<br/>60 commands"]:::surface
    MCP["internal/mcp<br/>~50 tools"]:::surface
    DASH["internal/dashboard<br/>+ cmd/chainbenchd"]:::surface
  end

  subgraph ORCH["Orchestration"]
    ENGINE["engine<br/>H1 오케스트레이터"]:::orch
    SPEC["testspec<br/>DSL 파싱 · 해석 · 액션11/어세션16"]:::orch
    ASSERT["testspec/assert<br/>타입인지 비교"]:::orch
  end

  subgraph MIDDLE["Core — 역할 완결"]
    SESSION["session ✅<br/>아티팩트 정본 · env 재사용 · 키 레지스트리 소유"]:::mid
    KEYREG["keyreg ✅<br/>named key · Literal · ExpectAddress"]:::mid
    KEYSRC["engine.KeySource ✅<br/>Preset | Generated"]:::mid
    PLACE["place ✅<br/>배치 · 포트 · 용량검증"]:::mid
    GENESIS["genesis ✅<br/>4모드"]:::mid
    PROV["provision ✅<br/>물질화 · upload-if-absent"]:::mid
    SUP["supervisor ✅<br/>기동 · 헬스게이트 · teardown"]:::mid
    PROC["procman ✅<br/>{PID,datadir,host} · StopOne/All"]:::mid
    COLL["collector ✅<br/>live tail · chainstate · bp참여 · fork"]:::mid
    LOPT["launchopt ☐<br/>Dialect + 옵션모듈 + Builder"]:::mid
  end

  subgraph LOW["Core — atomic"]
    DRIVER["driver<br/>Local | Remote"]:::low
    RPCC["rpc"]:::low
    NODEC["nodeconfig"]:::low
    PORTP["portplan · topology"]:::low
    CONF["config<br/>flag&gt;file&gt;default"]:::low
    OBS["obs.Bus<br/>bounded · drop-on-full"]:::low
    REMOTE["remote<br/>SSH · key_file"]:::low
    KEYGEN["keygen<br/>랜덤 프리셋 생성"]:::low
  end

  subgraph ACLB["Chain ACL — 유일한 변화점"]
    REG["registry<br/>ChainPlugin · ConsensusFamily · Capabilities"]:::acl
    CHAINS["chains/{stablenet,wbft,wemix,external}"]:::acl
    CONS["consensus/{wbft,poa,upgrade}"]:::acl
  end

  subgraph LEG["레거시 스택 (제거 대상 · T7.11)"]
    STATE["core/state →session"]:::legacy
    PIPE["core/pipeline/{setup,verify,attach,testrun}"]:::legacy
    TK["testkit →engine+testspec"]:::legacy
    PROBE["core/probe →collector"]:::legacy
    NETC["netcompose.Workspace →session"]:::legacy
  end

  CLI --> ENGINE
  MCP --> ENGINE
  DASH --> SESSION
  CLI -.-> LEG
  MCP -.-> LEG

  ENGINE --> SESSION & PLACE & KEYSRC & GENESIS & PROV & SUP & COLL & SPEC
  SPEC --> ASSERT & RPCC & COLL
  SESSION --> KEYREG
  KEYSRC --> KEYGEN
  KEYSRC --> KEYREG
  SUP --> PROC
  SUP --> DRIVER
  PROV --> DRIVER
  COLL --> DRIVER
  COLL --> OBS
  DRIVER --> REMOTE
  ENGINE --> NODEC & PORTP & CONF
  ENGINE --> REG
  GENESIS --> REG
  REG --> CHAINS & CONS
  ENGINE -.->|목표 T7.3| LOPT
```

---

## 2. C4 Level-2 — 컨테이너 관점

```mermaid
graph LR
  classDef c fill:#e8f0fe,stroke:#4a72c4
  classDef e fill:#f5f5f5,stroke:#999,stroke-dasharray:4 3

  U(["개발자 / LLM"])
  CLIC["chainbench<br/>CLI 바이너리"]:::c
  MCPC["chainbench-mcp<br/>MCP 서버"]:::c
  DC["chainbenchd<br/>대시보드 서버"]:::c
  FS[("세션 아티팩트<br/>.chainbench/&lt;session&gt;/")]:::c
  WS[("워크스페이스<br/>&lt;data-dir&gt;/state.json")]:::c

  NODE["노드 프로세스<br/>gstable / gwemix"]:::e
  BOOT["bootnode<br/>BLS/PoP"]:::e
  HOST["원격 서버<br/>SSH"]:::e

  U --> CLIC
  U --> MCPC
  U --> DC
  CLIC --> FS
  MCPC --> FS
  CLIC --> WS
  DC --> FS
  CLIC -->|exec init/launch/attach| NODE
  CLIC -->|키 생성| BOOT
  CLIC -->|provision · launch · tail| HOST
  HOST --> NODE
  NODE -->|JSON-RPC · WS · logs| CLIC
  CLIC -->|obs 이벤트 스트림| DC
```

---

## 3. 키 소싱 컴포넌트 (T7.1 배선 결과)

```mermaid
graph TB
  classDef new fill:#e6f4ea,stroke:#3f8a54

  FLAG["CLI --keys-source<br/>preset | generate"] --> KS
  KS{{"engine.KeySource<br/>Dir() · Ensure(ctx,n) · Describe()"}}:::new

  KS --> PKS["PresetKeySource<br/>keys.LoadPreset<br/>+ 노드수 사전검증"]:::new
  KS --> GKS["GeneratedKeySource<br/>metadata.json 있으면 재사용<br/>없으면 keygen.GeneratePreset"]:::new

  GKS -->|필수| BN["외부 bootnode 바이너리<br/>BLS pubkey + PoP"]
  PKS --> KSET["KeySet{Dir, Preset}"]:::new
  GKS --> KSET

  KSET --> RI["engine.RegisterIdentities<br/>node1..nodeN"]:::new
  RI --> REG["session.Keys()<br/>= keyreg.New(&lt;session&gt;/keys)"]:::new
  RI -.->|ExpectAddress| CHK{{"개인키→주소 재유도<br/>선언값과 대조"}}:::new
  CHK -->|불일치| ERR["오류 · 저장 안 함<br/>(디스크·메모리 모두)"]

  KSET --> LL["LocalLauncher<br/>--nodekey --unlock --password"]
  KSET --> PGS["PresetGenesisSource<br/>validators · BLS · extraData · alloc"]
```

| 구성요소 | 파일 | 책임 |
|---|---|---|
| `KeySource` | `internal/engine/keysource.go` | 알고리즘 2·3 의 결정 지점. `Dir()` 는 구성시점, `Ensure` 는 실행시점 |
| `PresetKeySource` | 〃 | 기존 세트 로드 + **노드 수 사전검증**(부족하면 기동 전에 실패) |
| `GeneratedKeySource` | 〃 | 신규 생성(idempotent). bootnode 부재 시 명확한 오류 |
| `RegisterIdentities` | 〃 | `sess.Keys()` 의 실소비자. C2 불변식 강제 |
| `keyreg.Literal` | `internal/core/keyreg/` | 보유 중인 키 자료의 진입 경로 |
| `keyreg.EnsureOpts.ExpectAddress` | 〃 | 선언 주소 ↔ 실제 키 대조 |
| `session.NewWithKeys` | `internal/core/session/` | 레지스트리를 세션 `keys/` 에 루팅(경로 단일소유) |

---

## 4. 목표 컴포넌트 델타

```mermaid
graph LR
  classDef add fill:#e6f4ea,stroke:#3f8a54
  classDef del fill:#fce8e6,stroke:#c5392e,stroke-dasharray:4 3

  subgraph NEW["신설 2개"]
    APP["internal/app<br/>유스케이스 = CLI·MCP 공용"]:::add
    LO["core/launchopt<br/>Dialect · 모듈10 · Builder"]:::add
  end
  subgraph GONE["흡수/폐기"]
    S1["core/state → session"]:::del
    S2["core/pipeline/* → engine+provision"]:::del
    S3["testkit → engine+testspec"]:::del
    S4["core/probe → collector"]:::del
    S5["core/keys → keyreg"]:::del
    S6["netcompose.Workspace → session"]:::del
    S7["chainsetup → app + profiles/"]:::del
  end
```

`internal/app` 도입 후 의존 방향:

```mermaid
graph LR
  CLI["cmd/chainbench<br/>목표: 커맨드당 ~40줄"] --> APP
  MCP["internal/mcp<br/>fan-out 19 → 1"] --> APP
  APP["internal/app"] --> ENG["engine · testspec"] --> CORE["core/*"] --> ACL["chains · consensus"]
```
