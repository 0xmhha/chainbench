# WEB-04 manifest module

The dashboard reuses registered chain plugins and `external.FromBytes` for imports.
The browser library lists embedded plugins, accepts manifest JSON and unmodified
engine template text, validates imports, saves immutable content-addressed bundles,
exports a selected bundle, and initializes an isolated local composition. Embedded
chain IDs cannot be replaced. Imports borrow an existing accounts protocol and its
family, dialect, bootstrap, RPC and genesis engine mapping; unknown JSON fields and
unsupported mappings are rejected.

Enable it with the existing private `-deployment-accounts` provider and
`-manifest-assets assets.json`. The data directory remains `-deployment-root`
(default `chainbench-out/web-ui`). `-manifest-keys` defaults to `presets/keys`.
An asset entry contains `id`, `chain`, `path`, `sha256` and the independently
verified source `commit`. Asset paths are administrator provisioning inputs and
cannot be supplied through the API. Selection verifies checksum, native binary
format/architecture and version identity, then records the help digest. Both gwemix
builds are distinguished by chain/commit evidence rather than filename.

Setup calls the existing app composition operations through allocate, keys, genesis,
config, launch options, provision and init. Four preset validators satisfy the
embedded Stablenet quorum requirement. Setup leaves nodes stopped and keeps its
engine state and any partial files. No raw argv or executable plugin installation
is exposed. Mutation and denial status receipts exclude request bodies and secrets.
Control-plane persistence belongs to `core/session`; node material stays owned by
the existing composition engine.

Run `bash tests/webui/verify.sh --criterion WEB-04 --require-live --output chainbench-out/web-ui-acceptance`.
`--output` also accepts an absolute destination. The runner always produces fresh
canonical receipts in `chainbench-out/web-ui-acceptance/WEB-04`, with workspace-relative
artifact paths, then copies those receipts to the requested destination.
The runner prepares fresh native assets and private accounts under the dedicated
runtime root, builds the SPA and Go server, runs the real browser against all three
embedded and three external selections, observes six real four-node database
initializations, and checks negative cases and restart restoration. An independent
verifier checks invocation, scenario coverage, digest-bound browser/API/engine and
database evidence. Fixtures stop their dashboard and remove disposable account
secrets; no node processes are started by these scenarios.
