# [WEMIX4.0] Test

> 출처: Confluence [[WEMIX4.0] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2636711400) (페이지 ID 2636711400, 버전 7, 최종 수정 2026-08-19)  
> 상위 페이지: Chainbench (폴더)  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

하위 페이지

| 파일 | Confluence 페이지 |
| --- | --- |
| [01-test-scenarios.md](01-test-scenarios.md) | [WEMIX 4.0] 테스트 시나리오 |
| [02-test-result.md](02-test-result.md) | [WEMIX 4.0] 테스트 결과 |
| [03-commit-change-log.md](03-commit-change-log.md) | [WEMIX 4.0] Commit Change Log |
| [04-2nd-change-test-cases.md](04-2nd-change-test-cases.md) | [WEMIX 4.0] 2nd Change Test Cases |

---

## 테스트 케이스 현황

| 영역 | 범위 | TC 수 |
| --- | --- | --- |
| NODE — 노드 / 네트워크 | NODE-001 \~ NODE-007 | 7 |
| TX — 트랜잭션 처리 / 가스 정책 | TX-001 \~ TX-020 | 20 |
| WBFT — WBFT 합의 | WBFT-001 \~ WBFT-013 | 13 |
| GOV — 거버넌스 | GOV-001 \~ GOV-024 | 24 |
| RPC — RPC / API | RPC-001 \~ RPC-023 | 23 |
| **합계** |  | **87** |

---

## 테스트 케이스 ID 체계

```
{영역코드}-{순번}

영역코드:
  NODE  — 노드 / 네트워크
  TX    — 트랜잭션 처리 / 가스 정책
  WBFT  — WBFT 합의
  GOV   — 거버넌스 (GovStaking, GovNCP, Validator)
  RPC   — RPC / API
```

---

## 전체 테스트 케이스 요약

### NODE — 노드 / 네트워크

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| NODE-001 | 하드포크 블록 전후 비교 | go-wemix가 생성한 블록 0\~99 데이터 준비 | 블록 99까지 wpoa 형식 / 블록 100부터 WBFTExtra 형식으로 전환 |
| NODE-002 | go-wemix 데이터로 go-wbft 기동 | go-wemix 노드가 블록 0\~99 생성 후 종료된 상태 | go-wbft가 기존 데이터를 오류 없이 인식, 블록 높이 및 상태 루트 일치 |
| NODE-003 | Full Sync 동기화 | 체인 200블록+, 하드포크 전후 각 타입별 tx 포함, --syncmode full | 신규 노드가 전체 블록 검증하며 동기화 완료, 상태 루트 일치 |
| NODE-004 | Snap Sync 동기화 | 체인 200블록+ (snap pivot이 Croissant 이후 형성), 하드포크 전후 각 타입별 tx 포함, --syncmode snap | 스냅 피벗 이후 상태 동기화 완료, 잔액 조회 가능 |
| NODE-005 | 노드 장애 후 복구 | 밸리데이터 7개 운영 중 (Phase 17), 1개 강제 종료 | 나머지 6개로 합의 지속, 재기동 노드 정상 동기화 |
| NODE-006 | NCP 주소 공백 포함 파싱 | UseNCP=true, 공백 포함 NCP 주소로 기동된 노드, NCP\_1\_ADDR 설정 | 공백 제거 후 NCP 정상 등록, 노드 기동 성공 |
| NODE-007 | 빈 NCP 목록 초기화 실패 | UseNCP=true, govNCP.params.ncps="" (빈 문자열) 설정 | gwemix init 실패, NCP 미설정 오류 반환 |

