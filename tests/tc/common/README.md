# tests/tc/common — 세 체인에서 같은 목적으로 도는 테스트

Confluence [[Common] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987917348)
가 가려낸 공통 테스트 CT 76개의 자리다. 저장소에 옮겨 둔 사본은
[`docs/tc/common/`](../../../docs/tc/common/) 에 있다.

> **2026-09-28 — 목록을 그대로 믿지 않고 코드에 대조했다.** Confluence 가 공통으로 적은
> 것 중 여덟은 go-stablenet 에만 있는 기능에 기대고 있었고, 스물셋은 반대로 근거 없는
> 제약이 붙어 다른 체인에서 건너뛰고 있었다. 4절이 그 내역이다. Confluence 목록도 같은
> 내용으로 고쳐야 한다.

## 1. 무엇이 여기 있나

케이스 **86개**다. CT 하나에 케이스가 여럿인 것은 지금 세 체인이 각자의 케이스를 갖고
있기 때문이다 — `CT-NODE-001` 은 `wemix-chain-up`·`wbft-chain-up`·`stablenet-chain-up`
셋이다.

```
tests/tc/common/
├── node/       21건   노드·동기화·네트워크
├── tx/         23건   트랜잭션 전송·거부
├── fee/         8건   수수료·가스 정책
├── contract/    9건   컨트랙트 실행
├── rpc/        16건   조회·구독 API
└── fault/       9건   장애·복구
```

파일명은 `<CT 번호>-<테스트 id>.json` 이다. 앞의 번호가 같은 파일들이 한 CT 를 이룬다.
번호 뒤의 이름은 옮기기 전과 한 글자도 바꾸지 않았다 — `id` 로 케이스를 부르는 문서와
검사가 여럿이라, 이름을 같이 바꾸면 무엇이 깨졌는지 옮긴 것과 구별할 수 없다.

CT 하나를 두 영역이 함께 거명한 것이 둘 있다. `chain-id` 는 `CT-NODE-003` 과
`CT-RPC-008`, `remote-chain-info` 는 `CT-NODE-014` 와 `CT-RPC-008` 이다. 앞의 CT 를
따라 `node/` 에 두었다.

## 2. 왜 86개를 CT 수만큼 합치지 않았나

합치는 것이 목표지만 한 번에 하지 않는다. `CT-NODE-001` 의 세 케이스는 같은 것을 보는
듯하지만 기대값이 다르다 — 검증자 수를 확인하는 방법이 체인마다 다르고, WEMIX3.0 은
검증자 조회 자체가 없다. 지금 합치면 그 차이가 코드가 아니라 사람의 기억 속으로 들어간다.

그래서 이번 단계는 자리만 옮긴다. 세 체인의 케이스가 한 디렉터리에서 번호로 묶여 나란히
보이게 된 것이 전부다. 나란히 놓고 봐야 무엇이 같고 무엇이 다른지 셀 수 있다.

## 3. 세 체인 모두에서 게이트를 통과한다

케이스의 `requires` 를 세 체인이 제공하는 capability 집합과 대조한 결과다.

| 체인 | 86건 중 게이트 통과 |
| --- | --- |
| go-stablenet | 86 |
| go-wbft | 86 |
| go-wemix | 86 |

**게이트를 통과한다는 것은 "돌 수 있다" 이지 "통과한다" 가 아니다.** 91건이던 때도 이 표는
91/91/91 이었고, 그대로 세 체인에 돌리자 다섯이 깨졌다. 게이트가 막지 못한 것은 케이스가
무엇을 필요로 하는지 말하지 않았기 때문이다. 다섯은 빠졌고, 무엇이 필요한지 이제
`requires` 에 적혀 있다(5절).

케이스가 이름 붙인 `chainPreset` 은 아직 대부분 stablenet 이다. 실행할 때 덮는다.

```sh
chainbench run tests/tc/common/tx/001-value-transfer.json \
  --workspace-dir ~/cbw/x --chain-preset wemix-bp4
```

