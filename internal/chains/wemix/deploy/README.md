# internal/chains/wemix/deploy

Remote **wemix+etcd** deployment for a go-wemix → go-wbft (Croissant) hardfork
chain on a closed network. This is the chainbench-native migration of the
`wemix4` SSH test suite; the architecture and phasing are in
[`docs/REMOTE_WEMIX_DEPLOY_DESIGN.md`](../../../../docs/REMOTE_WEMIX_DEPLOY_DESIGN.md).

## Status: Phases 1-5 — cluster + key read + provision/launch + governance/etcd + handoff

- **`cluster.go`** — declarative server model: `Cluster`/`Server`, roles
  (`wemix_bp` producer / `wbft_bp` validator / `en` endpoint / `pn` bootnode),
  `LoadCluster`, per-server SSH port / binary / sync-mode / RPC-URL resolution,
  role/launch-order helpers, and remote key paths. 1 to N servers.
- **`credentials.go`** — SSH auth from `credentials` (or `CHAINBENCH_REMOTE_USER`
  / `CHAINBENCH_REMOTE_PASS`), per-server overrides → `remote.Credentials`.
- **`keys.go`** — the remote key read: derive address + BLS pubkey/PoP via
  `bootnode -writeaddress` over SSH, pull the coinbase/operator keystores locally,
  emit an `accounts` fragment. Exposed as `chainbench remote keys read`.

```sh
chainbench remote keys read --cluster cluster.yaml --credentials credentials \
  --keystore-dir keystores --accounts-out accounts.generated
# or a single server:
chainbench remote keys read --cluster cluster.yaml --server 3
```

- **`plan.go`** — maps each cluster server to a `driver.NodeSpec` (role binary,
  ports, poa/wemix node config, launch args); `Describe` renders a dry-run plan.
- **`orchestrate.go`** — `Deploy`: over SSH, provision + init genesis + launch
  each server in launch order (endpoints/bootnodes first). Keys are read from the
  servers, not shipped. Exposed as `chainbench remote deploy`.

```sh
chainbench remote deploy --cluster cluster.yaml --dry-run          # print the plan
chainbench remote deploy --cluster cluster.yaml --credentials credentials \
  --genesis genesis.json                                           # provision + launch
```

- **`bootstrap.go` + `accounts.go`** — `Bootstrap`: on the boot producer (first
  `wemix_bp`), ship the wemix governance config (built from `accounts` via
  `BuildWemixConfig`) and run `poa.DeployGovernance` + `poa.EtcdInit` over an SSH
  `poa.Runner`. Exposed as `chainbench remote bootstrap`. gwemix embeds etcd.

```sh
chainbench remote bootstrap --cluster cluster.yaml --credentials credentials \
  --accounts accounts
```

- **`handoff.go`** — `WaitHandoff`: over an SSH tunnel to a `wbft_bp` validator's
  RPC (closed network), poll until the chain crosses the Croissant block and the
  next block is sealed by a validator rather than a wemix producer — proof the
  wemix→wbft hardfork handoff completed. Exposed as `chainbench remote handoff`.

```sh
chainbench remote handoff --cluster cluster.yaml --credentials credentials \
  --accounts accounts --wait 300
```

The last phase adds the wemix4 test-case ports.

## Config (sensitive — sample → real, gitignored)

| file | purpose |
|---|---|
| `cluster.yaml.sample` → `cluster.yaml` | chain params + server list (hosts/roles). |
| `credentials.sample` → `credentials` | SSH user + password (or key_file), optional per-server overrides. |
| `accounts.sample` → `accounts` | validator/operator addresses, BLS keys, stakes, test accounts (auto-fillable by the key-read). |
| `keystores/` | keystore files pulled from remote (all gitignored except the README). |

```sh
cp cluster.yaml.sample cluster.yaml     # then edit hosts/roles
cp credentials.sample credentials       # then edit SSH auth
cp accounts.sample accounts             # or let the key-read fill it
```

Only the `*.sample` files, `README.md`, and `keystores/README.md` are committed;
`.gitignore` excludes every real config so server IPs / SSH creds / keys never
reach git.
