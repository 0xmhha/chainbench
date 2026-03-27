# chainbench

A local blockchain sandbox test bench for [go-stablenet](https://github.com/user/go-stablenet) (geth fork with WBFT consensus). Manage multi-node local chains with a single CLI, run built-in tests, and integrate with AI agents via MCP.

## Features

- **Single CLI** — `init`, `start`, `stop`, `status`, `test`, `log` in one tool
- **Profile-based config** — YAML profiles with inheritance for different test scenarios
- **Preset keys** — Fixed validator keys for reproducible debugging (`node1 = 0xc17d...` always)
- **Built-in tests** — 10 tests covering consensus, fault tolerance, and stress
- **Log analysis** — Consensus timeline, anomaly detection, cross-node search
- **MCP server** — 13 tools for Claude Code / AI agent integration
- **No Docker required** — Pure process-based, runs on macOS and Linux

## Quick Start

```bash
# 1. Clone
git clone https://github.com/0xmhha/chainbench.git
cd chainbench

# 2. Set the gstable binary path in the profile
#    Edit profiles/default.yaml → chain.binary_path to your gstable build location
#    Example: binary_path: "/path/to/go-stablenet/build/bin/gstable"

# 3. Initialize the chain (4 validators + 1 endpoint)
./chainbench init

# 4. Start all nodes
./chainbench start

# 5. Check status
./chainbench status

# 6. Run basic tests
./chainbench test run basic

# 7. Stop
./chainbench stop
```

## Prerequisites

- **bash** 4.0+
- **python3** 3.6+ (macOS ships with it; PyYAML optional but recommended: `pip3 install pyyaml`)
- **curl**
- **Node.js** 18+ and **npm** (only for MCP server / Claude Code integration)
- **gstable** binary — build from [go-stablenet](https://github.com/user/go-stablenet):
  ```bash
  cd go-stablenet && make gstable
  # Binary at: build/bin/gstable
  ```
- **logrot** (optional) — log rotation utility. Place in `bin/logrot` if available. Without it, logs are written directly to files (no auto-rotation).

## Installation

### One-Command Setup (Recommended)

```bash
git clone https://github.com/0xmhha/chainbench.git
cd chainbench
./setup.sh
```

This will:
1. Check prerequisites (bash, python3, node, npm)
2. Build the MCP server
3. Register the MCP server in Claude Code (interactive prompt — global or project-level)

### Manual Setup

```bash
git clone https://github.com/0xmhha/chainbench.git
cd chainbench

# Build MCP server (optional, for Claude Code integration)
cd mcp-server && npm install && npm run build && cd ..
```

### Optional: Add to PATH

```bash
# Option A: symlink
ln -s "$(pwd)/chainbench" /usr/local/bin/chainbench

# Option B: add to shell profile
echo 'export PATH="/path/to/chainbench:$PATH"' >> ~/.zshrc
```

### Configuration

Edit `profiles/default.yaml` to set your environment:

```yaml
chain:
  binary_path: "/absolute/path/to/gstable"   # or relative to chainbench/

data:
  directory: /tmp/node-data                    # where node data is stored

nodes:
  validators: 4
  endpoints: 1
```

## CLI Reference

### Chain Lifecycle

```bash
./chainbench init [--profile <name>]    # Initialize chain from profile
./chainbench start                       # Start all nodes
./chainbench stop                        # Stop all nodes (SIGTERM → SIGKILL)
./chainbench restart                     # stop → clean → init → start
./chainbench status [--json]             # Show node status
./chainbench clean                       # Remove all node data
```

### Node Control

```bash
./chainbench node stop 3                 # Stop node 3 only
./chainbench node start 3                # Restart node 3
./chainbench node log 3                  # Show last 50 lines of node 3 log
./chainbench node log 3 --follow         # Tail -f node 3 log
./chainbench node rpc 1 eth_blockNumber  # RPC call to node 1
```

### Testing

```bash
./chainbench test list                   # List available tests
./chainbench test run basic/consensus    # Run single test
./chainbench test run basic              # Run all basic tests
./chainbench test run all                # Run everything
./chainbench report [--format json]      # Show test results
```

### Log Analysis

```bash
./chainbench log timeline                # Consensus event timeline
./chainbench log anomaly                 # Detect anomalous patterns
./chainbench log search "ROUND_CHANGE"   # Search across all node logs
```

### Profile Management

```bash
./chainbench profile list                # List available profiles
./chainbench profile show default        # Show profile content
./chainbench profile create my-test      # Create custom profile from default
```

## Built-in Tests

| Test | Description |
|------|-------------|
| `basic/consensus` | Verify block production, timing, miner diversity, sync |
| `basic/tx-send` | Send transaction and verify receipt |
| `basic/sync` | Check all nodes have synchronized block heights |
| `basic/peers` | Verify peer connectivity between nodes |
| `basic/rpc-health` | Check all RPC endpoints respond |
| `fault/node-crash` | Stop 1/4 validators → consensus continues → recover |
| `fault/node-recover` | Stop node, wait, restart, measure sync time |
| `fault/two-down` | Stop 2/4 validators → consensus halts → restore 1 → resumes |
| `stress/tx-flood` | Send N transactions, measure TPS |
| `stress/block-time` | Block time statistics (avg/min/max/p95) over 100 blocks |

## Profiles

Profiles are YAML files that define the entire chain configuration. They support inheritance with `inherits`.

| Profile | Description |
|---------|-------------|
| `default` | 4 validators + 1 endpoint, 1s block time |
| `minimal` | 2 validators, fast testing |
| `bft-limit` | 4 validators, 2s blocks, auto-runs fault tests |
| `large` | 7 validators + 3 endpoints |

Create custom profiles:

```bash
./chainbench profile create my-scenario
# Edit profiles/custom/my-scenario.yaml
./chainbench init --profile my-scenario
```

### Profile Schema (Key Fields)

```yaml
chain:
  binary: gstable                          # Binary name
  binary_path: "/path/to/gstable"          # Absolute or relative path
  chain_id: 8283                           # Chain ID

data:
  directory: /tmp/node-data                # Node data root directory

genesis:
  overrides:
    wbft:
      blockPeriodSeconds: 1                # Block interval
      requestTimeoutSeconds: 2             # Consensus timeout
      epochLength: 140                     # Epoch length (blocks)
      proposerPolicy: 0                    # 0=round-robin, 1=sticky

nodes:
  validators: 4                            # Validator count
  endpoints: 1                             # Non-mining node count
  verbosity: 4                             # Log level (0-5)
  gcmode: archive                          # archive or full
  cache: 2048                              # Cache size (MB)

keys:
  mode: static                             # static (reuse preset) | generate
  source: "keys/preset"                    # Preset keys directory

ports:
  base_p2p: 30301                          # Ports are base + node_index
  base_http: 8501
  base_ws: 9501
```

## Preset Keys

The `keys/preset/` directory contains pre-generated keys for 5 nodes. Using fixed keys means validator addresses are **always the same**, making log analysis and debugging much easier:

| Node | Address | Role |
|------|---------|------|
| node1 | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | Validator |
| node2 | `0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c` | Validator |
| node3 | `0x8c4a10b9108d49b9d23f764464090831d9c17764` | Validator |
| node4 | `0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6` | Validator |
| node5 | `0x5400d8b543eaf6738c7b44799623bea88fd0f5ee` | Endpoint |

## Claude Code / MCP Integration

chainbench includes an MCP (Model Context Protocol) server that allows AI agents like Claude Code to control the test chain directly.

### Setup

1. Build the MCP server:
   ```bash
   cd mcp-server && npm install && npm run build && cd ..
   ```

2. Register in Claude Code. Choose one of these methods:

   **Method A: Project-level** — create `.mcp.json` in your project root:
   ```json
   {
     "mcpServers": {
       "chainbench": {
         "command": "node",
         "args": ["/Users/yourname/chainbench/mcp-server/dist/index.js"],
         "env": {
           "CHAINBENCH_DIR": "/Users/yourname/chainbench"
         }
       }
     }
   }
   ```

   **Method B: Global** — add to `~/.claude/settings.local.json`:
   ```json
   {
     "mcpServers": {
       "chainbench": {
         "command": "node",
         "args": ["/Users/yourname/chainbench/mcp-server/dist/index.js"],
         "env": {
           "CHAINBENCH_DIR": "/Users/yourname/chainbench"
         }
       }
     }
   }
   ```

   > Replace `/Users/yourname/chainbench` with your actual chainbench installation path.
   > Use `pwd` inside the chainbench directory to get the absolute path.

3. Restart Claude Code (or run `/mcp` to reload). The following MCP tools become available:

### Available MCP Tools

| Tool | Description |
|------|-------------|
| `chainbench_init` | Initialize chain with a profile |
| `chainbench_start` | Start all nodes |
| `chainbench_stop` | Stop all nodes |
| `chainbench_restart` | Full restart cycle |
| `chainbench_status` | Get node status (JSON) |
| `chainbench_node_stop` | Stop a specific node |
| `chainbench_node_start` | Start a specific node |
| `chainbench_node_rpc` | Send RPC to a specific node |
| `chainbench_test_list` | List available tests |
| `chainbench_test_run` | Run a test suite |
| `chainbench_report` | Get test results |
| `chainbench_schema_query` | Query YAML profile schema |
| `chainbench_profile_send` | Create a custom profile via YAML content |
| `chainbench_log_search` | Search node logs |
| `chainbench_log_timeline` | Get consensus timeline |

### AI Agent Workflow Example

```
User: "Run a basic consensus test"

AI Agent:
  1. chainbench_status        → Check if chain is running
  2. chainbench_init           → Initialize if needed
  3. chainbench_start          → Start nodes
  4. chainbench_test_run       → Run "basic/consensus"
  5. chainbench_report         → Return results to user

User: "Kill node 3 and check if consensus continues"

AI Agent:
  1. chainbench_node_stop {node: 3}
  2. chainbench_test_run {test: "fault/node-crash"}
  3. chainbench_log_timeline
  4. Report analysis to user
```

### Schema Query

The `chainbench_schema_query` tool lets the AI understand what profile fields are available before creating custom profiles:

```
AI calls: chainbench_schema_query({section: "genesis.wbft"})
Response: blockPeriodSeconds (int, default: 1) - Target block interval...
AI calls: chainbench_profile_send({name: "slow-blocks", content: "..."})
AI calls: chainbench_init({profile: "slow-blocks"})
```

## Project Structure

```
chainbench/
├── chainbench              # Main CLI entry point (shell script)
├── lib/                    # CLI command implementations
│   ├── cmd_init.sh         # Chain initialization
│   ├── cmd_start.sh        # Node startup with PID tracking
│   ├── cmd_stop.sh         # Graceful shutdown
│   ├── cmd_status.sh       # Node status display
│   ├── cmd_node.sh         # Individual node control
│   ├── cmd_test.sh         # Test runner
│   ├── cmd_log.sh          # Log analysis dispatcher
│   ├── common.sh           # Shared utilities
│   └── profile.sh          # YAML profile parser
├── profiles/               # Test scenario profiles
│   ├── default.yaml
│   ├── minimal.yaml
│   ├── bft-limit.yaml
│   └── large.yaml
├── keys/preset/            # Pre-generated validator keys
│   ├── node1..5/           # Per-node: nodekey, address, bls_pubkey, keystore
│   ├── metadata.json       # Pre-computed validators, BLS keys, enode URLs
│   └── password            # Keystore password ("1")
├── templates/              # Genesis and TOML templates
├── tests/                  # Built-in test suites
│   ├── basic/              # Consensus, tx, sync, peers, rpc-health
│   ├── fault/              # Node crash, recovery, two-down
│   ├── stress/             # TX flood, block time
│   └── lib/                # Shared test libraries (rpc, assert, wait, report)
├── logs/                   # Log analysis tools (parser, timeline, anomaly)
├── mcp-server/             # MCP server for AI integration (TypeScript)
└── bin/                    # Platform binaries (logrot, not tracked in git)
```

## Contributing

### Adding a Test

1. Create a script in `tests/<category>/your-test.sh`
2. Add a header comment: `# Description: What this test verifies`
3. Source the libraries:
   ```bash
   source "$(dirname "$0")/../lib/rpc.sh"
   source "$(dirname "$0")/../lib/assert.sh"
   ```
4. Use `test_start()` / `test_result()` framing
5. The test will automatically appear in `chainbench test list`

### Adding a Profile

1. `./chainbench profile create my-profile`
2. Edit `profiles/custom/my-profile.yaml`
3. Use `inherits: default` to only override what you need

## Troubleshooting

**`chainbench init` fails with "Cannot find gstable binary"**
- Set `chain.binary_path` in `profiles/default.yaml` to the absolute path of your `gstable` binary
- Or add the directory containing `gstable` to your `$PATH`

**Nodes start but immediately die**
- Check logs: `cat /tmp/node-data/logs/node1.log`
- Common cause: port conflict. Change `ports.base_*` values in the profile
- Common cause: `extra_flags` contains invalid YAML — ensure it's a proper list or empty `[]`

**`chainbench stop` doesn't stop all processes**
- Manual cleanup: `pkill -15 gstable && sleep 3 && pkill -9 gstable`
- Then remove stale state: `rm -f state/pids.json`

**MCP server not recognized in Claude Code**
- Verify the path in `.mcp.json` is absolute (not relative)
- Verify `npm run build` succeeded (check `mcp-server/dist/index.js` exists)
- Run `/mcp` in Claude Code to reload servers
- Check Claude Code logs for MCP connection errors

**Port already in use**
- Default ports: HTTP 8501-8505, WS 9501-9505, P2P 30301-30305
- Change base ports in the profile: `ports.base_http: 18501`

## License

[Apache License 2.0](LICENSE)