### TX — 트랜잭션 처리 / 가스 정책

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| TX-001 | 일반 트랜잭션 생성/전파/포함/확정 | WBFT 네트워크 운영 중, 잔액 있는 계정 | 트랜잭션 블록 포함 및 확정, 수신자 잔액 증가 |
| TX-002 | BaseFee 미달 트랜잭션 거부 | 현재 BaseFee보다 낮은 maxFeePerGas 설정 | txpool 진입 거부 |
| TX-003 | DynamicFee Tx (type 0x2) 정상 처리 | maxFeePerGas ≥ BaseFee 설정 | type 0x2 트랜잭션 처리 성공, effectiveGasPrice 올바르게 계산 |
| TX-004 | Fee Delegation 트랜잭션 | Sender·FeePayer 각각 유효한 서명 | 가스비는 FeePayer 부담, Sender는 전송액만 차감 |
| TX-005 | 스마트 컨트랙트 배포 및 실행 | Croissant 이후, 잔액 있는 계정 | 배포 성공, 상태 변경·view 조회·revert·out-of-gas 모두 정상 동작 |
| TX-006 | Legacy Tx (type 0x0) | gasPrice 필드 사용 (maxFeePerGas 미사용) | type 0x0 처리 성공, effectiveGasPrice == gasPrice |
| TX-007 | AccessList Tx (type 0x1) | accessList 필드에 접근할 주소·슬롯 명시 | type 0x1 트랜잭션 처리 성공 |
| TX-008 | SetCode Tx (type 0x4, EIP-7702) | delegator 계정, 코드를 위임할 컨트랙트 배포 완료 | EOA 계정 코드가 위임 컨트랙트를 가리키도록 설정됨 |
| TX-009 | secp256r1 프리컴파일 유효 서명 검증 | Croissant 이후 (프리컴파일 활성화) | 유효한 P-256 서명 입력 시 0x01 반환 |
| TX-010 | Nonce 순서 보장 | 동일 계정에서 역순 nonce로 트랜잭션 제출 | nonce 오름차순으로 블록 포함 |
| TX-011 | 잔액 부족 tx 거부 | 전송 금액이 계정 잔액 초과 | 잔액 부족 오류로 txpool 거부 |
| TX-012 | Gas Limit 초과 tx 거부 | gasLimit이 블록 gas limit 초과 | txpool 진입 거부 |
| TX-013 | 트랜잭션 교체 (queued tx replacement) | nonce gap으로 queued 상태인 tx, 동일 nonce로 더 높은 gasFeeCap tx 재발행 | queued tx가 새 tx로 교체, 기존 tx 제거 |
| TX-014 | Fee Delegation Sender 서명 실패 | Sender 서명 필드 조작 | 서명 검증 실패로 txpool 거부 |
| TX-015 | Fee Delegation FeePayer 서명 실패 | FeePayer 서명 필드 조작 | 서명 검증 실패로 txpool 거부 |
| TX-016 | Fee Delegation FeePayer 잔액 부족 | FeePayer 계정 잔액이 가스비 미만 | 블록 실행 중 잔액 부족으로 트랜잭션 실패 |
| TX-017 | 컨트랙트 revert tx | revert하는 함수 호출, 배포된 컨트랙트 존재 | receipt.status == 0, 잔여 가스 환불, 상태 롤백 |
| TX-018 | 컨트랙트 out-of-gas tx | gasLimit을 실제 소비보다 부족하게 설정 | receipt.status == 0, gasUsed == gasLimit (환불 없음) |
| TX-019 | secp256r1 프리컴파일 무효 서명 검증 | 유효한 pubkey + 잘못된 r/s 값 | 0x00 반환 (검증 실패) |
| TX-020 | secp256r1 프리컴파일 잘못된 입력 길이 | 160바이트 미만 입력 | 빈 반환값 (`0x`) |

