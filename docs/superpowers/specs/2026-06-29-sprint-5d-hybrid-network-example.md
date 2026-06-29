# Sprint 5d — Hybrid Network Example

> 작성일: 2026-06-29
> 상태: SPEC (검토 대기)
> 선행: Sprint 5a (capability gate — `network.capabilities` Go/MCP/bash 3-layer), VISION §5.4–5.8 (per-node provider dispatch), `docs/REMAINING_WORK.md` §4 Priority 3
> 짝 plan: `docs/superpowers/plans/2026-06-29-sprint-5d-hybrid-network-example.md`

---

## 1. Goal

Hybrid 네트워크(노드별 provider가 섞인 단일 네트워크 — 예: local 3 + remote 1)를 **문서화된·검증된 사용 시나리오**로 만든다. 데이터 모델·capability 추론은 Sprint 5a 에서 이미 완성됐고(아래 §3), 빠진 것은 **사용자용 예제 + 사용 흐름 문서 + cross-layer 검증 테스트**뿐이다.

핵심 계약(이미 구현, 본 sprint 가 데모): hybrid 네트워크의 capability 는 **모든 노드 provider capability 의 교집합(보수적 하한)** 이다. local(`admin/fs/network-topology/process/rpc/ws`) ∩ remote(`rpc/ws`) = **`rpc, ws`**. 즉 process/fs 를 요구하는 fault/lifecycle 작업은 hybrid 에서 자동으로 게이팅된다.

---

## 2. Non-goals (의도적 제외 — 정직하게)

- **새 hybrid 생성 명령/핸들러 추가 안 함.** 현재 `network.attach`(순수 remote 단일 노드)·`remote add`(remotes.json 레지스트리)·profile(local XOR remote 단일 모드) 어디에도 hybrid 를 구성하는 사용자 경로가 없다. 본 sprint 는 **수동 구성(예제 JSON → `state/networks/<name>.json` 복사)을 v1 흐름으로 문서화**하고, 전용 구성 명령(`network attach-hybrid` / compose)은 **후속 sprint 로 명시 연기**한다 (§9).
- **`profile.sh` / profile 로더 확장 안 함.** profile 포맷은 단일 모드(local 또는 remote)다. hybrid 를 표현하도록 로더를 고치는 것은 (a) "작은 sprint, infra 존재" 범위를 벗어나고 (b) P2-1(임베디드 Python 추출, 고위험)이 손댈 파일을 미리 건드린다. 따라서 committed 예제는 **profile YAML 이 아니라 network-state JSON**(실제 로드 가능한 포맷)으로 제공한다 (§6 결정).
- **live hybrid 네트워크 대상 tx/fault 실행 검증 안 함.** CI 에 실 remote 노드가 없으므로, 검증은 state-file 기반 capability 게이팅(결정론)으로 한정한다.

---

## 3. 이미 존재하는 것 (본 sprint 가 재사용)

| 레이어 | 자산 | 위치 |
|---|---|---|
| Schema | per-node `provider ∈ {local,remote,ssh-remote}`, mixed 허용, homogeneity 제약 없음 | `network/schema/network.json` |
| Fixture | hybrid 예시 (name `mixed`, local 3 + remote 1) | `network/schema/fixtures/network-hybrid.json` |
| State I/O | `SaveRemote`/`loadRemote` provider 무제약 수용 | `network/internal/state/{remote,network}.go` |
| Capability 추론 | `inferNetworkCapabilities` = provider cap 집합 교집합 | `network/cmd/chainbench-net/handlers_network.go:41` |
| Go 핸들러 테스트 | `TestHandleNetworkCapabilities_Hybrid` (local∩remote=`[rpc,ws]`) | `handlers_test.go:1180` |
| bash 표면 | `cb_net_call "network.capabilities"` + `requires_capabilities` frontmatter 게이팅 | `lib/cmd_test.sh:84-114` |
| MCP 표면 | `chainbench_network_capabilities` | `mcp-server/src/tools/network.ts` |

→ **추가 런타임 코드 0. 본 sprint 는 예제 파일 + 문서 + 테스트 추가만.**

---

## 4. User-Facing Surface (이번에 추가)

