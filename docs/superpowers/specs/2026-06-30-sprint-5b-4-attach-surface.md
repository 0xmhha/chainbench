# Sprint 5b.4 — network.attach user surface (CLI + MCP)

> 작성일: 2026-06-30
> 상태: SPEC (검토 대기)
> 선행: 5b.3 (attach ssh-remote wire), Sprint 3b (network.attach), 5c.* (MCP reroute 패턴)
> 짝 plan: `docs/superpowers/plans/2026-06-30-sprint-5b-4-attach-surface.md`

---

## 1. Goal

지금까지 `network.attach` 는 **사용자 표면이 없는 wire 프리미티브**(테스트가 `cb_net_call` 로만 호출)였다. 이를 두 1차 표면에 노출한다:
- **bash CLI** (mode B, 사람): `chainbench network attach <name> <rpc_url> [flags]`
- **MCP tool** (mode A, LLM): `chainbench_network_attach`

둘 다 remote(기본) + ssh-remote(5b.3) provider 를 지원해, SSH 작업 전체가 실사용 가능해진다.

---

## 2. Non-goals

- 새 wire 동작 없음 — `network.attach`(5b.3) 는 그대로. 본 sprint 는 **표면만**.
- `remote add`(remotes.json 레지스트리) 통합/대체 안 함 — 별개 개념.
- hybrid compose, 키 인증 — 후속.

---

## 3. bash CLI — `lib/cmd_network.sh` (신규)

동적 dispatch(`chainbench <sub>` → `lib/cmd_<sub>.sh`)로 `network` 서브커맨드 신설. 첫 인자 = 액션(`attach`; 확장 여지).

```
chainbench network attach <name> <rpc_url> [flags]
  --type <chain_type>           probe override (stablenet|wbft|wemix|ethereum)
  --provider remote|ssh-remote  default remote
  # remote auth (api-key/jwt):
  --auth-type api-key|jwt        --auth-env <VAR>   --auth-header <H>   (api-key)
  # ssh-remote (→ ssh-password auth + provider_meta):
  --ssh-user <U>  --ssh-host <H>  --ssh-port <N>  --ssh-env <VAR>
  --log-file <PATH>  --start-cmd <C>  --stop-cmd <C>  --restart-cmd <C>
```

- flag → `auth`/`provider_meta` JSON 객체 빌드(python 또는 json_helpers) → `cb_net_call "network.attach" "$json"`.
- 결과(`cb_net_call` 의 NDJSON result)를 사람-읽기 출력(name/chain_type/chain_id/created) + `--json` 패스스루.
- 입력 검증: name/rpc_url 필수, provider=ssh-remote 면 ssh-user/host/env 필수, 알 수 없는 flag 경고. wire 가 최종 검증(중복 boundary 회피).

## 4. MCP tool — `chainbench_network_attach` (`network.ts`)

```ts
NetworkAttachArgs = z.object({
  name: z.string().min(1),
  rpc_url: z.string().min(1),
  override: z.string().optional(),                       // chain_type override
  provider: z.enum(["remote","ssh-remote"]).optional(),
  auth: z.object({
    type: z.enum(["api-key","jwt","ssh-password"]),
    env: z.string().optional(), header: z.string().optional(),
    user: z.string().optional(), host: z.string().optional(), port: z.number().optional(),
  }).strict().optional(),
  provider_meta: z.object({
    log_file: z.string().optional(),
    start_cmd: z.string().optional(), stop_cmd: z.string().optional(), restart_cmd: z.string().optional(),
  }).strict().optional(),
}).strict();
```
- `buildWireArgs`(P1-3)로 제공된 필드만 wire args 구성 → `callWire("network.attach", args)` → `formatWireResult`.
- description 에 보안 주의(시크릿은 `auth.env` 가 가리키는 env var 로, password/key 인라인 금지) 명시.

---

## 5. 보안

- 자격증명은 **env var 이름만** 전달(api-key/jwt/ssh-password 공통) — password/key 인라인 금지. CLI flag 도 `--*-env <VAR>` 만 받음(값 X). 5b.1~5b.3 경계 계승.
- ssh host key 정책은 wire(5b.1) 가 env 로 처리 — 표면은 관여 안 함.

---

## 6. Tests

- **bash** `tests/unit/tests/cmd-network-attach.sh`: 실 바이너리 + mock RPC 로 `chainbench network attach <n> <url>`(remote) happy → networks 파일 생성 + 출력. ssh-remote 는 SSH 서버 없이 **arg 검증**(필수 flag 누락 → CLI 에러; provider=ssh-remote + 정상 flag → JSON 빌드 후 wire 가 SSH dial 시도 = 표면 동작 증명). remote attach e2e 는 기존 `network-attach.sh` 가 wire 직접 커버.
- **MCP** `network.test.ts`: `_networkAttachHandler` — remote happy(mock wire result), ssh-remote args 매핑(provider/auth/provider_meta 가 callWire 로 전달; mock 이 echo), strict 거부(unknown key), wire 실패 passthrough.
- **ssh-remote 터널 e2e** 는 Go 핸들러 테스트(5b.3)가 이미 커버 — 표면 테스트는 arg/결과 매핑에 집중.
- 회귀: Go · vitest · bash green.

---

## 7. Out-of-Scope / 후속

- `network detach/list`(networks/* 관리), hybrid compose.
- `remote add` 와 `network attach` 통합(두 remote 개념 일원화) — 큰 결정, 별도.
- 키 인증/키체인.

---

## 8. 예상 커밋 (~5-6)

1. `docs: add Sprint 5b.4 spec + plan`
2. `feat(cli): chainbench network attach (lib/cmd_network.sh)`
3. `test(cli): network attach remote happy + ssh-remote arg validation`
4. `feat(mcp): chainbench_network_attach tool`
5. `test(mcp): network attach arg mapping + strict + passthrough`
6. `docs+chore(sprint-5b-4): roadmap + remaining-work + version bump`
