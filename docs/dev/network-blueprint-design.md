# 네트워크 청사진(Blueprint) — 구성 요소 전수 · 선언 → 해석 → 물질화

> **[현행 설계]** 체인 구성 선언·해석·물질화.
> 지금 향하는 목표. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])이고,
> 작업 순서는 [[chainbench-worklist]] §1g 다.

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

### 2.1b 노드 역할 — bp · en · pn (확정)

역할은 **세 가지**다. `boot` 은 역할이 아니라 **속성**이다 — 한 노드가 bp 이면서 부트노드일 수 있다.

| 역할 | 하는 일 | 합의 | RPC 제공 | 필요한 키 |
|---|---|---|---|---|
| **bp** | 블록 생성(봉인) | ✅ | 보통 비공개 | nodekey + 계정 (+BLS, wbft 계열) |
| **en** | endpoint node — 사용자에게 RPC 제공 | ❌ | ✅ | nodekey |
| **pn** | **proxy node** — bp 를 외부에서 가리고 p2p 를 중계 | ❌ | ❌(보통) | nodekey |
| *(속성)* `bootnode: true` | 최초 기동·거버넌스 배포 지점 | — | — | 해당 노드의 키 |

**`pn` 은 바이너리 모드가 아니다 (실측).** 세 체인 어디에도 proxy/sentry 노드 개념이 없다 —
`--proxy` 같은 플래그가 존재하지 않는다. `pn` 을 표현하는 것은 **연결 그래프**이고,
그것을 쓰는 수단은 범용 피어링 플래그(`--nodiscover`·`--netrestrict`·`--maxpeers`)와
**static-nodes 목록**뿐이다.

> 그래서 `pn` 은 `Family.StartFlags(role)` 이 아니라 **§2.3 연결 구성**에서 다뤄야 한다.
> 역할이 argv 를 바꾸는 것이 아니라 **누가 누구의 static-nodes 에 들어가는가**를 바꾼다.

**현재 코드와의 차이**: 지금은 `validator·endpoint·boot·en` 4종이고 `pn` 이 없다.
이관은 `validator→bp`, `endpoint→en`, `boot→(속성으로 강등)`, `pn` 신설이다.

---

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
| P8 | **합의 계정** (unlock·etherbase) | **bp만** | preset | **개별 지정 불가**. wbft 계열은 P7 에서 파생 |
| P9 | keystore 파일 · password | bp만 | preset | poa 는 P7 과 **독립된 키** |
| P10 | **BLS 공개키 · PoP** | **wbft 계열만** | preset | **P7 에서 파생** — 별도 선언 대상이 아니다 |
| P11 | sync mode | | `topology.sync_mode` | |
| P12 | datadir 경로(타깃 기준) | ✅ | 파생 | |
| P13 | 노드별 launch 옵션 | | `--launch-opt`(전역만) | **노드별 불가** |

**P7·P8 이 핵심 정정**: nodekey 는 devp2p 신원이라 **BP·EN 모두** 필요하고,
합의 계정은 **BP 에만** 필요하다. 개수가 다르다.

### 2.2b 키 파생 — 무엇을 선언하고 무엇이 파생되는가 (실측)

`bootnode -nodekeyhex <nodekey> -writeaddress` 는 **address · devp2p publicKey · BLS publicKey ·
BLS PoP** 를 전부 뱉는다. 즉 wbft 계열에서 **nodekey 하나가 루트 시크릿**이다.

| | nodekey | 합의 계정 | BLS |
|---|---|---|---|
| **stablenet · wbft** | 노드당 1개 (선언 대상) | **nodekey 에서 파생** | **nodekey 에서 파생** |
| **wemix (poa)** | 노드당 1개 (선언 대상) | **별도 keystore** (`wemix new-account`) | **사용 안 함** (`consensus/poa` BLS 참조 0건) |

**설계 귀결 3가지**

1. **BLS 는 Blueprint 의 선언 필드가 아니다.** nodekey 를 선언하면 따라온다.
   선언 필드로 두면 nodekey 와 어긋날 수 있고, 그 불일치는 genesis 를 통과한 뒤 합의에서 터진다.
2. **계정 선언 필요 여부가 패밀리마다 다르다.** wbft 계열은 nodekey 만으로 충분하고,
   poa 는 `account:` 를 따로 선언해야 한다. Blueprint 는 둘 다 받고, **패밀리가 무엇을 요구하는지 검증**한다.
