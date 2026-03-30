# chainbench

A local blockchain sandbox test bench for [go-stablenet](https://github.com/stable-net/go-stablenet) (geth fork with WBFT consensus). Manage multi-node local chains with a single CLI, run built-in tests, and integrate with AI agents via MCP.

- **Single CLI** for full chain lifecycle (`init`, `start`, `stop`, `test`, `log`)
- **YAML profiles** with inheritance for different test scenarios
- **Preset keys** for reproducible validator addresses across runs
- **10 built-in tests** covering consensus, fault tolerance, and stress
- **MCP server** for Claude Code / AI agent integration (per-project opt-in)
- **No Docker required** — pure process-based, runs on macOS and Linux

## Quick Start

### 1. Install

```bash
curl -fsSL https://raw.githubusercontent.com/0xmhha/chainbench/main/install.sh | bash
```

This clones chainbench to `~/.chainbench`, builds the MCP server, and registers `chainbench` and `chainbench-mcp` in your `$PATH`.

### 2. Enable MCP for your project

```bash
cd /path/to/your-chain-project    # must have gstable at build/bin/
chainbench mcp enable
```

This creates a portable `.mcp.json` in the project root (no absolute paths — works across machines). Restart Claude Code or run `/mcp` to load.

### 3. Run with Claude Code (recommended)

Open Claude Code in your chain project directory and ask:

```
"Initialize and start a local chain, then run a tx test"
```

Claude Code uses MCP tools to drive the full lifecycle:

```
chainbench_init     → Initialize 4 validators + 1 endpoint
chainbench_start    → Launch all nodes
chainbench_status   → Verify consensus OK, all nodes running
chainbench_test_run → Run "basic/tx-send"
chainbench_node_rpc → Send tx, query receipt by txHash
```

You can also interact step by step:

```
"Check chain status"
"Stop node 3 and see if consensus continues"
"Send 1 ETH from node1 to node2 and show the receipt"
"Run all fault tolerance tests"
```

### 4. Run with CLI

```bash
# Initialize and start (4 validators + 1 endpoint)
chainbench init
chainbench start

# Check status and run tests
chainbench status
chainbench test run basic

# Stop
chainbench stop
```

> **Note:** The CLI auto-detects the `gstable` binary from your project's `build/bin/` directory. If running from a different directory, set `chain.binary_path` in `~/.chainbench/profiles/default.yaml`.

### Uninstall

```bash
chainbench uninstall
```

## Prerequisites

| Dependency | Version | Required for |
|------------|---------|-------------|
| bash | 4.0+ | CLI |
| python3 | 3.6+ | Profile parsing, genesis generation |
| curl | any | RPC calls in tests |
| git | any | Installation |
| Node.js | 18+ | MCP server only |
| npm | any | MCP server only |
| gstable | latest | Chain binary ([build instructions](https://github.com/stable-net/go-stablenet)) |

## CLI Reference

### Chain Lifecycle

```bash
chainbench init [--profile <name>]     # Initialize chain from profile
chainbench start                        # Start all nodes
chainbench stop                         # Stop all nodes (SIGTERM -> SIGKILL)
chainbench restart                      # stop -> clean -> init -> start
chainbench status [--json]              # Show node status
chainbench clean                        # Remove all node data
```

### Node Control

```bash
chainbench node stop 3                  # Stop node 3 only
chainbench node start 3                 # Restart node 3
chainbench node log 3                   # Show last 50 lines of node 3 log
chainbench node log 3 --follow          # Tail -f node 3 log
chainbench node rpc 1 eth_blockNumber   # RPC call to node 1
```

### Testing

```bash
chainbench test list                    # List available tests
chainbench test run basic/consensus     # Run single test
chainbench test run basic               # Run all basic tests
chainbench test run all                 # Run everything
chainbench report [--format json]       # Show test results
```

### Log Analysis

```bash
chainbench log timeline                 # Consensus event timeline
chainbench log anomaly                  # Detect anomalous patterns
chainbench log search "ROUND_CHANGE"    # Search across all node logs
```

### Profile Management

```bash
chainbench profile list                 # List available profiles
chainbench profile show default         # Show profile content
chainbench profile create my-test       # Create custom profile from default
```

### MCP Management

```bash
chainbench mcp enable [--target <dir>]  # Enable MCP server for a project
chainbench mcp disable [--target <dir>] # Disable MCP server for a project
chainbench mcp status [--target <dir>]  # Check MCP status for a project
```

## Built-in Tests

| Test | Description |
|------|-------------|
| `basic/consensus` | Verify block production, timing, miner diversity, sync |
| `basic/tx-send` | Send transaction and verify receipt |
| `basic/sync` | Check all nodes have synchronized block heights |
| `basic/peers` | Verify peer connectivity between nodes |
| `basic/rpc-health` | Check all RPC endpoints respond |
| `fault/node-crash` | Stop 1/4 validators, verify consensus continues, recover |
| `fault/node-recover` | Stop node, wait, restart, measure sync time |
| `fault/two-down` | Stop 2/4 validators, consensus halts, restore 1, resumes |
| `stress/tx-flood` | Send N transactions, measure TPS |
| `stress/block-time` | Block time statistics (avg/min/max/p95) over 100 blocks |

## Profiles

Profiles are YAML files that define the entire chain configuration. They support inheritance via `inherits`.

| Profile | Description |
|---------|-------------|
| `default` | 4 validators + 1 endpoint, 1s block time |
| `minimal` | 2 validators, fast testing |
| `bft-limit` | 4 validators, 2s blocks, auto-runs fault tests |
| `large` | 7 validators + 3 endpoints |

### Creating Custom Profiles

```bash
chainbench profile create my-scenario
# Edit profiles/custom/my-scenario.yaml
chainbench init --profile my-scenario
```

### Profile Schema

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

The `keys/preset/` directory contains pre-generated keys for 5 nodes. Fixed keys ensure validator addresses are **always the same** across runs, making log analysis and debugging reproducible.

| Node | Address | Role |
|------|---------|------|
| node1 | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | Validator |
| node2 | `0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c` | Validator |
| node3 | `0x8c4a10b9108d49b9d23f764464090831d9c17764` | Validator |
| node4 | `0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6` | Validator |
| node5 | `0x5400d8b543eaf6738c7b44799623bea88fd0f5ee` | Endpoint |

## Claude Code / MCP Integration

chainbench includes an [MCP](https://modelcontextprotocol.io/) server that allows AI agents like Claude Code to control the test chain directly.

### Setup

```bash
# Enable for a specific project (recommended)
cd /path/to/my-chain-project
chainbench mcp enable
# Creates a portable .mcp.json: {"mcpServers": {"chainbench": {"command": "chainbench-mcp"}}}

# Disable when no longer needed (stops token consumption)
chainbench mcp disable
```

The generated `.mcp.json` uses the `chainbench-mcp` wrapper from `$PATH` — no absolute paths, so it works across machines and users without modification.

> **Note:** Per-project registration is preferred. Global registration
> (`~/.claude/settings.local.json`) activates the MCP server for all projects,
> which may consume tokens even when not needed.

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
  1. chainbench_status      -> Check if chain is running
  2. chainbench_init         -> Initialize if needed
  3. chainbench_start        -> Start nodes
  4. chainbench_test_run     -> Run "basic/consensus"
  5. chainbench_report       -> Return results to user

User: "Kill node 3 and check if consensus continues"

AI Agent:
  1. chainbench_node_stop {node: 3}
  2. chainbench_test_run {test: "fault/node-crash"}
  3. chainbench_log_timeline
  4. Report analysis to user
```

## Project Structure

```
chainbench/
├── install.sh              # Remote installer (curl | bash)
├── uninstall.sh            # Uninstaller
├── setup.sh                # Local setup (MCP build + PATH registration)
├── chainbench.sh           # Main CLI entry point
├── lib/                    # CLI command implementations
│   ├── cmd_init.sh         # Chain initialization
│   ├── cmd_start.sh        # Node startup with PID tracking
│   ├── cmd_stop.sh         # Graceful shutdown
│   ├── cmd_status.sh       # Node status display
│   ├── cmd_node.sh         # Individual node control
│   ├── cmd_test.sh         # Test runner
│   ├── cmd_log.sh          # Log analysis dispatcher
│   ├── cmd_mcp.sh          # MCP server enable/disable per project
│   ├── common.sh           # Shared utilities
│   └── profile.sh          # YAML profile parser
├── profiles/               # Test scenario profiles (YAML with inheritance)
├── keys/preset/            # Pre-generated validator keys (5 nodes)
├── templates/              # Genesis and TOML config templates
├── tests/                  # Built-in test suites (basic, fault, stress)
├── logs/                   # Log analysis tools (parser, timeline, anomaly)
├── mcp-server/             # MCP server for AI integration (TypeScript)
└── bin/                    # CLI wrappers and platform binaries
    └── chainbench-mcp      # MCP server entry point (resolves $HOME/.chainbench)
```

## Troubleshooting

**`chainbench init` fails with "Cannot find gstable binary"**
- Set `chain.binary_path` in `profiles/default.yaml` to the absolute path of your `gstable` binary
- Or add the directory containing `gstable` to your `$PATH`

**Nodes start but immediately die**
- Check logs: `cat /tmp/node-data/logs/node1.log`
- Common cause: port conflict. Change `ports.base_*` values in the profile

**`chainbench stop` doesn't stop all processes**
- Manual cleanup: `pkill -15 gstable && sleep 3 && pkill -9 gstable`
- Then remove stale state: `rm -f state/pids.json`

**MCP server not recognized in Claude Code**
- Verify `chainbench mcp status` shows "enabled"
- Verify MCP server is built: `ls ~/.chainbench/mcp-server/dist/index.js`
- Run `/mcp` in Claude Code to reload servers

**Port already in use**
- Default ports: HTTP 8501-8505, WS 9501-9505, P2P 30301-30305
- Change base ports in the profile: `ports.base_http: 18501`

## Contributing

Contributions are welcome! Please read the guidelines below before submitting.

### Development Setup

```bash
git clone https://github.com/0xmhha/chainbench.git
cd chainbench
./setup.sh
```

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

1. `chainbench profile create my-profile`
2. Edit `profiles/custom/my-profile.yaml`
3. Use `inherits: default` to only override what you need

### Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit with [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `refactor:`)
4. Push and open a Pull Request
5. Ensure all existing tests pass: `chainbench test run all`

### Reporting Issues

- Use [GitHub Issues](https://github.com/0xmhha/chainbench/issues)
- Include: chainbench version, OS, steps to reproduce, relevant logs

## License

This project is licensed under the [Apache License 2.0](LICENSE).
