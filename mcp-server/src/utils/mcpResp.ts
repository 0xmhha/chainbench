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

// textResult wraps a plain string as the canonical {content:[{type:"text"}]}
// success response. Shared by the RPC-backed tools (network, consensus) that
// hand-format their own text rather than going through formatWireResult.
export function textResult(text: string): FormattedToolResponse {
  return { content: [{ type: "text", text }] };
}

// formatExecResult renders a runChainbench {stdout, stderr, exitCode} into a
// plain string: stdout on success (or emptyFallback when stdout is blank),
// and "Error (exit N): <detail>" on failure. emptyFallback differs per tool
// surface ("Done." for actions, "No output." for log, "{}" for spec).
export function formatExecResult(
  result: { stdout: string; stderr: string; exitCode: number },
  emptyFallback = "Done.",
): string {
  if (result.exitCode === 0) {
    return result.stdout || emptyFallback;
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}
