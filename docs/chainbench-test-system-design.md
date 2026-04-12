# chainbench 기반 자동화 테스트 시스템 설계

> 작성일: 2026-04-12
> 대상: go-stablenet + chainbench + hardfork/regression 테스트 스펙

---

## 1. 현황 분석

### 1.1 chainbench 보유 기능

| 영역 | 보유 기능 | 비고 |
|------|-----------|------|
| 체인 라이프사이클 | init → start → stop → restart → clean | 프로파일 기반 |
| 제네시스 생성 | WBFT genesis.json 자동 생성 (템플릿 + 프로파일 치환) | 시스템 컨트랙트 5종 포함 |
| 노드 관리 | 개별 노드 start/stop/log/rpc | PID 추적, logrot 지원 |
| RPC 라이브러리 | rpc.sh, rpc_tx.sh, rpc_account.sh, rpc_block.sh, rpc_consensus.sh | bash 기반 |
| 어설션 | assert_true, assert_eq, assert_ge 등 | test lifecycle 관리 |
| 프리셋 키 | 5개 노드 고정 주소/BLS 키 | 재현성 보장 |
| 기존 테스트 | basic(7), fault(6), stress(2) = **15개** | bash 스크립트 |
| MCP 통합 | lifecycle, node, test, log 도구 | Claude Code 연동 가능 |

### 1.2 테스트 스펙 요구사항

| 스펙 | 테스트 케이스 수 | 핵심 요구사항 |
|------|-----------------|---------------|
| **hardfork-test-spec** | 70개 (5 섹션) | Boho 하드포크 활성화/비활성화 체인, 시스템 컨트랙트 v2 검증, EIP-7951, 가스 최소값 |
| **regression-test-spec** | 116개 (7 섹션 A~G) | 트랜잭션 타입별 검증, WBFT 합의, 가스 정책, Fee Delegation, 블랙리스트, 거버넌스 |

### 1.3 Gap 분석 — chainbench에 없는 것

| Gap | 필요한 기능 | 테스트 스펙 영향 |
|-----|-----------|----------------|
| **G1** | **서명된 트랜잭션 전송** (eth_sendRawTransaction) | 거의 모든 TC — 현재 send_tx는 eth_sendTransaction(언락 기반)만 지원 |
| **G2** | **컨트랙트 호출** (eth_call + ABI 인코딩) | F 섹션 전체, TC-1-1-* (GovMinter 거버넌스 호출) |
| **G3** | **이벤트 로그 파싱** (eth_getLogs + 토픽 디코딩) | BurnRefundClaimed, AuthorizedTxExecuted 등 이벤트 검증 |
| **G4** | **하드포크 블록 지정 프로파일** | TC-1-* — BohoBlock을 특정 블록으로 설정한 제네시스 필요 |
| **G5** | **Fee Delegation 트랜잭션 구성** (type 0x16) | D 섹션 전체 — 이중 서명 트랜잭션 생성 |
| **G6** | **컨트랙트 배포** | A-3 섹션 — 커스텀 컨트랙트 deploy + state 변경 검증 |
| **G7** | **EIP-7702 (SetCodeTx)** | RT-A-2-04, TC-4-2-* — authorization list 구성 |
| **G8** | **Account Extra 비트 검증** | TC-4-5-*, E 섹션 — isBlacklisted/isAuthorized 상태 확인 |

---

## 2. 실현 가능성 평가

### 2.1 결론: **구축 가능**, 단 계층적 접근 필요

chainbench의 기존 인프라(체인 라이프사이클, 프로파일, RPC, 어설션)는 충분히 강력하며, Gap은 **테스트 유틸리티 계층**을 추가하면 해결됩니다.

### 2.2 접근 전략: 3계층 구조

```
┌─────────────────────────────────────────────────┐
│  Layer 3: 테스트 스크립트                          │
│  tests/hardfork/*.sh, tests/regression/*.sh      │
│  (각 TC를 bash 스크립트로 구현)                    │
├─────────────────────────────────────────────────┤
│  Layer 2: 테스트 유틸리티 (새로 구축)               │
│  tests/lib/contract.sh  — ABI 인코딩, 컨트랙트 호출│
│  tests/lib/tx_builder.sh — 서명 트랜잭션, FD 구성  │
│  tests/lib/event.sh     — 이벤트 로그 파싱         │
│  tests/lib/chain_state.sh — 하드포크/상태 검증     │
├─────────────────────────────────────────────────┤
│  Layer 1: chainbench 기존 인프라                   │
│  rpc.sh, assert.sh, wait.sh, 프로파일, 라이프사이클│
└─────────────────────────────────────────────────┘
```

