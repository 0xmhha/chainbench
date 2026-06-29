# Sprint 5b.1 — SSHRemoteDriver (read-only RPC, pass 1)

> 작성일: 2026-06-29
> 상태: SPEC (검토 대기 — 보안 결정 §6 사인오프 필요)
> 선행: VISION §5.3 (L2 driver dispatch), §5.16 S6 (SSH 자격증명=세션 prompt/메모리), Q6 (auth=user+password), Sprint 3b.2b (remote auth RoundTripper), Sprint 5a (capability gate)
> 짝 plan: `docs/superpowers/plans/2026-06-29-sprint-5b-1-sshremote-readonly.md`
> 후속: Sprint 5b.2 (process/fs capability — lifecycle + tail_log over SSH)

---

## 1. Goal

`ssh-remote` provider 노드에 대해 **읽기 전용 JSON-RPC** 를 SSH 터널 경유로 가능하게 한다. SSH 가 net.Conn(전송)만 제공하고, 그 위에서 **기존 `remote.Client`(ethclient 래퍼)를 그대로 재사용** — read 핸들러(block_number/chain_id/balance/gas_price/contract_call/events_get/account_state/rpc/tx_wait)는 코드 변경 없이 ssh-remote 노드에서 동작한다.

핵심 통찰(조사 확정): `remote.DialWithOptions(url, {Transport})` 가 이미 `http.Client{Transport}` 를 받는다. SSH `client.DialContext` 를 `&http.Transport{DialContext: ...}` 에 주입하면 RPC 트래픽이 SSH 터널을 통과한다. 신규 드라이버는 **SSH 연결 수립 + 터널 transport 구성 + remote.Client 반환**만 한다.

---

## 2. Non-goals (5b.2+ 로 연기)

- **process / fs capability** (node stop/start/restart, tail_log over SSH shell exec) — 5b.2.
- **ssh-remote 노드 구성 명령** (`network attach --provider ssh-remote` 또는 신규 `network attach-ssh`). 5d 와 동일 패턴: 5b.1 은 **수동 구성(networks/<name>.json 직접 작성)** 을 v1 흐름으로, 구성 명령은 후속. (probe 가 http(s) 직결을 요구해 SSH 노드 자동 probe 는 별도 설계.)
- **SSH 키 기반 인증 / OS 키체인** — Q6 는 user+password. 키 인증은 후속(S6 "자동화 필요 시 키체인").
- **WS subscription** — `subscription.open` 자체가 미구현(전 provider 공통). ws capability 광고는 remote 와 동일하게 유지(전송 가능성 선언).

---

## 3. 이미 존재하는 것 (재사용)

| 자산 | 위치 | 5b.1 에서 |
|---|---|---|
| `remote.Client` + `DialWithOptions(Transport)` | `drivers/remote/client.go:59` | SSH transport 주입해 그대로 반환 |
| read 핸들러 + `resolveNode`/`dialNode` | `cmd/chainbench-net/handlers.go:133,170` | `dialNode` 에 provider 분기 추가 |
| `ssh-password` auth 스키마 (user/host/port) | `network/schema/network.json:52` | `env` 필드 추가(§5) |
| `ValidateAuth` ssh-password passthrough | `drivers/remote/auth.go:78` | env 필수화 |
| `providerCaps["ssh-remote"]` | `handlers_network.go:33` | 구현 현실에 맞춰 `{rpc, ws}` 로 정렬(§6 D2) |
| `golang.org/x/crypto v0.44.0` (indirect) | `network/go.mod:48` | `ssh` 직접 import → direct 승격 |

---

## 4. User-Facing Surface

- **신규 드라이버 패키지** `network/internal/drivers/sshremote/` — `Dial(ctx, creds, rpcURL) (*remote.Client, error)`.
- **node 핸들러는 표면 불변** — ssh-remote 노드를 가진 networks 파일이 있으면 기존 `node.*` read 명령이 그대로 동작(provider 분기는 내부).
- **자격증명 주입**: `ssh-password` auth 의 `env` 필드가 가리키는 환경변수에서 password 읽음(`os.Getenv`). signer 키(S4/S5)·api-key/jwt 와 동일한 env 주입 모델 — spawn-per-call(S2)에서 "세션"=spawn env. 평문 파일 저장 0.

---

## 5. Schema 변경

`network/schema/network.json` 의 `ssh-password` variant 에 `env` 추가(api-key/jwt 패턴 동일):
```json
{
  "properties": {
    "type": { "const": "ssh-password" },
    "user": { "type": "string" },
    "host": { "type": "string" },
    "port": { "type": "integer", "default": 22 },
    "env":  { "type": "string", "description": "env var holding the SSH password" }
  },
  "required": ["type", "user", "host", "env"]
}
```
→ `cd network && go generate ./...` 재생성, 같은 커밋에 포함.

---

## 6. 보안 결정 (사인오프 필요)

