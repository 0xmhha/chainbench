# Remote wemix+etcd deployment — design (wemix4 → chainbench Go)

> Migrate the `tests/wemix4/` **SSH-driven, closed-network** deploy+hardfork test
> suite into chainbench's Go implementation. Reference source:
> `…/packages/chainbench/tests/wemix4/`. This doc maps each wemix4 mechanism to a
> chainbench component (reuse vs build), defines the config surface, and phases
> the work. **Design for review before implementation.**
>
> **경로 주석(2026-08-27)**: 이 문서는 `pkg/` → `internal/` 이동(T0.0) 전에 쓰였다.
> 본문의 패키지 경로는 현재 트리에 맞춰 `internal/*` 로 고쳤다. 결정과 단계 서술은
> 작성 시점 그대로 두었으므로, 현재 구조는 코드가 이긴다.

## 0. What wemix4 does (one paragraph)

A bash suite deploys a **go-wemix (wpoa) → go-wbft (WBFT)** chain across N remote
servers (default 15) over SSH+SCP, brings it up (genesis → binary → governance +
**embedded-etcd init** → launch → pre-Croissant txs), and at Croissant block 100
the network switches header format to WBFT. Servers have roles (`wemix_bp`
producers on the pre-fork binary, `wbft_bp` validators, `en` endpoints, `pn`
bootnode). Keys live **on the remote servers**; the operator pulls keystores to
local and derives addresses/BLS keys remotely (`bootnode -writeaddress`). Then
~120 standalone test scripts (NODE/WBFT/RPC/GOV/TX) run against the live chain.

## 1. Mapping wemix4 → chainbench (reuse vs build)

| wemix4 mechanism | chainbench today | Action |
|---|---|---|
| `node_ctrl.sh ssh_exec` (sshpass, port 10022, password) | `internal/core/remote.Exec` (SSH, `Credentials`, password via env) | **Reuse.** Add per-server host/port/user. |
| `node_ctrl.sh scp upload` | `RemoteDriver.ProvisionFile` / `InitDatadir` (ship TO remote) | **Reuse.** |
| `node_ctrl.sh download` (SCP pull) | — (only ships TO remote) | **Build:** remote **read/query** (§4). |
| `setup.sh` (binary symlink, nodetype/syncmode, per server) | `RemoteDriver.Provision` + `nodeconfig` | **Extend** for role/binary/syncmode per server. |
| `init_wemix_gov.sh` (gov deploy + etcd init on a temp node) | `poa.DeployGovernance` + `poa.EtcdInit` (take a `Runner`) | **Reuse** — the `Runner` can be an SSH runner. |
| `bootstrap.sh` (genesis→setup→init→gov→run→txs→wait) | `poa.BootstrapPlan` (ordered steps) + `internal/consensus/upgrade` | **Extend** to a multi-server remote plan. |
| Hardfork handoff (Croissant, chaindata `geth/→gwemix/` migrate + binary swap) | `internal/consensus/upgrade` (LaunchHandoff) + `internal/core/hardfork` | **Reuse/extend** for the in-place data-migration variant (NODE-002). |
| `node_env.json` roles + `env.conf` `SERVER_N_IP` + `accounts.env` | `profiles/*.yaml` (single-machine) + `--remote-host` (one host) | **Build:** multi-server config (§3). |
| `.credentials` (global user/pw) | `CHAINBENCH_REMOTE_PASS` env | **Reuse**, extend for per-server + sample file. |
| gwemix embeds etcd | confirmed (`embed.StartEtcd`) | **No external etcd.** |
| Test scripts (NODE/WBFT/RPC/GOV/TX) | Go testkit (`tests/`) + e2e (`tests/e2e/`) | **Later phase** — port cases after deploy works. |

**Net:** the SSH transport, file-ship, poa bootstrap (governance/etcd), and
handoff framework already exist and are `Runner`-based (remote-capable). The
genuinely new pieces are (a) a **multi-server config**, (b) a **remote key
read/query**, and (c) **orchestrating a multi-server remote deploy** (upgrade run
is local-only today).

## 2. Where it lives

- **`internal/chains/wemix/`** — wemix-specific remote setup data + the deploy plan
  builder (roles, governance members, Croissant params). Keep chain knowledge here.
- **`internal/core/remote/`** — add a **key-read** helper (SSH-exec `bootnode
  -writeaddress` + SCP pull) → `NodeKeyInfo`.
- **`internal/core/driver/remote.go`** — add a `FileReader`/`ReadFile` capability
  (SCP pull) alongside the existing `FileProvisioner`.
- **`core/remote/cluster` (new)** — the multi-server model: parse the config,
  hold `[]Server{Index,Host,SSHPort,User,Roles,Binary,SyncMode}`, resolve
  node→server→RPC URL, iterate (forward/reverse).
  *(구현은 이 자리에 만들지 않았다. 클러스터 모델은 §7 의 결정 2 대로
  `internal/chains/wemix/deploy/cluster.go` 에 있다.)*
- **`cmd/chainbench remote-deploy` (new subcommand)** or extend `upgrade run`
  with `--cluster <config>` for the remote path.
- **`tests/wemix4/` config dir (new, in this repo)** — the sample config files
  (§3), gitignored real copies.

## 3. Config surface (sensitive → sample → gitignored)

Model each server **explicitly** (drop wemix4's fragile IP-last-octet == number
== target coupling). Proposed `env.conf` (copy from `env.conf.sample`):

