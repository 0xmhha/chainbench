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
| `network` | 4 | **4** | ✅ `examples/specs/network-*.json`(선행 이관분) 3건 + **admin-peers-populated** 라이브(gstable) — `rpcCall admin_peers` 를 `select:"#"`(배열 길이)≥1 & `select:"0.id"`(배열 인덱싱) NotEqual "" 로 검증. 갭 없음 |
| `accounts` | 35 | **17** | ✅ 라이브 통과 (gstable). value/legacy/dynamic-fee transfer·tx-count·tx-by-hash·receipt·effective-gas·genesis-balance·contract-roundtrip 9건 + **제출거부 3건**(insufficient-funds·dynamic-fee-below-basefee·gas-limit-exceeds-block, `expect:"reject"`) + **contract-event-emitted**(컨트랙트 생성 sendTx→receipt contractAddress→execute→`logs` topic0 매칭) + **access-list-tx**(EIP-2930 0x01: 빈 `accessList:[]`+gasPrice → `eth_getTransactionByHash` type==0x1) 라이브, secp256r1 precompile 3건은 wbft 전용(gstable 미탑재 확인)이라 오프라인 검증만. 잔여 18건은 문법 갭(아래) |
| `gas-policy` | 17 | **13** | ✅ 라이브 통과 (gstable). read류 3 + tx-flow 6 (basefee min/max·effective-gas·gastip-forced·feecap exact/above-min) + **제출거부 4건**(feecap-below-min·legacy-gasprice-below-min·gaslimit-exceeded·accesslist-gasprice-below-min, `expect:"reject"`). `read/assert:"derive"`(sum/diff) 로 정확 산술 비교, `read:"derive"`(read source) 로 `feeCap = baseFee+tip` 를 계산해 sendTx 인자로 주입. 잔여 4건은 문법 갭(아래) |
| `hardfork` | 8 | **2** | ✅ 라이브 통과 (gstable). boho-chain-config-active(blockNumber/chainId/baseFee >0)·govminter-v2-code(codeAt≠"0x"). 잔여 6건 갭(아래) |
| `system-contracts` | 46 | **12** | ✅ 라이브 통과 (gstable). system-contracts-deployed(EVM 5종 codeAt)·adapter-code·token-metadata(WKRC 정확)·total-supply/balance readable·account authorization/blacklist readable·minter-status·validator-metadata 9건 + **토큰 write+event 2건**(token-transfer-emits-event·token-approve-sets-allowance, sendTx ABI calldata→`logs` topic 필터/select + allowance `call`) + **거버넌스 쿼럼 1건**(mint-proposal-executes: propose→`receiptLog` topic1 로 proposalId 추출→`derive abiCall` 로 approve calldata 조립→정족수 자동 execute→잔고 증가). read source `call`+`$var` 보간으로 totalSupply≥balance 표현. 잔여 34건 갭(아래) |

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

### 재분류 — `logs` 어세션은 이미 topic 인덱싱을 지원한다

이전 원장은 이벤트 로그 케이스(contract-event-emitted·token-transfer-emits-event·token-approve-sets-allowance)를
"`select` 가 배열/topic 인덱스를 못 짚는다"는 갭으로 분류했으나, **이는 혼동이었다**. `logs` 어세션(`internal/testspec/logs.go`)은
실제로 (1) `topics` 필터에 위치별 매칭(문자열=정확 매칭, null/비문자열=와일드카드)을, (2) `select:topicN`(topic0..N)
으로 특정 topic 추출을, (3) `index` 로 매칭 로그 중 N번째 선택을 **이미 지원**한다. 따라서 위 3건은 **신규 문법 없이**
표현 가능했고 라이브 통과했다. (진짜 갭인 "`select` 배열 인덱싱"은 `resolve.go` 의 JSON-path 배열 인덱싱 — 별개 사안이며
`admin-peers-populated`·`prev-seals-quorum` 등에 여전히 적용된다.) 남은 `token-transfer-from-moves-balance` 는
로그 갭이 아니라 **2-계정 서명** 갭이다.

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
| `validator-set-count` | 검증자 수를 **토폴로지에서 파생**해 비교한다. spec 이 자기 토폴로지를 참조할 수단이 없다(현재는 `Len` 에 상수 4를 쓴다). 배열 길이 자체는 이제 `select:"#"` 로 얻지만, 비교 대상 quorum/검증자수를 토폴로지에서 파생하는 수단이 없다 |
| `prev-seals-quorum` | prevCommitted/prevPrepared seal 의 **sealer 수 >= quorum(ceil 2N/3)** 만 검사한다(서명 필드 없음). 배열 길이 비교는 이제 `select:"#"`+`GreaterOrEqual` 로 가능하나, **토폴로지 파생 quorum 산술**(ceil 2N/3)이 없다 — 그 갭으로 남긴다 |