### WBFT — WBFT 합의

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| WBFT-001 | 블록 생성 및 Finalize 정상 동작 | 밸리데이터 4개 이상, Croissant 이후 | 블록마다 쿼럼 이상의 BLS 집합 서명 존재 |
| WBFT-002 | 블록 생산 주기 1초 | 블록 생성 주기 1초 설정 | 연속 블록 간 타임스탬프 차이 = 1초 |
| WBFT-003 | View Change 발생 및 라운드 체인지 후 블록 연결 검증 | 현재 Proposer 노드 종료 | 타임아웃 후 다음 Proposer가 블록 생성, 라운드 체인지 이후 블록 parentHash 연결 유지 |
| WBFT-004 | 라운드 체인지 후 블록 연결 (WBFT-003에 통합) | — | WBFT-003 참조 |
| WBFT-005 | 에폭 전환 시 밸리데이터 세트 갱신 | 에폭 길이 10 설정, 스테이킹 상태 변화 있음 | 에폭 마지막 블록에 새 Validator 정보 기록, 다음 에폭부터 적용 |
| WBFT-006 | RoundRobin Proposer 정책 | 밸리데이터 5개 (wbft\_bp\_1\~5), RoundRobin 정책 | 연속 블록에서 각 밸리데이터가 균등하게 Proposer 역할 수행 |
| WBFT-007 | 밸리데이터 1/3 미만 장애 시 합의 지속 | 밸리데이터 7개 중 wbft\_bp\_7 종료 (쿼럼 5 충족) | 나머지 6개로 블록 생성 지속, 재기동 후 동기화 복구 |
| WBFT-008 | 밸리데이터 1/3 이상 장애 시 합의 중단 | 밸리데이터 7개 중 wbft\_bp\_5·6·7 종료 (쿼럼 5 미달) | 블록 생성 중단, 재기동 후 합의 재개 |
| WBFT-009 | PrevSeal 수집 확인 | Croissant 이후 블록 2개 이상 | 다음 블록 Extra에 이전 블록의 Commit·Prepare 서명이 쿼럼 이상 수집 |
| WBFT-010 | RandaoReveal / MixDigest 검증 | Croissant 이후 블록 3개 이상 | RandaoReveal이 Proposer의 ECDSA 서명이고, MixDigest가 이전 값과 XOR 연쇄 |
| WBFT-011 | 쿼럼 검증 — 밸리데이터 3개 (전원 필요) | 밸리데이터 3개, 1개 종료 | 1개 장애만으로 쿼럼 미달 → 합의 중단 |
| WBFT-012 | 쿼럼 검증 — 밸리데이터 6개 (쿼럼 5) | 밸리데이터 6개 | 1개 장애: 합의 유지 / 2개 장애: 합의 중단 |
| WBFT-013 | 쿼럼 검증 — 밸리데이터 6개, 2개 장애 시 합의 중단 | 밸리데이터 6개 | 2개 장애 시 4/6으로 쿼럼 미달, 합의 중단 |

