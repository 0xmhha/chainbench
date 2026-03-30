import { z } from "zod";
import { readFileSync, writeFileSync, mkdirSync, existsSync } from "fs";
import { resolve } from "path";
import { execFileSync } from "child_process";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { CHAINBENCH_DIR, buildEnv } from "../utils/exec.js";

// ---------------------------------------------------------------------------
// Schema documentation
// ---------------------------------------------------------------------------

const SECTION_DOCS: Record<string, string> = {
  top: `# Top-level fields
- \`name\` (string, required): Profile name.
- \`description\` (string, optional): Human-readable description.
- \`inherits\` (string, optional): Parent profile name for inheritance.`,

  chain: `# chain — Binary and network identity
- \`chain.binary\` (string, default: "gstable"): Executable name.
- \`chain.binary_path\` (string, default: ""): Absolute path to binary. Empty = search $PATH.
- \`chain.network_id\` (integer, default: 8283): P2P network ID.
- \`chain.chain_id\` (integer, default: 8283): EIP-155 chain ID.`,

  data: `# data — Data directory
- \`data.directory\` (string, default: "data"): Root directory for node data. Absolute or relative to chainbench.`,

  genesis: `# genesis — Genesis block configuration
- \`genesis.template\` (string): Path to genesis JSON template.
## genesis.overrides.wbft
- \`blockPeriodSeconds\` (int, default: 1): Block interval in seconds.
- \`requestTimeoutSeconds\` (int, default: 2): WBFT consensus timeout.
- \`epochLength\` (int, default: 140): Blocks per epoch.
- \`proposerPolicy\` (int, default: 0): 0=round-robin, 1=sticky.
- \`maxRequestTimeoutSeconds\` (int|null, default: null): Timeout cap.`,

  "genesis.wbft": `# genesis.overrides.wbft — WBFT Consensus parameters
- \`blockPeriodSeconds\` (int, default: 1): Block interval.
- \`requestTimeoutSeconds\` (int, default: 2): Consensus timeout.
- \`epochLength\` (int, default: 140): Blocks per epoch.
- \`proposerPolicy\` (int, default: 0): 0=round-robin, 1=sticky.
- \`maxRequestTimeoutSeconds\` (int|null): Timeout cap.`,

  "genesis.systemContracts": `# genesis.overrides.systemContracts
- govValidator (0x...1001): Validator governance. Params: quorum, expiry, gasTip, validators, members, blsPublicKeys
- nativeCoinAdapter (0x...1000): Native coin ERC-20. Params: name, symbol, currency, decimals, masterMinter, minters
- govMinter (0x...1003): Minting governance. Params: quorum, expiry, fiatToken, members
- govMasterMinter (0x...1002): Master minter. Params: quorum, expiry, fiatToken, minters, members
- govCouncil (0x...1004): Council. Params: quorum, expiry, members`,

  nodes: `# nodes — Node topology
- \`nodes.validators\` (int, default: 4): Validator count (with --mine).
- \`nodes.endpoints\` (int, default: 1): Non-mining node count.
- \`nodes.verbosity\` (int, default: 4): Validator log level (0-5).
- \`nodes.en_verbosity\` (int, default: 5): Endpoint log level.
- \`nodes.gcmode\` (string, default: "archive"): "archive" or "full".
- \`nodes.cache\` (int, default: 2048): Cache MB per node.
- \`nodes.extra_flags\` (string[], default: []): Extra gstable CLI flags.`,

  keys: `# keys — Key material
- \`keys.mode\` (string, default: "static"): "static" = reuse preset, "generate" = new keys each init.
- \`keys.source\` (string, default: "keys/preset"): Preset keys directory.`,

  ports: `# ports — Port allocation (base + node_index)
- \`ports.base_p2p\` (int, default: 30301)
- \`ports.base_http\` (int, default: 8501)
- \`ports.base_ws\` (int, default: 9501)
- \`ports.base_auth\` (int, default: 8551)
- \`ports.base_metrics\` (int, default: 6061)`,

  logging: `# logging — Log management
- \`logging.rotation\` (bool, default: true): Enable log rotation.
- \`logging.max_size\` (string, default: "10M"): Max file size.
- \`logging.max_files\` (int, default: 5): Files to keep.
- \`logging.directory\` (string, default: "data/logs"): Log directory.`,

  tests: `# tests — Auto-run tests
- \`tests.auto_run\` (string[], default: []): Tests to run after init.`,
};

const FULL_SCHEMA = Object.entries(SECTION_DOCS).map(([, v]) => v).join("\n\n---\n\n");

