import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import { formatWireResult } from "../utils/wireResult.js";
import { errorResp, type FormattedToolResponse } from "../utils/mcpResp.js";

// Sprint 5c.4.1 Task 2 — chainbench_stop now invokes the network.stop_all
// wire handler instead of shelling out via runChainbench. Exported as named
// schema + handler so lifecycle.test.ts can drive zod parse + handler logic
// directly without spinning up an MCP server stub (matches Sprint 5c.3
// NodeRpcArgs precedent in node.ts).
export const StopArgs = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'. Remote networks reject."),
}).strict();

export async function _stopHandler(
  args: z.infer<typeof StopArgs>,
): Promise<FormattedToolResponse> {
  const wireArgs: Record<string, unknown> = {};
  if (args.network !== undefined) wireArgs.network = args.network;
  const result = await callWire("network.stop_all", wireArgs);
  return formatWireResult(result);
}

// Sprint 5c.4.1 Task 4 — chainbench_status now invokes the network.status
// wire handler. Mirrors the StopArgs / _stopHandler shape (Task 2 precedent),
// just routes to a different wire command. Local network only — remote
// networks have no node lifecycle to inspect via the bash status path.
export const StatusArgs = z.object({
  network: z.string().min(1).optional()
    .describe("Network alias. Defaults to 'local'."),
}).strict();

export async function _statusHandler(
  args: z.infer<typeof StatusArgs>,
): Promise<FormattedToolResponse> {
  const wireArgs: Record<string, unknown> = {};
  if (args.network !== undefined) wireArgs.network = args.network;
  const result = await callWire("network.status", wireArgs);
  return formatWireResult(result);
}

const PROJECT_ROOT_SCHEMA = z
  .string()
  .optional()
  .describe(
    "Absolute path to the blockchain project root (e.g., go-stablenet repo). " +
    "Forwarded over the wire envelope as the chainbench.sh working directory " +
    "so its binary auto-detection finds <project_root>/build/bin/.",
  );

const BINARY_PATH_SCHEMA = z
  .string()
  .optional()
  .describe(
    "Absolute path to the chain binary (e.g., /opt/gstable/build/bin/gstable). " +
    "Overrides the profile's chain.binary_path for this invocation only. " +
    "Forwarded to the wire handler, which appends `--binary-path <path>` to " +
    "chainbench.sh argv.",
  );

function validateProjectRoot(projectRoot: string | undefined): string | null {
  if (projectRoot && !projectRoot.startsWith("/")) {
    return "project_root must be an absolute path.";
  }
  return null;
}

function validateBinaryPath(binaryPath: string | undefined): string | null {
  if (binaryPath === undefined) return null;
  if (binaryPath.length === 0) return "binary_path must not be empty.";
  if (!binaryPath.startsWith("/")) return "binary_path must be an absolute path.";
  return null;
}

// Sprint 5c.4.2 — chainbench_init/start/restart route through the Go wire
// (network.init / network.start_all / network.restart) instead of shelling
// out via runChainbench, matching the node.start precedent. Boundary checks
// stay here (defense in depth; the Go handler re-validates) so a bad profile
// or relative path is rejected before a wire spawn. profile / project_root /
// binary_path are forwarded in the wire envelope.
export const InitArgs = z.object({
  profile: z
    .string()
    .default("default")
    .describe("Profile name from the profiles/ directory (e.g. 'default', 'minimal', 'large')"),
  project_root: PROJECT_ROOT_SCHEMA,
  binary_path: BINARY_PATH_SCHEMA,
}).strict();

export async function _initHandler(
  args: z.infer<typeof InitArgs>,
): Promise<FormattedToolResponse> {
  const { profile, project_root, binary_path } = args;
  if (!/^[a-zA-Z0-9_\-/]+$/.test(profile)) {
    return errorResp(
      `invalid profile name '${profile}'. Only alphanumeric characters, dashes, underscores, and slashes are allowed.`,
    );
  }
  const rootError = validateProjectRoot(project_root);
  if (rootError) return errorResp(rootError);
  const bpError = validateBinaryPath(binary_path);
  if (bpError) return errorResp(bpError);

  const wireArgs: Record<string, unknown> = { profile };
  if (project_root !== undefined) wireArgs.project_root = project_root;
  if (binary_path !== undefined) wireArgs.binary_path = binary_path;
  const result = await callWire("network.init", wireArgs);
  return formatWireResult(result);
}

