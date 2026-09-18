# 문서 테스트 전체 목록

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260909/analyses/existing-tc-specs.md)

common 45행 + dual 29행 = 74행. 원본 중복 ID·스크립트도 합치지 않았다. 코드 근거는 모두 캡처한 PR head 기준이며 로컬 HEAD와 구분한다.

적용 가능은 기능 지원 판정이며 즉시 실행 가능을 뜻하지 않는다. 원문 .sh 존재·정상 실행은 확인하지 않았다. P0→P3 순서이며 제외 행의 P3는 목록 배치용이다.

| 목록 ID | 우선순위 | 적용 판정 | 테스트 | 원문 위치 |
|---|---|---|---|---|
| RT-A-2-04 / TX-010 | P0 | applicable | Nonce 순서 보장 (Nonce Ordering) | sources/common_test_scenarios.md:43 |
| RT-A-1-02 / NODE-003 | P0 | applicable | Full Sync 동기화 | sources/common_test_scenarios.md:96 |
| RT-A-1-06 | P0 | applicable | Block Downloader 경로 동기화 | sources/common_test_scenarios.md:100 |
| RT-A-1-07 | P0 | applicable | Block Fetcher 경로 동기화 | sources/common_test_scenarios.md:101 |
| RT-A-2-01 / TX-006 | P1 | applicable | Legacy Tx (Type 0x0) 전송 및 실행 | sources/common_test_scenarios.md:29 |
| RT-A-2-02 / TX-003 | P1 | applicable | Dynamic Fee Tx (Type 0x2, EIP-1559) | sources/common_test_scenarios.md:30 |
| RT-A-2-03 / TX-007 | P1 | applicable | Access List Tx (Type 0x1, EIP-2930) | sources/common_test_scenarios.md:31 |
| RT-D-01 / TX-004 | P1 | applicable | 수수료 대납 Tx (Type 0x16) 정상 처리 | sources/common_test_scenarios.md:32 |
| RT-D-03 / TX-014 | P1 | applicable | 대납 Tx Sender 서명 변조 거부 | sources/common_test_scenarios.md:33 |
| RT-D-04 / TX-015 | P1 | applicable | 대납 Tx FeePayer 서명 변조 거부 | sources/common_test_scenarios.md:34 |
| RT-D-05 / TX-016 | P1 | applicable | FeePayer 잔액 부족시 실행 실패 | sources/common_test_scenarios.md:35 |
| RT-A-2-05a | P1 | conditional | TipCap 미달 (Underpriced) 거부 | sources/common_test_scenarios.md:44 |
| RT-A-2-06 / TX-011 | P1 | applicable | 잔액 부족 (Insufficient Funds) 거부 | sources/common_test_scenarios.md:45 |
| RT-A-2-07 / TX-012 | P1 | applicable | Gas Limit 초과 거부 | sources/common_test_scenarios.md:46 |
| RT-A-2-08 | P1 | applicable | Effective GasPrice 노드 간 일관성 | sources/common_test_scenarios.md:47 |
| RT-A-2-09 / TX-013 | P1 | applicable | Queued/Pending 트랜잭션 교체 (Replacement Tx) | sources/common_test_scenarios.md:48 |
| RT-A-3-01 / TX-005 | P1 | applicable | 스마트 컨트랙트 배포 | sources/common_test_scenarios.md:56 |
| RT-A-3-02 | P1 | applicable | 컨트랙트 상태 변경 함수 호출 | sources/common_test_scenarios.md:57 |
| RT-A-3-06 / TX-017 | P1 | applicable | Revert 트랜잭션 (상태 롤백 & 가스 환불) | sources/common_test_scenarios.md:61 |
| RT-A-3-07 / TX-018 | P1 | applicable | Out-of-Gas 트랜잭션 (가스 Limit 전액 소모) | sources/common_test_scenarios.md:62 |
| RT-A-4-01 / RPC-001 | P1 | applicable | `eth_blockNumber` | sources/common_test_scenarios.md:70 |
| RT-A-4-03 | P1 | applicable | `eth_sendRawTransaction` | sources/common_test_scenarios.md:72 |
| RT-G-1-01 / RPC-002 | P1 | applicable | `eth_getBlockByNumber` | sources/common_test_scenarios.md:77 |
| RT-G-1-02 / RPC-002 | P1 | applicable | `eth_getBlockByHash` | sources/common_test_scenarios.md:78 |
| RT-G-1-04 / RPC-007 | P1 | applicable | `eth_getTransactionReceipt` | sources/common_test_scenarios.md:80 |
| RT-G-1-05 / RPC-015 | P1 | applicable | `eth_getTransactionCount` | sources/common_test_scenarios.md:81 |
| RT-A-1-03 / NODE-004 | P1 | conditional | Snap Sync 동기화 | sources/common_test_scenarios.md:97 |
| RT-A-1-04 / NODE-005 | P1 | applicable | 노드 재기동 후 동기화 유지 | sources/common_test_scenarios.md:98 |
| RT-A-3-03 | P2 | applicable | `eth_call` View/Pure 함수 조회 | sources/common_test_scenarios.md:58 |
| RT-A-3-04 | P2 | applicable | `eth_estimateGas` 가스 추정 | sources/common_test_scenarios.md:59 |
| RT-A-3-05 | P2 | applicable | `eth_call` 실행 중 Revert 메세지 반환 | sources/common_test_scenarios.md:60 |
| RT-A-4-02 / RPC-012 | P2 | applicable | `eth_getBalance` | sources/common_test_scenarios.md:71 |
| RT-A-4-04 / RPC-014 | P2 | applicable | `eth_getLogs` | sources/common_test_scenarios.md:73 |
| RT-A-4-06 / RPC-020 | P2 | applicable | WebSocket `eth_subscribe("newHeads")` | sources/common_test_scenarios.md:75 |
| RT-A-4-07 / RPC-021 | P2 | applicable | WebSocket `eth_subscribe("logs")` | sources/common_test_scenarios.md:76 |
| RT-G-1-03 | P2 | applicable | `eth_getTransactionByHash` | sources/common_test_scenarios.md:79 |
| RT-G-2-01 / RPC-016 | P2 | applicable | `eth_gasPrice` | sources/common_test_scenarios.md:82 |
| RT-G-2-02 | P2 | applicable | `eth_maxPriorityFeePerGas` | sources/common_test_scenarios.md:83 |
| RT-G-2-03 / RPC-017 | P2 | applicable | `eth_feeHistory` | sources/common_test_scenarios.md:84 |
| RT-G-4-02 / RPC-018 | P2 | applicable | `txpool_status` | sources/common_test_scenarios.md:85 |
| RT-G-4-03 | P2 | applicable | `txpool_content` | sources/common_test_scenarios.md:86 |
| RT-A-1-01 | P2 | applicable | 제네시스 초기화 (Genesis Initialization) | sources/common_test_scenarios.md:95 |
| RT-A-1-05 | P2 | applicable | P2P Bootnode 피어 연결 | sources/common_test_scenarios.md:99 |
| RPC-008 | P2 | conditional | `wemix_getBriocheBlockReward` 하위 호환성 | sources/dual_chain_test_scenarios.md:42 |
| TC-4-2-02 | P2 | conditional | EIP-7702 estimateGas baseline | sources/dual_chain_test_scenarios.md:86 |
| RT-A-4-05 / RPC-013 | P3 | applicable | `eth_chainId` | sources/common_test_scenarios.md:74 |
| RT-G-5-01 | P3 | applicable | FeePayer 서명 RPC 헬스체크 (`eth_signRawFeeDelegateTransaction`) | sources/common_test_scenarios.md:87 |
| RT-B-01 / WBFT-002 | P3 | excluded | 블록 생산 주기 1초 | sources/dual_chain_test_scenarios.md:52 |
| RT-B-02 / WBFT-001 | P3 | excluded | 블록 Finalize 및 CommittedSeal 존재 | sources/dual_chain_test_scenarios.md:53 |
| RT-B-03 / WBFT-005 | P3 | excluded | 에폭 전환 시 검증자 집합 갱신 | sources/dual_chain_test_scenarios.md:54 |
| RT-B-06 | P3 | excluded | 블록 헤더 WBFTExtra 필드 반영 | sources/dual_chain_test_scenarios.md:55 |
| RT-B-09 / WBFT-003 | P3 | excluded | View Change / 라운드 체인지 | sources/dual_chain_test_scenarios.md:56 |
| RT-B-10 / WBFT-003 | P3 | excluded | 라운드 체인지 후 블록 연결 | sources/dual_chain_test_scenarios.md:57 |
| RT-B-06 / WBFT-006 | P3 | excluded | RoundRobin Proposer 정책 | sources/dual_chain_test_scenarios.md:58 |
| RT-B-11 / WBFT-009 | P3 | excluded | PrevCommittedSeal 수집 | sources/dual_chain_test_scenarios.md:59 |
| RT-B-12 / WBFT-009 | P3 | excluded | PrevPreparedSeal 수집 | sources/dual_chain_test_scenarios.md:60 |
| WBFT-010 | P3 | excluded | RandaoReveal / MixDigest | sources/dual_chain_test_scenarios.md:61 |
| RT-B-08 | P3 | excluded | 쿼럼 미달 블록 수락 거부 | sources/dual_chain_test_scenarios.md:62 |
| WBFT-007 | P3 | excluded | 1/3 미만 장애 시 합의 지속 | sources/dual_chain_test_scenarios.md:63 |
| WBFT-008 | P3 | excluded | 1/3 이상 장애 시 합의 중단 | sources/dual_chain_test_scenarios.md:64 |
| WBFT-011 | P3 | excluded | 쿼럼 계산 — validator 3개 (전원 필요) | sources/dual_chain_test_scenarios.md:65 |
| WBFT-012 | P3 | excluded | 쿼럼 계산 — validator 6개, 1개 장애 | sources/dual_chain_test_scenarios.md:66 |
| WBFT-013 | P3 | excluded | 쿼럼 계산 — validator 6개, 2개 장애 | sources/dual_chain_test_scenarios.md:67 |
| RT-G-3-01 / RPC-010 | P3 | excluded | `istanbul_nodeAddress` | sources/dual_chain_test_scenarios.md:73 |
| RT-G-3-02 / RPC-003 | P3 | excluded | `istanbul_getValidators` | sources/dual_chain_test_scenarios.md:74 |
| RT-G-3-03 / RPC-004 | P3 | excluded | `istanbul_getCommitSignersFromBlock` | sources/dual_chain_test_scenarios.md:75 |
| RT-G-3-04 / RPC-005 | P3 | excluded | `istanbul_getWbftExtraInfo` | sources/dual_chain_test_scenarios.md:76 |
| RT-G-3-05 / RPC-006 | P3 | excluded | `istanbul_status` | sources/dual_chain_test_scenarios.md:77 |
| RT-G-3-06 / RPC-011 | P3 | excluded | `istanbul_isValidator` | sources/dual_chain_test_scenarios.md:78 |
| RT-A-2-10 / TX-008 | P3 | excluded | SetCode Tx (Type 0x4, EIP-7702) | sources/dual_chain_test_scenarios.md:84 |
| TC-4-2-01 / TC-4-2-03 | P3 | excluded | EIP-7702 AuthorizationList estimateGas 비용 | sources/dual_chain_test_scenarios.md:85 |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-009 | P3 | excluded | secp256r1 프리컴파일 — 유효 서명 | sources/dual_chain_test_scenarios.md:87 |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-019 | P3 | excluded | secp256r1 프리컴파일 — 무효 서명 | sources/dual_chain_test_scenarios.md:88 |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-020 | P3 | excluded | secp256r1 프리컴파일 — 잘못된 입력 길이 | sources/dual_chain_test_scenarios.md:89 |

