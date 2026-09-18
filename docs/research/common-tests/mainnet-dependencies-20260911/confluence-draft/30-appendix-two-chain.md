# 부록 A. 두 체인에서만 가능한 테스트

세 체인 중 두 체인에서만 같은 목적으로 실행할 수 있는 기존 테스트다. 새 ID는 부여하지 않고 기존 ID로 적는다. 대부분 WEMIX4.0과 StableNet이 같은 합의 방식(WBFT)을 쓰기 때문에 생기는 묶음이다.

## WEMIX4.0 + StableNet (58개)

| 기존 ID | 출처 | 내용 | 비고 |
|---|---|---|---|
| RPC-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 블록 조회 결과에 합의 정보가 포함되는지까지 본다 | 세 체인 공통인 부분: 번호/해시 조회 일치 |
| RPC-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RPC-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RPC-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RPC-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RPC-010 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RPC-011 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RPC-022 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-A-2-10 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| RT-B-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 | 세 체인 공통인 부분: 블록 주기 프로필 값 비교 |
| RT-B-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-07 | Regression Test Case with scenario, Regression Test Case | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-08 | Regression Test Case with scenario, Regression Test Case | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-09 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-10 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-11 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-B-12 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| RT-G-3-01 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RT-G-3-02 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RT-G-3-03 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RT-G-3-04 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RT-G-3-05 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| RT-G-3-06 | Regression Test Case with scenario, Regression Test Case, [WEMIX 4.0] 테스트 시나리오 | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| T-1 (StableNet) | 2nd Change Test Cases (StableNet) | 트랜잭션 풀의 누적 잔액 검사를 겨냥한다. WEMIX3.0은 건별 검사만 확인됐다 |  |
| T-4 (StableNet) | 2nd Change Test Cases (StableNet) | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| T-5 (StableNet) | 2nd Change Test Cases (StableNet) | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| T-1 (WEMIX4.0) | [WEMIX 4.0] 2nd Change Test Cases | 트랜잭션 풀의 누적 잔액 검사를 겨냥한다. WEMIX3.0은 건별 검사만 확인됐다 |  |
| T-4 (WEMIX4.0) | [WEMIX 4.0] 2nd Change Test Cases | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| T-5 (WEMIX4.0) | [WEMIX 4.0] 2nd Change Test Cases | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |  |
| TC-1-2-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-1-2-02 | 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-1-2-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-1-2-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-1-2-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-1-2-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-4-2-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-4-2-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TC-4-2-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TS-1-2 | 1st Test Scenarios | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TS-4-2 | 1st Test Scenarios | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TX-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TX-009 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TX-019 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| TX-020 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |  |
| WBFT-001 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-002 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-003 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-004 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-005 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-006 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-007 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-009 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-010 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-011 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-012 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |
| WBFT-013 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |  |

## WEMIX3.0 + WEMIX4.0 (4개)

| 기존 ID | 출처 | 내용 | 비고 |
|---|---|---|---|
| BRIOCHE-01 | [WEMIX3.0] 테스트 시나리오 | Brioche 블록 보상을 검사한다. StableNet에는 블록 보상이 없다 |  |
| BRIOCHE-02 | [WEMIX3.0] 테스트 시나리오 | Brioche 블록 보상을 검사한다. StableNet에는 블록 보상이 없다 |  |
| BRIOCHE-03 | [WEMIX3.0] 테스트 시나리오 | Brioche 블록 보상을 검사한다. StableNet에는 블록 보상이 없다 |  |
| RPC-008 | [WEMIX4.0] Test, [WEMIX 4.0] 테스트 시나리오 | Brioche 블록 보상을 검사한다. StableNet에는 블록 보상이 없다 |  |

## WEMIX3.0 + StableNet (1개)

| 기존 ID | 출처 | 내용 | 비고 |
|---|---|---|---|
| RT-C-07 | Regression Test Case with scenario, Regression Test Case | 기본 수수료 상한을 검사한다. WEMIX4.0에는 상한이 없다 |  |

## 두 체인용 자동 테스트

| 실행 ID | 파일 | 범위 | 내용 |
|---|---|---|---|
| basic-wbft-consensus | basic/07-basic-wbft-consensus.json | WEMIX4.0 + StableNet | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |
| estimategas-authorizationlist-cost | go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | WEMIX4.0 + StableNet | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |
| basefee-maximum | go-stablenet/regression/anzeon/07-basefee-maximum.json | WEMIX3.0 + StableNet | 기본 수수료 상한을 검사한다. WEMIX4.0에는 상한이 없다 |
| node-address-returned | go-stablenet/regression/api/11-node-address-returned.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| validator-set-nonempty | go-stablenet/regression/api/12-validator-set-nonempty.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| validator-set-count | go-stablenet/regression/api/12b-validator-set-count.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| commit-signers-quorum | go-stablenet/regression/api/13-commit-signers-quorum.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| wbft-extra-info-fields | go-stablenet/regression/api/14-wbft-extra-info-fields.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| istanbul-status-fields | go-stablenet/regression/api/15-istanbul-status-fields.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| is-validator-flags | go-stablenet/regression/api/16-is-validator-flags.json | WEMIX4.0 + StableNet | 합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다 |
| set-code-delegation | go-stablenet/regression/ethereum/18-set-code-delegation.json | WEMIX4.0 + StableNet | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |
| wbft-seals-quorum | go-stablenet/regression/wbft/02-wbft-seals-quorum.json | WEMIX4.0 + StableNet | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |
| epoch-transition-carries-epoch-info | go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | WEMIX4.0 + StableNet | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |
| prev-seals-quorum | go-stablenet/regression/wbft/11-prev-seals-quorum.json | WEMIX4.0 + StableNet | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |
| randao-and-mixdigest-present | go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | WEMIX4.0 + StableNet | WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다 |
| secp256r1-precompile-valid | go-wbft/accounts/01-secp256r1-precompile-valid.json | WEMIX4.0 + StableNet | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |
| secp256r1-precompile-invalid | go-wbft/accounts/02-secp256r1-precompile-invalid.json | WEMIX4.0 + StableNet | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |
| secp256r1-precompile-short-input | go-wbft/accounts/03-secp256r1-precompile-short-input.json | WEMIX4.0 + StableNet | 계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다 |
| wemix-brioche-block-reward | go-wemix/rpc/01-wemix-brioche-block-reward.json | WEMIX3.0 + WEMIX4.0 | Brioche 블록 보상을 검사한다. StableNet에는 블록 보상이 없다 |
