---
id: 2599289086
title: "1st Test Scenarios"
url: https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2599289086
version: 50
createdAt: 2026-08-05T01:42:49.314Z
fetched_at: 2026-09-11T07:02:34.095513+00:00
---

## 1. 문서 목적

- 릴리즈 변경사항을 실제 네트워크 흐름, 노드 기동/동기화, RPC 관찰, 하드포크 전환, 설정 재기동 관점에서 검증할 수 있도록 정의한다.
- 시나리오 만 보고 테스트 스크립트를 작성할 수 있는 수준의 상세도를 유지한다.

## 2. 범위와 제외

- 본문 매핑 범위: `TC-1-1-01` \~ `TC-5-3-01` + `TC-4-5-01` \~ `TC-4-5-12` 총 **70건**
- 취약점 패치 항목(`2-1` \~ `2-4`)은 운영 환경 자연 재현이 어렵기 때문에  별도 기술한다

### **2-1 테스트 제외 케이스**

> 정상 노드 환경에서 자연 발생하지 않으므로 통합 테스트 대상에서 제외한다. 단위 테스트로만 검증한다.

| **섹션** | **변경사항** | **단위 테스트 위치** |
| --- | --- | --- |
| 2-1 | 잘못된 KZG 증명 피어 차단 | `eth/fetcher/tx_fetcher_test.go` → `TestTransactionProtocolViolation` |
| 2-2 | 암호화 취약점 수정 - ECIES | `p2p/rlpx/rlpx_oracle_poc_test.go` |
| 2-3 | secp256k1 좌표 유효성 검사 강화 | `crypto/secp256k1` 패키지 테스트 |
| 2-4 | P2P 메시지 DoS 취약점 수정 | `p2p/tracker`, `eth/protocols/eth`, `eth/protocols/snap` 패키지 테스트 |

> 코드 기반 테스트 불가

| TC | 사유 |
| --- | --- |
| TC-1-1-08 | `_cleanupBurnDeposit`은 Solidity `private` 함수로 외부 직접 호출 불가. 동일 proposalId에 대해 2회 터미널 상태 전환이 컨트랙트 상태 머신 구조상 불가능하여 간접 검증도 불가 |
| TC-3-1-01 | "기존 대비 38% 향상" 비교는 이전 libsecp256k1 버전 부재로 수치 검증 불가 |
| TC-3-1-02 | 동일 사유 (BenchmarkRecover 28% 향상 비교 불가) |
| TC-3-1-03 | 동일 사유 (BenchmarkVerifySignature 29% 향상 비교 불가) |
| TC-3-1-04 | 노드 동기화 과정에서 자동 검증 |
| TC-5-1-02 | `make test` 전체 통과는 CI 파이프라인 의존적이며 통합 테스트 시나리오 범위 외 |
| TC-5-1-03 | `runtime.Version()` 확인은 빌드된 `gstable` 바이너리를 직접 실행해야 하며 Go 테스트로 자동화 불가 |

## 3. 테스트 환경 파라미터 정의

### 3-1. 네트워크 토폴로지

