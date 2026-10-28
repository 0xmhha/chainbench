# Stablenet 노드·검증 경계 실사

## 결론과 확신 수준

기준선의 `recipient not credited: 0 -> 0` 실패는 사실이다. 그러나 **송금이 영구적으로 누락됐다는 증거는 아니다.** 원본 실패 데이터의 복사본을 실제 gstable로 열어 보니 조회 대상 node1의 block 1 상태에는 수신 잔액 1 ETH가 저장되어 있었다. 실패 당시 receipt와 latest head/state를 동시에 기록하지 않았으므로 원래 순간의 직접 원인은 확정할 수 없다.

가장 유력한 조사 방향은 **receipt 노출과 latest 상태 관찰 사이의 시점 경계**다. 코드상 읽기 경로가 다르고 원자적 스냅샷을 공유하지 않는다. 이 가능성을 노드 결함으로 단정하지 않는다. 새 네트워크 진단에서는 같은 실패가 재현되지 않았다.

## 실제 노드 확인

기준선과 같은 gstable 바이너리 SHA-256을 확인했다(`inputs.json`). 보존된 `/tmp/cbe2e3643186464/node1..5`를 별도 `/tmp/sba-*`로 복사하고 각 노드를 순차 기동했다. 원본 DB는 열거나 변경하지 않았다. 복원 노드는 외부 peer 연결 없이 `maxpeers=0`, discovery/IPC 비활성화, localhost RPC로 조회했다. 각 노드는 조회 후 종료했다.

원본 거래: `0x0413b32b1eba2f3aff315b76a5b8e99e9a2bed7b31a5d364ea5282765b4d69e8`.

| 복원 노드 | latest 높이 | 수신 잔액 | 원본 거래 receipt |
|---|---|---|---|
| node1 | 1 | 1 ETH | status 0x1, block 1 |
| node2 | 0 | 0 | 없음 |
| node3 | 1 | 1 ETH | status 0x1, block 1 |
| node4 | 1 | 1 ETH | status 0x1, block 1 |
| node5 | 0 | 0 | 없음 |

node1·3·4의 receipt blockHash는 `0x364f707d64d5abfde92c94be63f8600d68242eb80d76680ae089b1b56a36eab5`로 일치한다. `latest`와 명시적인 `0x1` 잔액 조회가 모두 1 ETH였다. node2·5의 block 1 조회는 header not found였다. 이는 종료 시 보존된 진척 차이이며 영구적인 동기화 장애를 입증하지 않는다. 원문 응답은 `restored-state.json`에 있다.

새 4-validator/1-endpoint 네트워크에서는 Go overlay로 관측만 추가했다. 송금 receipt 이후 30회 × 5개 노드의 head·잔액·receipt를 수집했고 150개 잔액 응답 모두 1 ETH였다. 컨트랙트 배포/42 반환까지 진단 테스트가 통과했다. `live-probe.log`, `live-summary.json` 참조. 추가 RPC는 시점을 바꿀 수 있으므로 이 PASS로 원본 실패를 지우지 않는다.

## 책임 경계별 판단

| 경계 | 확인한 사실 | 후속 설계·검증 요구 |
|---|---|---|
| 구성 → 실행 가능한 네트워크 | `head()`는 RPC 오류를 -1로 반환하고 `waitAdvancing()`은 이후 값이 더 크기만 하면 성공한다. 따라서 오류 뒤 genesis 0을 블록 증가로 오인할 수 있다. 원본 실행에서 실제로 그 경로를 탔는지는 미기록이다. | RPC 성공·chain ID·genesis identity를 먼저 확인하고, 성공한 높이끼리 증가를 판단. readiness와 consensus progress를 분리. |
| accounts 송금 → RPC 관측 | 원본 노드 로그와 복원 transaction의 목적지·value가 요청과 일치하고 저장 상태에 1 ETH가 있다. 송금 인코딩/주소 오류를 뒷받침하는 증거는 없다. | tx hash/to/value를 receipt·상태 관측에 연결. |
| receipt → canonical state | 하네스 `waitReceiptSuccess()`는 status만 검사한다. 뒤의 `balance()`는 block 지정 없이 latest를 즉시 한 번 읽는다. | receipt의 blockNumber/blockHash를 보존하고 같은 canonical block의 state로 검증. latest 검사에는 유한한 대기와 관측 이력, reorg/hash 불일치 판정을 둔다. 잔액이 나올 때까지 무조건 기다리는 것으로 의미를 대체하지 않는다. |
| Stablenet RPC 내부 | `GetTransactionReceipt()`는 transaction lookup/receipt를 읽고 `GetBalance()`는 latest header의 state root를 읽는다. `writeHeadBlock()`은 DB batch 이후 in-memory currentBlock을 갱신한다. | 두 독립 RPC의 응답을 원자적이라고 가정하지 않는다. 원래 실패의 동시 head/receipt/state 증거가 없으므로 race는 후보로만 유지. |
| 노드 간 합의/전파 | 보존 DB 높이가 1/0/1/1/0으로 달랐다. 새 관측에서는 5개 노드의 송금 상태가 일치했다. | 단일 노드 receipt와 전체 네트워크 수렴을 별도 검증. same-height blockHash/stateRoot와 node identity를 기록. |
| 실패 → 정리 | 기존 잔액 실패는 receipt hash/head/balance의 묶음 진단을 남기지 않는다. `stop()`은 종료 오류를 무시한다. | 실패 직전 노드별 관측을 보존하고 정리 결과·잔여 PID를 별도 기록. 오류를 성공으로 세지 않는다. |

코드 근거: `tests/e2e/harness_test.go`의 head/waitAdvancing/sendValue/waitReceiptSuccess/balance/stop, `internal/core/rpc/client.go:BalanceAt`, 참조 체인의 `internal/ethapi/api.go:GetTransactionReceipt/GetBalance`, `eth/api_backend.go:StateAndHeaderByNumber`, `core/blockchain.go:writeHeadBlock`. 소스 해시는 `inputs.json`에 기록했다.

## 적용 범위

이번 실사는 노드 복원·라이브 관측·원본 테스트 재실행과 Seed 반영이다. 제품 코드와 체인 소스를 수정하지 않았다. 복원 DB 진척, 새 실행, 최초 실패를 서로 다른 증거로 유지한다. 외부 체인 저장소의 수정 계획으로 확대하지 않는다. 원본 기준선은 그대로 두고 이 보고서를 후속 증거로 연결한다.

추가 규명에 필요한 것은 실패 순간의 receipt blockHash/number, 동일 노드 latest header/state, 명시적 block state, 다른 노드의 동일 높이 관측이다. 현재 자료만으로 Stablenet의 영구적 상태 결함이나 특정 race를 확정할 수 없다.

## 원본 재실행 및 마무리

관측 overlay 없이 원본 `TestE2E_StablenetChain`도 `-count=1 -tags e2e -timeout=4m`으로 재실행하여 PASS(42.47초)를 확인했다(`unmodified-rerun.log`). 따라서 재현은 비결정적이며 원본 실패를 안정적으로 재현한 것은 아니다. 원본 FAIL, 진단 PASS, 원본 재실행 PASS를 모두 보존한다. 종료 후 남은 Stablenet/복원 노드 검색 결과는 `remaining-processes.log`에 기록했다.

Seed v1.0.3에 실사 근거와 경계별 검증 요구를 반영하고 Ouroboros 스키마 검증을 통과했다. 이 버전의 QA는 재실행하지 않았으므로 v1.0.2의 0.92 점수를 전용하지 않는다.
