# Chainbench — 남은 작업 리스트

> 작성일: 2026-04-30 (Sprint 5a 완료 시점)
> 목적: 다른 세션에서 맥락 없이도 즉시 착수 가능한 actionable 핸드오프.
> 본 문서는 self-contained — `NEXT_WORK.md` (full context) / `VISION_AND_ROADMAP.md` (비전 SSoT) 는 깊게 들어갈 때만.

---

## 0. 프로젝트 30초 컨텍스트

**Chainbench** = `go-stablenet` (geth 포크, WBFT 합의) 용 로컬 블록체인 샌드박스 + 테스트 프레임워크. 두 모드:
- **(A) coding agent evaluation harness** — `claude-ai` 기반 자동화 코드 개발 시스템이 LLM 으로 코드를 짜고 chainbench 를 e2e 호출하여 검증. **1차 표면 = MCP**.
- **(B) 독립 도구** — 사람이 직접 쓰는 chain 샌드박스. **1차 표면 = bash CLI** (`chainbench init/start/stop/test/log/...`).

**5개 핵심 능력**:
1. **EVM 다체인 지원** — `go-stablenet` (현재), `go-wbft` / `go-wemix` / `go-ethereum` (확장)
2. **로컬 또는 원격에 체인 네트워크 구성**
3. **이미 구성된 체인 네트워크에 attach**
4. **모든 tx 타입 전송** — legacy / EIP-1559 / EIP-7702 SetCode (0x4) / go-stablenet fee delegation (0x16)
5. **Log 수집** — chain log + application log 분리

**현재 진행 단계 (2026-04-30)**: Sprint 4 시리즈 (Go 능력 완비) 종료 → Sprint 5 시리즈 (MCP 노출 + 주변 인프라) 진행 중. Sprint 5c.1/5c.2 (high-level tool 노출) → 5c.3 (reroute pass 1) → 5a (capability gate) 까지 완료.

---

## 1. 아키텍처 빠른 지도

```
┌─────────────────────────────────────────────────────────────────┐
│ 상위 (외부 진입점)                                              │
│   bash CLI (chainbench)  │   MCP server (TS)                   │
│   사람 + 테스트 러너     │   coding agent (LLM)                │
└──────────────┬───────────┴───────────────┬──────────────────────┘
               │                           │
               │ runChainbench / cb_net_call (NDJSON envelope on stdin)
               │                           │
               ▼                           ▼
┌─────────────────────────────────────────────────────────────────┐
│ chainbench-net (Go binary)                                      │
│   network/cmd/chainbench-net  -- spawn-per-call                 │
│   stdin: {command, args}     stdout: NDJSON {event/progress/result} │
│   stderr: structured slog                                       │
├─────────────────────────────────────────────────────────────────┤
│ Internal layers (network/internal/)                             │
│   handlers/ (network.* + node.*)  ◄── 본 sprint 에 추가 핸들러 │
│   drivers/ (local · remote · ssh-remote (future))               │
│   adapters/ (stablenet 실, wbft/wemix skeleton)                 │
│   signer/ (sealed struct, redaction boundary)                   │
│   state/ (pids.json + networks/<name>.json)                     │
│   types/ (go-jsonschema 생성 — schema/*.json)                   │
└─────────────────────────────────────────────────────────────────┘
```

**핵심 포인트**:
- **Wire protocol** = NDJSON envelope (`{command, args}` 입력 + `event/progress/result` 출력). schema enum 은 `network/schema/command.json`.
- **`network.<verb>`** = 네트워크 단위 작업 (load/probe/attach/capabilities).
- **`node.<verb>`** = 노드 단위 작업 (start/stop/rpc/tx_send/...).
- **모든 handler 의 패턴**: `newHandleX(stateDir) Handler` 로 closure 반환 → `handlers.go` 의 `allHandlers` 맵에 등록 → `handlers_test.go` 에 단위 테스트.
- **MCP 측 패턴** (Sprint 5c.3 부터 정착): `XxxArgs.strict()` zod schema (no underscore) + `_xxxHandler` async (underscore for testability) + `callWire("namespace.verb", ...)` + `formatWireResult(result)`.

---

## 2. 빠른 상태 (2026-04-30 기준)

