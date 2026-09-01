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

레거시 등록 케이스 **134개** 기준(`testkit.Cases()` 실측).

> **2026-08-28 실측 (P8):** 134건 중 **96건이 같은 id 의 DSL 스펙**으로 이관됐고(`tests/specs` 92 +
> `examples/specs` 4), 이관이 끝난 **Go 케이스 파일 41개와 그 유닛테스트를 삭제**했다
> (`tests/api`·`tests/network` 는 패키지째 사라졌다). 남은 등록은 **56건**: 미이관 34건 + 표 구동
> 파일(`tx_rejections`·`gas_boundary`·`account_extra`·`gov_*_events` 등)에 이관분과 섞여 있는 22건.
> 남은 파일이 공유하던 헬퍼는 `tests/anzeon/helpers.go`·`tests/wbft/accounts/helpers.go` 로 모았다.
> 미이관 34건의 사유는 아래 "이관하지 않은 것과 그 이유" 와 카테고리별 잔여 표에 있다.
> 그것들이 이관되면 `testkit`·`testrun`·`chainbench test`·MCP `chainbench_test` 가 함께 은퇴한다.

| 카테고리 | 레거시 | 이관 | 상태 |
|---|---:|---:|---|
| `api` | 11 | **10** | ✅ 라이브 통과 (gstable) |
| `consensus` | 13 | **11** | ✅ 라이브 통과 (gstable·gwbft). +randao/mixdigest·block-period(parentHash 워크로 헤드 2쌍 `derive:diff`==1)·wbft-seals-quorum(seal 서명 present). 잔여 3건 갭(아래) |
| `network` | 4 | **4** | ✅ `examples/specs/network-*.json`(선행 이관분) 3건 + **admin-peers-populated** 라이브(gstable) — `rpcCall admin_peers` 를 `select:"#"`(배열 길이)≥1 & `select:"0.id"`(배열 인덱싱) NotEqual "" 로 검증. 갭 없음 |
| `accounts` | 35 | **30** | ✅ 라이브 통과 (gstable). value/legacy/dynamic-fee transfer·tx-count·tx-by-hash·receipt·effective-gas·genesis-balance·contract-roundtrip 9건 + **제출거부 3건**(insufficient-funds·dynamic-fee-below-basefee·gas-limit-exceeds-block, `expect:"reject"`) + **contract-event-emitted**(컨트랙트 생성 sendTx→receipt contractAddress→execute→`logs` topic0 매칭) + **access-list-tx**(EIP-2930 0x01: 빈 `accessList:[]`+gasPrice → `eth_getTransactionByHash` type==0x1) 라이브, secp256r1 precompile 3건은 wbft 전용(gstable 미탑재 확인)이라 오프라인 검증만. 잔여 18건은 문법 갭(아래) |
| `gas-policy` | 17 | **16** | ✅ 라이브 통과 (gstable). read류 3 + tx-flow 6 (basefee min/max·effective-gas·gastip-forced·feecap exact/above-min) + **제출거부 4건**(feecap-below-min·legacy-gasprice-below-min·gaslimit-exceeded·accesslist-gasprice-below-min, `expect:"reject"`). `read/assert:"derive"`(sum/diff) 로 정확 산술 비교, `read:"derive"`(read source) 로 `feeCap = baseFee+tip` 를 계산해 sendTx 인자로 주입. 잔여 4건은 문법 갭(아래) |
| `hardfork` | 8 | **4** | ✅ 라이브 통과 (gstable). boho-chain-config-active(blockNumber/chainId/baseFee >0)·govminter-v2-code(codeAt≠"0x"). 잔여 6건 갭(아래) |
| `system-contracts` | 46 | **45** | ✅ 라이브 통과 (gstable). system-contracts-deployed(EVM 5종 codeAt)·adapter-code·token-metadata(WKRC 정확)·total-supply/balance readable·account authorization/blacklist readable·minter-status·validator-metadata 9건 + **토큰 write+event 2건**(token-transfer-emits-event·token-approve-sets-allowance, sendTx ABI calldata→`logs` topic 필터/select + allowance `call`) + **거버넌스 쿼럼 12건**(mint·blacklist·authorize·configure-minter·authorized-account-added-event·**mint-transfer-event** 단일 라운드 + unauthorize·address-unblacklisted·**remove-minter-executes** 2-라운드 + **burn-proposal-executes**(payable proposeBurn→approve→`(H) derive op:"word"` 로 proposals() 상태 워드[9]==Executed(3) 디코드) + **quorum-deficient-stays-voting**(proposeMint→정족수 미달 executeProposal 을 `expect:"revert"` 로 확정→proposals() 상태 워드[9]==Voting(1)→cleanup approve) + **recipient-blacklisted-rejected**(GovCouncil blacklist→ 수취인에게 전송 시 `expect:"reject" reason:"blacklist"` 제출거부→cleanup unblacklist): `receiptLog` topic1 로 proposalId 추출→`derive abiCall` 로 approve/execute calldata 조립→정족수 자동 execute→`call`/상태워드/`logs` 확인). `unblacklist-restores` 는 address-unblacklisted-event 가 이미 전량 커버(중복). read source `call`+`$var` 보간으로 totalSupply≥balance 표현. **비회원 로컬서명 3건**(direct-blacklist·non-member-configure-minter·sender-blacklisted rejected — `newAccount`+`sendTx key` 로컬서명) + **burn-refund 라이프사이클 8건**(burn-transfer-event·burn-cancel-refundable·burn-execute-no-refundable·burn-reject-refundable·claim-burn-refund-succeeds·burn-refund-events·claim-zero-refund-reverts·claim-burn-refund-double-reverts — **Boho-v2 넷**(genesis 오버레이)에서 proposeBurn→cancel/disapprove/claim + refundableBalance **전후 델타**(`derive diff`) 어세션으로 재사용 넷 누적 간섭 회피). 잔여 11건 갭(아래) |

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

### 라이브 검증 근거 — 이벤트 로그 배치 (2026-08-14)

`logs` 어세션이 이미 topic 필터(와일드카드 위치 포함)·`select:topicN`·`index` 를 지원한다는 사실을
재확인(아래 "재분류" 참조). eth_getLogs 로 이벤트를 검증하는 케이스가 신규 문법 없이 표현 가능했다.
추가로 sendTx 가 **컨트랙트 생성**(`to` 생략)을 지원함을 라이브로 확인했다.

```
system-contracts (+2 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=2 fail=0
                     token-transfer-emits-event: sendTx(0x1000, transfer(recip,1) calldata)→
                     logs(address=0x1000, topics=[Transfer,·,recipTopic], select count≥1 & data==1).
                     token-approve-sets-allowance: sendTx(approve(spender,100))→
                     logs(Approval topic 매칭) + allowance(owner,spender) `call`==100.
accounts   (+1 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=1 fail=0
                     contract-event-emitted: sendTx(data=LOG1 컨트랙트 initcode, to 생략=생성)→
                     receipt.contractAddress 읽어 $addr→sendTx(to=$addr) execute→
                     logs(address=$addr, select topic0==0x1111..1111). sendTx 컨트랙트 생성 라이브 확인.
```

### 라이브 검증 근거 — 배열 인덱싱 배치 (2026-08-14)

`rpcCall` 의 `select` dot-path 에 **배열 인덱싱**(숫자 세그먼트 `peers.0.id`)과 **길이**(`#` 세그먼트,
십진 반환)를 추가(`internal/testspec/derived.go` `dotPath`). 배열을 반환하는 RPC 결과에서 "최소 N개"와
"N번째 엔트리의 필드"를 표현할 수 있게 됐다.