### 2.3 Gap별 해결 방안

| Gap | 해결 방안 | 난이도 | 비고 |
|-----|----------|--------|------|
| G1 | `tx_builder.sh` — go-stablenet의 `ethkey` 또는 Python `eth_account` 로 서명 후 `eth_sendRawTransaction` | 중 | 키스토어 파일 재사용 가능 |
| G2 | `contract.sh` — Python `web3` 또는 `cast` (foundry) 로 ABI 인코딩/디코딩 | 중 | 시스템 컨트랙트 ABI는 프로젝트에 포함됨 |
| G3 | `event.sh` — `eth_getLogs` + 토픽 해시 계산 (keccak256) | 저 | bash + python 조합 |
| G4 | 하드포크 프로파일 — `profiles/hardfork-boho.yaml`에 `bohoBlock` 오버라이드 | 저 | 기존 프로파일 시스템 활용 |
| G5 | `tx_builder.sh` 확장 — type 0x16 RLP 구성 + 이중 서명 | 고 | go-stablenet의 signing 로직 참조 필요 |
| G6 | `contract.sh` 확장 — 바이트코드 deploy 트랜잭션 | 중 | eth_sendRawTransaction + create TX |
| G7 | `tx_builder.sh` 확장 — EIP-7702 authorization list | 고 | 새 트랜잭션 타입 |
| G8 | `chain_state.sh` — AccountManager RPC 호출 래퍼 | 저 | eth_call로 isBlacklisted/isAuthorized 호출 |

---

## 3. 시스템 설계

### 3.1 디렉토리 구조 (chainbench 내 확장)

```
chainbench/
├── profiles/
│   ├── hardfork-boho-pre.yaml      # BohoBlock > 현재블록 (하드포크 전)
│   ├── hardfork-boho-post.yaml     # BohoBlock = 0 (하드포크 후)
│   ├── hardfork-boho-delayed.yaml  # BohoBlock = 100 (테스트넷 시뮬레이션)
│   └── regression.yaml             # 기존 유지 (모든 하드포크 활성)
├── tests/
│   ├── lib/
│   │   ├── rpc.sh                  # 기존
│   │   ├── assert.sh               # 기존
│   │   ├── contract.sh             # [신규] ABI 인코딩, eth_call, 컨트랙트 배포
│   │   ├── tx_builder.sh           # [신규] 서명 TX, Fee Delegation, EIP-7702
│   │   ├── event.sh                # [신규] 이벤트 로그 조회/파싱
│   │   ├── chain_state.sh          # [신규] 하드포크 상태, Account Extra 검증
│   │   └── system_contracts.sh     # [신규] 시스템 컨트랙트 주소/ABI 상수
│   ├── hardfork/                   # [신규] hardfork-test-spec 구현
│   │   ├── 1-1-govminter-upgrade.sh
│   │   ├── 1-2-secp256r1.sh
│   │   ├── 1-3-gas-fee-floor.sh
│   │   ├── 4-1-wbft-config.sh
│   │   ├── 4-4-simultaneous-activation.sh
│   │   ├── 4-5-account-extra-sync.sh
│   │   ├── 4-6-effective-gas-price.sh
│   │   └── 5-hardfork-management.sh
│   └── regression/                 # [신규] regression-test-spec 구현
│       ├── a1-node-sync.sh
│       ├── a2-tx-types.sh
│       ├── a3-contracts.sh
│       ├── a4-rpc-api.sh
│       ├── b-wbft-consensus.sh
│       ├── c-gas-price.sh
│       ├── d-fee-delegation.sh
│       ├── e-blacklist.sh
│       ├── f1-native-coin.sh
│       ├── f2-gov-validator.sh
│       ├── f3-gov-minter.sh
│       └── f4-gov-council.sh
```

### 3.2 프로파일 설계

**hardfork-boho-delayed.yaml** (BohoBlock=100, 하드포크 전후 테스트):
```yaml
extends: default
genesis:
  overrides:
    bohoBlock: 100
    anzeonBlock: 0
nodes:
  validators: 4
  endpoints: 1
```

