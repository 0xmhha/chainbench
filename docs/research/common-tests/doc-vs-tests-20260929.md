# 공통 테스트 문서와 케이스 대조 (2026-09-29)

기준: 작업 트리(브랜치 `feat/common-test-separation`, HEAD `2b207804` + 커밋 안 된 변경).
문서는 `docs/tc/common/`, 케이스는 `tests/tc/common/` 아래 JSON 82개다. 케이스는 실행하지 않고 읽어서만 판정했다.

## 1. 요약

- 문서 목록의 CT 는 70행이다. 그중 공통 케이스가 없는 CT 는 14개다(자동 테스트가 아예 없는 12개와 go-stablenet 전용만 있는 2개).
- 문서에 없는 공통 케이스는 없다. 문서가 적은 자동 테스트 id 는 모두 실제 케이스 id 와 맞는다. 예외는 `02-tx.md` 가 적은 지워진 이름 5개다.
- 절차나 기대값이 크게 어긋난 것은 CT-FAULT-004, CT-FAULT-005 둘이다. 케이스가 바뀌었는데 문서는 2026-09-11 판 그대로다.
- 문서끼리 수와 목록이 맞지 않는 곳이 많다. CONTRACT-002·005 복귀, FEE-003~006·011 제외, NODE-012 제외가 일부 페이지에만 반영됐다.

## 2. 공통 케이스가 없는 CT (14개)

| CT | 이름 | 비고 |
| --- | --- | --- |
| CT-NODE-004 | Full Sync 동기화 | `tests/e2e/wbft_sync_test.go` 등 Go e2e 가 대응하는지는 확인하지 않았다 |
| CT-NODE-005 | Snap Sync 동기화 | 위와 같다 |
| CT-NODE-006 | 누락 블록 일괄 동기화 | |
| CT-NODE-007 | 새 블록 실시간 수신 | |
| CT-NODE-015 | 블록 시각 단조 증가 | |
| CT-TX-002 | Legacy 트랜잭션 | go-stablenet 전용 케이스만 있다(`regression/ethereum/08-`) |
| CT-TX-003 | 동적 수수료 트랜잭션 | go-stablenet 전용 케이스만 있다(`regression/ethereum/09-`) |
| CT-TX-010 | 접근 목록을 붙인 대납 | |
| CT-TX-011 | 노드 키 저장소 경유 대납 서명 | |
| CT-TX-014 | 미포함 트랜잭션의 이월과 교체 | |
| CT-TX-017 | 거부와 실행 실패의 상태 구분 | |
| CT-FEE-009 | 스냅 동기화 노드의 영수증 가스 가격 보존 | |
| CT-CONTRACT-003 | 조회 함수 호출 | |
| CT-RPC-010 | 서명된 트랜잭션 전송과 풀 조회 | `tx/001-value-transfer` 는 전송만 본다. 풀 조회 단계가 없다 |

CT-TX-002·003 은 목록에 행이 있으나 공통 케이스는 없다. 그래서 위 표는 14행이다. tests README 5절은 이 둘을 빼고 12개로 센다.

부분 구현도 하나 있다. CT-FEE-002 는 "세 형식마다 미만·같음·초과"를 요구한다. 공통 케이스는 미만 세 건뿐이다. legacy 와 접근 목록 형식의 같음·초과는 어디에도 없다.

## 3. 케이스와 문서가 어긋나는 것

### 절차와 기대값이 다른 것

