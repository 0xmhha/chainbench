// mcpResp.ts — shared MCP response shape and INVALID_ARGS helper.
//
// Hosts FormattedToolResponse (the {content, isError} shape every MCP tool
// handler returns) and errorResp, the canonical "Error (INVALID_ARGS): <msg>"
// boundary response used by cross-field validators that zod schemas cannot
// express cleanly. Keeping both in one module ensures the structured error
// text stays in lock-step with formatWireResult's failure path so MCP
// clients can parse the two response sources uniformly.
//
// Sprint 5c.3 P3: hoisted from chain_read.ts and chain_tx.ts (rule of three —
// node.ts and additional reroute tools land in 5c.3 with the same shape).

export interface FormattedToolResponse {
  content: Array<{ type: "text"; text: string; [k: string]: unknown }>;
  isError?: boolean;
  [k: string]: unknown;
}

export function errorResp(msg: string): FormattedToolResponse {
  return {
    content: [{ type: "text", text: `Error (INVALID_ARGS): ${msg}` }],
    isError: true,
  };
}