**hardfork-boho-post.yaml** (BohoBlock=0, 하드포크 적용 상태):
```yaml
extends: default
genesis:
  overrides:
    bohoBlock: 0
    anzeonBlock: 0
nodes:
  validators: 4
  endpoints: 1
```

### 3.3 테스트 실행 흐름

```
[하드포크 테스트]
chainbench init --profile hardfork-boho-delayed
chainbench start
  → 블록 0~99: Boho 비활성 상태에서 TC-1-2-02 (precompile 부재 확인)
  → 블록 100 이후: Boho 활성 상태에서 TC-1-1-* (GovMinter v2 검증)
chainbench test run hardfork
chainbench stop

[회귀 테스트]
chainbench init --profile regression
chainbench start
chainbench test run regression
chainbench stop
```

### 3.4 커맨드 통합 설계

go-stablenet 프로젝트의 `.claude/commands/`에 새 커맨드를 추가하여 Claude Code에서 직접 테스트를 수행할 수 있도록 합니다:

```
.claude/commands/
├── stablenet-review-code.md      # 기존
├── stablenet-test-hardfork.md    # [신규] 하드포크 테스트 실행
└── stablenet-test-regression.md  # [신규] 회귀 테스트 실행
```

---

## 4. 구현 계획

### Phase 1: 인프라 (Layer 2 유틸리티) — 선행 필수

| 순서 | 작업 | 의존성 | 예상 파일 |
|------|------|--------|-----------|
| 1-1 | `system_contracts.sh` — 시스템 컨트랙트 주소/함수 시그니처 상수 | 없음 | 1 파일 |
| 1-2 | `contract.sh` — ABI 인코딩(keccak + 파라미터 패딩), eth_call 래퍼 | 1-1 | 1 파일 |
| 1-3 | `event.sh` — eth_getLogs, 토픽 해시 계산, 이벤트 디코딩 | 1-2 | 1 파일 |
| 1-4 | `chain_state.sh` — 하드포크 활성화 확인, Account Extra 검증 | 1-2 | 1 파일 |
| 1-5 | `tx_builder.sh` — 서명 트랜잭션 생성 (ethkey 또는 cast 활용) | 없음 | 1 파일 |
| 1-6 | 하드포크 프로파일 3종 생성 | 없음 | 3 파일 |

### Phase 2: 하드포크 테스트 (우선순위 1)

| 순서 | 테스트 그룹 | TC 수 | 의존 유틸 |
|------|-----------|-------|-----------|
| 2-1 | TC-4-1 WBFT Config | 3 | chain_state |
| 2-2 | TC-4-4 동시 활성화 | 4 | chain_state, 프로파일 |
| 2-3 | TC-1-3 가스 최소값 | 6 | tx_builder |
| 2-4 | TC-4-5 Account Extra 동기화 | 12 | chain_state, contract |
| 2-5 | TC-4-6 EffectiveGasPrice | 4 | tx_builder, event |
| 2-6 | TC-1-2 secp256r1 | 6 | contract (precompile 호출) |
| 2-7 | TC-1-1 GovMinter 업그레이드 | 12 | contract, event, tx_builder |
| 2-8 | TC-5 빌드/릴리즈 | 7 | chain_state |

### Phase 3: 회귀 테스트 (우선순위 2)

| 순서 | 테스트 그룹 | TC 수 | 의존 유틸 |
|------|-----------|-------|-----------|
| 3-1 | A-1 노드/동기화 | 7 | 기존 chainbench |
| 3-2 | A-4 RPC API | 7 | 기존 rpc.sh |
| 3-3 | B WBFT 합의 | 12 | rpc_consensus |
| 3-4 | C 가스 가격 | 7 | tx_builder, chain_state |
| 3-5 | A-2 트랜잭션 타입 | 10 | tx_builder |
| 3-6 | E 블랙리스트 | 9 | contract, chain_state |
| 3-7 | A-3 컨트랙트 | 7 | contract, tx_builder |
| 3-8 | F 시스템 컨트랙트 거버넌스 | 25+ | contract, event, tx_builder |
| 3-9 | D Fee Delegation | 4 | tx_builder (type 0x16) |

### Phase 4: CI 통합 및 커맨드

| 순서 | 작업 |
|------|------|
| 4-1 | Claude Code 커맨드 2종 작성 (hardfork, regression) |
| 4-2 | MCP 도구 확장 (hardfork/regression 테스트 실행) |
| 4-3 | GitHub Actions 워크플로우 통합 (선택) |

---

## 5. 기술 의사결정 포인트

