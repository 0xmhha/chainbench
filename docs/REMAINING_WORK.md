# Chainbench — 남은 작업 리스트

> 작성일: 2026-04-30 (Sprint 5a 완료 시점) · 최종 업데이트: 2026-06-29 (PR #1~#4 머지 — lifecycle reroute 완료 + clean-code/SSOT 리팩토링)
> 목적: 다른 세션에서 맥락 없이도 즉시 착수 가능한 actionable 핸드오프.
> 본 문서는 self-contained — `NEXT_WORK.md` (full context) / `VISION_AND_ROADMAP.md` (비전 SSoT) / `REFACTORING_PLAN.md` (clean-code/SSOT 트랙) 는 깊게 들어갈 때만.
>
> ⚠️ **2026-06 갱신**: 본 문서의 Sprint 5 시리즈 추적(§2/§4)은 2026-05-04(5c.4.1) 이후 멈춰 있었음. 그 사이 PR #1~#4 가 머지되어 **Sprint 5c.4.2 (lifecycle reroute) 가 사실상 완료**되었고, 별도의 **clean-code/SSOT 리팩토링 트랙**(`REFACTORING_PLAN.md`)이 추가됨. 아래 §2.0 참조.

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

**현재 진행 단계 (2026-05-04)**: Sprint 4 시리즈 (Go 능력 완비) 종료 → Sprint 5 시리즈 (MCP 노출 + 주변 인프라) 진행 중. Sprint 5c.1/5c.2 (high-level tool 노출) → 5c.3 (reroute pass 1) → 5a (capability gate) → 5c.4.1 (lifecycle reroute pass 1: stop + status) 까지 완료.

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

## 2.0 PR #1~#4 반영 (2026-06-29 — 최신)

5c.4.1 이후 머지된 작업. 본 §2/§4 의 옛 "다음 P1" 은 이미 처리됨:

| PR | 커밋 | 내용 | 상태 |
|---|---|---|---|
| #1 | `5a1d888` | **Sprint 5c.4.2 lifecycle reroute 완료** — Go wire 핸들러(init/start_all/restart/clean) + MCP reroute(init/start/restart) + report json contract fix | ✅ |
| #2 | `6ea996a` | local ↔ closed-net regression 테스트 환경 통합 | ✅ |
| #3 | `63f1d43` | clean-code/SSOT 리팩토링 P0+P1-1 + 사전 버그 2건 (mktemp 이식성, report zero-count) | ✅ |
| #4 | `2046b05` | command-schema drift fix(P1-2) + `chainbench_clean`(P1-2b) + `buildWireArgs`(P1-3) + defaults.json SSoT codegen(P1-4) + network_id 배선(P1-4b) | ✅ |

**lifecycle reroute 결과**: 6개 lifecycle MCP 툴(init/start/restart/stop/status/clean) **전부 callWire 경유** (lifecycle.ts 의 runChainbench 는 주석만). Sprint 5c.4.2 가 노렸던 reroute 5/38 → 9/38(~24%) 목표 달성.

**현재 테스트 상태(2026-06-29 직접 실행)**: Go 전 패키지 green · vitest 130 pass/2 skip · bash **37/39** (실패 2건 `lib-contract.sh`/`lib-event.sh` 는 `cast`(foundry) 미설치 환경 의존 — 코드 버그 아님, 설치 시 39/39).

**clean-code/SSOT 리팩토링 트랙**: `docs/REFACTORING_PLAN.md` 가 별도 추적. **P2-1 전체 완료** — P2-1a(profile.sh python 추출 + extends 버그)·P1-4a(stablenet chain_id SSoT, 2026-06-29) + P2-1b(json_helpers 단일화 + jq false-read 잠복버그, 2026-06-30) → CC-B1 닫힘. 남은 항목은 §6.2 참조 (P2-2 bash 분할, N2 cast, P1-4c=M4, N-A/N-B).

---

## 2. 빠른 상태 (2026-05-04 기준 — §2.0 으로 갱신됨)

**완료된 Sprint** (5 series):
- ✅ 5c.1 (2026-04-28) — MCP foundation + 첫 2 tool (`account_state`, `tx_send`)
- ✅ 5c.2 (2026-04-29) — MCP 남은 4 tool (`contract_deploy/call`, `events_get`, `tx_wait`) + tx_send 4-mode 완비 (legacy/1559/set_code/fee_delegation)
- ✅ 5c.3 (2026-04-29) — utils 추출 (mcpResp + hex) + Go `node.rpc` 핸들러 + 3 chainbench_node_* reroute (3/38 ≈ 8%) + real-binary integration test layer
- ✅ 5a (2026-04-29) — Capability gate (Go `network.capabilities` + MCP `chainbench_network_capabilities` + bash `requires_capabilities` frontmatter gating)
- ✅ 5c.4.1 (2026-05-04) — Lifecycle reroute pass 1 (stop + status thin Go wrappers) + integration harness `_harness.ts` 추출 + `node.start binary_path` Go 확장 (5c.3 fallback 제거)

**커버리지 지표**:
- EVALUATION_CAPABILITY MCP column: **~60%** (high-level tool 6종 + capability)
- Reroute 진행도: **5/38 (~13%)** (5c.3 의 3 + 5c.4.1 의 2)
- 테스트: vitest **100/100** · Go **16 packages** · bash **34/34** · 회귀 0

**다음 P1**: ~~5c.4.2~~ ✅ · ~~5d~~ ✅ · ~~5b.1/5b.2/5b.3~~ ✅ · ~~P2-1a/P2-1b~~ ✅ · ~~5b.4 (attach CLI/MCP 표면)~~ ✅ 완료(2026-06-30, `feat/sprint-5b-4-attach-surface`). Sprint 5 + CC-B1 + SSH 전체 완료 → 다음 후보 → `REFACTORING_PLAN.md` §6.2 (P2-2 bash 분할) 또는 5b 후속(키 인증, network detach/list).

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

### ✅ 완료 — Sprint 5c.4.2: Lifecycle reroute pass 2 (init/start/restart/clean) — PR #1 `5a1d888`

> **2026-06-29 갱신**: 본 sprint 는 PR #1 로 **완료**됨. Go wire 핸들러(init/start_all/restart/clean) + schema enum(P1-2 가 가드 테스트까지 추가) + MCP reroute(init/start/restart) + `chainbench_clean`(P1-2b) 모두 머지. 6개 lifecycle 툴 전부 callWire 경유. 아래 원본 계획은 **이력 보존용**.
>
> **다음 Priority 1 후보** = Sprint 5b.2 (SSH process/fs; 5b.1 완료) 또는 `REFACTORING_PLAN.md` §6.2.

<details><summary>원본 계획 (이력 보존)</summary>

**목표**: 잔여 4 lifecycle MCP tool (`chainbench_init/start/restart/clean`) 을 wire 경유로 전환. 5c.4.1 이 stop + status 만 reroute 했고 (2/6 완료), 잔여 4개는 init/start 의 ~600+ 줄 bash 로직을 가지고 있어 분리됨. 현재 `runChainbench` (bash CLI shell-out) 사용 중.

**5c.4.1 에서 완료된 항목** (이 sprint 에서 재작업 불필요):
- ✅ chainbench_stop reroute (`network.stop_all` thin Go wrapper)
- ✅ chainbench_status reroute (`network.status` thin Go wrapper)
- ✅ Integration test harness 추출 (`_harness.ts`)
- ✅ Integration test cleanup-await + port-race 진단 보강
- ✅ `chainbench_node_start binary_path` fallback 제거

**필요 작업**:
1. **Go-side wire 핸들러 4종 신규 작성** (chainbench-net 에 없음):
   - `network.init` — bash `chainbench init` 의 thin Go wrapper (`os/exec`) 또는 native Go 포팅 (Adapter Go 포팅이 3c 에 끝나서 native 도 가능)
   - `network.start_all` — 전체 노드 lifecycle thin wrapper of `chainbench.sh start`
   - `network.restart` — thin wrapper of `chainbench.sh restart`
   - `network.clean` — datadir / pids 초기화 thin wrapper
   - **schema enum 갱신 + `go generate ./...` 으로 `command_gen.go` 재생성 + 커밋**
2. **MCP 측 reroute** — `chainbench_init/start/restart/clean` 4 tool 을 callWire 경유 (`mcp-server/src/tools/lifecycle.ts`)
3. **`node.restart binary_path` symmetric override** — start 는 5c.4.1 에서 받지만 restart 는 hardcode `""`. symmetric 확장 (5c.4.1 P3 닫힘)
4. **`validateRpcMethod` duplicate in remote.ts 통합** (Sprint 5c.3 P3)
5. **`INTEGRATION_BEFOREALL_TIMEOUT_MS` (15s) 두 곳 중복** — 3rd integration test 추가 시 `_harness.ts` 로 추출 (5c.4.1 P3)
6. **Docs + version bump 0.7.1 → 0.8.0** (minor — 모든 lifecycle 6/6 reroute 완료 시점)

**완료 시 효과**: Reroute coverage 5/38 → **9/38 (~24%)**. lifecycle MCP 호출이 모두 wire 경유 → bash subprocess 지연 제거 + 일관 NDJSON 응답.

**예상 commit 수**: 8~10.

**첫 task 시작 템플릿**:
```
1. spec 작성: docs/superpowers/specs/YYYY-MM-DD-sprint-5c-4-2-lifecycle-reroute-pass-2.md
   - 5c.4.1 spec/plan 을 ref 로 (동일한 reroute 패턴, harness 재사용)
2. plan 작성: docs/superpowers/plans/...
3. Task 1-4 = Go 4 lifecycle 핸들러 (5c.4.1 의 stop_all / status 패턴 모방)
4. Task 5-8 = MCP reroute 4 tool
5. Task 9 = node.restart binary_path 확장 + remote.ts validateRpcMethod 통합
6. Task 10 = docs + version bump 0.8.0
```

</details>

---

### ✅ 완료 — Sprint 5b: SSHRemoteDriver (5b.1 + 5b.2)

> **5b.1 (2026-06-29, `feat/sprint-5b-1-sshremote`)**: read-only RPC over SSH 터널. `drivers/sshremote/` — SSH `DialContext`를 `http.Transport`에 주입해 기존 `remote.Client` 재사용(read 핸들러 무변경). `ssh-password` auth(스키마 `env` 필드, password env-only), host key known_hosts 기본 + `CHAINBENCH_SSH_INSECURE_HOST_KEY=1` opt-in.
>
> **5b.2 (2026-06-29, `feat/sprint-5b-2-sshremote-process-fs`)**: process/fs. `sshremote.Exec`(원격 명령) 추가. node stop/start/restart 가 `provider_meta` 의 `stop_cmd`/`start_cmd`/`restart_cmd` 를 SSH exec, tail_log 가 `log_file` 을 SSH `tail`. 미설정 명령/log_file → 런타임 NOT_SUPPORTED. providerCaps ssh-remote `{fs,process,rpc,ws}` 복원. in-process SSH 서버(터널+exec) 통합 테스트. 예제 `examples/networks/ssh-remote-example.json`(provider_meta 포함).
>
> **5b.3 완료 (2026-06-30, `feat/sprint-5b-3-ssh-attach`)**: `network.attach` 가 ssh-remote 구성 지원 — `provider:ssh-remote` + `provider_meta` + ssh-password auth, SSH 터널 경유 probe(`sshremote.DialTunnelClient` → `probe.Options.Client`)로 auto chain_id 감지. 수동 networks 파일 작성 불필요(wire 레벨).
>
> **5b.4 완료 (2026-06-30, `feat/sprint-5b-4-attach-surface`)**: `network.attach` 사용자 표면 — bash `chainbench network attach`(`lib/cmd_network.sh`) + MCP `chainbench_network_attach`. remote+ssh-remote 공통, 자격증명 env-var 이름만.
>
> **5b 후속 (잔여)**: 키 인증/OS 키체인(S6 후속), network 단위 start_all/stop_all 의 ssh-remote 반영, `network detach/list` + hybrid compose.
> spec/plan: `2026-06-29-sprint-5b-1-...` · `2026-06-29-sprint-5b-2-...`.

<details><summary>원본 목표 (이력)</summary>

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

</details>

---

### ✅ 완료 — Sprint 5d: Hybrid 네트워크 예제 (2026-06-29, `feat/sprint-5d-hybrid-example`)

**완료 내용**:
1. `examples/networks/hybrid-example.json` (local 3 + remote 1) + `examples/networks/README.md` (per-node provider 모델 + capability 하한 + 수동 구성 흐름 + remote auth env 참조). spec D1: profile YAML 이 아니라 **로드 가능한 network-state JSON** (profile 로더가 hybrid 미지원).
2. bash 실-바이너리 검증 `tests/unit/tests/network-hybrid-capabilities.sh` — 실 mixed-provider state 파일 → `network.capabilities` = `[rpc, ws]` 교집합 + process 게이팅 SKIP + rpc RUN.
3. MCP `network.test.ts:_Happy_Hybrid` — hybrid lower-bound passthrough.
4. Go `TestHandleNetworkCapabilities_Hybrid` (기존) 유지.

**후속 (Non-goal)**: 전용 구성 명령 `network attach-hybrid`/compose (local pids.json + remote 를 단일 networks 파일로 합성) — 별도 sprint. profile 로더 hybrid 지원은 P2-1 이후.

**참고 spec/plan**: `docs/superpowers/{specs,plans}/2026-06-29-sprint-5d-hybrid-network-example.md`.

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

### 5c.4.1 에서 닫힌 5c.3 P3
- ✅ `chainbench_node_start binary_path` fallback 제거 (Task 5)
- ✅ Integration test harness 추출 (`_harness.ts`) (Task 0)
- ✅ Integration test cleanup-await + port-race 진단 (Task 0 fix)

### 5c.4.2 lifecycle 작업 시 자연 흡수
- `node.restart binary_path` symmetric override (5c.4.1 P3)
- `INTEGRATION_BEFOREALL_TIMEOUT_MS` (15s) 두 곳 중복 → `_harness.ts` 로 추출 (5c.4.1 P3, 트리거: 3rd integration test 추가 시)
- `chain_tx.ts` 505 lines per-tool 분할 검토 (5c.2)
- `chainbench-net` 바이너리 staleness 진단 — `_harness.ts.hasBinary()` 가 mtime vs source compare (5c.4.1 P3, 트리거: integration test 작성 시 사전 체크)
- `lifecycle.test.ts` assertion-style 비대칭 (stop=JSON pretty form / status=field name `toContain`) → spec/plan 다음 갱신 시 1줄 rationale (5c.4.1 P3)

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
| `chainbench-net network.init` wire handler | Sprint 5c.4.2 의 일부로 흡수 |
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

### 8.9 macOS fork-EAGAIN under load (2026-05-04 발견)
- 장시간 세션 + 다수 subagent spawn 시 macOS user fork limit 도달 → `npm test` / shell startup 실패 (`Resource temporarily unavailable`)
- vitest fallback: `npm test -- --pool=forks --poolOptions.forks.singleFork=true`
- Go fallback: `go test -p=2 ./...`
- 새 세션 시작 시 회복

### 8.10 `chainbench-net` binary staleness
- `_harness.ts.hasBinary()` 가 mtime 검사 안 함 — 존재만 확인
- Go 변경 후 binary rebuild 필수: `cd network && go build -o bin/chainbench-net ./cmd/chainbench-net`
- 5c.4.1 Task 4 에서 발견 — 구 binary 가 새 wire command 거절하는 PROTOCOL_ERROR 발생
- 5c.4.1 P3 — 5c.4.2+ 에서 mtime 비교 추가 가치

### 8.11 Lifecycle 명령은 local 전용 (Sprint 5c.4.1 에서 정착)
- `network.stop_all` / `network.status` / (5c.4.2 의) init/start_all/restart/clean 모두 `network != "local"` → NOT_SUPPORTED
- Remote 네트워크는 노드 lifecycle 보유 안 함 — wire 핸들러가 boundary 에서 거절

### 8.12 Schema enum 알파벳 순 (Sprint 5c.4.1 review 에서 확립)
- `network/schema/command.json` 에 새 명령 추가 시 `network.<verb>` / `node.<verb>` 그룹 내 알파벳 순
- 5c.4.1 Task 3 review 가 catch — Task 1+3 가 `stop_all` 을 `status` 보다 먼저 추가 → reverse-alphabetical → `c16350c` 에서 swap

---

## 9. 검증된 패턴 코드 템플릿 (Sprint 5c.4.1 기준)

5c.4.2 / 5b / 5d 의 새 sprint 가 직접 카피해서 변형하면 됨. 모두 100/100 vitest + Go 16 packages green 으로 검증된 패턴.

### 9.1 Go thin-wrapper handler (lifecycle reroute 용)

`network/cmd/chainbench-net/handlers_network.go` 의 `newHandleNetworkStopAll` (`be7fd7c` 기준) 참조. 5c.4.2 의 init/start/restart/clean 핸들러는 다음 템플릿을 카피하여 변형:

```go
const <verb>Timeout = 30 * time.Second   // file scope const, 일관성 (stop_all / status 와 동일)

// newHandleNetwork<Verb> returns the "network.<verb>" handler.
//
// Args: { "network"?: "local" }  (default + non-local rejected with NOT_SUPPORTED)
//       (init only) { "profile": "<name>", "binary_path"?: "/abs/path" }
//
// Result: { "network": "local", "stdout": "<bash output>" }   ← stop_all pattern
//         OR parsed JSON map                                    ← status pattern
//
// Error mapping:
//   INVALID_ARGS    — args parse / shape failure
//   NOT_SUPPORTED   — non-local network
//   UPSTREAM_ERROR  — bash exit non-zero (with stderr in cause)
//   INTERNAL        — JSON parse failure (status only — invariant)
func newHandleNetwork<Verb>(stateDir, chainbenchDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network    string  `json:"network"`
            // (init only)
            // Profile    string  `json:"profile"`
            // BinaryPath *string `json:"binary_path"`   // *string for absent-vs-empty discrimination (5c.4.1 Task 5 fix)
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.Network == "" { req.Network = "local" }
        if req.Network != "local" {
            return nil, NewNotSupported(fmt.Sprintf(
                "network.<verb> only operates on the local network; got %q", req.Network))
        }
        _ = stateDir // future native-port reads pids.json from here

        // Shell-injection-safe: literal argv via execve (NOT bash -c "<concat>").
        // (init only) ensure profile is regex-validated upstream (zod) and pass via separate argv slot:
        //   cmdArgs := []string{"init", "--profile", req.Profile, "--quiet"}
        //   if req.BinaryPath != nil { cmdArgs = append(cmdArgs, "--binary-path", *req.BinaryPath) }

        ctx, cancel := context.WithTimeout(context.Background(), <verb>Timeout)
        defer cancel()
        cmd := exec.CommandContext(ctx, filepath.Join(chainbenchDir, "chainbench.sh"),
            "<verb>", "--quiet")
        cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)

        // For status-style (JSON output): use cmd.Output() not CombinedOutput()
        // and use errors.As(err, &exitErr) to extract exitErr.Stderr for the diagnostic.
        out, err := cmd.CombinedOutput()
        if err != nil {
            return nil, NewUpstream("chainbench <verb>",
                fmt.Errorf("%w: %s", err, string(out)))
        }

        // For status: json.Unmarshal(out, &result); INTERNAL on parse fail
        return map[string]any{"network": "local", "stdout": string(out)}, nil
    }
}
```

### 9.2 MCP reroute (lifecycle 용)

`mcp-server/src/tools/lifecycle.ts` 의 `StopArgs/_stopHandler` (`4ef8f13`) 참조:

```typescript
export const <Verb>Args = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'. Remote networks reject."),
  // (init only) profile: z.string().regex(/^[a-zA-Z0-9_\-/]+$/),
  // (init/start/restart) binary_path: z.string().optional(),
}).strict();

export async function _<verb>Handler(
  args: z.infer<typeof <Verb>Args>,
): Promise<FormattedToolResponse> {
  // (binary_path 있는 핸들러만) boundary validation:
  // if (args.binary_path !== undefined) {
  //   if (args.binary_path.length === 0) return errorResp("binary_path must not be empty");
  //   if (!args.binary_path.startsWith("/")) return errorResp("binary_path must be an absolute path");
  // }
  const wireArgs: Record<string, unknown> = {};
  if (args.network !== undefined) wireArgs.network = args.network;
  // if (args.profile !== undefined) wireArgs.profile = args.profile;
  // if (args.binary_path !== undefined) wireArgs.binary_path = args.binary_path;
  const result = await callWire("network.<verb>", wireArgs);
  return formatWireResult(result);
}

server.tool(
  "chainbench_<verb>",
  "<description — local-only, what bash does, what return shape is>",
  <Verb>Args.shape,
  _<verb>Handler,
);
```

### 9.3 Vitest unit test (3 tests per tool — `lifecycle.test.ts`)

```typescript
describe("chainbench_<verb> handler", () => {
  let savedBin: string | undefined;
  let savedScript: string | undefined;
  let savedDir: string | undefined;

  beforeEach(() => {
    savedBin = process.env.CHAINBENCH_NET_BIN;
    savedScript = process.env.MOCK_SCRIPT;
    savedDir = process.env.CHAINBENCH_DIR;
    delete process.env.CHAINBENCH_DIR;
  });

  afterEach(() => {
    if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
    else process.env.CHAINBENCH_NET_BIN = savedBin;
    if (savedScript === undefined) delete process.env.MOCK_SCRIPT;
    else process.env.MOCK_SCRIPT = savedScript;
    if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
    else process.env.CHAINBENCH_DIR = savedDir;
  });

  it("_<Verb>Happy", async () => {
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    process.env.MOCK_SCRIPT = script([{
      kind: "stdout",
      line: JSON.stringify({type:"result", ok:true, data:{network:"local", stdout:"<verb> done"}}),
    }]);
    const out = await _<verb>Handler({});
    expect(out.content[0]?.text ?? "").toContain("<verb> done");
    expect(out.isError).toBeFalsy();
  });

  it("_<Verb>StrictRejectsUnknownKeys", () => {
    expect(() => <Verb>Args.parse({network:"local", extra:"bar"} as any)).toThrow();
  });

  it("_<Verb>WireFailure_PassedThrough", async () => {
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    process.env.MOCK_SCRIPT = script([{
      kind: "stdout",
      line: JSON.stringify({type:"result", ok:false, error:{code:"UPSTREAM_ERROR", message:"chainbench <verb> failed"}}),
    }]);
    const out = await _<verb>Handler({});
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text ?? "").toContain("UPSTREAM_ERROR");
  });
});
```

### 9.4 Real-binary integration test (`lifecycle.integration.test.ts`)

```typescript
describe.skipIf(!hasBinary())("integration: chainbench_<verb>", () => {
  let harness: RealBinaryHarness;
  beforeAll(async () => {
    const fakeScript = `#!/bin/bash
if [[ "$1" == "<verb>" && "$2" == "--quiet" ]]; then
  echo "fake <verb>: success"
  exit 0
fi
exit 1
`;
    harness = await setupRealBinaryHarness({ fakeChainbenchScript: fakeScript });
  }, 15000);  // 5c.4.1 P3 — 3rd integration test 시 _harness.ts 의 INTEGRATION_BEFOREALL_TIMEOUT_MS 로 추출

  afterAll(() => harness?.teardown());

  it("end-to-end <verb> through chainbench-net", async () => {
    const { _<verb>Handler } = await import("../../src/tools/lifecycle.js");
    const out = await _<verb>Handler({});
    expect(out.content[0]?.text ?? "").toContain("success");
    expect(out.isError).toBeFalsy();
  });
});
```

### 9.5 Go unit test (4 tests per handler — `handlers_test.go`)

```go
// 헬퍼 (5c.4.1 Task 1 에서 정착, Task 3 가 stdout-emitting 변형 추가)
func writeFakeChainbench(t *testing.T, dir string, exitCode int) {
    t.Helper()
    script := fmt.Sprintf("#!/bin/bash\necho \"fake chainbench: $@\"\nexit %d\n", exitCode)
    p := filepath.Join(dir, "chainbench.sh")
    if err := os.WriteFile(p, []byte(script), 0755); err != nil {
        t.Fatalf("write fake script: %v", err)
    }
}
// (status-like 용) writeFakeChainbenchWithStdout(t, dir, exitCode, stdout) 도 sibling 으로 존재

func TestHandleNetwork<Verb>_Happy(t *testing.T) {
    chainbenchDir := t.TempDir()
    writeFakeChainbench(t, chainbenchDir, 0)
    handler := newHandleNetwork<Verb>(t.TempDir(), chainbenchDir)
    bus, _ := newTestBus(t)
    defer bus.Close()
    data, err := handler(json.RawMessage(`{}`), bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if data["network"] != "local" { t.Errorf("network=%v", data["network"]) }
    if !strings.Contains(data["stdout"].(string), "fake chainbench") {
        t.Errorf("stdout missing expected substring: %v", data["stdout"])
    }
}

func TestHandleNetwork<Verb>_RemoteRejected(t *testing.T) {
    handler := newHandleNetwork<Verb>(t.TempDir(), t.TempDir())
    bus, _ := newTestBus(t); defer bus.Close()
    args, _ := json.Marshal(map[string]any{"network": "sepolia"})
    _, err := handler(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
        t.Fatalf("want NOT_SUPPORTED, got %v", err)
    }
    if !strings.Contains(api.Message, "only operates on the local network") {
        t.Errorf("message=%q", api.Message)
    }
}

func TestHandleNetwork<Verb>_BashFailure(t *testing.T) {
    chainbenchDir := t.TempDir()
    writeFakeChainbench(t, chainbenchDir, 1)  // exit 1
    handler := newHandleNetwork<Verb>(t.TempDir(), chainbenchDir)
    bus, _ := newTestBus(t); defer bus.Close()
    _, err := handler(json.RawMessage(`{}`), bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
        t.Fatalf("want UPSTREAM_ERROR, got %v", err)
    }
}

func TestAllHandlers_IncludesNetwork<Verb>(t *testing.T) {
    h := allHandlers("x", "y")
    if _, ok := h["network.<verb>"]; !ok {
        t.Errorf("allHandlers missing network.<verb>")
    }
}
```

### 9.6 `*string` discrimination for optional-string args (5c.4.1 Task 5 fix)

caller-controlled optional string 인자 (예: `binary_path`) 를 받을 때 "key absent" vs "key present + empty" 를 구분해야 하면 `*string` 사용:

```go
// ❌ 잘못된 idiom (substring scan on raw JSON — false-positive risk):
//   if bytes.Contains(args, []byte(`"binary_path"`)) && pre.BinaryPath == "" { ... }
//   → unrelated key 의 value 에 "binary_path" 문자열 들어가도 발동

// ✅ 올바른 idiom (*string pointer):
var pre struct {
    BinaryPath *string `json:"binary_path"`
}
_ = json.Unmarshal(args, &pre)
// nil  → key 부재 (profile default 사용)
// non-nil → key 있음
if pre.BinaryPath != nil && *pre.BinaryPath == "" {
    return nil, NewInvalidArgs("args.binary_path must not be empty")
}
if pre.BinaryPath != nil && !strings.HasPrefix(*pre.BinaryPath, "/") {
    return nil, NewInvalidArgs(fmt.Sprintf("args.binary_path must be absolute: %q", *pre.BinaryPath))
}
var binaryPath string
if pre.BinaryPath != nil {
    binaryPath = *pre.BinaryPath
}
```

회귀 방지 테스트 추가 권장 (5c.4.1 의 `TestHandleNodeStart_UnrelatedKeyContainingBinaryPathSubstring` 참조 — unrelated key with substring → no false positive).

### 9.7 Schema enum 추가 + 재생성

```bash
# 1. network/schema/command.json 의 enum 에 알파벳 순 위치에 추가
#    (network.* 그룹 내, node.* 그룹 내)
$EDITOR network/schema/command.json

# 2. 재생성
cd network && go generate ./...
# → command_gen.go 에 `CommandCommandNetwork<Verb>` const + enumValues_CommandCommand entry 추가됨

# 3. 같은 commit 에 schema + 재생성 결과 모두 포함
git add network/schema/command.json network/internal/types/command_gen.go
```

---

## 10. 다음 세션에서 즉시 착수 가능한 단위

**큰 단위**: Sprint 5b (SSH driver) — 5b.1 (read-only RPC) ✅ + 5b.2 (process/fs over SSH shell exec) ✅ 완료. 후속: 구성 명령/키 인증.

**권장 순서** (2026-06-29 갱신 — 5c.4.2 PR #1, 5d 완료):
1. ~~**Sprint 5c.4.2**~~ ✅ 완료 (PR #1).
2. ~~**Sprint 5d**~~ ✅ 완료 (`feat/sprint-5d-hybrid-example`).
3. ~~**Sprint 5b** (5b.1 read-only RPC + 5b.2 process/fs)~~ ✅ 완료. 다음 후보 → `REFACTORING_PLAN.md` §6.2 (P2-1) 또는 5b 후속(구성 명령/키 인증).
4. (병행) `REFACTORING_PLAN.md` §6.2 리팩토링 잔여 — ~~P2-1a~~ ✅ · ~~P2-1b(json_helpers)~~ ✅. 남음: P2-2(bash 대형 파일 분할).

---

## 11. 새 세션 재개 명령 (체크리스트)

```bash
# 1. 현재 상태 확인 (경로는 환경마다 다름 — repo 루트로 이동)
cd "$(git rev-parse --show-toplevel)"
git log --oneline -15
git status

# 2. 회귀 테스트 (모두 green 이어야 함)
go -C network test ./... -count=1 -timeout=60s   # 16 packages
bash tests/unit/run.sh                            # 34/34
cd mcp-server && npm test                         # 100/100
cd .. && cd mcp-server && npx tsc --noEmit && npm run build && cd ..

# 3. (선택) chainbench-net binary 신선도 확인 — Go 변경 후 필수 rebuild (§8.10)
ls -l network/bin/chainbench-net 2>&1   # mtime 확인
# 최근 Go 변경 후 안 쌓였으면 (또는 처음 세션이면):
go -C network build -o bin/chainbench-net ./cmd/chainbench-net

# 4. 다음 작업 결정
cat docs/REMAINING_WORK.md              # 본 문서 — 가장 빠른 핸드오프
cat docs/NEXT_WORK.md                    # 전체 핸드오프 (자세한 컨텍스트, 트러블 발생 시)
cat docs/VISION_AND_ROADMAP.md           # SSoT 비전 + 로드맵 (디자인 결정 source)

# 5. Sprint 진행 시 참고 spec/plan
ls docs/superpowers/specs/ | sort | tail -10     # 최근 spec 목록
ls docs/superpowers/plans/ | sort | tail -10     # 최근 plan 목록

# 6. 코드 진입점 (3분 읽기로 모듈 이해)
# - Utils & 인프라
cat mcp-server/src/utils/wire.ts                              # spawn + NDJSON helper
cat mcp-server/src/utils/wireResult.ts                        # NDJSON → MCP transformer
cat mcp-server/src/utils/mcpResp.ts                           # FormattedToolResponse + errorResp
cat mcp-server/src/utils/hex.ts                               # HEX_*, SIGNER_ALIAS, RPC_METHOD
cat mcp-server/test/integration/_harness.ts                   # real-binary integration 인프라
# - MCP tool 패턴
cat mcp-server/src/tools/lifecycle.ts                         # 5c.4.1 reroute 패턴 (stop+status, init/start/restart 미rerouted)
cat mcp-server/src/tools/chain_read.ts                        # read-only MCP tool 패턴
cat mcp-server/src/tools/chain_tx.ts                          # write MCP tool 패턴 (4-mode tx_send)
cat mcp-server/src/tools/node.ts                              # node-level reroute + binary_path
# - Go handlers
cat network/cmd/chainbench-net/handlers_network.go            # network.* (5c.4.1 thin-wrapper 포함)
cat network/cmd/chainbench-net/handlers_node_lifecycle.go     # node.start (binary_path *string), stop, restart
cat network/cmd/chainbench-net/handlers_node_read.go          # node.rpc / contract_call / events_get / account_state / tx_wait
cat network/cmd/chainbench-net/handlers_node_tx.go            # node.tx_send / contract_deploy / fee_delegation
cat network/internal/drivers/local/start.go                   # LocalDriver.StartNode (binary_path 인자)
cat network/internal/signer/signer.go                         # 서명 경계 + redaction 패턴
# - bash 진입점
cat lib/network_client.sh                                     # bash → Go 바이너리 NDJSON 브리지
cat lib/cmd_test.sh                                           # bash test runner + capability gating (5a)
```

---

## 12. 문서 구조 — 본 문서 vs 형제 문서

세 문서가 보완 관계로 상호 의존:

| 문서 | 분량 | 역할 |
|---|---|---|
| **REMAINING_WORK.md** (본 문서) | ~620줄 | Actionable TODO + 30초 컨텍스트 + 빠른 시작 + 검증된 패턴 코드 템플릿. **새 세션 즉시 참조**. |
| **NEXT_WORK.md** | ~700줄 | 전체 핸드오프 — 디렉토리 레이아웃, 규약 풀버전, 주의사항 풀버전, 모든 sprint 별 P3 표 (closed + open). **트러블 발생 시 깊게 참조**. |
| **VISION_AND_ROADMAP.md** | ~870줄 | 비전 + 로드맵 SSoT. 디자인 결정의 source (Q1~Q6, S1~S8, §5.4 Provider Interface, §5.12 M2 LocalDriver-wraps-bash, §5.17 Go module 상세). **디자인 변경 시 갱신**. |

**갱신 시점**:
- 매 sprint 의 docs+chore 커밋에서 NEXT_WORK + VISION 갱신
- Sprint 종료 시점에 본 문서 (REMAINING_WORK) 갱신
- 본 문서가 stale 해 보이면 NEXT_WORK §2.1 timeline 의 가장 최근 row 와 본 문서 §2 의 "완료된 Sprint" 비교

**spec/plan 파일** (sprint 진행 시 참고):
- `docs/superpowers/specs/YYYY-MM-DD-<topic>.md` — 디자인 spec
- `docs/superpowers/plans/YYYY-MM-DD-<topic>.md` — task 단위 implementation plan
- 매 sprint 종료 후 보존 — 미래 sprint 가 패턴 참고용으로 사용

---

## 13. 보안 / 사용자 규칙 핵심 (다시 강조)

- **English commit messages** (docs 본문 Korean OK)
- **NO `Co-Authored-By` trailer** (사용자 명시 선호)
- **NO "Generated with Claude Code"**
- **NO emoji** in commits / code (요청 시에만)
- **사용자 한국어 입력 시 한국어로 응답**
- **작업 종료 시 uncommitted 변경사항이 있으면 사용자에게 commit 여부 확인 후 진행**
- **`git push` 는 사용자가 직접 결정** — 명시적 요청 없으면 push 하지 않음
- **Signer key material 절대 stdout/stderr/log 에 노출 금지** — `network/internal/signer` 의 sealed 패턴 강제
- **Memory 디렉토리에 프로젝트별 follow-up 저장 금지** — 프로젝트별 정보는 `docs/NEXT_WORK.md` / `docs/REMAINING_WORK.md` / spec-plan 파일에
- **Shell injection 회피** — bash spawn 시 args 는 execve literal argv 로 (`exec.CommandContext(scriptPath, "init", "--profile", profile)`), `bash -c "<concat>"` 금지. Caller-controlled args 는 zod regex 로 boundary 검증 (예: `profile: z.string().regex(/^[a-zA-Z0-9_\-/]+$/)`)

---

## 14. Sprint commit chain 빠른 참조

`origin/main` 대비 **11 commits ahead** (Sprint 5c.4.1 전체 — 사용자 push 결정 대기).

```
Sprint 5c.4.1 (be7fd7c, 2026-05-04, 11 commits, 0.7.1):
  6a790bd  docs: spec + plan
  297918c  refactor(test): _harness.ts extract (Task 0)
  d902cca  fix(test): tighten profileOverride + extract kill helper (Task 0 review)
  2ada02b  feat(network-net): network.stop_all (Task 1)
  4ef8f13  feat(mcp): reroute chainbench_stop (Task 2)
  5854d29  feat(network-net): network.status (Task 3)
  c16350c  chore(schema): swap stop_all/status enum order (Task 3 review)
  ad32ff8  feat(mcp): reroute chainbench_status + integration test (Task 4)
  3150719  feat(network-net+mcp): node.start binary_path + remove fallback (Task 5)
  5702604  fix(network-net): *string discrimination for binary_path (Task 5 review)
  be7fd7c  docs+chore(sprint-5c-4-1): roadmap + version 0.7.1 (Task 6)

Sprint 5a (bd7284e, 2026-04-29, 8 commits, 0.7.0):
  39b69aa..bd7284e — capability gate (Go + MCP + bash) + frontmatter PoC

Sprint 5c.3 (9084c4d, 2026-04-29, 8 commits, 0.6.0):
  e9ddf2d..9084c4d — utils 추출 + node.rpc + 3 chainbench_node_* reroute + integration test layer

Sprint 5c.2 (17f6e6f, 2026-04-29, 11 commits, 0.5.0):
  90a299d..17f6e6f — MCP 4 high-level tool + tx_send 4-mode + chain.ts split

Sprint 5c.1 (31e4bf9, 2026-04-28, 8 commits, 0.4.0):
  b35eb67..31e4bf9 — MCP foundation + 첫 2 high-level tool

Sprint 4 series (10 commits 누적, 4 → 4b → 4c → 4d, 2026-04-24~04-27):
  e9ef018 이전 — Go network/ tx + read 매트릭스 100% 도달
```

**Sprint 종료마다 docs+chore(sprint-X) 커밋 확인** — VISION + NEXT_WORK + REMAINING_WORK + EVALUATION_CAPABILITY + version 모두 동기화.
