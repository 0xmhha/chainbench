# chainbench test code (requirement #16)

This directory holds chainbench **test code** — runtime scenarios executed by
the `testrun` phase against a live NodeSet. It is separate from the test
**helper** module (`internal/testkit`): helpers live there, scenarios live here.

## Naming

```
tests/<family>/<category>/<name>.go
```

- `<family>` — consensus family the tests target: `wbft` or `poa`.
- `<category>` — `consensus`, `tx`, `fault`, `system`, `api`, …
- `<name>.go` — one file per case (or a small cohesive group).

Example: `tests/wbft/consensus/chain_id.go`.

## Rules

1. **godoc header (mandatory)** — every test file begins with a doc comment
   stating: Intent, Applies (which chains), Requires (capabilities), Method,
   and Pass criteria. See `wbft/consensus/chain_id.go`.
2. **Register at init** — a case registers a `testkit.Case` in `init()`; it is
   not a `go test` unit. The sibling `_test.go` validates registration and the
   pass path against a mock node.
3. **Gate, don't assume** — set `ChainCompat` (empty = all chains) and
   `RequiresCaps`; the runner skips cases that do not apply instead of failing.
4. **Drive via `testkit.T`** — use `t.Primary()/t.Node(i)` for RPC, the
   assertion helpers (`NoErr`, `Truef`, `Equalf`), and `WaitFor` for liveness.

## Coverage (why `fail=0` is not enough)

The runner reports `coverage = ran / applicable`: of the cases compatible with
the run's chain, how many actually executed. A skip is **never** a pass — it is
persisted as `obs.RunSkipped`, and a chain that gates most cases out shows low
coverage even with `fail=0`. Read low coverage as "under-tested", not "green".

`applicable` counts cases whose `ChainCompat` includes the run's chain (empty =
all). Capability-gated skips still count as applicable (they *could* run here),
so they lower coverage; chain-incompatible cases are excluded from the
denominator.

## Adding a chain

A chain the registry already supports has **no test home** until cases opt it
in — it would run only the empty-`ChainCompat` (all-chain) cases and skip the
rest, which the coverage figure makes visible. To bring an existing case to a
new chain:

1. Add the chain id to that case's `ChainCompat`.
2. Provide the fixture it needs **for that chain** — e.g. a genesis-alloc
   account funded in that chain's preset, not one borrowed from another chain's
   preset. A borrowed fixture passes only by accident and breaks when the chain
   has a different alloc.
3. If the case is chain-specific (system contracts, governance), put it under
   that chain's family dir rather than widening a generic case's gate.

## Keys in test source — TEST FIXTURE ONLY

Cases here hold **plaintext private keys inline** (`faucetKeyHex` and friends).
That is deliberate: a case must fund a transfer without a key-management step,
and the value must be reproducible across runs.

Every such key is a fixture from `keys/preset/` — the faucet key is the upstream
go-ethereum test key, public in every geth fork. **Never move one of these keys,
or an address derived from one, onto a shared network.** A secret scanner over
this directory reports them, and those reports are expected; see the caution
block in the [root README](../README.md).

When a new case needs a funded account, take it from the chain's preset alloc.
Do not generate a fresh key and commit it — a committed key that is *not* a
known fixture is indistinguishable from a real leak to anyone reading the diff.
