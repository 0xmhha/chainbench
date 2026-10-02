# tests/tc/common — 세 체인에서 같은 목적으로 도는 테스트

Confluence [[Common] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987917348)
가 가려낸 공통 테스트 CT 70개의 자리다. 저장소에 옮겨 둔 사본은
[`docs/tc/common/`](../../../docs/tc/common/) 에 있다.

여기 있는 케이스는 **파일 하나로 세 체인을 돈다.** 체인마다 사본을 두지 않고,
`stablenet-*` 프리셋으로 적어 두고 실행할 때 갈아 끼운다.

## 1. 무엇이 여기 있나

케이스 **69개**다. **CT 하나에 파일 하나**이고 여섯 영역이 모두 그렇다. CT 는 70개인데
파일이 69개인 것은 CT-RPC-008 하나 때문이다. 그 CT 는 전용 파일 없이 `node/` 의 두 파일과
목적이 겹쳐 그것을 공유한다.

```
tests/tc/common/
├── node/       15건   노드·동기화·네트워크
├── tx/         20건   트랜잭션 전송·거부
├── fee/         7건   수수료·가스 정책
├── contract/    7건   컨트랙트 실행
├── rpc/        14건   조회·구독 API
└── fault/       6건   장애·복구
```

파일명은 `CT-<영역>-<번호>-<간략설명>.json` 이다
(예: `node/CT-NODE-001-startup-block-production.json`). 한 CT 가 여러 가지를 보면 한 파일
안에서 차례로 검증한다. 파일 이름을 바꿔도 **파일 안의 `id` 는 그대로 둔다** — `id` 로
케이스를 부르는 문서와 검사가 여럿이라, 이름을 같이 바꾸면 무엇이 깨졌는지 구별할 수 없다.

CT 하나를 두 영역이 함께 거명한 것이 둘 있다. `CT-NODE-003-genesis-init` 은 CT-NODE-003 과
CT-RPC-008 을, `CT-NODE-014-sync-complete` 는 CT-NODE-014 와 CT-RPC-008 을 함께 본다. 앞의
CT 를 따라 `node/` 에 두었다.

체인 이름이 붙은 파일은 하나도 없다.

## 2. 세 체인에서 돈다

케이스의 `requires` 를 세 체인이 내놓는 capability 집합과 대조하면, **69개 중 68개가 세
체인 모두에서 게이트를 통과한다.**

남은 하나는 `fault/CT-FAULT-004-network-partition` 이다. 체인이 아니라 실행 대상에
`target:remote` 를 요구한다. 노드가 도는 기계에 셸과 방화벽 권한이 있어야 하므로, 원격
서버나 Docker 함대에서만 돌고 로컬에서는 건너뛴다.

**게이트를 통과한다는 것은 "돌 수 있다" 이지 "통과한다" 가 아니다.** 한때 이 표가 91/91/91
이었고, 그대로 세 체인에 돌리자 다섯이 깨졌다. 게이트가 막지 못한 것은 케이스가 자기에게
무엇이 필요한지 말하지 않았기 때문이다. 그 다섯은 3절에 있다.

체인을 바꾸는 방법은 프리셋을 덮는 것이다.

```sh
bin/chainbench run tests/tc/common/tx/CT-TX-001-value-transfer.json \
  --workspace-dir ~/cbw/x --chain-preset wemix-bp4 --binary <gwemix 경로>
```

세 체인이 모두 갖춘 모양은 `bp4`, `bp4-en1`, `bp4-en2-pn1`, `bp7-en7-pn1`, `bp9` 다섯이고
`presets/chain/` 에 세 벌씩 있다.

## 3. 여기 없는 것과 그 이유

공통으로 옮기려다 go-stablenet 영역으로 되돌린 것들이다. 셋은 읽어서 알았고 다섯은 세
체인에 돌려 보고서야 나왔다.

### go-stablenet 전용 기능에 기대는 여덟

`header.GasTip()` 과 `params.MinBaseFee` 는 go-stablenet 에만 있다. 같은
`istanbul_getWbftExtraInfo` 를 가진 go-wbft 도 응답에 `gasTip` 을 담지 않는다.

| 케이스 | CT | 지금 자리 |
| --- | --- | --- |
| `feecap-above-min-accepted` | CT-FEE-002 | `go-stablenet/regression/anzeon/08-` |
| `feecap-exact-min-accepted` | CT-FEE-002 | `go-stablenet/regression/anzeon/09-` |
| `basefee-minimum` | CT-FEE-006 | `go-stablenet/regression/anzeon/06-` |
| `gas-price-equals-basefee-plus-tip` | CT-FEE-010 | `go-stablenet/regression/api/07b-` |
| `max-priority-fee-equals-gastip` | CT-FEE-011 | `go-stablenet/regression/api/08-` |
| `legacy-transfer` | CT-TX-002 | `go-stablenet/regression/ethereum/08-` |
| `dynamic-fee-tx` | CT-TX-003 | `go-stablenet/regression/ethereum/09-` |
| `gaslimit-exceeded-rejected` | CT-TX-016 | `go-stablenet/regression/anzeon/11-` |