## RT-A-2-04 / TX-010 (분석 DOC-C-008) · Nonce 순서 보장 (Nonce Ordering)

- 우선순위: P0 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:43 / 2. 메모리풀 & 트랜잭션 검증 (Mempool & Tx Validation)
- 카테고리: **Mempool**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-04`<br>(`ethereum/11-nonce-ordering.sh`) <br>• **WEMIX4**: `TX-010`<br>(`wemix4/TX/tx-010-nonce-ordering.sh`)
- 목적(원문): 동일 계정에서 제출된 트랜잭션의 오름차순 순서 포함 검증
- 수행흐름(원문): 1. 동일 계정에서 Nonce N+2, N+1, N 역순으로 트랜잭션 전송 (gap 대기).<br>2. Nonce N 주입으로 gap 해소 후 블록 포함 순서 대조.
- 기대결과(원문): - **공통**: Nonce가 낮은 순서(N, N+1, N+2 오름차순)대로 차례대로 블록에 포함
- 판정·우선순위 근거: PR #196의 채굴 거래 선택 또는 블록 수신·검증 경로와 직접 접한다. 작은 블록 통과만으로 대용량 경계 회귀를 검증할 수 없으므로 추가 경계 테스트와 함께 우선 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/core/tx_pool.go:701, sources/pr-head/miner/worker.go:935, sources/pr-head/miner/worker.go:1065

## RT-A-1-02 / NODE-003 (분석 DOC-C-040) · Full Sync 동기화

- 우선순위: P0 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:96 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **Sync**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-02`<br>(`ethereum/02-full-sync.sh`) <br>• **WEMIX4**: `NODE-003`<br>(`wemix4/NODE/node-003-full-sync.sh`)
- 목적(원문): 블록 헤더와 모든 트랜잭션을 실행하며 동기화 검증
- 수행흐름(원문): 1. `--syncmode full`로 동기화 노드 기동.<br>2. 동기화 제공 노드의 높이가 2 이상 높은 상태에서 추격 동기화 관찰.
- 기대결과(원문): - **결과**: 동기화 노드의 헤드 블록 번호 및 `stateRoot`가 제공 노드와 완전히 일치
- 판정·우선순위 근거: PR #196의 채굴 거래 선택 또는 블록 수신·검증 경로와 직접 접한다. 작은 블록 통과만으로 대용량 경계 회귀를 검증할 수 없으므로 추가 경계 테스트와 함께 우선 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다.
- 코드: sources/pr-head/eth/downloader/downloader.go:1475, sources/pr-head/core/block_validator.go:55

## RT-A-1-06 (분석 DOC-C-044) · Block Downloader 경로 동기화

- 우선순위: P0 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:100 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **Sync**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-06`<br>(`ethereum/06-downloader-path.sh`)
- 목적(원문): 큰 블록 Gap($\ge 10$) 발생 시 Downloader 수집 검증
- 수행흐름(원문): 1. 제공 노드와 요청 노드 간 높이 차이를 10 이상 발생시킴.<br>2. Downloader 헤더/바디 큐 수집 추적.
- 기대결과(원문): - **결과**: Downloader가 누락 블록을 순서대로 수집하여 체인에 정상 반영
- 판정·우선순위 근거: PR #196의 채굴 거래 선택 또는 블록 수신·검증 경로와 직접 접한다. 작은 블록 통과만으로 대용량 경계 회귀를 검증할 수 없으므로 추가 경계 테스트와 함께 우선 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다. 높이 gap만으로 경로를 단정하지 않고 downloader/fetcher 진입을 관측한다.
- 코드: sources/pr-head/eth/downloader/downloader.go:1475, sources/pr-head/eth/protocols/eth/handler.go:254

## RT-A-1-07 (분석 DOC-C-045) · Block Fetcher 경로 동기화

- 우선순위: P0 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:101 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **Sync**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-07`<br>(`ethereum/07-block-fetcher-path.sh`)
- 목적(원문): P2P 전파(`NewBlock`/`NewBlockHashes`)를 통한 실시간 수집 검증
- 수행흐름(원문): 1. 실시간 블록 생성 중인 네트워크에 요청 노드 참가.<br>2. P2P 브로드캐스트 수신 관찰.
- 기대결과(원문): - **결과**: Block Fetcher가 브로드캐스트 블록을 수신 즉시 체인 탑재
- 판정·우선순위 근거: PR #196의 채굴 거래 선택 또는 블록 수신·검증 경로와 직접 접한다. 작은 블록 통과만으로 대용량 경계 회귀를 검증할 수 없으므로 추가 경계 테스트와 함께 우선 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다. 높이 gap만으로 경로를 단정하지 않고 downloader/fetcher 진입을 관측한다.
- 코드: sources/pr-head/eth/fetcher/block_fetcher.go:231, sources/pr-head/eth/protocols/eth/handler.go:254

## RT-A-2-01 / TX-006 (분석 DOC-C-001) · Legacy Tx (Type 0x0) 전송 및 실행

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:29 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-01`<br>(`ethereum/08-legacy-tx.sh`) <br>• **WEMIX4**: `TX-006`<br>(`wemix4/TX/tx-006-legacy-tx.sh`)
- 목적(원문): Legacy 타입 트랜잭션의 서명, 전파, 블록 포함 및 Receipt 처리 검증
- 수행흐름(원문): 1. `gasPrice` 필드를 사용하는 Type 0x0 서명 트랜잭션 생성 후 전송.<br>2. 블록 포함 확인 및 Receipt(`eth_getTransactionReceipt`) 조회.
- 기대결과(원문): - **표준 EVM / WEMIX 3.0**: `receipt.status == 0x1`, `effectiveGasPrice == gasPrice`<br>- **StableNet (Anzeon)**: `receipt.status == 0x1`, `effectiveGasPrice >= baseFee` (비인증 계정의 tip이 헤더 GasTip으로 강제 고정)
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. StableNet의 헤더 고정 tip 기대값을 제거하고 go-wemix receipt 계산식을 적용한다.
- 코드: sources/pr-head/core/types/transaction.go:45, sources/pr-head/miner/worker.go:935

## RT-A-2-02 / TX-003 (분석 DOC-C-002) · Dynamic Fee Tx (Type 0x2, EIP-1559)

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:30 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-02`<br>(`ethereum/09-dynamic-fee-tx.sh`) <br>• **WEMIX4**: `TX-003`<br>(`wemix4/TX/tx-003-dynamic-fee-tx.sh`)
- 목적(원문): EIP-1559 동적 수수료 트랜잭션 정상 실행 및 수수료 산출 검증
- 수행흐름(원문): 1. `maxFeePerGas` $\ge$ `baseFee + maxPriorityFeePerGas` 조건의 Type 0x2 트랜잭션 전송.<br>2. Receipt 검증.
- 기대결과(원문): - **공통**: `receipt.status == 0x1`<br>- **산출 수수료**: `effectiveGasPrice == baseFee + min(maxPriorityFeePerGas, maxFeePerGas - baseFee)`
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(sources/pr-head/core/tx_pool.go:651,655,659,1398).
- 코드: sources/pr-head/core/types/transaction.go:47, sources/pr-head/core/tx_pool.go:655

