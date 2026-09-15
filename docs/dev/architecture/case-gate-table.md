# 케이스별 게이트 판정표

> 측정일 2026-09-15 · 대상 205건 중 아직 `applicableChains` 를 든 **107건**
> 방법은 `mainnet-config-worklist.md` §7.0.2 와 그 아래 §7.0.3

## 1. 이 문서가 답하는 것

케이스마다 **도는 데 진짜로 필요한 것이 무엇인지**를 정하고, 그 근거를 남긴다.
지금은 전부 `applicableChains: "stablenet"` 처럼 **체인 이름**이 적혀 있는데, 그것은
"어디서 돌 수 있나" 가 아니라 **"어디서 쓰였나"** 다.

## 2. 넓이는 층의 성질이 아니다

층 이름으로 좁고 넓음을 줄 세울 수 없다. 세어 보면 같은 층 안에서 넓이가 제각각이다.

| 제공 체인 수 | 능력 |
|---|---|
| 1 | `contract:*` 10개 · `engine:anzeon` · `engine:croissant` · `family:poa` · `fork:boho` · `fork:croissant` |
| 2 | `family:wbft` · `fork:applepie` · `fork:brioche` · `fork:pangyo` |
| 3 | `fork:istanbul` · `tx:*` 6개 · `rpc` · `ws` · `consensus` · `process` |

`fork:boho` 는 1개 체인이고 `fork:istanbul` 은 3개다. **넓이는 그 값의 성질이다.**
그래서 좁은 것부터 세우려면 **제공 체인 수**로 센다 — 위 표가 그것이고 기계가 만든다.

## 3. 결과

| 처리 | 건수 | 뜻 |
|---|---|---|
| **변환 가능** | 58 | 증거가 조건을 결정한다. `requires` 로 옮긴다 |
| **게이트 불필요** | 21 | 체인 고유의 것을 하나도 안 쓴다. `applicableChains` 를 지운다 |
| **표현 불가** | 16 | 매니페스트가 말할 칸이 없다. 그대로 둔다 |
| **보류 (X9)** | 6 | 매니페스트의 포크 목록이 불완전하다 |
| **보류 (remote)** | 3 | attach 전용. R 묶음에서 본다 |
| **값** | 2 | 성질이 아니라 값에 의존한다. P2 가 답한다 |
| **판단 필요** | 1 | 증거가 안 잡혔다 |

**가장 위험한 줄은 "게이트 불필요" 21건이다.** 세 체인 모두에서 돌게 되므로 넓어지는
폭이 가장 크고, `validate` 는 통과를 보장하지 못한다. 다른 묶음보다 작게 쪼개고
라이브로 확인한다.

## 4. 이 표가 주장하지 않는 것

**"이 조건을 만족하는 체인에서 통과한다" 고 말하지 않는다.** 케이스 본문에서 읽을 수
있는 것은 **무엇을 필요로 하는가**까지다. 게이트를 통과시켜도 ABI 가 다르거나 값이
달라 실패할 수 있다. 통과 여부는 **돌려 봐야** 안다.

## 5. 적용하면서 배운 것 (2026-09-15)

표의 `contract:`·`engine:`·`fork:` 41건을 적용했다. 판정표가 맞았는지 두 검사가
말해 줬고, **둘 다 무언가를 잡았다.**

### 기존 `applicableChains` 도 증거다

증거를 케이스 본문에서만 뽑았더니, 이미 `stablenet,wbft` 라고 적힌 3건에
`engine:anzeon` 을 붙여 **wbft 에서 SKIP 되게 만들었다.** 판정표 비교가 잡았다.

누군가 이미 "이건 wbft 에서도 돈다" 고 판단해 둔 것이고, **내 추론이 그것을 조용히
뒤집을 근거는 없다.** 셋은 그대로 두고 `gasTip` 이 wbft 헤더에 있는지부터 확인해야
한다.

- `ethereum/08-legacy-transfer` · `ethereum/09-dynamic-fee-tx` · `wbft/01-block-period-one-second`

### 주소에서 이름을 되찾을 때는 그 케이스의 체인 표를 쓴다

`0x…1003` 은 stablenet 에서 govMinter 이고 wbft 에서 govNCP 다. 체인을 안 가리고
주소→이름 표를 하나로 만들었더니 **stablenet 케이스에 `contract:govNCP` 를 붙였다.**
이 트랙이 §7.0.1 에서 확인한 사실을 스크립트가 어겼다.

