# go-stablenet v2 회귀 테스트 — 사용 예시

`REGRESSION_TEST_CASES_v2.md`의 103개 테스트 케이스를 실제로 실행하는 단계별 가이드입니다.

---

## 📋 목차

1. [환경 준비](#1-환경-준비)
2. [첫 실행 시나리오](#2-첫-실행-시나리오)
3. [테스트 실행 방법](#3-테스트-실행-방법)
4. [결과 확인 및 보고서](#4-결과-확인-및-보고서)
5. [특정 시나리오별 실행 예시](#5-특정-시나리오별-실행-예시)
6. [디버깅 & 트러블슈팅](#6-디버깅--트러블슈팅)
7. [CI/CD 통합](#7-cicd-통합)
8. [Claude Code / MCP 활용](#8-claude-code--mcp-활용)

---

## 1. 환경 준비

### 1.1 사전 조건

| 항목 | 버전 | 설치 방법 |
|---|---|---|
| Go | 1.21+ | https://go.dev/dl/ |
| Python | 3.9+ | `brew install python3` |
| jq | any | `brew install jq` |
| curl | any | 기본 설치됨 |

### 1.2 Python 패키지 설치

```bash
pip3 install eth-account requests eth-utils eth-abi websockets rlp eth-keys
```

확인:
```bash
python3 -c "import eth_account, requests, eth_utils, eth_abi, websockets, rlp, eth_keys; print('OK')"
# 출력: OK
```

| 패키지 | 용도 |
|---|---|
| `eth-account` | 표준 EIP-1559 tx 서명 |
| `requests` | JSON-RPC HTTP 호출 |
| `eth-utils` | keccak256, ABI selector 계산 |
| `eth-abi` | GovMinter MintProof/BurnProof bytes 인코딩 (F-2) |
| `websockets` | eth_subscribe 테스트 (A-4-06/07) |
| `rlp` | FeeDelegateDynamicFeeTx (type 0x16) 수동 인코딩 (D 카테고리) |
| `eth-keys` | FeePayer 서명 (lib/fee_delegate.py) |

### 1.3 go-stablenet 빌드

```bash
cd /Users/wm-it-22-00661/Work/github/stable-net/regression-test-case/go-stablenet
make gstable
ls -la build/bin/gstable  # 바이너리 확인
```

### 1.4 chainbench 설정

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench

# gstable 바이너리 경로 설정
vim profiles/regression.yaml
# chain.binary_path: "../../../stable-net/regression-test-case/go-stablenet/build/bin/gstable"
# (또는 절대경로)
```

---

## 2. 첫 실행 시나리오

### 2.1 체인 초기화

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench

# regression 프로필로 초기화
./chainbench.sh init --profile regression
```

성공 출력 예시:
```
[INFO]  Loading profile: regression
[INFO]  Generating genesis.json...
[INFO]  Generated /tmp/node-data/genesis.json
[INFO]  Generating node TOML files...
[INFO]  Copying preset keys...
[INFO]  Initializing node1 with gstable init
[INFO]  Initializing node2 with gstable init
[INFO]  Initializing node3 with gstable init
[INFO]  Initializing node4 with gstable init
[INFO]  Initializing node5 with gstable init
[OK]    Initialization complete
```

### 2.2 체인 기동

```bash
./chainbench.sh start
```

5개 노드 (4 BP + 1 EN)가 백그라운드로 기동됩니다.

```bash
./chainbench.sh status
```

출력 예시:
```
Profile: regression
Nodes:
  node1  [BP]  PID=12345  http=8501  block=15  peers=4  RUNNING
  node2  [BP]  PID=12346  http=8502  block=15  peers=4  RUNNING
  node3  [BP]  PID=12347  http=8503  block=15  peers=4  RUNNING
  node4  [BP]  PID=12348  http=8504  block=15  peers=4  RUNNING
  node5  [EN]  PID=12349  http=8505  block=15  peers=4  RUNNING
```

### 2.3 기본 연결 확인

```bash
# chain ID 확인 (8283이어야 함)
curl -s -X POST http://127.0.0.1:8501 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' | jq .

# 출력:
# {"jsonrpc":"2.0","id":1,"result":"0x205b"}

# 현재 블록 번호
curl -s -X POST http://127.0.0.1:8501 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq .
```

### 2.4 TEST_ACC_A 잔액 확인 (genesis alloc 확인)

```bash
curl -s -X POST http://127.0.0.1:8501 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266","latest"],"id":1}' | jq .
# 출력: 1000000000000000000000000000 wei = 1B ETH
```

---

## 3. 테스트 실행 방법

### 3.1 전체 회귀 suite 실행

```bash
./tests/regression/run-all.sh
```

출력 예시:
```
========================================
go-stablenet v2 Regression Test Suite
========================================

Categories: a-ethereum b-wbft c-anzeon d-fee-delegation e-blacklist-authorized f-system-contracts g-api

--- Category: a-ethereum ---
  Running a1-01-genesis-init ... PASS
  Running a1-02-full-sync ... PASS
  Running a1-03-snap-sync ... PASS
  Running a1-04-node-restart ... PASS
  ...
  Running a4-07-ws-subscribe-logs ... PASS

========================================
Regression Test Summary
========================================
Total:    30
Passed:   30
Failed:   0
Skipped:  0
Duration: 145s
========================================
```

### 3.2 특정 카테고리만 실행

```bash
# Category A만
./tests/regression/run-all.sh a-ethereum

# Category A + B
./tests/regression/run-all.sh a-ethereum b-wbft

# Category F (거버넌스)만
./tests/regression/run-all.sh f-system-contracts
```

### 3.3 개별 테스트 실행

**방법 1 — 직접 실행**:
```bash
bash tests/regression/a-ethereum/a2-01-legacy-tx.sh
```

**방법 2 — chainbench 명령**:
```bash
./chainbench.sh test run regression/a-ethereum/a2-01-legacy-tx
```

**방법 3 — 테스트 목록에서 선택**:
```bash
./chainbench.sh test list
# regression/a-ethereum/a1-01-genesis-init
# regression/a-ethereum/a2-01-legacy-tx
# ...
```

---

## 4. 결과 확인 및 보고서

### 4.1 개별 테스트 결과 JSON

각 테스트는 실행 후 `state/results/<test_name>_<timestamp>.json` 파일을 생성합니다.

```bash
ls state/results/ | head -10
```

```json
// state/results/regression_a_ethereum_a2_01_legacy_tx_20260408_170530.json
{
  "test": "regression/a-ethereum/a2-01-legacy-tx",
  "status": "passed",
  "pass": 7,
  "fail": 0,
  "total": 7,
  "duration": 3,
  "timestamp": "2026-04-08T17:05:30Z",
  "failures": []
}
```

### 4.2 종합 보고서 생성

```bash
./chainbench.sh report --format json > regression-results.json
./chainbench.sh report --format markdown > regression-results.md
./chainbench.sh report --format text
```

### 4.3 실시간 로그 모니터링

```bash
# 테스트 실행 중 노드 로그 모니터링
./chainbench.sh log tail 1       # node1 로그 실시간
./chainbench.sh log search "ERROR" --tail 100
```

---

## 5. 특정 시나리오별 실행 예시

### 5.1 "PR #70 회귀 검증"만 실행

```bash
# RT-A-2-08: BP/EN 양쪽 effectiveGasPrice 비교
bash tests/regression/a-ethereum/a2-08-effective-gas-price.sh
```

### 5.2 "새 기능: EIP-7702 SetCodeTx" 검증

```bash
# RT-A-2-10: SetCodeTx + delegation prefix 확인
bash tests/regression/a-ethereum/a2-10-setcode-tx.sh
```

### 5.3 "새 기능: AuthorizedTxExecuted 이벤트" 검증 (E 카테고리 완성 후)

```bash
# RT-E-09: Authorized 계정 tx 실행 후 이벤트 확인
bash tests/regression/e-blacklist-authorized/e-09-authorized-tx-executed.sh
```

### 5.4 "GasTip 거버넌스 → 헤더 반영" 전체 흐름 (B+F 완성 후)

```bash
# RT-F-3-05: proposeGasTip → 승인 → execute → storage 변경
bash tests/regression/f-system-contracts/f3-05-proposegastip-lifecycle.sh

# RT-B-06: 위의 후속으로 다음 블록 헤더에 반영 확인
bash tests/regression/b-wbft/b-06-gastip-header-sync.sh
```

### 5.5 "transaction underpriced 두 경로" 검증

```bash
# tipCap < MinTip
bash tests/regression/a-ethereum/a2-05a-tipcap-underpriced.sh

# gasFeeCap < MinBaseFee + MinTip
bash tests/regression/a-ethereum/a2-05b-feecap-underpriced.sh
```

### 5.6 "WBFT 합의 엔진" 시나리오 (Category B)

**필수 선행**: 모든 validator keystore unlock, node1~4 정상 기동

```bash
# 블록 주기 & WBFTExtra 검증
bash tests/regression/b-wbft/b-01-block-period.sh       # 10 블록 연속 1초 주기
bash tests/regression/b-wbft/b-02-wbft-extra-seal.sh    # Committed + Prepared Seal 모두 quorum
bash tests/regression/b-wbft/b-03-epoch-transition.sh   # epoch 경계 블록 EpochInfo

# Validator member 추가/제거 (GovBase proposal + voting)
bash tests/regression/b-wbft/b-04-add-validator.sh      # proposeAddMember → approve → execute
bash tests/regression/b-wbft/b-05-remove-validator.sh   # proposeRemoveMember → approve → execute

# 라운드 체인지 (node1 중단 시)
bash tests/regression/b-wbft/b-09-round-change.sh
bash tests/regression/b-wbft/b-10-post-round-change.sh  # parentHash 체인 + round=0 복귀

# 쿼럼 미달 시 체인 halt 검증 (복구 포함)
bash tests/regression/b-wbft/b-08-quorum-deficient.sh
```

**주의**: `b-08`은 validator 2개를 잠시 중단하므로 테스트 중 chain이 멈춥니다. 자동 복구됩니다.
**주의**: `b-04`/`b-05`는 `unlock_all_validators`를 호출하며, keystore 비밀번호 `"1"` (preset) 사용.

### 5.7 "Anzeon 가스 정책" 시나리오 (Category C)

```bash
# 일반 계정 vs Authorized 계정 tip 정책
bash tests/regression/c-anzeon/c-01-regular-account-gastip-forced.sh    # 비인증: header.GasTip 강제
bash tests/regression/c-anzeon/c-02-authorized-account-gastip-free.sh   # 인증: 자유 설정 (F-5-03 선행)

# baseFee 변동 규칙 (블록 사용률 기반)
bash tests/regression/c-anzeon/c-03-basefee-increase.sh   # 25% 부하 → +2%
bash tests/regression/c-anzeon/c-04-basefee-stable.sh     # 10% 부하 → 변동 없음
bash tests/regression/c-anzeon/c-05-basefee-decrease.sh   # idle → -2%

# baseFee 경계값
bash tests/regression/c-anzeon/c-06-min-basefee.sh        # MinBaseFee(20 Gwei) 하한
bash tests/regression/c-anzeon/c-07-max-basefee.sh        # MaxBaseFee 상한
```

**주의**: `c-03` ~ `c-05`는 블록 사용률이 확률적이므로 간혹 window 내에서 목표 범위를 맞추지 못할 수 있습니다. 여러 번 재실행 권장.

### 5.8 "Fee Delegation (type 0x16)" 시나리오 (Category D)

`tests/regression/lib/fee_delegate.py`는 go-stablenet `core/types/tx_fee_delegation.go`의 RLP 구조 기반 직접 인코더/서명 도구입니다.

```bash
# Python 의존성 확인
pip3 install rlp eth-keys eth-account requests eth-utils

# Helper 단독 사용 예시 (tx 서명 + 전송)
python3 tests/regression/lib/fee_delegate.py send \
  --rpc http://127.0.0.1:8501 \
  --sender-pk 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  --fee-payer-pk 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a \
  --to 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 \
  --value 1000000000000000 \
  --gas 21000

# 출력: JSON (rawTx, txHash, senderAddr, feePayerAddr, sigHashes 등)
```

```bash
# D 카테고리 전체 테스트
bash tests/regression/d-fee-delegation/d-01-fee-delegate-normal.sh      # 정상 처리: Sender value / FeePayer gas
bash tests/regression/d-fee-delegation/d-03-sender-sig-invalid.sh       # --tamper sender → 거부
bash tests/regression/d-fee-delegation/d-04-feepayer-sig-invalid.sh     # --tamper feepayer → 거부
bash tests/regression/d-fee-delegation/d-05-feepayer-insufficient.sh    # 빈 계정 feePayer → insufficient funds
```

**`fee_delegate.py` 주요 옵션**:
- `sign` — 서명만 수행, rawTx를 JSON으로 반환
- `send` — 서명 + `eth_sendRawTransaction` 호출
- `--tamper sender` — Sender 서명 1바이트 변조 (negative test)
- `--tamper feepayer` — FeePayer 서명 1바이트 변조

**RLP 구조 검증** (문제 발생 시 디버깅):
```bash
python3 tests/regression/lib/fee_delegate.py sign \
  --sender-pk ... --fee-payer-pk ... --to ... | jq .
# 출력: rawTx, senderSigHash, feePayerSigHash 등 모두 확인 가능
```

### 5.9 "Governance proposal lifecycle" 헬퍼 활용 (Category E/F)

`lib/common.sh`의 governance 헬퍼로 proposal→approve→execute 전체 흐름 자동화:

```bash
source tests/regression/lib/common.sh
unlock_all_validators  # 모든 BP keystore unlock

# 방법 1 — gov_full_flow (한 번에)
receipt=$(gov_full_flow "$GOV_COUNCIL" "${propose_data}" "$VALIDATOR_1_ADDR" "$VALIDATOR_2_ADDR")
# propose + validator1 proposer + validator2 approve + validator1 execute

# 방법 2 — 단계별 호출
tx=$(gov_call "1" "$GOV_COUNCIL" "$propose_data" "$VALIDATOR_1_ADDR" 800000)
proposal_id=$(extract_proposal_id_from_receipt "1" "$tx")
gov_approve "1" "$GOV_COUNCIL" "$proposal_id" "$VALIDATOR_2_ADDR"
exec_tx=$(gov_execute "1" "$GOV_COUNCIL" "$proposal_id" "$VALIDATOR_1_ADDR")
```

**E/F 카테고리 대표 실행 예시**:
```bash
# 블랙리스트 전체 lifecycle
bash tests/regression/e-blacklist-authorized/e-01-sender-blacklisted.sh     # blacklist + tx reject
bash tests/regression/e-blacklist-authorized/e-04-unblacklist.sh            # unblacklist + tx 정상 처리

# Authorized 계정 + AuthorizedTxExecuted 이벤트 (PR #70 dependency)
bash tests/regression/e-blacklist-authorized/e-08-is-authorized.sh
bash tests/regression/e-blacklist-authorized/e-09-authorized-tx-executed.sh

# GovCouncil 4가지 이벤트 (F-5-06~09)
bash tests/regression/f-system-contracts/f5-01-blacklist.sh              # 선행: receipt 저장
bash tests/regression/f-system-contracts/f5-06-address-blacklisted-event.sh  # receipt 검증

# GasTip 거버넌스 + 헤더 동기화
bash tests/regression/f-system-contracts/f3-05-propose-gastip.sh   # proposeGasTip → execute → storage
bash tests/regression/b-wbft/b-06-gastip-header-sync.sh            # 다음 블록 WBFTExtra.GasTip 반영
```

---

## 6. 디버깅 & 트러블슈팅

### 6.1 테스트 실패 시 우선 확인할 것

```bash
# 1. 체인이 정상 기동 중인지
./chainbench.sh status

# 2. 블록이 생산되고 있는지 (1초 간격)
./chainbench.sh log tail 1 | grep "Commit new mining work"

# 3. 모든 노드가 동기화 상태인지
curl -s -X POST http://127.0.0.1:8501 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
curl -s -X POST http://127.0.0.1:8505 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### 6.2 단일 테스트를 verbose 모드로 실행

```bash
# 쉘 xtrace 활성화
bash -x tests/regression/a-ethereum/a2-01-legacy-tx.sh 2>&1 | tee /tmp/test-trace.log
```

### 6.3 특정 RPC 호출 수동 테스트

```bash
# helper 함수 직접 사용
source tests/regression/lib/common.sh

# raw tx 서명 & 전송
tx_hash=$(send_raw_tx "1" "$TEST_ACC_A_PK" "$TEST_ACC_B_ADDR" "1000000000000000" "" "21000" "dynamic")
echo "tx_hash=$tx_hash"

# receipt 대기
receipt=$(wait_tx_receipt_full "1" "$tx_hash" 30)
echo "$receipt" | jq .
```

### 6.4 상태 초기화 (누적 오염 제거)

```bash
# 체인 정지 + 데이터 삭제 + 재시작
./chainbench.sh stop
./chainbench.sh clean           # /tmp/node-data 삭제
./chainbench.sh init --profile regression
./chainbench.sh start
```

### 6.5 자주 발생하는 문제

#### 문제 1: "ImportError: No module named 'eth_account'"
```bash
pip3 install eth-account requests eth-utils websockets
```

#### 문제 2: "Connection refused to 127.0.0.1:8501"
```bash
# chainbench가 기동되지 않음
./chainbench.sh status
./chainbench.sh start
```

#### 문제 3: "nonce too low"
- 이전 테스트가 nonce를 소진한 후 같은 계정으로 재시도
- 해결: `./chainbench.sh clean && init && start`로 리셋

#### 문제 4: "transaction underpriced" — 예상치 못한 실패
- 실행 프로필의 `gasTip` (27600000000000 wei)보다 낮은 tip 사용
- `common.sh` 기본 tip은 27.6 Gwei

#### 문제 5: WebSocket 테스트 skip
- `pip3 install websockets` 필요
- `a4-06` / `a4-07` 실행 시 `SKIP: websockets not installed` 표시되면 설치

#### 문제 6: EIP-7702 SetCodeTx skip
- `eth-account` 0.13+ 필요. 구버전은 `sign_authorization` 미지원
- `pip3 install --upgrade eth-account`

---

## 7. CI/CD 통합

### 7.1 GitHub Actions 예시

`.github/workflows/regression-test.yml`:

```yaml
name: go-stablenet Regression Tests

on:
  push:
    branches: [dev, main]
  pull_request:
    branches: [dev, main]

jobs:
  regression:
    runs-on: ubuntu-latest
    timeout-minutes: 30

    steps:
      - name: Checkout go-stablenet
        uses: actions/checkout@v4
        with:
          path: go-stablenet

      - name: Checkout chainbench
        uses: actions/checkout@v4
        with:
          repository: 0xmhha/chainbench
          path: chainbench

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.11'

      - name: Install Python dependencies
        run: |
          pip3 install eth-account requests eth-utils websockets

      - name: Build gstable
        run: |
          cd go-stablenet
          make gstable

      - name: Configure chainbench binary path
        run: |
          cd chainbench
          sed -i 's|binary_path: ""|binary_path: "../go-stablenet/build/bin/gstable"|' profiles/regression.yaml

      - name: Initialize & start chain
        run: |
          cd chainbench
          ./setup.sh
          ./chainbench.sh init --profile regression
          ./chainbench.sh start
          sleep 5
          ./chainbench.sh status

      - name: Run regression tests
        run: |
          cd chainbench
          ./tests/regression/run-all.sh

      - name: Generate report
        if: always()
        run: |
          cd chainbench
          ./chainbench.sh report --format markdown > regression-report.md
          cat regression-report.md

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: regression-results
          path: chainbench/state/results/

      - name: Cleanup
        if: always()
        run: |
          cd chainbench
          ./chainbench.sh stop || true
```

### 7.2 로컬 pre-commit hook

`go-stablenet/.git/hooks/pre-push`:

```bash
#!/usr/bin/env bash
set -e

# 빠른 회귀 테스트 (Category A만)
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
./chainbench.sh status >/dev/null 2>&1 || {
  echo "chainbench is not running, skipping pre-push regression"
  exit 0
}

echo "Running quick regression (Category A)..."
./tests/regression/run-all.sh a-ethereum
```

---

## 8. Claude Code / MCP 활용

chainbench는 MCP (Model Context Protocol) 서버를 제공하므로, Claude Code에서 자연어로 테스트를 실행할 수 있습니다.

### 8.1 MCP 활성화

```bash
cd /Users/wm-it-22-00661/Work/github/stable-net/regression-test-case/go-stablenet
chainbench mcp enable
# → .mcp.json 생성

# Claude Code에서:
# /mcp
# → chainbench 서버 로드
```

### 8.2 Claude Code 자연어 예시

**예시 1 — 전체 회귀 테스트**:
```
User: "회귀 테스트 환경 초기화하고 전체 회귀 suite 실행해줘"

Claude:
  1. chainbench_init (profile: regression)
  2. chainbench_start
  3. chainbench_test_run (test: regression)
  4. chainbench_report
  → 결과 보고
```

**예시 2 — PR #70 회귀 검증**:
```
User: "PR #70 receipt.effectiveGasPrice 버그 회귀 테스트만 실행해"

Claude:
  1. chainbench_test_run (test: regression/a-ethereum/a2-08-effective-gas-price)
  2. 결과 확인 → BP/EN 양쪽 동일 값인지 검증
```

**예시 3 — 실패 디버깅**:
```
User: "regression/a-ethereum/a2-05a가 실패했는데 원인 알려줘"

Claude:
  1. chainbench_log_search (query: "underpriced", node: 1, tail: 50)
  2. 에러 메시지 분석 → 원인 설명
  3. 수정 제안 (예: tip/fee 값 조정)
```

**예시 4 — 새 테스트 케이스 추가**:
```
User: "RT-F-5-10 새 이벤트 테스트 추가해줘. AuthorizedAccountBatchAdded 이벤트를 검증하고 싶어"

Claude:
  1. 기존 f5-06 ~ f5-09 패턴 참고
  2. 새 파일 f5-10-authorized-batch-added.sh 생성
  3. 이벤트 시그니처 계산, proposal 실행, receipt logs 검색 로직 작성
  4. 카테고리 카운트 갱신
```

---

## 📚 관련 문서

- [REGRESSION_TEST_CASES_v2.md](../../../stable-net/regression-test-case/go-stablenet/REGRESSION_TEST_CASES_v2.md) — TC 정의 (103개)
- [REGRESSION_TEST_CASES_REVIEW.md](../../../stable-net/regression-test-case/go-stablenet/REGRESSION_TEST_CASES_REVIEW.md) — 각 TC의 상세 시나리오 + 코드 근거
- [README.md](README.md) — 테스트 suite 개요 및 TC↔파일 매핑
- [chainbench README](../../README.md) — chainbench 프레임워크 전반

---

## 🎯 권장 실행 순서

**개발 중 빠른 검증** (~2분):
```bash
./tests/regression/run-all.sh a-ethereum
```

**PR 검증** (~5분):
```bash
./tests/regression/run-all.sh a-ethereum b-wbft c-anzeon
```

**릴리즈 검증** (~15분):
```bash
./tests/regression/run-all.sh  # 전체
./chainbench.sh report --format markdown
```

**특정 기능 집중 검증**:
```bash
# Fee Delegation
./tests/regression/run-all.sh d-fee-delegation

# 거버넌스 이벤트
for f in tests/regression/f-system-contracts/f5-0*.sh; do bash "$f"; done
```