```
network    (+1 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=1 fail=0
                     admin-peers-populated: admin_peers 실제 3피어 반환(RPC 직접 확인),
                     select:"#"≥1 & select:"0.id" NotEqual "" 모두 통과. 단일노드면 피어 부재라
                     레거시처럼 vacuous 이나 다노드 스크래치넷에서 실질 검증.
```

### 라이브 검증 근거 — access-list(0x01) 배치 (2026-08-14)

`(C)` sendTx 에 **`accessList` 필드**를 추가(`internal/core/rpc/client.go` `SendTxArgs.AccessList any` +
`internal/testspec/builtins.go` 패스스루). 빈 리스트 `[]` 도 EIP-2930 type 0x01 을 선택하므로 —
`[]AccessTuple` + `omitempty` 였다면 빈 케이스가 조용히 legacy 로 강등된다 — `any` 로 선언해 verbatim 전달한다.

```
accounts   (+1 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=1 fail=0
                     access-list-tx: baseFee 읽어 gasPrice=2×base 주입, accessList:[] + gasPrice →
                     eth_getTransactionByHash type==0x1 라이브 확인(빈 리스트가 typed 봉투를 선택).
gas-policy (+1 spec) → 같은 스크래치넷: pass=1 fail=0
                     accesslist-gasprice-below-min-rejected: accessList:[] + gasPrice=1(<min) → 제출거부.
```

주의: 이 배치의 첫 라이브 실행은 type 0x0(legacy)로 실패했는데, 원인은 코드가 아니라 **attach-run 이 쓰는
`/tmp/cb` 바이너리가 accessList 패스스루 커밋 이전 것(stale)** 이었다. 유닛 테스트(직접 Args 주입)는 통과했으나
DSL 라이브 경로는 옛 바이너리를 실행했다. 재빌드 후 양쪽 pass — 인터폴레이션/`omitempty` 는 무관했다.

### 라이브 검증 근거 — 거버넌스 쿼럼(G) 배치 (2026-08-14)

`(G)` 거버넌스 다단계 흐름을 **신규 스텝 프리미티브 없이** 기존 sendTx + read source 두 개의 조합으로
표현했다. 신규 프리미티브 2종:
- **`receiptLog` read source**(`internal/testspec/builtins.go`) — 트랜잭션 receipt 의 로그에서 topic/data 를
  추출해 바인딩에 저장한다. `hash`(필수) + `address`/`topic0` 필터(hex 대소문자 무시) + `index`(기본 0) +
  `topic`(기본 1) 또는 `select:"data"`. 런타임에만 알 수 있는 `proposalId`(ProposalCreated 의 indexed topic1)를
  꺼내는 용도.
- **`derive op:"abiCall"`**(`internal/testspec/derived.go`) — `selector`(4바이트) + `of` 인자들을 32바이트
  좌패딩 워드로 이어붙여 calldata 를 조립한다. uint256·address 동일 인코딩. 추출한 `$pid` 를 approve/execute
  calldata 에 끼워넣는 용도.

이 조합으로 propose→proposalId 추출→approve calldata 조립→정족수 자동 execute 를 한 spec 안에서 표현한다.

```
system-contracts (+1 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=1 fail=0
                     mint-proposal-executes: balanceAt(수취인) $b0 → derive sum($b0,1e18)=$want →
                     sendTx(en1, proposeMint(proof)) $propHash →
                     receiptLog($propHash, topic0=ProposalCreated, topic1)=$pid →
                     derive abiCall(0x98951b56 approveProposal, [$pid])=$approveData →
                     sendTx(en2, $approveData) $apHash → txStatus 둘 다 0x1 & balanceAt==$want.
```

라이브에서 확정한 거버넌스 사실(스크래치넷 8601-8605):
1. **proposalId 결정적** — fresh net 첫 제안=1, 같은 넷에서 제안마다 증가. 배치 실행이 넷을 공유하면(엔진
   Fingerprint 재사용) id 가 spec 간 충돌 → 정적 id calldata 는 불안전. 그래서 `receiptLog` 로 **런타임 추출**한다.
2. **정족수=2** — 이 4검증자 넷에서 propose + approve 1건이면 정족수 도달, **자동 execute**(별도 execute tx 불필요).
3. **고정 timestamp(1700000001) 수용** — proof 에 freshness 검사 없음.
4. **GovMinter 는 예치 증명 유일성 강제** — 이미 민팅된 (depositID+bankReference+timestamp) 재제안은 revert(0x0).
   따라서 민팅 거버넌스 spec 은 **fresh net / 미사용 예치**를 가정한다(proposalId=1 fresh-net 가정과 동류).
   CI 는 fresh genesis 로 돌므로 성립. 이 spec 은 전용 예치(DSL-SPEC-MINT-A / DSL-SPEC-BANK-A / ts=1700000001)를
   예약해 다른 케이스와 충돌하지 않는다.

**attach 모드 노드 셀렉터 주의**: attach 모드에선 모든 노드가 RoleEndpoint 로 잡히므로(chainbench 가 실제
consensus role 을 알 수 없음) sendTx `on` 은 실제 역할과 무관하게 `en1`/`en2`/... 로 지정한다.

### 라이브 검증 근거 — 거버넌스 쿼럼(G) 확장 배치 (2026-08-14)

`(G)` 프리미티브(`receiptLog` + `derive abiCall`)만으로 GovCouncil(0x1004)·GovMasterMinter(0x1002) 흐름
6건을 추가 이관·라이브 확정했다. propose calldata 는 정적(selector + 고정 인자)이라 하드코딩하고, 런타임
proposalId 만 `receiptLog` 로 추출해 approve calldata 를 `derive abiCall` 로 조립한다. 2-라운드 케이스는
propose→approve 사이클을 두 번(add→remove) 이어 붙인다.

```
system-contracts (+6 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=6 fail=0
  단일 라운드:
    blacklist-proposal-executes:  proposeAddBlacklist(C0FFEE06)→approve→
                                  AccountManager.isBlacklisted==1 & GovCouncil AddressBlacklisted 로그.
    authorize-proposal-executes:  proposeAddAuthorizedAccount(C0FFEE0A)→approve→isAuthorized==1.
    configure-minter-proposal-executes: proposeConfigureMinter(C0FFEE07,10코인)→approve→
                                  NativeCoinAdapter.minterAllowance==10e18 (`call` Equal 전체 워드).
    authorized-account-added-event: proposeAddAuthorizedAccount(C0FFEE11)→approve→isAuthorized==1 &
                                  AuthorizedAccountAdded 로그.
  2-라운드(add→remove):
    unauthorize-proposal-executes: authorize(C0FFEE12)→approve→removeAuthorizedAccount→approve→
                                  isAuthorized==0 & AuthorizedAccountRemoved 로그.
    address-unblacklisted-event:   blacklist(C0FFEE13)→approve→removeBlacklist→approve→
                                  isBlacklisted==0 & AddressUnblacklisted 로그.
```

계약별 selector/이벤트 topic0 은 `accounts.EncodeCallArgs`/`EventTopic` 으로 오프라인 산출해 spec 에 고정.
상태 확인은 AccountManager(0x1003→isAuthorized/isBlacklisted는 0xB00003)·NativeCoinAdapter(0x1000) 의 read
getter 를 `call` 어세션으로 조회한다. 대상 주소는 케이스마다 전용 fixture(C0FFEE06/0A/07/11/12/13)를 예약해
서로 충돌하지 않는다(fresh-net 가정 유지).

### 거버넌스 쿼럼(G) — `(H) derive op:"word"` 로 상태 워드 디코드

