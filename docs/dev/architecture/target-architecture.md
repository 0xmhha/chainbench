# 목표 아키텍처 — 다이어그램

> 산문으로 흩어진 결정을 **그림 하나로 검토**하기 위한 문서.
> 각 결정의 근거와 실측은 아래 문서에 있고, 여기서는 반복하지 않는다.
>
> [[layers]](layers.md) 현재 레이어·상태 소유 · [[module-responsibilities]](module-responsibilities.md) 관심사 소유자 ·
> [[network-blueprint-design]](../network-blueprint-design.md) 청사진 · [[family-bringup-design]](../family-bringup-design.md) 패밀리 기동 ·
> [[keyring-design]](../keyring-design.md) 키 · [[surface-unification-design]](../surface-unification-design.md) 표면.
> **작업 순서·상태는 [[chainbench-worklist]](../chainbench-worklist.md) §1g.**
>
> 다이어그램은 mermaid v11 파서로 전수 검증했다.

---

## 1. 디렉토리와 호출은 다른 축이다

두 축을 한 그림에 그리면 `dsl` 이 `engine` 하위인 것처럼 읽힌다. **형제다.** 따로 그린다.

### 1-A. 디렉토리 (목표)

```mermaid
flowchart TD
    root["internal/"]
    root --- engine["engine/<br/>테스트벤치 엔진"]
    root --- dsl["dsl/<br/>언어"]
    root --- app["app/"]
    root --- feature["feature/"]
    root --- core["core/<br/>프리미티브·서비스"]
    root --- etc["chains/ · consensus/ · mcp/ · dashboard/"]

    dsl --- dslassert["assert/"]
    dsl --- dslbind["bind/"]
    dsl --- dslinterp["interp/"]
```

### 1-B. 호출 관계 — DSL 은 인터프리터이고 엔진이 주도한다

```mermaid
flowchart LR
    engine["engine (L4)<br/>주도"]
    interp["dsl/interp (L3)<br/>인터프리터"]
    dsl["dsl (L1)<br/>파서"]
    bind["dsl/bind (L1)"]
    assert["dsl/assert (L1)"]

    engine -->|"① 파싱"| dsl
    engine -->|"③ 순차 실행"| interp
    interp --> dsl
    interp --> bind
    interp --> assert
```

이 흐름은 **이미 코드에 있다** — `engine.Run` 이 파싱→환경 준비→인터프리터 호출→기록을 주도하고,
`engine/wire.go` 의 `NewRunSpec` 이 둘을 묶는다. 이름이 그것을 말하지 않았을 뿐이다.

엔진이 파서를 직접 부르는 것은 **L4 → L1 이라 하향**이며 층 위반이 아니다.

> **`engine` 은 하나뿐이다.** 인터프리터를 `dsl/engine` 으로 두면 이름이 겹쳐,
> 이 리팩토링이 없애려는 "유사하면서 다른 이름"이 하나 더 는다.

---

## 2. 레이어 (목표 — 신규 패키지 포함)

```mermaid
flowchart TD
    L6["L6 표면<br/>cmd · mcp · dashboard"]
    L5["L5 유스케이스<br/>app · feature"]
    L4["L4 오케스트레이션<br/>engine · netcompose · chainsetup"]
    L3["L3 도메인 서비스<br/>session · supervisor · collector · health<br/>resolve · materialize · dsl/interp"]
    L2b["L2b 체인<br/>chains/*"]
    L2a["L2a 패밀리<br/>consensus/wbft · consensus/poa"]
    L1["L1 프리미티브<br/>driver · remote · rpc · place · portplan · target<br/>keyring · registry · serverset · blueprint · peering · dsl"]
    L0["L0 커널<br/>node · config · obs · capability"]

    L6 --> L5 --> L4 --> L3 --> L2b --> L2a --> L1 --> L0
    L4 -.주입.-> L2b
    L3 -.-> L1
    L2a --> L1
```

점선은 **주입**이다. L3·L4 는 체인 패키지를 import 하지 않고 인터페이스를 받는다 —
그래서 core 가 어떤 체인도 모른다(C6 ACL).

현재 상태 실측: **상향 의존 0건**([[layers]] §4).

---

## 3. 체인 구성 파이프라인 — 순서가 있다