- **D1 — password 는 env-only, 절대 미저장/미로깅.** 에러 메시지는 env var **이름만** 참조(remote auth 관례). password 값은 stdout/stderr/log/네트워크 어디에도 금지. SSH config 의 password 는 dial 직후 사용, best-effort 메모리 상주.
- **D2 — capability 정직성.** 5b.1 에서 `providerCaps["ssh-remote"]` 를 forward-declared `{fs,process,rpc,ws}` → **구현된 `{rpc, ws}`** 로 정렬. (fs/process 미구현 상태로 광고하면 hybrid 교집합·게이팅이 거짓 양성.) 5b.2 가 fs/process 복원.
- **D3 — host key 검증** ⚠️ **사용자 결정 필요**. 옵션:
  - (a) **known_hosts 검증 (기본 보안)** — `~/.ssh/known_hosts` (또는 `CHAINBENCH_SSH_KNOWN_HOSTS`) 로 `ssh.FixedHostKey`/`knownhosts.New`. 미등록 호스트는 연결 거부. 안전하나 사전 등록 필요(테스트 샌드박스 마찰).
  - (b) **명시적 opt-in insecure** — 기본은 known_hosts, `CHAINBENCH_SSH_INSECURE_HOST_KEY=1` 일 때만 `InsecureIgnoreHostKey()`. 편의+안전 절충, 위험은 loud opt-in.
  - (c) insecure 기본 — 비권장(보안 경계 원칙 위배).
  - **권장: (b).** 기본 known_hosts, 위험은 명시 opt-in. mock SSH 서버 테스트는 opt-in 경로로.

---

## 7. 드라이버 설계 (구현 스케치)

```go
// network/internal/drivers/sshremote/sshremote.go
package sshremote

type Credentials struct{ User, Host string; Port int; Password string }

// Dial establishes an SSH connection and returns a remote.Client whose RPC
// traffic is tunneled through it. The returned client's Close() also closes
// the SSH connection.
func Dial(ctx context.Context, creds Credentials, rpcURL string, hostKey ssh.HostKeyCallback) (*remote.Client, error) {
    cfg := &ssh.ClientConfig{
        User: creds.User,
        Auth: []ssh.AuthMethod{ssh.Password(creds.Password)},
        HostKeyCallback: hostKey,             // §6 D3
        Timeout: sshDialTimeout,
    }
    sshC, err := ssh.Dial("tcp", net.JoinHostPort(creds.Host, strconv.Itoa(port)), cfg)
    // err: never include creds.Password
    transport := &http.Transport{DialContext: func(ctx, _ , addr) (net.Conn, error) {
        return sshC.DialContext(ctx, "tcp", addr)   // tunnel RPC TCP through SSH
    }}
    return remote.DialWithOptions(ctx, rpcURL, remote.DialOptions{Transport: transport, Closer: sshC})
}
```

`remote` 패키지 최소 확장: `DialOptions{ Closer io.Closer }` + `Client.Close()` 가 `extra` 도 닫음. (현 `Close()` 는 rpc 만 닫음 — SSH conn 누수 방지.)

`handlers.go:dialNode` 에 provider 분기:
```go
switch node.Provider {
case "remote", "": /* 기존 HTTP 경로 */
case "ssh-remote":
    // auth.type=ssh-password 강제, user/host/port/env 파싱, os.Getenv(env)→password
    // hostKey := resolveHostKeyCallback()  // §6 D3
    return sshremote.Dial(ctx, creds, node.Http, hostKey)
default: return nil, NewUpstream("unsupported provider", ...)
}
```

---

## 8. Tests

1. **Go 유닛 (sshremote)** — in-process SSH 서버(`golang.org/x/crypto/ssh` `NewServerConn` + channel accept → `direct-tcpip` forward) 가 mock JSON-RPC(httptest) 로 포워딩. `Dial` → `BlockNumber`/`ChainID` 가 터널 경유로 성공. password 인증 성공/실패, host key 거부(D3-a) 케이스.
2. **Go 핸들러** — networks 파일에 ssh-remote 노드 + `node.block_number` 가 dialNode 분기 타고 동작(mock SSH 서버). auth 누락/타입 불일치 → 분류된 에러.
3. **보안 negative** — password 값이 핸들러 stdout/stderr/에러에 안 나타남(grep 기반, signer 경계 테스트 패턴).
4. **redaction** — 에러는 env var 이름만.
5. 회귀: Go 전 패키지 · vitest · bash green.

---

## 9. Error Classification

| 코드 | 경우 |
|---|---|
| `INVALID_ARGS` | auth.type≠ssh-password, user/host/env 누락 |
| `UPSTREAM_ERROR` | SSH dial 실패, host key 거부, env var 빔, RPC 실패 |
| `NOT_SUPPORTED` | (5b.2) process/fs 요청이 5b.1 드라이버에 |

---

## 10. Out-of-Scope / 후속

- **5b.2**: shell exec 기반 process(stop/start/restart)·fs(tail_log) + providerCaps fs/process 복원.
- **ssh-remote 구성 명령**: attach 확장(SSH probe 포함) — 별도.
- **키 인증 / 키체인** — S6 후속.

---

## 11. 예상 커밋 (~6-8)

1. `docs: add Sprint 5b.1 spec + plan`
2. `feat(remote): add Closer to DialOptions for tunneled clients`
3. `feat(schema): add env field to ssh-password auth + regenerate`
4. `feat(sshremote): SSH-tunneled read-only RPC driver`
5. `feat(network-net): dialNode ssh-remote branch + ssh-password consumption`
6. `fix(network-net): align ssh-remote providerCaps to {rpc,ws}`
7. `test(sshremote): in-process SSH server integration + redaction`
8. `docs+chore(sprint-5b-1): roadmap + remaining-work + version bump`