## RT-A-2-03 / TX-007 (분석 DOC-C-003) · Access List Tx (Type 0x1, EIP-2930)

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:31 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-03`<br>(`ethereum/10-access-list-tx.sh`) <br>• **WEMIX4**: `TX-007`<br>(`wemix4/TX/tx-007-accesslist-tx.sh`)
- 목적(원문): Access List가 포함된 트랜잭션의 정상 실행 및 가스 할인 적용 검증
- 수행흐름(원문): 1. `accessList` 필드에 접근 대상 주소 및 스토리지 키를 포함하여 전송.<br>2. 실행 결과 확인.
- 기대결과(원문): - **공통**: Type 0x1 트랜잭션이 성공적으로 블록에 포함 및 실행 (`receipt.status == 0x1`)
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(sources/pr-head/core/tx_pool.go:651,655,659,1398).
- 코드: sources/pr-head/core/types/transaction.go:46, sources/pr-head/core/tx_pool.go:651

## RT-D-01 / TX-004 (분석 DOC-C-004) · 수수료 대납 Tx (Type 0x16) 정상 처리

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:32 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-D-01`<br>(`fee-delegation/01-fee-delegate-normal.sh`) <br>• **WEMIX4**: `TX-004`<br>(`wemix4/TX/tx-004-fee-delegation.sh`)
- 목적(원문): Sender와 FeePayer가 분리된 수수료 대납 트랜잭션 처리 검증
- 수행흐름(원문): 1. Sender 및 FeePayer 서명이 포함된 Type 0x16 RLP 트랜잭션 전송.<br>2. 계정 잔액 변화 확인.
- 기대결과(원문): - **Sender**: 잔액 미차감 (전송 금액만 차감)<br>- **FeePayer**: 가스비 차감 및 트랜잭션 정상 실행 (`status == 0x1`)
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(sources/pr-head/core/tx_pool.go:651,655,659,1398).
- 코드: sources/pr-head/core/types/transaction.go:48, sources/pr-head/core/tx_pool.go:709, sources/pr-head/core/state_transition.go:198

## RT-D-03 / TX-014 (분석 DOC-C-005) · 대납 Tx Sender 서명 변조 거부

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:33 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-D-03`<br>(`fee-delegation/02-sender-sig-invalid.sh`) <br>• **WEMIX4**: `TX-014`<br>(`wemix4/TX/tx-014-fd-sender-sig-invalid.sh`)
- 목적(원문): Sender 서명이 위조/조작된 대납 트랜잭션의 거부 검증
- 수행흐름(원문): 1. Sender 서명 필드(V/R/S)를 임의 조작한 대납 트랜잭션 제출.
- 기대결과(원문): - **결과**: `eth_sendRawTransaction` 전송 시점에 `invalid sender` / `invalid signature` 에러 응답 반환 및 전송 거부
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(sources/pr-head/core/tx_pool.go:651,655,659,1398). 원문의 에러 문자열을 그대로 강제하지 말고 실제 오류 종류·해당 계정 잔액 검사를 대조한다. R=0 등 ValidateSignatureValues가 확실히 거부하는 입력을 고정하고 invalid sender 분기 도달을 확인한다. 임의 비트 변경 후 잔액 부족/feePayer 오류를 서명 오류 검증 성공으로 세지 않는다.
- 코드: sources/pr-head/core/tx_pool.go:688

## RT-D-04 / TX-015 (분석 DOC-C-006) · 대납 Tx FeePayer 서명 변조 거부

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:34 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-D-04`<br>(`fee-delegation/03-feepayer-sig-invalid.sh`) <br>• **WEMIX4**: `TX-015`<br>(`wemix4/TX/tx-015-fd-feepayer-sig-invalid.sh`)
- 목적(원문): FeePayer 서명이 위조/조작된 대납 트랜잭션의 거부 검증
- 수행흐름(원문): 1. FeePayer 서명 필드(FV/FR/FS)를 임의 조작한 대납 트랜잭션 제출.
- 기대결과(원문): - **결과**: `eth_sendRawTransaction` 전송 시점에 `invalid feePayer` / `invalid signature` 에러 응답 반환 및 전송 거부
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(sources/pr-head/core/tx_pool.go:651,655,659,1398). 원문의 에러 문자열을 그대로 강제하지 말고 실제 오류 종류·해당 계정 잔액 검사를 대조한다.
- 코드: sources/pr-head/core/tx_pool.go:709

## RT-D-05 / TX-016 (분석 DOC-C-007) · FeePayer 잔액 부족시 실행 실패

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:35 / 1. 트랜잭션 타입 (Transaction Types)
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-D-05`<br>(`fee-delegation/04-feepayer-insufficient.sh`) <br>• **WEMIX4**: `TX-016`<br>(`wemix4/TX/tx-016-fd-feepayer-insufficient.sh`)
- 목적(원문): FeePayer 계정의 잔액이 가스비보다 부족할 때 거부 처리 검증
- 수행흐름(원문): 1. 잔액 0인 계정을 FeePayer로 지정하여 대납 트랜잭션 전송.<br>2. 전송 응답 확인.
- 기대결과(원문): - **결과**: `eth_sendRawTransaction` 전송 시점에 `insufficient funds` 에러 반환 및 즉시 전송 거부
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(sources/pr-head/core/tx_pool.go:651,655,659,1398). 원문의 에러 문자열을 그대로 강제하지 말고 실제 오류 종류·해당 계정 잔액 검사를 대조한다.
- 코드: sources/pr-head/core/tx_pool.go:713

## RT-A-2-05a (분석 DOC-C-009) · TipCap 미달 (Underpriced) 거부

- 우선순위: P1 / 적용: conditional / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:44 / 2. 메모리풀 & 트랜잭션 검증 (Mempool & Tx Validation)
- 카테고리: **Mempool**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-05a`<br>(`ethereum/12-tipcap-underpriced.sh`)
- 목적(원문): 노드 최소 수용 우선 수수료(Tip) 미달 트랜잭션의 거부 검증
- 수행흐름(원문): 1. 최소 팁 수용 임계값 미만인 `maxPriorityFeePerGas` 지정 트랜잭션 전송.<br>2. 거부 메시지 확인.
- 기대결과(원문): - **표준 EVM / WEMIX 3.0**: 노드 로컬 설정(`pool.gasPrice`) 미만 시 `underpriced` 에러 반환<br>- **StableNet (Anzeon)**: 온체인 헤더 최소 팁(`MinTip`) 미만 시 `underpriced` 에러 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 로컬·원격 제출을 분리한다. 로컬은 DropUnderPriced 및 EffectiveGasTip, 원격은 GasTipCap 비교이므로 문서의 pool.gasPrice 설명만으로 판정하지 않는다.
- 코드: sources/pr-head/core/tx_pool.go:693, sources/pr-head/core/tx_pool.go:697

## RT-A-2-06 / TX-011 (분석 DOC-C-010) · 잔액 부족 (Insufficient Funds) 거부

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:45 / 2. 메모리풀 & 트랜잭션 검증 (Mempool & Tx Validation)
- 카테고리: **Mempool**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-06`<br>(`ethereum/14-insufficient-funds.sh`) <br>• **WEMIX4**: `TX-011`<br>(`wemix4/TX/tx-011-insufficient-funds.sh`)
- 목적(원문): 계정 잔액을 초과하는 전송 요청의 txpool 즉시 거부 검증
- 수행흐름(원문): 1. `value + (gasLimit * gasPrice)`가 계정 잔액을 초과하는 트랜잭션 제출.
- 기대결과(원문): - **공통**: `eth_sendRawTransaction` 전송 시 `insufficient funds for transfer` 에러 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 원문의 에러 문자열을 그대로 강제하지 말고 실제 오류 종류·해당 계정 잔액 검사를 대조한다.
- 코드: sources/pr-head/core/tx_pool.go:720

## RT-A-2-07 / TX-012 (분석 DOC-C-011) · Gas Limit 초과 거부

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:46 / 2. 메모리풀 & 트랜잭션 검증 (Mempool & Tx Validation)
- 카테고리: **Mempool**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-07`<br>(`ethereum/15-gaslimit-exceeded.sh`) <br>• **WEMIX4**: `TX-012`<br>(`wemix4/TX/tx-012-gaslimit-exceeded.sh`)
- 목적(원문): 블록의 가스 한도를 초과하는 트랜잭션의 거부 검증
- 수행흐름(원문): 1. `gasLimit`을 현재 블록의 gas limit보다 크게 설정하여 제출.
- 기대결과(원문): - **공통**: `eth_sendRawTransaction` 전송 시 `exceeds block gas limit` 에러 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/core/tx_pool.go:673

## RT-A-2-08 (분석 DOC-C-012) · Effective GasPrice 노드 간 일관성

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:47 / 2. 메모리풀 & 트랜잭션 검증 (Mempool & Tx Validation)
- 카테고리: **Mempool**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-08`<br>(`ethereum/16-effective-gas-price.sh`)
- 목적(원문): BP/EN 노드 간 Receipt 내 `effectiveGasPrice` 일관성 검증 (PR #70 회귀 방지)
- 수행흐름(원문): 1. BP 노드에 트랜잭션 전송 후 Receipt 조회.<br>2. EN 노드 동기화 대기 후 EN Receipt 조회.<br>3. 두 Receipt의 `effectiveGasPrice` 비교.
- 기대결과(원문): - **공통**: BP 및 EN 노드의 Receipt 내 `effectiveGasPrice`가 null이 아님<br>- **일관성**: `bp_egp == en_egp` (노드 간 값 완전 동일)<br>- **하한선**: `effectiveGasPrice >= baseFee`
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. BP/EN 역할을 생산 노드·동기화 노드로 대응하고 동일 블록 해시에서 receipt를 대조한다. London 전후 기대값을 구분한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1871

## RT-A-2-09 / TX-013 (분석 DOC-C-013) · Queued/Pending 트랜잭션 교체 (Replacement Tx)

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:48 / 2. 메모리풀 & 트랜잭션 검증 (Mempool & Tx Validation)
- 카테고리: **Mempool**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-09`<br>(`ethereum/17-replacement-tx.sh`) <br>• **WEMIX4**: `TX-013`<br>(`wemix4/TX/tx-013-queued-tx-replacement.sh`)
- 목적(원문): 동일 Nonce + 높은 가스비 트랜잭션으로 txpool 교체 검증
- 수행흐름(원문): 1. Nonce Gap(미래 Nonce)으로 tx1 전송 (txpool queued 대기).<br>2. 동일 Nonce, 가스비 10%+ 인상하여 tx2 전송 (tx1 교체).<br>3. Nonce gap 채우기 tx 전송으로 queued 해소.<br>4. tx2 블록 포함 및 tx1 드롭(receipt null) 확인.
- 기대결과(원문): - **공통**: 가스비 10%+ 인상된 tx2만 txpool 교체 후 블록 포함 (`status == 0x1`), 기존 tx1은 receipt null
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. txpool.pricebump 실제 값을 읽고 maxFeePerGas/maxPriorityFeePerGas 양쪽을 인상한다. gap 해소 후 교체 tx만 포함되는지 검증한다.
- 코드: sources/pr-head/core/tx_pool.go:801, sources/pr-head/core/tx_pool.go:851