**enode 는 키와 주소가 모두 정해진 뒤에만 만들어진다.** 그래서 해석에는 강제된 순서가 있고,
genesis·config 는 그 뒤에만 만들 수 있다. 지금 코드가 스텝마다 조각을 다시 모으는 이유가
이 순서를 명시하지 않았기 때문이다.

```mermaid
flowchart TD
    bp["network.yaml<br/>청사진"]
    inv["서버 인벤토리<br/>IP · 포트 풀 · slots"]
    plug["ChainPlugin"]

    bp --> r1
    plug --> r1
    inv --> r2

    r1["① keyring<br/>라벨별 nodekey · 계정<br/>주소를 모른다"]
    r2["② netmap<br/>라벨별 host · port 배정<br/>키를 모른다"]
    r3["③ enode<br/>① + ② 를 합쳐야 만들어진다"]

    r1 --> r3
    r2 --> r3

    r3 --> r4["④ genesis<br/>bp 목록 · 거버넌스 member · alloc"]
    r3 --> r5["⑤ config<br/>static-nodes · peering"]

    r4 --> rn["ResolvedNetwork<br/>불변 스냅샷 + Sources"]
    r5 --> rn
    rn --> mat["materialize (L3)"]

    mat --> sink["provision.FileStore<br/>local | ssh<br/>내용 해시가 같으면 skip"]
    mat --> argv["launch argv"]
    sink --> sup["supervisor (L3)<br/>페이즈 실행"]
    argv --> sup
    sup --> drv["driver (L1)"]
```

**`ResolvedNetwork` 가 못이다.** ①②③ 이 끝난 상태이고, ④⑤ 와 그 아래 전부가 이것만 읽는다.
고쳐야 하면 청사진을 고치고 다시 해석한다.

출처 사슬: `청사진 명시값 > 인벤토리 > 키셋 > 체인 플러그인 > 패밀리 기본 > 내장 기본`.
어느 단계에서 왔는지를 `Sources` 에 기록한다 — 값의 유래가 추측 대상이면 안 된다.

### 3-B. 라벨이 주소를 대신한다

```mermaid
flowchart LR
    dslfile["DSL / 청사진<br/>bp01 · en01 · account1"]
    nm["netmap (L1)<br/>라벨 ↔ 엔드포인트"]
    kr["keyring (L1)<br/>라벨 ↔ 키 · 주소"]

    dslfile -->|라벨로 지칭| nm
    dslfile -->|라벨로 지칭| kr
    nm -->|"라벨 → host:port"| enode["enode"]
    kr -->|"라벨 → nodekey"| enode
    nm -->|"host:port → 라벨"| diag["충돌 검출 · 로그 역추적"]
```

**테스트 정의에 주소가 등장하지 않는다.** `account1` 로 tx 를 보내고, `bp01` 을 재기동한다.
역방향(`10.0.0.11:8545` → `bp01`)이 있어야 포트 충돌이 기동 전에 잡히고,
로그 한 줄에서 어느 노드인지 즉시 안다.

> 정방향 매핑은 **이미 만들어진다**(`place.NodePlacement{Name, Host, Ports}`).
> 문제는 **저장되지 않는다는 것**이다 — `netcompose.NodeState` 에 `Name` 필드가 없어
> 라벨이 배치 직후 사라진다. `netmap` 이 더하는 것은 영속·역방향·계정 라벨이다.

세 배치가 **같은 코드**이며 다른 것은 인벤토리 데이터뿐이다:

| 경우 | 인벤토리 | 결과 |
|---|---|---|
| 로컬 | 서버 1개, `slots` 큼 | 같은 host(127.0.0.1), 포트 증가 |
| 원격 · 서버당 1노드 | 서버 N개, `slots: 1` | host 다름, 포트 동일 가능 |
| 원격 · 서버당 여러 노드 | 서버 M개, `slots > 1` | host 안에서 포트 증가 |

---

## 4. 패밀리 분기는 두 곳뿐

```mermaid
flowchart LR
    subgraph common["공통 — 세 체인 같은 코드"]
        s1["new"] --> s2["allocate"] --> s3["keys"] --> s4["genesis"]
        s4 --> s5["config"] --> s6["launchopts"] --> s7["provision"] --> s8["init"] --> s9["start"]
    end

    s4 -.분기.-> g1["wbft: extraData RLP 치환"]
    s4 -.분기.-> g2["poa: 바이너리가 생성 + config.json"]
    s9 -.분기.-> b1["wbft: 1페이즈 동시 기동"]
    s9 -.분기.-> b2["poa: 2페이즈 + 거버넌스 · etcd"]
```

