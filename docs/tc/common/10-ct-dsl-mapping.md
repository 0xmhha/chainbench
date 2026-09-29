# CT ↔ DSL 파일 매핑

> 근거: 이 저장소의 `docs/tc/common/01-common-test-list.md`(문서화된 70개 CT)와
> `tests/tc/common/` 아래 실제 DSL 파일(99개)을 대조해 만든 파생 문서다.
> 대조 기준일: 2026-09-29. 파일이 추가·삭제되면 이 표도 다시 맞춰야 한다.

---

## 1. 개요

문서에는 세 체인 공통 테스트가 **70개(CT)** 있고, `tests/tc/common/` 아래에는 실행용
DSL 파일이 **99개** 있다. "70개를 세 체인용으로 각각 만들었다면 210개여야 한다"는
예상과 숫자가 다른 이유는 chainbench DSL 이 **체인-파라미터 방식**이기 때문이다.

공통 DSL 한 파일은 `stablenet-*` 프리셋을 기준으로 작성하고, 실행할 때
`--chain-preset <wbft/wemix 프리셋> --binary <해당 체인 바이너리>` 로 갈아 끼워 나머지
두 체인에서도 그대로 돌린다. 그래서 대부분의 CT 는 파일 하나로 세 체인을 덮는다.
파일을 체인별로 물리적으로 나눈 경우는 **런타임 프리셋 교체로는 공유할 수 없는**
소수의 CT(합의 방식·바이너리 동작이 갈리는 것)뿐이다.

정리하면 99 = **체인 이름이 붙은 물리 분리 파일 15개** + **공통 파일 84개**(§4 참조).

한편 한 CT 에 파일이 여러 개인 경우도 있다. 예전부터 있던 자동 테스트(실행 이름이 여럿)를
CT 하나로 묶었기 때문이며, 묶인 원본 실행 이름은 01 문서의 "비고"에 모두 적혀 있다.
이 매핑 표는 그 실행 이름을 실제 DSL 파일에 이어 붙인 것이다.

---

## 2. 영역별 요약

| 영역 | CT 수 | DSL 파일 수 | 파일이 CT 수보다 많은 이유 |
| --- | --- | --- | --- |
| NODE | 15 | 25 | 001·002 체인 3분할, 003/008/009/010/014/016 이 CT당 2파일 |
| TX | 20 | 24 | 001·015 이 CT당 3파일 |
| FEE | 7 | 12 | 002 가 경계값 형식별 6파일 |
| CONTRACT | 7 | 12 | 001(3)·006(4) 다파일 |
| RPC | 15 | 17 | 001·007·009 가 CT당 2파일, 008 은 전용 파일 없음 |
| FAULT | 6 | 9 | 001 이 4파일 |
| **합계** | **70** | **99** | orphan(문서에 없는) 파일 0 |

---

## 3. 영역별 매핑

각 행은 `CT ID → tests/tc/common/<영역>/ 아래 파일`. 파일명은 디렉터리 안 기준.

### NODE (25파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-NODE-001 | `001-stablenet-chain-up.json`, `001-wbft-chain-up.json`, `001-wemix-chain-up.json` | 체인 3분할. wbft/wemix 는 binvar 프리셋(env로 바이너리 지정) |
| CT-NODE-002 | `002-stablenet-chain-up-15.json`, `002-wbft-chain-up-15.json`, `002-wemix-chain-up-15.json` | 체인 3분할 |
| CT-NODE-003 | `003-genesis-block-hash-consistent.json`, `003-chain-id.json` | |
| CT-NODE-004 | `004-full-sync.json` | 신규(2026-09-29) |
| CT-NODE-005 | `005-snap-sync.json` | 신규(2026-09-29) |
| CT-NODE-006 | `006-downloader-catch-up.json` | 신규(2026-09-29) |
| CT-NODE-007 | `007-live-block-receive.json` | 신규(2026-09-29) |
| CT-NODE-008 | `008-basic-peers.json`, `008-admin-peers-populated.json` | |
| CT-NODE-009 | `009-basic-consensus.json`, `009-basic-sync.json` | |
| CT-NODE-010 | `010-proxied-pn-routing.json`, `010-wbft-proxied-routing.json` | wbft 물리 분리 |
| CT-NODE-011 | `011-e1-mixed-producers.json` | |
| CT-NODE-013 | `013-genesis-mismatch.json` | |
| CT-NODE-014 | `014-chain-not-syncing.json`, `014-remote-chain-info.json` | |
| CT-NODE-015 | `015-block-timestamp-monotonic.json` | 신규(2026-09-29) |
| CT-NODE-016 | `016-block-period-one-second.json`, `016-stress-block-time.json` | |

