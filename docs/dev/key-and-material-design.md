# 키와 자료의 소유 구조 — keyring 통합 · 원격 자료 배치

> **[현행 설계]** 키 자료와 원격 배치.
> 지금 향하는 목표. 근거는 정본([[chainbench-requirements-review]]·[[chainbench-feature-spec]])이고,
> 작업 순서는 [[chainbench-worklist]] §1g 다.

> [[keyring-design]] 이 *키가 무엇인가* 를 정한다면, 이 문서는 **그 키와 나머지 자료가 어디에
> 놓이고 언제 다시 올라가는가** 를 정한다. 실측: 2026-08-20.

---

## 1. 문제 — 측정한 것만

### 1.1 "신원" 을 뜻하는 타입이 다섯 개다

| 타입 | 어디 | 담는 것 |
|---|---|---|
| ~~`keys.NodeKey`~~ | `core/keys` | **K3 에서 제거** |
| ~~`keygen.Node`~~ | `keygen` | **K3 에서 제거** |
| ~~`keyreg.Key`~~ | `core/keyreg` | **K3 에서 제거** — `keyring.Entry` 가 대신한다 |
| ~~`deploy.NodeKeyInfo`~~ | `chains/wemix/deploy` | **K7b 에서 제거** — `ServerIdentity` 가 `keyring.Identity` 를 임베드한다 |
| `keyring.Identity` | `core/keyring` | pubkey · address · BLS *(신규)* |

다섯 개가 **같은 것의 부분집합**이다. 어느 것도 전체가 아니고, 어느 것도 다른 것으로 변환되지
않는다. 호출자는 매번 필드를 손으로 옮겨 담는다.

> `launchopt.Identity` 는 이름만 같고 **다른 개념**(argv 조립용)이다. 이것이 이름 규칙의 두 번째
> 절반 — *다른 개념은 다른 이름* — 을 어긴 사례다.

### 1.2 키 관심사가 6패키지 1,565줄(테스트 제외)에 흩어져 있다

```
core/keyring  352   생성·파생          (K0 신설)
keygen        378   preset 생성
keymat        338   HD경로·비밀번호·저장 백엔드·원격 읽기
core/keyreg   279   런타임 등록·주소대조·업로드
core/keys     136   preset 읽기
validatorset   82   체인별 검증자 로스터
```

**읽기는 `keys`, 쓰기는 `keygen`, 저장은 `keymat`, 런타임은 `keyreg`.** 동사로 패키지를 쪼갠
결과, 하나의 명사(키)를 넷이 나눠 갖고 있다.

### 1.3 `keyreg` 는 `keyring` 과 다른 것이 아니다 — 이미 keyring 이다

지적하신 것이 정확하다. `keyreg.Ensure` 의 `Source` 를 보면 **전부 "키를 어디서 가져오는가"** 다.

```go
type Source int
const (
    Random          // 새로 만든다
    LocalFile       // 로컬 파일에서 가져온다
    RemoteDownload  // 원격에서 가져온다
    Literal         // 이미 손에 있는 값을 등록한다
)
```

이것은 **`keyring import` 의 인자**이지 별도 개념이 아니다. 이름 붙은 키의 조회(`Get`)·
영속(`persist`)·전달(`UploadTo`) 도 마찬가지로 링의 동작이다.

**그리고 원격은 별개의 개념이 아니다.** "이미 있는 키를 쓴다" 가 개념이고, 그 키가 로컬 경로에
있느냐 원격 경로에 있느냐는 **경로 표기의 차이**일 뿐이다. `core/target.ParseTarget` 이 이미 세
표기를 하나로 파싱한다.

```
/srv/keys/node1                  로컬
srv://bp1/srv/keys/node1         인벤토리 이름 (명령줄에 IP 없음)
user@host:/srv/keys/node1        raw 원격
```

#### keyreg 가 분리돼 있던 이유는 K0 이 없앴다

`keyreg` 는 파생을 **주입**받는다.

```go
type BLSDeriver interface { Derive(ctx, private []byte) (bls, pop []byte, err error) }
type Deps struct { Generate func() ...; DeriveAddress func(priv []byte) (string, error); ... }
```