### 요구 목록은 케이스가 실제로 쓰는 것에서 뽑는다

처음에는 표의 조건을 그대로 적었는데, 손으로 판정한 둘이 틀렸다 —
`19-upgrade-registry-order` 와 `20-v1-params-init-storage` 에 govCouncil 을 적었지만
실제로 읽는 것은 govMinter 와 govValidator 다. **주소는 증거이고, "이 케이스는 무엇에
관한 것인가" 라는 내 짐작은 증거가 아니다.** 지금은 이름이든 주소든 케이스가 쓰는
컨트랙트를 그대로 모아 적고, X8 가드가 둘이 어긋나면 잡는다.

컨트랙트는 `steps` 밖에도 있다 — genesis 오버레이가 `govMinter` 를 이름으로 적는다
(`stand-alone/03-unsupported-version`). 문서 전체를 훑어야 한다.

---

## 6. 케이스별 판정

#### `contract:govMinter` — 9건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json` — 컨트랙트 govMinter
- `go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json` — 컨트랙트 govMinter

#### `engine:anzeon` — 9건

**근거**: gasTip 헤더 필드를 읽는다 — 그 필드는 stablenet genesis 의 anzeon 블록에만 있다

- `go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/ethereum/08-legacy-transfer.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/ethereum/09-dynamic-fee-tx.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/wbft/14-stablenet-gastip-field.json` — RPC istanbul_getWbftExtraInfo

#### `contract:govCouncil + engine:anzeon` — 6건

**근거**: 주소 자리에서 그 컨트랙트를 부른다; 오버레이가 genesis 의 anzeon 블록에 쓴다

- `go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json` — 컨트랙트 govCouncil; genesis anzeon
- `go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json` — 컨트랙트 govCouncil; genesis anzeon
- `go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json` — 컨트랙트 govCouncil; genesis anzeon
- `go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json` — 컨트랙트 govCouncil; genesis anzeon
- `go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json` — 컨트랙트 govCouncil; genesis anzeon
- `go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json` — 컨트랙트 govCouncil; genesis anzeon

#### `family:wbft` — 5건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_getWbftExtraInfo)

- `go-stablenet/regression/api/14-wbft-extra-info-fields.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/wbft/02-wbft-seals-quorum.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/wbft/11-prev-seals-quorum.json` — RPC istanbul_getWbftExtraInfo
- `go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json` — RPC istanbul_getWbftExtraInfo

#### `fork:boho` — 4건

**근거**: boho 포크를 켜거나 읽는다

- `go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json` — genesis boho,bohoBlock
- `go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json` — genesis hardfork:boho
- `go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json` — genesis hardfork:boho
- `go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json` — genesis boho,bohoBlock

#### `contract:nativeCoinAdapter` — 3건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-stablenet/regression/api/10-estimate-gas-token-transfer.json` — 컨트랙트 nativeCoinAdapter
- `go-stablenet/regression/api/22-token-total-supply-readable.json` — 컨트랙트 nativeCoinAdapter
- `go-stablenet/regression/api/23-token-approve-sets-allowance.json` — 컨트랙트 nativeCoinAdapter

#### `contract:govCouncil` — 2건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json` — 컨트랙트 govCouncil
- `go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json` — 컨트랙트 govCouncil

#### `contract:govValidator` — 2건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-stablenet/regression/wbft/04-validator-add-member-executes.json` — 컨트랙트 govValidator
- `go-stablenet/regression/wbft/05-validator-remove-member-executes.json` — 컨트랙트 govValidator

#### `family:wbft` — 2건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_getValidators)

- `go-stablenet/regression/api/12-validator-set-nonempty.json` — RPC istanbul_getValidators
- `go-stablenet/regression/api/12b-validator-set-count.json` — RPC istanbul_getValidators

#### `contract:accountManager + contract:govCouncil` — 1건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-stablenet/regression/anzeon/12-basefee-redistributed-not-burned.json` — 컨트랙트 accountManager,govCouncil

#### `contract:accountManager + contract:govCouncil + engine:anzeon` — 1건

**근거**: gasTip 헤더 필드를 읽는다 — 그 필드는 stablenet genesis 의 anzeon 블록에만 있다

