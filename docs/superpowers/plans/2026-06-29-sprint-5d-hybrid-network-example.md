# Sprint 5d — Hybrid Network Example — Plan

> 작성일: 2026-06-29
> 짝 spec: `docs/superpowers/specs/2026-06-29-sprint-5d-hybrid-network-example.md`
> 범위 확정: **예제 + 문서 + cross-layer 검증만** (구성 명령 후속 연기). 추가 런타임 코드 0.
> 실행 방식: 직접 구현(소규모) + 전 레이어 테스트로 cross-cutting 검증 (NEXT_WORK §3 fallback).

---

## 검증된 사전 사실 (조사 완료)

- `network.capabilities` 는 **RPC dial 없는 순수 state-file read** → bash 테스트에 JSON-RPC mock 불필요.
- 바이너리는 state dir 을 `CHAINBENCH_STATE_DIR` env 로 받음(e2e 패턴). hybrid state 는 `${CHAINBENCH_STATE_DIR}/networks/<name>.json`.
- `inferNetworkCapabilities` = provider cap 교집합. local∩remote = `[rpc, ws]`.
- bash 게이팅: `_cb_test_check_capabilities` → `cb_net_call "network.capabilities"` → `requires_capabilities` 비교 (`lib/cmd_test.sh:102`).
- MCP hybrid 는 remote 와 동일 응답(lower bound passthrough) → 기존 `network.test.ts:_Happy_Remote` 가 사실상 커버. 신규 MCP 케이스는 hybrid 의도 명시용 최소 1개.

---

## Task 0 — spec + plan 커밋

- `git add docs/superpowers/specs/2026-06-29-*.md docs/superpowers/plans/2026-06-29-*.md`
- commit: `docs: add Sprint 5d spec + plan for hybrid network example`

---

## Task 1 — 사용자용 hybrid 예제 + README

**파일**:
- `examples/networks/hybrid-example.json` — local 3 + remote 1, network.json 스키마 준수, 플레이스홀더 URL. 기존 fixture(`network/schema/fixtures/network-hybrid.json`) 내용을 기준으로 하되 `role` 포함하고 remote 노드에 auth(env 참조) 예시 주석을 README 로 분리.
- `examples/networks/README.md` — (a) per-node provider 모델 1단락, (b) capability 하한 = 모든 노드 provider cap 의 교집합(local∩remote=`rpc,ws`) 설명, (c) **사용 흐름**: `cp examples/networks/hybrid-example.json state/networks/<name>.json` → `chainbench-net network.capabilities {"network":"<name>"}` (또는 MCP `chainbench_network_capabilities`) → `[rpc, ws]`, (d) remote auth 는 `Auth.type=api-key|jwt` + `env` var 이름만(헤더 값 파일에 금지), (e) v1 은 수동 구성(전용 명령 후속) 명시.

**검증**: `python3 -c "import json,sys; json.load(open('examples/networks/hybrid-example.json'))"` 파싱 OK + 스키마 필수필드(name/chain_type/chain_id/nodes[].{id,provider,http}) 충족. 가능하면 schema validator 로 검증(fixture 와 동일 스키마).

- commit: `feat(examples): add hybrid network example + README`

---

## Task 2 — bash 실-바이너리 hybrid capability + 게이팅 e2e (검증 핵심)

**파일**: `tests/unit/tests/network-hybrid-capabilities.sh`

