# tests/tc/common — 세 체인에서 같은 목적으로 도는 테스트

Confluence [[Common] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987917348)
가 가려낸 공통 테스트 **CT 76개**의 자리다. 저장소에 옮겨 둔 사본은
[`docs/tc/common/`](../../../docs/tc/common/) 에 있고, 목록의 정본은 그쪽이다.

## 1. 무엇이 여기 있나

CT 76개 중 **자동 테스트가 이미 있는 64개**에 해당하는 케이스 **99개**다. CT 하나에
케이스가 여럿인 것은 지금 세 체인이 각자의 케이스를 갖고 있기 때문이다 — `CT-NODE-001`
은 `wemix-chain-up`·`wbft-chain-up`·`stablenet-chain-up` 셋이다.

```
tests/tc/common/
├── node/       21건   노드·동기화·네트워크
├── tx/         26건   트랜잭션 전송·거부
├── fee/        16건   수수료·가스 정책
├── contract/   11건   컨트랙트 실행
├── rpc/        16건   조회·구독 API
└── fault/       9건   장애·복구
```

파일명은 `<CT 번호>-<테스트 id>.json` 이다. 앞의 번호가 같은 파일들이 한 CT 를 이룬다.
번호 뒤의 이름은 옮기기 전과 **한 글자도 바꾸지 않았다** — `id` 로 케이스를 부르는 문서와
검사가 여럿이라, 이름을 같이 바꾸면 무엇이 깨졌는지 옮긴 것과 구별할 수 없다.

CT 하나를 두 영역이 함께 거명한 것이 둘 있다. `chain-id` 는 `CT-NODE-003` 과
`CT-RPC-008`, `remote-chain-info` 는 `CT-NODE-014` 와 `CT-RPC-008` 이다. 앞의 CT 를
따라 `node/` 에 두었다.

## 2. 왜 99개를 64개로 합치지 않았나

합치는 것이 목표지만 **한 번에 하지 않는다.** `CT-NODE-001` 의 세 케이스는 같은 것을
보는 듯하지만 기대값이 다르다 — 검증자 수를 확인하는 방법이 체인마다 다르고, WEMIX3.0
은 검증자 조회 자체가 없다. 지금 합치면 그 차이가 코드가 아니라 사람의 기억 속으로
들어간다.

그래서 이번 단계는 **자리만 옮긴다.** 케이스의 내용은 그대로이고, 세 체인의 케이스가
한 디렉터리에서 번호로 묶여 나란히 보이게 된 것이 전부다. 나란히 놓고 봐야 무엇이 같고
무엇이 다른지 셀 수 있다.

## 3. 아직 체인에 묶여 있다

옮긴 99건이 지금 이름 붙인 chain-preset 은 이렇다.

| 체인 | 건수 |
| --- | --- |
| stablenet | 86 |
| wbft | 7 |
| wemix | 6 |

디렉터리 의존은 없어졌지만 **preset 이름 의존은 남아 있다.** 다만 이것은 실행할 때
덮을 수 있다.

```sh
chainbench run tests/tc/common/tx/002-legacy-transfer.json \
  --workspace-dir ~/cbw/x --chain-preset wemix-bp4
```

`--chain-preset` 은 케이스가 무엇을 이름 붙였든 그 preset 으로 돌린다. 세 체인이 모두
갖춘 모양은 `bp4` 와 `bp7-en7-pn1` 둘이고, `presets/chain/` 에 세 벌씩 있다.

**세 체인에서 다 도는지는 아직 재지 않았다.** 그 측정이 다음 단계의 입력이다.

## 4. 여기 없는 것

CT 12개는 자동 테스트가 없어 옮길 것이 없었다.

`CT-NODE-004`·`005`·`006`·`007`(동기화 네 가지) · `CT-NODE-015` · `CT-TX-010`·`011`·
`014`·`017` · `CT-FEE-009` · `CT-CONTRACT-003` · `CT-RPC-010`

