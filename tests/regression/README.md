# go-stablenet v2 회귀 테스트 (Regression Tests)

`REGRESSION_TEST_CASES_v2.md` 문서의 103개 테스트 케이스를 chainbench 프레임워크를 통해 실제 로컬 체인에서 자동 실행할 수 있는 bash 테스트 suite입니다.

## 📁 디렉터리 구조

```
tests/regression/
├── README.md                     (이 파일)
├── lib/
│   └── common.sh                 공통 헬퍼 (raw tx, receipt, log, eth_call, governance)
├── a-ethereum/                   A. 이더리움 기본 기능 (30 파일)
│   ├── a1-01..05-*.sh            노드/동기화
│   ├── a2-01..10-*.sh            트랜잭션 (05a/05b 분할 포함)
│   ├── a3-01..07-*.sh            스마트 컨트랙트
│   └── a4-01..07-*.sh            RPC API
├── b-wbft/                       B. WBFT 합의 엔진 (12 TCs)
├── c-anzeon/                     C. Anzeon 가스 정책 (7 TCs)
├── d-fee-delegation/             D. Fee Delegation (4 TCs)
├── e-blacklist-authorized/       E. 블랙리스트/권한 (9 TCs)
├── f-system-contracts/           F. 시스템 컨트랙트 & 거버넌스 (27 TCs)
├── g-api/                        G. API 호출 (21 TCs)
└── run-all.sh                    전체 카테고리 일괄 실행
```

## 🚀 빠른 시작

### 1. 환경 준비

```bash
# Python 의존성 설치 (raw tx 서명, ABI 인코딩, FeeDelegate RLP 등)
pip3 install eth-account requests eth-utils eth-abi websockets rlp eth-keys

# gstable 바이너리 빌드 (go-stablenet 저장소에서)
cd /path/to/go-stablenet && make gstable
```

### 2. 회귀 테스트 프로필로 체인 초기화 & 기동

```bash
cd /path/to/chainbench
./chainbench.sh init --profile regression
./chainbench.sh start
./chainbench.sh status
```

프로필 구성 (`profiles/regression.yaml`):
- 4 BP (validator) + 1 EN, epoch=30 (짧게)
- 테스트 계정 5개 alloc (TEST_ACC_A~E, Hardhat 기본 keys)
- AnzeonBlock=0, BohoBlock=0, ApplepieBlock=0 (all active from genesis)

### 3. 개별 테스트 실행

```bash
# 단일 테스트
./chainbench.sh test run regression/a-ethereum/a2-01-legacy-tx

# 카테고리 전체
./chainbench.sh test run regression/a-ethereum

# 전체 회귀 suite
./chainbench.sh test run regression
# 또는
./tests/regression/run-all.sh
```

### 4. 결과 확인

```bash
./chainbench.sh report --format json > regression-results.json
ls state/results/                    # 각 테스트 별 JSON 결과
```

## 🔑 테스트 계정 (Hardhat 기본 keys)

`tests/regression/lib/common.sh`의 상수로 제공:

| 변수 | 주소 | 용도 |
|------|------|------|
| `TEST_ACC_A` | `0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266` | Sender (일반 tx 발행) |
| `TEST_ACC_B` | `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` | Recipient |
| `TEST_ACC_C` | `0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC` | FeePayer (D-fee-delegation) |
| `TEST_ACC_D` | `0x90F79bf6EB2c4f870365E785982E1f101E93b906` | Authorize 후보 |
| `TEST_ACC_E` | `0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65` | Blacklist 후보 |

Private key는 `common.sh` 내 `TEST_ACC_*_PK` 상수로 접근 (**⚠️ test only, never mainnet**).

## 🧩 공통 헬퍼 (`lib/common.sh`)

### Raw Tx
```bash
send_raw_tx <target> <private_key> <to> <value_wei> [data] [gas_limit] [tx_type]
# tx_type: legacy | dynamic | accesslist | setcode
```

### Receipt / Log
```bash
wait_tx_receipt_full <target> <tx_hash> [timeout]     # full JSON
get_receipt_field <target> <tx_hash> <field>           # 특정 필드
find_log_by_topic <receipt_json> <address> <topic0>    # 이벤트 검색
count_logs_by_address <receipt_json> <address>         # 로그 카운트
```