| CT | 문서 | 케이스 | 위치 |
| --- | --- | --- | --- |
| CT-FAULT-004 | 4대를 2대씩 피어를 끊어 가른다. 분리 중 피어 수 2 이하 | 생산 노드 9대를 node1~5 / node5~9 로 선언하고 다리 node5 를 멈춘다. 분리 중 양 끝 피어 수 **3**, 양쪽 블록 멈춤, 복구 뒤 해시 일치 | `06-fault.md:57-61`, `004:12-123` |
| CT-FAULT-005 | 허브 하나만 남기고 연결을 끊는다. 허브에 송금 | bp4·en2·pn1 로 처음부터 허브형. bp1 에 송금하고 en 에서 잔액 확인. 허브 pn1 을 멈추면 bp 는 계속, en 은 멈춤. 되살리면 따라잡음 | `06-fault.md:70-74`, `005:12-155` (커밋 안 됨) |
| CT-FAULT-006 | 노드1 에 송금, 노드1 중단, 노드2 확인 | node3·4 를 먼저 멈춰 합의를 세운 뒤 송금, 풀에 1건 보이면 node1 중단, node3·4 재기동 | `06-fault.md:83-86`, `006:24-103` |
| CT-RPC-011·012 | 합의 정지 단계 없음 | node3·4 를 멈춰 합의를 세운 뒤 트랜잭션을 넣는다 | `05-rpc.md:130-143`, `011:31-44` |
| CT-NODE-013 | 오류 문구 "database contains incompatible genesis", 로그 확인, 나머지 노드 영향 없음 | 부분 문자열 `genesis` 만 본다. 로그는 저장만 한다. node1 하나만 본다 | `01-node.md:158-162`, `013:24-47` |
| CT-NODE-016 | 시각 차이가 설정 주기와 같다(체인별) | one-second 는 1 초 고정, stress 는 평균 2초 이하(하한 없음) | `01-node.md:191-195` |
| CT-TX-001 | 새 계정으로 노드 계정이 송금, 잔액이 금액만큼 증가 | `basic-tx-send` 는 잔액 > 0 만 본다. `value-transfer` 는 고정 주소로 로컬 키 전송 | `02-tx.md:16-20` |
| CT-CONTRACT-006 | 쓴 가스만 차감된다 | wbft·wemix·negative 케이스는 상태 0x0 만 본다. 잔액은 `revert-tx-status-zero` 만 본다 | `04-contract.md:80` |
| CT-FEE-001 | 거부된다 | wbft 에서는 `keptOut`(풀에 들어가지만 포함 안 됨)을 기대한다 | `03-fee.md:19`, `001:27-29` |

### 목적을 검증하지 못하는 것 (tests README 7.2 와 같음)

- `node/010-wbft-proxied-routing`: `blockAdvance` 가 첫 대상 bp1 만 본다. en1 이 올라가는지는 확인되지 않는다.
- `rpc/009-contract-event-emitted`: 필터에 넣은 topic 을 그대로 기대값으로 비교해 늘 참이다.
- `node/010-*`: en1 피어가 1개인지만 본다. 그 피어가 pn 인지는 보지 않는다.
- `node/008-admin-peers-populated`: 첫 피어 id 가 비어 있지 않은지만 본다.

### 문서 비고가 낡은 것 (케이스는 이미 고쳐짐)

- CT-RPC-008: "0보다 큰지만 보므로 설정값 비교로 바꾼다". 지금은 체인별 값(8283/8284/8285)과 비교한다.
- CT-RPC-011: "형식만 보므로 건수 대조로 바꾼다". 지금은 pending·queued 건수를 대조한다.
- CT-NODE-002: "WEMIX3.0 은 검증자 조회가 없어 생략". `validators` 는 wemix 에서도 거버넌스로 답한다(`internal/testhelper/validators.go:12-26`).
- CT-FAULT-003 체인별 차이: 4대 구성에서는 세 체인 판정이 같다고 케이스 description 이 이미 결론을 냈다.
- CT-FEE-010: 목록은 합 검사를 StableNet 전용 케이스 몫으로 적는다. 그런데 공통 케이스도 `eth_maxPriorityFeePerGas` 를 팁으로 써서 합을 검사한다.

### 케이스 메타데이터 (문서 수정 대상 아님)

