# Sprint 5b.4 — attach surface (CLI + MCP) — Plan

> 짝 spec: `2026-06-30-sprint-5b-4-attach-surface.md`. 표면만(새 wire 동작 0). 커밋: English, no co-author, no emoji.

## Task 0 — spec + plan 커밋
- 브랜치 `feat/sprint-5b-4-attach-surface`.
- commit: `docs: add Sprint 5b.4 spec + plan for attach surface`

## Task 1 — bash `lib/cmd_network.sh`
- `# Command: network — ...` 헤더(dispatch 설명용). dispatch: `$1`=action; `attach` → `_cb_network_cmd_attach`, else usage.
- `_cb_network_cmd_attach`: cmd_remote.sh 의 flag-parse 패턴. positionals=name,rpc_url. flags(§3). 검증(name/rpc_url 필수; provider=ssh-remote → ssh-user/host/env 필수).
- auth/provider_meta JSON 빌드: python3(json.dumps; 빈 객체는 생략). wireArgs JSON 조립.
- `source lib/network_client.sh`; `cb_net_call "network.attach" "$json"`. 결과 파싱(jq/cb_json) → 사람 출력(name/chain_type/chain_id/created); `--json` 시 raw.
- commit: `feat(cli): chainbench network attach (lib/cmd_network.sh)`

## Task 2 — bash 테스트
- `tests/unit/tests/cmd-network-attach.sh`: 실 바이너리 빌드 + mock JSON-RPC(eth_chainId+istanbul → stablenet). `chainbench network attach netx http://127.0.0.1:PORT` 호출 경로(직접 `_cb_network_cmd_attach` 또는 chainbench.sh) → networks/netx.json 생성 + chain_id 8283 출력.
  - ssh-remote arg 검증: `--provider ssh-remote` + ssh-env 누락 → CLI 에러(rc≠0); 정상 flag → JSON 빌드 확인(wire 가 SSH dial 실패해도 표면이 args 전달했음 = handler 가 INVALID_ARGS 아닌 UPSTREAM/ssh 에러 반환).
- commit: `test(cli): network attach remote happy + ssh-remote arg validation`

## Task 3 — MCP `chainbench_network_attach`
- `network.ts`: `NetworkAttachArgs`(§4) + `_networkAttachHandler`(buildWireArgs → callWire("network.attach") → formatWireResult). `registerNetworkTools` 에 `server.tool("chainbench_network_attach", <desc, 보안주의>, NetworkAttachArgs.shape, _networkAttachHandler)`.
- export `NetworkAttachArgs`, `_networkAttachHandler`(테스트용).
- commit: `feat(mcp): chainbench_network_attach tool`

## Task 4 — MCP 테스트
- `network.test.ts`: `_Attach_Remote_Happy`(mock wire result → formatted 출력), `_Attach_SSH_ArgsMapped`(mock 이 받은 wire args echo — provider/auth/provider_meta 전달 확인; mock fixture 가 args 반영 가능하면), `_StrictRejectsUnknownKeys`, `_WireFailure_PassedThrough`.
  - mock 이 envelope args 를 못 잡으면, 최소 happy/strict/passthrough 로 한정하고 arg 매핑은 buildWireArgs 단위로 간접 보장.
- commit: `test(mcp): network attach arg mapping + strict + passthrough`

## Task 5 — 문서 + 버전
- VISION 5b: 5b.4(attach CLI/MCP 표면) 항목. REMAINING_WORK: 5b 후속 표면 ✅, 잔여(키 인증/hybrid compose).
- 버전 0.11.0 → 0.12.0(minor — 신규 사용자 표면).
- commit: `docs+chore(sprint-5b-4): roadmap + remaining-work + version 0.12.0`

## 완료 기준
- [ ] `chainbench network attach <n> <url>` (remote) 가 networks 파일 생성 + 출력
- [ ] ssh-remote flag → 올바른 auth/provider_meta JSON, 필수 flag 검증
- [ ] `chainbench_network_attach` MCP 툴 — remote/ssh-remote args 매핑 + strict + passthrough
- [ ] 자격증명 env-only(인라인 flag 없음)
- [ ] 전 레이어 green: Go · vitest · bash