**완료된 Sprint** (5 series):
- ✅ 5c.1 (2026-04-28) — MCP foundation + 첫 2 tool (`account_state`, `tx_send`)
- ✅ 5c.2 (2026-04-29) — MCP 남은 4 tool (`contract_deploy/call`, `events_get`, `tx_wait`) + tx_send 4-mode 완비 (legacy/1559/set_code/fee_delegation)
- ✅ 5c.3 (2026-04-29) — utils 추출 (mcpResp + hex) + Go `node.rpc` 핸들러 + 3 chainbench_node_* reroute (3/38 ≈ 8%) + real-binary integration test layer
- ✅ 5a (2026-04-29) — Capability gate (Go `network.capabilities` + MCP `chainbench_network_capabilities` + bash `requires_capabilities` frontmatter gating)

**커버리지 지표**:
- EVALUATION_CAPABILITY MCP column: **~60%** (high-level tool 6종 + capability)
- Reroute 진행도: **3/38 (~8%)**
- 테스트: vitest **94/94** · Go **16 packages** · bash **34/34** · 회귀 0

**다음 P1**: Sprint 5c.4 (lifecycle reroute) — 가장 큰 LLM-facing 효과 (reroute 8% → 24%)

---

## 3. Sprint 진행 패턴 (10+ sprint 에서 확립)

새 sprint 작업 시 **표준 플로우**:

1. **Spec 작성** → `docs/superpowers/specs/YYYY-MM-DD-<topic>.md`
   - Goal / Non-goals / Security Contract / User-Facing Surface / Package Layout / Tests / Schema / Error Classification / Migration / Out-of-Scope Reminders
2. **Plan 작성** → `docs/superpowers/plans/YYYY-MM-DD-<topic>.md`
   - Task 분해 + 각 task 의 step-by-step (RED → GREEN → commit)
   - 각 task 마다 단일 commit (또는 task + review fix 두 commit)
3. **Spec+plan 커밋**: `docs: add Sprint X spec + plan for <topic>`
4. **Subagent-driven execution**: task 마다 fresh `general-purpose` subagent → 구현 → 보고
5. **Code review**: task 마다 `superpowers:code-reviewer` subagent → Critical/Important/Minor → fix subagent (필요 시)
6. **Final review**: 전체 sprint 를 cross-cutting 관점으로
7. **Docs + version bump**: `docs+chore(sprint-X): roadmap + capability matrix + version 0.N.0`
8. **Roadmap update**: `VISION_AND_ROADMAP.md` 의 Sprint 체크박스 + `NEXT_WORK.md` (timeline + P1 + P3 + file size) + 본 문서

**참고 sprint** (가장 최근, 패턴 모방용):
- `docs/superpowers/specs/2026-04-29-sprint-5a-capability-gate.md` + plan — bash + Go + MCP 3-layer 통합 sprint
- `docs/superpowers/specs/2026-04-29-sprint-5c-3-mcp-reroute-pass-1.md` + plan — reroute 패턴 (Go handler + MCP rewrite + integration test)
- `docs/superpowers/specs/2026-04-28-sprint-5c-2-mcp-remaining-tools.md` + plan — MCP 도구 다수 추가 패턴

**커밋 discipline**:
- English commit messages (docs 본문은 Korean OK)
- NO `Co-Authored-By` trailer
- NO "Generated with Claude Code"
- NO emoji
- Conventional Commits prefix (`feat(scope):`, `fix(scope):`, `refactor(scope):`, `test(scope):`, `docs:`, `docs+chore(scope):`)
- 기존 commit 수정 금지 — 새 commit 으로 처리 (특히 review fix 는 별도 commit)

**Subagent 가용성 fallback**: org 한도 도달 시 직접 구현 가능. Sprint 5a 의 Tasks 3-5 가 이 케이스 — 단위 테스트 (9/9 + 회귀 검사) 가 cross-cutting 검증 대체.

---

## 4. Sprint 5 시리즈 — 남은 sprint

### 🟥 Priority 1 — Sprint 5c.4: Lifecycle reroute

**목표**: 6 lifecycle MCP tool (`chainbench_init/start/stop/restart/status/clean`) 을 wire 경유로 전환. 현재 `runChainbench` (bash CLI shell-out) 사용 중.

**필요 작업**:
1. **Go-side wire 핸들러 6종 신규 작성** (chainbench-net 에 없음):
   - `network.init` — bash `chainbench init` 의 Go 포팅 (Adapter Go 포팅이 3c 에 끝나서 가능)
   - `network.start_all` / `network.stop_all` / `network.restart` — 전체 노드 lifecycle
   - `network.status` — 노드 상태 합성 응답
   - `network.clean` — datadir / pids 초기화
   - **schema enum 갱신 + `go generate ./...` 으로 `command_gen.go` 재생성 + 커밋**