세 체인이 모두 갖춘 모양은 `bp4` 와 `bp7-en7-pn1` 둘이고 `presets/chain/` 에 세 벌씩 있다.

## 4. 공통 판정을 다시 한 내역 (2026-09-28)

세 체인의 노드 소스를 직접 대조했다. 결정적인 것은 셋이다.

| | go-stablenet | go-wbft | go-wemix |
| --- | --- | --- | --- |
| `header.GasTip()` | 있음 | 없음 | 없음 |
| `params.MinBaseFee` | 있음 | 없음 | 없음 |
| `istanbul_getWbftExtraInfo` 응답의 `gasTip` | 있음 | **없음** | 없음 |

세 체인 모두 genesis 가 `londonBlock: 0` 이고 시작 baseFee 가 1 Gwei 로 같다. 거부 문구
`transaction underpriced`·`exceeds block gas limit`·`insufficient funds` 도 셋 다 같다.

### 4.1 공통에서 뺀 여덟 — go-stablenet 전용이다

`gasTip` 또는 `MinBaseFee` 에 기대므로 다른 두 체인에는 물을 대상이 없다.

| 케이스 | CT | 어디로 갔나 |
| --- | --- | --- |
| `feecap-above-min-accepted` | CT-FEE-002 | `go-stablenet/regression/anzeon/08-` |
| `feecap-exact-min-accepted` | CT-FEE-002 | `go-stablenet/regression/anzeon/09-` |
| `basefee-minimum` | CT-FEE-006 | `go-stablenet/regression/anzeon/06-` |
| `gas-price-equals-basefee-plus-tip` | CT-FEE-010 | `go-stablenet/regression/api/07b-` |
| `max-priority-fee-equals-gastip` | CT-FEE-011 | `go-stablenet/regression/api/08-` |
| `legacy-transfer` | CT-TX-002 | `go-stablenet/regression/ethereum/08-` |
| `dynamic-fee-tx` | CT-TX-003 | `go-stablenet/regression/ethereum/09-` |
| `gaslimit-exceeded-rejected` | CT-TX-016 | `go-stablenet/regression/anzeon/11-` |

**셋은 검사하려는 것 자체는 공통이다.** CT-TX-002(Legacy 트랜잭션)·CT-TX-003(EIP-1559)·
CT-TX-016(가스 한도 초과)은 세 체인에서 다 되는 일이고, 공통이 아닌 것은 지금 구현이
gasPrice 를 `gasTip` 으로 구한다는 점이다. 다시 쓰면 살아난다. 나머지 넷(CT-FEE-002·
006·010·011)은 규칙 자체가 go-stablenet 것이라 살릴 수 없다.

### 4.2 제약을 뗀 스물셋 — 근거가 없었다

`family:wbft` 18건과 `engine:anzeon` 5건이다. 18건은 파일 안에서 `wbft` 라는 낱말이
나오는 곳이 `requires` 줄 하나뿐이었고, 쓰는 것은 전부 표준 이더리움 RPC 였다. 5건은
gasPrice 를 1 wei 로 보내거나 블록 가스 한도를 넘겨 거부를 기대하는 것이라, baseFee 가
1 Gwei 에서 시작하는 세 체인 어디서나 성립한다.

이 제약이 붙어 있는 동안 wemix 에서 31건이 **실패가 아니라 조용히 건너뛰고** 있었다
(`internal/testengine/capability.go`: "skipped, not failed").

### 4.3 아직 답을 못 낸 둘

`CT-NODE-016`(`016-block-period-one-second`·`016-stress-block-time`)은 블록 주기가
1초라는 것에 기댄다. go-stablenet 과 go-wbft 는 genesis 템플릿에 `blockPeriodSeconds: 1`
을 적지만 **go-wemix 는 적지 않는다** — 거버넌스 값이고 부트스트랩 때 정해진다. 둘은
게이트를 통과하므로 건너뛰지 않고 **돌아서 실패하거나 운으로 통과한다.** 체인별 기대값을
적을 자리가 필요하고, 그 설계는
[`design-v3/common-tc-01-chain-varying.md`](../../../docs/dev/architecture/design-v3/common-tc-01-chain-varying.md)
§3.2 에 있다.