- 파일명과 id 가 다른 것 6개(README "`<CT 번호>-<id>.json`" 규칙 위반): `fault/001-sample-lifecycle`, `contract/006-negative-tx-revert`, `node/010-proxied-pn-routing`, `node/013-genesis-mismatch`, `tx/001-sample-minimal`, `tx/020-faucet-funds-account`.
- description 이 내용과 다른 것: `fault/002`(동기화 시간 측정 안 함), `node/016-stress`(100블록 → 실제 15), `tx/018`(부하 없음), `tx/019`(처리량 측정 안 함), `contract/006-negative`(송금 단계 없음), `contract/007`(잔액 확인 없음), `node/013`(바이너리가 아니라 genesisOverlay), `node/010-wbft`(없는 경로 참조).
- description 이 없는 것: `contract/001-wemix-tx-and-contract`, `rpc/009-contract-event-emitted`, `node/014-chain-not-syncing`.
- 한 체인 프리셋에 묶인 것: `stablenet-bp4-en1`(node/008·009, tx/001-basic, tx/018), `stablenet-bp3-en1-pn1`, `stablenet-bp3-en1-table`, `stablenet-bp4-en1-snap`(fee/008), `wbft-bp2-en1-pn1`. 다른 두 체인에 같은 모양 프리셋이 없어 `--chain-preset` 으로 덮을 수 없다.
- remote 케이스 셋(`rpc/001-remote`, `rpc/007-remote`, `node/014-remote-chain-info`)은 노드 계정 서명이나 로컬 체인 ID 에 기대 공개 RPC 에서 돌 수 없다.

## 4. 문서끼리 어긋나는 것

| 페이지 | 적힌 것 | 실제 |
| --- | --- | --- |
| 목록 머리말 | 69개, CONTRACT-002·005 는 부록 B 로 옮김 | 70행, 두 CT 는 2026-09-28 복귀 |
| 목록 영역별 수 | NODE 16, CONTRACT 5 | NODE 15, CONTRACT 7 |
| 목록 분리 방식별 수 | 32 / 25 / 12 | 33 / 26 / 11 |
| 요약(부모) 결과 요약 | 69개, FEE 10, CONTRACT 7 | FEE 7 (로컬 사본 v5, Confluence 는 v7) |
| 상세 절차 목차 | 76개, NODE 16, FEE 12 | 70개, NODE 15, FEE 7 |
| 상세 절차 NODE | 머리말 16개 | 15절 |
| 상세 절차 FEE | 12개, FEE-003~006·011 절 그대로 | 제외된 CT 다섯 |
| 상세 절차 TX | 지워진 케이스 이름 5개(`fee-delegated-*` 셋, `out-of-order-nonces-mine`, `same-nonce-replacement`) | a9eec3ae 에서 삭제 |
| 상세 절차 CONTRACT-005 | 분리 방식 "설정으로 분리, 기대값은 체인별 계산" | 목록은 "설정으로 분리" |
| 별도 구현 항목 | 수수료 규칙 대상 CT-FEE-001~006, 011 | 003~006·011 은 제외 |
| 회귀 실행 묶음 | 74개, 묶음 E 에 FEE-003~006·011, 묶음 F 에 NODE-012, 묶음 G 는 생성 4 한 구성 | 제외 CT 포함, FAULT 는 세 구성, RPC-013·015 는 어느 묶음에도 없음, RPC-009 는 A·C 중복 |
| 부록 B 제외 자동 테스트 표 | `eth-call-revert-returns-error` 를 `go-stablenet/regression/ethereum/23-` 로 옮김 | 그 파일은 없다. 케이스는 `common/contract/005-` 에 있다 |
| 부록 B 제외 자동 테스트 표 | — | `feecap-above/exact-min-accepted`, `basefee-minimum`, `gas-price-equals-basefee-plus-tip`, `max-priority-fee-equals-gastip` 이 빠져 있다 |
| tests README | 82개, node 21건, contract 9건, CONTRACT "5개 중 4개", eth-call-revert 는 go-stablenet 으로 | node 20, contract 11, CONTRACT 7개 중 6개 |

## 5. Confluence 갱신안

