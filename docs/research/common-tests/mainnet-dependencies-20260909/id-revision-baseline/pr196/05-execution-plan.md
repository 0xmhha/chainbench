# 패치 적용 후 실행 순서와 판정 기준

이번 작업에서는 테스트를 실행하지 않았다. 아래는 적용할 패치와 환경을 확정한 뒤 사용할 실행 계획이다. 기존 테스트와 신규 제안을 구분하며, 단순 RPC 성공을 회귀 검증 통과로 세지 않는다.

## 1. 통합 대상과 비교 기준 고정

최종 통합 SHA, 빌드 옵션, Go 버전, 바이너리 SHA-256, genesis와 fork 높이, producer/EN 구성, etcd 구성, gas 한도, txpool 설정, PrefetchCount를 기록한다. 같은 환경에서 패치 전·후를 비교한다. PR head를 곧바로 로컬 dev의 대체물로 사용하지 않는다.

로컬 dev에 있는 `TestTimeItTimestampLowerBound`와 `TestSkipMiningTokenAcquisitionWhenWorkerStopped`를 통합 코드에서 보존해야 한다. PR 자체의 변경과 다른 브랜치에서 이미 수정한 부분을 구분한다. 두 테스트는 원본 263개에 포함되지 않은 **기존 Go 테스트**이며 신규 구현 16건과도 구분한다.

## 2. 기존 Go 테스트 먼저 확인

실제 통합 checkout에서 다음 5개 테스트가 존재하는지 먼저 확인한다. `go test -run`은 이름이 없어도 성공할 수 있으므로 로그의 PASS만 보지 않는다.

```bash
go test ./core -list '^TestValidateBodyBlockOversized$'
go test ./miner -list '^(TestCommitTransactionsBlockSizeLimit|TestCommitTransactionsSimpleBlockSizeLimit|TestTimeItTimestampLowerBound|TestSkipMiningTokenAcquisitionWhenWorkerStopped)$'
```

5개 이름이 모두 확인되면 해당 테스트를 실행하고 결과를 보존한다.

```bash
go test ./core -run '^TestValidateBodyBlockOversized$' -count=1
go test ./miner -run '^(TestCommitTransactionsBlockSizeLimit|TestCommitTransactionsSimpleBlockSizeLimit|TestTimeItTimestampLowerBound|TestSkipMiningTokenAcquisitionWhenWorkerStopped)$' -count=1
```

PR 테스트 3개는 Legacy tx 위주의 직접 함수 검증이다. 최종 WEMIX 블록의 RLP, 실제 etcd token, 메시지 상한, historical/snap 호환성은 증명하지 않는다. 생성·재생성·인코딩·GetBlockBodies/GetReceipts 등의 기존 harness와 관련 패키지 테스트를 이어서 실행할 범위는 `pr-impact.md`에 있다.

## 3. 격리 환경 기동과 기존 JSON 5개

먼저 go-wemix chain-up으로 환경을 확인하고 tx/contract, node-crash, Brioche를 실행한다. 15노드 테스트는 실제 13 BP+2 EN 배치와 바이너리 경로를 확인한 뒤 실행한다. 다른 체인의 handoff는 PR196 필수 테스트로 자동 편입하지 않는다.

현재 chainbench 진입점은 `chainbench run [spec.json ...]`이다(`sources/chainbench/cmd/chainbench/suitecmd/run.go:55`). 예를 들어 원본 chain-up에는 다음 형태를 사용할 수 있다. 바이너리와 preset key 경로가 준비된 chainbench 저장소에서 실행하며, 기존 세션과 겹치지 않는 workspace를 지정한다.

```bash
GWEMIX_BIN=/absolute/path/to/patched/gwemix chainbench run \
  tests/tc/go-wemix/chain-up/01-wemix-chain-up.json \
  --workspace-dir /private/tmp/pr196-chain-up
```

이 명령은 실행 예시이며 이번에 시험하지 않았다. JSON 89개 이식 후보는 env.chain, applicableChains, 참조 EN 노드, 바이너리, 계정, 합의와 기대값을 조정한 별도 테스트가 필요하다. 기존 원문을 일괄 변경하지 않는다.

## 4. 추가 P0 7건을 구현·검증

N-001~N-007을 [추가 테스트 문서](additional-tests.md)의 입력과 기대 결과대로 준비한다. 단위 검증이 충분한 곳은 L1/L2로 시작하고, 실제 token·전파·혼합 배포는 L3로 마무리한다. N-004에서 기존 체인에 과대블록이 있는지와 replay 정책을 확인하기 전 호환성이 유지된다고 결론 내리지 않는다.

실행 전 정책 결정이 필요한 항목은 명시적으로 남긴다. 예를 들어 historical block 처리와 mixed-version 배포 절차는 “테스트가 알려 줄 것”이라는 이유로 성공 기준을 비워 두면 안 된다.

## 5. 기존 목록 P0 → P1 → P2 → P3와 추가 P1/P2

[우선순위 목록](02-wemix-priority.md)을 따라 수행하되 기능이 겹치는 문서와 JSON을 별개 실행 횟수로 강제하지 않는다. [교차참조](source-crosswalk.md)의 partial 항목은 빠진 assertion을 채워야 문서 기대 결과를 통과했다고 할 수 있다.

기존 high-level 시나리오만 수행한 결과와 추가 RLP/메시지 경계를 검증한 결과를 분리한다. stress-tx-flood의 작은 gas workload를 8 MiB 경계 재현으로 대체하지 않는다.

## 공통 결과 기록

각 테스트에는 catalog/N ID, 바이너리 SHA, 조건, 실제 실행 경로, 결과, 증거 파일, 실패 이유를 기록한다. 다음 관측값을 필요한 항목에서 함께 비교한다.

- 실제 `rlp.EncodeToBytes(block)` 길이와 wire message 길이.
- 고정된 동일 높이의 block hash, stateRoot, receiptsRoot.
- tx hash, sender nonce, receipt와 포함 횟수. 의도된 다음 블록 이월을 실패로 세지 않는다.
- pending/queued 상태, token 소유·반납, 다음 높이 생산과 장애 복구 시간.
- 거부된 블록·거래가 상태를 바꾸지 않았는지와 정상 peer/블록 처리 회복.

최종 승인 조건은 “모든 명령의 exit code가 0”만이 아니다. 예상한 테스트와 경로가 실제 실행되었고, 바뀌어야 할 과대블록 처리는 바뀌었으며, 정상 블록과 실행 상태가 보존되어야 한다.