export const StartArgs = z.object({
  project_root: PROJECT_ROOT_SCHEMA,
  binary_path: BINARY_PATH_SCHEMA,
}).strict();

export async function _startHandler(
  args: z.infer<typeof StartArgs>,
): Promise<FormattedToolResponse> {
  const { project_root, binary_path } = args;
  const rootError = validateProjectRoot(project_root);
  if (rootError) return errorResp(rootError);
  const bpError = validateBinaryPath(binary_path);
  if (bpError) return errorResp(bpError);

  const wireArgs: Record<string, unknown> = {};
  if (project_root !== undefined) wireArgs.project_root = project_root;
  if (binary_path !== undefined) wireArgs.binary_path = binary_path;
  const result = await callWire("network.start_all", wireArgs);
  return formatWireResult(result);
}

export const RestartArgs = z.object({
  project_root: PROJECT_ROOT_SCHEMA,
  binary_path: BINARY_PATH_SCHEMA,
}).strict();

export async function _restartHandler(
  args: z.infer<typeof RestartArgs>,
): Promise<FormattedToolResponse> {
  const { project_root, binary_path } = args;
  const rootError = validateProjectRoot(project_root);
  if (rootError) return errorResp(rootError);
  const bpError = validateBinaryPath(binary_path);
  if (bpError) return errorResp(bpError);

  const wireArgs: Record<string, unknown> = {};
  if (project_root !== undefined) wireArgs.project_root = project_root;
  if (binary_path !== undefined) wireArgs.binary_path = binary_path;
  const result = await callWire("network.restart", wireArgs);
  return formatWireResult(result);
}

// chainbench_clean routes through the network.clean wire handler, which removes
// node data directories (keeping config/profiles) by spawning
// `chainbench.sh clean`. It takes no arguments — the Go handler ignores
// profile/root — so the only boundary is .strict() rejecting hallucinated keys.
export const CleanArgs = z.object({}).strict();

export async function _cleanHandler(
  _args: z.infer<typeof CleanArgs>,
): Promise<FormattedToolResponse> {
  const result = await callWire("network.clean", {});
  return formatWireResult(result);
}

export function registerLifecycleTools(server: McpServer): void {
  server.tool(
    "chainbench_init",
    "Initialize a local blockchain from a profile. Generates genesis.json, TOML configs, and initializes node data directories. Must be run before starting the chain.",
    InitArgs.shape,
    _initHandler,
  );

  server.tool(
    "chainbench_start",
    "Start all chain nodes. The chain must have been initialized with chainbench_init first.",
    StartArgs.shape,
    _startHandler,
  );

  server.tool(
    "chainbench_stop",
    "Stop all running chain nodes gracefully. Local network only — remote " +
    "networks reject (no process control). Returns the bash CLI's stdout " +
    "showing per-PID SIGTERM and graceful-shutdown wait status.",
    StopArgs.shape,
    _stopHandler,
  );

  server.tool(
    "chainbench_restart",
    "Restart the chain: stop all nodes, clean data, re-initialize with the same profile, and start fresh.",
    RestartArgs.shape,
    _restartHandler,
  );

  server.tool(
    "chainbench_status",
    "Return current node status as JSON. Includes per-node block height, " +
    "peer count, running state, and overall consensus health. Local network " +
    "only — remote networks reject (use chainbench_node_rpc for remote).",
    StatusArgs.shape,
    _statusHandler,
  );

  server.tool(
    "chainbench_clean",
    "Remove all node data directories (datadirs, pids), keeping config and " +
    "profiles. Use to reset chain state before a fresh chainbench_init. Local " +
    "network only — takes no arguments.",
    CleanArgs.shape,
    _cleanHandler,
  );
}
