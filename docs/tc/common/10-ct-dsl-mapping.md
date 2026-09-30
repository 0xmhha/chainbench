# CT ↔ DSL 파일 매핑

> 근거: 이 저장소의 `docs/tc/common/01-common-test-list.md`(문서화된 70개 CT)와
> `tests/tc/common/` 아래 실제 DSL 파일(69개)을 대조해 만든 파생 문서다.
> 대조 기준일: 2026-09-30. 파일이 추가·삭제되면 이 표도 다시 맞춰야 한다.

---

## 1. 개요

문서에는 세 체인 공통 테스트가 **70개(CT)** 있고, `tests/tc/common/` 아래에는 실행용
DSL 파일이 **69개** 있다. "70개를 세 체인용으로 각각 만들었다면 210개여야 한다"는
예상과 숫자가 다른 이유는 chainbench DSL 이 **체인-파라미터 방식**이기 때문이다.

공통 DSL 한 파일은 `stablenet-*` 프리셋을 기준으로 작성하고, 실행할 때
`--chain-preset <wbft/wemix 프리셋> --binary <해당 체인 바이너리>` 로 갈아 끼워 나머지
두 체인에서도 그대로 돌린다. 그래서 대부분의 CT 는 파일 하나로 세 체인을 덮는다.
파일을 체인별로 물리적으로 나눈 경우는 **런타임 프리셋 교체로는 공유할 수 없는**
소수의 CT(합의 방식·바이너리 동작이 갈리는 것)뿐이다.

정리하면 **69개 전부가 공통 파일**이다. 체인 이름을 붙여 따로 둔 파일은 2026-09-30 에 하나도
남지 않았다(§4). NODE 의 001·002·010 은 2026-09-29 에, CONTRACT 의 001·006 과 FAULT 의 001 은
2026-09-30 에 체인별 파일을 지우고 공통 파일 하나로 합쳤다.

파일명 규칙은 `CT-<영역>-<번호>-<간략설명>.json` 이고 **여섯 영역 모두 CT 하나에 파일 하나**다
(2026-09-29 NODE·TX·FEE, 2026-09-30 CONTRACT·RPC·FAULT). 한 CT 가 여러 가지를 보면 한 파일
안에서 차례로 검증한다.

70 과 69 가 어긋나는 것은 CT-RPC-008 하나 때문이다. 그 CT 는 전용 파일 없이 `node/` 의 두
파일과 목적이 겹쳐 그것을 공유한다(§5).

파일이 CT 보다 많던 때가 있었다. 예전부터 있던 자동 테스트(실행 이름이 여럿)를 CT 하나로
묶었기 때문이며, 묶인 원본 실행 이름은 01 문서의 "비고"에 모두 적혀 있다.

---

## 2. 영역별 요약

| 영역 | CT 수 | DSL 파일 수 | 파일이 CT 수보다 많은 이유 |
| --- | --- | --- | --- |
| NODE | 15 | 15 | CT당 1파일(2026-09-29 합침) |
| TX | 20 | 20 | CT당 1파일(2026-09-29 합침) |
| FEE | 7 | 7 | CT당 1파일(2026-09-30 합침) |
| CONTRACT | 7 | 7 | CT당 1파일(2026-09-30 합침) |
| RPC | 15 | 14 | CT당 1파일(2026-09-30 합침), 008 은 전용 파일 없음 |
| FAULT | 6 | 6 | CT당 1파일(2026-09-30 합침) |
| **합계** | **70** | **69** | orphan(문서에 없는) 파일 0 |

---

## 3. 영역별 매핑

각 행은 `CT ID → tests/tc/common/<영역>/ 아래 파일`. 파일명은 디렉터리 안 기준.