**RED→GREEN 단계**:
1. 표준 bash 테스트 헤더(assert.sh, CHAINBENCH_DIR) + `TMPDIR_ROOT=$(mktemp -d)` + trap cleanup.
2. chainbench-net 빌드: `( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net )`; `export CHAINBENCH_NET_BIN`.
3. `export CHAINBENCH_STATE_DIR="${TMPDIR_ROOT}/state"`; `mkdir -p "${CHAINBENCH_STATE_DIR}/networks"`; 예제(또는 인라인 hybrid JSON)를 `networks/hybrid-demo.json` 으로 복사.
4. `source "${CHAINBENCH_DIR}/lib/network_client.sh"`.
5. **단언 A (교집합)**: `data=$(cb_net_call "network.capabilities" '{"network":"hybrid-demo"}')`; `jq -r '.capabilities | join(",")'` == `"rpc,ws"`. (실 바이너리가 실 mixed-provider state 파일을 읽어 교집합 산출 — Go unit test 의 end-to-end 등가물.)
6. **단언 B (게이팅)**: hybrid 활성 가정 하에 `_cb_test_active_capabilities` 가 `rpc ws` 만 반환 → process 요구 작업이 게이팅됨을 보임. cmd_test.sh 헬퍼를 cmd-test-capabilities.sh 방식(awk 추출)으로 끌어와, 실 `cb_net_call`(network_client.sh)로 active caps 조회 후 `requires_capabilities:[process]` fixture 가 SKIP 판정되는지 단언. (mock 아님 — 실 바이너리 경유.)
7. **단언 C (대조)**: 같은 헬퍼로 `requires_capabilities:[rpc]` 는 RUN 판정(하한에 포함).
8. `unit_summary`.

> 주의: 예제 파일 경로를 테스트가 참조하면 Task 1 산출물에 의존 → 테스트 self-contained 위해 인라인 heredoc 으로 hybrid JSON 작성(예제와 동일 구조) 권장. README 에는 "테스트가 동일 구조를 검증한다" 한 줄.

**검증**: `bash tests/unit/run.sh` 에 신규 1개 포함 green (cast 무관 — RPC dial 없음).

- commit: `test(network): bash hybrid capability intersection + gating e2e`

---

## Task 3 — MCP hybrid lower-bound 케이스 (최소)

**파일**: `mcp-server/test/network.test.ts` 에 케이스 1개 추가 (`_Happy_Hybrid`).
- mock wire 가 `{network:"hybrid-example", capabilities:["rpc","ws"]}` 반환 시 `_networkCapabilitiesHandler({network:"hybrid-example"})` 가 `rpc,ws` 노출 + `process`/`admin` 부재(negative) 단언. `_Happy_Remote` 패턴 재사용하되 주석으로 "hybrid 도 동일 lower-bound passthrough 임을 명시(MCP 는 wire 결과 그대로 노출 — local∩remote 교집합은 Go/bash 에서 계산)".

**검증**: `cd mcp-server && npm test` green (+1).

- commit: `test(mcp): document hybrid network lower-bound passthrough`

---

## Task 4 — 문서 + 로드맵 + 버전

- `docs/VISION_AND_ROADMAP.md`: Sprint 5d 체크박스 `[x]` + 한 줄 요약(예제+검증, 구성명령 후속).
- `docs/REMAINING_WORK.md`: §4 Priority 3 (Sprint 5d) ✅ 완료 처리 + D1(JSON 예제) 사유 1줄 + "구성 명령 = 후속" 명시. §2.0 / 권장순서에서 5d 제거, 다음 = 5b.
- `docs/REFACTORING_PLAN.md`: 무관(건드리지 않음).
- 버전 bump: `mcp-server/package.json` 0.7.1 → 0.8.0 (minor — 신규 사용자-facing 예제 + 검증). 다른 버전 소스 있으면 동기화.
- commit: `docs+chore(sprint-5d): roadmap + remaining-work + version 0.8.0`

---

## 완료 기준 (체크리스트)

- [ ] `examples/networks/hybrid-example.json` 유효 JSON + 스키마 충족, README 사용 흐름 명확
- [ ] bash: 실 바이너리로 hybrid state → `[rpc,ws]` 교집합 + process 게이팅 SKIP + rpc RUN 단언
- [ ] MCP: hybrid lower-bound passthrough 케이스
- [ ] Go 기존 `TestHandleNetworkCapabilities_Hybrid` 유지(회귀 0)
- [ ] 전 레이어 green: Go 전 패키지 · vitest · bash
- [ ] 로드맵/REMAINING_WORK 갱신 + 버전 bump
- [ ] 커밋 규약: English, no co-author, no emoji, conventional prefix