9스텝 중 **7개가 완전 공통**이다. 특화가 사는 곳은 인터페이스 5개뿐:

```
Family.PortReservation()   포트 예약 폭       L1 선언 / L2a 구현
Family.BuildGenesis()      genesis 생성 방식   L1 선언 / L2a 구현
Family.BringUpPhases()     기동 순서          L1 선언 / L2a 구현
Family.SupportsRole()      pn 가용 여부       L1 선언 / L2a 구현
Deps.Action(name, node)    부트스트랩 액션     L3 호출 / L2a 구현
```

---

## 5. 키 — 세 체인이 동일하다

```mermaid
flowchart TD
    nk["nodekey<br/>32B secp256k1"]
    nk --> addr["address<br/>keccak(pubkey)[12:]"]
    nk --> pub["devp2p pubkey<br/>128 hex"]
    nk --> blspub["BLS pubkey<br/>blst.KeyGen → G1"]
    blspub --> pop["BLS PoP<br/>Sign(pubkey, DST)"]

    addr -.wbft 계열.-> acct["합의 계정 = 파생 주소"]
    addr -.poa.-> sep["합의 계정 = 별도 keystore"]
    blspub -.wbft 계열만.-> gen["genesis extraData"]
```

**nodekey 하나가 루트 시크릿**이고 나머지는 파생이다(실증: 배포 preset node1..5 바이트 동일).
그래서 **BLS 는 청사진의 선언 필드가 아니다** — 선언하면 nodekey 와 어긋날 수 있고,
그 불일치는 genesis 를 통과한 뒤 합의에서 터진다.

파생은 전부 순수 Go 로 가능하다(`CGO_ENABLED=0` 확인) — **키 생성에 체인 바이너리가 필요 없다.**

---

## 6. 피어링 그래프는 역할에서 파생된다

```mermaid
flowchart LR
    subgraph mesh["mesh — pn 없음 (기본)"]
        m1["bp"] --- m2["bp"]
        m1 --- m3["en"]
        m2 --- m3
    end

    subgraph proxied["proxied — pn 있음"]
        p1["bp"] --- p2["pn"]
        p3["bp"] --- p2
        p2 --- p4["en"]
    end
```

**`bp ↔ pn ↔ en`. en 은 bp 를 직접 알지 못한다** — 알면 프록시 계층이 우회되어 무의미해진다.

**poa 는 pn 을 쓰지 않는다** — etcd 가 그 자리다. 청사진이 poa 체인에 `pn` 을 선언하면
조용히 무시하지 않고 **오류**로 거부한다(`Family.SupportsRole`).

현재 구현은 **풀메시 고정**이라 pn 을 두어도 효과가 없다.

---

## 7. 표면 — 하나 등록, 셋이 소비

```mermaid
flowchart TD
    reg["feature 레지스트리 (L5)<br/>기능 · 입력 스키마 · Stage"]
    reg -->|읽기| cli["CLI (cmd)"]
    reg -->|읽기| mcp["MCP"]
    reg -->|"주입 (import 아님)"| interp["dsl/interp (L3)"]

    cli --> app["app 유스케이스 (L5)"]
    mcp --> app
    interp --> app
```

**엔진은 주입받는다.** L3 가 L5 를 import 하면 상향이라 불가능하지만,
인터프리터가 `Registry`·`Action`·`Assertion`·`Deps` 를 **스스로 정의**하고 L5 가 구현체를 넘기므로
값 전달이며 층을 거스르지 않는다 — 이미 그 모양이다.

`feature` 는 `app` 의 하위가 아니라 **별도 패키지**다. `app/feature` 로 두면 `Deps` 때문에
`app → app/feature → app` 참조 순환이 된다.

---

## 8. 아직 판단하지 않은 것

`internal/core/` 아래에 패키지가 몰려 있고 `core` 는 아무것도 말하지 않는다 —
이 문서들이 금지한 `util`/`common` 덤핑과 성격이 같고, 그 안에 L0·L1·L3 이 섞여 있다.

레이어를 디렉토리로 드러내는 안이 있으나 **전 패키지 경로가 바뀐다.**
리팩토링이 끝나 패키지 수가 줄어든 뒤에 판단한다 — 지금은 아니다.