### eth_call / ABI
```bash
eth_call_raw <target> <to> <data_hex>
selector <function_signature>                           # "transfer(address,uint256)" → "0xa9059cbb"
pad_address <addr>                                      # 20-byte → 32-byte
pad_uint256 <decimal_or_hex>                            # uint → 32-byte
```

### Governance
```bash
gov_call <target> <contract> <data> <from_addr> <gas>  # propose/approve/execute
```

### 상수
- 시스템 컨트랙트: `NATIVE_COIN_ADAPTER`, `GOV_VALIDATOR`, `GOV_MINTER`, `GOV_COUNCIL`, `ACCOUNT_MANAGER`
- 이벤트 시그: `TRANSFER_EVENT_SIG`, `AUTHORIZED_TX_EXECUTED_SIG`
- 기타: `MIN_BASE_FEE_WEI`, `ZERO_ADDRESS`

## 📋 TC ↔ 파일 매핑 (전체)

아래는 Category A만 상세하며, B~G는 파일명에서 TC ID를 바로 알 수 있음 (예: `b-09-round-change.sh` → RT-B-09).

### Category A (Ethereum 기본 기능)

| REVIEW TC ID | 파일 | 주요 검증 |
|---|---|---|
| RT-A-1-01 | `a1-01-genesis-init.sh` | 모든 노드 block 0 hash 동일 |
| RT-A-1-02 | `a1-02-full-sync.sh` | Full Sync: stop → gap ≥ 2 → restart → 동기화 (`--syncmode full`) |
| RT-A-1-03 | `a1-03-snap-sync.sh` | Snap Sync: 128+ 블록 조건 + state 접근 (`--syncmode snap`) |
| RT-A-1-04 | `a1-04-node-restart.sh` | 재시작 후 1초 주기 유지 |
| RT-A-1-05 | `a1-05-p2p-peers.sh` | admin_peers 응답 |
| **RT-A-1-06** | `a1-06-downloader-path.sh` | **[신규] Downloader 경로**: gap ≥ 2 → 헤더·바디 순차 다운로드 |
| **RT-A-1-07** | `a1-07-block-fetcher-path.sh` | **[신규] Block Fetcher 경로**: NewBlock 전파 → 1~2초 내 head 갱신 |
| RT-A-2-01 | `a2-01-legacy-tx.sh` | Legacy Tx + effectiveGasPrice + gasUsed==21000 |
| RT-A-2-02 | `a2-02-dynamic-fee-tx.sh` | DynamicFeeTx + gasLimit valid |
| RT-A-2-03 | `a2-03-access-list-tx.sh` | AccessListTx type=0x1 |
| RT-A-2-04 | `a2-04-nonce-ordering.sh` | 연속 nonce 순서 |
| RT-A-2-05a | `a2-05a-tipcap-underpriced.sh` | GasTipCap < MinTip, prefix "gas tip cap" |
| RT-A-2-05b | `a2-05b-feecap-underpriced.sh` | GasFeeCap < MinBaseFee, prefix "gas fee cap" |
| RT-A-2-06 | `a2-06-insufficient-funds.sh` | 잔액 부족 거부 |
| RT-A-2-07 | `a2-07-gaslimit-exceeded.sh` | 블록 gas limit 초과 거부 |
| RT-A-2-08 | `a2-08-effective-gas-price.sh` | BP/EN 양쪽 동일 (PR #70 회귀) |
| RT-A-2-09 | `a2-09-replacement-tx.sh` | 동일 nonce 더 높은 fee로 교체 |
| RT-A-2-10 | `a2-10-setcode-tx.sh` | EIP-7702 SetCodeTx + delegation prefix |
| RT-A-3-01 | `a3-01-contract-deploy.sh` | 컨트랙트 배포 + eth_getCode |
| RT-A-3-02 | `a3-02-contract-call.sh` | set(42) 후 eth_call 확인 |
| RT-A-3-03 | `a3-03-eth-call-view.sh` | view 함수 tx 없이 조회 |
| RT-A-3-04 | `a3-04-estimate-gas.sh` | eth_estimateGas |
| RT-A-3-05 | `a3-05-eth-call-revert.sh` | revert reason "BAD_INPUT" |
| RT-A-3-06 | `a3-06-revert-tx.sh` | revert tx status==0, gasUsed<gasLimit |
| RT-A-3-07 | `a3-07-out-of-gas.sh` | OOG gasUsed==gasLimit |
| RT-A-4-01 | `a4-01-eth-block-number.sh` | 단조 증가 |
| RT-A-4-02 | `a4-02-eth-get-balance.sh` | Wei 단위 |
| RT-A-4-03 | `a4-03-send-raw-tx.sh` | txpool pending 확인 |
| RT-A-4-04 | `a4-04-eth-get-logs.sh` | Transfer 이벤트 필터 |
| RT-A-4-05 | `a4-05-eth-chain-id.sh` | chainId == 8283 |
| RT-A-4-06 | `a4-06-ws-subscribe-heads.sh` | WebSocket newHeads |
| RT-A-4-07 | `a4-07-ws-subscribe-logs.sh` | WebSocket logs + Transfer 수신 |

### Categories B ~ G (요약)

- **B (WBFT)**: `b-01-block-period` ~ `b-12-prev-prepared-seal` — B-04/05는 GovBase proposeAddMember/proposeRemoveMember lifecycle 완전 구현
- **C (Anzeon)**: `c-01-regular-account-gastip-forced` ~ `c-07-max-basefee` — baseFee 증가/유지/감소 + Min/Max 경계
- **D (Fee Delegation)**: `d-01-fee-delegate-normal`, `d-03-sender-sig-invalid`, `d-04-feepayer-sig-invalid`, `d-05-feepayer-insufficient` — `lib/fee_delegate.py` helper로 type 0x16 RLP 직접 인코딩
- **E (Blacklist/Authorized)**: `e-01` ~ `e-09` — GovCouncil blacklist/authorize + AuthorizedTxExecuted 이벤트 (PR #70 dependency)
- **F (Governance)**: 27 files — F-1 NativeCoinAdapter(5), F-2 GovMinter(3), F-3 GovValidator(6), F-4 GovMasterMinter(4), F-5 GovCouncil(9, 4가지 이벤트 포함)
- **G (API)**: G-1 block/tx 조회(6), G-2 gas/fee(4), G-3 istanbul_*(6), G-4 관리/진단(4, pending/queued 분리), G-5 StableNet 고유(3)

## ⚠️ 주의사항

1. **테스트 순서 의존성**: A-3-01이 배포한 컨트랙트 주소를 A-3-02/03/04/07이 참조 (파일 `/tmp/chainbench-regression/simple_storage.addr`)
2. **테스트 간 상태 오염**: 블록체인 상태는 누적되므로, 특정 테스트가 실패하면 다음 테스트에 영향 가능 → `chainbench clean && init --profile regression` 후 재시도 권장
3. **WebSocket 테스트**: `pip3 install websockets` 필요. 미설치 시 자동 skip
4. **EIP-7702 SetCodeTx**: `eth-account` 라이브러리가 0.13+ 버전에서 `sign_authorization` 지원. 구버전이면 자동 skip
5. **Governance 테스트 (B/E/F)**: 여러 validator가 quorum 승인해야 하므로 **node 1~4 keystore unlock** 필요. 각 테스트 파일의 setup 단계에서 처리

## 🏗️ 진행 상태

- [x] 인프라 (profile + common.sh + fee_delegate.py)
- [x] **Category A — 이더리움 기본 기능 (30 files)**
- [x] **Category B — WBFT 합의 엔진 (12 files, validator add/remove 완전 구현)**
- [x] **Category C — Anzeon 가스 정책 (7 files)**
- [x] **Category D — Fee Delegation (4 files, tx_fee_delegation.go 기반 완전 구현)**
- [x] **Category E — 블랙리스트/권한 (9 files)**
- [x] **Category F — 시스템 컨트랙트 & 거버넌스 (27 files)**
- [x] **Category G — API 호출 (23 files)**
- [x] Test runner + README + USAGE_EXAMPLES

**완성 현황**: 인프라 + **113개 테스트 파일** / REGRESSION_TEST_CASES_v2.md 전체 103 TC 커버 (일부 TC는 subcase로 분할되어 파일 수 증가)