## RT-A-3-01 / TX-005 (분석 DOC-C-014) · 스마트 컨트랙트 배포

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:56 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-01`<br>(`ethereum/19-contract-deploy.sh`) <br>• **WEMIX4**: `TX-005`<br>(`wemix4/TX/tx-005-contract-deploy.sh`)
- 목적(원문): EVM 바이트코드의 성공적 배포 및 주소 생성 검증
- 수행흐름(원문): 1. Contract Creation 트랜잭션(`to=null`) 전송.<br>2. Receipt `contractAddress` 추출 후 `eth_getCode` 쿼리.
- 기대결과(원문): - **결과**: `contractAddress`에 배포된 EVM 바이트코드가 정상 존재
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다.
- 코드: sources/pr-head/core/vm/evm.go:499

## RT-A-3-02 (분석 DOC-C-015) · 컨트랙트 상태 변경 함수 호출

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:57 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-02`<br>(`ethereum/20-contract-call.sh`)
- 목적(원문): State를 변경하는 함수 호출 트랜잭션 전송 및 갱신 검증
- 수행흐름(원문): 1. 배포된 컨트랙트의 setter 함수 실행 트랜잭션 전송.<br>2. 블록 확정 후 `eth_call`로 getter 함수 쿼리.
- 기대결과(원문): - **결과**: 트랜잭션 성공(`status == 0x1`), 변경된 상태값이 온체인에 정상 반영됨
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다.
- 코드: sources/pr-head/core/vm/evm.go:168

## RT-A-3-06 / TX-017 (분석 DOC-C-019) · Revert 트랜잭션 (상태 롤백 & 가스 환불)

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:61 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-06`<br>(`ethereum/24-revert-tx.sh`) <br>• **WEMIX4**: `TX-017`<br>(`wemix4/TX/tx-017-contract-revert.sh`)
- 목적(원문): 트랜잭션 실행 중 Revert 시 상태 롤백 및 사용 가스만 차감 검증
- 수행흐름(원문): 1. `revert()`를 발생하는 함수 호출 트랜잭션 전송.<br>2. Receipt `status` 및 계정 잔액/상태 확인.
- 기대결과(원문): - **상태**: `receipt.status == 0x0`<br>- **가스**: 사용 가스만 차감되고 미사용 가스는 환불, 상태 롤백
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다.
- 코드: sources/pr-head/core/vm/evm.go:168, sources/pr-head/core/state_transition.go:402

## RT-A-3-07 / TX-018 (분석 DOC-C-020) · Out-of-Gas 트랜잭션 (가스 Limit 전액 소모)

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:62 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-07`<br>(`ethereum/25-out-of-gas.sh`) <br>• **WEMIX4**: `TX-018`<br>(`wemix4/TX/tx-018-contract-out-of-gas.sh`)
- 목적(원문): 실행 가스 부족 시 트랜잭션 실패 및 `gasLimit` 전액 소모 검증
- 수행흐름(원문): 1. `gasLimit`을 실제 필요 가스보다 적게 지정하여 전송.<br>2. Receipt 검증.
- 기대결과(원문): - **상태**: `receipt.status == 0x0`<br>- **가스**: `gasUsed == gasLimit` (가스 환불 없음)
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다.
- 코드: sources/pr-head/core/vm/evm.go:168, sources/pr-head/core/state_transition.go:314

## RT-A-4-01 / RPC-001 (분석 DOC-C-021) · `eth_blockNumber`

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:70 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-01`<br>(`ethereum/26-eth-block-number.sh`) <br>• **WEMIX4**: `RPC-001`<br>(`wemix4/RPC/rpc-001-block-number.sh`)
- 목적(원문): 최신 블록 번호 조회 및 지속적인 증가 검증
- 수행흐름(원문): 1. 일정 시간 간격으로 `eth_blockNumber` 연속 호출.
- 기대결과(원문): - **결과**: 블록 번호가 이전 호출 결과 이상으로 지속 증가(Monotonic Increase)
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. reorg 없는 제어된 체인에서 관찰 구간 내 실제 증가를 확인한다. 단순 비감소만으로 체인 정지를 통과시키면 안 된다.
- 코드: sources/pr-head/internal/ethapi/api.go:738

## RT-A-4-03 (분석 DOC-C-023) · `eth_sendRawTransaction`

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:72 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-03`<br>(`ethereum/28-send-raw-tx.sh`)
- 목적(원문): RLP 직렬화된 서명 트랜잭션 전송 및 전파 검증
- 수행흐름(원문): 1. 로컬에서 서명한 트랜잭션 RLP 헥사 문자열 전송.
- 기대결과(원문): - **결과**: 트랜잭션 해시 반환, 마이닝 전 `txpool`에서 조회 가능
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 즉시 채굴될 수 있으므로 txpool 조회 성공을 필수로 하려면 채굴을 제어한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1992

## RT-G-1-01 / RPC-002 (분석 DOC-C-028) · `eth_getBlockByNumber`

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:77 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-1-01`<br>(`api/01-get-block-by-number.sh`) <br>• **WEMIX4**: `RPC-002`<br>(`wemix4/RPC/rpc-002-get-block.sh`)
- 목적(원문): 블록 번호 기준 블록 객체 상세 조회 검증
- 수행흐름(원문): 1. `eth_getBlockByNumber("latest", true)` 호출.
- 기대결과(원문): - **결과**: 블록 번호, 해시, Parent Hash(`parentHash`) 및 트랜잭션 목록을 포함한 Block 객체 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:950

## RT-G-1-02 / RPC-002 (분석 DOC-C-029) · `eth_getBlockByHash`

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:78 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-1-02`<br>(`api/02-get-block-by-hash.sh`) <br>• **WEMIX4**: `RPC-002`<br>(`wemix4/RPC/rpc-002-get-block.sh`)
- 목적(원문): 블록 해시 기준 블록 객체 상세 조회 검증
- 수행흐름(원문): 1. `eth_getBlockByNumber`로 구한 해시로 `eth_getBlockByHash` 호출.
- 기대결과(원문): - **결과**: `eth_getBlockByNumber`와 완전히 동일한 Block 객체 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:970

## RT-G-1-04 / RPC-007 (분석 DOC-C-031) · `eth_getTransactionReceipt`

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:80 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-1-04`<br>(`api/04-get-tx-receipt.sh`) <br>• **WEMIX4**: `RPC-007`<br>(`wemix4/RPC/rpc-007-receipt.sh`)
- 목적(원문): 트랜잭션 Receipt 조회 검증
- 수행흐름(원문): 1. 마이닝된 트랜잭션 해시로 `eth_getTransactionReceipt` 쿼리.
- 기대결과(원문): - **결과**: `status`, `blockNumber`, `gasUsed`, `effectiveGasPrice`, `logs` 수록 Receipt 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1838

## RT-G-1-05 / RPC-015 (분석 DOC-C-032) · `eth_getTransactionCount`

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:81 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-1-05`<br>(`api/05-get-tx-count.sh`) <br>• **WEMIX4**: `RPC-015`<br>(`wemix4/RPC/rpc-015-tx-count.sh`)
- 목적(원문): 계정의 현재 Nonce (트랜잭션 수) 조회 검증
- 수행흐름(원문): 1. 트랜잭션 전송 계정에 대해 `eth_getTransactionCount` 쿼리.
- 기대결과(원문): - **결과**: 해당 계정이 전송하여 온체인에 포함된 총 트랜잭션 수와 일치하는 Nonce 반환
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 동일 확정 블록 기준 초기 nonce N 대비 포함된 sender 거래 수 k만큼 N+k를 기대한다. pending 조회는 별도 pool nonce 검증으로 분리한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1779

## RT-A-1-03 / NODE-004 (분석 DOC-C-041) · Snap Sync 동기화

- 우선순위: P1 / 적용: conditional / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:97 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **Sync**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-03`<br>(`ethereum/03-snap-sync.sh`) <br>• **WEMIX4**: `NODE-004`<br>(`wemix4/NODE/node-004-snap-sync.sh`)
- 목적(원문): State 피벗 기반의 고속 스냅샷 동기화 검증
- 수행흐름(원문): 1. `--syncmode snap` 설정 후 대량 블록이 생성된 체인에서 동기화 진행.<br>2. 동기화 완료 후 잔액 조회.
- 기대결과(원문): - **결과**: state 동기화 완료 후 `eth_getBalance` 정상 조회 및 `stateRoot` 일치
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다. snap 지원 피어·충분한 체인 길이·snapshot 상태가 필요하다. full sync로 fallback한 성공은 snap 경로 검증으로 세지 않는다.
- 코드: sources/pr-head/eth/downloader/downloader.go:1565, sources/pr-head/eth/downloader/downloader.go:1710

## RT-A-1-04 / NODE-005 (분석 DOC-C-042) · 노드 재기동 후 동기화 유지

- 우선순위: P1 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:98 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **Sync**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-04`<br>(`ethereum/04-node-restart.sh`) <br>• **WEMIX4**: `NODE-005`<br>(`wemix4/NODE/node-005-node-restart.sh`)
- 목적(원문): 노드 프로세스 재시작 시 체인 DB 보존 및 생성 재개 검증
- 수행흐름(원문): 1. 정상 운영 중인 노드 프로세스를 종료(`kill`).<br>2. 재기동 후 블록 생성을 모니터링.
- 기대결과(원문): - **결과**: DB 손상 없이 노드가 정상 재기동되고 블록 동기화/생성 재개
- 판정·우선순위 근거: 지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다.
- 코드: sources/pr-head/core/genesis.go:239, sources/pr-head/miner/worker.go:1627

