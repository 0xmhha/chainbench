# Generating preset key sets (`keys generate`)

The committed `keys/preset` ships 5 nodes, which caps a local network at 5. Some
cases need more (e.g. the n=6 WBFT quorum tests). `chainbench keys generate`
produces a preset of any size that `keys.LoadPreset` (and `setup`) consume.

## What it produces

For each node it generates a random nodekey and derives:

- the **address** + **BLS public key**/**PoP** — via the go-wbft
  `bootnode -writeaddress` tool,
- the devp2p **public key** / enode,
- an encrypted **keystore** — via the node binary's `account import`,

then writes a `metadata.json` (validators, aligned BLS keys, alloc, system-contract
members, nodes) and a per-node dir. Crucially, the croissant/WBFT validator set
lives in the **genesis config** (`croissant.init.validators`), not in the header
`extraData`, so `extraData` is a plain 32-byte vanity — **no istanbul RLP encoding
is needed**, which is what makes generating a working preset tractable.

## Use

```sh
# Build the go-wbft bootnode tool once:
#   (cd go-wbft && go build -o build/bin/bootnode ./cmd/bootnode)

chainbench keys generate \
  --nodes 6 --validators 6 \
  --bootnode /path/go-wbft/build/bin/bootnode \
  --binary   /path/go-wbft/build/bin/gwemix \
  --out /tmp/preset6

chainbench setup --launch --chain wbft --binary <gwemix> \
  --keys-dir /tmp/preset6 --validators 6
```

Flags: `--nodes` (total), `--validators` (default all), `--bootnode`, `--binary`,
`--out`, `--password` (default `1`), `--base-p2p`, `--balance`.

The gated e2e harness uses this (via `BOOTNODE_BIN`) to build networks larger than
the committed preset — see `tests/e2e/wbft_fault_test.go` (`genPreset`).
