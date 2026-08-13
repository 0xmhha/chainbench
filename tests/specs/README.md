# 이관된 테스트 스펙 (legacy Go-func → DSL)

> `internal/testkit` 에 Go 함수로 등록된 레거시 케이스를 DSL 정의서로 옮긴 것.
> `examples/specs/` 는 **문법 예시**이고, 여기는 **실제 이관분**이다.
> 은퇴 계획: [[legacy-retirement-plan]] (`docs/dev/legacy-retirement-plan.md`)

## 실행

```sh
# 오프라인 검증 (CI 가드가 이걸 돌린다)
chainbench validate tests/specs/*/*.json

# 로컬 체인 기동 후 실행
CHAIN=/path/to/chain
chainbench run --chain stablenet --binary $CHAIN/go-stablenet/build/bin/gstable \
  --keys keys/preset --artifact-root /tmp/out tests/specs/api/*.json

# 이미 떠 있는 네트워크에 붙여서
chainbench run --chain stablenet --rpc http://127.0.0.1:8600 tests/specs/api/*.json
```

## 이관 현황

레거시 등록 케이스 **134개** 기준.

| 카테고리 | 레거시 | 이관 | 상태 |
|---|---:|---:|---|
| `api` | 11 | **10** | ✅ 라이브 통과 (gstable) |
| `consensus` | 13 | **11** | ✅ 라이브 통과 (gstable·gwbft). +randao/mixdigest·block-period(parentHash 워크로 헤드 2쌍 `derive:diff`==1)·wbft-seals-quorum(seal 서명 present). 잔여 3건 갭(아래) |
| `network` | 4 | 3 | ✅ `examples/specs/network-*.json` (선행 이관분). 잔여 1건 `admin-peers-populated` 갭(아래) |
| `accounts` | 35 | **12** | ✅ 라이브 통과 (gstable). value/legacy/dynamic-fee transfer·tx-count·tx-by-hash·receipt·effective-gas·genesis-balance·contract-roundtrip 9건 라이브, secp256r1 precompile 3건은 wbft 전용(gstable 미탑재 확인)이라 오프라인 검증만. 잔여 23건은 문법 갭(아래) |
| `gas-policy` | 17 | **9** | ✅ 라이브 통과 (gstable). read류 3 + tx-flow 6 (basefee min/max·effective-gas·gastip-forced·feecap exact/above-min). `read/assert:"derive"`(sum/diff) 로 정확 산술 비교, `read:"derive"`(read source) 로 `feeCap = baseFee+tip` 를 계산해 sendTx 인자로 주입. 잔여 8건은 문법 갭(아래) |
| `hardfork` | 8 | 0 | ☐ |
| `system-contracts` | 46 | 0 | ☐ |

### 라이브 검증 근거 (2026-08-09)

```
api        (10 spec) → gstable 4노드: pass=10 fail=0
consensus  ( 8 spec) → gstable 4노드: pass=8 fail=0
                     → gwbft   4노드: pass=7 fail=0 skip=1  (stablenet 전용 1건 정상 skip)
```

### 라이브 검증 근거 (2026-08-13)

```
gas-policy ( 9 spec) → gstable 5노드(스크래치, 8601-8605): pass=9 fail=0
                     read 3 + tx-flow 6. sendTx→receipt→derive 산술비교, feeCap 주입 모두 라이브 확인
consensus  (+3 spec) → gstable 5노드(스크래치, 8601-8605):
                     randao-and-mixdigest-present pass, block-period-one-second pass (attach 모드)
                     wbft-seals-quorum 은 attach 모드가 rpc cap 만 광고해 skip →
                     committedSeal/preparedSeal.signature(194자 hex, ≠"0x") 직접 RPC 로 확인
accounts   ( 9 spec) → gstable 5노드(스크래치, 8601-8605): pass=9 fail=0 skip=3
                     value/legacy/dynamic-fee transfer, tx-count 증가, tx-by-hash/receipt 필드,
                     effective-gas, genesis-balance, contract-roundtrip 모두 라이브 확인.
                     secp256r1 precompile 3건은 gstable 에 P256VERIFY(0x100) 미탑재
                     (valid 벡터가 "0x" 반환) → wbft 전용이라 skip. gwbft 바이너리 부재로
                     라이브 미검증, 오프라인 validate + 레거시 자체 테스트 벡터로 이관
```

## 이관하면서 드러난 레거시 케이스의 결함

