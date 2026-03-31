import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { registerLifecycleTools } from "./tools/lifecycle.js";
import { registerNodeTools } from "./tools/node.js";
import { registerTestTools } from "./tools/test.js";
import { registerSchemaTools } from "./tools/schema.js";
import { registerLogTools } from "./tools/log.js";
import { registerRemoteTools } from "./tools/remote.js";
import { registerConsensusTools } from "./tools/consensus.js";
import { registerNetworkTools } from "./tools/network.js";

const server = new McpServer({
  name: "chainbench-mcp-server",
  version: "0.3.0",
});

// Register all tool groups
registerLifecycleTools(server);
registerNodeTools(server);
registerTestTools(server);
registerSchemaTools(server);
registerLogTools(server);
registerRemoteTools(server);
registerConsensusTools(server);
registerNetworkTools(server);

// Start server with stdio transport
const transport = new StdioServerTransport();
await server.connect(transport);
