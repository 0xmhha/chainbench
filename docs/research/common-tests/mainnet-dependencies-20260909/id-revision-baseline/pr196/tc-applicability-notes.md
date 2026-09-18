# tests/tc 적용 판단과 실제 검증 범위

JSON 189개를 파싱했고 189개 모두 `tc-catalog.json` 및 `tc-catalog.md`에 한 행씩 포함했다. README/SPECS/CHAIN-BRINGUP 문서 3개는 별도로 나열했다. 동명/유사 케이스도 삭제하지 않았다. 실행 결과가 아닌 정적 검토이며, direct를 포함한 전 항목은 **implemented; execution_unverified** 상태다.

| 적용 구분 | 수 | 의미 |
|---|---:|---|
| direct | 5 | 원문 env.chain=wemix. 바이너리·키·실행 환경을 준비한 뒤 실행할 수 있는 구현 |
| adapt | 89 | 테스트 의도는 가져올 수 있으나 환경/정책/기대조건 수정 필요 |
| conditional | 3 | handoff 바이너리, PN routing 지원 또는 faucet 자금원 등 추가 확인 필요 |
| excluded | 92 | 이번 go-wemix PoA PR 회귀 범위에 원문 그대로 적용 불가 |

우선순위는 P0 0개, P1 65개, P2 27개, P3 97개다. 이 수는 중복 제거 전 파일 수다. 기동과 기본 동기화 테스트는 P1 선행 점검이며 **PR196 재현/해소 증거를 대신하지 않는다**.

## 바로 실행 후보 5개

| 우선순위 | 파일 | 준비 및 검증 한계 |
|---|---|---|
| P1 | go-wemix/chain-up/01-wemix-chain-up.json | gwemix/PoA·etcd 부트스트랩, preset 키. 블록2·chainId·peer만 검증 |
| P1 | go-wemix/chain-up/02-wemix-chain-up-15.json | 13 BP+2 EN 및 해당 docker 바이너리 경로. 전 노드 동일 hash/생산자 참여를 보장하지 않음 |
| P1 | go-wemix/fault/01-wemix-node-crash.json | node3 중단 후 node1 생산 지속. 실제 leader 선택 및 restart 후 catch-up assert 없음 |
| P1 | go-wemix/tx/01-wemix-tx-and-contract.json | genesis funded account. 값 전송, 배포, 코드·return 값만 검증 |
| P2 | go-wemix/rpc/01-wemix-brioche-block-reward.json | Brioche overlay. 블록5 대비15 보상 RPC 감소 검증; 실제 지급 잔액/전체 경계 검증 아님 |

원문은 `../sources/chainbench/tests/tc/` 아래에 있다. WEMIX 플러그인은 PoA family를 사용한다(`sources/chainbench/internal/chains/wemix/wemix.go:34`). etcd 멤버는 producer들로 구성해야 하며 WBFT validator를 섞으면 안 된다는 구현 설명이 있다(`sources/chainbench/internal/consensus/poa/wemixconfig.go:12`). 따라서 generic 파일명이라도 env.chain=stablenet인 케이스를 direct로 판정하지 않았다.

재검토에서 11개 공통 기능을 이식 후보로 복원했다. 원래 Anzeon/WBFT 의미를 지원한다는 판정은 아니다. 세부 근거는 [재검토 결과](recheck-summary.md)에 있다.

## 이식 시 보존해야 할 조건

- 실행 대상: `env.chain`과 top-level `applicableChains`에 wemix를 반영한다. `on=en1`을 참조하는데 EN 정의가 없는 12개 항목은 `env.topology.en=1` 준비가 필요하다. SKIP을 PASS로 세지 않는다.
- 자금/서명: genesis alloc, sender/fee payer, chainId, unlock 또는 서명 경로를 실제 PoA 환경에 맞춘다. 고정 Stablenet 계정의 존재를 가정하지 않는다.
- 수수료: Istanbul GasTip 읽기를 go-wemix 정책/RPC로 대체한다. MinTip·baseFee·gasPrice·replacement bump를 혼동하지 않는다.
- 거부 테스트: 원문 `expect=reject`는 호출 실패를 기대한다. 연결 장애나 계정 부재로 인한 실패를 정상 검증으로 세지 말고 목적에 맞는 오류 원인, txpool 비진입, nonce/잔액 불변을 추가한다.
- 합의 장애: WBFT 3/4 설명을 PoA/etcd에 가져오지 않는다. producer 수·etcd leader·실제 장애 대상 및 etcd/P2P 분리 범위를 정한다. `fault/two-down`의 중단 기대도 etcd quorum 근거로 재검토한다.
- 동기화: `latest`는 계속 변하므로 패치 검증에서는 동일한 확정 높이의 hash/stateRoot/receiptsRoot를 비교한다. `waitBlock(target)`는 단순 도달이지 전체 노드 일치가 아니다.
- genesis/교체: swap 테스트에 이전/패치 go-wemix 바이너리·동일 DB/genesis를 명시한다. 다른 genesis fixture와 시스템 계약 버전 오류는 별개의 문제다.

