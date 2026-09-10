# Generating preset key sets (`validator set`)

> **[가이드]** 검증 기준 2026-09-11 · `cmd/chainbench/keyringcmd/validator.go:209`.
> 이 문서의 명령·플래그는 `chainbench validator set --help` 가 이긴다.
>
> Renamed: this preset/validator-set generator is now `chainbench validator set`
> (was `chainbench keys generate`). A preset is defined by its validator set, so
> it lives under `validator`. Raw keypair primitives live under `keyring`
> (`keyring new` / `keyring add`) — there is no top-level `keys` command group.

The committed `keys/preset` ships 5 nodes, which caps a local network at 5. Some
cases need more (e.g. the n=6 WBFT quorum tests). `chainbench validator set`
produces a preset of any size that `store.LoadPreset` (and every `chain` step)
consumes.

## What it produces

For each node it generates a random nodekey and derives:

- the **address** + **BLS public key**/**PoP** — derived **in process**
  (`internal/core/keyring/derive`); no external `bootnode` binary is involved,
- the devp2p **public key**,
- an encrypted **keystore** — via the accounts SDK; no node binary is involved,

then writes a `metadata.json` (validators, aligned BLS keys, alloc, system-contract
members, nodes) and a per-node dir. Crucially, the croissant/WBFT validator set
lives in the **genesis config** (`croissant.init.validators`), not in the header
`extraData`, so `extraData` is a plain 32-byte vanity — **no istanbul RLP encoding
is needed**, which is what makes generating a working preset tractable.

The generated metadata carries **no enode**. Enodes are assembled at compose time
from the public key and the node's actual host and port, which is the only place
both are known.

## Use

```sh
chainbench validator set \
  --nodes 6 --validators 6 \
  --out /tmp/preset6

chainbench chain up --chain wbft --binary <gwemix> \
  --keys /tmp/preset6 --validators 6
```

Flags: `--nodes` (total, required), `--out` (required), `--validators` (default all),
`--password` (default `1`), `--balance`. `--base-p2p` is accepted but deprecated
and ignored.

The gated e2e harness uses this to build networks larger than the committed
preset — see `tests/e2e/wbft_fault_test.go` (`genPreset`).