### GOV — 거버넌스

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| GOV-001 | Croissant 블록 거버넌스 컨트랙트 배포 | 하드포크 블록 100 설정 | 블록 99까지 컨트랙트 없음, 블록 100에서 4개 컨트랙트 배포 |
| GOV-002 | GovConfig 파라미터 초기값 검증 | 거버넌스 컨트랙트 배포 완료 | minimumStaking 등 설정값이 컨트랙트 스토리지에 올바르게 저장 |
| GOV-003 | GovStaking Staker 등록 | Operator 계정, Staker(coinbase) 계정, BLS 키페어, 최소 스테이킹 이상 잔액 | Staker ACTIVE 등록, 보상 수신 전용 GovRewardee 컨트랙트 자동 배포 |
| GOV-004 | GovStaking 언스테이킹 및 출금 | Staker 등록 완료 | 전액 언스테이킹 시 INACTIVE 전환, 7일 unbonding 후 출금 성공 |
| GOV-005 | 스테이킹 후 Validator 반영 | Staker 등록 완료 | 에폭 종료 후 다음 에폭부터 Validator 목록에 포함 |
| GOV-006 | GovNCP NCP 추가 제안 및 투표 | UseNCP=true, NCP 멤버 7명 (초기 등록), 4표 필요 | 과반수 찬성으로 신규 NCP 등록 |
| GOV-007 | GovNCP NCP 제거 (타인 제안) | NCP 추가 완료, NCP 7명 이상 | 과반수 찬성으로 대상 NCP 제거, 다음 에폭부터 Validator 후보 제외 |
| GOV-008 | GovNCP NCP 자기 탈퇴 (즉시) | NCP 2명 이상 | 본인 탈퇴는 투표 없이 즉시 처리, GovStaking 스테이킹은 유지 |
| GOV-009 | Validator 변경 프로세스 및 NCP 조건 검증 (GOV-024 통합) | TargetValidators=7, Staker A\~G(NCP, 7명)+H(non-NCP, 8위) | H가 최대 스테이킹해도 non-NCP이므로 Validator 미선정 → H NCP 등록 후 다음 에폭 Validator 선정, G 탈락 |
| GOV-010 | Stabilization Stage 검증 (staker 등록 → 해제) | GOV-003 통과 (NCP 1명), stabilizingStakersThreshold=5 | NCP 증가에 따라 Stabilizing 상태 추적, NCP 7명(≥ threshold) 등록 시 해제 및 validator 세트 갱신 |
| GOV-011 | Delegation 동작 | Staker 등록 완료, 위임할 계정 준비 | 위임 후 Staker의 총 스테이킹 증가, 72시간 unbonding 후 언위임 출금 |
| GOV-012 | 블록 리워드 누적 (엔진 → GovRewardee) | Staker 등록 완료, 블록 10개 이상 생성 | 블록 생성마다 합의 엔진이 Staker의 GovRewardee 컨트랙트로 리워드 전송 |
| GOV-013 | Operator 보상 수령 (Claim) | GovRewardee에 보상 누적, 위임자 없는 Staker | Operator(msg.sender) 잔액 증가, GovRewardee 컨트랙트 잔액 감소 |
| GOV-014 | 위임자 보상 수령 (Delegator Claim) | 위임 완료 후 블록 생성으로 보상 누적 | 위임자가 claim 호출 시 수수료 차감 후 위임 비례 보상 수령 |
| GOV-015 | 언스테이킹 실패 — minimum 미달 | 최소 스테이킹의 2배를 스테이킹한 상태 | 잔여량이 최소 스테이킹 미만이 되는 부분 언스테이킹은 revert |
| GOV-016 | 전액 언스테이킹 INACTIVE 전환 | GOV-003 통과 (ACTIVE staker 상태) | 잔여량 0이 되는 전액 언스테이킹 성공, Staker INACTIVE 전환 |
| GOV-017 | 긴급 모드 활성화 및 GovStaking 차단 | UseNCP=true, NCP 멤버 과반수 이상 | 긴급 모드 활성화 시 스테이킹·위임 등 모든 GovStaking 작업 차단 |
| GOV-018 | 긴급 모드 해제 및 GovStaking 재개 | 긴급 모드 활성화 상태 | 긴급 모드 해제 후 GovStaking 작업 정상 재개 |
| GOV-019 | Staker INACTIVE → ACTIVE 재활성화 | Staker가 전액 언스테이킹으로 INACTIVE 전환된 상태 | 최소 스테이킹 이상 재스테이킹 시 ACTIVE 전환, Validator 후보 복귀 |
| GOV-020 | 수수료율 변경 — 위임자 없음 (즉시) | 위임자 없는 Staker | 수수료 변경 요청 즉시 적용 |
| GOV-021 | 수수료율 변경 — 위임자 있음 (지연) | 위임자 있는 Staker | 수수료 변경 요청 후 수수료 변경 지연 기간 경과 후 적용 |
| GOV-022 | Claim 권한 — 타 Staker 보상 탈취 불가 | Staker A(Operator A), Staker B(Operator B) 보상 누적, 위임 없는 제3자 계정 C | Operator A가 Staker B의 claim 호출 시 B의 보상이 A에게 전달되지 않음 / 계정 C는 claim 불가 |
| GOV-023 | Credential 만료 기간 개별 검증 | OP\_A가 unstake → undelegate(VAL\_B) 순으로 동일 sender의 credential[N](staker), credential[N+1](delegator) 생성 | delegator unbonding 만료 시 `withdraw(1)` → staker credential skip, delegator credential 처리 성공; staker unbonding 만료 시 `withdraw(1)` → staker credential 처리 성공 |
| GOV-024 | ~~NCP Staker만 Validator 선정~~ (GOV-009으로 통합) | — | GOV-009에서 통합 검증 |

### RPC — RPC / API