3. ~~BLS 파생에는 외부 바이너리가 필요하다~~ → **필요 없다 (2026-08-18 실증).**
   go-wbft 의 파생은 전부 표준 알고리즘이다:

   ```go
   bls.DeriveFromECDSA(priv) = blst.KeyGen(nodekey32)                   // EIP-2333
   pop                       = sk.Sign(pub, DST="BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")
   ```

   `github.com/supranational/blst` 로 chainbench 가 직접 파생할 수 있고, **실제로 바이트 동일함을 확인했다**
   (`bootnode -writeaddress` 출력과 pubkey·PoP 모두 일치). 따라서 `--bootnode` 옵션은 사라지고
   `key new --with-bls` 가 그 자리를 대신한다.

**그래서 키 생성 전 과정이 외부 바이너리 없이 가능하다.**

```
nodekey (32B secp256k1, crypto/rand)
  ├─ address        keccak(pubkey)[12:]                 accounts.AddressForKey  (기존)
  ├─ devp2p pubkey  secp256k1 uncompressed 128hex       Go 내장
  ├─ BLS pubkey     blst.KeyGen(nodekey) → G1 48B       blst
  └─ BLS PoP        Sign(pubkey, DST=…_POP_) → G2 96B   blst
```

이것이 [[chainbench-worklist]] §1g 의 "raw 가 먼저" 를 실제로 가능하게 한다 —
새 키셋을 만드는 데 **어떤 체인 바이너리도 필요 없다.**

genesis 는 **검증자의 BLS 만** 소비한다(`len(Validators) == len(BLSKeys)` 강제).
preset 이 전 노드에 BLS 를 주는 것은 노드가 나중에 검증자로 승격될 수 있기 때문이고, 낭비가 아니다.

---

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

#### C5 — 피어링 그래프는 역할에 따라 달라진다 (신규)

현재 구현은 **풀메시**다. 전 노드가 전 노드를 static-nodes 에 넣는다.

```go
// netcompose.Config — 오늘
for _, ns := range w.state.Nodes { staticNodes = append(staticNodes, enode(ns)) }
```

**proxy 토폴로지는 정확히 이것을 하지 않는 구조다.** pn 이 존재하는 이유가 bp 를 외부에서
가리는 것이므로, bp 가 모두를 직접 알면 pn 은 의미가 없다.

| 그래프 | 언제 | bp 가 아는 것 | pn 이 아는 것 | en 이 아는 것 |
|---|---|---|---|---|
| **mesh** (기본) | pn 이 없을 때 | 전 노드 | — | 전 노드 |
| **proxied** | pn 이 하나라도 있을 때 | **pn 만** | bp + pn + en | **pn 만** |

**규칙은 한 문장이다: `bp ↔ pn ↔ en`.**
**en 은 bp 를 직접 알지 못한다** — pn 을 통해서만 도달한다. 이것이 pn 이 존재하는 이유이고,
en 이 bp 를 알면 프록시 계층이 우회되어 무의미해진다.

```yaml
peering: mesh        # 기본. pn 이 하나라도 있으면 proxied 가 기본이 된다
# peering: proxied
# peering: {custom: {bp1: [pn1, pn2], en1: [pn1]}}   # 최후 수단
```

#### wemix(poa)는 pn 을 쓰지 않는다 — etcd 가 그 자리다

wemix 는 static-nodes 대신 **거버넌스 member 목록 + etcd 클러스터**로 연결되고,
프록시 계층의 역할을 **etcd 가 대신한다.** 따라서:

- **poa 패밀리에서 `pn` 은 적용 대상이 아니다.**
- 청사진이 poa 체인에 `pn` 을 선언하면 **오류**다. 조용히 무시하면 사용자는 프록시 계층이
  선 줄 알지만 실제로는 없는 네트워크를 얻는다 — 이 프로젝트가 `LeaderGate`·`Action` 에
  이미 쓰는 계약("선언했는데 미배선이면 오류")과 같은 이유다.
- `peering` 은 poa 에서 **선언 자체가 무의미**하므로 마찬가지로 거부한다.

이것을 `Family` 가 말한다:

```go
// ConsensusFamily gains:
//   SupportsRole(node.Role) bool     // poa: pn -> false
//   SupportsPeeringGraph() bool      // poa: false (etcd 가 연결을 소유)
```

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
# network.yaml — 이 네트워크의 전부. 모든 필드는 선택이며, 없으면 §3.4 순서로 채워진다.
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

peering: mesh                       # C5 — mesh | proxied | {custom: …}

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

### 3.2 raw 가 먼저, preset 은 나중 (순서가 설계다)

**손으로 쓴 값만으로 네트워크가 서는 것을 먼저 만든다.** 그 다음에 preset 을 얹는다.
반대로 하면 preset 이 다시 전제가 되고, 지금과 같은 자리로 돌아온다.

```
1단계  raw 청사진만으로 3체인 기동         ← 여기가 완성돼야
2단계  preset 이 그 청사진을 생성            ← 이것이 순수한 시간 단축이 된다
```

`net up --blueprint network.yaml` 이 **preset 디렉토리 없이** 동작하는 것이 1단계의 게이트다.
지금은 `keys.LoadPreset` 이 필수라 이것이 불가능하다.

### 3.3 preset 은 Blueprint 의 **생성기**다

```sh
# preset 에서 청사진을 만든다 — 이후 손으로 고칠 수 있다
chainbench net blueprint --from-preset keys/preset --chain wemix --bp 4 --en 1 > network.yaml

# 청사진으로 구성한다
chainbench net up --blueprint network.yaml
```

이 뒤집기가 핵심이다. 지금은 `preset → (내부 조립) → 네트워크` 라서 중간을 볼 수도 고칠 수도 없다.
바뀌면 `preset → 청사진(볼 수 있음·고칠 수 있음) → 네트워크` 가 된다.

### 3.4 필드가 채워지는 순서 (Resolve)

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

### 3.5 ResolvedNetwork — 확정된 스냅샷

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

### 3.6 Materialize — 산출물은 노드별 묶음이다

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

## 6. 요구 1~12 → 설계 반영 (2026-08-19)

세 체인을 실제로 구성해 본 기록에서 다시 도출한 요구다. 각 항목이 **어느 모듈의 책임**인지 못박는다.

| # | 요구 | 책임 모듈 | 상태 |
|---|---|---|---|
| 1 | 로컬: 동일 IP(127.0.0.1) + 서로 다른 포트 | `netmap` ← `place`·`portplan` | 있음(포트) / **라벨 관리 없음** |
| 2 | 원격: IP 목록·가용 포트 목록을 **별도 파일**로. 어떤 노드가 어떤 ip·port 를 쓰는지 관리 | `serverset`(풀 선언) + `netmap`(배정·조회) | 인벤토리·`slots`·포트밴드 **있음** / **조회 없음** |
| 3 | DSL 이 bp/en/pn 을 지정. **라벨 ↔ ip·port 양방향 조회** | `netmap` | 정방향 **있음**(`place.NodePlacement.Name`) / **버려짐**·역방향 없음 |
| 4 | bp·en·pn 모두 nodekey 필요. enode = nodekey + ip + port → **설정에 순서가 있다** | `keyring` → `netmap` → `enode` | §6.2 |
| 5 | 연결: static enode / bootnode discovery / (wemix) etcd | `peering` + `Family` | 코드 확인 완료 §6.3 |
| 6 | genesis 에 bp 기입 + 거버넌스 등록. **faucet 계정 1개 상시**. `account1` 라벨 ↔ 실주소·개인키 | `keyring`(라벨·키) + 청사진(alloc) | **라벨 관리 없음** |
| 7 | config 로 체인 설정. genesis·config 는 **생성 또는 기존 사용**. config 가 **여러 개**이고 중간에 재시작 | `genesis`(SourceMode 있음) + `nodeconfig` + DSL 스키마 | **다중 config 없음** |
| 8 | 다중 config 는 **특정 노드에만**. DSL = 구성 정의 + 테스트 정의(pre/do/post) | 청사진(구성) + DSL v2(테스트) | **훅 미배선** |
| 9 | do-test 액션·기대값 비교. 실패 시 기록·로그 수집 | `dsl/interp` + `session` + `collector` | 대부분 있음 |
| 10 | 미지원 기능은 **문법 오류가 아니라 사유를 남기고 미수행** | `engine/capability`(게이팅) | 게이팅 있음 / **사유 미기록** |
| 11 | 로컬/원격을 IP 기반 HTTP 로 동일 스택 처리 | `target` + `netcompose` | 있음 |
| 12 | 신규 생성 일반화 + preset 지원. **내용이 같으면 deploy skip** | `provision`(Exists 있음) + **내용 비교 추가** | 존재 확인만 있음 |