2. **MCP 측 reroute** — `chainbench_init/start/stop/restart/status/clean` 6 tool 을 callWire 경유 (`mcp-server/src/tools/lifecycle.ts`)
3. **`chainbench_node_start binary_path` fallback 제거** — Go `node.start` 가 binary_path arg 수용하도록 확장 (Sprint 5c.3 P3 row 닫힘)
4. **Integration test harness 추출** — `mcp-server/test/integration/_harness.ts` 로 5c.3 의 단일 파일 setup 을 재사용 가능한 형태로 (Sprint 5c.3 P3 — 6+ integration test 추가 전 prerequisite)
5. **Integration test cleanup-await + port-race 진단 보강** (Sprint 5c.3 P3)
6. **`validateRpcMethod` duplicate in remote.ts 통합** (Sprint 5c.3 P3)
7. **Docs + version bump 0.7.0 → 0.8.0**

**완료 시 효과**: Reroute coverage 3/38 → **9/38 (~24%)**. lifecycle MCP 호출이 모두 wire 경유 → bash subprocess 지연 제거 + 일관 NDJSON 응답.

**예상 commit 수**: 8~10.

**첫 task 시작 템플릿**:
```
1. spec 작성: docs/superpowers/specs/YYYY-MM-DD-sprint-5c-4-lifecycle-reroute.md
   - 5c.3 spec 을 ref 로 (동일한 reroute 패턴)
2. plan 작성: docs/superpowers/plans/...
3. Task 0 = harness 추출 (5c.3 review 권장 — 6+ tool 추가 전 선행)
4. Task 1 = Go network.init 핸들러 (5c.3 Task 1 = node.rpc 핸들러 패턴 모방)
5. Task 2-6 = 나머지 5 lifecycle 핸들러
6. Task 7-8 = MCP reroute (5c.3 Task 2-3 = node tool reroute 패턴)
7. Task 9 = node.start binary_path 확장
8. Task 10 = docs + version bump
```

---

### 🟧 Priority 2 — Sprint 5b: SSHRemoteDriver

**목표**: `drivers/sshremote` 신설. 원격 머신에 SSH 로 chainbench-net (또는 노드 lifecycle) 을 spawn 가능하게.

**왜 필요한가**: VISION §5.16 의 S6 결정 — SSH 자격증명은 세션 prompt 로 받음 (평문 파일 저장 금지). VISION §5.4 의 capability set `[rpc, ws, process, fs]` 가 5a 에서 이미 declared 되어 있어, driver 구현만 추가하면 됨.

**필요 작업**:
1. **Driver 인터페이스 확장** — `network/internal/drivers/sshremote/` 디렉토리 신설. `local`, `remote` driver 의 인터페이스 (start/stop/restart/tail_log/RPC dial) 를 SSH 채널 위에 구현
2. **SSH 자격증명** — `golang.org/x/crypto/ssh` 또는 비슷한 패키지 도입. user+password 로 connect (S6 결정). 자동화는 OS 키체인 후속
3. **Node lifecycle over SSH** — `start_all` / `stop_all` 등을 원격 셸 실행으로 매핑. Output streaming 은 SSH session stdout 사용
4. **fs / process capability 활성화** — `tail_log` 가 SSH `tail -f` 로 동작
5. **`network.attach` 확장** — `state/networks/<name>.json` 의 node 별 `provider: "ssh-remote"` + `auth.type: "ssh"` (user + host + password 입력 흐름)
6. **Integration test** — `golang.org/x/crypto/ssh` 의 `Server` 로 mock SSH server, 또는 docker `linuxserver/openssh-server` container 활용 가능

**복잡도**: 큰 sprint — 두 패스 권장:
- **5b.1**: SSH dial + 인증 + read-only RPC (`capability: [rpc, ws]` — 기존 remote 와 동일)
- **5b.2**: Process / fs capability 활성화 (lifecycle + tail_log) — 5c.4 의 lifecycle 핸들러가 끝나야 의미 있음

**참고 spec**: 없음. `drivers/remote/` 가 read-only RPC + auth (Sprint 3b.2b) 의 reference. SSH 측은 새 영역.

**P3 carry-over**: Spec/plan terminology drift — 5a spec 이 `state.LoadNodes` / `state.Node` 표기. 실제는 `state.LoadActive` + `types.Node` (5b spec 작성 시 정정).

