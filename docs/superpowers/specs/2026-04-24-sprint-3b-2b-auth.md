# Sprint 3b.2b — Remote RPC Auth Design Spec (Minimal)

> 2026-04-24 · Sub-sprint of VISION §6 Sprint 3
> Scope: **lean, scaffolded**. Implement the auth machinery so 3b.2c and beyond
> can opt-in; wire through one handler (`node.block_number`) end-to-end for
> verification, but do not prioritize comprehensive handler integration.

## 1. Goal

Add http.RoundTripper-based authentication (API key / JWT) for `remote.Client`
so chainbench can attach to protected RPC endpoints (Infura with key, private
node behind API gateway, etc.). Auth material never appears in stdout/stderr
logs.

## 2. Non-Goals (Deferred)

- **SSH-password auth** — belongs to `SSHRemoteDriver`, not RPC client (future).
- **Signer (tx signing keys)** — Sprint 4. Different key class.
- **Keystore / encrypted files** — env vars only in 3b.2b.
- **3b.2c's command expansion** — only `node.block_number` uses auth here;
  `node.balance`, `node.chain_id`, etc. arrive in 3b.2c.
- **Retry / auth refresh** — if a JWT expires mid-session, caller must re-attach.

## 3. User-Facing Surface

### 3.1 `network.attach` gains optional `auth` arg

```json
{
  "command": "network.attach",
  "args": {
    "rpc_url": "https://mainnet.infura.io/v3/<key>",
    "name": "infura",
    "auth": {
      "type": "api-key",
      "header": "Authorization",
      "env": "INFURA_KEY"
    }
  }
}
```

Supported auth types (discriminated union — matches `network.json` schema):

| type | required fields | behavior |
|------|-----------------|----------|
| `api-key` | `env` (string), optional `header` (default `"Authorization"`) | Reads `$env` and sets request header `<header>: <value>` |
| `jwt` | `env` (string) | Reads `$env` and sets `Authorization: Bearer <token>` |

Omitting `auth` means unauthenticated (current 3b.2a behavior). `auth` is
persisted on `Node.Auth` in `state/networks/<name>.json`.

### 3.2 `node.block_number` — auth becomes automatic

No arg changes. When the resolved node has `node.Auth != nil`, the handler
uses `remote.AuthFromNode(node, os.Getenv)` to construct a RoundTripper and
passes it to `remote.DialWithOptions`. Unauthenticated nodes take the same
`remote.Dial` path as before.

### 3.3 Errors

- `INVALID_ARGS` — `auth.type` unknown, missing required field (`env`).
- `UPSTREAM_ERROR` — env var empty (could also be INVALID_ARGS; picking
  UPSTREAM because the caller's intent was correct but the deployment
  environment is misconfigured).
- `UPSTREAM_ERROR` — server returns 401/403 via ethclient (it surfaces as
  generic RPC error today; separate 401/403 code is a 3b.2c polish).

## 4. Auth Material Handling

- Env vars only. No keystore. No CLI `--key` flag.
- Auth values never appear in `slog` output. The Transport wrapper must not
  log request bodies or headers.
- Error messages reference env var **name**, never value. E.g., `"auth env
  INFURA_KEY is empty"` is fine; `"header set to Bearer ey..."` is not.
- State file (`networks/<name>.json`) stores `Node.Auth` **without the value**
  — only the env var name. Anyone reading the state file learns which env
  var to set, not the key itself.

## 5. Package Layout

```
network/internal/drivers/remote/
├── client.go        # existing; add DialWithOptions
├── auth.go          # new — APIKeyTransport, BearerTokenTransport,
│                    #   AuthFromNode helpers
├── auth_test.go     # new — header injection, missing-env, redaction checks
└── client_test.go   # existing; new case for DialWithOptions
```

### 5.1 Types

```go
package remote

type DialOptions struct {
    Transport http.RoundTripper // nil = default
}

func DialWithOptions(ctx context.Context, url string, opts DialOptions) (*Client, error)

// Existing Dial becomes a wrapper:
func Dial(ctx context.Context, url string) (*Client, error) {
    return DialWithOptions(ctx, url, DialOptions{})
}

// APIKeyTransport wraps base, injecting a header on every request.
// base may be nil (http.DefaultTransport is used).
func APIKeyTransport(base http.RoundTripper, header, value string) http.RoundTripper

// BearerTokenTransport wraps base, injecting Authorization: Bearer <token>.
func BearerTokenTransport(base http.RoundTripper, token string) http.RoundTripper

// AuthFromNode inspects node.Auth and constructs a RoundTripper via
// env lookup. Returns nil, nil if node.Auth is nil (unauthenticated).
// envLookup is injected for testability (real callers pass os.Getenv).
func AuthFromNode(node *types.Node, envLookup func(string) string) (http.RoundTripper, error)
```

### 5.2 `ethclient` integration

`ethclient.NewClient(rpc.NewClient(...))` accepts a pre-configured rpc.Client.
go-ethereum's `rpc.DialOptions` supports `rpc.WithHTTPClient(*http.Client)`.
`DialWithOptions` constructs an `http.Client{Transport: opts.Transport}` and
passes it via `rpc.DialOptions`.

## 6. Testing Strategy

**Unit (`auth_test.go`):**
- `APIKeyTransport`: request made through it includes the header.
- `BearerTokenTransport`: request includes `Authorization: Bearer <token>`.
- `AuthFromNode`: nil auth → nil, nil.
- `AuthFromNode`: api-key with env lookup → Transport produces the right header.
- `AuthFromNode`: jwt → Bearer Transport.
- `AuthFromNode`: empty env value → error referencing env name (no value).
- `AuthFromNode`: unknown auth type → error.

**Unit (`client_test.go` new case):**
- `DialWithOptions` with Transport injected → httptest server sees the header.

**Unit (`handlers_test.go`):**
- `node.block_number` against a httptest mock that requires `Authorization`;
  state has a node with `Auth{type:"api-key", env:"TEST_KEY"}`; env set via
  `t.Setenv("TEST_KEY", "secret")`; result returns block_number.

**Go E2E (`e2e_test.go`):**
- Attach with auth arg (via env), then block_number against the protected mock.

**Bash E2E (`tests/unit/tests/node-block-number-auth.sh`):**
- Python mock checks Authorization header; cb_net_call attach with auth,
  then block_number; assert result.

## 7. Schema

`command.json` — `network.attach` args shape is already `"type":"object"` with
no inner schema (wire-only enforcement at handler). No enum change.

`network.json` already declares `Auth` — no schema edit.

## 8. Package Boundaries

- `remote` package gains `auth.go` but does not reach into `state` or `types`
  except through a function parameter (`*types.Node`). Low coupling preserved.
- `types.Node.Auth` already exists in generated types. Handler passes it in
  to `AuthFromNode`.

## 9. Out-of-Scope Reminders (session-local)

**Still deferred from earlier sprints:**
- 2b.3 M3 — APIError.Details
- 2c M3 — jq 3→1
- 2c M4 — jq version gate
- 3a minor — isKnownOverride SSoT
- 3b minor — created flag TOCTOU
- 3b.2a M-3 — 401/403 as distinct APIError code (belongs here or 3b.2c polish)
- 3b.2a I-2 — Node.Http empty guard (still deferred; when 3b.2c adds IPC)

**New deferrals (introduced by 3b.2b scope):**
- Auth key rotation, expiry detection, refresh flow
- mTLS (client certs) auth
- Auth on WebSocket endpoints (ws:// Transport wrapping — 3b.2c if needed)
- Multiple auth providers per network (all nodes share node.Auth today)
