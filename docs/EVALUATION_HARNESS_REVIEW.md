# chainbench Evaluation-Harness Positioning Review

> **작성일**: 2026-04-27
> **Status**: open — 후속 sprint 정의 및 SSoT 재정렬을 위한 입력 문서
> **목적**: chainbench 의 상위 목적(coding agent evaluation harness) 을 명문화하고, 현재 표면(MCP/tx/결과 표현) 이 이 목적을 얼마나 충족하는지 점검한다. P0 작업으로 식별되는 문서 갱신과 후속 sprint 우선순위 재배치의 근거 자료.
> **Related**: `docs/VISION_AND_ROADMAP.md`, `docs/NEXT_WORK.md`, `docs/claudedocs/AUTOMATION_SYSTEM_PROPOSAL_v2.md` (Feature 3 — chainbench 연동), `docs/claudedocs/WORK_INSTRUCTION.md`

---

## 1. 상위 비전 (정정)

`docs/VISION_AND_ROADMAP.md` §1 의 5개 비전(다체인·local/remote·attach·tx·log) 보다
한 단계 위에 다음 명제가 있다.

> chainbench 는 (A) **자동화된 coding agent 의 e2e evaluation harness** 이자
> (B) **사람이 직접 쓰는 독립 도구** 이다. 두 모드는 동일한 핵심 기능 위에서 동작한다.

```
[claude-ai 기반 coding agent]
         │  (LLM 요청)
         ▼
[MCP interface]            ◀── 1차 evaluation 표면
         │  (e2e 의뢰 + 결과 회수)
         ▼
[chainbench]
   ├─ chain node 구성 (local / remote / hybrid)
   ├─ 체인 client 가 지원하는 모든 tx 타입 e2e 검증
   └─ LLM 친화 구조화 결과
```

이 위치 정의는 `docs/claudedocs/AUTOMATION_SYSTEM_PROPOSAL_v2.md` Feature 3
(체인 테스트벤치 통합 — chainbench 연동) 와 `WORK_INSTRUCTION.md` Feature 3
의 description 과 일치한다. 즉 chainbench 자체 문서가 이 위치를 가리키지
않을 뿐, 외부 시스템 문서는 이미 이 포지셔닝을 가정하고 있다.

### 1.1 두 모드의 함의

| 축 | 모드 (A) evaluation harness | 모드 (B) 독립 도구 |
|---|---|---|
| 호출자 | LLM (coding agent) | 사람 (CLI) |
| 1차 표면 | MCP tools | bash CLI |
| 결과 형식 | 구조화 (LLM 친화) | 사람 친화 텍스트 |
| 시나리오 | 코드 변경 → 빌드 → 체인 → 검증 자동 사이클 | 디버깅·점검·실험 |

두 모드가 동일 핵심 기능을 공유하므로, **모드 (A) 의 요구를 표면(MCP +
구조화 결과) 으로 분리**하면 모드 (B) 를 깨뜨리지 않고 추가 가능. 즉
대립 관계가 아니라 **동심원 관계**: 같은 functional core, 다른 표면.

---

## 2. 현재 상태 인벤토리

### 2.1 MCP 표면 (38 tool)

| 영역 | 도구 | 백엔드 |
|---|---|---|
| lifecycle | `chainbench_init` / `start` / `stop` / `restart` / `status` / `state_compact` | bash CLI 직접 |
| node | `chainbench_node_rpc` / `node_start` / `node_stop` | bash |
| test | `chainbench_test_run` / `test_list` / `test_hardfork` / `test_regression` / `test_run_remote` / `report` | bash 테스트 러너 |
| consensus | `chainbench_consensus_validators` / `status` / `block_info` / `health` | bash |
| network | `chainbench_network_peers` / `topology` / `partition` / `txpool_inspect` | bash |
| log | `chainbench_log_timeline` / `log_search` / `failure_context` | bash |
| config / profile / schema | `chainbench_config_get` / `set` / `list` / `profile_get` / `set` / `send` / `schema_query` | bash |
| remote | `chainbench_remote_add` / `list` / `info` / `remove` / `rpc` (+ `test_run_remote`) | bash |
| spec | `chainbench_spec_lookup` | bash |

특징:
- **38개 도구 전부 `network/cmd/chainbench-net` (Go wire protocol) 우회**
- bash CLI 의 stdout 텍스트만 LLM 에 도달
- NDJSON 의 `event` / `progress` / `result` 풍부한 신호 미활용
- `node_rpc` 는 raw JSON-RPC 패스스루 → tx 구성·서명·ABI 인코딩은 LLM 책임

