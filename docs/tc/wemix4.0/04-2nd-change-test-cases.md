# [WEMIX 4.0] 2nd Change Test Cases

> 출처: Confluence [[WEMIX 4.0] 2nd Change Test Cases](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2879193122) (페이지 ID 2879193122, 버전 3, 최종 수정 2026-08-05)  
> 상위 페이지: [WEMIX4.0] Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

---

# 테스트 대상 커밋 (런타임 테스트 가능 항목)

| # | PR | 커밋 | 항목 |
| --- | --- | --- | --- |
| 1 | #197 | `5ba5d62` | fix(txpool): restore cumulative affordability enforcement with fee-delegation accounting |
| 2 | #221 | `9318f96` | fix: pre-allocate AccessList slice in SetSenderTx before copying |
| 3 | #219 | `fdb2328` | fix: fix fee-delegated transaction signing in wallet and transaction API |
| 4 | #201 | `2ea9fd0` | fix: remove DecodeVanityData and return vanityData as raw hex |
| 5 | #195 | `377c719` | fix: harden istanbul\_status RPC against resource exhaustion and data integrity issues |

---

## 테스트 항목

### T-1. TxPool 누적 잔액 검증 (PR #197)

**배경**: sender/fee-payer 잔액을 각각 추적하도록 누적 초과인출 방어 로직 복원

#### 일반 tx (self-paid) 누적 검증

- \[ \] sender 잔액이 tx 1건 비용만 커버할 때, 동일 sender로 2건 이상 제출 시 초과분이 txpool에서 거부되는지 확인
- \[ \] 잔액이 충분한 sender가 복수 tx를 제출했을 때 모두 정상 수락되는지 확인 (regression)

#### fee-delegated tx — sender / fee-payer 분리 검증

- \[ \] 동일 fee-payer를 공유하는 여러 sender가 복수 tx를 제출했을 때, 누적 gas cost가 fee-payer 잔액을 초과하면 초과분이 거부되는지 확인
- \[ \] sender 잔액이 value만 커버하고 gas는 fee-payer가 부담하는 fee-delegated tx가 정상 수락되는지 확인
- \[ \] sender와 fee-payer 잔액이 모두 충분한 정상 케이스에서 fee-delegated tx가 수락되고 블록에 포함되는지 확인 (regression)

#### tx 교체 (replacement) 검증

- \[ \] pending에 있는 fee-delegated tx를 동일 nonce로 더 높은 gas price로 교체했을 때, 이전 tx 비용이 차감되고 신규 tx 비용으로 재검증되어 정상 교체되는지 확인

---

### T-2. AccessList 포함 Fee-Delegated Tx 정상 처리 (PR #221)

**배경**: SetSenderTx에서 AccessList 복사 전 슬라이스 미할당으로 서명 복원 실패하던 버그 수정

- \[ \] AccessList가 포함된 sender tx를 fee-payer가 fee-delegated tx로 조립해 제출했을 때 블록에 포함되고 `receipt.status == 1`인지 확인
- \[ \] AccessList 없는 fee-delegated tx도 정상 처리되는지 함께 확인 (regression)

---

### T-3. Fee-Delegated Tx 서명 처리 (PR #219)

**배경**: KeyStore·scwallet에서 FeeDelegateDynamicFeeTx 타입 처리 누락 수정, fee-payer 타입 불일치 가드 추가

- \[ \] `eth_sendTransaction` / `eth_signTransaction` 경로로 fee-delegated tx 발행 시 정상 서명 및 처리 확인
- \[ \] fee-payer 키로 서명 시 tx 타입 불일치가 있으면 서명이 거부되는지 확인

---

### T-4. vanityData raw hex 반환 (PR #201)

**배경**: DecodeVanityData 제거, 잘못된 vanity 데이터 디코딩 시 panic 수정

- \[ \] `istanbul_getSnapshot` 또는 블록 헤더 조회 RPC에서 extraData·vanityData가 raw hex 형식으로 정상 반환되는지 확인

---

### T-5. `istanbul_status` RPC 안정성 (PR #195)

**배경**: 블록 범위 상한 추가, epoch 기반 캐싱, 음수 블록 번호 언더플로우 수정

- \[ \] `istanbul_status` 호출 시 대량 블록 범위 요청이 상한에서 잘려 응답하는지 확인
- \[ \] `latest`, `earliest`, 음수 블록 번호 등 경계값 파라미터 입력 시 오류 없이 정상 응답하는지 확인
- \[ \] Epoch 경계 전후로 연속 호출 시 검증자 집합 캐싱이 올바르게 동작하는지 확인
