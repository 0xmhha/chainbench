# Chain presets

A chain-preset says what a network looks like: which chain, how many nodes and
in which roles, which binaries they run, where the keys come from, and the
launch and config knobs they take. A case names one and overrides what it needs
to.

```json
{ "schemaVersion": "2", "kind": "chain-preset", "id": "stablenet-bp4-en1",
  "chain": "stablenet",
  "keys": { "nodekeys": { "ref": "presets/keys", "source": "keyPreset" } },
  "topology": { "bp": 4, "en": 1 } }
```

A case reaches it by id:

```json
{ "schemaVersion": "2", "kind": "case", "id": "basic-tx-send",
  "chainPreset": "stablenet-bp4-en1",
  "steps": [ … ] }
```

The id is resolved against the case file's own directory, then each directory
above it (`<dir>/<id>.json`, `<dir>/chain-preset/<id>.json`), and finally here.
A suite that wants its own declaration keeps it beside its cases; one that is
shared by several lives in this directory.

## What belongs here, and what does not

A chain-preset names things logically and depends on no machine. Which servers
exist and which ports they offer is the **server set**'s; where files land on a
target is the **workspace config**'s. That separation is what lets one case run
on a laptop and on a server set without editing it, and it is why addresses and
ports are not in this directory — they are site-specific, and these files are
committed.

What a chain IS — its chain id, its flag dialect, the forks its build knows, the
genesis template — is the **chain-manifest**'s, which ships inside the binary
(`internal/chains/<id>/manifest.json`). A chain-preset does not repeat it; it
names the chain and overrides what this network needs differently.

## Hardforks

A hardfork is one of the shapes a chain-preset describes, not a separate kind of
document. It says which fork, at which block, and which binary takes over:

```json
"binaries": { "default": "gwemix", "next": { "binary": "gwbft", "chain": "wbft" } },
"upgrade": { "fork": "croissant", "at": 20, "from": "default", "to": "next",
             "style": "concurrent" }
```

The fork's name is checked against the chain the POST-fork binary runs, not the
network's — wemix does not know croissant and the wbft build taking over from it
does. `validate` does that offline, because a misspelled fork is otherwise
silent: the genesis writes `<name>Block` for whatever it is given, no chain
reaches a fork by that name, and the run reports what the pre-fork build did.

`style` is `concurrent` when another build takes over, `restart` when one chain
crosses its own fork and every node is relaunched on the build that knows it.

## The two yaml files

`wemix-upgrade.yaml` and `wemix-upgrade-15.yaml` are records, not declarations.
Nothing loads them.

They were a second layer: a case named a chain-preset, and the chain-preset
named one of these for the fork and its block. Both declarations that did so
already stated the fork and the block themselves, so the second document's only
remaining job was to be compared against — and comparing two documents says
nothing about whether either is right. The comparison became the check against
the chain-manifest described above, and the grammar that reached these files is
gone.

They stay because the identities of the environments they describe are written
down nowhere else: `wemix-upgrade-15.yaml` holds the 15 producer and 15
successor addresses of a verified 15+15 run. Turning either into a chain-preset
means deciding what a 30-node network's key set is, which is a decision and not
a conversion.
