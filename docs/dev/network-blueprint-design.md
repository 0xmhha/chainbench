# 네트워크 청사진(Blueprint) — 구성 요소 전수 · 선언 → 해석 → 물질화

> 문제: 네트워크를 구성하는 정보가 **4조각으로 흩어져** 있고, 어느 조각도 전체를 말하지 못한다.
> 그래서 genesis·config 를 "그때그때" 만들게 되고, preset 없이는 아무것도 구성할 수 없다.
> 목표: **하나의 선언 문서**가 네트워크 전체를 기술하고, 거기서 모든 산출물이 파생된다.
>
> 실측: 2026-08-18. 관련: [[surface-unification-design]](surface-unification-design.md) ·
> [[family-bringup-design]](family-bringup-design.md) · [[server-inventory]](server-inventory.md).
> 작업 순서는 [[chainbench-worklist]](chainbench-worklist.md) §1g.

---

## 1. 진단

### 1.1 구성 정보가 4조각으로 흩어져 있다

| 조각 | 담는 것 | 못 담는 것 |
|---|---|---|
| `topology.yaml` | index · role · sync_mode · bootnode | 서버·포트·키·계정·잔액 |
| `remote-server-config.yaml` | 호스트 · 포트 대역 · dataRoot · SSH | 어느 노드가 어디에 · 역할 |
| `keys/preset/metadata.json` | nodekey · 계정 · BLS · extraData | 배치·역할·잔액 정책 |
| `poa.Config`(wemix 전용) | members · accounts · env | 다른 체인에 쓸 수 없음 |

**어느 것도 "이 네트워크는 이렇게 생겼다"를 말하지 못한다.** 그래서 각 스텝이 조각을 모아
자기 몫을 재조립하고, 그 조립 로직이 스텝마다 조금씩 다르다.

### 1.2 preset 이 선택이 아니라 전제다

```go
// netcompose.Genesis
preset, err := keys.LoadPreset(w.state.KeysDir)   // ← 없으면 진행 불가
```

수동으로 값을 넣어 네트워크를 구성할 방법이 없다. **preset 은 시간 단축 수단이어야지 전제가 아니다.**

### 1.3 바이너리 경로가 20개 파일에 흩어져 있다

`--binary` 플래그 · `Manifest.Binary` · `cluster.BinaryFor` · `plan.Nodes[].Binary` ·
`LaunchOptions.Binary` · `hardfork.ToBinary` · 핸드오프 프로파일 … 같은 개념이 이름만 바꿔 반복된다.

---

## 2. 구성 요소 전수 (지금까지 누락됐던 것 포함)

### 2.1 네트워크 수준

| # | 요소 | 필수 | 지금 어디에 | 비고 |
|---|---|---|---|---|
| N1 | 체인 (id 또는 외부 매니페스트) | ✅ | `net new --chain` | |
| N2 | **실행 바이너리** (노드) | ✅ | 20곳 분산 | |
| N3 | 보조 바이너리 (bootnode 등) | | `--bootnode` | 키 생성용 |
| N4 | **노드별 바이너리 오버라이드** | | `plan.Nodes[].Binary` | **핸드오프·하드포크에 필수** |
| N5 | genesis 템플릿 | ✅ | 플러그인 내장 / `--genesis-template` | wemix 는 체인 저장소 것 |
| N6 | chainId · networkId | ✅ | 매니페스트 / `--chain-id` | `--networkid` **미방출**(F2) |
| N7 | 하드포크 활성 블록 | | `--set genesis.overrides.*` | |
| N8 | genesis 오버레이 | | `--overlay` | |
| N9 | **검증자 집합** (누가 BP 인가) | ✅ | preset 순서에 암묵 | **명시 수단 없음** |
| N10 | **검증자 스테이크** | poa | `poa.Member.Stake` | wemix 전용 |
| N11 | **alloc — 계정별 초기 잔액** | ✅ | preset `alloc` 고정 | **수정 수단 없음** |
| N12 | **거버넌스 역할 계정** (staker·ecosystem·maintenance·feecollector) | poa | `poa.Config` | wemix 전용 |
| N13 | **거버넌스 정책** (blockCreationTime·stakingMin…) | poa | `poa.Env` | wemix 전용 |
| N14 | **부트노드 선정** | ✅ | `topology.bootnode` | poa 는 필수, 현재 `BootRole` 이 부정확 |