side-effect(`call` getter/이벤트)가 없고 **실행 결과를 proposals() 상태 워드로만** 확인하는 케이스를 위해
`(H) derive op:"word"` 를 추가했다: 0x-hex blob 에서 N번째 32바이트 워드를 뽑는 `abiCall` 의 역연산이다.
`proposals(id)` 는 고정 레이아웃 10-필드 tuple 을 반환하고 status 는 워드[9](uint8)라, 전체 blob 을 비교하는
`call` 로는 timestamp/제안자 등 휘발 필드 때문에 status 만 짚을 수 없다. `derive word index:9` 로 그 워드를
추출해 `Executed(0x…03)` 와 비교한다. `derive` 는 read/assert 양쪽에 등록돼 있어 어세션으로 바로 쓴다.

```
system-contracts (+1 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=1 fail=0
  burn-proposal-executes: payable proposeBurn(구조화 BurnProof, msg.value==amount, 제안자=burn 대상)
                          →receiptLog proposalId 추출→derive abiCall approve→en2 승인(정족수 자동 execute)
                          →derive abiCall 로 proposals(id) calldata 조립→call 로 반환 blob 읽기
                          →assert derive word index:9 == Executed(3).
```

- GovMinter 는 **0x1003**(NativeCoinAdapter 0x1000 아님) — proposeBurn/approve/proposals() 모두 0x1003 대상.
- **가스캡 주의**: 이 스크래치 넷의 블록 가스한도가 idle 감쇠로 ≈921k 까지 내려가, 레거시 `govGas`(1.5M)는
  `exceeds block gas limit` 로 거부된다. proposeBurn 실측 ≈455k 라 캡을 **0xb71b0(750k)** 로 낮춰 통과.
  (fresh-net 에선 한도가 높아 1.5M 도 무방하나, 이식성을 위해 실측+여유 캡을 쓴다.)

### 라이브 검증 근거 — remove-minter-executes (2026-08-14)

`(G)`+`(H)` 조합으로 GovMasterMinter(0x1002) 2-라운드 흐름을 이관·라이브 확정했다. **흐름 중간 상태**를
`(H) derive op:"word"` 로 짚는 패턴이 신규다: configure→approve 로 minter 를 먼저 세팅하고, 그 시점의
`isMinter` 반환 blob 을 저장(`$cfgIsMinter`)해 두었다가 어세션에서 `derive word index:0 == 1`(설정됨)로
확인한 뒤, remove→approve 후 `call isMinter == 0`(제거됨)으로 라운드 결과를 대조한다.

```
system-contracts (+1 spec) → gstable 5노드(스크래치, 8601-8605, attach): pass=1 fail=0
  remove-minter-executes: proposeConfigureMinter(C0FFEE20, allowance 10코인)→en2 approve →
                          call isMinter(C0FFEE20) 저장($cfgIsMinter) →
                          proposeRemoveMinter(C0FFEE20)→en2 approve →
                          assert txStatus 둘 다 0x1 & derive word($cfgIsMinter)[0]==1 & call isMinter==0.
```

- selector: proposeConfigureMinter(address,uint256)=0x898420a9, proposeRemoveMinter(address)=0x93364117,
  isMinter(address)=0xaa271e1a(대상 0x1000 NativeCoinAdapter), approveProposal=0x98951b56.
- **가스캡**: 실행 시점 블록 한도가 ≈529k 로 더 감쇠해 있어 configure 실측 ≈284k 기준 캡을 **0x7a120(500k)** 로 낮춰 통과.

> **스크래치넷 가스 감쇠 경고**: 이 8601-8605 넷은 idle 블록마다 가스한도가 ≈1/1024 씩 내려간다
> (921k→529k→469k→399k, 2026-08-14 관측). burn(750k)·remove-minter(500k) 캡은 검증 당시엔 한도 아래였으나
> **감쇠 넷(≈399k)에선 재실행 불가**였다. mint/burn **실행** tx(≈455k)는 감쇠 넷에 안 들어간다.
> → **해결**: 동일 genesis(gasLimit 20M)로 5노드를 재-init·재기동해 한도를 20M 로 리셋(genesis 에 거버넌스
> 상태가 baked-in 이라 재기동만으로 council/quorum 복원). fresh 넷 window(~1h) 안에 실행 케이스를 검증한다.
> 이관된 spec 의 캡 값 자체는 정상(healthy/fresh 넷 기준 적정)이며, 감쇠는 넷 상태의 문제다.

### 라이브 검증 근거 — mint-transfer-event (2026-08-14, fresh 넷)

감쇠 넷을 20M 로 재기동한 뒤 mint **실행** 케이스를 라이브 확정했다. mint-proposal-executes 와 같은
proposeMint→approve 자동 execute 패턴에 **Transfer 이벤트 로그 확인**을 더한 것이다. mint 은 native-coin
발행이라 NativeCoinAdapter(0x1000)가 `Transfer(0x0 → 수취인)` 을 emit 한다.

```
system-contracts (+1 spec) → gstable 5노드(재기동 fresh, 8601-8605): pass=1 fail=0
  mint-transfer-event: balanceAt(C0FFEE30) $b0 → derive sum($b0,1e18)=$want →
                       proposeMint(C0FFEE30, 1e18, deposit DSL-MINT-EVT) $propHash →
                       receiptLog proposalId $pid → derive abiCall approve → en2 approve(자동 execute) →
                       rpcCall receipt.blockNumber $apBlk →
                       assert txStatus 둘 다 0x1 & balanceAt(C0FFEE30)==$want &
                       logs(0x1000, [Transfer, from=0x0, to=C0FFEE30], fromBlock=toBlock=$apBlk) count≥1.
```

- **5-엔드포인트 필수**: approve 는 `en2`(node2, 0x2493) 로 서명하므로 `--rpc` 를 5개(8601-8605) 모두 넘겨야
  en1/en2 가 해소된다. 1개만 넘기면 `testspec: no target node RPC URL` 로 approve 스텝이 실패한다.
- **deposit 유일성은 propose 시점 강제**: GovMinter 는 (depositID+bank+ts) 재제안을 **propose 단계에서** revert
  (status 0x0)한다 — 이미 실행된 게 아니라 **미실행 대기(Voting) 제안이 있어도** 중복을 막는다. 따라서 부분
  실패로 dangling 제안이 남으면 같은 deposit 재실행 불가 → fresh 넷에서 단발로 통과시킨다(DSL-MINT-EVT 전용 예약).

### 라이브 검증 근거 — 부정기대 2건 (2026-08-14, fresh 넷)

`expect:"revert"`(채굴후 status 0x0)·`expect:"reject"`(제출거부) 프리미티브(F1/F11, 이미 구현·커밋)를
거버넌스 부정 케이스에 실증했다. 둘 다 노드(=거버넌스 멤버) 서명만으로 표현 가능해 **비회원 로컬서명 갭과 무관**하다.

```
system-contracts (+2 spec) → gstable 5노드(fresh, 8601-8605): 각 pass=1 fail=0
  quorum-deficient-stays-voting: proposeMint(C0FFEE31, deposit DSL-QD-*) $propHash →
                       receiptLog $pid → derive abiCall executeProposal(0x0d61b519) [$pid] →
                       en1 executeProposal `expect:"revert"`(정족수 미달 → status 0x0) →
                       call proposals($pid) $proposalsRet(cleanup 전 read) → en2 approve cleanup →
                       assert txStatus $propHash==0x1 & derive word[9]($proposalsRet)==Voting(1) & 
                       txStatus $apHash==0x1.
  recipient-blacklisted-rejected: en1 proposeAddBlacklist(C0FFEE32,0x0d321273)→en2 approve(자동 execute) →
                       call isBlacklisted(C0FFEE32,0xfe575a87 on 0xB00003) $isBl →
                       en1 send→C0FFEE32 `expect:"reject" reason:"blacklist"`(제출거부) →
                       en1 proposeRemoveBlacklist(0x3d4c0452)→en2 approve cleanup →
                       assert txStatus add-approve==0x1 & derive word[0]($isBl)==1 & rm-approve==0x1.
```