### 5.1 트랜잭션 서명 도구 선택

| 옵션 | 장점 | 단점 |
|------|------|------|
| **A. cast (foundry)** | ABI 인코딩, TX 서명, 이벤트 디코딩 올인원 | 외부 의존성, Fee Delegation 미지원 |
| **B. ethkey (go-stablenet 내장)** | 의존성 없음 | ABI 인코딩 불가, 기능 제한적 |
| **C. Python web3.py** | 유연, ABI/서명/이벤트 완전 지원 | Python 의존성 |
| **D. Go 헬퍼 바이너리** | Fee Delegation/EIP-7702 완전 지원, 프로젝트 내 구축 | 개발 비용 높음 |

**권고**: **A(cast) + D(Go 헬퍼)** 조합
- 일반 TX/ABI/이벤트: cast로 처리 (대부분의 TC 커버)
- Fee Delegation(type 0x16): Go 헬퍼로 처리 (go-stablenet의 signing 로직 재사용)

### 5.2 테스트 실행 단위

| 옵션 | 설명 |
|------|------|
| **체인 per 스크립트** | 각 .sh가 init→start→test→stop 전체 수행 — 격리성 높음, 느림 |
| **체인 per 그룹** | 하드포크 전체가 하나의 체인에서 실행 — 빠름, TC 간 상태 간섭 가능 |
| **하이브리드** | 프로파일이 다른 그룹은 별도 체인, 같은 프로파일은 공유 |

**권고**: **하이브리드** — 프로파일별 체인 1개, 그룹 내 TC는 순차 실행

---

## 6. 자동화 커버리지 예상

### hardfork-test-spec (70 TC)

| 섹션 | TC 수 | 자동화 가능 | 수동 필요 | 비고 |
|------|-------|------------|----------|------|
| 1-1 GovMinter | 12 | 12 | 0 | contract.sh + event.sh 필요 |
| 1-2 secp256r1 | 6 | 6 | 0 | precompile eth_call |
| 1-3 Gas Fee Floor | 6 | 6 | 0 | tx_builder.sh |
| 3 Performance | 4 | 4 | 0 | 벤치마크 스크립트 |
| 4 Config/Integration | 28 | 28 | 0 | chain_state.sh + contract.sh |
| 5 Build/Release | 7 | 7 | 0 | make build + go test |
| **소계** | **63** | **63** | **0** | **90% 커버리지** (나머지 7개는 스펙 내 미분류) |

### regression-test-spec (116 TC)

| 섹션 | TC 수 | 자동화 가능 | 수동 필요 | 비고 |
|------|-------|------------|----------|------|
| A Node/TX/Contract/RPC | 31 | 31 | 0 | |
| B WBFT | 12 | 12 | 0 | |
| C Gas Price | 7 | 7 | 0 | |
| D Fee Delegation | 4 | 2 | 2 | type 0x16 TX 구성이 Go 헬퍼 필요 |
| E Blacklist | 9 | 9 | 0 | |
| F System Contracts | 25+ | 23+ | 2 | 복잡한 멀티시그 시나리오 |
| G API | 나머지 | 대부분 | 소수 | |
| **소계** | **116** | **~112** | **~4** | **96% 커버리지** |

---

## 7. 리스크 및 제약

| 리스크 | 영향 | 완화 방안 |
|--------|------|-----------|
| cast가 Fee Delegation(0x16) 미지원 | D 섹션 4개 TC 자동화 불가 | Go 헬퍼 바이너리 구축 |
| 하드포크 블록 도달 대기 시간 | BohoBlock=100일 때 ~100초 대기 | BohoBlock=10으로 축소한 fast 프로파일 |
| 시스템 컨트랙트 ABI 변경 시 테스트 깨짐 | 하드포크마다 발생 가능 | ABI를 go-stablenet에서 자동 추출 |
| chainbench의 bash 기반 한계 | 복잡한 ABI 인코딩이 어려움 | Python/Go 헬퍼로 위임 |

---

## 8. 다음 단계

Phase 1 (인프라)부터 시작 가능. 구체적 실행을 원하시면:
1. **tx_builder.sh 도구 선택 확정** (cast vs Python vs Go 헬퍼)
2. **하드포크 프로파일 생성** (즉시 가능)
3. **system_contracts.sh 상수 파일 생성** (즉시 가능)

이 3가지를 확정하면 Phase 1을 바로 진행할 수 있습니다.