### 2.2 노드 수준 — **누락이 가장 심한 부분**

| # | 요소 | 필수 | 지금 어디에 | 비고 |
|---|---|---|---|---|
| P1 | 이름 / 인덱스 | ✅ | `node1..N` 자동 | 이름 지정 불가 |
| P2 | **역할** (bp · en · boot) | ✅ | `topology.role` | |
| P3 | **어느 서버에** | ✅ | `serverset` + fleet 균등분배 | **노드↔서버 개별 지정 불가** |
| P4 | IP / 호스트 | ✅ | 배치에서 파생 | |
| P5 | 포트 (p2p·http·ws·auth·metrics) | ✅ | `portplan` 파생 | **개별 지정 불가** |
| P6 | ↳ 파생 포트 (etcd peer·client) | poa | 예약만 | |
| P7 | **nodekey** (devp2p 신원) | ✅ **전 노드** | preset | **개별 지정 불가** |
| P8 | **합의 계정** (unlock·etherbase) | **BP만** | preset | **개별 지정 불가** |
| P9 | keystore 파일 · password | BP만 | preset | |
| P10 | BLS 키 · PoP | wbft | preset | |
| P11 | sync mode | | `topology.sync_mode` | |
| P12 | datadir 경로(타깃 기준) | ✅ | 파생 | |
| P13 | 노드별 launch 옵션 | | `--launch-opt`(전역만) | **노드별 불가** |

**P7·P8 이 핵심 정정**: nodekey 는 devp2p 신원이라 **BP·EN 모두** 필요하고,
합의 계정은 **BP 에만** 필요하다. 개수가 다르다.

### 2.3 연결 구성 — 노드들이 서로를 찾는 방법

| # | 요소 | wbft/stablenet | wemix |
|---|---|---|---|
| C1 | 정적 피어 | `static-nodes` = 전 노드 enode | 사용 안 함 |
| C2 | 부트노드 | 선택 | **필수** (거버넌스 배포 지점) |
| C3 | 멤버 등록 | genesis extraData | **`members[]` (거버넌스)** |
| C4 | 클러스터 형성 | 없음 | **etcd** (peer=p2p+1, client=p2p+2) |

**"어떤 노드를 통해 체인에 연결하는가"의 답이 패밀리마다 다르다** —
wbft 는 static-nodes 목록, wemix 는 거버넌스 member 목록 + etcd. 이것이 C1~C4 를
청사진에서 **파생**시켜야 하는 이유다(사람이 enode 를 손으로 쓰면 안 된다).

---

## 3. 설계 — 선언 → 해석 → 물질화

```
① Blueprint (선언)          사람이 쓰거나 generator 가 만든다. 부분적이어도 된다.
        │  Resolve(blueprint, inventory, plugin, keysource)
        ▼
② ResolvedNetwork (확정)    모든 값이 채워진 불변 스냅샷. 직렬화된다.
        │  Materialize
        ▼
③ Artifacts (산출)          genesis · 노드별 config · 노드별 신원 · argv
        │  Sink(local | remote)
        ▼
④ Target (물질화)           로컬 폴더 또는 원격 호스트
```

**세 단계를 분리하는 이유**: 지금은 ①②③이 스텝마다 섞여 있어서, 같은 조립이 조금씩 다르게 반복된다.
②를 **직렬화 가능한 단일 스냅샷**으로 못박으면, 그 뒤 모든 것은 ②만 읽는다.

### 3.1 Blueprint — 하나의 선언 문서