## RT-A-3-03 (분석 DOC-C-016) · `eth_call` View/Pure 함수 조회

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:58 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-03`<br>(`ethereum/21-eth-call-view.sh`)
- 목적(원문): 트랜잭션 전송 없이 `eth_call`을 이용한 View 함수 조회 및 반환값 검증
- 수행흐름(원문): 1. `eth_call` JSON-RPC를 이용해 `x()` view 함수 호출 (트랜잭션 미전송).
- 기대결과(원문): - **결과**: 트랜잭션 생성 및 상태 변경 없이 정확한 32-byte Hex 반환값 수신
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1206

## RT-A-3-04 (분석 DOC-C-017) · `eth_estimateGas` 가스 추정

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:59 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-04`<br>(`ethereum/22-estimate-gas.sh`)
- 목적(원문): 트랜잭션 실행 전 필요한 가스 소비량 측정 정확성 검증
- 수행흐름(원문): 1. 컨트랙트 호출 데이터로 `eth_estimateGas` RPC 실행.<br>2. 실제 실행 시 사용된 `gasUsed`와 비교.
- 기대결과(원문): - **결과**: 실제 실행 시 소비되는 가스량 이상의 적정 추정치 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1340

## RT-A-3-05 (분석 DOC-C-018) · `eth_call` 실행 중 Revert 메세지 반환

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:60 / 3. 스마트 컨트랙트 (Smart Contracts / EVM Execution)
- 카테고리: **Contract**
- 원래 ID·스크립트: • **StableNet**: `RT-A-3-05`<br>(`ethereum/23-eth-call-revert.sh`)
- 목적(원문): `eth_call` 수행 중 Revert 발생 시 에러 사유 전달 검증
- 수행흐름(원문): 1. 조건 미충족 시 `revert("Reason")`을 발생시키는 함수를 `eth_call`로 호출.
- 기대결과(원문): - **결과**: `execution reverted` 에러 및 디코딩 가능한 Revert Reason 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1206

## RT-A-4-02 / RPC-012 (분석 DOC-C-022) · `eth_getBalance`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:71 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-02`<br>(`ethereum/27-eth-get-balance.sh`) <br>• **WEMIX4**: `RPC-012`<br>(`wemix4/RPC/rpc-012-get-balance.sh`)
- 목적(원문): 특정 계정의 Native 코인 잔액 조회 검증
- 수행흐름(원문): 1. 알려진 계정에 대해 `eth_getBalance("latest")` 쿼리.
- 기대결과(원문): - **결과**: Hex Wei 단위의 정확한 수량 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:843

## RT-A-4-04 / RPC-014 (분석 DOC-C-024) · `eth_getLogs`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:73 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-04`<br>(`ethereum/29-eth-get-logs.sh`) <br>• **WEMIX4**: `RPC-014`<br>(`wemix4/RPC/rpc-014-get-logs.sh`)
- 목적(원문): 조건별 이벤트 로그(Event Log) 조회 검증
- 수행흐름(원문): 1. 이벤트를 뿜는 컨트랙트 실행 후 `eth_getLogs` 쿼리 (`fromBlock`, `toBlock`, `address`, `topics`).
- 기대결과(원문): - **결과**: 조건에 부합하는 Log 객체 배열 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다.
- 코드: sources/pr-head/eth/filters/api.go:327

## RT-A-4-06 / RPC-020 (분석 DOC-C-026) · WebSocket `eth_subscribe("newHeads")`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:75 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-06`<br>(`ethereum/31-ws-subscribe-heads.sh`) <br>• **WEMIX4**: `RPC-020`<br>(`wemix4/RPC/rpc-020-subscribe-heads.sh`)
- 목적(원문): WebSocket 기반 블록 생성 실시간 구독 검증
- 수행흐름(원문): 1. WS 연결 후 `eth_subscribe("newHeads")` 요청.<br>2. 블록 생성 시 수신 프레임 모니터링.
- 기대결과(원문): - **결과**: 새 블록 마이닝 시마다 real-time 헤더 이벤트 데이터 수신
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. WebSocket 및 eth 구독 API가 활성화된 노드가 필요하다.
- 코드: sources/pr-head/eth/filters/api.go:208

## RT-A-4-07 / RPC-021 (분석 DOC-C-027) · WebSocket `eth_subscribe("logs")`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:76 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-07`<br>(`ethereum/32-ws-subscribe-logs.sh`) <br>• **WEMIX4**: `RPC-021`<br>(`wemix4/RPC/rpc-021-subscribe-logs.sh`)
- 목적(원문): WebSocket 기반 이벤트 로그 실시간 구독 검증
- 수행흐름(원문): 1. WS 연결 후 특정 주소/토픽 로그 구독.<br>2. 해당 이벤트 발생 트랜잭션 전송.
- 기대결과(원문): - **결과**: 조건 부합 이벤트 발생 즉시 real-time 로그 프레임 수신
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다. WebSocket 및 eth 구독 API가 활성화된 노드가 필요하다.
- 코드: sources/pr-head/eth/filters/api.go:238

## RT-G-1-03 (분석 DOC-C-030) · `eth_getTransactionByHash`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:79 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-1-03`<br>(`api/03-get-tx-by-hash.sh`)
- 목적(원문): 트랜잭션 해시 기준 트랜잭션 상세 조회 검증
- 수행흐름(원문): 1. 마이닝된 트랜잭션 해시로 `eth_getTransactionByHash` 쿼리.
- 기대결과(원문): - **결과**: Transaction 객체(`from`, `to`, `value`, `nonce`, `input` 등) 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1798

## RT-G-2-01 / RPC-016 (분석 DOC-C-033) · `eth_gasPrice`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:82 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-2-01`<br>(`api/07-gas-price.sh`) <br>• **WEMIX4**: `RPC-016`<br>(`wemix4/RPC/rpc-016-gas-price.sh`)
- 목적(원문): 기준 가스 가격 조회 검증
- 수행흐름(원문): 1. `eth_gasPrice` RPC 호출.
- 기대결과(원문): - **표준 EVM**: Oracle 알고리즘 기반 권장 가스 가격 반환<br>- **StableNet (WBFT/Anzeon)**: `gasPrice == baseFee + WBFTExtra.GasTip` (블록 헤더 GasTip 고정 공식 일치)
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. WBFTExtra.GasTip 기대값은 제외하고 go-wemix API·oracle 결과를 사용한다.
- 코드: sources/pr-head/internal/ethapi/api.go:92

## RT-G-2-02 (분석 DOC-C-034) · `eth_maxPriorityFeePerGas`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:83 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-2-02`<br>(`api/08-max-priority-fee.sh`)
- 목적(원문): 우선 수수료(Tip) 조회 검증
- 수행흐름(원문): 1. `eth_maxPriorityFeePerGas` RPC 호출.
- 기대결과(원문): - **표준 EVM**: 권장 우선 수수료(Tip) 반환<br>- **StableNet (Anzeon)**: `maxPriorityFeePerGas == WBFTExtra.GasTip` (추정값이 아닌 헤더 GasTip 값과 정확히 일치)
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. WBFTExtra.GasTip 기대값은 제외하고 go-wemix API·oracle 결과를 사용한다.
- 코드: sources/pr-head/internal/ethapi/api.go:104

## RT-G-2-03 / RPC-017 (분석 DOC-C-035) · `eth_feeHistory`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:84 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-2-03`<br>(`api/09-fee-history.sh`) <br>• **WEMIX4**: `RPC-017`<br>(`wemix4/RPC/rpc-017-fee-history.sh`)
- 목적(원문): 최근 N개 블록 수수료 이력 조회 검증
- 수행흐름(원문): 1. `eth_feeHistory(N, "latest", [10, 50])` 호출.
- 기대결과(원문): - **결과**: `baseFeePerGas` 배열, 가스 사용 비율 및 백분위 Tip 배열 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:119

## RT-G-4-02 / RPC-018 (분석 DOC-C-036) · `txpool_status`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:85 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-4-02`<br>(`api/18-txpool-status.sh`) <br>• **WEMIX4**: `RPC-018`<br>(`wemix4/RPC/rpc-018-txpool-status.sh`)
- 목적(원문): 메모리풀 트랜잭션 수량 조회 검증
- 수행흐름(원문): 1. pending 및 queued 트랜잭션을 txpool에 주입 후 쿼리.
- 기대결과(원문): - **결과**: `pending` 및 `queued` 정수 건수 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. txpool API 노출 및 채굴 제어가 필요하다. JSON 수량은 hex quantity로 디코딩한다.
- 코드: sources/pr-head/internal/ethapi/api.go:241

## RT-G-4-03 (분석 DOC-C-037) · `txpool_content`

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:86 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-4-03`<br>(`api/19-txpool-content.sh`)
- 목적(원문): 메모리풀 상세 트랜잭션 목록 조회 검증
- 수행흐름(원문): 1. txpool 트랜잭션 주입 후 `txpool_content` 쿼리.
- 기대결과(원문): - **결과**: 계정별 pending/queued 트랜잭션 상세 객체 트리 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. txpool API 노출 및 채굴 제어가 필요하다. JSON 수량은 hex quantity로 디코딩한다.
- 코드: sources/pr-head/internal/ethapi/api.go:191

## RT-A-1-01 (분석 DOC-C-039) · 제네시스 초기화 (Genesis Initialization)

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:95 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **Sync**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-01`<br>(`ethereum/01-genesis-init.sh`)
- 목적(원문): 제네시스 파일로 노드 DB 초기화 및 0번 블록 해시 검증
- 수행흐름(원문): 1. `genesis.json` 설정 기반 노드 데이터 디렉터리 초기화.<br>2. 노드 기동 후 블록 0 해시 및 `eth_chainId` 조회.
- 기대결과(원문): - **결과**: 블록 0 해시가 설정과 일치하며 `chainId` 정상 인식
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/core/genesis.go:239

