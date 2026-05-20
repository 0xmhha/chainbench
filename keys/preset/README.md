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

## How chainbench consumes these

`profiles/default.yaml` references this directory via:

```yaml
keys:
  mode: static
  source: "keys/preset"
```

`chainbench init` copies `password` and the per-node keystore/nodekey files
into the runtime data directory (`data/node-N/`) before starting `gstable`.