## 5. 여기 없는 것

CT 12개는 자동 테스트가 없어 옮길 것이 없었다.

`CT-NODE-004`·`005`·`006`·`007`(동기화 네 가지) · `CT-NODE-015` · `CT-TX-010`·`011`·
`014`·`017` · `CT-FEE-009` · `CT-CONTRACT-003` · `CT-RPC-010`

### 세 체인에 돌려 보고 되돌린 다섯 (2026-09-28)

처음에 91개를 옮겼다가 다섯을 go-stablenet 영역으로 되돌렸다. 읽어서는 보이지 않고
돌려야 나오는 차이였다. 되돌리면서 무엇이 필요한지 `requires` 에 적었으므로, 이제는 다른
체인에서 실패하지 않고 건너뛴다.

| 케이스 | 되돌아간 곳 | 필요한 것 | 왜 |
| --- | --- | --- | --- |
| `register-contract` | `go-stablenet/vocabulary/03-` | `evm:shanghai` | 배포하는 바이트코드가 PUSH0 를 45번 쓴다. go-wemix 의 EVM 은 London 세대라 그 옵코드가 없어 배포가 가스를 전부 태우고 실패한다 |
| `eth-call-revert-returns-error` | `go-stablenet/regression/ethereum/23-` | `evm:shanghai` | 같은 이유. PUSH0 9번 |
| `anzeon-basefee-increase` | `go-stablenet/regression/anzeon/03-` | `engine:anzeon` | anzeon 만 상승·하강 문턱 두 개를 쓴다. 표준 EIP-1559 는 목표 하나뿐이라 25% 채우기가 목표 미달이 되어 반대로 내려간다 |
| `anzeon-basefee-stable` | `go-stablenet/regression/anzeon/04-` | `engine:anzeon` | 표준 EIP-1559 에는 유지 구간이 없다 |
| `anzeon-basefee-decrease` | `go-stablenet/regression/anzeon/05-` | `engine:anzeon` | go-wbft 에서 통과했으나 재려던 것과 다른 이유였다 |

마지막 줄이 이 절의 요점이다. **통과했다고 잰 것은 아니다.** 표준 EIP-1559 에서 25%
채우기는 부하가 아니라 목표 미달이라, 부하를 거는 동안에도 기본 수수료가 이미 내려가고
있었다. 셋을 묶어 빼지 않으면 그 하나가 "wbft 에서 도는 공통 테스트" 로 남는다.

`evm:shanghai` 는 이번에 더한 capability 다. 세 체인이 포크를 brioche·croissant·anzeon
이라 부르고 어느 genesis 에도 이더리움 포크 이름이 없어 뽑아낼 데가 없으므로,
`precompile:` 과 같이 매니페스트가 선언한다.

## 6. CT 와 케이스 대응

