# Sprint 5b.1 — SSHRemoteDriver (read-only) — Plan

> 작성일: 2026-06-29
> 짝 spec: `docs/superpowers/specs/2026-06-29-sprint-5b-1-sshremote-readonly.md`
> 확정 결정: D3 = **known_hosts 기본 + `CHAINBENCH_SSH_INSECURE_HOST_KEY=1` opt-in insecure**.
> 실행: 직접 구현 + 전 레이어 테스트 검증. 커밋 규약: English, no co-author, no emoji, conventional prefix.

---

## 확정 설계 (조사 기반)

- SSH `client.DialContext` → `http.Transport{DialContext}` → `remote.DialWithOptions(Transport)` → `remote.Client` 재사용.
- `remote` 최소 확장: `DialOptions.Closer io.Closer` + `Client.Close()` 가 closer 도 닫음(SSH conn 누수 방지).
- `dialNode` (handlers.go:170) provider 분기: `remote|""` (기존) / `ssh-remote` (신규) / default 거부.
- `ssh-password` 스키마에 `env`(required) 추가 → password 는 `os.Getenv(env)`.
- providerCaps["ssh-remote"] `{fs,process,rpc,ws}` → `{rpc,ws}` (D2).
- host key: `resolveHostKeyCallback()` — `CHAINBENCH_SSH_INSECURE_HOST_KEY=1` → `InsecureIgnoreHostKey()`; else `knownhosts.New(CHAINBENCH_SSH_KNOWN_HOSTS || ~/.ssh/known_hosts)`.

---

## Task 0 — spec + plan 커밋
- 브랜치 `feat/sprint-5b-1-sshremote`.
- commit: `docs: add Sprint 5b.1 spec + plan for SSH-tunneled read-only RPC`

## Task 1 — remote.Client Closer 확장
- `drivers/remote/client.go`: `DialOptions` 에 `Closer io.Closer` 추가; `Client` 에 `extra io.Closer` 필드; `DialWithOptions` 가 opts.Closer 저장; `Close()` 가 rpc.Close 후 extra.Close(nil-safe, 양쪽 에러 join).
- 테스트: `client_test.go` 에 Closer 가 Close 시 호출되는지(스파이 io.Closer) 단언.
- 회귀: `go -C network test ./internal/drivers/remote/...`.
- commit: `feat(remote): add Closer to DialOptions for tunneled clients`

## Task 2 — ssh-password 스키마 env 필드 + 재생성
- `network/schema/network.json`: ssh-password 에 `env`(required) 추가.
- `cd network && go generate ./...` → `*_gen.go` 재생성(같은 커밋).
- `drivers/remote/auth.go ValidateAuth`: ssh-password 에 `env` 필수 검증 추가.
- fixtures: ssh-password 를 쓰는 스키마 fixture 있으면 env 추가(없으면 신규 valid/invalid fixture).
- 테스트: auth_test.go ValidateAuth(ssh-password without env)→error.
- commit: `feat(schema): require env on ssh-password auth + regenerate`

## Task 3 — sshremote 드라이버 패키지
- `network/internal/drivers/sshremote/sshremote.go`:
  - `Credentials{User,Host string; Port int; Password string}`.
  - `Dial(ctx, creds, rpcURL string, hostKey ssh.HostKeyCallback) (*remote.Client, error)` — §7 스케치. `net.JoinHostPort`, `ssh.Dial` timeout(const `sshDialTimeout=15s`), transport.DialContext=sshC.DialContext, `remote.DialWithOptions(...,{Transport,Closer:sshC})`. 에러에 password 절대 미포함.
  - `ResolveHostKeyCallback(env func(string)string) (ssh.HostKeyCallback, error)` — insecure opt-in / knownhosts.
  - `golang.org/x/crypto/ssh` + `.../ssh/knownhosts` import → go.mod direct 승격(`go mod tidy`).
- `doc.go` 패키지 설명(보안 경계: password env-only, host key 정책).
- commit: `feat(sshremote): SSH-tunneled read-only RPC driver`

## Task 4 — dialNode provider 분기 + ssh-password 소비
- `cmd/chainbench-net/handlers.go dialNode`: provider switch. ssh-remote → auth.type=ssh-password 강제, user/host/env 파싱(`*string`/맵 안전 캐스팅), port default 22, password=os.Getenv(env)(빔→UPSTREAM), hostKey=sshremote.ResolveHostKeyCallback(os.Getenv), sshremote.Dial. 에러 분류(§9).
- `getIntOrDefault` 헬퍼(맵 숫자→int, JSON float64 대응).
- 테스트: handlers_test.go — ssh-remote 노드 + mock SSH 서버로 node.block_number happy; auth 누락/타입불일치/env빔 → 분류 에러.
- commit: `feat(network-net): dialNode ssh-remote branch + ssh-password consumption`

## Task 5 — providerCaps 정직성 (D2)
- `handlers_network.go providerCaps["ssh-remote"]` → `{"rpc","ws"}`. 주석: 5b.1 read-only; fs/process 는 5b.2.
- 영향 테스트(있으면) 갱신 + rationale.
- commit: `fix(network-net): align ssh-remote capabilities to implemented {rpc,ws}`

## Task 6 — SSH 통합 테스트 + redaction
- `drivers/sshremote/sshremote_test.go`: in-process SSH 서버(`ssh.NewServerConn`, password callback, `direct-tcpip` 채널 수락 → mock httptest JSON-RPC 로 io.Copy 양방향). 케이스:
  - happy: Dial→BlockNumber/ChainID 터널 경유 성공.
  - bad password → dial 실패(에러에 password 없음).
  - host key mismatch (knownhosts 경로) → 거부.
  - insecure opt-in → 통과.
- 보안 negative: password 문자열이 에러/로그에 안 나타남(grep).
- commit: `test(sshremote): in-process SSH server integration + redaction`

## Task 7 — 문서 + 버전
- VISION 로드맵 Sprint 5b 체크박스: 5b.1 완료 표기(5b.2 잔여).
- REMAINING_WORK §4 Priority 2 (Sprint 5b): 5b.1 완료 처리, 5b.2 잔여 명시.
- 예제: `examples/networks/` 에 ssh-remote 노드 예제 + README 보강(env-only password, host key 정책, 수동 구성).
- 버전 bump 0.8.0 → 0.9.0 (minor — 신규 provider read 경로).
- commit: `docs+chore(sprint-5b-1): roadmap + ssh-remote example + version 0.9.0`

---

## 완료 기준
- [ ] ssh-remote 노드에서 read RPC 가 SSH 터널 경유 동작 (mock SSH 서버 통합 테스트)
- [ ] password env-only, 에러/로그에 값 미노출 (negative 테스트)
- [ ] host key: known_hosts 기본 + insecure opt-in
- [ ] providerCaps ssh-remote = {rpc,ws} (정직)
- [ ] 스키마 env 필드 + 재생성 커밋 포함
- [ ] 전 레이어 green: Go(전 패키지, vet/gofmt) · vitest · bash
- [ ] go.mod: x/crypto direct, tidy clean
