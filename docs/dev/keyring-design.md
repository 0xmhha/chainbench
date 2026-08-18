# keyring — 키 자료의 단일 소유자

> 첫 착수 대상. 키는 세 체인이 **동일하므로** 여기부터 정리하면 위쪽이 단순해진다.
>
> 실측: 2026-08-18. 관련: [[network-blueprint-design]](network-blueprint-design.md) ·
> [[surface-unification-design]](surface-unification-design.md).
> 작업 순서는 [[chainbench-worklist]](chainbench-worklist.md) §1g.

---

## 1. 키는 세 체인이 동일하다 (실증)

같은 nodekey 를 세 체인 도구에 넣어 확인했다.

| 검증 | 결과 |
|---|---|
| devp2p 공개키 | go-wbft `public key` == go-wemix `idv5` — **바이트 동일** |
| 주소 | 세 체인 모두 `0x707066b9…fdeccb4`. **chainbench 자체 Go 파생과도 일치** |
| BLS 공개키·PoP | **chainbench Go 파생이 `bootnode` 출력과 바이트 동일** |
| keystore 형식 | geth v3 JSON 공통 |

**다른 것은 두 가지뿐이고, 둘 다 키 구조가 아니라 사용 정책이다.**

1. **BLS 사용 여부** — wbft 계열만 쓴다(`consensus/poa` BLS 참조 0건).
2. **합의 계정이 nodekey 파생 주소인가** — wbft 계열은 파생, wemix 는 관례상 별도 keystore.

### 1.1 파생은 전부 Go 로 가능하다 — 외부 바이너리 불필요

```
nodekey (32B secp256k1, crypto/rand)
  ├─ address        keccak(pubkey)[12:]                    accounts.AddressForKey (기존)
  ├─ devp2p pubkey  secp256k1 uncompressed (128 hex)       Go 내장
  ├─ BLS pubkey     blst.KeyGen(nodekey) → G1 (48B)        blst
  └─ BLS PoP        Sign(pubkey, DST) → G2 (96B)           blst
                    DST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_"
```

> **DST 주의**: 빼먹으면 **형식은 멀쩡한데 검증에 실패하는 PoP** 이 나온다. 실제로 첫 시도에서
> 그렇게 됐다. 골든 테스트(배포된 preset 재현)로 고정한다.

의존성 `github.com/supranational/blst v0.3.16` — BLS12-381 표준 구현체이고,
**go-wbft 자신이 같은 버전을 쓴다**. 오타 낚시가 아니며 이더리움 합의 클라이언트 전반이 사용한다.

**결과: `--bootnode` 플래그가 사라진다.** 새 키셋 생성에 어떤 체인 바이너리도 필요 없다 —
이것이 §1g 의 "raw 가 먼저" 를 실제로 가능하게 한다.

---

## 2. 지금 preset 은 세 가지를 섞어 담고 있다 (실측)

`keys/preset/metadata.json` 의 내용을 성격별로 나누면:

| 담긴 것 | 성격 | 실제 주인 |
|---|---|---|
| `nodes[]` (5) | **신원** — nodekey·address·pubkey·bls·pop | **keyring** |
| `validators`·`blsPublicKeys` | **네트워크 결정** — 누가 검증자인가 | **청사진** |
| `alloc`·`systemContractMembers`·`systemContractBlsKeys` | **네트워크 결정** — 잔액·시스템 계정 | **청사진** |
| `extraData` | **파생 산출물** — 검증자+BLS 에서 계산됨 | **genesis (저장 대상 아님)** |

**이것이 "preset 이 전제가 된" 이유다.** 신원·결정·산출물을 한 파일이 다 갖고 있어서,
preset 없이는 아무것도 정할 수 없었다.

### 2.1 분해

```
keyring   nodes[]                        신원만. 네트워크를 모른다.
청사진    validators · alloc · roles      결정. 신원을 이름으로 참조한다.
genesis   extraData                       파생. 저장하지 않고 매번 계산한다.
```

`extraData` 를 저장하지 않는 것이 중요하다 — 저장하면 검증자 집합과 어긋날 수 있고,
그 불일치는 genesis 를 통과한 뒤 합의에서 터진다(BLS 를 선언 필드로 두면 안 되는 것과 같은 이유).
계산 함수는 이미 있다(`keygen.WBFTExtraData`, 배포 preset 재현 골든 테스트 통과) —
**`consensus/wbft` 로 옮긴다.** genesis 자료이지 키 자료가 아니다.

