# Generating preset key sets (`validator set`)

> **[가이드]** 검증 기준 2026-09-30 · `cmd/chainbench/keyringcmd/validator.go:210`.
> 이 문서의 명령·플래그는 `chainbench validator set --help` 가 이긴다.
>
> Renamed: this preset/validator-set generator is now `chainbench validator set`
> (was `chainbench keys generate`). A preset is defined by its validator set, so
> it lives under `validator`. General key sets live under `keyring`
> (`keyring new` / `keyring add`; `keyring new --with-bls` builds the same
> shape) — there is no top-level `keys` command group.

The committed `presets/keys` ships 5 nodes, which caps a local network at 5. Some
cases need more (e.g. the n=6 WBFT quorum tests). `chainbench validator set`
produces a preset of any size that `preset.LoadKeyPreset` (and every `chain` step)
consumes.

## What it produces

For each node it generates a random nodekey and derives:

- the **address** + **BLS public key**/**PoP** — derived **in process**
  (`internal/core/keyring/derive`); no external `bootnode` binary is involved,
- the devp2p **public key**,
- an encrypted **keystore** — via the accounts SDK; no node binary is involved,

then writes a `metadata.json` (validators, aligned BLS keys, alloc, system-contract
members, nodes), a shared `password` file, and a per-node dir. The preset carries
**no `extraData`**: the genesis step writes the validator set into both
`croissant.init.validators` and the header `extraData`, and computes the latter
(an RLP-encoded `WBFTExtra`) from the validator addresses and their aligned BLS
keys (`internal/consensus/wbft/extradata.go`). So a preset only has to get the
addresses and BLS keys right.

The generated metadata carries **no enode**. Enodes are assembled at compose time
from the public key and the node's actual host and port, which is the only place
both are known.

## Use

```sh
chainbench validator set \
  --nodes 6 --validators 6 \
  --out /tmp/preset6

chainbench chain up --chain wbft --binary <gwemix> \
  --keys /tmp/preset6 --bp 6
```

Flags: `--nodes` (total, required), `--out` (required), `--validators` (default all),
`--password` (default `1`), `--balance`. `--base-p2p` is accepted but deprecated
and ignored. It refuses an `--out` that already holds a key set (extend one with
`keyring add` instead) and `--validators` larger than `--nodes`.

The gated e2e harness builds networks larger than the committed preset the same
way, through `keyring new --count N --validators N --with-bls` — see
`tests/e2e/wbft_fault_test.go` (`genPreset`).