포팅은 케이스를 다시 읽게 만들고, 그 과정에서 원본의 오류가 드러났다.

| 레거시 케이스 | 문제 | 이관본에서의 처리 |
|---|---|---|
| `wbft-extra-info-fields` | `istanbul_getWbftExtraInfo` 를 **`"latest"` 태그로 호출** — 체인이 `block -2 not found` 로 거부한다. 구체적 블록 번호가 필요 | 헤드를 먼저 읽어(`read` + `$head`) 넘긴다 |
| `wbft-extra-info-fields` | `ChainCompat: [stablenet, wbft]` 인데 **`gasTip` 은 stablenet 전용** — wbft 응답에 그 필드가 없다 | 공통 필드(`committedSeal`·`preparedSeal`)만 양 체인 대상으로 남기고, `gasTip` 은 `stablenet-gastip-field.json` 으로 분리 |

두 건 모두 **라이브에서만** 드러난다. 레거시 유닛 테스트는 mock 노드를 쓰기 때문이다.

## 이관하지 않은 것과 그 이유

| 레거시 케이스 | 왜 |
|---|---|
| `ws-subscribe-logs` | **구독을 먼저 열고 그 다음 로그를 유발**해야 하는데, 어세션은 스텝 뒤에 실행되므로 이미 늦다. 현재 DSL 로는 순서를 표현할 수 없다 — 가짜로 만들지 않고 갭으로 남긴다 |
| ~~`block-period-one-second`~~ | ✅ **이관 완료** — `parentHash` 로 헤드에서 두 단계 되짚어 인접 블록 타임스탬프를 얻고 `derive`(diff)==blockPeriod(1) 로 검증. 파생 블록번호를 hex RPC 파라미터로 되먹일 수 없는 갭을 hash 워크로 우회 |
| `epoch-transition-carries-epoch-info` | 에폭 경계까지 대기 후 그 블록을 조회해야 한다. 조건부 대기 표현이 없다 |
| `validator-set-count` | 검증자 수를 **토폴로지에서 파생**해 비교한다. spec 이 자기 토폴로지를 참조할 수단이 없다(현재는 `Len` 에 상수 4를 쓴다) |
| `prev-seals-quorum` | prevCommitted/prevPrepared seal 의 **sealer 수 >= quorum(ceil 2N/3)** 만 검사한다(서명 필드 없음). 배열 길이 비교 연산자도, 토폴로지 파생 quorum 산술도 없다. `NotNil` 은 빈 배열에도 통과하므로 의미가 없다 — 갭 |
| `admin-peers-populated` (network) | `admin_peers` 결과가 **>=1 엔트리**이고 첫 피어 `id` 가 비어있지 않은지 검사. `select` 는 맵만 walk 하고 **배열 인덱스(`0.id`)를 못 짚으며**, 배열 길이>=N 비교 연산자도 없다 — 갭 |

### gas-policy 잔여 8건과 필요한 문법 확장

tx-flow 6건은 이관 완료. 나머지 8건은 아래 3가지 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `feecap-below-min-rejected` · `legacy-gasprice-below-min-rejected` · `tipcap-underpriced-rejected` | **풀 진입 거부**(eth_sendTransaction 이 에러 반환, 해시 없음). DSL 은 제출 에러를 스텝 실패로 처리할 뿐 부정 기대가 없다. `expectRevert` 는 *채굴 후 status 0x0* 만 표현 | (B) `expectReject`/`expectSubmitError` — 제출 자체가 실패해야 통과. + 이 체인의 below-min 이 제출거부인지 채굴-revert 인지 **라이브 확정** 필요(`examples/specs/stablenet-fee-boundary.json` 은 expectRevert 를 쓰는데 실제 거부면 실패함) |
| `accesslist-gasprice-below-min-rejected` | 위 + **access-list(0x01) 트랜잭션 타입** 미지원 | (B) + (C) sendTx accessList 필드 |
| `gaslimit-exceeded-rejected` | intrinsic-gas 초과 → 제출 거부 추정 | (B) (라이브 확정) |
| `revert-tx-status-zero` · `out-of-gas-consumes-all` | 되돌아가는/가스소진 **컨트랙트 자산**과 gasUsed==gasLimit 판정이 필요 | 컨트랙트 바이트코드 자산 + receipt `gasUsed`/`gasLimit` read (부분 표현 가능, 자산 확보 선행) |
| `authorized-account-gastip-free` | **거버넌스 쿼럼 흐름**(proposeAddAuthorizedAccount → 승인) 을 먼저 태워야 한다 | system-contracts 배치와 함께 — 거버넌스 스텝 표현 확보 후 |

