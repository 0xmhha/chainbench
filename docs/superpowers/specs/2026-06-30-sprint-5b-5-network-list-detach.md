# Sprint 5b.5 — network list / detach (CRUD completion)

> 작성일: 2026-06-30
> 상태: SPEC (검토 대기)
> 선행: 5b.4 (`chainbench network attach` 표면), Sprint 3b (attach/load)
> 짝 plan: `docs/superpowers/plans/2026-06-30-sprint-5b-5-network-list-detach.md`

---

## 1. Goal

`chainbench network attach`(5b.4) 로 네트워크를 붙일 수 있지만 **목록 조회·제거 수단이 없다**(networks/<name>.json 직접 편집뿐). `network list` 와 `network detach` 를 추가해 `network` 커맨드 CRUD 를 완성한다. 세 레이어(wire/bash/MCP) 일관 노출.

- `network.list` — `state/networks/` 의 attached 네트워크 요약 목록.
- `network.detach <name>` — `state/networks/<name>.json` 제거(attach 의 역).

---

## 2. Non-goals

- local 네트워크 제거/관리 안 함 — local 은 pids.json 기반(networks 파일 아님). list 는 attached(remote/ssh-remote/hybrid)만.
- 실행 중 프로세스 영향 없음 — attached 네트워크는 메타데이터일 뿐(노드 프로세스 미보유). detach 는 파일 제거.
- hybrid compose / 키 인증 — 별도 후속.

---

## 3. 설계

### 3.1 state (`internal/state/remote.go`)
```go
// ListRemotes: networks/ 의 *.json 을 loadRemote 로 파싱(검증/무결성 재사용),
// name 정렬. dir 부재 → 빈 슬라이스(에러 아님). 손상 파일은 skip.
func ListRemotes(stateDir string) ([]*types.Network, error)

// RemoveRemote: reserved/invalid name 거부, networks/<name>.json 제거.
// 부재 → ErrStateNotFound.
func RemoveRemote(stateDir, name string) error
```

### 3.2 wire 핸들러 (`cmd/chainbench-net/handlers_network.go`)
- `network.list` → `{networks: [{name, chain_type, chain_id, node_count}]}`. args 없음.
- `network.detach` → args `{name}`; reserved/invalid → INVALID_ARGS; 부재 → UPSTREAM(ErrStateNotFound); 성공 → `{name, detached: true}`.
- `allHandlers` 등록 + `command.json` enum 에 `network.detach`/`network.list` 알파벳 순 추가 + `go generate` 재생성.

### 3.3 bash (`lib/cmd_network.sh`)
- `chainbench network list [--json]` → `cb_net_call "network.list" '{}'` → 표 형식(name/chain_type/chain_id/nodes) 또는 `--json`.
- `chainbench network detach <name> [--json]` → `cb_net_call "network.detach" {name}` → "Detached <name>".

### 3.4 MCP (`network.ts`)
- `chainbench_network_list`(인자 없음) + `chainbench_network_detach`({name}) — callWire → formatWireResult.

---

## 4. Error Classification

| 코드 | 경우 |
|---|---|
| `INVALID_ARGS` | detach: name 누락/reserved("local")/invalid 패턴 |
| `UPSTREAM_ERROR` | detach: 존재하지 않는 네트워크(ErrStateNotFound), 파일 제거 실패 |

list 는 dir 부재여도 빈 목록(에러 아님).

---

## 5. Tests

- **Go state**: ListRemotes(빈/다건/손상 skip), RemoveRemote(성공/부재/reserved).
- **Go 핸들러**: list(SaveRemote 다건 → 정렬 목록), detach(성공 후 load 실패 확인 / 부재 → UPSTREAM / reserved·invalid → INVALID_ARGS / name 누락 → INVALID_ARGS), allHandlers 포함, 스키마 enum drift 가드(기존 TestAllHandlers_DispatchableCommandsAreInSchema 통과).
- **bash**: `cmd-network-attach.sh` 확장 또는 신규 — attach 후 list 에 보임, detach 후 안 보임(실 바이너리).
- **MCP**: list happy(mock), detach happy/strict/passthrough.
- 회귀: Go·vitest·bash green.

---

## 6. Out-of-Scope / 후속

- `network rename`, hybrid compose, 키 인증.
- local 네트워크를 list 에 합쳐 보여주기(별도 — pids.json 합성).

---

## 7. 예상 커밋 (~7-8)

1. `docs: add Sprint 5b.5 spec + plan`
2. `feat(state): ListRemotes + RemoveRemote`
3. `feat(network-net): network.list + network.detach handlers + schema`
4. `test(network-net): list/detach handler + state coverage`
5. `feat(cli): chainbench network list / detach`
6. `feat(mcp): chainbench_network_list + chainbench_network_detach`
7. `test(cli+mcp): list/detach surface coverage`
8. `docs+chore(sprint-5b-5): roadmap + remaining-work + version bump`
