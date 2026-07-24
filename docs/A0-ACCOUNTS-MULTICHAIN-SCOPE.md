# A0 Spike — accounts SDK 다체인화 범위 조사

> 작성일: 2026-07-24
> 성격: **spike(조사) 결과 — 코드 미변경**. `CHAINBENCH_GO_REDESIGN.md` §5.2(D10) 워크스트림 A의 선행.
> 목적: `github.com/0xmhha/accounts`(`../accounts`)가 go-stablenet에 얼마나 깊이 결합됐는지 확인해,
> 다체인 리팩토링(A1)의 범위·비용을 확정한다.
> 대상 근거: `../accounts` (커밋 `fb27df7`), `../chain/{go-stablenet,go-wbft,go-wemix}/core/types`.

---

## 1. 요약 (TL;DR)

- **결합은 얕고 이미 잘 격리돼 있다.** 암호·서명·tx 인코딩 계층은 대부분 체인 공통이고,
  stablenet-전용은 **계정상태(Extra) + 시스템컨트랙트 바인딩(governance/token/AccountFlags)** 3곳에 국한.
- **핵심 반전 2가지 (코드 근거)**:
  1. **0x16 fee-delegation은 stablenet 전용이 아니다.** 세 체인 모두 `FeeDelegateDynamicFeeTxType`(=0x16=22)를
     `core/types/transaction.go`에 정의. → accounts `tx/feedelegation.go`는 **공통 후보** (인코딩 동등성은 A1 골든검증).
  2. **Extra 비트맵(blacklist/authorized)은 stablenet 전용.** `state_account.go`에 Extra 필드:
     stablenet 5회, **wbft·wemix 0회**. → accounts `account/extra.go`·`transport.AccountFlags`·`governance` blacklist가 진짜 stablenet-결합.
- **결론**: A1 리팩토링은 "대규모 재작성"이 아니라 **(a) 공통 코어 유지 + (b) 시스템컨트랙트/계정상태를 프로파일로 분리**.
  예상 규모 = 중(中). accounts는 독립 repo·골든벡터가 있어 회귀 안전.

---

## 2. 조사 방법

- accounts 각 패키지의 라인 수·의존·stablenet 마커(`stablenet/0x16/extra/blacklist/anzeon/WKRC/chainId`) grep.
- 핵심 파일 정독: `tx/{types,feedelegation}.go`, `account/{extra,account}.go`, `transport/client.go`,
  `governance/governance.go`, `token/token.go`.
- 교차검증: 세 체인 `core/types/{transaction,state_account,feedelegate_*}.go`에서 0x16·Extra 실재 여부.

---

## 3. 패키지별 결합 지도 (증거 기반)

| accounts 패키지 | 라인 | stablenet 결합 | 다체인 판정 | 근거 |
|---|---|---|---|---|
| `crypto` | — | 없음 | **공통 그대로** | secp256k1/keccak/ECIES — EVM 3체인 공통 |
| `signing` (Scheme, EIP-191/712) | — | 없음 | **공통** | `Scheme`은 `Secp256k1{}` 하나, chainID는 파라미터 |
| `keystore` `types` `hdwallet` `vault` `abi` | — | 없음 | **공통** | 표준 프리미티브 |
| `tx/` 0x00–0x04 (legacy/accesslist/dynamicfee/blob/setcode) | ~370 | 없음 | **공통** | chainID는 인자, 표준 EIP 인코딩 |
| `tx/feedelegation.go` (0x16) | 137 | **없음(공유)** | **공통** ⭐ | 3체인 모두 0x16 정의. 인코딩 동등성만 A1 골든검증 |
| `account/account.go` | 98 | 없음 | **공통** | 키/주소/서명 래퍼 |
| `account/extra.go` (Extra 비트맵) | 68 | **stablenet 전용** ⭐ | **프로파일 분리** | Extra: stablenet만 `state_account.go` 보유(wbft/wemix 0) |
| `transport/client.go` base `eth_*` | ~200 | 없음 | **공통** | chainId/nonce/gasPrice/call/sendRaw/balance/code |
| `transport.AccountFlags` (eth_getProof.extra) | ~25 | **stablenet 전용** | **프로파일 분리** | Extra 노출은 anzeon만 |
| `governance/` (GovValidator/Council @1001/1004) | 126 | **stablenet 전용** | **체인별 바인딩** | anzeon 시스템컨트랙트 주소/ABI |
| `token/` (NativeCoinAdapter @1000, WKRC) | 197 | **stablenet 전용** | **체인별 바인딩** | anzeon 네이티브코인 어댑터 |

⭐ = A0에서 기존 가정이 뒤집힌 항목.

### 3.1 tx 타입 분포 (세 체인 교차)