### gas-policy 잔여 4건과 필요한 문법 확장

tx-flow 6 + 제출거부 4(feecap-below-min·legacy-gasprice-below-min·gaslimit-exceeded·accesslist-gasprice-below-min)건은 이관 완료.
`(B) expect:"reject"` 프리미티브(F1)로 제출거부가 라이브 확정됐고, `(C) accessList` 필드로
access-list(0x01) 제출거부까지 표현됐다. 나머지 4건은 아래 갭에 막혀
있다 — 가짜로 만들지 않고 남긴다. (`tipcap-underpriced-rejected` 는 라이브 반증으로 보류 — 결함 표 참조.)

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `revert-tx-status-zero` · `out-of-gas-consumes-all` | 되돌아가는/가스소진 **컨트랙트 자산**과 gasUsed==gasLimit 판정이 필요 | 컨트랙트 바이트코드 자산 + receipt `gasUsed`/`gasLimit` read (부분 표현 가능, 자산 확보 선행) |
| `authorized-account-gastip-free` | **거버넌스 쿼럼 흐름**(proposeAddAuthorizedAccount → 승인) 을 먼저 태워야 한다 | system-contracts 배치와 함께 — 거버넌스 스텝 표현 확보 후 |

### accounts 잔여 18건과 필요한 문법 확장

17건은 이관 완료(9 라이브 + 제출거부 3 라이브 + contract-event-emitted 라이브 + access-list-tx 라이브 + secp256r1 3 오프라인). 나머지 18건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.

| 레거시 케이스 | 갭 | 필요 확장 |
|---|---|---|
| `feepayer-insufficient-rejected` · `fee-delegated-unfunded-feepayer-rejected` | (B) `expect:"reject"` 는 확보됐으나 **fee-delegation(0x16)** tx(feePayer 이중서명) 미지원 — 제출거부 자체는 표현 가능하나 0x16 를 조립할 수단이 없다 | (D) sendTx feePayer + 0x16 인코딩 |
| `fd-sender-sig-invalid-rejected` · `fd-feepayer-sig-invalid-rejected` · `fee-delegated-sender-sig-invalid-rejected` · `fee-delegated-feepayer-sig-invalid-rejected` | 위 (B) + **손상된 이중서명 raw 트랜잭션 조립**(EncodeFeeDelegatedTampered) 을 spec 에서 만들 수단이 없다 | (B) + raw 서명 조립 자산 |
| `fee-delegated-transfer` · `external-fee-delegated-transfer` | **fee-delegation(0x16) 트랜잭션** — sendTx 에 feePayer(이중서명) 필드가 없다 | (D) sendTx feePayer + 0x16 인코딩 |
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
| `govminter-code-changes-at-boho` · `p256-inactive-before-boho` · `anzeon-active-before-boho` · `prealloc-preserved-across-boho` | **delayed-boho 교차포크**: bohoBlock=N 으로 지연 활성한 뒤 fork 전(블록 1)·후(latest) 상태를 조건부 대기(WaitFor 크로스오버)로 비교. 1회성 read spec 은 포크 크로스오버를 표현할 수 없고, DSL 에 delayed-boho 조건부 대기가 없다 | delayed-boho 기동 + 크로스포크 조건부 WaitFor + 블록 고정(`0x1` vs latest) 비교 |

### system-contracts 잔여 34건과 필요한 문법 확장

12건은 이관 완료(모두 라이브 — read 9 + 토큰 write+event 2 + 거버넌스 쿼럼 1). 나머지 34건은 아래 갭에 막혀 있다 — 가짜로 만들지 않고 남긴다.
잔여 다수(≈19건)가 **거버넌스 쿼럼 흐름**(propose → 검증자 N명 approve → 정족수 자동 execute)에
의존한다. `(G)` 프리미티브(`receiptLog` read + `derive abiCall`)로 이 흐름을 표현할 수 있음을
mint-proposal-executes 로 라이브 확정했고(위 배치 근거 참조), 나머지는 케이스별 calldata/예약 예치 + 라이브 검증만 남았다.