- `go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json` — 컨트랙트 accountManager,govCouncil; RPC istanbul_getWbftExtraInfo

#### `contract:govConfig + contract:govNCP + contract:govStaking + family:wbft` — 1건

**근거**: 주소 자리에서 그 컨트랙트를 부른다; istanbul_* 합의 RPC 를 쓴다 (istanbul_getValidators)

- `go-wbft/governance/01-wbft-govcontracts-at-genesis.json` — 컨트랙트 govConfig,govNCP,govStaking; RPC istanbul_getValidators

#### `contract:govCouncil` — 1건

**근거**: 업그레이드 레지스트리를 읽는다 — 시스템 컨트랙트 저장소

- `go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json`

#### `contract:govCouncil` — 1건

**근거**: 같음

- `go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json`

#### `contract:govCouncil + contract:govMasterMinter + contract:govMinter + contract:govValidator + contract:nativeCoinAdapter` — 1건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-stablenet/regression/api/06-system-contracts-deployed.json` — 컨트랙트 govCouncil,govMasterMinter,govMinter,govValidator,nativeCoinAdapter

#### `contract:govStaking` — 1건

**근거**: 주소 자리에서 그 컨트랙트를 부른다

- `go-wbft/governance/02-wbft-governance-register-staker.json` — 컨트랙트 govStaking

#### `contract:govValidator + engine:anzeon` — 1건

**근거**: 주소 자리에서 그 컨트랙트를 부른다; 오버레이가 genesis 의 anzeon 블록에 쓴다

- `go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json` — 컨트랙트 govValidator; RPC istanbul_getValidators,istanbul_isValidator; genesis anzeon

#### `engine:anzeon` — 1건

**근거**: blockPeriodSeconds 는 genesis 의 anzeon 블록이 정한다

- `go-stablenet/regression/wbft/01-block-period-one-second.json`

#### `family:wbft` — 1건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_nodeAddress)

- `go-stablenet/regression/api/11-node-address-returned.json` — RPC istanbul_nodeAddress

#### `family:wbft` — 1건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_getValidators, istanbul_getWbftExtraInfo)

- `go-stablenet/regression/api/12c-validator-equal-power.json` — RPC istanbul_getValidators,istanbul_getWbftExtraInfo

#### `family:wbft` — 1건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_getCommitSignersFromBlock, istanbul_getValidators)

- `go-stablenet/regression/api/13-commit-signers-quorum.json` — RPC istanbul_getCommitSignersFromBlock,istanbul_getValidators

#### `family:wbft` — 1건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_status)

- `go-stablenet/regression/api/15-istanbul-status-fields.json` — RPC istanbul_status

#### `family:wbft` — 1건

**근거**: istanbul_* 합의 RPC 를 쓴다 (istanbul_isValidator)

- `go-stablenet/regression/api/16-is-validator-flags.json` — RPC istanbul_isValidator

#### `fork:boho` — 1건

**근거**: boho 활성 상태를 읽는다

- `go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json`

#### `표현 불가 + engine:anzeon` — 1건

**근거**: gasTip 헤더 필드를 읽는다 — 그 필드는 stablenet genesis 의 anzeon 블록에만 있다

- `go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json` — RPC istanbul_getWbftExtraInfo; 거부 exceeds block gas limit

#### `게이트 불필요` — 19건

**근거**: 평범한 EVM 동작만 쓴다 — 체인 고유 컨트랙트·포크·필드가 없다

- `go-stablenet/regression/api/03-transaction-by-hash-fields.json`
- `go-stablenet/regression/api/04-transaction-receipt-fields.json`
- `go-stablenet/regression/api/05-transaction-count-increments.json`
- `go-stablenet/regression/ethereum/10-access-list-tx.json`
- `go-stablenet/regression/ethereum/11-nonce-ordering.json`
- `go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json`
- `go-stablenet/regression/ethereum/16-effective-gas-price.json`
- `go-stablenet/regression/ethereum/17-replacement-tx.json`
- `go-stablenet/regression/ethereum/17b-same-nonce-replacement.json`
- `go-stablenet/regression/ethereum/19-contract-roundtrip.json`
- `go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json`
- `go-stablenet/regression/ethereum/24-revert-tx-status-zero.json`
- `go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json`
- `go-stablenet/regression/ethereum/27-genesis-balance.json`
- `go-stablenet/regression/ethereum/27b-value-transfer.json`
- `go-stablenet/regression/ethereum/30-chain-id.json`
- `go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json`
- `go-stablenet/regression/ethereum/32-ws-subscribe-logs.json`
- `go-stablenet/regression/ethereum/36-contract-event-emitted.json`

