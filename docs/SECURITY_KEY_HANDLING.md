# Chainbench Key Handling Security Policy

> Last updated: 2026-04-27 (Sprint 4c — SignHash + EIP-7702 + Fee Delegation)

## Threat Model

Chainbench-net signs transactions locally with private keys supplied by the
operator's deployment environment. The threat model assumes:

- The host machine is trusted — operators don't run chainbench-net on
  adversarial hardware.
- Other processes on the same host may observe stdout, stderr, or log files.
- Remote RPCs MUST NEVER see plaintext key material — they receive only
  signed transactions.
- Any code path inside chainbench-net that could serialize a signer to
  stdout / stderr / log / disk is a contract violation.

## Key Injection (current — env-key OR keystore)

Private keys enter via env vars at chainbench-net spawn time. Two
provider paths are supported; both share the same alias model and the
same in-memory `signer.Signer` representation downstream.

**Path A — Raw key env (Sprint 4)**:

```
CHAINBENCH_SIGNER_<ALIAS>_KEY=0x<64-hex-chars>
```

**Path B — Keystore env pair (Sprint 4b)**:

```
CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE=/path/to/keystore.json
CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE_PASSWORD=<password>
```

Where `<ALIAS>` matches `[A-Za-z][A-Za-z0-9_]*` (POSIX-identifier shape,
leading letter required; the regex applies identically to both paths so
the resulting env var name is accepted by common deployment tooling —
shells, docker `-e`, systemd EnvironmentFile, Kubernetes ConfigMap).

**Resolution order** (`signer.Load`): if `_KEY` is set it wins; the
`_KEYSTORE`/`_KEYSTORE_PASSWORD` pair is consulted only when `_KEY` is
absent. This lets operators pin a deterministic test key without
removing the keystore env. If neither path is set → `ErrUnknownAlias`.

Commands reference the alias via `signer: "<alias>"`; the handler does
`signer.Load(alias)` → env read (raw `_KEY` first, then keystore
decrypt) → in-memory Signer. On process exit, the OS reclaims the
memory.

### Never

- Do NOT commit the env value to git, shell rc files, or CI configs where
  history is not purged.
- Do NOT place the key in the network state file. State files hold only
  aliases.
- Do NOT send the env file over the network (LLM chat, Slack, etc.).

### Operator Notes

- Keystore file permissions should be `0600` (owner read/write only).
  `chainbench-net` does NOT enforce this — the check is informational.
  Deployment tooling (Ansible, systemd, k8s init container, etc.) is
  responsible for ensuring tight permissions before chainbench-net is
  spawned.
- The decrypted key never leaves the `signer` package. Decryption
  happens once per `signer.Load` call; the password env var is read
  once via `os.Getenv` and is never cached on disk, in process state,
  or in any returned struct.
- Error messages from `signer.Load` reference the alias and env var
  name only — the keystore file path bytes, file content, and password
  value are NEVER embedded in error strings (verified by the
  `RedactionBoundary` and error-probe tests for the keystore variant).

## Operator Checklist

- [ ] Rotate keys on any suspected exposure — env history, error dumps,
      screenshots.
- [ ] Prefer short-lived test keys over long-lived production keys.
- [ ] Scrub `.env` files from CI artifact uploads and dev-machine backups.
- [ ] Verify on every release that the two boundary tests above stay in the
      required-green CI suite (`go test ./internal/core/keyring/...
      ./internal/core/session/...`). The end-to-end shell check they replaced
      went away with the legacy bash suite in `#43`.

## Developer Contract

- Key material lives ONLY in the `network/internal/signer` package's sealed
  struct. No method exposes the key bytes.
- The sealed struct implements:
  - `slog.LogValuer` → redacts to `"***"` in structured logs
  - `fmt.Stringer` → redacts to `"<signer:***>"` for `%s` and `%v`
  - `fmt.GoStringer` → redacts to `"<signer:***>"` for `%#v`
- `encoding/json` produces `{}` for a sealed struct (all fields unexported
  and no custom `MarshalJSON`).
- Error messages from the signer package reference the **alias** and the
  **env var name** only — never the key value or substrings of it.
- Alias regex rejects leading digit, leading underscore, leading hyphen,
  special characters, and all Unicode.
- New code that handles a `Signer` must pass:
  ```
  grep -rn 'signer\..*key\|privateKey\|PrivateKey' network/
  ```
  No export / serialization / log path may surface.
- `SignHash` returns 65 raw bytes; callers (handlers) compose into V/R/S
  without ever touching the underlying private key. Used for EIP-7702
  authorization tuples and go-stablenet 0x16 fee-payer outer hash. Same
  redaction guarantees as `Sign` — the sealed struct never exposes key
  material on any logging / serialization path.

## 폐쇄망 비밀은 server-set 이 가진다

> **경로 정정 (2026-10-01).** `tests/env/` 가 이 자리에 있었다. 폐쇄망 SSH 자격증명을
> `tests/env/secret/closednet.ssh` 에, 노드 주소를 `closednet.remotes`·`closednet.hosts`
> 에 두고, `secret.example/` 이 그 형식을 보였다. 그 디렉터리는 2026-10-01 에 지웠다 —
> 읽는 코드가 하나도 남아 있지 않았고, 담던 것을 전부 server-set 이 가져갔기 때문이다.

