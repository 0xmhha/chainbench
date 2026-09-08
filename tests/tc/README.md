# tests/tc — 레거시 스위트와 같은 구조로 정리한 테스트 케이스

`~/Work/github/packages/chainbench/tests` 의 셸 스위트를 그대로 따라가도록 배치했다. DSL 문서는 자기가 쓸 config·genesis 를 파일 안에 적으므로, 레거시의 `configs/` 같은 곁가지는 옮기지 않았다.

## 1. 구조

1단은 체인, 그 아래는 레거시의 도메인 이름을 그대로 쓴다.

```
tests/tc/
├── go-stablenet/           ← 레거시 tests/stablenet
│   ├── regression/{ethereum,wbft,anzeon,fee-delegation,
│   │                blacklist-authorized,system-contracts,api}
│   └── post-v1.0.0-change/{common-all,extra-state,
│                           effectivegasprice,string-handling,stand-alone}
├── go-wbft/                ← 레거시 tests/wemix4 중 wbft 대상 + wbft 전용
├── go-wemix/               ← 레거시 tests/wemix4 중 wemix(poa) 대상
├── basic/  fault/  stress/  remote/   ← 레거시 동명 폴더
└── samples/                ← 작성 샘플
```

파일명은 `<레거시 번호>-<테스트 id>.json` 이다. 번호는 레거시 스위트의 순번을 그대로
가져와 대조가 되게 했고, 뒤의 id 가 무엇을 검증하는지 말한다. 레거시 하나가 여러
스펙으로 나뉜 경우 `11-`, `11b-` 처럼 뒤에 글자를 붙였다.

번호가 비어 있는 자리는 그 레거시 테스트가 DSL 이 아니라 `tests/e2e/` 의 Go 테스트로
옮겨진 자리다. 3절에 목록이 있다.

### 문서 한 장이 전부다

모든 문서가 `schemaVersion: "2"` 의 케이스이고, **체인 구성을 파일 안에 담는다.**
하나를 열면 어떤 체인 위에서 어떤 바이너리로 어떤 genesis·config 를 쓰는지, 그리고
무엇을 검증하는지가 그 안에 있다. 공용 `env/` 디렉터리는 없앴다 — id 로 부르면
파일만 봐서는 어떤 네트워크인지 알 수 없기 때문이다.

`description` 은 그 문서가 무엇을 검증하는지 사람이 읽으라고 있는 자리다. 레거시에
대응이 있는 것은 원본의 id 와 이름을 옮겨 적었다.

함께 돌릴 케이스는 env 가 같아야 한다. 같은 폴더에 있어도 다를 수 있다 —
`string-handling` 여섯 건은 각자 다른 genesis 를 요구한다. 실행기가 구성이 갈리는
묶음을 거부하므로(`sameComposition`) 조용히 틀린 네트워크에서 돌지는 않는다.

작성 방법은 `../../docs/dev/dsl-authoring-guide.md` 에 있다.

## 2. 디렉터리별 내용

각 줄이 그 문서 하나의 요약이다. 체인·바이너리·토폴로지·genesis 는 문서의 `env` 에서 그대로 읽은 값이다.