셋(CT-TX-002·003·016)은 **검사하려는 일 자체는 공통이다.** 공통이 아닌 것은 그 구현이
가스 가격을 `gasTip` 으로 구한다는 점뿐이라, 다시 쓰면 살아난다. 실제로 세 CT 모두 지금
공통에 케이스를 갖고 있다. 나머지 넷은 규칙 자체가 go-stablenet 것이다.

### 돌려 보고서야 나온 다섯

| 케이스 | 지금 자리 | 필요한 것 | 왜 |
| --- | --- | --- | --- |
| `register-contract` | `go-stablenet/vocabulary/03-` | `evm:shanghai` | 배포 바이트코드가 PUSH0 를 45번 쓴다. go-wemix 의 EVM 은 London 세대라 그 옵코드가 없어 배포가 가스를 다 태우고 실패한다 |
| `eth-call-revert-returns-error` | 공통으로 돌아왔다 (`contract/CT-CONTRACT-005-`) | - | 같은 이유였으나 PUSH0 없는 컨트랙트로 다시 썼다 |
| `anzeon-basefee-increase` | `go-stablenet/regression/anzeon/03-` | `engine:anzeon` | anzeon 만 상승·하강 문턱 두 개를 쓴다. 표준 EIP-1559 는 목표 하나뿐이라 25% 채우기가 목표 미달이 되어 반대로 내려간다 |
| `anzeon-basefee-stable` | `go-stablenet/regression/anzeon/04-` | `engine:anzeon` | 표준 EIP-1559 에는 유지 구간이 없다 |
| `anzeon-basefee-decrease` | `go-stablenet/regression/anzeon/05-` | `engine:anzeon` | go-wbft 에서 통과했으나 재려던 것과 다른 이유였다 |

마지막 줄이 이 절의 요점이다. **통과했다고 잰 것은 아니다.** 표준 EIP-1559 에서 25%
채우기는 부하가 아니라 목표 미달이라, 부하를 거는 동안에도 기본 수수료가 이미 내려가고
있었다. 셋을 묶어 빼지 않으면 그 하나가 "wbft 에서 도는 공통 테스트" 로 남는다.

`evm:shanghai` 는 그때 더한 capability 다. 세 체인이 포크를 brioche·croissant·anzeon 이라
부르고 어느 genesis 에도 이더리움 포크 이름이 없어 뽑아낼 데가 없으므로,
`precompile:` 과 같이 매니페스트가 선언한다.

### 아직 답을 못 낸 하나

`CT-NODE-016`(블록 주기)은 체인마다 주기가 달라 경계를 넓게 잡고 있다. `isPerChain` 은
기댓값 하나만 바꾸므로 경계 두 개를 체인별로 줄 수 없고, 세 체인이 설정 주기를 같은
방법으로 RPC 로 내놓지도 않는다. 좁히려면 그 값을 읽어 견주어야 한다. 지금 구간도 멈춘
체인과 두 배 빠른 체인은 잡는다. 설계는
[`design-v3/common-tc-01-chain-varying.md`](../../../docs/dev/architecture/design-v3/common-tc-01-chain-varying.md)
§3.2 에 있다.

## 4. CT 와 케이스 대응

### NODE — CT 15개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-NODE-001 | 노드 기동과 블록 생성 | `node/CT-NODE-001-startup-block-production.json` (세 체인 공통) |
| CT-NODE-002 | 15노드 구성 기동 | `node/CT-NODE-002-startup-15-nodes.json` (세 체인 공통) |
| CT-NODE-003 | 제네시스 초기화와 블록 0 해시 일치 | `node/CT-NODE-003-genesis-init.json` |
| CT-NODE-004 | Full Sync 동기화 | `node/CT-NODE-004-full-sync.json` (2026-09-29 신규) |
| CT-NODE-005 | Snap Sync 동기화 | `node/CT-NODE-005-snap-sync.json` (2026-09-29 신규) |
| CT-NODE-006 | 누락 블록 일괄 동기화 | `node/CT-NODE-006-missing-block-catch-up.json` (2026-09-29 신규) |
| CT-NODE-007 | 새 블록 실시간 수신 | `node/CT-NODE-007-live-block-receive.json` (2026-09-29 신규) |
| CT-NODE-008 | 피어 연결과 피어 조회 | `node/CT-NODE-008-peers.json` |
| CT-NODE-009 | 노드 간 최신 블록 해시 일치 | `node/CT-NODE-009-head-hash-agreement.json` |
| CT-NODE-010 | 전달 노드 경유 동기화 | `node/CT-NODE-010-sync-via-proxy.json` |
| CT-NODE-011 | 종단 노드가 먼저 오는 혼합 배치 | `node/CT-NODE-011-endpoint-first-layout.json` |
| CT-NODE-013 | 제네시스 불일치 시 기동 거부 | `node/CT-NODE-013-genesis-mismatch-refused.json` |
| CT-NODE-014 | 동기화 완료 상태 확인 | `node/CT-NODE-014-sync-complete.json` |
| CT-NODE-015 | 블록 시각 단조 증가 | `node/CT-NODE-015-timestamp-monotonic.json` (2026-09-29 신규) |
| CT-NODE-016 | 블록 생성 주기 | `node/CT-NODE-016-block-period.json` |