주입한 이유는 파생이 **외부 바이너리 실행**이었기 때문이다 — `ctx` 로 프로세스를 제한해야 했고,
바이너리 부재를 오류로 만들어야 했다. **K1 이 그 실행을 없앴으므로 주입할 이유가 사라졌다.**
`keyring.Derive` 는 순수 함수이고 `ctx` 도 필요 없다.

남는 주입 대상은 `FetchRemote` 하나인데, 그것은 파생이 아니라 **읽기**이고 §3 의 `FileStore` 가
가져간다.

### 1.4 god function 둘

| 함수 | 줄 | 하는 일 |
|---|---:|---|
| `keyreg.Ensure` | 34 | 중복확인 · 4소스 분기 · 주소대조 · BLS파생 · 영속 · 메모이즈 — **6가지** |
| `keygen.GeneratePreset` | 61 | 검증 · mkdir · 비밀번호쓰기 · N회생성 · alloc · validators · 시스템컨트랙트 · extraData · 마샬 · 파일쓰기 — **10가지** |

둘 다 "실패하면 어디까지 됐는지 모르는" 형태다. `GeneratePreset` 은 5번째 노드에서 실패하면
앞선 4개의 디렉토리가 남는다.

### 1.5 재업로드 방지가 **존재 여부만** 본다 — 느린 게 아니라 틀린다

질문하신 부분이다. 검토됐고([[chainbench-worklist]] N12), **설계는 아직 없으며, 현재 동작은
비효율이 아니라 결함**이다.

```go
// internal/core/provision/provision.go:52
exists, err := p.sink.Exists(ctx, full)
if exists { res.Skipped++; continue }      // ← 내용을 보지 않는다
```

경로가 고정(`<datadir>/genesis.json`)이므로, **다른 테스트의 genesis 가 같은 경로에 있으면 그것을
그대로 쓴다.** 노드는 뜨고, 블록은 안 붙고, 원인은 로그에 없다. 앞서 wemix 가 블록 0 에서 멈춘
것과 같은 종류의 실패다.

즉 "다시 올리지 말자" 와 "틀린 것을 쓰지 말자" 는 **같은 문제의 양면**이고, §4 가 한 번에 푼다.

### 1.6 원격 경로가 한 사이트의 레이아웃으로 박혀 있다

```go
// chains/wemix/deploy/cluster.go
Nodekey:          "/data/go-wbft/conf/nodekey"
CoinbaseKeystore: "/data/go-wbft/conf/keystore/coinbase"
```

체인 구분도, 테스트 구분도, 버전 구분도 없다. 노드 하나에 자료 한 벌만 존재할 수 있다.

---

## 2. 원칙 — 이 문서가 적용하는 단일 설계 규칙

1. **명사로 소유한다.** 패키지는 동사(읽기/쓰기/저장)가 아니라 명사(키·자료)로 나눈다.
2. **로컬과 원격은 표기의 차이다.** 상위 레이어에 분기가 없다.
3. **이름은 선언에서 온다.** 사람이 읽는 이름(`env.id`)이 경로가 되고, 내용 동일성은
   해시로 *검사*한다 — 해시를 이름으로 쓰지 않는다.
4. **휘발 자료와 불변 자료를 섞지 않는다.** datadir 은 지워지고, 자료는 남는다.
5. **비밀은 타입이 가린다.** [[keyring-design]] 의 `Nodekey` 규칙을 유지한다.

---

## 3. 키 — `core/keyring` 하나로

```
core/keyring (L1)
├─ Nodekey        루트 비밀. 스스로를 가린다                      (K0 완료)
├─ Identity       파생된 공개 신원 — 다섯 타입을 대체              (K0 완료)
├─ Derive         주소·pubkey·BLS·PoP, in-process                (K0/K1 완료)
├─ Source         키가 어디서 오는가 — new | import              ← keyreg.Source 흡수
├─ Ring           이름 붙은 항목의 조회·영속·전달                  ← keyreg 흡수
└─ Backend        raw hex | keystore v3 + PasswordSource         ← keymat 승격
```

