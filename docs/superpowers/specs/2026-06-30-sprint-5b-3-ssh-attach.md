# Sprint 5b.3 — network.attach for ssh-remote (construction)

> 작성일: 2026-06-30
> 상태: SPEC (검토 대기)
> 선행: 5b.1/5b.2 (sshremote driver), Sprint 3b (network.attach), `docs/REMAINING_WORK.md` §4 (5b 후속)
> 짝 plan: `docs/superpowers/plans/2026-06-30-sprint-5b-3-ssh-attach.md`
> 범위 확정(사용자 2026-06-30): **wire 레벨만**. CLI/MCP 표면은 후속.

---

## 1. Goal

`network.attach` wire 핸들러를 확장해 **ssh-remote 노드를 구성**할 수 있게 한다. 5b.1/5b.2 로 ssh-remote 가 동작하지만 노드를 networks/<name>.json 에 **수동 작성**해야 했던 v1 한계를 제거한다. remote attach 와 동일하게 **auto chain_id/chain_type 감지**하되, RPC 가 SSH 터널 뒤에 있으므로 **터널 경유 probe** 를 쓴다.

핵심 통찰(조사 확정): `probe.Options` 에 이미 `Client *http.Client` 필드가 있다 → SSH 터널 transport 를 주입하면 **probe 코드 변경 0** 으로 터널 경유 감지가 된다.

---

## 2. Non-goals

- **CLI/MCP 표면 추가 안 함.** `network.attach` 는 현재 사용자 표면이 없는 wire 프리미티브(테스트가 `cb_net_call` 로 호출). ssh-remote 도 동일 레벨로 노출 — `chainbench network attach` CLI / MCP 툴은 별도 sprint(remote 도 함께).
- **스키마 변경 없음.** attach args 는 핸들러 struct 로 파싱(command.json 은 명령 enum 만 검증). network.json 상태 스키마는 이미 provider/provider_meta/ssh-password 지원.
- **키 인증/키체인** — Q6 user+password 유지(S6 후속).
- **`remote add`(remotes.json 레지스트리) 통합 안 함** — 별개 개념, 본 작업과 무관.

---

## 3. 설계

### 3.1 드라이버 — 터널된 http.Client

```go
// sshremote.go (신규)
// DialTunnelClient opens an SSH connection and returns an *http.Client whose
// TCP dials are tunneled through it, plus the SSH connection as an io.Closer to
// release when the caller is done (e.g. after a probe). Reuses dialSSH (5b.1).
func DialTunnelClient(creds Credentials, hostKey ssh.HostKeyCallback) (*http.Client, io.Closer, error)
```
- `Dial`(터널 RPC) 도 내부적으로 이 transport 를 만들므로 공통 `tunnelTransport(sshC)` 헬퍼로 묶어 재사용(동작 불변).

### 3.2 attach 핸들러 — provider 분기

`network.attach` req 에 `provider`, `provider_meta` 추가:
- `provider` 미지정 또는 `"remote"` → 기존 HTTP probe 경로(provider=remote 노드). **동작 불변**.
- `provider == "ssh-remote"`:
  1. `auth` 에서 ssh-password creds 추출(`sshCredsFromNode` 재사용 — type/user/host/env + os.Getenv + 검증).
  2. `hostKey = sshremote.ResolveHostKeyCallback(os.Getenv)`.
  3. `client, closer = sshremote.DialTunnelClient(creds, hostKey)` → `defer closer.Close()`.
  4. `probe.Detect(Options{RPCURL: req.RPCURL, Client: client, Override: req.Override})` → chain_type/chain_id (터널 경유).
  5. 노드 빌드: provider=ssh-remote, http=rpc_url, auth=ssh-password, provider_meta=req.ProviderMeta.
  6. `state.SaveRemote` 후 결과 반환(remote attach 와 동일 shape + created).
- 그 외 provider → `INVALID_ARGS`.

### 3.3 결과/계약

remote attach 와 동일: `{name, chain_type, chain_id, nodes, rpc_url, created}`. ssh-remote 노드는 auth + provider_meta 포함.

---

## 4. 보안 (5b.1/5b.2 계승)

- SSH password env-only(미저장/미로깅), 에러는 env var 이름만. probe 실패/터널 실패 시 password 미노출.
- host key known_hosts 기본 + `CHAINBENCH_SSH_INSECURE_HOST_KEY=1` opt-in.
- probe 후 SSH 연결 즉시 close(터널 누수 방지).
- provider_meta 명령은 operator 신뢰 입력(5b.2 D1) — attach 가 그대로 persist, 실행 안 함.

---

## 5. Tests

1. **드라이버**: `DialTunnelClient` 가 mock RPC 로 터널 — in-process SSH 서버(direct-tcpip → mock JSON-RPC). `http.Client.Get`/probe 성공.
2. **attach 핸들러(통합)**: in-process SSH 서버 + mock RPC(eth_chainId + istanbul_getValidators → stablenet) → `network.attach{provider:ssh-remote, rpc_url, auth:ssh-password, provider_meta}` → 저장된 노드가 provider=ssh-remote + chain_id 감지 + provider_meta 보존.
3. **에러 경로**(SSH 불필요): provider="ssh-remote" + auth 누락/타입불일치 → INVALID_ARGS; 알 수 없는 provider → INVALID_ARGS; env 빔 → UPSTREAM.
4. **remote attach 무회귀**: 기존 attach 테스트 green(provider 미지정 경로 불변).
5. 보안 negative: password 미노출.
6. 회귀: Go 전 패키지 · vitest · bash green.

---

## 6. Error Classification

| 코드 | 경우 |
|---|---|
| `INVALID_ARGS` | 알 수 없는 provider, ssh-password auth 불완전, name/rpc_url 누락 |
| `UPSTREAM_ERROR` | SSH dial/터널 실패, env var 빔, probe 실패(터널 경유) |

---

## 7. Out-of-Scope / 후속

- `chainbench network attach` CLI + MCP 툴(remote+ssh-remote 공통 표면).
- network 단위 다중 노드 hybrid attach(per-node provider 합성).
- 키 인증/키체인.

---

## 8. 예상 커밋 (~5)

1. `docs: add Sprint 5b.3 spec + plan`
2. `feat(sshremote): DialTunnelClient for tunneled probe`
3. `feat(network-net): network.attach supports ssh-remote provider`
4. `test(network-net): attach ssh-remote via tunneled probe`
5. `docs+chore(sprint-5b-3): roadmap + remaining-work + version bump`