### TX — CT 20개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-TX-001 | 일반 송금 | `tx/CT-TX-001-value-transfer.json` |
| CT-TX-002 | Legacy 트랜잭션 | `tx/CT-TX-002-legacy-transfer.json` (2026-09-29 신규) / 공통 아님: `go-stablenet/regression/ethereum/08-legacy-transfer.json` |
| CT-TX-003 | 동적 수수료 트랜잭션 | `tx/CT-TX-003-dynamic-fee-transfer.json` (2026-09-29 신규) / 공통 아님: `go-stablenet/regression/ethereum/09-dynamic-fee-tx.json` |
| CT-TX-004 | 접근 목록 트랜잭션 | `tx/CT-TX-004-access-list-tx.json` |
| CT-TX-005 | 수수료 대납 트랜잭션 | `tx/CT-TX-005-fee-delegated-transfer.json` |
| CT-TX-006 | 대납 트랜잭션의 보낸 이 서명 변조 거부 | `tx/CT-TX-006-fd-sender-sig-tampered-rejected.json` |
| CT-TX-007 | 대납 트랜잭션의 대납자 서명 변조 거부 | `tx/CT-TX-007-fd-feepayer-sig-tampered-rejected.json` |
| CT-TX-008 | 대납자 잔액 부족 거부 | `tx/CT-TX-008-feepayer-insufficient-rejected.json` |
| CT-TX-009 | 대납 서명 API 존재 | `tx/CT-TX-009-fee-delegate-sign-rpc-present.json` |
| CT-TX-010 | 접근 목록을 붙인 대납 트랜잭션 | `tx/CT-TX-010-fee-delegated-access-list.json` (2026-09-29 신규) |
| CT-TX-011 | 노드 키 저장소 경유 대납 서명 | `tx/CT-TX-011-keystore-fee-delegate-sign.json` (2026-09-29 신규) |
| CT-TX-012 | 계정별 nonce 순서 보장 | `tx/CT-TX-012-nonce-ordering.json` |
| CT-TX-013 | 같은 nonce 트랜잭션 교체 | `tx/CT-TX-013-same-nonce-replacement.json` |
| CT-TX-014 | 미포함 트랜잭션의 이월과 교체 | `tx/CT-TX-014-carry-over-and-replace.json` (2026-09-29 신규) |
| CT-TX-015 | 잔액 부족 트랜잭션 거부 | `tx/CT-TX-015-insufficient-funds-rejected.json` |
| CT-TX-016 | 블록 가스 한도 초과 트랜잭션 거부 | `tx/CT-TX-016-gas-limit-exceeds-block-rejected.json` / 공통 아님: `go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json` |
| CT-TX-017 | 거부와 실행 실패의 상태 구분 | `tx/CT-TX-017-reject-vs-execution-failure.json` (2026-09-29 신규) |
| CT-TX-018 | 트랜잭션 풀 전파 | `tx/CT-TX-018-txpool-propagation.json` |
| CT-TX-019 | 부하 전송 중 블록 진행 | `tx/CT-TX-019-block-progress-under-load.json` |
| CT-TX-020 | 테스트 계정 자금 지급 | `tx/CT-TX-020-test-account-funding.json` |

### FEE — CT 7개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-FEE-001 | 최소 팁 미달 거부 | `fee/CT-FEE-001-tip-below-min-rejected.json` |
| CT-FEE-002 | 최소 가스비 경계값 | `fee/CT-FEE-002-min-gas-price-boundary.json` (초과 부분은 2026-09-29 신규) |
| CT-FEE-007 | 실제 적용 가스 가격 기록 | `fee/CT-FEE-007-effective-gas-price.json` |
| CT-FEE-008 | 실제 적용 가스 가격 노드 간 일치 | `fee/CT-FEE-008-effective-gas-price-across-nodes.json` |
| CT-FEE-009 | 스냅 동기화 노드의 영수증 가스 가격 보존 | `fee/CT-FEE-009-snap-receipt-gas-price.json` (2026-09-29 신규) |
| CT-FEE-010 | 권장 가스 가격 조회 | `fee/CT-FEE-010-suggested-gas-price.json` / 공통 아님: `go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json` |
| CT-FEE-012 | 수수료 이력 조회 | `fee/CT-FEE-012-fee-history.json` |

