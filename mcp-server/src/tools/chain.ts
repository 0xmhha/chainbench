// chain.ts — registers all high-level chain evaluation tools.
//
// Tool implementations live in:
//   - chain_read.ts  — read-only tools (no signer)
//   - chain_tx.ts    — signer-required write tools
//
// This file exists as a single import surface for index.ts so the
// register-everything orchestration stays at the top of the file tree
// while implementations stay grouped by signer requirement.

import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { registerChainReadTools } from "./chain_read.js";
import { registerChainTxTools } from "./chain_tx.js";

export function registerChainTools(server: McpServer): void {
  registerChainReadTools(server);
  registerChainTxTools(server);
}
