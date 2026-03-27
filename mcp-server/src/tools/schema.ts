import { z } from "zod";
import { writeFileSync, mkdirSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CHAINBENCH_DIR = resolve(__dirname, "../../../..");

// ---------------------------------------------------------------------------
// Schema documentation strings
// ---------------------------------------------------------------------------

const SECTION_DOCS: Record<string, string> = {
  top: `# Top-level fields

- \`name\` (string, **required**): Profile name. Used in log messages and status output.
- \`description\` (string, optional): Human-readable description shown by \`chainbench status\`.
- \`inherits\` (string, optional): Name of a parent profile to inherit from. Values in this profile override the parent.`,

  chain: `# chain — Binary and network identity

- \`chain.binary\` (string, default: \`"gstable"\`): Executable name of the go-stablenet binary.
- \`chain.binary_path\` (string, default: \`""\`): Absolute path to the binary. Empty = search \`$PATH\`.
- \`chain.network_id\` (integer, default: \`8283\`): P2P network identifier used for peer discovery.
- \`chain.chain_id\` (integer, default: \`8283\`): EIP-155 chain ID used for transaction signing.`,

  genesis: `# genesis — Genesis block configuration

- \`genesis.template\` (string, default: \`"templates/genesis.template.json"\`): Path to the JSON template for the genesis block. Relative to the chainbench root directory.

## genesis.overrides.wbft — WBFT Consensus parameters

These values are injected into the \`config.wbft\` section of \`genesis.json\`.

- \`blockPeriodSeconds\` (integer, default: \`1\`): Target interval between blocks in seconds.
- \`requestTimeoutSeconds\` (integer, default: \`2\`): WBFT view-change timeout in seconds.
- \`epochLength\` (integer, default: \`140\`): Number of blocks per epoch. At each epoch boundary the validator set is checkpointed.
- \`proposerPolicy\` (integer, default: \`0\`): Block proposer selection strategy. \`0\` = round-robin, \`1\` = sticky (same proposer until view change).
- \`maxRequestTimeoutSeconds\` (integer | null, default: \`null\`): Upper cap on request timeout growth. \`null\` = no cap.

## genesis.overrides.alloc — Pre-funded accounts

Map of Ethereum address → initial balance in wei (as a string).

Example:
\`\`\`yaml
alloc:
  "0xc17d493883eaa3b4cceb0f214b273392d562f9d8": "1000000000000000000000000000"
\`\`\``,

  "genesis.wbft": `# genesis.overrides.wbft — WBFT Consensus parameters

These values are injected into the \`config.wbft\` section of \`genesis.json\`.

- \`blockPeriodSeconds\` (integer, default: \`1\`): Target interval between blocks in seconds.
- \`requestTimeoutSeconds\` (integer, default: \`2\`): WBFT view-change timeout in seconds.
- \`epochLength\` (integer, default: \`140\`): Number of blocks per epoch. At each epoch boundary the validator set is checkpointed.
- \`proposerPolicy\` (integer, default: \`0\`): Block proposer selection strategy. \`0\` = round-robin, \`1\` = sticky (same proposer until view change).
- \`maxRequestTimeoutSeconds\` (integer | null, default: \`null\`): Upper cap on request timeout growth. \`null\` = no cap.`,

  "genesis.systemContracts": `# genesis.overrides.systemContracts — System contracts

go-stablenet deploys governance and token contracts at fixed addresses during genesis.

## govValidator (address: \`0x0000000000000000000000000000000000001001\`)
Manages the validator set via on-chain proposals.
- \`quorum\` (string): Minimum approvals needed for a proposal to pass.
- \`expiry\` (string): Proposal expiry in seconds.
- \`maxProposals\` (string): Maximum number of simultaneous open proposals.
- \`memberVersion\` (string): Version tag for the member list.
- \`gasTip\` (string): Minimum gas tip (wei) validators expect.
- \`validators\` (string[]): Initial validator addresses.
- \`members\` (string[]): Governance member addresses.
- \`blsPublicKeys\` (string[]): BLS public keys for the validators.

## nativeCoinAdapter (address: \`0x0000000000000000000000000000000000001000\`)
ERC-20 wrapper for the native coin (WKRC).
- \`name\` (string): Token name (e.g. \`"WKRC"\`).
- \`symbol\` (string): Token symbol (e.g. \`"WKRC"\`).
- \`currency\` (string): Fiat currency code (e.g. \`"KRW"\`).
- \`decimals\` (string): Decimal precision (e.g. \`"18"\`).
- \`masterMinter\` (address string): Address of the master minter contract.
- \`minters\` (address string): Initial minter address.
- \`minterAllowed\` (string): Initial minting allowance (wei).

## govMinter (address: \`0x0000000000000000000000000000000000001003\`)
Governance contract for minting operations.
- \`quorum\`, \`expiry\`, \`maxProposals\`, \`memberVersion\`: Same semantics as govValidator.
- \`fiatToken\` (address string): Address of the nativeCoinAdapter contract.
- \`members\` (string[]): Governance member addresses.

## govMasterMinter (address: \`0x0000000000000000000000000000000000001002\`)
Governance contract for master minter management.
- \`quorum\`, \`expiry\`, \`maxProposals\`, \`memberVersion\`: Same semantics as govValidator.
- \`fiatToken\` (address string): Address of the nativeCoinAdapter contract.
- \`minters\` (string[]): Initial minter addresses.
- \`maxMinterAllowance\` (string): Maximum allowance any minter may be granted (wei).
- \`members\` (string[]): Governance member addresses.

## govCouncil (address: \`0x0000000000000000000000000000000000001004\`)
Top-level council governance contract.
- \`quorum\`, \`expiry\`, \`maxProposals\`, \`memberVersion\`: Same semantics as govValidator.
- \`members\` (string[]): Council member addresses.`,

  nodes: `# nodes — Node topology and runtime settings

- \`nodes.validators\` (integer, default: \`4\`): Number of validator nodes launched with \`--mine\`.
- \`nodes.endpoints\` (integer, default: \`1\`): Number of non-mining endpoint nodes.
- \`nodes.verbosity\` (integer, default: \`4\`): Log verbosity for validator nodes. Range 0–5 (5 = most verbose).
- \`nodes.en_verbosity\` (integer, default: \`5\`): Log verbosity for endpoint nodes.
- \`nodes.gcmode\` (string, default: \`"archive"\`): Garbage collection mode. \`"archive"\` retains full state history; \`"full"\` prunes old state.
- \`nodes.cache\` (integer, default: \`2048\`): In-memory cache size in MB per node.
- \`nodes.extra_flags\` (string[], default: \`[]\`): Additional CLI flags passed verbatim to \`gstable\` for every node.`,

  keys: `# keys — Key material sources

- \`keys.source\` (string, default: \`""\`): Path to a keystore directory. Empty = use the built-in test keys bundled with chainbench.
- \`keys.nodekeys\` (string, default: \`""\`): Path to a directory containing per-node \`nodekey\` files. Empty = auto-generate.`,

  ports: `# ports — Network port allocation

All values are **base** ports. The actual port for node N = base + (N - 1).

- \`ports.base_p2p\` (integer, default: \`30301\`): P2P discovery/transport port.
- \`ports.base_http\` (integer, default: \`8501\`): HTTP JSON-RPC port.
- \`ports.base_ws\` (integer, default: \`9501\`): WebSocket JSON-RPC port.
- \`ports.base_auth\` (integer, default: \`8551\`): Authenticated RPC port (engine API).
- \`ports.base_metrics\` (integer, default: \`6061\`): Prometheus metrics port.

Example with 4 validators:
| Node | P2P   | HTTP | WS   | Auth |
|------|-------|------|------|------|
| 1    | 30301 | 8501 | 9501 | 8551 |
| 2    | 30302 | 8502 | 9502 | 8552 |
| 3    | 30303 | 8503 | 9503 | 8553 |
| 4    | 30304 | 8504 | 9504 | 8554 |`,

  logging: `# logging — Log file management

- \`logging.rotation\` (boolean, default: \`true\`): Enable automatic log rotation via \`logrotate\` or equivalent.
- \`logging.max_size\` (string, default: \`"10M"\`): Maximum size of a single log file before rotation (e.g. \`"10M"\`, \`"1G"\`).
- \`logging.max_files\` (integer, default: \`5\`): Number of rotated log files to retain per node.
- \`logging.directory\` (string, default: \`"data/logs"\`): Directory for log files. Relative to the chainbench root.`,

  tests: `# tests — Automated test configuration

- \`tests.auto_run\` (string[], default: \`[]\`): List of test names to run automatically after a successful \`chainbench_init\`. Uses the same \`category/name\` format as \`chainbench_test_run\`.

Example:
\`\`\`yaml
tests:
  auto_run:
    - basic/consensus
    - basic/tx-send
\`\`\``,
};

const FULL_SCHEMA = `# chainbench YAML Profile Schema (go-stablenet)

Use \`chainbench_schema_query\` with a \`section\` argument to get focused documentation.
Available sections: chain, genesis, genesis.wbft, genesis.systemContracts, nodes, keys, ports, logging, tests

---

${SECTION_DOCS["top"]}

---

${SECTION_DOCS["chain"]}

---

${SECTION_DOCS["genesis"]}

---

${SECTION_DOCS["genesis.systemContracts"]}

---

${SECTION_DOCS["nodes"]}

---

${SECTION_DOCS["keys"]}

---

${SECTION_DOCS["ports"]}

---

${SECTION_DOCS["logging"]}

---

${SECTION_DOCS["tests"]}
`;

function getSchema(section: string | undefined): string {
  if (!section) {
    return FULL_SCHEMA;
  }

  const normalized = section.trim().toLowerCase();
  const doc = SECTION_DOCS[normalized];
  if (doc) {
    return doc;
  }

  const available = Object.keys(SECTION_DOCS).filter((k) => k !== "top").join(", ");
  return `Unknown section '${section}'. Available sections: ${available}\n\nOmit the section parameter to retrieve the full schema.`;
}

function validateProfileName(name: string): string | null {
  if (!/^[a-zA-Z0-9_\-]+$/.test(name)) {
    return "Profile name may only contain alphanumeric characters, dashes, and underscores.";
  }
  if (name.length > 64) {
    return "Profile name must be 64 characters or fewer.";
  }
  return null;
}

function validateYamlContent(content: string): string | null {
  if (content.trim().length === 0) {
    return "Profile content must not be empty.";
  }
  if (content.length > 64 * 1024) {
    return "Profile content exceeds the 64 KiB limit.";
  }
  return null;
}

export function registerSchemaTools(server: McpServer): void {
  server.tool(
    "chainbench_schema_query",
    "Query the YAML profile schema for chainbench. Returns documentation of all supported configuration fields, their types, defaults, and meanings. Use this before creating a custom profile to understand what settings are available.",
    {
      section: z
        .string()
        .optional()
        .describe(
          "Optional section to query. One of: 'chain', 'genesis', 'genesis.wbft', 'genesis.systemContracts', 'nodes', 'keys', 'ports', 'logging', 'tests'. Omit to retrieve the full schema."
        ),
    },
    async ({ section }) => {
      const text = getSchema(section);
      return { content: [{ type: "text" as const, text }] };
    }
  );

  server.tool(
    "chainbench_profile_send",
    "Create or overwrite a custom YAML profile. The profile will be saved under profiles/custom/<name>.yaml and can then be used with chainbench_init --profile custom/<name>. Use chainbench_schema_query first to understand all available fields.",
    {
      name: z
        .string()
        .describe("Profile name (alphanumeric, dashes, underscores only). Saved as profiles/custom/<name>.yaml."),
      content: z
        .string()
        .describe("Full YAML content for the profile. Must include at minimum a 'name' field."),
    },
    async ({ name, content }) => {
      const nameError = validateProfileName(name);
      if (nameError) {
        return { content: [{ type: "text" as const, text: `Error: ${nameError}` }] };
      }

      const contentError = validateYamlContent(content);
      if (contentError) {
        return { content: [{ type: "text" as const, text: `Error: ${contentError}` }] };
      }

      const customDir = resolve(CHAINBENCH_DIR, "profiles", "custom");
      const profilePath = resolve(customDir, `${name}.yaml`);

      // Ensure the resolved path stays within profiles/custom/ (path traversal guard)
      if (!profilePath.startsWith(customDir + "/") && profilePath !== customDir) {
        return {
          content: [
            {
              type: "text" as const,
              text: "Error: resolved profile path escapes the profiles/custom/ directory.",
            },
          ],
        };
      }

      try {
        mkdirSync(customDir, { recursive: true });
        writeFileSync(profilePath, content, { encoding: "utf-8" });
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        return {
          content: [{ type: "text" as const, text: `Error writing profile: ${message}` }],
        };
      }

      return {
        content: [
          {
            type: "text" as const,
            text: `Profile saved to profiles/custom/${name}.yaml.\nUse it with: chainbench_init({ profile: "custom/${name}" })`,
          },
        ],
      };
    }
  );
}