## RT-A-1-05 (분석 DOC-C-043) · P2P Bootnode 피어 연결

- 우선순위: P2 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:99 / 5. 노드 동기화 & P2P 네트워크 (Node Sync & P2P)
- 카테고리: **P2P**
- 원래 ID·스크립트: • **StableNet**: `RT-A-1-05`<br>(`ethereum/05-p2p-peers.sh`)
- 목적(원문): `--bootnodes` 설정을 통한 P2P 메시 네트워크 구성 검증
- 수행흐름(원문): 1. Bootnode ENode 주소를 지정하여 노드 기동.<br>2. `net_peerCount` 및 `admin_peers` 쿼리.
- 기대결과(원문): - **결과**: `net_peerCount` $\ge 1$, `admin_peers`에 피어 연결 정보 반환
- 판정·우선순위 근거: 지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다.
- 코드: sources/pr-head/internal/ethapi/api.go:2304, sources/pr-head/node/api.go:308

## RPC-008 (분석 DOC-D-001) · `wemix_getBriocheBlockReward` 하위 호환성

- 우선순위: P2 / 적용: conditional / 실행 준비: spec_only_or_port_needed
- 원본: sources/dual_chain_test_scenarios.md:42 / 그룹 A — WEMIX3.0 + WEMIX4.0 공통
- 카테고리: **RPC**
- 원래 ID·스크립트: • **WEMIX3.0**: (전용 스크립트 없음 / `go-wemix` 코드 지원 확인)<br>• **WEMIX4**: `RPC-008`<br>(`wemix4/RPC/rpc-008-brioche-reward.sh`)
- 목적(원문): Brioche 하드포크 블록 보상 산출 및 halving 스케줄 조회 검증
- 수행흐름(원문): 1. `wemix_getBriocheBlockReward` 호출.<br>2. 블록 높이 기준 보상·halving 결과 확인.
- 기대결과(원문): - **공통(WEMIX3.0 / WEMIX4)**: Brioche 규칙에 따른 블록 보상 값 반환<br>- **StableNet**: 미지원 (RPC 부재)
- 판정·우선순위 근거: Brioche 보상 조회는 구현되어 있으나 패치 변경부의 간접 회귀이다.
- go-wemix 적용 조건: wemixapi.Info와 Brioche 설정이 유효한 체인에서 실제 fork·halving 경계를 지정한다. WEMIX4 스크립트 이식 필요.
- 코드: sources/pr-head/eth/api.go:748, sources/pr-head/eth/api.go:764, sources/pr-head/params/config.go:443

## TC-4-2-02 (분석 DOC-D-026) · EIP-7702 estimateGas baseline

- 우선순위: P2 / 적용: conditional / 실행 준비: spec_only_or_port_needed
- 원본: sources/dual_chain_test_scenarios.md:86 / B-3. 특수 트랜잭션 및 프리컴파일
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `TC-4-2-02`<br>(`common-all/19-estimategas-no-authorizationlist.sh`)<br>• **WEMIX4**: (전용 스크립트 없음 / `go-wbft` EIP-7702 지원 확인)
- 목적(원문): AuthorizationList가 없는 일반 전송의 estimateGas baseline 검증
- 수행흐름(원문): 1. AuthorizationList 없는 일반 전송의 estimateGas 조회.<br>2. baseline 가스 비용 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: AuthorizationList 미포함 시 baseline 가스만 산출
- 판정·우선순위 근거: AuthorizationList가 없는 일반 estimateGas baseline은 지원된다. EIP-7702 지원을 검증한 것으로 기록하면 안 된다.
- go-wemix 적용 조건: 7702 필드·fixture를 제거하고 일반 전송 baseline으로 이식한다. RT-A-3-04 (분석 DOC-C-017)과 관련 있지만 원문 행은 유지한다.
- 코드: sources/pr-head/internal/ethapi/api.go:1340, sources/pr-head/internal/ethapi/transaction_args.go:52, sources/pr-head/core/types/transaction.go:189

## RT-A-4-05 / RPC-013 (분석 DOC-C-025) · `eth_chainId`

- 우선순위: P3 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:74 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (Ethereum)**
- 원래 ID·스크립트: • **StableNet**: `RT-A-4-05`<br>(`ethereum/30-eth-chain-id.sh`) <br>• **WEMIX4**: `RPC-013`<br>(`wemix4/RPC/rpc-013-chain-id.sh`)
- 목적(원문): EIP-155 체인 ID 조회 검증
- 수행흐름(원문): 1. `eth_chainId` RPC 호출.
- 기대결과(원문): - **결과**: Genesis에 설정된 Hex Chain ID 반환
- 판정·우선순위 근거: RPC 존재·고정 설정을 확인하는 smoke 성격으로 패치의 블록 크기·채굴 회귀 탐지력은 낮다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다.
- 코드: sources/pr-head/internal/ethapi/api.go:729

## RT-G-5-01 (분석 DOC-C-038) · FeePayer 서명 RPC 헬스체크 (`eth_signRawFeeDelegateTransaction`)

- 우선순위: P3 / 적용: applicable / 실행 준비: spec_only_or_port_needed
- 원본: sources/common_test_scenarios.md:87 / 4. 표준 JSON-RPC API (Standard JSON-RPC APIs)
- 카테고리: **RPC (API)**
- 원래 ID·스크립트: • **StableNet**: `RT-G-5-01`<br>(`api/21-sign-raw-fee-delegate.sh`)
- 목적(원문): `eth_signRawFeeDelegateTransaction` RPC 엔드포인트 존재 여부 검증 (Smoke Test)
- 수행흐름(원문): 1. 의도적으로 잘못된 파라미터로 `eth_signRawFeeDelegateTransaction` 호출.
- 기대결과(원문): - **결과**: 응답 에러 메시지가 `method not found` 계열이 아님을 확인 (= RPC 엔드포인트 존재 확인)
- 판정·우선순위 근거: RPC 존재·고정 설정을 확인하는 smoke 성격으로 패치의 블록 크기·채굴 회귀 탐지력은 낮다.
- go-wemix 적용 조건: go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다. eth API 노출을 확인한다. method-not-found가 아님은 기능적 서명 성공의 증거가 아니다.
- 코드: sources/pr-head/internal/ethapi/api.go:2404