### NODE (15파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-NODE-001 | `CT-NODE-001-startup-block-production.json` | 세 체인 공통. 2026-09-29 에 wbft·wemix 사본을 합쳤다 |
| CT-NODE-002 | `CT-NODE-002-startup-15-nodes.json` | 세 체인 공통. 2026-09-29 에 wbft·wemix 사본을 합쳤다 |
| CT-NODE-003 | `CT-NODE-003-genesis-init.json` | 2026-09-29 에 genesis-block-hash-consistent·chain-id 를 합쳤다 |
| CT-NODE-004 | `CT-NODE-004-full-sync.json` | 신규(2026-09-29) |
| CT-NODE-005 | `CT-NODE-005-snap-sync.json` | 신규(2026-09-29) |
| CT-NODE-006 | `CT-NODE-006-missing-block-catch-up.json` | 신규(2026-09-29) |
| CT-NODE-007 | `CT-NODE-007-live-block-receive.json` | 신규(2026-09-29) |
| CT-NODE-008 | `CT-NODE-008-peers.json` | 2026-09-29 에 basic-peers·admin-peers-populated 를 합쳤다 |
| CT-NODE-009 | `CT-NODE-009-head-hash-agreement.json` | 2026-09-29 에 basic-sync·basic-consensus 를 합쳤다 |
| CT-NODE-010 | `CT-NODE-010-sync-via-proxy.json` | `bp4-en2-pn1` 기준 공통. 2026-09-29 에 `010-wbft-proxied-routing` 을 합쳤다 |
| CT-NODE-011 | `CT-NODE-011-endpoint-first-layout.json` | |
| CT-NODE-013 | `CT-NODE-013-genesis-mismatch-refused.json` | |
| CT-NODE-014 | `CT-NODE-014-sync-complete.json` | 2026-09-29 에 chain-not-syncing·remote-chain-info 를 합쳤다 |
| CT-NODE-015 | `CT-NODE-015-timestamp-monotonic.json` | 신규(2026-09-29) |
| CT-NODE-016 | `CT-NODE-016-block-period.json` | 2026-09-29 에 block-period-one-second·stress-block-time 을 합쳤다 |

### TX (20파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-TX-001 | `CT-TX-001-value-transfer.json` | 2026-09-29 에 basic-tx-send·sample-minimal-value-transfer·value-transfer 를 합쳤다 |
| CT-TX-002 | `CT-TX-002-legacy-transfer.json` | 신규. stablenet 전용 `legacy-transfer`는 공통 아님(제외) |
| CT-TX-003 | `CT-TX-003-dynamic-fee-transfer.json` | 신규. stablenet 전용 `dynamic-fee-tx`는 제외 |
| CT-TX-004 | `CT-TX-004-access-list-tx.json` | |
| CT-TX-005 | `CT-TX-005-fee-delegated-transfer.json` | |
| CT-TX-006 | `CT-TX-006-fd-sender-sig-tampered-rejected.json` | |
| CT-TX-007 | `CT-TX-007-fd-feepayer-sig-tampered-rejected.json` | |
| CT-TX-008 | `CT-TX-008-feepayer-insufficient-rejected.json` | |
| CT-TX-009 | `CT-TX-009-fee-delegate-sign-rpc-present.json` | |
| CT-TX-010 | `CT-TX-010-fee-delegated-access-list.json` | 신규(2026-09-29) |
| CT-TX-011 | `CT-TX-011-keystore-fee-delegate-sign.json` | 신규(2026-09-29) |
| CT-TX-012 | `CT-TX-012-nonce-ordering.json` | |
| CT-TX-013 | `CT-TX-013-same-nonce-replacement.json` | |
| CT-TX-014 | `CT-TX-014-carry-over-and-replace.json` | 신규(2026-09-29) |
| CT-TX-015 | `CT-TX-015-insufficient-funds-rejected.json` | 세 체인 공통. 2026-09-29 에 wbft·wemix 사본(단계가 같았다)을 합쳤다 |
| CT-TX-016 | `CT-TX-016-gas-limit-exceeds-block-rejected.json` | stablenet 전용 `gaslimit-exceeded-rejected`는 제외 |
| CT-TX-017 | `CT-TX-017-reject-vs-execution-failure.json` | 신규(2026-09-29) |
| CT-TX-018 | `CT-TX-018-txpool-propagation.json` | `bp4-en2-pn1` 기준 공통. 2026-09-30 에 풀 전파를 실제로 재도록 구성을 바꿨다 |
| CT-TX-019 | `CT-TX-019-block-progress-under-load.json` | |
| CT-TX-020 | `CT-TX-020-test-account-funding.json` | |

