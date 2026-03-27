import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { registerLifecycleTools } from "./tools/lifecycle.js";
import { registerNodeTools } from "./tools/node.js";
import { registerTestTools } from "./tools/test.js";
import { registerSchemaTools } from "./tools/schema.js";
import { registerLogTools } from "./tools/log.js";

const server = new McpServer({
  name: "chainbench-mcp-server",
  version: "0.1.0",
});

// Register all tool groups
registerLifecycleTools(server);
registerNodeTools(server);
registerTestTools(server);
registerSchemaTools(server);
registerLogTools(server);

// Start server with stdio transport
const transport = new StdioServerTransport();
await server.connect(transport);