```yaml
# network.yaml — 이 네트워크의 전부. 모든 필드는 선택이며, 없으면 §3.3 순서로 채워진다.
version: 1
chain: wemix                        # 또는 manifest: ./my-chain.json

binaries:
  node: /path/to/gwemix             # N2
  bootnode: /path/to/bootnode       # N3
  overrides:                        # N4 — 핸드오프/하드포크
    - nodes: [bp5, bp6]
      node: /path/to/gwbft

nodes:                              # P1~P13. 개수가 곧 네트워크 크기다.
  - name: bp1
    role: bp                        # P2
    server: local                   # P3 — serverset 항목 이름
    ports: {p2p: 8589, http: 8588}  # P5 — 생략하면 배치가 정한다
    nodekey: {file: ./keys/bp1.nodekey}      # P7 — 또는 hex: 0x…
    account:                                  # P8·P9 — BP 만
      keystore: ./keys/bp1.json
      password: ./keys/password
    syncmode: full                  # P11
    launch: {maxtxsperblock: 1000}  # P13 — 이 노드만
  - {name: en1, role: en, server: srv2}

validators:                         # N9·N10
  from: role                        # role=bp 인 노드 (기본) | explicit: [bp1, bp2]
  stake: 1500000000000000000000000

alloc:                              # N11
  - {account: bp1, balance: 200000000000000000000000}   # 노드 이름 참조
  - {address: "0xdead…", balance: 1000000000000000000}  # 또는 raw 주소

genesis:                            # N6·N7·N8
  chainId: 8285
  overrides: {bohoBlock: 10}
  overlay: ./overlay.json

governance:                         # N12·N13 — poa 패밀리만. 다른 체인에서는 거부.
  roles: {staker: "0x…", ecosystem: bp1, maintenance: mnt, feecollector: mnt}
  env:   {blockCreationTime: 1000, stakingMin: 1500000000000000000000000}
```

**설계 원칙 3가지**

1. **모든 필드가 선택이다.** `{chain: wemix, nodes: [{role: bp} × 4]}` 만으로도 네트워크가 선다.
2. **명시가 항상 이긴다.** 값을 쓰면 그 값이 쓰인다 — preset 도 생성기도 덮지 못한다.
3. **참조가 가능하다.** `alloc[].account: bp1` 처럼 노드 이름으로 가리킨다. 주소를 두 번 쓰지 않는다.

### 3.2 preset 은 Blueprint 의 **생성기**다

```sh
# preset 에서 청사진을 만든다 — 이후 손으로 고칠 수 있다
chainbench net blueprint --from-preset keys/preset --chain wemix --bp 4 --en 1 > network.yaml

# 청사진으로 구성한다
chainbench net up --blueprint network.yaml
```

이 뒤집기가 핵심이다. 지금은 `preset → (내부 조립) → 네트워크` 라서 중간을 볼 수도 고칠 수도 없다.
바뀌면 `preset → 청사진(볼 수 있음·고칠 수 있음) → 네트워크` 가 된다.

### 3.3 필드가 채워지는 순서 (Resolve)

각 필드마다 **출처 사슬**이 고정된다. 이것이 "자유도는 높되 역할은 분명"의 구현이다.

| 순위 | 출처 | 예 |
|---|---|---|
| 1 | **Blueprint 명시값** | `ports: {p2p: 8589}` |
| 2 | **인벤토리** (`serverset`) | 서버의 포트 대역·dataRoot |
| 3 | **키셋** (preset 또는 생성) | nodekey·계정 |
| 4 | **체인 플러그인** | chainId·바이너리 이름·하드포크 |
| 5 | **패밀리 기본값** | 포트 예약 폭·거버넌스 env 기본 |
| 6 | **내장 기본값** | 배치 기본 대역 |

**출처를 기록한다.** `ResolvedNetwork` 는 값과 함께 "어디서 왔는지"를 남긴다 —
[[server-inventory]] 에서 포트 출처를 표시한 것과 같은 이유로, 값의 유래가 추측 대상이면 안 된다.

### 3.4 ResolvedNetwork — 확정된 스냅샷

