# chainbench Evaluation Capability Matrix

> **작성일**: 2026-04-27
> **Status**: active — sprint 진행마다 cell 갱신
> **목적**: coding agent 가 chainbench 를 통해 수행할 수 있는 evaluation 시나리오의 현재 능력과 sprint 별 도달 목표 매트릭스. Sprint spec 작성 시 어느 cell 을 채우는지 명시한다.
> **Related**: `docs/VISION_AND_ROADMAP.md` §1.3, `docs/NEXT_WORK.md` §3

---

## 0. 사용법

- 새 sprint 가 cell 을 채우면 본 문서를 갱신. Sprint spec 의 "Goal" 절에 "본 문서의 §X 의 cell A→B 채움" 명시.
- "지원" 정의: **coding agent 가 MCP 또는 bash CLI 를 통해 호출 가능 + 결과를 구조화된 형태로 회수 가능**.
- bash 가 지원해도 MCP 노출이 없으면 (A) evaluation harness 모드에서는 미지원으로 간주. raw `chainbench_node_rpc` 로 우회는 가능하나 LLM 책임이 커지므로 cell 평가에서 제외.

범례:
- ✅ 지원
- ⚠️ 부분 (제약 조건 명시)
- ❌ 미지원
- 🚧 진행 중 (sprint 명시)

---

## 1. Capability Axes

evaluation 능력은 3축으로 정의:

- **Tx 축**: 어떤 종류의 tx 를 구성·서명·전송·검증할 수 있는가
- **환경 축**: local / remote / hybrid / attached / ssh-remote
- **검증 축**: receipt / event / state / consensus / failure

매 cell 은 surface (bash CLI / Go `network/` / MCP high-level) 별로 평가.

---

## 2. Tx 매트릭스

기준: 체인 client 가 지원하는 tx 타입을 모두 커버한다.

| tx 타입 / 동작 | bash `tests/lib/*` | Go `network/` (`node.tx_send`) | MCP high-level | 비고 |
|---|---|---|---|---|
| Legacy tx (0x0) value transfer | ✅ `tx_builder.sh` (cast) | ✅ Sprint 4 | ❌ raw `node_rpc` 만 | Go 는 env signer + auto-fill nonce/gas |
| EIP-1559 (0x2) dynamic fee | ✅ (cast) | ✅ Sprint 4b | ❌ | `max_fee_per_gas` / `max_priority_fee_per_gas` |
| Fee Delegation (0x16) | ✅ Layer 2 lib (Go helper `chainutil`) | ✅ Sprint 4c (`node.tx_fee_delegation_send`) | ❌ | 이중 서명 — go-stablenet 고유 |
| EIP-7702 SetCode (0x4) | ✅ Layer 2 lib (Go helper) | ✅ Sprint 4c (`authorization_list`) | ❌ | authorization list |
| Contract deploy | ✅ `contract.sh` (cast) | ❌ Sprint 4d (가칭) | ❌ | bytecode + constructor args |
| Contract call (eth_call+ABI) | ✅ `contract.sh` (cast) | ❌ Sprint 4d (가칭) | ❌ | ABI encode/decode |
| Event log fetch (eth_getLogs) | ✅ `event.sh` | ❌ Sprint 4d (가칭) | ❌ | 토픽 필터 |
| Event log decode | ✅ `event.sh` (cast / keccak) | ❌ Sprint 4d (가칭) | ❌ | ABI 기반 decode |
| Receipt polling (status+logs) | ✅ Layer 2 lib | ✅ Sprint 4b | ❌ | exponential backoff, timeout |
| Account state assert (balance/nonce/code/storage) | ✅ `chain_state.sh` | ❌ Sprint 4d (가칭) | ❌ | block number 옵션 |
| Account Extra (블랙리스트/authorized 등) | ✅ `chain_state.sh` | ❌ | ❌ | go-stablenet 고유 |
| Malformed/invalid tx (거부 경로) | ✅ Python (`rlp`) | ❌ | ❌ | negative test |

**해석**:
- bash Layer 2 lib (1071 줄, 단위 테스트 20개) 는 evaluation 시나리오에 필요한 거의 모든 tx 능력을 보유. 단 이는 bash → cast / Go helper / Python 외부 도구로 위임된 형태.
- Go `network/` 는 legacy tx 1종 + 진행 중인 1559/wait 만. 다른 cell 은 Sprint 4 시리즈로 채워야 함.
- MCP 는 어떤 high-level tx 도구도 노출 안 함 → coding agent 는 raw RPC + 스스로 서명/ABI 인코딩이 필요한 상황.

---

## 3. 환경 매트릭스

| 환경 | bash CLI | Go binary (`chainbench-net`) | MCP | 비고 |
|---|---|---|---|---|
| local (init + start + stop) | ✅ `cmd_init/start/stop` | ⚠️ lifecycle 핸들러 미포팅 (read RPC + node lifecycle 일부) | ✅ `chainbench_init/start/stop/...` (bash 우회) | M4 가 점진 흡수 중 |
| local node 조작 (start/stop/restart/tail_log) | ✅ `cmd_node` | ✅ Sprint 2b | ✅ | |
| local hardfork 프로파일 | ✅ `profiles/hardfork-*.yaml` | partial (genesis/toml 만 Go 포팅) | ✅ | Sprint 3c |
| remote (attach + read RPC) | ✅ `cmd_remote` | ✅ Sprint 3b/3b.2a-c | ✅ `chainbench_remote_*` | API key / JWT auth 지원 |
| remote write (tx send) | ✅ (수동) | ✅ Sprint 4 (`node.tx_send`) | ❌ | env signer 필요 |
| hybrid (local + remote nodes 한 네트워크) | ⚠️ 수동 구성 | ⚠️ 스키마 지원 (`networks/<name>.json`), 시나리오/예제 부재 | ❌ | Sprint 5d 예제 |
| ssh-remote | ❌ | ❌ Sprint 5b | ❌ | Q6 / S6 |
| attached (read-only) | ✅ `remote add` 후 read | ✅ | ✅ | |

