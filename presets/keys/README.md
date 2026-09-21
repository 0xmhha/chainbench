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
> **NEVER use anything in this directory on a production, mainnet, testnet,
> staging, or shared network. Not to hold value, not to seal a block, not to
> sign anything, not once, not "just to try it."** This is not a precaution
> against a key leaking — the key is already published, here, in this file
> tree. Rotating it afterwards does not undo what was signed in the meantime.
>
> The rule is the same for every file here, whatever produced it: the node
> identities under `node{1..5}/`, generated once and committed, and the dev
> accounts under `dev1/`, minted by the harness on its first run and committed
> so a label keeps naming one address.
>
> Use `keys.mode: generate` in your profile (or override the `source` to a
> directory outside of git) when you need keys that are not public.
>
> The repository's secret scanner is configured to stay quiet about this
> directory ([`.betterleaks.toml`](../../.betterleaks.toml)), because every
> finding here would be a true positive about a key that is public on purpose.
> That silence is scoped to these paths and nowhere else.

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
| `dev1/address` | Dev account address (public). |
| `dev1/private` | secp256k1 **private** key (test-only, public-equivalent). |

## Funding a test account

Every `node{1..5}` account is in the genesis `alloc` with a large balance, so a
test that needs to spend funds signs with one of those node keys (the DSL's
`sendTx from: <node address>` does this through the session keyring; the
remaining Go-func cases take node1's key from this preset). There is no
separate faucet key to keep anywhere.

`metadata.json` `alloc` also carries one extra account
(`0x71562b71999873db5b286df957af199ec94617f7`) that nothing spends from: tests
read it as a prealloc balance that must survive a fork or a re-sync.

## The `dev1` account

A spec that declares `accounts: {"dev1": {"fund": …}}` gets an account the
harness holds the key for and signs with itself, funded at bring-up from the
network's funded account. It is not in the genesis `alloc` — it starts at zero
and the run funds it.

`accountSource` (`internal/testengine/accounts.go`) mints the key on the first
run that asks for one and reads it back afterwards, so the label keeps naming
the same address. Committing it extends that from one machine to all of them:
without the file, `dev1` is a different address in every checkout, which is the
drift labels exist to remove. It is a test fixture on exactly the terms above —
a plaintext private key, public-equivalent, never to be funded anywhere real.

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