> **하위호환**: 기존 `keys/preset/metadata.json` 은 그대로 읽는다. keyring 은 `nodes[]` 만 취하고,
> 나머지는 `net blueprint --from-keyring` 이 청사진 초안으로 옮긴다. 파일을 깨지 않는다.

---

## 3. 모듈 — 5개 → 1개

지금 키 관심사가 5개 패키지 **1,236줄**에 흩어져 있다.

| 지금 | 줄 | 하는 일 | 어디로 |
|---|---:|---|---|
| `core/keys` | 136 | preset **읽기** | → keyring |
| `keygen` | 401 | preset **생성**(bootnode 경유) | → keyring (bootnode 제거) |
| `keygen.WBFTExtraData` | | genesis extraData 계산 | → **`consensus/wbft`** |
| `keymat` | 338 | HD path · password · 저장 백엔드 · 원격 조회 | → keyring |
| `core/keyreg` | 279 | 런타임 등록·주소대조·0600·원격 전달 | → keyring |
| `validatorset` | 82 | 체인별 검증자 로스터 | → **`resolve`(L3)** |

**읽기는 `keys`, 쓰기는 `keygen`, 저장은 `keymat`, 런타임은 `keyreg`** — 같은 것을 넷이 나눠 갖고 있다.

### 3.1 목표

```
core/keyring (L1)
├─ 생성    secp256k1 (crypto/rand)                  체인 무관
├─ 파생    address · pubkey · enode · BLS · PoP     Go 내장 + blst
├─ 백엔드  raw hex | keystore v3                    ← keymat.Store 승격
│          + PasswordSource                          (이미 seam 존재)
├─ 링      이름 붙은 항목 · 조회 · 원격 전달        ← keyreg 흡수
└─ 색인    metadata.json — 링의 목차
```

**`keyreg` 를 흡수하는 근거**: 그것이 이미 keyring 이다 —
`Ensure(name, …)` · `Get(name)` · `UploadTo(…)` 는 이름 붙은 키 집합의 조회·영속·전달이다.
이름을 둘로 두면 이 프로젝트가 반복해 온 "유사하면서 다른 것"이 하나 더 는다.

**백엔드가 이미 여럿이라는 것도 근거다** — `RawFileStore`(raw hex) / `KeystoreStore`(암호화 v3) +
`PasswordSource`. cosmos-sdk 의 `--keyring-backend` 와 같은 모양이 이미 구현돼 있다.

---

## 4. 명령

> cosmos-sdk 는 **명령이 `keys`, `keyring` 은 백엔드 이름**이다. chainbench 는 명사 기반 CLI 로
> 가므로(`net`·`test`·`report`·`chain`·`server`) **명령도 `keyring`** 으로 둔다 — 의식적 이탈이다.

### 4.1 링의 위치는 항상 명시된다

```
--keyring <dir>        대상 링. 기본값 ./keys/default, 환경변수 CHAINBENCH_KEYRING 로 덮음
```

`keys/preset` 은 특별한 것이 아니라 **디스크에 있는 링 하나**다.

### 4.2 동사

```sh
# 링 통째로 생성 (= preset 생성). --out 이 링의 위치다.
keyring new  --out ./keys/dev --count 4 [--with-bls]

# 기존 링에 항목 추가
keyring add  --keyring ./keys/dev --count 2 [--with-bls] [--name en1]

# 조회
keyring list --keyring ./keys/dev                    요약 표 (name·address·role 후보)
keyring show --keyring ./keys/dev --name node1       상세 (address·pubkey·enode·bls·pop)
keyring show --keyring ./keys/dev --json             전체를 기계 판독 형식으로

# 가져오기 / 내보내기
keyring import --keyring ./keys/dev --hex 0x…               --name node5
keyring import --keyring ./keys/dev --keystore f.json       --name node5
keyring import --keyring ./keys/dev --from <경로>           --name node5   # §4.3
keyring export --keyring ./keys/dev --name node1
```

`keys` · `validator` · `account` 세 명령이 여기로 모인다.
**`--with-bls` 는 wbft 계열에서만 의미가 있고, 붙이지 않으면 BLS 를 만들지 않는다** —
wemix 는 BLS 를 쓰지 않으므로 불필요한 계산을 하지 않는다.

### 4.3 링의 위치는 로컬이든 원격이든 같은 규격이다