- **부정 스텝이 fail-fast 로 검증을 담보**: `expect:"revert"`/`expect:"reject"` 스텝은 기대가 어긋나면
  (revert 인데 성공, reject 인데 수락) 스텝이 실패→스펙 전체 fail. 따라서 pass 는 "정족수 미달 execute 가
  실제 revert" · "blacklist 수취인 전송이 실제 제출거부"를 원자적으로 확정한다.
- **cleanup 은 assertion 이전(step) 에 실행**: v1 은 steps→assertions 순서라, 상태워드/isBlacklisted 는
  cleanup **직전** step 에서 read 해 저장(`$proposalsRet`/`$isBl`)한 뒤 assertion 이 그 값을 비교한다.
  cleanup(approve/unblacklist)은 dangling 제안·잔존 blacklist 를 제거해 재사용 넷 오염을 막는다.

### 라이브 검증 근거 — (E) 비회원 로컬서명 3건 (2026-08-14, fresh 넷)

`newAccount`(런타임 키 생성) + `sendTx key`(로컬 서명 → `eth_sendRawTransaction`) 프리미티브를 추가해, 노드
키스토어(=거버넌스 멤버)로는 표현 불가능했던 **비회원 발신자** 부정 케이스 3건을 실증했다. 키는 세션 한정
휘발성이며 엔진에 하드코딩하지 않는다 — 매 실행 생성 후 node1(멤버)이 node-서명으로 펀딩한다.

```
system-contracts (+3 spec) → gstable 5노드(fresh, 8601-8605): 각 pass=1 fail=0
  direct-blacklist-call-rejected: newAccount $acct/$acctKey → node1 펀딩(10 ETH) →
                       sendTx key:$acctKey to AccountManager(0xB00003) blacklist(addr) `expect:"reject"`.
  non-member-configure-minter-rejected: newAccount → node1 펀딩 →
                       sendTx key to GovMasterMinter(0x1002) proposeConfigureMinter(addr,uint256) `expect:"reject"`.
  sender-blacklisted-rejected: newAccount → node1 펀딩 → derive abiCall proposeAddBlacklist($acct) →
                       en1 propose → receiptLog $pid → en2 approve(자동 execute, $acct blacklist) →
                       sendTx key:$acctKey 전송 `expect:"reject" reason:"blacklist"` →
                       assert txStatus fund/approve 둘 다 0x1.
```

- **비회원 서명 경로**: `sendTx` 에 `key` 인자가 있으면 노드 `eth_sendTransaction` 대신 주입된
  `AccountProvider.OpenWallet` 로 로컬 서명 후 raw 제출한다(`data` 있으면 Execute, value-only 는 SendCoin).
  member-only 가드·blacklist-발신자는 이 경로로만 태울 수 있다(모든 노드 코인베이스가 거버넌스 멤버이므로).
  `newAccount` 는 주소를 `save`, 개인키 hex 를 `saveKey` 로 바인딩하며 `Unresolved` 가 둘 다 인식한다.
- **펀딩 순서**: sender-blacklisted 는 blacklist **이전**에 펀딩해야 한다(수취인 blacklist 후엔 전송이 거부됨).
  펀딩으로 거부 사유가 "insufficient funds" 가 아닌 가드/blacklist revert 임을 보장한다.
- **엔진 배선**: `AccountProvider` 를 `internal/engine/{app,attach}.go` 의 `testspec.Deps.Accounts` 로 주입한다
  (기존엔 미배선이라 `key` 사용 시 "no account provider" 로 실패했다).

### 라이브 검증 근거 — burn-refund 라이프사이클 8건 (2026-08-14, Boho-v2 넷)

refund 함수(`claimBurnRefund` 0x936834b9, `refundableBalance` 0xb03d36cd)는 **v2 GovMinter(Boho 하드포크
업그레이드 계약)에만** 존재한다. chainbench 의 stablenet genesis 템플릿엔 `boho` 섹션이 없어 어떤 넷에서도
기본 활성되지 않으므로, **`--genesis-overlay`**(`{"genesis":{"config":{"bohoBlock":2,"boho":{"systemContracts":
{"govMinter":{"address":"0x…1003","version":"v2"}}}}}}`)로 별도 넷(8611-8615)을 기동해 검증했다. 8601-8605
운영 넷은 건드리지 않았다.

```
system-contracts (+8 spec) → gstable 5노드(Boho-v2 fresh, 8611-8615): 각 pass=1 fail=0
  burn-cancel-refundable: refBefore(refundableBalance[node2]) → node2 proposeBurn(en2,value 1e18) →
                       receiptLog $pid → derive abiCall cancelProposal(0xe0a8f6f5) → node2 cancel →
                       refAfter → assert derive diff(refAfter,refBefore)==1e18.
  burn-execute-no-refundable: proposeBurn(node1,en1)→approve(en2 자동 execute)→proposals()[9]==Executed(3)
                       & refundableBalance 델타==0 (실행은 refund 미발생).
  burn-reject-refundable: proposeBurn(node3,en3)→비제안자 node1/2/4 disapprove(0xc8541fe0, en1/2/4) →
                       proposals()[9]==Rejected(7) & refundableBalance[node3] 델타==1e18
                       (maxRejections=memberCount-quorum=4-2=2, disapproval>2 즉 3건에서 Rejected).
  burn-transfer-event: proposeBurn(node1,en1) → logs(0x1000,[Transfer,from=node1,to=0x1003]) count≥1.
  claim-burn-refund-succeeds: propose→cancel(refMid 델타==1e18)→claim(0x936834b9)→refAfter 절대 0
                       (claim 은 msg.sender 전액을 0 으로).
  burn-refund-events: propose→cancel→claim; cancel 블록 logs(0x1003, BurnDepositRefunded) count≥1 &
                       claim 블록 logs(0x1003, BurnRefundClaimed) count≥1.
  claim-zero-refund-reverts: newAccount(refundable 0)→node1 펀딩→로컬서명 claim `expect:"revert"`(gas 명시).
  claim-burn-refund-double-reverts: node2 propose→cancel→claim1 성공(전액 0화)→claim2 `expect:"revert"`.
```

- **누적 간섭 회피 = 전후 델타**: refundableBalance 는 재사용 넷에서 제안자별로 **누적**된다. 절대값
  어세션(`>0`/`==0`)은 실행 순서에 취약하므로, cancel/reject 는 `derive diff(refAfter,refBefore)==1e18`
  로 **증분**만 검사한다. claim 후 잔액은 claim 이 전액을 0 으로 만드니 **절대 0** 이 순서와 무관하게 성립한다.
- **제안자↔RPC 짝**: 멤버 키는 각자 노드에만 unlock 돼 있어 `from`=멤버는 **자기 노드 RPC**(en1..en4=node1..4)
  로 보내야 한다. proposeBurn 은 payable(msg.value==amount) + from==msg.sender + onlyActiveMember.
- **withdrawalId 유일성**: proposeBurn 은 (withdrawalId, proofHash) 재사용을 propose 단계에서 revert 한다.
  각 spec 은 고유 wid(wd-xfer/wd-execute/wd-cancel/wd-reject/wd-claim/wd-events2/wd-double)를 쓰며,
  **fresh Boho-v2 넷에서 단발 통과**를 전제로 한다(같은 넷 재실행 시 wid 소진으로 propose fail — 다른 spec 과 동일).
