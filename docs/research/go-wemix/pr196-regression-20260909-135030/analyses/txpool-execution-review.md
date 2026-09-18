# 트랜잭션 풀 건수와 교체 테스트 구현 검토

현재 go-wemix 체크아웃 902f9fce8과 chainbench 구현을 읽어 정리한 설계안이다. 테스트 코드와 노드를 변경하거나 실행하지 않았다. 실행할 패치 버전에서 수수료 정책과 교체 설정을 다시 확인해야 한다.

## RT-A-2-02

기존 항목 자체가 type 0x2의 전송 및 성공 처리를 확인한다. 별도 type 0x2 항목을 만들지 않고 같은 ID로 전송, 블록 포함, 조회 type, receipt status, nonce, 수신 금액, 실제 수수료를 함께 검증한다. 다른 유형과의 통합은 이번 변경에 포함하지 않는다.

## RT-G-4-02 / RPC-018: 정확한 건수를 확인하는 방법

동작 중인 체인의 전체 pending 수를 특정 값으로 유지하는 방식은 경쟁 조건이 있다. 블록을 생성하지 않는 전용 EN을 동기화한 뒤 모든 피어와 분리하고, 외부 입력이 없는 고정 상태에서 RPC를 검증하는 방식을 권장한다.

1. 테스트 계정 A/B에 자금을 지급하고 EN에 반영된 블록과 잔액을 확인한다. 계정의 최신 확정 nonce를 각각 a/b로 조회한다.
2. EN의 모든 피어를 분리한다. 자동 피어 탐색·재연결을 막고 peerCount=0, 블록 해시 불변, 자체 블록 생성 없음, txpool의 초기 pending=0/queued=0을 확인한다. EN은 계속 실행해야 한다.
3. A에서 nonce a, a+1을 전송한다. B에서는 b를 보내지 않고 b+1, b+2를 전송한다. 모든 전송은 receipt를 기다리지 않는다. 유효한 수수료, 충분한 잔액, 풀 용량을 확보한다.
4. txpool_content에서 A 두 건은 pending, B 두 건은 queued임을 해시와 nonce로 확인한다. 풀의 비동기 승격이 끝날 때까지 조건을 조회한다.
5. txpool_content → txpool_status → txpool_content 순으로 같은 EN에서 읽는다. 양쪽 content의 해시·nonce·분류가 동일하고 블록과 피어 상태도 유지되었는지 확인한다. status를 16진수에서 정수로 변환해 pending=2, queued=2와 비교한다. content는 계정 수가 아니라 각 계정 아래 트랜잭션 수를 합산한다.
6. B의 nonce b를 전송하고 안정된 상태에서 pending=5, queued=0을 검증한다. nonce 공백이 채워지면서 queued가 pending으로 승격되는 동작도 확인한다.
7. 격리를 해제하고 트랜잭션의 블록 포함을 확인한다. 실패하더라도 정리 절차에서 네트워크 연결을 복원한다.

이 테스트의 pending은 해당 EN의 실행 가능한 풀 항목을 뜻한다. EN의 격리는 채굴 노드가 그 트랜잭션을 받지 못하게 하므로 선처리를 방지한다. 단순 miner_stop 호출이나 낮은 수수료만으로 미처리를 보장하지 않는다. 격리·풀 안정화 조건을 만들지 못하면 정확한 건수 검증은 미수행으로 기록한다.

chainbench의 partition/healPartition을 재사용할 수 있으나, partition은 지정된 노드 사이의 현재 연결을 끊는 기능이다. 모든 피어를 그룹에 포함하고 자동 재연결 방지 조건을 확인해야 한다. 일반 트랜잭션 유입이 있는 공유 환경은 이 정확한 건수 테스트에 사용하지 않는다.

## N-008: queued 상태에서 교체를 검증하는 방법

교체 대상이 먼저 처리되지 않도록 nonce 공백을 사용한다. 체인 전체의 블록 생성을 중단할 필요는 없다.

1. 자금 지급을 완료한 전용 계정의 최신 확정 nonce n을 조회하고, 해당 계정에 기존 pending/queued가 없는지 확인한다.
2. nonce n은 전송하지 않는다. nonce n+1의 원본 T1(type 0x2)을 전송하고 해시를 저장한다. wait:false는 receipt 대기를 생략할 뿐 처리 자체를 막지 않는다. 처리를 막는 조건은 nonce n의 부재다.
3. 같은 노드의 txpool_contentFrom 또는 txpool_content에서 T1이 queued에 들어간 것을 확인한다. 계정 nonce=n, T1 receipt=null도 확인한다. 조건이 충족되기 전에는 교체를 제출하지 않는다.
4. 같은 송신자·nonce·수신자·금액·data·gasLimit을 사용하고 maxFeePerGas와 maxPriorityFeePerGas를 인상한 T2를 새로 서명해 같은 RPC에 전송한다.
5. queued의 n+1에 T2가 있고 T1이 제거되었는지 확인한다. 계정의 queued 건수는 교체 전후 동일해야 한다. 이 시점에 T1과 T2 receipt가 모두 없어야 한다.
6. queued 트랜잭션의 자동 전파를 전제로 하지 않는다. 같은 서명된 T2를 실제 블록 생성 노드들에 명시적으로 전달하고, 각 풀의 n+1에 T2가 있는지 확인한다. T1을 보유했던 노드는 T1 제거도 확인한다. 모든 확인이 끝난 뒤 nonce n의 선행 트랜잭션 T0를 전송한다.
7. T0와 T2의 성공 receipt, nonce 순서, 수신 금액과 수수료를 확인한다. T2가 확정 체인에 포함된 후 T1 receipt가 없고 동일 nonce에 T2만 포함되었는지 확인한다. 교체가 끝나기 전에 nonce n이 처리되었다면 환경 조건 실패로 구분한다.

