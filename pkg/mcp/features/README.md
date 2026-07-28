# chainbench capabilities

The **capability registry** (`pkg/mcp/capability`) is the layered, per-project
feature model behind the chainbench MCP and CLI surfaces. It answers "which
features does chain X support, and how are they called" as data, so the exposed
tool set grows with chains/projects **without editing a central switch**.

## Model

A capability is addressed hierarchically: `<version>.<chain>.<name>`, e.g.
`v1.common.chains.info` or `v1.stablenet.governance.propose_mint`.

- `chain = "common"` — applies to every chain, implemented once (`features/common`).
- `chain = <id>` — specific to that chain, implemented in `features/<id>`.

A capability is **exposed only if registered**: it needs a catalog entry (a
`.jsonl` line) *and* either a bound handler (a generated tool) or a pre-existing
flat tool it maps to.

- **Discovery** — `chainbench.capabilities` (MCP tool) and `chainbench
  capabilities` (CLI) list the version → chain → feature tree, optionally scoped
  to one chain (returns common + that chain's features).
- **Invocation** — a handler-backed capability is exposed as an MCP tool named
  `chainbench.<version>.<chain>.<name>`. The 30 built-in tools keep their
  established `chainbench_*` names and are folded into the catalog for discovery.

## Add a project's capabilities

A "project" is a chain (or a shared set). To add capabilities:

1. Create `pkg/mcp/features/<project>/`.
2. Add `<project>.jsonl` — one `capability.Descriptor` per line:
   ```json
   {"version":"v1","chain":"wemix","name":"bootstrap.plan","summary":"…","params":[{"name":"x","type":"string","required":true}]}
   ```
3. Add the Go file with handlers, registering both at `init()`:
   ```go
   //go:embed <project>.jsonl
   var catalog []byte

   func init() {
       if err := capability.LoadCatalog(catalog); err != nil { panic(err) }
       capability.RegisterHandler("v1", "wemix", "bootstrap.plan", bootstrapPlan)
   }
   ```
4. Blank-import the package from `pkg/mcp/features/all/all.go`.

That's it — the capability appears in discovery and (for handler-backed ones)
as a callable tool, gated to its chain. A chain with no unique feature simply
has no `features/<id>` package and exposes only the common set.

## Conventions

- Keep handler logic thin: call the same core packages the CLI uses.
- Chain-specific bindings (e.g. stablenet governance ABIs) live under
  `pkg/chains/<id>/…`; the capability handler wires them to the MCP surface.
- Names are dotted feature paths (`governance.propose_mint`), lowercase.