| tx type | stablenet | wbft | wemix | accounts 처리 |
|---|---|---|---|---|
| 0x00–0x04 (표준 EIP) | ✅ | ✅ | ✅ | 공통 baseline |
| **0x16 fee-delegation** | ✅ | ✅ | ✅ | **공통** (envelope/sighash 동등성 A1 검증) |

### 3.2 계정상태·시스템컨트랙트 분포

| 기능 | stablenet | wbft | wemix |
|---|---|---|---|
| Extra(blacklist/authorized) 비트맵 | ✅ | ❌ | ❌ |
| governance 시스템컨트랙트(0x1001/1002/1004) | ✅ (anzeon) | 다름(GovContracts) | 다름(etcd governance) |
| NativeCoinAdapter(0x1000) | ✅ (WKRC) | ? | ? |

→ 계정상태·시스템컨트랙트는 **체인/합의별로 상이** = 프로파일 분리 지점.

---

## 4. A1 리팩토링 권고 (범위·구조)

accounts를 **공통 코어 + 체인 프로파일**로 분리. 기존 flat-root 레이아웃 유지, 결합부만 프로파일화.

```
accounts/
├─ crypto signing keystore types hdwallet vault abi   # 공통 (변경 최소)
├─ tx/                                                 # 공통: baseline + 0x16
│   └─ (Protocol 프로파일로 "지원 tx타입 셋" 선택; 0x16 opt-in이 아니라 공통 후보)
├─ account/                                            # 공통 코어
│   └─ extra.go → profile/stablenet 하위로 (anzeon 전용)
├─ transport/                                          # base eth_* 공통
│   └─ AccountFlags → profile/stablenet (eth_getProof.extra)
├─ profile/                                            # (신규) 체인/합의별 상이부
│   ├─ stablenet/  governance(1001..) + token(1000/WKRC) + Extra
│   ├─ wbft/       GovContracts 바인딩 (go-wbft params 기준)
│   └─ wemix/      etcd/governance 바인딩
└─ docs/spec/protocol/{v0=stablenet, ...}              # 프로토콜 스펙 체인별 버전
```

핵심 원칙(accounts repo의 clean-room·spec-first·골든벡터 계승):
1. **공통(crypto/signing/tx baseline)은 건드리지 않음** — 회귀 위험 0.
2. **0x16은 공통으로 승격하되** 세 체인 노드 골든벡터로 envelope/sighash 동등성 검증(다르면 프로파일별 분기).
3. **Extra·governance·token은 `profile/<chain>`로 이동** — stablenet 것은 그대로, wbft/wemix는 각 체인 스펙으로 신규.
4. `Protocol` 프로파일 = { 지원 tx타입, 계정상태 모델, 시스템컨트랙트 주소/ABI } 를 캡슐화 → chainbench `AccountProvider`가 이걸 주입받음.

---

## 5. A1 → chainbench 연결

- chainbench `pkg/accounts.AccountProvider`(D3)는 `accounts.Protocol`을 소비.
  wbft family(stablenet+wbft)는 A1의 stablenet/wbft 프로파일로 즉시 충족 → chainbench G1 진입 가능.
- **의존 순서 확정**: A0(완료) → **A1: 공통 유지 + stablenet/wbft 프로파일 우선** → chainbench G1.
  wemix 프로파일은 뒤로(chainbench G7 poa와 정렬).

---

## 6. 미해결 (A1에서 확정)

- **0x16 envelope 동등성**: ✅ **정적 소스 비교로 확인됨(2026-07-24)**. 세 체인의 fee-payer sigHash가 동일:
  - stablenet/wbft `tx_fee_delegation.go`: `prefixedRlpHash(0x16, [[chainID,nonce,tipCap,feeCap,gas,to,value,data,accessList,senderV,senderR,senderS], feePayer])`
  - wemix `feeDelegateSigner.Hash`(transaction_signing.go): 동일 공식.
  - accounts `tx/feedelegation.go` `FeePayerSigHash`: 동일 공식.
  - stablenet vs wbft 파일 차이는 `effectiveGasPrice`의 gas 계산 1줄뿐(sighash/envelope 무관).
  → **0x16은 진정 공통.** 다만 build한 3개 노드로 raw tx 바이트를 뽑는 **live golden 회귀**는 A1 착수 시 최종 확인.
- **wbft/wemix 시스템컨트랙트 스펙**: governance/token 주소·ABI를 각 체인 `params`·`systemcontracts`에서 스펙화.
  wbft `GovContracts`(config.croissant), wemix etcd governance가 출발점.
- **wemix 계정상태 특이점**: Extra는 없으나 poa 고유 상태(스테이킹/멤버십)가 SDK 노출 대상인지 G7 전 판단.