### NODE — CT 16개 중 11개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-NODE-001 | 노드 기동과 블록 생성 | `node/001-wemix-chain-up.json` · `node/001-wbft-chain-up.json` · `node/001-stablenet-chain-up.json` |
| CT-NODE-002 | 15노드 구성 기동 | `node/002-wemix-chain-up-15.json` · `node/002-wbft-chain-up-15.json` · `node/002-stablenet-chain-up-15.json` |
| CT-NODE-003 | 제네시스 초기화와 블록 0 해시 일치 | `node/003-genesis-block-hash-consistent.json` · `node/003-chain-id.json` |
| CT-NODE-004 | Full Sync 동기화 | **없다** |
| CT-NODE-005 | Snap Sync 동기화 | **없다** |
| CT-NODE-006 | 누락 블록 일괄 동기화 | **없다** |
| CT-NODE-007 | 새 블록 실시간 수신 | **없다** |
| CT-NODE-008 | 피어 연결과 피어 조회 | `node/008-basic-peers.json` · `node/008-admin-peers-populated.json` |
| CT-NODE-009 | 노드 간 최신 블록 해시 일치 | `node/009-basic-consensus.json` · `node/009-basic-sync.json` |
| CT-NODE-010 | 전달 노드 경유 동기화 | `node/010-proxied-pn-routing.json` · `node/010-wbft-proxied-routing.json` |
| CT-NODE-011 | 종단 노드가 먼저 오는 혼합 배치 | `node/011-e1-mixed-producers.json` |
| CT-NODE-012 | 노드 프로그램 교체 후 서명 호환 | `node/012-signature-compat-across-swap.json` |
| CT-NODE-013 | 제네시스 불일치 시 기동 거부 | `node/013-genesis-mismatch.json` |
| CT-NODE-014 | 동기화 완료 상태 확인 | `node/014-chain-not-syncing.json` · `node/014-remote-chain-info.json` |
| CT-NODE-015 | 블록 시각 단조 증가 | **없다** |
| CT-NODE-016 | 블록 생성 주기 | `node/016-block-period-one-second.json` · `node/016-stress-block-time.json` |

### TX — CT 20개 중 14개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-TX-001 | 일반 송금 | `tx/001-basic-tx-send.json` · `tx/001-sample-minimal.json` · `tx/001-value-transfer.json` |
| CT-TX-002 | Legacy 트랜잭션 | **없다**공통 아님: `go-stablenet/regression/ethereum/08-legacy-transfer.json` |
| CT-TX-003 | 동적 수수료 트랜잭션 | **없다**공통 아님: `go-stablenet/regression/ethereum/09-dynamic-fee-tx.json` |
| CT-TX-004 | 접근 목록 트랜잭션 | `tx/004-access-list-tx.json` |
| CT-TX-005 | 수수료 대납 트랜잭션 | `tx/005-fee-delegated-transfer.json` |
| CT-TX-006 | 대납 트랜잭션의 보낸 이 서명 변조 거부 | `tx/006-fd-sender-sig-invalid-rejected.json` · `tx/006-fee-delegated-sender-sig-invalid-rejected.json` |
| CT-TX-007 | 대납 트랜잭션의 대납자 서명 변조 거부 | `tx/007-fd-feepayer-sig-invalid-rejected.json` · `tx/007-fee-delegated-feepayer-sig-invalid-rejected.json` |
| CT-TX-008 | 대납자 잔액 부족 거부 | `tx/008-feepayer-insufficient-rejected.json` · `tx/008-fee-delegated-unfunded-feepayer-rejected.json` |
| CT-TX-009 | 대납 서명 API 존재 | `tx/009-fee-delegate-sign-rpc-present.json` |
| CT-TX-010 | 접근 목록을 붙인 대납 트랜잭션 | **없다** |
| CT-TX-011 | 노드 키 저장소 경유 대납 서명 | **없다** |
| CT-TX-012 | 계정별 nonce 순서 보장 | `tx/012-nonce-ordering.json` · `tx/012-out-of-order-nonces-mine.json` |
| CT-TX-013 | 같은 nonce 트랜잭션 교체 | `tx/013-replacement-tx.json` · `tx/013-same-nonce-replacement.json` |
| CT-TX-014 | 미포함 트랜잭션의 이월과 교체 | **없다** |
| CT-TX-015 | 잔액 부족 트랜잭션 거부 | `tx/015-insufficient-funds-rejected.json` · `tx/015-wbft-insufficient-funds-rejected.json` · `tx/015-wemix-insufficient-funds-rejected.json` |
| CT-TX-016 | 블록 가스 한도 초과 트랜잭션 거부 | `tx/016-gas-limit-exceeds-block-rejected.json` / 공통 아님: `go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json` |
| CT-TX-017 | 거부와 실행 실패의 상태 구분 | **없다** |
| CT-TX-018 | 트랜잭션 풀 전파 | `tx/018-basic-txpool-propagation.json` |
| CT-TX-019 | 부하 전송 중 블록 진행 | `tx/019-stress-tx-flood.json` |
| CT-TX-020 | 테스트 계정 자금 지급 | `tx/020-faucet-funds-account.json` |

