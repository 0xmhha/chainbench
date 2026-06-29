# Sprint 5b.2 — SSHRemoteDriver (process + fs capability)

> 작성일: 2026-06-29
> 상태: SPEC (검토 대기)
> 선행: Sprint 5b.1 (SSH 터널 read-only RPC, `drivers/sshremote`), VISION §5.4 (provider capability), §5.16 S6
> 짝 plan: `docs/superpowers/plans/2026-06-29-sprint-5b-2-sshremote-process-fs.md`
> 확정 결정: process 제어 = **ProviderMeta 명령 설정 모델** (사용자 사인오프 2026-06-29).

---

## 1. Goal

`ssh-remote` provider 노드에 **process**(node stop/start/restart)와 **fs**(tail_log) capability 를 SSH shell exec 로 부여한다. 5b.1 의 SSH 연결·자격증명·host key 인프라를 재사용하고, 드라이버에 **Exec**(원격 명령 실행)을 추가한다.

완료 시 `providerCaps["ssh-remote"]` = `{fs, process, rpc, ws}` (5b.1 의 `{rpc, ws}` 에서 복원) — ssh-remote 가 capability 면에서 local 에 근접(admin/network-topology 제외).

---

## 2. process 제어 모델 (확정)

원격 호스트의 노드 lifecycle 명령을 **노드의 `provider_meta` 에 설정**하고 SSH 로 exec 한다. 원격 툴링(chainbench/systemctl/수동 스크립트)을 가정하지 않는 가장 일반적 모델.

```json
{
  "id": "node1", "provider": "ssh-remote", "http": "http://127.0.0.1:8545",
  "auth": { "type": "ssh-password", "user": "deploy", "host": "10.0.0.42", "env": "CHAINBENCH_SSH_PASSWORD" },
  "provider_meta": {
    "log_file":    "/var/lib/gstable/node.log",
    "start_cmd":   "systemctl start gstable",
    "stop_cmd":    "systemctl stop gstable",
    "restart_cmd": "systemctl restart gstable"
  }
}
```

- stop/start/restart → 해당 `*_cmd` 가 없으면 `NOT_SUPPORTED`(해당 노드가 그 동작을 선언 안 함).
- restart 는 `restart_cmd` 가 있으면 그것을, 없으면 stop_cmd→start_cmd 합성(5b.1 lifecycle 합성 패턴) — 둘 다 없으면 NOT_SUPPORTED.
- **명령은 operator 가 networks 파일에 직접 쓴 신뢰 입력**(자기 인프라). tail_log 의 logfile/lines 처럼 caller-uncontrolled. 문서에 명시.

---

## 3. fs 제어 (tail_log)

`provider_meta.log_file` 경로를 SSH 로 `tail` 한다: `tail -n <lines> -- <log_file>`.
- `lines` 는 핸들러에서 검증된 정수(1..1000, 기존 제약 재사용), `--` 로 옵션 주입 차단, log_file 은 operator-set.
- log_file 미설정 → `UPSTREAM_ERROR`(기존 local 동작과 동일 분류) 또는 `NOT_SUPPORTED`(fs 미선언) — §6 D2.

---

## 4. 드라이버 확장 (sshremote)

```go
// Exec runs a single command on the remote host over SSH and captures output.
// Reuses Credentials + host key policy from 5b.1. Errors never include the password.
type ExecResult struct { Stdout, Stderr string; ExitCode int }
func Exec(ctx context.Context, creds Credentials, hostKey ssh.HostKeyCallback, command string) (ExecResult, error)
```
- `ssh.Dial` → `client.NewSession()` → `session.Run(command)`; stdout/stderr 버퍼 캡처; `*ssh.ExitError` → ExitCode 추출(비-zero 는 에러 아님, ExecResult.ExitCode 로 전달 — 핸들러가 분류).
- dial/세션 실패만 error. 연결은 매 호출 spawn(S2 일관) — 단발 명령에 적합.

---

## 5. 핸들러 변경 (provider-aware lifecycle)

현재 stop/start/restart/tail_log 는 local 하드게이트 + `resolveNodeID`(pids 번호). 일반화:

- node 를 **network-aware** 로 해석: `network=="local"|""` → 기존 local 경로(`resolveNodeID` + LocalDriver) 무변경. 그 외 → `resolveNode`(types.Node + Provider + Auth + ProviderMeta).
- 비-local 노드 분기:
  - `provider=="ssh-remote"` → SSH exec 경로(provider_meta 명령 / tail).
  - `provider=="remote"` → `NOT_SUPPORTED`(remote 는 process/fs 없음 — capability 와 일관).
