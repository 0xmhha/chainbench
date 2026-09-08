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
└── env/                    ← 케이스가 id 로 참조하는 환경 선언
```

파일명은 `<레거시 번호>-<테스트 id>.json` 이다. 번호는 레거시 스위트의 순번을 그대로 가져와 대조가 되게 했고, 뒤의 id 가 무엇을 검증하는지 말한다. 레거시 하나가 여러 스펙으로 나뉜 경우 `11-`, `11b-` 처럼 뒤에 글자를 붙였다.

번호가 비어 있는 자리는 그 레거시 테스트가 DSL 이 아니라 `tests/e2e/` 의 Go 테스트로 옮겨졌거나, 아직 옮겨지지 않은 자리다. 3절과 4절에 그 목록이 있다.

## 2. 디렉터리별 내용

### `basic` (7)

| 파일 | 레거시 대응 |
|---|---|
| `01-basic-consensus.json` | `basic/consensus` |
| `02-basic-peers.json` | `regression/ethereum/05-test-p2p-peers`, `regression/api/17-test-net-peer-count`, `basic/peers` |
| `03-basic-rpc-health.json` | `regression/ethereum/26-test-eth-block-number`, `basic/rpc-health` |
| `04-basic-sync.json` | `basic/sync` |
| `05-basic-tx-send.json` | `basic/tx-send` |
| `06-basic-txpool-propagation.json` | `basic/txpool-propagation` |
| `07-basic-wbft-consensus.json` | `basic/wbft-consensus` |

### `fault` (6)

| 파일 | 레거시 대응 |
|---|---|
| `01-fault-network-partition.json` | `fault/network-partition` |
| `02-fault-node-crash.json` | `fault/node-crash` |
| `03-fault-node-recover.json` | `regression/ethereum/04-test-node-restart`, `fault/node-recover` |
| `04-fault-p2p-topology.json` | `fault/p2p-topology` |
| `05-fault-two-down.json` | `fault/two-down` |
| `06-fault-txpool-leader-change.json` | `fault/txpool-leader-change` |

### `go-stablenet/post-v1.0.0-change/common-all` (18)

| 파일 | 레거시 대응 |
|---|---|
| `01-stablenet-delayed-fork.json` | `post-v1.0.0-change/common-all/01-test-inject-contracts-delayed`, `post-v1.0.0-change/extra-state/07-test-extra-v1-to-v2-delayed` |
| `02-govminter-v2-code.json` | `post-v1.0.0-change/common-all/02-test-v2-code-only-hash`, `post-v1.0.0-change/common-all/10-test-govminter-bytecode-hash-state-preserved` |
| `03-burn-cancel-refundable.json` | `post-v1.0.0-change/common-all/03-test-burn-cancel-refund` |
| `04-burn-reject-refundable.json` | `post-v1.0.0-change/common-all/04-test-burn-reject-refund` |
| `05-burn-expire-refundable.json` | `post-v1.0.0-change/common-all/05-test-burn-expire-refund` (신규 작성) |
| `06-burn-execute-no-refundable.json` | `post-v1.0.0-change/common-all/06-test-burn-execute-no-refund` |
| `07-claim-burn-refund-succeeds.json` | `post-v1.0.0-change/common-all/07-test-claim-refund-success` |
| `08-claim-zero-refund-reverts.json` | `post-v1.0.0-change/common-all/08-test-claim-refund-zero-revert` |
| `09-claim-burn-refund-double-reverts.json` | `post-v1.0.0-change/common-all/09-test-claim-refund-double-revert` |
| `10-prealloc-preserved-across-boho.json` | `post-v1.0.0-change/common-all/10-test-govminter-bytecode-hash-state-preserved` |
| `12-legacy-gasprice-below-min-rejected.json` | `post-v1.0.0-change/common-all/12-test-legacy-gasprice-below-min-revert` |
| `13-accesslist-gasprice-below-min-rejected.json` | `post-v1.0.0-change/common-all/13-test-accesslist-gasprice-below-min-revert` |
| `14-feecap-below-min-rejected.json` | `post-v1.0.0-change/common-all/14-test-dynamic-fee-tipcap-below-min-revert` |
| `15-boho-chain-config-active.json` | `post-v1.0.0-change/common-all/15-test-chain-config-boho` |
| `16-anzeon-active-before-boho.json` | `post-v1.0.0-change/common-all/16-test-boho-chain-config-activation` |
| `19-upgrade-registry-order.json` | `post-v1.0.0-change/common-all/19-test-upgrade-registry` (신규 작성) |
| `20-v1-params-init-storage.json` | `post-v1.0.0-change/common-all/20-test-v1-params-init` (신규 작성) |
| `21-burn-refund-events.json` | 신규(레거시 대응 없음) |

### `go-stablenet/post-v1.0.0-change/effectivegasprice` (1)

| 파일 | 레거시 대응 |
|---|---|
| `02-effective-gas-price-regular.json` | `post-v1.0.0-change/effectivegasprice/02-test-regular-account` |

### `go-stablenet/post-v1.0.0-change/extra-state` (6)

| 파일 | 레거시 대응 |
|---|---|
| `01-authorized-extra-bit-synced.json` | `post-v1.0.0-change/extra-state/01-test-extra-alloc-to-contract` |
| `01b-blacklisted-extra-bit-synced.json` | `post-v1.0.0-change/extra-state/01-test-extra-alloc-to-contract` |
| `02-stablenet-account-extra.json` | `post-v1.0.0-change/extra-state/02-test-extra-contract-to-alloc` |
| `03-extra-union-merge.json` | `post-v1.0.0-change/extra-state/03-test-extra-union-merge` (신규 작성) |
| `04-dual-status-extra.json` | `post-v1.0.0-change/extra-state/04-test-extra-dual-status` |
| `05-extra-balance-preserved.json` | `post-v1.0.0-change/extra-state/05-test-extra-balance-preserved` |

### `go-stablenet/post-v1.0.0-change/stand-alone` (1)

| 파일 | 레거시 대응 |
|---|---|
| `04-genesis-block-hash-consistent.json` | `post-v1.0.0-change/stand-alone/04-test-testnet-genesis-hash` (신규 작성) |

### `go-stablenet/post-v1.0.0-change/string-handling` (6)

| 파일 | 레거시 대응 |
|---|---|
| `01-authorized-accounts-no-space.json` | `post-v1.0.0-change/string-handling/01-test-authorized-no-space` (신규 작성) |
| `02-authorized-accounts-space.json` | `post-v1.0.0-change/string-handling/02-test-authorized-space` (신규 작성) |
| `03-authorized-accounts-trim.json` | `post-v1.0.0-change/string-handling/03-test-authorized-trim` (신규 작성) |
| `04-authorized-accounts-empty-item.json` | `post-v1.0.0-change/string-handling/04-test-authorized-empty-item` (신규 작성) |
| `05-authorized-accounts-single.json` | `post-v1.0.0-change/string-handling/05-test-authorized-single` (신규 작성) |
| `06-authorized-accounts-empty.json` | `post-v1.0.0-change/string-handling/06-test-authorized-empty` (신규 작성) |

### `go-stablenet/regression/anzeon` (10)

| 파일 | 레거시 대응 |
|---|---|
| `01-regular-account-gastip-forced.json` | `regression/anzeon/01-test-regular-account-gastip-forced` |
| `02-authorized-account-gastip-free.json` | `regression/anzeon/02-test-authorized-account-gastip-free`, `post-v1.0.0-change/effectivegasprice/01-test-authorized-account` |
| `03-anzeon-basefee-increase.json` | `regression/anzeon/03-test-basefee-increase` |
| `04-anzeon-basefee-stable.json` | `regression/anzeon/04-test-basefee-stable` |
| `05-anzeon-basefee-decrease.json` | `regression/anzeon/05-test-basefee-decrease` |
| `06-basefee-minimum.json` | `regression/anzeon/06-test-min-basefee` |
| `07-basefee-maximum.json` | `regression/anzeon/07-test-max-basefee` |
| `08-feecap-above-min-accepted.json` | 신규(레거시 대응 없음) |
| `09-feecap-exact-min-accepted.json` | 신규(레거시 대응 없음) |
| `11-gaslimit-exceeded-rejected.json` | 신규(레거시 대응 없음) |

### `go-stablenet/regression/api` (25)

| 파일 | 레거시 대응 |
|---|---|
| `01-block-transactions-field.json` | `regression/api/01-test-get-block-by-number` |
| `02-block-by-hash-consistency.json` | `regression/api/02-test-get-block-by-hash` |
| `03-transaction-by-hash-fields.json` | `regression/api/03-test-get-tx-by-hash` |
| `04-transaction-receipt-fields.json` | `regression/api/04-test-get-tx-receipt` |
| `05-transaction-count-increments.json` | `regression/api/05-test-get-tx-count` |
| `06-system-contracts-deployed.json` | `regression/api/06-test-get-code-system` |
| `07-gas-price-positive.json` | `regression/api/07-test-gas-price` |
| `07b-gas-price-equals-basefee-plus-tip.json` | `regression/api/07-test-gas-price` |
| `08-max-priority-fee-equals-gastip.json` | `regression/api/08-test-max-priority-fee` |
| `09-fee-history-well-formed.json` | `regression/api/09-test-fee-history` |
| `10-estimate-gas-token-transfer.json` | `regression/api/10-test-estimate-system-call` |
| `11-node-address-returned.json` | `regression/api/11-test-node-address` |
| `12-validator-set-nonempty.json` | `regression/api/12-test-get-validators`, `regression/wbft/07-test-istanbul-get-validators` |
| `12b-validator-set-count.json` | `regression/api/12-test-get-validators` |
| `13-commit-signers-quorum.json` | `regression/api/13-test-get-commit-signers` |
| `14-wbft-extra-info-fields.json` | `regression/api/14-test-get-wbft-extra` |
| `15-istanbul-status-fields.json` | `regression/api/15-test-istanbul-status` |
| `16-is-validator-flags.json` | `regression/api/16-test-is-validator` |
| `18-txpool-status.json` | `regression/api/18-test-txpool-status` |
| `19-txpool-content-well-formed.json` | `regression/api/19-test-txpool-content` |
| `20-admin-peers-populated.json` | `regression/ethereum/05-test-p2p-peers`, `regression/api/20-test-admin-peers` |
| `21-fee-delegate-sign-rpc-present.json` | `regression/api/21-test-sign-raw-fee-delegate` |
| `22-token-total-supply-readable.json` | `regression/api/22-test-total-supply` |
| `23-token-approve-sets-allowance.json` | `regression/api/23-test-allowance`, `regression/system-contracts/03-test-approve-transferfrom` |
| `24-chain-not-syncing.json` | 신규(레거시 대응 없음) |

### `go-stablenet/regression/blacklist-authorized` (9)

| 파일 | 레거시 대응 |
|---|---|
| `01-sender-blacklisted-rejected.json` | `regression/blacklist-authorized/01-test-sender-blacklisted` |
| `02-recipient-blacklisted-rejected.json` | `regression/blacklist-authorized/02-test-recipient-blacklisted` |
| `03-feepayer-blacklisted-rejected.json` | `regression/blacklist-authorized/03-test-feepayer-blacklisted` |
| `04-address-unblacklisted-event.json` | `regression/blacklist-authorized/04-test-unblacklist`, `regression/system-contracts/17-test-unblacklist`, `regression/system-contracts/22-test-address-unblacklisted-event` |
| `05-zero-address-transfer-rejected.json` | `regression/blacklist-authorized/05-test-zero-address` (신규 작성) |
| `06-precompile-transfer-rejected.json` | `regression/blacklist-authorized/06-test-precompile-transfer` (신규 작성) |
| `07-account-blacklist-readable.json` | `regression/blacklist-authorized/07-test-is-blacklisted` |
| `08-account-authorization-readable.json` | `regression/blacklist-authorized/08-test-is-authorized` |
| `09-authorized-tx-executed-event.json` | `regression/blacklist-authorized/09-test-authorized-tx-executed`, `post-v1.0.0-change/effectivegasprice/03-test-auth-tx-event-last` |

### `go-stablenet/regression/ethereum` (26)

| 파일 | 레거시 대응 |
|---|---|
| `01-stablenet-chain-up.json` | `regression/ethereum/01-test-genesis-init` |
| `08-legacy-transfer.json` | `regression/ethereum/08-test-legacy-tx` |
| `09-dynamic-fee-tx.json` | `regression/ethereum/09-test-dynamic-fee-tx` |
| `10-access-list-tx.json` | `regression/ethereum/10-test-access-list-tx` |
| `11-nonce-ordering.json` | `regression/ethereum/11-test-nonce-ordering` |
| `11b-out-of-order-nonces-mine.json` | `regression/ethereum/11-test-nonce-ordering` |
| `12-dynamic-fee-below-basefee-rejected.json` | `regression/ethereum/12-test-tipcap-underpriced`, `regression/ethereum/13-test-feecap-underpriced` |
| `14-insufficient-funds-rejected.json` | `regression/ethereum/14-test-insufficient-funds` |
| `15-gas-limit-exceeds-block-rejected.json` | `regression/ethereum/15-test-gaslimit-exceeded` |
| `16-effective-gas-price.json` | `regression/ethereum/16-test-effective-gas-price`, `post-v1.0.0-change/effectivegasprice/02-test-regular-account` |
| `17-replacement-tx.json` | `regression/ethereum/17-test-replacement-tx` |
| `17b-same-nonce-replacement.json` | `regression/ethereum/17-test-replacement-tx` |
| `18-set-code-delegation.json` | `regression/ethereum/18-test-setcode-tx` |
| `19-contract-roundtrip.json` | `regression/ethereum/19-test-contract-deploy`, `regression/ethereum/20-test-contract-call`, `regression/ethereum/21-test-eth-call-view` |
| `22-estimate-gas.json` | `regression/ethereum/22-test-estimate-gas`, `regression/api/10-test-estimate-system-call`, `post-v1.0.0-change/common-all/18-test-estimategas-no-authorizationlist` |
| `23-eth-call-revert-returns-error.json` | `regression/ethereum/23-test-eth-call-revert` |
| `24-revert-tx-status-zero.json` | `regression/ethereum/24-test-revert-tx` |
| `25-out-of-gas-consumes-all.json` | `regression/ethereum/25-test-out-of-gas` |
| `27-genesis-balance.json` | `regression/ethereum/27-test-eth-get-balance` |
| `27b-value-transfer.json` | `regression/ethereum/27-test-eth-get-balance`, `regression/ethereum/28-test-send-raw-tx` |
| `29-logs-query-well-formed.json` | `regression/ethereum/29-test-eth-get-logs` |
| `30-chain-id.json` | `regression/ethereum/01-test-genesis-init`, `regression/ethereum/30-test-eth-chain-id` |
| `31-ws-subscribe-new-heads.json` | `regression/ethereum/31-test-ws-subscribe-heads` |
| `32-ws-subscribe-logs.json` | `regression/ethereum/32-test-ws-subscribe-logs` |
| `33-stablenet-chain-up-15.json` | 신규(레거시 대응 없음) |
| `36-contract-event-emitted.json` | 신규(레거시 대응 없음) |

### `go-stablenet/regression/fee-delegation` (7)

| 파일 | 레거시 대응 |
|---|---|
| `01-fee-delegated-transfer.json` | `regression/fee-delegation/01-test-fee-delegate-normal` |
| `02-fd-sender-sig-invalid-rejected.json` | `regression/fee-delegation/02-test-sender-sig-invalid` |
| `03-fd-feepayer-sig-invalid-rejected.json` | `regression/fee-delegation/03-test-feepayer-sig-invalid` |
| `04-feepayer-insufficient-rejected.json` | `regression/fee-delegation/04-test-feepayer-insufficient` |
| `05-fee-delegated-sender-sig-invalid-rejected.json` | 신규(레거시 대응 없음) |
| `06-fee-delegated-feepayer-sig-invalid-rejected.json` | 신규(레거시 대응 없음) |
| `07-fee-delegated-unfunded-feepayer-rejected.json` | 신규(레거시 대응 없음) |

### `go-stablenet/regression/system-contracts` (23)

| 파일 | 레거시 대응 |
|---|---|
| `01-native-coin-adapter-code.json` | `regression/system-contracts/01-test-native-transfer` |
| `01b-token-transfer-emits-event.json` | `regression/system-contracts/01-test-native-transfer` |
| `02-token-balance-readable.json` | `regression/system-contracts/02-test-balance-of` |
| `03-token-transfer-from-moves-balance.json` | `regression/system-contracts/03-test-approve-transferfrom` |
| `04-mint-transfer-event.json` | `regression/system-contracts/04-test-mint-transfer-event` |
| `05-burn-transfer-event.json` | `regression/system-contracts/05-test-burn-transfer-event` |
| `06-mint-proposal-executes.json` | `regression/system-contracts/06-test-mint-proposal` |
| `07-burn-proposal-executes.json` | `regression/system-contracts/07-test-burn-proposal` |
| `08-quorum-deficient-stays-voting.json` | `regression/system-contracts/08-test-quorum-deficient` |
| `09-validator-metadata-readable.json` | `regression/system-contracts/09-test-validator-metadata` |
| `10-gastip-governance-updates-header.json` | `regression/wbft/06-test-gastip-header-sync`, `regression/system-contracts/10-test-propose-gastip` |
| `11-proposal-expiry-transitions.json` | `regression/system-contracts/11-test-proposal-expiry` |
| `12-configure-minter-proposal-executes.json` | `regression/system-contracts/12-test-add-minter` |
| `13-remove-minter-executes.json` | `regression/system-contracts/13-test-remove-minter` |
| `14-masterminter-member-add-remove.json` | `regression/system-contracts/14-test-masterminter-self-member` |
| `15-non-member-configure-minter-rejected.json` | `regression/system-contracts/15-test-non-member-rejected` |
| `16-blacklist-proposal-executes.json` | `regression/system-contracts/16-test-blacklist`, `regression/system-contracts/21-test-address-blacklisted-event` |
| `18-authorize-proposal-executes.json` | `regression/system-contracts/18-test-authorize`, `regression/system-contracts/19-test-unauthorize`, `regression/system-contracts/24-test-authorized-account-removed-event` |
| `19-unauthorize-proposal-executes.json` | `regression/system-contracts/19-test-unauthorize`, `regression/system-contracts/24-test-authorized-account-removed-event` |
| `20-direct-blacklist-call-rejected.json` | `regression/system-contracts/20-test-direct-blacklist-rejected` |
| `23-authorized-account-added-event.json` | `regression/system-contracts/23-test-authorized-account-added-event` |
| `25-token-metadata.json` | 신규(레거시 대응 없음) |
| `28-minter-status-readable.json` | 신규(레거시 대응 없음) |

### `go-stablenet/regression/wbft` (8)

| 파일 | 레거시 대응 |
|---|---|
| `01-block-period-one-second.json` | `regression/wbft/01-test-block-period` |
| `02-wbft-seals-quorum.json` | `regression/wbft/02-test-wbft-extra-seal` |
| `03-epoch-transition-carries-epoch-info.json` | `regression/wbft/03-test-epoch-transition` |
| `04-validator-add-member-executes.json` | `regression/wbft/04-test-add-validator` |
| `05-validator-remove-member-executes.json` | `regression/wbft/05-test-remove-validator` (신규 작성) |
| `11-prev-seals-quorum.json` | `regression/wbft/11-test-prev-committed-seal`, `regression/wbft/12-test-prev-prepared-seal` |
| `13-randao-and-mixdigest-present.json` | 신규(레거시 대응 없음) |
| `14-stablenet-gastip-field.json` | 신규(레거시 대응 없음) |

### `go-wbft/accounts` (3)

| 파일 | 레거시 대응 |
|---|---|
| `01-secp256r1-precompile-valid.json` | 신규(레거시 대응 없음) |
| `02-secp256r1-precompile-invalid.json` | 신규(레거시 대응 없음) |
| `03-secp256r1-precompile-short-input.json` | 신규(레거시 대응 없음) |

### `go-wbft/chain-up` (2)

| 파일 | 레거시 대응 |
|---|---|
| `01-wbft-chain-up.json` | 신규(레거시 대응 없음) |
| `02-wbft-chain-up-15.json` | 신규(레거시 대응 없음) |

### `go-wbft/consensus` (1)

| 파일 | 레거시 대응 |
|---|---|
| `01-e1-mixed-producers.json` | 신규(레거시 대응 없음) |

### `go-wemix/chain-up` (2)

| 파일 | 레거시 대응 |
|---|---|
| `01-wemix-chain-up.json` | 신규(레거시 대응 없음) |
| `02-wemix-chain-up-15.json` | 신규(레거시 대응 없음) |

### `go-wemix/handoff` (1)

| 파일 | 레거시 대응 |
|---|---|
| `01-wemix-wbft-handoff.json` | 신규(레거시 대응 없음) |

### `go-wemix/rpc` (1)

| 파일 | 레거시 대응 |
|---|---|
| `01-wemix-brioche-block-reward.json` | 신규(레거시 대응 없음) |

### `go-wemix/tx` (1)

| 파일 | 레거시 대응 |
|---|---|
| `01-wemix-tx-and-contract.json` | 신규(레거시 대응 없음) |

### `stress` (2)

| 파일 | 레거시 대응 |
|---|---|
| `01-stress-block-time.json` | `stress/block-time` |
| `02-stress-tx-flood.json` | `stress/tx-flood` |

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