### FEE — CT 9개 중 6개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-FEE-001 | 최소 팁 미달 거부 | `fee/001-dynamic-fee-below-basefee-rejected.json` |
| CT-FEE-002 | 최소 가스비 경계값 | `fee/002-legacy-gasprice-below-min-rejected.json` · `fee/002-accesslist-gasprice-below-min-rejected.json` · `fee/002-feecap-below-min-rejected.json` / 공통 아님: `go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json` · `go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json` |
| CT-FEE-006 | 기본 수수료 하한 | **없다**공통 아님: `go-stablenet/regression/anzeon/06-basefee-minimum.json` |
| CT-FEE-007 | 실제 적용 가스 가격 기록 | `fee/007-effective-gas-price.json` |
| CT-FEE-008 | 실제 적용 가스 가격 노드 간 일치 | `fee/008-effective-gas-price-regular-bp-en.json` |
| CT-FEE-009 | 스냅 동기화 노드의 영수증 가스 가격 보존 | **없다** |
| CT-FEE-010 | 권장 가스 가격 조회 | `fee/010-gas-price-positive.json` / 공통 아님: `go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json` |
| CT-FEE-011 | 권장 팁 조회 | **없다**공통 아님: `go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json` |
| CT-FEE-012 | 수수료 이력 조회 | `fee/012-fee-history-well-formed.json` |

### CONTRACT — CT 5개 중 4개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-CONTRACT-001 | 컨트랙트 배포 | `contract/001-contract-roundtrip.json` · `contract/001-wemix-tx-and-contract.json` · `contract/001-wbft-tx-and-contract.json` |
| CT-CONTRACT-003 | 조회 함수 호출 | **없다** |
| CT-CONTRACT-004 | 가스 추정 | `contract/004-estimate-gas.json` |
| CT-CONTRACT-006 | 되돌림 트랜잭션의 실패 상태 | `contract/006-revert-tx-status-zero.json` · `contract/006-wbft-revert-status-zero.json` · `contract/006-wemix-revert-status-zero.json` · `contract/006-negative-tx-revert.json` |
| CT-CONTRACT-007 | 가스 소진 트랜잭션 | `contract/007-out-of-gas-consumes-all.json` |

### RPC — CT 15개 중 14개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
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
| CT-RPC-010 | 서명된 트랜잭션 전송과 풀 조회 | **없다** |
| CT-RPC-011 | 트랜잭션 풀 건수 조회 | `rpc/011-txpool-status.json` |
| CT-RPC-012 | 트랜잭션 풀 내용 조회 | `rpc/012-txpool-content-well-formed.json` |
| CT-RPC-013 | 새 블록 구독 | `rpc/013-ws-subscribe-new-heads.json` |
| CT-RPC-014 | 이벤트 로그 구독 | `rpc/014-ws-subscribe-logs.json` |
| CT-RPC-015 | 노드 지표 조회 | `rpc/015-metric-head-block.json` |

### FAULT — CT 6개 중 6개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-FAULT-001 | 생성 노드 1대 중단과 재시작 | `fault/001-fault-node-crash.json` · `fault/001-wemix-node-crash.json` · `fault/001-wbft-node-crash.json` · `fault/001-sample-lifecycle.json` |
| CT-FAULT-002 | 중단 노드 복구 후 동기화 | `fault/002-fault-node-recover.json` |
| CT-FAULT-003 | 생성 노드 2대 중단 시 합의 중단과 복구 | `fault/003-fault-two-down.json` |
| CT-FAULT-004 | 네트워크 분리와 복구 | `fault/004-fault-network-partition.json` |
| CT-FAULT-005 | 허브형 연결에서 합의와 전파 | `fault/005-fault-p2p-topology.json` |
| CT-FAULT-006 | 생산 노드 중단 시 대기 트랜잭션 처리 | `fault/006-fault-txpool-leader-change.json` |