### 3.1 `Ensure` 를 넷으로 쪼갠다

god function 을 없애되, **각 조각이 자기 이유를 갖도록** 한다.

```go
// 1. 자료를 얻는다 — 이것이 "이미 있는 것을 쓴다" 의 전부다
func (r *Ring) Import(ctx context.Context, name Label, from target.TargetSpec) (Entry, error)
func (r *Ring) New(name Label, d Derivation) (Entry, error)

// 2. 선언과 실물이 어긋나는지 본다 — 별도 함수여야 테스트가 쉽다
func (e Entry) Verify(declared Address) error

// 3. 저장한다
func (r *Ring) Save(e Entry) error
```

`Import` 가 로컬·원격을 **모두** 처리한다. 분기는 `target.Resolve` 안에서 끝나고, 위로 새지 않는다.

### 3.2 `Label` 을 명명된 타입으로

지금 이름은 `string` 이고 map 키로 쓰인다. 규칙(*식별자가 패키지 경계를 넘거나 map 키가 되면
명명된 타입*)의 대상이다.

```go
// Label names one entry in a ring: "bp1", "en2", "faucet".
type Label string
```

`place.NodePlacement.Name` · `keyreg.Key.Name` · 계정 라벨이 전부 이 타입으로 모인다
([[network-blueprint-design]] N7 `netmap`).

### 3.3 `FileStore` — 읽기를 더한다 (K6)

```go
type FileStore interface {
    Exists(ctx, path) (bool, error)
    Read(ctx, path) ([]byte, error)      // 추가
    Write(ctx, path, content, mode) error
}
```

`keymat.RemoteFileSource.sshRead` 와 `deploy.readRemoteFile` 이 각자 만든 SSH 읽기가
**하나로 접힌다.** `deploy.ParseBootnodeOutput` 과 `deploy.NodeKeyInfo` 는 그때 사라진다.

> **보안 결정(확정)**: 원격 평문 nodekey 를 로컬로 내린다. 저장 위치는 **세션 트리의 `keys/`
> 한정**(이미 0700)이며, `/tmp` 를 쓰지 않는다. 세션은 `gc` 로 지우는 경로가 이미 있다.

---

## 4. 자료 — destination 레이아웃

### 4.1 두 개의 서로 다른 수명

| | 수명 | 예 |
|---|---|---|
| **material** | 만들어지면 **안 바뀐다** | genesis · config · keystore · nodekey · 바이너리 |
| **run** | 노드가 죽으면 **지운다** | datadir(chaindata·로그·etcd 상태) |

지금은 둘이 같은 디렉토리에 섞여 있어서, datadir 을 지우려면 자료도 같이 지워진다. 그래서 매번
다시 올린다.

### 4.2 이름은 이미 정해져 있다 — DSL 의 `env.id`

자료의 이름을 새로 만들 필요가 없다. **DSL 이 이미 이름을 선언한다.**

```go
// internal/testspec/spec_v2.go — EnvV2 는 "재사용 단위" 로 정의돼 있다
type EnvV2 struct {
    ID      string   `json:"id"`                // ← 이 환경의 이름
    Chain   string   `json:"chain"`
    Config  string   `json:"config,omitempty"`  // ← 어떤 config 를 쓰는가
    ...
}
```

테스트도 이미 약속된 이름을 갖는다 — `tests/specs/consensus/istanbul-status-fields.json` 의
`id` 는 `istanbul-status-fields` 다. 자료 경로는 **그 이름을 그대로 쓴다.**

런타임은 그 이름과 서버 정보를 조합해 실제 경로를 만든다. 선언에는 경로가 없고, 경로에는
호스트가 없다 — 어느 쪽도 상대를 알 필요가 없다.

```
DSL:      { "id": "delayed-fork-4bp", "chain": "stablenet", "config": "delayed-fork-4bp" }
인벤토리:  bp1 → 호스트·자격증명·destination 루트
─────────────────────────────────────────────────────────────────
런타임:    <destination>/material/stablenet/config/delayed-fork-4bp/node1.toml
```

### 4.3 레이아웃