- **레거시 상수 결함 동시 수정**: `govbind.BurnDepositRefundedTopic` 이 틀린 파라미터 순서로 pin 돼 있었다
  (위 "레거시 결함" 표 참조). 라이브 방출 topic `0x116044c8…` 으로 교체.

### 재분류 — `logs` 어세션은 이미 topic 인덱싱을 지원한다

이전 원장은 이벤트 로그 케이스(contract-event-emitted·token-transfer-emits-event·token-approve-sets-allowance)를
"`select` 가 배열/topic 인덱스를 못 짚는다"는 갭으로 분류했으나, **이는 혼동이었다**. `logs` 어세션(`internal/testspec/logs.go`)은
실제로 (1) `topics` 필터에 위치별 매칭(문자열=정확 매칭, null/비문자열=와일드카드)을, (2) `select:topicN`(topic0..N)
으로 특정 topic 추출을, (3) `index` 로 매칭 로그 중 N번째 선택을 **이미 지원**한다. 따라서 위 3건은 **신규 문법 없이**
표현 가능했고 라이브 통과했다. (진짜 갭인 "`select` 배열 인덱싱"은 `resolve.go` 의 JSON-path 배열 인덱싱 — 별개 사안이며
`admin-peers-populated`·`prev-seals-quorum` 등에 여전히 적용된다.) `token-transfer-from-moves-balance` 도
`sendTx key:`(로컬 서명) + multi-arg `derive abiCall` 만으로 표현 가능했다 — 별도 2-계정 서명 Go 프리미티브는 불필요.

## 이관하면서 드러난 레거시 케이스의 결함

포팅은 케이스를 다시 읽게 만들고, 그 과정에서 원본의 오류가 드러났다.

| 레거시 케이스 | 문제 | 이관본에서의 처리 |
|---|---|---|
| `wbft-extra-info-fields` | `istanbul_getWbftExtraInfo` 를 **`"latest"` 태그로 호출** — 체인이 `block -2 not found` 로 거부한다. 구체적 블록 번호가 필요 | 헤드를 먼저 읽어(`read` + `$head`) 넘긴다 |
| `wbft-extra-info-fields` | `ChainCompat: [stablenet, wbft]` 인데 **`gasTip` 은 stablenet 전용** — wbft 응답에 그 필드가 없다 | 공통 필드(`committedSeal`·`preparedSeal`)만 양 체인 대상으로 남기고, `gasTip` 은 `stablenet-gastip-field.json` 으로 분리 |
| `p256-precompile-active` (hardfork) | 소스는 "stablenet 이 Boho 를 genesis 활성 → 0x100 P256VERIFY 가 valid 벡터에 `0x..01` 반환"을 단언하나, **실제 gstable 빌드는 0x100 에 대해 `"0x"` 반환** — precompile 미탑재. accounts secp256r1 3건과 동일 증상 | 라이브 반증으로 이관 보류. Boho/P256 탑재 바이너리 확보 여부를 체인팀에 확인 후 재분류 |
| `govbind.BurnDepositRefundedTopic` (binding) | 상수가 `0x334fe3ea…`(=`BurnDepositRefunded(address,uint256,uint256)`)로 **파라미터 순서가 틀렸다**. 라이브 v2 넷에서 cancel 은 `0x116044c8…`(=`BurnDepositRefunded(uint256,address,uint256)` — proposalId 가 첫 indexed)을 방출. 레거시 `tests/anzeon/gov_burn_refund.go:197` 이 이 상수로 cancel 영수증을 검사하므로 v2 넷 대상 실행 시 실패했을 것 | 라이브로 확인한 `0x116044c8…` 으로 상수 교체 + 유닛테스트를 `EventTopic("BurnDepositRefunded(uint256,address,uint256)")` 파생 검증으로 강화(len==66 → 정확 매칭). BurnRefundClaimed 는 기존 그대로 정확 |
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
| `validator-set-count` | 검증자 수를 **토폴로지에서 파생**해 비교한다. spec 이 자기 토폴로지를 참조할 수단이 없다(현재는 `Len` 에 상수 4를 쓴다). 배열 길이 자체는 이제 `select:"#"` 로 얻지만, 비교 대상 quorum/검증자수를 토폴로지에서 파생하는 수단이 없다 |
| `prev-seals-quorum` | prevCommitted/prevPrepared seal 의 **sealer 수 >= quorum(ceil 2N/3)** 만 검사한다(서명 필드 없음). 배열 길이 비교는 이제 `select:"#"`+`GreaterOrEqual` 로 가능하나, **토폴로지 파생 quorum 산술**(ceil 2N/3)이 없다 — 그 갭으로 남긴다 |

### gas-policy 잔여 4건과 필요한 문법 확장

tx-flow 6 + 제출거부 4(feecap-below-min·legacy-gasprice-below-min·gaslimit-exceeded·accesslist-gasprice-below-min)건은 이관 완료.
`(B) expect:"reject"` 프리미티브(F1)로 제출거부가 라이브 확정됐고, `(C) accessList` 필드로
access-list(0x01) 제출거부까지 표현됐다. 나머지 4건은 아래 갭에 막혀
있다 — 가짜로 만들지 않고 남긴다. (`tipcap-underpriced-rejected` 는 라이브 반증으로 보류 — 결함 표 참조.)

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| ~~`revert-tx-status-zero` · `out-of-gas-consumes-all`~~ ✅ **이관 완료(라이브 pass)** — 자산은 spec 안의 initcode 리터럴(REVERT/무한루프 런타임, contract-event-emitted 와 같은 sendTx 생성 패턴). revert 는 `expect:"revert"`+receipt status, out-of-gas 는 receipt `gasUsed` 를 `derive diff`==0 으로 gasLimit(50000)와 대조 | — |
| ~~`authorized-account-gastip-free`~~ ✅ **이관 완료(라이브 pass)** — (G) 쿼럼 흐름으로 fresh 계정 인가→`waitFor call isAuthorized==1`(반영 대기 필수)→**key 서명 sendTx 에 fee cap 지원 추가**(maxFee/maxPriorityFee → `SendDynamicFeeGas`; 없으면 지갑이 제안값=강제 gasTip 을 쓴다)→3×gasTip 요청이 그대로 청구됨을 `derive diff(egp,base)`==3×gasTip 로, AuthorizedTxExecuted 를 receiptLog 로 확인 | — |

### accounts 잔여 18건과 필요한 문법 확장