## RT-B-01 / WBFT-002 (분석 DOC-D-002) · 블록 생산 주기 1초

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:52 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-01`<br>(`wbft/01-block-period.sh`)<br>• **WEMIX4**: `WBFT-002`<br>(`wemix4/WBFT/wbft-002-block-period.sh`)
- 목적(원문): WBFT 합의의 고정 블록 주기(1초) 준수 검증
- 수행흐름(원문): 1. 연속 블록의 timestamp 차이 측정.<br>2. 블록 주기가 1초로 유지되는지 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 연속 블록 간 timestamp 차 == 1초
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-02 / WBFT-001 (분석 DOC-D-003) · 블록 Finalize 및 CommittedSeal 존재

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:53 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-02`<br>(`wbft/02-wbft-extra-seal.sh`)<br>• **WEMIX4**: `WBFT-001`<br>(`wemix4/WBFT/wbft-001-finalize.sh`)
- 목적(원문): 정상 블록 생성·확정 시 WBFTExtra에 CommittedSeal(BLS 서명)이 quorum 이상 존재하는지 검증
- 수행흐름(원문): 1. 연속 블록 생성 대기.<br>2. 블록별 WBFTExtra의 CommittedSeal(및 PreparedSeal) 존재·수량 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 각 블록의 CommittedSeal이 quorum 이상 존재, Finalize 정상 동작
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-03 / WBFT-005 (분석 DOC-D-004) · 에폭 전환 시 검증자 집합 갱신

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:54 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-03`<br>(`wbft/03-epoch-transition.sh`)<br>• **WEMIX4**: `WBFT-005`<br>(`wemix4/WBFT/wbft-005-epoch-transition.sh`)
- 목적(원문): 에폭 경계에서 밸리데이터 세트가 재구성되는지 검증
- 수행흐름(원문): 1. 에폭 마지막 블록 도달.<br>2. 다음 에폭의 밸리데이터 집합 갱신 확인.
- 기대결과(원문): - **공통**: 에폭 전환 블록에서 밸리데이터 세트 갱신, WBFTExtra에 EpochInfo 포함
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-06 (분석 DOC-D-005) · 블록 헤더 WBFTExtra 필드 반영

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:55 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-06`<br>(`wbft/06-gastip-header-sync.sh`)<br>• **WEMIX4**: (전용 스크립트 없음 / `go-wbft` 코드 지원 확인)
- 목적(원문): 거버넌스 변경값이 블록 헤더 WBFTExtra에 반영되는지 검증 (StableNet은 GasTip 필드)
- 수행흐름(원문): 1. 헤더 WBFTExtra 필드 조회.<br>2. 블록 간 값 반영·동기화 확인.
- 기대결과(원문): - **StableNet**: GasTip 거버넌스 변경 → WBFTExtra.GasTip 반영<br>- **WEMIX4**: WBFTExtra 필드 반영 (GasTip 필드는 부재)
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-09 / WBFT-003 (분석 DOC-D-006) · View Change / 라운드 체인지

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:56 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-09`<br>(`wbft/09-round-change.sh`)<br>• **WEMIX4**: `WBFT-003`<br>(`wemix4/WBFT/wbft-003-view-change.sh`)
- 목적(원문): 제안자 노드 중단 시 라운드 체인지 발생 검증
- 수행흐름(원문): 1. 현재 라운드 제안자 노드 중단.<br>2. 라운드 체인지 후 새 제안자가 블록 생성 확인.
- 기대결과(원문): - **공통**: 라운드 체인지 후 새 제안자가 블록 정상 생성
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-10 / WBFT-003 (분석 DOC-D-007) · 라운드 체인지 후 블록 연결

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:57 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-10`<br>(`wbft/10-post-round-change.sh`)<br>• **WEMIX4**: (전용 스크립트 없음 / `WBFT-003` 후속 검증에 포함)
- 목적(원문): 라운드 체인지 이후 생성된 블록의 체인 연속성 검증
- 수행흐름(원문): 1. 라운드 체인지 발생 후 신규 블록 생성.<br>2. parentHash 체인 정상 연결 확인.
- 기대결과(원문): - **공통**: 라운드 체인지 후 블록의 parentHash가 직전 블록과 정상 연결
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-06 / WBFT-006 (분석 DOC-D-008) · RoundRobin Proposer 정책

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:58 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-06` (RoundRobin 검증 미포함)<br>• **WEMIX4**: `WBFT-006`<br>(`wemix4/WBFT/wbft-006-roundrobin-proposer.sh`)
- 목적(원문): 제안자가 RoundRobin 정책에 따라 균등 순환하는지 검증
- 수행흐름(원문): 1. 연속 블록의 miner(Proposer) 주소 수집.<br>2. Proposer 다양성·균등 분배 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 제안자가 밸리데이터 집합을 RoundRobin으로 균등 순환 (StableNet도 `go-stablenet` RoundRobin 정책 지원)
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-11 / WBFT-009 (분석 DOC-D-009) · PrevCommittedSeal 수집

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:59 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-11`<br>(`wbft/11-prev-committed-seal.sh`)<br>• **WEMIX4**: `WBFT-009`<br>(`wemix4/WBFT/wbft-009-prev-seal.sh`)
- 목적(원문): 블록 N+1의 PrevCommittedSeal이 블록 N의 committer를 포함하는지 검증
- 수행흐름(원문): 1. 블록 N의 committer 집합 수집.<br>2. 블록 N+1의 WBFTExtra.PrevCommittedSeal 확인.
- 기대결과(원문): - **공통**: 블록 N+1의 PrevCommittedSeal이 블록 N committer를 quorum 이상 포함
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-12 / WBFT-009 (분석 DOC-D-010) · PrevPreparedSeal 수집

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:60 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-12`<br>(`wbft/12-prev-prepared-seal.sh`)<br>• **WEMIX4**: `WBFT-009`<br>(`wemix4/WBFT/wbft-009-prev-seal.sh`)
- 목적(원문): 블록 N+1의 PrevPreparedSeal이 블록 N의 prepare 서명자를 포함하는지 검증
- 수행흐름(원문): 1. 블록 N의 prepare 서명자 집합 수집.<br>2. 블록 N+1의 WBFTExtra.PrevPreparedSeal 확인.
- 기대결과(원문): - **공통**: 블록 N+1의 PrevPreparedSeal이 블록 N prepare 서명자를 포함
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## WBFT-010 (분석 DOC-D-011) · RandaoReveal / MixDigest

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:61 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: (전용 스크립트 없음 / `go-stablenet` RandaoMix 지원 확인)<br>• **WEMIX4**: `WBFT-010`<br>(`wemix4/WBFT/wbft-010-randao-mixdigest.sh`)
- 목적(원문): 블록 헤더의 RandaoReveal 및 MixDigest 생성·검증
- 수행흐름(원문): 1. 연속 블록의 RandaoReveal·MixDigest(mixHash) 조회.<br>2. 필드 존재·검증.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 각 블록에 RandaoReveal·MixDigest 존재 (StableNet도 `go-stablenet` CalculateRandaoMix 지원)
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-B-08 (분석 DOC-D-012) · 쿼럼 미달 블록 수락 거부

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:62 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **StableNet**: `RT-B-08`<br>(`wbft/08-quorum-deficient.sh`)<br>• **WEMIX4**: (해당 없음 — StableNet 전용 관점)
- 목적(원문): 서명이 쿼럼에 미달하는 블록을 노드가 거부하는지 검증
- 수행흐름(원문): 1. 쿼럼 미달 서명 블록 유도.<br>2. 블록 수락 여부 확인.
- 기대결과(원문): - **StableNet**: 쿼럼 미달 블록 수락 거부
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## WBFT-007 (분석 DOC-D-013) · 1/3 미만 장애 시 합의 지속

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:63 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **WEMIX4**: `WBFT-007`<br>(`wemix4/WBFT/wbft-007-fault-under-third.sh`)<br>• **StableNet**: (전용 스크립트 없음 / WBFT 쿼럼 로직 동일)
- 목적(원문): 밸리데이터 1/3 미만 장애 시에도 쿼럼 충족으로 합의가 지속되는지 검증
- 수행흐름(원문): 1. 밸리데이터 1개 종료(6/7 운영, 쿼럼 5 충족).<br>2. 합의 지속 및 재기동 후 동기화 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 쿼럼 충족 시 합의 지속, 재기동 노드 동기화
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## WBFT-008 (분석 DOC-D-014) · 1/3 이상 장애 시 합의 중단

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:64 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **WEMIX4**: `WBFT-008`<br>(`wemix4/WBFT/wbft-008-fault-over-third.sh`)<br>• **StableNet**: (전용 스크립트 없음 / WBFT 쿼럼 로직 동일)
- 목적(원문): 밸리데이터 1/3 이상 장애 시 쿼럼 미달로 합의가 중단되는지 검증
- 수행흐름(원문): 1. 밸리데이터 3개 종료(4/7 운영, 쿼럼 5 미달).<br>2. 합의 중단 확인 후 재기동 시 합의 재개.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 쿼럼 미달 시 합의 중단, 재기동 후 재개
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## WBFT-011 (분석 DOC-D-015) · 쿼럼 계산 — validator 3개 (전원 필요)

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:65 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **WEMIX4**: `WBFT-011`<br>(`wemix4/WBFT/wbft-011-quorum-3-validators.sh`)<br>• **StableNet**: (전용 스크립트 없음 / WBFT 쿼럼 로직 동일)
- 목적(원문): 밸리데이터 3개 구성에서 쿼럼(3, 전원)이 요구되는지 검증
- 수행흐름(원문): 1. validator 3개 구성.<br>2. 1개 종료 시 쿼럼 미달 → 합의 중단, 재기동 후 재개 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: validator 3개는 전원 필요, 1개 장애 시 합의 중단
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## WBFT-012 (분석 DOC-D-016) · 쿼럼 계산 — validator 6개, 1개 장애

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:66 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **WEMIX4**: `WBFT-012`<br>(`wemix4/WBFT/wbft-012-quorum-6-val-1-fault.sh`)<br>• **StableNet**: (전용 스크립트 없음 / WBFT 쿼럼 로직 동일)
- 목적(원문): 밸리데이터 6개 구성에서 1개 장애 시 쿼럼(5) 충족으로 합의 지속 검증
- 수행흐름(원문): 1. validator 6개 구성.<br>2. 1개 종료(5/6 운영) 시 CommitSigners == 5 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 6개 중 1개 장애 시 쿼럼 5 충족, 합의 지속
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## WBFT-013 (분석 DOC-D-017) · 쿼럼 계산 — validator 6개, 2개 장애

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:67 / B-1. WBFT 합의 (Consensus)
- 카테고리: **WBFT**
- 원래 ID·스크립트: • **WEMIX4**: `WBFT-013`<br>(`wemix4/WBFT/wbft-013-quorum-6-val-2-fault.sh`)<br>• **StableNet**: (전용 스크립트 없음 / WBFT 쿼럼 로직 동일)
- 목적(원문): 밸리데이터 6개 구성에서 2개 장애 시 쿼럼(5) 미달로 합의 중단 검증
- 수행흐름(원문): 1. validator 6개 구성.<br>2. 2개 종료(4/6 운영, 쿼럼 5 미달) 시 합의 중단 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 6개 중 2개 장애 시 쿼럼 미달, 합의 중단
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-G-3-01 / RPC-010 (분석 DOC-D-018) · `istanbul_nodeAddress`

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:73 / B-2. Istanbul/WBFT 전용 RPC (`istanbul_*`)
- 카테고리: **RPC**
- 원래 ID·스크립트: • **StableNet**: `RT-G-3-01`<br>(`api/11-node-address.sh`)<br>• **WEMIX4**: `RPC-010`<br>(`wemix4/RPC/rpc-010-node-address.sh`)
- 목적(원문): 노드 자신의 서명 주소 조회 검증
- 수행흐름(원문): 1. `istanbul_nodeAddress` 호출.<br>2. 반환 주소가 노드 서명 주소와 일치하는지 확인.
- 기대결과(원문): - **공통**: 노드의 밸리데이터 서명 주소 반환
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-G-3-02 / RPC-003 (분석 DOC-D-019) · `istanbul_getValidators`

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:74 / B-2. Istanbul/WBFT 전용 RPC (`istanbul_*`)
- 카테고리: **RPC**
- 원래 ID·스크립트: • **StableNet**: `RT-G-3-02`<br>(`api/12-get-validators.sh`)<br>• **WEMIX4**: `RPC-003`<br>(`wemix4/RPC/rpc-003-get-validators.sh`)
- 목적(원문): 특정 블록 기준 밸리데이터 집합 조회 검증
- 수행흐름(원문): 1. `istanbul_getValidators` 호출.<br>2. 반환된 밸리데이터 목록 확인.
- 기대결과(원문): - **공통**: 해당 블록 시점의 밸리데이터 주소 배열 반환
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-G-3-03 / RPC-004 (분석 DOC-D-020) · `istanbul_getCommitSignersFromBlock`

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:75 / B-2. Istanbul/WBFT 전용 RPC (`istanbul_*`)
- 카테고리: **RPC**
- 원래 ID·스크립트: • **StableNet**: `RT-G-3-03`<br>(`api/13-get-commit-signers.sh`)<br>• **WEMIX4**: `RPC-004`<br>(`wemix4/RPC/rpc-004-commit-signers.sh`)
- 목적(원문): 블록의 커밋 서명자 목록 조회 검증
- 수행흐름(원문): 1. `istanbul_getCommitSignersFromBlock` 호출.<br>2. 서명자 목록이 밸리데이터 부분집합인지 확인.
- 기대결과(원문): - **공통**: 블록의 CommittedSeal 서명자 주소 목록 반환 (quorum 이상)
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-G-3-04 / RPC-005 (분석 DOC-D-021) · `istanbul_getWbftExtraInfo`

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:76 / B-2. Istanbul/WBFT 전용 RPC (`istanbul_*`)
- 카테고리: **RPC**
- 원래 ID·스크립트: • **StableNet**: `RT-G-3-04`<br>(`api/14-get-wbft-extra.sh`)<br>• **WEMIX4**: `RPC-005`<br>(`wemix4/RPC/rpc-005-wbft-extra-normal.sh`)
- 목적(원문): 블록 헤더 WBFTExtra 구조 조회 검증
- 수행흐름(원문): 1. `istanbul_getWbftExtraInfo` 호출.<br>2. RandaoReveal·PrevSeal·CommittedSeal 필드 확인.
- 기대결과(원문): - **공통**: 일반 블록은 EpochInfo null, WBFTExtra 필드 정상 반환
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-G-3-05 / RPC-006 (분석 DOC-D-022) · `istanbul_status`

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:77 / B-2. Istanbul/WBFT 전용 RPC (`istanbul_*`)
- 카테고리: **RPC**
- 원래 ID·스크립트: • **StableNet**: `RT-G-3-05`<br>(`api/15-istanbul-status.sh`)<br>• **WEMIX4**: `RPC-006`<br>(`wemix4/RPC/rpc-006-istanbul-status.sh`)
- 목적(원문): 합의 상태(서명 카운트) 조회 검증
- 수행흐름(원문): 1. `istanbul_status` 호출.<br>2. 밸리데이터별 서명 통계 확인.
- 기대결과(원문): - **공통**: 밸리데이터별 sealer 활동 통계 반환
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-G-3-06 / RPC-011 (분석 DOC-D-023) · `istanbul_isValidator`

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:78 / B-2. Istanbul/WBFT 전용 RPC (`istanbul_*`)
- 카테고리: **RPC**
- 원래 ID·스크립트: • **StableNet**: `RT-G-3-06`<br>(`api/16-is-validator.sh`)<br>• **WEMIX4**: `RPC-011`<br>(`wemix4/RPC/rpc-011-is-validator.sh`)
- 목적(원문): 지정 주소의 밸리데이터 여부 판정 검증
- 수행흐름(원문): 1. 밸리데이터/비밸리데이터 주소로 `istanbul_isValidator` 호출.<br>2. 판정 결과 확인.
- 기대결과(원문): - **공통**: 밸리데이터 → `true`, 비밸리데이터 → `false`
- 판정·우선순위 근거: 이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.
- go-wemix 적용 조건: WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.
- 코드: sources/pr-head/eth/ethconfig/config.go:217, sources/pr-head/miner/worker.go:1627, sources/pr-head/miner/worker.go:1812

## RT-A-2-10 / TX-008 (분석 DOC-D-024) · SetCode Tx (Type 0x4, EIP-7702)

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:84 / B-3. 특수 트랜잭션 및 프리컴파일
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `RT-A-2-10`<br>(`ethereum/18-setcode-tx.sh`)<br>• **WEMIX4**: `TX-008`<br>(`wemix4/TX/tx-008-setcode-tx.sh`)
- 목적(원문): EOA에 계정 코드를 위임하는 EIP-7702 트랜잭션 정상 동작 검증
- 수행흐름(원문): 1. AuthorizationList가 포함된 Type 0x4 트랜잭션 전송.<br>2. 대상 EOA의 코드 위임 결과 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: `receipt.status == 0x1`, 대상 EOA에 위임 코드(delegation designator) 설정
- 판정·우선순위 근거: typed transaction 디코더는 0x1·0x2·0x16만 수용하며 EIP-7702 0x4/AuthorizationList 실행은 지원하지 않는다.
- go-wemix 적용 조건: go-wemix #196 회귀 실행 대상에서 제외. 미지원 타입 거부 검증은 별도의 다른 테스트이다.
- 코드: sources/pr-head/core/types/transaction.go:189, sources/pr-head/internal/ethapi/transaction_args.go:52

## TC-4-2-01 / TC-4-2-03 (분석 DOC-D-025) · EIP-7702 AuthorizationList estimateGas 비용

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:85 / B-3. 특수 트랜잭션 및 프리컴파일
- 카테고리: **Tx Type**
- 원래 ID·스크립트: • **StableNet**: `TC-4-2-01`, `TC-4-2-03`<br>(`common-all/18-estimategas-authorizationlist-cost.sh`)<br>• **WEMIX4**: (전용 스크립트 없음 / `go-wbft` EIP-7702 지원 확인)
- 목적(원문): AuthorizationList 항목 수에 따른 `eth_estimateGas` 비용 반영 검증
- 수행흐름(원문): 1. AuthorizationList 1건·2건 포함 트랜잭션의 estimateGas 조회.<br>2. 항목 수에 비례한 가스 비용 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: AuthorizationList 항목당 per-auth 가스 비용이 estimateGas에 반영
- 판정·우선순위 근거: typed transaction 디코더는 0x1·0x2·0x16만 수용하며 EIP-7702 0x4/AuthorizationList 실행은 지원하지 않는다.
- go-wemix 적용 조건: go-wemix #196 회귀 실행 대상에서 제외. 미지원 타입 거부 검증은 별도의 다른 테스트이다.
- 코드: sources/pr-head/core/types/transaction.go:189, sources/pr-head/internal/ethapi/transaction_args.go:52

## TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-009 (분석 DOC-D-027) · secp256r1 프리컴파일 — 유효 서명

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:87 / B-3. 특수 트랜잭션 및 프리컴파일
- 카테고리: **Precompile**
- 원래 ID·스크립트: • **StableNet**: `TC-1-2-01~06`<br>(`common-all/12-all-secp256r1-precompile.sh`)<br>• **WEMIX4**: `TX-009`<br>(`wemix4/TX/tx-009-secp256r1-valid.sh`)
- 목적(원문): 주소 `0x100` P256VERIFY 프리컴파일의 유효 서명 검증
- 수행흐름(원문): 1. 유효한 P-256 서명·공개키·메시지 해시 입력.<br>2. 프리컴파일 호출 결과 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 유효 서명 → 성공 반환(`0x...01`)
- 판정·우선순위 근거: 등록된 EVM 프리컴파일 집합에 RIP-7212의 0x100 P256VERIFY가 없다. 빈 반환만으로 무효 서명 검증이 성공했다고 판단할 수 없다.
- go-wemix 적용 조건: go-wemix #196 회귀 실행 대상에서 제외한다.
- 코드: sources/pr-head/core/vm/contracts.go:48, sources/pr-head/core/vm/contracts.go:84

## TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-019 (분석 DOC-D-028) · secp256r1 프리컴파일 — 무효 서명

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:88 / B-3. 특수 트랜잭션 및 프리컴파일
- 카테고리: **Precompile**
- 원래 ID·스크립트: • **StableNet**: `TC-1-2-01~06`<br>(`common-all/12-all-secp256r1-precompile.sh`)<br>• **WEMIX4**: `TX-019`<br>(`wemix4/TX/tx-019-secp256r1-invalid.sh`)
- 목적(원문): 무효 서명 입력 시 프리컴파일이 실패(빈 반환)하는지 검증
- 수행흐름(원문): 1. 무효한 P-256 서명 입력.<br>2. 프리컴파일 호출 결과 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 무효 서명 → 빈 반환/검증 실패
- 판정·우선순위 근거: 등록된 EVM 프리컴파일 집합에 RIP-7212의 0x100 P256VERIFY가 없다. 빈 반환만으로 무효 서명 검증이 성공했다고 판단할 수 없다.
- go-wemix 적용 조건: go-wemix #196 회귀 실행 대상에서 제외한다.
- 코드: sources/pr-head/core/vm/contracts.go:48, sources/pr-head/core/vm/contracts.go:84

## TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-020 (분석 DOC-D-029) · secp256r1 프리컴파일 — 잘못된 입력 길이

- 우선순위: P3 / 적용: excluded / 실행 준비: not_target
- 원본: sources/dual_chain_test_scenarios.md:89 / B-3. 특수 트랜잭션 및 프리컴파일
- 카테고리: **Precompile**
- 원래 ID·스크립트: • **StableNet**: `TC-1-2-01~06`<br>(`common-all/12-all-secp256r1-precompile.sh`)<br>• **WEMIX4**: `TX-020`<br>(`wemix4/TX/tx-020-secp256r1-short-input.sh`)
- 목적(원문): 규격보다 짧은 입력 길이일 때 프리컴파일 처리 검증
- 수행흐름(원문): 1. 규격 미달(짧은) 입력 전달.<br>2. 프리컴파일 호출 결과 확인.
- 기대결과(원문): - **공통(WEMIX4 / StableNet)**: 잘못된 입력 길이 → 빈 반환/검증 실패
- 판정·우선순위 근거: 등록된 EVM 프리컴파일 집합에 RIP-7212의 0x100 P256VERIFY가 없다. 빈 반환만으로 무효 서명 검증이 성공했다고 판단할 수 없다.
- go-wemix 적용 조건: go-wemix #196 회귀 실행 대상에서 제외한다.
- 코드: sources/pr-head/core/vm/contracts.go:48, sources/pr-head/core/vm/contracts.go:84