1. **예제 파일** — `examples/networks/hybrid-example.json` (committed, 로드 가능한 network-state 포맷). local 3 + remote 1, 플레이스홀더 URL.
2. **예제 README** — `examples/networks/README.md`. per-node provider 모델 1단락 + "이 파일을 `state/networks/<name>.json` 로 복사 → `chainbench-net network.capabilities` / MCP `chainbench_network_capabilities` 가 `[rpc, ws]` 반환" 사용 흐름 + capability 하한 의미 + remote 노드 auth(`env` 참조) 주의.
3. **문서 갱신** — VISION 로드맵 Sprint 5d 체크박스 ✅, REMAINING_WORK §4 Priority 3 완료 처리.

---

## 5. Tests (검증 핵심 — 본 sprint 의 실가치)

세 레이어에서 "hybrid → capability 하한 + 게이팅"을 end-to-end 로 못박는다:

1. **Go (이미 존재)** — `TestHandleNetworkCapabilities_Hybrid` 유지. 핸들러 레벨 교집합.
2. **bash 신규** — `tests/unit/tests/network-hybrid-capabilities.sh`:
   - `CHAINBENCH_STATE_DIR/networks/hybrid-demo.json` 에 hybrid state 기록(예제 파일 복사 또는 fixture 재사용).
   - `cb_net_call "network.capabilities" '{"network":"hybrid-demo"}'` → `.capabilities == ["rpc","ws"]` 단언.
   - capability 게이팅 데모: `requires_capabilities: [process]` 를 가진 더미 test meta 에 대해 `_cb_test_check_capabilities` 가 hybrid 활성 시 **SKIP** 판정하는지 단언 (process 가 하한에 없음).
3. **MCP 신규** — `mcp-server/test/network.test.ts` 에 케이스 추가: mock wire 가 hybrid state 를 반영할 때 `chainbench_network_capabilities` 가 `rpc, ws` 만 반환(하한) 검증. (기존 network.test.ts 패턴 따름.)

회귀: Go 전 패키지 · vitest · bash 모두 green 유지.

---

## 6. 결정 (Decisions)

- **D1 — committed 예제는 `.json`(network-state), `.yaml`(profile) 아님.** profile 로더가 hybrid 를 소비 못 하므로 `profiles/hybrid-example.yaml` 은 "로드되는 것처럼 보이지만 안 되는" 오해를 낳는다. 실제 로드 경로(`state/networks/<name>.json`)와 동일 포맷의 예제가 정직하다. 원래 task 명("profiles/hybrid-example.yaml")에서 의도적 이탈 — REMAINING_WORK 에 사유 기록.
- **D2 — 예제 위치 `examples/networks/`.** `state/` 는 런타임(gitignore), `network/schema/fixtures/` 는 스키마 테스트용(비발견적). 사용자 발견성을 위해 신규 `examples/` 트리.
- **D3 — 수동 구성이 v1 흐름.** 전용 구성 명령은 §9 후속. 본 sprint 는 "할 수 있다 + 어떻게 + 게이팅이 옳게 동작한다"를 증명.

---

## 7. Security Contract

- 예제는 플레이스홀더 URL(`https://devnet.example.com` 등)만. 시크릿/키 material 없음.
- remote 노드 auth 는 기존 `Auth` 스키마(`type: api-key|jwt`, `env` 참조)로 문서화 — 헤더 값 자체를 파일에 넣지 않는다(env var 이름만). VISION §5.14 / signer redaction 경계 불변.

---

## 8. Error Classification (변경 없음)

기존 `network.capabilities` 매핑 유지: 잘못된 name → `INVALID_ARGS`, 누락 state 파일 → `UPSTREAM_ERROR`. 본 sprint 는 새 에러 경로 도입 없음.

---

## 9. Out-of-Scope / 후속 (명시)

- **Sprint 5d-follow (구성 명령)**: `network attach-hybrid <local-net> <remote-alias>` 또는 `network compose` — 기존 local(pids.json) + remote(remotes.json/attach)를 단일 `networks/<name>.json` 으로 합성. Go 핸들러 + schema enum + bash/MCP 표면 필요(중간 규모).
- **profile 로더 hybrid 지원**: P2-1(profile.sh Python 추출) 이후 재평가.
- **live hybrid e2e**: 실 remote 노드 가용 환경에서 read-only test 통과 검증.

---

## 10. 예상 커밋 (~4-5)

1. `docs: add Sprint 5d spec + plan for hybrid network example`
2. `feat(examples): add hybrid network example + README`
3. `test(network): bash hybrid capability + gating e2e`
4. `test(mcp): chainbench_network_capabilities hybrid lower bound`
5. `docs+chore(sprint-5d): roadmap + REMAINING_WORK + version bump`
