# server-set and workspace-config: creating the two files

chainbench takes two site-specific files so one DSL runs unchanged across
local, docker, and remote targets. You create them once per environment and
name them on the command line; you never edit the DSL for a new environment.

Neither real file is committed — only the two `*.sample.yaml` templates at the
repo root are tracked. Copy a sample, fill it in, and keep the real file
outside the repo or under a gitignored name (`*workspace-config.yaml`,
`*server-set.yaml`).

## What each file owns

| File | Owns | Does not own |
|---|---|---|
| **server-set** | the servers chainbench may use: their names, addresses, SSH login, port bands, and how many slots each host holds | the data root, test content |
| **workspace-config** | where files live on the target: `dataRoot` and the purpose directories under it, plus the local results root and how inputs are prepared | the server list, credentials, node count/roles |
| DSL (test spec) | test content, node composition, logical file references | absolute paths, credentials |

The split is the point: change servers by swapping the server-set, change the
target's directory layout by swapping the workspace-config, and the DSL stays
the same.

## Create a server-set

```sh
cp server-set.sample.yaml server-set.yaml   # gitignored
```

Fill in `pool.hosts` (a loopback address runs nodes on this machine; anything
else is reached over SSH), `slots`, `ports`, and the `ssh` block. A one-host
pool with several slots is this machine running a whole network; a many-host
pool with one slot each spreads one node per server.

`dataRoot` is **not** set here any more — it moved to workspace-config (see
below). A server-set that still sets `dataRoot` is rejected with a message
pointing here.

## Create a workspace-config

```sh
cp workspace-config.sample.yaml workspace-config.yaml   # gitignored
```

