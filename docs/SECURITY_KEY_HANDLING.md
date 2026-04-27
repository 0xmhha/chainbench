# Chainbench Key Handling Security Policy

> Last updated: 2026-04-27 (Sprint 4b — keystore + EIP-1559 + tx.wait)

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
- [ ] Verify on every release that `tests/unit/tests/security-key-boundary.sh`
      is part of the required-green CI suite.

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

## Boundary Enforcement

Two tests enforce the boundary:

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
   literal. Any match fails the test with exit 1.

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