### 2.2 tx 능력 매트릭스 — Go 포팅 격차

| tx / 동작 | bash `tests/lib/*` | Go `network/` (`node.tx_send`) | MCP 노출 (high-level) |
|---|---|---|---|
| Legacy tx (0x0) | ✅ `tx_builder.sh` | ✅ Sprint 4 | ❌ (raw rpc 만) |
| EIP-1559 (0x2) | ✅ | ❌ Sprint 4b 계획 | ❌ |
| Fee Delegation (0x16) | ✅ Layer 2 lib | ❌ | ❌ |
| EIP-7702 SetCode (0x4) | ✅ Layer 2 lib | ❌ | ❌ |
| Contract deploy | ✅ `contract.sh` | ❌ | ❌ |
| Contract call (eth_call+ABI) | ✅ `contract.sh` | ❌ | ❌ |
| Event log decode | ✅ `event.sh` | ❌ | ❌ |
| Receipt polling | ✅ | ❌ Sprint 4b 계획 | ❌ |
| Account state assert | ✅ `chain_state.sh` | ❌ | ❌ |

해석:
- bash Layer 2 라이브러리는 evaluation 시나리오에 필요한 거의 모든 능력 보유
- Go 포팅이 legacy tx 1종에 머물러 있음 → evaluation harness 를 Go binary
  경유로 통일하려면 Sprint 여러 번의 ground 가 더 필요
- 단기에는 **bash → MCP 직접 호출 + 구조화** 가 evaluation 시야 우선이고,
  Go 포팅은 그 다음

### 2.3 wire NDJSON vs MCP 응답

`network/cmd/chainbench-net` 가 emit:
- `event` (예: `node.started`, `chain.block`, `chain.tx`)
- `progress` (단계별 진행률)
- `result` (성공/실패 + 구조화 데이터)

현재 MCP tool 응답:
- bash CLI 의 stdout 을 그대로 또는 약하게 가공한 텍스트
- NDJSON 의 풍부한 정보 → MCP 까지 도달 안 함
- 실패 시 root cause 분석에 필요한 phase 단위 신호 누락

### 2.4 문서 정합성 점검

| 문서 | 상위 비전 반영 | 비고 |
|---|---|---|
| `VISION_AND_ROADMAP.md` §1 | ❌ 누락 | 5개 비전이 chainbench 자체 기능 관점. evaluation harness 절 부재 |
| `VISION_AND_ROADMAP.md` §6 sprint 우선순위 | ⚠️ 정합 부족 | MCP 이관(Sprint 5c) 후순위. 사용자 의도상 우선순위 상승 후보 |
| `NEXT_WORK.md` §3 | ⚠️ 부분 정합 | Sprint 4b/5 가 internal Go 작업 중심. evaluation 표면 강화 시각 누락 |
| `claudedocs/*` | ⚠️ cross-ref 부재 | chainbench 레포 내 문서가 이 자료를 가리키지 않음 |
| `ADAPTER_CONTRACT.md` / `HARDCODING_AUDIT.md` / `SECURITY_KEY_HANDLING.md` | ✅ 자기 scope 안에서 정확 | 단 evaluation harness 와의 관계는 미언급 |

---

## 3. 핵심 Gap 5가지

| # | Gap | 영향 | 근거 |
|---|---|---|---|
| G1 | **상위 비전 명문화 부재** | sprint 우선순위 결정 기준이 흐려짐 | §2.4 VISION §1 |
| G2 | **MCP 가 evaluation 표면으로 부족** | LLM 이 e2e 시나리오 작성 시 raw JSON-RPC + 직접 서명/ABI 인코딩 필요 → coding agent 토큰비/오류 증가 | §2.1 — 38 tool 전부 bash 우회 |
| G3 | **tx 능력 Go 포팅 격차** | bash 의존 끊지 못함. Go binary 통일 evaluation 경로 미완 | §2.2 매트릭스 |
| G4 | **wire NDJSON → MCP 응답 변환 layer 부재** | 실패 root cause 정보 손실. coding agent 의 자동 재시도/수정 사이클 효율 저하 | §2.3 |
| G5 | **claudedocs/ cross-reference 부재** | 새 세션이 chainbench 의 외부 위치(coding agent automation Feature 3) 를 인지 못함 | §2.4 |

---

## 4. 다음 작업 리스트 (우선순위 정렬)

### P0 — 문서 SSoT 정렬 (코드 변경 없음)

