# Chainbench — 남은 작업 리스트

> 작성일: 2026-04-30 (Sprint 5a 완료 시점)
> 목적: 다음 세션에서 즉시 착수 가능한 actionable TODO 리스트.
> 전체 컨텍스트는 `docs/NEXT_WORK.md` (handoff) + `docs/VISION_AND_ROADMAP.md` (SSoT).

---

## 빠른 상태 (2026-04-30 기준)

**완료된 Sprint** (5 series):
- ✅ 5c.1 (2026-04-28) — MCP foundation + 첫 2 tool
- ✅ 5c.2 (2026-04-29) — MCP 남은 4 tool + tx_send 4-mode 완비
- ✅ 5c.3 (2026-04-29) — utils 추출 + node.rpc Go 핸들러 + 3 chainbench_node_* reroute (3/38 ≈ 8%) + integration test layer
- ✅ 5a (2026-04-29) — Capability gate (Go + MCP + bash 3-layer)

**EVALUATION_CAPABILITY MCP coverage**: ~60% (high-level tool 6종 + capability gate)
**Reroute 진행도**: 3/38 (~8%)
**테스트 매트릭스**: vitest 94/94 · Go 16 packages · bash 34/34

**다음 P1**: Sprint 5c.4 (lifecycle reroute)

---

## Sprint 5 시리즈 — 남은 sprint

### 🟥 Priority 1 — Sprint 5c.4: Lifecycle reroute

**목표**: 6 lifecycle MCP tool (`chainbench_init/start/stop/restart/status/clean`) 을 wire 경유로 전환. 현재 `runChainbench` (bash CLI shell-out) 사용 중.

**필요 작업**:
1. **Go-side wire 핸들러 6종 신규 작성** (chainbench-net 에 없음):
   - `network.init` — bash `chainbench init` 의 Go 포팅 (Adapter Go 포팅이 3c 에 끝나서 가능)
   - `network.start_all` / `network.stop_all` / `network.restart` — 전체 노드 lifecycle
   - `network.status` — 노드 상태 합성 응답
   - `network.clean` — datadir / pids 초기화
2. **MCP 측 reroute** — `chainbench_init/start/stop/restart/status/clean` 6 tool 을 callWire 경유
3. **`chainbench_node_start binary_path` fallback 제거** — Go `node.start` 가 binary_path arg 수용하도록 확장 (Sprint 5c.3 P3 row 닫힘)
4. **Integration test harness 추출** — `mcp-server/test/integration/_harness.ts` 로 5c.3 의 단일 파일 setup 을 재사용 가능한 형태로 (Sprint 5c.3 P3 row 닫힘 — 6+ integration test 추가 전 prerequisite)
5. **Integration test cleanup-await + port-race 진단 보강** (Sprint 5c.3 P3)
6. **`validateRpcMethod` duplicate in remote.ts 통합** (Sprint 5c.3 P3 — remote.ts reroute 시점)
7. **Docs + version bump 0.7.0 → 0.8.0**

**완료 시 효과**: Reroute coverage 3/38 → 9/38 (~24%). lifecycle MCP 호출이 모두 wire 경유 → bash subprocess 지연 제거 + 일관 NDJSON 응답.

**예상 commit 수**: 8~10 (Go 핸들러 6 + MCP 6 + harness 추출 + docs).

---

### 🟧 Priority 2 — Sprint 5b: SSHRemoteDriver

**목표**: `drivers/sshremote` 신설. 원격 머신에 SSH 로 chainbench-net (또는 노드 lifecycle) 을 spawn 가능하게.

**필요 작업**:
1. **Driver 인터페이스 확장** — capability `ssh-remote` (rpc/ws/process/fs) 가 5a 에서 이미 declared. 실 driver 만 추가
2. **SSH 자격증명 (S6 = 세션 prompt)** — 평문 파일 저장 금지. 자동화 시 OS 키체인 연동 후속
3. **Node lifecycle 구현** — start/stop/restart over SSH 채널
4. **fs / process capability 활성화** — log tail / pids 검사 등
5. **`network.attach` 확장** — `ssh-remote` provider 인식 + 자격 입력 흐름
6. **Integration test** — 로컬 SSH 서버 (ssh-keygen + sshd container) 또는 mock SSH

**복잡도**: 큰 sprint — 새 driver + 자격 관리 + lifecycle 구현. 두 패스로 나눌 가능성:
- **5b.1**: SSH dial + read-only RPC (capability `[rpc, ws]`)
- **5b.2**: Process / fs capability 활성화 + node lifecycle

---

### 🟦 Priority 3 — Sprint 5d: Hybrid 네트워크 예제

**목표**: Local 노드 + Remote 노드 혼합 네트워크의 실제 사용 시나리오 + 예제.

**필요 작업**:
1. **`profiles/hybrid-example.yaml`** — local 3 + remote 1 구성 sample
2. **Hybrid network attach 흐름** — 기존 `state/networks/<name>.json` 스키마는 이미 hybrid 지원 (per-node provider 바인딩) — 사용 시나리오만 부재
3. **테스트 시나리오** — hybrid 환경에서 fault test 의 capability 게이팅이 의도대로 동작 검증 (5a 의 set 교집합 결과)
4. **MCP integration** — `chainbench_network_capabilities` 가 hybrid 의 lower bound 응답 검증

**복잡도**: 작은 sprint (~3-5 commits). 인프라는 이미 존재 — 예제 + 검증만.

---

## Priority 2 — Sprint 4d 후속 (트리거 조건 시)

별도 sprint 가 아닌 트리거 조건 충족 시 진행:

| 항목 | 트리거 |
|---|---|
| Account Extra (`isBlacklisted` / `isAuthorized` 등 stablenet 고유) | go-stablenet evaluation 시나리오에서 실수요 발생 시 |
| `abiutil` tuple / nested-array / fixed-bytesN(N≠32) 지원 | 사용자가 raw calldata fallback 으로 우회 못하는 경우 |
| `abiutil.DecodeLog` anonymous event | anonymous event 디코딩 필요 발생 시 |

---

## Priority 3 — 누적 tech debt 백로그

기존 sprint 에서 이월된 항목들. **별도 sprint 가 아니라 관련 코드 건드릴 때 흡수**.
전체 표는 `NEXT_WORK.md` §3 P3 참조. 핵심만 요약:

### 5c.4 lifecycle 작업 시 자연 흡수
- `chainbench_node_start binary_path` fallback 제거 (5c.3)
- Integration test harness 추출 (`_harness.ts`) (5c.3)
- Integration test cleanup-await + port-race 진단 (5c.3)
- `chain_tx.ts` 505 lines per-tool 분할 검토 (5c.2)

### 5c.5+ remote.ts reroute 시 흡수
- `validateRpcMethod` duplicate in remote.ts (5c.3)

### 5번째 chain-specific tx 타입 도입 시
- `feeDelegationAllowedChains` hardcoded 맵 → `Adapter.SupportedTxTypes()` 인터페이스 promotion (4c)

### 다음 `handlers_node_tx.go` 핸들러 추가 시
- 핸들러 closure 길이 (~184~211줄) — phase 별 helper 추출 (4c)

### 5번째 read handler 추가 시
- `handlers_node_read.go` 파일 크기 (800줄 근접) 분할 (4d)

### Sprint 5b 시
- Spec/plan terminology drift — `state.LoadNodes` / `state.Node` 가 spec 에 등장. 실제는 `state.LoadActive` + `types.Node` (5a)

### LLM ergonomics 강화 시
- MCP capability-aware tool gating — capability 부재 시 자동 retry/skip layer (5a)
- Runtime capability probe — provider declaration 외에 admin RPC 실제 동작 여부 확인 (5a)

### 점진 진행 (트리거: remote 환경에서 부적합 test 발견 시)
- All fault/regression tests 의 `requires_capabilities` frontmatter 부여 — 5a 는 fault/node-crash + node-recover 만 demo. 남은: network-partition, p2p-topology, two-down, txpool-leader-change, regression 카테고리들 (5a)

---

## Sprint 5 시리즈 외 — 후순위 큰 작업

| 항목 | 예상 시기 |
|---|---|
| WebSocket subscription / chain log streaming | Sprint 6+ (subscription.open wire surface 신설 필요) |
| `chainbench-net network.init` wire handler | Sprint 5c.4 의 일부로 흡수 |
| wbft `GenerateGenesis`/`GenerateToml` 실구현 | wbft 체인 실제 사용 시 |
| wemix 실구현 | wemix 체인 실제 사용 시 |
| 401/403 distinct APIError code | 3b.2b follow-up — remote auth 실패 진단 정밀화 |

---

## 다음 세션에서 즉시 착수 가능한 단위

**가장 작은 단위**: Sprint 5d (hybrid 예제) — 인프라 존재, 예제 + 테스트만. ~3-5 commits.

**중간 단위**: Sprint 5c.4 (lifecycle reroute) — Go 핸들러 6종 + MCP reroute 6 + harness 추출. ~8-10 commits. 가장 가치 있음 (reroute coverage 3/38 → 9/38).

**큰 단위**: Sprint 5b (SSH driver) — 새 driver 구현. 두 패스로 나눠 진행 권장.

**권장 순서** (본 sprint 종료 시점 기준):
1. Sprint 5c.4 (P1 — lifecycle reroute) — 가장 큰 LLM-facing 효과
2. Sprint 5d (smallest, easy win)
3. Sprint 5b (큰 작업, 두 패스 권장)

---

## 새 세션 재개 명령 (체크리스트)

```bash
# 1. 현재 상태 확인
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git log --oneline -10
git status

# 2. 회귀 테스트 (모두 green 이어야 함)
go -C network test ./... -count=1 -timeout=60s   # 16 packages
bash tests/unit/run.sh                            # 34/34
cd mcp-server && npm test                         # 94/94
cd .. && cd mcp-server && npx tsc --noEmit && npm run build

# 3. 다음 작업 결정
cat docs/REMAINING_WORK.md         # 본 문서 (요약)
cat docs/NEXT_WORK.md              # 전체 핸드오프 (자세한 컨텍스트)
cat docs/VISION_AND_ROADMAP.md     # SSoT 비전 + 로드맵

# 4. Sprint 5c.4 시작 시
cat docs/superpowers/specs/2026-04-29-sprint-5c-3-mcp-reroute-pass-1.md   # 직전 reroute 패턴
cat docs/superpowers/specs/2026-04-29-sprint-5a-capability-gate.md         # 직전 sprint 패턴
ls network/cmd/chainbench-net/handlers_*.go    # 추가할 핸들러 위치
```

---

## 참고: 본 문서 vs `NEXT_WORK.md`

- **본 문서 (REMAINING_WORK.md)**: actionable TODO 리스트. 빠른 스캔용. ~200줄.
- **NEXT_WORK.md**: 전체 핸드오프 (프로젝트 컨텍스트, 디렉토리 레이아웃, 규약, 주의사항, 재개 가이드). ~600줄.
- **VISION_AND_ROADMAP.md**: 비전 + 로드맵 SSoT. 디자인 결정의 source of truth.

세 문서는 보완 관계. 본 문서는 sprint 종료 시점에 갱신되며, NEXT_WORK / VISION 은 매 sprint 의 docs+chore 커밋에서 갱신됨.