---

### 🟦 Priority 3 — Sprint 5d: Hybrid 네트워크 예제

**목표**: Local 노드 + Remote 노드 혼합 네트워크의 실제 사용 시나리오 + 예제.

**필요 작업**:
1. **`profiles/hybrid-example.yaml`** — local 3 + remote 1 구성 sample (1 remote = sepolia 또는 임의 testnet attach)
2. **Hybrid network attach 흐름 문서화** — 기존 `state/networks/<name>.json` 스키마는 이미 hybrid 지원 (per-node provider 바인딩) — 사용 시나리오만 부재
3. **테스트 시나리오** — hybrid 환경에서 fault test 의 capability 게이팅이 의도대로 동작 검증 (5a 의 set 교집합 결과: hybrid = local ∩ remote = `[rpc, ws]` only)
4. **MCP integration** — `chainbench_network_capabilities` 가 hybrid 의 lower bound 응답 검증 (Sprint 5a 에서 이미 `TestHandleNetworkCapabilities_Hybrid` 단위 테스트 통과)
5. **(선택) Layer 2 test 가 hybrid 환경에서 작동 검증** — read-only test 만 통과해야 함

**복잡도**: 작은 sprint (~3-5 commits). 인프라는 이미 존재 — 예제 + 검증만.

**참고 spec**: VISION §5.6 의 hybrid 데이터 모델 + §5.7 capability 시나리오.

---

## 5. Priority 2 — Sprint 4d 후속 (트리거 조건 시)

별도 sprint 가 아닌 트리거 조건 충족 시 진행:

| 항목 | 트리거 |
|---|---|
| Account Extra (`isBlacklisted` / `isAuthorized` 등 stablenet 고유) | go-stablenet evaluation 시나리오에서 실수요 발생 시 |
| `abiutil` tuple / nested-array / fixed-bytesN(N≠32) 지원 | 사용자가 raw calldata fallback 으로 우회 못하는 경우 |
| `abiutil.DecodeLog` anonymous event | anonymous event 디코딩 필요 발생 시 |

---

## 6. Priority 3 — 누적 tech debt 백로그

기존 sprint 에서 이월된 항목들. **별도 sprint 가 아니라 관련 코드 건드릴 때 흡수**.
전체 표는 `NEXT_WORK.md` §3 P3 참조. 트리거 조건별 그룹화:

### 5c.4 lifecycle 작업 시 자연 흡수
- `chainbench_node_start binary_path` fallback 제거 (5c.3)
- Integration test harness 추출 (`_harness.ts`) (5c.3) — **prerequisite**
- Integration test cleanup-await + port-race 진단 (5c.3)
- `chain_tx.ts` 505 lines per-tool 분할 검토 (5c.2)

### 5c.5+ remote.ts reroute 시
- `validateRpcMethod` duplicate in remote.ts (5c.3)

### 5번째 chain-specific tx 타입 도입 시
- `feeDelegationAllowedChains` hardcoded 맵 → `Adapter.SupportedTxTypes()` 인터페이스 promotion (4c)

### 다음 `handlers_node_tx.go` 핸들러 추가 시
- 핸들러 closure 길이 (~184~211줄) — phase 별 helper 추출 (4c)

### 5번째 read handler 추가 시
- `handlers_node_read.go` 파일 크기 (800줄 근접) 분할 (4d)

### Sprint 5b 시
- Spec/plan terminology drift — `state.LoadNodes` / `state.Node` → `state.LoadActive` + `types.Node` (5a)

### LLM ergonomics 강화 시
- MCP capability-aware tool gating — capability 부재 시 자동 retry/skip layer (5a)
- Runtime capability probe — provider declaration 외에 admin RPC 실제 동작 여부 확인 (5a)

### 점진 진행 (트리거: remote 환경에서 부적합 test 발견 시)
- All fault/regression tests 의 `requires_capabilities` frontmatter 부여 — 5a 는 fault/node-crash + node-recover 만 demo. 남은: network-partition, p2p-topology, two-down, txpool-leader-change, regression 카테고리들 (5a)

---

## 7. Sprint 5 시리즈 외 — 후순위 큰 작업

| 항목 | 예상 시기 |
|---|---|
| WebSocket subscription / chain log streaming | Sprint 6+ (subscription.open wire surface 신설 필요) |
| `chainbench-net network.init` wire handler | Sprint 5c.4 의 일부로 흡수 |
| wbft `GenerateGenesis`/`GenerateToml` 실구현 | wbft 체인 실제 사용 시 |
| wemix 실구현 | wemix 체인 실제 사용 시 |
| 401/403 distinct APIError code | 3b.2b follow-up — remote auth 실패 진단 정밀화 |