이미 만들어 둔 단일 경로 문법([[server-inventory]] · T7.10 `netcompose.ParseTarget`)을 그대로 쓴다.
원 요구의 key point 2 — *상위 레이어는 하나의 "경로"만 본다* — 가 정확히 이것이다.

```
/srv/keys/node1                  로컬 (맨 경로)
srv://bp1/srv/keys/node1         인벤토리 항목 "bp1"        ← 권장
user@host:/srv/keys/node1        raw 원격
ssh://user@host:2222/srv/keys/…  포트 명시
```

**로컬에 루프백 표기를 강제하지 않는다.** `localhost:/path` 를 매번 쓰게 하면 로컬이 예외처럼
느껴진다. 맨 경로가 로컬이고, 그것이 기본이다.

**`srv://` 를 더하는 이유는 보안이다.** `user@host:/path` 를 쓰면 명령줄과 스크립트에 내부 IP 가
박힌다 — 서버 인벤토리를 gitignore 에 둔 이유와 정면으로 충돌한다. 인벤토리가 호스트·포트·
자격증명을 이미 갖고 있으므로 **명령은 이름만 쓴다.**

```sh
keyring import --from srv://bp1/srv/keys/node1 --name bp1
keyring import --from /mnt/backup/keys/node1   --name bp1
keyring list   --keyring srv://bp1/srv/keys/dev        # 링 자체가 원격일 수도 있다
```

현재의 두 플래그(`--server 3 --remote-path /keys/node1`)는 `--from` 하나로 접힌다 —
`net new --target` 이 레거시 4플래그를 접은 것과 같은 정리다.

#### 이를 위해 `FileSink` 가 읽기를 가져야 한다

```go
// 지금 — 쓰기만 있다
type FileSink interface {
    Exists(ctx, path) (bool, error)
    Write(ctx, path, content []byte, mode fs.FileMode) error
}
```

**읽기가 없어서 `keymat.RemoteFileSource` 가 자체 SSH 읽기를 따로 구현했다**(`sshRead` →
`remote.ReadFile`). 키가 자기만의 원격 경로를 갖게 된 원인이 이 비대칭이다 —
추상화가 한쪽 방향만 있으면 반대 방향은 옆에 새로 생긴다.

```go
// 목표 — 타깃 파일에 대한 양방향 통로
type FileStore interface {
    Exists(ctx context.Context, path string) (bool, error)
    Read(ctx context.Context, path string) ([]byte, error)   // 추가
    Write(ctx context.Context, path string, content []byte, mode fs.FileMode) error
}
```

읽기 구현은 이미 있다(`remote.ReadFile`, 로컬은 `os.ReadFile`) — **묶기만 하면 된다.**
이 변경은 keyring 을 넘어선다: 청사진 읽기·genesis 확인·산출물 검증이 전부 같은 통로를 쓰게 된다.

### 4.4 디스크 형태 (기존과 동일)

```
keys/dev/                ← 링 하나
  metadata.json          ← 색인: nodes[] 만 (신원)
  node1/nodekey          ← 항목: raw hex
  node1/keystore/UTC--…  ← 항목: keystore v3 (계정이 별도인 경우)
  password
```

기존 preset 과 같은 배치라 **호환이 깨지지 않는다.** 바뀌는 것은 `metadata.json` 에서
네트워크 결정(validators·alloc·extraData)이 **빠진다**는 것뿐이고, 그것들은 청사진으로 간다.

---

## 5. 게이트

| # | 검증 |
|---|---|
| K1 | **골든**: 배포된 `keys/preset` 의 node1..5 를 nodekey 만으로 재현 — address·pubkey·BLS·PoP 바이트 동일 |
| K2 | `--with-bls` 없이 생성하면 BLS 필드가 **없다**(0값이 아니라 부재) |
| K3 | 링을 명시하지 않으면 기본 경로를 쓰고, **그 경로를 출력에 밝힌다** |
| K4 | `keyring new` 가 **어떤 체인 바이너리도 실행하지 않는다** (프로세스 실행 0회) |
| K5 | 기존 `keys/preset/metadata.json` 을 읽어 `nodes[]` 를 그대로 복원 |
| K6 | 파서 fuzz (metadata.json) |
| K7 | `--from` 이 로컬·`srv://`·`user@host:` 세 형태를 같은 코드로 처리 (원격은 SSH 게이트) |
| K8 | `srv://` 가 **인벤토리에서 호스트·포트·자격증명을 가져온다** — 명령줄에 IP 가 없다 |