### 6.1 신설이 필요한 것은 하나뿐 — `netmap`

요구 1·2·3·4·6 이 전부 같은 것을 가리킨다: **라벨과 엔드포인트의 관계를 소유하는 모듈.**

> **정정 (검토 중 발견)**: 처음에 "라벨 관리가 없다"고 적었으나 **틀렸다.**
> `place.NodeReq{Name}` → `place.NodePlacement{Name, Host, Ports}` 로 **정방향 매핑은 이미 만들어진다.**
> 실제 문제는 다르다 — **그 라벨이 생성 직후 버려진다.** `netcompose.NodeState` 에는 `Name` 필드가
> 아예 없고(`grep -c Name` = 0), `Index` 와 `Role` 만 저장된다. 즉 `val1`·`ep1` 이라는 이름이
> 한 함수 호출 동안만 존재하고 워크스페이스에 남지 않는다.
>
> 따라서 `netmap` 이 더할 것은 세 가지다: **(a) 라벨의 영속**, **(b) 역방향 조회**,
> **(c) 노드 아닌 라벨(계정)까지 포함.** 정방향 계산 자체는 재발명하지 않는다.

```go
// core/netmap (L1) — 라벨 ↔ 엔드포인트. 불변이며 resolve 가 만든다.

// Label is how a blueprint and a DSL case name a thing without knowing its
// address: "bp01" · "en01" · "pn01" · "account1". Every downstream artifact
// (enode, genesis member, static-nodes, tx sender) resolves through a label,
// so an address never appears in a test definition.
type Label string

// Endpoint is where a labeled node actually listens.
type Endpoint struct {
    Host  string          // 127.0.0.1 (로컬) | 10.0.0.11 (원격)
    Ports node.Endpoints  // p2p · http · ws · auth · metrics
}

// Map answers both directions. The reverse direction is not a convenience:
// it is how a port collision is caught before launch, and how an operator
// reading a log line finds which node it came from.
type Map struct { … }

func (m Map) Endpoint(Label) (Endpoint, bool)   // 라벨 → 주소
func (m Map) LabelsOn(host string) []Label      // IP → 라벨들
func (m Map) LabelAt(host string, port int) (Label, bool)  // 포트 → 라벨
func (m Map) Enode(Label) (string, error)       // nodekey + host + p2p
```

**역방향이 있어야 하는 이유가 둘이다.** (a) 같은 서버에 여러 노드를 두면 포트 충돌이
기동 전에 잡혀야 한다. (b) 로그·오류에서 `10.0.0.11:8545` 를 보고 어느 노드인지 즉시 알아야 한다.

**가용 자원은 인벤토리가 선언한다** — `netmap` 은 배정 결과를 소유하고, 풀은 `serverset` 이 갖는다.
`slots` 와 포트 밴드는 **이미 구현돼 있다**(2026-08-18). 명시적 범위 풀(`pool:`)은 밴드로 덮이지
않는 경우에만 더한다 — 지금 필요하다는 근거는 없다.

```yaml
servers:
  - name: srv1
    host: 10.0.0.11
    slots: 4                       # 이 서버가 감당할 노드 수
    ports:
      p2pBase: 30303   p2pStep: 10
      rpcBase: 8545    rpcStep: 10
      # 또는 명시적 풀:
      # pool: {p2p: [30303-30400], rpc: [8545-8600]}
```

- **로컬**(요구 1): 서버 1개, `slots` 큼 → 같은 host, 포트가 step 만큼 증가
- **원격, 서버당 1노드**(요구 2): 서버 N개, `slots: 1` → host 다름, 포트 동일 가능
- **원격, 서버당 여러 노드**(요구 2): 서버 M개, `slots > 1` → 같은 host 안에서 포트 증가

**세 경우가 같은 코드다.** 다른 것은 인벤토리 데이터뿐이다.

### 6.2 설정에는 순서가 있다 (요구 4)

enode 는 **키와 주소가 모두 정해진 뒤에만** 만들 수 있다. 그래서 resolve 가 이 순서를 강제한다.