```
<destination>/                      운영자 지정. 서버마다 같은 구조
├── bin/
│   └── <chain>/<build>/            체인 · 빌드별 실행 바이너리
│       └── gstable                 하드포크는 두 빌드가 동시에 필요하다
│
├── material/                       불변. 이름은 선언에서 온다
│   └── <chain>/
│       ├── genesis/<env-id>.json
│       ├── config/<env-id>/node1.toml …
│       ├── nodekey/<keyset>/node1 …          0600
│       └── keystore/<keyset>/node1/UTC--… …  0600
│
├── run/
│   └── <run-id>/node<N>/           datadir. 시작 전 삭제·재생성
│
└── .material.json                  업로드 대장 (§4.4)
```

**`<chain>` 을 최상위로 두는 이유**: 세 체인이 한 서버에 공존할 수 있고(하드포크 핸드오프는
실제로 그렇다), 자료의 형식 자체가 체인마다 다르다.

**`<build>` 를 바이너리 경로에 두는 이유**: 하드포크 테스트는 **같은 서버에서 두 바이너리가
동시에** 필요하다([[chainbench-requirements-review]] §D-2.8). 경로가 하나면 표현할 수 없다.

**`config/` 아래에 분류 폴더(`api/`·`consensus/`…)를 두지 않는다.** `env-id` 만으로 이미
유일하고, 분류를 넣으면 런타임이 경로를 만들기 위해 **선언에 없는 정보(어느 분류인가)를 따로
알아야 한다.** 분류는 저장소의 `tests/specs/<분류>/` 가 갖고, 배치는 이름만 갖는다.
분류가 정말로 경로에 필요해지면 그때 `env.id` 에 접두사를 넣는 편이 낫다 — 선언 안에 남는다.

**`<keyset>`** 은 키셋의 이름이다(`preset`·`dev4`·운영자가 정한 이름). 키는 env 보다 수명이
길다 — 여러 env 가 같은 키셋을 공유하므로 env-id 로 묶으면 같은 키를 여러 번 올리게 된다.

### 4.4 이름이 같은데 내용이 바뀐 경우 — 대장으로 잡는다

이름을 쓰면 §1.5 의 결함이 그대로 남는다. `delayed-fork-4bp` 의 genesis 를 고쳤는데 서버에
같은 이름의 옛 파일이 있으면, `Exists` 는 참이고 **옛 것이 그대로 쓰인다.**

그래서 **내용 해시를 이름이 아니라 데이터로** 쓴다. destination 루트에 대장 하나를 둔다.

```json
// <destination>/.material.json
{
  "material/stablenet/genesis/delayed-fork-4bp.json": {
    "sha256": "a3f9c1e0…", "size": 4211, "uploadedAt": "2026-08-20T04:11:03Z"
  }
}
```

업로드 판정은 두 조건을 **모두** 본다.

| 대장의 해시 | 파일 존재 | 판정 |
|---|---|---|
| 같음 | 있음 | **생략** — 반복 실행이 빨라진다 |
| 다름 | 있음 | **올린다** — 고친 자료가 반영된다 |
| — | 없음 | 올린다 (손으로 지웠거나 첫 실행) |

대장 읽기는 세션당 한 번, 작은 파일 하나다. 파일마다 원격을 읽어 비교하는 것과 달리 왕복이
늘지 않는다.

> **이것이 digest 를 경로에 넣지 않고도 안전한 이유다.** 해시는 *식별자*가 아니라 *검사값*으로
> 쓰인다 — 사람이 읽는 이름은 선언에서 오고, 기계가 보는 동일성은 대장이 판정한다.

> **동시 실행 주의**: 같은 `env-id` 로 내용이 다른 두 실행이 겹치면 나중 것이 이긴다. 대장에
> 그 사실이 남으므로 조용히 틀리지는 않지만, 막지도 못한다. 동시 실행이 필요해지면 `env-id` 를
> 실행마다 다르게 짓는 것이 답이고, 경로 규칙은 그대로 둔다.

### 4.4 datadir 은 material 을 **참조**한다