### TX (24파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-TX-001 | `001-basic-tx-send.json`, `001-sample-minimal.json`, `001-value-transfer.json` | |
| CT-TX-002 | `002-legacy-value-transfer.json` | 신규. stablenet 전용 `legacy-transfer`는 공통 아님(제외) |
| CT-TX-003 | `003-dynamic-fee-transfer.json` | 신규. stablenet 전용 `dynamic-fee-tx`는 제외 |
| CT-TX-004 | `004-access-list-tx.json` | |
| CT-TX-005 | `005-fee-delegated-transfer.json` | |
| CT-TX-006 | `006-fd-sender-sig-invalid-rejected.json` | |
| CT-TX-007 | `007-fd-feepayer-sig-invalid-rejected.json` | |
| CT-TX-008 | `008-feepayer-insufficient-rejected.json` | |
| CT-TX-009 | `009-fee-delegate-sign-rpc-present.json` | |
| CT-TX-010 | `010-fee-delegated-access-list.json` | 신규(2026-09-29) |
| CT-TX-011 | `011-keystore-fee-delegate-sign.json` | 신규(2026-09-29) |
| CT-TX-012 | `012-nonce-ordering.json` | |
| CT-TX-013 | `013-replacement-tx.json` | |
| CT-TX-014 | `014-carry-over-and-replace.json` | 신규(2026-09-29) |
| CT-TX-015 | `015-insufficient-funds-rejected.json`, `015-wbft-insufficient-funds-rejected.json`, `015-wemix-insufficient-funds-rejected.json` | 체인 3분할(자동 테스트가 이미 셋) |
| CT-TX-016 | `016-gas-limit-exceeds-block-rejected.json` | stablenet 전용 `gaslimit-exceeded-rejected`는 제외 |
| CT-TX-017 | `017-reject-vs-execution-failure.json` | 신규(2026-09-29) |
| CT-TX-018 | `018-basic-txpool-propagation.json` | |
| CT-TX-019 | `019-stress-tx-flood.json` | |
| CT-TX-020 | `020-faucet-funds-account.json` | |

### FEE (12파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-FEE-001 | `001-tip-below-min-rejected.json` | |
| CT-FEE-002 | `002-legacy-gasprice-below-min-rejected.json`, `002-accesslist-gasprice-below-min-rejected.json`, `002-feecap-below-min-rejected.json`, `002-legacy-gasprice-above-min-accepted.json`, `002-accesslist-gasprice-above-min-accepted.json`, `002-dynamic-feecap-above-min-accepted.json` | 형식별(legacy/accesslist/dynamic) × (미만 거부/초과 수용). above-min 3개는 신규. stablenet 전용 `feecap-above-min-accepted`·`feecap-exact-min-accepted`는 제외 |
| CT-FEE-007 | `007-effective-gas-price.json` | |
| CT-FEE-008 | `008-effective-gas-price-regular-bp-en.json` | |
| CT-FEE-009 | `009-snap-receipt-gas-price.json` | 신규(2026-09-29) |
| CT-FEE-010 | `010-gas-price-positive.json` | stablenet 전용 `gas-price-equals-basefee-plus-tip`는 제외 |
| CT-FEE-012 | `012-fee-history-well-formed.json` | |

### CONTRACT (12파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-CONTRACT-001 | `001-contract-roundtrip.json`, `001-wbft-tx-and-contract.json`, `001-wemix-tx-and-contract.json` | wbft/wemix 물리 분리 |
| CT-CONTRACT-002 | `002-storage-write-and-read.json` | |
| CT-CONTRACT-003 | `003-view-call-leaves-state.json` | 신규(2026-09-29) |
| CT-CONTRACT-004 | `004-estimate-gas.json` | |
| CT-CONTRACT-005 | `005-eth-call-revert-returns-error.json` | |
| CT-CONTRACT-006 | `006-revert-tx-status-zero.json`, `006-negative-tx-revert.json`, `006-wbft-revert-status-zero.json`, `006-wemix-revert-status-zero.json` | wbft/wemix 물리 분리 |
| CT-CONTRACT-007 | `007-out-of-gas-consumes-all.json` | |