Set `dataRoot` to the absolute path on the target under which everything lives
(on a remote host, a path on that host; local, a path on this machine). Under
it, `paths` names one directory per purpose. `control.artifactRoot` is the one
local path — where run results and the final report land on the machine running
chainbench (`~` expands locally; a relative value is relative to the config
file's directory; `run --artifact-root` overrides it). The optional
`control.binaries` is a second local path: a directory of node binaries the
deploy step sends to a remote target when the target's copy is missing or its
sha256 differs (unset means the target must already have them).

`inputs.mode` is `generated` (build test keys and genesis/config with the
in-process builders) or `existing` (use files already on the target, named by an
`existingInputs` bundle through `inputs.name`). `execution.chain` is `fresh`,
`reuse-if-matching`, or
`attach` — see below. The optional `limits.minFreeDisk` (`2GiB`, `500MiB`, a
byte count, or `"0"` to skip) is the free space `dataRoot` must have on every
server a node lands on before genesis is written; unset means 2GiB.

The file is validated strictly: `version` must be `1`; `dataRoot` must be
absolute with no `~` or `$`; all eight `paths` entries (`binaries`, `configs`,
`genesis`, `keystore`, `keyrings`, `nodes`, `runtime`, `logs`) are required and
must be relative with no `..`; `control.artifactRoot`, `inputs.mode`, and
`execution.chain` are required; and an unknown or duplicated key is an error
rather than silently ignored.

## execution.chain: how a run treats an existing composition

`execution.chain` is read by `chain up` and by a composing `run` alike. `fresh`
(what a run without a workspace-config gets) composes and launches the network
as always; it does not disturb another composition's processes or data.

`reuse-if-matching` reconciles a running network node by node before it deploys
or launches anything. For each node it compares the config and binary against
what the run would produce now, and probes whether the node is still answering.
A node whose inputs are unchanged and that is up is left running; only nodes
that drifted or stopped are torn down and brought back. Of fifteen nodes, if one
changed, one is redone and fourteen keep running. When this workspace has no
record of a node — a first run against a target something is already up on — the
baseline is read back from the running process itself: its `--config` is
recovered from the command line and hashed on the machine, so a node already up
with the config this run would give it is reused in place, and one up with a
different config or binary is refused as foreign rather than composed over. The one
exception is the
genesis: it is shared by every node, so a changed genesis is a different chain
and cannot be reconciled onto a running network — the whole reuse is refused,
and nothing is touched. Use it to continue an expensive environment across runs.

`attach` tests an already-running chain. It does not create, deploy, or init, so
`chain up` (and a composing `run`) refuses it — bring the chain up separately and
use the attach/run path.

## Run with both files

```sh
chainbench run tests/tc/common/tx/CT-TX-001-value-transfer.json \
  --server-set  ./server-set.yaml \
  --workspace-config ./workspace-config.yaml \
  --workspace-dir /tmp/control/chain-a
```

`--workspace-dir` is the local control directory (where the composition's state
is recorded); it is separate from the target's `dataRoot`. Add `--docker` when
the server-set's hosts are local docker containers.

`--workspace-config` is accepted by the one-shot `run`, by the step-form
`chain new` and `chain up`, by `file upload`/`download` (where it is required),
by `clean --stale-compositions`, and by the MCP tools `chainbench_chain_new` and
`chainbench_chain_up` — every surface resolves the target's `dataRoot` from the
file the same way.

## What is live today

- The workspace-config file format, its validation, and its `dataRoot` are
  live: `dataRoot` comes from this file, and the server-set no longer carries
  one.
- `--workspace-config` is wired into the step-form CLI (`chain new`/`chain up`)
  and the MCP step tools, not only the one-shot `run`.
- `execution.chain` is live for `chain up` and a composing `run`: `fresh`,
  `reuse-if-matching` (per-node reconciliation, above), and `attach` (refused by
  both).
- An existing genesis is checked against the composed keys: for a
  wbft-family chain, the validators the genesis names must be exactly the key
  set the network runs, or the compose is refused (block signing would stall
  consensus otherwise). A generated genesis is built from the keys and cannot
  disagree.
- An `existingInputs` bundle's `keyring` may name a key set on a server
  (`keyring: srv://server-01/...`): the keys step downloads it to a local
  directory (elevating through sudo where the server set permits it) and a node
  signs with keys at that local path. A local absolute keyring is used in place.
- `chainbench file upload`/`download` move files between this machine and a
  server's data plane — a replacement binary onto `bin`, a node log or key back
  off — refusing a name collision on upload unless `--force-upload`.
- An `existingInputs` bundle's `configs` map is applied: a node table names its
  configs logically (`config: validator`) and the bundle says which file that
  name is on this target (`configs: {validator: srv://.../v.toml}`), so one spec
  runs against different targets by swapping the map. A config value that is not
  a key in the map stays a direct file reference.
- `binaryAliases` **는 이제 동작한다.** 대상에서 bare 이름으로 바이너리를 가리키면
  `dataRoot` + `paths.binaries` 아래에서 찾고, 그 이름에 별칭이 선언돼 있으면 별칭을 적용한다.
  즉 DSL 이나 `chain up --binary` 가 `gwemix` 라고만 적어도 (`dataRoot: /data`,
  `paths.binaries: bin` 이면) 대상의 `/data/bin/gwemix` 로 풀린다. 절대경로는 그대로 쓴다. 아키텍처가 다른 빌드를 나란히 두었다면
  `binaryAliases: {gwemix: linux-arm64/gwemix}` 로 한 줄만 바꿔 가른다.

  ```yaml
  binaryAliases:
    gwemix: linux-arm64/gwemix
  ```

- `paths` 의 여덟 용도 디렉터리는 모두 소비된다. `nodes`/`runtime`/`logs` 는 구성 ID 별로
  격리된 노드 DB·생성 genesis/config·로그 위치가 되고(`clean --stale-compositions` 도 이 셋을
  뒤진다), `binaries` 는 위 바이너리 해석, `configs`/`genesis`/`keyrings` 는 상대 파일명 참조의
  기준, `keystore` 는 `file upload`/`download` 의 목적지다. 배선 경위는 인계 문서
  `docs/research/chainbench/analyses/10-prepared-inputs-server-ref-handoff.md` 참고.
- **`existingInputs` 참조는 문자열 세 형식이 전부다** — `srv://<서버>/<절대경로>`, 대상의 용도별
  디렉터리 아래를 가리키는 상대 파일명, 그리고 이 기계의 절대경로(로컬 키셋을 가리킬 때).
  객체형 참조(`{server, ref}`·`{serverIndex, ref}`·`{localPath}`)는 **계획을 철회했다
  (2026-09-12)**. 샘플에 "미구현" 딱지와 함께 예시로 남아 있던 것도 걷었다.
  - 철회한 이유는 표현력이 아니라 값이다. `srv://` 와 상대 파일명이 실제로 쓰이는 경우를
    모두 덮고 있고, 객체형을 이 묶음에서 끝까지 나르려면 `KeysDir`·`GenesisExisting` 을
    문자열로 쓰는 **32개 파일 105곳**을 구조체로 바꿔야 하는데, 그 표현력을 요구하는
    호출자가 하나도 없다. 소비자 없는 구조를 넓게 배선하는 것은 이 저장소가 이미
    두 번(`Transport` 타입, health→inspector 재배선) 되돌린 모양이다.
  - **포기한 것은 하나다**: `serverIndex` — 서버를 이름이 아니라 **순번**으로 고르는 것은
    문자열로 표현할 수 없다. 환경마다 서버 이름이 다른 곳에 같은 묶음을 쓰려면 필요해질
    수 있고, 그때는 요구와 함께 다시 연다.
  - `resource.InputRef` 자체는 네 형식을 모두 구현·테스트한 채로 남는다. 내부에서 쓰이며,
    다시 열 때 배선만 하면 되는 상태다.