### `basic` (7)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-basic-consensus.json` | Verify blocks are being produced and all validators participate (원본 basic/consensus.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |
| `02-basic-peers.json` | Verify all nodes have proper peer connectivity (원본 basic/peers.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |
| `03-basic-rpc-health.json` | Verify all node RPC endpoints are responding (원본 basic/rpc-health.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |
| `04-basic-sync.json` | Verify all running nodes have synchronized block heights (원본 basic/sync.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |
| `05-basic-tx-send.json` | Send a transaction and verify it gets included in a block (원본 basic/tx-send.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |
| `06-basic-txpool-propagation.json` | Verify TX propagation across nodes and txpool drain under load (원본 basic/txpool-propagation.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |
| `07-basic-wbft-consensus.json` | Verify WBFT protocol properties - validator participation, round stability, commit seals (원본 basic/wbft-consensus.sh) | stablenet | `default=gstable` | bp=4, en=1 | — |

### `fault` (6)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-fault-network-partition.json` | Simulate network partition via admin_removePeer - verify consensus halts and recovers after heal (원본 fault/network-partition.sh) | stablenet | `default=gstable` | bp=4 | — |
| `02-fault-node-crash.json` | Stop 1 validator and verify consensus continues with 3/4 (원본 fault/node-crash.sh) | stablenet | `default=gstable` | bp=4 | — |
| `03-fault-node-recover.json` | Stop a node, wait, restart, and measure sync time (원본 fault/node-recover.sh) | stablenet | `default=gstable` | bp=4 | — |
| `04-fault-p2p-topology.json` | Test consensus and TX propagation under restricted hub-spoke P2P topology (원본 fault/p2p-topology.sh) | stablenet | `default=gstable` | bp=4 | — |
| `05-fault-two-down.json` | Stop 2/4 validators - consensus should halt, recover when 1 returns (원본 fault/two-down.sh) | stablenet | `default=gstable` | bp=4 | — |
| `06-fault-txpool-leader-change.json` | Verify pending transactions survive leader node failure and get processed by remaining validators (원본 fault/txpool-leader-change.sh) | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/post-v1.0.0-change/common-all` (19)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-stablenet-delayed-fork.json` | — | stablenet | `default=gstable` | bp=4 | 있음 |
| `02-govminter-v2-code.json` | TC-5-2-05: GovMinter v2 업그레이드는 코드만 교체하고 잔액은 건드리지 않는다. 하드포크 전후의 코드가 다르고 잔액이 같은지를 함께 본다. | stablenet | `default=gstable` | bp=4 | — |
| `03-burn-cancel-refundable.json` | TC-1-1-01, TC-1-1-10 — 소각 제안 취소 → refundableBalance 이동 및 BurnDepositRefunded 이벤트 검증 (원본 post-v1.0.0-change/common-all/03-test-burn-cancel-refund) | stablenet | `default=gstable` | bp=4 | — |
| `04-burn-reject-refundable.json` | TC-1-1-02 — 소각 제안 거부(reject) → refundableBalance 이동 → claimBurnRefund 정상 출금 검증 (원본 post-v1.0.0-change/common-all/04-test-burn-reject-refund) | stablenet | `default=gstable` | bp=4 | — |
| `05-burn-expire-refundable.json` | TC-1-1-03: 소각 제안이 만료되면 GovMinter 로 옮겨진 예치금이 환불 가능 잔액이 된다. proposeBurn 직후 GovMinter 잔액 증가, 만료 후 상태 Expired(5), refundableBalance 증가분, BurnDepositRefunded 이벤트를 확인한다. | stablenet | `default=gstable` | bp=4 | — |
| `06-burn-execute-no-refundable.json` | TC-1-1-04 — 소각 제안 승인(approve) → 자동 실행(execute) → refundableBalance == 0 검증 (원본 post-v1.0.0-change/common-all/06-test-burn-execute-no-refund) | stablenet | `default=gstable` | bp=4 | — |
| `07-claim-burn-refund-succeeds.json` | TC-1-1-05, TC-1-1-09 — claimBurnRefund 정상 출금 및 BurnRefundClaimed 이벤트 검증 (원본 post-v1.0.0-change/common-all/07-test-claim-refund-success) | stablenet | `default=gstable` | bp=4 | — |
| `08-claim-zero-refund-reverts.json` | TC-1-1-06 — refundableBalance 0 계정 claimBurnRefund revert 검증 (원본 post-v1.0.0-change/common-all/08-test-claim-refund-zero-revert) | stablenet | `default=gstable` | bp=4 | — |
| `09-claim-burn-refund-double-reverts.json` | TC-1-1-07 — claimBurnRefund 중복 호출 revert 검증 (원본 post-v1.0.0-change/common-all/09-test-claim-refund-double-revert) | stablenet | `default=gstable` | bp=4 | — |
| `10-prealloc-preserved-across-boho.json` | TC-1-1-11/12: 하드포크가 prealloc 계정의 잔액·nonce 를 보존하고, 시스템 컨트랙트의 스토리지 슬롯도 그대로 둔다. | stablenet | `default=gstable` | bp=4 | — |
| `12-legacy-gasprice-below-min-rejected.json` | TC-1-3-04 — LegacyTx gasPrice 최소 가스비 하한선 미만 거부 검증 (원본 post-v1.0.0-change/common-all/12-test-legacy-gasprice-below-min-revert) | stablenet | `default=go-stablenet` | bp=4 | — |
| `13-accesslist-gasprice-below-min-rejected.json` | TC-1-3-05 — AccessListTx gasPrice 최소 가스비 하한선 미만 거부 검증 (원본 post-v1.0.0-change/common-all/13-test-accesslist-gasprice-below-min-revert) | stablenet | `default=go-stablenet` | bp=4 | — |
| `14-feecap-below-min-rejected.json` | TC-1-3-06 — DynamicFeeTx gasTipCap 최소값 미만 거부 검증 (원본 post-v1.0.0-change/common-all/14-test-dynamic-fee-tipcap-below-min-revert) | stablenet | `default=go-stablenet` | bp=4 | — |
| `15-boho-chain-config-active.json` | TC-4-1-01 — Boho hardfork chain config verification (원본 post-v1.0.0-change/common-all/15-test-chain-config-boho) | stablenet | `default=gstable` | bp=4 | — |
| `16-anzeon-active-before-boho.json` | TC-4-1-02 — Boho 하드포크 값의 체인 설정 반영 및 런타임 활성화 검증 (원본 post-v1.0.0-change/common-all/16-test-boho-chain-config-activation) | stablenet | `default=gstable` | bp=4 | — |
| `17-estimategas-authorizationlist-cost.json` | TC-4-2-01/03: EIP-7702 authorizationList 를 1건·2건 붙였을 때 eth_estimateGas 가 그만큼 늘어난다. 경계값은 preActions 에서 이름을 붙여 한 번씩만 적는다. 상한은 노드의 estimateGas 오차 1.5% 정책에서 나온다 — 절대 상한은 하한 x 1.015, 증가분 상한은 절대 상한에서 baseline(21000)을 뺀 값이다. | stablenet | `default=gstable` | bp=4 | — |
| `19-upgrade-registry-order.json` | TC-5-2-01/02/03: block 0 에 Anzeon baseline 이 등록돼 있고, BohoBlock(100) 전까지 중간 업그레이드가 없으며, BohoBlock 에서 GovMinter 코드가 교체된다. | stablenet | `default=gstable` | bp=4 | — |
| `20-v1-params-init-storage.json` | TC-5-2-04: v1 시스템 컨트랙트의 Params 가 genesis 에서 초기화된다. GovValidator gasTip 슬롯(0x39)과 GovMinter quorum 슬롯(0x04)이 0이 아니어야 한다. | stablenet | `default=gstable` | bp=4 | — |
| `21-burn-refund-events.json` | — | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/post-v1.0.0-change/effectivegasprice` (4)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-effective-gas-price-authorized-bp-en.json` | TC-4-6-01 — 인가 계정 tx 의 effectiveGasPrice 가 블록 생산 노드와 snap-sync 엔드포인트에서 같다 | stablenet | `default=gstable` | bp=4, en=1, syncMode=snap | — |
| `02-effective-gas-price-regular.json` | TC-4-6-02 — EffectiveGasPrice for regular (non-authorized) account (BP vs snap-sync EN comparison) (원본 post-v1.0.0-change/effectivegasprice/02-test-regular-account) | stablenet | `default=go-stablenet` | bp=4 | — |
| `02b-effective-gas-price-regular-bp-en.json` | TC-4-6-02 — 일반 계정 tx 의 effectiveGasPrice 가 두 노드에서 같다 | stablenet | `default=gstable` | bp=4, en=1, syncMode=snap | — |
| `03-auth-tx-event-last-bp-en.json` | TC-4-6-04 — AuthorizedTxExecuted 가 영수증 로그의 마지막이고, 두 노드가 같은 값을 보고한다 | stablenet | `default=gstable` | bp=4, en=1, syncMode=snap | — |

### `go-stablenet/post-v1.0.0-change/extra-state` (8)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-authorized-extra-bit-synced.json` | TC-4-5-01,TC-4-5-02 — Account Extra alloc bits reflected in AccountManager (authorized + blacklisted) (원본 post-v1.0.0-change/extra-state/01-test-extra-alloc-to-contract) | stablenet | `default=gstable` | bp=4 | — |
| `01b-blacklisted-extra-bit-synced.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `02-stablenet-account-extra.json` | — | stablenet | `default=gstable` | bp=4 | 있음 |
| `03-extra-union-merge.json` | TC-4-5-05/06: alloc.Extra 와 GovCouncil params 가 서로 다른 계정을 인가하면 합집합이 되고, 양쪽에 다 있는 계정은 중복 없이 한 번만 센다 (3개, 4개가 아님). | stablenet | `default=gstable` | bp=4 | 있음 |
| `04-dual-status-extra.json` | TC-4-5-07 — 동일 주소 dual-status — authorized AND blacklisted 동시 반영 (원본 post-v1.0.0-change/extra-state/04-test-extra-dual-status) | stablenet | `default=gstable` | bp=4 | — |
| `05-extra-balance-preserved.json` | TC-4-5-08 — 동기화 시 무관 계정 잔액 보존 (원본 post-v1.0.0-change/extra-state/05-test-extra-balance-preserved) | stablenet | `default=gstable` | bp=4 | — |
| `06-invalid-extra-reject.json` | TC-4-5-09 — 미정의 Extra 비트를 가진 genesis 를 받은 노드는 부팅에 실패한다 | stablenet | `default=gstable` | bp=4, en=1 | — |
| `07b-extra-state-across-delayed-boho.json` | TC-4-5-10/11/12 — 하드포크가 지연돼도 alloc.Extra 가 AccountManager 에 반영되고 잔액은 보존된다 | stablenet | `default=gstable` | bp=4 | 있음 |

### `go-stablenet/post-v1.0.0-change/stand-alone` (4)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01b-signature-compat-across-swap.json` | TC-3-1-04 — 노드가 다른 바이너리로 재기동해도 기존 tx 의 blockNumber·status·from·to 가 보존된다 | stablenet | `default=gstable, upgrade=${GSTABLE_UPGRADE_BIN:-gstable}` | bp=4, en=1 | — |
| `02-genesis-mismatch.json` | TC-4-1-03 — 이 체인과 다른 genesis 로 빌드된 바이너리는 GenesisMismatch 로 기동에 실패한다 | stablenet | `default=gstable, mismatch=${GSTABLE_MISMATCH_BIN:-gstable-genesis-mismatch}` | bp=4 | — |
| `03-unsupported-version.json` | TC-5-2-06 — genesis 가 지원하지 않는 시스템 컨트랙트 버전을 선언하면 BohoBlock 을 커밋하지 못하고 멈춘다 | stablenet | `default=gstable` | bp=4 | 있음 |
| `04-genesis-block-hash-consistent.json` | TC-5-3-01: block 0 해시가 모든 노드에서 같고 parentHash 가 0 이다. 레거시는 릴리스 고정 해시와 비교했으나, 사설망은 env 마다 genesis 가 달라 노드 간 일치와 genesis 형태로 검증한다. | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/post-v1.0.0-change/string-handling` (6)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-authorized-accounts-no-space.json` | TC-4-3-01: GovCouncil authorizedAccounts splitAndTrim — 공백 없음 "0xaaa,0xbbb,0xccc" → 3 | stablenet | `default=gstable` | bp=4 | 있음 |
| `02-authorized-accounts-space.json` | TC-4-3-02: GovCouncil authorizedAccounts splitAndTrim — 항목 사이 공백 "0xaaa, 0xbbb, 0xccc" → 3 | stablenet | `default=gstable` | bp=4 | 있음 |
| `03-authorized-accounts-trim.json` | TC-4-3-03: GovCouncil authorizedAccounts splitAndTrim — 앞뒤 공백 " 0xaaa , 0xbbb " → 2 | stablenet | `default=gstable` | bp=4 | 있음 |
| `04-authorized-accounts-empty-item.json` | TC-4-3-04: GovCouncil authorizedAccounts splitAndTrim — 빈 항목 "0xaaa,,0xbbb" → 2 | stablenet | `default=gstable` | bp=4 | 있음 |
| `05-authorized-accounts-single.json` | TC-4-3-05: GovCouncil authorizedAccounts splitAndTrim — 단일 항목 "0xaaa" → 1 | stablenet | `default=gstable` | bp=4 | 있음 |
| `06-authorized-accounts-empty.json` | TC-4-3-06: GovCouncil authorizedAccounts splitAndTrim — 빈 문자열 "" → 0 | stablenet | `default=gstable` | bp=4 | 있음 |

### `go-stablenet/regression/anzeon` (10)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-regular-account-gastip-forced.json` | RT-C-01 — 일반 계정의 tipCap이 header.GasTip()으로 강제 대체됨 (원본 regression/anzeon/01-test-regular-account-gastip-forced) | stablenet | `default=go-stablenet` | bp=4 | — |
| `02-authorized-account-gastip-free.json` | RT-C-02 — Authorized 계정은 tipCap을 자유롭게 설정할 수 있음 (원본 regression/anzeon/02-test-authorized-account-gastip-free) | stablenet | `default=gstable` | bp=4 | — |
| `03-anzeon-basefee-increase.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `04-anzeon-basefee-stable.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `05-anzeon-basefee-decrease.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `06-basefee-minimum.json` | RT-C-06 — baseFee가 MinBaseFee(20 Gwei) 아래로 내려가지 않음 (원본 regression/anzeon/06-test-min-basefee) | stablenet | `default=go-stablenet` | bp=4 | — |
| `07-basefee-maximum.json` | RT-C-07 — baseFee가 MaxBaseFee(20,000,000 Gwei) 상한을 초과하지 않음 (원본 regression/anzeon/07-test-max-basefee) | stablenet | `default=go-stablenet` | bp=4 | — |
| `08-feecap-above-min-accepted.json` | — | stablenet | `default=go-stablenet` | bp=4 | — |
| `09-feecap-exact-min-accepted.json` | — | stablenet | `default=go-stablenet` | bp=4 | — |
| `11-gaslimit-exceeded-rejected.json` | — | stablenet | `default=go-stablenet` | bp=4 | — |

### `go-stablenet/regression/api` (25)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-block-transactions-field.json` | RT-G-1-01 — eth_getBlockByNumber(latest) (원본 regression/api/01-test-get-block-by-number) | stablenet | `default=gstable` | bp=4 | — |
| `02-block-by-hash-consistency.json` | RT-G-1-02 — eth_getBlockByHash (원본 regression/api/02-test-get-block-by-hash) | stablenet | `default=gstable` | bp=4 | — |
| `03-transaction-by-hash-fields.json` | RT-G-1-03 — eth_getTransactionByHash (원본 regression/api/03-test-get-tx-by-hash) | stablenet | `default=gstable` | bp=4 | — |
| `04-transaction-receipt-fields.json` | RT-G-1-04 — eth_getTransactionReceipt (PR #70 fix 확인) (원본 regression/api/04-test-get-tx-receipt) | stablenet | `default=gstable` | bp=4 | — |
| `05-transaction-count-increments.json` | RT-G-1-05 — eth_getTransactionCount (nonce 조회) (원본 regression/api/05-test-get-tx-count) | stablenet | `default=gstable` | bp=4 | — |
| `06-system-contracts-deployed.json` | RT-G-1-06 — eth_getCode on NativeCoinAdapter (0x1000) (원본 regression/api/06-test-get-code-system) | stablenet | `default=gstable` | bp=4 | — |
| `07-gas-price-positive.json` | RT-G-2-01 — eth_gasPrice == baseFee + GasTip (원본 regression/api/07-test-gas-price) | stablenet | `default=gstable` | bp=4 | — |
| `07b-gas-price-equals-basefee-plus-tip.json` | — | stablenet | `default=go-stablenet` | bp=4 | — |
| `08-max-priority-fee-equals-gastip.json` | RT-G-2-02 — eth_maxPriorityFeePerGas == WBFTExtra.GasTip (원본 regression/api/08-test-max-priority-fee) | stablenet | `default=go-stablenet` | bp=4 | — |
| `09-fee-history-well-formed.json` | RT-G-2-03 — eth_feeHistory (원본 regression/api/09-test-fee-history) | stablenet | `default=gstable` | bp=4 | — |
| `10-estimate-gas-token-transfer.json` | RT-G-2-04 — eth_estimateGas (NativeCoinAdapter.transfer) (원본 regression/api/10-test-estimate-system-call) | stablenet | `default=go-stablenet` | bp=4 | — |
| `11-node-address-returned.json` | RT-G-3-01 — istanbul_nodeAddress (원본 regression/api/11-test-node-address) | stablenet | `default=gstable` | bp=4 | — |
| `12-validator-set-nonempty.json` | RT-G-3-02 — istanbul_getValidators (원본 regression/api/12-test-get-validators) | stablenet | `default=gstable` | bp=4 | — |
| `12b-validator-set-count.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `13-commit-signers-quorum.json` | RT-G-3-03 — istanbul_getCommitSignersFromBlock (원본 regression/api/13-test-get-commit-signers) | stablenet | `default=gstable` | bp=4 | — |
| `14-wbft-extra-info-fields.json` | RT-G-3-04 — istanbul_getWbftExtraInfo (원본 regression/api/14-test-get-wbft-extra) | stablenet | `default=gstable` | bp=4 | — |
| `15-istanbul-status-fields.json` | RT-G-3-05 — istanbul_status (원본 regression/api/15-test-istanbul-status) | stablenet | `default=gstable` | bp=4 | — |
| `16-is-validator-flags.json` | RT-G-3-06 — istanbul_isValidator (원본 regression/api/16-test-is-validator) | stablenet | `default=gstable` | bp=4 | — |
| `18-txpool-status.json` | RT-G-4-02 — txpool_status: pending(연속 nonce) + queued(nonce gap) 분리 (원본 regression/api/18-test-txpool-status) | stablenet | `default=gstable` | bp=4 | — |
| `19-txpool-content-well-formed.json` | RT-G-4-03 — txpool_content: pending/queued 분리 내용 확인 (원본 regression/api/19-test-txpool-content) | stablenet | `default=gstable` | bp=4 | — |
| `20-admin-peers-populated.json` | RT-A-1-05 — P2P 피어 연결 확인 (원본 regression/ethereum/05-test-p2p-peers) | stablenet | `default=gstable` | bp=4 | — |
| `21-fee-delegate-sign-rpc-present.json` | RT-G-5-01 — eth_signRawFeeDelegateTransaction (원본 regression/api/21-test-sign-raw-fee-delegate) | stablenet | `default=gstable` | bp=4 | — |
| `22-token-total-supply-readable.json` | RT-G-5-02 — eth_call NativeCoinAdapter.totalSupply (원본 regression/api/22-test-total-supply) | stablenet | `default=gstable` | bp=4 | — |
| `23-token-approve-sets-allowance.json` | RT-G-5-03 — eth_call NativeCoinAdapter.allowance (원본 regression/api/23-test-allowance) | stablenet | `default=gstable` | bp=4 | — |
| `24-chain-not-syncing.json` | — | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/regression/blacklist-authorized` (9)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-sender-blacklisted-rejected.json` | RT-E-01 — 블랙리스트 계정이 Sender인 tx 거부 (ErrBlacklistedAccount) (원본 regression/blacklist-authorized/01-test-sender-blacklisted) | stablenet | `default=gstable` | bp=4 | — |
| `02-recipient-blacklisted-rejected.json` | RT-E-02 — 블랙리스트 계정이 Recipient인 tx 거부 (원본 regression/blacklist-authorized/02-test-recipient-blacklisted) | stablenet | `default=gstable` | bp=4 | — |
| `03-feepayer-blacklisted-rejected.json` | RT-E-03 — FeePayer가 블랙리스트 계정이면 거부 (원본 regression/blacklist-authorized/03-test-feepayer-blacklisted) | stablenet | `default=gstable` | bp=4 | — |
| `04-address-unblacklisted-event.json` | RT-E-04 — 블랙리스트 해제 거버넌스 흐름 검증 (원본 regression/blacklist-authorized/04-test-unblacklist) | stablenet | `default=gstable` | bp=4 | — |
| `05-zero-address-transfer-rejected.json` | RT-E-05: 0x0 주소로의 전송은 제출 단계에서 거부된다 (ErrZeroAddressTransfer). | stablenet | `default=gstable` | bp=4 | — |
| `06-precompile-transfer-rejected.json` | RT-E-06: 프리컴파일·시스템 컨트랙트 주소로의 값 전송은 5개 주소 모두 거부된다. | stablenet | `default=gstable` | bp=4 | — |
| `07-account-blacklist-readable.json` | RT-E-07 — AccountManager.isBlacklisted() 조회 검증 (원본 regression/blacklist-authorized/07-test-is-blacklisted) | stablenet | `default=gstable` | bp=4 | — |
| `08-account-authorization-readable.json` | RT-E-08 — AccountManager.isAuthorized() 조회 검증 (원본 regression/blacklist-authorized/08-test-is-authorized) | stablenet | `default=gstable` | bp=4 | — |
| `09-authorized-tx-executed-event.json` | RT-E-09 — AuthorizedTxExecuted 이벤트 발생 검증 (원본 regression/blacklist-authorized/09-test-authorized-tx-executed) | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/regression/ethereum` (26)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-stablenet-chain-up.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `08-legacy-transfer.json` | RT-A-2-01 — Legacy Tx (type 0x0) 발행 (원본 regression/ethereum/08-test-legacy-tx) | stablenet | `default=gstable` | bp=4 | — |
| `09-dynamic-fee-tx.json` | RT-A-2-02 — EIP-1559 DynamicFeeTx (type 0x2) 발행 (원본 regression/ethereum/09-test-dynamic-fee-tx) | stablenet | `default=gstable` | bp=4 | — |
| `10-access-list-tx.json` | RT-A-2-03: eth_createAccessList 로 노드가 만든 접근 목록을 붙여 type 0x01 트랜잭션을 보낸다. | stablenet | `default=gstable` | bp=4 | — |
| `11-nonce-ordering.json` | RT-A-2-04 — Nonce 순서 보장 (원본 regression/ethereum/11-test-nonce-ordering) | stablenet | `default=gstable` | bp=4 | — |
| `11b-out-of-order-nonces-mine.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `12-dynamic-fee-below-basefee-rejected.json` | RT-A-2-05a — GasTipCap < MinTip tx 거부 검증 (원본 regression/ethereum/12-test-tipcap-underpriced) | stablenet | `default=gstable` | bp=4 | — |
| `14-insufficient-funds-rejected.json` | RT-A-2-06 — 잔액 부족 tx 거부 (원본 regression/ethereum/14-test-insufficient-funds) | stablenet | `default=gstable` | bp=4 | — |
| `15-gas-limit-exceeds-block-rejected.json` | RT-A-2-07 — Gas Limit 초과 tx 거부 (블록 gas limit 초과) (원본 regression/ethereum/15-test-gaslimit-exceeded) | stablenet | `default=gstable` | bp=4 | — |
| `16-effective-gas-price.json` | RT-A-2-08 — eth_getTransactionReceipt의 effectiveGasPrice 검증 (원본 regression/ethereum/16-test-effective-gas-price) | stablenet | `default=gstable` | bp=4 | — |
| `17-replacement-tx.json` | RT-A-2-09 — 동일 nonce, 더 높은 GasFeeCap으로 tx 교체 (원본 regression/ethereum/17-test-replacement-tx) | stablenet | `default=gstable` | bp=4 | — |
| `17b-same-nonce-replacement.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `18-set-code-delegation.json` | RT-A-2-10 — SetCodeTx (type 0x4 / EIP-7702) 계정 코드 위임 (원본 regression/ethereum/18-test-setcode-tx) | stablenet | `default=gstable` | bp=4 | 있음 |
| `19-contract-roundtrip.json` | RT-A-3-01 — 컨트랙트 배포 (원본 regression/ethereum/19-test-contract-deploy) | stablenet | `default=gstable` | bp=4 | — |
| `22-estimate-gas.json` | RT-A-3-04 — eth_estimateGas 정상 동작 (원본 regression/ethereum/22-test-estimate-gas) | stablenet | `default=gstable` | bp=4 | — |
| `23-eth-call-revert-returns-error.json` | RT-A-3-05 — eth_call로 revert하는 함수 호출 시 에러 반환 (원본 regression/ethereum/23-test-eth-call-revert) | stablenet | `default=gstable` | bp=4 | — |
| `24-revert-tx-status-zero.json` | RT-A-3-06 — revert tx: receipt.status == 0x0, gasUsed만 차감 검증 (원본 regression/ethereum/24-test-revert-tx) | stablenet | `default=gstable` | bp=4 | — |
| `25-out-of-gas-consumes-all.json` | RT-A-3-07 — out-of-gas tx: gasUsed == gasLimit, 잔액 전량 차감 검증 (원본 regression/ethereum/25-test-out-of-gas) | stablenet | `default=gstable` | bp=4 | — |
| `27-genesis-balance.json` | RT-A-4-02 — eth_getBalance 정상 조회 (원본 regression/ethereum/27-test-eth-get-balance) | stablenet | `default=gstable` | bp=4 | — |
| `27b-value-transfer.json` | RT-A-4-03 — eth_sendRawTransaction 서명된 tx 전파 (원본 regression/ethereum/28-test-send-raw-tx) | stablenet | `default=gstable` | bp=4 | — |
| `29-logs-query-well-formed.json` | RT-A-4-04 — eth_getLogs 이벤트 로그 조회 (원본 regression/ethereum/29-test-eth-get-logs) | stablenet | `default=gstable` | bp=4 | — |
| `30-chain-id.json` | RT-A-1-01 — 제네시스 블록으로 노드 초기화 (원본 regression/ethereum/01-test-genesis-init) | stablenet | `default=gstable` | bp=4 | — |
| `31-ws-subscribe-new-heads.json` | RT-A-4-06 — eth_subscribe(newHeads) WebSocket 구독 (원본 regression/ethereum/31-test-ws-subscribe-heads) | stablenet | `default=gstable` | bp=4 | — |
| `32-ws-subscribe-logs.json` | RT-A-4-07 — eth_subscribe(logs) WebSocket 구독 (원본 regression/ethereum/32-test-ws-subscribe-logs) | stablenet | `default=gstable` | bp=4 | — |
| `33-stablenet-chain-up-15.json` | — | stablenet | `default=/data/chainbench/bin/gstable` | bp=13, pn=1, en=1 | — |
| `36-contract-event-emitted.json` | — | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/regression/fee-delegation` (7)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-fee-delegated-transfer.json` | RT-D-01 — FeeDelegateDynamicFeeTx (type 0x16) 정상 처리 (원본 regression/fee-delegation/01-test-fee-delegate-normal) | stablenet | `default=gstable` | bp=4 | 있음 |
| `02-fd-sender-sig-invalid-rejected.json` | RT-D-03 — Sender 서명 변조 시 거부 (원본 regression/fee-delegation/02-test-sender-sig-invalid) | stablenet | `default=gstable` | bp=4 | 있음 |
| `03-fd-feepayer-sig-invalid-rejected.json` | RT-D-04 — FeePayer 서명 변조 시 거부 (원본 regression/fee-delegation/03-test-feepayer-sig-invalid) | stablenet | `default=gstable` | bp=4 | 있음 |
| `04-feepayer-insufficient-rejected.json` | RT-D-05 — FeePayer 잔액 부족 시 거부 (원본 regression/fee-delegation/04-test-feepayer-insufficient) | stablenet | `default=gstable` | bp=4 | 있음 |
| `05-fee-delegated-sender-sig-invalid-rejected.json` | — | stablenet | `default=gstable` | bp=4 | 있음 |
| `06-fee-delegated-feepayer-sig-invalid-rejected.json` | — | stablenet | `default=gstable` | bp=4 | 있음 |
| `07-fee-delegated-unfunded-feepayer-rejected.json` | — | stablenet | `default=gstable` | bp=4 | 있음 |

### `go-stablenet/regression/system-contracts` (23)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-native-coin-adapter-code.json` | RT-F-1-01 — NativeCoinAdapter.transfer → 기본 코인 전송과 동일 (원본 regression/system-contracts/01-test-native-transfer) | stablenet | `default=gstable` | bp=4 | — |
| `01b-token-transfer-emits-event.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `02-token-balance-readable.json` | RT-F-1-02 — NativeCoinAdapter.balanceOf == eth_getBalance (원본 regression/system-contracts/02-test-balance-of) | stablenet | `default=gstable` | bp=4 | — |
| `03-token-transfer-from-moves-balance.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `04-mint-transfer-event.json` | RT-F-1-04 — Mint 실행 시 Transfer(0x0 → beneficiary) 이벤트 발생 (원본 regression/system-contracts/04-test-mint-transfer-event) | stablenet | `default=gstable` | bp=4 | — |
| `05-burn-transfer-event.json` | RT-F-1-05 — Burn 실행 시 Transfer(account → 0x0) 이벤트 발생 (원본 regression/system-contracts/05-test-burn-transfer-event) | stablenet | `default=gstable` | bp=4 | — |
| `06-mint-proposal-executes.json` | RT-F-2-01 — 코인 발행: proposeMint(proofData) → 승인 → execute (원본 regression/system-contracts/06-test-mint-proposal) | stablenet | `default=gstable` | bp=4 | — |
| `07-burn-proposal-executes.json` | RT-F-2-02 — 코인 소각: proposeBurn(proofData) payable → 승인 → execute (원본 regression/system-contracts/07-test-burn-proposal) | stablenet | `default=gstable` | bp=4 | — |
| `08-quorum-deficient-stays-voting.json` | RT-F-2-03 — quorum 미달 → proposal 상태 Voting 유지, 발행 미실행 (원본 regression/system-contracts/08-test-quorum-deficient) | stablenet | `default=gstable` | bp=4 | — |
| `09-validator-metadata-readable.json` | RT-F-3-04 — validatorList() + validatorToOperator(v) + validatorToBlsKey(v) 다중 호출 (원본 regression/system-contracts/09-test-validator-metadata) | stablenet | `default=gstable` | bp=4 | — |
| `10-gastip-governance-updates-header.json` | RT-B-06 — GasTip 거버넌스 변경 → 블록 헤더 WBFTExtra.GasTip 반영 검증 + 원복 (원본 regression/wbft/06-test-gastip-header-sync) | stablenet | `default=gstable` | bp=4 | — |
| `11-proposal-expiry-transitions.json` | RT-F-3-06 — proposal expiry 초과 → Expired 상태 전환 → execute 불가 (원본 regression/system-contracts/11-test-proposal-expiry) | stablenet | `default=gstable` | bp=4 | — |
| `12-configure-minter-proposal-executes.json` | RT-F-4-01 — GovMasterMinter.proposeConfigureMinter(address, uint256) → 승인 → execute (원본 regression/system-contracts/12-test-add-minter) | stablenet | `default=gstable` | bp=4 | — |
| `13-remove-minter-executes.json` | RT-F-4-02 — GovMasterMinter.proposeRemoveMinter(address) → 승인 → execute (원본 regression/system-contracts/13-test-remove-minter) | stablenet | `default=gstable` | bp=4 | — |
| `14-masterminter-member-add-remove.json` | RT-F-4-03 — GovMasterMinter 자체 멤버 추가/제거 (proposeAddMember, proposeRemoveMember) (원본 regression/system-contracts/14-test-masterminter-self-member) | stablenet | `default=gstable` | bp=4 | — |
| `15-non-member-configure-minter-rejected.json` | RT-F-4-04 — 비멤버 계정의 GovMasterMinter.proposeConfigureMinter 호출 거부 (원본 regression/system-contracts/15-test-non-member-rejected) | stablenet | `default=gstable` | bp=4 | — |
| `16-blacklist-proposal-executes.json` | RT-F-5-01 — GovCouncil blacklist proposal → execute → isBlacklisted == true (원본 regression/system-contracts/16-test-blacklist) | stablenet | `default=gstable` | bp=4 | — |
| `18-authorize-proposal-executes.json` | RT-F-5-03 — GovCouncil authorized account proposal → execute → isAuthorized == true (원본 regression/system-contracts/18-test-authorize) | stablenet | `default=gstable` | bp=4 | — |
| `19-unauthorize-proposal-executes.json` | f5-04-unauthorize (원본 regression/system-contracts/19-test-unauthorize) | stablenet | `default=gstable` | bp=4 | — |
| `20-direct-blacklist-call-rejected.json` | RT-F-5-05 — 비멤버가 AccountManager.blacklist 직접 호출 → revert (원본 regression/system-contracts/20-test-direct-blacklist-rejected) | stablenet | `default=gstable` | bp=4 | — |
| `23-authorized-account-added-event.json` | RT-F-5-08 — AuthorizedAccountAdded 이벤트 (원본 regression/system-contracts/23-test-authorized-account-added-event) | stablenet | `default=gstable` | bp=4 | — |
| `25-token-metadata.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `28-minter-status-readable.json` | — | stablenet | `default=gstable` | bp=4 | — |

### `go-stablenet/regression/wbft` (9)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-block-period-one-second.json` | RT-B-01 — 블록 생산 주기 1초 간격 (원본 regression/wbft/01-test-block-period) | stablenet | `default=gstable` | bp=4 | — |
| `02-wbft-seals-quorum.json` | RT-B-02 — WBFTExtra에 Committed Seal + Prepared Seal 모두 존재하고 quorum 이상 (원본 regression/wbft/02-test-wbft-extra-seal) | stablenet | `default=gstable` | bp=4 | — |
| `03-epoch-transition-carries-epoch-info.json` | RT-B-03 — 에폭 전환 — 검증자 집합 갱신 (원본 regression/wbft/03-test-epoch-transition) | stablenet | `default=gstable` | bp=4 | — |
| `04-validator-add-member-executes.json` | RT-B-04 — 신규 검증자 추가 및 에폭 합의 참여 확인 (원본 regression/wbft/04-test-add-validator) | stablenet | `default=gstable` | bp=4 | — |
| `04b-validator-add-member-epoch-activates.json` | RT-B-04 — 멤버로 추가된 노드가 에폭 경계에서 실제 검증자 집합에 들어간다 | stablenet | `default=gstable` | bp=4, en=1 | 있음 |
| `05-validator-remove-member-executes.json` | RT-B-05: proposeRemoveMember 가 실행되면 GovValidator 멤버에서 빠진다. 제거 대상을 먼저 추가해 자기 완결로 만든다. | stablenet | `default=gstable` | bp=4 | — |
| `11-prev-seals-quorum.json` | RT-B-11 — 블록 N+1의 PrevCommittedSeal이 블록 N의 committers를 포함 (원본 regression/wbft/11-test-prev-committed-seal) | stablenet | `default=gstable` | bp=4 | — |
| `13-randao-and-mixdigest-present.json` | — | stablenet | `default=gstable` | bp=4 | — |
| `14-stablenet-gastip-field.json` | — | stablenet | `default=gstable` | bp=4 | — |

### `go-wbft/accounts` (3)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-secp256r1-precompile-valid.json` | — | wbft | `default=gwbft` | bp=4 | — |
| `02-secp256r1-precompile-invalid.json` | — | wbft | `default=gwbft` | bp=4 | — |
| `03-secp256r1-precompile-short-input.json` | — | wbft | `default=gwbft` | bp=4 | — |

### `go-wbft/chain-up` (2)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-wbft-chain-up.json` | — | wbft | `default=${GWBFT_BIN:-gwbft}` | bp=4 | — |
| `02-wbft-chain-up-15.json` | — | wbft | `default=/data/chainbench/bin/gwbft` | bp=13, pn=1, en=1 | — |

### `go-wbft/consensus` (1)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-e1-mixed-producers.json` | — | stablenet | `default=gstable` | nodes=[{'index': 1, 'role': 'en'}, {'index': 2, 'role': 'bp'}, {'index': 3, 'role': 'bp'}, {'index': 4, 'role': 'bp'}] | — |

### `go-wemix/chain-up` (2)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-wemix-chain-up.json` | — | wemix | `default=${GWEMIX_BIN:-gwemix}` | bp=4 | — |
| `02-wemix-chain-up-15.json` | — | wemix | `default=/data/chainbench/bin/gwemix` | bp=13, en=2 | — |

### `go-wemix/handoff` (1)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-wemix-wbft-handoff.json` | — | wbft | `producer=${GWEMIX_BIN:-gwemix}, validator=${GWBFT_BIN:-gwbft}` | bp=4 | — |

### `go-wemix/rpc` (1)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-wemix-brioche-block-reward.json` | — | wemix | `default=${GWEMIX_BIN:-gwemix}` | bp=4 | 있음 |

### `go-wemix/tx` (1)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-wemix-tx-and-contract.json` | — | wemix | `default=${GWEMIX_BIN:-gwemix}` | bp=4 | 있음 |

### `remote` (3)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-remote-rpc-health.json` | 레거시 remote/rpc-health: 붙은 엔드포인트가 살아 있고 기본 RPC 셋이 응답한다. chainbench run --rpc <endpoint> 로 실행한다. | stablenet | `default=gstable` | bp=4 | — |
| `02-remote-chain-info.json` | 레거시 remote/chain-info: 붙은 체인이 chainId 를 보고하고 동기화가 끝나 있다. | stablenet | `default=gstable` | bp=4 | — |
| `03-remote-balance-check.json` | 레거시 remote/balance-check: 잔액 조회가 16진 수량으로 돌아온다. 레거시 기본값과 같이 0 주소를 본다 — 어느 체인에나 있고 값이 변하지 않는다. | stablenet | `default=gstable` | bp=4 | — |

### `samples` (2)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-sample-spec.json` | v1 스펙 샘플. 이미 떠 있는 체인에 붙어 실행한다. steps 로 값을 모으고 assertions 로 판정한다. chainbench validate tests/tc/samples/01-sample-spec.json | stablenet | `default=gstable` | bp=4 | — |
| `02-sample-case.json` | 작성 샘플 — 노드를 멈췄다 살리고 체인이 이어지는지 확인한다 (docs/dev/dsl-authoring-guide.md) | stablenet | `default=${GSTABLE_BIN:-gstable}` | bp=4, en=1 | 있음 |

### `stress` (2)

| 파일 | 검증 내용 | 체인 | 바이너리 | 토폴로지 | genesis overlay |
|---|---|---|---|---|---|
| `01-stress-block-time.json` | Measure block production time statistics over last 100 blocks (원본 stress/block-time.sh) | stablenet | `default=gstable` | bp=4 | — |
| `02-stress-tx-flood.json` | Send N transactions rapidly and measure throughput (원본 stress/tx-flood.sh) | stablenet | `default=gstable` | bp=4 | — |

## 3. Go e2e 로 옮겨진 레거시 테스트

아래 테스트는 노드 생명주기·동기화·합의 정지처럼 DSL 이 표현하기 어려운 것이라 `tests/e2e/` 의 Go 테스트가 맡는다. 그래서 tc/ 에는 그 번호가 비어 있다.

| 레거시 | Go 테스트 |
|---|---|
| `post-v1.0.0-change/stand-alone/01-test-signature-compat-sync` | e2e TestE2E_StablenetHardforkSwap |
| `regression/ethereum/02-test-full-sync` | e2e TestE2E_StablenetSyncGap |
| `regression/ethereum/03-test-snap-sync` | e2e TestE2E_StablenetSyncGap / TestE2E_WbftSnapSync |
| `regression/ethereum/04-test-node-restart` | cases/fault-node-recover + e2e ConsensusLifecycle |
| `regression/ethereum/06-test-downloader-path` | e2e TestE2E_StablenetSyncGap |
| `regression/ethereum/07-test-block-fetcher-path` | e2e TestE2E_StablenetBlockPropagation |
| `regression/system-contracts/11-test-proposal-expiry` | specs/system-contracts/proposal-expiry-transitions + e2e ProposalExpiry |
| `regression/wbft/08-test-quorum-deficient` | e2e TestE2E_WbftFaultHalt / WbftQuorum* |
| `regression/wbft/09-test-round-change` | e2e TestE2E_WbftViewChange |
| `regression/wbft/10-test-post-round-change` | e2e TestE2E_WbftRoundRobinProposer |

## 4. 레거시 대비 커버리지

감사에서 나온 **누락 18건은 전부 옮겼다.** 부분 포팅으로 남아 있던 것 중
검증 범위가 눈에 띄게 좁았던 셋(effectivegasprice 3건의 노드 간 일치,
wbft add-validator 의 에폭 경계)도 채웠다.

남은 것은 사설망에서 값을 고정할 수 없어 등호를 부등호로 낮춘 항목들과,
fee-delegation 4건의 `personal_*` API 경로다. 후자는 로컬 서명으로 대체한 것이
의도인지 먼저 정해야 한다. 자세한 목록은
`../../docs/dev/legacy-port-audit/03-port-audit.md` 3.3 절에 있다.

### 4.1 이번에 추가한 스펙

| 새 스펙 | 레거시 | 원본과 달라진 점 |
|---|---|---|
| `regression/blacklist-authorized/05-zero-address-transfer-rejected` | RT-E-05 | 거부 사유를 `zero` 로 잡는다. 원본은 `zero address` 와 `ZeroAddress` 두 철자를 받았다 |
| `regression/blacklist-authorized/06-precompile-transfer-rejected` | RT-E-06 | 같음 (5개 주소 모두 제출 거부) |
| `regression/wbft/05-validator-remove-member-executes` | RT-B-05 | 자기 완결로 바꿨다. 원본은 04 가 먼저 추가해 둔 멤버를 지웠지만, 여기서는 추가와 제거를 한 스펙 안에서 한다 |
| `post-v1.0.0-change/common-all/05-burn-expire-refundable` | TC-1-1-03 | 원본 9개 단언 중 6개를 옮겼다. GovMinter 잔액 증가, Expired(5), refundableBalance 증가분, BurnDepositRefunded 이벤트는 그대로다. 빠진 것은 소각 계정 잔액 감소, `burnBalance == 0`, 환불 청구(claimBurnRefund) 후 출금 확인 세 가지다. `short-expiry` 를 요구한다 |
| `post-v1.0.0-change/common-all/19-upgrade-registry-order` | TC-5-2-01/02/03 | 같음 (block 0 / 0x63 / 0x64 코드 비교) |
| `post-v1.0.0-change/common-all/20-v1-params-init-storage` | TC-5-2-04 | gasTip 슬롯을 상수와 등호로 비교하지 않고 0이 아님으로 본다. 사설망마다 gasTip 이 다르다 |
| `post-v1.0.0-change/extra-state/03-extra-union-merge` | TC-4-5-05/06 | genesisOverlay 로 alloc.Extra 와 GovCouncil params 를 함께 넣어 자기 완결로 만들었다. 계정은 원본과 다르지만 합집합·중복 제거라는 명제는 같다 |
| `post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent` | TC-5-3-01 | 릴리스 고정 해시 대신 노드 간 일치와 `parentHash == 0` 으로 본다. 사설망은 env 마다 genesis 가 달라 고정값을 쓸 수 없다 |
| `post-v1.0.0-change/string-handling/01~06` | TC-4-3-01~06 | 입력 문자열을 genesisOverlay 로 넣는다. 원본이 시나리오마다 전용 바이너리를 바꿔 끼우던 자리다. `01` 의 입력은 레거시 원본대로 `0xaaa,0xbbb,0xccc` 를 쓴다 |

| `post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost` | TC-4-2-01/03 | 위임 대상을 고정 주소 대신 새로 만든 계정으로 잡는다. 경계값은 preActions 에서 이름을 붙여 한 번씩만 적는다 |
| `post-v1.0.0-change/stand-alone/02-genesis-mismatch` | TC-4-1-03 | 원본은 SSH 로 바이너리를 직접 실행해 stderr 를 봤다. 여기서는 `swapNode` 에 `expect: fail` 을 걸고 노드가 RPC 에 응답하지 않는 것과 그 이유가 genesis 임을 확인한다. mismatch 바이너리가 사전 조건이다 (env 의 `GSTABLE_MISMATCH_BIN`) |
| `post-v1.0.0-change/stand-alone/03-unsupported-version` | TC-5-2-06 | 원본은 BohoBlock 커밋 실패를 로그로 봤다. 여기서는 genesis overlay 로 지원하지 않는 버전을 선언하고, BohoBlock 직전까지 생산한 뒤 `blockStalled` 로 멈춤을 확인하며 노드 로그를 아티팩트에 남긴다 |

| `post-v1.0.0-change/extra-state/06-invalid-extra-reject` | TC-4-5-09 | 원본은 미정의 Extra 비트가 박힌 바이너리로 노드를 띄웠다. 여기서는 `swapNode` 의 `genesisOverlay` 로 EN 한 대에만 그 genesis 를 주고 `expect:"fail"` 로 부팅 거부를 확인한다. 원본의 cleanup 단계는 옮기지 않았다 — compose 는 실행마다 새 워크스페이스를 쓴다 |
| `post-v1.0.0-change/effectivegasprice/01·02b·03` | TC-4-6-01/02/04 | 원본이 보던 **BP 와 snap-sync EN 의 값 일치**를 되살렸다. 기존 단일 노드 스펙은 그대로 두고, snap 엔드포인트가 있는 env 위에서 두 노드를 비교하는 케이스를 더했다. 03 은 AuthorizedTxExecuted 가 로그의 *마지막*인지도 두 노드에서 확인한다 |
| `regression/wbft/04b-validator-add-member-epoch-activates` | RT-B-04 | 원본이 보던 **에폭 경계에서 검증자 집합이 커지는지**를 되살렸다. 승격 대상은 살아 있는 EN 노드 자신(`istanbul_nodeAddress`)이라 실제로 합의에 들어간다. 에폭 길이는 genesis overlay 로 10블록으로 줄였다 |

## 5. 이번에 추가한 DSL 문법

옮기지 못하던 테스트를 표현하려고 세 가지를 더했다.

| 문법 | 뜻 | 구현 |
|---|---|---|
| `{"do":"signAuthorization","authorityKey":…,"delegate":…,"save":…}` | EIP-7702 인증 튜플에 서명만 하고 보내지 않는다. `eth_estimateGas` 의 `authorizationList` 에 넣을 수 있다 | `internal/accounts.Wallet.SignAuthorization`, `internal/testhelper/txprobe.go` |
| `{"do":"readNodeLog","on":…,"maxBytes":…,"save":…}` | 특정 노드가 남긴 stdout/stderr 의 끝부분을 읽는다 | `interp.NodeLogReader`, `testengine.workspaceNodes.Log`, `internal/testhelper/fault.go` |
| `{"do":"swapNode"…,"expect":"fail","reason":…}` | 이 기동은 실패해야 한다. RPC 가 응답하지 않는 것을 확인하고, 런처 에러와 노드 로그를 합쳐 `reason` 을 맞춘다 | `internal/testhelper/fault.go` |
| `{"expect":"blockStalled","on":…,"timeout":…}` | `blockAdvance` 의 반대. 창 내내 head 가 움직이지 않아야 통과한다 | `internal/testhelper/read.go` |
| `{"do":"swapNode"…,"genesisOverlay":{…}}` | 네트워크 genesis 에 조각을 덮어 **그 노드만** 다시 init 한다. 한 대에만 다른 genesis 를 줄 수 있다 | `chainsetup.SwapNodeOpts`, `interp.NodeChange.GenesisOverlay` |

`expect: "fail"` 은 `sendTx` 의 `expect: "reject"` 와 같은 모양이다. 실패를 기대하는
표현이 트랜잭션과 노드 기동 두 곳에서 같게 읽힌다.

## 6. env 참조 규칙

케이스는 `"env": "<id>"` 로 환경을 부른다. `internal/dsl.ReadFiles` 가 케이스 파일이 있는 디렉터리부터 위로 올라가며 `<id>.env.json` 과 `env/<id>.env.json` 을 찾는다. 그래서 `tests/tc/env/` 하나로 모든 깊이의 케이스가 같은 환경 선언을 공유한다.

## 7. 함께 있는 문서

- `SPECS.md` — 스펙 이관 기록 (레거시 시절 `tests/specs/README.md`)
- `CHAIN-BRINGUP.md` — 체인 구성 케이스 설명 (레거시 시절 `tests/cases/README.md`)
- `../../docs/dev/legacy-port-audit/` — 포팅 감사 (그래프 2종 + 대응표)