| IP | 역할 | 타입 |
| --- | --- | --- |
| <내부 IP 생략: 노드 1~7> | Validator Node | 블록 생성·합의 |
| <내부 IP 생략: 노드 8> | EN Node (Snap Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 9> | EN Node (Snap Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 10> | EN Node (Snap Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 11> | EN Node (Full Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 12> | EN Node (Full Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 13> | EN Node (Full Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 14> | EN Node (Full Sync) | 동기화 노드 |
| <내부 IP 생략: 노드 15> | PN (EN + Bootnode) | P2P 연결 기준점 |

#### 3-2. 거버넌스 파라미터 

| 변수명 | 제안 값 | 정의 |
| --- | --- | --- |
| `QUORUM` | `2` (절대 수, 명) | 제안 승인에 필요한 최소 찬성 수. 운영 genesis 기준 quorum=2. 코드: `uint32 public quorum` |
| `EXPIRY` | `60` (= 1분) | 제안 만료까지의 시간(**초** 단위, timestamp 기반). 코드: `proposalExpiry` (`block.timestamp` 비교) |

#### 3-3. 최소 테스트 계정 

> 테스트에 필요한 최소 계정과 조건 권한등은 아래와 같다. 테스트 효율화를 위해 임의로 만들어 사용하고 필요에 따라 더 늘릴 수 있다.
> 
> 1. **M1\~M3**: 테스트 전용 keystore를 새로 생성할지, 기존 validator 계정을 재사용할지 결정 필요. 비밀번호 관리 방식(평문 파일 vs 환경변수) 결정 필요
> 2. **인증 계정 A1**: `AuthorizedAccount`로 등록하는 방법 및 등록 주체 확인 필요 (TC-4-6-01\~04)
> 3. **잔액**: M1은 `DEPOSIT_AMOUNT` + 가스비 이상의 잔액 필요. 초기 배분 방법 확인 필요

| 역할 | 최소 필요 조건 |
| --- | --- |
| minter 멤버1 (제안자) minter 멤버12(제안자) | GovMinter 멤버 등록, 충분한 잔액 |
| minter 멤버2 (투표자) | GovMinter 멤버 등록 |
| minter 멤버3 (투표자) | GovMinter 멤버 등록 |
| 일반 계정 | 잔액 보유, 특수 권한 없음 |
| 인증 계정 (AuthorizedAccount) | GovCouncil `_currentAuthorizedAccounts`에 등록된 계정. 트랜잭션 전송 시 프로토콜이 `AuthorizedTxExecuted` 이벤트를 receipt 마지막 로그에 자동 추가 |

## 4. 시나리오 목록 과 시나리오 정의

### 4.1 GovMinter 시스템 컨트랙트 업그레이드

- 시나리오별 네트워크 구동에 필요한 제네시스 파일을 첨부를 활용하여 이용한다.

> TS-1-1-01 (취소·환불) / TS-1-1-02 (거부·실행, private)  : 
> 
> TS-1-1-03 (만료 배치, private)  :  활용

---

**TS-1-1-01  GovMinter v2 취소 기반 환불 검증**:  

**목적**

- Boho 하드포크 이후 GovMinter v2에서 소각 제안이 취소되면 proposer의 `burnBalance`가 `refundableBalance`로 이동하는지 검증한다. 
- M1, M2가 각자 자기 소각 제안을 생성하고 취소하는 흐름을 검증한다.

**사전 조건**:

- genesis-standard.json( 에 필요한 부분을 수정한다.(예: m1, m2, bohoblock , validator )
- 테스트넷 기동
- M1, M2는 각각 자기 명의의 burn proposal을 생성하고 취소할 수 있는 멤버 계정이어야 함
- M1, M2 모두 시작 시점에 `burnBalance == 0`, `refundableBalance == 0`
- `claimBurnRefund()` 호출용 일반 계정 또는 refundableBalance가 0인 계정 준비 

**실행 단계**

**A. M1 취소 후 refundableBalance 생성**

1. M1이 `proposeBurn`으로 자기 명의의 소각 제안을 생성한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "proposeBurn(bytes)" <PROOF_DATA> \
  --value <BURN_AMOUNT_WEI> \
  --rpc-url <RPC_URL> \
  --private-key <M1_PRIVATE_KEY>
```
2. 생성 직후 `burnBalance[M1]`가 예치금만큼 증가했는지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```
3. M1이 자신의 `proposalId`에 대해 `cancelProposal`을 호출한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "cancelProposal(uint256)" <PROPOSAL_ID> \
  --rpc-url <RPC_URL> \
  --private-key <M1_PRIVATE_KEY>
```
4. 취소 직후 proposal 상태가 `Cancelled`인지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" <PROPOSAL_ID> \
  --rpc-url <RPC_URL>
```

   `ProposalStatus.Cancelled` 값은 `4`이다.
5. 취소 직후 `burnBalance[M1] == 0`인지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```
6. 취소 직후 `refundableBalance[M1] == M1 예치금`인지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

**B. M2 취소 후 refundableBalance 생성**

1. M2가 `proposeBurn`으로 자기 명의의 소각 제안을 생성한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "proposeBurn(bytes)" <PROOF_DATA> \
  --value <BURN_AMOUNT_WEI> \
  --rpc-url <RPC_URL> \
  --private-key <M2_PRIVATE_KEY>
```
2. 생성 직후 `burnBalance[M2]`가 예치금만큼 증가했는지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M2_ADDR> \
  --rpc-url <RPC_URL>
```
3. M2가 자신의 `proposalId`에 대해 `cancelProposal`을 호출한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "cancelProposal(uint256)" <PROPOSAL_ID> \
  --rpc-url <RPC_URL> \
  --private-key <M2_PRIVATE_KEY>
```
4. 취소 직후 proposal 상태가 `Cancelled`인지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" <PROPOSAL_ID> \
  --rpc-url <RPC_URL>
```

   `ProposalStatus.Cancelled` 값은 `4`이다.
5. 취소 직후 `burnBalance[M2] == 0`인지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M2_ADDR> \
  --rpc-url <RPC_URL>
```
6. 취소 직후 `refundableBalance[M2] == M2 예치금`인지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <M2_ADDR> \
  --rpc-url <RPC_URL>
```

**C. 환불 청구 및 재청구 방어**

1. A 또는 B에서 `refundableBalance > 0` 상태가 된 계정으로 `claimBurnRefund()`를 호출한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "claimBurnRefund()" \
  --rpc-url <RPC_URL> \
  --private-key <PRIVATE_KEY>
```
2. 호출 직후 `refundableBalance`가 `0`으로 초기화되었는지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <CLAIMANT_ADDR> \
  --rpc-url <RPC_URL>
```
3. 해당 계정의 WKRC 잔액이 환불 금액만큼 증가했는지 확인한다.
4. 동일 계정으로 `claimBurnRefund()`를 다시 호출한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "claimBurnRefund()" \
  --rpc-url <RPC_URL> \
  --private-key <PRIVATE_KEY>
```
5. 두 번째 호출이 `NoRefundAvailable`로 revert 되는지 확인한다.

**D. 초기 잔액 0 계정의 즉시 청구 실패**

1. `refundableBalance == 0`인 계정을 준비한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <ZERO_REFUND_ADDR> \
  --rpc-url <RPC_URL>
```
2. 해당 계정으로 `claimBurnRefund()`를 호출한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "claimBurnRefund()" \
  --rpc-url <RPC_URL> \
  --private-key <ZERO_REFUND_PRIVATE_KEY>
```
3. 호출이 `NoRefundAvailable`로 revert 되는지 확인한다.

**검증 포인트**

- M1, M2가 각자 자기 proposal을 취소한 뒤 proposer별 `burnBalance`는 0이 되고 `refundableBalance`는 각 예치금만큼 증가한다.
- 취소 후 `claimBurnRefund` 1회 호출 시 환불 금액이 계정으로 전송되고 `refundableBalance`는 0이 된다.
- 환불 수령 직후 동일 계정의 재호출은 `NoRefundAvailable`로 revert 된다.
- 시작 시점에 `refundableBalance == 0`인 계정의 `claimBurnRefund` 호출도 `NoRefundAvailable`로 revert 된다.

**테스트케이스 매핑**

- A: `TC-1-1-01`
- B: `TC-1-1-01`
- C: `TC-1-1-05`, `TC-1-1-07`
- D: `TC-1-1-06`

---

#### TS-1-1-02: GovMinter v2 거부·실행·정리 로직 검증

**목적**

- Boho 하드포크 이후 GovMinter v2에서 소각 제안이 `Rejected` 상태가 되면 환불 가능 잔액으로 이동하고, `Executed` 상태에서는 환불 잔액이 증가하지 않으며, 종료 후 정리 로직과 환불 이벤트가 기대대로 동작하는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예: m1, m2, bohoblock , validator )
- 테스트넷 기동
- M1은 burn proposal proposer, M2/M3는 투표 참여 가능한 멤버 계정이어야 함
- 거부 정족수와 실행 정족수를 충족할 수 있는 멤버 구성 준비
- 실행 경로 검증을 위해 GovMinter가 소각 가능한 WKRC 잔액을 보유한 상태 준비
- 이벤트 검증을 위해 트랜잭션 receipt와 로그 디코딩 환경 준비

**실행 단계**

**A. 거부 시 refundableBalance 이동**

1. M1이 `proposeBurn`으로 자기 명의의 소각 제안을 생성한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "proposeBurn(bytes)" <PROOF_DATA> \
  --value <BURN_AMOUNT_WEI> \
  --rpc-url <RPC_URL> \
  --private-key <M1_PRIVATE_KEY>
```
2. 생성 직후 `burnBalance[M1]`가 예치금만큼 증가했는지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```
3. M2, M3가 `disapproveProposal`을 호출해 제안을 `Rejected` 상태로 만든다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "disapproveProposal(uint256)" <PROPOSAL_ID> \
  --rpc-url <RPC_URL> \
  --private-key <M2_PRIVATE_KEY>
```

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "disapproveProposal(uint256)" <PROPOSAL_ID> \
  --rpc-url <RPC_URL> \
  --private-key <M3_PRIVATE_KEY>
```
4. 종료 직후 `burnBalance[M1]`, `refundableBalance[M1]`, `getProposal(<PROPOSAL_ID>)`를 조회한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" <PROPOSAL_ID> \
  --rpc-url <RPC_URL>
```

   `getProposal` 반환값의 `status(uint8)`가 `7`이면 `ProposalStatus.Rejected` 상태다.
5. 거부 트랜잭션 receipt에서 `BurnDepositRefunded(proposalId, proposer, amount)` 이벤트를 확인한다.

```bash
cast receipt <REJECT_TX_HASH> --rpc-url <RPC_URL>
```

**B. 실행 성공 시 환불 미이동**

1. M1이 새 `proposeBurn` 제안을 생성한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "proposeBurn(bytes)" <PROOF_DATA> \
  --value <BURN_AMOUNT_WEI> \
  --rpc-url <RPC_URL> \
  --private-key <M1_PRIVATE_KEY>
```
2. 실행에 필요한 WKRC 잔액이 GovMinter에 준비되어 있는지 확인한다.
3. M2 또는 다른 멤버가 `approveProposal`을 호출해 제안을 `Executed` 상태로 만든다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "approveProposal(uint256)" <PROPOSAL_ID> \
  --rpc-url <RPC_URL> \
  --private-key <M2_PRIVATE_KEY>
```
4. 실행 직후 `burnBalance[M1]`, `refundableBalance[M1]`, `getProposal(<PROPOSAL_ID>)`를 조회한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" <PROPOSAL_ID> \
  --rpc-url <RPC_URL>
```

   `getProposal` 반환값의 `status(uint8)`가 `3`이면 `ProposalStatus.Executed` 상태다.

**C. 정리 로직 멱등성 확인**

1. 종료 상태가 된 burn proposal 1건을 준비한다.
2. 첫 번째 종료 처리 직후 `burnBalance`, `refundableBalance`, 관련 이벤트 기록을 저장한다.
3. 동일 `proposalId`에 대해 정리 경로를 다시 태우는 하네스(테스트 전용 helper) 또는 helper를 사용한다.
4. 두 번째 실행 후 동일 상태값과 이벤트 개수를 다시 확인한다.

**검증 포인트**

- `Rejected` 상태에서는 `burnBalance[M1] == 0`이고 `refundableBalance[M1] == 예치금`이다.
- 거부를 발생시킨 최종 트랜잭션 receipt에 `BurnDepositRefunded(proposalId, proposer, amount)` 이벤트가 1회 기록된다.
- 이벤트의 `proposalId`, `proposer`, `amount` 값은 실제 제안 정보와 일치한다.
- `Executed` 상태에서는 `burnBalance[M1]`는 0으로 소각 처리되고 `refundableBalance[M1]`는 증가하지 않는다.
- 동일 `proposalId`에 대해 정리 로직을 다시 실행해도 추가 환불, 추가 차감, 중복 이벤트가 발생하지 않는다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-1-1-02`, `TC-1-1-10`
- B: `TC-1-1-04`
- C: `TC-1-1-08`

---

#### TS-1-1-03: GovMinter v2 만료 전용 배치 검증

**목적**

- Boho 하드포크 이후 GovMinter v2에서 만료된 소각 제안의 예치금이 `refundableBalance`로 이동하는지 검증한다. 만료 테스트는 proposal expiry를 `60초`로 설정한 별도 네트워크를 생성해야 하므로, 만료 관련 검증을 한 번에 묶어 실행한다.

**사전 조건**

- genesis-expiry60.json 에 필요한 부분을 수정한다.(예: m1, m2, bohoblock , validator )
-  테스트넷 기동
- M1은 proposer, M2는 `expireProposal` 호출 가능한 멤버 계정이어야 함
- 만료 전용 네트워크에서 `burnBalance == 0`, `refundableBalance == 0` 상태 확인
- 이벤트 검증을 위해 트랜잭션 receipt와 로그 디코딩 환경 준비

**실행 단계**

**A. 만료 후 refundableBalance 이동**

1. 만료 전용 네트워크를 기동하고 expiry 설정값이 `60초`인지 확인한다.

- GovMinter params의 \`expiry\`를 \`60\`으로 설정 후 체인을 구동
- 노드 기동 후 GovMinter의 \`proposalExpiry()\`를 조회해 온체인 설정값이 \`60\`인지 확인한다.

1. M1이 `proposeBurn`으로 자기 명의의 소각 제안을 생성한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "proposeBurn(bytes)" <PROOF_DATA> \
  --value <BURN_AMOUNT_WEI> \
  --rpc-url <RPC_URL> \
  --private-key <M1_PRIVATE_KEY>
```
2. 생성 직후 `burnBalance[M1]`가 예치금만큼 증가했는지 확인한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```
3. proposal expiry가 지나도록 대기한 뒤 M2가 `expireProposal`을 호출한다.

```bash
cast send 0x0000000000000000000000000000000000001003 \
  "expireProposal(uint256)" <PROPOSAL_ID> \
  --rpc-url <RPC_URL> \
  --private-key <M2_PRIVATE_KEY>
```
4. 종료 직후 proposal 상태를 확인하고 `burnBalance[M1]`, `refundableBalance[M1]`를 조회한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" <PROPOSAL_ID> \
  --rpc-url <RPC_URL>
```

   `ProposalStatus.Expired` 값은 `5`이다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "burnBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "refundableBalance(address)(uint256)" <M1_ADDR> \
  --rpc-url <RPC_URL>
```

**B. 만료 시 환불 이벤트 확인**

1. A 단계의 만료 트랜잭션 receipt를 수집한다.

```bash
cast receipt <EXPIRE_TX_HASH> --rpc-url <RPC_URL>
```
2. `BurnDepositRefunded(proposalId, proposer, amount)` 이벤트를 디코딩한다.
3. 이벤트 값과 실제 proposal 정보를 대조한다.

**C. 동일 네트워크에서 만료 케이스 일괄 반복**

1. 같은 만료 전용 네트워크에서 추가 burn proposal을 생성한다.
2. expiry 대기 후 다시 `expireProposal`을 호출한다.
3. 각 proposal별 `refundableBalance` 증가와 이벤트 값을 비교한다.

**검증 포인트**

- 만료된 proposal의 상태는 `Expired`다.
- 만료 직후 `burnBalance[M1] == 0`이고 `refundableBalance[M1] == 예치금`이다.
- 만료 트랜잭션 receipt에 `BurnDepositRefunded(proposalId, proposer, amount)` 이벤트가 1회 기록된다.
- 동일 만료 전용 네트워크에서 여러 proposal을 반복 검증해도 proposal별 환불 금액과 이벤트 값이 일치한다.
- 만료 테스트는 운영 기본 네트워크가 아니라 expiry=`60초` 전용 네트워크에서 일괄 실행한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-1-1-03`
- B: `TC-1-1-10`
- C: `TC-1-1-03`, `TC-1-1-10`

---

#### TS-1-1-04: Boho 하드포크 전후 GovMinter 바이트코드 및 스토리지 보존 검증

**목적**

- Boho 하드포크 블록에 도달했을 때 GovMinter 시스템 컨트랙트의 바이트코드가 v1에서 v2로 정상적으로 교체되는지 확인한다(HardFork)
- 업그레이드 이후에도 기존 v1에서 사용하던 스토리지 상태(예: mintBalance 등)가 유실 없이 보존되는지 검증한다.

**사전 조건**

- Boho 하드포크가 특정 블록(예: `bohoBlock: 100`)에 활성화되도록 설정된 테스트넷 기동
- GovMinter v1 상태에서 일부 데이터가 존재해야 함 (예: 특정 계정의 `mintBalance`가 0이 아님)

**실행 단계**

**A. Boho 하드포크 전 상태 기록 (Block \< BohoBlock)**

1. Boho 하드포크 이전 블록에서 GovMinter의 바이트코드를 조회하고 길이를 기록한다.

```bash
cast code 0x0000000000000000000000000000000000001003 --rpc-url <RPC_URL> | wc -c
```
2. v1 상태의 데이터를 확인한다. 여기서는 `mintBalance`를 예로 사용한다.

```bash
cast call 0x0000000000000000000000000000000000001003 \
  "mintBalance(address)(uint256)" <TEST_ADDR> \
  --rpc-url <RPC_URL>
```

**B. Boho 하드포크 도달 및 바이트코드 변경 확인 (Block \>= BohoBlock)**

1. 네트워크 블록 번호가 `bohoBlock` 이상이 될 때까지 대기한다.

```bash
cast block-number --rpc-url <RPC_URL>
```
2. 하드포크 이후 GovMinter의 바이트코드를 다시 조회하여 변경되었는지 확인한다.

```bash
cast code 0x0000000000000000000000000000000000001003 --rpc-url <RPC_URL> | wc -c
```

   *기대 결과: v1 바이트코드 길이보다 증가하거나 내용이 변경되어야 함.*

**C. 기존 v1 스토리지 상태 보존 확인**

1. 하드포크 이전에 확인했던 quorum 보존 , gasTip 보존, v2 신규 함수 호출 가능 (빈 매핑 = 0) 조회한다.
2. `TEST_ADDR`의 `mintBalance`를 다시 조회한다. (BOHO\_BLOCK =100)

```bash
QUORUM_100=$(cast call "$GV" 'quorum()(uint256)' --block "$BOHO_BLOCK" --rpc-url )
GASTIP_100=$(cast call "$GV" 'gasTip()(uint256)' --block "$BOHO_BLOCK" --rpc-url )
V2_CALL=$(cast call "$GMI" 'refundableBalance(address)(uint256)' "${TEST_ACCOUNT}" \
  --block "$BOHO_BLOCK" --rpc-url )
```
3. 단계 A-2에서 기록한 값과 동일한지 비교한다.

**검증 포인트**

- Boho 블록 도달 시 GovMinter의 바이트코드가 v1에서 v2로 자동 교체된다 (코드 길이 또는 해시 변경).
- 하드포크 이후에도 기존에 저장되어 있던 `mintBalance`, `proposal` 등의 스토리지 데이터가 그대로 유지된다.
- 업그레이드 과정에서 컨트랙트의 잔액(Balance)도 변함없이 유지된다.

**테스트케이스 매핑**

- A, B: `TC-1-1-11`
- C: `TC-1-1-12`

---

### 4.2 secp256r1 서명 검증 지원 (EIP-7951)

> secp256r1 프리컴파일 주소: `0x0000000000000000000000000000000000000100`
> 
> P256\_VALID\_INPUT 값: `core/vm/contracts_test.go:422` 에 정의된 값

---

#### TS-1-2: secp256r1 프리컴파일 런타임 및 단위 검증

**목적**

- Boho 하드포크 이후 secp256r1 프리컴파일이 런타임 `eth_call` 경로에서 정상 동작하는지 검증
- `core/vm` 레벨에서 주소 등록 및 입력 처리 규칙이 코드와 일치하는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동
- 노드가 `eth_call` RPC를 처리할 수 있는 상태
- `curl`, `jq` 또는 동등한 JSON 응답 확인 도구 준비
- 단위 테스트 검증 시 `go test ./core/vm/...` 실행 가능한 환경 준비

**실행 단계**

**A. Boho 환경 런타임 cast call 연속 검증**

1. 아래 유효 입력을 준비한다.

```bash
P256_VALID_INPUT=0x4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e
```
2. 유효 입력으로 `cast call`을 실행한다.

```bash
cast call 0x0000000000000000000000000000000000000100 \
  --data "$P256_VALID_INPUT" \
  --rpc-url <RPC_URL>

반환값이 아래 값과 일치하는지 확인한다. (성공)
   ```text
   0x0000000000000000000000000000000000000000000000000000000000000001
   ```
```
3. 아래 무효 입력을 준비한다.

```bash
P256_INVALID_INPUT=0xbb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023d45c5740946b2a147f59262ee6f5bc90bd01ed280528b62b3aed5fc93f06f739b329f479a2bbd0a5c384ee1493b1f5186a87139cac5df4087c134b49156847db2927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e
```
4. 무효 입력으로 `cast call`을 실행한다.

```bash
cast call 0x0000000000000000000000000000000000000100 \
  --data "$P256_INVALID_INPUT" \
  --rpc-url <RPC_URL>

반환값이 빈 결과인지 확인한다.(실패/조건불충분)
   ```text
   0x
   ```
```
5. 길이 부족 입력을 준비한다.

```bash
P256_SHORT_INPUT=0x$(printf 'aa%.0s' {1..159})
```
6. 길이 부족 입력으로 `cast call`을 실행한다.

```bash
cast call 0x0000000000000000000000000000000000000100 \
  --data "$P256_SHORT_INPUT" \
  --rpc-url <RPC_URL>

반환값이 빈 결과인지 확인한다.(실패/조건불충분)
   ```text
   0x
   ```
```

**B. 단위 테스트 기반 등록 확인**

1. `go test ./core/vm/...`를 실행한다.

```bash
go test ./core/vm/...
```
2. `core/vm/contracts.go`에서 `PrecompiledContractsBoho`, `PrecompiledAddressesBoho`, `ActivePrecompiles()`를 확인한다.

**검증 포인트**

- 유효 입력 호출은 명령 실패 없이 성공해야 한다.
- 유효 입력 호출 결과는 `0x0000000000000000000000000000000000000000000000000000000000000001`이어야 한다.
- 무효 입력 호출은 명령 실패 없이 빈 결과 `0x`를 반환해야 한다.
- 길이 부족 입력 호출도 명령 실패 없이 빈 결과 `0x`를 반환해야 한다.
- `core/vm/contracts.go`에는 `PrecompiledContractsBoho`와 `PrecompiledAddressesBoho`에 `0x0100` 주소가 등록되어 있어야 한다.
- 현재 저장소에는 `PrecompiledAddressesBoho` 전용 테스트 함수가 보이지 않으므로, `TC-1-2-06`을 자동화하려면 전용 테스트 추가가 필요하다.

**실행 단계와 관련 테스트케이스 매핑**

- A-1, A-2: `TC-1-2-01`, `TC-1-2-03`
- A-3, A-4: `TC-1-2-04`
- A-5, A-6: `TC-1-2-05`
- B: `TC-1-2-06`

---

### 4.3 최소 가스비 하한선 적용

검증 기준값들

> - `MinBaseFee`: `20,000,000,000,000 wei`
> - `InitialGasTip`: `27,600,000,000,000 wei`
> - `minFee`: `47,600,000,000,000 wei`
> - `belowMinFee`: `47,599,999,999,999 wei`

---

#### TS-1-3-01: GasFeeCap 경계값 검증

**목적**

- Anzeon + London 하드포크가 활성화된 환경에서 DynamicFeeTx의 `GasFeeCap`이 `minFee (= MinBaseFee + MinTip)` 기준으로 `미만`, `동등`, `초과`일 때 txpool 진입 여부가 올바르게 달라지는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동
- txpool이 `opts.MinTip = 27,600,000,000,000 wei`로 초기화된 상태
- 테스트 계정 A가 수수료를 감당할 수 있는 충분한 잔액 보유
- `eth_sendRawTransaction`, `eth_getTransactionByHash`, `txpool_content` RPC 사용 가능
- 단계별 서명된 raw transaction 준비 

**실행 단계**

**A. belowMinFee 거부 확인**

1. 아래 조건으로 DynamicFeeTx raw transaction을 준비한다.

```text
nonce      = 0
gas        = 21000
gasFeeCap  = 47599999999999
gasTipCap  = 27600000000000
value      = 0
```
2. raw transaction을 제출한다.

```bash
cast rpc eth_sendRawTransaction <RAW_TX_BELOW> --rpc-url <RPC_URL>
```
3. 반환값이 에러인지 확인한다.  
기대 결과: 에러 메시지에 아래 문자열이 포함된다.

```text
underpriced transaction: gas fee cap 47599999999999, minimum needed 47600000000000
```
4. txpool에 해당 tx가 없는지 확인한다.

```bash
cast rpc txpool_content --rpc-url <RPC_URL>
```

   기대 결과: `pending`과 `queued`에 `<SENDER_ADDR>`의 해당 nonce tx가 없다.

**B. atMinFee 수락 확인**

1. 아래 조건으로 DynamicFeeTx raw transaction을 준비한다.

```text
nonce      = 0
gas        = 21000
gasFeeCap  = 47600000000000
gasTipCap  = 27600000000000
value      = 0
```
2. raw transaction을 제출한다.

```bash
cast rpc eth_sendRawTransaction <RAW_TX_EQUAL> --rpc-url <RPC_URL>
```
3. 반환값이 tx hash인지 확인한다.  
기대 결과: `0x`로 시작하는 32바이트 tx hash가 반환된다.
4. tx 조회 결과를 확인한다.

```bash
cast rpc eth_getTransactionByHash <TX_HASH_EQUAL> --rpc-url <RPC_URL>
```

   기대 결과: `result.hash`가 `<TX_HASH_EQUAL>`과 같고, pending 상태면 `result.blockHash`는 `null`이다.

**C. aboveMinFee 수락 확인**

1. 아래 조건으로 DynamicFeeTx raw transaction을 준비한다.

```text
nonce      = 1
gas        = 21000
gasFeeCap  = 47600000000001
gasTipCap  = 27600000000000
value      = 0
```
2. raw transaction을 제출한다.

```bash
cast rpc eth_sendRawTransaction <RAW_TX_ABOVE> --rpc-url <RPC_URL>
```
3. 반환값이 tx hash인지 확인한다.  
기대 결과: `0x`로 시작하는 32바이트 tx hash가 반환된다.
4. tx 조회 결과를 확인한다.

```bash
cast rpc eth_getTransactionByHash <TX_HASH_ABOVE> --rpc-url <RPC_URL>
```

   기대 결과: `result.hash`가 `<TX_HASH_ABOVE>`와 같고, pending 상태면 `result.blockHash`는 `null`이다.

**검증 포인트**

- belowMinFee 제출은 `txpool.ErrUnderpriced` 계열 에러로 거부되어야 한다.
- belowMinFee 에러 문자열에는 `gas fee cap 47599999999999, minimum needed 47600000000000`가 포함되어야 한다.
- atMinFee 제출은 에러 없이 수락되고 32바이트 tx hash가 반환되어야 한다.
- aboveMinFee 제출도 에러 없이 수락되고 32바이트 tx hash가 반환되어야 한다.
- 수락된 tx는 `eth_getTransactionByHash` 조회 시 `result != null`이어야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-1-3-01`
- B: `TC-1-3-02`
- C: `TC-1-3-03`

---

#### TS-1-3-02: tx 타입별 최솟값 미만 가스비 거부 검증

**목적**

- Anzeon + London 하드포크가 활성화된 환경에서 `LegacyTx`, `AccessListTx`, `DynamicFeeTx`가 모두 동일한 최소 가스비 하한선 `minFee (= MinBaseFee + MinTip)`의 적용을 받아, 하한선 미만이면 txpool 진입 전에 거부되는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동
- txpool이 `opts.MinTip = 27,600,000,000,000 wei`로 초기화된 상태 
- 테스트 계정 A가 수수료를 감당할 수 있는 충분한 잔액 보유
- `eth_sendRawTransaction`, `txpool_content` RPC 사용 가능
- 타입별 서명된 raw transaction 준비 

**실행 단계**

**A. LegacyTx 거부 확인**

1. 아래 조건으로 LegacyTx raw transaction을 준비한다.

```text
type       = LegacyTx
nonce      = 0
gas        = 21000
gasPrice   = 47599999999999
value      = 0
```
2. raw transaction을 제출한다.

```bash
cast rpc eth_sendRawTransaction <LEGACY_RAW_TX> --rpc-url <RPC_URL>
```
3. 에러가 반환되고, 에러 메시지에 아래 문자열이 포함되는지 확인한다.

```text
underpriced transaction: gas fee cap 47599999999999, minimum needed 47600000000000
```

**B. AccessListTx 거부 확인**

1. 아래 조건으로 AccessListTx raw transaction을 준비한다.

```text
type       = AccessListTx
nonce      = 0
gas        = 21000
gasPrice   = 47599999999999
accessList = []
value      = 0
```
2. raw transaction을 제출한다.

```bash
cast rpc eth_sendRawTransaction <ACCESSLIST_RAW_TX> --rpc-url <RPC_URL>
```
3. 에러가 반환되고, 에러 메시지에 아래 문자열이 포함되는지 확인한다.

```text
underpriced transaction: gas fee cap 47599999999999, minimum needed 47600000000000
```

**C. DynamicFeeTx 거부 확인**

1. 아래 조건으로 DynamicFeeTx raw transaction을 준비한다.

```text
type       = DynamicFeeTx
nonce      = 0
gas        = 21000
gasFeeCap  = 47599999999999
gasTipCap  = 27600000000000
value      = 0
```
2. raw transaction을 제출한다.

```bash
cast rpc eth_sendRawTransaction <DYNAMIC_RAW_TX> --rpc-url <RPC_URL>
```
3. 에러가 반환되고, 에러 메시지에 아래 문자열이 포함되는지 확인한다.

```text
underpriced transaction: gas fee cap 47599999999999, minimum needed 47600000000000
```
4. MinTip 선행 거부와 혼동되지 않는지 확인한다.  
기대 결과: 에러 메시지가 `gas tip cap ..., minimum needed ...`가 아니라 `gas fee cap ..., minimum needed ...` 형식이어야 한다.

**D. txpool 미진입 확인**

1. txpool 상태를 조회한다.

```bash
cast rpc txpool_content --rpc-url <RPC_URL>
```
2. sender 기준으로 pending/queued를 확인한다.  
기대 결과: `pending`과 `queued`에 `<SENDER_ADDR>`의 nonce `0` tx가 없다.

**E. 단위 테스트 확인**

1. 최소 가스비 검증 테스트를 실행한다.

```bash
go test ./core/txpool/legacypool/... -run TestMinimumGasFeeValidation -v
```
2. 출력에 아래 서브테스트가 통과하는지 확인한다.

```text
reject legacy tx under minimum gas price
reject dynamic fee tx under minimum gas fee
```
3. AccessListTx는 저장소에 전용 테스트가 없는지 확인한다.  
기대 결과: 현재 `TestMinimumGasFeeValidation`에는 AccessListTx 전용 서브테스트가 없다.

**검증 포인트**

- LegacyTx는 `GasPrice`가 `belowMinFee`이면 거부되어야 한다.
- AccessListTx도 `GasPrice`가 `belowMinFee`이면 동일하게 거부되어야 한다.
- DynamicFeeTx는 `GasTipCap >= InitialGasTip`를 만족해도 `GasFeeCap < minFee`이면 거부되어야 한다.
- 세 타입 모두 에러 메시지는 `underpriced transaction: gas fee cap 47599999999999, minimum needed 47600000000000` 형식을 따라야 한다.
- 세 타입 모두 txpool `pending`과 `queued`에 남지 않아야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-1-3-04`
- B: `TC-1-3-05`
- C: `TC-1-3-06`
- D: `TC-1-3-04`, `TC-1-3-05`, `TC-1-3-06`
- E: `TC-1-3-04`, `TC-1-3-06`

---

### 4.4 체인 설정 초기화 오류 수정

---

#### TS-4-1-01: 체인 설정 적용 후 엔진 초기화 검증

**목적**

- 노드가 엔진을 생성하기 전에 체인 설정을 먼저 로드하고, 그 설정값으로 WBFT 엔진과  
하드포크 정보가 올바르게 초기화되는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동

**실행 단계 **

**A. StableNet 체인 설정으로 WBFT 엔진 정상 생성**

1. StableNet 체인 정상 동작 후 `istanbul_getValidators` RPC 응답 확인
    1. Anzeon 미활성화 노드라면 `ErrIsNotWBFTBlock` 에러 반환 

**B. Boho 하드포크 값의 체인 설정 반영 확인**

1. StableNet 체인 정상 동작 후 `eth_getCode` 전/후 값을 비교
    1. `eth_getCode(0x1003, block 99) → v1 바이트코드 (또는 genesis 직후 배포된 v1)`
    2. `eth_getCode(0x1003, block 100) → v2 바이트코드 (포크 처리로 덮어씀)`

**검증 포인트**

- Anzeon이 활성화된 StableNet 체인 설정으로 초기화할 때 생성 엔진은 WBFT 엔진이어야 한다.
- 엔진 생성 전에 로드된 체인 설정이 그대로 엔진 초기화에 사용되어야 한다.
- `BohoBlock`이 지정된 체인 설정은 로드 이후에도 동일 값을 유지해야 한다.
- Boho 값이 반영된 체인 설정은 WBFT 설정 변환 및 시스템 컨트랙트 업그레이드 등록 경로에 전달되어야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-1-01`
- B: `TC-4-1-02`

---

#### TS-4-1-02: 저장된 제네시스와 입력 제네시스 불일치 감지 검증

**목적**

- DB에 이미 저장된 canonical genesis가 있을 때, 다른 해시를 만드는 genesis 설정으로 노드를 시작하면 `GenesisMismatchError`가 반환되는지 검증한다. 

**사전 조건**

- 테스트용 DB에 canonical genesis block이 이미 저장되어 있어야 함
- 저장된 genesis와 다른 해시를 생성하는 대체 genesis 준비
- 대체 genesis는 `alloc`, `ChainID`, `Nonce`, `ExtraData` 등 hash에 영향을 주는 값을 변경해 구성할 수 있어야 함

**실행 단계 **

1. 기준 genesis를 사용해 테스트 DB에 canonical genesis block을 저장한다.
2. 일부 필드를 변경한 새 genesis를 준비한다.
3. 저장된 DB를 그대로 사용한 상태에서 새 genesis를 입력으로 체인 설정 로드 경로를 실행여 노드를 시작한다.
4. GenesisMismatchError\`가 반환되고\`, 오류의 \`Stored\`와 \`New\` 값이 각각 어떤 genesis hash를 가리키는지 확인한다.

**검증 포인트**

- 체인 설정 로드가 성공하지 않고 `GenesisMismatchError`를 반환해야 한다.
- 오류의 `Stored` 값은 DB에 저장된 canonical genesis hash와 일치해야 한다.
- 오류의 `New` 값은 새로 입력한 genesis가 계산한 hash와 일치해야 한다.
- 노드는 불일치한 genesis 설정으로 체인을 계속 초기화하지 않아야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- 전체 시나리오: `TC-4-1-03`

---

### 4.5 가스 추정 오류 수정

---

#### TS-4-2: AuthorizationList 포함 EIP-7702 트랜잭션 가스 추정 검증

**목적**

- eth\_estimateGas 호출 시 authorizationList가 포함된 EIP-7702(SetCode Tx) 트랜잭션의 가스 추정값에 auth tuple 수만큼의 비용이 정확히 반영되는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동

**실행 단계**

**A. AuthorizationList 없는 기준 가스 추정 확인**

1. `authorizationList` 없이 일반 전송 형태의 `eth_estimateGas` 요청을 준비한다.

```json
{
  "jsonrpc":"2.0",
  "method":"eth_estimateGas",
  "params":[{
    "from":"0x<FROM_ADDR>",
    "to":"0x<TO_ADDR>",
    "value":"0x0"
  },"latest"],
  "id":1
}
```
2. 아래 \`cast rpc\` 명령으로 \`eth\_estimateGas\`를 호출한다.

```bash
```bash
   cast rpc eth_estimateGas \
     '{
       "from":"0x<FROM_ADDR>",
       "to":"0x<TO_ADDR>",
       "value":"0x0"
     }' \
     latest \
     --rpc-url <RPC_URL>
   ```

  기대 결과: 에러 없이 `0x5208`이 반환되어야 한다.
```

**B. AuthorizationList 1개 포함 가스 추정 확인**

1. auth tuple 1개를 포함한 `eth_estimateGas` 요청을 준비한다.

```json
{
  "jsonrpc":"2.0",
  "method":"eth_estimateGas",
  "params":[{
    "from":"0x<FROM_ADDR>",
    "to":"0x<TO_ADDR>",
    "value":"0x0",
    "authorizationList":[
      {
        "chainId":"0x<CHAIN_ID>",
        "address":"0x<DELEGATE_ADDR>",
        "nonce":"0x<SIGNER_NONCE>",
        "v":"0x<V>",
        "r":"0x<R>",
        "s":"0x<S>"
      }
    ]
  },"latest"],
  "id":2
}
```
2. 아래 \`cast rpc\` 명령으로 \`eth\_estimateGas\`를 호출한다.

```bash
 ```bash
   cast rpc eth_estimateGas \
     '{
       "from":"0x<FROM_ADDR>",
       "to":"0x<TO_ADDR>",
       "value":"0x0",
       "authorizationList":[
         {
           "chainId":"0x<CHAIN_ID>",
           "address":"0x<DELEGATE_ADDR>",
           "nonce":"0x<SIGNER_NONCE>",
           "v":"0x<V>",
           "r":"0x<R>",
           "s":"0x<S>"
         }
       ]
     }' \
     latest \
     --rpc-url <RPC_URL>
   ```
   반환된 값을 확인한다.
   기대 결과: 에러 없이 `0xB3B0`이 반환되어야 한다.
```
3. 기준값 대비 auth tuple 1개 비용이 추가되었는지 비교한다.  
- 단계 A의 반환값이 0x5208이고 단계 B의 반환값이 0xB3B0이어야 하며, 두 값의 차이는 25,000 가스여야 한다.

**C. AuthorizationList 2개 포함 가스 추정 확인**

1. auth tuple 2개를 포함한 `eth_estimateGas` 요청을 준비한다.

```json
{
  "jsonrpc":"2.0",
  "method":"eth_estimateGas",
  "params":[{
    "from":"0x<FROM_ADDR>",
    "to":"0x<TO_ADDR>",
    "value":"0x0",
    "authorizationList":[
      {
        "chainId":"0x<CHAIN_ID>",
        "address":"0x<DELEGATE_ADDR_1>",
        "nonce":"0x<SIGNER_NONCE_1>",
        "v":"0x<V1>",
        "r":"0x<R1>",
        "s":"0x<S1>"
      },
      {
        "chainId":"0x<CHAIN_ID>",
        "address":"0x<DELEGATE_ADDR_2>",
        "nonce":"0x<SIGNER_NONCE_2>",
        "v":"0x<V2>",
        "r":"0x<R2>",
        "s":"0x<S2>"
      }
    ]
  },"latest"],
  "id":3
}
```
2. 아래 \`cast rpc\` 명령으로 \`eth\_estimateGas\`를 호출한다.

```bash
```bash
   cast rpc eth_estimateGas \
     '{
       "from":"0x<FROM_ADDR>",
       "to":"0x<TO_ADDR>",
       "value":"0x0",
       "authorizationList":[
         {
           "chainId":"0x<CHAIN_ID>",
           "address":"0x<DELEGATE_ADDR_1>",
           "nonce":"0x<SIGNER_NONCE_1>",
           "v":"0x<V1>",
           "r":"0x<R1>",
           "s":"0x<S1>"
         },
         {
           "chainId":"0x<CHAIN_ID>",
           "address":"0x<DELEGATE_ADDR_2>",
           "nonce":"0x<SIGNER_NONCE_2>",
           "v":"0x<V2>",
           "r":"0x<R2>",
           "s":"0x<S2>"
         }
       ]
     }' \
     latest \
     --rpc-url <RPC_URL>
   ```
   반환된 값을 확인한다.
   기대 결과: 에러 없이 `0x115C8`이 반환되어야 한다.
```
3. auth tuple 수에 비례해 비용이 누적되는지 비교한다.  
기대 결과: 단계 A의 결과 \`0x5208\` 대비 \`50,000\` 가스가 추가되어야 하며, 일반화하면 auth tuple이 \`N\`개일 때 추정값은 \`21,000 + 25,000 \* N\`이어야 한다.

**검증 포인트**

- AuthorizationList가 없는 일반 요청은 기본 전송 가스 `21,000`만 반환해야 한다.
- auth tuple 1개가 포함되면 `25,000` 가스가 추가된 `46,000`이 반환되어야 한다.
- auth tuple 2개가 포함되면 `50,000` 가스가 추가된 `71,000`이 반환되어야 한다.
- `authorizationList` 포함 여부에 따라 추정값 차이가 명확하게 드러나야 한다.
- 수정 전 버그 상태처럼 AuthorizationList가 무시되어 `21,000`만 반환되면 실패로 판단해야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-2-02`
- B: `TC-4-2-01`
- C: `TC-4-2-03`
- D: `TC-4-2-01`, `TC-4-2-03`

---

### 4.6. 설정 문자열 처리 수정

---

#### TS-4-3-01: 쉼표 구분 주소 목록 파싱 및 정규화 검증

**목적**

- 시스템 컨트랙트 초기화에 사용하는 쉼표 구분 설정 문자열이 공백 유무나 빈 항목 존재 여부와 관계없이 동일한 방식으로 정규화되는지 검증한다. 

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동
- 비교용 주소 문자열은 아래와 같이 완전한 20바이트 주소 형식으로 준비(``A / B / C / D는 서로 다른 `members` 문자열``)
    - `A = 0x0000000000000000000000000000000000000AAA`
    - `B = 0x0000000000000000000000000000000000000BBB`
    - `C = 0x0000000000000000000000000000000000000CCC`

**실행 단계**

**A. 공백 없는 쉼표 구분 목록**

1. `authorizedAccounts`(또는 동일 패턴 파라미터)에 아래 한 줄을 넣는다 (A, B, C는 위 표의 풀 주소).
2. 

   `A,B,C` → 예: `0x...0AAA,0x...0BBB,0x...0CCC`
3. `gstable init` 또는 동일한 system contract 초기화 경로를 실행한다.
4. 아래 명령으로 초기화된 멤버 목록을 조회한다.

```bash
cast call $GC 'isAuthorizedAccount(address)(bool)' $A --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $B --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $C --rpc-url $RPC   # true
cast call $GC 'getAuthorizedAccountCount()(uint256)' --rpc-url $RPC    # 3
```

**B. 공백 포함 쉼표 구분 목록**

1. `authorizedAccounts` 등에 `A, B, C` 형태로 넣는다 (쉼표 뒤 공백 포함).

   예: `0x...0AAA, 0x...0BBB, 0x...0CCC`
2. 
3. `gstable init` 또는 동일한 system contract 초기화 경로를 실행한다.
4. 아래 명령으로 초기화된 멤버 목록을 조회한다.

```bash
cast call $GC 'isAuthorizedAccount(address)(bool)' $A --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $B --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $C --rpc-url $RPC   # true
cast call $GC 'getAuthorizedAccountCount()(uint256)' --rpc-url $RPC    # 3
```
5. 단계 A의 결과와 비교한다.
6. 기대 결과: 단계 A와 완전히 동일한 순서와 값의 목록이어야 한다.

**C. 항목 앞뒤 공백 포함 목록 **

1. `authorizedAccounts`(또는 동일 패턴 파라미터)에 `0x...0AAA , 0x...0BBB`를 넣는다 (양끝과 쉼표 주변에 공백 포함).
2. `gstable init` 또는 동일한 system contract 초기화 경로를 실행한다.
3. 아래 명령으로 초기화된 멤버 목록을 조회한다.
4. 

```bash
cast call $GC 'isAuthorizedAccount(address)(bool)' $A --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $B --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $C --rpc-url $RPC   # false
cast call $GC 'getAuthorizedAccountCount()(uint256)' --rpc-url $RPC    # 2
```
5. 기대 결과: 반환된 멤버 목록은 길이 **2**이어야 하며, 각 항목은 `A`, `B`여야 한다.

**D. 빈 항목 포함 목록 **

1. `authorizedAccounts`(또는 동일 패턴 파라미터)에 `0x...0AAA,,0x...0BBB`를 넣는다 (연속 쉼표로 빈 항목 포함).
2. `gstable init` 또는 동일한 system contract 초기화 경로를 실행한다.
3. 아래 명령으로 초기화된 멤버 목록을 조회한다.

```bash
cast call $GC 'isAuthorizedAccount(address)(bool)' $A --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $B --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $C --rpc-url $RPC   # false
cast call $GC 'getAuthorizedAccountCount()(uint256)' --rpc-url $RPC    # 2
```
4. 기대 결과: 반환된 멤버 목록은 길이 **2**이어야 하며, 각 항목은 `A`, `B`여야 한다.

**E. 단일 주소 **

1. `authorizedAccounts`(또는 동일 패턴 파라미터)에 풀 주소 하나만 넣는다 (쉼표 없음).
2. `gstable init` 또는 동일한 system contract 초기화 경로를 실행한다.
3. 아래 명령으로 초기화된 멤버 목록을 조회한다.

```bash
cast call $GC 'isAuthorizedAccount(address)(bool)' $A --rpc-url $RPC   # true
cast call $GC 'isAuthorizedAccount(address)(bool)' $B --rpc-url $RPC   # false
cast call $GC 'isAuthorizedAccount(address)(bool)' $C --rpc-url $RPC   # false
cast call $GC 'getAuthorizedAccountCount()(uint256)' --rpc-url $RPC    # 1
```
4. 기대 결과: 반환된 멤버 목록은 길이 **1**이어야 하며, 항목은 `A`여야 한다.

**F. 빈 문자열 **

1. `authorizedAccounts`(또는 동일 패턴 파라미터)를 비우거나 빈 문자열 `""`을 넣는다.
2. `gstable init` 또는 동일한 system contract 초기화 경로를 실행한다.
3. 아래 명령으로 초기화된 멤버 목록을 조회한다.

```bash
cast call $GC 'isAuthorizedAccount(address)(bool)' $A --rpc-url $RPC   # false
cast call $GC 'isAuthorizedAccount(address)(bool)' $B --rpc-url $RPC   # false
cast call $GC 'isAuthorizedAccount(address)(bool)' $C --rpc-url $RPC   # false
cast call $GC 'getAuthorizedAccountCount()(uint256)' --rpc-url $RPC    # 0
```
4. 기대 결과: 반환된 멤버 목록은 길이 **0**의 빈 슬라이스여야 한다.

**검증 포인트**

- `A,B,C`는 3개 항목으로 파싱되어야 한다.
- `A, B, C`는 공백 제거 후 `A,B,C`와 동일한 3개 항목 결과를 반환해야 한다.
- `A , B`는 공백 제거 후 2개 항목 결과를 반환해야 한다.
- `A,,B`는 빈 항목을 무시하고 `["A", "B"]`를 반환해야 한다.
- `A`는 1개 항목 결과를 반환해야 한다.
- `""`는 빈 슬라이스를 반환해야 한다.
- 같은 주소 집합을 공백만 다르게 입력한 경우 정규화 결과는 동일한 순서와 값의 목록이어야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-3-01`
- B: `TC-4-3-02`
- C: `TC-4-3-03`
- D: `TC-4-3-04`
- E: `TC-4-3-05`
- F: `TC-4-3-06`

---

### 4.7. 동일 블록 복수 하드포크 누락 버그 수정

---

#### TS-4-4: 동일 블록에 복수 하드포크 적용 시 누락 버그 수정 검증

**목적**

- 같은 블록 높이에 두 개 이상의 하드포크가 등록된 경우에도 첫 번째 하드포크만 적용되고 나머지가 누락되지 않고, 제네시스 경로와 런타임 경로 모두에서 모든 업그레이드가 순서대로 병합 적용되는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  bohoblock , validator )
- 테스트넷 기동

**실행 단계**

**A. 블록 0에 Anzeon + Boho 하드포크가 함께 반영되는지 확인**

1. `AnzeonBlock=0`, `BohoBlock=0`이 설정된 genesis로 체인을 초기화한다.

```bash
gstable init --datadir <DATADIR> <GENESIS_BOHO_0>
```
2. 해당 datadir로 노드를 기동한다.
3. 아래 명령으로 블록 0의 `GovMinter` 코드를 조회한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block 0
```
4. 아래 명령으로 블록 0의 `GovValidator`, `GovCouncil` 코드도 조회한다.

```bash
cast code <GOVVALIDATOR_ADDR> --rpc-url <RPC_URL> --block 0
cast code <GOVCOUNCIL_ADDR> --rpc-url <RPC_URL> --block 0
```
5. 반환된 코드를 확인한다.  
기대 결과: 블록 0에서 `GovMinter` 코드는 비어 있지 않아야 하며 v2 바이트코드여야 한다. `GovValidator`와 `GovCouncil` 코드도 비어 있지 않아야 한다.

**B. 동일 블록 N에 두 하드포크 결과가 함께 반영되는지 확인**

1. `AnzeonBlock=<BOHO_BLOCK>`, `BohoBlock=<BOHO_BLOCK>` 또는 동일 블록에 두 업그레이드가 걸리도록 설정한 genesis로 체인을 초기화하고 노드를 기동한다.
2. 포크 직전 블록을 기준으로 `GovMinter`와 `GovCouncil` 코드를 조회한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <BOHO_BLOCK_MINUS_1>
cast code <GOVCOUNCIL_ADDR> --rpc-url <RPC_URL> --block <BOHO_BLOCK_MINUS_1>
```
3. 체인이 포크 블록을 지난 뒤 같은 주소들의 코드를 다시 조회한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <BOHO_BLOCK_PLUS_1>
cast code <GOVCOUNCIL_ADDR> --rpc-url <RPC_URL> --block <BOHO_BLOCK_PLUS_1>
```
4. 포크 전후 결과를 비교한다.  
기대 결과: 포크 전에는 두 주소의 코드가 비어 있거나 포크 이전 상태여야 하고, 포크 후에는 `GovMinter`와 `GovCouncil` 코드가 모두 존재해야 한다. 

**C. BohoBlock=0일 때 GovMinter Code만 바뀌고 Balance·Storage는 보존되는지 확인**

1. `AnzeonBlock=0`, `BohoBlock=0`이 설정된 genesis로 체인을 초기화하고 노드를 기동한다.
2. 아래 명령으로 블록 0의 `GovMinter` 코드와 balance를 조회한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block 0
cast balance <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block 0
```
3. 보존 확인이 필요한 storage slot이 있으면 아래 명령으로 함께 조회한다.

```bash
cast rpc eth_getStorageAt <GOVMINTER_ADDR> <SLOT_KEY> 0x0 --rpc-url <RPC_URL>
```
4. 조회 결과를 확인한다.  
기대 결과: `GovMinter` 코드는 v2여야 한다. balance는 Anzeon baseline 값과 동일해야 하며, baseline에서 초기화된 storage slot 값도 유지되어야 한다.

**D. BohoBlock=N(\>0)일 때 제네시스에는 Boho가 반영되지 않는지 확인**

1. `AnzeonBlock=0`, `BohoBlock=<BOHO_BLOCK>`이 설정된 genesis로 체인을 초기화하고 노드를 기동한다.

```bash
gstable init --datadir <DATADIR> <GENESIS_BOHO_N>
```
2. 블록 0 또는 포크 직전 블록에서 `GovMinter` 코드를 조회한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block 0
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <BOHO_BLOCK_MINUS_1>
```
3. 포크 통과 후 같은 주소의 코드를 다시 조회한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <BOHO_BLOCK_PLUS_1>
```
4. 포크 전후 결과를 비교한다.  
기대 결과: 블록 0과 포크 직전 블록에서는 `GovMinter`가 v1 코드여야 한다. 포크 이후에는 v2 코드로 바뀌어야 한다.

**검증 포인트**

- `BohoBlock=0`일 때는 블록 0부터 `GovMinter` 코드가 v2여야 한다.
- `BohoBlock=0`일 때도 `GovValidator`, `GovCouncil` 같은 Anzeon baseline 컨트랙트는 함께 존재해야 한다.
- 동일 블록에 여러 하드포크가 걸린 경우 포크 이후 상태에서 관련 컨트랙트 코드가 모두 존재해야 한다.
- `BohoBlock=0`일 때는 `GovMinter` 코드만 바뀌고 balance와 기존 storage는 유지되어야 한다.
- `BohoBlock=N(>0)`일 때는 제네시스와 포크 전 블록에서 `GovMinter`가 v1이어야 하고, 포크 후에만 v2가 되어야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-4-01`
- B: `TC-4-4-02`
- C: `TC-4-4-03`
- D: `TC-4-4-04`

---

### 4.8. Genesis 인증 계정·블랙리스트 상태 불일치 수정

---

#### TS-4-5-01: alloc.Extra 와 GovCouncil params 동기화 검증

**목적**

- genesis 초기화 시 `alloc.Extra`와 GovCouncil params에 정의된 authorized/blacklist 상태가 합집합으로 병합되고, 최종 상태가 `alloc.Extra`와 GovCouncil storage 양쪽에 일관되게 반영되는지 검증한다.

**사전조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예: alloc에 extra , blacklist , authorizedAccounts )
- 테스트넷 기동

**실행 단계**

**A. alloc.Extra 에만 있는 상태가 GovCouncil 에 반영되는지 확인**

1. `alloc.Extra`에만 blacklist 또는 authorized 비트가 설정된 주소를 포함한 genesis를 준비한다.
2. `gstable init`을 실행하고 노드를 기동한다.

```bash
gstable init --datadir <DATADIR> <GENESIS_PATH>
```
3. 아래 명령으로 GovCouncil 상태를 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "isAuthorizedAccount(address)(bool)" <ADDR_B> --rpc-url <RPC_URL>
```
4. 기대 결과: `alloc.Extra`에만 있던 상태가 GovCouncil query 결과에도 반영되어야 한다.

**B. GovCouncil params 에만 있는 상태가 alloc.Extra 에 반영되는지 확인**

1. GovCouncil params의 `blacklist` 또는 `authorizedAccounts`에만 주소를 지정한 genesis를 준비한다.
2. `gstable init`을 실행하고 노드를 기동한다.
3. 아래 명령으로 GovCouncil 상태를 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "isAuthorizedAccount(address)(bool)" <ADDR_B> --rpc-url <RPC_URL>
```
4. 블록 0 상태 덤프에서 해당 계정이 생성되었거나 기존 계정의 `extra` 비트가 갱신되었는지 확인한다.

```bash
gstable dump 0 --datadir <DATADIR> --iterative=false
```
5. 기대 결과: params에만 있던 주소가 GovCouncil query 결과에 반영되어야 하며, 블록 0 상태에도 대응하는 `extra` 비트가 존재해야 한다.

**C. alloc.Extra 와 params 의 합집합 병합을 확인**

1. 일부 주소는 `alloc.Extra`에만, 일부 주소는 GovCouncil params에만 존재하도록 genesis를 준비한다.
2. `gstable init`을 실행하고 노드를 기동한다.
3. 아래 명령으로 blacklist/authorized 여부와 count를 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_B> --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "getBlacklistCount()(uint256)" --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "getAuthorizedAccountCount()(uint256)" --rpc-url <RPC_URL>
```
4. 기대 결과: 최종 blacklist/authorized 집합은 두 소스의 합집합이어야 하며, count도 합집합 기준이어야 한다.

**D. 중복 주소가 1회만 반영되는지 확인**

1. 동일 주소를 `alloc.Extra`와 GovCouncil params 양쪽에 모두 넣은 genesis를 준비한다.
2. `gstable init`을 실행하고 노드를 기동한다.
3. 아래 명령으로 count를 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "getBlacklistCount()(uint256)" --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "getAuthorizedAccountCount()(uint256)" --rpc-url <RPC_URL>
```
4. 기대 결과: 같은 주소가 두 소스에 모두 있어도 각 목록과 count에는 1회만 반영되어야 한다.

**E. 같은 주소가 blacklist 와 authorized 를 동시에 가질 수 있는지 확인**

1. 같은 주소에 대해 `alloc.Extra`에는 blacklist 비트, params에는 `authorizedAccounts`를 넣은 genesis를 준비한다.
2. `gstable init`을 실행하고 노드를 기동한다.
3. 아래 명령으로 두 상태를 각각 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "isAuthorizedAccount(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
```
4. 상태 덤프에서 해당 계정의 `extra` 값도 확인한다.

```bash
gstable dump 0 --datadir <DATADIR> --iterative=false
```
5. 기대 결과: 같은 주소에 대해 blacklist와 authorized 상태가 모두 유지되어야 한다.

**F. 무관 alloc 엔트리가 보존되는지 확인**

1. 동기화 대상과 무관한 계정에 balance를 둔 genesis를 준비한다.
2. 별도 주소에 대해서만 blacklist 또는 authorized 동기화가 일어나도록 설정한다.
3. `gstable init`을 실행한다.
4. 블록 0 상태 덤프에서 무관 계정의 balance와 `extra` 값을 확인한다.

```bash
gstable dump 0 --datadir <DATADIR> --iterative=false
```
5. 기대 결과: 동기화 대상이 아닌 alloc 엔트리의 balance와 기존 정보는 변경되지 않아야 한다.

**검증 포인트**

- `alloc.Extra`에만 존재하던 상태는 GovCouncil query 결과에도 반영되어야 한다.
- GovCouncil params에만 존재하던 상태는 블록 0 상태의 `extra` 비트에도 반영되어야 한다.
- 최종 blacklist/authorized 집합은 두 소스의 합집합이어야 한다.
- 동일 주소가 두 소스에 모두 있어도 count는 중복 없이 계산되어야 한다.
- 같은 주소가 blacklist와 authorized를 동시에 가질 수 있어야 한다.
- 동기화 대상이 아닌 alloc 엔트리는 보존되어야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-5-01`, `TC-4-5-02`
- B: `TC-4-5-03`, `TC-4-5-04`
- C: `TC-4-5-05`
- D: `TC-4-5-06`
- E: `TC-4-5-07`
- F: `TC-4-5-08`

---

#### TS-4-5-02: 정의되지 않은 Account.Extra 비트 거부 검증

**목적**

- genesis alloc의 `Account.Extra`에 Authorized/Blacklisted 외 정의되지 않은 비트가 설정된 경우, genesis 초기화가 오류로 중단되는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예:  잘못된 \`Extra\` 비트 )
- 테스트넷 기동

**실행 단계**

1. alloc의 특정 주소에 정의되지 않은 `Extra` 비트를 설정한 genesis를 준비한다.
    1. 

```
``json
"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": {
  "balance": "0x0",
  "extra": 1
}
```
```
2. 아래 명령으로 체인 초기화를 시도한다.

```bash
gstable init --datadir <DATADIR> <GENESIS_INVALID_EXTRA>
```
3. 명령의 종료 코드와 stderr/stdout의 오류 메시지를 확인한다.  
기대 결과: 초기화는 성공하면 안 되며, 오류 메시지에 `invalid account extra`와 `unknown bits set in account extra`가 포함되어야 한다.

**검증 포인트**

- 정의되지 않은 `Extra` 비트가 있는 genesis는 초기화에 성공하면 안 된다.
- 오류 메시지는 `invalid account extra`와 `unknown bits set in account extra`를 포함해야 한다.
- 잘못된 genesis로 체인 초기화가 계속 진행되면 안 된다.

**실행 단계와 관련 테스트케이스 매핑**

- 전체 시나리오: `TC-4-5-09`

---

#### TS-4-5-03: decodePrealloc Extra 복원 검증

**목적**

- decodePrealloc 경로: mkalloc이 만든 RLP prealloc을 디코드할 때 Extra(Authorized / Blacklisted / 둘 다)가 손실 없이 types.Account.Extra로 복원되는지 검증한다.
- genesis에 넣은 `alloc.extra`가 **실제 노드가 쓰는 상태**와 **GovCouncil 컨트랙트 저장소**에 기대대로 반영되는지 확인한다

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.
    - **시스템 컨트랙트 주소와 안 겹치는** 테스트 주소 하나에만 "extra" 필드 추가
- 테스트넷 기동

**실행 단계**

1. Anzeon이 포함된 \*\*완전한 `genesis.json`\*\*을 복사한다.
2. 시스템 컨트랙트 주소와 겹치지 않는 테스트 주소 `<ADDR_A>`의 `alloc`에 `balance`와 함께 `extra`를 넣는다.
    - **A:** Authorized만
    - **B:** Blacklisted만
    - **C:** 두 비트 모두 (`0xc000000000000000` 등)
3. `gstable init --datadir <DATADIR> genesis.json` 후 노드 기동.
4. `cast call`로 GovCouncil 조회, 필요 시 `gstable dump 0`으로 계정 `extra` 확인.

**A. Authorized Extra 복원 확인**

```bash
cast call <GOVCOUNCIL_ADDR> "isAuthorizedAccount(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
```

1. 기대 결과: 해당 주소는 authorized 상태로 반영되어야 한다.

**B. Blacklisted Extra 복원 확인**

1. 아래 명령으로 해당 주소의 blacklist 상태를 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
```
2. 기대 결과: 해당 주소는 blacklisted 상태로 반영되어야 한다.

**C. Authorized | Blacklisted 조합 복원 확인**

1. `gstable init`으로 체인을 초기화하고 노드를 기동한다.
2. 아래 명령으로 두 상태를 각각 조회한다.

```bash
cast call <GOVCOUNCIL_ADDR> "isBlacklisted(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "isAuthorizedAccount(address)(bool)" <ADDR_A> --rpc-url <RPC_URL>
```
3. 기대 결과: 해당 주소는 blacklist와 authorized 상태를 모두 가져야 한다.

**검증 포인트**

- Authorized 입력은 authorized 상태로 복원되어야 한다.
- Blacklisted 입력은 blacklisted 상태로 복원되어야 한다.
- 조합 입력은 두 상태가 모두 복원되어야 한다.
- GovCouncil 상태 반영은 `decodePrealloc` 단독 책임이 아니라 이후 genesis 초기화 경로까지 포함한 end-to-end 결과로 확인해야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-5-10`
- B: `TC-4-5-11`
- C: `TC-4-5-12`

---

### 4.9 Snap Sync EffectiveGasPrice 조회 오류 수정

---

#### TS-4-6: Snap Sync 환경에서 EffectiveGasPrice 조회 오류 수정 검증

**목적**

- Snap Sync로 동기화된 노드에서 receipt의 `EffectiveGasPrice`가 비어 있어도, Anzeon 규칙에 따라 올바른 값으로 조회되는지 검증한다.

**사전 조건**

- genesis-standard.json 에 필요한 부분을 수정한다.
    - govCouncil.params.authorizedAccounts 추가
- 테스트넷 기동
- 인증 계정과 일반 계정 각각에 대해 트랜잭션을 전송할 수 있어야 한다.

**실행 단계**

**A. authorizedAccounts 에 등록된 계정 tx의 EffectiveGasPrice 재계산 확인**

1. GovCouncil의 `authorizedAccounts`에 등록된 계정으로 아래 명령을 사용해 DynamicFeeTx를 전송하고 트랜잭션 해시를 확보한다.

```bash
cast send <TO_ADDR> \
  --from <AUTHORIZED_FROM> \
  --value 0 \
  --gas-limit 21000 \
  --priority-gas-price <AUTHORIZED_GAS_TIP_CAP> \
  --max-fee-per-gas <AUTHORIZED_GAS_FEE_CAP> \
  --rpc-url <PRODUCER_RPC_URL>
```
2. Snap Sync 노드에서 receipt를 조회한다.

```bash
cast rpc eth_getTransactionReceipt <TX_HASH_AUTH> --rpc-url <SNAP_RPC_URL>
```
3. 같은 트랜잭션이 포함된 블록의 정보를 조회한다.

```bash
cast rpc eth_getBlockByNumber <BLOCK_NUMBER_HEX> false --rpc-url <SNAP_RPC_URL>
```
4. receipt의 마지막 로그를 확인한다.  
기대 결과: 마지막 로그의 `address`는 `<ACCOUNT_MANAGER_ADDR>`여야 하고, `topics[0]`는 `<AUTHORIZED_TX_EVENT_SIG>`여야 한다.
5. receipt의 `effectiveGasPrice`를 확인한다.  
기대 결과: `effectiveGasPrice`는 비어 있지 않아야 하며, `authorizedAccounts`에 등록된 계정 규칙에 따라 원본 `GasTipCap`을 사용해 계산된 값과 일치해야 한다.

**B. authorizedAccounts 에 등록되지 않은 일반 계정 tx의 EffectiveGasPrice 재계산 확인**

1. GovCouncil의 `authorizedAccounts`에 등록되지 않은 일반 계정으로 아래 명령을 사용해 DynamicFeeTx를 전송하고 트랜잭션 해시를 확보한다.

```bash
cast send <TO_ADDR> \
  --from <NORMAL_FROM> \
  --value 0 \
  --gas-limit 21000 \
  --priority-gas-price <NORMAL_GAS_TIP_CAP> \
  --max-fee-per-gas <NORMAL_GAS_FEE_CAP> \
  --rpc-url <PRODUCER_RPC_URL>
```
2. Snap Sync 노드에서 receipt를 조회한다.

```bash
cast rpc eth_getTransactionReceipt <TX_HASH_NORMAL> --rpc-url <SNAP_RPC_URL>
```
3. 같은 트랜잭션이 포함된 블록 정보를 조회한다.

```bash
cast rpc eth_getBlockByNumber <BLOCK_NUMBER_HEX> false --rpc-url <SNAP_RPC_URL>
```
4. receipt 로그를 확인한다.  
기대 결과: 마지막 로그가 `<ACCOUNT_MANAGER_ADDR>`의 `AuthorizedTxExecuted` 이벤트가 아니어야 한다.
5. receipt의 `effectiveGasPrice`를 확인한다.  
기대 결과: `effectiveGasPrice`는 비어 있지 않아야 하며, `authorizedAccounts`에 등록되지 않은 일반 계정 규칙에 따라 `headerGasTip` 기반으로 계산된 값과 일치해야 한다.

**C. EffectiveGasPrice가 이미 있는 receipt는 유지되는지 확인**

1. full 노드(`<PRODUCER_RPC_URL>`)에서 특정 트랜잭션 receipt를 조회한다.

```bash
cast rpc eth_getTransactionReceipt <TX_HASH_AUTH> --rpc-url <PRODUCER_RPC_URL>
```
2. 같은 트랜잭션 receipt를 Snap Sync 노드(`<SNAP_RPC_URL>`)에서도 조회한다.

```bash
cast rpc eth_getTransactionReceipt <TX_HASH_AUTH> --rpc-url <SNAP_RPC_URL>
```
3. 두 노드의 `effectiveGasPrice`를 비교한다.  
기대 결과: 두 노드의 `effectiveGasPrice` 값은 같아야 한다.

**D. AuthorizedTxExecuted 이벤트가 마지막 로그인지 확인**

1. GovCouncil의 `authorizedAccounts`에 등록된 계정의 트랜잭션 receipt를 조회한다.

```bash
cast rpc eth_getTransactionReceipt <TX_HASH_AUTH> --rpc-url <PRODUCER_RPC_URL>
```
2. receipt의 `logs` 배열 마지막 항목을 확인한다.  
기대 결과: 마지막 로그의 `address`는 `<ACCOUNT_MANAGER_ADDR>`여야 하고, `topics[0]`는 `<AUTHORIZED_TX_EVENT_SIG>`여야 한다.
3. GovCouncil의 `authorizedAccounts`에 등록되지 않은 일반 계정의 트랜잭션 receipt도 조회한다.

```bash
cast rpc eth_getTransactionReceipt <TX_HASH_NORMAL> --rpc-url <PRODUCER_RPC_URL>
```
4. receipt의 로그를 확인한다.  
기대 결과: 일반 계정 receipt에는 `AuthorizedTxExecuted` 이벤트가 없어야 한다.

**검증 포인트**

- Snap Sync 노드에서 조회한 receipt의 `effectiveGasPrice`는 비어 있으면 안 된다.
- 인증 계정 트랜잭션은 마지막 로그가 `AuthorizedTxExecuted`여야 한다.
- 인증 계정 트랜잭션의 `effectiveGasPrice`는 원본 `GasTipCap` 기준 계산값과 일치해야 한다.
- 일반 계정 트랜잭션의 `effectiveGasPrice`는 `headerGasTip` 기준 계산값과 일치해야 한다.
- 생산 노드와 Snap Sync 노드가 같은 트랜잭션에 대해 동일한 `effectiveGasPrice`를 반환해야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-4-6-01`
- B: `TC-4-6-02`
- C: `TC-4-6-03`
- D: `TC-4-6-04`

---

### 4.10 Go 런타임 1.23.12 업그레이드

> 테스트 제외 : TC-5-1-02 , TC-5-1-03 
> 
> 네트워크 생성 불필요

---

#### TS-5-1-01: Go 1.23.12 환경 전체 바이너리 빌드 검증

**목적**

- Go 1.23.12 환경에서 `make all`이 성공하고, fresh `build/bin/` 기준으로 전체 바이너리 산출물이 정상 생성되는지 검증한다. 

**사전 조건**

- Go 1.23.12가 설치되어 있어야 한다.

**실행 단계**

1. `go version`을 실행해 Go 런타임 버전을 확인한다.
2. 저장소 루트에서 `make all`을 실행한다.
3. 빌드 완료 후 `build/bin/` 산출물 개수를 확인한다.

**검증 포인트**

- fresh `build/bin/` 기준으로 생성된 바이너리 수가 13개여야 한다.
- 빌드 과정에서 치명적인 컴파일 오류가 발생하지 않아야 한다.

관련 테스트케이스 매핑

- `TC-5-1-01`

---

### 4.11 하드포크 업그레이드 시스템 통합

> - | TC-5-2-01 \~ 3 | CollectUpgrades가 등록된 모든 하드포크 반환  → 온체인에 없음 → 단위테스트(TS-5-2-01)
> - | TC-5-2-06 | 미등록 컨트랙트 타입/버전 요청 시 에러 | 존재하지 않는 컨트랙트 버전 | `getContractCode` 에러 반환  ← evm 응답과 다른 것, 단위테스트(TS-5-2-03 의 B)


---

#### TS-5-2-01: CollectUpgrades 기반 업그레이드 레지스트리 로드 검증

**목적**

- CollectUpgrades()가 체인 설정에 정의된 시스템 컨트랙트 하드포크 업그레이드를 단일 레지스트리로 수집하고, SetConfigFromChainConfig()가 이를 WBFT 설정에 반영하는지 검증한다. 

**실행 단계(단위테스트)**

**A. CollectUpgrades 반환값 확인**

1. `Anzeon` 기본 시스템 컨트랙트와 `Boho`의 `GovMinter v2` 업그레이드가 포함된 테스트 체인 설정을 사용한다.
2. 아래 명령으로 `CollectUpgrades()` 검증 테스트를 실행한다.

```bash
go test ./params/... -run "TestCollectUpgrades_Order|TestCollectUpgrades_NilHandling" -v
```
3. 출력과 테스트 코드가 검증하는 조건을 확인한다.  
기대 결과: `CollectUpgrades()` 반환값에는 현재 코드 기준으로 `Boho` 업그레이드 1건이 포함되어야 하며, 반환된 항목의 블록 번호와 시스템 컨트랙트 버전이 체인 설정과 일치해야 한다.

**B. SetConfigFromChainConfig가 CollectUpgrades 결과를 사용하는지 확인**

1. 아래 명령으로 `SetConfigFromChainConfig()`와 런타임 업그레이드 반영 경로를 검증하는 테스트를 실행한다.

```bash
go test ./eth/ethconfig/... -run "TestSetConfigFromChainConfig_Boho|TestSetConfigFromChainConfig_NoBoho" -v
go test ./consensus/wbft/... -run "TestSetConfigFromChainConfig_CollectUpgrades|TestGetSystemContractsStateTransition_Merge" -v
```
2. 출력과 테스트 코드가 검증하는 조건을 확인한다.
3. 첫 번째 업그레이드 항목이 `Anzeon` baseline block `0`인지 확인한다.
4. 다음 항목이 `CollectUpgrades()`에서 수집한 `Boho` 업그레이드인지 확인한다.
5. fork 이전 블록에서는 추가 state transition이 없는지 확인한다.
6. fork 블록에서는 `GetSystemContractsStateTransition(wbftCfg, BohoBlock)` 결과에 `GovMinter` 주소의 코드 전이 항목이 포함되는지 확인한다.

**검증 포인트**

- `CollectUpgrades()`는 현재 코드 기준으로 `Boho`의 `SystemContracts`를 반환해야 한다.
- `SetConfigFromChainConfig()`는 수동 append 경로 대신 `chainCfg.CollectUpgrades()` 결과를 `wbftCfg.SystemContractUpgrades`에 반영해야 한다.
- `wbftCfg.SystemContractUpgrades`에는 `Anzeon` block `0` baseline과 `BohoBlock` 업그레이드가 순서대로 포함되어야 한다.
- fork 이전 블록에서는 `GetSystemContractsStateTransition()` 결과가 `nil`이어야 한다.
- fork 블록에서는 `GetSystemContractsStateTransition(wbftCfg, BohoBlock)` 결과가 `nil`이 아니어야 하며, `GovMinter` 주소에 대한 코드 전이 항목이 포함되어야 한다.

**관련 테스트케이스 매핑**

- A: `TC-5-2-01`
- A, B: `TC-5-2-02`
- B: `TC-5-2-03`

---

#### TS-5-2-02: 시스템 컨트랙트 초기화 경로와 업그레이드 경로 분리 검증

**목적**

- 시스템 컨트랙트의 최초 배포와 하드포크 업그레이드 경로가 버전 기준으로 분리되어 있는지 검증한다. 
- `v1`은 코드 배포와 상태 초기화를 함께 수행하고, `v2` 이상은 코드 교체만 수행하며 기존 Balance와 Storage를 유지하는지 검증한다.

**사전조건**

- genesis-standard.json 에 필요한 부분을 수정한다.(예: m1, m2, bohoblock , validator )
- 테스트넷 기동

**실행 단계**

**A. v1 초기화 경로 확인 **

1. **코드 존재:** GovMinter 주소에 바이트코드가 있는지 확인한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL>
```

   기대: **0이 아닌 길이**의 코드(빈 `0x` 아님).
2. **초기화된 storage가 있음을 뷰로 확인** (예시 — genesis `params`에 맞는 값이 보여야 한다).
    - **GovCouncil** (`GOVCOUNCIL_ADDR`): 블랙리스트/허용 계정이 params에 있으면 개수가 0보다 클 수 있다.

```bash
cast call <GOVCOUNCIL_ADDR> "getBlacklistCount()(uint256)" --rpc-url <RPC_URL>
cast call <GOVCOUNCIL_ADDR> "getAuthorizedAccountCount()(uint256)" --rpc-url <RPC_URL>
```
    - **GovMinter** (GovBase 상속): 멤버 스냅샷이 제네시스에 깔렸다면 예를 들어 버전 `1` 기준 멤버 수를 조회할 수 있다.

```bash
cast call <GOVMINTER_ADDR> "getMemberCount(uint256)(uint256)" 1 --rpc-url <RPC_URL>
```

      기대: genesis `members` 등과 **모순 없는** 양수(또는 설정에 맞는 값

**B. v2 이상 업그레이드 경로 확인** 

1. Anzeon에서 Minter v1+params, `BohoBlock` = **100** 인 genesis로 init 후 노드를 **블록 100까지** 진행시킨다
2. 포크 직전 스냅샷: 블록 99 기준으로 기록한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <99>
cast storage <GOVMINTER_ADDR> <SLOT> --rpc-url <RPC_URL> --block <99>
cast balance <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <99>
```
3. **포크 적용 후:** 블록 **100** (또는 그 이상 확정 헤드)에서 동일 명령을 반복한다.

```bash
cast code <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <100>
cast storage <GOVMINTER_ADDR> <SLOT> --rpc-url <RPC_URL> --block <100>
cast balance <GOVMINTER_ADDR> --rpc-url <RPC_URL> --block <100>
```

4.비교:

- **코드:** 99과 100의 `cast code` 결과가 **다름**(v1 vs v2 바이트코드).
- **Storage:** 2번에서 고른 **같은 슬롯**의 값이 99 과 **동일**해야 한다(업그레이드가 storage를 덮어쓰지 않는 정책).
- **Balance:** Minter 컨트랙트 주소의 balance가 99 과 100에서 **동일**해야 한다(일반적으로 0 유지).

**검증 포인트**

- `Version == "v1"` 이고 `Params != nil`인 경우에만 `States`가 생성되어야 한다.
- `GovCouncil v1`, `GovMinter v1`은 코드와 함께 초기화용 storage state가 생성되어야 한다.
- `GovMinter v2`는 `Params` 유무와 관계없이 코드만 배포되고 `States`는 비어 있어야 한다.
- 업그레이드 적용 시 기존 account의 Balance와 Storage는 유지되고, Code만 새 버전으로 교체되어야 한다.
- block `0` overlay에서도 `GovMinter`의 storage는 유지된 상태로 code만 `v2`로 바뀌어야 한다.

**관련 테스트케이스 매핑**

- A: `TC-5-2-04`
- B: `TC-5-2-05`

---

#### TS-5-2-03: 미등록 시스템 컨트랙트 타입 및 버전 오류 처리 검증

**목적**

- 지원하지 않는 시스템 컨트랙트 버전이 genesis에 포함된 경우 `gstable init` 또는 설정 로드 단계에서 오류가 발생하는지 검증한다(`getContractCode`호출하는 곳).  
- 미등록 컨트랙트 /타입 오류 단위 테스트로 만 검증한다.

**실행 단계(단위테스트)**

**A. 지원하지 않는 버전이 genesis 초기화에서 거부되는지 확인**

1. 예를 들어 `systemContracts.govMinter.version = "v99"`처럼 지원하지 않는 버전을 포함한 genesis를 준비한다.
2. 아래 명령으로 체인 초기화를 시도한다.

```bash
gstable init --datadir <DATADIR> <GENESIS_INVALID_VERSION>
```
3. 명령의 종료 코드와 stderr/stdout의 오류 메시지를 확인한다.  
기대 결과: 초기화는 성공하면 안 되며, 오류 메시지에 `unsupported version`이 포함되어야 한다.

**B. 미등록 컨트랙트/ 타입 오류는 단위 테스트로  확인**

1. 아래 명령으로 `getContractCode()` 오류 처리를 검증하는 테스트를 실행한다.

```bash
go test ./systemcontracts/... -run "TestGetContractCode_Invalid" -v
```
2. 출력과 테스트 코드가 검증하는 조건을 확인한다.  
기대 결과: `getContractCode(CONTRACT_GOV_MINTER, "v99")`는 `unsupported version` 오류를 반환해야 하고, `getContractCode("nonexistent", "v1")`는 `unknown contract type` 오류를 반환해야 한다.

**검증 포인트**

- 지원하지 않는 버전이 포함된 genesis는 `gstable init` 단계에서 거부되어야 한다.
- 오류 메시지에 `unsupported version`이 포함되어야 한다.
- 잘못된 버전의 시스템 컨트랙트가 포함된 genesis로 체인 초기화가 계속 진행되면 안 된다.
- `unknown contract type`은 내부 함수 수준에서만 직접 확인 가능하며, 단위 테스트에서 오류 문자열이 검증되어야 한다.

**관련 테스트케이스 매핑**

- A, B: `TC-5-2-06`

---

### 4.12  제네시스 블록 구성 방식 표준화

> - 참조 해시
>     - mainnet: `0xf192f2ba82c9265777bad7d33b7fd561430ae5e1f60f2e99c893073c81dc5b7b`
>     - testnet: `0x2bdf79b3d3cc49f9e6638ff81f3bb85065c79945a8fe4556cd0ff47bbfc02490`
> - 단일노드

---

#### TS-5-3-01: decodePrealloc 기반 제네시스 일관성 검증

**목적**

- decodePrealloc 방식으로 생성한 StableNet mainnet/testnet 제네시스가 기존 JSON 참조 파일과 alloc, 설정, 해시 기준으로 일관된지 검증한다.

**실행 단계**

**A. JSON 참조 genesis로 초기화한 체인의 블록 0 해시 확인**

1. mainnet 참조 JSON으로 체인을 초기화한다.

```bash
gstable init --datadir <DATADIR_MAINNET> core/genesis_mainnet.json
```
2. testnet 참조 JSON으로 체인을 초기화한다.

```bash
gstable init --datadir <DATADIR_TESTNET> core/genesis_testnet.json
```
3. 각 datadir에서 블록 0 상태 또는 genesis 정보를 확인한다.

```bash
gstable dump 0 --datadir <DATADIR_MAINNET> --iterative=false
gstable dump 0 --datadir <DATADIR_TESTNET> --iterative=false
```
4. 기대 결과: mainnet/testnet 제네시스 해시는 각각 `params`에 정의된 StableNet 제네시스 해시와 일치해야 한다.

**B. alloc 일관성의 간접 네트워크 확인**

1. 블록 0 상태 덤프에서 주요 계정의 balance, code, storage가 참조 JSON과 일치하는지 표본 확인한다.

```bash
gstable dump 0 --datadir <DATADIR_MAINNET> --iterative=false
gstable dump 0 --datadir <DATADIR_TESTNET> --iterative=false
```
2. 기대 결과: 블록 0 상태 덤프의 대표 계정 값은 참조 JSON의 alloc과 일치해야 한다.

**검증 포인트**

- mainnet/testnet 제네시스 해시는 `params`에 정의된 StableNet 제네시스 해시와 일치해야 한다.
- 블록 0 상태 덤프의 대표 계정 값은 참조 JSON과 일치해야 한다.
- alloc 전체의 주소, balance, code, storage 슬롯 값은 `decodePrealloc` 결과와 JSON 참조 alloc가 완전히 일치해야 한다.
- `decodePrealloc` 방식으로 전환되어도 제네시스 블록의 최종 결과는 기존 JSON 방식과 동일해야 한다.

**실행 단계와 관련 테스트케이스 매핑**

- A: `TC-5-3-01`
- B: `TC-5-3-01`
- C: `TC-5-3-01`

---

### Appendex  취약점 패치 — 단위 테스트 검증

> **운영 환경 재현 불가 이유**    
> 테스트 케이스의 섹션 2-1\~2-4의 취약점은 정상 노드 간 정상 프로토콜 흐름에서는 자연 발생하지 않는다.    
> 재현하려면 악의적으로 변조된 메시지를 전송하는 커스텀 빌드 피어가 필요하며, 단위 테스트로만 패치 효과를 검증할 수 있다.  

---

#### TS-2-1(`섹션 2-1 (315f31e7e)`)

**목적**: 잘못된 KZG 증명이 포함된 tx를 수신할 때 해당 피어를 차단하는 로직이 올바르게 동작하는지 단위 테스트로 검증한다.

**검증 영역**: 설정/버그 수정 재발 방지 검증

**사전 조건**:

- Go 1.23.12 설치된 빌드 호스트 (`go version` 으로 확인)
- 저장소 루트(`/path/to/stable-one`)에서 실행
- 운영 환경 재현 불가 이유: 정상 노드는 tx 전파 전 자체 KZG 검증을 수행하므로 잘못된 증명 tx가 네트워크에 자연 전파되지 않음. StableNet이 blob tx를 미지원하므로 해당 오류 경로가 자연 발생하지 않음
- 환경설정

```bash
export REPO_ROOT="/path/to/stable-one"
cd $REPO_ROOT

# Go 버전 확인
go version | grep -q "go1.23.12" || { echo "ERROR: Go 1.23.12 필요"; exit 1; }
echo "✓ Go 1.23.12 확인"

# 테스트 파일 존재 확인
[ -f "eth/fetcher/tx_fetcher_test.go" ] || { echo "ERROR: tx_fetcher_test.go 없음"; exit 1; }
echo "✓ 테스트 파일 존재 확인"
```

**실행 절차**:

1. KZG 증명 위반 시 피어 차단 동작을 검증하는 단위 테스트를 실행한다:
2. 

```
eth/fetcher/tx_fetcher_test.go → TestTransactionProtocolViolation
```
3. eth/fetcher 패키지 전체 테스트를 실행하여 회귀 없음을 확인한다:
4. 테스트 결과 요약 출력:

**기대결과(검증 포인트)**:

- `TestTransactionProtocolViolation` 테스트가 PASS 한다.
- `eth/fetcher` 패키지 전체 FAIL 없음 (`^FAIL` 라인 미존재).
- 잘못된 KZG 증명 tx 수신 시 피어 차단 경로가 올바르게 실행됨을 테스트로 확인.

---

#### TS-2-2(`섹션 2-2 (c37ae123a)`)

**목적**: ECIES 암호화의 invalid-curve 공격 취약점 수정이 올바르게 적용되었는지 단위 테스트로 검증한다.

**검증 영역**: 설정/버그 수정 재발 방지 검증

**사전 조건**:

- Go 1.23.12 설치된 빌드 호스트
- 저장소 루트에서 실행
- 운영 환경 재현 불가 이유: P2P 핸드셰이크는 노드 간 연결 시점에 발생하며 정상 노드는 항상 유효한 공개키를 사용함. invalid-curve 공개키(곡선 외부 점)를 전송하는 커스텀 핸드셰이크 피어 구성이 필요함
- 환경설정

```bash
export REPO_ROOT="/path/to/stable-one"
cd $REPO_ROOT

go version | grep -q "go1.23.12" || { echo "ERROR: Go 1.23.12 필요"; exit 1; }
echo "✓ Go 1.23.12 확인"

# 관련 파일 존재 확인
[ -d "crypto/ecies" ] || { echo "ERROR: crypto/ecies 디렉토리 없음"; exit 1; }
echo "✓ crypto/ecies 패키지 확인"
ls p2p/rlpx/rlpx_oracle_poc_test.go 2>/dev/null && echo "✓ rlpx oracle POC 테스트 파일 존재" || echo "INFO: rlpx_oracle_poc_test.go 없음 (선택적)"
```

**실행 절차**:

1. ECIES 암호화 패키지 단위 테스트를 실행한다:
2. P2P RLPX Oracle POC 테스트를 실행한다 
3. p2p/rlpx 패키지 전체 테스트를 실행하여 회귀 없음을 확인한다:
4. 테스트 결과 요약:

**기대결과(검증 포인트)**:

- `crypto/ecies` 패키지 전체 PASS.
- `p2p/rlpx` 패키지 전체 PASS.
- invalid-curve 공격 시나리오 테스트 PASS (oracle POC 포함, 파일 존재 시).
- FAIL 라인 없음.

---

#### TS-2-3(`섹션 2-3 (f867ab623)`)

**목적**: secp256k1 좌표 유효성 검사 강화 패치가 올바르게 적용되어 곡선 외부 좌표값 입력 시 에러를 반환하는지 단위 테스트로 검증한다.

**검증 영역**: 설정/버그 수정 재발 방지 검증

**사전 조건**:

- Go 1.23.12 설치된 빌드 호스트
- 저장소 루트에서 실행
- 운영 환경 재현 불가 이유: 정상적인 키 생성·서명 과정에서는 곡선 외부 좌표가 생성되지 않음. 조작된 좌표값을 crypto 함수에 직접 전달하는 테스트 harness가 필요함
- 환경설정

```bash
export REPO_ROOT="/path/to/stable-one"
cd $REPO_ROOT

go version | grep -q "go1.23.12" || { echo "ERROR: Go 1.23.12 필요"; exit 1; }
echo "✓ Go 1.23.12 확인"

[ -d "crypto/secp256k1" ] || { echo "ERROR: crypto/secp256k1 디렉토리 없음"; exit 1; }
echo "✓ crypto/secp256k1 패키지 확인"
```

**실행 절차**:

1. secp256k1 패키지 전체 단위 테스트를 실행한다:
2. 좌표 유효성 검사 관련 테스트만 필터링하여 확인한다:
3. crypto 패키지 전체 테스트로 회귀 없음을 확인한다:
4. 테스트 결과 요약:

**기대결과(검증 포인트)**:

- `crypto/secp256k1` 패키지 전체 PASS.
- `crypto` 패키지 전체 PASS (회귀 없음).
- 곡선 외부 좌표 입력 시 에러 반환 동작 테스트 PASS.
- FAIL 라인 없음.

---

#### TS-2-4(`섹션 2-4 (b23c5a831)`)

**목적**: P2P 메시지 DoS 취약점 수정 패치가 올바르게 적용되어 허용 크기 초과 메시지를 차단하는 로직이 동작하는지 단위 테스트로 검증한다.

**검증 영역**: 설정/버그 수정 재발 방지 검증

**사전 조건**:

- Go 1.23.12 설치된 빌드 호스트
- 저장소 루트에서 실행
- 운영 환경 재현 불가 이유: 정상 노드는 규격에 맞는 크기의 P2P 메시지만 전송하므로 자연 발생하지 않음. 허용 크기를 초과하는 메시지를 의도적으로 전송하는 커스텀 피어 노드가 필요함
- 환경설정

```bash
export REPO_ROOT="/path/to/stable-one"
cd $REPO_ROOT

go version | grep -q "go1.23.12" || { echo "ERROR: Go 1.23.12 필요"; exit 1; }
echo "✓ Go 1.23.12 확인"

# 관련 패키지 존재 확인
for pkg in "p2p/tracker" "rlp" "eth/protocols/eth" "eth/protocols/snap"; do
  [ -d "$pkg" ] || { echo "ERROR: $pkg 디렉토리 없음"; exit 1; }
  echo "✓ $pkg 패키지 확인"
done
```

**실행 절차**:

1. P2P tracker 패키지 단위 테스트를 실행한다:
2. RLP 패키지 단위 테스트를 실행한다 (메시지 크기 제한 관련):
3. `eth/protocols/eth` 패키지 단위 테스트를 실행한다:
4. Snap 프로토콜 패키지 단위 테스트를 실행한다:
5. 전체 패키지 결과를 통합 요약한다:

**기대결과(검증 포인트)**:

- `p2p/tracker` 패키지 전체 PASS.
- `rlp` 패키지 전체 PASS (메시지 크기 제한 로직 포함).
- `eth/protocols/eth` 패키지 전체 PASS.
- `eth/protocols/snap` 패키지 전체 PASS.
- FAIL 라인 없음.
- 허용 크기 초과 메시지 차단 동작 테스트 PASS.