### RPC (17파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-RPC-001 | `001-basic-rpc-health.json`, `001-remote-rpc-health.json` | |
| CT-RPC-002 | `002-block-transactions-field.json` | |
| CT-RPC-003 | `003-block-by-hash-consistency.json` | |
| CT-RPC-004 | `004-transaction-by-hash-fields.json` | |
| CT-RPC-005 | `005-transaction-receipt-fields.json` | |
| CT-RPC-006 | `006-transaction-count-increments.json` | |
| CT-RPC-007 | `007-genesis-balance.json`, `007-remote-balance-check.json` | |
| CT-RPC-008 | (전용 파일 없음) | `node/003-chain-id.json`, `node/014-remote-chain-info.json` 와 목적이 겹쳐 공유 |
| CT-RPC-009 | `009-logs-query-well-formed.json`, `009-contract-event-emitted.json` | |
| CT-RPC-010 | `010-signed-tx-seen-in-pool.json` | 신규(2026-09-29) |
| CT-RPC-011 | `011-txpool-status.json` | |
| CT-RPC-012 | `012-txpool-content-well-formed.json` | |
| CT-RPC-013 | `013-ws-subscribe-new-heads.json` | |
| CT-RPC-014 | `014-ws-subscribe-logs.json` | |
| CT-RPC-015 | `015-metric-head-block.json` | |

### FAULT (9파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-FAULT-001 | `001-fault-node-crash.json`, `001-sample-lifecycle.json`, `001-wbft-node-crash.json`, `001-wemix-node-crash.json` | wbft/wemix 물리 분리 |
| CT-FAULT-002 | `002-fault-node-recover.json` | |
| CT-FAULT-003 | `003-fault-two-down.json` | |
| CT-FAULT-004 | `004-fault-network-partition.json` | target:remote — 방화벽으로 분리하므로 로컬 실행은 건너뛴다 |
| CT-FAULT-005 | `005-fault-p2p-topology.json` | |
| CT-FAULT-006 | `006-fault-txpool-leader-change.json` | |

---

## 4. 체인별 물리 분리 파일 (15개)

런타임 프리셋 교체로 공유하지 못해 체인 이름을 붙여 따로 둔 파일이다. 나머지 84개는
`stablenet-*` 프리셋 기준으로 쓰고 실행 시 프리셋·바이너리를 갈아 세 체인을 덮는다.

| CT | stablenet(공통) | wbft | wemix |
| --- | --- | --- | --- |
| CT-NODE-001 | `node/001-stablenet-chain-up.json` | `node/001-wbft-chain-up.json` | `node/001-wemix-chain-up.json` |
| CT-NODE-002 | `node/002-stablenet-chain-up-15.json` | `node/002-wbft-chain-up-15.json` | `node/002-wemix-chain-up-15.json` |
| CT-NODE-010 | `node/010-proxied-pn-routing.json` | `node/010-wbft-proxied-routing.json` | (공통 파일로 덮음) |
| CT-TX-015 | `tx/015-insufficient-funds-rejected.json` | `tx/015-wbft-insufficient-funds-rejected.json` | `tx/015-wemix-insufficient-funds-rejected.json` |
| CT-CONTRACT-001 | `contract/001-contract-roundtrip.json` | `contract/001-wbft-tx-and-contract.json` | `contract/001-wemix-tx-and-contract.json` |
| CT-CONTRACT-006 | `contract/006-revert-tx-status-zero.json` (+`006-negative-tx-revert.json`) | `contract/006-wbft-revert-status-zero.json` | `contract/006-wemix-revert-status-zero.json` |
| CT-FAULT-001 | `fault/001-fault-node-crash.json` (+`001-sample-lifecycle.json`) | `fault/001-wbft-node-crash.json` | `fault/001-wemix-node-crash.json` |

체인 이름이 붙은 파일만 세면 wbft 7개 + wemix 6개 + stablenet-접두 2개(node/001·002) = 15개다.

---

## 5. 특이사항

- **CT-RPC-008(체인 ID 조회)** 은 전용 DSL 파일이 없다. `node/003-chain-id.json`,
  `node/014-remote-chain-info.json` 와 목적이 같아 그 파일을 공유한다.
- **stablenet 전용으로 분류돼 공통에서 빠진 자동 테스트**: `legacy-transfer`,
  `dynamic-fee-tx`, `gaslimit-exceeded-rejected`, `feecap-above-min-accepted`,
  `feecap-exact-min-accepted`, `gas-price-equals-basefee-plus-tip`.
  이들은 StableNet 전용 `gasTip` 을 읽어 세 체인
  공통이 아니므로 `tests/tc/common/` 에는 없다(01 문서 각 CT 비고 참조).
- **binvar 예외**: `node/001-wbft-chain-up.json`·`node/001-wemix-chain-up.json` 두 개만
  `--binary` 대신 `GWBFT_BIN`/`GWEMIX_BIN` 환경변수로 바이너리를 받는다.
- 이 매핑의 파일별 실행 명령은 `tests/tc/common/HOW-TO-USE.md` §9 에 정리돼 있다.
