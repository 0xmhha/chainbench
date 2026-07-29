# pkg/chains/wemix/deploy

Remote **wemix+etcd** deployment for a go-wemix → go-wbft (Croissant) hardfork
chain on a closed network. This is the chainbench-native migration of the
`wemix4` SSH test suite; the architecture and phasing are in
[`docs/REMOTE_WEMIX_DEPLOY_DESIGN.md`](../../../../docs/REMOTE_WEMIX_DEPLOY_DESIGN.md).

## Status: Phase 1 — config + cluster model

`cluster.go` provides the declarative server model: `Cluster`/`Server`, roles
(`wemix_bp` producer / `wbft_bp` validator / `en` endpoint / `pn` bootnode),
`LoadCluster`, per-server SSH port / binary / sync-mode / RPC-URL resolution, and
role/launch-order helpers. Supports 1 to N servers (count = number of `servers:`
entries). No network I/O yet.

Later phases add: remote key read (`bootnode -writeaddress` + SCP pull), remote
provision/launch, governance+etcd bootstrap, hardfork handoff, and test-case
ports.

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