실행 도구에 없는 기능에 막힌 것이 섞여 있다. 무엇이 막고 있는지는
[`docs/tc/common/03-separate-implementation-items.md`](../../../docs/tc/common/03-separate-implementation-items.md) §2 와
[`docs/dev/architecture/mainnet-config-worklist.md` §8](../../../docs/dev/architecture/mainnet-config-worklist.md) 에 있다.

## 5. CT 와 케이스 대응

### NODE — CT 16개 중 11개가 케이스를 갖고 있다

| CT | 무엇을 보나 | 지금 있는 케이스 |
| --- | --- | --- |
| CT-NODE-001 | 노드 기동과 블록 생성 | `node/001-wemix-chain-up.json` · `node/001-wbft-chain-up.json` · `node/001-stablenet-chain-up.json` |
| CT-NODE-002 | 15노드 구성 기동 | `node/002-wemix-chain-up-15.json` · `node/002-wbft-chain-up-15.json` · `node/002-stablenet-chain-up-15.json` |
| CT-NODE-003 | 제네시스 초기화와 블록 0 해시 일치 | `node/003-genesis-block-hash-consistent.json` · `node/003-chain-id.json` |
| CT-NODE-004 | Full Sync 동기화 | **없다 — 새로 써야 한다** |
| CT-NODE-005 | Snap Sync 동기화 | **없다 — 새로 써야 한다** |
| CT-NODE-006 | 누락 블록 일괄 동기화 | **없다 — 새로 써야 한다** |
| CT-NODE-007 | 새 블록 실시간 수신 | **없다 — 새로 써야 한다** |
| CT-NODE-008 | 피어 연결과 피어 조회 | `node/008-basic-peers.json` · `node/008-admin-peers-populated.json` |
| CT-NODE-009 | 노드 간 최신 블록 해시 일치 | `node/009-basic-consensus.json` · `node/009-basic-sync.json` |
| CT-NODE-010 | 전달 노드 경유 동기화 | `node/010-proxied-pn-routing.json` · `node/010-wbft-proxied-routing.json` |
| CT-NODE-011 | 종단 노드가 먼저 오는 혼합 배치 | `node/011-e1-mixed-producers.json` |
| CT-NODE-012 | 노드 프로그램 교체 후 서명 호환 | `node/012-signature-compat-across-swap.json` |
| CT-NODE-013 | 제네시스 불일치 시 기동 거부 | `node/013-genesis-mismatch.json` |
| CT-NODE-014 | 동기화 완료 상태 확인 | `node/014-chain-not-syncing.json` · `node/014-remote-chain-info.json` |
| CT-NODE-015 | 블록 시각 단조 증가 | **없다 — 새로 써야 한다** |
| CT-NODE-016 | 블록 생성 주기 | `node/016-block-period-one-second.json` · `node/016-stress-block-time.json` |

### TX — CT 20개 중 16개가 케이스를 갖고 있다