```go
// internal/core/blueprint (L1: 선언 파싱) / internal/app (L5: 해석)

// ResolvedNetwork is the network with every value decided. Nothing downstream
// asks a question again: genesis, configs, argv and the deploy set are all
// derived from this one snapshot, which is why it is serialized alongside them.
type ResolvedNetwork struct {
    Chain      registry.ChainPlugin
    Nodes      []ResolvedNode        // 전 노드, 전 필드 확정
    Validators []string              // 봉인자 주소, 순서 확정
    Alloc      []AllocEntry
    Genesis    GenesisSpec
    Governance *GovernanceSpec       // poa 만
    Sources    map[string]string     // 필드 → 출처 ("blueprint" | "inventory" | …)
}

type ResolvedNode struct {
    Name, Role   string
    Host         string
    Ports        node.Endpoints
    NodeKey      KeyRef               // 전 노드
    Account      *AccountRef          // BP 만 (nil = 봉인 안 함)
    Binary       string               // 오버라이드 적용 후
    DataDir      string               // 타깃 기준 경로
    SyncMode     string
    LaunchOpts   []launchopt.Override
}
```

**불변이다.** 만들어진 뒤 아무도 고치지 않는다. 고쳐야 하면 Blueprint 를 고치고 다시 Resolve 한다.

### 3.5 Materialize — 산출물은 노드별 묶음이다

```
ResolvedNetwork
   ├─ genesis.json            ← 전 노드 공용 (Family.BuildGenesis)
   ├─ wemix-config.json       ← poa 만 (GenesisArtifacts.Extra)
   └─ 노드별:
        config_<name>.toml    ← nodeconfig.Generate
        <datadir>/geth/nodekey
        <datadir>/keystore/*  (BP 만)
        password
        argv                  ← launchopt.Builder
```

그리고 **노드별 묶음이 통째로 Sink 로 간다** — 로컬이면 폴더, 원격이면 SSH.
스텝은 로컬/원격을 분기하지 않는다([[layers]] §5 상태 규칙).

---

## 4. 이것이 해결하는 것

| 지금 | 이후 |
|---|---|
| 구성 정보가 4조각, 전체를 말하는 것이 없음 | Blueprint 하나 |
| preset 없이는 구성 불가 | preset 은 Blueprint 생성기 |
| 노드별 nodekey·계정·포트·서버 지정 불가 | 노드마다 명시 가능 |
| 바이너리 경로 20곳 | `binaries:` 한 곳 + 노드별 오버라이드 |
| alloc·검증자 집합 수정 불가 | 선언 |
| 값의 출처를 알 수 없음 | `Sources` 에 기록 |
| genesis/config 조립이 스텝마다 조금씩 다름 | 전부 `ResolvedNetwork` 하나만 읽음 |

---

## 5. 공통과 특화 — 다시

Blueprint 관점에서 보면 **체인 특화는 두 곳뿐**이다.

| | 공통 | 특화 |
|---|---|---|
| **선언** | `chain·binaries·nodes·validators·alloc·genesis` | `governance:`(poa 만) |
| **해석** | 배치·포트·키·잔액 확정 | 포트 예약 폭 |
| **물질화** | config·nodekey·keystore·argv | genesis 생성 방식 · `Extra` 산출물 |
| **기동** | — | 페이즈 수와 사이 액션 |

즉 §2 의 요소 **N1~N14 · P1~P13 · C1~C4 중 특화는 N10·N12·N13·P6·C3·C4 여섯 개**이고,
전부 **poa(wemix) 한 패밀리에만** 속한다. 나머지는 세 체인이 같다.

---

## 6. 남은 결정

1. **`pn` 역할을 도입할 것인가.** 현재 역할은 `validator·endpoint·boot·en` 4종이고 `pn` 은 없다.
   wemix 에서 EN/PN 을 구분해야 한다면 역할 표를 넓혀야 한다 — **체인 팀 확인 필요.**
2. **Blueprint 를 어디에 둘 것인가.** 노드 IP·포트를 담으므로 [[server-inventory]] 와 같은 민감도다.
   서버 인벤토리를 참조만 하고(`server: local`) 자신은 IP 를 담지 않게 하면 커밋 가능해진다 —
   **그 편이 낫다고 본다.**
3. **DSL `env` 선언과 Blueprint 의 관계.** [[dsl-v2-proposal]] 의 `env` 가 Blueprint 의 부분집합인가,
   Blueprint 를 가리키는 참조인가. 후자가 단순하다.
4. **기존 `topology.yaml` 을 어떻게 할 것인가.** Blueprint 의 `nodes:` 가 상위집합이므로
   흡수 대상이다. 이관 기간에는 둘 다 받되 혼용은 거부한다.
