# Preset Keys — TEST FIXTURE ONLY

> ⚠️ **DO NOT IMPORT THESE KEYS INTO ANY NON-LOCAL ENVIRONMENT.**
>
> The files in this directory are intentionally committed to git so that
> local test runs produce reproducible validator addresses, genesis blocks,
> and enode URLs.
>
> They MUST be treated as public:
> - The keystore password is the single character `1` (see `password`).
> - The `nodekey` files are plaintext secp256k1 private keys.
> - The `keystore/UTC--*` files use scrypt KDF with the same `1` password,
>   so the standard KDF protection is effectively zero given the colocated
>   password file.
> - Every enode in `metadata.json` binds to `127.0.0.1`.
>
> If any of these keys appear on a public RPC endpoint with a non-zero
> balance, anyone observing this repository can drain the account
> immediately. Likewise, if any validator slot in a real network is ever
> bound to one of these addresses, anyone can forge votes from it.
>
> Use `keys.mode: generate` in your profile (or override the `source` to a
> directory outside of git) when you need keys that are not public.

## Contents

| File | Purpose |
|---|---|
| `metadata.json` | Bundle metadata: validators, BLS public keys, extraData, alloc, system contract config. Also includes the plaintext `nodekey` for each node. |
| `password` | The plaintext keystore password (`1`). |
| `node{1..5}/address` | Validator/EN account address (public). |
| `node{1..5}/pubkey` | secp256k1 public key (public). |
| `node{1..5}/bls_pubkey` | BLS public key (public). |
| `node{1..5}/nodekey` | secp256k1 **private** key (test-only, public-equivalent). |
| `node{1..4}/keystore/UTC--*` | Ethereum keystore (encrypted with password `1`). |

## Funding a test account

Every `node{1..5}` account is in the genesis `alloc` with a large balance, so a
test that needs to spend funds signs with one of those node keys (the DSL's
`sendTx from: <node address>` does this through the session keyring; the
remaining Go-func cases take node1's key from this preset). There is no
separate faucet key to keep anywhere.

`metadata.json` `alloc` also carries one extra account
(`0x71562b71999873db5b286df957af199ec94617f7`) that nothing spends from: tests
read it as a prealloc balance that must survive a fork or a re-sync.

## How chainbench consumes these

This directory is the built-in default, not a special case: the config defaults
in `internal/core/config` are `keys.mode: static` and `keys.source: keys/preset`,
and a profile under `profiles/` overrides them.

```yaml
keys:
  mode: static
  source: "keys/preset"
```

The `keys` step of a network resolves that source
(`chainbench net keys --keys-source preset`, or `chainbench net up`, which runs
the steps in order). Genesis is built from
`metadata.json`, and the launcher ships each node's identity — `password`,
`keystore/`, `nodekey` — into that run's data directory before starting the node,
locally or over the file seam for a remote node.