| 페이지 | ID | 현재 | 고칠 것 |
| --- | --- | --- | --- |
| [Common] Test (요약) | 2987917348 | v7 | 결과 요약의 수(70개, 영역별), 분리 방식별 수. 로컬 사본이 v5 라 먼저 v7 을 받아야 한다 |
| 공통 테스트 목록 | 2988965889 | v8 | 머리말(70개, CONTRACT 두 건 복귀), 영역별 수, 분리 방식별 수, NODE-012 행 삭제(작업 트리에 반영됨), RPC-008·011·NODE-002 비고, FAULT-004·005 기대 결과 |
| 별도 구현이 필요한 항목 | 2987884735 | v3 | 수수료 규칙 대상에서 FEE-003~006·011 삭제, FAULT-004 wemix 재연결 제약 |
| 회귀 실행 묶음 | 2986934428 | v3 | 총수, 묶음 E·F 의 제외 CT 삭제, 묶음 G 세 구성, RPC-013·015 배정, RPC-009 중복 정리 |
| 상세 실행 절차(목차) | 2987196682 | v3 | 총수와 영역별 수 |
| 상세 NODE | 2988539943 | v1 | 머리말 15개, NODE-012 절 삭제(작업 트리에 반영됨), NODE-002 근거, NODE-013 기대 결과 |
| 상세 TX | 2987720880 | v1 | 지워진 케이스 이름 5개 삭제 |
| 상세 FEE | 2987819122 | v1 | 머리말 7개, FEE-003~006·011 절 삭제(부록 B 로 안내) |
| 상세 CONTRACT | 2987786466 | v1 | CONTRACT-005 분리 방식 |
| 상세 RPC | 2988638234 | v1 | RPC-008·011 비고, RPC-011·012 절차에 합의 정지 단계 |
| 상세 FAULT | 2987524207 | v1 | FAULT-004·005·006 절차·기대 결과·체인별 차이 다시 쓰기, FAULT-002·003 보강 |
| 부록 B | 2987524189 | v7 | NODE-012 절 추가(작업 트리에 반영됨), eth-call-revert 행 삭제, go-stablenet 으로 옮긴 자동 테스트 다섯 행 추가 |

### CT-FAULT 다시 쓸 문안

**CT-FAULT-004 네트워크 분리와 복구**

- 절차:
  1. 생성 노드 9대를 두 무리(node1~node5, node5~node9)로 선언해 세운다. node5 는 두 무리에 모두 속한 다리 노드다.
  2. 블록 높이가 3 이상이고 모든 노드의 최신 해시가 같은지 확인한다. 피어 수가 node1 4, node5 8, node9 4 인지 본다.
  3. node5 를 멈춘다. 살아 있는 8대는 정족수 6을 넘으므로, 체인이 멈춘다면 노드가 빠져서가 아니라 네트워크가 갈라져서다.
  4. node1 과 node9 의 피어 수가 3인지, 양쪽 모두 블록이 멈추는지 본다.
  5. node5 를 다시 띄운다. 양쪽이 블록을 다시 만들고, node9 가 node1 높이를 따라잡은 뒤 최신 해시를 비교한다.
- 기대 결과: 분리 중 양 끝 노드의 피어 수는 3이고 양쪽 모두 블록 생성이 멈춘다. 다리 노드를 되살리면 양쪽이 다시 블록을 만들고 모든 노드의 최신 해시가 같다.
- 의존 요소: 노드 직접 제어, 연결 구성 선언
- 체인별 차이: WEMIX3.0 은 거버넌스 멤버에게 30초마다 스스로 다시 연결해 선언한 분리가 유지되지 않는다. (처리 방침 결정 후 문장 확정)

**CT-FAULT-005 허브형 연결에서 합의와 전파**

- 절차:
  1. 생성 노드 4대, 종단 노드 2대, 전달 노드 1대로 세운다. 종단 노드는 전달 노드 하나에만 연결된다.
  2. 피어 수가 en1 1, en2 1, pn1 6, bp1 4 인지 확인한다.
  3. 생성 노드 bp1 에 송금을 보내고, en1 과 en2 에서 받는 계정 잔액이 보이는지 확인한다.
  4. 허브 pn1 을 멈춘다. 생성 노드는 블록을 계속 만들고, en1 피어는 0이 되며, en1 과 en2 는 블록이 멈추는지 본다.
  5. pn1 을 다시 띄운다. en1 과 en2 가 bp1 높이까지 따라잡은 뒤 최신 해시를 비교한다.