| 레거시 케이스 (건수) | 갭 | 필요 확장 |
|---|---|---|
| `burn-proposal-executes`·`configure-minter-proposal-executes`·`gastip-governance-updates-header`·`blacklist-proposal-executes`·`authorize-proposal-executes`·`validator-add-member-executes`·`masterminter-member-add-remove`·`remove-minter-executes`·`unblacklist-restores`·`burn-cancel-refundable`·`burn-execute-no-refundable`·`burn-reject-refundable`·`claim-burn-refund-succeeds`·`burn-refund-events`·`authorized-account-added-event`·`unauthorize-proposal-executes`·`address-unblacklisted-event`·`mint-transfer-event`·`burn-transfer-event` (19) | **거버넌스 쿼럼 흐름** — propose(payable 포함)→검증자 N명 approve→정족수 execute→상태/이벤트 확인. `(G)` 프리미티브로 표현 가능(mint-proposal-executes 로 입증)하나, 케이스별 propose calldata·예약 예치·이벤트 로그(Transfer/AddressBlacklisted 등) 매칭을 각각 조립+라이브 검증해야 한다 | (G) `receiptLog`+`derive abiCall` (확보) + 케이스별 calldata + 이벤트 로그 매칭 |
| `quorum-deficient-stays-voting`·`claim-zero-refund-reverts`·`claim-burn-refund-double-reverts`·`direct-blacklist-call-rejected`·`non-member-configure-minter-rejected` (5) | 위 (G) + **부정 기대**(정족수 미달 execute revert / 이중 claim revert / 비회원 호출 reject-or-revert). DSL 은 제출에러를 스텝 실패로 처리할 뿐 부정 기대가 없다 | (G) + (B) `expectReject`/채굴후 status 0x0 기대 (accounts·gas-policy 와 동일) |
| `sender-blacklisted-rejected`·`recipient-blacklisted-rejected` (2) | 거버넌스 blacklist 후 **제출 거부**(에러 메시지 "blacklist"). 거버넌스 + 제출거부 부정기대 | (G) + (B) |
| `feepayer-blacklisted-rejected` (1) | 위 + **fee-delegation(0x16)** tx (feePayer 이중서명) 미지원 | (G) + (B) + (D) sendTx feePayer |
| `authorized-tx-executed-event` (1) | **런타임 키 생성 + faucet 펀딩 + 거버넌스 authorize** 후 tx 의 AuthorizedTxExecuted 로그 확인. 키 생성·env 주입·거버넌스·로그매칭 모두 부재 | (G) + 키 생성 소스 + 로그 매칭 |
| `authorized-extra-bit-synced`·`blacklisted-extra-bit-synced`·`dual-status-extra`·`extra-balance-preserved` (4) | 순수 read(isAuthorized/isBlacklisted/getBalance)라 **표현 자체는 가능**하나 fixture 계정이 **`account-extra` 제네시스 오버레이**에서만 존재/설정됨. 표준 stablenet 엔 없어 라이브 미검증 | `account-extra` cap 오버레이 기동으로 라이브 검증 후 이관(현재는 표현 가능·미기동) |
| `proposal-expiry-transitions` (1) | **`short-expiry` 오버레이 + 만료 시간 대기**(35s) 후 expireProposal. 조건부/시간 대기 표현 없음 | `short-expiry` cap + 시간/조건 대기 |
| `token-transfer-from-moves-balance` (1) | transferFrom 은 owner≠spender **2계정**(owner approve → spender 가 대신 transferFrom)이 필요하다. DSL sendTx 는 단일 노드서명 계정만 태운다. (approve/transfer 이벤트 로그 검증분은 이관 완료 — `logs` 가 topic 필터·`select:topicN`·`index` 를 이미 지원함을 재확인) | 2-계정 서명(별도 서명자 키) |

## 규약

- **파일 1개 = 케이스 1개**, 파일명 = `id` = 레거시 `Name`(kebab-case). CI 가 id 중복을 막는다.
- `applicableChains` 로 대상 체인을 선언한다 — 미적용 체인에서는 fail 이 아니라 **skip**.
- `requires` 로 필요한 capability 를 선언한다(`rpc`, `ws`, `consensus`, `process`).
- 레거시 케이스는 **소비자 이관 전까지 제거하지 않는다**. 지금은 두 경로가 병존한다.