### FEE (7파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-FEE-001 | `CT-FEE-001-tip-below-min-rejected.json` | 2026-09-30 에 다시 썼다. txpool.nolocals, 충분한 최대 수수료, 팁 1 wei |
| CT-FEE-002 | `CT-FEE-002-min-gas-price-boundary.json` | 2026-09-30 에 형식별 미만·초과 여섯 파일을 합쳤다. 미만 세 건은 계정을 나눠 보낸다 |
| CT-FEE-007 | `CT-FEE-007-effective-gas-price.json` | |
| CT-FEE-008 | `CT-FEE-008-effective-gas-price-across-nodes.json` | |
| CT-FEE-009 | `CT-FEE-009-snap-receipt-gas-price.json` | 신규(2026-09-29) |
| CT-FEE-010 | `CT-FEE-010-suggested-gas-price.json` | stablenet 전용 `gas-price-equals-basefee-plus-tip`는 제외 |
| CT-FEE-012 | `CT-FEE-012-fee-history.json` | |

### CONTRACT (7파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-CONTRACT-001 | `CT-CONTRACT-001-deploy-and-call.json` | 세 체인 공통. 2026-09-30 에 wbft·wemix 사본을 합쳤다 |
| CT-CONTRACT-002 | `CT-CONTRACT-002-storage-write-and-read.json` | |
| CT-CONTRACT-003 | `CT-CONTRACT-003-view-call-leaves-state.json` | 신규(2026-09-29) |
| CT-CONTRACT-004 | `CT-CONTRACT-004-estimate-gas.json` | |
| CT-CONTRACT-005 | `CT-CONTRACT-005-eth-call-revert-returns-error.json` | |
| CT-CONTRACT-006 | `CT-CONTRACT-006-revert-tx-status-zero.json` | 세 체인 공통. 2026-09-30 에 wbft·wemix 사본과 negative-tx-revert 를 합쳤다 |
| CT-CONTRACT-007 | `CT-CONTRACT-007-out-of-gas-consumes-all.json` | |

### RPC (14파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-RPC-001 | `CT-RPC-001-block-number-advances.json` | 2026-09-30 에 remote-rpc-health 를 합쳤다 |
| CT-RPC-002 | `CT-RPC-002-block-transactions-field.json` | |
| CT-RPC-003 | `CT-RPC-003-block-by-hash-consistency.json` | |
| CT-RPC-004 | `CT-RPC-004-transaction-by-hash-fields.json` | |
| CT-RPC-005 | `CT-RPC-005-transaction-receipt-fields.json` | |
| CT-RPC-006 | `CT-RPC-006-transaction-count-increments.json` | |
| CT-RPC-007 | `CT-RPC-007-balance-query.json` | 2026-09-30 에 remote-balance-check 를 합쳤다 |
| CT-RPC-008 | (전용 파일 없음) | `node/CT-NODE-003-genesis-init.json`, `node/CT-NODE-014-sync-complete.json` 와 목적이 겹쳐 공유 |
| CT-RPC-009 | `CT-RPC-009-event-logs-query.json` | 2026-09-30 에 logs-query-well-formed 를 합쳤다 |
| CT-RPC-010 | `CT-RPC-010-signed-tx-seen-in-pool.json` | 신규(2026-09-29) |
| CT-RPC-011 | `CT-RPC-011-txpool-status.json` | |
| CT-RPC-012 | `CT-RPC-012-txpool-content-well-formed.json` | |
| CT-RPC-013 | `CT-RPC-013-ws-subscribe-new-heads.json` | |
| CT-RPC-014 | `CT-RPC-014-ws-subscribe-logs.json` | |
| CT-RPC-015 | `CT-RPC-015-metric-head-block.json` | |

### FAULT (6파일)

| CT | 파일 | 비고 |
| --- | --- | --- |
| CT-FAULT-001 | `CT-FAULT-001-producer-crash-and-restart.json` | 세 체인 공통. 2026-09-30 에 wbft·wemix 사본과 sample-lifecycle 을 합쳤다 |
| CT-FAULT-002 | `CT-FAULT-002-node-recover-and-sync.json` | |
| CT-FAULT-003 | `CT-FAULT-003-two-producers-down.json` | |
| CT-FAULT-004 | `CT-FAULT-004-network-partition.json` | target:remote — 방화벽으로 분리하므로 로컬 실행은 건너뛴다 |
| CT-FAULT-005 | `CT-FAULT-005-hub-topology.json` | |
| CT-FAULT-006 | `CT-FAULT-006-txpool-leader-change.json` | |