### accounts 잔여 23건과 필요한 문법 확장

12건은 이관 완료(9 라이브 + secp256r1 3 오프라인). 나머지 23건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `insufficient-funds-rejected` · `dynamic-fee-below-basefee-rejected` · `gas-limit-exceeds-block-rejected` · `feepayer-insufficient-rejected` · `fee-delegated-unfunded-feepayer-rejected` | **제출 거부**(eth_sendTransaction/RawTransaction 이 에러 반환). DSL 은 제출 에러를 스텝 실패로 처리할 뿐 부정 기대가 없다 | (B) `expectReject`/`expectSubmitError` — gas-policy 갭과 동일 |
| `fd-sender-sig-invalid-rejected` · `fd-feepayer-sig-invalid-rejected` · `fee-delegated-sender-sig-invalid-rejected` · `fee-delegated-feepayer-sig-invalid-rejected` | 위 (B) + **손상된 이중서명 raw 트랜잭션 조립**(EncodeFeeDelegatedTampered) 을 spec 에서 만들 수단이 없다 | (B) + raw 서명 조립 자산 |
| `fee-delegated-transfer` · `external-fee-delegated-transfer` | **fee-delegation(0x16) 트랜잭션** — sendTx 에 feePayer(이중서명) 필드가 없다 | (D) sendTx feePayer + 0x16 인코딩 |
| `access-list-tx` | **access-list(0x01) 트랜잭션 타입** 미지원 | (C) sendTx accessList 필드 |
| `set-code-delegation` | **EIP-7702(0x04) set-code** — authorization 리스트/authority 서명 미지원 | (E) sendTx authorizationList + 신규 키 생성 |
| `nonce-ordering` · `replacement-tx` · `out-of-order-nonces-mine` · `same-nonce-replacement` | sendTx 가 **제출 후 receipt 동기 대기** → gap 난 nonce(N+1) 를 큐잉만 하고 나중에 채굴시킬 수 없고, "tx1 은 채굴되면 안 된다" 는 **부정 채굴 기대**도 없다 | 비동기 제출(대기 안 함) + 부정 채굴 assertion |
| `zero-address-transfer-blocked` · `precompile-transfer-blocked` | accounts SDK 의 **클라이언트측 정적 가드**(제출 전 거부)를 검사. DSL sendTx 는 노드로 직행하므로 이 가드를 태우지 못한다(의미가 다름) | SDK 가드 경로는 DSL 로 표현 대상 아님 — 갭으로 남김 |
| `external-value-transfer` | **operator 공급 funded 키**(CHAINBENCH_FUNDED_KEY)와 **런타임 신규 수취인 생성**이 필요. DSL 은 env 키 주입/키 생성 수단이 없다 | env 키 바인딩 + 키 생성 소스 |
| `fee-delegate-sign-rpc-present` | `eth_signRawFeeDelegateTransaction` 가 **method-not-found(-32601) 이 아님**을 확인(오류는 나도 됨). rpcCall assertion 은 RPC 오류를 스텝 실패로 처리 → "오류지만 not-found 는 아님" 을 표현 못함 | 메서드 존재 프로브(오류 코드 구분) |
| `eth-call-revert-returns-error` | eth_call 이 **revert 오류를 반환**해야 통과. `call` assertion 은 호출 오류를 스텝 실패로 처리할 뿐 부정 기대가 없다 | `call` 부정 기대(expectCallError) |
| `contract-event-emitted` | receipt `logs` 배열의 첫 로그 `topics[0]` 를 검사. `select` 는 **배열 인덱스(`logs.0.topics.0`)를 못 짚고**, 로그 토픽 존재 판정 연산자도 없다 | select 배열 인덱싱 + 로그 매칭 |

## 규약

- **파일 1개 = 케이스 1개**, 파일명 = `id` = 레거시 `Name`(kebab-case). CI 가 id 중복을 막는다.
- `applicableChains` 로 대상 체인을 선언한다 — 미적용 체인에서는 fail 이 아니라 **skip**.
- `requires` 로 필요한 capability 를 선언한다(`rpc`, `ws`, `consensus`, `process`).
- 레거시 케이스는 **소비자 이관 전까지 제거하지 않는다**. 지금은 두 경로가 병존한다.