```
① keyring     라벨별 nodekey · 계정          (주소를 모른다)
② netmap      라벨별 host · port 배정         (키를 모른다)
③ enode       ① + ② 를 합쳐야 만들어진다
④ genesis     bp 목록 · 거버넌스 member · alloc   ← ③ 의 재료(id·ip·port)를 쓴다
⑤ config      static-nodes · peering          ← ③ 을 쓴다
```

> **정확히**: wemix `poa.Member` 는 `ID`·`IP`·`Port` 를 따로 갖고 enode 문자열은 갖지 않는다
> (런타임 `wemixNode` 가 `enode://Id@Ip:Port` 로 조립한다). 즉 ④ 는 ③ 의 **재료**를 쓴다.
> wbft 계열의 static-nodes 는 완성된 enode 문자열을 쓴다.

**③ 이전에 ④·⑤ 를 만들 수 없다.** 지금 코드가 스텝마다 조각을 다시 모으는 이유가 이 순서를
명시하지 않았기 때문이다. `ResolvedNetwork` 는 ①②③ 이 끝난 상태이고, ④⑤ 는 그것만 읽는다.

### 6.3 연결 메커니즘 — 코드로 확인 (요구 5)

| 체인 | 메커니즘 | 근거 |
|---|---|---|
| wbft · stablenet | `p2p.Config` 의 `StaticNodes`·`BootstrapNodes`·`TrustedNodes`, 또는 datadir `static-nodes.json` | `p2p/server.go:106-121` · `node/config.go:39` |
| wemix (poa) | 거버넌스 member(`enode·ip·port`)를 읽어 **노드가 스스로** `admin_addPeer` | `wemix/admin.go:566-576` |

wemix 는 chainbench 가 피어를 붙일 필요가 없다 — 라이브 실행에서 static-nodes 없이,
수동 addPeer 없이 4노드가 연결됐다. **그래서 `peering` 선언은 wbft 계열에만 의미가 있다.**

### 6.4 계정 라벨 (요구 6)

테스트는 주소를 쓰지 않는다. `account1` 같은 라벨을 쓰고, **개인키를 아는 쪽이 서명한다.**

```yaml
accounts:                       # 청사진
  - {label: faucet,   balance: 1000000000000000000000000}   # 항상 하나 있어야 한다
  - {label: account1, balance: 1000000000000000000000}
  - {label: account2, balance: 0}
```

- **라벨 → 주소·개인키**: `keyring` 이 소유 (nodekey 와 같은 링, 다른 이름 공간)
- **주소 → genesis alloc**: 청사진이 선언, resolve 가 확정
- **faucet 은 생략할 수 없다** — 잔액 0 인 계정으로 tx 를 보내려면 가스가 필요하고,
  그 자금원이 없으면 테스트가 조용히 실패한다. 청사진에 없으면 **오류**로 만든다.

### 6.5 config 는 여러 개이고, 노드마다 다르다 (요구 7·8)

한 테스트가 config A 로 띄웠다가 멈추고 config B 로 다시 띄우는 경우가 있고,
**전 노드가 아니라 일부 노드만** 그렇게 한다.

```yaml
nodes:
  - name: bp01
    role: bp
    config: cfgA                 # 이름으로 참조
  - name: bp02
    role: bp
    config: cfgA

configs:                          # 이름 → 내용(또는 파일)
  cfgA: {file: ./conf/a.toml}
  cfgB: {template: default, set: {syncmode: snap}}
```

DSL 테스트 정의 쪽에서 전환을 명령한다:

```yaml
steps:
  - restartNode: {node: bp02, config: cfgB}    # bp02 만 cfgB 로 재기동
```

> `restartNode`·`stopNode`·`startNode` 액션은 **이미 있다**(`testspec/fault.go`).
> 새로 필요한 것은 **`config:` 인자 하나**이지 새 액션이 아니다.

**genesis 와 동일하게 config 도 "생성 또는 기존"이다.** `genesis.SourceMode` 가 이미
`ModeExisting`·`ModeBuild`·`ModeTemplateOverride`·`ModeUpgradeInherit` 를 정의해 두었고,
config 에도 같은 축(`file` | `template`+`set`)이 필요하다.

### 6.6 DSL 은 구성 정의와 테스트 정의로 나뉜다 (요구 8)