문서 작업만으로 G1·G5 해소 + 후속 sprint 의 우선순위 결정 기준 확보.

1. **`VISION_AND_ROADMAP.md` §1 재작성**
   - "상위 목적: coding agent evaluation harness + 독립 도구" 절 추가
   - 5개 자체 비전을 (A)/(B) 모드의 sub-goal 로 재포지셔닝
   - claudedocs/AUTOMATION_SYSTEM_PROPOSAL_v2.md Feature 3 cross-reference

2. **`docs/claudedocs/README.md` 신설** 또는 chainbench README 한 절
   - claudedocs 3 문서가 chainbench 외부 컨텍스트(자동화 시스템 안에서의 위치) 임을 명시
   - chainbench = Feature 3 evaluation tool 명시

3. **`NEXT_WORK.md` §3 우선순위 재정렬**
   - 새 P1: evaluation 표면 강화 (MCP + tx 매트릭스 + 결과 변환 layer)
   - 기존 Sprint 4b 와 5c 의 관계 정리

4. **`docs/EVALUATION_CAPABILITY.md` 신설** (이름 후보)
   - tx 타입 × 환경(local/remote/hybrid) × 검증 항목(receipt/event/state) 매트릭스
   - 현재 지원 / Sprint 별 도달 목표
   - bash Layer 2 lib 와 Go `network/` 의 능력 차이를 명시
   - §2.2 의 표를 정식 SSoT 로 승격

### P1 — Sprint 정의 (P0 후 사용자 합의 필요)

5. **Sprint 4b 재정의 검토**
   - 현재 scope: keystore + EIP-1559 + tx.wait
   - 옵션: evaluation 매트릭스 1차로 확장 — fee delegation 0x16 / contract deploy 일부 포함

6. **Sprint 5c (MCP 이관) 우선 상승 검토**
   - 현재: 후순위 (Sprint 5 의 부분)
   - 옵션: 4b 와 병렬 또는 우선. NDJSON event/progress/result 를 MCP tool response 로 구조화 변환 layer 설계

7. **Sprint 신규 — "result formatting for LLM"**
   - NDJSON → MCP response schema 매핑 정식화
   - 실패 시 root cause hint 포함 (어떤 phase 에서 어떤 신호가 비정상)
   - LLM 이 다음 행동을 결정하기에 충분한 구조

### P2 — 내부 정리 (NEXT_WORK §3.1 P1 그대로)

8. `CHAINBENCH_NET_LOG` env var 동작 수정
9. `network/cmd/chainbench-net/handlers.go` (1046 줄) 분할
10. `signer.SignTx` ctx 파라미터 보존 의도 주석

### P3 — 후순위 (기존 Sprint 5a/5b/5d)

11. capability gate (Sprint 5a)
12. SSHRemoteDriver (Sprint 5b)
13. Hybrid 네트워크 예제 (Sprint 5d)

---

## 5. Open Questions

P0 진행 전 사용자 확인 필요 항목.

| # | 질문 | 영향 |
|---|---|---|
| Q1 | 명명: "coding agent evaluation harness" 표현이 의도와 맞는가? 다른 명명 선호 (예: "automated agent's e2e validator") 가 있는가? | 문서 전반 용어 통일 |
| Q2 | `docs/EVALUATION_CAPABILITY.md` 매트릭스 문서를 신설할지, VISION §3 Gap 분석을 확장할지 | SSoT 위치 결정 |
| Q3 | Sprint 4b scope 를 keystore+1559+wait 그대로 둘지, evaluation 매트릭스 1차로 확장할지 | 다음 sprint 크기 |
| Q4 | MCP 이관(Sprint 5c) 을 4b 보다 우선시킬지, 4b 와 병렬로 할지, 4b 후순위 유지할지 | sprint 순서 |
| Q5 | claudedocs/ 의 chainbench 위치 명시를 README 절로 둘지 별도 README.md 로 둘지 | 문서 구조 |

---

## 6. Exit criteria

이 review 가 closed 되는 시점:

1. Q1~Q5 의 결정이 사용자와 합의됨
2. P0 의 1~4번이 commit 됨 (`VISION_AND_ROADMAP §1` 갱신, claudedocs cross-ref, NEXT_WORK 우선순위 재정렬, EVALUATION_CAPABILITY 신설)
3. P1 의 5~7번이 sprint spec 으로 등록됨 (`docs/superpowers/specs/` 하위)

이후 본 문서는 `docs/superpowers/specs/2026-04-27-evaluation-positioning-review.md` 로 archive 또는 삭제.