지금 원격 서버의 주소와 자격증명은 **server-set 파일 하나**에 있다.

| 담는 것 | 어디에 |
|---|---|
| SSH 사용자·포트 | `ssh.user`, `ssh.port` |
| SSH 비밀번호 | `ssh.password_file`(한 줄, 0600) 또는 `ssh.password` |
| SSH 키와 그 암호 | `ssh.key_file`, `ssh.key_passphrase_file` |
| 노드가 도는 기계의 주소 | `pool.hosts` |

경계는 `.gitignore` 가 긋는다. `server-set*.yaml` 과 `workspace-config*.yaml` 은
전부 무시되고, 추적되는 것은 값이 없는 `server-set.sample.yaml` 뿐이다. 그 파일
머리말이 같은 말을 한다 — "NEVER commit the real file."

규칙은 그대로다. 실제 자격증명은 저장소 밖에 두고, 값을 `cat`·`echo`·로그·전송
어디에도 내놓지 않으며, 리터럴로 커밋되거나 출력에 보이면 즉시 교체한다.

## Boundary Enforcement

> **경로 정정 (2026-09-29).** 아래 두 이름은 그때의 트리다. `network/internal/signer/`
> 는 `internal/core/keyring/` 이 되었고, `tests/unit/` 의 셸 스위트는 `#43` 에서
> 레거시 bash 스택과 함께 없어졌다. `RedactionBoundary` 라는 타입도 코드에 없다.
> 지금 경계를 지키는 것은 아래 둘이다.

오늘 경계를 지키는 테스트:

1. **단위 — `internal/core/keyring/keyring_test.go`** —
   `TestNodekey_DoesNotLeakWhenFormatted` 가 `%v`·`%+v`·`%#v`·`%s` 와 `slog` 출력에서
   비밀이 `redacted` 로 나오는지 본다. 비밀 자체는
   `internal/core/keyring/derive/privatekey.go` 가 갖고, `String`·`GoString` 이
   `keyring.PrivateKey(redacted)` 만 내놓는다.
2. **증적 — `internal/core/session/scrub_test.go`** —
   `TestScrub` 가 실행 증적을 쓰기 전에 비밀을 지우는지 본다. 대상은
   `internal/core/session/scrub.go` 의 두 규칙이다: JSON 필드
   (`key`·`privateKey`·`saveKey`·`password`·`mnemonic`·`secret`)와 기록된 명령줄의
   `--password`/`--unlock` 값. 해시·주소는 건드리지 않는다.

둘 다 signer 코드나 signer 별칭을 받는 핸들러를 건드리는 PR 에서 녹색이어야 한다.

### 그때의 기록 (2026-04, 경로는 위 정정을 따른다)

1. **Unit — `network/internal/signer/signer_test.go`** — verifies
   redaction across `%v`, `%+v`, `%#v`, `%s`, `slog.TextHandler`,
   `slog.JSONHandler`, `fmt.Errorf("%v", s)`, and deep-nested
   containers. Also asserts error messages never contain the raw hex
   (including a probe test with a valid-length but
   cryptographically-invalid key). Sprint 4b added 6 keystore cases
   covering the env-pair resolution, decryption errors, missing
   password, malformed keystore JSON, raw-key-wins-over-keystore
   precedence, and `RedactionBoundary` for a keystore-loaded `*sealed`
   (same `***` / `<signer:***>` outputs as the raw-key path).

2. **End-to-end — `tests/unit/tests/security-key-boundary.sh`** —
   spawns the actual chainbench-net binary and performs a
   `node.tx_send` against a Python JSON-RPC mock, then
   case-insensitively greps stdout, stderr, and the log file for any
   key-shaped substring. Sprint 4b extended the script with a
   **Scenario 3 (keystore variant)**: it generates a per-run keystore
   file + random password, loads signer "bob" via the
   `_KEYSTORE`/`_KEYSTORE_PASSWORD` env pair, runs `node.tx_send`,
   and greps for BOTH the underlying raw hex AND the password
   literal. Sprint 4c added **Scenario 4 (fee-delegation
   variant)**: a `node.tx_fee_delegation_send` with two distinct
   signers (sender raw `_KEY` for "carol" + fee_payer keystore for
   "bob", reused from Scenario 3). Asserts none of {sender hex,
   fee_payer raw hex, keystore password} surfaces in stdout / stderr
   / log. Any match fails the test with exit 1.

Both must stay green for any PR touching signer code or any handler
that accepts a signer alias.

## Out of Scope (current sprint)

- HSM / hardware-wallet integration
- Multi-party signing / threshold keys
- Key derivation from seed phrases inside chainbench — assume operator
  derives externally and exports keys as env
- Audit logging of sign operations — would need redaction patterns for
  the alias/address pair and a separate audit stream

## Resolved Latent Issues

- **CHAINBENCH_NET_LOG / CHAINBENCH_NET_LOG_LEVEL** (resolved 2026-04-27):
  the `run` subcommand previously hard-wired logs to stderr regardless of
  env. `runOnce` now uses `wire.SetupLoggerWithFallback(stderr)` which
  routes to the env-configured file when set and falls back to the
  injected stderr writer otherwise. The security-key-boundary test
  remains effective: when the env var redirects logs to a file, the test
  scans the file as well as stderr (the file is part of the audit
  surface).