- 기대 결과: 송금이 허브를 거쳐 종단 노드까지 전파된다. 허브를 멈추면 생성 노드는 계속 블록을 만들고 종단 노드는 멈춘다. 허브를 되살리면 종단 노드가 따라잡고 모든 노드의 최신 해시가 같다.
- 체인별 차이: 연결을 끊지 않고 처음부터 허브형으로 세우므로 관리 API 가 필요 없다.

**CT-FAULT-006 생산 노드 중단 시 대기 트랜잭션 처리**

- 절차:
  1. 생성 노드 node3·node4 를 멈춰 블록 생성을 세운다.
  2. node1 에 송금을 보내고, node2 풀에 대기 건이 1건 보일 때까지 기다린다.
  3. node1 을 멈추고 node3·node4 를 다시 띄운다.
  4. node2 에서 영수증 상태, 받는 계정 잔액, 풀 대기 건수를 확인한다.
- 기대 결과: 송금이 성공(0x1)으로 처리되고 잔액이 보낸 만큼이며 대기 건수가 0이다.
- 비고: 블록 주기가 1초라 합의를 세우지 않으면 node1 을 멈추기 전에 이미 포함된다.

## 6. 공통 케이스가 없는 CT 검토 (2026-09-29)

14개와 CT-FEE-002 의 같음·초과를 더한 15건을 세 노드 소스(go-wemix 902f9fce8, go-wbft 172a1302d, go-stablenet 740526d03)와 DSL 로 대조했다. 실행하지 않고 읽어서 판정했다.

### 6.1 판정

| CT | 판정 | 근거 요약 |
| --- | --- | --- |
| CT-NODE-004 Full Sync | 누락 (지금 구현 가능) | full 경로가 세 체인 같다. "새 노드" 대신 멈췄다 다시 띄운 뒤처진 노드로 확인한다 |
| CT-NODE-005 Snap Sync | 도구 기능 추가 후 가능 | 세 노드 모두 head 가 0 일 때만 snap 을 유지한다(`eth/handler.go`). 블록을 하나도 받지 않은 노드를 늦게 띄울 수단이 없다 |
| CT-NODE-006 누락 블록 일괄 동기화 | 누락 (지금 구현 가능) | downloader 구조와 metric 이름(`eth_downloader_headers_in`)이 세 체인 같다 |
| CT-NODE-007 새 블록 실시간 수신 | 누락 (지금 구현 가능) | fetcher 경로가 같다. 받는 방식(전체 블록/해시 알림)은 피어 수에 따라 갈린다 |
| CT-NODE-015 블록 시각 단조 증가 | 누락 (지금 구현 가능) | 세 엔진 모두 부모 시각 이상을 보장한다. wemix 는 같은 초 블록이 있어 "이상" 이어야 한다 |
| CT-TX-002 Legacy | 도구 기능 추가 후 가능 | 로컬 키 `sendTx` 가 `gasPrice` 를 버리고 0x2 로 보낸다. 지갑의 `SendLegacyGas` 는 DSL 에서 부르는 곳이 없다 |
| CT-TX-003 동적 수수료 | 누락 (지금 구현 가능) | 적용 가격이 세 체인 모두 `min(tip + baseFee, feeCap)` 꼴이다. tip 을 `eth_maxPriorityFeePerGas` 로 잡으면 한 식이 된다 |
| CT-TX-010 접근 목록 대납 | 누락 (지금 구현 가능) | `eth_signRawFeeDelegateTransaction` 본문이 세 체인 같다. 노드가 조립해야 원래 버그 경로(`SetSenderTx`)를 탄다 |
| CT-TX-011 키 저장소 대납 서명 | 정상 경로는 누락, 입력 형식 거부는 도구 기능 추가 후 가능, 타입 불일치 가드는 공통 아님 | `checkFeeDelegateTx` 가 go-wemix 에 없다 |
| CT-TX-014 미포함 트랜잭션 이월과 교체 | 누락 (지금 구현 가능) | 블록 크기 한도는 go-wemix master 에만 있다. 가스 한도로 넘치게 하면 공통이다 |
| CT-TX-017 거부와 실행 실패 구분 | 누락 (지금 구현 가능) | 거부(CT-TX-015)와 되돌림·가스 소진(CT-CONTRACT-006·007)을 한 계정에서 나란히 대조한다 |
| CT-FEE-002 같음 | 공통 아님 | 경계가 go-stablenet 고정 상수, go-wemix 다음 블록 baseFee + 거버넌스 tip, go-wbft 없음(RPC 트랜잭션)으로 다르다 |
| CT-FEE-002 초과 | 누락 (지금 구현 가능) | 세 형식 모두 `eth_gasPrice` 또는 `baseFee + eth_maxPriorityFeePerGas` 로 보내면 받아들여진다 |
| CT-FEE-009 스냅 노드 영수증 가스 가격 | 도구 기능 추가 후 가능 | CT-NODE-005 와 같다. 목적 문장("덮어쓰지 않는지")은 StableNet 내부 구현이라 고쳐야 한다 |
| CT-CONTRACT-003 조회 함수 호출 | 누락 (지금 구현 가능) | 값 확인은 contract/001·002 가 한다. "상태 불변" 확인이 없다 |
| CT-RPC-010 서명 전송과 풀 조회 | 누락 (지금 구현 가능) | 합의를 세운 뒤 보낸 해시가 풀에 있는지, 포함 뒤 영수증이 있는지 본다 |