---

## 4. 체인별 물리 분리 파일 (0개)

체인 이름을 붙여 따로 둔 파일은 2026-09-30 에 하나도 남지 않았다. 69개 전부를 `stablenet-*`
프리셋 기준으로 쓰고, 실행할 때 `--chain-preset` 과 `--binary` 를 갈아 세 체인을 덮는다.

합친 내역은 이렇다.

**TX 015**(2026-09-29). 사본 둘은 단계가 같고 id·설명·preset 만 달랐다.

**NODE 001·002·010**(2026-09-29). wbft 사본은 preset 이름만 달랐고, wemix 사본은 검증자 수
검사를 뺐다(go-wemix 에 검증자 조회 RPC 가 없어서). 지금은 wemix 검증자를 거버넌스
컨트랙트로 읽으므로 뺄 이유가 없다. 010 의 두 파일은 체인 차이가 아니라 원본 테스트
둘(stablenet-proxied-pn-routing, wbft-proxied-routing)이었다.

**CONTRACT 001·006**(2026-09-30). 체인별 사본 넷은 stablenet 판과 preset·id 만 달랐다. 001 의
사본이 더 들고 있던 송금 두 단계는 CT-TX-001 이 재는 것이라 옮기지 않았고, genesis 에 얹던
node1 잔액도 옮기지 않았다. 006 의 사본이 보던 `blockAdvance` 와 negative-tx-revert 가 보던
배포 코드 확인은 합친 파일로 옮겼다.

**FAULT 001**(2026-09-30). 체인별 사본 둘은 preset·id 와 대기 시간 셋만 달랐다(wemix 가 길다).
다만 남긴 모양은 stablenet 판이 아니라 사본 쪽이다. stablenet 판은 다시 띄운 노드를 고정 높이
8 까지만 기다렸는데, 그 높이는 체인이 이미 지나간 뒤라 아무것도 기다리지 않는다. 사본은
멈춘 사이 나아간 기준 노드의 높이를 읽어 두고 거기까지 오기를 기다린다. 대기 시간은 가장 긴
wemix 값으로 통일했다. 같은 번호에 있던 sample-lifecycle 은 체인 사본이 아니라
`docs/guide/dsl-authoring.md` 가 가리키는 작성 샘플이었다. 그 파일의 송금 세 단계와 binvar
preset 은 옮기지 않았고, `readNodeLog` 만 옮겨 왔다.

---

## 5. 특이사항

- **CT-RPC-008(체인 ID 조회)** 은 전용 DSL 파일이 없다. `node/CT-NODE-003-genesis-init.json`,
  `node/CT-NODE-014-sync-complete.json` 와 목적이 같아 그 파일을 공유한다.
- **stablenet 전용으로 분류돼 공통에서 빠진 자동 테스트**: `legacy-transfer`,
  `dynamic-fee-tx`, `gaslimit-exceeded-rejected`, `feecap-above-min-accepted`,
  `feecap-exact-min-accepted`, `gas-price-equals-basefee-plus-tip`.
  이들은 StableNet 전용 `gasTip` 을 읽어 세 체인
  공통이 아니므로 `tests/tc/common/` 에는 없다(01 문서 각 CT 비고 참조).
- **binvar**: 바이너리를 환경변수로 받는 preset 을 `tests/tc/common/` 에서 쓰는 곳은 2026-09-30
  현재 없다. `wemix-bp4-binvar` 는 `contract/001-wemix-tx-and-contract.json` 이,
  `stablenet-bp4-en1-binvar` 는 `fault/001-sample-lifecycle.json` 이 합쳐지면서 쓰는 곳을 잃었다.
  저장소 전체로는 `tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json` 이
  `wemix-bp4-binvar` 를 쓰고, `stablenet-bp4-en1-binvar` 는 아무 데서도 쓰지 않는다.
- **작성 샘플**: `docs/guide/dsl-authoring.md` 가 가리키는 공통 묶음 안의 샘플 둘은
  `tx/CT-TX-001-value-transfer.json`(값 전송)과
  `fault/CT-FAULT-001-producer-crash-and-restart.json`(노드 중단·재시작)이다.
- 이 매핑의 파일별 실행 명령은 `tests/tc/common/HOW-TO-USE.md` §9 에 정리돼 있다.