```conf
# chain params
rpc_port = 8601
ws_port  = 8701
croissant_block = 100
epoch_length    = 10
target_validators = 7
genesis_file = /data/go-wbft/genesis_private.json
# binaries on the remote servers (embedded etcd — no external etcd)
wemix_binary = /data/bin/gwemix3   # pre-fork wpoa producer binary
wbft_binary  = /data/bin/gwemix4   # post-fork WBFT binary
# servers: one [[server]] block each — 1 to N, count is the number of blocks
[[server]]
index = 1
host = 10.0.0.11
ssh_port = 10022
role = wemix_bp        # wemix_bp | wbft_bp | en | pn
sync_mode = full
[[server]]
index = 2
host = 10.0.0.12
role = wbft_bp
# ... repeat per server
```

Companion samples (all gitignored; `.sample` committed):
- **`credentials.sample`** → `credentials`: SSH `user`, `password` (or
  `key_file`), optional per-server override. (wemix4: global user/pw only.)
- **`accounts.sample`** → `accounts`: keystore dir/password, per-validator
  `addr`/`operator`/`bls`/`bls_pop`/`stake`, test accounts. **Most of these can be
  auto-populated by the key-read (§4)** rather than hand-entered.

`.gitignore` (new, in the config dir): `env.conf`, `credentials`, `accounts`,
`keystores/*` (keep `keystores/README.md`).

> Format choice to confirm: TOML-ish `[[server]]` (shown) vs YAML list vs reuse
> the existing `profiles/*.yaml` schema extended with a `servers:` list. YAML
> reuses the existing profile loader; recommend **YAML `servers:` list**.

## 4. Remote key read/query (the closed-network requirement)

The requirement: keys live on remote servers at fixed paths; read them to local.
wemix4 does this manually (`download` + copy-paste `bootnode -writeaddress`).
**Automate it in Go:**

```
chainbench remote keys read --cluster env.conf [--server N]
```
Per selected server, over SSH:
1. `bootnode -nodekey <remote nodekey> -writeaddress` → parse 3 lines →
   `NodeKeyInfo{ Address, BLSPubKey (48B), BLSPoP (96B) }`.
2. SCP-pull the coinbase + operator keystores to `keystores/keystore_N` /
   `operator_N` locally.
3. Emit an `accounts` fragment (addr/operator/bls/bls_pop per server) so the
   operator doesn't transcribe by hand.

Remote path contract (from wemix4, keep configurable):
`nodekey=/data/go-wbft/conf/nodekey`, `coinbase=/data/go-wbft/conf/keystore/coinbase`,
`operator=/data/go-wbft/conf/keystore/operator`. This is a **test-convenience**
path for the current closed network; the normal flow (generate keys locally →
ship with the binary) stays supported via the existing `FileProvisioner`.

## 5. Deploy orchestration (multi-server, remote)

New plan builder in `internal/chains/wemix` produces an ordered, per-server plan the
existing primitives execute over an SSH `Runner`/`RemoteDriver`:

1. **Provision** each server: ship genesis + node config + (optional) keys; set
   binary (wemix_bp → wemix_binary, others → wbft_binary); write role/syncmode.
2. **Governance + etcd**: boot the first `wemix_bp`, `poa.DeployGovernance` +
   `poa.EtcdInit` (remote Runner), register stakers/NCP per validator.
3. **Launch** all servers (endpoints/bootnode before producers, per wemix4).
4. **Wait** for block production; send pre-Croissant txs.
5. **Handoff** at Croissant: producers stop at block N-1; the WBFT validators
   (already synced) continue. In-place migration variant (NODE-002) via chaindata
   symlink + binary swap is a `hardfork`-style step.
6. **Verify/test** over RPC (reuse `internal/core/rpc`, testkit cases later).

Reuses `internal/consensus/upgrade.LaunchHandoff` + `internal/consensus/poa`; the only new
wiring is (a) a `RemoteDriver` per server instead of `NewLocalDriver()`, and (b)
the multi-server cluster iteration.

## 6. Phasing

1. **Config + cluster model** — `servers:` config + loader + `Server`/`Cluster`
   types + samples + `.gitignore`. (No network yet; unit-tested.)
2. **Remote key read** — `remote keys read` (SSH bootnode-parse + SCP pull →
   `NodeKeyInfo` + accounts fragment). Verifiable against one server.
3. **Remote provision + launch** — ship genesis/config, set binary/role, launch a
   multi-server wemix network (no hardfork yet); confirm block production.
4. **Governance + etcd bootstrap** remotely; register stakers/NCP.
5. **Hardfork handoff** across the cluster; post-fork verification.
6. **Port test cases** (NODE/WBFT/RPC/GOV/TX) to Go testkit/e2e, gated —
   gap-analysis driven (much of wemix4 is already covered by the testkit corpus);
   see [`dev/wemix4-port-tracker.md`](dev/wemix4-port-tracker.md).

Each phase is independently useful and testable; 1–2 are pure local/config work,
3+ need the remote servers.

## 7. Decisions (confirmed)

1. **Config format**: **YAML `servers:` list** (reuses the profile loader
   convention, `go.yaml.in/yaml/v3`). ✓ Phase 1.
2. **Config location**: **`internal/chains/wemix/deploy/`** (chain-adjacent). ✓
3. **Key model**: **read-from-remote** is the closed-network default; the
   generate-and-ship path (existing `FileProvisioner`) stays the general default.
4. **Entry point**: **new `chainbench remote` subcommand group** (`remote keys`,
   `remote deploy`, …). ✓
5. **Test-case scope**: deploy + hardfork + a smoke set first (NODE-001/002, a
   few RPC/TX), then broaden. ✓

### Phase 1 delivered
`internal/chains/wemix/deploy/` — `cluster.go` (`Cluster`/`Server`, roles, `LoadCluster`,
per-server resolution, role/launch-order helpers, 1..N servers) + `cluster_test.go`
+ samples (`cluster.yaml.sample`, `credentials.sample`, `accounts.sample`) +
`.gitignore` + `keystores/README.md` + `README.md`. No network I/O yet.