| ID | 테스트 케이스 | 전제조건 | 기대 결과 |
| --- | --- | --- | --- |
| RPC-001 | eth\_blockNumber | 노드 운영 중 | 블록 번호가 단조 증가하며 반환 |
| RPC-002 | eth\_getBlockByNumber / eth\_getBlockByHash | Croissant 이후 블록 존재 | 블록 번호·해시 두 방식 모두 동일한 블록 반환, WBFTExtra 포함 |
| RPC-003 | istanbul\_getValidators | Croissant 이후, 에폭 전환 포함 | 블록 번호 기준 해당 에폭의 Validator 주소 목록 반환 |
| RPC-004 | istanbul\_getCommitSignersFromBlock | Croissant 이후 블록 | 블록에 Commit 서명한 Validator 목록 반환, 쿼럼 이상 |
| RPC-005 | istanbul\_getWbftExtraInfo | Croissant 이후 블록 | 일반 블록: EpochInfo 없음 / 에폭 마지막 블록: EpochInfo 존재 |
| RPC-006 | istanbul\_status | 블록 50개 이상 존재 | 조회 범위 내 Validator별 블록 서명 통계, 블록 제안 횟수, 라운드 분포 반환 |
| RPC-007 | eth\_getTransactionReceipt | 블록에 포함된 트랜잭션 존재 | 트랜잭션 결과(status, blockNumber, gasUsed 등) 포함 receipt 반환 |
| RPC-008 | wemix\_getBriocheBlockReward | Brioche 하드포크 이전·이후 블록 존재 | 이전: 원래 리워드 / 이후: 반감기 적용 리워드 반환 |
| RPC-009 | eth\_call (거버넌스 컨트랙트 읽기) | 거버넌스 컨트랙트 배포 완료 | minimumStaking 등 view 함수 올바른 값 반환, 스테이킹 후 즉시 반영 |
| RPC-010 | istanbul\_nodeAddress | 밸리데이터·비밸리데이터 노드 운영 중 | 밸리데이터 노드: Validator 목록에 포함된 주소 / 비밸리데이터: 미포함 주소 반환 |
| RPC-011 | istanbul\_isValidator | 밸리데이터·비밸리데이터 노드 각각 | 밸리데이터 노드: true / 비밸리데이터 노드: false |
| RPC-012 | eth\_getBalance | 잔액 있는 계정, 트랜잭션 전·후 | Wei 단위 정확한 잔액 반환, 트랜잭션 후 변화 즉시 반영 |
| RPC-013 | eth\_chainId | 노드 운영 중 | genesis.json에 설정된 chainId와 동일한 값 반환 |
| RPC-014 | eth\_getLogs | 이벤트를 발생시키는 컨트랙트 배포 후 이벤트 발생 | 주소·토픽 필터 조건에 맞는 이벤트 로그 반환 |
| RPC-015 | eth\_getTransactionCount | 트랜잭션을 발행한 계정 | 계정이 발행한 트랜잭션 수(nonce)와 일치하는 값 반환 |
| RPC-016 | eth\_gasPrice | Croissant 이후 | 현재 baseFee 이상의 권장 가스 가격 반환 |
| RPC-017 | eth\_feeHistory | 블록 여러 개 생성된 상태 | 최근 N블록의 baseFee 이력과 가스 사용률 배열 반환 |
| RPC-018 | txpool\_status / txpool\_content | pending tx(연속 nonce)·queued tx(nonce gap) 주입 | txpool\_status: pending·queued 건수 정확 / txpool\_content: 각 tx 상세 정보 반환 |
| RPC-019 | admin\_peers | 2개 이상 노드가 피어 연결된 상태 | 연결된 피어의 ID, 네트워크 주소, 프로토콜 정보 반환 |
| RPC-020 | eth\_subscribe (newHeads) | WebSocket 연결, 블록 생성 중 | 블록 생성마다 블록 헤더(번호, 해시, 타임스탬프 등) 이벤트 실시간 수신 |
| RPC-021 | eth\_subscribe (logs) | WebSocket 연결, 이벤트 발생 컨트랙트 배포 완료 | 지정 주소·토픽 조건에 맞는 이벤트 발생 시 즉시 수신 |
| RPC-022 | istanbul\_getWbftExtraInfo — 에폭 마지막 블록 | EpochLength=10, 블록 110 존재 | EpochInfo.Validators·Stakers 배열 비어있지 않음 |
| RPC-023 | istanbul\_isValidator — 에폭 전환 후 제외된 노드 | 밸리데이터 언스테이킹으로 다음 에폭 제외 상태 | 에폭 전환 전: true / 전환 후: false |