---

## 8. 주의사항 (trap & gotcha — 자주 빠지는 함정)

### 8.1 Bash 3.2 호환성 (macOS 기본)
- `tests/unit/run.sh` 가 자동으로 bash 4+ 로 re-exec — 새 unit test 작성 시 가급적 bash 3.2 safe
- 금지: associative array (`declare -A`), `&>`, `${var^^}`, `${var,,}`
- 권장: `printf` 보다 `echo` 단순 출력, `for r in $array` 형식

### 8.2 go-stablenet ↔ go-ethereum module path 충돌
- 둘 다 module path = `github.com/ethereum/go-ethereum` → 한 binary 에 둘 다 import 불가
- 결정 (Sprint 3b.2a): 업스트림 `go-ethereum` 사용. go-stablenet 노드는 표준 eth_* RPC 제공하므로 호환. 체인 특화 RPC (`istanbul_*` 등) 는 raw `rpc.Client.Call` 로

### 8.3 Signer redaction boundary (보안 — 절대 깨뜨리지 말 것)
- `network/internal/signer/sealed` 의 모든 출력 경로가 `"***"` 로 redact (slog / fmt.Stringer / fmt.GoStringer)
- 에러 메시지에 key material 포함 금지 — alias 와 env var 이름만 참조
- 새 boundary 추가 시 `tests/unit/tests/security-key-boundary.sh` 패턴 모방

### 8.4 Schema 변경 시 `go generate ./...` 필수
- `network/schema/*.json` 수정 시 `cd network && go generate ./...` 으로 `command_gen.go` 등 재생성
- 재생성 결과 반드시 같은 commit 에 포함

### 8.5 Stale LSP diagnostics
- 새 심볼 (handler 함수, 새 enum) 도입 직후 LSP 가 "unused" 또는 "undefined" 경고
- 무시 — `go test ./... -count=1` 통과하면 stale cache. `go vet` + `gofmt -l` clean 도 confirm

### 8.6 Subagent org 한도
- 한 세션에서 다수 sprint 진행 시 org 한도 도달 가능
- Fallback: 직접 구현 + 회귀 테스트 (vitest/Go/bash 모두 green) 로 cross-cutting 검증
- Sprint 5a Tasks 3-5 가 이 케이스 — 모두 회귀 0 으로 완료

### 8.7 commit-hook orphan lock
- `.git/index.lock` 가 가끔 orphan 상태로 남음 → `lsof .git/index.lock` 로 active git 프로세스 확인 후 `rm -f` (active 없을 때만)

### 8.8 `types.Auth` 는 tagged union 아님
- `network/schema/network.json` 의 `Auth` 는 oneOf 구조지만 go-jsonschema 가 `map[string]interface{}` 로 fallback
- `auth["type"].(string)` 으로 분기. tagged union refactor 는 후속

---

## 9. 다음 세션에서 즉시 착수 가능한 단위

**가장 작은 단위**: Sprint 5d (hybrid 예제) — 인프라 존재, 예제 + 테스트만. ~3-5 commits.

**중간 단위**: Sprint 5c.4 (lifecycle reroute) — Go 핸들러 6종 + MCP reroute 6 + harness 추출. ~8-10 commits. **가장 가치 있음** (reroute coverage 3/38 → 9/38).

**큰 단위**: Sprint 5b (SSH driver) — 새 driver 구현. 두 패스로 나눠 진행 권장.

**권장 순서** (본 sprint 종료 시점 기준):
1. **Sprint 5c.4** (P1) — 가장 큰 LLM-facing 효과. P3 4건 자연 흡수 (harness 추출 등).
2. **Sprint 5d** (smallest, easy win) — capability gate 의 dividend 검증.
3. **Sprint 5b** (큰 작업, 두 패스) — 5c.4 끝나야 lifecycle 측면이 의미. 5b.1 (read-only) 부터.

---

## 10. 새 세션 재개 명령 (체크리스트)