노드를 띄울 때 material 을 datadir 로 복사하지 않는다. 필요한 것만 링크하거나 인자로 가리킨다.

```
run/<run-id>/node1/          ← 지워도 되는 것만 들어간다
  geth/                        chaindata
  nodekey                    → material/stablenet/nodekey/<keyset>/node1
  keystore/                  → material/stablenet/keystore/<keyset>/node1/
```

genesis 와 config 는 **argv 로 경로를 가리키므로 복사가 아예 필요 없다.**

```
--datadir run/<run-id>/node1  --config material/stablenet/config/<env-id>/node1.toml
```

그래서 "테스트마다 다른 자료를 쓰되 업로드는 한 번" 이 성립한다 — 경로만 바꿔 지정한다.

### 4.5 세션 트리와의 관계

세션 트리(로컬, 이미 존재)와 destination(원격/로컬 타깃)은 **역할이 다르다.**

| | 세션 트리 `<artifact-root>/UTC-…/` | destination |
|---|---|---|
| 무엇 | 이 실행의 **기록** — 키·환경·테스트 결과 | 노드가 **읽는 자료** |
| 수명 | 감사·재현용으로 남는다 | 재사용된다 |
| 위치 | 항상 로컬 | 로컬 또는 원격 |

새 수집 장소를 만들지 않는다. 세션 트리는 그대로 두고, destination 을 **명시적 개념으로 승격**한다.

---

## 5. 착수 순서 (K6·K7 을 앞으로)

| # | 작업 | 게이트 |
|---|---|---|
| **K6** | `FileSink` → `FileStore`(읽기 추가). `keymat.sshRead`·`deploy.readRemoteFile` 흡수 | ☑ 자체 SSH 파일 I/O 9곳 → 0 |
| **K7** | `--from` — 로컬·`srv://`·`user@host:`·`ssh://` 한 코드로 | ☑ **명령줄에 IP 가 없다** · 플래그 4 → 1 |
| **K7b** | `deploy.ReadServerKeys` 를 K7 위에 재구성 | ☑ `ParseBootnodeOutput`·`NodeKeyInfo` 소멸 · 서버에 bootnode 불필요 · 신원 타입 5 → 3 |
| **M1** | `material` 레이아웃(`env-id` 이름) + 업로드 대장 | 같은 자료 재업로드 0회 · **이름이 같고 내용이 다르면 반드시 다시 올린다** |
| **M2** | `run/<run-id>` 분리 + datadir 재생성 | datadir 삭제가 자료를 건드리지 않음 |
| **M3** | `bin/<chain>/<build>` | 한 서버에 두 빌드 공존(하드포크) |
| **K2** | `keygen.WBFTExtraData` → `consensus/wbft.ExtraData` | ☑ 파생값은 저장하지 않는다 — `BuildGenesis` 가 계산 |
| **K3** | `keys`·`keygen`·`keymat`·`keyreg` → `keyring` | ☑ 5패키지 → 1 · 신원 타입 5 → 1 |
| **K4** | `keyring` 명령 (new/add/list/show/import/export) | `keys`·`validator`·`account` 대체 |
| **K5** | preset 분해 — `metadata.json` 은 `nodes[]` 만 | 기존 파일 호환 |

**K6·K7 을 K2·K3 보다 먼저** 하는 이유: K3 는 `keymat`·`keyreg` 를 흡수하는데, 그 둘이 가진
원격 읽기가 K6 에서 정리되지 않으면 **정리되지 않은 채로 흡수**된다. 흡수 전에 접어야 한다.

---

## 6. 아직 결정하지 않은 것

| # | 질문 |
|---|---|
| M-a | 대장(`.material.json`)의 위치 — destination 루트인가, 서버별 상태 디렉토리인가 |
| M-b | material 의 회수 정책 — 무한히 쌓인다. `gc` 대상인가, 운영자 책임인가 |
| M-c | datadir 의 material 참조 — 심볼릭 링크인가 복사인가 (원격 파일시스템·컨테이너에서 링크가 안 될 수 있다) |
| M-d | `<build>` 식별자 — 바이너리 내용 해시인가, 운영자가 붙인 이름인가 |
