# Chainbench Key Handling Security Policy

> Last updated: 2026-04-24 (Sprint 4 env-signer)

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

## Key Injection (current — env only)

Private keys enter via env vars at chainbench-net spawn time:

```
CHAINBENCH_SIGNER_<ALIAS>_KEY=0x<64-hex-chars>
```

Where `<ALIAS>` matches `[A-Za-z][A-Za-z0-9_]*` (POSIX-identifier shape so
the resulting env var name is accepted by common deployment tooling —
shells, docker `-e`, systemd EnvironmentFile, Kubernetes ConfigMap).

Commands reference the alias via `signer: "<alias>"`; the handler does
`signer.Load(alias)` → env read → in-memory Signer. On process exit, the
OS reclaims the memory.

### Never

- Do NOT commit the env value to git, shell rc files, or CI configs where
  history is not purged.
- Do NOT place the key in the network state file. State files hold only
  aliases.
- Do NOT send the env file over the network (LLM chat, Slack, etc.).

## Key Injection (planned — Sprint 4b)

Keystore provider: `CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE` + `_KEYSTORE_PASSWORD`
env pair. Same alias model; on load, decrypt keystore file in memory.

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

1. **Unit — `network/internal/signer/signer_test.go`** — verifies redaction
   across `%v`, `%+v`, `%#v`, `%s`, `slog.TextHandler`, `slog.JSONHandler`,
   `fmt.Errorf("%v", s)`, and deep-nested containers. Also asserts error
   messages never contain the raw hex (including a probe test with a
   valid-length but cryptographically-invalid key).

2. **End-to-end — `tests/unit/tests/security-key-boundary.sh`** — spawns
   the actual chainbench-net binary with `CHAINBENCH_SIGNER_ALICE_KEY=...`
   set in the environment, performs a `node.tx_send`, and case-insensitively
   greps stdout, stderr, and the log file for the raw key hex. Any match
   fails the test with exit 1.

Both must stay green for any PR touching signer code or any handler that
accepts a signer alias.

## Out of Scope (current sprint)

- HSM / hardware-wallet integration
- Multi-party signing / threshold keys
- Key derivation from seed phrases inside chainbench — assume operator
  derives externally and exports keys as env
- Audit logging of sign operations — would need redaction patterns for
  the alias/address pair and a separate audit stream
- EIP-1559 dynamic-fee tx signing (Sprint 4b)
- Tx confirmation polling (Sprint 4b or later)

## Known Latent Issue

`CHAINBENCH_NET_LOG` env var is documented in `network/README.md` but is
not honored by the `run` subcommand path — `cmd/chainbench-net/run.go`
always routes logs to stderr. The security-key-boundary test works
correctly (it scans stderr, which is where logs actually go), but the
env var appears to have no effect. Tracked as a session-local follow-up
to be addressed in a later sprint.