### 6.2 필요한 도구 기능

1. 블록을 하나도 받지 않은 노드를 나중에 띄우는 수단 (CT-NODE-005, CT-FEE-009)
2. 로컬 키 legacy 전송: `sendTx` 에 `key` 와 `gasPrice` 가 같이 오면 0x0 으로 보낸다 (CT-TX-002)
3. 임의 RPC 호출이 특정 오류로 실패해야 함을 확인하는 검사 (CT-TX-011)

### 6.3 검토 중 나온 문제

- "snap" 이라 부르는 기존 테스트는 snap 경로를 타지 않았다. 세 노드는 head 가 0 보다 크면 snap 을 full 로 바꾼다. `stablenet-bp4-en1-snap` 프리셋 케이스와 Go e2e 동기화 테스트가 여기에 걸린다.
- `sendTx` 가 필드를 조용히 버린다. `key` + `gasPrice` 는 gasPrice 를, `feePayerKey` 는 accessList·gas·data 를 버린다.
- CT-FEE-002 의 체인별 차이 문장은 RPC 로 보낸 트랜잭션에 맞지 않는다. go-stablenet 은 MinBaseFee 만, go-wemix 는 baseFee + tip 을 검사하고, go-wbft 는 제출 때 거부하지 않는다.
- CT-NODE-006·007 의 "로그 문구가 체인마다 다르다" 는 틀렸다. 세 체인이 같은 문구와 metric 이름을 쓴다.
- `go-stablenet/regression/anzeon/09-feecap-exact-min` 은 RPC 트랜잭션의 실제 경계(MinBaseFee)가 아니라 `baseFee + gasTip` 으로 보낸다. 경계값을 재지 못할 수 있다(미실행).

### 6.4 결정 (2026-09-29, 사용자)

- 공통 아닌 두 부분(CT-FEE-002 의 같음, CT-TX-011 의 타입 불일치 가드)은 공통 목록에서 뺀다.
- 도구 기능 셋을 이번에 추가한다.
- 빠진 케이스를 지금 구현하고 문서도 정리한다.
- 테스트 로직, 로컬 문서, Confluence 가 같은 내용이 되게 한다.