Type 0/1/2/22가 지원되는 것은 `sources/pr-head/core/types/transaction.go:45`, typed decoder `:189`에서 확인된다. FeeDelegateDynamicFeeTx는 `:198`, 서명 RPC는 `sources/pr-head/internal/ethapi/api.go:2404`이므로 fee-delegation 7건을 지원 불명으로 제외하지 않았다. Type 4/SetCode 성공 기대는 decoder 지원 목록과 맞지 않는다. WBFT seals, Istanbul API, Anzeon 계정/수수료, Stablenet 시스템 계약/Boho, P256 성공 테스트는 이번 실행 목록에서 제외했다. 같은 이름의 개념이 있어도 계약 주소·ABI·정책이 같다는 근거는 없다.

## 제목보다 좁게 구현된 핵심 테스트

| 파일 | 실제 검증 | 빠진 검증 |
|---|---|---|
| basic/01-basic-consensus.json | waitBlock, blockAdvance, sameBlockHash | 모든 producer 참여 |
| basic/06-basic-txpool-propagation.json | receipt 받은 1건 전송과 node2 잔액 | pending 전파, 부하, txpool 배출 |
| fault/01-fault-network-partition.json | peerCount 감소, heal 후 동일 hash | partition 중 blockHalt |
| fault/03-fault-node-recover.json | 높이12 대기, 동일 hash | sync 시간 계측과 목표 시간 |
| fault/06-fault-txpool-leader-change.json | receipt 확인 후 node1 중단, pending=0 | 미포함 pending 보존, 실제 leader 장애 |
| stress/01-stress-block-time.json | 15블록/maxSeconds=60 | 제목의 100블록 통계 |
| stress/02-stress-tx-flood.json | 30% gas 부하를 15블록, 생산 지속 | 동시 대량 tx, TPS, 바이트 크기 경계 |
| go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | receipt.status=0 | 송금 미반영·gas-only 잔액 차감 |
| go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | gasUsed==gasLimit | 잔액/nonce/상태 변경 검증 |

`load`는 18바이트 gas-burn initcode를 사용한다(`sources/chainbench/internal/testhelper/load.go:20`). 블록마다 한 건 보내고 영수증을 기다린다(`:31`, `:69`, `:73`). 따라서 높은 gas 사용량이 높은 RLP 바이트 사용량이라는 결론은 성립하지 않는다. PR 상수는 RLP 블록 8,388,608바이트(`sources/pr-head/params/protocol_params.go:129`), miner의 buffer와 strict `<` 조건(`sources/pr-head/miner/worker.go:143`, `:147`), eth 메시지는 10MiB(`sources/pr-head/eth/protocols/eth/protocol.go:51`)다. 기존 189개에는 이 세 경계의 크기를 만들고 직렬화 크기를 assert하는 케이스가 없다.

## 실행 명령을 준비할 때

현재 CLI는 `chainbench run [spec.json ...]`이며 compose는 `--workspace-dir`, 기존 workspace는 `--workspace-dir --attach`, 순수 RPC 연결은 `--chain wemix --rpc` 방식이다(`sources/chainbench/cmd/chainbench/suitecmd/run.go:55`, `:60`, `:72`, `:98`). compose에서 `--chain`은 spec 선언과 일치해야 한다. 따라서 `--chain wemix` 한 옵션으로 stablenet JSON 전체가 이식되지는 않는다. 프로세스/partition 테스트는 해당 capability가 있는 workspace 환경을 사용해야 한다. 이 검토에서는 명령을 실행하지 않았다.

추가로 읽은 chainbench 구현 4개 파일은 `sources/chainbench-code-evidence-manifest.json`에 관측 시각/SHA256/원본과 복사본 일치 여부를 남겼다. 테스트 원문 스냅샷 이후 읽은 보조 근거이므로 시각을 구분한다.