function getSchema(section: string | undefined): string {
  if (!section) return FULL_SCHEMA;
  const doc = SECTION_DOCS[section.trim().toLowerCase()];
  if (doc) return doc;
  return `Unknown section '${section}'. Available: ${Object.keys(SECTION_DOCS).filter(k => k !== "top").join(", ")}`;
}

// ---------------------------------------------------------------------------
// Profile helpers
// ---------------------------------------------------------------------------

function findProfilePath(name: string): string | null {
  for (const p of [
    resolve(CHAINBENCH_DIR, "profiles", `${name}.yaml`),
    resolve(CHAINBENCH_DIR, "profiles", "custom", `${name}.yaml`),
  ]) {
    if (existsSync(p)) return p;
  }
  return null;
}

function getActiveProfileName(): string | null {
  const f = resolve(CHAINBENCH_DIR, "state", "current-profile.yaml");
  if (!existsSync(f)) return null;
  const m = readFileSync(f, "utf-8").match(/^name:\s*(.+)$/m);
  return m ? m[1].trim() : null;
}

function readProfileAsJson(name: string): Record<string, unknown> | null {
  if (name === "__active__") {
    const merged = resolve(CHAINBENCH_DIR, "state", "current-profile-merged.json");
    if (existsSync(merged)) return JSON.parse(readFileSync(merged, "utf-8"));
  }
  const path = findProfilePath(name);
  if (!path) return null;
  try {
    // Use execFileSync (no shell) with python3 to parse YAML safely
    const pyScript = `
import json, sys
try:
    import yaml
    with open(sys.argv[1]) as f:
        print(json.dumps(yaml.safe_load(f)))
except ImportError:
    print('{}')
`;
    const result = execFileSync("python3", ["-c", pyScript, path], {
      encoding: "utf-8",
      timeout: 5000,
      env: buildEnv(),
    });
    return JSON.parse(result.trim());
  } catch {
    return null;
  }
}