| CT | 무엇을 보나 | 지금 있는 케이스 |
| --- | --- | --- |
| CT-TX-001 | 일반 송금 | `tx/001-basic-tx-send.json` · `tx/001-sample-minimal.json` · `tx/001-value-transfer.json` |
| CT-TX-002 | Legacy 트랜잭션 | `tx/002-legacy-transfer.json` |
| CT-TX-003 | 동적 수수료 트랜잭션 | `tx/003-dynamic-fee-tx.json` |
| CT-TX-004 | 접근 목록 트랜잭션 | `tx/004-access-list-tx.json` |
| CT-TX-005 | 수수료 대납 트랜잭션 | `tx/005-fee-delegated-transfer.json` |
| CT-TX-006 | 대납 트랜잭션의 보낸 이 서명 변조 거부 | `tx/006-fd-sender-sig-invalid-rejected.json` · `tx/006-fee-delegated-sender-sig-invalid-rejected.json` |
| CT-TX-007 | 대납 트랜잭션의 대납자 서명 변조 거부 | `tx/007-fd-feepayer-sig-invalid-rejected.json` · `tx/007-fee-delegated-feepayer-sig-invalid-rejected.json` |
| CT-TX-008 | 대납자 잔액 부족 거부 | `tx/008-feepayer-insufficient-rejected.json` · `tx/008-fee-delegated-unfunded-feepayer-rejected.json` |
| CT-TX-009 | 대납 서명 API 존재 | `tx/009-fee-delegate-sign-rpc-present.json` |
| CT-TX-010 | 접근 목록을 붙인 대납 트랜잭션 | **없다 — 새로 써야 한다** |
| CT-TX-011 | 노드 키 저장소 경유 대납 서명 | **없다 — 새로 써야 한다** |
| CT-TX-012 | 계정별 nonce 순서 보장 | `tx/012-nonce-ordering.json` · `tx/012-out-of-order-nonces-mine.json` |
| CT-TX-013 | 같은 nonce 트랜잭션 교체 | `tx/013-replacement-tx.json` · `tx/013-same-nonce-replacement.json` |
| CT-TX-014 | 미포함 트랜잭션의 이월과 교체 | **없다 — 새로 써야 한다** |
| CT-TX-015 | 잔액 부족 트랜잭션 거부 | `tx/015-insufficient-funds-rejected.json` · `tx/015-wbft-insufficient-funds-rejected.json` · `tx/015-wemix-insufficient-funds-rejected.json` |
| CT-TX-016 | 블록 가스 한도 초과 트랜잭션 거부 | `tx/016-gas-limit-exceeds-block-rejected.json` · `tx/016-gaslimit-exceeded-rejected.json` |
| CT-TX-017 | 거부와 실행 실패의 상태 구분 | **없다 — 새로 써야 한다** |
| CT-TX-018 | 트랜잭션 풀 전파 | `tx/018-basic-txpool-propagation.json` |
| CT-TX-019 | 부하 전송 중 블록 진행 | `tx/019-stress-tx-flood.json` |
| CT-TX-020 | 테스트 계정 자금 지급 | `tx/020-faucet-funds-account.json` |

### FEE — CT 12개 중 11개가 케이스를 갖고 있다

| CT | 무엇을 보나 | 지금 있는 케이스 |
| --- | --- | --- |
| CT-FEE-001 | 최소 팁 미달 거부 | `fee/001-dynamic-fee-below-basefee-rejected.json` |
| CT-FEE-002 | 최소 가스비 경계값 | `fee/002-legacy-gasprice-below-min-rejected.json` · `fee/002-accesslist-gasprice-below-min-rejected.json` · `fee/002-feecap-below-min-rejected.json` · `fee/002-feecap-above-min-accepted.json` · `fee/002-feecap-exact-min-accepted.json` |
| CT-FEE-003 | 사용률 높을 때 기본 수수료 증가 | `fee/003-anzeon-basefee-increase.json` |
| CT-FEE-004 | 사용률 보통일 때 기본 수수료 유지 | `fee/004-anzeon-basefee-stable.json` |
| CT-FEE-005 | 사용률 낮을 때 기본 수수료 감소 | `fee/005-anzeon-basefee-decrease.json` |
| CT-FEE-006 | 기본 수수료 하한 | `fee/006-basefee-minimum.json` |
| CT-FEE-007 | 실제 적용 가스 가격 기록 | `fee/007-effective-gas-price.json` |
| CT-FEE-008 | 실제 적용 가스 가격 노드 간 일치 | `fee/008-effective-gas-price-regular-bp-en.json` |
| CT-FEE-009 | 스냅 동기화 노드의 영수증 가스 가격 보존 | **없다 — 새로 써야 한다** |
| CT-FEE-010 | 권장 가스 가격 조회 | `fee/010-gas-price-positive.json` · `fee/010-gas-price-equals-basefee-plus-tip.json` |
| CT-FEE-011 | 권장 팁 조회 | `fee/011-max-priority-fee-equals-gastip.json` |
| CT-FEE-012 | 수수료 이력 조회 | `fee/012-fee-history-well-formed.json` |

### CONTRACT — CT 7개 중 6개가 케이스를 갖고 있다