교체 거부도 별도 세부 조건으로 확인할 수 있다. tip만 올리거나 fee cap만 올린 후보를 보내 교체 거부와 T1 유지를 확인한 다음, 두 값을 모두 올린 T2로 성공을 검증한다. 같은 raw transaction 재전송에 따른 already known과 교체 수수료 부족 오류를 구분한다.

### 사용할 기능과 수수료 계산

- 권장 기능: 동일 nonce의 새 트랜잭션을 로컬 서명하여 eth_sendRawTransaction으로 전송. chainbench의 sendTx에서 key, nonce, maxFeePerGas, maxPriorityFeePerGas, wait:false를 활용할 수 있다.
- go-wemix의 기본 PriceBump는 10이지만 실제 실행 설정을 기준으로 한다. 기존 값 x와 인상률 p에 대해 새 값은 max(x+1, floor(x*(100+p)/100)) 이상으로 정하고, fee cap과 tip cap 각각 적용한다. 추가로 fee cap >= tip cap, 체인의 최소 수수료, 계정 잔액 및 RPC 수수료 상한을 만족해야 한다. 계산에는 부동소수점이 아닌 큰 정수를 쓴다.
- type 0x2에는 maxFeePerGas와 maxPriorityFeePerGas를 사용한다. gasLimit을 늘리는 것은 교체 수수료 인상이 아니다. Legacy의 gasPrice와 혼합하지 않는다.
- eth_resend는 이번 queued 교체의 기본 수단으로 사용하지 않는다. 현재 구현은 GetPoolTransactions의 pending만 검색하며 gasPrice/gasLimit 인자를 사용한다. queued type 0x2 교체는 재서명이 명확하다.

### 블록 크기로 인한 미포함 검증과 구분

nonce 공백은 queued 교체를 검증하지만 PR196의 블록 크기 제한으로 미포함된 트랜잭션의 이월을 검증하지는 않는다. N-008 안에서 두 세부 조건을 분리해야 한다.

- queued 교체: 위 절차를 사용한다.
- 크기 제한 이월: nonce가 연속인 유효 트랜잭션을 충분히 공급하고, 첫 블록의 크기 제한으로 미포함된 해시가 풀에 남아 후속 블록에 포함되는지 검증한다. 가스·트랜잭션 수·생성 종료 조건 때문에 빠진 결과와 구분할 근거가 필요하다. 첫 블록 직후 풀 상태를 잡을 제어/관찰 기능이 없다면 이 세부 조건은 별도 구현 필요로 남긴다. queued 교체 성공으로 대체하지 않는다.

## 재사용 범위와 추가 구현

기존 17b-same-nonce-replacement.json은 nonce 1 두 건을 보낸 후 nonce 0을 보내는 골격을 이미 갖고 있다. 다만 StableNet 설정과 고정 수수료를 사용하고, 교체 전 queued 확인 및 생성 노드별 전파 확인이 없다. WEMIX 실행 환경, 실제 nonce·수수료 조회, 단계별 풀 검증을 추가해야 한다.

18-txpool-status.json은 설명과 달리 현재 pending/queued 응답이 16진수 형식인지 확인할 뿐, 정확한 건수를 검증하지 않는다.

추가할 보조 기능 후보는 다음과 같다. 아래 이름은 구현 제안이며 현재 사용 가능한 DSL 명령이 아니다.

- 풀 상태 대기: 계정·nonce·해시·pending/queued 분류를 조건으로 조회하고 제한 횟수/기한 안에 충족하지 않으면 실패.
- 풀 스냅샷 비교: 전체 트랜잭션 수 집계, 교체 전후 해시 변경, 조회 사이 블록·피어·content 불변 여부 확인.
- 수수료 인상 계산: 실제 PriceBump와 체인 수수료 정책을 받아 두 cap을 정수 계산.
- 생성 노드별 교체 확인 및 실패 시 정리: 선행 nonce 전송 전 교체 전파 확인, 격리 복원 보장.

## 확인한 코드

- go-wemix/core/tx_list.go:281 — 같은 nonce 교체 조건, 두 cap 증가 및 인상률 검사.
- go-wemix/core/tx_pool.go:182,845,1446 — 기본 PriceBump, queued 교체, nonce 연속성에 따른 승격.
- go-wemix/internal/ethapi/api.go:241,2122 — txpool_status와 eth_resend.
- go-wemix/eth/api_backend.go:253 — GetPoolTransactions는 pending만 반환.
- chainbench/internal/testhelper/builtins.go:152,251 — sendTx와 명시적 nonce·수수료 및 wait:false.
- chainbench/internal/accounts/wallet.go:293 — type 0x2 로컬 서명 후 raw 전송.
- chainbench/internal/testhelper/fault.go:360 — partition의 범위와 재연결 주의점.
- chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json — nonce 공백 기반 기존 교체 테스트.
- chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json — 현재 형식 검증 범위.