### CONTRACT — CT 7개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-CONTRACT-001 | 컨트랙트 배포 | `contract/CT-CONTRACT-001-deploy-and-call.json` |
| CT-CONTRACT-002 | 상태 변경 함수 호출 | `contract/CT-CONTRACT-002-storage-write-and-read.json` |
| CT-CONTRACT-003 | 조회 함수 호출 | `contract/CT-CONTRACT-003-view-call-leaves-state.json` (2026-09-29 신규) |
| CT-CONTRACT-004 | 가스 추정 | `contract/CT-CONTRACT-004-estimate-gas.json` |
| CT-CONTRACT-005 | 조회 호출의 되돌림 오류 | `contract/CT-CONTRACT-005-eth-call-revert-returns-error.json` |
| CT-CONTRACT-006 | 되돌림 트랜잭션의 실패 상태 | `contract/CT-CONTRACT-006-revert-tx-status-zero.json` |
| CT-CONTRACT-007 | 가스 소진 트랜잭션 | `contract/CT-CONTRACT-007-out-of-gas-consumes-all.json` |

### RPC — CT 15개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-RPC-001 | 최신 블록 번호 조회 | `rpc/CT-RPC-001-block-number-advances.json` |
| CT-RPC-002 | 블록 번호로 블록 조회 | `rpc/CT-RPC-002-block-transactions-field.json` |
| CT-RPC-003 | 블록 해시로 블록 조회 | `rpc/CT-RPC-003-block-by-hash-consistency.json` |
| CT-RPC-004 | 해시로 트랜잭션 조회 | `rpc/CT-RPC-004-transaction-by-hash-fields.json` |
| CT-RPC-005 | 영수증 조회 | `rpc/CT-RPC-005-transaction-receipt-fields.json` |
| CT-RPC-006 | 계정 nonce 조회 | `rpc/CT-RPC-006-transaction-count-increments.json` |
| CT-RPC-007 | 잔액 조회 | `rpc/CT-RPC-007-balance-query.json` |
| CT-RPC-008 | 체인 ID 조회 | `node/CT-NODE-003-genesis-init.json` · `node/CT-NODE-014-sync-complete.json` |
| CT-RPC-009 | 이벤트 로그 조회 | `rpc/CT-RPC-009-event-logs-query.json` |
| CT-RPC-010 | 서명된 트랜잭션 전송과 풀 조회 | `rpc/CT-RPC-010-signed-tx-seen-in-pool.json` (2026-09-29 신규) |
| CT-RPC-011 | 트랜잭션 풀 건수 조회 | `rpc/CT-RPC-011-txpool-status.json` |
| CT-RPC-012 | 트랜잭션 풀 내용 조회 | `rpc/CT-RPC-012-txpool-content-well-formed.json` |
| CT-RPC-013 | 새 블록 구독 | `rpc/CT-RPC-013-ws-subscribe-new-heads.json` |
| CT-RPC-014 | 이벤트 로그 구독 | `rpc/CT-RPC-014-ws-subscribe-logs.json` |
| CT-RPC-015 | 노드 지표 조회 | `rpc/CT-RPC-015-metric-head-block.json` |

### FAULT — CT 6개 중 6개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-FAULT-001 | 생성 노드 1대 중단과 재시작 | `fault/CT-FAULT-001-producer-crash-and-restart.json` |
| CT-FAULT-002 | 중단 노드 복구 후 동기화 | `fault/CT-FAULT-002-node-recover-and-sync.json` |
| CT-FAULT-003 | 생성 노드 2대 중단 시 합의 중단과 복구 | `fault/CT-FAULT-003-two-producers-down.json` |
| CT-FAULT-004 | 네트워크 분리와 복구 | `fault/CT-FAULT-004-network-partition.json` |
| CT-FAULT-005 | 허브형 연결에서 합의와 전파 | `fault/CT-FAULT-005-hub-topology.json` |
| CT-FAULT-006 | 생산 노드 중단 시 대기 트랜잭션 처리 | `fault/CT-FAULT-006-txpool-leader-change.json` |

## 5. 돌리는 법

케이스별 실행 명령, 바이너리를 무엇으로 주는지, 판정을 어떻게 읽는지, Docker 함대 준비는
[`../HOW-TO-USE.md`](../HOW-TO-USE.md) 에 있다.

CT 와 DSL 파일의 대응을 더 자세히 보려면
[`docs/tc/common/10-ct-dsl-mapping.md`](../../../docs/tc/common/10-ct-dsl-mapping.md) 가
70개 CT 를 한 줄씩 적는다.