| CT | 무엇을 보나 | 지금 있는 케이스 |
| --- | --- | --- |
| CT-CONTRACT-001 | 컨트랙트 배포 | `contract/001-contract-roundtrip.json` · `contract/001-wemix-tx-and-contract.json` · `contract/001-wbft-tx-and-contract.json` |
| CT-CONTRACT-002 | 상태 변경 함수 호출 | `contract/002-register-contract.json` |
| CT-CONTRACT-003 | 조회 함수 호출 | **없다 — 새로 써야 한다** |
| CT-CONTRACT-004 | 가스 추정 | `contract/004-estimate-gas.json` |
| CT-CONTRACT-005 | 조회 호출의 되돌림 오류 | `contract/005-eth-call-revert-returns-error.json` |
| CT-CONTRACT-006 | 되돌림 트랜잭션의 실패 상태 | `contract/006-revert-tx-status-zero.json` · `contract/006-wbft-revert-status-zero.json` · `contract/006-wemix-revert-status-zero.json` · `contract/006-negative-tx-revert.json` |
| CT-CONTRACT-007 | 가스 소진 트랜잭션 | `contract/007-out-of-gas-consumes-all.json` |

### RPC — CT 15개 중 14개가 케이스를 갖고 있다

| CT | 무엇을 보나 | 지금 있는 케이스 |
| --- | --- | --- |
| CT-RPC-001 | 최신 블록 번호 조회 | `rpc/001-basic-rpc-health.json` · `rpc/001-remote-rpc-health.json` |
| CT-RPC-002 | 블록 번호로 블록 조회 | `rpc/002-block-transactions-field.json` |
| CT-RPC-003 | 블록 해시로 블록 조회 | `rpc/003-block-by-hash-consistency.json` |
| CT-RPC-004 | 해시로 트랜잭션 조회 | `rpc/004-transaction-by-hash-fields.json` |
| CT-RPC-005 | 영수증 조회 | `rpc/005-transaction-receipt-fields.json` |
| CT-RPC-006 | 계정 nonce 조회 | `rpc/006-transaction-count-increments.json` |
| CT-RPC-007 | 잔액 조회 | `rpc/007-genesis-balance.json` · `rpc/007-remote-balance-check.json` |
| CT-RPC-008 | 체인 ID 조회 | `node/003-chain-id.json` · `node/014-remote-chain-info.json` |
| CT-RPC-009 | 이벤트 로그 조회 | `rpc/009-logs-query-well-formed.json` · `rpc/009-contract-event-emitted.json` |
| CT-RPC-010 | 서명된 트랜잭션 전송과 풀 조회 | **없다 — 새로 써야 한다** |
| CT-RPC-011 | 트랜잭션 풀 건수 조회 | `rpc/011-txpool-status.json` |
| CT-RPC-012 | 트랜잭션 풀 내용 조회 | `rpc/012-txpool-content-well-formed.json` |
| CT-RPC-013 | 새 블록 구독 | `rpc/013-ws-subscribe-new-heads.json` |
| CT-RPC-014 | 이벤트 로그 구독 | `rpc/014-ws-subscribe-logs.json` |
| CT-RPC-015 | 노드 지표 조회 | `rpc/015-metric-head-block.json` |

### FAULT — CT 6개 중 6개가 케이스를 갖고 있다

| CT | 무엇을 보나 | 지금 있는 케이스 |
| --- | --- | --- |
| CT-FAULT-001 | 생성 노드 1대 중단과 재시작 | `fault/001-fault-node-crash.json` · `fault/001-wemix-node-crash.json` · `fault/001-wbft-node-crash.json` · `fault/001-sample-lifecycle.json` |
| CT-FAULT-002 | 중단 노드 복구 후 동기화 | `fault/002-fault-node-recover.json` |
| CT-FAULT-003 | 생성 노드 2대 중단 시 합의 중단과 복구 | `fault/003-fault-two-down.json` |
| CT-FAULT-004 | 네트워크 분리와 복구 | `fault/004-fault-network-partition.json` |
| CT-FAULT-005 | 허브형 연결에서 합의와 전파 | `fault/005-fault-p2p-topology.json` |
| CT-FAULT-006 | 생산 노드 중단 시 대기 트랜잭션 처리 | `fault/006-fault-txpool-leader-change.json` |