17건은 이관 완료(9 라이브 + 제출거부 3 라이브 + contract-event-emitted 라이브 + access-list-tx 라이브 + secp256r1 3 오프라인). 나머지 18건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| ~~`feepayer-insufficient-rejected` · `fee-delegated-unfunded-feepayer-rejected`~~ ✅ **이관 완료(라이브 pass, applepie 오버레이 넷)** — (D) `feePayerKey`(feepayer-blacklisted 에서 확보) + 미펀딩 fresh feePayer + `expect:"reject"`. 오버레이는 spec 의 `chain.genesisOverlay` 로 선언해 suite 가 자동 구성 | — |
| `fd-sender-sig-invalid-rejected` · `fd-feepayer-sig-invalid-rejected` · `fee-delegated-sender-sig-invalid-rejected` · `fee-delegated-feepayer-sig-invalid-rejected` | 위 (B) + **손상된 이중서명 raw 트랜잭션 조립**(EncodeFeeDelegatedTampered) 을 spec 에서 만들 수단이 없다 | (B) + raw 서명 조립 자산 |
| ~~`fee-delegated-transfer`~~ ✅ **이관 완료(라이브 pass, applepie 오버레이 넷)** — fresh sender·feePayer 펀딩 후 `key`+`feePayerKey` 0x16 전송, 수취인 balanceAt==금액. `external-fee-delegated-transfer` 는 남음(operator 공급 키 갭) | (D 확보) / external: env 키 바인딩 |
| `set-code-delegation` | **EIP-7702(0x04) set-code** — authorization 리스트/authority 서명 미지원 | (E) sendTx authorizationList + 신규 키 생성 |
| `nonce-ordering` · `replacement-tx` · `out-of-order-nonces-mine` · `same-nonce-replacement` | sendTx 가 **제출 후 receipt 동기 대기** → gap 난 nonce(N+1) 를 큐잉만 하고 나중에 채굴시킬 수 없고, "tx1 은 채굴되면 안 된다" 는 **부정 채굴 기대**도 없다 | 비동기 제출(대기 안 함) + 부정 채굴 assertion |
| `zero-address-transfer-blocked` · `precompile-transfer-blocked` | accounts SDK 의 **클라이언트측 정적 가드**(제출 전 거부)를 검사. DSL sendTx 는 노드로 직행하므로 이 가드를 태우지 못한다(의미가 다름) | SDK 가드 경로는 DSL 로 표현 대상 아님 — 갭으로 남김 |
| `external-value-transfer` | **operator 공급 funded 키**(CHAINBENCH_FUNDED_KEY)와 **런타임 신규 수취인 생성**이 필요. DSL 은 env 키 주입/키 생성 수단이 없다 | env 키 바인딩 + 키 생성 소스 |
| `fee-delegate-sign-rpc-present` | `eth_signRawFeeDelegateTransaction` 가 **method-not-found(-32601) 이 아님**을 확인(오류는 나도 됨). rpcCall assertion 은 RPC 오류를 스텝 실패로 처리 → "오류지만 not-found 는 아님" 을 표현 못함 | 메서드 존재 프로브(오류 코드 구분) |
| `eth-call-revert-returns-error` | eth_call 이 **revert 오류를 반환**해야 통과. `call` assertion 은 호출 오류를 스텝 실패로 처리할 뿐 부정 기대가 없다 | `call` 부정 기대(expectCallError) |

### hardfork 잔여 6건과 필요한 문법 확장

2건은 이관 완료(라이브). 나머지 6건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `p256-precompile-active` · `p256-rejects-invalid` | **라이브 반증**: gstable 빌드의 0x100 이 valid/corrupt/short 세 벡터 모두 `"0x"` 반환 → P256VERIFY 미탑재. `-active` 는 라이브 fail, `-rejects-invalid` 는 "precompile 부재"로 우연히 통과할 뿐 검증 의미 없음. 레거시 소스는 Boho-genesis P256 활성을 단언하나 바이너리와 불일치 | Boho/P256 탑재 gstable 바이너리 확보 후 재분류(체인팀 확인 필요). 현 빌드로는 표현해도 무의미 |
| ~~`anzeon-active-before-boho` · `prealloc-preserved-across-boho`~~ ✅ **이관 완료(라이브 pass, bohoBlock=10 넷)** — spec 의 `hardforks:{"boho":10}` 선언이 suite 를 지연 포크 넷으로 구성한다(genesis config.bohoBlock=10 확인). anzeon-active 는 `eth_getCode(GovValidator,"0x1")` Regexp(실코드), prealloc 은 블록 1 잔고 저장→`waitFor blockNumber≥11`(포크 통과 신호)→latest 잔고 Equal·nonce 0x0 | — |
| `govminter-code-changes-at-boho` | **라이브 반증**: chainbench genesis 빌더는 bohoBlock=10 이어도 GovMinter 코드를 처음부터 최종본으로 굽는다 — 블록 1 과 latest 의 getCode 가 동일(38250 hex, md5 일치, head 0x20 에서 실측). "v1→v2 코드 스왑" 신호 자체가 이 구성에는 없다 | delayed-boho 넷에 v1 코드를 굽는 genesis 소스(체인 특화 충실도) 확보 후 재분류 — P256 2건과 같은 부류 |
| `p256-inactive-before-boho` | 위 P256 2건과 동일 — 이 빌드의 0x100 은 포크 후에도 "0x" 를 반환해 "활성" 쪽 절반이 성립 불가 | Boho/P256 탑재 바이너리 확보 후 |

### system-contracts 잔여 21건과 필요한 문법 확장

33건 커버 완료(라이브 — read 9 + 토큰 write+event 2 + 거버넌스 쿼럼 10 + 비회원 로컬서명 3 + burn-refund 라이프사이클 8, +unblacklist-restores 는 address-unblacklisted-event 로 중복 커버). 나머지 13건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.
잔여 다수(≈9건)가 여전히 **거버넌스 쿼럼 흐름**에 의존하나, `(G)` 프리미티브(`receiptLog` + `derive abiCall`)로
이 흐름을 표현할 수 있음은 이미 8건으로 입증됐다(위 배치 근거). 남은 갭은 케이스별로 다르다:
① 실행 결과를 **proposals() 상태 워드 디코드**로만 확인하는 케이스(자연스러운 `call`/이벤트 side-effect 부재) —
`(H) derive op:"word"`(hex blob 의 N번째 32바이트 워드 추출) **확보**. burn-proposal-executes 로 status==Executed(3)
표현을 라이브 확정했다(위 배치 근거). 남은 상태워드 케이스도 동일 수단으로 이관 가능.
② **payable proposeBurn + refund 라이프사이클**(cancel/disapprove/claim) — proposeBurn 자체는 확보,
refundableBalance 가 재사용 넷에서 제안자별로 누적돼 `>0`/`==0` 어세션이 간섭하는 순서 위험이 남는다.
③ **부정 기대**(정족수 미달/이중 claim/비회원 호출). ④ **fee-delegation·키생성·2계정 서명·시간대기** 등 기존 갭.