- ssh-remote 자격증명 추출은 5b.1 `dialSSHNode` 의 ssh-password 파싱 로직 재사용(공통 헬퍼로 추출: `sshCredsFromNode(node) (Credentials, error)`).

이벤트(node.stopped/started)·결과 shape·에러 분류는 기존과 동일 유지.

---

## 6. 결정 (Decisions)

- **D1 — 명령은 operator 신뢰 입력.** provider_meta 명령은 networks 파일 작성자(operator)가 제공 → SSH 셸에서 실행. caller(LLM)가 임의 명령을 주입하는 표면 아님(node_id 만 선택). 문서에 신뢰 경계 명시.
- **D2 — capability 정직성(5b.1 패턴 계승).** `providerCaps["ssh-remote"]` = `{fs, process, rpc, ws}`. 단, **개별 노드가 해당 명령/log_file 을 설정 안 했으면 런타임에 `NOT_SUPPORTED`** — capability 는 provider 레벨 상한, 노드 레벨 미설정은 런타임 거부. (capability 집합은 provider 단위라 노드별 차등은 표현 못 함 — 이 한계 문서화.)
- **D3 — password env-only(5b.1 계승).** Exec 에러에 password 미포함, env var 이름만.
- **D4 — restart 합성.** restart_cmd 있으면 단일 명령, 없으면 stop→start 합성(5b.1 local restart 이벤트 순서 불변 계승).

---

## 7. Security Contract

- 5b.1 의 SSH 보안 경계(password env-only/미저장/미로깅, host key known_hosts 기본+insecure opt-in) 전부 계승.
- SSH exec 명령 문자열은 operator 제공 — `bash -c` 동등의 원격 셸 실행. caller-controlled 부분(node_id, lines)은 명령에 직접 보간하지 않음(node_id 는 node 선택용, lines 는 검증 정수 + `--`).
- 명령 stdout/stderr 는 caller 에 반환(노드 로그/명령 출력 — 시크릿 아님). signer 키 경계 불변.

---

## 8. Tests

1. **sshremote.Exec 유닛** — in-process SSH 서버에 **session+exec 채널 처리 추가**(5b.1 서버는 direct-tcpip 만). exec 성공(stdout/exit 0), 비-zero exit(ExitCode 전달), bad password(누출 없음).
2. **핸들러** — ssh-remote 노드 stop/start/restart 가 provider_meta 명령을 SSH exec(mock 서버가 명령 echo/exit); tail_log 가 log_file 을 tail. 명령 미설정 → NOT_SUPPORTED. 비-zero exit → UPSTREAM. remote provider → NOT_SUPPORTED. local 경로 무회귀.
3. **capability** — ssh-remote = {fs,process,rpc,ws} 단언. hybrid(local+ssh-remote) 교집합 갱신 확인.
4. **보안 negative** — password 미노출(grep).
5. 회귀: Go 전 패키지 · vitest · bash green.

---

## 9. Error Classification

| 코드 | 경우 |
|---|---|
| `INVALID_ARGS` | ssh-password auth 불완전(5b.1), lines 범위 밖 |
| `NOT_SUPPORTED` | remote provider 의 process/fs; ssh-remote 노드에 해당 *_cmd/log_file 미설정 |
| `UPSTREAM_ERROR` | SSH dial/exec 실패, 명령 비-zero exit, env var 빔 |

---

## 10. Out-of-Scope / 후속

- ssh-remote 구성 명령(attach 확장) — 여전히 수동 networks 파일(v1). 별도.
- 키 인증/키체인(S6 후속).
- `network.start_all/stop_all`(네트워크 단위) 의 ssh-remote 반영 — node 단위 우선, 네트워크 단위는 후속.
- streaming tail (`tail -f`) — 단발 tail 만(subscription 표면 미도입).

---

## 11. 예상 커밋 (~7-9)

1. `docs: add Sprint 5b.2 spec + plan`
2. `feat(sshremote): add Exec for remote command execution`
3. `refactor(network-net): extract sshCredsFromNode helper` (5b.1 dialSSHNode 공통화)
4. `feat(network-net): node.tail_log over SSH for ssh-remote (fs)`
5. `feat(network-net): node stop/start/restart over SSH for ssh-remote (process)`
6. `fix(network-net): restore ssh-remote capabilities to {fs,process,rpc,ws}`
7. `test(sshremote): SSH exec server harness + process/fs handler coverage`
8. `docs+chore(sprint-5b-2): roadmap + example + version bump`