```bash
# 1. 현재 상태 확인
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git log --oneline -10
git status

# 2. 회귀 테스트 (모두 green 이어야 함)
go -C network test ./... -count=1 -timeout=60s   # 16 packages
bash tests/unit/run.sh                            # 34/34
cd mcp-server && npm test                         # 94/94
cd .. && cd mcp-server && npx tsc --noEmit && npm run build && cd ..

# 3. 다음 작업 결정
cat docs/REMAINING_WORK.md              # 본 문서 — 가장 빠른 핸드오프
cat docs/NEXT_WORK.md                    # 전체 핸드오프 (자세한 컨텍스트, 트러블 발생 시)
cat docs/VISION_AND_ROADMAP.md           # SSoT 비전 + 로드맵 (디자인 결정 source)

# 4. Sprint 진행 시 참고 spec/plan
ls docs/superpowers/specs/ | sort | tail -10     # 최근 spec 목록
ls docs/superpowers/plans/ | sort | tail -10     # 최근 plan 목록

# 5. 코드 진입점 (3분 읽기로 모듈 이해)
cat mcp-server/src/utils/wire.ts                 # spawn + NDJSON helper
cat mcp-server/src/utils/wireResult.ts           # NDJSON → MCP response transformer
cat mcp-server/src/tools/chain_read.ts           # read-only MCP tool 패턴
cat mcp-server/src/tools/chain_tx.ts             # write MCP tool 패턴 (4-mode tx_send)
cat mcp-server/src/tools/node.ts                 # reroute 패턴 (5c.3)
cat network/cmd/chainbench-net/handlers_network.go    # network.<verb> 핸들러
cat network/cmd/chainbench-net/handlers_node_read.go  # node.rpc / contract_call / events_get / account_state / tx_wait
cat network/cmd/chainbench-net/handlers_node_tx.go    # node.tx_send / contract_deploy / fee_delegation
cat network/internal/signer/signer.go            # 서명 경계 + redaction 패턴
cat lib/network_client.sh                        # bash → Go 바이너리 NDJSON 브리지
cat lib/cmd_test.sh                              # bash test runner + capability gating (Sprint 5a)
```

---

## 11. 문서 구조 — 본 문서 vs 형제 문서

세 문서가 보완 관계로 상호 의존:

| 문서 | 분량 | 역할 |
|---|---|---|
| **REMAINING_WORK.md** (본 문서) | ~330줄 | Actionable TODO + 30초 컨텍스트 + 빠른 시작. **새 세션 즉시 참조**. |
| **NEXT_WORK.md** | ~600줄 | 전체 핸드오프 — 디렉토리 레이아웃, 규약 풀버전, 주의사항 풀버전, 모든 sprint 별 P3 표. **트러블 발생 시 깊게 참조**. |
| **VISION_AND_ROADMAP.md** | ~860줄 | 비전 + 로드맵 SSoT. 디자인 결정의 source (Q1~Q6, S1~S8, §5.4 Provider Interface, §5.17 Go module 상세). **디자인 변경 시 갱신**. |

**갱신 시점**:
- 매 sprint 의 docs+chore 커밋에서 NEXT_WORK + VISION 갱신
- Sprint 종료 시점에 본 문서 (REMAINING_WORK) 갱신
- 본 문서가 stale 해 보이면 NEXT_WORK §2.1 timeline 의 가장 최근 row 와 본 문서 §2 의 "완료된 Sprint" 비교

**spec/plan 파일** (sprint 진행 시 참고):
- `docs/superpowers/specs/YYYY-MM-DD-<topic>.md` — 디자인 spec
- `docs/superpowers/plans/YYYY-MM-DD-<topic>.md` — task 단위 implementation plan
- 매 sprint 종료 후 보존 — 미래 sprint 가 패턴 참고용으로 사용

---

## 12. 보안 / 사용자 규칙 핵심 (다시 강조)

- **English commit messages** (docs 본문 Korean OK)
- **NO `Co-Authored-By` trailer** (사용자 명시 선호)
- **NO "Generated with Claude Code"**
- **NO emoji** in commits / code (요청 시에만)
- **사용자 한국어 입력 시 한국어로 응답**
- **작업 종료 시 uncommitted 변경사항이 있으면 사용자에게 commit 여부 확인 후 진행**
- **`git push` 는 사용자가 직접 결정** — 명시적 요청 없으면 push 하지 않음
- **Signer key material 절대 stdout/stderr/log 에 노출 금지** — `network/internal/signer` 의 sealed 패턴 강제
- **Memory 디렉토리에 프로젝트별 follow-up 저장 금지** — 프로젝트별 정보는 `docs/NEXT_WORK.md` / `docs/REMAINING_WORK.md` / spec-plan 파일에