| 레거시 케이스 (건수) | 갭 | 필요 확장 |
|---|---|---|
| ~~`gastip-governance-updates-header`~~ ✅ **이관 완료(라이브 pass, 4검증자 넷)** — 두 갭을 신규 프리미티브로 해소: (1) 시간 대기 → `waitFor`(source+comparator 폴링, `waitBlock` 일반화); (2) 폴링마다 최신 블록 재확인 → `rpcCall` params `"@latest"` 센티넬(호출 시점 `eth_blockNumber` 로 치환, `$head` 고정과 달리 매 폴링 재해석)로 `istanbul_getWbftExtraInfo("@latest").gasTip` 를 추적. 흐름: proposeGasTip(en1)→approve(en2 자동 execute)→`waitFor gasTip==25000000000000`→`$orig` 로 복원(멱등). GasTipUpdated 방출은 `logs` count≥1. **라이브 확인**: 넷 기본 헤더 gasTip=30000000000000(genesis param 27.6e12 과 별개의 WBFT 헤더 기본값) — 최초 target 30000000000000 이 기본값과 동일해 `proposeGasTip` 이 `SameGasTip()` revert 하던 것을 target 25000000000000(≠기본값·genesis param) 으로 수정해 해소. `$orig` 복원이라 재실행 멱등. `chainbench run` (엔진, `chainbench test` 아님) 으로 검증 | ✅ (G)+`waitFor`+`@latest` (확보) |
| ~~`validator-add-member-executes`·`masterminter-member-add-remove`~~ ✅ **이관 완료(라이브 pass)** — R4 이후 suite 가 매 실행 fresh 워크스페이스를 구성하므로 "전용 격리 넷" 조건이 기본값이 됐다. validator-add 는 preset node5 를 GovValidator 에 추가 후 `members()` 워드[0]==1; masterminter 는 add(quorum 3)→isMember/quorum 확인→remove(quorum 2, en2+en3 승인) 왕복 | — |
| ~~`burn-proposal-executes`~~ ✅ **이관 완료**(라이브) — payable proposeBurn→approve→`(H) derive word` status==Executed(3) | — | — |
| ~~`remove-minter-executes`~~ ✅ **이관 완료**(라이브) — configure→approve→`(H) derive word` 로 중간 isMinter==1 확인→remove→approve→`call` isMinter==0 | — | — |
| ~~`mint-transfer-event`~~ ✅ **이관 완료**(라이브, fresh 넷) — proposeMint(C0FFEE30)→approve 자동 execute→`balanceAt`==1e18 & NativeCoinAdapter Transfer(0x0→C0FFEE30) `logs`. 감쇠 넷 재기동(20M) 후 검증 | — | — |
| ~~`unblacklist-restores`~~ ✅ **커버 완료**(중복) — address-unblacklisted-event 가 add→approve→remove→approve→isBlacklisted==0 을 이미 전량 포함(이벤트 확인만 추가) | — | — |
| ~~`burn-transfer-event`·`burn-cancel-refundable`·`burn-execute-no-refundable`·`burn-reject-refundable`·`claim-burn-refund-succeeds`·`burn-refund-events`~~ ✅ **이관 완료**(라이브, Boho-v2 넷) — proposeBurn→cancel/disapprove/claim. 누적 간섭은 **refundableBalance 전후 델타**(`derive diff` before/after==1e18, execute==0, claim 후 절대 0)로 회피. reject 는 비제안자 3명 disapprove(maxRejections=2, >2 에서 Rejected)→상태워드[9]==7. transfer-event=NativeCoinAdapter Transfer, refund-event=BurnDepositRefunded+BurnRefundClaimed `logs` | — | — |
| ~~`quorum-deficient-stays-voting`~~ ✅ **이관 완료**(라이브 pass) — proposeMint→정족수 미달 executeProposal 을 `expect:"revert"`(채굴후 status 0x0)로 확정→proposals() 상태워드[9]==Voting(1)→cleanup approve. (B) `expect:"revert"` 프리미티브 실증 | — | — |
| ~~`direct-blacklist-call-rejected`·`non-member-configure-minter-rejected`~~ ✅ **이관 완료**(라이브 pass) — `newAccount`(런타임 키생성)→node1(멤버) 펀딩→비회원 로컬서명(`sendTx key`)으로 member-only 가드(AccountManager.blacklist / GovMasterMinter.proposeConfigureMinter)를 호출 시 `expect:"reject"` 제출거부 확정. (E) 비회원 로컬서명 프리미티브 실증 | — | — |
| ~~`claim-zero-refund-reverts`·`claim-burn-refund-double-reverts`~~ ✅ **이관 완료**(라이브, Boho-v2 넷) — zero=`newAccount`(refundable 0)→로컬서명 claim `expect:"revert"`; double=cancel 로 refundable 적립→claim1 성공(전액 0화)→claim2 `expect:"revert"`. gas 명시로 제출→채굴 status 0x0 | — | — |
| ~~`recipient-blacklisted-rejected`~~ ✅ **이관 완료**(라이브 pass) — GovCouncil 로 fresh 수취인 blacklist→ 노드(멤버)가 그 주소로 전송 시 `expect:"reject" reason:"blacklist"` 로 제출거부 확정→cleanup unblacklist. (G)+(B) 실증 | — | — |
| ~~`sender-blacklisted-rejected`~~ ✅ **이관 완료**(라이브 pass) — `newAccount`→node1 펀딩→GovCouncil 로 그 주소 blacklist(자동 execute)→비회원 로컬서명 전송 시 `expect:"reject" reason:"blacklist"` 제출거부 확정. (E)+(G) 실증 | — | — |
| ~~`feepayer-blacklisted-rejected`~~ ✅ **이관 완료(라이브 pass, Applepie 오버레이 넷)** — **(D) fee-delegation(0x16) 지원 추가**: `sendTx` 에 `feePayerKey` 인자 신설(`key`=sender 서명, `feePayerKey`=가스 대납 → `wallet.SendFeeDelegated` 0x16 라우팅). 흐름: `newAccount`×2(sender·feePayer)→둘 다 node1 펀딩(feePayer 는 blacklist 전에 펀딩해 거부사유를 insufficient-funds 아닌 blacklist 로 고정)→GovCouncil 로 feePayer blacklist(자동 execute)→sender 로컬서명 0x16 전송(feePayer 대납)을 `expect:"reject" reason:"blacklist"` 로 확정. SDK 정적 value-transfer 가드는 sender·recipient 만 검사하고 fee payer 는 검사 안 함 → tx 가 노드에 도달, 노드가 거부. (G)+(B)+(D) 실증. **라이브 전제**: type 0x16 은 Applepie 포크 필요 — chainbench genesis 빌더가 `config.applepieBlock` 을 누락(0x16 "pool not yet in Applepie" 거부) → genesis 에 `applepieBlock:0` 오버레이 후 검증(Boho-v2 오버레이와 동종 seam). `chainbench run` 으로 검증 | ✅ Applepie 오버레이 넷 라이브 pass | — |
| ~~`authorized-tx-executed-event`~~ 🟡 **이관 완료(오프라인 validate·라이브 미검증)** — 신규 DSL 코드 불필요(기존 프리미티브 조합): `newAccount`→node1 펀딩→GovCouncil(0x…1004) `proposeAddAuthorizedAccount(0x93a8bb99)`→en2 approve(자동 execute)로 fresh 계정 authorize→그 계정 로컬서명(`sendTx key`)으로 값 전송→그 tx 영수증에 AccountManager 의 **AuthorizedTxExecuted** 로그(topic0 `0x40e728a8…`)가 붙는지 `receiptLog` **어세션**(topic:0, expected=topic0 자기자신 → 존재 시 Equal 통과, 부재 시 no-matching-log 오류)으로 확인. (G) 실증 | fresh 넷 라이브 pass 확인 | — |
| ~~`authorized-extra-bit-synced`·`blacklisted-extra-bit-synced`·`dual-status-extra`·`extra-balance-preserved`~~ (4) ✅ **이관 완료(라이브 pass, account-extra 오버레이 넷)** — 순수 read(isAuthorized `0xfe9fbb80`/isBlacklisted `0xfe575a87` on AccountManager 0x…B00003, `balanceAt`). fixture 계정(bit62 authorized 0x90F7…, bit63 blacklisted 0x15d3…, bits62+63 dual 0x9965…, 각 balance 1e18)은 `account-extra` 제네시스 오버레이의 alloc `extra` 비트로 genesis-init 시 AccountManager 상태를 시드. **엔진 seam 추가**: attach 모드가 RPC 로 감지 못하는 오버레이 cap 을 운영자가 `chainbench run --cap account-extra` 로 단언 → 게이팅된 spec 이 skip 아닌 run. `chainbench run` 으로 검증 | ✅ account-extra 오버레이 넷 라이브 pass (`--cap`) |
| ~~`proposal-expiry-transitions`~~ (1) ✅ **이관 완료(라이브 pass, short-expiry 오버레이 넷)** — proposeGasTip→ProposalCreated 로그로 pid 추출→`@latest` 블록 timestamp 기준 만료시각(+40s) 까지 `waitFor`(bare sleep 아닌 읽기값 폴링, GreaterOrEqual)→expireProposal→proposals() 상태워드[9]==Expired(5). `short-expiry` 오버레이가 GovValidator expiry=30s 로 단축. `chainbench run --cap short-expiry` | ✅ short-expiry 오버레이 넷 라이브 pass (`--cap`) |
| ~~`token-transfer-from-moves-balance`~~ ✅ **이관 완료(라이브 pass)** — transferFrom 은 owner≠spender **2계정**(owner approve → spender 가 대신 transferFrom)이 필요하다. **갭 노트는 stale 였다**: `sendTx key:` 로컬서명(feepayer 케이스에서 추가)이 호출마다 별도 지갑을 열어 임의 키로 서명하므로, 한 spec 안에서 owner·spender 두 계정이 각자 서명 가능 — **신규 Go 프리미티브 불필요**. 흐름: `newAccount`×3(owner·spender·recipient)→owner·spender 를 node1 펀딩(spender 는 이 넷 고 gasPrice 로 transferFrom 가스가 ~7 ether 라 10 ether 펀딩)→`derive abiCall(0x095ea7b3 approve,[$spender,1000000])` 을 owner 키로 서명→`derive abiCall(0x23b872dd transferFrom,[$owner,$recipient,1000000])` 을 spender 키로 서명→`derive abiCall(0x70a08231 balanceOf,[$recipient])` calldata 로 `call`==0xf4240 확인. 토큰 0x1000(네이티브코인 어댑터) balanceOf==네이티브 잔고 실측 확인. fresh 계정이라 재실행 멱등 | ✅ (기존) `sendTx key:` + multi-arg `derive abiCall` (신규 불필요) |