---

## 4. 검증 매트릭스

| 검증 종류 | bash | Go `network/` | MCP | 비고 |
|---|---|---|---|---|
| Tx receipt status | ✅ Layer 2 lib | ✅ Sprint 4b (`node.tx_wait`) | ❌ | success / failed / pending |
| Tx receipt event log | ✅ `event.sh` | ❌ Sprint 4d (가칭) | ❌ | topics + data decode |
| Block 정보 (height/hash/miner/ts) | ✅ `rpc_block.sh` | ✅ (`node.block_number`) | ✅ `chainbench_consensus_block_info` | |
| Account state (balance/nonce/code/storage) | ✅ | partial (balance 만) | partial (`chainbench_node_rpc` raw) | |
| Account Extra (chain-specific) | ✅ | ❌ | ❌ | |
| Validator set (`istanbul_*` 등) | ✅ `rpc_consensus.sh` | partial (probe 에서만 호출) | ✅ `chainbench_consensus_validators` | adapter 축 의존 |
| Consensus health (참여율/round) | ✅ | ❌ | ✅ `chainbench_consensus_health` | |
| Network peers / topology | ✅ | ❌ | ✅ `chainbench_network_*` | |
| Network partition / heal | ✅ | ❌ | ✅ `chainbench_network_partition` | local only (capability: `network-topology`) |
| Application log (stderr) | ✅ `cmd_log` | ✅ (`node.tail_log` — local only) | ✅ `chainbench_log_*` | capability: `fs` |
| Chain log (`eth_subscribe`) | ❌ | ❌ Sprint 5+ | ❌ | §5.10 split |
| Failure context (process crash 등) | ✅ `failure_context.sh` | ❌ | ✅ `chainbench_failure_context` | |
| Hardfork 활성 검증 | ✅ `chain_state.sh` | ❌ | partial | `eth_call` 기반 |

---

## 5. Sprint 별 도달 목표 (계획 + 진행)

| Sprint | 채우는 cell | 상태 |
|---|---|---|
| 2a~2c | wire protocol + LocalDriver + bash client | ✅ 완료 |
| 3a | chain_type probe | ✅ 완료 |
| 3b / 3b.2a-c | network.attach + remote read RPC + auth | ✅ 완료 |
| 3c | adapter Go 포팅 (stablenet 실, wbft/wemix skeleton) | ✅ 완료 |
| 4 | env signer + `node.tx_send` (legacy tx) | ✅ 완료 |
| **4b** | keystore signer + EIP-1559 + receipt polling (`node.tx_wait`) | ✅ 완료 (2026-04-27) |
| **4c** | Fee Delegation (0x16) + EIP-7702 SetCode (Go 포팅) + SignHash 인터페이스 | ✅ 완료 (2026-04-27) |
| **4d (가칭)** | Contract deploy/call (ABI encode/decode) + event log fetch+decode + state assert | 🚧 계획 |
| **5** | MCP 이관 (high-level evaluation tool) — wire protocol 경유 + NDJSON event/progress/result → MCP response 구조화 변환 | 🚧 계획 (Sprint 4 시리즈 후) |
| 5a/5b/5d | capability gate / SSH driver / hybrid 예제 | 후순위 |

**Sprint 4 시리즈의 의의**: bash Layer 2 lib 가 외부 도구(cast / Go helper / Python) 위에서 제공하는 능력을 Go `network/` 로 이식하여, coding agent 가 단일 surface (chainbench-net wire) 로 모든 tx 시나리오를 호출 가능하게 하는 것이 목표.

**Sprint 5 의 의의**: Sprint 4 시리즈가 채운 Go 능력을 MCP high-level 도구로 노출. 동시에 NDJSON event/progress/result 를 LLM 친화 응답으로 변환하는 layer 구축.

---

## 6. Coverage 측정 방법

- 각 cell 의 ✅ 비율 = 능력 coverage
- Sprint 종료 시 본 문서 갱신 + commit
- coding agent (외부 시스템) 가 본 문서를 읽고 어떤 시나리오가 자동화 가능한지 자가 진단 가능하게 유지

현재 (2026-04-27) coverage 요약:
- bash CLI surface: 약 90% (Layer 2 lib 거의 완비)
- Go `network/` surface: 약 45% (Sprint 2a~4c 누적 — 1559 / receipt polling / keystore signer / 0x4 SetCode / 0x16 fee-delegation 추가로 4b → 4c 단계에서 +10%pt)
- MCP high-level tool surface: 약 10% (lifecycle / read RPC 위주, tx/contract/event 미노출)

→ evaluation harness 모드에서 coding agent 가 누리는 실효 coverage 는 MCP 기준이므로 약 10%. Sprint 4 시리즈 + Sprint 5 가 이를 끌어올리는 메인 동력.

Sprint 4b 완료 (2026-04-27): EIP-1559 / receipt polling / keystore signer 추가.

Sprint 4c 완료 (2026-04-27): EIP-7702 SetCodeTx + go-stablenet 0x16 fee delegation 추가. SignHash 인터페이스 도입으로 chain-specific tx 타입 확장 기반 마련.
