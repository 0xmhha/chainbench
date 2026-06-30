# Sprint 5b.5 — network list / detach — Plan

> 짝 spec: `2026-06-30-sprint-5b-5-network-list-detach.md`. 커밋: English, no co-author, no emoji.

## Task 0 — spec + plan 커밋 (branch `feat/sprint-5b-5-network-list-detach`)
- commit: `docs: add Sprint 5b.5 spec + plan for network list/detach`

## Task 1 — state: ListRemotes + RemoveRemote
- `internal/state/remote.go`: 두 함수(§3.1). loadRemote 재사용. sort by name. dir 부재 → 빈 슬라이스.
- `remote_test.go`: ListRemotes(빈/다건/손상 skip) + RemoveRemote(성공/부재 ErrStateNotFound/reserved ErrReservedName).
- commit: `feat(state): ListRemotes + RemoveRemote`

## Task 2 — wire 핸들러 + 스키마
- `handlers_network.go`: `newHandleNetworkList(stateDir)`, `newHandleNetworkDetach(stateDir)`. list → {networks:[{name,chain_type,chain_id,node_count}]}; detach → 분류(§4) + {name,detached:true}.
- `handlers.go allHandlers`: 등록.
- `command.json`: enum 에 `network.detach`/`network.list` 알파벳 순 추가 → `cd network && go generate ./...` 재생성(같은 커밋).
- commit: `feat(network-net): network.list + network.detach handlers + schema`

## Task 3 — wire 핸들러 테스트
- `handlers_test.go`: list(다건 정렬), detach(성공→load 실패/부재→UPSTREAM/reserved·invalid·누락→INVALID_ARGS), allHandlers 포함. drift 가드 통과 확인.
- commit: `test(network-net): list/detach handler + state coverage`

## Task 4 — bash CLI
- `cmd_network.sh`: dispatch 에 `list`/`detach` 추가. `_cb_network_cmd_list`(표/--json), `_cb_network_cmd_detach <name>`(--json). usage 갱신.
- commit: `feat(cli): chainbench network list / detach`

## Task 5 — MCP
- `network.ts`: `NetworkListArgs`(빈 strict)/`_networkListHandler`, `NetworkDetachArgs`({name})/`_networkDetachHandler`. register 2 tools.
- commit: `feat(mcp): chainbench_network_list + chainbench_network_detach`

## Task 6 — 표면 테스트
- bash `cmd-network-attach.sh` 확장: attach → list 에 보임 → detach → 안 보임.
- MCP `network.test.ts`: list happy(mock), detach happy/strict/passthrough.
- commit: `test(cli+mcp): list/detach surface coverage`

## Task 7 — 문서 + 버전
- VISION 5b.5 항목. REMAINING_WORK: network detach/list ✅. 버전 0.12.0 → 0.13.0.
- commit: `docs+chore(sprint-5b-5): roadmap + remaining-work + version 0.13.0`

## 완료 기준
- [ ] network.list/detach wire + bash + MCP 동작, attach↔detach round-trip
- [ ] detach 에러 분류(reserved/invalid/누락→INVALID_ARGS, 부재→UPSTREAM)
- [ ] 스키마 enum + 재생성 + drift 가드 통과
- [ ] 전 레이어 green: Go(전 패키지, vet/gofmt) · vitest · bash · go.mod tidy 안정