## 규약

- **파일 1개 = 케이스 1개**, 파일명 = `id` = 레거시 `Name`(kebab-case). CI 가 id 중복을 막는다.
- `applicableChains` 로 대상 체인을 선언한다 — 미적용 체인에서는 fail 이 아니라 **skip**.
- `requires` 로 필요한 capability 를 선언한다(`rpc`, `ws`, `consensus`, `process`).
- 레거시 케이스는 **소비자 이관 전까지 제거하지 않는다**. 지금은 두 경로가 병존한다.

### 라이브 검증 근거 — R5 착수 배치 8건 (2026-09-01, R4 suite 경로)

이 배치부터 검증은 attach 가 아니라 **suite 경로**(`chainbench run --workspace-dir …`,
매 실행 fresh 4노드 워크스페이스)다. 그 전환이 드러낸 런타임 갭 2건을 함께 고쳤다:

- **역할 셀렉터 인덱스 폴백**(`session.Resolve`): suite 의 노드 표는 전원 validator 라
  `en1`/`en2` 가 아무것도 못 찾았다(attach 표는 반대로 전원 endpoint). 요청한 역할이
  표에 **아예 없을 때만** 서수를 인덱스 순서로 폴백한다 — 일부라도 있으면 엄격 유지.
- **key 서명 sendTx 의 fee cap**: 로컬 서명 경로가 fee 인자를 무시하고 `SendCoin`
  (지갑 제안값 = 강제 gasTip)으로 보냈다. `maxFeePerGas`+`maxPriorityFeePerGas` 가
  있으면 `SendDynamicFeeGas` 로 태운다.

```
gas-policy (+3) → fresh 4노드(31000/8600 기본 밴드, --wait-blocks 2): pass=3
                  revert-tx-status-zero · out-of-gas-consumes-all · authorized-account-gastip-free
system-contracts (+2) → 같은 배치: pass=2
                  validator-add-member-executes · masterminter-member-add-remove
accounts (+3) → applepie 오버레이 넷(spec 의 chain.genesisOverlay 로 자동 구성): pass=3
                  fee-delegated-transfer · fee-delegated-unfunded-feepayer-rejected ·
                  feepayer-insufficient-rejected
```

**잔여 미이관 24건** (등록 56 − 이관과 동명 spec 보유 32):
accounts 15 (fee-delegation 서명변조 4 · nonce 4 · SDK 정적가드 2 · external 2 ·
eth-call-revert 1 · sign-rpc 프로브 1 · set-code(0x04) 1) · api 1 (ws-subscribe-logs) ·
consensus 3 (epoch 대기 · 토폴로지 파생 2) · gas-policy 1 (tipcap-underpriced, 라이브 반증 보류) ·
hardfork 4 (P256 바이너리 3 · govminter 코드스왑 반증 1) · system-contracts 0.

delayed-boho 2건(anzeon-active·prealloc-preserved)은 위 배치 뒤에 이관됐다: spec 이
`hardforks` 로 지연 포크를 선언하면 suite 가 그 넷을 구성한다. govminter-code-changes 는
라이브 반증(코드 스왑 신호 부재), p256-inactive 는 기존 P256 반증에 합류.

### 라이브 검증 근거 — R5 프리미티브 배치 10건 (2026-09-01)

잔여의 "새 DSL 프리미티브 필요" 갭을 프리미티브 4종으로 닫고 accounts 10건을 이관했다.
전부 suite 경로(fresh 넷)에서 라이브 통과.

프리미티브(모두 체인 이름을 모른다 — C6 ACL 유지):
- **`txMined` 리드/어세션**: 해시에 영수증이 있으면 "true", 없으면 "false". `waitFor`
  로 한 tx 채굴을 기다리고 `assert txMined expected:"false"` 로 다른 tx 의 부재를
  확인한다(교체 tx 시나리오).
- **key 서명 sendTx 의 `nonce`·`wait:false`**: 로컬 서명 경로가 명시 nonce 와 비동기
  제출을 지원(`SendDynamicFeeTx` 경유). gap 난 nonce 를 큐잉하고 나중에 채우는 순서
  케이스가 표현 가능.
- **`sendRawTampered` 액션**: 0x16 이중서명 tx 를 만들어 sender/feepayer 한쪽 서명을
  손상시키고 raw 제출→노드 거부를 통과 조건으로. chainId·nonce·fee 는 노드에서 읽어
  서명 손상만이 유일한 거부 사유가 되게 한다. 계정 provider 의 `SupportsTxType(0x16)`
  로 게이팅.
- **`callError` 어세션**: eth_call 이 오류(revert)를 반환하면 통과. 값이 오면 실패.
- **`methodPresent` 어세션**: 메서드가 등록돼 있으면 통과(throwaway 인자로 오류는 나도
  됨). method-not-found(-32601)만 부재로 판정.

```
accounts (+10) → fresh 4노드(--wait-blocks 2): pass=10
  nonce (fresh 계정 로컬서명, maxFee 1e14·tip 3e13 > 강제 gasTip 2.76e13):
    nonce-ordering · out-of-order-nonces-mine (nonce 2,1,0 큐잉→nonceAt==3)
    replacement-tx · same-nonce-replacement (교체 tx 는 feeCap·tip 둘 다 ≥10% bump →
      txMined 교체==true, 원본==false)
  fee-delegation 서명변조 (applepie 오버레이 넷, sendRawTampered):
    fd-sender/fd-feepayer-sig-invalid-rejected ·
    fee-delegated-sender/feepayer-sig-invalid-rejected
  eth-call-revert-returns-error (Reverter 배포→callError) ·
  fee-delegate-sign-rpc-present (methodPresent eth_signRawFeeDelegateTransaction)
```

**잔여 미이관 14건**: 카테고리·소스파일·사유·손실의 정본 표는
[[legacy-retirement-plan]] §5 (`docs/dev/legacy-retirement-plan.md`). 세 갈래 —
A 설계 경계 6건(토폴로지 자기참조·구독 순서·SDK 정적가드), B 외부 블로커 3건
(external 2·set-code 0x04 1), C 라이브 반증 4건(P256 3·govminter 코드스왑 1).
