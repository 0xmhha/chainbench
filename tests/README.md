# chainbench test code (requirement #16)

This directory holds chainbench **test code** — runtime scenarios executed by
the `testrun` phase against a live NodeSet. It is separate from the test
**helper** module (`pkg/testkit`): helpers live there, scenarios live here.

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