```
env      구성 정의 — 체인·노드·역할·config·키·genesis      (= 청사진, 또는 청사진 참조)
case     테스트 정의
  pre-test    준비 훅 — 환경 스캔·자금 공급 등
  do-test     본 테스트 — 액션 + 기대값 비교
  post-test   정리 훅 — 수집·복구
```

**pre-test 이전에 환경이 정상인지 스캔하는 책임**이 있어야 한다(요구 8) —
블록이 전진하는지, 전 노드가 같은 높이인지, 피어가 붙었는지. `health`·`collector` 가 그 재료를
갖고 있고, 스캔을 **게이트로 강제**하는 것은 엔진의 몫이다.

### 6.7 미지원 기능은 조용히 실패하지 않는다 (요구 10)

체인마다 지원 기능이 다르다. 미지원 기능을 만나면 **문법 오류로 처리하지 않고**,
무엇이 왜 지원되지 않는지 남기고 그 케이스를 수행하지 않는다.

**주의: "capability" 라는 이름이 둘이다** (검토 중 발견 — 이 프로젝트가 반복해 온 패턴).

| 이름 | 하는 일 | 쓰는 곳 |
|---|---|---|
| `internal/engine/capability.go` | spec 의 `requires` 를 체인 제공집합과 대조해 **skip 판정** | 엔진 (DSL 게이팅) |
| `internal/core/capability` | `Catalog`·`Descriptor`·`Handler` — **MCP/CLI 기능 카탈로그** | `cmd capabilities` · `mcp` · `chains/*/caps.go` |

**서로 무관하다.** DSL 게이팅은 앞의 것이고, 뒤의 것은 표면 카탈로그다.
S 계열(표면 통일)에서 뒤의 것이 `feature` 레지스트리와 겹치므로 함께 정리해야 한다.

게이팅 자체는 동작한다(`satisfies`·`applicableWithCaps`). 없는 것은 **사유 기록**이다 —
`session.TestRecord.Status(s TestStatus)` 에 사유 인자가 없어(`record_impl.go:79`)
"왜 skip 됐는지"가 아티팩트에 남지 않는다.

### 6.8 deploy skip 은 존재가 아니라 내용으로 판단한다 (요구 12)

원격에 이미 바이너리·nodekey·계정이 올라가 있으면 배포 단계를 건너뛸 수 있다.
그러나 **파일이 있다는 것만으로 건너뛰면 안 된다** — 내용이 다르면 다른 네트워크가 된다.

```
① 대상 경로에 파일이 있는가            provision.Exists  (있음)
② 내용이 우리가 보낼 것과 같은가        해시 비교         (없음 — 추가)
   같으면 skip, 다르면 덮어쓰기(또는 오류)
```

현재 `upload-if-absent` 는 ①만 본다. **②를 더해야 요구 12 가 성립한다.**
해시는 `FileStore.Read` 로 얻는다(§4.3 에서 이미 필요하다고 판단한 그 읽기다).

---

## 6. 남은 결정

1. ~~`pn` 역할을 도입할 것인가~~ → **확정: `bp·en·pn` 3종, `boot` 은 속성**(§2.1b).
   `pn`=proxy, `en`=endpoint. **실행 옵션 차이는 없다** — 세 체인에 proxy 모드 플래그가 없다(실측).
   ~~proxied 그래프 규칙~~ → **확정: `bp ↔ pn ↔ en`, en 은 bp 를 직접 알지 못한다.**
   ~~wemix 의 pn~~ → **확정: poa 는 pn 을 쓰지 않는다(etcd 가 그 자리). 선언하면 오류.**
2. **Blueprint 를 어디에 둘 것인가.** 노드 IP·포트를 담으므로 [[server-inventory]] 와 같은 민감도다.
   서버 인벤토리를 참조만 하고(`server: local`) 자신은 IP 를 담지 않게 하면 커밋 가능해진다 —
   **그 편이 낫다고 본다.**
3. **DSL `env` 선언과 Blueprint 의 관계.** [[dsl-v2-proposal]] 의 `env` 가 Blueprint 의 부분집합인가,
   Blueprint 를 가리키는 참조인가. 후자가 단순하다.
4. **기존 `topology.yaml` 을 어떻게 할 것인가.** Blueprint 의 `nodes:` 가 상위집합이므로
   흡수 대상이다. 이관 기간에는 둘 다 받되 혼용은 거부한다.