#### `게이트 불필요` — 1건

**근거**: 모든 노드의 genesis 해시가 같은지 — 체인과 무관

- `go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json`

#### `게이트 불필요` — 1건

**근거**: 잔고 읽기 — 어느 EVM 에서나 성립

- `samples/01-sample-minimal.json`

#### `표현 불가` — 4건

**근거**: 바이너리의 거부 규칙(transaction underpriced) — 매니페스트가 말할 칸이 없다

- `go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json` — 거부 transaction underpriced
- `go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json` — 거부 transaction underpriced
- `go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json` — 거부 transaction underpriced
- `go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json` — 거부 transaction underpriced

#### `표현 불가` — 2건

**근거**: 바이너리의 거부 규칙(insufficient feePayer's funds) — 매니페스트가 말할 칸이 없다

- `go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json` — genesis applepieBlock; 거부 insufficient feePayer's funds
- `go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json` — genesis applepieBlock; 거부 insufficient feePayer's funds

#### `표현 불가` — 2건

**근거**: 같음

- `go-wbft/accounts/02-secp256r1-precompile-invalid.json`
- `go-wbft/accounts/03-secp256r1-precompile-short-input.json`

#### `값` — 1건

**근거**: baseFee 하한 20 Gwei 라는 값에 의존. engine:anzeon 은 하한이 있다는 것까지만 말한다

- `go-stablenet/regression/anzeon/06-basefee-minimum.json`

#### `값` — 1건

**근거**: baseFee 상한 값에 의존. 같은 이유

- `go-stablenet/regression/anzeon/07-basefee-maximum.json`

#### `표현 불가` — 1건

**근거**: EIP-7702 authorizationList 가스 비용. applepie 와 같은 처지 (X9)

- `go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json`

#### `표현 불가` — 1건

**근거**: account-extra 는 케이스가 스스로 선언해 자기를 만족시킨다

- `go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json`

#### `표현 불가` — 1건

**근거**: eth_signRawFeeDelegateTransaction 의 존재 여부. RPC 메서드 목록을 매니페스트가 말하지 않는다

- `go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json`

#### `표현 불가` — 1건

**근거**: 바이너리의 거부 규칙(zero) — 매니페스트가 말할 칸이 없다

- `go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json` — 거부 zero

#### `표현 불가` — 1건

**근거**: 바이너리의 거부 규칙(precompile) — 매니페스트가 말할 칸이 없다

- `go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json` — 컨트랙트 accountManager; 거부 precompile

#### `표현 불가` — 1건

**근거**: 바이너리의 거부 규칙(insufficient funds) — 매니페스트가 말할 칸이 없다

- `go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json` — 거부 insufficient funds

#### `표현 불가` — 1건

**근거**: 바이너리의 거부 규칙(exceeds block gas limit) — 매니페스트가 말할 칸이 없다

- `go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json` — 거부 exceeds block gas limit

#### `표현 불가` — 1건

**근거**: secp256r1 프리컴파일 존재 여부. precompile: 어휘가 없다

- `go-wbft/accounts/01-secp256r1-precompile-valid.json`

#### `보류 (X9)` — 6건

**근거**: applepie 를 켜는데 stablenet 매니페스트의 포크 목록에 없다

- `go-stablenet/regression/ethereum/18-set-code-delegation.json` — genesis applepieBlock
- `go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json` — genesis applepieBlock
- `go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json` — genesis applepieBlock
- `go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json` — genesis applepieBlock
- `go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json` — genesis applepieBlock
- `go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json` — genesis applepieBlock

#### `보류` — 2건

**근거**: 같음

- `remote/01-remote-rpc-health.json` — RPC net_peerCount,web3_clientVersion
- `remote/03-remote-balance-check.json`

#### `보류` — 1건

**근거**: remote 묶음은 attach 전용 — R 묶음에서 본다

- `remote/02-remote-chain-info.json`

#### `판단 필요` — 1건

**근거**: 증거가 안 잡혔다

- `go-stablenet/regression/api/20-admin-peers-populated.json` — RPC admin_peers