function getNestedValue(obj: Record<string, unknown>, dotPath: string): unknown {
  let current: unknown = obj;
  for (const key of dotPath.split(".")) {
    if (current === null || current === undefined || typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[key];
  }
  return current;
}

function setNestedValue(obj: Record<string, unknown>, dotPath: string, value: unknown): void {
  const keys = dotPath.split(".");
  let current: Record<string, unknown> = obj;
  for (let i = 0; i < keys.length - 1; i++) {
    if (!(keys[i] in current) || typeof current[keys[i]] !== "object" || current[keys[i]] === null) {
      current[keys[i]] = {};
    }
    current = current[keys[i]] as Record<string, unknown>;
  }
  current[keys[keys.length - 1]] = value;
}

function jsonToYaml(obj: Record<string, unknown>, indent = 0): string {
  const pad = "  ".repeat(indent);
  let r = "";
  for (const [k, v] of Object.entries(obj)) {
    if (v === null || v === undefined) { r += `${pad}${k}: null\n`; }
    else if (Array.isArray(v)) {
      if (v.length === 0) { r += `${pad}${k}: []\n`; }
      else { r += `${pad}${k}:\n`; for (const item of v) r += `${pad}  - ${JSON.stringify(item)}\n`; }
    }
    else if (typeof v === "object") { r += `${pad}${k}:\n` + jsonToYaml(v as Record<string, unknown>, indent + 1); }
    else if (typeof v === "string") { r += `${pad}${k}: "${v}"\n`; }
    else { r += `${pad}${k}: ${v}\n`; }
  }
  return r;
}

function parseInputValue(raw: string): unknown {
  try { return JSON.parse(raw); } catch { return raw; }
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

export function registerSchemaTools(server: McpServer): void {
  // --- Schema query ---
  server.tool(
    "chainbench_schema_query",
    "Query the YAML profile schema. Returns field documentation with types, defaults, and meanings.",
    { section: z.string().optional().describe("Section: chain, data, genesis, genesis.wbft, genesis.systemContracts, nodes, keys, ports, logging, tests. Omit for full schema.") },
    async ({ section }) => ({ content: [{ type: "text" as const, text: getSchema(section) }] })
  );

  // --- Profile send (create full profile) ---
  server.tool(
    "chainbench_profile_send",
    "Create a custom YAML profile. Saved under profiles/custom/<name>.yaml.",
    {
      name: z.string().describe("Profile name (alphanumeric, dashes, underscores)."),
      content: z.string().describe("Full YAML content."),
    },
    async ({ name, content }) => {
      if (!/^[a-zA-Z0-9_-]+$/.test(name)) return { content: [{ type: "text" as const, text: "Error: invalid name." }] };
      if (!content.trim()) return { content: [{ type: "text" as const, text: "Error: empty content." }] };
      const dir = resolve(CHAINBENCH_DIR, "profiles", "custom");
      const path = resolve(dir, `${name}.yaml`);
      if (!path.startsWith(dir)) return { content: [{ type: "text" as const, text: "Error: path traversal." }] };
      mkdirSync(dir, { recursive: true });
      writeFileSync(path, content, "utf-8");
      return { content: [{ type: "text" as const, text: `Saved: profiles/custom/${name}.yaml\nUse: chainbench_init({ profile: "custom/${name}" })` }] };
    }
  );

  // --- Profile get (NEW) ---
  server.tool(
    "chainbench_profile_get",
    "Read a profile's content or a specific field value. Use to inspect current configuration before making changes.",
    {
      name: z.string().optional().describe("Profile name. Omit or 'active' for the currently active profile."),
      field: z.string().optional().describe("Dot-notation path to read a specific field (e.g., 'chain.binary_path', 'nodes.validators'). Omit for full profile."),
    },
    async ({ name, field }) => {
      const target = (!name || name === "active") ? "__active__" : name;

      if (field) {
        const json = readProfileAsJson(target);
        if (!json) return { content: [{ type: "text" as const, text: `Error: ${target === "__active__" ? "No active profile. Run chainbench init first." : `Profile '${name}' not found.`}` }] };
        const val = getNestedValue(json, field);
        if (val === undefined) return { content: [{ type: "text" as const, text: `Field '${field}' not found in profile.` }] };
        const display = typeof val === "object" ? JSON.stringify(val, null, 2) : String(val);
        return { content: [{ type: "text" as const, text: `${field} = ${display}` }] };
      }

      // Full profile
      if (target === "__active__") {
        const stateFile = resolve(CHAINBENCH_DIR, "state", "current-profile.yaml");
        if (!existsSync(stateFile)) return { content: [{ type: "text" as const, text: "No active profile. Run chainbench init first." }] };
        return { content: [{ type: "text" as const, text: `# Active profile\n\n${readFileSync(stateFile, "utf-8")}` }] };
      }

      const path = findProfilePath(target);
      if (!path) return { content: [{ type: "text" as const, text: `Profile '${name}' not found.` }] };
      return { content: [{ type: "text" as const, text: `# Profile: ${name}\n\n${readFileSync(path, "utf-8")}` }] };
    }
  );

  // --- Profile set (NEW) ---
  server.tool(
    "chainbench_profile_set",
    "Update a specific field in a profile. Uses dot-notation for nested fields. After modifying, run chainbench_init to apply.",
    {
      name: z.string().default("default").describe("Profile to modify (e.g., 'default', 'minimal', 'custom/my-profile')."),
      field: z.string().describe("Dot-notation path (e.g., 'chain.binary_path', 'nodes.validators', 'data.directory')."),
      value: z.string().describe("New value. Numbers/booleans auto-detected. Use JSON for arrays/objects. Strings used as-is."),
    },
    async ({ name, field, value }) => {
      const profilePath = findProfilePath(name);
      if (!profilePath) return { content: [{ type: "text" as const, text: `Error: Profile '${name}' not found.` }] };

      const json = readProfileAsJson(name);
      if (!json) return { content: [{ type: "text" as const, text: `Error: Could not parse profile '${name}'.` }] };

      const oldVal = getNestedValue(json, field);
      const oldDisplay = oldVal === undefined ? "(not set)" : (typeof oldVal === "object" ? JSON.stringify(oldVal) : String(oldVal));

      const parsed = parseInputValue(value);
      setNestedValue(json, field, parsed);

      const yaml = `# chainbench profile: ${json["name"] || name}\n` + jsonToYaml(json);
      writeFileSync(profilePath, yaml, "utf-8");

      // Update active state if this profile is active
      const active = getActiveProfileName();
      if (active === name || active === (json["name"] as string)) {
        const stateYaml = resolve(CHAINBENCH_DIR, "state", "current-profile.yaml");
        const stateMerged = resolve(CHAINBENCH_DIR, "state", "current-profile-merged.json");
        if (existsSync(stateYaml)) writeFileSync(stateYaml, yaml, "utf-8");
        if (existsSync(stateMerged)) writeFileSync(stateMerged, JSON.stringify(json, null, 2), "utf-8");
      }

      const newDisplay = typeof parsed === "object" ? JSON.stringify(parsed) : String(parsed);
      return {
        content: [{ type: "text" as const, text: `Updated '${name}':\n  ${field}: ${oldDisplay} → ${newDisplay}\n\nRun chainbench_init to apply.` }],
      };
    }
  );
}
