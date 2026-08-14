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
| `accounts` | 35 | **15** | ✅ 라이브 통과 (gstable). value/legacy/dynamic-fee transfer·tx-count·tx-by-hash·receipt·effective-gas·genesis-balance·contract-roundtrip 9건 + **제출거부 3건**(insufficient-funds·dynamic-fee-below-basefee·gas-limit-exceeds-block, `expect:"reject"`) 라이브, secp256r1 precompile 3건은 wbft 전용(gstable 미탑재 확인)이라 오프라인 검증만. 잔여 20건은 문법 갭(아래) |
| `gas-policy` | 17 | **12** | ✅ 라이브 통과 (gstable). read류 3 + tx-flow 6 (basefee min/max·effective-gas·gastip-forced·feecap exact/above-min) + **제출거부 3건**(feecap-below-min·legacy-gasprice-below-min·gaslimit-exceeded, `expect:"reject"`). `read/assert:"derive"`(sum/diff) 로 정확 산술 비교, `read:"derive"`(read source) 로 `feeCap = baseFee+tip` 를 계산해 sendTx 인자로 주입. 잔여 5건은 문법 갭(아래) |
| `hardfork` | 8 | **2** | ✅ 라이브 통과 (gstable). boho-chain-config-active(blockNumber/chainId/baseFee >0)·govminter-v2-code(codeAt≠"0x"). 잔여 6건 갭(아래) |
| `system-contracts` | 46 | **9** | ✅ 라이브 통과 (gstable). system-contracts-deployed(EVM 5종 codeAt)·adapter-code·token-metadata(WKRC 정확)·total-supply/balance readable·account authorization/blacklist readable·minter-status·validator-metadata. read source `call`+`$var` 보간으로 totalSupply≥balance 표현. 잔여 37건 갭(아래) |

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
hardfork   ( 2 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=2 fail=0
                     boho-chain-config-active(blockNumber/chainId/baseFee 모두 >0),
                     govminter-v2-code(getCode 0x..1003 = 38250 hex chars, ≠"0x") 라이브 확인.
                     hardfork p256 2건은 라이브 반증: gstable 의 0x100 이 valid/corrupt/short
                     세 벡터 모두 "0x" 반환 → 이 빌드에 P256VERIFY 미탑재. accounts 결과와 일치.
                     레거시 소스(hardfork_reads.go)는 Boho-genesis 로 P256 활성을 단언하나
                     실제 바이너리와 불일치 — 갭으로 남김(아래)
system-contracts (9 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=9 fail=0
                     EVM 거버넌스 5종(0x1000-0x1004) codeAt≠"0x", adapter code, token-metadata
                     (name/symbol == "WKRC" ABI 정확 일치), totalSupply/balanceOf readable,
                     totalSupply≥balanceOf(read source `call`→$bal, assert `call` GreaterOrEqual $bal),
                     account authorization/blacklist/minter readable, validatorList readable 모두 라이브.
                     system-contracts-deployed 는 처음 8종 전체 codeAt 로 라이브 fail →
                     0xB00001-3(bls-pop·native-coin-manager·account-manager)은 getCode="0x" 인
                     네이티브 precompile(eth_call 은 응답)로 판명, EVM 5종만 남기고 재범위. 갭 아래
```

### 라이브 검증 근거 — 제출거부 배치 (2026-08-13)

`expect:"reject"` 프리미티브(F1) 신규. sendTx 스텝이 노드의 **제출 시점 거부**(eth_sendTransaction 이
해시 없이 에러 반환)를 통과 조건으로 삼는다 — 채굴 후 status 0x0 인 `expect:"revert"`/`expectRevert`
와 구분된다. 부정 스텝(do)이 통과/실패로 판정을 내리고, 스키마상 필요한 어세션은 라이브 응답성
대조(blockNumber ≥ 0x1)로 채운다.

```
gas-policy (+3 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=3 fail=0
                     feecap-below-min(maxFeePerGas=1)·legacy-gasprice-below-min(gasPrice=1)·
                     gaslimit-exceeded(gas=blockGasLimit+1, derive sum) 모두 제출거부 확인.
accounts   (+3 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=3 fail=0
                     insufficient-funds(balance+1e18, balanceAt→derive sum→value 주입)·
                     dynamic-fee-below-basefee(maxFeePerGas=1)·gas-limit-exceeds-block(gas=gasLimit+1)
                     모두 제출거부 확인.
examples   (fee-boundary) → 수정 후 pass. 아래 결함 표 참조.
```

## 이관하면서 드러난 레거시 케이스의 결함

포팅은 케이스를 다시 읽게 만들고, 그 과정에서 원본의 오류가 드러났다.

| 레거시 케이스 | 문제 | 이관본에서의 처리 |
|---|---|---|
| `wbft-extra-info-fields` | `istanbul_getWbftExtraInfo` 를 **`"latest"` 태그로 호출** — 체인이 `block -2 not found` 로 거부한다. 구체적 블록 번호가 필요 | 헤드를 먼저 읽어(`read` + `$head`) 넘긴다 |
| `wbft-extra-info-fields` | `ChainCompat: [stablenet, wbft]` 인데 **`gasTip` 은 stablenet 전용** — wbft 응답에 그 필드가 없다 | 공통 필드(`committedSeal`·`preparedSeal`)만 양 체인 대상으로 남기고, `gasTip` 은 `stablenet-gastip-field.json` 으로 분리 |
| `p256-precompile-active` (hardfork) | 소스는 "stablenet 이 Boho 를 genesis 활성 → 0x100 P256VERIFY 가 valid 벡터에 `0x..01` 반환"을 단언하나, **실제 gstable 빌드는 0x100 에 대해 `"0x"` 반환** — precompile 미탑재. accounts secp256r1 3건과 동일 증상 | 라이브 반증으로 이관 보류. Boho/P256 탑재 바이너리 확보 여부를 체인팀에 확인 후 재분류 |
| `system-contracts-deployed` (system-contracts) | `eth_getCode ≠ "0x"` 를 **8개 시스템 주소 전체**에 걸었으나, `0xB00001-3`(bls-pop·native-coin-manager·account-manager)은 **네이티브 precompile** — `getCode` 는 `"0x"`(빈 코드)를 반환하고 `eth_call` 로만 응답한다. getCode 로 "배포" 여부를 판정하는 건 precompile 엔 부적절 | EVM 바이트코드 계약 5종(0x1000-0x1004)만 codeAt 로 판정. precompile 은 각자의 read 케이스(account-*-readable 등)로 활성 확인 |
| `tipcap-underpriced-rejected` (gas-policy) | 소스는 "유효 feeCap + tipCap=1 wei 는 MinTip 미만 → ErrUnderpriced 로 제출거부"를 단언하나, **실제 gstable 빌드는 이 tx 를 수락·채굴**(블록 0x18, status 0x1, maxPriorityFeePerGas=0x1). 이 빌드는 제출 시점에 MinTip 을 강제하지 않는다 — p256 3건과 동일한 라이브 반증 유형 | 이관 보류(삭제). MinTip 강제 빌드/설정 확보 여부를 체인팀에 확인 후 재분류. 현 빌드로는 표현해도 무의미 |
| `stablenet-fee-boundary` (examples, 문법예시) | ① below-min tx 에 `expectRevert:true` 를 썼으나 실제로는 **제출거부**(채굴 후 revert 아님) — feecap-below-min-rejected 로 라이브 확정. ② "accepted" tx 가 feeCap 을 1e12 로 하드코딩했는데 이 네트워크 최소치는 2e13 이라 **이것마저 거부**됨 | ① `expect:"reject"` 로 교정. ② baseFee 를 읽어 `derive sum [$base,$base]` 로 feeCap 을 산출해 주입 → 수정 후 라이브 pass |

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

### gas-policy 잔여 5건과 필요한 문법 확장

tx-flow 6 + 제출거부 3(feecap-below-min·legacy-gasprice-below-min·gaslimit-exceeded)건은 이관 완료.
`(B) expect:"reject"` 프리미티브(F1)로 제출거부 3건이 라이브 확정됐다. 나머지 5건은 아래 갭에 막혀
있다 — 가짜로 만들지 않고 남긴다. (`tipcap-underpriced-rejected` 는 라이브 반증으로 보류 — 결함 표 참조.)

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `accesslist-gasprice-below-min-rejected` | (B) 는 확보됐으나 **access-list(0x01) 트랜잭션 타입** 미지원 | (C) sendTx accessList 필드 |
| `revert-tx-status-zero` · `out-of-gas-consumes-all` | 되돌아가는/가스소진 **컨트랙트 자산**과 gasUsed==gasLimit 판정이 필요 | 컨트랙트 바이트코드 자산 + receipt `gasUsed`/`gasLimit` read (부분 표현 가능, 자산 확보 선행) |
| `authorized-account-gastip-free` | **거버넌스 쿼럼 흐름**(proposeAddAuthorizedAccount → 승인) 을 먼저 태워야 한다 | system-contracts 배치와 함께 — 거버넌스 스텝 표현 확보 후 |

### accounts 잔여 20건과 필요한 문법 확장

15건은 이관 완료(9 라이브 + 제출거부 3 라이브 + secp256r1 3 오프라인). 나머지 20건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `feepayer-insufficient-rejected` · `fee-delegated-unfunded-feepayer-rejected` | (B) `expect:"reject"` 는 확보됐으나 **fee-delegation(0x16)** tx(feePayer 이중서명) 미지원 — 제출거부 자체는 표현 가능하나 0x16 를 조립할 수단이 없다 | (D) sendTx feePayer + 0x16 인코딩 |
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

### hardfork 잔여 6건과 필요한 문법 확장

2건은 이관 완료(라이브). 나머지 6건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `p256-precompile-active` · `p256-rejects-invalid` | **라이브 반증**: gstable 빌드의 0x100 이 valid/corrupt/short 세 벡터 모두 `"0x"` 반환 → P256VERIFY 미탑재. `-active` 는 라이브 fail, `-rejects-invalid` 는 "precompile 부재"로 우연히 통과할 뿐 검증 의미 없음. 레거시 소스는 Boho-genesis P256 활성을 단언하나 바이너리와 불일치 | Boho/P256 탑재 gstable 바이너리 확보 후 재분류(체인팀 확인 필요). 현 빌드로는 표현해도 무의미 |
| `govminter-code-changes-at-boho` · `p256-inactive-before-boho` · `anzeon-active-before-boho` · `prealloc-preserved-across-boho` | **delayed-boho 교차포크**: bohoBlock=N 으로 지연 활성한 뒤 fork 전(블록 1)·후(latest) 상태를 조건부 대기(WaitFor 크로스오버)로 비교. 1회성 read spec 은 포크 크로스오버를 표현할 수 없고, DSL 에 delayed-boho 조건부 대기가 없다 | delayed-boho 기동 + 크로스포크 조건부 WaitFor + 블록 고정(`0x1` vs latest) 비교 |

### system-contracts 잔여 37건과 필요한 문법 확장

9건은 이관 완료(모두 라이브). 나머지 37건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.
압도적 다수(≈30건)가 **거버넌스 쿼럼 흐름**(propose → 검증자 N명 approve → 정족수 자동 execute)에
의존하는데, DSL 에는 이 다단계 서명 흐름을 태울 스텝 프리미티브가 없다. 이것이 이 배치의 핵심 갭이다.

| 레거시 케이스 (건수) | 갭 | 필요 확장 |
|---|---|---|
| `burn-proposal-executes`·`mint-proposal-executes`·`configure-minter-proposal-executes`·`gastip-governance-updates-header`·`blacklist-proposal-executes`·`authorize-proposal-executes`·`validator-add-member-executes`·`masterminter-member-add-remove`·`remove-minter-executes`·`unblacklist-restores`·`burn-cancel-refundable`·`burn-execute-no-refundable`·`burn-reject-refundable`·`claim-burn-refund-succeeds`·`burn-refund-events`·`authorized-account-added-event`·`unauthorize-proposal-executes`·`address-unblacklisted-event`·`mint-transfer-event`·`burn-transfer-event` (20) | **거버넌스 쿼럼 흐름** — propose(payable 포함)→검증자 N명 approve→정족수 execute→상태/이벤트 확인. DSL sendTx 는 단일 노드서명 tx 만 태우고, propose/approve/execute·정족수·검증자 셋 순회를 표현할 수 없다. 다수는 추가로 이벤트 로그(Transfer/AddressBlacklisted 등) 확인 필요 | (G) 거버넌스 스텝 프리미티브(propose/approve/execute + quorum) + 이벤트 로그 매칭 |
| `quorum-deficient-stays-voting`·`claim-zero-refund-reverts`·`claim-burn-refund-double-reverts`·`direct-blacklist-call-rejected`·`non-member-configure-minter-rejected` (5) | 위 (G) + **부정 기대**(정족수 미달 execute revert / 이중 claim revert / 비회원 호출 reject-or-revert). DSL 은 제출에러를 스텝 실패로 처리할 뿐 부정 기대가 없다 | (G) + (B) `expectReject`/채굴후 status 0x0 기대 (accounts·gas-policy 와 동일) |
| `sender-blacklisted-rejected`·`recipient-blacklisted-rejected` (2) | 거버넌스 blacklist 후 **제출 거부**(에러 메시지 "blacklist"). 거버넌스 + 제출거부 부정기대 | (G) + (B) |
| `feepayer-blacklisted-rejected` (1) | 위 + **fee-delegation(0x16)** tx (feePayer 이중서명) 미지원 | (G) + (B) + (D) sendTx feePayer |
| `authorized-tx-executed-event` (1) | **런타임 키 생성 + faucet 펀딩 + 거버넌스 authorize** 후 tx 의 AuthorizedTxExecuted 로그 확인. 키 생성·env 주입·거버넌스·로그매칭 모두 부재 | (G) + 키 생성 소스 + 로그 매칭 |
| `authorized-extra-bit-synced`·`blacklisted-extra-bit-synced`·`dual-status-extra`·`extra-balance-preserved` (4) | 순수 read(isAuthorized/isBlacklisted/getBalance)라 **표현 자체는 가능**하나 fixture 계정이 **`account-extra` 제네시스 오버레이**에서만 존재/설정됨. 표준 stablenet 엔 없어 라이브 미검증 | `account-extra` cap 오버레이 기동으로 라이브 검증 후 이관(현재는 표현 가능·미기동) |
| `proposal-expiry-transitions` (1) | **`short-expiry` 오버레이 + 만료 시간 대기**(35s) 후 expireProposal. 조건부/시간 대기 표현 없음 | `short-expiry` cap + 시간/조건 대기 |
| `token-approve-sets-allowance`·`token-transfer-emits-event`·`token-transfer-from-moves-balance` (3) | 토큰 write(approve/transfer/transferFrom)의 **이벤트 로그 topic 인덱싱**(`Approval`/`Transfer` 의 topics[2]==spender/recipient) 확인. `select` 는 배열/topic 인덱스를 못 짚는다. transferFrom 은 owner≠spender **2계정** 필요 | select 배열/topic 인덱싱 + 로그 매칭 (+ transferFrom 은 다중 서명자) |

## 규약

- **파일 1개 = 케이스 1개**, 파일명 = `id` = 레거시 `Name`(kebab-case). CI 가 id 중복을 막는다.
- `applicableChains` 로 대상 체인을 선언한다 — 미적용 체인에서는 fail 이 아니라 **skip**.
- `requires` 로 필요한 capability 를 선언한다(`rpc`, `ws`, `consensus`, `process`).
- 레거시 케이스는 **소비자 이관 전까지 제거하지 않는다**. 지금은 두 경로가 병존한다.
